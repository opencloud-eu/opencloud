"""OpenCloud CI definition
"""

# Production release tags
# NOTE: need to be updated if new production releases are determined
# - follow semver
# - omit 'v' prefix
PRODUCTION_RELEASE_TAGS = ["5.0", "7"]

# Repository

repo_slug = "opencloud-eu/opencloud"
docker_repo_slug = "opencloudeu/opencloud"

# images
ALPINE_GIT = "alpine/git:latest"
APACHE_TIKA = "apache/tika:2.8.0.0"
CHKO_DOCKER_PUSHRM = "chko/docker-pushrm:1"
COLLABORA_CODE = "collabora/code:24.04.5.1.1"
INBUCKET_INBUCKET = "inbucket/inbucket"
MINIO_MC = "minio/mc:RELEASE.2021-10-07T04-19-58Z"
OC_CI_ALPINE = "owncloudci/alpine:latest"
OC_CI_BAZEL_BUILDIFIER = "owncloudci/bazel-buildifier:latest"
OC_CI_CLAMAVD = "owncloudci/clamavd"
OC_CI_DRONE_ANSIBLE = "owncloudci/drone-ansible:latest"
OC_CI_DRONE_SKIP_PIPELINE = "owncloudci/drone-skip-pipeline"
OC_CI_GOLANG = "owncloudci/golang:1.24"
OC_CI_NODEJS = "owncloudci/nodejs:%s"
OC_CI_PHP = "owncloudci/php:%s"
OC_CI_WAIT_FOR = "owncloudci/wait-for:latest"
OC_CS3_API_VALIDATOR = "owncloud/cs3api-validator:0.2.1"
OC_LITMUS = "owncloudci/litmus:latest"
OC_UBUNTU = "owncloud/ubuntu:20.04"
ONLYOFFICE_DOCUMENT_SERVER = "onlyoffice/documentserver:7.5.1"
PLUGINS_CODACY = "plugins/codacy:1"
PLUGINS_DOCKER = "plugins/docker:latest"
PLUGINS_GH_PAGES = "plugins/gh-pages:1"
PLUGINS_GITHUB_RELEASE = "plugins/github-release:1"
PLUGINS_GIT_ACTION = "plugins/git-action:1"
PLUGINS_MANIFEST = "plugins/manifest:1"
PLUGINS_S3 = "plugins/s3:1"
PLUGINS_S3_CACHE = "plugins/s3-cache:1"
PLUGINS_SLACK = "plugins/slack:1"
REDIS = "redis:6-alpine"
SONARSOURCE_SONAR_SCANNER_CLI = "sonarsource/sonar-scanner-cli:11.0"

DEFAULT_PHP_VERSION = "8.2"
DEFAULT_NODEJS_VERSION = "20"

dirs = {
    "base": "/woodpecker/src/github.com/opencloud-eu/opencloud",
    "web": "/woodpecker/src/github.com/opencloud-eu/opencloud/webTestRunner",
    "zip": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip",
    "webZip": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip/web.tar.gz",
    "webPnpmZip": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip/pnpm-store.tar.gz",
    "baseGo": "/go/src/github.com/opencloud-eu/opencloud",
    "gobinTar": "go-bin.tar.gz",
    "gobinTarPath": "/go/src/github.com/opencloud-eu/opencloud/go-bin.tar.gz",
    "ocisConfig": "tests/config/drone/ocis-config.json",
    "ocis": "/woodpecker/src/github.com/opencloud-eu/opencloud/srv/app/tmp/ocis",
    "ocisRevaDataRoot": "/woodpecker/src/github.com/opencloud-eu/opencloud/srv/app/tmp/ocis/owncloud/data",
    "ocisWrapper": "/woodpecker/src/github.com/opencloud-eu/opencloud/tests/ociswrapper",
    "bannedPasswordList": "tests/config/drone/banned-password-list.txt",
    "ocmProviders": "tests/config/drone/providers.json",
    "opencloudBinPath": "opencloud/bin",
    "opencloudBin": "opencloud/bin/opencloud",
    "opencloudBinArtifact": "opencloud-binary-amd64",
}

# OpenCloud URLs
OC_SERVER_NAME = "opencloud-server"
OCIS_URL = "https://%s:9200" % OC_SERVER_NAME
OCIS_DOMAIN = "%s:9200" % OC_SERVER_NAME
FED_OC_SERVER_NAME = "federation-opencloud-server"
OC_FED_URL = "https://%s:10200" % FED_OC_SERVER_NAME
OC_FED_DOMAIN = "%s:10200" % FED_OC_SERVER_NAME

# configuration
config = {
    "cs3ApiTests": {
        "skip": False,
    },
    "wopiValidatorTests": {
        "skip": False,
    },
    "k6LoadTests": {
        "skip": False,
    },
    "localApiTests": {
        "basic": {
            "suites": [
                "apiArchiver",
                "apiContract",
                "apiCors",
                "apiAsyncUpload",
                "apiDownloads",
                "apiDepthInfinity",
                "apiLocks",
                "apiActivities",
            ],
            "skip": False,
        },
        "settings": {
            "suites": [
                "apiSettings",
            ],
            "skip": False,
            "withRemotePhp": [True],
            "emailNeeded": True,
            "extraEnvironment": {
                "EMAIL_HOST": "email",
                "EMAIL_PORT": "9000",
            },
            "extraServerEnvironment": {
                "OCIS_ADD_RUN_SERVICES": "notifications",
                "NOTIFICATIONS_SMTP_HOST": "email",
                "NOTIFICATIONS_SMTP_PORT": "2500",
                "NOTIFICATIONS_SMTP_INSECURE": True,
                "NOTIFICATIONS_SMTP_SENDER": "ownCloud <noreply@example.com>",
                "NOTIFICATIONS_DEBUG_ADDR": "0.0.0.0:9174",
            },
        },
        "graph": {
            "suites": [
                "apiGraph",
                "apiServiceAvailability",
            ],
            "skip": False,
            "withRemotePhp": [True],
        },
        "graphUserGroup": {
            "suites": [
                "apiGraphUserGroup",
            ],
            "skip": False,
            "withRemotePhp": [True],
        },
        "spaces": {
            "suites": [
                "apiSpaces",
            ],
            "skip": False,
        },
        "spacesShares": {
            "suites": [
                "apiSpacesShares",
            ],
            "skip": False,
        },
        "spacesDavOperation": {
            "suites": [
                "apiSpacesDavOperation",
            ],
            "skip": False,
        },
        "search1": {
            "suites": [
                "apiSearch1",
            ],
            "skip": False,
        },
        "search2": {
            "suites": [
                "apiSearch2",
            ],
            "skip": False,
        },
        "sharingNg": {
            "suites": [
                "apiReshare",
                "apiSharingNg1",
                "apiSharingNg2",
            ],
            "skip": False,
        },
        "sharingNgShareInvitation": {
            "suites": [
                "apiSharingNgShareInvitation",
            ],
            "skip": False,
        },
        "sharingNgLinkShare": {
            "suites": [
                "apiSharingNgLinkSharePermission",
                "apiSharingNgLinkShareRoot",
            ],
            "skip": False,
        },
        "accountsHashDifficulty": {
            "skip": False,
            "suites": [
                "apiAccountsHashDifficulty",
            ],
            "accounts_hash_difficulty": "default",
        },
        "notification": {
            "suites": [
                "apiNotification",
            ],
            "skip": False,
            "withRemotePhp": [True],
            "emailNeeded": True,
            "extraEnvironment": {
                "EMAIL_HOST": "email",
                "EMAIL_PORT": "9000",
            },
            "extraServerEnvironment": {
                "OCIS_ADD_RUN_SERVICES": "notifications",
                "NOTIFICATIONS_SMTP_HOST": "email",
                "NOTIFICATIONS_SMTP_PORT": "2500",
                "NOTIFICATIONS_SMTP_INSECURE": True,
                "NOTIFICATIONS_SMTP_SENDER": "ownCloud <noreply@example.com>",
                "NOTIFICATIONS_DEBUG_ADDR": "0.0.0.0:9174",
            },
        },
        "antivirus": {
            "suites": [
                "apiAntivirus",
            ],
            "skip": False,
            "antivirusNeeded": True,
            "extraServerEnvironment": {
                "ANTIVIRUS_SCANNER_TYPE": "clamav",
                "ANTIVIRUS_CLAMAV_SOCKET": "tcp://clamav:3310",
                "POSTPROCESSING_STEPS": "virusscan",
                "OCIS_ASYNC_UPLOADS": True,
                "OCIS_ADD_RUN_SERVICES": "antivirus",
                "ANTIVIRUS_DEBUG_ADDR": "0.0.0.0:9297",
            },
        },
        "searchContent": {
            "suites": [
                "apiSearchContent",
            ],
            "skip": False,
            "tikaNeeded": True,
        },
        "ocm": {
            "suites": [
                "apiOcm",
            ],
            "skip": False,
            "withRemotePhp": [True],
            "federationServer": True,
            "emailNeeded": True,
            "extraEnvironment": {
                "EMAIL_HOST": "email",
                "EMAIL_PORT": "9000",
            },
            "extraServerEnvironment": {
                "OCIS_ADD_RUN_SERVICES": "ocm,notifications",
                "OCIS_ENABLE_OCM": True,
                "OCM_OCM_INVITE_MANAGER_INSECURE": True,
                "OCM_OCM_SHARE_PROVIDER_INSECURE": True,
                "OCM_OCM_STORAGE_PROVIDER_INSECURE": True,
                "OCM_OCM_PROVIDER_AUTHORIZER_PROVIDERS_FILE": "%s" % dirs["ocmProviders"],
                # mail notifications
                "NOTIFICATIONS_SMTP_HOST": "email",
                "NOTIFICATIONS_SMTP_PORT": "2500",
                "NOTIFICATIONS_SMTP_INSECURE": True,
                "NOTIFICATIONS_SMTP_SENDER": "ownCloud <noreply@example.com>",
            },
        },
        "wopi": {
            "suites": [
                "apiCollaboration",
            ],
            "skip": False,
            "collaborationServiceNeeded": True,
            "extraServerEnvironment": {
                "GATEWAY_GRPC_ADDR": "0.0.0.0:9142",
            },
        },
        "authApp": {
            "suites": [
                "apiAuthApp",
            ],
            "skip": False,
            "withRemotePhp": [True],
            "extraServerEnvironment": {
                "OCIS_ADD_RUN_SERVICES": "auth-app",
                "PROXY_ENABLE_APP_AUTH": True,
            },
        },
        "cliCommands": {
            "suites": [
                "cliCommands",
            ],
            "skip": False,
            "withRemotePhp": [True],
            "antivirusNeeded": True,
            "extraServerEnvironment": {
                "ANTIVIRUS_SCANNER_TYPE": "clamav",
                "ANTIVIRUS_CLAMAV_SOCKET": "tcp://clamav:3310",
                "OCIS_ASYNC_UPLOADS": True,
                "OCIS_ADD_RUN_SERVICES": "antivirus",
            },
        },
    },
    "apiTests": {
        "numberOfParts": 7,
        "skip": False,
        "skipExceptParts": [],
    },
    "e2eTests": {
        "part": {
            "skip": False,
            "totalParts": 4,  # divide and run all suites in parts (divide pipelines)
            "xsuites": ["search", "app-provider", "oidc", "ocm"],  # suites to skip
        },
        "search": {
            "skip": False,
            "suites": ["search"],  # suites to run
            "tikaNeeded": True,
        },
    },
    "e2eMultiService": {
        "testSuites": {
            "skip": False,
            "suites": [
                "smoke",
                "shares",
                "search",
                "journeys",
                "file-action",
                "spaces",
            ],
            "tikaNeeded": True,
        },
    },
    "rocketchat": {
        "channel": "builds",
        "channel_cron": "builds",
        "from_secret": "rocketchat_talk_webhook",
    },
    "binaryReleases": {
        "os": ["linux", "darwin"],
    },
    "dockerReleases": {
        "architectures": ["arm64", "amd64"],
    },
    "litmus": True,
    "codestyle": True,
}

