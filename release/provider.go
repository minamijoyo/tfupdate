package release

import "fmt"

// OfficialProviderRelease is a release implementation which provides version information with GitHub Release.
type OfficialProviderRelease struct {
	gh *GitHubRelease
}

// NewOfficialProviderRelease is a factory method which returns an OfficialProviderRelease instance.
func NewOfficialProviderRelease(name string) (Release, error) {
	owner := "terraform-providers"
	repo := fmt.Sprintf("terraform-provider-%s", name)
	r, err := NewGitHubRelease(owner, repo)
	if err != nil {
		return nil, err
	}

	return &OfficialProviderRelease{
		gh: r.(*GitHubRelease),
	}, nil
}

// Latest returns a latest version.
func (r *OfficialProviderRelease) Latest() (string, error) {
	return r.gh.Latest()
}
