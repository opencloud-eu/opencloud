# Changelog

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