# workspace for pipeline to cache Go dependencies between steps of a pipeline
# to be used in combination with stepVolumeGo
workspace = \
    {
        "base": "/go",
        "path": "src/github.com/opencloud-eu/opencloud/",
    }

# minio mc environment variables
MINIO_MC_ENV = {
    "CACHE_BUCKET": {
        "from_secret": "cache_s3_bucket",
    },
    "MC_HOST": {
        "from_secret": "cache_s3_server",
    },
    "AWS_ACCESS_KEY_ID": {
        "from_secret": "cache_s3_access_key",
    },
    "AWS_SECRET_ACCESS_KEY": {
        "from_secret": "cache_s3_secret_key",
    },
}

CI_HTTP_PROXY_ENV = {
    "HTTP_PROXY": {
        "from_secret": "ci_http_proxy",
    },
    "HTTPS_PROXY": {
        "from_secret": "ci_http_proxy",
    },
}

def pipelineDependsOn(pipeline, dependant_pipelines):
    if "depends_on" in pipeline.keys():
        pipeline["depends_on"] = pipeline["depends_on"] + getPipelineNames(dependant_pipelines)
    else:
        pipeline["depends_on"] = getPipelineNames(dependant_pipelines)
    return pipeline

def pipelinesDependsOn(pipelines, dependant_pipelines):
    pipes = []
    for pipeline in pipelines:
        pipes.append(pipelineDependsOn(pipeline, dependant_pipelines))

    return pipes

def getPipelineNames(pipelines = []):
    """getPipelineNames returns names of pipelines as a string array

    Args:
      pipelines: array of drone pipelines

    Returns:
      names of the given pipelines as string array
    """
    names = []
    for pipeline in pipelines:
        names.append(pipeline["name"])
    return names

def main(ctx):
    """main is the entrypoint for drone

    Args:
      ctx: drone passes a context with information which the pipeline can be adapted to

    Returns:
      none
    """

    pipelines = []

    build_release_helpers = \
        changelog() + \
        docs() + \
        licenseCheck(ctx)

    test_pipelines = \
        codestyle(ctx) + \
        checkGherkinLint(ctx) + \
        checkTestSuitesInExpectedFailures(ctx) + \
        buildWebCache(ctx) + \
        getGoBinForTesting(ctx) + \
        buildOpencloudBinaryForTesting(ctx) + \
        checkStarlark() + \
        build_release_helpers + \
        testOpencloudAndUploadResults(ctx)

    # testPipelines(ctx)
    # build_release_pipelines =  \
    #      dockerReleases(ctx) + \
    #      binaryReleases(ctx)
    build_release_pipelines = binaryReleases(ctx)

    test_pipelines.append(
        pipelineDependsOn(
            purgeBuildArtifactCache(ctx),
            testPipelines(ctx),
        ),
    )

    pipelines = test_pipelines + build_release_pipelines

    # if ctx.build.event == "cron":
    #     pipelines = \
    #         pipelines + \
    #         example_deploys(ctx)
    # else:
    #     pipelines = \
    #         pipelines + \
    #         pipelinesDependsOn(
    #             example_deploys(ctx),
    #             pipelines,
    #         )

    # always append notification step
    pipelines.append(
        pipelineDependsOn(
            notify(ctx),
            pipelines,
        ),
    )

    pipelineSanityChecks(ctx, pipelines)
    return pipelines

def cachePipeline(name, steps):
    return {
        "name": "build-%s-cache" % name,
        "skip_clone": True,
        "steps": steps,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": ["main", "stable-*"],
            },
            {
                "event": "pull_request",
            },
        ],
    }

def buildWebCache(ctx):
    return [
        cachePipeline("web", generateWebCache(ctx)),
        cachePipeline("web-pnpm", generateWebPnpmCache(ctx)),
    ]

def testOpencloudAndUploadResults(ctx):
    pipeline = testOpencloud(ctx)

    ######################################################################
    # The triggers have been disabled for now, since the govulncheck can #
    # not silence single, acceptable vulnerabilities.                    #
    # See https://github.com/owncloud/ocis/issues/9527 for more details. #
    # FIXME: RE-ENABLE THIS ASAP!!!                                      #
    ######################################################################

    #security_scan = scanOcis(ctx)
    #return [security_scan, pipeline, scan_result_upload]
    return [pipeline]

def testPipelines(ctx):
    pipelines = []

    if config["litmus"]:
        pipelines += litmus(ctx, "ocis")

    if "skip" not in config["cs3ApiTests"] or not config["cs3ApiTests"]["skip"]:
        pipelines.append(cs3ApiTests(ctx, "ocis", "default"))
    if "skip" not in config["wopiValidatorTests"] or not config["wopiValidatorTests"]["skip"]:
        pipelines.append(wopiValidatorTests(ctx, "ocis", "builtin", "default"))
        pipelines.append(wopiValidatorTests(ctx, "ocis", "cs3", "default"))

    pipelines += localApiTestPipeline(ctx)

    if "skip" not in config["apiTests"] or not config["apiTests"]["skip"]:
        pipelines += apiTests(ctx)

    pipelines += e2eTestPipeline(ctx) + multiServiceE2ePipeline(ctx)

    if ("skip" not in config["k6LoadTests"] or not config["k6LoadTests"]["skip"]) and ("k6-test" in ctx.build.title.lower() or ctx.build.event == "cron"):
        pipelines += k6LoadTests(ctx)

    return pipelines

def getGoBinForTesting(ctx):
    return [{
        "name": "get-go-bin-cache",
        "steps": checkGoBinCache() +
                 cacheGoBin(),
        "when": [
            {
                "event": ["push", "manual"],
                "branch": ["main", "stable-*"],
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "unit-tests"),
                },
            },
        ],
        "workspace": workspace,
    }]

def checkGoBinCache():
    return [{
        "name": "check-go-bin-cache",
        "image": OC_UBUNTU,
        "environment": {
            "CACHE_ENDPOINT": {
                "from_secret": "cache_s3_server",
            },
            "CACHE_BUCKET": {
                "from_secret": "cache_s3_bucket",
            },
        },
        "commands": [
            "bash -x %s/tests/config/drone/check_go_bin_cache.sh %s %s" % (dirs["baseGo"], dirs["baseGo"], dirs["gobinTar"]),
        ],
    }]

def cacheGoBin():
    return [
        {
            "name": "bingo-get",
            "image": OC_CI_GOLANG,
            "commands": [
                "make bingo-update",
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
        {
            "name": "archive-go-bin",
            "image": OC_UBUNTU,
            "commands": [
                "tar -czvf %s /go/bin" % dirs["gobinTarPath"],
            ],
        },
        {
            "name": "cache-go-bin",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                # .bingo folder will change after 'bingo-get'
                # so get the stored hash of a .bingo folder
                "BINGO_HASH=$(cat %s/.bingo_hash)" % dirs["baseGo"],
                # cache using the minio client to the public bucket (long term bucket)
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r %s s3/$CACHE_BUCKET/opencloud/go-bin/$BINGO_HASH" % (dirs["gobinTarPath"]),
            ],
        },
    ]

def restoreGoBinCache():
    return [
        {
            "name": "restore-go-bin-cache",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                "BINGO_HASH=$(cat %s/.bingo/* | sha256sum | cut -d ' ' -f 1)" % dirs["baseGo"],
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r -a s3/$CACHE_BUCKET/opencloud/go-bin/$BINGO_HASH/%s %s" % (dirs["gobinTar"], dirs["baseGo"]),
            ],
        },
        {
            "name": "extract-go-bin-cache",
            "image": OC_UBUNTU,
            "commands": [
                "tar -xvmf %s -C /" % dirs["gobinTarPath"],
            ],
        },
    ]

def testOpencloud(ctx):
    steps = restoreGoBinCache() + makeGoGenerate("") + [
        {
            "name": "golangci-lint",
            "image": OC_CI_GOLANG,
            "commands": [
                "mkdir -p cache/checkstyle",
                "make ci-golangci-lint",
                "mv checkstyle.xml cache/checkstyle/checkstyle.xml",
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
        {
            "name": "test",
            "image": OC_CI_GOLANG,
            "environment": CI_HTTP_PROXY_ENV,
            "commands": [
                "mkdir -p cache/coverage",
                "make test",
                "mv coverage.out cache/coverage/",
            ],
        },
        {
            "name": "scan-result-cache",
            "image": PLUGINS_S3,
            "settings": {
                "endpoint": {
                    "from_secret": "cache_s3_server",
                },
                "bucket": "cache",
                "source": "cache/**/*",
                "target": "%s/%s" % (repo_slug, ctx.build.commit + "-${DRONE_BUILD_NUMBER}"),
                "path_style": True,
                "access_key": {
                    "from_secret": "cache_s3_access_key",
                },
                "secret_key": {
                    "from_secret": "cache_s3_secret_key",
                },
            },
        },
    ]

    return {
        "name": "linting_and_unitTests",
        "steps": steps,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "unit-tests"),
                },
            },
        ],
        "depends_on": getPipelineNames(getGoBinForTesting(ctx)),
        "workspace": workspace,
    }

def scanOcis(ctx):
    steps = restoreGoBinCache() + makeGoGenerate("") + [
        {
            "name": "govulncheck",
            "image": OC_CI_GOLANG,
            "commands": [
                "make govulncheck",
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
    ]

    return {
        "name": "go-vulnerability-scanning",
        "steps": steps,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                },
            },
        ],
        "depends_on": getPipelineNames(getGoBinForTesting(ctx)),
        "workspace": workspace,
    }

def buildOpencloudBinaryForTesting(ctx):
    return [{
        "name": "build_opencloud_binary_for_testing",
        "steps": makeNodeGenerate("") +
                 makeGoGenerate("") +
                 build() +
                 rebuildBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]),
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "unit-tests"),
                },
            },
        ],
        "workspace": workspace,
    }]

def vendorbinCodestyle(phpVersion):
    return [{
        "name": "vendorbin-codestyle",
        "image": OC_CI_PHP % phpVersion,
        "environment": {
            "COMPOSER_HOME": "%s/.cache/composer" % dirs["base"],
        },
        "commands": [
            "make vendor-bin-codestyle",
        ],
    }]

def vendorbinCodesniffer(phpVersion):
    return [{
        "name": "vendorbin-codesniffer",
        "image": OC_CI_PHP % phpVersion,
        "environment": {
            "COMPOSER_HOME": "%s/.cache/composer" % dirs["base"],
        },
        "commands": [
            "make vendor-bin-codesniffer",
        ],
    }]

def checkTestSuitesInExpectedFailures(ctx):
    return [{
        "name": "check-suites-in-expected-failures",
        "steps": [
            {
                "name": "check-suites",
                "image": OC_CI_ALPINE,
                "commands": [
                    "%s/tests/acceptance/check-deleted-suites-in-expected-failure.sh" % dirs["base"],
                ],
            },
        ],
        "when": [
            {
                "event": "pull_request",
            },
        ],
    }]

def checkGherkinLint(ctx):
    return [{
        "name": "check-gherkin-standard",
        "steps": [
            {
                "name": "lint-feature-files",
                "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
                "commands": [
                    "npm install -g @gherlint/gherlint@1.1.0",
                    "make test-gherkin-lint",
                ],
            },
        ],
        "when": [
            {
                "event": "pull_request",
            },
        ],
    }]

