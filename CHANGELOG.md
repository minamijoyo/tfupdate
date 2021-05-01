## master (Unreleased)

ENHANCEMENTS:

* Update Go to v1.16.3 ([#37](https://github.com/minamijoyo/tfupdate/pull/37))
* Update hcl to v2.10.0 ([#38](https://github.com/minamijoyo/tfupdate/pull/38))
* Update alpine to v3.12 ([#39](https://github.com/minamijoyo/tfupdate/pull/39))

## 0.4.3 (2020/12/10)

BUG FIXES:

* Fix unexpected broken parentheses expression ([#34](https://github.com/minamijoyo/tfupdate/pull/34))

ENHANCEMENTS:

* Prevent uploading pre-release to Homebrew ([#29](https://github.com/minamijoyo/tfupdate/pull/29))

## 0.4.2 (2020/10/01)

NEW FEATURES:

* (experimental) Support getting release versions of a provider from the terraform registry ([#26](https://github.com/minamijoyo/tfupdate/pull/26))

The release list/latest command now allows you to get the release version from Terraform Registry with `--source-type tfregistryProvider`, which is an experimental feature because we are currently depending on an undocumented Registry API. We are planning to switch another API which Terraform CLI depends on.

ENHANCEMENTS:

* Ignore case for log level passed in TFUPDATE_LOG environment variable ([#25](https://github.com/minamijoyo/tfupdate/pull/25))

## 0.4.1 (2020/07/09)

BUG FIXES:

* Use static link on build for alpine compatible ([#23](https://github.com/minamijoyo/tfupdate/pull/23))

## 0.4.0 (2020/06/18)

NEW FEATURES:

* Support a new provider source syntax in Terraform v0.13 ([#21](https://github.com/minamijoyo/tfupdate/pull/21))

ENHANCEMENTS:

* Update Go to v1.14.4 ([#20](https://github.com/minamijoyo/tfupdate/pull/20))

## 0.3.6 (2020/06/11)

BUG FIXES:

* Fix panic with legacy dot access of numeric indexes ([#19](https://github.com/minamijoyo/tfupdate/pull/19))

## 0.3.5 (2020/02/26)

NEW FEATURES:

* Add release list command ([#16](https://github.com/minamijoyo/tfupdate/pull/16))

## 0.3.4 (2020/02/13)

NEW FEATURES:

* Add support for Terraform Registry Module as a release data source ([#15](https://github.com/minamijoyo/tfupdate/pull/15))

## 0.3.3 (2020/01/09)

BUG FIXES:

* Fix a bug for parsing a map index ([#13](https://github.com/minamijoyo/tfupdate/pull/13))

## 0.3.2 (2019/12/30)

NEW FEATURES:

* Add support for GitLab projects to release latest ([#11](https://github.com/minamijoyo/tfupdate/pull/11))

## 0.3.1 (2019/12/19)

NEW FEATURES:

* Add support for GitHub private repository ([#9](https://github.com/minamijoyo/tfupdate/pull/9))

ENHANCEMENTS:

* Make release interface more flexible ([#8](https://github.com/minamijoyo/tfupdate/pull/8))

BUG FIXES:

* Fix instruction for building from source ([#6](https://github.com/minamijoyo/tfupdate/pull/6))

## 0.3.0 (2019/11/28)

NEW FEATURES:

* Add module support ([#2](https://github.com/minamijoyo/tfupdate/pull/2))

Note: Automatic latest version resolution is not currently supported for modules.

## 0.2.1 (2019/11/27)

BUG FIXES:

* Fix typo in PROVIDER_NAME argument documentation ([#1](https://github.com/minamijoyo/tfupdate/pull/1))

## 0.2.0 (2019/11/09)

Initial release
