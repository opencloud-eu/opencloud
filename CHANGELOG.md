# Changelog

## [3.5.0](https://github.com/opencloud-eu/opencloud/releases/tag/v3.5.0) - 2025-09-02

### ❤️ Thanks to all contributors! ❤️

@aduffeck, @prashant-gurung899, @rhafer

### 📈 Enhancement

- Tracing improvements [[#1436](https://github.com/opencloud-eu/opencloud/pull/1436)]

### 🐛 Bug Fixes

- Do not intertwine different batch operations [[#1317](https://github.com/opencloud-eu/opencloud/pull/1317)]

### ✅ Tests

- [full-ci] purge browsers, opencloud web and playwright tracing cache [[#1403](https://github.com/opencloud-eu/opencloud/pull/1403)]

### 📦️ Dependencies

- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.7 to 2.11.8 [[#1410](https://github.com/opencloud-eu/opencloud/pull/1410)]
- build(deps): bump github.com/gabriel-vasile/mimetype from 1.4.9 to 1.4.10 [[#1413](https://github.com/opencloud-eu/opencloud/pull/1413)]

## [3.4.0](https://github.com/opencloud-eu/opencloud/releases/tag/v3.4.0) - 2025-09-02

### ❤️ Thanks to all contributors! ❤️

@ScharfViktor, @butonic, @dragonchaser, @fschade, @individual-it, @kulmann, @pbleser-oc, @prashant-gurung899, @rhafer, @tammi-23, @tylerlm

### ✨ Features

- feat: added capability for Edit Login Allowed [[#1406](https://github.com/opencloud-eu/opencloud/pull/1406)]
- Search-service: add opensearch as distributed search backend [[#1290](https://github.com/opencloud-eu/opencloud/pull/1290)]
- initial skel for user soft delete [[#1344](https://github.com/opencloud-eu/opencloud/pull/1344)]

### 🐛 Bug Fixes

- fix(antivirus): the file bytesize differs if the file is larger than … [[#1408](https://github.com/opencloud-eu/opencloud/pull/1408)]
- Correct app store URL [[#1412](https://github.com/opencloud-eu/opencloud/pull/1412)]
- ack tag events [[#1381](https://github.com/opencloud-eu/opencloud/pull/1381)]
- fix(proxy): First login fails in auto provision setups [[#1353](https://github.com/opencloud-eu/opencloud/pull/1353)]

### 📈 Enhancement

- directly connect to frontend [[#1373](https://github.com/opencloud-eu/opencloud/pull/1373)]
- Dockerfile cleanup [[#1352](https://github.com/opencloud-eu/opencloud/pull/1352)]
- feat: add defaultAppId option for the web config.json [[#1354](https://github.com/opencloud-eu/opencloud/pull/1354)]

### ✅ Tests

- tests for collaborativePosixFS [[#1342](https://github.com/opencloud-eu/opencloud/pull/1342)]
- [full-ci] add pipeline to send CI notifications to matrix [[#1249](https://github.com/opencloud-eu/opencloud/pull/1249)]

### 📦️ Dependencies

- [decomposed] bump-version-v3.4.0 [[#1442](https://github.com/opencloud-eu/opencloud/pull/1442)]
- [full-ci] revaBump-2.37.0 [[#1433](https://github.com/opencloud-eu/opencloud/pull/1433)]
- Use bitnamilegacy [[#1418](https://github.com/opencloud-eu/opencloud/pull/1418)]
- build(deps): bump github.com/nats-io/nats.go from 1.44.0 to 1.45.0 [[#1401](https://github.com/opencloud-eu/opencloud/pull/1401)]
- build(deps): bump github.com/stretchr/testify from 1.10.0 to 1.11.0 [[#1400](https://github.com/opencloud-eu/opencloud/pull/1400)]
- build(deps): bump github.com/olekukonko/tablewriter from 1.0.8 to 1.0.9 [[#1376](https://github.com/opencloud-eu/opencloud/pull/1376)]
- build(deps): bump github.com/onsi/ginkgo/v2 from 2.24.0 to 2.25.1 [[#1396](https://github.com/opencloud-eu/opencloud/pull/1396)]
- [full-ci] Bump reva to latest main [[#1372](https://github.com/opencloud-eu/opencloud/pull/1372)]
- build(deps): bump github.com/prometheus/client_golang from 1.22.0 to 1.23.0 [[#1385](https://github.com/opencloud-eu/opencloud/pull/1385)]
- build(deps): bump github.com/onsi/ginkgo/v2 from 2.23.4 to 2.24.0 [[#1375](https://github.com/opencloud-eu/opencloud/pull/1375)]
- build(deps): bump github.com/gookit/config/v2 from 2.2.6 to 2.2.7 [[#1359](https://github.com/opencloud-eu/opencloud/pull/1359)]
- build(deps): bump golang.org/x/net from 0.42.0 to 0.43.0 [[#1356](https://github.com/opencloud-eu/opencloud/pull/1356)]
- chore(dependencies): bump reva 19625996460b2e68da3bbaf539e554366c59e111 [[#1357](https://github.com/opencloud-eu/opencloud/pull/1357)]
- build(deps): bump golang.org/x/image from 0.28.0 to 0.30.0 [[#1323](https://github.com/opencloud-eu/opencloud/pull/1323)]
- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.6 to 2.11.7 [[#1339](https://github.com/opencloud-eu/opencloud/pull/1339)]
- build(deps): bump github.com/onsi/gomega from 1.37.0 to 1.38.0 [[#1266](https://github.com/opencloud-eu/opencloud/pull/1266)]

## [3.3.0](https://github.com/opencloud-eu/opencloud/releases/tag/v3.3.0) - 2025-08-12

### ❤️ Thanks to all contributors! ❤️

@ScharfViktor, @aduffeck, @michaelstingl

### ✨ Features

- Tenant [[#1274](https://github.com/opencloud-eu/opencloud/pull/1274)]

### 📈 Enhancement

- chore: bump web to v3.3.0 [[#1329](https://github.com/opencloud-eu/opencloud/pull/1329)]

### ✅ Tests

- multiTenancyTests [[#1313](https://github.com/opencloud-eu/opencloud/pull/1313)]

### 📚 Documentation

- Fix posix driver documentation in STORAGE_USERS_DRIVER description [[#1305](https://github.com/opencloud-eu/opencloud/pull/1305)]

### 🐛 Bug Fixes

- Improve indexing performance using batches [[#1306](https://github.com/opencloud-eu/opencloud/pull/1306)]
- Do not run the timout func if the work func has run [[#1302](https://github.com/opencloud-eu/opencloud/pull/1302)]
- Make sure to register prometheus collectors only once [[#1295](https://github.com/opencloud-eu/opencloud/pull/1295)]

### 📦️ Dependencies

- [decomposed] bump-version-v3.3.0 [[#1332](https://github.com/opencloud-eu/opencloud/pull/1332)]
- [full-ci] Reva bump 2.36.0 [[#1328](https://github.com/opencloud-eu/opencloud/pull/1328)]
- Bump reva [[#1315](https://github.com/opencloud-eu/opencloud/pull/1315)]

## [3.2.1](https://github.com/opencloud-eu/opencloud/releases/tag/v3.2.1) - 2025-07-30

### ❤️ Thanks to all contributors! ❤️

@aduffeck, @dragonchaser, @individual-it

### 🐛 Bug Fixes

- Do not try to log metrics when we failed to get the consumer info [[#1289](https://github.com/opencloud-eu/opencloud/pull/1289)]
- Add thumbnails to sharedWithMe and sharedByMe requests [[#1257](https://github.com/opencloud-eu/opencloud/pull/1257)]

## [3.2.0](https://github.com/opencloud-eu/opencloud/releases/tag/v3.2.0) - 2025-07-21

### ❤️ Thanks to all contributors! ❤️

@AlexAndBear, @JammingBen, @ScharfViktor, @Svanvith, @aduffeck, @butonic, @dragonchaser, @fschade, @individual-it, @jnweiger, @micbar, @rhafer

### ✨ Features

- Metrics [[#1242](https://github.com/opencloud-eu/opencloud/pull/1242)]
- Add `HasTrashedItems` property to /me/drives endpoint [[#1163](https://github.com/opencloud-eu/opencloud/pull/1163)]

### 📈 Enhancement

- [full-ci] chore: bump web to v3.2.0 [[#1253](https://github.com/opencloud-eu/opencloud/pull/1253)]
- proxy(sign_url_auth): Allow to verify server signed URLs [[#1191](https://github.com/opencloud-eu/opencloud/pull/1191)]
- Switch to the raw nats consumer instead of the go-micro events [[#1171](https://github.com/opencloud-eu/opencloud/pull/1171)]
- change: adjust default values for the S3 Uploads [[#1224](https://github.com/opencloud-eu/opencloud/pull/1224)]
- feat(web): add dark mode and adjust light theme colors [[#1188](https://github.com/opencloud-eu/opencloud/pull/1188)]
- change: set better decomposedS3 defaults for multipart upload [[#1200](https://github.com/opencloud-eu/opencloud/pull/1200)]
- add missing full username mapper to the full example [[#1181](https://github.com/opencloud-eu/opencloud/pull/1181)]

### 🐛 Bug Fixes

- fix ready checks [[#1222](https://github.com/opencloud-eu/opencloud/pull/1222)]
- Update config.go [[#1183](https://github.com/opencloud-eu/opencloud/pull/1183)]
- Fix wrong build version [[#1210](https://github.com/opencloud-eu/opencloud/pull/1210)]
- Update Makefile [[#1187](https://github.com/opencloud-eu/opencloud/pull/1187)]
- fix(collaboration): re register app providers in a configurable interval [[#1035](https://github.com/opencloud-eu/opencloud/pull/1035)]
- Fix lico idp doesn't load opencloud font anymore [[#1153](https://github.com/opencloud-eu/opencloud/pull/1153)]

### 📦️ Dependencies

- [decomposed] bump-version-v3.2.0 [[#1258](https://github.com/opencloud-eu/opencloud/pull/1258)]
- [full-ci] Reva bump 2.35.0 [[#1255](https://github.com/opencloud-eu/opencloud/pull/1255)]
- build(deps): bump golang.org/x/net from 0.41.0 to 0.42.0 [[#1232](https://github.com/opencloud-eu/opencloud/pull/1232)]
- build(deps): bump github.com/KimMachineGun/automemlimit from 0.7.3 to 0.7.4 [[#1226](https://github.com/opencloud-eu/opencloud/pull/1226)]
- build(deps): bump golang.org/x/text from 0.26.0 to 0.27.0 [[#1227](https://github.com/opencloud-eu/opencloud/pull/1227)]
- build(deps): bump golang.org/x/sync from 0.15.0 to 0.16.0 [[#1209](https://github.com/opencloud-eu/opencloud/pull/1209)]
- build(deps): bump golang.org/x/term from 0.32.0 to 0.33.0 [[#1208](https://github.com/opencloud-eu/opencloud/pull/1208)]
- build(deps): bump github.com/olekukonko/tablewriter from 1.0.7 to 1.0.8 [[#1174](https://github.com/opencloud-eu/opencloud/pull/1174)]
- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.5 to 2.11.6 [[#1164](https://github.com/opencloud-eu/opencloud/pull/1164)]
- build(deps): bump github.com/go-playground/validator/v10 from 10.26.0 to 10.27.0 [[#1165](https://github.com/opencloud-eu/opencloud/pull/1165)]
- build(deps): bump github.com/pkg/xattr from 0.4.11 to 0.4.12 [[#1156](https://github.com/opencloud-eu/opencloud/pull/1156)]
- build(deps): bump go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp from 0.61.0 to 0.62.0 [[#1155](https://github.com/opencloud-eu/opencloud/pull/1155)]
- build(deps): bump github.com/open-policy-agent/opa from 1.5.1 to 1.6.0 [[#1148](https://github.com/opencloud-eu/opencloud/pull/1148)]
- build(deps): bump github.com/oklog/run from 1.1.0 to 1.2.0 [[#1150](https://github.com/opencloud-eu/opencloud/pull/1150)]

## [3.1.0](https://github.com/opencloud-eu/opencloud/releases/tag/v3.1.0) - 2025-06-30

### ❤️ Thanks to all contributors! ❤️

@06kellyjac, @AlexAndBear, @Leander-Wendt, @ScharfViktor, @aduffeck, @fschade, @individual-it, @kulmann, @rhafer

### ✨ Features

- feat: adjust space template image to match brand color [[#1098](https://github.com/opencloud-eu/opencloud/pull/1098)]

### ✅ Tests

- enable user-settings e2e tests [[#1140](https://github.com/opencloud-eu/opencloud/pull/1140)]

### 🐛 Bug Fixes

- Only remove obsolete IDs from the index [[#1127](https://github.com/opencloud-eu/opencloud/pull/1127)]
- fix: collabora use metrics instead of imperial metric system [[#1086](https://github.com/opencloud-eu/opencloud/pull/1086)]

### 📚 Documentation

- [full-ci] chore: bump web to v3.1.0 [[#1129](https://github.com/opencloud-eu/opencloud/pull/1129)]
- Update the href of CONTRIBUTING to the dev docs [[#1077](https://github.com/opencloud-eu/opencloud/pull/1077)]
- fix(docs): WEB_ASSET_PATH was still mentioned in the web readme [[#943](https://github.com/opencloud-eu/opencloud/pull/943)]
- Fix link in CONTRIBUTING.md [[#1048](https://github.com/opencloud-eu/opencloud/pull/1048)]

### 📈 Enhancement

- feat: re-enable Save As and Export in collabora [[#1119](https://github.com/opencloud-eu/opencloud/pull/1119)]
- Add a "posixfs consistency" command [[#1091](https://github.com/opencloud-eu/opencloud/pull/1091)]
- feat: add accessibility url to theme.json files [[#1108](https://github.com/opencloud-eu/opencloud/pull/1108)]
- cleanup: Avoid fetching group membership when not needed [[#1036](https://github.com/opencloud-eu/opencloud/pull/1036)]

### 📦️ Dependencies

- [decomposed] bump-version-v3.1.0 [[#1142](https://github.com/opencloud-eu/opencloud/pull/1142)]
- build(deps): bump go.etcd.io/bbolt from 1.4.1 to 1.4.2 [[#1131](https://github.com/opencloud-eu/opencloud/pull/1131)]
- [full-ci] chore:reva bump v.2.34 [[#1139](https://github.com/opencloud-eu/opencloud/pull/1139)]
- build(deps): bump go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc from 0.61.0 to 0.62.0 [[#1122](https://github.com/opencloud-eu/opencloud/pull/1122)]
- build(deps): bump go.opentelemetry.io/contrib/zpages from 0.61.0 to 0.62.0 [[#1123](https://github.com/opencloud-eu/opencloud/pull/1123)]
- build(deps): bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc from 1.36.0 to 1.37.0 [[#1111](https://github.com/opencloud-eu/opencloud/pull/1111)]
- build(deps): bump go.opentelemetry.io/otel from 1.36.0 to 1.37.0 [[#1112](https://github.com/opencloud-eu/opencloud/pull/1112)]
- build(deps): bump github.com/go-chi/chi/v5 from 5.2.1 to 5.2.2 [[#1075](https://github.com/opencloud-eu/opencloud/pull/1075)]
- build(deps): bump github.com/grpc-ecosystem/grpc-gateway/v2 from 2.26.3 to 2.27.0 [[#1072](https://github.com/opencloud-eu/opencloud/pull/1072)]
- build(deps): bump github.com/jellydator/ttlcache/v3 from 3.3.0 to 3.4.0 [[#1071](https://github.com/opencloud-eu/opencloud/pull/1071)]
- build(deps): bump github.com/urfave/cli/v2 from 2.27.6 to 2.27.7 [[#1061](https://github.com/opencloud-eu/opencloud/pull/1061)]
- build(deps): bump github.com/KimMachineGun/automemlimit from 0.7.2 to 0.7.3 [[#1062](https://github.com/opencloud-eu/opencloud/pull/1062)]
- Bump reva to pull in the latest fixes [[#1063](https://github.com/opencloud-eu/opencloud/pull/1063)]
- build(deps): bump go.etcd.io/bbolt from 1.4.0 to 1.4.1 [[#1045](https://github.com/opencloud-eu/opencloud/pull/1045)]
- build(deps): bump google.golang.org/grpc from 1.72.2 to 1.73.0 [[#1034](https://github.com/opencloud-eu/opencloud/pull/1034)]
- build(deps): bump golang.org/x/net from 0.40.0 to 0.41.0 [[#1033](https://github.com/opencloud-eu/opencloud/pull/1033)]
- build(deps-dev): bump jest from 29.7.0 to 30.0.0 in /services/idp [[#1040](https://github.com/opencloud-eu/opencloud/pull/1040)]
- build(deps-dev): bump css-minimizer-webpack-plugin from 7.0.0 to 7.0.2 in /services/idp [[#1038](https://github.com/opencloud-eu/opencloud/pull/1038)]
- build(deps): bump query-string from 9.1.1 to 9.2.0 in /services/idp [[#1031](https://github.com/opencloud-eu/opencloud/pull/1031)]

## [3.0.0](https://github.com/opencloud-eu/opencloud/releases/tag/v3.0.0) - 2025-06-10

### ❤️ Thanks to all contributors! ❤️

@AlexAndBear, @ScharfViktor, @VuiMuich, @aduffeck, @butonic, @fschade, @kulmann, @micbar, @prashant-gurung899, @rhafer

### 💥 Breaking changes

- do not automatically expand drive root permissions [[#495](https://github.com/opencloud-eu/opencloud/pull/495)]

### ✨ Features

- Enhancement: Introduced support for PrivateLink in WebDAV search responses [[#983](https://github.com/opencloud-eu/opencloud/pull/983)]
- Add profile photo [[#864](https://github.com/opencloud-eu/opencloud/pull/864)]
- feat: hide close button in collabora [[#828](https://github.com/opencloud-eu/opencloud/pull/828)]

### 📈 Enhancement

- graph: Add $filter to only list (and/or count) member permissions [[#996](https://github.com/opencloud-eu/opencloud/pull/996)]
- [full-ci] chore: bump web to v3.0.0 [[#1026](https://github.com/opencloud-eu/opencloud/pull/1026)]
- [full-ci] chore: bump web to v3.0.0-alpha.1 [[#972](https://github.com/opencloud-eu/opencloud/pull/972)]
- feat: add shareType to sharees field on activities api [[#954](https://github.com/opencloud-eu/opencloud/pull/954)]
- graph: Add more $select options to ListPermissions endpoint [[#916](https://github.com/opencloud-eu/opencloud/pull/916)]
- feat: add webp format [[#869](https://github.com/opencloud-eu/opencloud/pull/869)]

### ✅ Tests

- apiTest. count permission in the list permissions endpoint [[#1010](https://github.com/opencloud-eu/opencloud/pull/1010)]
- apiTest. select option for root/permissions endpoint [[#942](https://github.com/opencloud-eu/opencloud/pull/942)]
- [full-ci] ApiTest. checking private link in report response [[#993](https://github.com/opencloud-eu/opencloud/pull/993)]
- [full-ci] Change `eicar_com.zip` virus file and update tests [[#992](https://github.com/opencloud-eu/opencloud/pull/992)]

### 🐛 Bug Fixes

- Fix broken urls in README.md of deployment example [[#1023](https://github.com/opencloud-eu/opencloud/pull/1023)]
- Make activitylog service scalable [[#941](https://github.com/opencloud-eu/opencloud/pull/941)]
- Fix purging revisions from decomposeds3 blobstores [[#958](https://github.com/opencloud-eu/opencloud/pull/958)]
- fix(graph-metadata): lazy cs3 metadata storage initialization [[#946](https://github.com/opencloud-eu/opencloud/pull/946)]
- always get the user email for admin user [[#898](https://github.com/opencloud-eu/opencloud/pull/898)]

### 📚 Documentation

- Updated boxes in readme [[#970](https://github.com/opencloud-eu/opencloud/pull/970)]

### 📦️ Dependencies

- [decomposed] bump-version-v3.0.0 [[#1030](https://github.com/opencloud-eu/opencloud/pull/1030)]
- [full-ci] chore:reva bump v.2.33.1 [[#1027](https://github.com/opencloud-eu/opencloud/pull/1027)]
- build(deps): bump i18next from 25.1.2 to 25.2.1 in /services/idp [[#1024](https://github.com/opencloud-eu/opencloud/pull/1024)]
- build(deps): bump golang.org/x/image from 0.27.0 to 0.28.0 [[#1012](https://github.com/opencloud-eu/opencloud/pull/1012)]
- build(deps): bump @types/node from 22.15.29 to 22.15.30 in /services/idp [[#1008](https://github.com/opencloud-eu/opencloud/pull/1008)]
- build(deps): bump github.com/open-policy-agent/opa from 1.5.0 to 1.5.1 [[#1000](https://github.com/opencloud-eu/opencloud/pull/1000)]
- build(deps): bump golang.org/x/sync from 0.14.0 to 0.15.0 [[#1006](https://github.com/opencloud-eu/opencloud/pull/1006)]
- build(deps-dev): bump eslint-plugin-react from 7.37.2 to 7.37.5 in /services/idp [[#1004](https://github.com/opencloud-eu/opencloud/pull/1004)]
- build(deps-dev): bump postcss-normalize from 13.0.0 to 13.0.1 in /services/idp [[#1003](https://github.com/opencloud-eu/opencloud/pull/1003)]
- build(deps): bump @testing-library/react from 11.2.7 to 12.1.5 in /services/idp [[#994](https://github.com/opencloud-eu/opencloud/pull/994)]
- build(deps): bump github.com/blevesearch/bleve/v2 from 2.5.1 to 2.5.2 [[#999](https://github.com/opencloud-eu/opencloud/pull/999)]
- build(deps): bump @fontsource/roboto from 5.1.0 to 5.2.5 in /services/idp [[#995](https://github.com/opencloud-eu/opencloud/pull/995)]
- build(deps): bump google.golang.org/grpc from 1.72.1 to 1.72.2 [[#991](https://github.com/opencloud-eu/opencloud/pull/991)]
- build(deps): bump github.com/nats-io/nats.go from 1.42.0 to 1.43.0 [[#990](https://github.com/opencloud-eu/opencloud/pull/990)]
- build(deps): bump @types/jest from 29.5.12 to 29.5.14 in /services/idp [[#987](https://github.com/opencloud-eu/opencloud/pull/987)]
- build(deps): bump github.com/leonelquinteros/gotext from 1.7.1 to 1.7.2 [[#981](https://github.com/opencloud-eu/opencloud/pull/981)]
- build(deps): bump @types/node from 22.15.19 to 22.15.29 in /services/idp [[#980](https://github.com/opencloud-eu/opencloud/pull/980)]
- build(deps): bump github.com/opencloud-eu/libre-graph-api-go from 1.0.6 to 1.0.7 [[#982](https://github.com/opencloud-eu/opencloud/pull/982)]
- build(deps-dev): bump sass-loader from 16.0.4 to 16.0.5 in /services/idp [[#979](https://github.com/opencloud-eu/opencloud/pull/979)]
- build(deps): bump web-vitals from 4.2.4 to 5.0.2 in /services/idp [[#978](https://github.com/opencloud-eu/opencloud/pull/978)]
- build(deps): bump github.com/open-policy-agent/opa from 1.4.2 to 1.5.0 [[#977](https://github.com/opencloud-eu/opencloud/pull/977)]
- build(deps-dev): bump cldr from 7.5.0 to 7.9.0 in /services/idp [[#975](https://github.com/opencloud-eu/opencloud/pull/975)]
- build(deps): bump github.com/olekukonko/tablewriter from 1.0.6 to 1.0.7 [[#974](https://github.com/opencloud-eu/opencloud/pull/974)]
- build(deps): bump go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc from 0.60.0 to 0.61.0 [[#915](https://github.com/opencloud-eu/opencloud/pull/915)]
- build(deps): bump go.opentelemetry.io/contrib/zpages from 0.60.0 to 0.61.0 [[#938](https://github.com/opencloud-eu/opencloud/pull/938)]
- build(deps): bump @testing-library/user-event from 14.5.2 to 14.6.1 in /services/idp [[#939](https://github.com/opencloud-eu/opencloud/pull/939)]
- build(deps): bump i18next-browser-languagedetector from 7.2.1 to 8.1.0 in /services/idp [[#937](https://github.com/opencloud-eu/opencloud/pull/937)]
- build(deps): bump go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp from 0.60.0 to 0.61.0 [[#923](https://github.com/opencloud-eu/opencloud/pull/923)]
- build(deps): bump github.com/nats-io/nats-server/v2 from 2.11.3 to 2.11.4 [[#914](https://github.com/opencloud-eu/opencloud/pull/914)]
- build(deps): bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc from 1.35.0 to 1.36.0 [[#907](https://github.com/opencloud-eu/opencloud/pull/907)]
- build(deps): bump go.opentelemetry.io/otel/trace from 1.35.0 to 1.36.0 [[#906](https://github.com/opencloud-eu/opencloud/pull/906)]
- build(deps): bump github.com/blevesearch/bleve/v2 from 2.5.0 to 2.5.1 [[#900](https://github.com/opencloud-eu/opencloud/pull/900)]
- build(deps): bump axios from 1.7.7 to 1.8.2 in /services/idp [[#902](https://github.com/opencloud-eu/opencloud/pull/902)]
- build(deps): bump github.com/opencloud-eu/libre-graph-api-go from 1.0.5 to 1.0.6 [[#899](https://github.com/opencloud-eu/opencloud/pull/899)]
- build(deps): bump @types/node from 20.14.11 to 22.15.19 in /services/idp [[#886](https://github.com/opencloud-eu/opencloud/pull/886)]
- build(deps-dev): bump i18next-conv from 14.1.0 to 15.1.1 in /services/idp [[#887](https://github.com/opencloud-eu/opencloud/pull/887)]
- build(deps): bump golang.org/x/net from 0.39.0 to 0.40.0 [[#889](https://github.com/opencloud-eu/opencloud/pull/889)]
- build(deps): bump github.com/olekukonko/tablewriter from 0.0.5 to 1.0.6 [[#888](https://github.com/opencloud-eu/opencloud/pull/888)]

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