def codestyle(ctx):
    pipelines = []

    if "codestyle" not in config:
        return []

    default = {
        "phpVersions": [DEFAULT_PHP_VERSION],
    }

    if "defaults" in config:
        if "codestyle" in config["defaults"]:
            for item in config["defaults"]["codestyle"]:
                default[item] = config["defaults"]["codestyle"][item]

    codestyleConfig = config["codestyle"]

    if type(codestyleConfig) == "bool":
        if codestyleConfig:
            # the config has 'codestyle' true, so specify an empty dict that will get the defaults
            codestyleConfig = {}
        else:
            return pipelines

    if len(codestyleConfig) == 0:
        # 'codestyle' is an empty dict, so specify a single section that will get the defaults
        codestyleConfig = {"doDefault": {}}

    for category, matrix in codestyleConfig.items():
        params = {}
        for item in default:
            params[item] = matrix[item] if item in matrix else default[item]

        for phpVersion in params["phpVersions"]:
            name = "coding-standard-php%s" % phpVersion

            result = {
                "name": name,
                "steps": vendorbinCodestyle(phpVersion) +
                         vendorbinCodesniffer(phpVersion) +
                         [
                             {
                                 "name": "php-style",
                                 "image": OC_CI_PHP % phpVersion,
                                 "commands": [
                                     "make test-php-style",
                                 ],
                             },
                             {
                                 "name": "check-env-var-annotations",
                                 "image": OC_CI_PHP % phpVersion,
                                 "commands": [
                                     "make check-env-var-annotations",
                                 ],
                             },
                         ],
                "depends_on": [],
                "when": [
                    {
                        "event": ["push", "manual"],
                        "branch": "main",
                    },
                    {
                        "event": "pull_request",
                        "path": {
                            "exclude": skipIfUnchanged(ctx, "lint"),
                        },
                    },
                ],
            }

            pipelines.append(result)

    return pipelines

def localApiTestPipeline(ctx):
    pipelines = []

    with_remote_php = [True]
    if ctx.build.event == "cron" or "full-ci" in ctx.build.title.lower():
        with_remote_php.append(False)

    defaults = {
        "suites": {},
        "skip": False,
        "extraEnvironment": {},
        "extraServerEnvironment": {},
        "storages": ["ocis"],
        "accounts_hash_difficulty": 4,
        "emailNeeded": False,
        "antivirusNeeded": False,
        "tikaNeeded": False,
        "federationServer": False,
        "collaborationServiceNeeded": False,
        "extraCollaborationEnvironment": {},
        "withRemotePhp": with_remote_php,
    }

    if "localApiTests" in config:
        for name, matrix in config["localApiTests"].items():
            if "skip" not in matrix or not matrix["skip"]:
                params = {}
                for item in defaults:
                    params[item] = matrix[item] if item in matrix else defaults[item]
                for storage in params["storages"]:
                    for run_with_remote_php in params["withRemotePhp"]:
                        pipeline = {
                            "name": "%s-%s%s" % ("CLI" if name.startswith("cli") else "API", name, "-withoutRemotePhp" if not run_with_remote_php else ""),
                            "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                                     (tikaService() if params["tikaNeeded"] else []) +
                                     (waitForServices("online-offices", ["collabora:9980", "onlyoffice:443", "fakeoffice:8080"]) if params["collaborationServiceNeeded"] else []) +
                                     ocisServer(storage, params["accounts_hash_difficulty"], extra_server_environment = params["extraServerEnvironment"], with_wrapper = True, tika_enabled = params["tikaNeeded"]) +
                                     (waitForClamavService() if params["antivirusNeeded"] else []) +
                                     (waitForEmailService() if params["emailNeeded"] else []) +
                                     (ocisServer(storage, params["accounts_hash_difficulty"], deploy_type = "federation", extra_server_environment = params["extraServerEnvironment"]) if params["federationServer"] else []) +
                                     ((wopiCollaborationService("fakeoffice") + wopiCollaborationService("collabora") + wopiCollaborationService("onlyoffice")) if params["collaborationServiceNeeded"] else []) +
                                     (ocisHealthCheck("wopi", ["wopi-collabora:9304", "wopi-onlyoffice:9304", "wopi-fakeoffice:9304"]) if params["collaborationServiceNeeded"] else []) +
                                     localApiTests(ctx, name, params["suites"], storage, params["extraEnvironment"], run_with_remote_php) +
                                     logRequests(),
                            "services": (emailService() if params["emailNeeded"] else []) +
                                        (clamavService() if params["antivirusNeeded"] else []) +
                                        ((fakeOffice() + collaboraService() + onlyofficeService()) if params["collaborationServiceNeeded"] else []),
                            "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
                            "when": [
                                {
                                    "event": ["push", "manual"],
                                    "branch": "main",
                                },
                                {
                                    "event": "pull_request",
                                    "path": {
                                        "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                                    },
                                },
                            ],
                        }
                        pipelines.append(pipeline)
    return pipelines

def localApiTests(ctx, name, suites, storage = "ocis", extra_environment = {}, with_remote_php = False):
    test_dir = "%s/tests/acceptance" % dirs["base"]
    expected_failures_file = "%s/expected-failures-localAPI-on-%s-storage.md" % (test_dir, storage.upper())

    environment = {
        "TEST_SERVER_URL": OCIS_URL,
        "TEST_SERVER_FED_URL": OC_FED_URL,
        "OCIS_REVA_DATA_ROOT": "%s" % (dirs["ocisRevaDataRoot"] if storage == "owncloud" else ""),
        "SEND_SCENARIO_LINE_REFERENCES": True,
        "STORAGE_DRIVER": storage,
        "BEHAT_SUITES": ",".join(suites),
        "BEHAT_FILTER_TAGS": "~@skip&&~@skipOnGraph&&~@skipOnOcis-%s-Storage" % ("OC" if storage == "owncloud" else "OCIS"),
        "EXPECTED_FAILURES_FILE": expected_failures_file,
        "UPLOAD_DELETE_WAIT_TIME": "1" if storage == "owncloud" else 0,
        "OCIS_WRAPPER_URL": "http://%s:5200" % OC_SERVER_NAME,
        "WITH_REMOTE_PHP": with_remote_php,
        "COLLABORATION_SERVICE_URL": "http://wopi-fakeoffice:9300",
    }

    for item in extra_environment:
        environment[item] = extra_environment[item]

    return [{
        "name": "localApiTests-%s" % name,
        "image": OC_CI_PHP % DEFAULT_PHP_VERSION,
        "environment": environment,
        "commands": [
            # merge the expected failures
            "" if with_remote_php else "cat %s/expected-failures-without-remotephp.md >> %s" % (test_dir, expected_failures_file),
            "make -C %s test-acceptance-api" % (dirs["base"]),
        ],
    }]

def cs3ApiTests(ctx, storage, accounts_hash_difficulty = 4):
    return {
        "name": "cs3ApiTests",
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 ocisServer(storage, accounts_hash_difficulty, [], [], "cs3api_validator") +
                 [
                     {
                         "name": "cs3ApiTests",
                         "image": OC_CS3_API_VALIDATOR,
                         "environment": {},
                         "commands": [
                             "/usr/bin/cs3api-validator /var/lib/cs3api-validator --endpoint=%s:9142" % OC_SERVER_NAME,
                         ],
                     },
                 ],
        "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                },
            },
        ],
    }

def wopiValidatorTests(ctx, storage, wopiServerType, accounts_hash_difficulty = 4):
    testgroups = [
        "BaseWopiViewing",
        "CheckFileInfoSchema",
        "EditFlows",
        "Locks",
        "AccessTokens",
        "GetLock",
        "ExtendedLockLength",
        "FileVersion",
        "Features",
    ]
    builtinOnlyTestGroups = [
        "PutRelativeFile",
        "RenameFileIfCreateChildFileIsNotSupported",
    ]

    validatorTests = []
    wopiServer = []
    extra_server_environment = {}

    if wopiServerType == "cs3":
        wopiServer = [
            {
                "name": "wopi-fakeoffice",
                "image": "cs3org/wopiserver:v10.4.0",
                "detach": True,
                "commands": [
                    "cp %s/tests/config/drone/wopiserver.conf /etc/wopi/wopiserver.conf" % (dirs["base"]),
                    "echo 123 > /etc/wopi/wopisecret",
                    "/app/wopiserver.py",
                ],
            },
        ]
    else:
        extra_server_environment = {
            "OCIS_EXCLUDE_RUN_SERVICES": "app-provider",
        }

        wopiServer = wopiCollaborationService("fakeoffice")

    for testgroup in testgroups:
        validatorTests.append({
            "name": "wopiValidatorTests-%s" % testgroup,
            "image": "owncloudci/wopi-validator",
            "commands": [
                "export WOPI_TOKEN=$(cat accesstoken)",
                "echo $WOPI_TOKEN",
                "export WOPI_TTL=$(cat accesstokenttl)",
                "echo $WOPI_TTL",
                "export WOPI_SRC=$(cat wopisrc)",
                "echo $WOPI_SRC",
                "cd /app",
                "/app/Microsoft.Office.WopiValidator -t $WOPI_TOKEN -w $WOPI_SRC -l $WOPI_TTL --testgroup %s" % testgroup,
            ],
        })
    if wopiServerType == "builtin":
        for builtinOnlyGroup in builtinOnlyTestGroups:
            validatorTests.append({
                "name": "wopiValidatorTests-%s" % builtinOnlyGroup,
                "image": "owncloudci/wopi-validator",
                "commands": [
                    "export WOPI_TOKEN=$(cat accesstoken)",
                    "echo $WOPI_TOKEN",
                    "export WOPI_TTL=$(cat accesstokenttl)",
                    "echo $WOPI_TTL",
                    "export WOPI_SRC=$(cat wopisrc)",
                    "echo $WOPI_SRC",
                    "cd /app",
                    "/app/Microsoft.Office.WopiValidator -s -t $WOPI_TOKEN -w $WOPI_SRC -l $WOPI_TTL --testgroup %s" % builtinOnlyGroup,
                ],
            })

    return {
        "name": "wopiValidatorTests-%s" % wopiServerType,
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 fakeOffice() +
                 waitForServices("fake-office", ["fakeoffice:8080"]) +
                 ocisServer(storage, accounts_hash_difficulty, deploy_type = "wopi_validator", extra_server_environment = extra_server_environment) +
                 wopiServer +
                 waitForServices("wopi-fakeoffice", ["wopi-fakeoffice:9300"]) +
                 [
                     {
                         "name": "prepare-test-file",
                         "image": OC_CI_ALPINE,
                         "environment": {},
                         "commands": [
                             "curl -v -X PUT '%s/remote.php/webdav/test.wopitest' -k --fail --retry-connrefused --retry 7 --retry-all-errors -u admin:admin -D headers.txt" % OCIS_URL,
                             "cat headers.txt",
                             "export FILE_ID=$(cat headers.txt | sed -n -e 's/^.*Oc-Fileid: //p')",
                             "export URL=\"%s/app/open?app_name=FakeOffice&file_id=$FILE_ID\"" % OCIS_URL,
                             "export URL=$(echo $URL | tr -d '[:cntrl:]')",
                             "curl -v -X POST \"$URL\" -k --fail --retry-connrefused --retry 7 --retry-all-errors -u admin:admin > open.json",
                             "cat open.json",
                             "cat open.json | jq .form_parameters.access_token | tr -d '\"' > accesstoken",
                             "cat open.json | jq .form_parameters.access_token_ttl | tr -d '\"' > accesstokenttl",
                             "echo -n 'http://wopi-fakeoffice:9300/wopi/files/' > wopisrc",
                             "cat open.json | jq .app_url | sed -n -e 's/^.*files%2F//p' | tr -d '\"' >> wopisrc",
                         ],
                     },
                 ] +
                 validatorTests,
        "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                },
            },
        ],
    }

