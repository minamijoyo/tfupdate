package tfupdate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"github.com/zclconf/go-cty/cty"
)

// providerConstraints groups declarations by their effective lockfile identity.
type providerConstraints struct {
	address string
	source  string
	clauses []string
}

type providerLockPlan struct {
	address       string
	version       string
	constraints   string
	hashes        []string
	replaceHashes bool
}

// collectProviderConstraints keeps unconstrained and legacy declarations out of
// resolution, but combines all supported declarations before choosing a version.
func (u *LockUpdater) collectProviderConstraints(mc *ModuleContext) []providerConstraints {
	groups := make(map[string]*providerConstraints)
	for _, name := range slices.Sorted(maps.Keys(mc.requiredProviders)) {
		requirement := mc.requiredProviders[name]
		if len(requirement.VersionConstraints) == 0 {
			continue
		}
		address, err := u.fullyQualifiedProviderAddress(requirement.Source)
		if err != nil {
			continue
		}
		group, ok := groups[address]
		if !ok {
			group = &providerConstraints{address: address, source: strings.Join(strings.Split(address, "/")[1:], "/")}
			groups[address] = group
		}
		for _, expression := range requirement.VersionConstraints {
			for _, clause := range strings.Split(expression, ",") {
				group.clauses = append(group.clauses, strings.TrimSpace(clause))
			}
		}
	}
	result := make([]providerConstraints, 0, len(groups))
	for _, address := range slices.Sorted(maps.Keys(groups)) {
		group := groups[address]
		slices.Sort(group.clauses)
		group.clauses = slices.Compact(group.clauses)
		result = append(result, *group)
	}
	return result
}

// parseProviderConstraints validates all clauses, including those following an
// exact pin. An equality pin avoids a registry listing, not other constraints.
func parseProviderConstraints(clauses []string) (version.Constraints, *version.Version, error) {
	var constraints version.Constraints
	var pin *version.Version
	for _, clause := range clauses {
		parsed, err := version.NewConstraint(clause)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid constraint %q: %w", clause, err)
		}
		constraints = append(constraints, parsed...)
		literal := clause
		if strings.HasPrefix(literal, "=") {
			literal = strings.TrimSpace(strings.TrimPrefix(literal, "="))
		}
		exact, err := version.NewVersion(literal)
		if err != nil {
			// Comparisons and pessimistic constraints are not exact pins.
			continue
		}
		if pin == nil {
			pin = exact
		}
	}
	if pin != nil && !constraints.Check(pin) {
		return nil, nil, fmt.Errorf("pinned version %s does not satisfy %s", pin, constraints)
	}
	return constraints, pin, nil
}

// providerCatalog caches only successful responses. The configured client uses
// the same endpoint rules as the checksum index, rather than release discovery.
func (u *LockUpdater) providerCatalog(ctx context.Context, address string) ([]tfregistry.ProviderVersion, error) {
	if versions, ok := u.catalogs[address]; ok {
		return versions, nil
	}
	// address was validated while collecting declarations.
	parts := strings.Split(address, "/")
	// NewLockUpdater validated this immutable configuration through the same factory.
	client, _ := tfregistry.NewClient(u.tfregistryConfig)
	response, err := client.ListProviderVersions(ctx, &tfregistry.ListProviderVersionsRequest{
		Namespace: parts[1], Type: parts[2],
	})
	if err != nil {
		return nil, err
	}
	versions := response.Versions
	if u.catalogs == nil {
		u.catalogs = make(map[string][]tfregistry.ProviderVersion)
	}
	u.catalogs[address] = versions
	return versions, nil
}

