module github.com/minamijoyo/tfupdate

go 1.12

require (
	github.com/hashicorp/hcl2 v0.0.0-20190809210004-72d32879a5c5
	github.com/hashicorp/logutils v1.0.0
	github.com/mitchellh/cli v1.0.0
	github.com/pkg/errors v0.8.1
	github.com/zclconf/go-cty v1.0.0
)

replace github.com/hashicorp/hcl2 => github.com/minamijoyo/hcl2 v0.0.0-20190817150234-1aba4ac822ee
