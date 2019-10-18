// +build tools

package tools

import (
	_ "github.com/goreleaser/goreleaser" // executable dependency for development
	_ "golang.org/x/lint/golint"         // executable dependency for development
)