def coreApiTests(ctx, part_number = 1, number_of_parts = 1, with_remote_php = False, storage = "ocis", accounts_hash_difficulty = 4):
    filterTags = "~@skipOnGraph&&~@skipOnOcis-%s-Storage" % ("OC" if storage == "owncloud" else "OCIS")
    test_dir = "%s/tests/acceptance" % dirs["base"]
    expected_failures_file = "%s/expected-failures-API-on-%s-storage.md" % (test_dir, storage.upper())

    return {
        "name": "Core-API-Tests-%s%s" % (part_number, "-withoutRemotePhp" if not with_remote_php else ""),
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 ocisServer(storage, accounts_hash_difficulty, with_wrapper = True) +
                 [
                     {
                         "name": "oC10ApiTests-%s" % part_number,
                         "image": OC_CI_PHP % DEFAULT_PHP_VERSION,
                         "environment": {
                             "TEST_SERVER_URL": OCIS_URL,
                             "OCIS_REVA_DATA_ROOT": "%s" % (dirs["ocisRevaDataRoot"] if storage == "owncloud" else ""),
                             "SEND_SCENARIO_LINE_REFERENCES": True,
                             "STORAGE_DRIVER": storage,
                             "BEHAT_FILTER_TAGS": filterTags,
                             "DIVIDE_INTO_NUM_PARTS": number_of_parts,
                             "RUN_PART": part_number,
                             "ACCEPTANCE_TEST_TYPE": "core-api",
                             "EXPECTED_FAILURES_FILE": expected_failures_file,
                             "UPLOAD_DELETE_WAIT_TIME": "1" if storage == "owncloud" else 0,
                             "OCIS_WRAPPER_URL": "http://%s:5200" % OC_SERVER_NAME,
                             "WITH_REMOTE_PHP": with_remote_php,
                         },
                         "commands": [
                             # merge the expected failures
                             "" if with_remote_php else "cat %s/expected-failures-without-remotephp.md >> %s" % (test_dir, expected_failures_file),
                             "make -C %s test-acceptance-api" % (dirs["base"]),
                         ],
                     },
                 ] +
                 logRequests(),
        "services": redisForOCStorage(storage),
        "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                },
            },
        ],
    }

def apiTests(ctx):
    pipelines = []
    debugParts = config["apiTests"]["skipExceptParts"]
    debugPartsEnabled = (len(debugParts) != 0)

    with_remote_php = [True]
    if ctx.build.event == "cron" or "full-ci" in ctx.build.title.lower():
        with_remote_php.append(False)

    defaults = {
        "withRemotePhp": with_remote_php,
    }

    for runPart in range(1, config["apiTests"]["numberOfParts"] + 1):
        for run_with_remote_php in defaults["withRemotePhp"]:
            if (not debugPartsEnabled or (debugPartsEnabled and runPart in debugParts)):
                pipelines.append(coreApiTests(ctx, runPart, config["apiTests"]["numberOfParts"], run_with_remote_php))

    return pipelines

def e2eTestPipeline(ctx):
    defaults = {
        "skip": False,
        "suites": [],
        "xsuites": [],
        "totalParts": 0,
        "tikaNeeded": False,
    }

    extra_server_environment = {
        "OCIS_PASSWORD_POLICY_BANNED_PASSWORDS_LIST": "%s" % dirs["bannedPasswordList"],
    }

    e2e_trigger = [
        {
            "event": ["push", "manual"],
            "branch": "main",
        },
        {
            "event": "pull_request",
            "path": {
                "exclude": skipIfUnchanged(ctx, "e2e-tests"),
            },
        },
        {
            "event": "tag",
            "ref": "refs/tags/**",
        },
    ]

    pipelines = []

    if ("skip-e2e" in ctx.build.title.lower()):
        return pipelines

    if (ctx.build.event == "tag"):
        return pipelines

    for name, suite in config["e2eTests"].items():
        if "skip" in suite and suite["skip"]:
            continue

        params = {}
        for item in defaults:
            params[item] = suite[item] if item in suite else defaults[item]

        e2e_args = ""
        if params["totalParts"] > 0:
            e2e_args = "--total-parts %d" % params["totalParts"]
        elif params["suites"]:
            e2e_args = "--suites %s" % ",".join(params["suites"])

        # suites to skip
        if params["xsuites"]:
            e2e_args += " --xsuites %s" % ",".join(params["xsuites"])

        steps_before = \
            restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBin"]) + \
            restoreWebCache() + \
            restoreWebPnpmCache() + \
            (tikaService() if params["tikaNeeded"] else []) + \
            ocisServer(extra_server_environment = extra_server_environment, tika_enabled = params["tikaNeeded"])

        step_e2e = {
            "name": "e2e-tests",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "environment": {
                "BASE_URL_OCIS": OCIS_DOMAIN,
                "HEADLESS": True,
                "RETRY": "1",
                "WEB_UI_CONFIG_FILE": "%s/%s" % (dirs["base"], dirs["ocisConfig"]),
                "LOCAL_UPLOAD_DIR": "/uploads",
            },
            "commands": [
                "cd %s/tests/e2e" % dirs["web"],
            ],
        }

        # steps_after = uploadTracingResult(ctx) + \
        steps_after = logTracingResults()

        if params["totalParts"]:
            for index in range(params["totalParts"]):
                run_part = index + 1
                run_e2e = {}
                run_e2e.update(step_e2e)
                run_e2e["commands"] = [
                    "cd %s/tests/e2e" % dirs["web"],
                    "bash run-e2e.sh %s --run-part %d" % (e2e_args, run_part),
                ]
                pipelines.append({
                    "name": "e2e-tests-%s-%s" % (name, run_part),
                    "steps": steps_before + [run_e2e] + steps_after,
                    "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx) + buildWebCache(ctx)),
                    "when": e2e_trigger,
                })
        else:
            step_e2e["commands"].append("bash run-e2e.sh %s" % e2e_args)
            pipelines.append({
                "name": "e2e-tests-%s" % name,
                "steps": steps_before + [step_e2e] + steps_after,
                "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx) + buildWebCache(ctx)),
                "when": e2e_trigger,
            })

    return pipelines

def multiServiceE2ePipeline(ctx):
    pipelines = []

    defaults = {
        "skip": False,
        "suites": [],
        "xsuites": [],
        "tikaNeeded": False,
    }

    e2e_trigger = {
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "e2e-tests"),
                },
            },
        ],
    }

    if ("skip-e2e" in ctx.build.title.lower()):
        return pipelines

    # run this pipeline only for cron jobs and full-ci PRs
    if (not "full-ci" in ctx.build.title.lower() and ctx.build.event != "cron"):
        return pipelines

    extra_server_environment = {
        "OCIS_PASSWORD_POLICY_BANNED_PASSWORDS_LIST": "%s" % dirs["bannedPasswordList"],
        "OCIS_JWT_SECRET": "some-ocis-jwt-secret",
        "OCIS_SERVICE_ACCOUNT_ID": "service-account-id",
        "OCIS_SERVICE_ACCOUNT_SECRET": "service-account-secret",
        "OCIS_EXCLUDE_RUN_SERVICES": "storage-users",
        "OCIS_GATEWAY_GRPC_ADDR": "0.0.0.0:9142",
        "SETTINGS_GRPC_ADDR": "0.0.0.0:9191",
        "GATEWAY_STORAGE_USERS_MOUNT_ID": "storage-users-id",
    }

    storage_users_environment = {
        "OCIS_CORS_ALLOW_ORIGINS": "%s,https://%s:9201" % (OCIS_URL, OC_SERVER_NAME),
        "STORAGE_USERS_JWT_SECRET": "some-ocis-jwt-secret",
        "STORAGE_USERS_MOUNT_ID": "storage-users-id",
        "STORAGE_USERS_SERVICE_ACCOUNT_ID": "service-account-id",
        "STORAGE_USERS_SERVICE_ACCOUNT_SECRET": "service-account-secret",
        "STORAGE_USERS_GATEWAY_GRPC_ADDR": "%s:9142" % OC_SERVER_NAME,
        "STORAGE_USERS_EVENTS_ENDPOINT": "%s:9233" % OC_SERVER_NAME,
        "STORAGE_USERS_DATA_GATEWAY_URL": "%s/data" % OCIS_URL,
        "OCIS_CACHE_STORE": "nats-js-kv",
        "OCIS_CACHE_STORE_NODES": "%s:9233" % OC_SERVER_NAME,
        "MICRO_REGISTRY_ADDRESS": "%s:9233" % OC_SERVER_NAME,
    }
    storage_users1_environment = {
        "STORAGE_USERS_GRPC_ADDR": "storageusers1:9157",
        "STORAGE_USERS_HTTP_ADDR": "storageusers1:9158",
        "STORAGE_USERS_DEBUG_ADDR": "storageusers1:9159",
        "STORAGE_USERS_DATA_SERVER_URL": "http://storageusers1:9158/data",
    }
    for item in storage_users_environment:
        storage_users1_environment[item] = storage_users_environment[item]

    storage_users2_environment = {
        "STORAGE_USERS_GRPC_ADDR": "storageusers2:9157",
        "STORAGE_USERS_HTTP_ADDR": "storageusers2:9158",
        "STORAGE_USERS_DEBUG_ADDR": "storageusers2:9159",
        "STORAGE_USERS_DATA_SERVER_URL": "http://storageusers2:9158/data",
    }
    for item in storage_users_environment:
        storage_users2_environment[item] = storage_users_environment[item]

    storage_volume = [{
        "name": "storage",
        "path": "/root/.ocis",
    }]
    storage_users_services = startOcisService("storage-users", "storageusers1", storage_users1_environment, storage_volume) + \
                             startOcisService("storage-users", "storageusers2", storage_users2_environment, storage_volume) + \
                             ocisHealthCheck("storage-users", ["storageusers1:9159", "storageusers2:9159"])

    for _, suite in config["e2eMultiService"].items():
        if "skip" in suite and suite["skip"]:
            continue

        params = {}
        for item in defaults:
            params[item] = suite[item] if item in suite else defaults[item]

        e2e_args = ""
        if params["suites"]:
            e2e_args = "--suites %s" % ",".join(params["suites"])

        # suites to skip
        if params["xsuites"]:
            e2e_args += " --xsuites %s" % ",".join(params["xsuites"])

        steps = \
            restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBin"]) + \
            restoreWebCache() + \
            restoreWebPnpmCache() + \
            tikaService() + \
            ocisServer(extra_server_environment = extra_server_environment, tika_enabled = params["tikaNeeded"]) + \
            storage_users_services + \
            [{
                "name": "e2e-tests",
                "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
                "environment": {
                    "BASE_URL_OCIS": OCIS_DOMAIN,
                    "HEADLESS": True,
                    "RETRY": "1",
                },
                "commands": [
                    "cd %s/tests/e2e" % dirs["web"],
                    "bash run-e2e.sh %s" % e2e_args,
                ],
            }] + logTracingResults()

        # uploadTracingResult(ctx) + \
        pipelines.append({
            "name": "e2e-tests-multi-service",
            "steps": steps,
            "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx) + buildWebCache(ctx)),
            "workspace": e2e_trigger,
        })
    return pipelines

def uploadTracingResult(ctx):
    return [{
        "name": "upload-tracing-result",
        "image": PLUGINS_S3,
        "settings": {
            "bucket": {
                "from_secret": "cache_public_s3_bucket",
            },
            "endpoint": {
                "from_secret": "cache_public_s3_server",
            },
            "path_style": True,
            "source": "webTestRunner/reports/e2e/playwright/tracing/**/*",
            "strip_prefix": "webTestRunner/reports/e2e/playwright/tracing",
            "target": "/${DRONE_REPO}/${DRONE_BUILD_NUMBER}/tracing",
        },
        "environment": {
            "AWS_ACCESS_KEY_ID": {
                "from_secret": "cache_public_s3_access_key",
            },
            "AWS_SECRET_ACCESS_KEY": {
                "from_secret": "cache_public_s3_secret_key",
            },
        },
        "when": {
            "status": [
                "failure",
            ],
            "event": [
                "pull_request",
                "cron",
            ],
        },
    }]

def logTracingResults():
    return [{
        "name": "log-tracing-result",
        "image": OC_UBUNTU,
        "commands": [
            "cd %s/reports/e2e/playwright/tracing/" % dirs["web"],
            'echo "To see the trace, please open the following link in the console"',
            'for f in *.zip; do echo "npx playwright show-trace https://cache.owncloud.com/public/${DRONE_REPO}/${DRONE_BUILD_NUMBER}/tracing/$f \n"; done',
        ],
        "when": {
            "status": [
                "failure",
            ],
            "event": [
                "pull_request",
                "cron",
            ],
        },
    }]

