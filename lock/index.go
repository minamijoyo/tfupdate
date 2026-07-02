package lock

import (
	"context"
	"fmt"
	"log"
	"maps"
	"runtime"
	"slices"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/minamijoyo/tfupdate/tfregistry"
)

// Index is an in-memory data store for caching provider hash values.
type Index interface {
	// GetOrCreateProviderVersion returns a cached provider version if available,
	// otherwise creates it.
	// address is a provider address such as hashicorp/null.
	// version is a version number such as 3.2.1.
	// platforms is a list of target platforms to generate hash values.
	// Target platform names consist of an operating system and a CPU architecture such as darwin_arm64.
	GetOrCreateProviderVersion(ctx context.Context, address string, version string, platforms []string) (*ProviderVersion, error)
}

// index is an implementation for Index interface.
type index struct {
	// providers is a dictionary of providerIndex.
	// The key is a provider address such as hashicorp/null.
	providers map[string]*providerIndex

	// papi is a ProviderLockAPI interface implementation used for locking provider.
	papi ProviderLockAPI
}

// NewIndexFromConfig returns a new instance of Index with the given registry config.
func NewIndexFromConfig(config tfregistry.Config) (Index, error) {
	client, err := NewProviderLockClient(config)
	if err != nil {
		return nil, err
	}

	index := NewIndex(client)

	return index, nil
}

// NewIndex returns a new instance of Index with the given ProviderLockAPI.
func NewIndex(papi ProviderLockAPI) Index {
	providers := make(map[string]*providerIndex)
	return &index{
		providers: providers,
		papi:      papi,
	}
}

// GetOrCreateProviderVersion returns a cached provider version if available,
// otherwise creates it.
func (i *index) GetOrCreateProviderVersion(ctx context.Context, address string, version string, platforms []string) (*ProviderVersion, error) {
	pi, ok := i.providers[address]
	if !ok {
		// cache miss
		pi = newProviderIndex(address, i.papi)
		i.providers[address] = pi
	}
	// Delegate to ProviderIndex.
	return pi.getOrCreateProviderVersion(ctx, version, platforms)
}

// The providerIndex holds multiple version data for a specific provider.
type providerIndex struct {
	// address is a provider address such as hashicorp/null.
	address string

	// versions is a dictionary of ProviderVersion.
	// The key is a version number such as 3.2.1.
	versions map[string]*ProviderVersion

	// papi is a ProviderLockAPI interface implementation used for locking provider.
	papi ProviderLockAPI
}

// newProviderIndex returns a new instance of providerIndex.
func newProviderIndex(address string, papi ProviderLockAPI) *providerIndex {
	versions := make(map[string]*ProviderVersion)
	return &providerIndex{
		address:  address,
		versions: versions,
		papi:     papi,
	}
}

// getOrCreateProviderVersion returns a cached provider version if available,
// otherwise creates it.
func (pi *providerIndex) getOrCreateProviderVersion(ctx context.Context, version string, platforms []string) (*ProviderVersion, error) {
	pv, ok := pi.versions[version]
	if !ok {
		// cache miss
		var err error
		pv, err = pi.createProviderVersion(ctx, version, platforms)
		if err != nil {
			return nil, err
		}
		pi.versions[version] = pv
	}
	return pv, nil
}

