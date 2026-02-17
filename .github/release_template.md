### Template
[Release Template](https://github.com/opencloud-eu/opencloud/blob/main/.github/release_template.md)

### Prerequisites

* [ ] DEV/QA: bump web version
* [ ] DEV/QA: bump reva version
* [ ] DEV/QA: DEV: Create rc tag `vx.y.z-rc.x`
* [ ] DEV: update introductionVersion
* [ ] DEV: add new production version

### QA Phase

* [ ] QA: Compatibility test with posix fs
* [ ] QA: Compatibility test with decomposed fs
* [ ] DEV/QA: Performance test
  * [ ] STORAGE_USERS_DRIVER=posix
    * [ ] 75vu's, 60m
    * [ ] 75vu's, 60m
  * [ ] STORAGE_USERS_DRIVER=decomposed
    * [ ] 75vu's, 60m
    * [ ] 75vu's, 60m
* [ ] QA: documentation test
  * [ ] QA: Review documentation
  * [ ] QA: Verify all new features documented
  * [ ] QA: Create upgrade documentation
  * [ ] QA: Check installation guides

* [ ] QA: e2e with different storage:
  * [ ] QA: decomposed
  * [ ] QA: decomposeds3
  * [ ] QA: posix
  * [ ] QA: posix with enabled watch_fs
* [ ] QA: e2e with different deployments deployments:
  * [ ] QA: e2e tests agains opencloud-charts
  * [ ] QA: binary 
  * [ ] QA: multitanacy 
  * [ ] QA: docker using [docker-compose_test_plan](https://github.com/opencloud-eu/qa/blob/main/.github/ISSUE_TEMPLATE/docker-compose_test_plan_template.md)
* [ ] QA: Different clients:
  * [ ] QA: desktop (define version) https://github.com/opencloud-eu/client/releases
    * [ ] QA: against mac - exploratory testing
    * [ ] QA: against windows - exploratory testing
    * [ ] QA: against linux (use auto tests)
  * [ ] QA: android (define version) https://github.com/opencloud-eu/android/releases
  * [ ] QA: ios (define version)
* [ ] QA: check docs german translation
  * [ ] QA: german translations desktop at 100%
* [ ] QA: exploratory testing

### Collected bugs
* [ ] Please place all bugs found here

### After QA Phase (IT related)

* [ ] QA:bump version in pkg/version.go
* [ ] QA: Run CI
* [ ] DEV/QA: create final tag
* [ ] QA: observe CI Run on tag
* [ ] DEV/QA: Create a new `stable-*` branch
  * [ ] (opencloud)[https://github.com/opencloud-eu/opencloud/branches]
  * [ ] (web)[https://github.com/opencloud-eu/web/branches]
  * [ ] (reva)[https://github.com/opencloud-eu/reva/branches]
  * [ ] (opencloud-compose)[https://github.com/opencloud-eu/opencloud-compose/branches]
* [ ] DEV/QA:: publish release notes to the docs
* [ ] DEV/QA:: update (demo.opencloud.eu)[https://demo.opencloud.eu/]

### After QA Phase ( Marketing / Product / Sales related )

* [ ] notify marketing that the release is ready @tbsbdr
* [ ] announce in the public channel (matrix channel)[https://matrix.to/#/#opencloud:matrix.org]
* [ ] press information @AnneGo137
  * [ ] press information @AnneGo137
  * [ ] Blogentry @AnneGo137
  * [ ] Internal meeting (Groupe Pre-Webinar) @db-ot
  * [ ] Partner briefing (Partner should be informed about features, new) @matthias
* [ ] Webinar DE & EN @AnneGo137
  * [ ] Präsentation DE @tbsbdr / @db-ot
  * [ ] Präsentation EN @tbsbdr / @db-ot
* [ ] Website ergänzen @AnneGo137
  * [ ] Features @AnneGo137
  * [ ] Service & Support - New Enterprise Features @tbsbdr
  * [ ] OpenCloud_Benefits.pdf updates @AnneGo137
  * [ ] Welcome Files: Features as media @tbsbdr
* [ ] Flyer update @AnneGo137
* [ ] Sales presentation @matthias
