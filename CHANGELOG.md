# Changelog

## [0.6.5-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.6.4-alpha...v0.6.5-alpha) (2022-07-19)


### Bug Fixes

* complete trigger async binary file route ([5003e5c](https://github.com/instill-ai/pipeline-backend/commit/5003e5c613d28f918ce92835c4636d05ec13b5a9))

## [0.6.4-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.6.3-alpha...v0.6.4-alpha) (2022-07-11)


### Miscellaneous Chores

* release v0.6.4-alpha ([ff401be](https://github.com/instill-ai/pipeline-backend/commit/ff401be68129589aefcfd4647794ff23f5e073a1))

## [0.6.3-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.6.2-alpha...v0.6.3-alpha) (2022-07-07)


### Miscellaneous Chores

* release v0.6.3-alpha ([47468f2](https://github.com/instill-ai/pipeline-backend/commit/47468f27886fb5d966c5620b07f504cefa737dcb))

## [0.6.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.6.1-alpha...v0.6.2-alpha) (2022-06-27)


### Bug Fixes

* close [#56](https://github.com/instill-ai/pipeline-backend/issues/56) ([c627a53](https://github.com/instill-ai/pipeline-backend/commit/c627a539999d65bc96ac6f88e2bd203548c34825))
* fix empty description update ([2579f2e](https://github.com/instill-ai/pipeline-backend/commit/2579f2eb6d1b1cd0935feffa3dd5084e3ec0851d))

## [0.6.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.6.0-alpha...v0.6.1-alpha) (2022-06-27)


### Miscellaneous Chores

* release v0.6.1-alpha ([38c781c](https://github.com/instill-ai/pipeline-backend/commit/38c781c8fb6af410669bed8e020f74785e44f2fe))

## [0.6.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.5.2-alpha...v0.6.0-alpha) (2022-06-26)


### Features

* add async pipeline trigger ([6d39b0a](https://github.com/instill-ai/pipeline-backend/commit/6d39b0aac6bf3475cfc29899e47810a38501a9ea))
* add filter for list pipeline ([ffe8856](https://github.com/instill-ai/pipeline-backend/commit/ffe8856dba23128e190c9a841eba616c80f0ba8e))
* add usage collection ([7e71744](https://github.com/instill-ai/pipeline-backend/commit/7e71744b6c1dd78612bcb849042d4129b04b017e))
* support trigger multi model instances ([e3d4263](https://github.com/instill-ai/pipeline-backend/commit/e3d4263caf7f32d0e4bc25783d148a34c26268da))


### Bug Fixes

* fix create pipeline recipe resource name ([bab3eaa](https://github.com/instill-ai/pipeline-backend/commit/bab3eaa588be4ce058027967c015ed2dbba2bc5b))
* fix duration configuration bug ([998eafa](https://github.com/instill-ai/pipeline-backend/commit/998eafaf1d43fd568ae0af8699e7470f99c975cb))
* fix usage collection ([243e7a1](https://github.com/instill-ai/pipeline-backend/commit/243e7a14298cc4db8981e39e6c1dc078e721826e))
* fix usage disbale logic ([962823b](https://github.com/instill-ai/pipeline-backend/commit/962823b9822da510efa703f82a0f7a8612ce3348))
* fix usage-backend non-tls dial ([b864df3](https://github.com/instill-ai/pipeline-backend/commit/b864df3c7461b06059b82b7e09be6bf9e793bc28))

### [0.5.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.5.1-alpha...v0.5.2-alpha) (2022-05-31)


### Features

* add cors support ([c847912](https://github.com/instill-ai/pipeline-backend/commit/c847912a9f5745c5dd807fdecb7be97b77592655))


### Miscellaneous Chores

* release 0.5.2-alpha ([3bb261e](https://github.com/instill-ai/pipeline-backend/commit/3bb261eebce009385018004ce20c78ae8ef62ab9))

### [0.5.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.5.0-alpha...v0.5.1-alpha) (2022-05-19)


### Bug Fixes

* fix create dup error code ([5d3a0c9](https://github.com/instill-ai/pipeline-backend/commit/5d3a0c98c480e22f03da43126742466a91f14993))

## [0.5.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.4.0-alpha...v0.5.0-alpha) (2022-05-19)


### Features

* integrate with mgmt-backend ([6514bf4](https://github.com/instill-ai/pipeline-backend/commit/6514bf431975cd84bad5a6f54b4e137f25db5941))


### Bug Fixes

* fix list empty case ([5050693](https://github.com/instill-ai/pipeline-backend/commit/5050693177c60ee81e04434efe0036393160e6cf))
* refactor pipeline JSON schema ([1f88481](https://github.com/instill-ai/pipeline-backend/commit/1f88481b393ed48999d64bf2a44ff68e27c06a5d))

## [0.4.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.3.1-alpha...v0.4.0-alpha) (2022-05-13)


### Features

* integrate with all backends ([0dcdbff](https://github.com/instill-ai/pipeline-backend/commit/0dcdbff578e922804f9a8060ebd20f5e5b151794))


### Bug Fixes

* fix model-backend config ([0a92bef](https://github.com/instill-ai/pipeline-backend/commit/0a92bef81834e417f2e43b74cd73d589dd0095ae))
* refactor JSON Schema ([#48](https://github.com/instill-ai/pipeline-backend/issues/48)) ([d57f2db](https://github.com/instill-ai/pipeline-backend/commit/d57f2db9e1e11ea1e13a8f7f725c70b39ceee03c))
* use InvalidArgument instead of FailedPrecondition ([54bb2a4](https://github.com/instill-ai/pipeline-backend/commit/54bb2a4036006c6151eb09a3ed909746b0b676f8))

### [0.3.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.3.0-alpha...v0.3.1-alpha) (2022-03-22)


### Miscellaneous Chores

* release 0.3.1-alpha ([c0b6219](https://github.com/instill-ai/pipeline-backend/commit/c0b6219a34db501988e40c1bc319e484869e3e20))

## [0.3.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.2.1-alpha...v0.3.0-alpha) (2022-03-21)


### Miscellaneous Chores

* release 0.3.0-alpha ([0f6a208](https://github.com/instill-ai/pipeline-backend/commit/0f6a2085b6a727996d94e5364969ee989137c52f))

### [0.2.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.2.0-alpha...v0.2.1-alpha) (2022-02-24)


### Bug Fixes

* add support URL/base64 endpoint ([#29](https://github.com/instill-ai/pipeline-backend/issues/29)) ([21f6c6d](https://github.com/instill-ai/pipeline-backend/commit/21f6c6d665c48cd78d2d41036ab50a50663a98bc))
* change struct definition from private to public ([#23](https://github.com/instill-ai/pipeline-backend/issues/23)) ([ffee642](https://github.com/instill-ai/pipeline-backend/commit/ffee6425c6c0f9833bde2dd7c47baae548326d26))
* expose all structs inside pkg folder ([#25](https://github.com/instill-ai/pipeline-backend/issues/25)) ([345639f](https://github.com/instill-ai/pipeline-backend/commit/345639f70bf1fcfb6d0c1a049f5cfa0620840e3a))

## [0.2.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.1.0-alpha...v0.2.0-alpha) (2022-02-14)


### Features

* add model validation while creating pipeline and files ([#5](https://github.com/instill-ai/pipeline-backend/issues/5)) ([8bb7af3](https://github.com/instill-ai/pipeline-backend/commit/8bb7af3e342a0fba865b7c7568aaa258766f6d8e))


### Bug Fixes

* fix vdp import path ([#8](https://github.com/instill-ai/pipeline-backend/issues/8)) ([d119411](https://github.com/instill-ai/pipeline-backend/commit/d119411d04c768992860d943081a275284b330bc))

## [0.1.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.0.0-alpha...v0.1.0-alpha) (2022-02-11)


### Features

* add keyset pagination and refactor recipe ([#3](https://github.com/instill-ai/pipeline-backend/issues/3)) ([9daedf0](https://github.com/instill-ai/pipeline-backend/commit/9daedf0f6ea6280f70381cbc66ddb726bc7ae339))
* initiate repo ([#1](https://github.com/instill-ai/pipeline-backend/issues/1)) ([6ec4a9a](https://github.com/instill-ai/pipeline-backend/commit/6ec4a9abf969d1e561f4097b4c74916b73e2e88a))
