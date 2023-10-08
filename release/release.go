package release

import (
	"context"
	"errors"
)

// Release is an interface which provides version information of a module or provider.
type Release interface {
	// ListReleases returns a list of unsorted all releases including pre-release.
	ListReleases(ctx context.Context) ([]string, error)
}

// Latest returns the latest release.
// Note that GetLatestRelease API in GitHub and GitLab returns the most recent
// release, which doesn't means the latest stable release. I'm not sure it also
// affects Terraform Registry but I think we should use the same strategy for
// consistency. So we sort versions in semver order and find the latest non
// pre-release.
func Latest(ctx context.Context, r Release) (string, error) {
	versions, err := List(ctx, r, 1, false)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errors.New("no releases found")
	}

	return versions[0], nil
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