def dockerReleases(ctx):
    pipelines = []
    docker_repos = []
    build_type = "daily"

    # dockerhub repo
    #  - "owncloud/ocis-rolling"
    repo = docker_repo_slug + "-rolling"
    docker_repos.append(repo)

    # production release repo
    if ctx.build.event == "tag":
        tag = ctx.build.ref.replace("refs/tags/v", "").lower()
        for prod_tag in PRODUCTION_RELEASE_TAGS:
            if tag.startswith(prod_tag):
                docker_repos.append(docker_repo_slug)
                break

    for repo in docker_repos:
        repo_pipelines = []
        if ctx.build.event == "tag":
            build_type = "rolling" if "rolling" in repo else "production"

        for arch in config["dockerReleases"]["architectures"]:
            repo_pipelines.append(dockerRelease(ctx, arch, repo, build_type))

        manifest = releaseDockerManifest(ctx, repo, build_type)
        manifest["depends_on"] = getPipelineNames(repo_pipelines)
        repo_pipelines.append(manifest)

        readme = releaseDockerReadme(ctx, repo, build_type)
        readme["depends_on"] = getPipelineNames(repo_pipelines)
        repo_pipelines.append(readme)

        pipelines.extend(repo_pipelines)

    return pipelines

def dockerRelease(ctx, arch, repo, build_type):
    build_args = [
        "REVISION=%s" % (ctx.build.commit),
        "VERSION=%s" % (ctx.build.ref.replace("refs/tags/", "") if ctx.build.event == "tag" else "master"),
    ]
    depends_on = getPipelineNames(testOpencloudAndUploadResults(ctx) + testPipelines(ctx))

    if ctx.build.event == "tag":
        depends_on = []

    return {
        "name": "docker-%s-%s" % (arch, build_type),
        "labels": {
            "platform": "linux/%s" % arch,
        },
        "steps": makeNodeGenerate("") +
                 makeGoGenerate("") + [
            {
                "name": "build",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make -C opencloud release-linux-docker-%s ENABLE_VIPS=true" % (arch),
                ],
            },
            {
                "name": "dryrun",
                "image": PLUGINS_DOCKER,
                "settings": {
                    "dry_run": True,
                    "context": "ocis",
                    "tags": "linux-%s" % (arch),
                    "dockerfile": "ocis/docker/Dockerfile.linux.%s" % (arch),
                    "repo": repo,
                    "build_args": build_args,
                },
                "when": {
                    "ref": {
                        "include": [
                            "refs/pull/**",
                        ],
                    },
                },
            },
            {
                "name": "docker",
                "image": PLUGINS_DOCKER,
                "settings": {
                    "username": {
                        "from_secret": "docker_username",
                    },
                    "password": {
                        "from_secret": "docker_password",
                    },
                    "auto_tag": True,
                    "context": "ocis",
                    "auto_tag_suffix": "linux-%s" % (arch),
                    "dockerfile": "ocis/docker/Dockerfile.linux.%s" % (arch),
                    "repo": repo,
                    "build_args": build_args,
                },
                "when": [
                    {
                        "event": ["push", "manual"],
                        "branch": "main",
                    },
                    {
                        "event": "tag",
                    },
                ],
            },
        ],
        "depends_on": depends_on,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "build-docker"),
                },
            },
            {
                "event": "tag",
            },
        ],
    }

def binaryReleases(ctx):
    pipelines = []
    targets = []
    build_type = "daily"

    # uploads binary to https://download.owncloud.com/ocis/ocis/daily/
    target = "/ocis/%s/daily" % (ctx.repo.name.replace("ocis-", ""))
    depends_on = getPipelineNames(testOpencloudAndUploadResults(ctx) + testPipelines(ctx))

    if ctx.build.event == "tag":
        depends_on = []

        buildref = ctx.build.ref.replace("refs/tags/v", "").lower()
        target_path = "/ocis/%s" % ctx.repo.name.replace("ocis-", "")

        if buildref.find("-") != -1:  # "x.x.x-alpha", "x.x.x-beta", "x.x.x-rc"
            folder = "testing"
            target = "%s/%s/%s" % (target_path, folder, buildref)
            targets.append(target)
            build_type = "testing"
        else:
            # uploads binary to eg. https://download.owncloud.com/ocis/ocis/rolling/1.0.0/
            folder = "rolling"
            target = "%s/%s/%s" % (target_path, folder, buildref)
            targets.append(target)

            for prod_tag in PRODUCTION_RELEASE_TAGS:
                if buildref.startswith(prod_tag):
                    # uploads binary to eg. https://download.owncloud.com/ocis/ocis/stable/2.0.0/
                    folder = "stable"
                    target = "%s/%s/%s" % (target_path, folder, buildref)
                    targets.append(target)
                    break

    else:
        targets.append(target)

    for target in targets:
        if "rolling" in target:
            build_type = "rolling"
        elif "stable" in target:
            build_type = "production"
        elif "testing" in target:
            build_type = "testing"

        for os in config["binaryReleases"]["os"]:
            pipelines.append(binaryRelease(ctx, os, build_type, target, depends_on))

    return pipelines

def binaryRelease(ctx, arch, build_type, target, depends_on = []):
    settings = {
        "endpoint": {
            "from_secret": "upload_s3_endpoint",
        },
        "access_key": {
            "from_secret": "upload_s3_access_key",
        },
        "secret_key": {
            "from_secret": "upload_s3_secret_key",
        },
        "bucket": {
            "from_secret": "upload_s3_bucket",
        },
        "path_style": True,
        "strip_prefix": "ocis/dist/release/",
        "source": "ocis/dist/release/*",
        "target": target,
    }

    return {
        "name": "binaries-%s-%s" % (arch, build_type),
        "steps": makeNodeGenerate("") +
                 makeGoGenerate("") + [
            {
                "name": "build",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make -C opencloud release-%s" % (arch),
                ],
            },
            {
                "name": "finish",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make -C opencloud release-finish",
                    "cp assets/End-User-License-Agreement-for-ownCloud-Infinite-Scale.pdf ocis/dist/release/",
                ],
                "when": [
                    {
                        "event": ["push", "manual"],
                        "branch": "main",
                    },
                    {
                        "event": "tag",
                    },
                ],
            },
            {
                "name": "upload",
                "image": PLUGINS_S3,
                "settings": settings,
                "when": [
                    {
                        "event": ["push", "manual"],
                        "branch": "main",
                    },
                    {
                        "event": "tag",
                    },
                ],
            },
            {
                "name": "changelog",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make changelog CHANGELOG_VERSION=%s" % ctx.build.ref.replace("refs/tags/v", ""),
                ],
                "when": [
                    {
                        "event": "tag",
                    },
                ],
            },
            {
                "name": "release",
                "image": PLUGINS_GITHUB_RELEASE,
                "settings": {
                    "api_key": {
                        "from_secret": "github_token",
                    },
                    "files": [
                        "ocis/dist/release/*",
                    ],
                    "title": ctx.build.ref.replace("refs/tags/v", ""),
                    "note": "ocis/dist/CHANGELOG.md",
                    "overwrite": True,
                    "prerelease": len(ctx.build.ref.split("-")) > 1,
                },
                "when": [
                    {
                        "event": "tag",
                    },
                ],
            },
        ],
        "depends_on": depends_on,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "build-binary"),
                },
            },
            {
                "event": "tag",
            },
        ],
    }

def licenseCheck(ctx):
    # uploads third-party-licenses to https://download.owncloud.com/ocis/ocis/daily/
    target = "/ocis/%s/daily" % (ctx.repo.name.replace("ocis-", ""))
    if ctx.build.event == "tag":
        # uploads third-party-licenses to eg. https://download.owncloud.com/ocis/ocis/1.0.0-beta9/
        folder = "stable"
        buildref = ctx.build.ref.replace("refs/tags/v", "")
        buildref = buildref.lower()
        if buildref.find("-") != -1:  # "x.x.x-alpha", "x.x.x-beta", "x.x.x-rc"
            folder = "testing"
        target = "/ocis/%s/%s/%s" % (ctx.repo.name.replace("ocis-", ""), folder, buildref)

    settings = {
        "endpoint": {
            "from_secret": "upload_s3_endpoint",
        },
        "access_key": {
            "from_secret": "upload_s3_access_key",
        },
        "secret_key": {
            "from_secret": "upload_s3_secret_key",
        },
        "bucket": {
            "from_secret": "upload_s3_bucket",
        },
        "path_style": True,
        "source": "third-party-licenses.tar.gz",
        "target": target,
    }

    return [{
        "name": "check-licenses",
        "steps": [
            {
                "name": "node-check-licenses",
                "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
                "commands": [
                    "make ci-node-check-licenses",
                ],
            },
            {
                "name": "node-save-licenses",
                "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
                "commands": [
                    "make ci-node-save-licenses",
                ],
            },
            {
                "name": "go-check-licenses",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make ci-go-check-licenses",
                ],
            },
            {
                "name": "go-save-licenses",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make ci-go-save-licenses",
                ],
            },
            {
                "name": "tarball",
                "image": OC_CI_ALPINE,
                "commands": [
                    "cd third-party-licenses && tar -czf ../third-party-licenses.tar.gz *",
                ],
            },
            {
                "name": "upload",
                "image": PLUGINS_S3,
                "settings": settings,
                "when": [
                    {
                        "event": "tag",
                    },
                    {
                        "event": ["push", "manual"],
                        "branch": "main",
                    },
                ],
            },
            {
                "name": "changelog",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make changelog CHANGELOG_VERSION=%s" % ctx.build.ref.replace("refs/tags/v", "").split("-")[0],
                ],
                "when": [
                    {
                        "event": "tag",
                    },
                ],
            },
            {
                "name": "release",
                "image": PLUGINS_GITHUB_RELEASE,
                "settings": {
                    "api_key": {
                        "from_secret": "github_token",
                    },
                    "files": [
                        "third-party-licenses.tar.gz",
                    ],
                    "title": ctx.build.ref.replace("refs/tags/v", ""),
                    "note": "ocis/dist/CHANGELOG.md",
                    "overwrite": True,
                    "prerelease": len(ctx.build.ref.split("-")) > 1,
                },
                "when": [
                    {
                        "event": "tag",
                    },
                ],
            },
        ],
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
            },
            {
                "event": "tag",
            },
        ],
        "workspace": workspace,
    }]

def releaseDockerManifest(ctx, repo, build_type):
    spec = "manifest.tmpl"
    spec_latest = "manifest-latest.tmpl"
    if "rolling" not in repo:
        spec = "manifest.production.tmpl"
        spec_latest = "manifest.production-latest.tmpl"

    steps = [
        {
            "name": "execute",
            "image": PLUGINS_MANIFEST,
            "settings": {
                "username": {
                    "from_secret": "docker_username",
                },
                "password": {
                    "from_secret": "docker_password",
                },
                "spec": "ocis/docker/%s" % spec,
                "auto_tag": True if ctx.build.event == "tag" else False,
                "ignore_missing": True,
            },
        },
    ]
    if len(ctx.build.ref.split("-")) == 1:
        steps.append(
            {
                "name": "execute-latest",
                "image": PLUGINS_MANIFEST,
                "settings": {
                    "username": {
                        "from_secret": "docker_username",
                    },
                    "password": {
                        "from_secret": "docker_password",
                    },
                    "spec": "ocis/docker/%s" % spec_latest,
                    "auto_tag": True,
                    "ignore_missing": True,
                },
                "when": [
                    {
                        "event": "tag",
                    },
                ],
            },
        )

    return {
        "name": "manifest-%s" % build_type,
        "steps": steps,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "tag",
            },
        ],
    }

def changelog():
    return [{
        "name": "changelog",
        "steps": [
            {
                "name": "generate",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make -C opencloud changelog",
                ],
            },
            {
                "name": "diff",
                "image": OC_CI_ALPINE,
                "commands": [
                    "git diff",
                ],
            },
            {
                "name": "output",
                "image": OC_CI_ALPINE,
                "commands": [
                    "cat CHANGELOG.md",
                ],
            },
            {
                "name": "publish",
                "image": PLUGINS_GIT_ACTION,
                "settings": {
                    "actions": [
                        "commit",
                        "push",
                    ],
                    "message": "Automated changelog update [skip ci]",
                    "branch": "master",
                    "author_email": "devops@opencloud.eu",
                    "author_name": "openclouders",
                    "netrc_machine": "github.com",
                    "netrc_username": {
                        "from_secret": "github_username",
                    },
                    "netrc_password": {
                        "from_secret": "github_token",
                    },
                },
                "when": [
                    {
                        "event": ["push", "manual"],
                        "branch": "main",
                    },
                ],
            },
        ],
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
            },
        ],
    }]

