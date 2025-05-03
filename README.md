# tfupdate
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/minamijoyo/tfupdate.svg)](https://github.com/minamijoyo/tfupdate/releases/latest)
[![GoDoc](https://godoc.org/github.com/minamijoyo/tfupdate/tfupdate?status.svg)](https://godoc.org/github.com/minamijoyo/tfupdate)

## Features

- Update version constraints of Terraform core, OpenTofu core, providers, and modules
- Update dependency lock files (.terraform.lock.hcl) without Terraform / OpenTofu CLI
- Update all your Terraform / OpenTofu configurations and lock files recursively under a given directory
- Get the latest release version from the GitHub, GitLab, Terraform Registry, or OpenTofu Registry
- Terraform v0.12+ / OpenTofu v1.6+ support

If you integrate tfupdate with your favorite CI or job scheduler, you can check the latest release daily and create a Pull Request automatically.

## Why?
It is a best practice to break your Terraform configuration and state into small pieces to minimize the impact of an accident.
It is also recommended to lock versions of Terraform core, providers and modules to avoid unexpected breaking changes.
If you decided to lock version constraints, you probably want to keep them up-to-date frequently to reduce the risk of version upgrade failures.
It's easy to update a single directory, but what if they are scattered across multiple directories?

That is why I wrote a tool which parses Terraform configurations and updates all version constraints at once.

## Install

### macOS

If you are a macOS user, you can install `tfupdate` via either [Homebrew](https://brew.sh) or [MacPorts](https://www.macports.org):

#### Homebrew

```bash
$ brew install minamijoyo/tfupdate/tfupdate
```

#### MacPorts

```bash
$ sudo port install tfupdate
```

### Download

Download the latest compiled binaries and put it anywhere in your executable path.

https://github.com/minamijoyo/tfupdate/releases

### Source

If you have Go 1.24+ development environment:

```
$ go install github.com/minamijoyo/tfupdate@latest
$ tfupdate --version
```

### Docker

You can also run it with Docker:

```
$ docker run -it --rm minamijoyo/tfupdate --version
```

## Usage

```
$ tfupdate --help
Usage: tfupdate [--version] [--help] <command> [<args>]

Available commands are:
    lock         Update dependency lock files
    module       Update version constraints for module
    opentofu     Update version constraints for opentofu
    provider     Update version constraints for provider
    release      Get release version information
    terraform    Update version constraints for terraform
```

### terraform

```
$ tfupdate terraform --help
Usage: tfupdate terraform [options] <PATH>

Arguments
  PATH               A path of file or directory to update

Options:
  -v  --version      A new version constraint (default: latest)
                     If the version is omitted, the latest version is automatically checked and set.
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
```

If you have `main.tf` like the following:

```
$ cat main.tf
terraform {
  required_version = "0.12.15"
}
```

Execute the following command:

```
$ tfupdate terraform -v 0.12.16 main.tf
```

```
$ cat main.tf
terraform {
  required_version = "0.12.16"
}
```

A value of version flag accepts any string literal. You can also pass a [version constraint](https://www.terraform.io/language/expressions/version-constraints):

```
$ tfupdate terraform -v "~> 1.0" main.tf

$ cat main.tf
terraform {
  required_version = "~> 1.0"
}
```

If you want to update all your Terraform configurations under the current directory recursively,
use `-r (--recursive)` option:

```
$ tfupdate terraform -v 0.12.16 -r ./
```

You can also ignore some path patterns with `-i (--ignore-path)` option:

```
$ tfupdate terraform -v 0.12.16 -i modules/ -r ./
```

If the version is omitted, the latest version is automatically checked and set.

```
$ tfupdate terraform -r ./
```

### opentofu

```
$ tfupdate opentofu --help
Usage: tfupdate opentofu [options] <PATH>

Arguments
  PATH               A path of file or directory to update

Options:
  -v  --version      A new version constraint (default: latest)
                     If the version is omitted, the latest version is automatically checked and set.
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
```

If you have `main.tf` like the following:

```
$ cat main.tf
terraform {
  required_version = "1.8.0"
}
```

Execute the following command:

```
$ tfupdate opentofu -v 1.9.0 main.tf
```

```
$ cat main.tf
terraform {
  required_version = "1.9.0"
}
```

### provider

```
$ tfupdate provider --help
Usage: tfupdate provider [options] <PROVIDER_NAME> <PATH>

Arguments
  PROVIDER_NAME      A name of provider (e.g. aws or integrations/github)
  PATH               A path of file or directory to update

Options:
  -v  --version      A new version constraint (default: latest)
                     If the version is omitted, the latest version is automatically checked and set.
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
```

```
$ cat main.tf
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "3.70.0"
    }
  }
}

$ tfupdate provider aws -v 3.74.0 main.tf

$ cat main.tf
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "3.74.0"
    }
  }
}
```

A value of version flag accepts any string literal. You can also pass a [version constraint](https://www.terraform.io/language/expressions/version-constraints):

```
$ tfupdate provider aws -v "~> 3.0" main.tf

$ cat main.tf
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}
```

For updating the dependency lock file (.terraform.lock.hcl), use the `tfupdate lock` command.

### module

```
$ tfupdate module --help
Usage: tfupdate module [options] <MODULE_NAME> <PATH>

Arguments
  MODULE_NAME        A name of module or a regular expression in RE2 syntax
                     e.g.
                       terraform-aws-modules/vpc/aws
                       git::https://example.com/vpc.git
                       git::https://example\.com/.+
  PATH               A path of file or directory to update

Options:
  -v  --version       A new version constraint (required)
                      Automatic latest version resolution is not currently supported for modules.
  -r  --recursive     Check a directory recursively (default: false)
  -i  --ignore-path   A regular expression for path to ignore
                      If you want to ignore multiple directories, set the flag multiple times.
  --source-match-type Define how to match MODULE_NAME to the module source URLs. Valid values are "full" or "regex". (default: full)
```

```
$ cat main.tf
module "s3_bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "2.14.0"

  bucket = "my-s3-bucket"
  acl    = "private"

  versioning = {
    enabled = true
  }
}

$ tfupdate module -v 2.14.1 terraform-aws-modules/s3-bucket/aws main.tf

$ cat main.tf
module "s3_bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "2.14.1"

  bucket = "my-s3-bucket"
  acl    = "private"

  versioning = {
    enabled = true
  }
}
```

A value of version flag accepts any string literal. You can also pass a [version constraint](https://www.terraform.io/language/expressions/version-constraints):

```
$ tfupdate module -v "~> 2.14.1" terraform-aws-modules/s3-bucket/aws main.tf

$ cat main.tf
module "s3_bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "~> 2.14.1"

  bucket = "my-s3-bucket"
  acl    = "private"

  versioning = {
    enabled = true
  }
}
```

### release

```
$ tfupdate release --help
Usage: tfupdate release <subcommand> [options] [args]

  This command has subcommands for release version information.

Subcommands:
    latest    Get the latest release version
    list      Get a list of release versions
```

```
$ tfupdate release latest --help
Usage: tfupdate release latest [options] <SOURCE>

Arguments
  SOURCE             A path of release data source.
                     Valid format depends on --source-type option.
                       - github or gitlab:
                         owner/repo
                         e.g. terraform-providers/terraform-provider-aws
                       - tfregistryModule:
                         namespace/name/provider
                         e.g. terraform-aws-modules/vpc/aws
                       - tfregistryProvider:
                         namespace/type
                         e.g. hashicorp/aws

Options:
  -s  --source-type  A type of release data source.
                     Valid values are
                       - github (default)
                       - gitlab
                       - tfregistryModule
                       - tfregistryProvider
```

```
$ tfupdate release latest terraform-providers/terraform-provider-aws
2.40.0
```

If you want to access private repositories on GitHub, export your access token to the `GITHUB_TOKEN` environment variable.

If you want to access public or private repositories on GitLab, export your access token with api permissions to the `GITLAB_TOKEN` environment variable. If you are using an instance that is not `https://gitlab.com`, set the correct base URL to the `GITLAB_BASE_URL` environment variable (defaults to `https://gitlab.com/api/v4/`).

If you want to use the public OpenTofu registry, set the `TFREGISTRY_BASE_URL` environment variable to `https://registry.opentofu.org/`.

```
$ tfupdate release list --help
Usage: tfupdate release list [options] <SOURCE>

Arguments
  SOURCE             A path of release data source.
                     Valid format depends on --source-type option.
                       - github or gitlab:
                         owner/repo
                         e.g. terraform-providers/terraform-provider-aws
                       - tfregistryModule:
                         namespace/name/provider
                         e.g. terraform-aws-modules/vpc/aws
                       - tfregistryProvider:
                         namespace/type
                         e.g. hashicorp/aws

Options:
  -s  --source-type  A type of release data source.
                     Valid values are
                       - github (default)
                       - gitlab
                       - tfregistryModule
                       - tfregistryProvider
  -n  --max-length   The maximum length of list.
```

```
$ tfupdate release list -n 5 hashicorp/terraform
0.12.17
0.12.18
0.12.19
0.12.20
0.12.21
```

### lock

The tfupdate lock command updates the dependency lock file (.terraform.lock.hcl).
For more information on the dependency lock file, see the official Terraform documentation:
https://developer.hashicorp.com/terraform/language/files/dependency-lock

```
$ tfupdate lock --help
Usage: tfupdate lock [options] <PATH>

Arguments
  PATH               A relative path of directory to update

Options:
      --platform     Specify a platform to update dependency lock files.
                     At least one or more --platform flags must be specified.
                     Use this option multiple times to include checksums for multiple target systems.
                     Target platform names consist of an operating system and a CPU architecture.
                     (e.g. linux_amd64, darwin_amd64, darwin_arm64)
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
```

If you want to use the public OpenTofu registry, set the `TFREGISTRY_BASE_URL` environment variable to `https://registry.opentofu.org/`.

```
$ export TFREGISTRY_BASE_URL=https://registry.opentofu.org/
```

Given the following configuration:

```
$ cat test-fixtures/lock/simple/main.tf
terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "3.1.1"
    }
  }
}
```

As you know, you can generate the dependency lock file by the terraform providers lock command:

```
$ terraform -chdir=test-fixtures/lock/simple providers lock -platform=linux_amd64 -platform=darwin_amd64 -platform=darwin_arm64
```

```
$ cat test-fixtures/lock/simple/.terraform.lock.hcl
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.1.1"
  constraints = "3.1.1"
  hashes = [
    "h1:71sNUDvmiJcijsvfXpiLCz0lXIBSsEJjMxljt7hxMhw=",
    "h1:Pctug/s/2Hg5FJqjYcTM0kPyx3AoYK1MpRWO0T9V2ns=",
    "h1:YvH6gTaQzGdNv+SKTZujU1O0bO+Pw6vJHOPhqgN8XNs=",
    "zh:063466f41f1d9fd0dd93722840c1314f046d8760b1812fa67c34de0afcba5597",
    "zh:08c058e367de6debdad35fc24d97131c7cf75103baec8279aba3506a08b53faf",
    "zh:73ce6dff935150d6ddc6ac4a10071e02647d10175c173cfe5dca81f3d13d8afe",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:8fdd792a626413502e68c195f2097352bdc6a0df694f7df350ed784741eb587e",
    "zh:976bbaf268cb497400fd5b3c774d218f3933271864345f18deebe4dcbfcd6afa",
    "zh:b21b78ca581f98f4cdb7a366b03ae9db23a73dfa7df12c533d7c19b68e9e72e5",
    "zh:b7fc0c1615dbdb1d6fd4abb9c7dc7da286631f7ca2299fb9cd4664258ccfbff4",
    "zh:d1efc942b2c44345e0c29bc976594cb7278c38cfb8897b344669eafbc3cddf46",
    "zh:e356c245b3cd9d4789bab010893566acace682d7db877e52d40fc4ca34a50924",
    "zh:ea98802ba92fcfa8cf12cbce2e9e7ebe999afbf8ed47fa45fc847a098d89468b",
    "zh:eff8872458806499889f6927b5d954560f3d74bf20b6043409edf94d26cd906f",
  ]
}
```

When updating provider version, the lock file must also be updated:

```
$ tfupdate provider null -v 3.2.1 ./test-fixtures/lock/simple/
```

```
$ cat test-fixtures/lock/simple/main.tf
terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }
  }
}
```

You can update the lock file by the tfupdate lock command without Terraform CLI:

```
$ tfupdate lock --platform=linux_amd64 --platform=darwin_amd64 --platform=darwin_arm64 ./test-fixtures/lock/simple/
```

Note that unlike the terraform providers lock command, the `--platform` flag requires two hyphens.

```
$ cat test-fixtures/lock/simple/.terraform.lock.hcl
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}
```

The tfupdate lock command parses the `required_providers` block in your configuration, downloads provider packages and calculates hash values under the hood. The most important point is that it caches calculated hash values in memory, which gives us a huge performance advantage when updating multiple directories at once using the `-r (--recursive)` option.

To skip terraform init, we assume that all dependencies are pinned to a specific version in the required_providers block of the root module. Note that version constraint expressions or indirect dependencies via modules are not supported and ignored.

## Keep your dependencies up-to-date

If you integrate tfupdate with your favorite CI or job scheduler, you can check the latest release daily and create a Pull Request automatically.

An example for tfupdate with CircleCI is available:

https://github.com/minamijoyo/tfupdate-circleci-example

You can also use a CircleCI orb:

https://github.com/masutaka/circleci-tfupdate-orb

## License

MIT
