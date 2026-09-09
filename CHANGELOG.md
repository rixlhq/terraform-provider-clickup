# Changelog

## [0.3.2](https://github.com/rixlhq/terraform-provider-clickup/compare/v0.3.1...v0.3.2) (2026-09-09)


### Bug Fixes

* CI fmt gate plus mock TF_ACC, audit tool, README gaps ([4c3a719](https://github.com/rixlhq/terraform-provider-clickup/commit/4c3a71998f8bf402a64262f03a3d5578254ffd08))
* **ci:** keep parallel steps, add fmt gate and mock TF_ACC coverage ([ff00f55](https://github.com/rixlhq/terraform-provider-clickup/commit/ff00f557267e4038497673c7daeaef37c2ae03c9))
* **ci:** replace invalid parallel block with sequential steps ([26d1f6c](https://github.com/rixlhq/terraform-provider-clickup/commit/26d1f6c7b7633f774a4bbecd1a95235da6fbf1ba))
* **tools:** make audit_coverage usable without generated config ([8cb96e8](https://github.com/rixlhq/terraform-provider-clickup/commit/8cb96e80cec25487cc201a78eef0927087b47775))
* **tools:** validate audit config shape before use ([94ee9b2](https://github.com/rixlhq/terraform-provider-clickup/commit/94ee9b2053455bc5d0950e412c7987beb9523354))
* **workflows:** set timeout for CI, release, and spec regeneration jobs ([569910e](https://github.com/rixlhq/terraform-provider-clickup/commit/569910e16843355621e529980db6b8d8387faf3c))

## [0.3.1](https://github.com/rixlhq/terraform-provider-clickup/compare/v0.3.0...v0.3.1) (2026-08-24)


### Bug Fixes

* **release:** use_existing_draft to prevent duplicate releases ([8112ec5](https://github.com/rixlhq/terraform-provider-clickup/commit/8112ec59c4559a13f4b3c41671ba3ef070392324))

## [0.3.0](https://github.com/rixlhq/terraform-provider-clickup/compare/v0.2.0...v0.3.0) (2026-08-24)


### Features

* V3 client + 19 V2 resources + release draft fix ([#12](https://github.com/rixlhq/terraform-provider-clickup/issues/12)) ([4e42618](https://github.com/rixlhq/terraform-provider-clickup/commit/4e426185e0e94dc01a214d34ace8616c36875d46))


### Bug Fixes

* **v2:** address CodeRabbit review comments on resource implementations ([af944a2](https://github.com/rixlhq/terraform-provider-clickup/commit/af944a2368e004fbcb76c727b2476dea9cda0e96))
* **v2:** fix task_dependency DELETE query params, time_entry stop field, checklist_item ID extraction ([eca10ef](https://github.com/rixlhq/terraform-provider-clickup/commit/eca10ef1ba75a0c8b48e1400b3cf2fa70dff4187))

## [0.2.0](https://github.com/rixlhq/terraform-provider-clickup/compare/v0.1.1...v0.2.0) (2026-08-24)


### Features

* **v3:** generate ogen V3 client and add chat_channel resource + audit_logs data source ([e58c180](https://github.com/rixlhq/terraform-provider-clickup/commit/e58c18017cf7bdbaf9d22855ff6721d470bfb9d3))


### Bug Fixes

* **v3:** handle HTTP 204 empty body, fix lint, add tests and docs ([a0bb889](https://github.com/rixlhq/terraform-provider-clickup/commit/a0bb889d933c198465b3f26aedbadcc96455ba32))

## [0.1.1](https://github.com/rixlhq/terraform-provider-clickup/compare/v0.1.0...v0.1.1) (2026-08-24)


### Bug Fixes

* **docs:** generate registry docs for all resources and data sources ([6cbed18](https://github.com/rixlhq/terraform-provider-clickup/commit/6cbed1820f3618cbc87039c85e8c3a9130200538))

## [0.1.0](https://github.com/rixlhq/terraform-provider-clickup/compare/v0.0.1...v0.1.0) (2026-08-19)


### Features

* **provider:** add clickup_chat_channel resource ([4b6b898](https://github.com/rixlhq/terraform-provider-clickup/commit/4b6b8981ef03fe09126856baa40cdf8a46d63945))
* **provider:** add generic CRUD, hand-written resources and generated v2 schemas ([37eb49e](https://github.com/rixlhq/terraform-provider-clickup/commit/37eb49e08258eac5011d34fd46c2e8c69d2f4f4a))
* **provider:** add hand-written clickup_folder resource ([aed594a](https://github.com/rixlhq/terraform-provider-clickup/commit/aed594a0e521e4875b14881406f32e196c470aa6))
* **provider:** add list-read support and comment resources ([5bdb5ab](https://github.com/rixlhq/terraform-provider-clickup/commit/5bdb5ab84bb65aa03bebc8e26988566d55aab2a8))
* **provider:** generate data sources from ClickUp OpenAPI v3 spec ([e848da3](https://github.com/rixlhq/terraform-provider-clickup/commit/e848da300f353d92e06c33c818e3344feab33857))
* **provider:** regenerate all v2 data sources, add goal resource and webhook resource ([56c62fb](https://github.com/rixlhq/terraform-provider-clickup/commit/56c62fbff1c530c35db56426216c010fca3af4b0))
* **provider:** switch to ClickUp API v2 and add generic CRUD resources ([cc613ac](https://github.com/rixlhq/terraform-provider-clickup/commit/cc613ac766843cae07a0e97db08a53a14e7f502a))


### Bug Fixes

* **clickupclient:** use json.Marshal in TestServer.RegisterStatic ([a9be772](https://github.com/rixlhq/terraform-provider-clickup/commit/a9be7727fc01f822899e614eef43137fe3da110d))
* **goal:** unwrap read response and transform owners list ([cacea51](https://github.com/rixlhq/terraform-provider-clickup/commit/cacea51183123d3dab084557f2c63e794b3649e0))
* paginate list reads, RequiresReplace for create scope, preserve missing fields, require comment text ([b5ac467](https://github.com/rixlhq/terraform-provider-clickup/commit/b5ac467dcc2493b97844654048682e14e4cdd5e3))
* **provider:** load schema per CRUD op and use full schema type for path params ([8342fda](https://github.com/rixlhq/terraform-provider-clickup/commit/8342fda8b3852129e06f0dee39d3fb17fd9b00bf))
* **provider:** remove schema cache and recover state on failed reads ([6e770e7](https://github.com/rixlhq/terraform-provider-clickup/commit/6e770e7450d6e1bca8dde65c34740224a5008c72))
* **space_tag:** rename color keys for update body ([18da003](https://github.com/rixlhq/terraform-provider-clickup/commit/18da0037b77129c5d3f9641e7a16c371c13830ec))
* **tools:** derive resource names from selected read, warn on missing packages, fix $ref and parameter handling ([b3ef6df](https://github.com/rixlhq/terraform-provider-clickup/commit/b3ef6df483181605c547f56f14d3f3e7e720984c))
* **user_group:** default members to empty add/rem on create ([0251fab](https://github.com/rixlhq/terraform-provider-clickup/commit/0251fabddde96a49257f9f1b7604fcf1a3101e93))
* **webhook:** not-found handling, scope-id parsing, RequiresReplace, ImportState, health status ([aeca814](https://github.com/rixlhq/terraform-provider-clickup/commit/aeca8146088397eaa0caa5b3bff6f32a92fe7154))