def releaseDockerReadme(ctx, repo, build_type):
    return {
        "name": "readme-%s" % build_type,
        "steps": [
            {
                "name": "execute",
                "image": CHKO_DOCKER_PUSHRM,
                "environment": {
                    "DOCKER_USER": {
                        "from_secret": "docker_username",
                    },
                    "DOCKER_PASS": {
                        "from_secret": "docker_password",
                    },
                    "PUSHRM_TARGET": repo,
                    "PUSHRM_SHORT": "Docker images for %s" % (ctx.repo.name),
                    "PUSHRM_FILE": "README.md",
                },
            },
        ],
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "tag",
            },
        ],
    }

def docs():
    return [
        {
            "name": "dev-docs",
            "steps": [
                {
                    "name": "devdocs",
                    "image": "codeberg.org/xfix/plugin-codeberg-pages-deploy:1",
                    "settings": {
                        "folder": "docs",
                        "branch": "docs",
                        "git_config_email": "${CI_COMMIT_AUTHOR_EMAIL}",
                        "git_config_name": "${CI_COMMIT_AUTHOR}",
                        "ssh_key": {
                            "from_secret": "ssh_key",
                        },
                    },
                },
            ],
            "when": [
                {
                    "event": "push",
                    "branch": "main",
                },
            ],
        },
    ]

def makeNodeGenerate(module):
    if module == "":
        make = "make"
    else:
        make = "make -C %s" % (module)
    return [
        {
            "name": "generate nodejs",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "environment": {
                "CHROMEDRIVER_SKIP_DOWNLOAD": True,  # install fails on arm and chromedriver is a test only dependency
            },
            "commands": [
                "pnpm config set store-dir ./.pnpm-store",
                "retry -t 3 '%s node-generate-prod'" % (make),
            ],
        },
    ]

def makeGoGenerate(module):
    if module == "":
        make = "make"
    else:
        make = "make -C %s" % (module)
    return [
        {
            "name": "generate go",
            "image": OC_CI_GOLANG,
            "commands": [
                "retry -t 3 '%s go-generate'" % (make),
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
    ]

def notify(ctx):
    status = ["failure"]
    channel = config["rocketchat"]["channel"]
    if ctx.build.event == "cron":
        status.append("success")
        channel = config["rocketchat"]["channel_cron"]

    return {
        "name": "chat-notifications",
        "skip_clone": True,
        "steps": [
            {
                "name": "notify-rocketchat",
                "image": PLUGINS_SLACK,
                "settings": {
                    "webhook": {},
                    "channel": channel,
                },
            },
        ],
        "depends_on": [],
        "when": [
            {
                "event": ["push", "manual"],
                "branch": ["main", "release-*"],
                "status": status,
            },
            {
                "event": "tag",
                "status": status,
            },
        ],
    }

def ocisServer(storage = "ocis", accounts_hash_difficulty = 4, volumes = [], depends_on = [], deploy_type = "", extra_server_environment = {}, with_wrapper = False, tika_enabled = False):
    user = "0:0"
    container_name = OC_SERVER_NAME
    environment = {
        "OCIS_URL": OCIS_URL,
        "OCIS_CONFIG_DIR": "/root/.ocis/config",  # needed for checking config later
        "STORAGE_USERS_DRIVER": "%s" % (storage),
        "PROXY_ENABLE_BASIC_AUTH": True,
        "WEB_UI_CONFIG_FILE": "%s/%s" % (dirs["base"], dirs["ocisConfig"]),
        "OCIS_LOG_LEVEL": "error",
        "IDM_CREATE_DEMO_USERS": True,  # needed for litmus and cs3api-validator tests
        "IDM_ADMIN_PASSWORD": "admin",  # override the random admin password from `ocis init`
        "FRONTEND_SEARCH_MIN_LENGTH": "2",
        "OCIS_ASYNC_UPLOADS": True,
        "OCIS_EVENTS_ENABLE_TLS": False,
        "NATS_NATS_HOST": "0.0.0.0",
        "NATS_NATS_PORT": 9233,
        "OCIS_JWT_SECRET": "some-ocis-jwt-secret",
        "EVENTHISTORY_STORE": "memory",
        "OCIS_TRANSLATION_PATH": "%s/tests/config/translations" % dirs["base"],
        # debug addresses required for running services health tests
        "ACTIVITYLOG_DEBUG_ADDR": "0.0.0.0:9197",
        "APP_PROVIDER_DEBUG_ADDR": "0.0.0.0:9165",
        "APP_REGISTRY_DEBUG_ADDR": "0.0.0.0:9243",
        "AUTH_BASIC_DEBUG_ADDR": "0.0.0.0:9147",
        "AUTH_MACHINE_DEBUG_ADDR": "0.0.0.0:9167",
        "AUTH_SERVICE_DEBUG_ADDR": "0.0.0.0:9198",
        "CLIENTLOG_DEBUG_ADDR": "0.0.0.0:9260",
        "EVENTHISTORY_DEBUG_ADDR": "0.0.0.0:9270",
        "FRONTEND_DEBUG_ADDR": "0.0.0.0:9141",
        "GATEWAY_DEBUG_ADDR": "0.0.0.0:9143",
        "GRAPH_DEBUG_ADDR": "0.0.0.0:9124",
        "GROUPS_DEBUG_ADDR": "0.0.0.0:9161",
        "IDM_DEBUG_ADDR": "0.0.0.0:9239",
        "IDP_DEBUG_ADDR": "0.0.0.0:9134",
        "INVITATIONS_DEBUG_ADDR": "0.0.0.0:9269",
        "NATS_DEBUG_ADDR": "0.0.0.0:9234",
        "OCDAV_DEBUG_ADDR": "0.0.0.0:9163",
        "OCM_DEBUG_ADDR": "0.0.0.0:9281",
        "OCS_DEBUG_ADDR": "0.0.0.0:9114",
        "POSTPROCESSING_DEBUG_ADDR": "0.0.0.0:9255",
        "PROXY_DEBUG_ADDR": "0.0.0.0:9205",
        "SEARCH_DEBUG_ADDR": "0.0.0.0:9224",
        "SETTINGS_DEBUG_ADDR": "0.0.0.0:9194",
        "SHARING_DEBUG_ADDR": "0.0.0.0:9151",
        "SSE_DEBUG_ADDR": "0.0.0.0:9139",
        "STORAGE_PUBLICLINK_DEBUG_ADDR": "0.0.0.0:9179",
        "STORAGE_SHARES_DEBUG_ADDR": "0.0.0.0:9156",
        "STORAGE_SYSTEM_DEBUG_ADDR": "0.0.0.0:9217",
        "STORAGE_USERS_DEBUG_ADDR": "0.0.0.0:9159",
        "THUMBNAILS_DEBUG_ADDR": "0.0.0.0:9189",
        "USERLOG_DEBUG_ADDR": "0.0.0.0:9214",
        "USERS_DEBUG_ADDR": "0.0.0.0:9145",
        "WEB_DEBUG_ADDR": "0.0.0.0:9104",
        "WEBDAV_DEBUG_ADDR": "0.0.0.0:9119",
        "WEBFINGER_DEBUG_ADDR": "0.0.0.0:9279",
    }

    if deploy_type == "":
        environment["FRONTEND_OCS_ENABLE_DENIALS"] = True

        # fonts map for txt thumbnails (including unicode support)
        environment["THUMBNAILS_TXT_FONTMAP_FILE"] = "%s/tests/config/drone/fontsMap.json" % (dirs["base"])

    if deploy_type == "cs3api_validator":
        environment["GATEWAY_GRPC_ADDR"] = "0.0.0.0:9142"  #  make gateway available to cs3api-validator
        environment["OCIS_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD"] = False

    if deploy_type == "wopi_validator":
        environment["GATEWAY_GRPC_ADDR"] = "0.0.0.0:9142"  # make gateway available to wopi server
        environment["APP_PROVIDER_EXTERNAL_ADDR"] = "com.owncloud.api.app-provider"
        environment["APP_PROVIDER_DRIVER"] = "wopi"
        environment["APP_PROVIDER_WOPI_APP_NAME"] = "FakeOffice"
        environment["APP_PROVIDER_WOPI_APP_URL"] = "http://fakeoffice:8080"
        environment["APP_PROVIDER_WOPI_INSECURE"] = True
        environment["APP_PROVIDER_WOPI_WOPI_SERVER_EXTERNAL_URL"] = "http://wopi-fakeoffice:9300"
        environment["APP_PROVIDER_WOPI_FOLDER_URL_BASE_URL"] = OCIS_URL

    if deploy_type == "federation":
        environment["OCIS_URL"] = OC_FED_URL
        environment["PROXY_HTTP_ADDR"] = OC_FED_DOMAIN
        container_name = FED_OC_SERVER_NAME

    if tika_enabled:
        environment["FRONTEND_FULL_TEXT_SEARCH_ENABLED"] = True
        environment["SEARCH_EXTRACTOR_TYPE"] = "tika"
        environment["SEARCH_EXTRACTOR_TIKA_TIKA_URL"] = "http://tika:9998"
        environment["SEARCH_EXTRACTOR_CS3SOURCE_INSECURE"] = True

    # Pass in "default" accounts_hash_difficulty to not set this environment variable.
    # That will allow OCIS to use whatever its built-in default is.
    # Otherwise pass in a value from 4 to about 11 or 12 (default 4, for making regular tests fast)
    # The high values cause lots of CPU to be used when hashing passwords, and really slow down the tests.
    if (accounts_hash_difficulty != "default"):
        environment["ACCOUNTS_HASH_DIFFICULTY"] = accounts_hash_difficulty

    for item in extra_server_environment:
        environment[item] = extra_server_environment[item]

    wrapper_commands = [
        "make -C %s build" % dirs["ocisWrapper"],
        "%s/bin/ociswrapper serve --bin %s --url %s --admin-username admin --admin-password admin" % (dirs["ocisWrapper"], dirs["opencloudBin"], environment["OCIS_URL"]),
    ]

    wait_for_ocis = {
        "name": "wait-for-%s" % (container_name),
        "image": OC_CI_ALPINE,
        "commands": [
            # wait for ocis-server to be ready (5 minutes)
            "timeout 300 bash -c 'while [ $(curl -sk -uadmin:admin " +
            "%s/graph/v1.0/users/admin " % environment["OCIS_URL"] +
            "-w %{http_code} -o /dev/null) != 200 ]; do sleep 1; done'",
        ],
        "depends_on": depends_on,
    }

    return [
        {
            "name": container_name,
            "image": OC_CI_GOLANG,
            "detach": True,
            "environment": environment,
            "backend_options": {
                "docker": {
                    "user": user,
                },
            },
            "commands": [
                "%s init --insecure true" % dirs["opencloudBin"],
                "cat $OCIS_CONFIG_DIR/ocis.yaml",
                "cp tests/config/drone/app-registry.yaml /root/.ocis/config/app-registry.yaml",
            ] + (wrapper_commands),
            "depends_on": depends_on,
        },
        wait_for_ocis,
    ]

def startOcisService(service = None, name = None, environment = {}, volumes = []):
    """
    Starts an OCIS service in a detached container.

    Args:
        service (str): The name of the service to start.
        name (str): The name of the container.
        environment (dict): The environment variables to set in the container.
        volumes (list): The volumes to mount in the container.

    Returns:
        list: A list of pipeline steps to start the service.
    """

    if not service:
        return []
    if not name:
        name = service

    return [
        {
            "name": name,
            "image": OC_CI_GOLANG,
            "detach": True,
            "environment": environment,
            "commands": [
                "%s %s server" % (dirs["opencloudBin"], service),
            ],
        },
    ]

def redis():
    return [
        {
            "name": "redis",
            "image": REDIS,
        },
    ]

def redisForOCStorage(storage = "ocis"):
    if storage == "owncloud":
        return redis()
    else:
        return

def build():
    return [
        {
            "name": "build",
            "image": OC_CI_GOLANG,
            "commands": [
                "retry -t 3 'make -C opencloud build'",
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
    ]

def skipIfUnchanged(ctx, type):
    if "full-ci" in ctx.build.title.lower() or ctx.build.event == "tag" or ctx.build.event == "cron":
        return []

    base = [
        "^.github/.*",
        "^.vscode/.*",
        "^changelog/.*",
        "^docs/.*",
        "^deployments/.*",
        "CHANGELOG.md",
        "CONTRIBUTING.md",
        "LICENSE",
        "README.md",
    ]
    unit = [
        ".*_test.go$",
    ]
    acceptance = [
        "^tests/acceptance/.*",
    ]

    skip = []
    if type == "acceptance-tests" or type == "e2e-tests" or type == "lint":
        skip = base + unit
    elif type == "unit-tests":
        skip = base + acceptance
    elif type == "build-binary" or type == "build-docker" or type == "litmus":
        skip = base + unit + acceptance
    elif type == "cache":
        skip = base
    else:
        return []

    return skip

def example_deploys(ctx):
    on_merge_deploy = [
        "ocis_full/master.yml",
        "ocis_full/onlyoffice-master.yml",
    ]
    nightly_deploy = [
        "ocis_ldap/rolling.yml",
        "ocis_keycloak/rolling.yml",
        "ocis_full/production.yml",
        "ocis_full/rolling.yml",
        "ocis_full/onlyoffice-rolling.yml",
        "ocis_full/s3-rolling.yml",
    ]

    # if on master branch:
    configs = on_merge_deploy
    rebuild = False

    if ctx.build.event == "tag":
        configs = nightly_deploy
        rebuild = False

    if ctx.build.event == "cron":
        configs = on_merge_deploy + nightly_deploy
        rebuild = True

    deploys = []
    for config in configs:
        deploys.append(deploy(ctx, config, rebuild))

    return deploys

def deploy(ctx, config, rebuild):
    return {
        "name": "deploy_%s" % (config),
        "steps": [
            {
                "name": "clone continuous deployment playbook",
                "image": ALPINE_GIT,
                "commands": [
                    "cd deployments/continuous-deployment-config",
                    "git clone https://github.com/owncloud-devops/continuous-deployment.git",
                ],
            },
            {
                "name": "deploy",
                "image": OC_CI_DRONE_ANSIBLE,
                "failure": "ignore",
                "environment": {
                    "CONTINUOUS_DEPLOY_SERVERS_CONFIG": "../%s" % (config),
                    "REBUILD": rebuild,
                    "HCLOUD_API_TOKEN": {
                        "from_secret": "hcloud_api_token",
                    },
                    "CLOUDFLARE_API_TOKEN": {
                        "from_secret": "cloudflare_api_token",
                    },
                },
                "settings": {
                    "playbook": "deployments/continuous-deployment-config/continuous-deployment/playbook-all.yml",
                    "galaxy": "deployments/continuous-deployment-config/continuous-deployment/requirements.yml",
                    "requirements": "deployments/continuous-deployment-config/continuous-deployment/py-requirements.txt",
                    "inventory": "localhost",
                    "private_key": {
                        "from_secret": "ssh_private_key",
                    },
                },
            },
        ],
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "tag",
            },
        ],
    }

def checkStarlark():
    return [{
        "name": "check-starlark",
        "steps": [
            {
                "name": "format-check-starlark",
                "image": OC_CI_BAZEL_BUILDIFIER,
                "commands": [
                    "buildifier --mode=check .woodpecker.star",
                ],
            },
            {
                "name": "show-diff",
                "image": OC_CI_BAZEL_BUILDIFIER,
                "commands": [
                    "buildifier --mode=fix .woodpecker.star",
                    "git diff",
                ],
                "when": [
                    {
                        "status": "failure",
                    },
                ],
            },
        ],
        "depends_on": [],
        "when": [
            {
                "event": "pull_request",
            },
        ],
    }]

def genericCache(name, action, mounts, cache_path):
    rebuild = False
    restore = False
    if action == "rebuild":
        rebuild = True
        action = "rebuild"
    else:
        restore = True
        action = "restore"

    step = {
        "name": "%s_%s" % (action, name),
        "image": PLUGINS_S3_CACHE,
        "settings": {
            "endpoint": {
                "from_secret": "cache_s3_server",
            },
            "rebuild": rebuild,
            "restore": restore,
            "mount": mounts,
            "access_key": {
                "from_secret": "cache_s3_access_key",
            },
            "secret_key": {
                "from_secret": "cache_s3_secret_key",
            },
            "filename": "%s.tar" % (name),
            "path": cache_path,
            "fallback_path": cache_path,
        },
    }
    return step

def genericCachePurge(flush_path):
    return {
        "name": "purge_build_artifact_cache",
        "steps": [
            {
                "name": "purge-cache",
                "image": PLUGINS_S3_CACHE,
                "settings": {
                    "access_key": {
                        "from_secret": "cache_s3_access_key",
                    },
                    "secret_key": {
                        "from_secret": "cache_s3_secret_key",
                    },
                    "endpoint": {
                        "from_secret": "cache_s3_server",
                    },
                    "flush": True,
                    "flush_age": 1,
                    "flush_path": flush_path,
                },
            },
        ],
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
            },
            # {
            #     "status": ["success", "failure"]
            # },
        ],
    }

