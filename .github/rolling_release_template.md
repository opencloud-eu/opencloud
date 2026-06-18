### Rolling release template
[Release Template](https://github.com/opencloud-eu/opencloud/blob/main/.github/rolling_release_template.md)

## Prerequisites
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
* [ ] n8n integration QA
* [ ] confirmatory testing, if needed

## Collected bugs

## After QA Phase
* [ ] replace `%%NEXT%%` wuth the release version
* [ ] squash and merge Release PR
* [ ] publish release notes to the docs
* [ ] add migration guide to changelog with prefix `**ACTION REQUIRED:**`, if needed.  
* [ ] add release notes from web and reva to opencloud changelog
* [ ] update the public matrix channel topic
* [ ] update https://update.opencloud.eu/server.json
* [ ] update the version on demo.opencloud.eu
