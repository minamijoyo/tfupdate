## master (Unreleased)

## 0.9.2 (2025/08/09)

ENHANCEMENTS:

* Update golang.org/x/crypto to v0.41.0 ([#143](https://github.com/minamijoyo/tfupdate/pull/143))
* Update hcl to v2.24.0 ([#144](https://github.com/minamijoyo/tfupdate/pull/144))
* Add support for Terraform 1.12 ([#145](https://github.com/minamijoyo/tfupdate/pull/145))
* Add support for OpenTofu 1.10 ([#146](https://github.com/minamijoyo/tfupdate/pull/146))

## 0.9.1 (2025/05/05)

BUG FIXES:

* Fix the build issue with the replace directive ([#136](https://github.com/minamijoyo/tfupdate/pull/136))

## 0.9.0 (2025/05/03)

NEW FEATURES:

The tfupdate now supports OpenTofu, a community fork of Terraform.

If you want to use the public OpenTofu registry, set the `TFREGISTRY_BASE_URL` environment variable to `https://registry.opentofu.org/`.

```
$ export TFREGISTRY_BASE_URL=https://registry.opentofu.org/
```

* Add support for updating version constraints of opentofu core ([#127](https://github.com/minamijoyo/tfupdate/pull/127))
* Add support for .tofu extension ([#128](https://github.com/minamijoyo/tfupdate/pull/128))
* Add support for the OpenTofu registry as a release source ([#130](https://github.com/minamijoyo/tfupdate/pull/130))
* Allow TFREGISTRY_BASE_URL to set the host of the Terraform registry ([#132](https://github.com/minamijoyo/tfupdate/pull/132))
* Add support for updating .terraform.lock.hcl using OpenTofu registry ([#134](https://github.com/minamijoyo/tfupdate/pull/134))

ENHANCEMENTS:

* Update Go to v1.24 ([#124](https://github.com/minamijoyo/tfupdate/pull/124))
* Update hcl to v2.23.0 ([#125](https://github.com/minamijoyo/tfupdate/pull/125))
* Add support for Terraform 1.11 ([#126](https://github.com/minamijoyo/tfupdate/pull/126))
* Pin all GitHub Actions ([#129](https://github.com/minamijoyo/tfupdate/pull/129))
* Unify tfregistry config for release and lock packages ([#133](https://github.com/minamijoyo/tfupdate/pull/133))

NOTE:

This release contains breaking changes as Go packages, but as a CLI, it should not affect end users.

## 0.8.5 (2024/08/02)

ENHANCEMENTS:

* Update goreleaser to v2 ([#121](https://github.com/minamijoyo/tfupdate/pull/121))
* Switch to the official action for creating GitHub App token ([#122](https://github.com/minamijoyo/tfupdate/pull/122))

## 0.8.4 (2024/08/01)

ENHANCEMENTS:

* Pin goreleaser to v1 ([#119](https://github.com/minamijoyo/tfupdate/pull/119))

## 0.8.3 (2024/08/01)

ENHANCEMENTS:

* Update hcl to v2.21.0 ([#114](https://github.com/minamijoyo/tfupdate/pull/114))
* Use docker compose command instead of docker-compose ([#115](https://github.com/minamijoyo/tfupdate/pull/115))
* Add support for Terraform 1.9 ([#116](https://github.com/minamijoyo/tfupdate/pull/116))
* Update alpine to v3.20 ([#117](https://github.com/minamijoyo/tfupdate/pull/117))
* Update golangci lint to v1.59.1 ([#118](https://github.com/minamijoyo/tfupdate/pull/118))

## 0.8.2 (2024/04/15)

ENHANCEMENTS:

* feat: update to use go 1.22 ([#111](https://github.com/minamijoyo/tfupdate/pull/111))
* Add support for provider-defined functions ([#112](https://github.com/minamijoyo/tfupdate/pull/112))
* Add support for Terraform 1.8 ([#113](https://github.com/minamijoyo/tfupdate/pull/113))

## 0.8.1 (2024/01/26)

NEW FEATURES:

* tfupdate module command support regex matches ([#108](https://github.com/minamijoyo/tfupdate/pull/108))

ENHANCEMENTS:

* Update hcl to v2.19.1 ([#109](https://github.com/minamijoyo/tfupdate/pull/109))
* Add support for Terraform 1.7 ([#110](https://github.com/minamijoyo/tfupdate/pull/110))

## 0.8.0 (2023/10/09)

NOTE:

Starting from v0.8.0, the tfupdate provider command now supports namespaces. To maintain backward compatibility, the left-hand key in the required_providers block is still evaluated as the short name if the namespace is omitted. Still, if the provider name contains /, we assume that users intend to use namespaces and check the source address.

BREAKING CHANGES:

While backward compatibility is maintained for most use cases, some partner providers that relied on the legacy terraform-providers/ org redirects must explicitly specify their namespace. Specifically, namespaces must be explicit if all the following conditions are met:

- The tfupdate provider command does not explicitly specify the namespace.
- The tfupdate provider command does not explicitly specify the version.
- Redirected from legacy terraform-providers/ org to partner org on GitHub.
- Not hosted under hashicorp/ org on GitHub.
- Not redirected from hashicorp/ org to partner org on GitHub.

NEW FEATURES:

* Add support for provider namespace ([#102](https://github.com/minamijoyo/tfupdate/pull/102))

BUG FIXES:

* Fixed a crash when parsing invalid release versions as SemVer ([#103](https://github.com/minamijoyo/tfupdate/pull/103))

ENHANCEMENTS:

* deps: update to use go1.21 ([#98](https://github.com/minamijoyo/tfupdate/pull/98))
* Update actions/checkout to v4 ([#100](https://github.com/minamijoyo/tfupdate/pull/100))
* Update hcl to v2.18.1 ([#101](https://github.com/minamijoyo/tfupdate/pull/101))
* Add support for Terraform v1.6 ([#104](https://github.com/minamijoyo/tfupdate/pull/104))

## 0.7.2 (2023/07/07)

BUG FIXES:

* Fix a regression issue for updating .hcl file ([#97](https://github.com/minamijoyo/tfupdate/pull/97))

## 0.7.1 (2023/07/05)

BUG FIXES:

* Fix a regression issue for using absolute path ([#94](https://github.com/minamijoyo/tfupdate/pull/94))

ENHANCEMENTS:

* Set docker build timeout to 20m ([#91](https://github.com/minamijoyo/tfupdate/pull/91))
* Update docker related actions to latest ([#92](https://github.com/minamijoyo/tfupdate/pull/92))

## 0.7.0 (2023/07/04)

NEW FEATURES:

* Add native support for updating .terraform.lock.hcl ([#90](https://github.com/minamijoyo/tfupdate/pull/90))

## 0.6.8 (2023/06/06)

ENHANCEMENTS:

* Update Go to v1.20 and Alpine 3.18 ([#87](https://github.com/minamijoyo/tfupdate/pull/87))
* Update hcl to v2.17.0 ([#88](https://github.com/minamijoyo/tfupdate/pull/88))
* Set docker build timeout to 10m ([#89](https://github.com/minamijoyo/tfupdate/pull/89))

## 0.6.7 (2022/08/29)

ENHANCEMENTS:

* deps: update to use go1.19 ([#74](https://github.com/minamijoyo/tfupdate/pull/74))
* Add jq, openssl, and curl to Docker image ([#76](https://github.com/minamijoyo/tfupdate/pull/76))

## 0.6.6 (2022/08/10)

ENHANCEMENTS:

* Update golangci-lint to v1.45.2 and actions to latest ([#71](https://github.com/minamijoyo/tfupdate/pull/71))
* Use GitHub App token for updating brew formula on release ([#73](https://github.com/minamijoyo/tfupdate/pull/73))

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