def genericBuildArtifactCache(ctx, name, action, path):
    if action == "rebuild" or action == "restore":
        cache_path = "%s/%s/%s" % ("cache", repo_slug, ctx.build.commit + "-${DRONE_BUILD_NUMBER}")
        name = "%s_build_artifact_cache" % (name)
        return genericCache(name, action, [path], cache_path)

    if action == "purge":
        flush_path = "%s/%s" % ("cache", repo_slug)
        return genericCachePurge(flush_path)
    return []

def restoreBuildArtifactCache(ctx, name, path):
    return [genericBuildArtifactCache(ctx, name, "restore", path)]

def rebuildBuildArtifactCache(ctx, name, path):
    return [genericBuildArtifactCache(ctx, name, "rebuild", path)]

def purgeBuildArtifactCache(ctx):
    return genericBuildArtifactCache(ctx, "", "purge", [])

def pipelineSanityChecks(ctx, pipelines):
    """pipelineSanityChecks helps the CI developers to find errors before running it

    These sanity checks are only executed on when converting starlark to yaml.
    Error outputs are only visible when the conversion is done with the drone cli.

    Args:
      ctx: drone passes a context with information which the pipeline can be adapted to
      pipelines: pipelines to be checked, normally you should run this on the return value of main()

    Returns:
      none
    """

    # check if name length of pipeline and steps are exceeded.
    max_name_length = 50
    for pipeline in pipelines:
        pipeline_name = pipeline["name"]
        if len(pipeline_name) > max_name_length:
            print("Error: pipeline name %s is longer than 50 characters" % (pipeline_name))

        for step in pipeline["steps"]:
            step_name = step["name"]
            if len(step_name) > max_name_length:
                print("Error: step name %s in pipeline %s is longer than 50 characters" % (step_name, pipeline_name))

    # check for non existing depends_on
    possible_depends = []
    for pipeline in pipelines:
        possible_depends.append(pipeline["name"])

    for pipeline in pipelines:
        if "depends_on" in pipeline.keys():
            for depends in pipeline["depends_on"]:
                if not depends in possible_depends:
                    print("Error: depends_on %s for pipeline %s is not defined" % (depends, pipeline["name"]))

    # check for non declared volumes
    # for pipeline in pipelines:
    #   pipeline_volumes = []
    #   if "workspace" in pipeline.keys():
    #     for volume in pipeline["workspace"]:
    #       pipeline_volumes.append(volume["base"])
    #
    #   for step in pipeline["steps"]:
    #     if "workspace" in step.keys():
    #       for volume in step["workspace"]:
    #         if not volume["base"] in pipeline_volumes:
    #           print("Warning: volume %s for step %s is not defined in pipeline %s" % (volume["base"], step["name"], pipeline["name"]))

    # list used docker images
    print("")
    print("List of used docker images:")

    images = {}

    for pipeline in pipelines:
        for step in pipeline["steps"]:
            image = step["image"]
            if image in images.keys():
                images[image] = images[image] + 1
            else:
                images[image] = 1

    for image in images.keys():
        print(" %sx\t%s" % (images[image], image))

def litmus(ctx, storage):
    pipelines = []

    if (config["litmus"] == False):
        return pipelines

    environment = {
        "LITMUS_PASSWORD": "admin",
        "LITMUS_USERNAME": "admin",
        "TESTS": "basic copymove props http",
    }

    litmusCommand = "/usr/local/bin/litmus-wrapper"

    result = {
        "name": "litmus",
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 ocisServer(storage) +
                 setupForLitmus() +
                 [
                     {
                         "name": "old-endpoint",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/webdav"' % OCIS_URL,
                             litmusCommand,
                         ],
                     },
                     {
                         "name": "new-endpoint",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/dav/files/admin"' % OCIS_URL,
                             litmusCommand,
                         ],
                     },
                     {
                         "name": "new-shared",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/dav/files/admin/Shares/new_folder/"' % OCIS_URL,
                             litmusCommand,
                         ],
                     },
                     {
                         "name": "old-shared",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/webdav/Shares/new_folder/"' % OCIS_URL,
                             litmusCommand,
                         ],
                     },
                     #  {
                     #      "name": "public-share",
                     #      "image": OC_LITMUS,
                     #      "environment": {
                     #          "LITMUS_PASSWORD": "admin",
                     #          "LITMUS_USERNAME": "admin",
                     #          "TESTS": "basic copymove http",
                     #      },
                     #      "commands": [
                     #          "source .env",
                     #          "export LITMUS_URL='%s/remote.php/dav/public-files/'$PUBLIC_TOKEN" % OCIS_URL,
                     #          litmusCommand,
                     #      ],
                     #  },
                     {
                         "name": "spaces-endpoint",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             "export LITMUS_URL='%s/remote.php/dav/spaces/'$SPACE_ID" % OCIS_URL,
                             litmusCommand,
                         ],
                     },
                 ],
        "services": redisForOCStorage(storage),
        "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
        "when": [
            {
                "event": ["push", "manual"],
                "branch": "main",
            },
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "litmus"),
                },
            },
        ],
    }
    pipelines.append(result)

    return pipelines

def setupForLitmus():
    return [{
        "name": "setup-for-litmus",
        "image": OC_UBUNTU,
        "environment": {
            "TEST_SERVER_URL": OCIS_URL,
        },
        "commands": [
            "bash ./tests/config/drone/setup-for-litmus.sh",
            "cat .env",
        ],
    }]

def getDroneEnvAndCheckScript(ctx):
    ocis_git_base_url = "https://raw.githubusercontent.com/opencloud-eu/opencloud"
    path_to_drone_env = "%s/%s/.drone.env" % (ocis_git_base_url, ctx.build.commit)
    path_to_check_script = "%s/%s/tests/config/drone/check_web_cache.sh" % (ocis_git_base_url, ctx.build.commit)
    return {
        "name": "get-drone-env-and-check-script",
        "image": OC_UBUNTU,
        "commands": [
            "curl -s -o .drone.env %s" % path_to_drone_env,
            "curl -s -o check_web_cache.sh %s" % path_to_check_script,
        ],
    }

def checkForWebCache(name):
    return {
        "name": "check-for-%s-cache" % name,
        "image": OC_UBUNTU,
        "environment": {
            "CACHE_ENDPOINT": {
                "from_secret": "cache_s3_server",
            },
            "CACHE_BUCKET": {
                "from_secret": "cache_s3_bucket",
            },
        },
        "commands": [
            "bash -x check_web_cache.sh %s" % name,
        ],
    }

