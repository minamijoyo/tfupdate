## master (Unreleased)

## 0.6.5 (2022/03/24)

BUG FIXES:

* Fix go install error ([#70](https://github.com/minamijoyo/tfupdate/pull/70))

## 0.6.4 (2022/01/13)

ENHANCEMENTS:

* Use golangci-lint instead of golint ([#60](https://github.com/minamijoyo/tfupdate/pull/60))
* Fix lint errors ([#61](https://github.com/minamijoyo/tfupdate/pull/61))
* Add support for linux/arm64 Docker image ([#62](https://github.com/minamijoyo/tfupdate/pull/62))

## 0.6.3 (2021/11/12)

ENHANCEMENTS:

* Update Go to v1.17.3 and Alpine to 3.14 ([#56](https://github.com/minamijoyo/tfupdate/pull/56))
* Update hcl to v2.10.1 ([#57](https://github.com/minamijoyo/tfupdate/pull/57))
* Add arm64 builds to support M1 mac ([#58](https://github.com/minamijoyo/tfupdate/pull/58))

## 0.6.2 (2021/10/25)

BUG FIXES:

* Fix panic when version key is quoted ([#52](https://github.com/minamijoyo/tfupdate/pull/52))

ENHANCEMENTS:

* Restrict permissions for GitHub Actions ([#53](https://github.com/minamijoyo/tfupdate/pull/53))
* Set timeout for GitHub Actions ([#54](https://github.com/minamijoyo/tfupdate/pull/54))

## 0.6.1 (2021/07/19)

BUG FIXES:

* Fix goreleaser settings for brew ([#50](https://github.com/minamijoyo/tfupdate/pull/50))

## 0.6.0 (2021/07/19)

BREAKING CHANGES:

* Build & push docker images on GitHub Actions ([#49](https://github.com/minamijoyo/tfupdate/pull/49))

The `latest` tag of docker image now points at the latest release. Previously the `latest` tag pointed at the master branch, if you want to use the master branch, use the `master` tag instead.

ENHANCEMENTS:

* Drop goreleaser dependencies ([#48](https://github.com/minamijoyo/tfupdate/pull/48))
* Move CI to GitHub Actions ([#47](https://github.com/minamijoyo/tfupdate/pull/47))

## 0.5.1 (2021/05/27)

ENHANCEMENTS:

* Allow to parse the configuration_aliases syntax in Terraform v0.15 ([#43](https://github.com/minamijoyo/tfupdate/pull/43))

## 0.5.0 (2021/05/14)

BREAKING CHANGES:

* Sort releases in semver order ([#41](https://github.com/minamijoyo/tfupdate/pull/41))
* Hide pre-releases by default in the release list command ([#42](https://github.com/minamijoyo/tfupdate/pull/42))

The `release latest` command now returns the latest release in semantic versioning order. Previously it returned the most recent release. In many cases it was the same, but in some cases the most recent older patch release was returned.

The `release list` command now sorts releases in semantic versioning order and hides pre-releases. If you want to show pre-releases, use the `--pre-release` flag.

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
