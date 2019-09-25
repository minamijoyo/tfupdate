# tfupdate

Update version constraints in your Terraform configurations.

It is a best practice to break your Terraform configuration and state into small pieces based on the environments and frequency of changes to minimize the impact of an accident.
It is also recommended that you lock versions of Terraform core and dependent providers to avoid unexpected breaking changes. If you decided to lock version constraints, you probably want to keep them up to date frequently to reduce the risk of version upgrade failures.
It's easy to update a single directory, but what if it's scattered across multiple directories?
Of course you can do it with find, xargs, and sed, but it is fragile because it doesn't really understand HCL.

That is why I wrote a tool which parses Terraform configurations and updates all version constraints at once.

# Features

- Update version constraints of Terraform core and providers
- Update all your Terraform configurations recursively under a given directory

# Supported Terraform version

- Terraform v0.12+

# Install

```
$ go get github.com/minamijoyo/tfupdate
```

# Example

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
$ tfupdate terraform 0.12.8 main.tf

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
$ tfupdate provider aws@2.28.0 ./

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
$ tfupdate terraform 0.12.8 ./ -r
```

# Usage

```
$ tfupdate --help
Usage: tfupdate [--version] [--help] <command> [<args>]

Available commands are:
    provider     Update version constraints for provider
    terraform    Update version constraints for terraform

```

```
$ tfupdate terraform --help
Usage: tfupdate terraform [options] <VERSION> <PATH>

Arguments
  VERSION            A new version constraint
  PATH               A path of file or directory to update

Options:
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
```

```
$ tfupdate provider --help
Usage: tfupdate provider [options] <PROVIER_NAME>@<VERSION> <PATH>

Arguments
  PROVIER_NAME       A name of provider (e.g. aws, google, azurerm)
  VERSION            A new version constraint
  PATH               A path of file or directory to update

Options:
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
```

# License

MIT
