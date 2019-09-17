# tfupdate

Update version constraints in your Terraform configurations.

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

$ tfupdate terraform -v 0.12.8 -f main.tf

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

$ tfupdate provider -v aws@2.28.0 -f main.tf

$ git diff
diff --git a/main.tf b/main.tf
index ce0ff1c..241ac69 100644
--- a/main.tf
+++ b/main.tf
@@ -1,7 +1,7 @@
 terraform {
-  required_version = "0.12.7"
+  required_version = "0.12.8"
 }

 provider "aws" {
-  version = "2.27.0"
+  version = "2.28.0"
 }
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
Usage: tfupdate terraform [options]

Options:
  -v    A new version constraint
  -f    A path to filename to update
```

```
$ tfupdate provider --help
Usage: tfupdate provider [options]

Options:
  -v    A new version constraint.
        The valid format is <PROVIER_NAME>@<VERSION>
  -f    A path to filename to update
```

# License

MIT
