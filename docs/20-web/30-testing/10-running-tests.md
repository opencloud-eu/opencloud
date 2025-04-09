---
title: 'Running tests'
sidebar_position: 1
id: running-tests
---



## Introduction

In order to allow us to make changes quickly, often and with a high level of confidence, we heavily rely on tests within the `web` repository.

All the steps below require you to have the `web` repo cloned locally and dependencies installed.
This can be achieved by running

```shell
$ git clone https://github.com/opencloud-eu/web.git
$ cd web
$ pnpm install
```

### Unit Tests

We have a steadily growing coverage of unit tests. You can run them locally via

```shell
$ pnpm test:unit
```

You can also specify which tests to run by giving a path param, like so: `pnpm test:unit packages/<app-name>/tests/unit/path/to/test.spec.js`.

#### Unit Test File Structure

Our unit tests spec files follow a simple structure:

- fixtures and mocks at the top
- helper functions at the bottom
- tests in between

We usually organize tests with nested `describe` blocks. If you would like to get feedback from the core team about
the structure, scope and goals of your unit tests before actually writing some, we invite you to make a pull request
with only `describe` blocks and nested `it.todo("put your test description here")` lines.

### E2E Tests (Playwright)

Our end-to-end test suite is built upon the [Playwright Framework](https://github.com/microsoft/playwright),
which makes it easy to write tests, debug them and have them run cross-browser with minimal overhead.

#### Preparation

Please make sure you have installed all dependencies and started the server(s) as described in [tooling](./../20-development/30-tooling.md).

#### Prepare & Start Web

Bundle the web frontend with the following command:

```shell
$ pnpm build:w
```

Our compose setup automatically mounts it into an OpenCloud backend, respectively. Web also gets recompiled on changes.

#### Run E2E Tests

The following command will run all available e2e tests:

```shell
$ pnpm test:e2e:cucumber 'tests/e2e/cucumber/**/*.feature'
```

#### Options

To run a particular test, simply add the feature file and line number to the test command, e.g. `pnpm test:e2e:cucumber tests/e2e/cucumber/features/smoke/admin-settings/users.feature:84`

Various options are available via ENV variables, e.g.

- `OC_BASE_URL` # use your OpenCloud URL. Default value: host.docker.internal:9200
- `BASIC_AUTH=true` use basic authorization for api requests.
- `RETRY=n` to retry failures `n` times
- `SLOW_MO=n` to slow the execution time by `n` milliseconds
- `TIMEOUT=n` to set tests to timeout after `n` milliseconds
- `HEADLESS=bool` to open the browser while the tests run (defaults to true => headless mode)
- `BROWSER=name` to run tests against a specific browser. Defaults to chromium, available are chromium, firefox, webkit, chromium
- `ADMIN_PASSWORD` to set administrator password. By default, the `admin` password is used in the test
- `PARALLEL` for parallel test execution

For debugging reasons, you may want to record a video or traces of your test run.
Again, you can use the following ENV variables in your command:

- `REPORT_DIR=another/path` to set a directory for your recorded files (defaults to "reports")
- `REPORT_VIDEO=true` to record a video of the test run
- `REPORT_HAR=true` to save request information from the test run
- `REPORT_TRACING=true` to record traces from the test run

To then open e.g. the tracing from the `REPORT_DIR`, run

```shell
$ npx playwright show-trace path/to/file.zip
```

#### Lint E2E Test Code

Run the following command to find out the lint issues early in the test codes:

```shell
$ pnpm lint
```

And to fix the lint problems run the following command:

```shell
$ pnpm lint --fix
```

If the lint problems are not fixed by `--fix` option, we have to manually fix the code.

### Analyze the Test Report

The cucumber library is used as the test runner for e2e tests. The report generator script lives inside the `tests/e2e/cucumber/report` folder. If you want to create a report after the tests are done, run the command:

```bash
node tests/e2e/cucumber/report --report-input=tests/e2e/cucumber/report/report.json
```

By default, the report gets generated to reports/e2e/cucumber/releaseReport/cucumber_report.html.
The location can be changed by adding the `--report-location` flag.

To see all available options run

```bash
node tests/e2e/cucumber/report --help
```

### E2E Tests on OpenCloud With Keycloak

We can run some of the e2e tests on OpenCloud setup with Keycloak as an external idp. To run tests against locally, please follow the steps below:

#### Run OpenCloud With Keycloak

There's a documentation to serve [OpenCloud with Keycloak](https://docs.opencloud.eu/opencloud/deployment/opencloud_keycloak/). Please follow each step to run **OpenCloud with Keycloak**.

#### Run E2E Tests

```bash
KEYCLOAK=true \
BASE_URL_OPEN_CLOUD=demo.opencloud.test \
pnpm run test:e2e:cucumber tests/e2e/cucumber/features/journeys
```

Following environment variables come in use while running e2e tests on OpenCloud with Keycloak:

- `BASE_URL_OPENCLOUD` sets OpenCloud url (e.g.: demo.opencloud.test)
- `KEYCLOAK_HOST` sets Keycloak url (e.g.: keycloak.opencloud.test)
- `KEYCLOAK=true` runs the tests with Keycloak
- `KEYCLOAK_REALM` sets OpenCloud realm name used on Keycloak
