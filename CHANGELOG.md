# Changelog

## [0.0.11](https://github.com/lakekeeper/go-lakekeeper/compare/v0.0.23...v0.0.11) (2026-04-29)


### ⚠ BREAKING CHANGES

* add explicit context argument to all API methods ([#92](https://github.com/lakekeeper/go-lakekeeper/issues/92))
* rename project Default method to GetDefault ([#21](https://github.com/lakekeeper/go-lakekeeper/issues/21))
* create management module for related apis ([#20](https://github.com/lakekeeper/go-lakekeeper/issues/20))
* **storage:** not sending errors back on storage creds/profile options func ([#16](https://github.com/lakekeeper/go-lakekeeper/issues/16))
* init client structure ([#5](https://github.com/lakekeeper/go-lakekeeper/issues/5))

### Features

* add control on bootstrap user role ([#82](https://github.com/lakekeeper/go-lakekeeper/issues/82)) ([76982f3](https://github.com/lakekeeper/go-lakekeeper/commit/76982f3a39223617c5d99e537c90c2581399352c))
* Add docs and some migrations to the lakekeeper/go-lakekeeper repo ([#1](https://github.com/lakekeeper/go-lakekeeper/issues/1)) ([4263450](https://github.com/lakekeeper/go-lakekeeper/commit/42634504f3f74a990eed00de78bdd5d9063b9999))
* add explicit context argument to all API methods ([#92](https://github.com/lakekeeper/go-lakekeeper/issues/92)) ([7eb0818](https://github.com/lakekeeper/go-lakekeeper/commit/7eb0818a1b6cfe90a766be3ad842ff8b1d5827a1))
* add integration with go-iceberg for catalog endpoints ([#89](https://github.com/lakekeeper/go-lakekeeper/issues/89)) ([553afcb](https://github.com/lakekeeper/go-lakekeeper/commit/553afcbfc4b30966ee0f4a5b1dd3be53e96d0ef2))
* add statistics and protection warehouse/project actions ([#162](https://github.com/lakekeeper/go-lakekeeper/issues/162)) ([ef5feed](https://github.com/lakekeeper/go-lakekeeper/commit/ef5feed1aec6b75d3e640920475ca21f65b40246))
* **auth:** add k8s service account token authentication ([#27](https://github.com/lakekeeper/go-lakekeeper/issues/27)) ([d2813e2](https://github.com/lakekeeper/go-lakekeeper/commit/d2813e27fba9ddc5341a84f7e9e4065dbf0eb9eb))
* **cli:** add role assignments add command ([#118](https://github.com/lakekeeper/go-lakekeeper/issues/118)) ([ad35389](https://github.com/lakekeeper/go-lakekeeper/commit/ad353898461062c947bf30d534fd260169390959))
* **cli:** add server permissions-related commands ([#126](https://github.com/lakekeeper/go-lakekeeper/issues/126)) ([dc5adc0](https://github.com/lakekeeper/go-lakekeeper/commit/dc5adc03cd374da3571df655175119ce965545d8))
* **cli:** introduction of tab writer ([#124](https://github.com/lakekeeper/go-lakekeeper/issues/124)) ([c1eb5ac](https://github.com/lakekeeper/go-lakekeeper/commit/c1eb5ac66fd4c9411b59a478c577834d61346322))
* **cli:** introduction of the command line interface ([#103](https://github.com/lakekeeper/go-lakekeeper/issues/103)) ([7133351](https://github.com/lakekeeper/go-lakekeeper/commit/7133351991a341a31618d9c5ada998f8a2e410a1))
* **cli:** rename project asssignments update command to add ([#119](https://github.com/lakekeeper/go-lakekeeper/issues/119)) ([91c8d22](https://github.com/lakekeeper/go-lakekeeper/commit/91c8d22f11e208281503f9b339e66c329af03566))
* **cli:** warehouse commands add/delete/list ([#121](https://github.com/lakekeeper/go-lakekeeper/issues/121)) ([73c5879](https://github.com/lakekeeper/go-lakekeeper/commit/73c5879d57c5ae1e265716ef32ab1ef8215d968c))
* **core:** add Ptr helper method ([#13](https://github.com/lakekeeper/go-lakekeeper/issues/13)) ([e6a04d5](https://github.com/lakekeeper/go-lakekeeper/commit/e6a04d54aa4f99a2968500a78b312f05a6632ee9))
* create management module for related apis ([#20](https://github.com/lakekeeper/go-lakekeeper/issues/20)) ([d1b512e](https://github.com/lakekeeper/go-lakekeeper/commit/d1b512e72b2ccc147f9fb39d6ba12852c5c44745))
* init client structure ([#5](https://github.com/lakekeeper/go-lakekeeper/issues/5)) ([e76b0db](https://github.com/lakekeeper/go-lakekeeper/commit/e76b0db6169eb033aeff04e4804dcd75552041b7))
* Migration ([#2](https://github.com/lakekeeper/go-lakekeeper/issues/2)) ([a8e619b](https://github.com/lakekeeper/go-lakekeeper/commit/a8e619bedabe380ab31b3b61b8a4810d88c6b26f))
* **permission:** add missing GetAccess on role ([#86](https://github.com/lakekeeper/go-lakekeeper/issues/86)) ([516c9f1](https://github.com/lakekeeper/go-lakekeeper/commit/516c9f17d99cc7e56a3ee7f7c63ed761975dd5d1))
* **permission:** add project interface support ([#75](https://github.com/lakekeeper/go-lakekeeper/issues/75)) ([3b11c0f](https://github.com/lakekeeper/go-lakekeeper/commit/3b11c0faf3a43a069cf270e4247bb4aea056ea5a))
* **permission:** add role interfaces ([#78](https://github.com/lakekeeper/go-lakekeeper/issues/78)) ([df69f56](https://github.com/lakekeeper/go-lakekeeper/commit/df69f565219390524df5e91bd30f5a81097c6cae))
* **permission:** add warehouse interfaces ([#85](https://github.com/lakekeeper/go-lakekeeper/issues/85)) ([a41b874](https://github.com/lakekeeper/go-lakekeeper/commit/a41b874ced65d263220a6fb5d3960fc941224f34))
* **permission:** implement server permissions interfaces ([#52](https://github.com/lakekeeper/go-lakekeeper/issues/52)) ([5f992a7](https://github.com/lakekeeper/go-lakekeeper/commit/5f992a7e9c49a5400490677758b4af744e64dbd4))
* **permission:** remove project scope on warehouse ([#87](https://github.com/lakekeeper/go-lakekeeper/issues/87)) ([df3d613](https://github.com/lakekeeper/go-lakekeeper/commit/df3d613f4e43ca49fec0dfd435cbc756b888811c))
* **permissions:** add filtering support to server get access endpoint ([#69](https://github.com/lakekeeper/go-lakekeeper/issues/69)) ([7c6204a](https://github.com/lakekeeper/go-lakekeeper/commit/7c6204adf31d4527b3c5306a10a7ff1be4284d4f))
* **permissions:** get access can now be successfully filtered ([7c6204a](https://github.com/lakekeeper/go-lakekeeper/commit/7c6204adf31d4527b3c5306a10a7ff1be4284d4f))
* **project:** add get api statistics endpoint support ([#70](https://github.com/lakekeeper/go-lakekeeper/issues/70)) ([30e2f38](https://github.com/lakekeeper/go-lakekeeper/commit/30e2f381cc4eebbefc2ac2b79a57c50134852c30))
* **project:** add GetAllowedActions ([#216](https://github.com/lakekeeper/go-lakekeeper/issues/216)) ([9a08270](https://github.com/lakekeeper/go-lakekeeper/commit/9a08270f345abc2a5e881dad91a53b870d05a51e))
* **project:** add missing methods DeleteDefault/RenameDefault ([#22](https://github.com/lakekeeper/go-lakekeeper/issues/22)) ([c5b3ad8](https://github.com/lakekeeper/go-lakekeeper/commit/c5b3ad8ca6a134606b515f7bd4c78454210e05e4))
* remove deprecated default-project related endpoints ([#181](https://github.com/lakekeeper/go-lakekeeper/issues/181)) ([ca7779c](https://github.com/lakekeeper/go-lakekeeper/commit/ca7779c1c64a016ebb86de2eba94a2485a214f94))
* rename project Default method to GetDefault ([#21](https://github.com/lakekeeper/go-lakekeeper/issues/21)) ([e3165d1](https://github.com/lakekeeper/go-lakekeeper/commit/e3165d1dcfa317be04b52e78d2c5775ef5ef64a8))
* **role:** add new get allowed authorizer actions ([#200](https://github.com/lakekeeper/go-lakekeeper/issues/200)) ([fbcb6df](https://github.com/lakekeeper/go-lakekeeper/commit/fbcb6df42b9b9340a063cd5b48df676dc6525a92))
* **role:** add search method ([#23](https://github.com/lakekeeper/go-lakekeeper/issues/23)) ([c774335](https://github.com/lakekeeper/go-lakekeeper/commit/c774335c70b03ec45931fa0d841b5fa45637aba7))
* **server:** add GetAllowedActions ([#215](https://github.com/lakekeeper/go-lakekeeper/issues/215)) ([0db6ed5](https://github.com/lakekeeper/go-lakekeeper/commit/0db6ed5987f9538890431a5db924e013eb406757))
* **server:** add new get allowed authorizer actions ([#197](https://github.com/lakekeeper/go-lakekeeper/issues/197)) ([09beac7](https://github.com/lakekeeper/go-lakekeeper/commit/09beac7778cda490720ff053079f533aaa8e7f79))
* **storage:** not sending errors back on storage creds/profile options func ([#16](https://github.com/lakekeeper/go-lakekeeper/issues/16)) ([4c7f02f](https://github.com/lakekeeper/go-lakekeeper/commit/4c7f02fc4f2f33381f9f3e0b1e0a5555ab444045))
* **test:** add client options tests ([#99](https://github.com/lakekeeper/go-lakekeeper/issues/99)) ([08d7779](https://github.com/lakekeeper/go-lakekeeper/commit/08d777929a585641aeb978eddd2b763896af290e))
* **test:** add missing tests for warehouse service ([#15](https://github.com/lakekeeper/go-lakekeeper/issues/15)) ([1105bb7](https://github.com/lakekeeper/go-lakekeeper/commit/1105bb7461c5227b17d8565d853332404b89bb2e))
* **test:** add unit tests ([#6](https://github.com/lakekeeper/go-lakekeeper/issues/6)) ([8c26112](https://github.com/lakekeeper/go-lakekeeper/commit/8c26112329f63774125dbee3fc1136d344a39468))
* **test:** proposal unit tests structure ([#10](https://github.com/lakekeeper/go-lakekeeper/issues/10)) ([cacf1b6](https://github.com/lakekeeper/go-lakekeeper/commit/cacf1b65098a0260244dce80b9ebd90298aad40e))
* use debian trixie in container ([#194](https://github.com/lakekeeper/go-lakekeeper/issues/194)) ([c6d7e8f](https://github.com/lakekeeper/go-lakekeeper/commit/c6d7e8f5e37ad06e2dcc066e8968d49da9d3739f))
* **user:** add search and list methods ([#25](https://github.com/lakekeeper/go-lakekeeper/issues/25)) ([35f85ac](https://github.com/lakekeeper/go-lakekeeper/commit/35f85acbe57255f3989012b00220383e1bc6e7a8))
* **warehouse:** add deprecation notice for GetProtection ([#96](https://github.com/lakekeeper/go-lakekeeper/issues/96)) ([df774ba](https://github.com/lakekeeper/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add few missing methods ([#94](https://github.com/lakekeeper/go-lakekeeper/issues/94)) ([20e080b](https://github.com/lakekeeper/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add get statistics ([#95](https://github.com/lakekeeper/go-lakekeeper/issues/95)) ([cc8ecff](https://github.com/lakekeeper/go-lakekeeper/commit/cc8ecffc5a3ba428e8c81a91b1a1678c1aa80be2))
* **warehouse:** add GetNamespaceProtection ([#94](https://github.com/lakekeeper/go-lakekeeper/issues/94)) ([20e080b](https://github.com/lakekeeper/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add GetTableProtection method ([#96](https://github.com/lakekeeper/go-lakekeeper/issues/96)) ([df774ba](https://github.com/lakekeeper/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add GetViewProtection method ([#96](https://github.com/lakekeeper/go-lakekeeper/issues/96)) ([df774ba](https://github.com/lakekeeper/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add ListSoftDeletedTabular ([#94](https://github.com/lakekeeper/go-lakekeeper/issues/94)) ([20e080b](https://github.com/lakekeeper/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add new actions `get_all_tasks` and `control_all_tasks` ([#143](https://github.com/lakekeeper/go-lakekeeper/issues/143)) ([acab155](https://github.com/lakekeeper/go-lakekeeper/commit/acab15570352548da7d033f329b9d762b0a70f7b))
* **warehouse:** add new permission methods - mark `GetAccess` as deprecated ([#195](https://github.com/lakekeeper/go-lakekeeper/issues/195)) ([98491bd](https://github.com/lakekeeper/go-lakekeeper/commit/98491bdda47f29cba48c5ef662e284d3884a3987))
* **warehouse:** add SetNamespaceProtection ([#94](https://github.com/lakekeeper/go-lakekeeper/issues/94)) ([20e080b](https://github.com/lakekeeper/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add SetTableProtection method ([#96](https://github.com/lakekeeper/go-lakekeeper/issues/96)) ([df774ba](https://github.com/lakekeeper/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add SetViewProtection method ([#96](https://github.com/lakekeeper/go-lakekeeper/issues/96)) ([df774ba](https://github.com/lakekeeper/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add table and view protection methods ([#96](https://github.com/lakekeeper/go-lakekeeper/issues/96)) ([df774ba](https://github.com/lakekeeper/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add UndropTabular ([#94](https://github.com/lakekeeper/go-lakekeeper/issues/94)) ([20e080b](https://github.com/lakekeeper/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))


### Bug Fixes

* **ci:** delete old release please config ([#34](https://github.com/lakekeeper/go-lakekeeper/issues/34)) ([f6140c3](https://github.com/lakekeeper/go-lakekeeper/commit/f6140c3e7162659848e85578a04bf2f3c12f4506))
* **cli:** no authentication on version command ([#113](https://github.com/lakekeeper/go-lakekeeper/issues/113)) ([d5687de](https://github.com/lakekeeper/go-lakekeeper/commit/d5687de8f48a6bd2941b1ce93a51c0700aaf9fee))
* **cli:** project was not used in role/warehouse commands ([#128](https://github.com/lakekeeper/go-lakekeeper/issues/128)) ([6251582](https://github.com/lakekeeper/go-lakekeeper/commit/6251582c18402f455aa71ab2f1b31981f1867251))
* **deps:** update module github.com/apache/iceberg-go to v0.5.0 ([#238](https://github.com/lakekeeper/go-lakekeeper/issues/238)) ([423f34e](https://github.com/lakekeeper/go-lakekeeper/commit/423f34e9b4d0a2c708a25dec7f75ab8ee0634107))
* **deps:** update module github.com/google/go-querystring to v1.2.0 ([#219](https://github.com/lakekeeper/go-lakekeeper/issues/219)) ([bf5b076](https://github.com/lakekeeper/go-lakekeeper/commit/bf5b0768b8c3c4f2128f3afac98aeec1e3dd4e3d))
* **deps:** update module github.com/sirupsen/logrus to v1.9.4 ([#223](https://github.com/lakekeeper/go-lakekeeper/issues/223)) ([bbb1bd0](https://github.com/lakekeeper/go-lakekeeper/commit/bbb1bd07c1af4c178a809c667e48eaa33bd354ee))
* **deps:** update module github.com/spf13/cobra to v1.10.2 ([#214](https://github.com/lakekeeper/go-lakekeeper/issues/214)) ([d0dd036](https://github.com/lakekeeper/go-lakekeeper/commit/d0dd0364bda2545795657a27a10de2245af46bd8))
* **deps:** update module golang.org/x/oauth2 to v0.34.0 ([#218](https://github.com/lakekeeper/go-lakekeeper/issues/218)) ([38dbde9](https://github.com/lakekeeper/go-lakekeeper/commit/38dbde93e8b203f79a778ebc7019615047572306))
* **deps:** update module golang.org/x/oauth2 to v0.35.0 ([#226](https://github.com/lakekeeper/go-lakekeeper/issues/226)) ([cc5e24a](https://github.com/lakekeeper/go-lakekeeper/commit/cc5e24af1d3cee54a97cbb2da0d1dd3616f10a78))
* **deps:** update module golang.org/x/oauth2 to v0.36.0 ([#241](https://github.com/lakekeeper/go-lakekeeper/issues/241)) ([7b93a16](https://github.com/lakekeeper/go-lakekeeper/commit/7b93a16d2318391e7c5cb11d04b8d49cd320ae60))
* ensure services are initialized with client ([#4](https://github.com/lakekeeper/go-lakekeeper/issues/4)) ([a74c0f1](https://github.com/lakekeeper/go-lakekeeper/commit/a74c0f1e281f80069dc90e323aa785a34c473f89))
* **permission:** rename all project related objects in server ([#74](https://github.com/lakekeeper/go-lakekeeper/issues/74)) ([49e8e0a](https://github.com/lakekeeper/go-lakekeeper/commit/49e8e0a67324044dbd96fc1525246a70e8648050))
* Resolve failing integration tests on permissions ([#182](https://github.com/lakekeeper/go-lakekeeper/issues/182)) ([ad8011f](https://github.com/lakekeeper/go-lakekeeper/commit/ad8011f86ab35951ef22e0eb331fc69baf4dbc07))
* temporary workaround for endpoints that does not use x-project-id ([#14](https://github.com/lakekeeper/go-lakekeeper/issues/14)) ([1c14ef9](https://github.com/lakekeeper/go-lakekeeper/commit/1c14ef9fa5910682a8a68dc40bb66397e0566456))
* **warehouse:** filter by status ([#102](https://github.com/lakekeeper/go-lakekeeper/issues/102)) ([a97ff1e](https://github.com/lakekeeper/go-lakekeeper/commit/a97ff1e904951b3476d67b78e4724a6dc0cc73bb))
* **warehouse:** rename remote signing url styles for s3 storage profile ([#130](https://github.com/lakekeeper/go-lakekeeper/issues/130)) ([82f30bf](https://github.com/lakekeeper/go-lakekeeper/commit/82f30bf3d10d391dd95d5352d84085ea193a7e96))


### Documentation

* add context in readme examples ([#93](https://github.com/lakekeeper/go-lakekeeper/issues/93)) ([14ccc07](https://github.com/lakekeeper/go-lakekeeper/commit/14ccc07341fb0cf59c7f31a5613cff0068fb4f63))
* generate CLI documentation ([#127](https://github.com/lakekeeper/go-lakekeeper/issues/127)) ([0610765](https://github.com/lakekeeper/go-lakekeeper/commit/0610765ea2b227c4e55b37bda97987c19c47a4b0))


### Miscellaneous Chores

* add Ptr helper method ([e6a04d5](https://github.com/lakekeeper/go-lakekeeper/commit/e6a04d54aa4f99a2968500a78b312f05a6632ee9))
* add release workflow ([#2](https://github.com/lakekeeper/go-lakekeeper/issues/2)) ([6a88c96](https://github.com/lakekeeper/go-lakekeeper/commit/6a88c96b00f018d8109b2a23b495dd523cb07dba))
* add release workflow ([#3](https://github.com/lakekeeper/go-lakekeeper/issues/3)) ([5383a81](https://github.com/lakekeeper/go-lakekeeper/commit/5383a81c3b113653f54cc0c67221bc06089f27e7))
* add status badges in README.md ([#98](https://github.com/lakekeeper/go-lakekeeper/issues/98)) ([15b9850](https://github.com/lakekeeper/go-lakekeeper/commit/15b98504727ef31025e6b72f20349f53b0d55832))
* add todo notic on missing endpoints ([bb4951d](https://github.com/lakekeeper/go-lakekeeper/commit/bb4951ddccb1313da9fa31337dab66d2c862dd77))
* add TODO notice on missing endpoints ([#19](https://github.com/lakekeeper/go-lakekeeper/issues/19)) ([bb4951d](https://github.com/lakekeeper/go-lakekeeper/commit/bb4951ddccb1313da9fa31337dab66d2c862dd77))
* bootstrap releases for path: . ([#35](https://github.com/lakekeeper/go-lakekeeper/issues/35)) ([cc54387](https://github.com/lakekeeper/go-lakekeeper/commit/cc543871555124d80e7de0a927494c45b8a575ab))
* **build:** set go version to 1.24 ([#101](https://github.com/lakekeeper/go-lakekeeper/issues/101)) ([21cf182](https://github.com/lakekeeper/go-lakekeeper/commit/21cf182758e89c93f1873b0e03ca91589a4bd10a))
* **ci:** Add PR title checker ([#123](https://github.com/lakekeeper/go-lakekeeper/issues/123)) ([8ca0ca9](https://github.com/lakekeeper/go-lakekeeper/commit/8ca0ca9636f6cec60bdd7df11d46ca5ab343b0ae))
* **ci:** add release please manifest file ([#38](https://github.com/lakekeeper/go-lakekeeper/issues/38)) ([53dae40](https://github.com/lakekeeper/go-lakekeeper/commit/53dae4002e12e5520a8023cbb7ebfde64e555262))
* **ci:** add v0.9.3 support ([#80](https://github.com/lakekeeper/go-lakekeeper/issues/80)) ([9d6d2c3](https://github.com/lakekeeper/go-lakekeeper/commit/9d6d2c3f650362a0b5378ce94a79782a6d4ca1a7))
* **ci:** bump minor version pre major ([#41](https://github.com/lakekeeper/go-lakekeeper/issues/41)) ([402f3bb](https://github.com/lakekeeper/go-lakekeeper/commit/402f3bb32ffc40fe80949e3facdcbf91759f7f7e))
* **ci:** enable skip storage validation ([#44](https://github.com/lakekeeper/go-lakekeeper/issues/44)) ([2215b52](https://github.com/lakekeeper/go-lakekeeper/commit/2215b52b3204f296e4f12b7d237a6e66d3ef4fba))
* **ci:** fix lint CLI add warehouse command ([#122](https://github.com/lakekeeper/go-lakekeeper/issues/122)) ([91b7cb9](https://github.com/lakekeeper/go-lakekeeper/commit/91b7cb9bf8b54824e372352f17f1d0de053ce0d0))
* **ci:** fix version.go file name in release please config ([0f2f520](https://github.com/lakekeeper/go-lakekeeper/commit/0f2f520c8a09bd72ea48632998b4224be7d68d34))
* **ci:** publish container image on main branch ([#106](https://github.com/lakekeeper/go-lakekeeper/issues/106)) ([62e20ff](https://github.com/lakekeeper/go-lakekeeper/commit/62e20ffab931d331804f60e3620cd6c9d83b29bc))
* **ci:** remove daily schedule on renovate config ([#166](https://github.com/lakekeeper/go-lakekeeper/issues/166)) ([d37272b](https://github.com/lakekeeper/go-lakekeeper/commit/d37272bd322b25ba621e3f4f45ae6540509ebb80))
* **ci:** remove lock workflow ([#134](https://github.com/lakekeeper/go-lakekeeper/issues/134)) ([db69bb1](https://github.com/lakekeeper/go-lakekeeper/commit/db69bb1ae160c523e2531db6dd1016b762581a29))
* **ci:** rename published binaries ([#117](https://github.com/lakekeeper/go-lakekeeper/issues/117)) ([a1e5f52](https://github.com/lakekeeper/go-lakekeeper/commit/a1e5f52c18dfbcf9546b6145d22db5efce73b560))
* **ci:** set docs label on docs/** change ([#125](https://github.com/lakekeeper/go-lakekeeper/issues/125)) ([b06c2a1](https://github.com/lakekeeper/go-lakekeeper/commit/b06c2a1180fd29cd80368885d224e4d9113bd78a))
* **ci:** set up renovate ([#163](https://github.com/lakekeeper/go-lakekeeper/issues/163)) ([cf6078c](https://github.com/lakekeeper/go-lakekeeper/commit/cf6078c406747b02043a852856b628d895cc3c51))
* **ci:** update goreleaser configuration ([#184](https://github.com/lakekeeper/go-lakekeeper/issues/184)) ([d15ee51](https://github.com/lakekeeper/go-lakekeeper/commit/d15ee51033332b605ac702f8e8652b32e3d8b596))
* **ci:** use `latest` version of golangci-lint ([#183](https://github.com/lakekeeper/go-lakekeeper/issues/183)) ([6a26f9f](https://github.com/lakekeeper/go-lakekeeper/commit/6a26f9f837b6a89c2b5c5a122273cfad4a220ed2))
* **ci:** use docker buildx ([935261d](https://github.com/lakekeeper/go-lakekeeper/commit/935261d4bc02f8c2842d40fa74db972750095984))
* **ci:** use labeler for PR labeling ([#81](https://github.com/lakekeeper/go-lakekeeper/issues/81)) ([757ac84](https://github.com/lakekeeper/go-lakekeeper/commit/757ac8478a1920ee75c8b34ed7d704ea0acc2762))
* **ci:** use lakekeeper v0.10.0 ([#144](https://github.com/lakekeeper/go-lakekeeper/issues/144)) ([0ae88f2](https://github.com/lakekeeper/go-lakekeeper/commit/0ae88f22ba1a8de82d040f2a0205203d4d97f04e))
* clean CHANGELOG.md ([#50](https://github.com/lakekeeper/go-lakekeeper/issues/50)) ([f3fbf32](https://github.com/lakekeeper/go-lakekeeper/commit/f3fbf3208b16621c02f14b14797507eb264772e9))
* clean code ([#97](https://github.com/lakekeeper/go-lakekeeper/issues/97)) ([9cf0d13](https://github.com/lakekeeper/go-lakekeeper/commit/9cf0d13bef51dd4652910d24221bd6b3684dd37d))
* **config:** migrate Renovate config ([#172](https://github.com/lakekeeper/go-lakekeeper/issues/172)) ([7bc83cb](https://github.com/lakekeeper/go-lakekeeper/commit/7bc83cb748f334836ebf229b86b2a259791be9a0))
* **deps:** bump actions/checkout from 4 to 5 in the github-actions group ([#132](https://github.com/lakekeeper/go-lakekeeper/issues/132)) ([5ec4f2c](https://github.com/lakekeeper/go-lakekeeper/commit/5ec4f2c875f8cc4402a68b2ebe7badfc79053299))
* **deps:** bump github.com/apache/iceberg-go from 0.3.0 to 0.4.0 ([#147](https://github.com/lakekeeper/go-lakekeeper/issues/147)) ([98b9ef4](https://github.com/lakekeeper/go-lakekeeper/commit/98b9ef453f52fa2c31220f07271b44af110c3488))
* **deps:** bump github.com/go-viper/mapstructure/v2 ([f6a6bc7](https://github.com/lakekeeper/go-lakekeeper/commit/f6a6bc7d1ecc51078645ba3312f1d3bf41faace1))
* **deps:** bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in the go_modules group ([#105](https://github.com/lakekeeper/go-lakekeeper/issues/105)) ([f6a6bc7](https://github.com/lakekeeper/go-lakekeeper/commit/f6a6bc7d1ecc51078645ba3312f1d3bf41faace1))
* **deps:** bump github.com/go-viper/mapstructure/v2 from 2.3.0 to 2.4.0 in the go_modules group ([#135](https://github.com/lakekeeper/go-lakekeeper/issues/135)) ([d0400a9](https://github.com/lakekeeper/go-lakekeeper/commit/d0400a9acec2b9ed16d20e3202206aca122a3c7f))
* **deps:** bump github.com/spf13/cobra from 1.9.1 to 1.10.1 ([#138](https://github.com/lakekeeper/go-lakekeeper/issues/138)) ([15bcbc3](https://github.com/lakekeeper/go-lakekeeper/commit/15bcbc3073a8e4a4b1d3c19d4708154858175b37))
* **deps:** bump github.com/stretchr/testify from 1.10.0 to 1.11.0 ([#136](https://github.com/lakekeeper/go-lakekeeper/issues/136)) ([1f94fb8](https://github.com/lakekeeper/go-lakekeeper/commit/1f94fb87408ed55c2a9de43222dcfa4835f2e10e))
* **deps:** bump github.com/stretchr/testify from 1.11.0 to 1.11.1 ([#137](https://github.com/lakekeeper/go-lakekeeper/issues/137)) ([5f1f15f](https://github.com/lakekeeper/go-lakekeeper/commit/5f1f15f747759f3fa3517abfb6c7477e1659165e))
* **deps:** bump go.opentelemetry.io/otel/sdk ([#233](https://github.com/lakekeeper/go-lakekeeper/issues/233)) ([be7de33](https://github.com/lakekeeper/go-lakekeeper/commit/be7de33f2ea86d95822f9aa6beefa8b97d526ce3))
* **deps:** bump golang.org/x/crypto from 0.42.0 to 0.45.0 in the go_modules group across 1 directory ([#177](https://github.com/lakekeeper/go-lakekeeper/issues/177)) ([495ef25](https://github.com/lakekeeper/go-lakekeeper/commit/495ef2580fa6c26755ac6bf049783319eaa36426))
* **deps:** bump golang.org/x/oauth2 from 0.30.0 to 0.31.0 ([#140](https://github.com/lakekeeper/go-lakekeeper/issues/140)) ([b991475](https://github.com/lakekeeper/go-lakekeeper/commit/b991475e0c318cd7a38123ac23a043ef3a1fbe7e))
* **deps:** bump golang.org/x/oauth2 from 0.31.0 to 0.32.0 ([#146](https://github.com/lakekeeper/go-lakekeeper/issues/146)) ([ece634e](https://github.com/lakekeeper/go-lakekeeper/commit/ece634e7559c05e72e277f5d95a69329484b9fa1))
* **deps:** bump golang.org/x/oauth2 from 0.32.0 to 0.33.0 ([#150](https://github.com/lakekeeper/go-lakekeeper/issues/150)) ([961f61a](https://github.com/lakekeeper/go-lakekeeper/commit/961f61a71164d0777f8418d0aaf721d006de57f1))
* **deps:** bump golangci/golangci-lint-action from 8.0.0 to 9.0.0 in the github-actions group ([#149](https://github.com/lakekeeper/go-lakekeeper/issues/149)) ([7ba84a0](https://github.com/lakekeeper/go-lakekeeper/commit/7ba84a018920096b792a156e3a6cb8e67fc45db6))
* **deps:** bump google.golang.org/grpc ([#244](https://github.com/lakekeeper/go-lakekeeper/issues/244)) ([6c6f6d2](https://github.com/lakekeeper/go-lakekeeper/commit/6c6f6d250cdc166ae56833e58a04ea7c7b7fa4dd))
* **deps:** bump the github-actions group with 2 updates ([#104](https://github.com/lakekeeper/go-lakekeeper/issues/104)) ([914b439](https://github.com/lakekeeper/go-lakekeeper/commit/914b4394defa652f3cd31ad331365d5072bb67bd))
* **deps:** bump the github-actions group with 2 updates ([#139](https://github.com/lakekeeper/go-lakekeeper/issues/139)) ([099c378](https://github.com/lakekeeper/go-lakekeeper/commit/099c378c18cfcc9aac0c07a5a0c668decd542af4))
* **deps:** update actions/checkout action to v6 ([#179](https://github.com/lakekeeper/go-lakekeeper/issues/179)) ([2cd360c](https://github.com/lakekeeper/go-lakekeeper/commit/2cd360ca302d21cf060582caf412d737e88b7c69))
* **deps:** update actions/checkout action to v6.0.1 ([#211](https://github.com/lakekeeper/go-lakekeeper/issues/211)) ([559b8ae](https://github.com/lakekeeper/go-lakekeeper/commit/559b8ae4b92f621d63439bf087dc5ce9fa7ae10d))
* **deps:** update actions/checkout action to v6.0.2 ([#225](https://github.com/lakekeeper/go-lakekeeper/issues/225)) ([806e586](https://github.com/lakekeeper/go-lakekeeper/commit/806e5862cff84a03fb1d55b1a1908fd9b8e90908))
* **deps:** update actions/setup-go action to v6.2.0 ([#222](https://github.com/lakekeeper/go-lakekeeper/issues/222)) ([69b5477](https://github.com/lakekeeper/go-lakekeeper/commit/69b547754a4efb07a1886743e9852ec60e3bada2))
* **deps:** update actions/setup-go action to v6.3.0 ([#232](https://github.com/lakekeeper/go-lakekeeper/issues/232)) ([af882d1](https://github.com/lakekeeper/go-lakekeeper/commit/af882d1c4c38767a6ad805feb32c2eadef438007))
* **deps:** update actions/setup-go action to v6.4.0 ([#249](https://github.com/lakekeeper/go-lakekeeper/issues/249)) ([2091be4](https://github.com/lakekeeper/go-lakekeeper/commit/2091be4a61c88150518670b570e47cc36817a7aa))
* **deps:** update all non-major dependencies ([df084fc](https://github.com/lakekeeper/go-lakekeeper/commit/df084fcba13a369578f7ed4b24b45e49aa93b028))
* **deps:** update all non-major dependencies ([95db8cb](https://github.com/lakekeeper/go-lakekeeper/commit/95db8cb086609791a7ba8986e083dceaafcd5d67))
* **deps:** update all non-major dependencies (minor) ([#168](https://github.com/lakekeeper/go-lakekeeper/issues/168)) ([df084fc](https://github.com/lakekeeper/go-lakekeeper/commit/df084fcba13a369578f7ed4b24b45e49aa93b028))
* **deps:** update all non-major dependencies (minor) ([#178](https://github.com/lakekeeper/go-lakekeeper/issues/178)) ([d37e55a](https://github.com/lakekeeper/go-lakekeeper/commit/d37e55a78c554e0c717d1e20ece5524ed9f5193c))
* **deps:** update all non-major dependencies (patch) ([#167](https://github.com/lakekeeper/go-lakekeeper/issues/167)) ([95db8cb](https://github.com/lakekeeper/go-lakekeeper/commit/95db8cb086609791a7ba8986e083dceaafcd5d67))
* **deps:** update codecov/codecov-action action to v6 ([#248](https://github.com/lakekeeper/go-lakekeeper/issues/248)) ([642ab08](https://github.com/lakekeeper/go-lakekeeper/commit/642ab08c914f086ab4d3f43cce904514ad921ac1))
* **deps:** update crazy-max/ghaction-import-gpg action to v7 ([#234](https://github.com/lakekeeper/go-lakekeeper/issues/234)) ([e6b3323](https://github.com/lakekeeper/go-lakekeeper/commit/e6b3323feeab62834c0421b7e4f2baaf378596f1))
* **deps:** update dependency go to v1.25.5 ([#210](https://github.com/lakekeeper/go-lakekeeper/issues/210)) ([555bc20](https://github.com/lakekeeper/go-lakekeeper/commit/555bc203266fa758bdc859830cccc3bef5bee46f))
* **deps:** update dependency go to v1.25.7 ([#224](https://github.com/lakekeeper/go-lakekeeper/issues/224)) ([82eb60f](https://github.com/lakekeeper/go-lakekeeper/commit/82eb60fc835b6f29bbd4b710c18fa7aa5c34c86b))
* **deps:** update dependency go to v1.26.2 ([#250](https://github.com/lakekeeper/go-lakekeeper/issues/250)) ([c01f396](https://github.com/lakekeeper/go-lakekeeper/commit/c01f396ea2bf96493f451709a3e4f891fab3f906))
* **deps:** update dependency mkdocs-material to v9 ([#159](https://github.com/lakekeeper/go-lakekeeper/issues/159)) ([64af7d1](https://github.com/lakekeeper/go-lakekeeper/commit/64af7d1e07998e84a4a6fc41a456d4383bb5dc95))
* **deps:** update dependency ubuntu to v24 ([#160](https://github.com/lakekeeper/go-lakekeeper/issues/160)) ([11ac251](https://github.com/lakekeeper/go-lakekeeper/commit/11ac251bb114d26967a1ad729e2e72e993d50f8c))
* **deps:** update docker/build-push-action action to v7 ([#239](https://github.com/lakekeeper/go-lakekeeper/issues/239)) ([a925732](https://github.com/lakekeeper/go-lakekeeper/commit/a9257320595b532207f6dedb0759e0405715efdd))
* **deps:** update docker/login-action action to v4 ([#235](https://github.com/lakekeeper/go-lakekeeper/issues/235)) ([67fc176](https://github.com/lakekeeper/go-lakekeeper/commit/67fc176ea975be8cc816e00078e8eda8a4281f3f))
* **deps:** update docker/setup-buildx-action action to v4 ([#237](https://github.com/lakekeeper/go-lakekeeper/issues/237)) ([d9eaeb2](https://github.com/lakekeeper/go-lakekeeper/commit/d9eaeb20f941b073428ff73af3e53c69fbc06a53))
* **deps:** update docker/setup-qemu-action action to v4 ([#236](https://github.com/lakekeeper/go-lakekeeper/issues/236)) ([185328c](https://github.com/lakekeeper/go-lakekeeper/commit/185328c2fe0c6a46d5684c13a31d17fa48ba61ad))
* **deps:** update go-version ([#175](https://github.com/lakekeeper/go-lakekeeper/issues/175)) ([b4d6bb2](https://github.com/lakekeeper/go-lakekeeper/commit/b4d6bb25d9f7bb52c22e3184241300dc8d49436e))
* **deps:** update go-version ([#228](https://github.com/lakekeeper/go-lakekeeper/issues/228)) ([b54bc31](https://github.com/lakekeeper/go-lakekeeper/commit/b54bc312b1080eea81d7f5c4580ba35b5e7a4250))
* **deps:** update golangci/golangci-lint-action action to v9.2.0 ([#212](https://github.com/lakekeeper/go-lakekeeper/issues/212)) ([50b283d](https://github.com/lakekeeper/go-lakekeeper/commit/50b283d2189e4bc82c8527bfda4c5264b975dfb9))
* **deps:** update goreleaser/goreleaser-action action to v7 ([#231](https://github.com/lakekeeper/go-lakekeeper/issues/231)) ([cea131f](https://github.com/lakekeeper/go-lakekeeper/commit/cea131fd148db728dc858e6c56ec61e89c1f1eb9))
* **deps:** update marocchino/sticky-pull-request-comment action to v3 ([#242](https://github.com/lakekeeper/go-lakekeeper/issues/242)) ([d430dfd](https://github.com/lakekeeper/go-lakekeeper/commit/d430dfd523333fb418ea9c661af0150b39b1dea2))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.10.1 ([#229](https://github.com/lakekeeper/go-lakekeeper/issues/229)) ([4c12c93](https://github.com/lakekeeper/go-lakekeeper/commit/4c12c936e3594cdda20039ff6b48000a749cb2bd))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.11.3 ([#240](https://github.com/lakekeeper/go-lakekeeper/issues/240)) ([3776d19](https://github.com/lakekeeper/go-lakekeeper/commit/3776d19f833f908e0b87aaa3e2421aff9980b3cf))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.11.4 ([#246](https://github.com/lakekeeper/go-lakekeeper/issues/246)) ([d253c6b](https://github.com/lakekeeper/go-lakekeeper/commit/d253c6b142e2cbc3396beb4853c3af9e8c5d369b))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.7.1 ([#213](https://github.com/lakekeeper/go-lakekeeper/issues/213)) ([3f660a4](https://github.com/lakekeeper/go-lakekeeper/commit/3f660a4ee7d082d0efca2ecec45f961731b1ac17))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.7.2 ([#217](https://github.com/lakekeeper/go-lakekeeper/issues/217)) ([7395b33](https://github.com/lakekeeper/go-lakekeeper/commit/7395b3314dee113c2976fcc9009d36c715f1aba4))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.8.0 ([#221](https://github.com/lakekeeper/go-lakekeeper/issues/221)) ([8221a3d](https://github.com/lakekeeper/go-lakekeeper/commit/8221a3d8c3b94d920ef952f2cd95bb7d7f56f16e))
* **deps:** update openfga/openfga docker tag to v1.12 ([#243](https://github.com/lakekeeper/go-lakekeeper/issues/243)) ([10dffd9](https://github.com/lakekeeper/go-lakekeeper/commit/10dffd98d15593b7b0c4763999db78096e9e8150))
* **deps:** update openfga/openfga docker tag to v1.14 ([#247](https://github.com/lakekeeper/go-lakekeeper/issues/247)) ([d3dd2e6](https://github.com/lakekeeper/go-lakekeeper/commit/d3dd2e656b168c703f43752f2b51bf30f544dad0))
* **deps:** update postgres docker tag to v18 ([#161](https://github.com/lakekeeper/go-lakekeeper/issues/161)) ([6e3d99e](https://github.com/lakekeeper/go-lakekeeper/commit/6e3d99e1b61811f88df2b08e57aa8647c17088c2))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.4.6 ([#188](https://github.com/lakekeeper/go-lakekeeper/issues/188)) ([4ba6a08](https://github.com/lakekeeper/go-lakekeeper/commit/4ba6a083ee9f0eff60d0210c9968e13864997c20))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.4.7 ([#209](https://github.com/lakekeeper/go-lakekeeper/issues/209)) ([1549c97](https://github.com/lakekeeper/go-lakekeeper/commit/1549c9741b9b262dd5583f19c9bfed30bc8e6779))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.5.2 ([#220](https://github.com/lakekeeper/go-lakekeeper/issues/220)) ([e0b5d3f](https://github.com/lakekeeper/go-lakekeeper/commit/e0b5d3fc339c594161965b8312c570fd5d89a92e))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.5.3 ([#227](https://github.com/lakekeeper/go-lakekeeper/issues/227)) ([1695331](https://github.com/lakekeeper/go-lakekeeper/commit/1695331b0b6ec001ff08545f27876af36aab0dee))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.5.5 ([#230](https://github.com/lakekeeper/go-lakekeeper/issues/230)) ([7ec6307](https://github.com/lakekeeper/go-lakekeeper/commit/7ec6307550c0f61d16ebbe6c920eed0155aad9f2))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.6.0 ([#251](https://github.com/lakekeeper/go-lakekeeper/issues/251)) ([eaecd49](https://github.com/lakekeeper/go-lakekeeper/commit/eaecd49d58b7cb681b1fbda87cf673121c1285d1))
* do not comment on correct PR title ([#196](https://github.com/lakekeeper/go-lakekeeper/issues/196)) ([1df4597](https://github.com/lakekeeper/go-lakekeeper/commit/1df45978ea9ebf7a6db08a237a8822793531fced))
* **docs:** add a table of contents in README.me ([#116](https://github.com/lakekeeper/go-lakekeeper/issues/116)) ([486f4c9](https://github.com/lakekeeper/go-lakekeeper/commit/486f4c994e24886554a806c030948d7bda908820))
* **docs:** add CLI examples ([#120](https://github.com/lakekeeper/go-lakekeeper/issues/120)) ([ed6d451](https://github.com/lakekeeper/go-lakekeeper/commit/ed6d45163fb99d167afb552829090e61b5ab405d))
* **docs:** add CLI usage in README.md ([#114](https://github.com/lakekeeper/go-lakekeeper/issues/114)) ([6844b14](https://github.com/lakekeeper/go-lakekeeper/commit/6844b14cfd06a3c231dcffc24ba81d85dacfde61))
* **docs:** add README.md ([#7](https://github.com/lakekeeper/go-lakekeeper/issues/7)) ([cb39cda](https://github.com/lakekeeper/go-lakekeeper/commit/cb39cda59439288132eaf8ea8f12e854bb732290))
* **docs:** fix Default / GetDefault renamed method ([#28](https://github.com/lakekeeper/go-lakekeeper/issues/28)) ([a5c25d9](https://github.com/lakekeeper/go-lakekeeper/commit/a5c25d954771aee518a43e39c4078b04b72b2a6a))
* **docs:** replace nightly badge in README.md ([#112](https://github.com/lakekeeper/go-lakekeeper/issues/112)) ([5aed91f](https://github.com/lakekeeper/go-lakekeeper/commit/5aed91f451c82acdebb56cff6b6285f2d44cade9))
* DRY in integration tests ([#76](https://github.com/lakekeeper/go-lakekeeper/issues/76)) ([93cab0f](https://github.com/lakekeeper/go-lakekeeper/commit/93cab0fcc09ea0dd806f668a3c88f8d5af2657dc))
* fix comment errors on list users ([#26](https://github.com/lakekeeper/go-lakekeeper/issues/26)) ([de09274](https://github.com/lakekeeper/go-lakekeeper/commit/de09274cdaaffdb709a150f51ff4db77c37220d3))
* fix goreleaser ([#185](https://github.com/lakekeeper/go-lakekeeper/issues/185)) ([2511d56](https://github.com/lakekeeper/go-lakekeeper/commit/2511d56029e22cd2f5545fd9cab4f40e59f4436c))
* fix goreleaser release repo name ([#110](https://github.com/lakekeeper/go-lakekeeper/issues/110)) ([320e29a](https://github.com/lakekeeper/go-lakekeeper/commit/320e29ae9a2567ed7e154d67b2852bff47392eef))
* fix publish container image on release ([#108](https://github.com/lakekeeper/go-lakekeeper/issues/108)) ([ace86ef](https://github.com/lakekeeper/go-lakekeeper/commit/ace86efdbc04cf5afd5752f036ffb0d6710c3af7))
* initial commit ([2216f89](https://github.com/lakekeeper/go-lakekeeper/commit/2216f8902046738e7e86c88fb8c9d6b49e0861cc))
* **main:** release 0.0.10 ([#88](https://github.com/lakekeeper/go-lakekeeper/issues/88)) ([cbbd591](https://github.com/lakekeeper/go-lakekeeper/commit/cbbd591b57bd4239dd2fdd11fa6cea3f5fcedfb0))
* **main:** release 0.0.11 ([#91](https://github.com/lakekeeper/go-lakekeeper/issues/91)) ([cba91d9](https://github.com/lakekeeper/go-lakekeeper/commit/cba91d92f55451a167c7d7158aeeda1a9e957f69))
* **main:** release 0.0.12 ([#100](https://github.com/lakekeeper/go-lakekeeper/issues/100)) ([a738228](https://github.com/lakekeeper/go-lakekeeper/commit/a7382282197eeaf718dd34e17345440a1648a710))
* **main:** release 0.0.13 ([#109](https://github.com/lakekeeper/go-lakekeeper/issues/109)) ([6108d57](https://github.com/lakekeeper/go-lakekeeper/commit/6108d576e44c046ac1fce5e0b1716f4c3159b28f))
* **main:** release 0.0.14 ([#111](https://github.com/lakekeeper/go-lakekeeper/issues/111)) ([41582b2](https://github.com/lakekeeper/go-lakekeeper/commit/41582b2ca19fed2b0ce4764178a3cdc6b7263591))
* **main:** release 0.0.15 ([#115](https://github.com/lakekeeper/go-lakekeeper/issues/115)) ([cfe0f73](https://github.com/lakekeeper/go-lakekeeper/commit/cfe0f733b5064bd99261980d7a92e3493495bf47))
* **main:** release 0.0.16 ([#129](https://github.com/lakekeeper/go-lakekeeper/issues/129)) ([390ad36](https://github.com/lakekeeper/go-lakekeeper/commit/390ad367e59e12803bfe43f84c0024ac79c736a8))
* **main:** release 0.0.17 ([#131](https://github.com/lakekeeper/go-lakekeeper/issues/131)) ([268cba9](https://github.com/lakekeeper/go-lakekeeper/commit/268cba98d27171680a5ef112eb6999ba764a132d))
* **main:** release 0.0.18 ([#133](https://github.com/lakekeeper/go-lakekeeper/issues/133)) ([d77e6e2](https://github.com/lakekeeper/go-lakekeeper/commit/d77e6e20c0916fc6d75938c54575aa7fafdcf584))
* **main:** release 0.0.19 ([#145](https://github.com/lakekeeper/go-lakekeeper/issues/145)) ([0c8a65b](https://github.com/lakekeeper/go-lakekeeper/commit/0c8a65b4b50d4fba43c7120452958fa41a7f1e99))
* **main:** release 0.0.20 ([#148](https://github.com/lakekeeper/go-lakekeeper/issues/148)) ([ba7c34b](https://github.com/lakekeeper/go-lakekeeper/commit/ba7c34bde5b842ef80cf46b50a8fe74323f99ee2))
* **main:** release 0.0.21 ([#186](https://github.com/lakekeeper/go-lakekeeper/issues/186)) ([a503747](https://github.com/lakekeeper/go-lakekeeper/commit/a5037479502e27641fb82db4a8af9b02b78344cd))
* **main:** release 0.0.22 ([#187](https://github.com/lakekeeper/go-lakekeeper/issues/187)) ([b9b2f45](https://github.com/lakekeeper/go-lakekeeper/commit/b9b2f45d262c5573007b836e639b533d54dd6b1a))
* **main:** release 0.0.23 ([#189](https://github.com/lakekeeper/go-lakekeeper/issues/189)) ([91263be](https://github.com/lakekeeper/go-lakekeeper/commit/91263be399c66b38f471ec958d64691f71dfd0a9))
* **main:** release 0.0.6 ([#49](https://github.com/lakekeeper/go-lakekeeper/issues/49)) ([26ae082](https://github.com/lakekeeper/go-lakekeeper/commit/26ae0822b31c1641ce1b27568ea6c38b99eaabad))
* **main:** release 0.0.7 ([#51](https://github.com/lakekeeper/go-lakekeeper/issues/51)) ([7e74b2f](https://github.com/lakekeeper/go-lakekeeper/commit/7e74b2f31ca174e2948d408a91a81720728990f5))
* **main:** release 0.0.8 ([#79](https://github.com/lakekeeper/go-lakekeeper/issues/79)) ([3da6009](https://github.com/lakekeeper/go-lakekeeper/commit/3da6009bd5d8dfc608f6ff87da1beb1a600c4d3d))
* **main:** release 0.0.9 ([#83](https://github.com/lakekeeper/go-lakekeeper/issues/83)) ([6e12b61](https://github.com/lakekeeper/go-lakekeeper/commit/6e12b610045feada08894b14c0d3ca954ee6e605))
* **main:** release github.com/baptistegh/go-lakekeeper 0.1.0 ([#42](https://github.com/lakekeeper/go-lakekeeper/issues/42)) ([c0d910d](https://github.com/lakekeeper/go-lakekeeper/commit/c0d910d192d8255ec08a775accfef967ff892940))
* prepare client migration from terraform provider repo ([#1](https://github.com/lakekeeper/go-lakekeeper/issues/1)) ([24bfa65](https://github.com/lakekeeper/go-lakekeeper/commit/24bfa65c9b0b1b34428dd0262f17b416078b7a6c))
* prepare release 0.0.11 ([afa161a](https://github.com/lakekeeper/go-lakekeeper/commit/afa161a43e419f61143ef8c5e92c46035ae5d437))
* release 0.0.6 ([#43](https://github.com/lakekeeper/go-lakekeeper/issues/43)) ([2b04a81](https://github.com/lakekeeper/go-lakekeeper/commit/2b04a81cd6acfe00dde81c0b1db4f9a8863a0bee))
* **release-please:** fix previous tag ([#46](https://github.com/lakekeeper/go-lakekeeper/issues/46)) ([4c73fed](https://github.com/lakekeeper/go-lakekeeper/commit/4c73fed63df61d40a1b715254d9d42fef70b2454))
* **release-please:** rename package name ([#45](https://github.com/lakekeeper/go-lakekeeper/issues/45)) ([67c9398](https://github.com/lakekeeper/go-lakekeeper/commit/67c9398c55f514d54c00ad85f88e70c37928d74b))
* **release-please:** rework v0.0.0 ([#48](https://github.com/lakekeeper/go-lakekeeper/issues/48)) ([fdda97f](https://github.com/lakekeeper/go-lakekeeper/commit/fdda97f611cc6b041135cbfbb3c36bedd6dce022))
* remove bitnami postgresql image ([#142](https://github.com/lakekeeper/go-lakekeeper/issues/142)) ([dc5881f](https://github.com/lakekeeper/go-lakekeeper/commit/dc5881fabd457414a711788efb3f900f50182261))
* remove license headers ([#193](https://github.com/lakekeeper/go-lakekeeper/issues/193)) ([59fbb2f](https://github.com/lakekeeper/go-lakekeeper/commit/59fbb2f15fb9acf66b3f1c4a48a5bd320a55336a))
* **renovate:** not grouping go toolchain updates ([#169](https://github.com/lakekeeper/go-lakekeeper/issues/169)) ([9414097](https://github.com/lakekeeper/go-lakekeeper/commit/941409787329c4904413e28f4de3d9b6fb7017df))
* **renovate:** remove grouping dependencies ([#180](https://github.com/lakekeeper/go-lakekeeper/issues/180)) ([6a09a93](https://github.com/lakekeeper/go-lakekeeper/commit/6a09a93b8a282c02a17fdb7422615b5d8aefe3b1))
* **renovate:** try gomod grouName ([#173](https://github.com/lakekeeper/go-lakekeeper/issues/173)) ([9563c73](https://github.com/lakekeeper/go-lakekeeper/commit/9563c733711d8c140b5b96eef66ee207d9c7e32a))
* **renovate:** Update configuration ([#171](https://github.com/lakekeeper/go-lakekeeper/issues/171)) ([bcf8dfb](https://github.com/lakekeeper/go-lakekeeper/commit/bcf8dfb1aeece94a461a6153a900a340dadac110))
* set up release please sections ([#107](https://github.com/lakekeeper/go-lakekeeper/issues/107)) ([2c04c77](https://github.com/lakekeeper/go-lakekeeper/commit/2c04c778c7b64d675c2349e81732aa0bac33425a))

## [0.0.23](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.22...v0.0.23) (2026-04-12)


### Features

* **project:** add GetAllowedActions ([#216](https://github.com/baptistegh/go-lakekeeper/issues/216)) ([9a08270](https://github.com/baptistegh/go-lakekeeper/commit/9a08270f345abc2a5e881dad91a53b870d05a51e))
* **role:** add new get allowed authorizer actions ([#200](https://github.com/baptistegh/go-lakekeeper/issues/200)) ([fbcb6df](https://github.com/baptistegh/go-lakekeeper/commit/fbcb6df42b9b9340a063cd5b48df676dc6525a92))
* **server:** add GetAllowedActions ([#215](https://github.com/baptistegh/go-lakekeeper/issues/215)) ([0db6ed5](https://github.com/baptistegh/go-lakekeeper/commit/0db6ed5987f9538890431a5db924e013eb406757))
* **server:** add new get allowed authorizer actions ([#197](https://github.com/baptistegh/go-lakekeeper/issues/197)) ([09beac7](https://github.com/baptistegh/go-lakekeeper/commit/09beac7778cda490720ff053079f533aaa8e7f79))
* use debian trixie in container ([#194](https://github.com/baptistegh/go-lakekeeper/issues/194)) ([c6d7e8f](https://github.com/baptistegh/go-lakekeeper/commit/c6d7e8f5e37ad06e2dcc066e8968d49da9d3739f))
* **warehouse:** add new permission methods - mark `GetAccess` as deprecated ([#195](https://github.com/baptistegh/go-lakekeeper/issues/195)) ([98491bd](https://github.com/baptistegh/go-lakekeeper/commit/98491bdda47f29cba48c5ef662e284d3884a3987))


### Bug Fixes

* **deps:** update module github.com/apache/iceberg-go to v0.5.0 ([#238](https://github.com/baptistegh/go-lakekeeper/issues/238)) ([423f34e](https://github.com/baptistegh/go-lakekeeper/commit/423f34e9b4d0a2c708a25dec7f75ab8ee0634107))
* **deps:** update module github.com/google/go-querystring to v1.2.0 ([#219](https://github.com/baptistegh/go-lakekeeper/issues/219)) ([bf5b076](https://github.com/baptistegh/go-lakekeeper/commit/bf5b0768b8c3c4f2128f3afac98aeec1e3dd4e3d))
* **deps:** update module github.com/sirupsen/logrus to v1.9.4 ([#223](https://github.com/baptistegh/go-lakekeeper/issues/223)) ([bbb1bd0](https://github.com/baptistegh/go-lakekeeper/commit/bbb1bd07c1af4c178a809c667e48eaa33bd354ee))
* **deps:** update module github.com/spf13/cobra to v1.10.2 ([#214](https://github.com/baptistegh/go-lakekeeper/issues/214)) ([d0dd036](https://github.com/baptistegh/go-lakekeeper/commit/d0dd0364bda2545795657a27a10de2245af46bd8))
* **deps:** update module golang.org/x/oauth2 to v0.34.0 ([#218](https://github.com/baptistegh/go-lakekeeper/issues/218)) ([38dbde9](https://github.com/baptistegh/go-lakekeeper/commit/38dbde93e8b203f79a778ebc7019615047572306))
* **deps:** update module golang.org/x/oauth2 to v0.35.0 ([#226](https://github.com/baptistegh/go-lakekeeper/issues/226)) ([cc5e24a](https://github.com/baptistegh/go-lakekeeper/commit/cc5e24af1d3cee54a97cbb2da0d1dd3616f10a78))
* **deps:** update module golang.org/x/oauth2 to v0.36.0 ([#241](https://github.com/baptistegh/go-lakekeeper/issues/241)) ([7b93a16](https://github.com/baptistegh/go-lakekeeper/commit/7b93a16d2318391e7c5cb11d04b8d49cd320ae60))


### Miscellaneous Chores

* **deps:** bump go.opentelemetry.io/otel/sdk ([#233](https://github.com/baptistegh/go-lakekeeper/issues/233)) ([be7de33](https://github.com/baptistegh/go-lakekeeper/commit/be7de33f2ea86d95822f9aa6beefa8b97d526ce3))
* **deps:** bump google.golang.org/grpc ([#244](https://github.com/baptistegh/go-lakekeeper/issues/244)) ([6c6f6d2](https://github.com/baptistegh/go-lakekeeper/commit/6c6f6d250cdc166ae56833e58a04ea7c7b7fa4dd))
* **deps:** update actions/checkout action to v6.0.1 ([#211](https://github.com/baptistegh/go-lakekeeper/issues/211)) ([559b8ae](https://github.com/baptistegh/go-lakekeeper/commit/559b8ae4b92f621d63439bf087dc5ce9fa7ae10d))
* **deps:** update actions/checkout action to v6.0.2 ([#225](https://github.com/baptistegh/go-lakekeeper/issues/225)) ([806e586](https://github.com/baptistegh/go-lakekeeper/commit/806e5862cff84a03fb1d55b1a1908fd9b8e90908))
* **deps:** update actions/setup-go action to v6.2.0 ([#222](https://github.com/baptistegh/go-lakekeeper/issues/222)) ([69b5477](https://github.com/baptistegh/go-lakekeeper/commit/69b547754a4efb07a1886743e9852ec60e3bada2))
* **deps:** update actions/setup-go action to v6.3.0 ([#232](https://github.com/baptistegh/go-lakekeeper/issues/232)) ([af882d1](https://github.com/baptistegh/go-lakekeeper/commit/af882d1c4c38767a6ad805feb32c2eadef438007))
* **deps:** update actions/setup-go action to v6.4.0 ([#249](https://github.com/baptistegh/go-lakekeeper/issues/249)) ([2091be4](https://github.com/baptistegh/go-lakekeeper/commit/2091be4a61c88150518670b570e47cc36817a7aa))
* **deps:** update codecov/codecov-action action to v6 ([#248](https://github.com/baptistegh/go-lakekeeper/issues/248)) ([642ab08](https://github.com/baptistegh/go-lakekeeper/commit/642ab08c914f086ab4d3f43cce904514ad921ac1))
* **deps:** update crazy-max/ghaction-import-gpg action to v7 ([#234](https://github.com/baptistegh/go-lakekeeper/issues/234)) ([e6b3323](https://github.com/baptistegh/go-lakekeeper/commit/e6b3323feeab62834c0421b7e4f2baaf378596f1))
* **deps:** update dependency go to v1.25.5 ([#210](https://github.com/baptistegh/go-lakekeeper/issues/210)) ([555bc20](https://github.com/baptistegh/go-lakekeeper/commit/555bc203266fa758bdc859830cccc3bef5bee46f))
* **deps:** update dependency go to v1.25.7 ([#224](https://github.com/baptistegh/go-lakekeeper/issues/224)) ([82eb60f](https://github.com/baptistegh/go-lakekeeper/commit/82eb60fc835b6f29bbd4b710c18fa7aa5c34c86b))
* **deps:** update dependency go to v1.26.2 ([#250](https://github.com/baptistegh/go-lakekeeper/issues/250)) ([c01f396](https://github.com/baptistegh/go-lakekeeper/commit/c01f396ea2bf96493f451709a3e4f891fab3f906))
* **deps:** update docker/build-push-action action to v7 ([#239](https://github.com/baptistegh/go-lakekeeper/issues/239)) ([a925732](https://github.com/baptistegh/go-lakekeeper/commit/a9257320595b532207f6dedb0759e0405715efdd))
* **deps:** update docker/login-action action to v4 ([#235](https://github.com/baptistegh/go-lakekeeper/issues/235)) ([67fc176](https://github.com/baptistegh/go-lakekeeper/commit/67fc176ea975be8cc816e00078e8eda8a4281f3f))
* **deps:** update docker/setup-buildx-action action to v4 ([#237](https://github.com/baptistegh/go-lakekeeper/issues/237)) ([d9eaeb2](https://github.com/baptistegh/go-lakekeeper/commit/d9eaeb20f941b073428ff73af3e53c69fbc06a53))
* **deps:** update docker/setup-qemu-action action to v4 ([#236](https://github.com/baptistegh/go-lakekeeper/issues/236)) ([185328c](https://github.com/baptistegh/go-lakekeeper/commit/185328c2fe0c6a46d5684c13a31d17fa48ba61ad))
* **deps:** update go-version ([#228](https://github.com/baptistegh/go-lakekeeper/issues/228)) ([b54bc31](https://github.com/baptistegh/go-lakekeeper/commit/b54bc312b1080eea81d7f5c4580ba35b5e7a4250))
* **deps:** update golangci/golangci-lint-action action to v9.2.0 ([#212](https://github.com/baptistegh/go-lakekeeper/issues/212)) ([50b283d](https://github.com/baptistegh/go-lakekeeper/commit/50b283d2189e4bc82c8527bfda4c5264b975dfb9))
* **deps:** update goreleaser/goreleaser-action action to v7 ([#231](https://github.com/baptistegh/go-lakekeeper/issues/231)) ([cea131f](https://github.com/baptistegh/go-lakekeeper/commit/cea131fd148db728dc858e6c56ec61e89c1f1eb9))
* **deps:** update marocchino/sticky-pull-request-comment action to v3 ([#242](https://github.com/baptistegh/go-lakekeeper/issues/242)) ([d430dfd](https://github.com/baptistegh/go-lakekeeper/commit/d430dfd523333fb418ea9c661af0150b39b1dea2))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.10.1 ([#229](https://github.com/baptistegh/go-lakekeeper/issues/229)) ([4c12c93](https://github.com/baptistegh/go-lakekeeper/commit/4c12c936e3594cdda20039ff6b48000a749cb2bd))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.11.3 ([#240](https://github.com/baptistegh/go-lakekeeper/issues/240)) ([3776d19](https://github.com/baptistegh/go-lakekeeper/commit/3776d19f833f908e0b87aaa3e2421aff9980b3cf))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.11.4 ([#246](https://github.com/baptistegh/go-lakekeeper/issues/246)) ([d253c6b](https://github.com/baptistegh/go-lakekeeper/commit/d253c6b142e2cbc3396beb4853c3af9e8c5d369b))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.7.1 ([#213](https://github.com/baptistegh/go-lakekeeper/issues/213)) ([3f660a4](https://github.com/baptistegh/go-lakekeeper/commit/3f660a4ee7d082d0efca2ecec45f961731b1ac17))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.7.2 ([#217](https://github.com/baptistegh/go-lakekeeper/issues/217)) ([7395b33](https://github.com/baptistegh/go-lakekeeper/commit/7395b3314dee113c2976fcc9009d36c715f1aba4))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.8.0 ([#221](https://github.com/baptistegh/go-lakekeeper/issues/221)) ([8221a3d](https://github.com/baptistegh/go-lakekeeper/commit/8221a3d8c3b94d920ef952f2cd95bb7d7f56f16e))
* **deps:** update openfga/openfga docker tag to v1.12 ([#243](https://github.com/baptistegh/go-lakekeeper/issues/243)) ([10dffd9](https://github.com/baptistegh/go-lakekeeper/commit/10dffd98d15593b7b0c4763999db78096e9e8150))
* **deps:** update openfga/openfga docker tag to v1.14 ([#247](https://github.com/baptistegh/go-lakekeeper/issues/247)) ([d3dd2e6](https://github.com/baptistegh/go-lakekeeper/commit/d3dd2e656b168c703f43752f2b51bf30f544dad0))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.4.6 ([#188](https://github.com/baptistegh/go-lakekeeper/issues/188)) ([4ba6a08](https://github.com/baptistegh/go-lakekeeper/commit/4ba6a083ee9f0eff60d0210c9968e13864997c20))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.4.7 ([#209](https://github.com/baptistegh/go-lakekeeper/issues/209)) ([1549c97](https://github.com/baptistegh/go-lakekeeper/commit/1549c9741b9b262dd5583f19c9bfed30bc8e6779))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.5.2 ([#220](https://github.com/baptistegh/go-lakekeeper/issues/220)) ([e0b5d3f](https://github.com/baptistegh/go-lakekeeper/commit/e0b5d3fc339c594161965b8312c570fd5d89a92e))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.5.3 ([#227](https://github.com/baptistegh/go-lakekeeper/issues/227)) ([1695331](https://github.com/baptistegh/go-lakekeeper/commit/1695331b0b6ec001ff08545f27876af36aab0dee))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.5.5 ([#230](https://github.com/baptistegh/go-lakekeeper/issues/230)) ([7ec6307](https://github.com/baptistegh/go-lakekeeper/commit/7ec6307550c0f61d16ebbe6c920eed0155aad9f2))
* **deps:** update quay.io/keycloak/keycloak docker tag to v26.6.0 ([#251](https://github.com/baptistegh/go-lakekeeper/issues/251)) ([eaecd49](https://github.com/baptistegh/go-lakekeeper/commit/eaecd49d58b7cb681b1fbda87cf673121c1285d1))
* do not comment on correct PR title ([#196](https://github.com/baptistegh/go-lakekeeper/issues/196)) ([1df4597](https://github.com/baptistegh/go-lakekeeper/commit/1df45978ea9ebf7a6db08a237a8822793531fced))
* remove license headers ([#193](https://github.com/baptistegh/go-lakekeeper/issues/193)) ([59fbb2f](https://github.com/baptistegh/go-lakekeeper/commit/59fbb2f15fb9acf66b3f1c4a48a5bd320a55336a))

## [0.0.22](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.21...v0.0.22) (2025-11-25)


### Miscellaneous Chores

* **ci:** use docker buildx ([935261d](https://github.com/baptistegh/go-lakekeeper/commit/935261d4bc02f8c2842d40fa74db972750095984))

## [0.0.21](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.20...v0.0.21) (2025-11-24)


### Miscellaneous Chores

* fix goreleaser ([#185](https://github.com/baptistegh/go-lakekeeper/issues/185)) ([2511d56](https://github.com/baptistegh/go-lakekeeper/commit/2511d56029e22cd2f5545fd9cab4f40e59f4436c))

## [0.0.20](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.19...v0.0.20) (2025-11-24)


### Features

* add statistics and protection warehouse/project actions ([#162](https://github.com/baptistegh/go-lakekeeper/issues/162)) ([ef5feed](https://github.com/baptistegh/go-lakekeeper/commit/ef5feed1aec6b75d3e640920475ca21f65b40246))
* remove deprecated default-project related endpoints ([#181](https://github.com/baptistegh/go-lakekeeper/issues/181)) ([ca7779c](https://github.com/baptistegh/go-lakekeeper/commit/ca7779c1c64a016ebb86de2eba94a2485a214f94))


### Bug Fixes

* Resolve failing integration tests on permissions ([#182](https://github.com/baptistegh/go-lakekeeper/issues/182)) ([ad8011f](https://github.com/baptistegh/go-lakekeeper/commit/ad8011f86ab35951ef22e0eb331fc69baf4dbc07))


### Miscellaneous Chores

* **ci:** remove daily schedule on renovate config ([#166](https://github.com/baptistegh/go-lakekeeper/issues/166)) ([d37272b](https://github.com/baptistegh/go-lakekeeper/commit/d37272bd322b25ba621e3f4f45ae6540509ebb80))
* **ci:** set up renovate ([#163](https://github.com/baptistegh/go-lakekeeper/issues/163)) ([cf6078c](https://github.com/baptistegh/go-lakekeeper/commit/cf6078c406747b02043a852856b628d895cc3c51))
* **ci:** update goreleaser configuration ([#184](https://github.com/baptistegh/go-lakekeeper/issues/184)) ([d15ee51](https://github.com/baptistegh/go-lakekeeper/commit/d15ee51033332b605ac702f8e8652b32e3d8b596))
* **ci:** use `latest` version of golangci-lint ([#183](https://github.com/baptistegh/go-lakekeeper/issues/183)) ([6a26f9f](https://github.com/baptistegh/go-lakekeeper/commit/6a26f9f837b6a89c2b5c5a122273cfad4a220ed2))
* **config:** migrate Renovate config ([#172](https://github.com/baptistegh/go-lakekeeper/issues/172)) ([7bc83cb](https://github.com/baptistegh/go-lakekeeper/commit/7bc83cb748f334836ebf229b86b2a259791be9a0))
* **deps:** bump github.com/apache/iceberg-go from 0.3.0 to 0.4.0 ([#147](https://github.com/baptistegh/go-lakekeeper/issues/147)) ([98b9ef4](https://github.com/baptistegh/go-lakekeeper/commit/98b9ef453f52fa2c31220f07271b44af110c3488))
* **deps:** bump golang.org/x/crypto from 0.42.0 to 0.45.0 in the go_modules group across 1 directory ([#177](https://github.com/baptistegh/go-lakekeeper/issues/177)) ([495ef25](https://github.com/baptistegh/go-lakekeeper/commit/495ef2580fa6c26755ac6bf049783319eaa36426))
* **deps:** bump golang.org/x/oauth2 from 0.31.0 to 0.32.0 ([#146](https://github.com/baptistegh/go-lakekeeper/issues/146)) ([ece634e](https://github.com/baptistegh/go-lakekeeper/commit/ece634e7559c05e72e277f5d95a69329484b9fa1))
* **deps:** bump golang.org/x/oauth2 from 0.32.0 to 0.33.0 ([#150](https://github.com/baptistegh/go-lakekeeper/issues/150)) ([961f61a](https://github.com/baptistegh/go-lakekeeper/commit/961f61a71164d0777f8418d0aaf721d006de57f1))
* **deps:** bump golangci/golangci-lint-action from 8.0.0 to 9.0.0 in the github-actions group ([#149](https://github.com/baptistegh/go-lakekeeper/issues/149)) ([7ba84a0](https://github.com/baptistegh/go-lakekeeper/commit/7ba84a018920096b792a156e3a6cb8e67fc45db6))
* **deps:** update actions/checkout action to v6 ([#179](https://github.com/baptistegh/go-lakekeeper/issues/179)) ([2cd360c](https://github.com/baptistegh/go-lakekeeper/commit/2cd360ca302d21cf060582caf412d737e88b7c69))
* **deps:** update all non-major dependencies ([df084fc](https://github.com/baptistegh/go-lakekeeper/commit/df084fcba13a369578f7ed4b24b45e49aa93b028))
* **deps:** update all non-major dependencies ([95db8cb](https://github.com/baptistegh/go-lakekeeper/commit/95db8cb086609791a7ba8986e083dceaafcd5d67))
* **deps:** update all non-major dependencies (minor) ([#168](https://github.com/baptistegh/go-lakekeeper/issues/168)) ([df084fc](https://github.com/baptistegh/go-lakekeeper/commit/df084fcba13a369578f7ed4b24b45e49aa93b028))
* **deps:** update all non-major dependencies (minor) ([#178](https://github.com/baptistegh/go-lakekeeper/issues/178)) ([d37e55a](https://github.com/baptistegh/go-lakekeeper/commit/d37e55a78c554e0c717d1e20ece5524ed9f5193c))
* **deps:** update all non-major dependencies (patch) ([#167](https://github.com/baptistegh/go-lakekeeper/issues/167)) ([95db8cb](https://github.com/baptistegh/go-lakekeeper/commit/95db8cb086609791a7ba8986e083dceaafcd5d67))
* **deps:** update dependency mkdocs-material to v9 ([#159](https://github.com/baptistegh/go-lakekeeper/issues/159)) ([64af7d1](https://github.com/baptistegh/go-lakekeeper/commit/64af7d1e07998e84a4a6fc41a456d4383bb5dc95))
* **deps:** update dependency ubuntu to v24 ([#160](https://github.com/baptistegh/go-lakekeeper/issues/160)) ([11ac251](https://github.com/baptistegh/go-lakekeeper/commit/11ac251bb114d26967a1ad729e2e72e993d50f8c))
* **deps:** update go-version ([#175](https://github.com/baptistegh/go-lakekeeper/issues/175)) ([b4d6bb2](https://github.com/baptistegh/go-lakekeeper/commit/b4d6bb25d9f7bb52c22e3184241300dc8d49436e))
* **deps:** update postgres docker tag to v18 ([#161](https://github.com/baptistegh/go-lakekeeper/issues/161)) ([6e3d99e](https://github.com/baptistegh/go-lakekeeper/commit/6e3d99e1b61811f88df2b08e57aa8647c17088c2))
* **renovate:** not grouping go toolchain updates ([#169](https://github.com/baptistegh/go-lakekeeper/issues/169)) ([9414097](https://github.com/baptistegh/go-lakekeeper/commit/941409787329c4904413e28f4de3d9b6fb7017df))
* **renovate:** remove grouping dependencies ([#180](https://github.com/baptistegh/go-lakekeeper/issues/180)) ([6a09a93](https://github.com/baptistegh/go-lakekeeper/commit/6a09a93b8a282c02a17fdb7422615b5d8aefe3b1))
* **renovate:** try gomod grouName ([#173](https://github.com/baptistegh/go-lakekeeper/issues/173)) ([9563c73](https://github.com/baptistegh/go-lakekeeper/commit/9563c733711d8c140b5b96eef66ee207d9c7e32a))
* **renovate:** Update configuration ([#171](https://github.com/baptistegh/go-lakekeeper/issues/171)) ([bcf8dfb](https://github.com/baptistegh/go-lakekeeper/commit/bcf8dfb1aeece94a461a6153a900a340dadac110))

## [0.0.19](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.18...v0.0.19) (2025-10-02)


### Features

* **warehouse:** add new actions `get_all_tasks` and `control_all_tasks` ([#143](https://github.com/baptistegh/go-lakekeeper/issues/143)) ([acab155](https://github.com/baptistegh/go-lakekeeper/commit/acab15570352548da7d033f329b9d762b0a70f7b))


### Miscellaneous Chores

* **ci:** use lakekeeper v0.10.0 ([#144](https://github.com/baptistegh/go-lakekeeper/issues/144)) ([0ae88f2](https://github.com/baptistegh/go-lakekeeper/commit/0ae88f22ba1a8de82d040f2a0205203d4d97f04e))

## [0.0.18](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.17...v0.0.18) (2025-09-18)


### Miscellaneous Chores

* **ci:** remove lock workflow ([#134](https://github.com/baptistegh/go-lakekeeper/issues/134)) ([db69bb1](https://github.com/baptistegh/go-lakekeeper/commit/db69bb1ae160c523e2531db6dd1016b762581a29))
* **deps:** bump actions/checkout from 4 to 5 in the github-actions group ([#132](https://github.com/baptistegh/go-lakekeeper/issues/132)) ([5ec4f2c](https://github.com/baptistegh/go-lakekeeper/commit/5ec4f2c875f8cc4402a68b2ebe7badfc79053299))
* **deps:** bump github.com/go-viper/mapstructure/v2 from 2.3.0 to 2.4.0 in the go_modules group ([#135](https://github.com/baptistegh/go-lakekeeper/issues/135)) ([d0400a9](https://github.com/baptistegh/go-lakekeeper/commit/d0400a9acec2b9ed16d20e3202206aca122a3c7f))
* **deps:** bump github.com/spf13/cobra from 1.9.1 to 1.10.1 ([#138](https://github.com/baptistegh/go-lakekeeper/issues/138)) ([15bcbc3](https://github.com/baptistegh/go-lakekeeper/commit/15bcbc3073a8e4a4b1d3c19d4708154858175b37))
* **deps:** bump github.com/stretchr/testify from 1.10.0 to 1.11.0 ([#136](https://github.com/baptistegh/go-lakekeeper/issues/136)) ([1f94fb8](https://github.com/baptistegh/go-lakekeeper/commit/1f94fb87408ed55c2a9de43222dcfa4835f2e10e))
* **deps:** bump github.com/stretchr/testify from 1.11.0 to 1.11.1 ([#137](https://github.com/baptistegh/go-lakekeeper/issues/137)) ([5f1f15f](https://github.com/baptistegh/go-lakekeeper/commit/5f1f15f747759f3fa3517abfb6c7477e1659165e))
* **deps:** bump golang.org/x/oauth2 from 0.30.0 to 0.31.0 ([#140](https://github.com/baptistegh/go-lakekeeper/issues/140)) ([b991475](https://github.com/baptistegh/go-lakekeeper/commit/b991475e0c318cd7a38123ac23a043ef3a1fbe7e))
* **deps:** bump the github-actions group with 2 updates ([#139](https://github.com/baptistegh/go-lakekeeper/issues/139)) ([099c378](https://github.com/baptistegh/go-lakekeeper/commit/099c378c18cfcc9aac0c07a5a0c668decd542af4))
* remove bitnami postgresql image ([#142](https://github.com/baptistegh/go-lakekeeper/issues/142)) ([dc5881f](https://github.com/baptistegh/go-lakekeeper/commit/dc5881fabd457414a711788efb3f900f50182261))

## [0.0.17](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.16...v0.0.17) (2025-08-05)


### Bug Fixes

* **warehouse:** rename remote signing url styles for s3 storage profile ([#130](https://github.com/baptistegh/go-lakekeeper/issues/130)) ([82f30bf](https://github.com/baptistegh/go-lakekeeper/commit/82f30bf3d10d391dd95d5352d84085ea193a7e96))

## [0.0.16](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.15...v0.0.16) (2025-08-01)


### Bug Fixes

* **cli:** project was not used in role/warehouse commands ([#128](https://github.com/baptistegh/go-lakekeeper/issues/128)) ([6251582](https://github.com/baptistegh/go-lakekeeper/commit/6251582c18402f455aa71ab2f1b31981f1867251))

## [0.0.15](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.14...v0.0.15) (2025-08-01)


### Features

* **cli:** add role assignments add command ([#118](https://github.com/baptistegh/go-lakekeeper/issues/118)) ([ad35389](https://github.com/baptistegh/go-lakekeeper/commit/ad353898461062c947bf30d534fd260169390959))
* **cli:** add server permissions-related commands ([#126](https://github.com/baptistegh/go-lakekeeper/issues/126)) ([dc5adc0](https://github.com/baptistegh/go-lakekeeper/commit/dc5adc03cd374da3571df655175119ce965545d8))
* **cli:** introduction of tab writer ([#124](https://github.com/baptistegh/go-lakekeeper/issues/124)) ([c1eb5ac](https://github.com/baptistegh/go-lakekeeper/commit/c1eb5ac66fd4c9411b59a478c577834d61346322))
* **cli:** rename project asssignments update command to add ([#119](https://github.com/baptistegh/go-lakekeeper/issues/119)) ([91c8d22](https://github.com/baptistegh/go-lakekeeper/commit/91c8d22f11e208281503f9b339e66c329af03566))
* **cli:** warehouse commands add/delete/list ([#121](https://github.com/baptistegh/go-lakekeeper/issues/121)) ([73c5879](https://github.com/baptistegh/go-lakekeeper/commit/73c5879d57c5ae1e265716ef32ab1ef8215d968c))


### Bug Fixes

* **cli:** no authentication on version command ([#113](https://github.com/baptistegh/go-lakekeeper/issues/113)) ([d5687de](https://github.com/baptistegh/go-lakekeeper/commit/d5687de8f48a6bd2941b1ce93a51c0700aaf9fee))


### Documentation

* generate CLI documentation ([#127](https://github.com/baptistegh/go-lakekeeper/issues/127)) ([0610765](https://github.com/baptistegh/go-lakekeeper/commit/0610765ea2b227c4e55b37bda97987c19c47a4b0))


### Miscellaneous Chores

* **ci:** Add PR title checker ([#123](https://github.com/baptistegh/go-lakekeeper/issues/123)) ([8ca0ca9](https://github.com/baptistegh/go-lakekeeper/commit/8ca0ca9636f6cec60bdd7df11d46ca5ab343b0ae))
* **ci:** fix lint CLI add warehouse command ([#122](https://github.com/baptistegh/go-lakekeeper/issues/122)) ([91b7cb9](https://github.com/baptistegh/go-lakekeeper/commit/91b7cb9bf8b54824e372352f17f1d0de053ce0d0))
* **ci:** rename published binaries ([#117](https://github.com/baptistegh/go-lakekeeper/issues/117)) ([a1e5f52](https://github.com/baptistegh/go-lakekeeper/commit/a1e5f52c18dfbcf9546b6145d22db5efce73b560))
* **ci:** set docs label on docs/** change ([#125](https://github.com/baptistegh/go-lakekeeper/issues/125)) ([b06c2a1](https://github.com/baptistegh/go-lakekeeper/commit/b06c2a1180fd29cd80368885d224e4d9113bd78a))
* **docs:** add a table of contents in README.me ([#116](https://github.com/baptistegh/go-lakekeeper/issues/116)) ([486f4c9](https://github.com/baptistegh/go-lakekeeper/commit/486f4c994e24886554a806c030948d7bda908820))
* **docs:** add CLI examples ([#120](https://github.com/baptistegh/go-lakekeeper/issues/120)) ([ed6d451](https://github.com/baptistegh/go-lakekeeper/commit/ed6d45163fb99d167afb552829090e61b5ab405d))
* **docs:** add CLI usage in README.md ([#114](https://github.com/baptistegh/go-lakekeeper/issues/114)) ([6844b14](https://github.com/baptistegh/go-lakekeeper/commit/6844b14cfd06a3c231dcffc24ba81d85dacfde61))
* **docs:** replace nightly badge in README.md ([#112](https://github.com/baptistegh/go-lakekeeper/issues/112)) ([5aed91f](https://github.com/baptistegh/go-lakekeeper/commit/5aed91f451c82acdebb56cff6b6285f2d44cade9))

## [0.0.14](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.13...v0.0.14) (2025-07-30)


### Miscellaneous Chores

* fix goreleaser release repo name ([#110](https://github.com/baptistegh/go-lakekeeper/issues/110)) ([320e29a](https://github.com/baptistegh/go-lakekeeper/commit/320e29ae9a2567ed7e154d67b2852bff47392eef))

## [0.0.13](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.12...v0.0.13) (2025-07-30)


### Miscellaneous Chores

* fix publish container image on release ([#108](https://github.com/baptistegh/go-lakekeeper/issues/108)) ([ace86ef](https://github.com/baptistegh/go-lakekeeper/commit/ace86efdbc04cf5afd5752f036ffb0d6710c3af7))

## [0.0.12](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.11...v0.0.12) (2025-07-30)


### Features

* **cli:** introduction of the command line interface ([#103](https://github.com/baptistegh/go-lakekeeper/issues/103)) ([7133351](https://github.com/baptistegh/go-lakekeeper/commit/7133351991a341a31618d9c5ada998f8a2e410a1))
* **test:** add client options tests ([#99](https://github.com/baptistegh/go-lakekeeper/issues/99)) ([08d7779](https://github.com/baptistegh/go-lakekeeper/commit/08d777929a585641aeb978eddd2b763896af290e))


### Bug Fixes

* **warehouse:** filter by status ([#102](https://github.com/baptistegh/go-lakekeeper/issues/102)) ([a97ff1e](https://github.com/baptistegh/go-lakekeeper/commit/a97ff1e904951b3476d67b78e4724a6dc0cc73bb))


### Miscellaneous Chores

* add status badges in README.md ([#98](https://github.com/baptistegh/go-lakekeeper/issues/98)) ([15b9850](https://github.com/baptistegh/go-lakekeeper/commit/15b98504727ef31025e6b72f20349f53b0d55832))
* **build:** set go version to 1.24 ([#101](https://github.com/baptistegh/go-lakekeeper/issues/101)) ([21cf182](https://github.com/baptistegh/go-lakekeeper/commit/21cf182758e89c93f1873b0e03ca91589a4bd10a))
* **ci:** publish container image on main branch ([#106](https://github.com/baptistegh/go-lakekeeper/issues/106)) ([62e20ff](https://github.com/baptistegh/go-lakekeeper/commit/62e20ffab931d331804f60e3620cd6c9d83b29bc))
* **deps:** bump github.com/go-viper/mapstructure/v2 ([f6a6bc7](https://github.com/baptistegh/go-lakekeeper/commit/f6a6bc7d1ecc51078645ba3312f1d3bf41faace1))
* **deps:** bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in the go_modules group ([#105](https://github.com/baptistegh/go-lakekeeper/issues/105)) ([f6a6bc7](https://github.com/baptistegh/go-lakekeeper/commit/f6a6bc7d1ecc51078645ba3312f1d3bf41faace1))
* **deps:** bump the github-actions group with 2 updates ([#104](https://github.com/baptistegh/go-lakekeeper/issues/104)) ([914b439](https://github.com/baptistegh/go-lakekeeper/commit/914b4394defa652f3cd31ad331365d5072bb67bd))
* set up release please sections ([#107](https://github.com/baptistegh/go-lakekeeper/issues/107)) ([2c04c77](https://github.com/baptistegh/go-lakekeeper/commit/2c04c778c7b64d675c2349e81732aa0bac33425a))

## [0.0.11](https://github.com/baptistegh/go-lakekeeper/compare/v0.0.10...v0.0.11) (2025-07-21)


### ⚠ BREAKING CHANGES

* add explicit context argument to all API methods ([#92](https://github.com/baptistegh/go-lakekeeper/issues/92))

### Features

* add explicit context argument to all API methods ([#92](https://github.com/baptistegh/go-lakekeeper/issues/92)) ([7eb0818](https://github.com/baptistegh/go-lakekeeper/commit/7eb0818a1b6cfe90a766be3ad842ff8b1d5827a1))
* add integration with go-iceberg for catalog endpoints ([#89](https://github.com/baptistegh/go-lakekeeper/issues/89)) ([553afcb](https://github.com/baptistegh/go-lakekeeper/commit/553afcbfc4b30966ee0f4a5b1dd3be53e96d0ef2))
* **warehouse:** add deprecation notice for GetProtection ([#96](https://github.com/baptistegh/go-lakekeeper/issues/96)) ([df774ba](https://github.com/baptistegh/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add few missing methods ([#94](https://github.com/baptistegh/go-lakekeeper/issues/94)) ([20e080b](https://github.com/baptistegh/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add get statistics ([#95](https://github.com/baptistegh/go-lakekeeper/issues/95)) ([cc8ecff](https://github.com/baptistegh/go-lakekeeper/commit/cc8ecffc5a3ba428e8c81a91b1a1678c1aa80be2))
* **warehouse:** add GetNamespaceProtection ([#94](https://github.com/baptistegh/go-lakekeeper/issues/94)) ([20e080b](https://github.com/baptistegh/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add GetTableProtection method ([#96](https://github.com/baptistegh/go-lakekeeper/issues/96)) ([df774ba](https://github.com/baptistegh/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add GetViewProtection method ([#96](https://github.com/baptistegh/go-lakekeeper/issues/96)) ([df774ba](https://github.com/baptistegh/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add ListSoftDeletedTabular ([#94](https://github.com/baptistegh/go-lakekeeper/issues/94)) ([20e080b](https://github.com/baptistegh/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add SetNamespaceProtection ([#94](https://github.com/baptistegh/go-lakekeeper/issues/94)) ([20e080b](https://github.com/baptistegh/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))
* **warehouse:** add SetTableProtection method ([#96](https://github.com/baptistegh/go-lakekeeper/issues/96)) ([df774ba](https://github.com/baptistegh/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add SetViewProtection method ([#96](https://github.com/baptistegh/go-lakekeeper/issues/96)) ([df774ba](https://github.com/baptistegh/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add table and view protection methods ([#96](https://github.com/baptistegh/go-lakekeeper/issues/96)) ([df774ba](https://github.com/baptistegh/go-lakekeeper/commit/df774baaac5af01e8514d529523daddb00cd4835))
* **warehouse:** add UndropTabular ([#94](https://github.com/baptistegh/go-lakekeeper/issues/94)) ([20e080b](https://github.com/baptistegh/go-lakekeeper/commit/20e080b70cd32600c4744711ce472f89447888c8))


### Miscellaneous Chores

* prepare release 0.0.11 ([afa161a](https://github.com/baptistegh/go-lakekeeper/commit/afa161a43e419f61143ef8c5e92c46035ae5d437))

## 0.0.10 (2025-07-19)

<!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### 🎉 Features
* feat(permission): remove project scope on warehouse by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/87


**Full Changelog**: https://github.com/baptistegh/go-lakekeeper/compare/v0.0.9...v0.0.10

## 0.0.9 (2025-07-18)

<!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### 🎉 Features
* feat: add control on bootstrap user role by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/82
* feat(permission): add warehouse interfaces by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/85
* feat(permission): add missing GetAccess on role by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/86
### Other Changes
* chore(ci): add v0.9.3 support by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/80


**Full Changelog**: https://github.com/baptistegh/go-lakekeeper/compare/v0.0.8...v0.0.9

## 0.0.8 (2025-07-17)

<!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### 🎉 Features
* feat(permission): add role interfaces by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/78


**Full Changelog**: https://github.com/baptistegh/go-lakekeeper/compare/v0.0.7...v0.0.8

## 0.0.7 (2025-07-16)

<!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### 🎉 Features
* feat(permission): implement server permissions interfaces by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/52
* feat(permissions): add filtering support to server get access endpoint by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/69
* feat(permission): add project interface support by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/75
* feat(project): add get api statistics endpoint support by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/70
### ✅ Bug Fixes
* fix(permission): rename all project related objects in server by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/74
### 📚 Documentation
* chore: clean CHANGELOG.md by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/50
### Other Changes
* chore: DRY in integration tests by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/76


**Full Changelog**: https://github.com/baptistegh/go-lakekeeper/compare/v0.0.6...v0.0.7

## 0.0.6 (2025-07-15)

<!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Other Changes
* chore(release-please): fix previous tag by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/46
* chore(release-please): rework v0.0.0 by @baptistegh in https://github.com/baptistegh/go-lakekeeper/pull/48


**Full Changelog**: https://github.com/baptistegh/go-lakekeeper/commits/v0.0.6
