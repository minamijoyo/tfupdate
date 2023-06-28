package lock

import (
	"context"
	"fmt"
	"log"
	"strings"

	tfaddr "github.com/hashicorp/terraform-registry-address"
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

	// papi is a ProviderDownloaderAPI interface implementation used for downloading provider.
	papi ProviderDownloaderAPI
}

// NewDefaultIndex returns a new instance of default Index.
func NewDefaultIndex() (Index, error) {
	client, err := NewProviderDownloaderClient(TFRegistryConfig{})
	if err != nil {
		return nil, err
	}

	index := NewIndex(client)

	return index, nil
}

// NewIndex returns a new instance of Index.
func NewIndex(papi ProviderDownloaderAPI) Index {
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

	// papi is a ProviderDownloaderAPI interface implementation used for downloading provider.
	papi ProviderDownloaderAPI
}

// newProviderIndex returns a new instance of providerIndex.
func newProviderIndex(address string, papi ProviderDownloaderAPI) *providerIndex {
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
		pv, err := buildProviderVersion(pi.address, version, platform, res)
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

// newProviderDownloadRequest is a helper function for building the parameters for downloading provider.
// address is a provider address such as hashicorp/null.
// version is a version number such as 3.2.1.
// platform is a target platform name such as darwin_arm64.
func newProviderDownloadRequest(address string, version string, platform string) (*ProviderDownloadRequest, error) {
	// We parse an provider address by using the terraform-registry-address
	// library to support fully qualified addresses such as
	// registry.terraform.io/hashicorp/null in the future, but note that the
	// current ProviderDownloaderClient implementation only supports the public
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

	pf := strings.Split(platform, "_")
	if len(pf) != 2 {
		return nil, fmt.Errorf("failed to parse platform: %s", platform)
	}
	os := pf[0]
	arch := pf[1]

	req := &ProviderDownloadRequest{
		Namespace: pAddr.Namespace,
		Type:      pAddr.Type,
		Version:   version,
		OS:        os,
		Arch:      arch,
	}

	return req, nil
}

// buildProviderVersion calculates hash values from the ProviderDownloadResponse
// and returns an instance of the ProviderVersion.
func buildProviderVersion(address string, version string, platform string, res *ProviderDownloadResponse) (*ProviderVersion, error) {
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
