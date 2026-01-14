# Acceptance Testing

To run tests in the test suite you have two options. You may go the easy way and just run the test suite in docker. But for some tasks you could also need to install the test suite natively, which requires a little more setup since PHP and some dependencies need to be installed.

Both ways to run tests with the test suites are described here.

## Running Test Suite in Docker

Let's see what is available. Invoke the following command from within the root of the OpenCloud repository.

```bash
make -C tests/acceptance/docker help
```

Basically we have two sources for feature tests and test suites:

- [OpenCloud feature test and test suites](https://github.com/opencloud-eu/opencloud/tree/main/tests/acceptance/features)
- [tests and test suites transferred from core, they have prefix coreApi](https://github.com/opencloud-eu/opencloud/tree/main/tests/acceptance/features)

At the moment, both can be applied to OpenCloud.

As a storage backend, we support the OpenCloud native storage, also called `decomposed`. This stores files directly on disk. Along with that we also provide `decomposeds3`, `posix`  storage drivers.

You can invoke two types of test suite runs:

- run a full test suite, which consists of multiple feature tests
- run a single feature or single scenario in a feature

### Run Full Test Suite

#### Local OpenCloud Tests (prefix `api`)

The names of the full test suite make targets have the same naming as in the CI pipeline. See the available local OpenCloud specific test suites [here](https://github.com/opencloud-eu/opencloud/tree/main/tests/acceptance/features). They can be run with `decomposed` storage, `decomposeds3` storage and `posix` storage

For example, command:

```bash
make -C tests/acceptance/docker localApiTests-apiGraph-decomposed
```

runs the same tests as the `localApiTests-apiGraph-decomposed` CI pipeline, which runs the OpenCloud test suite "apiGraph" against the OpenCloud server with `decomposed` storage.

command:

```bash
make -C tests/acceptance/docker localApiTests-apiGraph-decomposeds3
```

runs the OpenCloud test suite `apiGraph` against the OpenCloud server with `decomposeds3` storage.

And command:

```bash
make -C tests/acceptance/docker localApiTests-apiGraph-posix
```

runs the OpenCloud test suite `apiGraph` against the OpenCloud server with `posix` storage.

Note:
While running the tests, OpenCloud server is started with [ocwrapper](https://github.com/opencloud-eu/opencloud/blob/main/tests/ocwrapper/README.md) (i.e. `WITH_WRAPPER=true`) by default. In order to run the tests without ocwrapper, provide `WITH_WRAPPER=false` when running the tests. For example:

```bash
WITH_WRAPPER=false \
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature:26' \
make -C tests/acceptance/docker test-opencloud-feature-decomposed-storage
```

But some test suites that are tagged with `@env-config` require the OpenCloud server to be run with ocwrapper. So, running those tests require `WITH_WRAPPER=true` (default setting).

Note:
To run the tests that require an email server (tests tagged with `@email`), you need to provide `START_EMAIL=true` while running the tests.

```bash
START_EMAIL=true \
BEHAT_FEATURE='tests/acceptance/features/apiNotification/emailNotification.feature' \
make -C tests/acceptance/docker test-opencloud-feature-decomposed-storage
```

Note:
To run the tests that require tika service (tests tagged with `@tikaServiceNeeded`), you need to provide `START_TIKA=true` while running the tests.

```bash
START_TIKA=true \
BEHAT_FEATURE='tests/acceptance/features/apiSearchContent/contentSearch.feature' \
make -C tests/acceptance/docker test-opencloud-feature-decomposed-storage
```

Note:
To run the tests that require an antivirus service (tests tagged with `@antivirus`), you need to provide the following environment variables while running the tests.

```bash
START_ANTIVIRUS=true \
OC_ASYNC_UPLOADS=true \
OC_ADD_RUN_SERVICES=antivirus \
POSTPROCESSING_STEPS=virusscan \
BEHAT_FEATURE='tests/acceptance/features/apiAntivirus/antivirus.feature' \
make -C tests/acceptance/docker test-opencloud-feature-decomposed-storage
```

#### Tests Transferred From Core (prefix `coreApi`)

Command `make -C tests/acceptance/docker Core-API-Tests-decomposed-storage-3` runs the same tests as the `Core-API-Tests-decomposed-storage-3` CI pipeline, which runs the third (out of ten) test suite groups transferred from core against the OpenCloud server with `decomposed` storage.

And `make -C tests/acceptance/docker Core-API-Tests-decomposeds3-storage-3` runs the third (out of ten) test suite groups transferred from core against the OpenCloud server with `decomposeds3` storage.

### Run Single Feature Test

The tests for a single feature (a feature file) can also be run against the different storage backends. To do that, multiple make targets with the schema **test-_\<test-source\>_-feature-_\<storage-backend\>_** are available. To select a single feature you have to add an additional `BEHAT_FEATURE=<path-to-feature-file>` parameter when invoking the make command.

For example;

```bash
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature' \
make -C tests/acceptance/docker test-opencloud-feature-decomposed-storage
```

Note:
`BEHAT_FEATURE` must be pointing to a valid feature file

And to run a single scenario in a feature, you can do:

Note:
A specific scenario from a feature can be run by adding `:<line-number>` at the end of the feature file path. For example, to run the scenario at line 26 of the feature file `apiGraphUserGroup/createUser.feature`, simply add the line number like this: `apiGraphUserGroup/createUser.feature:26`. Note that the line numbers mentioned in the examples might not always point to a scenario, so always check the line numbers before running the test.

```bash
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature:26' \
make -C tests/acceptance/docker test-opencloud-feature-decomposed-storage
```

Similarly, with `decomposeds3` storage;

```bash
# run a whole feature
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature' \
make -C tests/acceptance/docker test-opencloud-feature-decomposeds3-storage

# run a single scenario
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature:26' \
make -C tests/acceptance/docker test-opencloud-feature-decomposeds3-storage
```

In the same way, tests transferred from core can be run as:

```bash
# run a whole feature
BEHAT_FEATURE='tests/acceptance/features/coreApiAuth/webDavAuth.feature' \
make -C tests/acceptance/docker test-core-feature-decomposed-storage

# run a single scenario
BEHAT_FEATURE='tests/acceptance/features/coreApiAuth/webDavAuth.feature:15' \
make -C tests/acceptance/docker test-core-feature-decomposed-storage
```

Note:
The test suites transferred from core have `coreApi` prefixed

### OpenCloud Image to Be Tested (Skip Local Image Build)

By default, the tests will be run against the docker image built from your current working state of the OpenCloud repository. For some purposes it might also be handy to use an OpenCloud image from Docker Hub. Therefore, you can provide the optional flag `OC_IMAGE_TAG=...` which must contain an available docker tag of the [opencloud-eu/opencloud registry on Docker Hub](https://hub.docker.com/r/opencloud-eu/opencloud) (e.g. 'latest').

```bash
OC_IMAGE_TAG=latest \
make -C tests/acceptance/docker localApiTests-apiGraph-opencloud
```

### Test Log Output

While a test is running or when it is finished, you can attach to the logs generated by the tests.

```bash
make -C tests/acceptance/docker show-test-logs
```

Note:
The log output is opened in `less`. You can navigate up and down with your cursors. By pressing "F" you can follow the latest line of the output.

### Cleanup

During testing we start a redis and OpenCloud docker container. These will not be stopped automatically. You can stop them with:

```bash
make -C tests/acceptance/docker clean
```

## Running Test Suite in Local Environment

### Run OpenCloud

Create an up-to-date OpenCloud binary by [building OpenCloud]({{< ref "build" >}})

To start OpenCloud:

```bash
IDM_ADMIN_PASSWORD=admin \
opencloud/bin/opencloud init --insecure true

OC_INSECURE=true PROXY_ENABLE_BASIC_AUTH=true \
opencloud/bin/opencloud server
```

`PROXY_ENABLE_BASIC_AUTH` will allow the acceptance tests to make requests against the provisioning api (and other endpoints) using basic auth.

#### Run Local OpenCloud Tests (prefix `api`) and Tests Transferred From Core (prefix `coreApi`)

```bash
make test-acceptance-api \
TEST_SERVER_URL=https://localhost:9200 \
```

Useful environment variables:

`TEST_SERVER_URL`: OpenCloud server url. Please, adjust the server url according to your setup.

`BEHAT_FEATURE`: to run a single feature

Note:
A specific scenario from a feature can be run by adding `:<line-number>` at the end of the feature file path. For example, to run the scenario at line 26 of the feature file `apiGraphUserGroup/createUser.feature`, simply add the line number like this: `apiGraphUserGroup/createUser.feature:26`. Note that the line numbers mentioned in the examples might not always point to a scenario, so always check the line numbers before running the test.

> Example:
>
> BEHAT_FEATURE=tests/acceptance/features/apiGraphUserGroup/createUser.feature
>
> Or
>
> BEHAT_FEATURE=tests/acceptance/features/apiGraphUserGroup/createUser.feature:13

`BEHAT_SUITE`: to run a single suite

> Example:
>
> BEHAT_SUITE=apiGraph

`STORAGE_DRIVER`: to run tests with a different user storage driver. Available options are `decomposed` (default), `owncloudsql` and `decomposeds3`

> Example:
>
> STORAGE_DRIVER=owncloudsql

`STOP_ON_FAILURE`: to stop running tests after the first failure

> Example:
>
> STOP_ON_FAILURE=true

### Use Existing Tests for BDD

As a lot of scenarios are written for core, we can use those tests for Behaviour driven development in OpenCloud.
Every scenario that does not work in OpenCloud with `decomposed` storage, is listed in `tests/acceptance/expected-failures-API-on-decomposed-storage.md` with a link to the related issue.

Those scenarios are run in the ordinary acceptance test pipeline in CI. The scenarios that fail are checked against the
expected failures. If there are any differences then the CI pipeline fails.

If you want to work on a specific issue

1. locally run each of the tests marked with that issue in the expected failures file.

   E.g.:

   ```bash
   make test-acceptance-api \
   TEST_SERVER_URL=https://localhost:9200 \
   STORAGE_DRIVER=decomposed \
   BEHAT_FEATURE='tests/acceptance/features/coreApiVersions/fileVersions.feature:141'
   ```

2. the tests will fail, try to understand how and why they are failing
3. fix the code
4. go back to 1. and repeat till the tests are passing.
5. remove those tests from the expected failures file
6. make a PR that has the fixed code, and the relevant lines removed from the expected failures file.

## Running Tests With And Without `remote.php`

By default, the tests are run with `remote.php` enabled. If you want to run the tests without `remote.php`, you can disable it by setting the environment variable `WITH_REMOTE_PHP=false` while running the tests.

```bash
WITH_REMOTE_PHP=false \
TEST_SERVER_URL="https://localhost:9200" \
make test-acceptance-api
```

## Running ENV Config Tests (@env-Config)

Test suites tagged with `@env-config` are used to test the environment variables that are used to configure OpenCloud. These tests are special tests that require the OpenCloud server to be run using [ocwrapper](https://github.com/opencloud-eu/opencloud/blob/main/tests/ocwrapper/README.md).

### Run OpenCloud With ocwrapper

```bash
# working dir: OpenCloud repo root dir

# init OpenCloud
IDM_ADMIN_PASSWORD=admin \
opencloud/bin/opencloud init --insecure true

# build the wrapper
cd tests/ocwrapper
make build

# run OpenCloud
PROXY_ENABLE_BASIC_AUTH=true \
./bin/ocwrapper serve --bin=../../opencloud/bin/opencloud
```

### Run the Tests

```bash
OC_WRAPPER_URL=http://localhost:5200 \
TEST_SERVER_URL="https://localhost:9200" \
BEHAT_FEATURE=tests/acceptance/features/apiAsyncUpload/delayPostprocessing.feature \
make test-acceptance-api
```

### Writing New ENV Config Tests

While writing tests for a new OpenCloud ENV configuration, please make sure to follow these guidelines:

1. Tag the test suite (or test scenarios) with `@env-config`
2. Use `OcConfigHelper.php` for helper functions - provides functions to reconfigure the running OpenCloud instance.
3. Recommended: add the new step implementations in `OcConfigContext.php`

## Running Test Suite With Email Service (@email)

Test suites that are tagged with `@email` require an email service. We use inbucket as the email service in our tests.

### Setup Inbucket

Run the following command to setup inbucket

```bash
docker run -d -p9000:9000 -p2500:2500 --name inbucket inbucket/inbucket
```

### Run OpenCloud

Documentation for environment variables is available [here](https://docs.opencloud.eu/services/notifications/#environment-variables)

```bash
# init OpenCloud
IDM_ADMIN_PASSWORD=admin \
opencloud/bin/opencloud init --insecure true

# run OpenCloud
PROXY_ENABLE_BASIC_AUTH=true \
OC_ADD_RUN_SERVICES=notifications \
NOTIFICATIONS_SMTP_HOST=localhost \
NOTIFICATIONS_SMTP_PORT=2500 \
NOTIFICATIONS_SMTP_INSECURE=true \
NOTIFICATIONS_SMTP_SENDER="OpenCloud <noreply@example.com>" \
opencloud/bin/opencloud server
```

### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
EMAIL_HOST="localhost" \
EMAIL_PORT=9000 \
BEHAT_FEATURE="tests/acceptance/features/apiNotification/emailNotification.feature" \
make test-acceptance-api
```

## Running Test Suite With Tika Service (@tikaServiceNeeded)

Test suites that are tagged with `@tikaServiceNeeded` require tika service.

### Setup Tika Service

Run the following docker command to setup tika service

```bash
docker run -d -p 127.0.0.1:9998:9998 apache/tika
```

### Run OpenCloud

TODO: Documentation related to the content based search and tika extractor will be added later.

```bash
# init OpenCloud
IDM_ADMIN_PASSWORD=admin \
opencloud/bin/opencloud init --insecure true

# run OpenCloud
PROXY_ENABLE_BASIC_AUTH=true \
OC_INSECURE=true \
SEARCH_EXTRACTOR_TYPE=tika \
SEARCH_EXTRACTOR_TIKA_TIKA_URL=http://localhost:9998 \
SEARCH_EXTRACTOR_CS3SOURCE_INSECURE=true \
opencloud/bin/opencloud server
```

### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
BEHAT_FEATURE="tests/acceptance/features/apiSearchContent/contentSearch.feature" \
make test-acceptance-api
```

## Running Test Suite With Antivirus Service (@antivirus)

Test suites that are tagged with `@antivirus` require antivirus service. TODO The available antivirus and the configuration related to them will be added latert. This documentation is only going to use `clamav` as antivirus.

### Setup clamAV

#### 1. Setup Locally

##### Linux OS user

Run the following command to set up calmAV and clamAV daemon

```bash
sudo apt install clamav clamav-daemon -y
```

Make sure that the clamAV daemon is up and running

```bash
sudo service clamav-daemon status
```

Note:
The commands are ubuntu specific and may differ according to your system. You can find information related to installation of clamAV in their official documentation [here](https://docs.clamav.net/manual/Installing/Packages.html).

##### Mac OS user

Install ClamAV using [here](https://gist.github.com/mendozao/3ea393b91f23a813650baab9964425b9)
Start ClamAV daemon

```bash
/your/location/to/brew/Cellar/clamav/1.1.0/sbin/clamd
```

#### 2. Setup clamAV With Docker

Run `clamAV` through docker

```bash
docker run -d -p 3310:3310 opencloudeu/clamav-ci:latest
```

### Run OpenCloud

As `antivirus` service is not enabled by default we need to enable the service while running OpenCloud server. We also need to enable `async upload` and as virus scan is performed in post-processing step, we need to set it as well. Documentation for environment variables related to antivirus is available [here](https://docs.opencloud.eu/services/antivirus/#environment-variables)

```bash
# init OpenCloud
IDM_ADMIN_PASSWORD=admin \
opencloud/bin/opencloud init --insecure true

# run OpenCloud
PROXY_ENABLE_BASIC_AUTH=true \
ANTIVIRUS_SCANNER_TYPE="clamav" \
ANTIVIRUS_CLAMAV_SOCKET="tcp://host.docker.internal:3310" \
POSTPROCESSING_STEPS="virusscan" \
OC_ASYNC_UPLOADS=true \
OC_ADD_RUN_SERVICES="antivirus" \
opencloud/bin/opencloud server
```

Note:
The value for `ANTIVIRUS_CLAMAV_SOCKET` is an example which needs adaption according your OS.

For antivirus running localy on Linux OS, use `ANTIVIRUS_CLAMAV_SOCKET= "/var/run/clamav/clamd.ctl"`.
For antivirus running localy on Mac OS, use `ANTIVIRUS_CLAMAV_SOCKET= "/tmp/clamd.sock"`.
For antivirus running with docker, use `ANTIVIRUS_CLAMAV_SOCKET= "tcp://host.docker.internal:3310"`

### Create virus files

The antivirus tests require EICAR test files which are not stored in the repository
They are generated dynamically when needed for testing.

```bash
tests/acceptance/scripts/generate-virus-files.sh
```

#### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
BEHAT_FEATURE="tests/acceptance/features/apiAntivirus/antivirus.feature" \
make test-acceptance-api
```

## Running Test Suite With Federated Sharing (@ocm)

Test suites that are tagged with `@ocm` require running two different OpenCloud instances. TODO More detailed information and configuration related to it will be added later.

### Setup First OpenCloud Instance

```bash
# init OpenCloud
IDM_ADMIN_PASSWORD=admin \
opencloud/bin/opencloud init --insecure true

# run OpenCloud
OC_URL="https://localhost:9200" \
PROXY_ENABLE_BASIC_AUTH=true \
OC_ENABLE_OCM=true \
OCM_OCM_PROVIDER_AUTHORIZER_PROVIDERS_FILE="tests/config/local/providers.json" \
OC_ADD_RUN_SERVICES="ocm" \
OCM_OCM_INVITE_MANAGER_INSECURE=true \
OCM_OCM_SHARE_PROVIDER_INSECURE=true \
OCM_OCM_STORAGE_PROVIDER_INSECURE=true \
WEB_UI_CONFIG_FILE="tests/config/local/opencloud-web.json" \
opencloud/bin/opencloud server
```

The first OpenCloud instance should be available at: https://localhost:9200/

### Setup Second OpenCloud Instance

You can run the second OpenCloud instance in two ways:

#### Using `.vscode/launch.json`

From the `Run and Debug` panel of VSCode, select `Fed OpenCloud Server` and start the debugger.

#### Using env file

```bash
# init OpenCloud
source tests/config/local/.env-federation && opencloud/bin/opencloud init

# run OpenCloud
opencloud/bin/opencloud server
```

The second OpenCloud instance should be available at: https://localhost:10200/

Note:
To enable ocm in the web interface, you need to set the following envs:
`OC_ENABLE_OCM="true"`
`OC_ADD_RUN_SERVICES="ocm"`

#### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
TEST_SERVER_FED_URL="https://localhost:10200" \
BEHAT_FEATURE="tests/acceptance/features/apiOcm/ocm.feature" \
make test-acceptance-api
```

## Running Text Preview Tests Containing Unicode Characters

There are some tests that check the text preview of files containing Unicode characters. The OpenCloud server by default cannot generate the thumbnail of such files correctly but it provides an environment variable to allow the use of custom fonts that support Unicode characters. So to run such tests successfully, we have to run the OpenCloud server with this environment variable.

```bash
...
THUMBNAILS_TXT_FONTMAP_FILE="/path/to/fontsMap.json"
opencloud/bin/opencloud server
```

The sample `fontsMap.json` file is located in `tests/config/drone/fontsMap.json`.

```json
{
  "defaultFont": "/path/to/opencloud/tests/config/drone/NotoSans.ttf"
}
```

## Running All API Tests Locally

### Build dev docker

```bash
make -C opencloud dev-docker 
```

### Choose STORAGE_DRIVER
By default, the system uses `posix` storage. However, you can override this by setting the `STORAGE_DRIVER` environment variable.


### Run a script that starts the openCloud server in the docker and runs the API tests locally (for debugging purposes) 

```bash
STORAGE_DRIVER=posix ./tests/acceptance/run_api_tests.sh 
```

## Running WOPI Validator Tests

### Available Test Groups

```text
  BaseWopiViewing
  CheckFileInfoSchema
  EditFlows
  Locks
  AccessTokens
  GetLock
  ExtendedLockLength
  FileVersion
  Features
  PutRelativeFile
  RenameFileIfCreateChildFileIsNotSupported
```

### Run Test

```bash
TEST_GROUP=BaseWopiViewing docker compose -f tests/acceptance/docker/src/wopi-validator-test.yml up -d
```

### for macOS use arm image
```bash
WOPI_VALIDATOR_IMAGE=scharfvi/wopi-validator \
TEST_GROUP=BaseWopiViewing \
docker compose -f tests/acceptance/docker/src/wopi-validator-test.yml up -d
```
