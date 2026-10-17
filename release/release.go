package release

import (
	"context"
	"errors"
	"fmt"
	"math"

	version "github.com/hashicorp/go-version"
)

// Release is an interface which provides version information of a module or provider.
type Release interface {
	// ListReleases returns a list of unsorted all releases including pre-release.
	ListReleases(ctx context.Context) ([]string, error)
}

// Latest returns the latest release.
// Note that GetLatestRelease API in GitHub and GitLab returns the most recent
// release, which doesn't mean the latest stable release. I'm not sure if it also
// affects Terraform Registry but I think we should use the same strategy for
// consistency. So we sort versions in semver order and find the latest non
// pre-release.
func Latest(ctx context.Context, r Release) (string, error) {
	return LatestMatching(ctx, r, "")
}

// LatestMatching returns the latest stable release that satisfies constraint.
// An empty constraint allows any stable version.
func LatestMatching(ctx context.Context, r Release, constraint string) (string, error) {
	var constraints version.Constraints
	if constraint != "" {
		var err error
		constraints, err = version.NewConstraint(constraint)
		if err != nil {
			return "", fmt.Errorf("invalid version constraints %q: %w", constraint, err)
		}
	}

	versions, err := List(ctx, r, math.MaxInt, false)
	if err != nil {
		return "", err
	}

	for i := len(versions) - 1; i >= 0; i-- {
		v, err := version.NewVersion(versions[i])
		if err != nil {
			return "", err
		}
		if constraints.Check(v) {
			return versions[i], nil
		}
	}

	if constraint == "" {
		return "", errors.New("no releases found")
	}
	return "", fmt.Errorf("no stable release found matching constraints %q", constraint)
}

// List returns a list of releases in semver order.
// If preRelease is set to false, the result doesn't contain pre-releases.
func List(ctx context.Context, r Release, maxLength int, preRelease bool) ([]string, error) {
	res, err := r.ListReleases(ctx)
	if err != nil {
		return nil, err
	}

	versions := toVersions(res)
	sorted := sortVersions(versions)
	rels := sorted

	if !preRelease {
		rels = excludePreReleases(sorted)
	}

	releases := fromVersions(rels)
	start := len(releases) - minInt(maxLength, len(releases))
	return releases[start:], nil
}
