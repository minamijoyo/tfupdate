package command

// Env is a set of configurations read from environment variables.
type Env struct {
	// GitHubBaseURL is a URL for GtiHub API requests.
	// Defaults to the public GitHub API.
	GitHubBaseURL string `envconfig:"GITHUB_BASE_URL" default:"https://api.github.com/"`
	// GitHubToken is a personal access token for GitHub.
	// This allows access to a private repository.
	GitHubToken string `envconfig:"GITHUB_TOKEN"`
}
