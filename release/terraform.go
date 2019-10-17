package release

// TerraformRelease is a release implementation which provides version information with GitHub Release.
type TerraformRelease struct {
	gh *GitHubRelease
}

// NewTerraformRelease is a factory method which returns an TerraformRelease instance.
func NewTerraformRelease() (Release, error) {
	r, err := NewGitHubRelease("hashicorp", "terraform")
	if err != nil {
		return nil, err
	}

	return &TerraformRelease{
		gh: r.(*GitHubRelease),
	}, nil
}

// Latest returns a latest version.
func (r *TerraformRelease) Latest() (string, error) {
	return r.gh.Latest()
}
