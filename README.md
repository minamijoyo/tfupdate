# tfupdate
[![GitHub release](http://img.shields.io/github/release/minamijoyo/tfupdate.svg?style=flat-square)](https://github.com/minamijoyo/tfupdate/releases)
[![GoDoc](https://godoc.org/github.com/minamijoyo/tfupdate/tfupdate?status.svg)](https://godoc.org/github.com/minamijoyo/tfupdate)

## Features

- Update version constraints of Terraform core and providers
- Update all your Terraform configurations recursively under a given directory
- Get the latest release version automatically from GitHub Release
- Terraform v0.12+ support

## Why?
It is a best practice to break your Terraform configuration and state into small pieces based on the environments and frequency of changes to minimize the impact of an accident.
It is also recommended that you lock versions of Terraform core and dependent providers to avoid unexpected breaking changes. If you decided to lock version constraints, you probably want to keep them up to date frequently to reduce the risk of version upgrade failures.
It's easy to update a single directory, but what if it's scattered across multiple directories?
Of course you can do it with find, xargs, and sed, but it is fragile because it doesn't really understand HCL.

That is why I wrote a tool which parses Terraform configurations and updates all version constraints at once.

## Install

You can install it in several ways.

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

## Example

```
$ cat main.tf
terraform {
  required_version = "0.12.7"
}

provider "aws" {
  version = "2.27.0"
}
```

```
$ tfupdate terraform -v 0.12.8 main.tf

$ git diff
diff --git a/main.tf b/main.tf
index ce0ff1c..1dd7294 100644
--- a/main.tf
+++ b/main.tf
@@ -1,5 +1,5 @@
 terraform {
-  required_version = "0.12.7"
+  required_version = "0.12.8"
 }

 provider "aws" {

$ git add . && git commit -m "Update terraform to v0.12.8"
[master dc46c06] Update terraform to v0.12.8
 1 file changed, 1 insertion(+), 1 deletion(-)
```

```
$ tfupdate provider aws -v 2.28.0 ./

$ git diff
diff --git a/main.tf b/main.tf
index 1dd7294..241ac69 100644
--- a/main.tf
+++ b/main.tf
@@ -3,5 +3,5 @@ terraform {
 }

 provider "aws" {
-  version = "2.27.0"
+  version = "2.28.0"
 }

$ git add . && git commit -m "Update terraform-provider-aws to v2.28.0"
[master 0e298ac] Update terraform-provider-aws to v2.28.0
 1 file changed, 1 insertion(+), 1 deletion(-)
```

If you want to update all your Terraform configurations under the current directory recursively,
run a command like this:

```
$ tfupdate terraform -v 0.12.8 -r ./
```

You can also ignore some paths:

```
$ tfupdate terraform -v 0.12.8 -i modules/ -r ./
```

If the version is omitted, the latest version is automatically checked and set.

```
$ tfupdate terraform -r ./
```

## Usage

```
$ tfupdate --help
Usage: tfupdate [--version] [--help] <command> [<args>]

Available commands are:
    provider     Update version constraints for provider
    release      Get release version information
    terraform    Update version constraints for terraform
```

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

```
$ tfupdate provider --help
Usage: tfupdate provider [options] <PROVIER_NAME> <PATH>

Arguments
  PROVIER_NAME       A name of provider (e.g. aws, google, azurerm)
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
$ tfupdate release --help
Usage: tfupdate release <subcommand> [options] [args]

  This command has subcommands for release version information.

Subcommands:
    latest    Get the latest release version from GitHub Release
```

```
$ tfupdate release latest --help
Usage: tfupdate release latest [options] <REPOSITORY>

Arguments
  REPOSITORY         A path of the the GitHub repository
                     (e.g. terraform-providers/terraform-provider-aws)
```

## License

MIT
