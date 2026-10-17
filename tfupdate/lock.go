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
	"github.com/minamijoyo/tfupdate/release"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"github.com/zclconf/go-cty/cty"
)

// LockUpdater is a updater implementation which updates the dependency lock file.
type LockUpdater struct {
	platforms []string

	// index is a cached index for updating dependency lock files.
	index lock.Index

	// registryIndexes caches hashes for registries identified by lock file entries.
	registryIndexes map[string]lock.Index

	// tfregistryConfig is a configuration for Terraform Registry API.
	tfregistryConfig tfregistry.Config

	// newProviderRelease is a factory for provider release sources.
	newProviderRelease func(string, tfregistry.Config) (release.Release, error)
}

// NewLockUpdater is a factory method which returns a LockUpdater instance.
func NewLockUpdater(platforms []string, tfregistryConfig tfregistry.Config) (Updater, error) {
	// Create a new index with the provided registry config
	index, err := lock.NewIndexFromConfig(tfregistryConfig)
	if err != nil {
		return nil, err
	}

	return &LockUpdater{
		platforms:          platforms,
		index:              index,
		tfregistryConfig:   tfregistryConfig,
		newProviderRelease: release.NewTFRegistryProviderRelease,
	}, nil
}

// Update updates the dependency lock file.
// Note that this method will rewrite the AST passed as an argument.
func (u *LockUpdater) Update(ctx context.Context, mc *ModuleContext, filename string, f *hclwrite.File) error {
	if filepath.Base(filename) != ".terraform.lock.hcl" {
		// Skip other than the lock file.
		return nil
	}

	if mc.Option().UseLockfile {
		return u.updateProvidersFromLockfile(ctx, f)
	}

	return u.updateLockfile(ctx, mc, f)
}

// updateLockfile updates the dependency lock file.
func (u *LockUpdater) updateLockfile(ctx context.Context, mc *ModuleContext, f *hclwrite.File) error {
	for _, p := range mc.SelectedProviders() {
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
			err := u.updateProviderBlock(ctx, pBlock, p, false)
			if err != nil {
				return err
			}
		} else {
			// create a new provider block
			f.Body().AppendNewline()
			pBlock = f.Body().AppendNewBlock("provider", []string{pAddr})

			err := u.updateProviderBlock(ctx, pBlock, p, false)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// updateProvidersFromLockfile selects provider versions using only existing lock entries.
func (u *LockUpdater) updateProvidersFromLockfile(ctx context.Context, f *hclwrite.File) error {
	for _, pBlock := range allMatchingBlocksByType(f.Body(), "provider") {
		labels := pBlock.Labels()
		if len(labels) != 1 {
			return fmt.Errorf("expected one source address in lock file provider block")
		}
		source := labels[0]
		pAddr, err := lock.ParseProviderAddress(source)
		if err != nil {
			return err
		}

		constraint := ""
		if attr := pBlock.Body().GetAttribute("constraints"); attr != nil {
			constraint = getAttributeValueAsUnquotedString(attr)
		}
		config := u.tfregistryConfig
		if config.BaseURL == "" {
			config.BaseURL = fmt.Sprintf("https://%s/", pAddr.Hostname)
		}
		r, err := u.newProviderRelease(pAddr.Namespace+"/"+pAddr.Type, config)
		if err != nil {
			return err
		}
		latest, err := release.LatestMatching(ctx, r, constraint)
		if err != nil {
			return fmt.Errorf("failed to select provider version for %s: %w", source, err)
		}

		index := u.index
		if u.tfregistryConfig.BaseURL == "" && pAddr.Hostname != tfaddr.DefaultProviderRegistryHost {
			if u.registryIndexes == nil {
				u.registryIndexes = make(map[string]lock.Index)
			}
			index = u.registryIndexes[config.BaseURL]
			if index == nil {
				index, err = lock.NewIndexFromConfig(config)
				if err != nil {
					return err
				}
				u.registryIndexes[config.BaseURL] = index
			}
		}
		providerUpdater := &LockUpdater{platforms: u.platforms, index: index}
		p := SelectedProvider{Source: source, Version: latest}
		if err := providerUpdater.updateProviderBlock(ctx, pBlock, p, true); err != nil {
			return err
		}
	}

	return nil
}

// updateProviderBlock updates the provider block in the dependency lock file.
func (u *LockUpdater) updateProviderBlock(ctx context.Context, pBlock *hclwrite.Block, p SelectedProvider, preserveConstraints bool) error {
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
	if !preserveConstraints {
		pBlock.Body().SetAttributeValue("constraints", cty.StringVal(p.Version))
	}

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
	pAddr, err := lock.ParseProviderAddress(address)
	if err != nil {
		return "", err
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