func (u *LockUpdater) resolveProviderVersion(ctx context.Context, group providerConstraints) (string, error) {
	constraints, pin, err := parseProviderConstraints(group.clauses)
	if err != nil {
		return "", err
	}
	if pin != nil {
		return pin.String(), nil
	}
	catalog, err := u.providerCatalog(ctx, group.address)
	if err != nil {
		return "", fmt.Errorf("list available versions: %w", err)
	}
	var selected *version.Version
	for _, entry := range compatibleProviderVersions(catalog, u.platforms) {
		candidate := entry
		if !constraints.Check(candidate) {
			continue
		}
		if selected == nil || candidate.GreaterThan(selected) ||
			(candidate.Equal(selected) && candidate.String() > selected.String()) {
			selected = candidate
		}
	}
	if selected == nil {
		return "", fmt.Errorf("no available version satisfies %s", constraints)
	}
	return selected.String(), nil
}

// planProviderLocks completes fallible work before the caller changes any block.
func (u *LockUpdater) planProviderLocks(ctx context.Context, mc *ModuleContext, f *hclwrite.File) ([]providerLockPlan, error) {
	if mc.inspectionError != nil {
		return nil, fmt.Errorf("module %s: %w", mc.dir, mc.inspectionError)
	}
	groups := u.collectProviderConstraints(mc)
	plans := make([]providerLockPlan, 0, len(groups))
	for _, group := range groups {
		selected, err := u.resolveProviderVersion(ctx, group)
		if err != nil {
			return nil, fmt.Errorf("module %s provider %s: %w", mc.dir, group.address, err)
		}
		plan := providerLockPlan{
			address:       group.address,
			version:       selected,
			constraints:   strings.Join(group.clauses, ", "),
			replaceHashes: true,
		}
		block := f.Body().FirstMatchingBlock("provider", []string{group.address})
		if block != nil {
			attr, err := getHCLNativeAttribute(block.Body(), "version")
			if err != nil {
				return nil, fmt.Errorf("module %s provider %s locked version: %w", mc.dir, group.address, err)
			}
			if attr != nil {
				lockedValue, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					return nil, fmt.Errorf("module %s provider %s locked version: %w", mc.dir, group.address, diags)
				}
				if lockedValue.Type() != cty.String || lockedValue.IsNull() || !lockedValue.IsKnown() {
					return nil, fmt.Errorf("module %s provider %s locked version must be a known non-null string", mc.dir, group.address)
				}
				locked := lockedValue.AsString()
				lockedVersion, err := version.NewVersion(locked)
				if err != nil {
					return nil, fmt.Errorf("module %s provider %s invalid locked version %q: %w", mc.dir, group.address, locked, err)
				}
				if lockedVersion.String() == selected {
					plan.replaceHashes = false
				}
			}
		}
		if plan.replaceHashes {
			// The checksum index uses namespace/type with the configured endpoint.
			pv, err := u.index.GetOrCreateProviderVersion(ctx, group.source, selected, u.platforms)
			if err != nil {
				return nil, fmt.Errorf("module %s provider %s version %s checksums: %w", mc.dir, group.address, selected, err)
			}
			plan.hashes = pv.AllHashes()
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// compatibleProviderVersions merges catalog fragments before checking platform
// coverage. Build metadata remains part of the normalized version identity.
func compatibleProviderVersions(catalog []tfregistry.ProviderVersion, platforms []string) []*version.Version {
	type advertisement struct {
		version   *version.Version
		platforms map[string]bool
	}
	versions := make(map[string]*advertisement)
	for _, item := range catalog {
		parsed, err := version.NewVersion(item.Version)
		if err != nil {
			continue
		}
		normalized := parsed.String()
		entry, ok := versions[normalized]
		if !ok {
			entry = &advertisement{version: parsed, platforms: make(map[string]bool)}
			versions[normalized] = entry
		}
		for _, platform := range item.Platforms {
			entry.platforms[platform.OS+"_"+platform.Arch] = true
		}
	}
	result := make([]*version.Version, 0, len(versions))
	for _, entry := range versions {
		compatible := true
		for _, platform := range platforms {
			if !entry.platforms[platform] {
				compatible = false
				break
			}
		}
		if compatible {
			result = append(result, entry.version)
		}
	}
	return result
}
