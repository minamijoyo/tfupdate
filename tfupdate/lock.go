package tfupdate

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/minamijoyo/tfupdate/lock"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"github.com/zclconf/go-cty/cty"
)

// LockUpdater is a updater implementation which updates the dependency lock file.
type LockUpdater struct {
	platforms []string

	// index is a cached index for updating dependency lock files.
	index lock.Index

	// tfregistryConfig is a configuration for Terraform Registry API.
	tfregistryConfig tfregistry.Config
}

// NewLockUpdater is a factory method which returns an LockUpdater instance.
func NewLockUpdater(platforms []string, tfregistryConfig tfregistry.Config) (Updater, error) {
	// Create a new index with the provided registry config
	index, err := lock.NewIndexFromConfig(tfregistryConfig)
	if err != nil {
		return nil, err
	}

	return &LockUpdater{
		platforms:        platforms,
		index:            index,
		tfregistryConfig: tfregistryConfig,
	}, nil
}

// Update updates the dependency lock file.
// Note that this method will rewrite the AST passed as an argument.
func (u *LockUpdater) Update(ctx context.Context, mc *ModuleContext, filename string, f *hclwrite.File) error {
	if filepath.Base(filename) != ".terraform.lock.hcl" {
		// Skip other than the lock file.
		return nil
	}

	return u.updateLockfile(ctx, mc, f)
}

// updateLockfile updates the dependency lock file.
func (u *LockUpdater) updateLockfile(ctx context.Context, mc *ModuleContext, f *hclwrite.File) error {
	for _, p := range mc.SelecetedProviders() {
		pAddr, err := u.fullyQualifiedProviderAddress(p.Source)
		if err != nil {
			// Unsupported formats, such as legacy abbreviated notation, will result
			// in parse errors, but should be ignored without returning an error if
			// possible.
			log.Printf("[DEBUG] LockUpdater.updateLockfile: ignore legacy provider address notation: %s", p.Source)
			continue
		}

		pBlock := f.Body().FirstMatchingBlock("provider", []string{pAddr})
		if pBlock != nil {
			// update the existing provider block
			err := u.updateProviderBlock(ctx, pBlock, p)
			if err != nil {
				return err
			}
		} else {
			// create a new provider block
			f.Body().AppendNewline()
			pBlock = f.Body().AppendNewBlock("provider", []string{pAddr})

			err := u.updateProviderBlock(ctx, pBlock, p)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// updateProviderBlock updates the provider block in the dependency lock file.
func (u *LockUpdater) updateProviderBlock(ctx context.Context, pBlock *hclwrite.Block, p SelectedProvider) error {
	vAttr := pBlock.Body().GetAttribute("version")
	if vAttr != nil {
		// a version attribute found
		vVal := getAttributeValueAsUnquotedString(vAttr)
		log.Printf("[DEBUG] check provider version in lock file: address = %s, lock = %s, config = %s", p.Source, vVal, p.Version)
		if vVal == p.Version {
			// Avoid unnecessary recalculations if no version change
			return nil
		}
	}

	pBlock.Body().SetAttributeValue("version", cty.StringVal(p.Version))

	//Strictly speaking, constraints can contain multiple constraint expressions,
	//including comparison operators, but in the tfupdate use case, we assume
	//that the required_providers are pinned to a specific version to detect the
	//required version without terraform init, so we can simply specify the
	//constraints attribute as the same as the version. This may differ from what
	//terraform generates, but we expect that it doesn't matter in practice.
	pBlock.Body().SetAttributeValue("constraints", cty.StringVal(p.Version))

	// Calculate the hash value of the provider.
	// Note that the provider will be downloaded if cache miss.
	pv, err := u.index.GetOrCreateProviderVersion(ctx, p.Source, p.Version, u.platforms)
	if err != nil {
		return err
	}

	hashes := pv.AllHashes()
	pBlock.Body().SetAttributeRaw("hashes", tokensForListPerLine(hashes))

	return nil
}

// fullyQualifiedProviderAddress converts the short form of the provider
// address into the fully qualified form.
// Example: hashicorp/null => registry.terraform.io/hashicorp/null
// If BaseURL is set (e.g., https://registry.opentofu.org/), it will use its hostname
// instead of the default one (e.g., hashicorp/null => registry.opentofu.org/hashicorp/null).
func (u *LockUpdater) fullyQualifiedProviderAddress(address string) (string, error) {
	pAddr, err := tfaddr.ParseProviderSource(address)
	if err != nil {
		return "", fmt.Errorf("failed to parse provider address: %s", address)
	}

	// Since .terraform.lock.hcl was introduced from v0.14, we assume that
	// provider address is qualified with namespaces at least. We won't support
	// implicit legacy things.
	if !pAddr.HasKnownNamespace() {
		return "", fmt.Errorf("failed to parse unknown provider address: %s", address)
	}
	if pAddr.IsLegacy() {
		return "", fmt.Errorf("failed to parse legacy provider address: %s", address)
	}

	// If BaseURL is set, use its hostname
	if u.tfregistryConfig.BaseURL != "" {
		baseURL, err := url.Parse(u.tfregistryConfig.BaseURL)
		if err == nil && baseURL.Hostname() != "" {
			// Use the hostname from BaseURL with type casting to svchost.Hostname
			pAddr.Hostname = svchost.Hostname(baseURL.Hostname())
		}
	}

	return pAddr.String(), nil
}
