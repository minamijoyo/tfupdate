module github.com/minamijoyo/tfupdate

go 1.13

require (
	github.com/google/go-github/v28 v28.1.1
	github.com/goreleaser/goreleaser v0.119.0
	github.com/hashicorp/hcl/v2 v2.0.1-0.20191111231842-63cdbee39ce7
	github.com/hashicorp/logutils v1.0.0
	github.com/mitchellh/cli v1.0.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/afero v1.2.2
	github.com/spf13/pflag v1.0.5
	github.com/zclconf/go-cty v1.1.0
	golang.org/x/lint v0.0.0-20190930215403-16217165b5de
)

// Fix invalid pseudo-version: revision is longer than canonical (b0274f40d4c7)
replace github.com/go-macaron/cors => github.com/go-macaron/cors v0.0.0-20190925001837-b0274f40d4c7