def cloneWeb():
    return {
        "name": "clone-web",
        "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
        "commands": [
            ". ./.drone.env",
            "rm -rf %s" % dirs["web"],
            "git clone -b $WEB_BRANCH --single-branch --no-tags https://github.com/owncloud/web.git %s" % dirs["web"],
            "cd %s && git checkout $WEB_COMMITID" % dirs["web"],
        ],
    }

def generateWebPnpmCache(ctx):
    return [
        getDroneEnvAndCheckScript(ctx),
        checkForWebCache("web-pnpm"),
        cloneWeb(),
        {
            "name": "install-pnpm",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "commands": [
                "cd %s" % dirs["web"],
                'npm install --silent --global --force "$(jq -r ".packageManager" < package.json)"',
                "pnpm config set store-dir ./.pnpm-store",
                "retry -t 3 'pnpm install'",
            ],
        },
        {
            "name": "zip-pnpm",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "commands": [
                # zip the pnpm deps before caching
                "if [ ! -d '%s' ]; then mkdir -p %s; fi" % (dirs["zip"], dirs["zip"]),
                "cd %s" % dirs["web"],
                "tar -czf %s .pnpm-store" % dirs["webPnpmZip"],
            ],
        },
        {
            "name": "cache-pnpm",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                "source ./.drone.env",
                # cache using the minio/mc client to the public bucket (long term bucket)
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r -a %s s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID" % dirs["webPnpmZip"],
            ],
        },
    ]

def generateWebCache(ctx):
    return [
        getDroneEnvAndCheckScript(ctx),
        checkForWebCache("web"),
        cloneWeb(),
        {
            "name": "zip-web",
            "image": OC_UBUNTU,
            "commands": [
                "if [ ! -d '%s' ]; then mkdir -p %s; fi" % (dirs["zip"], dirs["zip"]),
                "tar -czvf %s webTestRunner" % dirs["webZip"],
            ],
        },
        {
            "name": "cache-web",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                "source ./.drone.env",
                # cache using the minio/mc client to the 'owncloud' bucket (long term bucket)
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r -a %s s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID" % dirs["webZip"],
            ],
        },
    ]

def restoreWebCache():
    return [{
        "name": "restore-web-cache",
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": [
            "source ./.drone.env",
            "rm -rf %s" % dirs["web"],
            "mkdir -p %s" % dirs["web"],
            "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
            "mc cp -r -a s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID/web.tar.gz %s" % dirs["zip"],
        ],
    }, {
        "name": "unzip-web-cache",
        "image": OC_UBUNTU,
        "commands": [
            "tar -xvf %s -C ." % dirs["webZip"],
        ],
    }]

def restoreWebPnpmCache():
    return [{
        "name": "restore-web-pnpm-cache",
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": [
            "source ./.drone.env",
            "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
            "mc cp -r -a s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID/pnpm-store.tar.gz %s" % dirs["zip"],
        ],
    }, {
        # we need to install again because the node_modules are not cached
        "name": "unzip-and-install-pnpm",
        "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
        "commands": [
            "cd %s" % dirs["web"],
            "rm -rf .pnpm-store",
            "tar -xvf %s" % dirs["webPnpmZip"],
            'npm install --silent --global --force "$(jq -r ".packageManager" < package.json)"',
            "pnpm config set store-dir ./.pnpm-store",
            "retry -t 3 'pnpm install'",
        ],
    }]

def emailService():
    return [{
        "name": "email",
        "image": INBUCKET_INBUCKET,
    }]

def waitForEmailService():
    return [{
        "name": "wait-for-email",
        "image": OC_CI_WAIT_FOR,
        "commands": [
            "wait-for -it email:9000 -t 600",
        ],
    }]

def clamavService():
    return [{
        "name": "clamav",
        "image": OC_CI_CLAMAVD,
    }]

def waitForClamavService():
    return [{
        "name": "wait-for-clamav",
        "image": OC_CI_WAIT_FOR,
        "commands": [
            "wait-for -it clamav:3310 -t 600",
        ],
    }]

def fakeOffice():
    return [
        {
            "name": "fakeoffice",
            "image": OC_CI_ALPINE,
            "detach": True,
            "environment": {},
            "commands": [
                "sh %s/tests/config/drone/serve-hosting-discovery.sh" % (dirs["base"]),
            ],
        },
    ]

def wopiCollaborationService(name):
    service_name = "wopi-%s" % name

    environment = {
        "MICRO_REGISTRY": "nats-js-kv",
        "MICRO_REGISTRY_ADDRESS": "%s:9233" % OC_SERVER_NAME,
        "COLLABORATION_LOG_LEVEL": "debug",
        "COLLABORATION_GRPC_ADDR": "0.0.0.0:9301",
        "COLLABORATION_HTTP_ADDR": "0.0.0.0:9300",
        "COLLABORATION_DEBUG_ADDR": "0.0.0.0:9304",
        "COLLABORATION_APP_PROOF_DISABLE": True,
        "COLLABORATION_APP_INSECURE": True,
        "COLLABORATION_CS3API_DATAGATEWAY_INSECURE": True,
        "OCIS_JWT_SECRET": "some-ocis-jwt-secret",
        "COLLABORATION_WOPI_SECRET": "some-wopi-secret",
    }

    if name == "collabora":
        environment["COLLABORATION_APP_NAME"] = "Collabora"
        environment["COLLABORATION_APP_PRODUCT"] = "Collabora"
        environment["COLLABORATION_APP_ADDR"] = "https://collabora:9980"
        environment["COLLABORATION_APP_ICON"] = "https://collabora:9980/favicon.ico"
    elif name == "onlyoffice":
        environment["COLLABORATION_APP_NAME"] = "OnlyOffice"
        environment["COLLABORATION_APP_PRODUCT"] = "OnlyOffice"
        environment["COLLABORATION_APP_ADDR"] = "https://onlyoffice"
        environment["COLLABORATION_APP_ICON"] = "https://onlyoffice/web-apps/apps/documenteditor/main/resources/img/favicon.ico"
    elif name == "fakeoffice":
        environment["COLLABORATION_APP_NAME"] = "FakeOffice"
        environment["COLLABORATION_APP_PRODUCT"] = "Microsoft"
        environment["COLLABORATION_APP_ADDR"] = "http://fakeoffice:8080"

    environment["COLLABORATION_WOPI_SRC"] = "http://%s:9300" % service_name

    return startOcisService("collaboration", service_name, environment)

def tikaService():
    return [{
        "name": "tika",
        "type": "docker",
        "image": APACHE_TIKA,
        "detach": True,
    }, {
        "name": "wait-for-tika-service",
        "image": OC_CI_WAIT_FOR,
        "commands": [
            "wait-for -it tika:9998 -t 300",
        ],
    }]

def logRequests():
    return [{
        "name": "api-test-failure-logs",
        "image": OC_CI_PHP % DEFAULT_PHP_VERSION,
        "commands": [
            "cat %s/tests/acceptance/logs/failed.log" % dirs["base"],
        ],
        "when": {
            "status": [
                "failure",
            ],
        },
    }]

def k6LoadTests(ctx):
    ocis_remote_environment = {
        "SSH_OCIS_REMOTE": {
            "from_secret": "k6_ssh_ocis_remote",
        },
        "SSH_OCIS_USERNAME": {
            "from_secret": "k6_ssh_ocis_user",
        },
        "SSH_OCIS_PASSWORD": {
            "from_secret": "k6_ssh_ocis_pass",
        },
        "TEST_SERVER_URL": {
            "from_secret": "k6_ssh_ocis_server_url",
        },
    }
    k6_remote_environment = {
        "SSH_K6_REMOTE": {
            "from_secret": "k6_ssh_k6_remote",
        },
        "SSH_K6_USERNAME": {
            "from_secret": "k6_ssh_k6_user",
        },
        "SSH_K6_PASSWORD": {
            "from_secret": "k6_ssh_k6_pass",
        },
    }
    environment = {}
    environment.update(ocis_remote_environment)
    environment.update(k6_remote_environment)

    if "skip" in config["k6LoadTests"] and config["k6LoadTests"]["skip"]:
        return []

    ocis_git_base_url = "https://raw.githubusercontent.com/opencloud-eu/opencloud"
    script_link = "%s/%s/tests/config/drone/run_k6_tests.sh" % (ocis_git_base_url, ctx.build.commit)

    event_array = ["cron"]

    if "k6-test" in ctx.build.title.lower():
        event_array.append("pull_request")

    return [{
        "name": "k6-load-test",
        "skip_clone": True,
        "steps": [
            {
                "name": "k6-load-test",
                "image": OC_CI_ALPINE,
                "environment": environment,
                "commands": [
                    "curl -s -o run_k6_tests.sh %s" % script_link,
                    "apk add --no-cache openssh-client sshpass",
                    "sh %s/run_k6_tests.sh" % (dirs["base"]),
                ],
            },
            {
                "name": "ocis-log",
                "image": OC_CI_ALPINE,
                "environment": ocis_remote_environment,
                "commands": [
                    "curl -s -o run_k6_tests.sh %s" % script_link,
                    "apk add --no-cache openssh-client sshpass",
                    "sh %s/run_k6_tests.sh --ocis-log" % (dirs["base"]),
                ],
                "when": [
                    {
                        "status": ["success", "failure"],
                    },
                ],
            },
            {
                "name": "open-grafana-dashboard",
                "image": OC_CI_ALPINE,
                "commands": [
                    "echo 'Grafana Dashboard: https://grafana.k6.infra.owncloud.works'",
                ],
                "when": [
                    {
                        "status": ["success", "failure"],
                    },
                ],
            },
        ],
        "depends_on": [],
        "when": [
            {
                "event": event_array,
            },
        ],
    }]

def waitForServices(name, services = []):
    services = ",".join(services)
    return [{
        "name": "wait-for-%s" % name,
        "image": OC_CI_WAIT_FOR,
        "commands": [
            "wait-for -it %s -t 300" % services,
        ],
    }]

def ocisHealthCheck(name, services = []):
    commands = []
    timeout = 300
    curl_command = ["timeout %s bash -c 'while [ $(curl -s %s/%s ", "-w %{http_code} -o /dev/null) != 200 ]; do sleep 1; done'"]
    for service in services:
        commands.append(curl_command[0] % (timeout, service, "healthz") + curl_command[1])
        commands.append(curl_command[0] % (timeout, service, "readyz") + curl_command[1])

    return [{
        "name": "health-check-%s" % name,
        "image": OC_CI_ALPINE,
        "commands": commands,
    }]

def collaboraService():
    return [
        {
            "name": "collabora",
            "type": "docker",
            "image": COLLABORA_CODE,
            "detach": True,
            "environment": {
                "DONT_GEN_SSL_CERT": "set",
                "extra_params": "--o:ssl.enable=true --o:ssl.termination=true --o:welcome.enable=false --o:net.frame_ancestors=%s" % OCIS_URL,
            },
            "commands": [
                "coolconfig generate-proof-key",
                "bash /start-collabora-online.sh",
            ],
        },
    ]

def onlyofficeService():
    return [
        {
            "name": "onlyoffice",
            "type": "docker",
            "image": ONLYOFFICE_DOCUMENT_SERVER,
            "detach": True,
            "environment": {
                "WOPI_ENABLED": True,
                "USE_UNAUTHORIZED_STORAGE": True,  # self signed certificates
            },
            "commands": [
                "cp %s/tests/config/drone/only-office.json /etc/onlyoffice/documentserver/local.json" % dirs["base"],
                "openssl req -x509 -newkey rsa:4096 -keyout onlyoffice.key -out onlyoffice.crt -sha256 -days 365 -batch -nodes",
                "mkdir -p /var/www/onlyoffice/Data/certs",
                "cp onlyoffice.key /var/www/onlyoffice/Data/certs/",
                "cp onlyoffice.crt /var/www/onlyoffice/Data/certs/",
                "chmod 400 /var/www/onlyoffice/Data/certs/onlyoffice.key",
                "/app/ds/run-document-server.sh",
            ],
        },
    ]
