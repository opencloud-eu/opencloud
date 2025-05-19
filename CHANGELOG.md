# Changelog

## [2.3.0](https://github.com/opencloud-eu/opencloud/releases/tag/v2.3.0) - 2025-05-19

### ❤️ Thanks to all contributors! ❤️

@AlexAndBear, @ScharfViktor, @aduffeck, @butonic, @micbar, @rhafer

### ✨ Features

- deployment: Adapt opencloud_full to include radicale [[#773](https://github.com/opencloud-eu/opencloud/pull/773)]
- proxy(router): Allow to set some outgoing headers [[#756](https://github.com/opencloud-eu/opencloud/pull/756)]
- feat: set idp logo defaul url [[#746](https://github.com/opencloud-eu/opencloud/pull/746)]

### 📈 Enhancement

- Reduce load caused by the activitylog service [[#842](https://github.com/opencloud-eu/opencloud/pull/842)]

### ✅ Tests

- PosixTest. Check that version, share and link still exist [[#837](https://github.com/opencloud-eu/opencloud/pull/837)]
- [test-only] test for #452 [[#826](https://github.com/opencloud-eu/opencloud/pull/826)]
- collaboration posix tests [[#780](https://github.com/opencloud-eu/opencloud/pull/780)]
- collaborative posix test [[#672](https://github.com/opencloud-eu/opencloud/pull/672)]

### 🐛 Bug Fixes

- nats: Don't enable debug and trace logging by default [[#825](https://github.com/opencloud-eu/opencloud/pull/825)]
- fix: show special roles at the end of the list [[#806](https://github.com/opencloud-eu/opencloud/pull/806)]
- fix: idp login logo url exceeds logo [[#742](https://github.com/opencloud-eu/opencloud/pull/742)]

### 📦️ Dependencies

- [full-ci] chore(web): bump web to v2.3.0 [[#885](https://github.com/opencloud-eu/opencloud/pull/885)]
- chore:reva bump v.2.33 [[#884](https://github.com/opencloud-eu/opencloud/pull/884)]
- build(deps): bump google.golang.org/grpc from 1.72.0 to 1.72.1 [[#862](https://github.com/opencloud-eu/opencloud/pull/862)]
- build(deps): bump golang.org/x/net from 0.39.0 to 0.40.0 [[#855](https://github.com/opencloud-eu/opencloud/pull/855)]
- build(deps-dev): bump dotenv-expand from 10.0.0 to 12.0.2 in /services/idp [[#831](https://github.com/opencloud-eu/opencloud/pull/831)]
- build(deps): bump github.com/libregraph/lico from 0.65.2-0.20250428103211-356e98f98457 to 0.66.0 [[#839](https://github.com/opencloud-eu/opencloud/pull/839)]
- build(deps): bump i18next from 23.16.8 to 25.1.2 in /services/idp [[#832](https://github.com/opencloud-eu/opencloud/pull/832)]
- build(deps): bump dario.cat/mergo from 1.0.1 to 1.0.2 [[#829](https://github.com/opencloud-eu/opencloud/pull/829)]
- build(deps): bump golang.org/x/image from 0.26.0 to 0.27.0 [[#817](https://github.com/opencloud-eu/opencloud/pull/817)]
- build(deps): bump github.com/CiscoM31/godata from 1.0.10 to 1.0.11 [[#815](https://github.com/opencloud-eu/opencloud/pull/815)]
- build(deps): bump github.com/KimMachineGun/automemlimit from 0.7.1 to 0.7.2 [[#803](https://github.com/opencloud-eu/opencloud/pull/803)]
- build(deps): bump golang.org/x/crypto from 0.37.0 to 0.38.0 [[#802](https://github.com/opencloud-eu/opencloud/pull/802)]
- build(deps): bump github.com/open-policy-agent/opa from 1.3.0 to 1.4.2 [[#784](https://github.com/opencloud-eu/opencloud/pull/784)]
- build(deps): bump golang.org/x/sync from 0.13.0 to 0.14.0 [[#785](https://github.com/opencloud-eu/opencloud/pull/785)]
- build(deps-dev): bump eslint-plugin-import from 2.30.0 to 2.31.0 in /services/idp [[#777](https://github.com/opencloud-eu/opencloud/pull/777)]
- build(deps): bump github.com/nats-io/nats.go from 1.41.2 to 1.42.0 [[#776](https://github.com/opencloud-eu/opencloud/pull/776)]
- build(deps): bump golang.org/x/oauth2 from 0.29.0 to 0.30.0 [[#775](https://github.com/opencloud-eu/opencloud/pull/775)]
- build(deps): bump i18next-http-backend from 2.5.2 to 3.0.2 in /services/idp [[#774](https://github.com/opencloud-eu/opencloud/pull/774)]
- build(deps): bump github.com/beevik/etree from 1.5.0 to 1.5.1 [[#759](https://github.com/opencloud-eu/opencloud/pull/759)]
- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.2 to 2.11.3 [[#762](https://github.com/opencloud-eu/opencloud/pull/762)]
- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.1 to 2.11.2 [[#754](https://github.com/opencloud-eu/opencloud/pull/754)]
- build(deps): bump github.com/gookit/config/v2 from 2.2.5 to 2.2.6 [[#753](https://github.com/opencloud-eu/opencloud/pull/753)]
- build(deps-dev): bump css-loader from 5.2.7 to 7.1.2 in /services/idp [[#740](https://github.com/opencloud-eu/opencloud/pull/740)]
- build(deps): bump react-i18next from 15.1.1 to 15.5.1 in /services/idp [[#741](https://github.com/opencloud-eu/opencloud/pull/741)]
- build(deps): bump github.com/blevesearch/bleve/v2 from 2.4.4 to 2.5.0 [[#743](https://github.com/opencloud-eu/opencloud/pull/743)]
- build(deps): bump github.com/gabriel-vasile/mimetype from 1.4.8 to 1.4.9 [[#744](https://github.com/opencloud-eu/opencloud/pull/744)]

## [2.2.0](https://github.com/opencloud-eu/opencloud/releases/tag/v2.2.0) - 2025-04-28

### ❤️ Thanks to all contributors! ❤️

@AlexAndBear, @JammingBen, @ScharfViktor, @Svanvith, @TheOneRing, @aduffeck, @amrita-shrestha, @butonic, @dragonchaser, @dragotin, @fschade, @individual-it, @jnweiger, @micbar, @michaelstingl, @rhafer

### ✨ Features

- add new property IdentifierDefaultLogoTargetURI [[#684](https://github.com/opencloud-eu/opencloud/pull/684)]
- feat: add dev docs for web [[#623](https://github.com/opencloud-eu/opencloud/pull/623)]
- feat: improve the info about storage path in deployment example [[#617](https://github.com/opencloud-eu/opencloud/pull/617)]

### 📈 Enhancement

- [full-ci] chore(web): bump web to v2.3.0 [[#738](https://github.com/opencloud-eu/opencloud/pull/738)]
- bare-metal-deploy. getting latest version [[#699](https://github.com/opencloud-eu/opencloud/pull/699)]
- Automatically find the latest released version of opencloud [[#687](https://github.com/opencloud-eu/opencloud/pull/687)]
- Expose more config vars for the posix fs watchers [[#669](https://github.com/opencloud-eu/opencloud/pull/669)]
- Add env var to make the inotify stats frequency configurable [[#552](https://github.com/opencloud-eu/opencloud/pull/552)]
- feat(web): remove old and unused color tokens [[#665](https://github.com/opencloud-eu/opencloud/pull/665)]
- Feat: install.sh now honors OC_BASE_DIR and OC_HOST [[#574](https://github.com/opencloud-eu/opencloud/pull/574)]
- revert: completely remove "edition" from capabilities [[#601](https://github.com/opencloud-eu/opencloud/pull/601)]

### 📚 Documentation

- Update descirption of COLLABORA_SSL_ENABLE [[#724](https://github.com/opencloud-eu/opencloud/pull/724)]
- Fix broken links in opencloud_full README.md [[#643](https://github.com/opencloud-eu/opencloud/pull/643)]
- chore: move dev docs to opencloud-eu/docs repo [[#635](https://github.com/opencloud-eu/opencloud/pull/635)]

### 🐛 Bug Fixes

- Makefile: fix protobuf dependencies [[#714](https://github.com/opencloud-eu/opencloud/pull/714)]
- Some smaller Makefile adjustments [[#709](https://github.com/opencloud-eu/opencloud/pull/709)]
- fix(decomposeds3): enable async-uploads by default [[#686](https://github.com/opencloud-eu/opencloud/pull/686)]
- fix deployment: do not create demo accounts when using keycloak [[#671](https://github.com/opencloud-eu/opencloud/pull/671)]
- fix: web dev docs broken links [[#633](https://github.com/opencloud-eu/opencloud/pull/633)]
- fix inbucket setup [[#619](https://github.com/opencloud-eu/opencloud/pull/619)]

### ✅ Tests

- update test docs [[#652](https://github.com/opencloud-eu/opencloud/pull/652)]

### 📦️ Dependencies

- chore:reva bump v.2.32 [[#737](https://github.com/opencloud-eu/opencloud/pull/737)]
- build(deps): bump golang.org/x/image from 0.25.0 to 0.26.0 [[#726](https://github.com/opencloud-eu/opencloud/pull/726)]
- build(deps): bump golang.org/x/net from 0.38.0 to 0.39.0 [[#725](https://github.com/opencloud-eu/opencloud/pull/725)]
- build(deps): bump github.com/nats-io/nats.go from 1.41.0 to 1.41.2 [[#722](https://github.com/opencloud-eu/opencloud/pull/722)]
- build(deps): bump google.golang.org/grpc from 1.71.1 to 1.72.0 [[#721](https://github.com/opencloud-eu/opencloud/pull/721)]
- build(deps): bump golang.org/x/oauth2 from 0.28.0 to 0.29.0 [[#602](https://github.com/opencloud-eu/opencloud/pull/602)]
- build(deps): bump @testing-library/jest-dom from 6.4.8 to 6.6.3 in /services/idp [[#666](https://github.com/opencloud-eu/opencloud/pull/666)]
- build(deps): bump golang.org/x/text from 0.23.0 to 0.24.0 [[#641](https://github.com/opencloud-eu/opencloud/pull/641)]
- build(deps-dev): bump webpack from 5.96.1 to 5.99.6 in /services/idp [[#707](https://github.com/opencloud-eu/opencloud/pull/707)]
- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.0 to 2.11.1 [[#679](https://github.com/opencloud-eu/opencloud/pull/679)]
- build(deps): bump github.com/onsi/ginkgo/v2 from 2.23.3 to 2.23.4 [[#637](https://github.com/opencloud-eu/opencloud/pull/637)]
- build(deps): bump github.com/coreos/go-oidc/v3 from 3.13.0 to 3.14.1 [[#603](https://github.com/opencloud-eu/opencloud/pull/603)]
- build(deps-dev): bump typescript from 5.7.3 to 5.8.3 in /services/idp [[#604](https://github.com/opencloud-eu/opencloud/pull/604)]

## [2.1.0](https://github.com/opencloud-eu/opencloud/releases/tag/v2.1.0) - 2025-04-07

### ❤️ Thanks to all contributors! ❤️

@AlexAndBear, @JammingBen, @ScharfViktor, @aduffeck, @butonic, @fschade, @individual-it, @kulmann, @micbar, @michaelstingl, @rhafer

### 🐛 Bug Fixes

- feat(antivirus): add partial scanning mode [[#559](https://github.com/opencloud-eu/opencloud/pull/559)]
- Simplify item-trashed SSEs. Also fixes it for coll. posix fs. [[#565](https://github.com/opencloud-eu/opencloud/pull/565)]
- fix(opencloud_full): add missing SMTP env vars [[#563](https://github.com/opencloud-eu/opencloud/pull/563)]
- fix: full deployment tika description is wrong [[#553](https://github.com/opencloud-eu/opencloud/pull/553)]
- fix: traefik credentials [[#555](https://github.com/opencloud-eu/opencloud/pull/555)]
- Enable scan/watch in the storageprovider only [[#546](https://github.com/opencloud-eu/opencloud/pull/546)]
- fix: typo in dev docs [[#540](https://github.com/opencloud-eu/opencloud/pull/540)]

### 📈 Enhancement

- [full-ci] reva bump 2.31.0 [[#599](https://github.com/opencloud-eu/opencloud/pull/599)]
- feat: support svg as icon [[#538](https://github.com/opencloud-eu/opencloud/pull/538)]
- feat: change theme.json primary color [[#536](https://github.com/opencloud-eu/opencloud/pull/536)]
- graph: reduce memory allocations [[#494](https://github.com/opencloud-eu/opencloud/pull/494)]

### ✅ Tests

- [full-ci] fix expected spanish string in test [[#596](https://github.com/opencloud-eu/opencloud/pull/596)]
- Revert "Disable the 'exclude' patterns on the path conditional for now" [[#561](https://github.com/opencloud-eu/opencloud/pull/561)]

### 📦️ Dependencies

- build(deps): bump github.com/go-playground/validator/v10 from 10.25.0 to 10.26.0 [[#571](https://github.com/opencloud-eu/opencloud/pull/571)]
- build(deps): bump github.com/nats-io/nats.go from 1.39.1 to 1.41.0 [[#567](https://github.com/opencloud-eu/opencloud/pull/567)]
- [full-ci] chore(web): bump web to v2.2.0 [[#570](https://github.com/opencloud-eu/opencloud/pull/570)]
- build(deps): bump github.com/onsi/gomega from 1.36.3 to 1.37.0 [[#566](https://github.com/opencloud-eu/opencloud/pull/566)]
- build(deps): bump golang.org/x/net from 0.37.0 to 0.38.0 [[#557](https://github.com/opencloud-eu/opencloud/pull/557)]
- build(deps-dev): bump eslint-plugin-jsx-a11y from 6.9.0 to 6.10.2 in /services/idp [[#542](https://github.com/opencloud-eu/opencloud/pull/542)]
- build(deps): bump web-vitals from 3.5.2 to 4.2.4 in /services/idp [[#541](https://github.com/opencloud-eu/opencloud/pull/541)]
- build(deps): bump github.com/open-policy-agent/opa from 1.2.0 to 1.3.0 [[#508](https://github.com/opencloud-eu/opencloud/pull/508)]
- build(deps): bump github.com/urfave/cli/v2 from 2.27.5 to 2.27.6 [[#509](https://github.com/opencloud-eu/opencloud/pull/509)]
- fix keycloak example #465 [[#535](https://github.com/opencloud-eu/opencloud/pull/535)]

## [2.0.0](https://github.com/opencloud-eu/opencloud/releases/tag/v2.0.0) - 2025-03-26

### ❤️ Thanks to all contributors! ❤️

@JammingBen, @ScharfViktor, @aduffeck, @amrita-shrestha, @butonic, @dragonchaser, @dragotin, @individual-it, @kulmann, @micbar, @prashant-gurung899, @rhafer

### 💥 Breaking changes

- [posix] change storage users default to posixfs [[#237](https://github.com/opencloud-eu/opencloud/pull/237)]

### 🐛 Bug Fixes

- Bump reva to 2.29.1 [[#501](https://github.com/opencloud-eu/opencloud/pull/501)]
- remove workaround for translation formatting [[#491](https://github.com/opencloud-eu/opencloud/pull/491)]
- [full-ci] fix(collaboration): hide SaveAs and ExportAs buttons in web office [[#471](https://github.com/opencloud-eu/opencloud/pull/471)]
- fix: add missing debug docker [[#481](https://github.com/opencloud-eu/opencloud/pull/481)]
- Downgrade nats.go to 1.39.1 [[#479](https://github.com/opencloud-eu/opencloud/pull/479)]
-  fix cli driver initialization for "posix"  [[#459](https://github.com/opencloud-eu/opencloud/pull/459)]
- Do not cache when there was an error gathering the data [[#462](https://github.com/opencloud-eu/opencloud/pull/462)]
- fix(storage-users): 'uploads sessions' command crash [[#446](https://github.com/opencloud-eu/opencloud/pull/446)]
- fix: org name in multiarch dev build [[#431](https://github.com/opencloud-eu/opencloud/pull/431)]
- fix local setup [[#440](https://github.com/opencloud-eu/opencloud/pull/440)]

### 📈 Enhancement

- [full-ci] chore(web): update web to v2.1.0 [[#497](https://github.com/opencloud-eu/opencloud/pull/497)]
- Bump reva [[#474](https://github.com/opencloud-eu/opencloud/pull/474)]
- Bump reva to pull in the latest fixes [[#451](https://github.com/opencloud-eu/opencloud/pull/451)]
- Switch to jsoncs3 backend for app tokens and enable service by default [[#433](https://github.com/opencloud-eu/opencloud/pull/433)]
- Completely remove "edition" from capabilities [[#434](https://github.com/opencloud-eu/opencloud/pull/434)]
- feat: add post logout redirect uris for mobile clients [[#411](https://github.com/opencloud-eu/opencloud/pull/411)]
- chore: bump version to v1.1.0 [[#422](https://github.com/opencloud-eu/opencloud/pull/422)]

### ✅ Tests

- [full-ci] add one more TUS test to expected to fail file [[#489](https://github.com/opencloud-eu/opencloud/pull/489)]
- [full-ci]Remove mtime 500 issue from expected failure [[#467](https://github.com/opencloud-eu/opencloud/pull/467)]
- add auth app to ocm test setup [[#472](https://github.com/opencloud-eu/opencloud/pull/472)]
- use opencloudeu/cs3api-validator in CI [[#469](https://github.com/opencloud-eu/opencloud/pull/469)]
- fix(test): Run app-auth test with jsoncs3 backend [[#460](https://github.com/opencloud-eu/opencloud/pull/460)]
- Always run CLI tests with the decomposed storage driver [[#435](https://github.com/opencloud-eu/opencloud/pull/435)]
- Disable the 'exclude' patterns on the path conditional for now [[#439](https://github.com/opencloud-eu/opencloud/pull/439)]
- run CS3 API tests in CI [[#415](https://github.com/opencloud-eu/opencloud/pull/415)]
- fix: fix path exclusion glob patterns [[#427](https://github.com/opencloud-eu/opencloud/pull/427)]
- Cleanup woodpecker [[#430](https://github.com/opencloud-eu/opencloud/pull/430)]
- enable main API test suite to run in CI [[#419](https://github.com/opencloud-eu/opencloud/pull/419)]
- Run wopi tests in CI [[#416](https://github.com/opencloud-eu/opencloud/pull/416)]
- Run `cliCommands` tests pipeline in CI [[#413](https://github.com/opencloud-eu/opencloud/pull/413)]

### 📚 Documentation

- docs(idp): Document how to add custom OIDC clients [[#476](https://github.com/opencloud-eu/opencloud/pull/476)]
- Clean invalid documentation links [[#466](https://github.com/opencloud-eu/opencloud/pull/466)]

### 📦️ Dependencies

- build(deps): bump github.com/grpc-ecosystem/grpc-gateway/v2 from 2.26.1 to 2.26.3 [[#480](https://github.com/opencloud-eu/opencloud/pull/480)]
- chore: update alpine to 3.21 [[#483](https://github.com/opencloud-eu/opencloud/pull/483)]
- build(deps): bump github.com/nats-io/nats.go from 1.39.1 to 1.40.0 [[#464](https://github.com/opencloud-eu/opencloud/pull/464)]
- build(deps): bump github.com/spf13/afero from 1.12.0 to 1.14.0 [[#436](https://github.com/opencloud-eu/opencloud/pull/436)]
- build(deps): bump github.com/KimMachineGun/automemlimit from 0.7.0 to 0.7.1 [[#437](https://github.com/opencloud-eu/opencloud/pull/437)]
- build(deps): bump golang.org/x/image from 0.24.0 to 0.25.0 [[#426](https://github.com/opencloud-eu/opencloud/pull/426)]
- build(deps): bump go.opentelemetry.io/contrib/zpages from 0.57.0 to 0.60.0 [[#425](https://github.com/opencloud-eu/opencloud/pull/425)]
