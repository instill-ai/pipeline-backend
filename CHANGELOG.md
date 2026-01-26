# Changelog

## [0.61.1](https://github.com/instill-ai/pipeline-backend/compare/v0.61.0...v0.61.1) (2026-01-26)


### Features

* **component,data,artifact:** update instillartifact component to latest protobuf ([#1139](https://github.com/instill-ai/pipeline-backend/issues/1139)) ([4c1f508](https://github.com/instill-ai/pipeline-backend/commit/4c1f50807b2220fc67a926f6c91ab7414ccb0ace))
* **data:** add WebM audio support and improve constant naming ([#1142](https://github.com/instill-ai/pipeline-backend/issues/1142)) ([bd4e10e](https://github.com/instill-ai/pipeline-backend/commit/bd4e10e5b8c3582f74b6b17d5c0697b145fa17ab))
* **pipeline:** add creator tracking to pipeline resource ([#1147](https://github.com/instill-ai/pipeline-backend/issues/1147)) ([a5ddeed](https://github.com/instill-ai/pipeline-backend/commit/a5ddeed1ecb9546331d34befa8edccc2b8bed5ec))


### Bug Fixes

* **document:** resolve audio/video format conversion and HTML-to-PDF issues ([#1141](https://github.com/instill-ai/pipeline-backend/issues/1141)) ([683de3a](https://github.com/instill-ai/pipeline-backend/commit/683de3a3f25c536130b1ee0cbbf2de7c6bccff46))
* **worker:** prevent concurrent map access race condition ([#1143](https://github.com/instill-ai/pipeline-backend/issues/1143)) ([e24a0b6](https://github.com/instill-ai/pipeline-backend/commit/e24a0b676ee7f4ad124b68f73450dad93aad7af7))


### Miscellaneous

* **deps:** bump github.com/weaviate/weaviate from 1.26.0-rc.1 to 1.30.20 ([#1146](https://github.com/instill-ai/pipeline-backend/issues/1146)) ([95a7a61](https://github.com/instill-ai/pipeline-backend/commit/95a7a61de07bd6e85dbf9d8b750642f0a17f656b))
* **deps:** bump golang.org/x/crypto from 0.36.0 to 0.45.0 in /pkg/component/tools/compogen ([#1144](https://github.com/instill-ai/pipeline-backend/issues/1144)) ([a57ead2](https://github.com/instill-ai/pipeline-backend/commit/a57ead24f55f621e200e087d74f7767bc5f18ae3))
* **deps:** bump golang.org/x/crypto from 0.39.0 to 0.45.0 ([#1145](https://github.com/instill-ai/pipeline-backend/issues/1145)) ([a64bc3d](https://github.com/instill-ai/pipeline-backend/commit/a64bc3d62f98b03c3285d5f047eed8ade25918e8))
* **deps:** upgrade Go to 1.25.6 and weaviate-go-client to v5 ([#1150](https://github.com/instill-ai/pipeline-backend/issues/1150)) ([2d74899](https://github.com/instill-ai/pipeline-backend/commit/2d7489964e2fcf786f91700877d17d3e4e77a99c))


### Refactor

* **acl:** migrate to shared ACL client library ([#1152](https://github.com/instill-ai/pipeline-backend/issues/1152)) ([eafe245](https://github.com/instill-ai/pipeline-backend/commit/eafe245388410a3806128dba8fea6f59330dd4d4))
* **api:** adopt flattened namespace prefixes from protobufs ([#1153](https://github.com/instill-ai/pipeline-backend/issues/1153)) ([9cd910c](https://github.com/instill-ai/pipeline-backend/commit/9cd910cf698b8fadb8cc0facef683725f7e515f2))
* **api:** update tests for AIP-compliant APIs and add private service tests ([#1151](https://github.com/instill-ai/pipeline-backend/issues/1151)) ([dc79b3d](https://github.com/instill-ai/pipeline-backend/commit/dc79b3da6f98bc429eca0bf7bed89e0c7e35fb07))
* **artifact:** adopt AIP-122 resource name fields from protobufs ([#1154](https://github.com/instill-ai/pipeline-backend/issues/1154)) ([db2b2d9](https://github.com/instill-ai/pipeline-backend/commit/db2b2d9ef5b2924f5eda58b11f421f21bce4aad7))
* **pipeline:** implement AIP-compliant resource naming with hash-based IDs ([#1149](https://github.com/instill-ai/pipeline-backend/issues/1149)) ([f694e8a](https://github.com/instill-ai/pipeline-backend/commit/f694e8a6a93fc3907f8b391b88441a2ffaac190d))
* **service:** remove organization support and add internal service bypass ([#1148](https://github.com/instill-ai/pipeline-backend/issues/1148)) ([1f54cc3](https://github.com/instill-ai/pipeline-backend/commit/1f54cc32a6502d870e418eeaec97b7be71386376))

## [0.61.0](https://github.com/instill-ai/pipeline-backend/compare/v0.60.0...v0.61.0) (2025-10-07)


### Features

* **component,ai,gemini:** add image generation support ([#1122](https://github.com/instill-ai/pipeline-backend/issues/1122)) ([d986614](https://github.com/instill-ai/pipeline-backend/commit/d986614184cf969208f82b529c4d07f624ab7907))
* **component,ai,gemini:** add multimedia support with unified format… ([#1114](https://github.com/instill-ai/pipeline-backend/issues/1114)) ([291b379](https://github.com/instill-ai/pipeline-backend/commit/291b379d4b9c29946a5f3dd566397574400b58f4))
* **component,ai,gemini:** add text embeddings task support ([#1129](https://github.com/instill-ai/pipeline-backend/issues/1129)) ([d7ca6cf](https://github.com/instill-ai/pipeline-backend/commit/d7ca6cf8fb0015393e05a22eeaa221942bdf03fd))
* **component,ai,gemini:** enhance streaming to output all fields ([#1106](https://github.com/instill-ai/pipeline-backend/issues/1106)) ([dfb6b24](https://github.com/instill-ai/pipeline-backend/commit/dfb6b247a1cad692854be45da4eed92de2b23bc8))
* **component,ai,gemini:** implement automatic format conversion for unsupported media types ([#1128](https://github.com/instill-ai/pipeline-backend/issues/1128)) ([f767b8a](https://github.com/instill-ai/pipeline-backend/commit/f767b8a95235e8f89290472e4919d2998b740d7b))
* **component,ai,gemini:** implement File API support for large files… ([#1118](https://github.com/instill-ai/pipeline-backend/issues/1118)) ([b51c8f4](https://github.com/instill-ai/pipeline-backend/commit/b51c8f4c2040c914bf9e950c2001b650e6a852e4))
* **data:** add comprehensive AVIF image format support ([#1135](https://github.com/instill-ai/pipeline-backend/issues/1135)) ([76d6941](https://github.com/instill-ai/pipeline-backend/commit/76d6941ac25da5717a9af516cd24730ab1410c7e))
* **data:** add HEIC/HEIF image support and normalize MIME types ([#1127](https://github.com/instill-ai/pipeline-backend/issues/1127)) ([2dfa254](https://github.com/instill-ai/pipeline-backend/commit/2dfa254fc7d185d2e96a982ae93ade7e9355e789))
* **data:** enhance unmarshaler with JSON string to struct conversion ([#1116](https://github.com/instill-ai/pipeline-backend/issues/1116)) ([9e06b7c](https://github.com/instill-ai/pipeline-backend/commit/9e06b7c12a5942e6a1ca50c8980fc16d184128ab))
* **data:** implement time types support with pattern validation ([#1115](https://github.com/instill-ai/pipeline-backend/issues/1115)) ([79630c0](https://github.com/instill-ai/pipeline-backend/commit/79630c0b3056aae21f47ae08c6e9e75d4071c98d))


### Bug Fixes

* **compogen:** escape curly braces for readme.com compatibility ([#1124](https://github.com/instill-ai/pipeline-backend/issues/1124)) ([904992d](https://github.com/instill-ai/pipeline-backend/commit/904992d509cf5d08ec8b230b58b17977159fb9ac))
* **component,ai,gemini:** add operation validation for cache task ([#1130](https://github.com/instill-ai/pipeline-backend/issues/1130)) ([9e19255](https://github.com/instill-ai/pipeline-backend/commit/9e1925524d78e1dd6582ad67cd1cb9cf10ee2056))
* **component,ai,gemini:** correct text-based documents logic ([#1103](https://github.com/instill-ai/pipeline-backend/issues/1103)) ([ed5a111](https://github.com/instill-ai/pipeline-backend/commit/ed5a11167f6b9a39c35e3646372752bfc9af06d0))
* **component,ai,gemini:** unify InlineData processing and enable images in streaming responses ([#1125](https://github.com/instill-ai/pipeline-backend/issues/1125)) ([3117046](https://github.com/instill-ai/pipeline-backend/commit/311704698553bd2abfa9e8faa77f8a699edc071b))
* **component,document:** fix incorrect expected value in the unit test ([#1138](https://github.com/instill-ai/pipeline-backend/issues/1138)) ([189dbd6](https://github.com/instill-ai/pipeline-backend/commit/189dbd6bf16bb95cfe3fe5f905b18020053573e4))
* **data:** remove duplicate dot in generated filenames ([#1136](https://github.com/instill-ai/pipeline-backend/issues/1136)) ([0a74a00](https://github.com/instill-ai/pipeline-backend/commit/0a74a00c36fb6db5bdcaa5439469da33ed63020c))
* **external:** fix Content-Disposition header parsing for filename extraction ([#1132](https://github.com/instill-ai/pipeline-backend/issues/1132)) ([869b081](https://github.com/instill-ai/pipeline-backend/commit/869b081e0c3a4022a11dcb65765ae8ef8c74965f))
* **service:** handle null JSON metadata in pipeline conversion ([#1134](https://github.com/instill-ai/pipeline-backend/issues/1134)) ([b244784](https://github.com/instill-ai/pipeline-backend/commit/b2447846277723df0df5938324d3ccac4c8480b5))
* **text:** correct positions on duplicate markdown chunks ([#1120](https://github.com/instill-ai/pipeline-backend/issues/1120)) ([1b4cd1f](https://github.com/instill-ai/pipeline-backend/commit/1b4cd1f0a8236c07bfb58117dfb1e692b90b14d1))
* **usage:** add missing error filtering for users/admin ([#1119](https://github.com/instill-ai/pipeline-backend/issues/1119)) ([cd1bd55](https://github.com/instill-ai/pipeline-backend/commit/cd1bd55680d69b0a6541e3c34058cb49c0af2aa6))


### Miscellaneous

* release v0.61.0 ([e1db93c](https://github.com/instill-ai/pipeline-backend/commit/e1db93c2f5563f6c1caf1f34309ccff5d46a9a54))


### Refactor

* **component,ai,gemini:** merge usage and usage-metadata fields into single usage field ([#1126](https://github.com/instill-ai/pipeline-backend/issues/1126)) ([a6046cd](https://github.com/instill-ai/pipeline-backend/commit/a6046cd151c9d1db6ab33ef67f5bb547d3cbc461))
* **component,ai.gemini:** standardize file api timeout and use native embedding type ([#1133](https://github.com/instill-ai/pipeline-backend/issues/1133)) ([174f7d6](https://github.com/instill-ai/pipeline-backend/commit/174f7d63f821cc4bd0f60616d0cbaead3ca5794d))
* **component,generic,http:** move test functions to test files and improve code legibility ([#1131](https://github.com/instill-ai/pipeline-backend/issues/1131)) ([1153a09](https://github.com/instill-ai/pipeline-backend/commit/1153a09aa3d51657372f25bb010e4ba8e31d591a))
* **component,generic,http:** replace env-based URL validation with constructor injection ([#1121](https://github.com/instill-ai/pipeline-backend/issues/1121)) ([f1f7d2f](https://github.com/instill-ai/pipeline-backend/commit/f1f7d2f2620ffc6356612797212ff041d599b439))


### Tests

* **component,generic,http:** replace external httpbin.org dependency with local test server ([#1101](https://github.com/instill-ai/pipeline-backend/issues/1101)) ([a82d155](https://github.com/instill-ai/pipeline-backend/commit/a82d155cf986f23996187881abca11852b6d7283))

## [0.60.0](https://github.com/instill-ai/pipeline-backend/compare/v0.59.2...v0.60.0) (2025-09-18)


### Features

* **artifact:** expose chunk file reference in search task ([#1085](https://github.com/instill-ai/pipeline-backend/issues/1085)) ([39bbe95](https://github.com/instill-ai/pipeline-backend/commit/39bbe958ffd792ceed1c8443c14d2e0c7bfd6b71))
* **component,ai:** add Gemini integration ([#1088](https://github.com/instill-ai/pipeline-backend/issues/1088)) ([cea127a](https://github.com/instill-ai/pipeline-backend/commit/cea127ad207f71e24db5194c4e32592cd87e01eb))
* **component,cohere:** add rerank indexes in the response ([#1087](https://github.com/instill-ai/pipeline-backend/issues/1087)) ([fe6366a](https://github.com/instill-ai/pipeline-backend/commit/fe6366a14a079920753b7888773198435fea7916))


### Bug Fixes

* **compogen:** remove redundant escape characters ([#1089](https://github.com/instill-ai/pipeline-backend/issues/1089)) ([9d21061](https://github.com/instill-ai/pipeline-backend/commit/9d21061e18d17b4544d10c4719feedef9a892a7a))
* **component,ai,instillmodel:** fix outdated data struct ([#1095](https://github.com/instill-ai/pipeline-backend/issues/1095)) ([c81f59c](https://github.com/instill-ai/pipeline-backend/commit/c81f59ce720950b667de9a1e97c3607362c42c95))
* **component,ai,instillmodel:** resolve panics and test failures ([#1100](https://github.com/instill-ai/pipeline-backend/issues/1100)) ([34fc930](https://github.com/instill-ai/pipeline-backend/commit/34fc9307fdb08b83b664393d7448b3078d0e60c9))
* **recipe:** support nil, null, undefined for condition field ([#1091](https://github.com/instill-ai/pipeline-backend/issues/1091)) ([a249070](https://github.com/instill-ai/pipeline-backend/commit/a249070ba2f7a399b3312933072a125987fcbf66))
* **usage:** treat input rendering error as fatal ([#1098](https://github.com/instill-ai/pipeline-backend/issues/1098)) ([06c8025](https://github.com/instill-ai/pipeline-backend/commit/06c8025ecc35774cf3cc2959546cb42c5c49caab))


### Miscellaneous

* **ce:** release v0.60.0 ([#1099](https://github.com/instill-ai/pipeline-backend/issues/1099)) ([09c5c0f](https://github.com/instill-ai/pipeline-backend/commit/09c5c0fa812f9a87c9a8abdc6596f9f3e1effea9))
* **compogen:** update component document layout ([#1090](https://github.com/instill-ai/pipeline-backend/issues/1090)) ([5613ee3](https://github.com/instill-ai/pipeline-backend/commit/5613ee3a2447170e4ca35ca2462cb58281677a2d))
* **component,ai:** remove unused files ([#1094](https://github.com/instill-ai/pipeline-backend/issues/1094)) ([11b0f4a](https://github.com/instill-ai/pipeline-backend/commit/11b0f4af3622ff5183b7a372467efbff6b324af6))
* **component,gemini:** optimize the IO struct ([#1092](https://github.com/instill-ai/pipeline-backend/issues/1092)) ([a0772d2](https://github.com/instill-ai/pipeline-backend/commit/a0772d2b040229b2156c3de54314692f7187534e))
* **data,component,gemini:** improve error msg ([#1093](https://github.com/instill-ai/pipeline-backend/issues/1093)) ([c2ea248](https://github.com/instill-ai/pipeline-backend/commit/c2ea248920a32b0bbe805e24d304a545b30edc08))
* **data:** improve unified Instill Type data presentation ([#1078](https://github.com/instill-ai/pipeline-backend/issues/1078)) ([abcccd6](https://github.com/instill-ai/pipeline-backend/commit/abcccd6ed0629d9a9b8c0fa16ba53ed3aea0d866))


### Documentation

* **component:** update description format ([#1084](https://github.com/instill-ai/pipeline-backend/issues/1084)) ([faaaed0](https://github.com/instill-ai/pipeline-backend/commit/faaaed0773a20914e43894903da8501449456ee6))


### Refactor

* **component,ai,gemini:** enhance document processing with text … ([#1097](https://github.com/instill-ai/pipeline-backend/issues/1097)) ([38639c6](https://github.com/instill-ai/pipeline-backend/commit/38639c6bb767403f094f32ad48f99698e84ff965))


### Tests

* **integration:** tinker script ([#1083](https://github.com/instill-ai/pipeline-backend/issues/1083)) ([1712bb6](https://github.com/instill-ai/pipeline-backend/commit/1712bb6cb06f607b1079af258c9313afbd068535))

## [0.59.2](https://github.com/instill-ai/pipeline-backend/compare/v0.59.1...v0.59.2) (2025-09-09)


### Miscellaneous

* **component-run:** make input and output blob paths consistent ([#1079](https://github.com/instill-ai/pipeline-backend/issues/1079)) ([58a01dc](https://github.com/instill-ai/pipeline-backend/commit/58a01dcbc0bdae4ba8c09386a7b0f3d0111b7635))
* **x:** update x to v0.10.0-alpha ([#1081](https://github.com/instill-ai/pipeline-backend/issues/1081)) ([6fae5fa](https://github.com/instill-ai/pipeline-backend/commit/6fae5fab187759553d6c4d19b1f761a4ea987ea2))

## [0.59.1](https://github.com/instill-ai/pipeline-backend/compare/v0.59.0...v0.59.1) (2025-09-02)


### Bug Fixes

* **log:** fix permission check for run logs ([#1076](https://github.com/instill-ai/pipeline-backend/issues/1076)) ([0757c79](https://github.com/instill-ai/pipeline-backend/commit/0757c7986d03ed6eed18b85b9b7b09a6c19cf64f))
* **type,document:** fix .doc OLE format ([#1075](https://github.com/instill-ai/pipeline-backend/issues/1075)) ([f541948](https://github.com/instill-ai/pipeline-backend/commit/f541948e2f58eee22e8a25e71080041e7ade832b))


### Documentation

* **component:** fix broken links ([#1074](https://github.com/instill-ai/pipeline-backend/issues/1074)) ([0dafd56](https://github.com/instill-ai/pipeline-backend/commit/0dafd56ccb5be7a31e1659c824a1b561bb00302f))

## [0.59.0](https://github.com/instill-ai/pipeline-backend/compare/v0.58.4...v0.59.0) (2025-08-26)


### Features

* **artifact:** allow search and query tasks to filter by several files ([#1069](https://github.com/instill-ai/pipeline-backend/issues/1069)) ([8b212fe](https://github.com/instill-ai/pipeline-backend/commit/8b212fe38e7a987cd6d08b363df5ab27749a149a))
* **minio:** use new bucket names ([#1071](https://github.com/instill-ai/pipeline-backend/issues/1071)) ([c40fbed](https://github.com/instill-ai/pipeline-backend/commit/c40fbedd63257adcf7369c5d4b794b792bbbb707))


### Bug Fixes

* **run:** return correct total duration ([#1072](https://github.com/instill-ai/pipeline-backend/issues/1072)) ([5a15c45](https://github.com/instill-ai/pipeline-backend/commit/5a15c45aef907263691f695482d9ed549bd465d7))


### Miscellaneous

* release v0.59.0 ([#1073](https://github.com/instill-ai/pipeline-backend/issues/1073)) ([143fd2b](https://github.com/instill-ai/pipeline-backend/commit/143fd2be031bfcf0283d3193466bfb36319dcb06))

## [0.58.4](https://github.com/instill-ai/pipeline-backend/compare/v0.58.3...v0.58.4) (2025-08-08)


### Features

* **component,openai:** add reasoning-effort and verbosity fields ([#1067](https://github.com/instill-ai/pipeline-backend/issues/1067)) ([d2f0f87](https://github.com/instill-ai/pipeline-backend/commit/d2f0f876f3f8fb09355b280abda6e0e336778514))

## [0.58.3](https://github.com/instill-ai/pipeline-backend/compare/v0.58.2...v0.58.3) (2025-08-07)


### Bug Fixes

* **component,googlesheets:** fix the wrong cell type ([#1065](https://github.com/instill-ai/pipeline-backend/issues/1065)) ([3ee8ba8](https://github.com/instill-ai/pipeline-backend/commit/3ee8ba8daef19982fc7f6a9ef46156d918cccdc0))

## [0.58.2](https://github.com/instill-ai/pipeline-backend/compare/v0.58.1...v0.58.2) (2025-08-07)


### Features

* **temporal:** remove Temporal namespace initialization ([#1059](https://github.com/instill-ai/pipeline-backend/issues/1059)) ([c940fa0](https://github.com/instill-ai/pipeline-backend/commit/c940fa0e4e1ea84335c8c1e82dfbd1d08a9fde94))


### Bug Fixes

* **component,googlesheets:** fix the wrong cell positions ([#1064](https://github.com/instill-ai/pipeline-backend/issues/1064)) ([d4de9e4](https://github.com/instill-ai/pipeline-backend/commit/d4de9e4a019023b596d26595fa96887ba16c025a))

## [0.58.1](https://github.com/instill-ai/pipeline-backend/compare/v0.58.0...v0.58.1) (2025-08-04)


### Features

* **component,perplexity:** adopt latest perplexity API ([#1060](https://github.com/instill-ai/pipeline-backend/issues/1060)) ([bc8a2b0](https://github.com/instill-ai/pipeline-backend/commit/bc8a2b0eb2d6446bcb481767b9854cb275f4a68a))

## [0.58.0](https://github.com/instill-ai/pipeline-backend/compare/v0.57.0...v0.58.0) (2025-07-31)


### Bug Fixes

* **cmd:** move Temporal namespace creation to cmd/init ([#1053](https://github.com/instill-ai/pipeline-backend/issues/1053)) ([11c6847](https://github.com/instill-ai/pipeline-backend/commit/11c6847609d0f85ee904e5c4982e11c5f0bd2c89))
* **component:** fix missing propagation of original request header instill-artifact component ([#1056](https://github.com/instill-ai/pipeline-backend/issues/1056)) ([2b9f265](https://github.com/instill-ai/pipeline-backend/commit/2b9f265aea2e1aa938cb3e990e2af32a9ee747b5))


### Miscellaneous

* **otel,config:** fix missing settings and configs ([#1057](https://github.com/instill-ai/pipeline-backend/issues/1057)) ([d8d99b2](https://github.com/instill-ai/pipeline-backend/commit/d8d99b22b0d3297e1d6a4a30a89436dc5046d2c9))
* release v0.58.0 ([4efe51a](https://github.com/instill-ai/pipeline-backend/commit/4efe51a6b409893dd9001ccc398ec464bded7d03))

## [0.57.0](https://github.com/instill-ai/pipeline-backend/compare/v0.56.0...v0.57.0) (2025-07-16)


### Features

* **artifact:** add file UID param in retrieval task ([#1046](https://github.com/instill-ai/pipeline-backend/issues/1046)) ([df241f5](https://github.com/instill-ai/pipeline-backend/commit/df241f514304f7d0abd79b8ed4c97c3b0b9c0a37))
* **component:** retire openai.v1 and universal-ai component ([#1042](https://github.com/instill-ai/pipeline-backend/issues/1042)) ([d332bbe](https://github.com/instill-ai/pipeline-backend/commit/d332bbe87fba2114b4c38ad3d384546a89673646))
* **external:** support new blob-url path in artifactBinaryFetcher ([#1037](https://github.com/instill-ai/pipeline-backend/issues/1037)) ([11a5d16](https://github.com/instill-ai/pipeline-backend/commit/11a5d16057f5b4dd775e4348265a2c671560e62a))
* **otel:** integrate OTEL using gRPC interceptor ([#1050](https://github.com/instill-ai/pipeline-backend/issues/1050)) ([70edddd](https://github.com/instill-ai/pipeline-backend/commit/70edddd40d249d58584b678b511e2cedbe4a91b3))


### Bug Fixes

* **external:** use URLEncoding to decode blob URL ([#1047](https://github.com/instill-ai/pipeline-backend/issues/1047)) ([893cf5b](https://github.com/instill-ai/pipeline-backend/commit/893cf5b797d27bc2a0532eb838b4bf73da0a087c))
* **init:** remove components from the database if they no longer exist in the definition list ([#1049](https://github.com/instill-ai/pipeline-backend/issues/1049)) ([82fe19d](https://github.com/instill-ai/pipeline-backend/commit/82fe19df724d24e61af4b613a5d83bc8ee39295c))


### Miscellaneous

* **dep:** bump up usage-client version ([#1043](https://github.com/instill-ai/pipeline-backend/issues/1043)) ([f357f03](https://github.com/instill-ai/pipeline-backend/commit/f357f035ebc5c83159c5bf6ec25a184e79cc400a))
* **deps:** bump github.com/go-chi/chi/v5 from 5.2.1 to 5.2.2 ([#1030](https://github.com/instill-ai/pipeline-backend/issues/1030)) ([6ad1380](https://github.com/instill-ai/pipeline-backend/commit/6ad138028e58cebe04db14b9846cc10ac3e87034))
* release v0.57.0 ([5789441](https://github.com/instill-ai/pipeline-backend/commit/5789441fa53d9dd930518902eb47b2b4ad9ea985))


### Refactor

* **main:** align backend codebase ([#1048](https://github.com/instill-ai/pipeline-backend/issues/1048)) ([9de0b03](https://github.com/instill-ai/pipeline-backend/commit/9de0b039b9618a760fcb9114e03b819ff4306322))

## [0.56.0](https://github.com/instill-ai/pipeline-backend/compare/v0.55.0...v0.56.0) (2025-07-01)


### Features

* **document:** add TASK_SPLIT_IN_PAGES ([#1035](https://github.com/instill-ai/pipeline-backend/issues/1035)) ([3bcb944](https://github.com/instill-ai/pipeline-backend/commit/3bcb94455c7c142ea8844c5013c795b76d011a6c))
* **http:** add header parameter in HTTP component ([#1028](https://github.com/instill-ai/pipeline-backend/issues/1028)) ([64d0807](https://github.com/instill-ai/pipeline-backend/commit/64d0807d0152876bad5164687f0447366e7a6ac1))


### Bug Fixes

* **component,document:** fix the wrong python script output parsing ([#1031](https://github.com/instill-ai/pipeline-backend/issues/1031)) ([b306708](https://github.com/instill-ai/pipeline-backend/commit/b306708cf7ca794a58ae5c97ed3c0051f037c8d0))
* **Dockerfile:** correct serviceVersion injection ([#1036](https://github.com/instill-ai/pipeline-backend/issues/1036)) ([61573ac](https://github.com/instill-ai/pipeline-backend/commit/61573ac1961e92645254a706b0a747c6f6f98a47))
* **document:** make Python scripts fail silently on document-to-markdown conversion ([#1032](https://github.com/instill-ai/pipeline-backend/issues/1032)) ([8b97006](https://github.com/instill-ai/pipeline-backend/commit/8b97006ec422013f75712d88f1ea82b3c4112fed))


### Miscellaneous

* **http:** block internal endpoints in HTTP component. ([#1033](https://github.com/instill-ai/pipeline-backend/issues/1033)) ([dff15b1](https://github.com/instill-ai/pipeline-backend/commit/dff15b1abfbc6514d110401c57cbccca6be3fc37))
* **main:** release v0.56.0 ([713e079](https://github.com/instill-ai/pipeline-backend/commit/713e079a02a0b71459e7290bd04df2b367dfcd95))
* **main:** release v0.56.0 ([#1039](https://github.com/instill-ai/pipeline-backend/issues/1039)) ([2273d5b](https://github.com/instill-ai/pipeline-backend/commit/2273d5bbcaa1686b31c1e26b9e906acc64c96c91))

## [0.55.0](https://github.com/instill-ai/pipeline-backend/compare/v0.54.0-rc...v0.55.0) (2025-06-18)


### Bug Fixes

* **component,http:** fix incorrect marshaling in the request body ([#1022](https://github.com/instill-ai/pipeline-backend/issues/1022)) ([1f19681](https://github.com/instill-ai/pipeline-backend/commit/1f19681974d1a819478b02f01a5c331e18321481))
* **component,openai:** resize the image before sending to OpenAI ([#1020](https://github.com/instill-ai/pipeline-backend/issues/1020)) ([0a65d6b](https://github.com/instill-ai/pipeline-backend/commit/0a65d6bf206ac31b93e44ac80b415b2a3849ac2b))


### Miscellaneous

* **config:** update config.json ([#1018](https://github.com/instill-ai/pipeline-backend/issues/1018)) ([56d80d6](https://github.com/instill-ai/pipeline-backend/commit/56d80d6d4054126b9a3c293196dec56855955b7a))
* **proto:** adopt latest `protogen-go` package ([#1024](https://github.com/instill-ai/pipeline-backend/issues/1024)) ([427598d](https://github.com/instill-ai/pipeline-backend/commit/427598d3c1bb3aa025a3fd0870bb491784612b36))
* release v0.55.0 ([3b516e0](https://github.com/instill-ai/pipeline-backend/commit/3b516e04aa4d1095c3cc93e9723e582a4b7b4d39))
* release v0.55.0 ([2cfa80c](https://github.com/instill-ai/pipeline-backend/commit/2cfa80c3203e08a72024f3454e3a7f4a3b01de8c))


### Documentation

* **CONTRIBUTING:** update content ([#1027](https://github.com/instill-ai/pipeline-backend/issues/1027)) ([3b84ff0](https://github.com/instill-ai/pipeline-backend/commit/3b84ff05fea11f379b05dfc4c1d2a0643d3b017d))

## [0.54.0-rc](https://github.com/instill-ai/pipeline-backend/compare/v0.53.0-beta...v0.54.0-rc) (2025-06-06)


### Features

* **iterator:** execute elements in iterator concurrently ([#1014](https://github.com/instill-ai/pipeline-backend/issues/1014)) ([25b3a35](https://github.com/instill-ai/pipeline-backend/commit/25b3a35b147a3a915b9283fe4108465a8ae87be1))


### Miscellaneous Chores

* **main:** release v0.45.0-rc ([#1016](https://github.com/instill-ai/pipeline-backend/issues/1016)) ([04d53cb](https://github.com/instill-ai/pipeline-backend/commit/04d53cbfad651c8bbd16f839c4f90ea866610f65))
* **main:** release v0.54.0-rc ([#1017](https://github.com/instill-ai/pipeline-backend/issues/1017)) ([3ab7010](https://github.com/instill-ai/pipeline-backend/commit/3ab7010aebde8b7e4a3bcbe59a7033a329a5199a))

## [0.53.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.52.5-beta...v0.53.0-beta) (2025-06-03)


### Features

* **temporal:** propagate request metadata to Temporal headers ([#1009](https://github.com/instill-ai/pipeline-backend/issues/1009)) ([0e90a99](https://github.com/instill-ai/pipeline-backend/commit/0e90a99775e7e69990b979817a94b386a15432bf))


### Bug Fixes

* **worker:** catch the error when creating Temporal session failed ([#1011](https://github.com/instill-ai/pipeline-backend/issues/1011)) ([d238d3b](https://github.com/instill-ai/pipeline-backend/commit/d238d3b4821459e425639d7355096bedc27ddea4))

## [0.52.5-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.52.4-beta...v0.52.5-beta) (2025-05-14)


### Bug Fixes

* **component-run:** set component status when execution failed ([#1008](https://github.com/instill-ai/pipeline-backend/issues/1008)) ([2addf5b](https://github.com/instill-ai/pipeline-backend/commit/2addf5b03ffcf439c1509011ce48d172077ecf0b))
* **component,audio:** fix wrong instill type ([#1005](https://github.com/instill-ai/pipeline-backend/issues/1005)) ([b999cf4](https://github.com/instill-ai/pipeline-backend/commit/b999cf4b7f2981cd2a72049c7091752bb7c55d1e))

## [0.52.4-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.52.3-beta...v0.52.4-beta) (2025-04-24)


### Bug Fixes

* **component,openai:** remove response format check ([#1003](https://github.com/instill-ai/pipeline-backend/issues/1003)) ([c6d02eb](https://github.com/instill-ai/pipeline-backend/commit/c6d02eb8e879a6888e66d7a422f78d353ae87a80))

## [0.52.3-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.52.2-beta...v0.52.3-beta) (2025-04-24)


### Bug Fixes

* **component,artifact:** remove duplicated connection close ([#1001](https://github.com/instill-ai/pipeline-backend/issues/1001)) ([6e98c0e](https://github.com/instill-ai/pipeline-backend/commit/6e98c0ed421a2b65409615cbca04cc7dde517b5f))

## [0.52.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.52.1-beta...v0.52.2-beta) (2025-04-10)


### Bug Fixes

* **iterator:** always initialize pipeline output template ([#997](https://github.com/instill-ai/pipeline-backend/issues/997)) ([3020465](https://github.com/instill-ai/pipeline-backend/commit/3020465c7ccee23c6729d2a41601c04a67ae03e6))

## [0.52.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.52.0-beta...v0.52.1-beta) (2025-04-01)


### Bug Fixes

* **component:** make component definition response compatiable with Console ([#995](https://github.com/instill-ai/pipeline-backend/issues/995)) ([9e4286c](https://github.com/instill-ai/pipeline-backend/commit/9e4286c5e31a39115fe58bdfe3990c05a4e522b1))

## [0.52.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.51.0-beta...v0.52.0-beta) (2025-03-28)


### Features

* **memory:** remove recipe from in-process memory ([#984](https://github.com/instill-ai/pipeline-backend/issues/984)) ([ca57ca8](https://github.com/instill-ai/pipeline-backend/commit/ca57ca82ea4550f2d0fc0d0c997184bdb376d3f9))
* **streaming:** use Redis pub/sub to stream pipeline triggers ([#981](https://github.com/instill-ai/pipeline-backend/issues/981)) ([80b7f17](https://github.com/instill-ai/pipeline-backend/commit/80b7f17eab2cb22fba7353101bb8313ec10b6294))
* **trigger:** use MinIO to store workflow memory ([#986](https://github.com/instill-ai/pipeline-backend/issues/986)) ([fc36c6c](https://github.com/instill-ai/pipeline-backend/commit/fc36c6c2d833a0882d47439600f35cfd35a6a053))
* **worker:** emit Temporal SDK metrics ([#990](https://github.com/instill-ai/pipeline-backend/issues/990)) ([1be2b44](https://github.com/instill-ai/pipeline-backend/commit/1be2b4427d8aa3d453a17bea6cf07a4a6a1c4c76))
* **worker:** extract worker back to its own process ([#987](https://github.com/instill-ai/pipeline-backend/issues/987)) ([8925b0f](https://github.com/instill-ai/pipeline-backend/commit/8925b0f0fb4efcfb24e06be411ace3c5233356be))


### Bug Fixes

* **ci:** optimize disk usage in coverage worklfow ([#982](https://github.com/instill-ai/pipeline-backend/issues/982)) ([f8f2707](https://github.com/instill-ai/pipeline-backend/commit/f8f2707e96abcd9587fbc99e992ac057a368f753))
* **component,document:** fix the document cannot be converted to an image ([#994](https://github.com/instill-ai/pipeline-backend/issues/994)) ([22e879f](https://github.com/instill-ai/pipeline-backend/commit/22e879f0b11185912bdfd5f92e401256efdf7443))
* **worker:** use pipeline server host in pipeline client ([#989](https://github.com/instill-ai/pipeline-backend/issues/989)) ([06e2a9d](https://github.com/instill-ai/pipeline-backend/commit/06e2a9deebb40c0015cdffbe3f311b28c4331f3f))

## [0.51.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.50.0-beta...v0.51.0-beta) (2025-02-25)


### Features

* **all:** rename VDP to pipeline ([#963](https://github.com/instill-ai/pipeline-backend/issues/963)) ([8ba570a](https://github.com/instill-ai/pipeline-backend/commit/8ba570ae4013d78e1454be4725211c9539b53b3f))
* **component:** support metadata filter in artifact component ([#979](https://github.com/instill-ai/pipeline-backend/issues/979)) ([624029a](https://github.com/instill-ai/pipeline-backend/commit/624029a8db9f2070ed54d429936eae54f5a33b31))
* **Docling:** prefetch model artifacts ([#964](https://github.com/instill-ai/pipeline-backend/issues/964)) ([c9ff323](https://github.com/instill-ai/pipeline-backend/commit/c9ff323851381de7bc4179cab129b4cb7241b434))
* **document:** convert PDF to Markdown with Docling ([#959](https://github.com/instill-ai/pipeline-backend/issues/959)) ([a9dbf55](https://github.com/instill-ai/pipeline-backend/commit/a9dbf5576e68617ca40693f7839636c80a48487e))
* **document:** log execution times for benchmarking ([#969](https://github.com/instill-ai/pipeline-backend/issues/969)) ([ac3e2c3](https://github.com/instill-ai/pipeline-backend/commit/ac3e2c3425a1c351ddda5ea0dd1e9d71ddf99ba0))
* **init:** remove preset pipeline downloader ([#970](https://github.com/instill-ai/pipeline-backend/issues/970)) ([11f8f5c](https://github.com/instill-ai/pipeline-backend/commit/11f8f5cbab1b8993248b3f48b3651a1b46ca857a))
* **minio:** add client info and user header to artifact binary fetcher ([#978](https://github.com/instill-ai/pipeline-backend/issues/978)) ([78c9c1f](https://github.com/instill-ai/pipeline-backend/commit/78c9c1f511faa4872d4c42161123ee6a6956669a))
* **minio:** add service name and version to MinIO requests ([#976](https://github.com/instill-ai/pipeline-backend/issues/976)) ([39c66cd](https://github.com/instill-ai/pipeline-backend/commit/39c66cdaef2e3ce73bd1f35cb46ac90306657262))
* **minio:** log MinIO actions with requester ([#972](https://github.com/instill-ai/pipeline-backend/issues/972)) ([8ba353e](https://github.com/instill-ai/pipeline-backend/commit/8ba353ef3e19a9071a06f6ecfa7f1e3fa4ef5931))
* **perplexity:** add new Sonar models ([#957](https://github.com/instill-ai/pipeline-backend/issues/957)) ([2699679](https://github.com/instill-ai/pipeline-backend/commit/2699679c34cb5f2d7dd76b568bd778762423cc99))
* **recipe:** rename `format` to `type` in variable section ([#971](https://github.com/instill-ai/pipeline-backend/issues/971)) ([88ead91](https://github.com/instill-ai/pipeline-backend/commit/88ead914d6477a9697e81537e77e3f58600ed80d))
* **x:** update MinIO package to delegate audit logs ([#973](https://github.com/instill-ai/pipeline-backend/issues/973)) ([f81287b](https://github.com/instill-ai/pipeline-backend/commit/f81287bfe1ee4b8e240796f1aa49e1cf78f6866e))


### Bug Fixes

* **ci:** registry image build ([#960](https://github.com/instill-ai/pipeline-backend/issues/960)) ([3a56698](https://github.com/instill-ai/pipeline-backend/commit/3a56698fe5f4d4109cbc1d8e713727bd8500c640))

## [0.50.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.49.1-beta...v0.50.0-beta) (2025-01-16)


### Features

* **component,github:** add more user and org tasks ([#880](https://github.com/instill-ai/pipeline-backend/issues/880)) ([d6466bf](https://github.com/instill-ai/pipeline-backend/commit/d6466bfae501f54b6cd6e6edb9410ce1e10c933f))
* **component,openai:** add supports for tools and predicted output ([#953](https://github.com/instill-ai/pipeline-backend/issues/953)) ([fc808a7](https://github.com/instill-ai/pipeline-backend/commit/fc808a7cfb888c0f59c2f84a55a490601c7969ca))
* **pipeline:** return the error from a component inside an iterator ([#955](https://github.com/instill-ai/pipeline-backend/issues/955)) ([bebe57f](https://github.com/instill-ai/pipeline-backend/commit/bebe57f58d7735e36e83c5181b0eacd4d550988a))

## [0.49.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.49.0-beta...v0.49.1-beta) (2024-12-23)


### Features

* **component,video:** add task to embed audio to video ([#939](https://github.com/instill-ai/pipeline-backend/issues/939)) ([1aa40c2](https://github.com/instill-ai/pipeline-backend/commit/1aa40c274ccc9ea3ad1e7e6e6d10e48740ed9eda))


### Bug Fixes

* **component,github:** fix data structs ([#944](https://github.com/instill-ai/pipeline-backend/issues/944)) ([804e56a](https://github.com/instill-ai/pipeline-backend/commit/804e56a59204bc52d4e0e8f3075331340fbbed68))
* **mod:** update golang.org/x/net module to fix vulnerability issue ([a2db7de](https://github.com/instill-ai/pipeline-backend/commit/a2db7dea69f47b7e0fa8fba5d70aecea405d88f7))


### Miscellaneous Chores

* release v0.49.1-beta ([51d676d](https://github.com/instill-ai/pipeline-backend/commit/51d676d1e1ab1a966d86baf4c1dcb62facdc916f))

## [0.49.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.48.5-beta...v0.49.0-beta) (2024-12-17)


### Features

* **pipeline:** add blob expiration time to run logs ([#938](https://github.com/instill-ai/pipeline-backend/issues/938)) ([fa7ef0e](https://github.com/instill-ai/pipeline-backend/commit/fa7ef0ee11cb4539e51bad7d92556aa9ca3d6f5d))

## [0.48.5-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.48.4-beta...v0.48.5-beta) (2024-12-11)


### Features

* **pinecone:** pinecone batch upsert ([#927](https://github.com/instill-ai/pipeline-backend/issues/927)) ([398adf9](https://github.com/instill-ai/pipeline-backend/commit/398adf90954ad4df27bf9228e97bdc106dd25464))
* **vdp:** upload component input output data ([#923](https://github.com/instill-ai/pipeline-backend/issues/923)) ([8b6caab](https://github.com/instill-ai/pipeline-backend/commit/8b6caab5b9add954cc56e1a400839a0e29ff2446))


### Bug Fixes

* **trigger:** add component intermediate data in the trigger stream/response ([#932](https://github.com/instill-ai/pipeline-backend/issues/932)) ([2077ae7](https://github.com/instill-ai/pipeline-backend/commit/2077ae7167eb0d4e1f06dfe8f599104eee37a206))
* **trigger:** resolve issue where default value for number cannot be an integer ([#933](https://github.com/instill-ai/pipeline-backend/issues/933)) ([134032a](https://github.com/instill-ai/pipeline-backend/commit/134032a26174e033b2b30983949b2e5858017eeb))


### Miscellaneous Chores

* release v0.48.5-beta ([043788d](https://github.com/instill-ai/pipeline-backend/commit/043788d091a758a97fc6be453c3211fe1cbfb26d))

## [0.48.4-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.48.3-beta...v0.48.4-beta) (2024-12-09)


### Features

* **pinecone:** Add rerank task for Pinecone component ([#773](https://github.com/instill-ai/pipeline-backend/issues/773)) ([e1fd611](https://github.com/instill-ai/pipeline-backend/commit/e1fd611b55a11798e5d1b5b6f52152392fe6f00f))
* **vdp:** upload raw inputs for run log ([#904](https://github.com/instill-ai/pipeline-backend/issues/904)) ([960f4c2](https://github.com/instill-ai/pipeline-backend/commit/960f4c2c131eff9dfcfb11cc1a90237f2179192c))


### Bug Fixes

* **component,http:** fix the request body marshalling ([#928](https://github.com/instill-ai/pipeline-backend/issues/928)) ([b47cc71](https://github.com/instill-ai/pipeline-backend/commit/b47cc71bff76b569ce7e19d934697ff7bdaf41a7))


### Miscellaneous Chores

* release v0.48.4-beta ([b08878e](https://github.com/instill-ai/pipeline-backend/commit/b08878ebf309ff18272538120dda06db2ef718f9))

## [0.48.3-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.48.2-beta...v0.48.3-beta) (2024-12-04)


### Bug Fixes

* **repository:** fix edition filter ([#921](https://github.com/instill-ai/pipeline-backend/issues/921)) ([634cacd](https://github.com/instill-ai/pipeline-backend/commit/634cacdcff0f976ef4e2c79db906eed3cdcabda9))

## [0.48.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.48.1-beta...v0.48.2-beta) (2024-12-04)


### Features

* **compogen:** replace * to any ([#917](https://github.com/instill-ai/pipeline-backend/issues/917)) ([87ca33a](https://github.com/instill-ai/pipeline-backend/commit/87ca33ab08de355040463af4ee1eb528eb6c5407))
* **compogen:** replace array[*] to array[any] ([#919](https://github.com/instill-ai/pipeline-backend/issues/919)) ([9990f0e](https://github.com/instill-ai/pipeline-backend/commit/9990f0ecd708099501eb55207df959fb7d647497))


### Miscellaneous Chores

* release v0.48.2-beta ([4989887](https://github.com/instill-ai/pipeline-backend/commit/4989887a703513329a6495e338434cdf816b3ece))

## [0.48.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.48.0-beta...v0.48.1-beta) (2024-12-03)


### Features

* **compogen:** add {{}} case for compogen ([#913](https://github.com/instill-ai/pipeline-backend/issues/913)) ([b976387](https://github.com/instill-ai/pipeline-backend/commit/b976387d62996f4140c8000a063b04259a8965ed))


### Bug Fixes

* **workflow:** fix wrong component id in SchedulePipelineWorkflow ([#916](https://github.com/instill-ai/pipeline-backend/issues/916)) ([934dc3d](https://github.com/instill-ai/pipeline-backend/commit/934dc3d9e299aabfea431a6d2f67536214f0814d))


### Miscellaneous Chores

* release v0.48.1-beta ([a448e4b](https://github.com/instill-ai/pipeline-backend/commit/a448e4bd96e7dd9bbff938fb8b42474a581090f0))

## [0.48.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.47.2-beta...v0.48.0-beta) (2024-12-03)


### Features

* **component,data,base:** enhance component features ([#885](https://github.com/instill-ai/pipeline-backend/issues/885)) ([68b396f](https://github.com/instill-ai/pipeline-backend/commit/68b396fb014d5018e854eb1b1e2f0131670f6e1f))
* **component,googlesheet:** implement the Google Sheet component ([#878](https://github.com/instill-ai/pipeline-backend/issues/878)) ([8a0ab93](https://github.com/instill-ai/pipeline-backend/commit/8a0ab93a5e3239c9bade1c7e9e0d106120852543))
* **component,instillapp:** remove instill-app component ([#911](https://github.com/instill-ai/pipeline-backend/issues/911)) ([4a7c970](https://github.com/instill-ai/pipeline-backend/commit/4a7c970fa1a3183190d84d51faa98fafda49139f))
* **component,openai:** support streaming for o1-mini and o1-preview models ([#882](https://github.com/instill-ai/pipeline-backend/issues/882)) ([367e957](https://github.com/instill-ai/pipeline-backend/commit/367e9577942992e754901163838c3b9d998f0e68))
* **component,schedule:** introduce the `schedule` component ([#900](https://github.com/instill-ai/pipeline-backend/issues/900)) ([9582942](https://github.com/instill-ai/pipeline-backend/commit/9582942cd0ff0b131b3fb9c831242a1a351faa35))
* **connection:** add lookup connection endpoint ([#888](https://github.com/instill-ai/pipeline-backend/issues/888)) ([86bffe8](https://github.com/instill-ai/pipeline-backend/commit/86bffe87de6e7d2c476eede73a4da45cbb722410))
* **data:** add support for default value in `instill` Go tag ([#891](https://github.com/instill-ai/pipeline-backend/issues/891)) ([b9c2d05](https://github.com/instill-ai/pipeline-backend/commit/b9c2d057f6c15c8ca9b5948b1b21d49122a58409))
* **leadiq:** implement leadiq component ([#874](https://github.com/instill-ai/pipeline-backend/issues/874)) ([78c2ec7](https://github.com/instill-ai/pipeline-backend/commit/78c2ec7020137ff25d8e1b4ac7ac4807896a1637))
* **perplexity:** add perplexity component ([#861](https://github.com/instill-ai/pipeline-backend/issues/861)) ([1fc7dbe](https://github.com/instill-ai/pipeline-backend/commit/1fc7dbee9d46c2bba87127194c3fdc5ce2ee30c7))
* **recipe:** separate the template and the rendered value for input and setup ([#886](https://github.com/instill-ai/pipeline-backend/issues/886)) ([3ec3dd6](https://github.com/instill-ai/pipeline-backend/commit/3ec3dd60a45c9f1358c29c09923fa221e91213b4))
* **recipe:** support `required` attribute for variable ([#901](https://github.com/instill-ai/pipeline-backend/issues/901)) ([da7f0e9](https://github.com/instill-ai/pipeline-backend/commit/da7f0e97beb202312a6150394707184fbe902091))
* **service:** add the file extension to the output filename ([#873](https://github.com/instill-ai/pipeline-backend/issues/873)) ([22b92b0](https://github.com/instill-ai/pipeline-backend/commit/22b92b03198bea0231e9780ef4f76b6552b88fc1))
* **service:** trigger latest release version for pipeline with run-on-event setting ([#896](https://github.com/instill-ai/pipeline-backend/issues/896)) ([0b1c071](https://github.com/instill-ai/pipeline-backend/commit/0b1c0711ec5a130f8781caef89fc8cebc000134c))
* **smartlead:** implement smartlead ([#879](https://github.com/instill-ai/pipeline-backend/issues/879)) ([f6baf2a](https://github.com/instill-ai/pipeline-backend/commit/f6baf2a898147a7504ce326dec015838ffd54b20))
* **text:** improve markdown chunking ([#889](https://github.com/instill-ai/pipeline-backend/issues/889)) ([d48b3ec](https://github.com/instill-ai/pipeline-backend/commit/d48b3ec444ef5e83a46dd260d21181076cd5bdb2))
* **trigger:** accept connection references in the pipeline trigger data ([#883](https://github.com/instill-ai/pipeline-backend/issues/883)) ([937bd01](https://github.com/instill-ai/pipeline-backend/commit/937bd01436091ec1cc303d07f84a2f257554a63c))
* **trigger:** enable optional values for variables ([#884](https://github.com/instill-ai/pipeline-backend/issues/884)) ([187f5fd](https://github.com/instill-ai/pipeline-backend/commit/187f5fd60fba7b5d7b74bb7b6c63e04d77fb3742))
* **vdp:** skip google drive for cloud version ([#899](https://github.com/instill-ai/pipeline-backend/issues/899)) ([5089397](https://github.com/instill-ai/pipeline-backend/commit/50893971ea8c6f1e1e18b8a886de9a4c890e1983))


### Bug Fixes

* **ci:** use xk6-sql driver and pin down versions ([#876](https://github.com/instill-ai/pipeline-backend/issues/876)) ([1f64d6c](https://github.com/instill-ai/pipeline-backend/commit/1f64d6c26a694a76831b53ccf69a2f0efae40915))
* **component,openai:** enable the use of system messages with chat history ([#905](https://github.com/instill-ai/pipeline-backend/issues/905)) ([ef3e66f](https://github.com/instill-ai/pipeline-backend/commit/ef3e66fd4fd9360730d4da0cbfe65e0bc210c309))
* **leadiq, smartlead:** change field to optional ([#892](https://github.com/instill-ai/pipeline-backend/issues/892)) ([9bd995d](https://github.com/instill-ai/pipeline-backend/commit/9bd995d001c019d2f54d87b977d2c702052b63ef))
* **web:** fix url and markdown position ([#893](https://github.com/instill-ai/pipeline-backend/issues/893)) ([af8f412](https://github.com/instill-ai/pipeline-backend/commit/af8f41204f5cb6a66628677c633ec4fcef7fbd22))

## [0.47.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.47.0-beta...v0.47.2-beta) (2024-11-21)


### Bug Fixes

* **github:** fix time format bug ([#866](https://github.com/instill-ai/pipeline-backend/issues/866)) ([94edc7c](https://github.com/instill-ai/pipeline-backend/commit/94edc7ca5971a0f51b7dca3fcdd08303bb6686a0))
* **migration:** add array index check for migration 36 ([03bbf91](https://github.com/instill-ai/pipeline-backend/commit/03bbf91ef78d0357c164bd8469e2f24d9200a97a))


### Miscellaneous Chores

* release v0.47.2-beta ([a5f5c07](https://github.com/instill-ai/pipeline-backend/commit/a5f5c07e1b733be9f7b586d40b7b3d27a7b37791))

## [0.47.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.46.0-beta...v0.47.0-beta) (2024-11-20)


### Features

* **component:** add support for event specifications ([#837](https://github.com/instill-ai/pipeline-backend/issues/837)) ([47a61cd](https://github.com/instill-ai/pipeline-backend/commit/47a61cd2173a7038a662d8cd9458b51de4f124e8))
* **component:** implement run-on-event for Slack and GitHub component ([#842](https://github.com/instill-ai/pipeline-backend/issues/842)) ([1b6a569](https://github.com/instill-ai/pipeline-backend/commit/1b6a5696e4b33d79d0c92bb1aff8a357020cc44f))
* convert pdf to image concurrently ([#818](https://github.com/instill-ai/pipeline-backend/issues/818)) ([4c0ad97](https://github.com/instill-ai/pipeline-backend/commit/4c0ad97673a6e2935cd8bc9608b57d2ad72b89b3))
* improve markdown chunking ([#822](https://github.com/instill-ai/pipeline-backend/issues/822)) ([af1a36a](https://github.com/instill-ai/pipeline-backend/commit/af1a36a60d1ae36b02d5cd422b357feccaaf233a))
* **json:** Support Rename Fields for JSON operator ([#813](https://github.com/instill-ai/pipeline-backend/issues/813)) ([093714e](https://github.com/instill-ai/pipeline-backend/commit/093714e34ac0846be8e10e0be61534a339e632c5))
* **recipe:** refactor run-on-event recipe structure ([#835](https://github.com/instill-ai/pipeline-backend/issues/835)) ([78ea418](https://github.com/instill-ai/pipeline-backend/commit/78ea4183cf236b7533ca4b890cbfcae836df3158))
* **recipe:** rename `instill-format` to `format` ([#798](https://github.com/instill-ai/pipeline-backend/issues/798)) ([80a9fc9](https://github.com/instill-ai/pipeline-backend/commit/80a9fc90a472b48c6ad97c45887da7e3be7709e0))
* **service:** implement PipelineErrorUpdated streaming event for pipeline errors ([#846](https://github.com/instill-ai/pipeline-backend/issues/846)) ([3156a5f](https://github.com/instill-ai/pipeline-backend/commit/3156a5fc406914035a482d3aa061205caec271e3))
* **vdp:** integrate blob storage to vdp ([#834](https://github.com/instill-ai/pipeline-backend/issues/834)) ([5311549](https://github.com/instill-ai/pipeline-backend/commit/53115493edbebfe2be8bee5218808faea7bded36))
* **web:** add input schema to improve web operator ([#819](https://github.com/instill-ai/pipeline-backend/issues/819)) ([f7e1fe9](https://github.com/instill-ai/pipeline-backend/commit/f7e1fe9dabc0affce606ee824f373e11774843c5))


### Bug Fixes

* **data:** refactor numberData to support both float and integer types ([#832](https://github.com/instill-ai/pipeline-backend/issues/832)) ([cf27452](https://github.com/instill-ai/pipeline-backend/commit/cf27452e529779fab314e843540afd732e6382d0))
* **document:** fix bug about convert to image ([#848](https://github.com/instill-ai/pipeline-backend/issues/848)) ([a381c27](https://github.com/instill-ai/pipeline-backend/commit/a381c2771dbed32bc52e7eb532a7aa5cfcc646e5))
* fix bug about unit type ([#826](https://github.com/instill-ai/pipeline-backend/issues/826)) ([a89fdf7](https://github.com/instill-ai/pipeline-backend/commit/a89fdf7db08a76a579a561abe2c88a892e2f8bde))
* **integration-test:** maximize build space on image build & push ([#823](https://github.com/instill-ai/pipeline-backend/issues/823)) ([a439d22](https://github.com/instill-ai/pipeline-backend/commit/a439d22849911ecde600057e4e8fc487b12354bd))
* **run:** set pipeline run status as failed when component fails ([#836](https://github.com/instill-ai/pipeline-backend/issues/836)) ([70a5c52](https://github.com/instill-ai/pipeline-backend/commit/70a5c526630393b163575987914d132d28704dca))
* **service:** add MIME type detection in the backend binaryFetcher ([#854](https://github.com/instill-ai/pipeline-backend/issues/854)) ([f434b2b](https://github.com/instill-ai/pipeline-backend/commit/f434b2bc46ae2472a50960e7232c66d0dac40957))
* **service:** add missing nil check in includeIteratorComponentDetail() ([#831](https://github.com/instill-ai/pipeline-backend/issues/831)) ([9cb5e9e](https://github.com/instill-ai/pipeline-backend/commit/9cb5e9e45f9070c84055c20966f50a5025db0e52))
* **service:** skip empty component definition in API response ([#847](https://github.com/instill-ai/pipeline-backend/issues/847)) ([d61b55e](https://github.com/instill-ai/pipeline-backend/commit/d61b55eaae8e16d50dfe347de2b608034e0860b2))
* unit tests ([#820](https://github.com/instill-ai/pipeline-backend/issues/820)) ([717200c](https://github.com/instill-ai/pipeline-backend/commit/717200cc96518435f1e89b506090c94785fa54ed))
* **vdp:** item does not contain the instill format, so we insert it ([#858](https://github.com/instill-ai/pipeline-backend/issues/858)) ([2d25401](https://github.com/instill-ai/pipeline-backend/commit/2d25401204bfdc2cb7ae052e0f722a5c92ea3ab9))
* **workflow:** allow integration usage within iterator ([#833](https://github.com/instill-ai/pipeline-backend/issues/833)) ([c9bd169](https://github.com/instill-ai/pipeline-backend/commit/c9bd169e05479f2b69f7694ee95e7c8209862a41))

## [0.46.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.45.2-beta...v0.46.0-beta) (2024-11-05)


### Features

* add `condition` field support for iterator ([#803](https://github.com/instill-ai/pipeline-backend/issues/803)) ([04b1252](https://github.com/instill-ai/pipeline-backend/commit/04b12529eb5a1fe5c4bc13fe89828fbdc403bc54))
* add markdowns per pages ([#792](https://github.com/instill-ai/pipeline-backend/issues/792)) ([3ee428e](https://github.com/instill-ai/pipeline-backend/commit/3ee428ef7be4a4c0f2e2e22bfe8b795b89333a0c))
* add resolution field ([#808](https://github.com/instill-ai/pipeline-backend/issues/808)) ([f15f6bf](https://github.com/instill-ai/pipeline-backend/commit/f15f6bf5de177f23b529a2b87c1809b9c10f4265))
* add task sync ([#793](https://github.com/instill-ai/pipeline-backend/issues/793)) ([41a1eeb](https://github.com/instill-ai/pipeline-backend/commit/41a1eeb47e7eadcf17c6db7141418aef41affb19))
* **component,audio:** add TASK_DETECT_ACTIVITY and TASK_SEGMENT ([#762](https://github.com/instill-ai/pipeline-backend/issues/762)) ([9e92a31](https://github.com/instill-ai/pipeline-backend/commit/9e92a311e76a4e9d937f372fa3f203d9b84ee22e))
* **component,http:** refactor `restapi` component to `http` component ([#797](https://github.com/instill-ai/pipeline-backend/issues/797)) ([c2b1862](https://github.com/instill-ai/pipeline-backend/commit/c2b18620a38e5f69cfc4b30e61f371a8824d9cd1))
* **component:** add error handling for missing conversation ([#806](https://github.com/instill-ai/pipeline-backend/issues/806)) ([54cc616](https://github.com/instill-ai/pipeline-backend/commit/54cc616726c2b6dd5c6ddaf3b677443de73e71c0))
* **component:** inject global secrets as environment variables ([#786](https://github.com/instill-ai/pipeline-backend/issues/786)) ([8d842a6](https://github.com/instill-ai/pipeline-backend/commit/8d842a668c96e4557db9a63421d94f4c352be17b))
* convert time type to string ([#809](https://github.com/instill-ai/pipeline-backend/issues/809)) ([7de8465](https://github.com/instill-ai/pipeline-backend/commit/7de84651b4019e69806c1b0ea461084a37d6fbd5))
* **googledrive:** add the tasks for google drive ([#725](https://github.com/instill-ai/pipeline-backend/issues/725)) ([b6fe968](https://github.com/instill-ai/pipeline-backend/commit/b6fe9686c688f9c3b7de5942510e2818d893ade6))
* **integration:** identify supported OAuth integrations through global secrets ([#791](https://github.com/instill-ai/pipeline-backend/issues/791)) ([5a96453](https://github.com/instill-ai/pipeline-backend/commit/5a964532e6cbc35eb6778626f8a5161c29dffc15))
* **minio:** import updated minio package and add tag on file upload ([#779](https://github.com/instill-ai/pipeline-backend/issues/779)) ([ef86318](https://github.com/instill-ai/pipeline-backend/commit/ef863189a3c1617e7a47e89ba1632863f7e122ec))
* revamp Instill Format ([#774](https://github.com/instill-ai/pipeline-backend/issues/774)) ([24153e2](https://github.com/instill-ai/pipeline-backend/commit/24153e2c57ba4ce508059a0bd1af8528b07b5ed3))
* support `length` attribute for array data ([#810](https://github.com/instill-ai/pipeline-backend/issues/810)) ([fb4f4f7](https://github.com/instill-ai/pipeline-backend/commit/fb4f4f73064327be025999a4f9fbf31b8ac2e230))
* **web:** refactor the web operator ([#772](https://github.com/instill-ai/pipeline-backend/issues/772)) ([ae4e3c2](https://github.com/instill-ai/pipeline-backend/commit/ae4e3c2d21951d666bbf0b0ae4c384d18446e41f))


### Bug Fixes

* **component,image:** fix missing show score draw ([#801](https://github.com/instill-ai/pipeline-backend/issues/801)) ([a405bf7](https://github.com/instill-ai/pipeline-backend/commit/a405bf70070222108a8500ae946dbf653c1ca1a0))
* fix bug not to return error if there is no app or conversation ([#816](https://github.com/instill-ai/pipeline-backend/issues/816)) ([a946cfd](https://github.com/instill-ai/pipeline-backend/commit/a946cfd22ede65f608d79efcbb2d52e14fadc692))
* fix iterator upstream check ([#794](https://github.com/instill-ai/pipeline-backend/issues/794)) ([671971f](https://github.com/instill-ai/pipeline-backend/commit/671971f5e1ed69f87f76ec75b2a0d96db3637e62))
* **run:** add metadata retention handler ([#800](https://github.com/instill-ai/pipeline-backend/issues/800)) ([25ec0c2](https://github.com/instill-ai/pipeline-backend/commit/25ec0c227521cebbe79e873be9e859649341079d))
* **run:** add namespace id in response ([#811](https://github.com/instill-ai/pipeline-backend/issues/811)) ([8d29ffb](https://github.com/instill-ai/pipeline-backend/commit/8d29ffbc813545833f9694b78bef07a31b428a22))
* **run:** rename pipeline run columns and fix tests ([#776](https://github.com/instill-ai/pipeline-backend/issues/776)) ([98f1e00](https://github.com/instill-ai/pipeline-backend/commit/98f1e001f683c8b5ae72b553e3217f0459ac3ef8))
* **slack:** correct link to OAuth config in documentation ([#805](https://github.com/instill-ai/pipeline-backend/issues/805)) ([aa0752d](https://github.com/instill-ai/pipeline-backend/commit/aa0752dca35ff49960c35d6082e295d3a11b16d6))

## [0.45.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.45.1-beta...v0.45.2-beta) (2024-10-29)


### Features

* **collection:** add task split ([#780](https://github.com/instill-ai/pipeline-backend/issues/780)) ([1719e48](https://github.com/instill-ai/pipeline-backend/commit/1719e48fbea205ed8a8ac84c6d6c262fd89ab86e))


### Miscellaneous Chores

* release v0.45.2-beta ([777a362](https://github.com/instill-ai/pipeline-backend/commit/777a36212312797e32beedae1d9b967c3b8f04e4))

## [0.45.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.45.0-beta...v0.45.1-beta) (2024-10-25)


### Features

* **component, video:** implemented fractional fps value - fps as number, and added some debugging flags for ffmpeg ([#765](https://github.com/instill-ai/pipeline-backend/issues/765)) ([8a93442](https://github.com/instill-ai/pipeline-backend/commit/8a934420a01c986181b1660d1c13178ff7e79105))
* **component:** add support for streaming in the Anthropic and Mistral components ([#781](https://github.com/instill-ai/pipeline-backend/issues/781)) ([66f15fe](https://github.com/instill-ai/pipeline-backend/commit/66f15fe935183bb79610cd397d259b95bd253a17))


### Miscellaneous Chores

* release v0.45.1-beta ([60a58bd](https://github.com/instill-ai/pipeline-backend/commit/60a58bde6c163e50dfa53c522187441ec098d043))

## [0.45.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.44.0-beta...v0.45.0-beta) (2024-10-23)


### Features

* **run:** add requester id in list pipeline run response ([#770](https://github.com/instill-ai/pipeline-backend/issues/770)) ([a89a03d](https://github.com/instill-ai/pipeline-backend/commit/a89a03d2a9f4961831da88a25d7a2f6dd3bb3f73))

## [0.44.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.43.2-beta...v0.44.0-beta) (2024-10-22)


### Features

* **collection:** add concat ([#748](https://github.com/instill-ai/pipeline-backend/issues/748)) ([04d1467](https://github.com/instill-ai/pipeline-backend/commit/04d1467cef580ee423651ebed86a7a5ef6e64b64))
* **compogen:** improve Title Case capitalization ([#757](https://github.com/instill-ai/pipeline-backend/issues/757)) ([f956ead](https://github.com/instill-ai/pipeline-backend/commit/f956ead548168e8cd2e1a20a99cf6bb0d03a057f))
* **component:** update documentation URL to component ID ([#749](https://github.com/instill-ai/pipeline-backend/issues/749)) ([d4083c2](https://github.com/instill-ai/pipeline-backend/commit/d4083c262801f302ad705c830db2907c1273560a))
* **instillmodel:** implement instill model embedding ([#727](https://github.com/instill-ai/pipeline-backend/issues/727)) ([17d88bc](https://github.com/instill-ai/pipeline-backend/commit/17d88bc46dbcf118dcf3e181c6886de6f309b29e))
* **run:** run logging data list by requester API ([#730](https://github.com/instill-ai/pipeline-backend/issues/730)) ([e1e844b](https://github.com/instill-ai/pipeline-backend/commit/e1e844b4f6db226a29ba2937f0f05005a5899d49))
* **slack:** enable OAuth 2.0 integration ([#758](https://github.com/instill-ai/pipeline-backend/issues/758)) ([8043dc5](https://github.com/instill-ai/pipeline-backend/commit/8043dc5b1d564cecd4f5227ad57b926189ab243f))
* standardize the tag naming convention ([#767](https://github.com/instill-ai/pipeline-backend/issues/767)) ([fd0500f](https://github.com/instill-ai/pipeline-backend/commit/fd0500f56f91c006f82a3201f435c032215a4513))
* **web:** refactor web operator ([#753](https://github.com/instill-ai/pipeline-backend/issues/753)) ([700805f](https://github.com/instill-ai/pipeline-backend/commit/700805f492652482c07472a9965ae06e1384c86c))


### Bug Fixes

* **groq:** use credential-supported model in example ([#752](https://github.com/instill-ai/pipeline-backend/issues/752)) ([fc90435](https://github.com/instill-ai/pipeline-backend/commit/fc904350c978dd5c57fbe24aaf32be8945f3b9f5))
* **run:** not return minio error in list pipeline run ([#744](https://github.com/instill-ai/pipeline-backend/issues/744)) ([4d0afa1](https://github.com/instill-ai/pipeline-backend/commit/4d0afa16baa0ddcfe29371d8d3106eb7d066574f))
* **whatsapp:** fix dir name ([#763](https://github.com/instill-ai/pipeline-backend/issues/763)) ([029aef9](https://github.com/instill-ai/pipeline-backend/commit/029aef91459feca4c1111372ba386d9272a1870c))

## [0.43.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.43.1-beta...v0.43.2-beta) (2024-10-16)


### Bug Fixes

* fix InstillFormat validate function ([#745](https://github.com/instill-ai/pipeline-backend/issues/745)) ([b2e0705](https://github.com/instill-ai/pipeline-backend/commit/b2e070591c1d6795a70f27ac37a4b82629d2d6a3))

## [0.43.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.43.0-beta...v0.43.1-beta) (2024-10-15)


### Bug Fixes

* **web:** fix bug ([#742](https://github.com/instill-ai/pipeline-backend/issues/742)) ([6dbaf6e](https://github.com/instill-ai/pipeline-backend/commit/6dbaf6ee1bcf26cea44ded69ecfe69d5495315a8))

## [0.43.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.42.2-beta...v0.43.0-beta) (2024-10-15)


### Features

* **component:** implement instill app features ([#670](https://github.com/instill-ai/pipeline-backend/issues/670)) ([6def4ac](https://github.com/instill-ai/pipeline-backend/commit/6def4acac0148f5dfd873f477a05e70337389559))
* **slack:** allow posting messages as user or bot ([#728](https://github.com/instill-ai/pipeline-backend/issues/728)) ([7f35ca5](https://github.com/instill-ai/pipeline-backend/commit/7f35ca5b3afd3f2ad6186eb0bbc2d214ec41572c))


### Bug Fixes

* hide preset pipelines in GET /pipelines endpoint ([#733](https://github.com/instill-ai/pipeline-backend/issues/733)) ([baa8919](https://github.com/instill-ai/pipeline-backend/commit/baa891931771a351280c19d430b0d7f6fd2d4b76))
* remove redundant message in streaming ([#739](https://github.com/instill-ai/pipeline-backend/issues/739)) ([50ce029](https://github.com/instill-ai/pipeline-backend/commit/50ce0293d5805da1365a0fafa7e39f257ae83028))
* **text:** filter out the empty chunk ([#734](https://github.com/instill-ai/pipeline-backend/issues/734)) ([4037ba6](https://github.com/instill-ai/pipeline-backend/commit/4037ba6657975bed24601a60457aa80661396198))
* the InstillAcceptFormats doesn't works with collection component ([#732](https://github.com/instill-ai/pipeline-backend/issues/732)) ([3d1c053](https://github.com/instill-ai/pipeline-backend/commit/3d1c053685ba10ea50b61d5674547134482d11c7))

## [0.42.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.42.1-beta...v0.42.2-beta) (2024-10-08)


### Bug Fixes

* fix he pipeline output streaming was not functioning properly ([#723](https://github.com/instill-ai/pipeline-backend/issues/723)) ([6ca5060](https://github.com/instill-ai/pipeline-backend/commit/6ca50603f7bd96c0c1b085b186ac8dcc7b644bc3))

## [0.42.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.42.0-beta...v0.42.1-beta) (2024-10-08)


### Bug Fixes

* fix event for iterator input updates was not being sent ([#721](https://github.com/instill-ai/pipeline-backend/issues/721)) ([046d532](https://github.com/instill-ai/pipeline-backend/commit/046d53278e82d91f766d384f7e8fdd15a234de3e))

## [0.42.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.41.0-beta...v0.42.0-beta) (2024-10-08)


### Features

* **document:** repair pdf with libreoffice ([#683](https://github.com/instill-ai/pipeline-backend/issues/683)) ([b6a738f](https://github.com/instill-ai/pipeline-backend/commit/b6a738f7633125e04797a7fa975febb7280f17f9))
* **integration:** add identity to OAuth connections ([#699](https://github.com/instill-ai/pipeline-backend/issues/699)) ([e2e9237](https://github.com/instill-ai/pipeline-backend/commit/e2e92376583283c2f7178bb6226f586ef407c1e3))
* merge worker into main process and optimize graceful shutdown mechanism ([#713](https://github.com/instill-ai/pipeline-backend/issues/713)) ([69b64a0](https://github.com/instill-ai/pipeline-backend/commit/69b64a02e0aff2873521c6291eb6f1b046d4b36a))
* **universalai:** add instill credit function for universal ai component ([#680](https://github.com/instill-ai/pipeline-backend/issues/680)) ([9ce10b5](https://github.com/instill-ai/pipeline-backend/commit/9ce10b5f6cbc75a444fef1937dd97dfc0439c8ff))
* use explicit parameter for target namespace when cloning pipeline ([#711](https://github.com/instill-ai/pipeline-backend/issues/711)) ([ab2e3a6](https://github.com/instill-ai/pipeline-backend/commit/ab2e3a6ad51f6801fcb689a0d5b836a967fb99a3))
* **web:** fix bug and add mock users' behaviours by chromedp ([#701](https://github.com/instill-ai/pipeline-backend/issues/701)) ([ca312fe](https://github.com/instill-ai/pipeline-backend/commit/ca312feb5ac0e908a8f667e4447526c7eae83a15))


### Bug Fixes

* **component:** use kebab-case for icon name ([#703](https://github.com/instill-ai/pipeline-backend/issues/703)) ([cd587e2](https://github.com/instill-ai/pipeline-backend/commit/cd587e2df22134afd05a1da1a403f1cc87689d91))
* **document:** catch the error if the bbox is out of boundary ([#718](https://github.com/instill-ai/pipeline-backend/issues/718)) ([4fe03e6](https://github.com/instill-ai/pipeline-backend/commit/4fe03e6d5a94dd4dac36d361caf020efa8ba4e1b))
* fix the component render error was not being returned ([#707](https://github.com/instill-ai/pipeline-backend/issues/707)) ([931f067](https://github.com/instill-ai/pipeline-backend/commit/931f067dd82d68f05233eaa72bb8d34784f4284f))
* fix the wrong target namespace when cloning pipeline ([#706](https://github.com/instill-ai/pipeline-backend/issues/706)) ([75a757b](https://github.com/instill-ai/pipeline-backend/commit/75a757b4c4ae65218e9ae670ffb5a7a35bf0a966))
* **integration:** update GitHub scopes to read/write repository issues ([#709](https://github.com/instill-ai/pipeline-backend/issues/709)) ([c1cf4ce](https://github.com/instill-ai/pipeline-backend/commit/c1cf4ce08b1e5917260b9e23f362dfa2751eb7ae))
* **text:** fix the bug about the list position ([#712](https://github.com/instill-ai/pipeline-backend/issues/712)) ([49aed28](https://github.com/instill-ai/pipeline-backend/commit/49aed286b413c9e37b664e32e195c108d4e1adeb))

## [0.41.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.40.0-beta...v0.41.0-beta) (2024-10-03)


### Features

* add `range` support for iterator implementation ([#693](https://github.com/instill-ai/pipeline-backend/issues/693)) ([47fafdd](https://github.com/instill-ai/pipeline-backend/commit/47fafdd60a3fe4f8e6bfb176147db78f13c8b0a3))
* Add assignees and labels support for create issue task in GitHub component ([#686](https://github.com/instill-ai/pipeline-backend/issues/686)) ([c1c2517](https://github.com/instill-ai/pipeline-backend/commit/c1c2517c96e0e52eafe8bbeefedab95777354041))
* **component:** add scopes to OAuth integrations ([#696](https://github.com/instill-ai/pipeline-backend/issues/696)) ([8fd9149](https://github.com/instill-ai/pipeline-backend/commit/8fd9149c6c470550f2f832b7158f6cbe5584f48b))
* **component:** update contribution guidelines ([#685](https://github.com/instill-ai/pipeline-backend/issues/685)) ([6c42b71](https://github.com/instill-ai/pipeline-backend/commit/6c42b71bcdc166f2325d6da5b0ae30e0de9f8d0f))
* support map representation for `range` in iterator ([#695](https://github.com/instill-ai/pipeline-backend/issues/695)) ([956b1d1](https://github.com/instill-ai/pipeline-backend/commit/956b1d1fb2389d0e51e7039f390b51f1e5020594))


### Bug Fixes

* **component:** update source URL in component definition ([#694](https://github.com/instill-ai/pipeline-backend/issues/694)) ([90ad5ce](https://github.com/instill-ai/pipeline-backend/commit/90ad5ce2fe53a0d90c08819bdf0ed497c7e19ed1))
* the fallback mechanism for handling Instill Format with subtypes ([#697](https://github.com/instill-ai/pipeline-backend/issues/697)) ([9844756](https://github.com/instill-ai/pipeline-backend/commit/9844756d1166386479dff3b63bd9259813153192))

## [0.40.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.39.1-beta...v0.40.0-beta) (2024-10-01)


### Features

* **image:** add basic operation tasks ([#684](https://github.com/instill-ai/pipeline-backend/issues/684)) ([2a2ce6c](https://github.com/instill-ai/pipeline-backend/commit/2a2ce6ca0e1e56fefe5bff6da07cfaa909ea7a09))

## [0.39.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.39.0-beta...v0.39.1-beta) (2024-09-30)


### Features

* **integration:** load only connections present in the recipe and just once ([#671](https://github.com/instill-ai/pipeline-backend/issues/671)) ([99fdbaf](https://github.com/instill-ai/pipeline-backend/commit/99fdbaf313b5ba7b582d44cf858fa8c7e6be9ff1))


### Bug Fixes

* **document:** change code for component dependency ([#676](https://github.com/instill-ai/pipeline-backend/issues/676)) ([1f52488](https://github.com/instill-ai/pipeline-backend/commit/1f5248865f7fc1395037da41c3b7f684bd481d1c))
* **proto:** remove protoc-gen-validate dependencies ([#679](https://github.com/instill-ai/pipeline-backend/issues/679)) ([5e51349](https://github.com/instill-ai/pipeline-backend/commit/5e51349251ee1f8f3540d4c030edbe6ccf0a9c30))
* **streaming:** close connection and purge workflow memory at the end of the trigger workflow ([#672](https://github.com/instill-ai/pipeline-backend/issues/672)) ([127a68e](https://github.com/instill-ai/pipeline-backend/commit/127a68e00f14f305501910e466b8ff56ece03ce4))
* the output schema is incorrect when there is an indeterministic instillFormat ([#677](https://github.com/instill-ai/pipeline-backend/issues/677)) ([b4d5be4](https://github.com/instill-ai/pipeline-backend/commit/b4d5be4ee7359be21f0f1272b9c551f4642b355a))


### Miscellaneous Chores

* release v0.39.1-beta ([86507bb](https://github.com/instill-ai/pipeline-backend/commit/86507bbecafd6e2549c475e2cc4a42d31511c112))

## [0.39.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.38.4-beta...v0.39.0-beta) (2024-09-24)


### Features

* add `endpoints` field in pipeline response ([#665](https://github.com/instill-ai/pipeline-backend/issues/665)) ([f866aa0](https://github.com/instill-ai/pipeline-backend/commit/f866aa070a8ec9e87d94cf7d30f452581fe32848))
* **integration:** support OAuth connections ([#661](https://github.com/instill-ai/pipeline-backend/issues/661)) ([a48275b](https://github.com/instill-ai/pipeline-backend/commit/a48275b924442a2395c6c14b1bbb158188e513b4))


### Bug Fixes

* **document:** fix bugs about image to text ([#662](https://github.com/instill-ai/pipeline-backend/issues/662)) ([163a9f7](https://github.com/instill-ai/pipeline-backend/commit/163a9f70c35d72f5abdb865743646fc5a489bb32))
* make the pipeline share-code immutable ([#666](https://github.com/instill-ai/pipeline-backend/issues/666)) ([529de97](https://github.com/instill-ai/pipeline-backend/commit/529de9746ee595fd633b61a8a06d9a82b6ff2021))
* the pipeline output couldn't reference properties in JSON data ([#664](https://github.com/instill-ai/pipeline-backend/issues/664)) ([698cbf1](https://github.com/instill-ai/pipeline-backend/commit/698cbf10c6e21ea2a9b9cb644284230d29fb7627))

## [0.38.4-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.38.3-beta...v0.38.4-beta) (2024-09-18)


### Miscellaneous Chores

* release v0.38.4-beta ([2343c27](https://github.com/instill-ai/pipeline-backend/commit/2343c27b305a781067dea402f19ffd9fba7f91bd))

## [0.38.3-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.38.2-beta...v0.38.3-beta) (2024-09-16)


### Bug Fixes

* fix condition field can not reference variable ([#654](https://github.com/instill-ai/pipeline-backend/issues/654)) ([2342857](https://github.com/instill-ai/pipeline-backend/commit/23428573b1a2bb1d9ec81b24d14882c68b9fa8e0))

## [0.38.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.38.1-beta...v0.38.2-beta) (2024-09-13)


### Features

* allow users in organization to have owner permission in run log… ([#632](https://github.com/instill-ai/pipeline-backend/issues/632)) ([69158b9](https://github.com/instill-ai/pipeline-backend/commit/69158b9f74fb766f665553d2a1ee8ead7dd75ed0))


### Miscellaneous Chores

* release v0.38.2-beta ([14fbab4](https://github.com/instill-ai/pipeline-backend/commit/14fbab431d28372959e62edca487b92acb11627f))
* release v0.38.2-beta ([cab5fbc](https://github.com/instill-ai/pipeline-backend/commit/cab5fbca7d5201e5fbf013d8174e12aa5d328dd6))

## [0.38.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.38.0-beta...v0.38.1-beta) (2024-09-12)


### Miscellaneous Chores

* release v0.38.1-beta ([59b0830](https://github.com/instill-ai/pipeline-backend/commit/59b08305ef31048e640ecf3017bd79c5562dc460))

## [0.38.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.37.0-beta...v0.38.0-beta) (2024-09-10)


### Features

* add endpoints to read integrations ([#611](https://github.com/instill-ai/pipeline-backend/issues/611)) ([9be491b](https://github.com/instill-ai/pipeline-backend/commit/9be491b6e8f9fa92295ea73f8df10fef97897855))
* **integration:** add create, update, delete connection endpoints ([#629](https://github.com/instill-ai/pipeline-backend/issues/629)) ([b784553](https://github.com/instill-ai/pipeline-backend/commit/b78455394f5e532c82f7280dabdfa33cd23fe0d2))
* **integration:** add delete and update connection endpoints ([#636](https://github.com/instill-ai/pipeline-backend/issues/636)) ([b28db2c](https://github.com/instill-ai/pipeline-backend/commit/b28db2c7987b08174c9ae8dd5b8f99989d65cca1))
* **integration:** add GetConnection endpoint ([#633](https://github.com/instill-ai/pipeline-backend/issues/633)) ([0f4e9ca](https://github.com/instill-ai/pipeline-backend/commit/0f4e9ca38454525f99d722250147658837975ddb))
* **integration:** add ListNamespaceConnections endpoint ([#635](https://github.com/instill-ai/pipeline-backend/issues/635)) ([404621a](https://github.com/instill-ai/pipeline-backend/commit/404621ae00c8067bb0d76fc1e94f73d167eb093c))
* **integration:** list pipeline IDs by connection reference ([#642](https://github.com/instill-ai/pipeline-backend/issues/642)) ([0c398fe](https://github.com/instill-ai/pipeline-backend/commit/0c398fe317d02dea5c88dceb99a000308e554569))
* **integration:** reference connections in pipelines ([#638](https://github.com/instill-ai/pipeline-backend/issues/638)) ([21fcbdb](https://github.com/instill-ai/pipeline-backend/commit/21fcbdbc6cb223e29034a294c9cecaa31e3cdb64))
* **run:** capture component run error for run logging ([#639](https://github.com/instill-ai/pipeline-backend/issues/639)) ([1492214](https://github.com/instill-ai/pipeline-backend/commit/1492214d6b497f16dac1d80812ceace818061950))


### Bug Fixes

* fix duplicated metric datapoint ([#645](https://github.com/instill-ai/pipeline-backend/issues/645)) ([c93cf23](https://github.com/instill-ai/pipeline-backend/commit/c93cf2333449cbd7b52a72849913964d97b67b04))
* fix memory leak ([b97ba80](https://github.com/instill-ai/pipeline-backend/commit/b97ba8037a36062c507e772a1566cbb363b8af59))
* fix nil-check ([a86722f](https://github.com/instill-ai/pipeline-backend/commit/a86722ffec4d42365f94054929d12ce02b8bf0f3))
* remove goroutine in ConvertPipelinesToPB ([2d15fbe](https://github.com/instill-ai/pipeline-backend/commit/2d15fbe9c8f871dac43c657a0151de8ec46dfe87))
* reset the preset pipeline sharing setting ([#643](https://github.com/instill-ai/pipeline-backend/issues/643)) ([1ca23ac](https://github.com/instill-ai/pipeline-backend/commit/1ca23acaca559fd583ca233834f0c867f4db9a88))

## [0.37.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.36.0-beta...v0.37.0-beta) (2024-08-29)


### Features

* add support for filtering by numberOfRuns and numberOfClones ([#609](https://github.com/instill-ai/pipeline-backend/issues/609)) ([a5b9571](https://github.com/instill-ai/pipeline-backend/commit/a5b957177b46c9f20330bc5c9a2d70d0852c7fa4))
* add supports for Data properties ([#613](https://github.com/instill-ai/pipeline-backend/issues/613)) ([1c605dc](https://github.com/instill-ai/pipeline-backend/commit/1c605dc153cc97fd4d8a56aa3e7743fa6e3dde88))
* allow storing illegal YAML recipes ([#618](https://github.com/instill-ai/pipeline-backend/issues/618)) ([2d585dc](https://github.com/instill-ai/pipeline-backend/commit/2d585dc593dc6bea556546bf2e356e0b57123b50))
* re-implement streaming and introduce new data struct ([#603](https://github.com/instill-ai/pipeline-backend/issues/603)) ([fe7851f](https://github.com/instill-ai/pipeline-backend/commit/fe7851f3b759c3a1d1806000ffbb9e876ccbd306))
* return component status in all component events ([#612](https://github.com/instill-ai/pipeline-backend/issues/612)) ([9c1fad8](https://github.com/instill-ai/pipeline-backend/commit/9c1fad890859b7210e3496fbabb1be373523f1fd))
* return error message when using streaming ([#608](https://github.com/instill-ai/pipeline-backend/issues/608)) ([3f47833](https://github.com/instill-ai/pipeline-backend/commit/3f478334d47ad4052d4bd705f7d5a72becf3bb80))
* **run:** add dataSpecification in pipeline run logging response for… ([#606](https://github.com/instill-ai/pipeline-backend/issues/606)) ([1173def](https://github.com/instill-ai/pipeline-backend/commit/1173def4b7d423828dd2c90ee6202f0bf9e45066))
* **run:** added pipeline run logging upsert points & upload inputs to minio activity ([#596](https://github.com/instill-ai/pipeline-backend/issues/596)) ([787de28](https://github.com/instill-ai/pipeline-backend/commit/787de28f0b2e9236965cf45d27313dc565a127a2))
* **run:** pipeline & component run logging query APIs and component outputs ([#602](https://github.com/instill-ai/pipeline-backend/issues/602)) ([dba547d](https://github.com/instill-ai/pipeline-backend/commit/dba547da1559e830c86757e5025661096d1eca5b))
* **run:** run logging data model & repo func ([#595](https://github.com/instill-ai/pipeline-backend/issues/595)) ([cbc6c48](https://github.com/instill-ai/pipeline-backend/commit/cbc6c48ac183d60868481be13baa8d73400ab2ed))
* support new Instill Format in variable section ([#624](https://github.com/instill-ai/pipeline-backend/issues/624)) ([c7cb2ef](https://github.com/instill-ai/pipeline-backend/commit/c7cb2efee25f30de0490b4b4d542c14e9e5ee1e0))
* **web:** add chromium ([#601](https://github.com/instill-ai/pipeline-backend/issues/601)) ([a0a6c7b](https://github.com/instill-ai/pipeline-backend/commit/a0a6c7b5ba17bc5bbfe8640399fb41fa233d1680))


### Bug Fixes

* return raw recipe for pipeline release ([#604](https://github.com/instill-ai/pipeline-backend/issues/604)) ([07f64e9](https://github.com/instill-ai/pipeline-backend/commit/07f64e9c010857bb84e756fa3c5f0da724ceca96))
* **run:** fix the issue that sometimes recipe and input uploading fail in temporal with no error message ([#607](https://github.com/instill-ai/pipeline-backend/issues/607)) ([2895f1c](https://github.com/instill-ai/pipeline-backend/commit/2895f1ccb34347b2dc6d99f3fbe67695d87486ef))
* the workflow timeout is using wrong value ([#614](https://github.com/instill-ai/pipeline-backend/issues/614)) ([a6118af](https://github.com/instill-ai/pipeline-backend/commit/a6118afeb00fcc8c657ebfcdf72994a248e86f46))

## [0.36.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.35.0-beta...v0.36.0-beta) (2024-08-16)


### Features

* **minio:** added minio connection ([#593](https://github.com/instill-ai/pipeline-backend/issues/593)) ([bf23217](https://github.com/instill-ai/pipeline-backend/commit/bf23217f3c950c9c5ac560d6f0b91ac204537f91))


### Bug Fixes

* fix go-fitz not being able to be built into a binary ([#597](https://github.com/instill-ai/pipeline-backend/issues/597)) ([a0a0d73](https://github.com/instill-ai/pipeline-backend/commit/a0a0d73bc7d9315c332c86581403269235202ae2))

## [0.35.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.34.1-beta...v0.35.0-beta) (2024-08-13)


### Features

* pass component ID to execution ([#577](https://github.com/instill-ai/pipeline-backend/issues/577)) ([2757e7c](https://github.com/instill-ai/pipeline-backend/commit/2757e7c90c9db2ef2cc3d64e7939b1c94476b70f))
* **text:** add dependency for vendors' tokenizer ([#586](https://github.com/instill-ai/pipeline-backend/issues/586)) ([ac40497](https://github.com/instill-ai/pipeline-backend/commit/ac40497ef2cb0afd434610b651727e707b36e4dd))
* use pdf2md in document operator ([#589](https://github.com/instill-ai/pipeline-backend/issues/589)) ([f35f79e](https://github.com/instill-ai/pipeline-backend/commit/f35f79e070d801c9575e340ac1c8c6a163f6852a))

## [0.34.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.34.0-beta...v0.34.1-beta) (2024-07-31)


### Bug Fixes

* fix pipeline profile image ([9641b3c](https://github.com/instill-ai/pipeline-backend/commit/9641b3c83696ff358632c56c20f07affee61df21))


### Miscellaneous Chores

* release v0.34.1-beta ([d89e51b](https://github.com/instill-ai/pipeline-backend/commit/d89e51b8e2557198920756db366c240484a54f7d))

## [0.34.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.33.0-beta...v0.34.0-beta) (2024-07-31)


### Features

* add ffmpeg for audio operator ([#558](https://github.com/instill-ai/pipeline-backend/issues/558)) ([e1536fc](https://github.com/instill-ai/pipeline-backend/commit/e1536fc967c85bceef00f8d7902f2375a3f035a5))
* add ffmpeg-go package for video operator ([#560](https://github.com/instill-ai/pipeline-backend/issues/560)) ([db98471](https://github.com/instill-ai/pipeline-backend/commit/db98471f0182383461d3671f077915ebabd1bcfc))
* add requester information to pipeline metrics ([#556](https://github.com/instill-ai/pipeline-backend/issues/556)) ([7c3cc3c](https://github.com/instill-ai/pipeline-backend/commit/7c3cc3c8cddbc5bd6f347dbce02193dec2e7d41b))
* implement fuzzy search for namespace and pipeline ID ([#563](https://github.com/instill-ai/pipeline-backend/issues/563)) ([9cf8fa5](https://github.com/instill-ai/pipeline-backend/commit/9cf8fa5b2bf87e79f1241b273f10aee0555fb08d))
* implement namespace endpoints ([#561](https://github.com/instill-ai/pipeline-backend/issues/561)) ([5920f69](https://github.com/instill-ai/pipeline-backend/commit/5920f699f6e298f201241e8808f7e4bf03aa9753))
* support run-on-event/schedule pipeline ([#571](https://github.com/instill-ai/pipeline-backend/issues/571)) ([19730cf](https://github.com/instill-ai/pipeline-backend/commit/19730cf30a8d0b222859ca5c982943fca64c291a))
* **text:** add libreoffice to container ([#555](https://github.com/instill-ai/pipeline-backend/issues/555)) ([2410b4b](https://github.com/instill-ai/pipeline-backend/commit/2410b4b3d9305dbb50fc3281e6f5b1bf4e71895c))
* use explicit `user_id` and `organization_id` in mgmt request ([#559](https://github.com/instill-ai/pipeline-backend/issues/559)) ([2b273b4](https://github.com/instill-ai/pipeline-backend/commit/2b273b4247575891555a325f71c0461129bd8a50))


### Bug Fixes

* change the dependency to fix bugs ([#570](https://github.com/instill-ai/pipeline-backend/issues/570)) ([175a566](https://github.com/instill-ai/pipeline-backend/commit/175a566261421a050ea85734000c8c203c9a4fe6))
* fix form-data handler ([#578](https://github.com/instill-ai/pipeline-backend/issues/578)) ([33e0aab](https://github.com/instill-ai/pipeline-backend/commit/33e0aab58c6f367778c37ae91116c10023a95fa4))
* fix pipeline could not have more than one iterator ([#575](https://github.com/instill-ai/pipeline-backend/issues/575)) ([9529d96](https://github.com/instill-ai/pipeline-backend/commit/9529d96b177b8f3e9ae2e44b10c854d56a1a551b))
* fix SQL transpiler bug ([#579](https://github.com/instill-ai/pipeline-backend/issues/579)) ([eefedaa](https://github.com/instill-ai/pipeline-backend/commit/eefedaac830bfe0acf444f1332e2ec41f3c2a6a8))
* send old and new pipeline trigger measurements ([#565](https://github.com/instill-ai/pipeline-backend/issues/565)) ([33cf3c7](https://github.com/instill-ai/pipeline-backend/commit/33cf3c78dcb3c0b3e62f09255959656691d418cc))

## [0.33.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.32.1-beta...v0.33.0-beta) (2024-07-19)


### Bug Fixes

* parse end-user errors on stream trigger endpoint ([#551](https://github.com/instill-ai/pipeline-backend/issues/551)) ([3ae2432](https://github.com/instill-ai/pipeline-backend/commit/3ae2432a7a103517753a47b9ca73bd3f355d305e))


### Miscellaneous Chores

* release v0.33.0-beta ([e9c9275](https://github.com/instill-ai/pipeline-backend/commit/e9c927546161416a87095b01465056da934c2fe4))

## [0.32.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.32.0-beta...v0.32.1-beta) (2024-07-17)


### Bug Fixes

* fix pageToken not working when ordering by certain fields ([#552](https://github.com/instill-ai/pipeline-backend/issues/552)) ([c7821e5](https://github.com/instill-ai/pipeline-backend/commit/c7821e5d9800dcdcd3e4e1dd721cf2f288cae5f6))

## [0.32.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.31.0-beta...v0.32.0-beta) (2024-07-16)


### Features

* add profile_image and several profile fields for pipeline ([#544](https://github.com/instill-ai/pipeline-backend/issues/544)) ([f628efd](https://github.com/instill-ai/pipeline-backend/commit/f628efdba7cd7549bd62edfdc4e083c2488bbadc))
* inject single usage handler creator in components ([#541](https://github.com/instill-ai/pipeline-backend/issues/541)) ([ccfdfbe](https://github.com/instill-ai/pipeline-backend/commit/ccfdfbef27558f726fbcc6c61c88c0a2ef7194d2))
* Intermediate Result Streaming for User Pipelines ([#534](https://github.com/instill-ai/pipeline-backend/issues/534)) ([c8be9a0](https://github.com/instill-ai/pipeline-backend/commit/c8be9a0c6fa14afd20b089405310f58fd7e4aafc))
* prevent users from tagging pipelines with a reserved tag ([#545](https://github.com/instill-ai/pipeline-backend/issues/545)) ([29cdaed](https://github.com/instill-ai/pipeline-backend/commit/29cdaed142ba9df6e0d14e0d973b555b15993650))
* users can create and update tags by updating pipeline ([#497](https://github.com/instill-ai/pipeline-backend/issues/497)) ([54aded8](https://github.com/instill-ai/pipeline-backend/commit/54aded83a510de7ab1ef68733b628b8eca5c0116))


### Bug Fixes

* return end-user error on component execution failure ([#543](https://github.com/instill-ai/pipeline-backend/issues/543)) ([a4afad0](https://github.com/instill-ai/pipeline-backend/commit/a4afad02e2677f39381c83a771448f5951d65ef9))


### Miscellaneous Chores

* release v0.32.0-beta ([6247a0b](https://github.com/instill-ai/pipeline-backend/commit/6247a0baa816a563ddbd4332e4ad2b9e2e039990))

## [0.31.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.30.1-beta...v0.31.0-beta) (2024-07-02)


### Features

* allow user to store recipe in YAML format ([#524](https://github.com/instill-ai/pipeline-backend/issues/524)) ([41bed7f](https://github.com/instill-ai/pipeline-backend/commit/41bed7f59947dcfa88d02dd482e402d8f265c9bf))
* check trigger permissions when impersonating an organization ([#527](https://github.com/instill-ai/pipeline-backend/issues/527)) ([8da41fe](https://github.com/instill-ai/pipeline-backend/commit/8da41fe7cdee15650a1c64201c48daef30ae2e09))
* download preset pipelines from Instill Cloud ([#531](https://github.com/instill-ai/pipeline-backend/issues/531)) ([0ec3026](https://github.com/instill-ai/pipeline-backend/commit/0ec30268afa970a990424c67b43b4c7c609b7ae4))
* refactor pipeline clone endpoints ([#529](https://github.com/instill-ai/pipeline-backend/issues/529)) ([97d78b0](https://github.com/instill-ai/pipeline-backend/commit/97d78b0b20966693b8bae276c0853fefb3f541b5))
* support case-insensitive search for pipelines and components ([#536](https://github.com/instill-ai/pipeline-backend/issues/536)) ([23cb39d](https://github.com/instill-ai/pipeline-backend/commit/23cb39db3822cf1c4c68fe20efeb426a032982d7))


### Bug Fixes

* block user saving secret fields as plaintext in pipeline ([#537](https://github.com/instill-ai/pipeline-backend/issues/537)) ([1e16d7f](https://github.com/instill-ai/pipeline-backend/commit/1e16d7ff51ef2c9168134b648a0da736b85ea0f6))
* prevent a crash in Console caused by an empty map in iterator ([#533](https://github.com/instill-ai/pipeline-backend/issues/533)) ([3a645d4](https://github.com/instill-ai/pipeline-backend/commit/3a645d4c1ff86ca5e5271ea694a57146b49e5ed8))

## [0.30.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.30.0-beta...v0.30.1-beta) (2024-06-21)


### Miscellaneous Chores

* release v0.30.1-beta ([97431ae](https://github.com/instill-ai/pipeline-backend/commit/97431aeac094edb432c7693cc25665e82580a23d))

## [0.30.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.29.2-beta...v0.30.0-beta) (2024-06-19)


### Features

* **endpoints:** use camelCase for `filter` and `orderBy` query strings ([#517](https://github.com/instill-ai/pipeline-backend/issues/517)) ([116467e](https://github.com/instill-ai/pipeline-backend/commit/116467eafbc9a7275664ace112df42b8b9d35357))
* render secret value when using dynamic definition ([#509](https://github.com/instill-ai/pipeline-backend/issues/509)) ([44eaacb](https://github.com/instill-ai/pipeline-backend/commit/44eaacb5eebeaaa0c2f4f8e940f2677c92dce9de))
* store recipe as YAML ([#515](https://github.com/instill-ai/pipeline-backend/issues/515)) ([4690835](https://github.com/instill-ai/pipeline-backend/commit/4690835732033c3c56662bc1372865d8a45951a8))
* use camelCase for HTTP body ([#512](https://github.com/instill-ai/pipeline-backend/issues/512)) ([7cdce38](https://github.com/instill-ai/pipeline-backend/commit/7cdce388cb7818a4e2b0d0a5ffb03fd13c196da7))


### Bug Fixes

* improve pipeline usage error messages ([#513](https://github.com/instill-ai/pipeline-backend/issues/513)) ([ff4b631](https://github.com/instill-ai/pipeline-backend/commit/ff4b63174a438cf590a6e67b9e3697118ca02f29))
* resolve the issue with the component condition not working ([#518](https://github.com/instill-ai/pipeline-backend/issues/518)) ([fa27de1](https://github.com/instill-ai/pipeline-backend/commit/fa27de1dca1444922176d850f8cb9b7e920c71b5))


### Miscellaneous Chores

* release v0.30.0-beta ([bc653a3](https://github.com/instill-ai/pipeline-backend/commit/bc653a39b64493798cf70af3cdfc697caffab69d))

## [0.29.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.29.1-beta...v0.29.2-beta) (2024-06-12)


### Features

* return dynamic component definition in definition endpoints ([#510](https://github.com/instill-ai/pipeline-backend/issues/510)) ([1483148](https://github.com/instill-ai/pipeline-backend/commit/1483148518acab23180f1b3adfb0f80d92b80caf))


### Miscellaneous Chores

* release v0.29.2-beta ([0b7f613](https://github.com/instill-ai/pipeline-backend/commit/0b7f6138d1553ec226b2927fa2e939609a09ba6a))

## [0.29.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.29.0-beta...v0.29.1-beta) (2024-06-07)


### Features

* return the owner in pipeline response when using VIEW_BASIC ([#506](https://github.com/instill-ai/pipeline-backend/issues/506)) ([a4fea42](https://github.com/instill-ai/pipeline-backend/commit/a4fea42fbddcec95e9e1f5a876b844f0c99e1980))


### Miscellaneous Chores

* release v0.29.1-beta ([f89af80](https://github.com/instill-ai/pipeline-backend/commit/f89af80a49494adfadb09c2820e7220f131e6ad5))

## [0.29.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.28.1-beta...v0.29.0-beta) (2024-06-06)


### Features

* implement new recipe format ([#498](https://github.com/instill-ai/pipeline-backend/issues/498)) ([de0c2bc](https://github.com/instill-ai/pipeline-backend/commit/de0c2bc23fd44023f9449a9cc9997f243cdb9daf))
* implement pipeline `number_of_runs` and `last_run_time` ([#501](https://github.com/instill-ai/pipeline-backend/issues/501)) ([0e6cd12](https://github.com/instill-ai/pipeline-backend/commit/0e6cd12236beaf10fd281ddabcf2a3f07371f43c))
* support python code in components ([#492](https://github.com/instill-ai/pipeline-backend/issues/492)) ([5417e5f](https://github.com/instill-ai/pipeline-backend/commit/5417e5f3ed4e5a05a83aa614ccfca3700b9f3ad4))


### Miscellaneous Chores

* release v0.29.0-beta ([e3be475](https://github.com/instill-ai/pipeline-backend/commit/e3be47539578b4e1cf6811f434ecc49e6b8e5273))

## [0.28.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.28.0-beta...v0.28.1-beta) (2024-05-17)


### Bug Fixes

* move usage_check and collect to Temporal activity ([7e07972](https://github.com/instill-ai/pipeline-backend/commit/7e07972172b6d253ac52f8d68cf16d1d090df4f2))
* resolve condition field bug ([#491](https://github.com/instill-ai/pipeline-backend/issues/491)) ([19b8d43](https://github.com/instill-ai/pipeline-backend/commit/19b8d43980296d7869b61f01f99b8aed39c3108e))

## [0.28.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.27.3-beta...v0.28.0-beta) (2024-05-16)


### Features

* add pipelineUsageHandler ([#485](https://github.com/instill-ai/pipeline-backend/issues/485)) ([d53147f](https://github.com/instill-ai/pipeline-backend/commit/d53147f9bdb30c4c7577785ada77f294bbcaa927))
* implement hub-stats endpoint ([#487](https://github.com/instill-ai/pipeline-backend/issues/487)) ([4a68f5e](https://github.com/instill-ai/pipeline-backend/commit/4a68f5e1cef148be5f940d5681dfc334dcfb51b3))
* implement tag system for pipeline ([#481](https://github.com/instill-ai/pipeline-backend/issues/481)) ([7823db3](https://github.com/instill-ai/pipeline-backend/commit/7823db3b5383eca49fe330a496b1d7439f19d3d5))
* remove subscription related logic ([#479](https://github.com/instill-ai/pipeline-backend/issues/479)) ([8c4ef38](https://github.com/instill-ai/pipeline-backend/commit/8c4ef3849deac27a7796e62ae205ea4789cb8e61))
* support sorting pipelines options by id and update_time ([#486](https://github.com/instill-ai/pipeline-backend/issues/486)) ([e7f2847](https://github.com/instill-ai/pipeline-backend/commit/e7f2847222e3dda0390b0dd97b3a1864b95ef708))
* use global secrets in connectors ([#474](https://github.com/instill-ai/pipeline-backend/issues/474)) ([5c0a12a](https://github.com/instill-ai/pipeline-backend/commit/5c0a12ae3bd4458a29c7fcc68da0b4d61f6eaec1))

## [0.27.3-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.27.2-beta...v0.27.3-beta) (2024-05-07)


### Miscellaneous Chores

* release v0.27.3-beta ([0cfb27f](https://github.com/instill-ai/pipeline-backend/commit/0cfb27fe70bf7b41d01e709b579eea44ed2c0e2e))

## [0.27.2-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.27.1-beta...v0.27.2-beta) (2024-05-02)


### Miscellaneous Chores

* release v0.27.2-beta ([c40bfc5](https://github.com/instill-ai/pipeline-backend/commit/c40bfc5232b18a3fa2fb07a7ece1a493ee5470b3))

## [0.27.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.27.0-beta...v0.27.1-beta) (2024-04-26)


### Bug Fixes

* fix trigger pipeline bug when the secret value is nil ([#467](https://github.com/instill-ai/pipeline-backend/issues/467)) ([d8ecbc1](https://github.com/instill-ai/pipeline-backend/commit/d8ecbc1657e66a3e239566de5bacb84cdd877ded))

## [0.27.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.26.0-beta...v0.27.0-beta) (2024-04-25)


### Features

* implement system variables and adopt latest component interface ([#456](https://github.com/instill-ai/pipeline-backend/issues/456)) ([4b3b980](https://github.com/instill-ai/pipeline-backend/commit/4b3b980b23c1e961a4e9ec3ae113464044a12f3e))
* prevent user using plaintext for credential fields ([#458](https://github.com/instill-ai/pipeline-backend/issues/458)) ([a0cf986](https://github.com/instill-ai/pipeline-backend/commit/a0cf98672d9731893f6429e31824a2a7edd236fc))
* revamp recipe and retire connector resource ([#453](https://github.com/instill-ai/pipeline-backend/issues/453)) ([420b8c6](https://github.com/instill-ai/pipeline-backend/commit/420b8c6df5a9fd804668c0af45cfc8a3abda069b))
* support hyphen in reference syntax ([#462](https://github.com/instill-ai/pipeline-backend/issues/462)) ([a6f1b0c](https://github.com/instill-ai/pipeline-backend/commit/a6f1b0c6f9e06ea0a72931bfe7439d9b3ad23520))


### Bug Fixes

* update error message for creating duplicated resource ([#465](https://github.com/instill-ai/pipeline-backend/issues/465)) ([a208c17](https://github.com/instill-ai/pipeline-backend/commit/a208c17091f4735d9d6ce0841b571903f5817e38))

## [0.26.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.25.1-beta...v0.26.0-beta) (2024-04-10)


### Features

* add repository tests against DB ([#451](https://github.com/instill-ai/pipeline-backend/issues/451)) ([05bf562](https://github.com/instill-ai/pipeline-backend/commit/05bf562d81290667205b1f56584cdaf5b6451b60))


### Bug Fixes

* fix `uidAllowList` bug when listing pipeline ([#449](https://github.com/instill-ai/pipeline-backend/issues/449)) ([7fd5a26](https://github.com/instill-ai/pipeline-backend/commit/7fd5a2612208b7f86be097af549c7c512666bc18))
* fix missing `releases` data in pipeline response ([68fc80e](https://github.com/instill-ai/pipeline-backend/commit/68fc80eb08ecd4257eae4275ce10d1ebb46815bf))

## [0.25.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.25.0-beta...v0.25.1-beta) (2024-04-08)


### Bug Fixes

* fix iterator cannot be executed. ([1f3714e](https://github.com/instill-ai/pipeline-backend/commit/1f3714e160e8377c6a56db1ee2d30ef343fa0667))
* fix shared pipeline can not be triggered ([3388594](https://github.com/instill-ai/pipeline-backend/commit/33885945149fca322d235fac0865a2ae6658fa2c))
* improve nil check ([9195791](https://github.com/instill-ai/pipeline-backend/commit/91957914a2b2a9716e5ed9e6e3bad753c307df18))

## [0.25.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.24.1-beta...v0.25.0-beta) (2024-04-01)


### Features

* add configuration for read-replica database ([#431](https://github.com/instill-ai/pipeline-backend/issues/431)) ([125fb6d](https://github.com/instill-ai/pipeline-backend/commit/125fb6dcdd57222fea0f22a8890176012cd990dd))
* add connection to read replica OpenFGA ([#434](https://github.com/instill-ai/pipeline-backend/issues/434)) ([b229b8c](https://github.com/instill-ai/pipeline-backend/commit/b229b8cd648869fafbb4f9d0df87faae78e9159a))
* pin the user to read from the primary database for a certain time frame after mutating the data ([#433](https://github.com/instill-ai/pipeline-backend/issues/433)) ([30e1de2](https://github.com/instill-ai/pipeline-backend/commit/30e1de289d14f947fde4d3d062ae98b869d5a2c3))


### Bug Fixes

* fix multi-region connection problem for Instill Model connector ([#439](https://github.com/instill-ai/pipeline-backend/issues/439)) ([a02add6](https://github.com/instill-ai/pipeline-backend/commit/a02add615cd3a6a40895e59cf8a45ef545db38d5))

## [0.24.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.24.0-beta...v0.24.1-beta) (2024-03-20)


### Bug Fixes

* duplicated values in component list ([#426](https://github.com/instill-ai/pipeline-backend/issues/426)) ([2e910e9](https://github.com/instill-ai/pipeline-backend/commit/2e910e925175617965cf76054cb88d0e251467ac))
* fix GeneratePipelineDataSpec bug when task is empty ([181df09](https://github.com/instill-ai/pipeline-backend/commit/181df09c1968ce9db9808faf06a4c76d7a8885ee))

## [0.24.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.23.0-beta...v0.24.0-beta) (2024-03-13)


### Features

* add migration script for new recipe format ([#415](https://github.com/instill-ai/pipeline-backend/issues/415)) ([af8a512](https://github.com/instill-ai/pipeline-backend/commit/af8a512abb1a30b1f8527728f67b6f3e6759f364))
* Introduce component definition list filtering ([#410](https://github.com/instill-ai/pipeline-backend/issues/410)) ([08cf677](https://github.com/instill-ai/pipeline-backend/commit/08cf677f5b31be01cab12644a168fa049f7cb4c5))
* revamp recipe format and implement iterator ([#405](https://github.com/instill-ai/pipeline-backend/issues/405)) ([1a676ff](https://github.com/instill-ai/pipeline-backend/commit/1a676fff87f3061b35f87606cea5812ce303875f))
* simplify openapi_specifications to data_specifications ([#411](https://github.com/instill-ai/pipeline-backend/issues/411)) ([deaef9b](https://github.com/instill-ai/pipeline-backend/commit/deaef9b34fbc67b24b6d4bf1e11231f8c36a9ac0))
* sort component list by score ([#412](https://github.com/instill-ai/pipeline-backend/issues/412)) ([2849555](https://github.com/instill-ai/pipeline-backend/commit/2849555ea1c0c44d8172c228bb17b9972526966d))
* support unimplemented release stages in component definitions ([#414](https://github.com/instill-ai/pipeline-backend/issues/414)) ([c235592](https://github.com/instill-ai/pipeline-backend/commit/c2355921dc933a6de0a37419442d2b4e4086f645))


### Bug Fixes

* allow incomplete configuration in includeIteratorComponentDetail(). ([#413](https://github.com/instill-ai/pipeline-backend/issues/413)) ([2999599](https://github.com/instill-ai/pipeline-backend/commit/29995992bd9f2685e314122a1896bdb0c03e1a3e))
* fix condition field bugs ([#417](https://github.com/instill-ai/pipeline-backend/issues/417)) ([ce720d5](https://github.com/instill-ai/pipeline-backend/commit/ce720d5d75e543dff1dbf0351cfc3ba811921d49))
* fix missing error return in pipeline trigger ([a743ab1](https://github.com/instill-ai/pipeline-backend/commit/a743ab191f4070acfabf52335a3f4b9c6c862149))
* fix missing param for Instill Model connector ([6d372bb](https://github.com/instill-ai/pipeline-backend/commit/6d372bb6ddfb81f2f2c26beaf8a735c706ff040f))


### Miscellaneous Chores

* release v0.24.0-beta ([d4e3f2b](https://github.com/instill-ai/pipeline-backend/commit/d4e3f2b4e915f52f2df012cb862bba1358332bd6))

## [0.23.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.22.0-beta...v0.23.0-beta) (2024-03-01)


### Features

* add component definition list endpoint ([#396](https://github.com/instill-ai/pipeline-backend/issues/396)) ([b8728c1](https://github.com/instill-ai/pipeline-backend/commit/b8728c16483fb68cdaed65dff191d2500ec06e4e))
* rename blockchain connector type to application ([#397](https://github.com/instill-ai/pipeline-backend/issues/397)) ([80aa6a5](https://github.com/instill-ai/pipeline-backend/commit/80aa6a5c725fa18c34a84db4ec426f60461fd702))
* support reference with `foo["bar"]` syntax ([#394](https://github.com/instill-ai/pipeline-backend/issues/394)) ([ed82215](https://github.com/instill-ai/pipeline-backend/commit/ed8221504572ba67e39367b651085df438306c7d))
* use `mgmtpb.Owner` to embed the owner information in response ([#392](https://github.com/instill-ai/pipeline-backend/issues/392)) ([d071461](https://github.com/instill-ai/pipeline-backend/commit/d0714617e34aa7779947b153297a7bffea7bd08f))


### Bug Fixes

* fix component ID with a hyphen cannot be referenced ([#401](https://github.com/instill-ai/pipeline-backend/issues/401)) ([1958168](https://github.com/instill-ai/pipeline-backend/commit/1958168681ef8a625106a18a6799fa6b75acf5f3))


### Miscellaneous Chores

* release v0.23.0-beta ([e3ab340](https://github.com/instill-ai/pipeline-backend/commit/e3ab3400299e6352487d8f34f2f3a928b46e09ec))

## [0.22.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.21.1-beta...v0.22.0-beta) (2024-02-16)


### Features

* add end-user errors on CreateExecution error ([#380](https://github.com/instill-ai/pipeline-backend/issues/380)) ([093c11c](https://github.com/instill-ai/pipeline-backend/commit/093c11c13188632229a5e42b55445ef5e3680096))
* allow the string data to reference all data types that can be stringified ([#381](https://github.com/instill-ai/pipeline-backend/issues/381)) ([9342d23](https://github.com/instill-ai/pipeline-backend/commit/9342d233ccaa71d34107f0e7635cc081c13049bc))
* remove `${[ ]}` operator in component reference syntax ([#383](https://github.com/instill-ai/pipeline-backend/issues/383)) ([c121da8](https://github.com/instill-ai/pipeline-backend/commit/c121da86867ff1f3f938a97f434862c9f811b0a8))


### Bug Fixes

* fix reference bug ([#388](https://github.com/instill-ai/pipeline-backend/issues/388)) ([968c0ec](https://github.com/instill-ai/pipeline-backend/commit/968c0ec9aa513eabb9a3e301f69477cf8afd9368))
* **worker:** fix temporal cloud namespace init ([#387](https://github.com/instill-ai/pipeline-backend/issues/387)) ([e42cf13](https://github.com/instill-ai/pipeline-backend/commit/e42cf134d2d6fdbd817126292f7621de381368a0))

## [0.21.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.21.0-beta...v0.21.1-beta) (2024-02-06)


### Miscellaneous Chores

* release v0.21.1-beta ([e38033d](https://github.com/instill-ai/pipeline-backend/commit/e38033dbaf73843941ae74881086c36867b68c5b))

## [0.21.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.20.0-beta...v0.21.0-beta) (2024-01-30)


### Features

* add `CheckName` endpoint ([#375](https://github.com/instill-ai/pipeline-backend/issues/375)) ([7e248ed](https://github.com/instill-ai/pipeline-backend/commit/7e248ed605b693d39f3e7bd881f09088b3c6bbf9))
* add `CloneUserPipeline` and `CloneOrganizationPipeline` endpoints ([#374](https://github.com/instill-ai/pipeline-backend/issues/374)) ([abf8514](https://github.com/instill-ai/pipeline-backend/commit/abf85141ec5b08c2dc32a8afe65f89e05a3f4168))
* add end-user errors on CreateExecution error ([#369](https://github.com/instill-ai/pipeline-backend/issues/369)) ([b72ac08](https://github.com/instill-ai/pipeline-backend/commit/b72ac08f1a5b6fa3d6ee8ea0808efd5350bdac56))
* execute component in parallel ([#366](https://github.com/instill-ai/pipeline-backend/issues/366)) ([1a18d62](https://github.com/instill-ai/pipeline-backend/commit/1a18d62ca3cad984c94d90130066dbd99c2eadc5))
* support `visibility` param in list namespace pipelines endpoints ([#372](https://github.com/instill-ai/pipeline-backend/issues/372)) ([e0b2c48](https://github.com/instill-ai/pipeline-backend/commit/e0b2c481f341307ce9a53c7b908a03abefa78364))


### Bug Fixes

* fix can not restore pipeline recipe from releases ([#376](https://github.com/instill-ai/pipeline-backend/issues/376)) ([5163aec](https://github.com/instill-ai/pipeline-backend/commit/5163aecafdbbd8bdc888ce74adb0ccf0dcef066a))

## [0.20.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.19.0-beta...v0.20.0-beta) (2024-01-15)


### Features

* inject mgmt_backend address into connector configuration ([ca2586c](https://github.com/instill-ai/pipeline-backend/commit/ca2586c15a736e39e5528757e63c1f5b4c91188b))
* **recipe:** use `${}` as reference syntax ([#358](https://github.com/instill-ai/pipeline-backend/issues/358)) ([f86fae1](https://github.com/instill-ai/pipeline-backend/commit/f86fae150644464c9801e72f897704e48340ab05))
* remove controller-vdp ([#354](https://github.com/instill-ai/pipeline-backend/issues/354)) ([afc3d21](https://github.com/instill-ai/pipeline-backend/commit/afc3d2136a14e5a6474814ab688bb3d1cdb4d4bb))
* support `q` filter for fuzzy search on `id` ([#360](https://github.com/instill-ai/pipeline-backend/issues/360)) ([ec3b439](https://github.com/instill-ai/pipeline-backend/commit/ec3b439253afdebf0f2d8f6aea9e28fb40d8567f))
* support dynamic connector and operator definition ([#359](https://github.com/instill-ai/pipeline-backend/issues/359)) ([1485877](https://github.com/instill-ai/pipeline-backend/commit/1485877a84015023aae84d3fa7dc51ec9160dc8a))
* support filter pipelines with visibility ([#357](https://github.com/instill-ai/pipeline-backend/issues/357)) ([499b112](https://github.com/instill-ai/pipeline-backend/commit/499b11227509886f66ce310659ca58ef2faf901b))


### Bug Fixes

* fix condition field not working when component name has `-` ([#362](https://github.com/instill-ai/pipeline-backend/issues/362)) ([92682ce](https://github.com/instill-ai/pipeline-backend/commit/92682ce81735a9293227896fd2b6ef8e251ea2e7))
* fix includeDetailInRecipe() ([7d7749b](https://github.com/instill-ai/pipeline-backend/commit/7d7749b63a301752751bd8fbfd35a78a3b03c170))
* fix wrong global reference for Numbers connector ([#363](https://github.com/instill-ai/pipeline-backend/issues/363)) ([5c5eda8](https://github.com/instill-ai/pipeline-backend/commit/5c5eda8438dfe60e3670397af18e004902dd763a))


### Miscellaneous Chores

* release v0.20.0-beta ([150c83b](https://github.com/instill-ai/pipeline-backend/commit/150c83bfe28c068cdd5093dcb836e91dd5da5c46))

## [0.19.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.18.1-beta...v0.19.0-beta) (2024-01-02)


### Bug Fixes

* fix the wrong Temporal workflow retry setting ([#351](https://github.com/instill-ai/pipeline-backend/issues/351)) ([c3b71fc](https://github.com/instill-ai/pipeline-backend/commit/c3b71fc87da43802943e2341d54b709d60271c02))
* force the connector and pipeline belong to same namespace ([#353](https://github.com/instill-ai/pipeline-backend/issues/353)) ([7bbed56](https://github.com/instill-ai/pipeline-backend/commit/7bbed5667b0d7c3494ad2fc510e18261e520a3c6))
* remove unnecessary mgmt-backend request ([#349](https://github.com/instill-ai/pipeline-backend/issues/349)) ([9bfe43c](https://github.com/instill-ai/pipeline-backend/commit/9bfe43ca1fa96d6683db9404275e87718da99c30))


### Miscellaneous Chores

* release v0.19.0-beta ([6ffa11c](https://github.com/instill-ai/pipeline-backend/commit/6ffa11ca1cdc3f89f7ac81aa3d4eb858f2b4624d))

## [0.18.1-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.18.0-beta...v0.18.1-beta) (2023-12-25)


### Features

* Improve error messages in connector execution ([#311](https://github.com/instill-ai/pipeline-backend/issues/311)) ([6e282eb](https://github.com/instill-ai/pipeline-backend/commit/6e282eba2dce24d20598a3f2de4e275869532eae))


### Bug Fixes

* calculate the trigger_count with batch_size ([#338](https://github.com/instill-ai/pipeline-backend/issues/338)) ([423e6c9](https://github.com/instill-ai/pipeline-backend/commit/423e6c97093470288ff1bce72540609d538de1d9))
* fix pipeline can not generate correct output schema ([#342](https://github.com/instill-ai/pipeline-backend/issues/342)) ([502f1c4](https://github.com/instill-ai/pipeline-backend/commit/502f1c415a3939e624e3a43af968e1370361b870))


### Miscellaneous Chores

* release v0.18.1-beta ([6deb019](https://github.com/instill-ai/pipeline-backend/commit/6deb0195f59df0ab7d27d4d664e9917e4b39e762))

## [0.18.0-beta](https://github.com/instill-ai/pipeline-backend/compare/v0.17.0-alpha...v0.18.0-beta) (2023-12-16)


### Features

* **fga:** adopt OpenFGA and implement pipeline and connector FGA ([#310](https://github.com/instill-ai/pipeline-backend/issues/310)) ([416dc75](https://github.com/instill-ai/pipeline-backend/commit/416dc7514e5d0f3e2a67e02e7205f4e3d31bf728))
* **organization:** add organization API endpoints ([#308](https://github.com/instill-ai/pipeline-backend/issues/308)) ([0deeca7](https://github.com/instill-ai/pipeline-backend/commit/0deeca7d68c7fb64ecdf146d007f6fd3b5907af3))
* **pipeline:** implement component status ([#317](https://github.com/instill-ai/pipeline-backend/issues/317)) ([42d8277](https://github.com/instill-ai/pipeline-backend/commit/42d8277740771b8ee974d145d84dc0cb0eb9ce4d))
* **service:** implement conditional component ([#318](https://github.com/instill-ai/pipeline-backend/issues/318)) ([15312d3](https://github.com/instill-ai/pipeline-backend/commit/15312d3177ab1f7b1f5749ee7eef79c13b2c2791))
* **service:** implement trigger quota-limit ([#322](https://github.com/instill-ai/pipeline-backend/issues/322)) ([043ee04](https://github.com/instill-ai/pipeline-backend/commit/043ee04fec6af24c95aa7fe9c57228eb193ed580))
* **service:** implement trigger rate-limit ([#321](https://github.com/instill-ai/pipeline-backend/issues/321)) ([91a9706](https://github.com/instill-ai/pipeline-backend/commit/91a970632f562a40440184abfe08965403a409ef))
* **service:** user can not trigger non-latest pipeline release under freemium plan ([#324](https://github.com/instill-ai/pipeline-backend/issues/324)) ([f2e82c9](https://github.com/instill-ai/pipeline-backend/commit/f2e82c9065322126de69c4221a70a9c9020ed191))


### Bug Fixes

* **service:** fix permission field bug ([1304969](https://github.com/instill-ai/pipeline-backend/commit/130496916356b8cb31df95d8ec3b0826ca475356))


### Miscellaneous Chores

* release v0.18.0-beta ([23028b4](https://github.com/instill-ai/pipeline-backend/commit/23028b429470fb30d8ec15cb13f35a34d35f45d0))

## [0.17.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.16.2-alpha...v0.17.0-alpha) (2023-11-28)


### Miscellaneous Chores

* release v0.17.0-alpha ([a0d546c](https://github.com/instill-ai/pipeline-backend/commit/a0d546c8ff7d91b90f3fedb004ec3545ab6a0396))

## [0.16.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.16.1-alpha...v0.16.2-alpha) (2023-11-11)


### Bug Fixes

* **pipeline:** fix trigger error when start operator has field `input` ([#287](https://github.com/instill-ai/pipeline-backend/issues/287)) ([9f7ae76](https://github.com/instill-ai/pipeline-backend/commit/9f7ae76cf35a5fb47e86eb8d66408a6d85e6b6a5))

## [0.16.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.16.0-alpha...v0.16.1-alpha) (2023-10-27)


### Bug Fixes

* **migration:** fix db connection bug ([#279](https://github.com/instill-ai/pipeline-backend/issues/279)) ([028a093](https://github.com/instill-ai/pipeline-backend/commit/028a093d7e8eeeb12755f144c9a9b0b7bbf7d01c))
* **service:** fix basic view should return null recipe ([#281](https://github.com/instill-ai/pipeline-backend/issues/281)) ([5d0367c](https://github.com/instill-ai/pipeline-backend/commit/5d0367c441b77b176c0fe0c509b47d98ba123bc1))


### Miscellaneous Chores

* **release:** release v0.16.1-alpha ([8552d59](https://github.com/instill-ai/pipeline-backend/commit/8552d59a7c10ee711b2153dadd504acd5fea7ec8))

## [0.16.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.15.1-alpha...v0.16.0-alpha) (2023-10-13)


### Features

* **repository:** add metadata field for pipeline ([#272](https://github.com/instill-ai/pipeline-backend/issues/272)) ([550e606](https://github.com/instill-ai/pipeline-backend/commit/550e606f20526141528adfe44a66abbef44655bf))


### Bug Fixes

* **component:** fix task bug ([#275](https://github.com/instill-ai/pipeline-backend/issues/275)) ([48427d8](https://github.com/instill-ai/pipeline-backend/commit/48427d882c45a15a5ed7e5f6e2da188775e82dfb))


### Miscellaneous Chores

* **release:** release v0.16.0-alpha ([ee1fc5e](https://github.com/instill-ai/pipeline-backend/commit/ee1fc5e8b6984988c2c5ef596270ed8af769eca1))

## [0.15.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.15.0-alpha...v0.15.1-alpha) (2023-09-30)


### Features

* **permission:** support permission setting, sharing public, sharing by code(link) ([#256](https://github.com/instill-ai/pipeline-backend/issues/256)) ([a9e42e2](https://github.com/instill-ai/pipeline-backend/commit/a9e42e29055f672f668f45957ce8b86360173a89))


### Bug Fixes

* **handler:** fix PATCH pipeline mask bug ([#263](https://github.com/instill-ai/pipeline-backend/issues/263)) ([60c41d5](https://github.com/instill-ai/pipeline-backend/commit/60c41d5b3a9d7290551320c8ca7aa20de0102661))
* **proto,handler:** use `int32` in proto pkg to prevent the `total_size` converted to `string` ([#248](https://github.com/instill-ai/pipeline-backend/issues/248)) ([7ca469b](https://github.com/instill-ai/pipeline-backend/commit/7ca469b67fbf35d0edecd0215e2c0f963fdabdf7))
* **service:** delete pipeline_release when pipeline is deleted ([#254](https://github.com/instill-ai/pipeline-backend/issues/254)) ([066682d](https://github.com/instill-ai/pipeline-backend/commit/066682dfa046ab238c755dcce41c9749442188d0))
* **service:** fix pipeline_release recipe conversion bug ([#251](https://github.com/instill-ai/pipeline-backend/issues/251)) ([5558d7c](https://github.com/instill-ai/pipeline-backend/commit/5558d7c877a674cf62906c5e0b5f86c9fab348b8))
* **service:** fix pipeline_release state bug ([#264](https://github.com/instill-ai/pipeline-backend/issues/264)) ([d642f8d](https://github.com/instill-ai/pipeline-backend/commit/d642f8d7c65219a6f06c01dad357334ad7ee1dac))
* **service:** fix the component type is unspecified when `resource_name` in not set ([#258](https://github.com/instill-ai/pipeline-backend/issues/258)) ([1410706](https://github.com/instill-ai/pipeline-backend/commit/141070603f54c7b2cda03b8ac10ea29a8381b730))


### Miscellaneous Chores

* **release:** release v0.15.1-alpha ([de2fb57](https://github.com/instill-ai/pipeline-backend/commit/de2fb5778bb7e1e35f93e49f2e1677935dd73e2f))

## [0.15.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.14.1-alpha...v0.15.0-alpha) (2023-09-13)


### Miscellaneous Chores

* **release:** release v0.15.0-alpha ([350ddff](https://github.com/instill-ai/pipeline-backend/commit/350ddff095e788ac875bd9299dde6699baa2c5a9))

## [0.14.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.14.0-alpha...v0.14.1-alpha) (2023-08-03)


### Miscellaneous Chores

* **release:** release v0.14.1-alpha ([5e73969](https://github.com/instill-ai/pipeline-backend/commit/5e739699f6436fe14794634e138523b9de845963))

## [0.14.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.13.0-alpha...v0.14.0-alpha) (2023-07-20)


### Miscellaneous Chores

* **release:** release v0.14.0-alpha ([3d83761](https://github.com/instill-ai/pipeline-backend/commit/3d83761caaaa296182399e7c937d8b2899b8cd9a))

## [0.13.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.12.2-alpha...v0.13.0-alpha) (2023-07-09)


### Miscellaneous Chores

* **release:** release v0.13.0-alpha ([d3abf57](https://github.com/instill-ai/pipeline-backend/commit/d3abf575d77495b856bcc10f6b7d12069a9a096f))

## [0.12.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.12.1-alpha...v0.12.2-alpha) (2023-06-21)


### Miscellaneous Chores

* **release:** release 0.12.2-alpha ([9f13992](https://github.com/instill-ai/pipeline-backend/commit/9f13992f24ddc5c30215c0a6457da57ccb98be13))

## [0.12.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.12.0-alpha...v0.12.1-alpha) (2023-06-11)


### Miscellaneous Chores

* **release:** release v0.12.1-alpha ([c2a9ff4](https://github.com/instill-ai/pipeline-backend/commit/c2a9ff424e6f9fe10ef9edbd26955a6b2f1d1b70))

## [0.12.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.6-alpha...v0.12.0-alpha) (2023-06-02)


### Miscellaneous Chores

* **release:** release v0.12.0-alpha ([6a20a45](https://github.com/instill-ai/pipeline-backend/commit/6a20a455e2483d6c4de3b6f8d8f426133feec39f))

## [0.11.6-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.5-alpha...v0.11.6-alpha) (2023-05-11)


### Miscellaneous Chores

* **release:** release v0.11.6-alpha ([f257848](https://github.com/instill-ai/pipeline-backend/commit/f2578480e6340007c7850b646bf1b9e682455ae8))

## [0.11.5-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.4-alpha...v0.11.5-alpha) (2023-05-06)


### Miscellaneous Chores

* **release:** release v0.11.5-alpha ([292db7d](https://github.com/instill-ai/pipeline-backend/commit/292db7dd32dcf2be64fa1912818c3a8cc8cfe475))

## [0.11.4-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.3-alpha...v0.11.4-alpha) (2023-05-05)


### Bug Fixes

* **controller:** fix rename id and execution order ([#148](https://github.com/instill-ai/pipeline-backend/issues/148)) ([ae29a07](https://github.com/instill-ai/pipeline-backend/commit/ae29a07a56644b47ab895a62f612b3e3793c68c1))

## [0.11.3-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.2-alpha...v0.11.3-alpha) (2023-05-05)


### Miscellaneous Chores

* **release:** release v0.11.3-alpha ([a018ce3](https://github.com/instill-ai/pipeline-backend/commit/a018ce3588665ba98b5c2b7ec28e377ea36a433f))

## [0.11.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.1-alpha...v0.11.2-alpha) (2023-04-26)


### Miscellaneous Chores

* **release:** release v0.11.2-alpha ([7ec125f](https://github.com/instill-ai/pipeline-backend/commit/7ec125f2880cba1c3ac386cc68c4072db8c2c5ea))

## [0.11.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.11.0-alpha...v0.11.1-alpha) (2023-04-15)


### Miscellaneous Chores

* **release:** release v0.11.1-alpha ([8c45e85](https://github.com/instill-ai/pipeline-backend/commit/8c45e851d52329088da8f8971e6ed22d1ec82126))

## [0.11.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.10.0-alpha...v0.11.0-alpha) (2023-04-09)


### Features

* **controller:** add pipeline state monitoring with controller ([#134](https://github.com/instill-ai/pipeline-backend/issues/134)) ([e77a8d8](https://github.com/instill-ai/pipeline-backend/commit/e77a8d8d3b7f2632d87491a561c3000331fcf892))

## [0.10.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.8-alpha...v0.10.0-alpha) (2023-03-26)


### Features

* adopt private/public apis for pipeline ([#111](https://github.com/instill-ai/pipeline-backend/issues/111)) ([14bc109](https://github.com/instill-ai/pipeline-backend/commit/14bc1094bd059b67031dd1844e39beb1eeaa4af3))


### Bug Fixes

* support multipart trigger for new tasks ([#109](https://github.com/instill-ai/pipeline-backend/issues/109)) ([0e7e9fa](https://github.com/instill-ai/pipeline-backend/commit/0e7e9fab79aaefb104fabbd2b59e75dfa5f3d2ed))

## [0.9.8-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.7-alpha...v0.9.8-alpha) (2023-02-23)


### Miscellaneous Chores

* release v0.9.8-alpha ([9051972](https://github.com/instill-ai/pipeline-backend/commit/9051972f3e3150a795dafd9e0336b6bac11dbe13))

## [0.9.7-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.6-alpha...v0.9.7-alpha) (2023-02-13)


### Miscellaneous Chores

* release v0.9.7-alpha ([cdb8b25](https://github.com/instill-ai/pipeline-backend/commit/cdb8b25738371c0643c100c0523b6f6789f7a018))

## [0.9.6-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.5-alpha...v0.9.6-alpha) (2023-02-10)


### Bug Fixes

* fix usage client nil issue when mgmt-backend not ready ([d7c47fd](https://github.com/instill-ai/pipeline-backend/commit/d7c47fdd77e25304e09d36c45fbf763b59483cdf))
* replace fatal logs with error logs ([#102](https://github.com/instill-ai/pipeline-backend/issues/102)) ([a410b29](https://github.com/instill-ai/pipeline-backend/commit/a410b29ab8c8fe15bae615e0a034cf7028ded34f))

## [0.9.5-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.4-alpha...v0.9.5-alpha) (2023-01-20)


### Bug Fixes

* fix multipart file already closed issue ([c5b6050](https://github.com/instill-ai/pipeline-backend/commit/c5b6050721054b2969e7bb0368cd9adc2b1e82e4))

## [0.9.4-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.3-alpha...v0.9.4-alpha) (2023-01-14)


### Miscellaneous Chores

* release v0.9.4-alpha ([e636cef](https://github.com/instill-ai/pipeline-backend/commit/e636cef571534ebb03ed9ffd7b3f8abe6434c540))

## [0.9.3-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.2-alpha...v0.9.3-alpha) (2022-12-24)


### Miscellaneous Chores

* release v0.9.3-alpha ([724dec3](https://github.com/instill-ai/pipeline-backend/commit/724dec3e74658d92b04c36c3e3e04be692e4583f))

## [0.9.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.1-alpha...v0.9.2-alpha) (2022-11-30)


### Miscellaneous Chores

* release 0.9.2-alpha ([4465142](https://github.com/instill-ai/pipeline-backend/commit/4465142ba8bfe00057d01a7a58974db19b12394d))

## [0.9.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.9.0-alpha...v0.9.1-alpha) (2022-10-25)


### Bug Fixes

* fix pipeline trigger model hanging ([#80](https://github.com/instill-ai/pipeline-backend/issues/80)) ([7ba58e5](https://github.com/instill-ai/pipeline-backend/commit/7ba58e510826b202eec4f1aad39c2f120f8a06b0))

## [0.9.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.8.0-alpha...v0.9.0-alpha) (2022-10-19)


### Miscellaneous Chores

* release v0.9.0-alpha ([991cee6](https://github.com/instill-ai/pipeline-backend/commit/991cee657e8f77b14ce8306555d13f829cef0c7d))

## [0.8.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.7.2-alpha...v0.8.0-alpha) (2022-09-14)


### Features

* add data mapping ([0db4bfe](https://github.com/instill-ai/pipeline-backend/commit/0db4bfe169ee8acb3a1da471d2c807b2b3cf78fc))


### Bug Fixes

* fix async trigger block issue ([59f0fb8](https://github.com/instill-ai/pipeline-backend/commit/59f0fb8f8102804ff432c565979e1cf337631bb8)), closes [#67](https://github.com/instill-ai/pipeline-backend/issues/67)
* fix multipart trigger data_mapping_indices empty ([d3160b4](https://github.com/instill-ai/pipeline-backend/commit/d3160b492a26c7a991b13cfcf3421f7af595eb8b))

## [0.7.2-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.7.1-alpha...v0.7.2-alpha) (2022-08-21)


### Miscellaneous Chores

* release 0.7.2-alpha ([575a7b1](https://github.com/instill-ai/pipeline-backend/commit/575a7b18a0150311dd0d6eb00d216e39965a948a))

## [0.7.1-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.7.0-alpha...v0.7.1-alpha) (2022-08-17)


### Bug Fixes

* fix multipart trigger resp fmt inconsist ([0663542](https://github.com/instill-ai/pipeline-backend/commit/06635427edf5a4d18c73dcef53f42daa2324be1b))

## [0.7.0-alpha](https://github.com/instill-ai/pipeline-backend/compare/v0.6.5-alpha...v0.7.0-alpha) (2022-07-29)


### Features

* add data association with dst conn ([9233429](https://github.com/instill-ai/pipeline-backend/commit/9233429a1a36eb8d2d864baa454c7b01c997f4f4))

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
