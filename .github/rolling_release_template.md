### Rolling release template
[Release Template](https://github.com/opencloud-eu/opencloud/blob/main/.github/rolling_release_template.md)

## Prerequisites
* [ ] replace `%%NEXT%%` with the release version

* [ ] web release
  * [ ] bump web version
  * [ ] squash and merge the web release PR
  * [ ] bump web v.x.y.z in opencloud

* [ ] reva release
  * [ ] squash and merge the reva Release PR
  * [ ] bump reva and update opencloud version in `pkg/version.go`

## QA Phase
* [ ] bump `opencloud_commitid` in web and run all working tests in CI
* [ ] compatibility test
* [ ] confirmatory testing, if needed
* [ ] squash and merge Release PR

## Collected bugs

## After QA Phase
* [ ] publish release notes to the docs
* [ ] add migration guide to changelog with prefix `**ACTION REQUIRED:**`, if needed.  
* [ ] add release notes from web and reva to opencloud changelog
* [ ] n8n integration - update new opencloud version
* [ ] docker-compose - update new opencloud version
* [ ] update the public matrix channel topic
* [ ] update https://update.opencloud.eu/server.json
* [ ] update the version on demo.opencloud.eu