// createProviderVersion downloads the specified provider, calculates the hash
// value and returns an instance of the ProviderVersion.
func (pi *providerIndex) createProviderVersion(ctx context.Context, version string, platforms []string) (*ProviderVersion, error) {
	// Starting with OpenTofu v1.12, the OpenTofu Registry now returns both the
	// zh hash and the precomputed h1 hash, so fetching only the metadata allows
	// us to skip downloading the provider’s binary.
	// If the platform is omitted, we assume that the registry metadata returns the h1 hash values for all platforms.
	if len(platforms) == 0 {
		// The metadata request returns hash values for all platforms, but we need to specify a platform when making the call.
		platform := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

		pv, err := pi.fetchProviderPackageMetadata(ctx, version, platform)
		if err != nil {
			return nil, err
		}

		if len(pv.h1Hashes) == 0 {
			return nil, fmt.Errorf("failed to fetch provider package metadata for %s %s. The registry does not support h1 hashes. Please specify the platform on which the hash value should be calculated", pi.address, version)
		}
		// If h1 hashes are available, we can skip downloading the provider binary.
		log.Printf("[DEBUG] providerIndex.createProviderVersion: %s, %s. The registry returns both h1 and zh hashes. Skipping provider download.", pi.address, version)
		return pv, nil
	}

	ret := newEmptyProviderVersion(pi.address, version)
	for _, platform := range platforms {
		req, err := newProviderDownloadRequest(pi.address, version, platform)
		if err != nil {
			return nil, err
		}

		// Download a given provider from registry.
		log.Printf("[DEBUG] providerIndex.createProviderVersion: %s, %s, %s", pi.address, version, platform)
		res, err := pi.papi.ProviderDownload(ctx, req)
		if err != nil {
			return nil, err
		}

		// Currently the Terraform Registry returns the zh hash for all platforms,
		// but not the h1 hash, so the h1 hash has to be calculated separately.
		// We need to calculate the values for each platform and merge the results.
		pv, err := buildProviderVersionFromDownload(pi.address, version, platform, res)
		if err != nil {
			return nil, err
		}

		err = ret.Merge(pv)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// fetchProviderPackageMetadata fetches the provider package metadata from the registry and returns an instance of the ProviderVersion.
func (pi *providerIndex) fetchProviderPackageMetadata(ctx context.Context, version string, platform string) (*ProviderVersion, error) {
	req, err := newProviderPackageMetadataRequest(pi.address, version, platform)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] providerIndex.fetchProviderPackageMetadata: %s, %s, %s", pi.address, version, platform)
	res, err := pi.papi.ProviderPackageMetadata(ctx, req)
	if err != nil {
		return nil, err
	}

	pv, err := buildProviderVersionFromPackageMetadata(pi.address, version, res)
	if err != nil {
		return nil, err
	}

	return pv, nil
}

// newProviderPackageMetadataRequest is a helper function for building the parameters for fetching provider package metadata.
// address is a provider address such as hashicorp/null.
// version is a version number such as 3.2.1.
// platform is a target platform name such as darwin_arm64.
func newProviderPackageMetadataRequest(address string, version string, platform string) (*ProviderPackageMetadataRequest, error) {
	pAddr, err := parseProviderAddress(address)
	if err != nil {
		return nil, err
	}

	os, arch, err := parseProviderPlatform(platform)
	if err != nil {
		return nil, err
	}

	metadataReq := &ProviderPackageMetadataRequest{
		Namespace: pAddr.Namespace,
		Type:      pAddr.Type,
		Version:   version,
		OS:        os,
		Arch:      arch,
	}

	return metadataReq, nil
}

// buildProviderVersion calculates hash values from the ProviderPackageMetadataResponse
// and returns an instance of the ProviderVersion.
// Note that while OpenTofu Registry responses include both h1 and zh hashes, Terraform Registry responses include only the zh hash.
func buildProviderVersionFromPackageMetadata(address string, version string, res *ProviderPackageMetadataResponse) (*ProviderVersion, error) {
	h1Hashes := make(map[string]string)
	zhHashes := make(map[string]string)

	for platform, pkg := range res.Packages {
		// Historically, the zh hash in the Terraform Registry contains `manifest.json`,
		// so the key for the `ProviderVersion` map is `filename`, not `platform`.
		// To ensure the same results with Terraform and OpenTofu,
		// we need to build filename for each platform.
		// e.g.) darwin_arm64 => terraform-provider-null_3.2.1_darwin_arm64.zip
		pAddr, err := parseProviderAddress(address)
		if err != nil {
			return nil, err
		}
		filename := fmt.Sprintf("terraform-provider-%s_%s_%s.zip", pAddr.Type, version, platform)
		for _, h := range pkg.Hashes {
			if strings.HasPrefix(h, "h1:") {
				h1Hashes[filename] = h
			} else if strings.HasPrefix(h, "zh:") {
				zhHashes[filename] = h
			} else {
				return nil, fmt.Errorf("unknown hash type: %s", h)
			}
		}
	}

	platforms := slices.Sorted(maps.Keys(res.Packages))
	pv := &ProviderVersion{
		address:   address,
		version:   version,
		platforms: platforms,
		h1Hashes:  h1Hashes,
		zhHashes:  zhHashes,
	}

	return pv, nil
}

// parseProviderAddress parses a provider address and returns an instance of tfaddr.Provider.
// The provider address is expected to be in the format of "namespace/type", such as "hashicorp/null".
func parseProviderAddress(address string) (*tfaddr.Provider, error) {
	// We parse an provider address by using the terraform-registry-address
	// library to support fully qualified addresses such as
	// registry.terraform.io/hashicorp/null in the future, but note that the
	// current ProviderLockClient implementation only supports the public
	// standard registry (registry.terraform.io).
	pAddr, err := tfaddr.ParseProviderSource(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provider aaddress: %s", address)
	}

	// Since .terraform.lock.hcl was introduced from v0.14, we assume that
	// provider address is qualified with namespaces at least. We won't support
	// implicit legacy things.
	if !pAddr.HasKnownNamespace() {
		return nil, fmt.Errorf("failed to parse unknown provider aaddress: %s", address)
	}
	if pAddr.IsLegacy() {
		return nil, fmt.Errorf("failed to parse legacy provider aaddress: %s", address)
	}

	return &pAddr, nil
}

// parseProviderPlatform parses a platform name and returns the operating system and CPU architecture.
// The platform name is expected to be in the format of "os_arch", such as "darwin_arm64".
func parseProviderPlatform(platform string) (string, string, error) {
	pf := strings.Split(platform, "_")
	if len(pf) != 2 {
		return "", "", fmt.Errorf("failed to parse platform: %s", platform)
	}

	os := pf[0]
	arch := pf[1]
	return os, arch, nil
}

// newProviderDownloadRequest is a helper function for building the parameters for downloading provider.
// address is a provider address such as hashicorp/null.
// version is a version number such as 3.2.1.
// platform is a target platform name such as darwin_arm64.
func newProviderDownloadRequest(address string, version string, platform string) (*ProviderDownloadRequest, error) {
	pAddr, err := parseProviderAddress(address)
	if err != nil {
		return nil, err
	}

	os, arch, err := parseProviderPlatform(platform)
	if err != nil {
		return nil, err
	}

	req := &ProviderDownloadRequest{
		Namespace: pAddr.Namespace,
		Type:      pAddr.Type,
		Version:   version,
		OS:        os,
		Arch:      arch,
	}

	return req, nil
}

// buildProviderVersionFromDownload calculates hash values from the ProviderDownloadResponse
// and returns an instance of the ProviderVersion.
func buildProviderVersionFromDownload(address string, version string, platform string, res *ProviderDownloadResponse) (*ProviderVersion, error) {
	h1Hashes := make(map[string]string)

	h1, err := zipDataToH1Hash(res.zipData)
	if err != nil {
		return nil, err
	}
	h1Hashes[res.filename] = h1

	zhHashes, err := shaSumsDataToZhHash(res.shaSumsData)
	if err != nil {
		return nil, err
	}

	pv := &ProviderVersion{
		address:   address,
		version:   version,
		platforms: []string{platform},
		h1Hashes:  h1Hashes,
		zhHashes:  zhHashes,
	}

	return pv, nil
}
