package tfupdate

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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

	// catalogs contains successful catalog responses for this updater only.
	catalogs map[string][]tfregistry.ProviderVersion
}

// NewLockUpdater is a factory method which returns a LockUpdater instance.
func NewLockUpdater(platforms []string, tfregistryConfig tfregistry.Config) (Updater, error) {
	// Create a new index with the provided registry config
	index, err := lock.NewIndexFromConfig(tfregistryConfig)
	if err != nil {
		return nil, err
	}

	// Catalog selection and checksum work use the same distinct target set.
	var targets []string
	seen := make(map[string]bool, len(platforms))
	for _, platform := range platforms {
		if !seen[platform] {
			seen[platform] = true
			targets = append(targets, platform)
		}
	}

	return &LockUpdater{
		platforms:        targets,
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

// updateLockfile resolves and prepares every provider before changing the AST.
func (u *LockUpdater) updateLockfile(ctx context.Context, mc *ModuleContext, f *hclwrite.File) error {
	plans, err := u.planProviderLocks(ctx, mc, f)
	if err != nil {
		return u.lockfileError(mc, err)
	}
	working, err := prepareLockfile(f, plans)
	if err != nil {
		return u.lockfileError(mc, err)
	}
	for _, plan := range plans {
		block := working.Body().FirstMatchingBlock("provider", []string{plan.address})
		if block == nil {
			working.Body().AppendNewline()
			block = working.Body().AppendNewBlock("provider", []string{plan.address})
		}
		block.Body().SetAttributeValue("version", cty.StringVal(plan.version))
		block.Body().SetAttributeValue("constraints", cty.StringVal(plan.constraints))
		if plan.replaceHashes {
			block.Body().SetAttributeRaw("hashes", tokensForListPerLine(plan.hashes))
		}
	}
	*f = *working
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

	// Since .terraform.lock.hcl was introduced in v0.14, we assume that
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

// prepareLockfile expands affected block boundaries without changing the input
// tree. HCL single-line blocks cannot accept additional attributes as-is.
func prepareLockfile(f *hclwrite.File, plans []providerLockPlan) (*hclwrite.File, error) {
	before, after := make(map[*hclwrite.Token]bool), make(map[*hclwrite.Token]bool)
	for _, plan := range plans {
		block := f.Body().FirstMatchingBlock("provider", []string{plan.address})
		if block == nil {
			continue
		}
		tokens := block.BuildTokens(nil)
		for i, token := range tokens {
			if token.Type == hclsyntax.TokenOBrace {
				if tokens[i+1].Type != hclsyntax.TokenNewline {
					after[token] = true
				}
				break
			}
		}
		for i := len(tokens) - 1; i >= 0; i-- {
			if tokens[i].Type == hclsyntax.TokenCBrace {
				if tokens[i-1].Type != hclsyntax.TokenNewline {
					before[tokens[i]] = true
				}
				break
			}
		}
	}
	tokens := hclwrite.Tokens{}
	for _, token := range f.BuildTokens(nil) {
		if before[token] {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
		tokens = append(tokens, token)
		if after[token] {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}
	}
	working, diags := hclwrite.ParseConfig(tokens.Bytes(), ".terraform.lock.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("invalid lockfile syntax: %s", diags)
	}
	return working, nil
}
