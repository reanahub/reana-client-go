<!-- markdownlint-disable MD013 -->
<!-- markdownlint-disable MD024 -->

# Changelog

## [0.95.0](https://github.com/reanahub/reana-client-go/compare/v0.95.0-alpha.0...v0.95.0) (2026-09-22)


### ⚠ BREAKING CHANGES

* **client:** `REANA_SERVER_URL` and `REANA_SERVER_TLS_VERIFY` environment variables are no longer client inputs. Unset them and run `login --server URL`, adding `--no-tls-verify` for a self-signed development server when needed.
* **auth:** Use this client with the OIDC-compatible REANA server and run reana-client-go login, or supply an issuer-issued JWT. Legacy opaque REANA tokens no longer authenticate. TLS certificate verification is enabled by default; configure REANA_SERVER_CA_CERTS for private certificates. Replace static-token automation with credentials whose refresh-token lifetime and rotation policy fit the job, and upgrade the client and cluster together for the OIDC cut-over.

### Build

* **make:** add configurable installation targets ([#202](https://github.com/reanahub/reana-client-go/issues/202)) ([5548270](https://github.com/reanahub/reana-client-go/commit/55482705451de9b2f00460537b4c051bac19be8b))


### Features

* **auth:** add OIDC login and credential management ([#201](https://github.com/reanahub/reana-client-go/issues/201)) ([fa5a5a9](https://github.com/reanahub/reana-client-go/commit/fa5a5a92d68406f6a1a1b1d2f5d75e381aceafd1))
* **client:** manage saved REANA server connections ([#214](https://github.com/reanahub/reana-client-go/issues/214)) ([0618535](https://github.com/reanahub/reana-client-go/commit/06185358a9645cab2c05a9d7995a78a2a5c806a0))
* **client:** save server and TLS settings at login ([#213](https://github.com/reanahub/reana-client-go/issues/213)) ([726a34b](https://github.com/reanahub/reana-client-go/commit/726a34b20f70f666a68ce002100fa53d6b31c347))
* **cmd:** add create, validate, and run commands ([#185](https://github.com/reanahub/reana-client-go/issues/185)) ([cff473e](https://github.com/reanahub/reana-client-go/commit/cff473efee27f899b5d09d876146b34b385484c9))
* **delete:** support safe restart-series deletion ([#197](https://github.com/reanahub/reana-client-go/issues/197)) ([835c300](https://github.com/reanahub/reana-client-go/commit/835c300e06578f29d495c18bae2d5623dc5e1540)), closes [#192](https://github.com/reanahub/reana-client-go/issues/192)
* **diff:** accept -U as unified context alias ([#196](https://github.com/reanahub/reana-client-go/issues/196)) ([25bbd16](https://github.com/reanahub/reana-client-go/commit/25bbd168221c7def7370b15fe1f33498f7161d88))
* **share:** add JSON output and sharing option checks ([#194](https://github.com/reanahub/reana-client-go/issues/194)) ([5f1bd06](https://github.com/reanahub/reana-client-go/commit/5f1bd06b34bde52b0623862bb851fd31d0fdd82f)), closes [#190](https://github.com/reanahub/reana-client-go/issues/190)
* **test:** add command to test completed workflow runs ([#189](https://github.com/reanahub/reana-client-go/issues/189)) ([783c6d6](https://github.com/reanahub/reana-client-go/commit/783c6d6e0e9297e473597ac83512f4edc634a4ad)), closes [#162](https://github.com/reanahub/reana-client-go/issues/162)


### Bug fixes

* **cmd:** handle optional workflow specification fields ([#188](https://github.com/reanahub/reana-client-go/issues/188)) ([ee9834f](https://github.com/reanahub/reana-client-go/commit/ee9834f71e9e1486b113e6dfb23b21ca055e919a)), closes [#187](https://github.com/reanahub/reana-client-go/issues/187)
* **download:** continue after individual failures ([#198](https://github.com/reanahub/reana-client-go/issues/198)) ([52d9eb5](https://github.com/reanahub/reana-client-go/commit/52d9eb5d45f098348905a2af0cf0a0f95aa38276))
* **help:** list create under workflow management ([#196](https://github.com/reanahub/reana-client-go/issues/196)) ([558ffff](https://github.com/reanahub/reana-client-go/commit/558ffffaa2bd36dc6214d82239add8e882426714)), closes [#191](https://github.com/reanahub/reana-client-go/issues/191)
* **quota-show:** accept report names case-insensitively ([#196](https://github.com/reanahub/reana-client-go/issues/196)) ([09bf85c](https://github.com/reanahub/reana-client-go/commit/09bf85cbbf199c0406372a9822e012b5bc0a4af7))
* **rm:** continue after individual API failures ([#198](https://github.com/reanahub/reana-client-go/issues/198)) ([41501c7](https://github.com/reanahub/reana-client-go/commit/41501c74e0927c3ae19d1ce27e8abefd68933cb6)), closes [#193](https://github.com/reanahub/reana-client-go/issues/193)
* **upload:** continue after individual failures ([#198](https://github.com/reanahub/reana-client-go/issues/198)) ([475d1ad](https://github.com/reanahub/reana-client-go/commit/475d1ad07076ca93dafa0f8e8bfcb510231d0abe))


### Test suite

* **validate:** stabilise image cleanup timeout test ([#195](https://github.com/reanahub/reana-client-go/issues/195)) ([9ea9962](https://github.com/reanahub/reana-client-go/commit/9ea99626fd348af2039cc265d54d1a0524cb4fee))


### Continuous integration

* **release-please:** prepare stable releases ([#200](https://github.com/reanahub/reana-client-go/issues/200)) ([e614c62](https://github.com/reanahub/reana-client-go/commit/e614c62fab13bcd20525017d0b706fd8df872e98)), closes [#199](https://github.com/reanahub/reana-client-go/issues/199)
* **release:** publish standalone binaries ([#200](https://github.com/reanahub/reana-client-go/issues/200)) ([cee3947](https://github.com/reanahub/reana-client-go/commit/cee39478a864bdecdac87f263736d6b5a9c2a602))

## 0.95.0-alpha.0 (unreleased baseline)
