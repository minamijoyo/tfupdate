package command

// Env is a set of configurations read from environment variables.
type Env struct {
	// GitHubBaseURL is a base URL for GtiHub API requests.
	// Defaults to the public GitHub API.
	GitHubBaseURL string `envconfig:"GITHUB_BASE_URL" default:"https://api.github.com/"`
	// GitHubToken is a personal access token for GitHub.
	// This allows access to a private repository.
	GitHubToken string `envconfig:"GITHUB_TOKEN"`
	// GitLabBaseURL is a base URL for GitLab API requests.
	// Defaults to the public GitLab API.
	GitLabBaseURL string `envconfig:"GITLAB_BASE_URL" default:"https://gitlab.com/api/v4/"`
	// GitLabToken is a personal access token for GitLab.
	// This is needed for public and private projects on all instances.
	GitLabToken string `envconfig:"GITLAB_TOKEN"`
	// TFRegistryBaseURL is a base URL for Terraform registry.
	// Defaults to the public Terraform registry.
	// To use the public OpenTofu registry, set this to `https://registry.opentofu.org/`.
	TFRegistryBaseURL string `envconfig:"TFREGISTRY_BASE_URL" default:"https://registry.terraform.io/"`
}
