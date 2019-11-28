# tfupdate
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/minamijoyo/tfupdate.svg)](https://github.com/minamijoyo/tfupdate/releases/latest)
[![GoDoc](https://godoc.org/github.com/minamijoyo/tfupdate/tfupdate?status.svg)](https://godoc.org/github.com/minamijoyo/tfupdate)

## Features

- Update version constraints of Terraform core, providers, and modules
- Update all your Terraform configurations recursively under a given directory
- Get the latest release version from GitHub Release
- Terraform v0.12+ support

If you integrate tfupdate with your favorite CI or job scheduler, you can check the latest release daily and create a Pull Request automatically.

## Why?
It is a best practice to break your Terraform configuration and state into small pieces to minimize the impact of an accident.
It is also recommended to lock versions of Terraform core, providers and modules to avoid unexpected breaking changes.
If you decided to lock version constraints, you probably want to keep them up-to-date frequently to reduce the risk of version upgrade failures.
It's easy to update a single directory, but what if it's scattered across multiple directories?

That is why I wrote a tool which parses Terraform configurations and updates all version constraints at once.

## Install

### Homebrew

If you are macOS user:

```
$ brew install minamijoyo/tfupdate/tfupdate
```

### Download

Download the latest compiled binaries and put it anywhere in your executable path.

https://github.com/minamijoyo/tfupdate/releases

### Source

If you have Go 1.13+ development environment:

```
$ go get github.com/minamijoyo/tfupdate
```

### Docker

You can also run it with Docker:

```
$ docker run -it --rm minamijoyo/tfupdate --version
```

## Usage

```
tfupdate --help
Usage: tfupdate [--version] [--help] <command> [<args>]

Available commands are:
    module       Update version constraints for module
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

### provider

```
$ tfupdate provider --help
Usage: tfupdate provider [options] <PROVIDER_NAME> <PATH>

Arguments
  PROVIDER_NAME      A name of provider (e.g. aws, google, azurerm)
  PATH               A path of file or directory to update

Options:
  -v  --version      A new version constraint (default: latest)
                     If the version is omitted, the latest version is automatically checked and set.
                     Getting the latest version automatically is supported only for official providers.
                     If you have an unofficial provider, use release latest command.
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
```

```
$ cat main.tf
provider "aws" {
  version = "2.39.0"
}

$ tfupdate provider aws -v 2.40.0 main.tf

$ cat main.tf
provider "aws" {
  version = "2.40.0"
}
```

### module

```
$ tfupdate module --help
Usage: tfupdate module [options] <MODULE_NAME> <PATH>

Arguments
  MODULE_NAME        A name of module
                     e.g.
                       terraform-aws-modules/vpc/aws
                       git::https://example.com/vpc.git
  PATH               A path of file or directory to update

Options:
  -v  --version      A new version constraint (required)
                     Automatic latest version resolution is not currently supported for modules.
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
```

```
$ cat main.tf
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.20.0"
}

$ tfupdate module -v 2.21.0 terraform-aws-modules/vpc/aws main.tf

$ cat main.tf
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.21.0"
}
```

### release

```
$ tfupdate release --help
Usage: tfupdate release <subcommand> [options] [args]

  This command has subcommands for release version information.

Subcommands:
    latest    Get the latest release version from GitHub Release

$ tfupdate release latest --help
Usage: tfupdate release latest [options] <REPOSITORY>

Arguments
  REPOSITORY         A path of the the GitHub repository
                     (e.g. terraform-providers/terraform-provider-aws)
```

```
$ tfupdate release latest terraform-providers/terraform-provider-aws
2.40.0
```

## Keep your dependencies up-to-date

If you integrate tfupdate with your favorite CI or job scheduler, you can check the latest release daily and create a Pull Request automatically.

An example for tfupdate with CircleCI is available:

https://github.com/minamijoyo/tfupdate-circleci-example

## License

MIT
