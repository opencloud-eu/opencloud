"""OpenCloud CI definition
"""

# Repository

repo_slug = "opencloud-eu/opencloud"
docker_repo_slug = "opencloudeu/opencloud"

# images
ALPINE_GIT = "alpine/git:latest"
APACHE_TIKA = "apache/tika:2.8.0.0"
CHKO_DOCKER_PUSHRM = "chko/docker-pushrm:1"
COLLABORA_CODE = "collabora/code:24.04.5.1.1"
OPEN_SEARCH = "opensearchproject/opensearch:2"
INBUCKET_INBUCKET = "inbucket/inbucket"
MINIO_MC = "minio/mc:RELEASE.2021-10-07T04-19-58Z"
OC_CI_ALPINE = "owncloudci/alpine:latest"
OC_CI_BAZEL_BUILDIFIER = "owncloudci/bazel-buildifier:latest"
OC_CI_CLAMAVD = "owncloudci/clamavd"
OC_CI_DRONE_ANSIBLE = "owncloudci/drone-ansible:latest"
OC_CI_GOLANG = "docker.io/golang:1.24"
OC_CI_NODEJS = "owncloudci/nodejs:%s"
OC_CI_PHP = "owncloudci/php:%s"
OC_CI_WAIT_FOR = "owncloudci/wait-for:latest"
OC_CS3_API_VALIDATOR = "opencloudeu/cs3api-validator:latest"
OC_LITMUS = "owncloudci/litmus:latest"
OC_UBUNTU = "owncloud/ubuntu:20.04"
ONLYOFFICE_DOCUMENT_SERVER = "onlyoffice/documentserver:7.5.1"
PLUGINS_DOCKER_BUILDX = "woodpeckerci/plugin-docker-buildx:latest"
PLUGINS_GITHUB_RELEASE = "woodpeckerci/plugin-release"
PLUGINS_GIT_ACTION = "quay.io/thegeeklab/wp-git-action"
PLUGINS_S3 = "plugins/s3:1"
PLUGINS_S3_CACHE = "plugins/s3-cache:1"
PLUGINS_SLACK = "plugins/slack:1"
REDIS = "redis:6-alpine"
READY_RELEASE_GO = "woodpeckerci/plugin-ready-release-go:latest"
OPENLDAP = "bitnamilegacy/openldap:2.6"

DEFAULT_PHP_VERSION = "8.2"
DEFAULT_NODEJS_VERSION = "20"

CACHE_S3_SERVER = "https://s3.ci.opencloud.eu"
INSTALL_LIBVIPS_COMMAND = "apt-get update; apt-get install libvips42 -y"

dirs = {
    "base": "/woodpecker/src/github.com/opencloud-eu/opencloud",
    "web": "/woodpecker/src/github.com/opencloud-eu/opencloud/webTestRunner",
    "zip": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip",
    "webZip": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip/web.tar.gz",
    "webPnpmZip": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip/web-pnpm.tar.gz",
    "playwrightBrowsersArchive": "/woodpecker/src/github.com/opencloud-eu/opencloud/zip/playwright-browsers.tar.gz",
    "baseGo": "/go/src/github.com/opencloud-eu/opencloud",
    "gobinTar": "go-bin.tar.gz",
    "gobinTarPath": "/go/src/github.com/opencloud-eu/opencloud/go-bin.tar.gz",
    "opencloudConfig": "tests/config/woodpecker/opencloud-config.json",
    "opencloudRevaDataRoot": "/woodpecker/src/github.com/opencloud-eu/opencloud/srv/app/tmp/ocis/owncloud/data",
    "multiServiceOcBaseDataPath": "/woodpecker/src/github.com/opencloud-eu/opencloud/multiServiceData",
    "ocWrapper": "/woodpecker/src/github.com/opencloud-eu/opencloud/tests/ocwrapper",
    "bannedPasswordList": "tests/config/woodpecker/banned-password-list.txt",
    "ocmProviders": "tests/config/woodpecker/providers.json",
    "opencloudBinPath": "opencloud/bin",
    "opencloudBin": "opencloud/bin/opencloud",
    "opencloudBinArtifact": "opencloud-binary-amd64",
}

# OpenCloud URLs
OC_SERVER_NAME = "opencloud-server"
OC_URL = "https://%s:9200" % OC_SERVER_NAME
OC_DOMAIN = "%s:9200" % OC_SERVER_NAME
FED_OC_SERVER_NAME = "federation-opencloud-server"
OC_FED_URL = "https://%s:10200" % FED_OC_SERVER_NAME
OC_FED_DOMAIN = "%s:10200" % FED_OC_SERVER_NAME

event = {
    "base": {
        "event": ["push", "manual"],
        "branch": "main",
    },
    "cron": {
        "event": "cron",
        "branch": "main",
    },
    "pull_request": {
        "event": "pull_request",
    },
    "tag": {
        "event": "tag",
    },
}

# configuration
config = {
    "cs3ApiTests": {
        "skip": False,
    },
    "wopiValidatorTests": {
        "skip": False,
    },
    "k6LoadTests": {
        "skip": True,
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
            "extraTestEnvironment": {
                "EMAIL_HOST": "email",
                "EMAIL_PORT": "9000",
            },
            "extraServerEnvironment": {
                "OC_ADD_RUN_SERVICES": "notifications",
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
                "collaborativePosix",
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
            "extraTestEnvironment": {
                "EMAIL_HOST": "email",
                "EMAIL_PORT": "9000",
            },
            "extraServerEnvironment": {
                "OC_ADD_RUN_SERVICES": "notifications",
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
                "OC_ASYNC_UPLOADS": True,
                "OC_ADD_RUN_SERVICES": "antivirus",
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
            "extraTestEnvironment": {
                "EMAIL_HOST": "email",
                "EMAIL_PORT": "9000",
            },
            "extraServerEnvironment": {
                "OC_ADD_RUN_SERVICES": "ocm,notifications",
                "OC_ENABLE_OCM": True,
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
                "OC_ASYNC_UPLOADS": True,
                "OC_ADD_RUN_SERVICES": "antivirus",
                "STORAGE_USERS_DRIVER": "decomposed",
            },
        },
        "multiTenancy": {
            "suites": [
                "apiTenancy",
            ],
            "skip": False,
            "withRemotePhp": [True],
            "ldapNeeded": True,
            "extraTestEnvironment": {
                "USE_PREPARED_LDAP_USERS": True,
            },
            "extraServerEnvironment": {
                "OC_MULTI_TENANT_ENABLED": True,
                "OC_LDAP_USER_SCHEMA_TENANT_ID": "departmentNumber",
                "OC_LDAP_URI": "ldaps://ldap-server:1636",
                "OC_LDAP_INSECURE": True,
                "OC_LDAP_BIND_DN": "cn=admin,dc=opencloud,dc=eu",
                "OC_LDAP_BIND_PASSWORD": "admin",
                "OC_LDAP_GROUP_BASE_DN": "ou=groups,dc=opencloud,dc=eu",
                "OC_LDAP_GROUP_SCHEMA_ID": "entryUUID",
                "OC_LDAP_USER_BASE_DN": "ou=users,dc=opencloud,dc=eu",
                "OC_LDAP_USER_FILTER": "(objectclass=inetOrgPerson)",
                "OC_LDAP_USER_SCHEMA_ID": "entryUUID",
                "OC_LDAP_DISABLE_USER_MECHANISM": "none",
                "GRAPH_IDENTITY_BACKEND": "cs3",
                "GRAPH_LDAP_SERVER_UUID": True,
                "GRAPH_LDAP_GROUP_CREATE_BASE_DN": "ou=custom,ou=groups,dc=opencloud,dc=eu",
                "GRAPH_LDAP_REFINT_ENABLED": True,
                "GROUPS_DRIVER": "null",
                "FRONTEND_READONLY_USER_ATTRIBUTES": "user.onPremisesSamAccountName,user.displayName,user.mail,user.passwordProfile,user.accountEnabled,user.appRoleAssignments",
                "OC_LDAP_SERVER_WRITE_ENABLED": False,
                "OC_EXCLUDE_RUN_SERVICES": "idm",
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
            "xsuites": ["search", "app-provider", "app-provider-onlyOffice", "app-store", "keycloak", "oidc", "ocm", "a11y", "mobile-view", "navigation"],  # suites to skip
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
        "production": {
            # NOTE: need to be updated if new production releases are determined
            "tags": ["2.0"],
            "repo": docker_repo_slug,
            "build_type": "production",
        },
        "rolling": {
            "repo": docker_repo_slug + "-rolling",
            "build_type": "rolling",
        },
        "daily": {
            "repo": docker_repo_slug + "-rolling",
            "build_type": "daily",
        },
    },
    "litmus": True,
    "codestyle": True,
}

GRAPH_AVAILABLE_ROLES = "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5,a8d5fe5e-96e3-418d-825b-534dbdf22b99,fb6c3e19-e378-47e5-b277-9732f9de6e21,58c63c02-1d89-4572-916a-870abc5a1b7d,2d00ce52-1fc2-4dbc-8b95-a73b73395f5a,1c996275-f1c9-4e71-abdf-a42f6495e960,312c0871-5ef7-4b3a-85b6-0e4074c64049,aa97fe03-7980-45ac-9e50-b325749fd7e6,63e64e19-8d43-42ec-a738-2b6af2610efa"

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
    "MC_HOST": CACHE_S3_SERVER,
    "AWS_ACCESS_KEY_ID": {
        "from_secret": "cache_s3_access_key",
    },
    "AWS_SECRET_ACCESS_KEY": {
        "from_secret": "cache_s3_secret_key",
    },
    "PUBLIC_BUCKET": "public",
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
      pipelines: array of woodpecker pipelines

    Returns:
      names of the given pipelines as string array
    """
    names = []
    for pipeline in pipelines:
        names.append(pipeline["name"])
    return names

def main(ctx):
    """main is the entrypoint for woodpecker

    Args:
      ctx: woodpecker passes a context with information which the pipeline can be adapted to

    Returns:
      none
    """

    if ctx.build.event == "cron" and ctx.build.sender == "translation-sync":
        return translation_sync(ctx)

    build_release_helpers = \
        readyReleaseGo()

    build_release_helpers.append(
        pipelineDependsOn(
            licenseCheck(ctx),
            getGoBinForTesting(ctx),
        ),
    )

    test_pipelines = \
        codestyle(ctx) + \
        checkGherkinLint(ctx) + \
        checkTestSuitesInExpectedFailures(ctx) + \
        buildWebCache(ctx) + \
        cacheBrowsers(ctx) + \
        getGoBinForTesting(ctx) + \
        buildOpencloudBinaryForTesting(ctx) + \
        checkStarlark(ctx) + \
        build_release_helpers + \
        testOpencloudAndUploadResults(ctx) + \
        testPipelines(ctx)

    build_release_pipelines = \
        dockerReleases(ctx) + \
        binaryReleases(ctx)

    test_pipelines.append(
        pipelineDependsOn(
            purgeBuildArtifactCache(ctx),
            testPipelines(ctx),
        ),
    )

    test_pipelines.append(
        pipelineDependsOn(
            purgeBrowserCache(ctx),
            testPipelines(ctx),
        ),
    )

    test_pipelines.append(
        pipelineDependsOn(
            purgeTracingCache(ctx),
            testPipelines(ctx),
        ),
    )

    test_pipelines.append(
        pipelineDependsOn(
            purgeOpencloudWebBuildCache(ctx),
            testPipelines(ctx),
        ),
    )

    test_pipelines.append(
        pipelineDependsOn(
            purgeGoBinCache(ctx),
            testPipelines(ctx),
        ),
    )

    pipelines = test_pipelines + build_release_pipelines + notifyMatrix(ctx)

    pipelineSanityChecks(pipelines)
    return pipelines

def cachePipeline(ctx, name, steps):
    return {
        "name": "build-%s-cache" % name,
        "skip_clone": True,
        "steps": steps,
        "when": [
            {
                "event": ["push", "manual"],
                "branch": ["main", "stable-*"],
            },
            event["cron"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "base"),
                },
            },
        ],
    }

def buildWebCache(ctx):
    return [
        cachePipeline(ctx, "web", generateWebCache(ctx)),
        cachePipeline(ctx, "web-pnpm", generateWebPnpmCache(ctx)),
    ]

def testOpencloudAndUploadResults(ctx):
    pipeline = testOpencloud(ctx)

    ######################################################################
    # The triggers have been disabled for now, since the govulncheck can #
    # not silence single, acceptable vulnerabilities.                    #
    # See https://github.com/owncloud/ocis/issues/9527 for more details. #
    # FIXME: RE-ENABLE THIS ASAP!!!                                      #
    ######################################################################

    #security_scan = scanOpencloud(ctx)
    #return [security_scan, pipeline, scan_result_upload]
    return [pipeline]

def testPipelines(ctx):
    pipelines = []

    if config["litmus"]:
        pipelines += litmus(ctx, "decomposed")

    storage = "posix"
    if "[decomposed]" in ctx.build.title.lower():
        storage = "decomposed"

    if "skip" not in config["cs3ApiTests"] or not config["cs3ApiTests"]["skip"]:
        pipelines.append(cs3ApiTests(ctx, storage, "default"))
    if "skip" not in config["wopiValidatorTests"] or not config["wopiValidatorTests"]["skip"]:
        pipelines.append(wopiValidatorTests(ctx, storage, "builtin", "default"))
        pipelines.append(wopiValidatorTests(ctx, storage, "cs3", "default"))

    pipelines += localApiTestPipeline(ctx)

    if "skip" not in config["apiTests"] or not config["apiTests"]["skip"]:
        pipelines += apiTests(ctx)

    enable_watch_fs = [False]
    if ctx.build.event == "cron" or "full-ci" in ctx.build.title.lower():
        enable_watch_fs.append(True)

    for run_with_watch_fs_enabled in enable_watch_fs:
        pipelines += e2eTestPipeline(ctx, run_with_watch_fs_enabled) + multiServiceE2ePipeline(ctx, run_with_watch_fs_enabled)

    if ("skip" not in config["k6LoadTests"] or not config["k6LoadTests"]["skip"]) and ("k6-test" in ctx.build.title.lower() or ctx.build.event == "cron"):
        pipelines += k6LoadTests(ctx)

    return pipelines

def getGoBinForTesting(ctx):
    return [{
        "name": "get-go-bin-cache",
        "steps": checkGoBinCache() +
                 cacheGoBin(),
        "when": [
            event["tag"],
            event["cron"],
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
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": [
            "bash -x %s/tests/config/woodpecker/check_go_bin_cache.sh %s %s" % (dirs["baseGo"], dirs["baseGo"], dirs["gobinTar"]),
        ],
    }]

def cacheGoBin():
    return [
        {
            "name": "bingo-get",
            "image": OC_CI_GOLANG,
            "commands": [
                ". ./.env",
                "if $BIN_CACHE_FOUND; then exit 0; fi",
                "make bingo-update",
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
        {
            "name": "archive-go-bin",
            "image": OC_UBUNTU,
            "commands": [
                ". ./.env",
                "if $BIN_CACHE_FOUND; then exit 0; fi",
                "tar -czf %s /go/bin" % dirs["gobinTarPath"],
            ],
        },
        {
            "name": "cache-go-bin",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                ". ./.env",
                "if $BIN_CACHE_FOUND; then exit 0; fi",
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
                "tar -xmf %s -C /" % dirs["gobinTarPath"],
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
            "name": "open-search",
            "image": OPEN_SEARCH,
            "detach": True,
            "environment": {
                "discovery.type": "single-node",
                "DISABLE_INSTALL_DEMO_CONFIG": True,
                "DISABLE_SECURITY_PLUGIN": True,
            },
            "entrypoint": ["/usr/share/opensearch/opensearch-docker-entrypoint.sh", "opensearch"],
        },
        {
            "name": "wait-for-open-search",
            "image": OC_CI_ALPINE,
            "commands": [
                "bash -c '" +
                "until curl -sS \"http://open-search:9200/_cat/health?h=status\" | grep \"green\\|yellow\"; do\n" +
                "  echo \"Waiting for http://open-search:9200 to be healthy...\"\n" +
                "  sleep 5\n" +
                "done\n" +
                "echo \"http://open-search:9200 healthy...\"\n" +
                "'",
            ],
        },
        {
            "name": "test",
            "image": OC_CI_GOLANG,
            "environment": {
                "SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ADDRESSES": "http://open-search:9200",
            },
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
                "endpoint": CACHE_S3_SERVER,
                "bucket": "cache",
                "source": "cache/**/*",
                "target": "%s/%s" % (repo_slug, ctx.build.commit + "-${CI_PIPELINE_NUMBER}"),
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
            event["base"],
            event["cron"],
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

def scanOpencloud(ctx):
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
            event["base"],
            event["cron"],
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
            event["base"],
            event["cron"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "base"),
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
            event["base"],
            event["cron"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                },
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
            event["base"],
            event["cron"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "lint"),
                },
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
                    event["base"],
                    event["cron"],
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
    enable_watch_fs = [False]
    if ctx.build.event == "cron" or "full-ci" in ctx.build.title.lower():
        with_remote_php.append(False)
        enable_watch_fs.append(True)

    storages = ["posix"]
    if "[decomposed]" in ctx.build.title.lower():
        storages = ["decomposed"]

    defaults = {
        "suites": {},
        "skip": False,
        "extraTestEnvironment": {},
        "extraServerEnvironment": {},
        "storages": storages,
        "accounts_hash_difficulty": 4,
        "emailNeeded": False,
        "antivirusNeeded": False,
        "tikaNeeded": False,
        "federationServer": False,
        "collaborationServiceNeeded": False,
        "extraCollaborationEnvironment": {},
        "withRemotePhp": with_remote_php,
        "enableWatchFs": enable_watch_fs,
        "ldapNeeded": False,
    }

    if "localApiTests" in config:
        for name, matrix in config["localApiTests"].items():
            if "skip" not in matrix or not matrix["skip"]:
                params = {}
                for item in defaults:
                    params[item] = matrix[item] if item in matrix else defaults[item]
                for storage in params["storages"]:
                    for run_with_remote_php in params["withRemotePhp"]:
                        for run_with_watch_fs_enabled in params["enableWatchFs"]:
                            pipeline = {
                                "name": "%s-%s%s-%s%s" % ("CLI" if name.startswith("cli") else "API", name, "-withoutRemotePhp" if not run_with_remote_php else "", "decomposed" if name.startswith("cli") else storage, "-watchfs" if run_with_watch_fs_enabled else ""),
                                "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                                         (tikaService() if params["tikaNeeded"] else []) +
                                         (waitForServices("online-offices", ["collabora:9980", "onlyoffice:443", "fakeoffice:8080"]) if params["collaborationServiceNeeded"] else []) +
                                         (waitForClamavService() if params["antivirusNeeded"] else []) +
                                         (waitForEmailService() if params["emailNeeded"] else []) +
                                         (ldapService() if params["ldapNeeded"] else []) +
                                         (waitForLdapService() if params["ldapNeeded"] else []) +
                                         opencloudServer(storage, params["accounts_hash_difficulty"], extra_server_environment = params["extraServerEnvironment"], with_wrapper = True, tika_enabled = params["tikaNeeded"], watch_fs_enabled = run_with_watch_fs_enabled) +
                                         (opencloudServer(storage, params["accounts_hash_difficulty"], deploy_type = "federation", extra_server_environment = params["extraServerEnvironment"], watch_fs_enabled = run_with_watch_fs_enabled) if params["federationServer"] else []) +
                                         ((wopiCollaborationService("fakeoffice") + wopiCollaborationService("collabora") + wopiCollaborationService("onlyoffice")) if params["collaborationServiceNeeded"] else []) +
                                         (openCloudHealthCheck("wopi", ["wopi-collabora:9304", "wopi-onlyoffice:9304", "wopi-fakeoffice:9304"]) if params["collaborationServiceNeeded"] else []) +
                                         localApiTests(name, params["suites"], storage, params["extraTestEnvironment"], run_with_remote_php) +
                                         logRequests(),
                                "services": (emailService() if params["emailNeeded"] else []) +
                                            (clamavService() if params["antivirusNeeded"] else []) +
                                            ((fakeOffice() + collaboraService() + onlyofficeService()) if params["collaborationServiceNeeded"] else []),
                                "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
                                "when": [
                                    event["base"],
                                    event["cron"],
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

def localApiTests(name, suites, storage = "decomposed", extra_environment = {}, with_remote_php = False):
    test_dir = "%s/tests/acceptance" % dirs["base"]
    expected_failures_file = "%s/expected-failures-localAPI-on-%s-storage.md" % (test_dir, storage)

    environment = {
        "TEST_SERVER_URL": OC_URL,
        "TEST_SERVER_FED_URL": OC_FED_URL,
        "SEND_SCENARIO_LINE_REFERENCES": True,
        "STORAGE_DRIVER": storage,
        "BEHAT_SUITES": ",".join(suites),
        "BEHAT_FILTER_TAGS": "~@skip&&~@skipOnOpencloud-%s-Storage" % storage,
        "EXPECTED_FAILURES_FILE": expected_failures_file,
        "UPLOAD_DELETE_WAIT_TIME": "1" if storage == "owncloud" else 0,
        "OC_WRAPPER_URL": "http://%s:5200" % OC_SERVER_NAME,
        "WITH_REMOTE_PHP": with_remote_php,
        "COLLABORATION_SERVICE_URL": "http://wopi-fakeoffice:9300",
        "OC_STORAGE_PATH": "$HOME/.opencloud/storage/users",
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
        "name": "cs3ApiTests-%s" % storage,
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 opencloudServer(storage, accounts_hash_difficulty, deploy_type = "cs3api_validator") +
                 [
                     {
                         "name": "cs3ApiTests",
                         "image": OC_CS3_API_VALIDATOR,
                         "environment": {},
                         "commands": [
                             "apk --no-cache add curl",
                             "curl '%s/graph/v1.0/users' -k -X POST  --data-raw '{\"onPremisesSamAccountName\":\"marie\",\"displayName\":\"Marie Curie\",\"mail\":\"marie@opencloud.eu\",\"passwordProfile\":{\"password\":\"radioactivity\"}}' -uadmin:admin" % OC_URL,
                             "/usr/bin/cs3api-validator /var/lib/cs3api-validator --endpoint=%s:9142" % OC_SERVER_NAME,
                         ],
                     },
                 ],
        "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
        "when": [
            event["base"],
            event["cron"],
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
    extra_server_environment = {}

    if wopiServerType == "cs3":
        wopiServer = [
            {
                "name": "wopi-fakeoffice",
                "image": "cs3org/wopiserver:v10.4.0",
                "detach": True,
                "commands": [
                    "cp %s/tests/config/woodpecker/wopiserver.conf /etc/wopi/wopiserver.conf" % (dirs["base"]),
                    "echo 123 > /etc/wopi/wopisecret",
                    "/app/wopiserver.py",
                ],
            },
        ]
    else:
        extra_server_environment = {
            "OC_EXCLUDE_RUN_SERVICES": "app-provider",
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
        "name": "wopiValidatorTests-%s-%s" % (wopiServerType, storage),
        "services": fakeOffice(),
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 waitForServices("fake-office", ["fakeoffice:8080"]) +
                 opencloudServer(storage, accounts_hash_difficulty, deploy_type = "wopi_validator", extra_server_environment = extra_server_environment) +
                 wopiServer +
                 waitForServices("wopi-fakeoffice", ["wopi-fakeoffice:9300"]) +
                 [
                     {
                         "name": "prepare-test-file",
                         "image": OC_CI_ALPINE,
                         "environment": {},
                         "commands": [
                             "curl -v -X PUT '%s/remote.php/webdav/test.wopitest' -k --fail --retry-connrefused --retry 7 --retry-all-errors -u admin:admin -D headers.txt" % OC_URL,
                             "cat headers.txt",
                             "export FILE_ID=$(cat headers.txt | sed -n -e 's/^.*Oc-Fileid: //p')",
                             "export URL=\"%s/app/open?app_name=FakeOffice&file_id=$FILE_ID\"" % OC_URL,
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
            event["base"],
            event["cron"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "acceptance-tests"),
                },
            },
        ],
    }

def coreApiTests(ctx, part_number = 1, number_of_parts = 1, with_remote_php = False, accounts_hash_difficulty = 4, watch_fs_enabled = False):
    storage = "posix"
    if "[decomposed]" in ctx.build.title.lower():
        storage = "decomposed"
    filterTags = "~@skipOnOpencloud-%s-Storage" % storage
    test_dir = "%s/tests/acceptance" % dirs["base"]
    expected_failures_file = "%s/expected-failures-API-on-%s-storage.md" % (test_dir, storage)

    return {
        "name": "Core-API-Tests-%s%s-%s%s" % (part_number, "-withoutRemotePhp" if not with_remote_php else "", storage, "-watchfs" if watch_fs_enabled else ""),
        "steps": restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBinPath"]) +
                 opencloudServer(storage, accounts_hash_difficulty, with_wrapper = True, watch_fs_enabled = watch_fs_enabled) +
                 [
                     {
                         "name": "oC10ApiTests-%s" % part_number,
                         "image": OC_CI_PHP % DEFAULT_PHP_VERSION,
                         "environment": {
                             "TEST_SERVER_URL": OC_URL,
                             "OC_REVA_DATA_ROOT": "%s" % (dirs["opencloudRevaDataRoot"] if storage == "owncloud" else ""),
                             "SEND_SCENARIO_LINE_REFERENCES": True,
                             "STORAGE_DRIVER": storage,
                             "BEHAT_FILTER_TAGS": filterTags,
                             "DIVIDE_INTO_NUM_PARTS": number_of_parts,
                             "RUN_PART": part_number,
                             "ACCEPTANCE_TEST_TYPE": "core-api",
                             "EXPECTED_FAILURES_FILE": expected_failures_file,
                             "UPLOAD_DELETE_WAIT_TIME": "1" if storage == "owncloud" else 0,
                             "OC_WRAPPER_URL": "http://%s:5200" % OC_SERVER_NAME,
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
            event["base"],
            event["cron"],
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
    enable_watch_fs = [False]
    if ctx.build.event == "cron" or "full-ci" in ctx.build.title.lower():
        with_remote_php.append(False)
        enable_watch_fs.append(True)

    defaults = {
        "withRemotePhp": with_remote_php,
        "enableWatchFs": enable_watch_fs,
    }

    for runPart in range(1, config["apiTests"]["numberOfParts"] + 1):
        for run_with_remote_php in defaults["withRemotePhp"]:
            for run_with_watch_fs_enabled in defaults["enableWatchFs"]:
                if not debugPartsEnabled or (debugPartsEnabled and runPart in debugParts):
                    pipelines.append(coreApiTests(ctx, runPart, config["apiTests"]["numberOfParts"], run_with_remote_php, watch_fs_enabled = run_with_watch_fs_enabled))

    return pipelines

def e2eTestPipeline(ctx, watch_fs_enabled = False):
    defaults = {
        "skip": False,
        "suites": [],
        "xsuites": [],
        "totalParts": 0,
        "tikaNeeded": False,
        "reportTracing": False,
    }

    extra_server_environment = {
        "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST": "%s" % dirs["bannedPasswordList"],
        "OC_SHOW_USER_EMAIL_IN_RESULTS": True,
        # Needed for enabling all roles
        "GRAPH_AVAILABLE_ROLES": "%s" % GRAPH_AVAILABLE_ROLES,
    }

    e2e_trigger = [
        event["base"],
        event["cron"],
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

    if "skip-e2e" in ctx.build.title.lower():
        return pipelines

    if ctx.build.event == "tag":
        return pipelines

    storage = "posix"
    if "[decomposed]" in ctx.build.title.lower():
        storage = "decomposed"

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

        if "with-tracing" in ctx.build.title.lower():
            params["reportTracing"] = True

        steps_before = \
            restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBin"]) + \
            restoreWebCache() + \
            restoreWebPnpmCache() + \
            restoreBrowsersCache() + \
            (tikaService() if params["tikaNeeded"] else []) + \
            opencloudServer(storage, extra_server_environment = extra_server_environment, tika_enabled = params["tikaNeeded"], watch_fs_enabled = watch_fs_enabled)

        step_e2e = {
            "name": "e2e-tests",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "environment": {
                "OC_BASE_URL": OC_DOMAIN,
                "HEADLESS": True,
                "RETRY": "1",
                "WEB_UI_CONFIG_FILE": "%s/%s" % (dirs["base"], dirs["opencloudConfig"]),
                "LOCAL_UPLOAD_DIR": "/uploads",
                "PLAYWRIGHT_BROWSERS_PATH": "%s/%s" % (dirs["base"], ".playwright"),
                "BROWSER": "chromium",
                "REPORT_TRACING": params["reportTracing"],
            },
            "commands": [
                "cd %s/tests/e2e" % dirs["web"],
            ],
        }

        steps_after = uploadTracingResult(ctx)

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
                    "name": "e2e-tests-%s-%s-%s%s" % (name, run_part, storage, "-watchfs" if watch_fs_enabled else ""),
                    "steps": steps_before + [run_e2e] + steps_after,
                    "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx) + buildWebCache(ctx)),
                    "when": e2e_trigger,
                })
        else:
            step_e2e["commands"].append("bash run-e2e.sh %s" % e2e_args)
            pipelines.append({
                "name": "e2e-tests-%s-%s%s" % (name, storage, "-watchfs" if watch_fs_enabled else ""),
                "steps": steps_before + [step_e2e] + steps_after,
                "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx) + buildWebCache(ctx)),
                "when": e2e_trigger,
            })

    return pipelines

def multiServiceE2ePipeline(ctx, watch_fs_enabled = False):
    pipelines = []

    defaults = {
        "skip": False,
        "suites": [],
        "xsuites": [],
        "tikaNeeded": False,
        "reportTracing": False,
    }

    e2e_trigger = [
        event["base"],
        event["cron"],
        {
            "event": "pull_request",
            "path": {
                "exclude": skipIfUnchanged(ctx, "e2e-tests"),
            },
        },
    ]

    if "skip-e2e" in ctx.build.title.lower():
        return pipelines

    # run this pipeline only for cron jobs and full-ci PRs
    if not "full-ci" in ctx.build.title.lower() and ctx.build.event != "cron":
        return pipelines

    storage = "posix"
    if "[decomposed]" in ctx.build.title.lower():
        storage = "decomposed"

    extra_server_environment = {
        "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST": "%s" % dirs["bannedPasswordList"],
        "OC_JWT_SECRET": "some-opencloud-jwt-secret",
        "OC_SERVICE_ACCOUNT_ID": "service-account-id",
        "OC_SERVICE_ACCOUNT_SECRET": "service-account-secret",
        "OC_EXCLUDE_RUN_SERVICES": "storage-users",
        "OC_GATEWAY_GRPC_ADDR": "0.0.0.0:9142",
        "SETTINGS_GRPC_ADDR": "0.0.0.0:9191",
        "GATEWAY_STORAGE_USERS_MOUNT_ID": "storage-users-id",
        "OC_SHOW_USER_EMAIL_IN_RESULTS": True,
        # Needed for enabling all roles
        "GRAPH_AVAILABLE_ROLES": "%s" % GRAPH_AVAILABLE_ROLES,
    }

    if watch_fs_enabled:
        extra_server_environment["STORAGE_USES_POSIX_WATCH_FS"] = True

    storage_users_environment = {
        "OC_CORS_ALLOW_ORIGINS": "%s,https://%s:9201" % (OC_URL, OC_SERVER_NAME),
        "STORAGE_USERS_JWT_SECRET": "some-opencloud-jwt-secret",
        "STORAGE_USERS_MOUNT_ID": "storage-users-id",
        "STORAGE_USERS_SERVICE_ACCOUNT_ID": "service-account-id",
        "STORAGE_USERS_SERVICE_ACCOUNT_SECRET": "service-account-secret",
        "STORAGE_USERS_GATEWAY_GRPC_ADDR": "%s:9142" % OC_SERVER_NAME,
        "STORAGE_USERS_EVENTS_ENDPOINT": "%s:9233" % OC_SERVER_NAME,
        "STORAGE_USERS_DATA_GATEWAY_URL": "%s/data" % OC_URL,
        "OC_CACHE_STORE": "nats-js-kv",
        "OC_CACHE_STORE_NODES": "%s:9233" % OC_SERVER_NAME,
        "MICRO_REGISTRY_ADDRESS": "%s:9233" % OC_SERVER_NAME,
        "OC_BASE_DATA_PATH": dirs["multiServiceOcBaseDataPath"],
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

    storage_users_services = startOpenCloudService("storage-users", "storageusers1", storage_users1_environment) + \
                             startOpenCloudService("storage-users", "storageusers2", storage_users2_environment) + \
                             openCloudHealthCheck("storage-users", ["storageusers1:9159", "storageusers2:9159"])

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

        if "with-tracing" in ctx.build.title.lower():
            params["reportTracing"] = True

        steps = \
            restoreBuildArtifactCache(ctx, dirs["opencloudBinArtifact"], dirs["opencloudBin"]) + \
            restoreWebCache() + \
            restoreWebPnpmCache() + \
            restoreBrowsersCache() + \
            tikaService() + \
            opencloudServer(storage, extra_server_environment = extra_server_environment, tika_enabled = params["tikaNeeded"]) + \
            storage_users_services + \
            [{
                "name": "e2e-tests",
                "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
                "environment": {
                    "OC_BASE_URL": OC_DOMAIN,
                    "HEADLESS": True,
                    "RETRY": "1",
                    "REPORT_TRACING": params["reportTracing"],
                    "PLAYWRIGHT_BROWSERS_PATH": "%s/%s" % (dirs["base"], ".playwright"),
                    "BROWSER": "chromium",
                },
                "commands": [
                    "cd %s/tests/e2e" % dirs["web"],
                    "bash run-e2e.sh %s" % e2e_args,
                ],
            }] + \
            uploadTracingResult(ctx)

        pipelines.append({
            "name": "e2e-tests-multi-service%s" % ("-watchfs" if watch_fs_enabled else ""),
            "steps": steps,
            "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx) + buildWebCache(ctx)),
            "when": e2e_trigger,
        })
    return pipelines

def uploadTracingResult(ctx):
    status = ["failure"]
    if "with-tracing" in ctx.build.title.lower():
        status = ["failure", "success"]

    return [{
        "name": "upload-tracing-result",
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": [
            "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
            "mc cp -a %s/reports/e2e/playwright/tracing/* s3/$PUBLIC_BUCKET/web/tracing/$CI_REPO_NAME/$CI_PIPELINE_NUMBER/" % dirs["web"],
            "cd %s/reports/e2e/playwright/tracing/" % dirs["web"],
            'echo "To see the trace, please open the following link in the console"',
            'for f in *.zip; do echo "npx playwright show-trace $MC_HOST/$PUBLIC_BUCKET/web/tracing/$CI_REPO_NAME/$CI_PIPELINE_NUMBER/$f \n"; done',
        ],
        "when": {
            "status": status,
        },
    }]

def dockerReleases(ctx):
    pipelines = []
    docker_repos = []
    build_type = ""

    if ctx.build.event == "tag":
        tag = ctx.build.ref.replace("refs/tags/v", "").lower()

        is_production = False
        for prod_tag in config["dockerReleases"]["production"]["tags"]:
            if tag.startswith(prod_tag):
                is_production = True
                break

        if is_production:
            docker_repos.append(config["dockerReleases"]["production"]["repo"])
            build_type = config["dockerReleases"]["production"]["build_type"]

        else:
            docker_repos.append(config["dockerReleases"]["rolling"]["repo"])
            build_type = config["dockerReleases"]["rolling"]["build_type"]

    else:
        docker_repos.append(config["dockerReleases"]["daily"]["repo"])
        build_type = config["dockerReleases"]["daily"]["build_type"]

    for repo in docker_repos:
        repo_pipelines = []
        repo_pipelines.append(dockerRelease(ctx, repo, build_type))

        # manifest = releaseDockerManifest(ctx, repo, build_type)
        # manifest["depends_on"] = getPipelineNames(repo_pipelines)
        # repo_pipelines.append(manifest)

        readme = releaseDockerReadme(repo, build_type)
        readme["depends_on"] = getPipelineNames(repo_pipelines)
        repo_pipelines.append(readme)

        pipelines.extend(repo_pipelines)

    return pipelines

def dockerRelease(ctx, repo, build_type):
    build_args = {
        "REVISION": "%s" % ctx.build.commit,
        "VERSION": "%s" % (ctx.build.ref.replace("refs/tags/", "") if ctx.build.event == "tag" else "daily"),
    }

    depends_on = getPipelineNames(getGoBinForTesting(ctx))

    if ctx.build.event == "tag":
        depends_on = []

    return {
        "name": "container-build-%s" % build_type,
        "steps": makeNodeGenerate("") +
                 makeGoGenerate("") + [
            {
                "name": "dryrun",
                "image": PLUGINS_DOCKER_BUILDX,
                "settings": {
                    "dry_run": True,
                    "platforms": "linux/amd64",  # do dry run only on the native platform
                    "repo": "%s,quay.io/%s" % (repo, repo),
                    "auto_tag": False if build_type == "daily" else True,
                    "tag": "daily" if build_type == "daily" else "",
                    "default_tag": "daily",
                    "dockerfile": "opencloud/docker/Dockerfile.multiarch",
                    "build_args": build_args,
                    "pull_image": False,
                    "http_proxy": {
                        "from_secret": "ci_http_proxy",
                    },
                    "https_proxy": {
                        "from_secret": "ci_http_proxy",
                    },
                },
                "when": [event["pull_request"]],
            },
            {
                "name": "build-and-push",
                "image": PLUGINS_DOCKER_BUILDX,
                "settings": {
                    "repo": "%s,quay.io/%s" % (repo, repo),
                    "platforms": "linux/amd64,linux/arm64",  # we can add remote builders
                    "auto_tag": False if build_type == "daily" else True,
                    "tag": "daily" if build_type == "daily" else "",
                    "default_tag": "daily",
                    "dockerfile": "opencloud/docker/Dockerfile.multiarch",
                    "build_args": build_args,
                    "pull_image": False,
                    "http_proxy": {
                        "from_secret": "ci_http_proxy",
                    },
                    "https_proxy": {
                        "from_secret": "ci_http_proxy",
                    },
                    "logins": [
                        {
                            "registry": "https://index.docker.io/v1/",
                            "username": {
                                "from_secret": "docker_username",
                            },
                            "password": {
                                "from_secret": "docker_password",
                            },
                        },
                        {
                            "registry": "https://quay.io",
                            "username": {
                                "from_secret": "quay_username",
                            },
                            "password": {
                                "from_secret": "quay_password",
                            },
                        },
                    ],
                },
                "when": [
                    event["cron"],
                    event["base"],
                    event["tag"],
                ],
            },
        ],
        "depends_on": depends_on,
        "when": [
            event["cron"],
            event["base"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "build-docker"),
                },
            },
            event["tag"],
        ],
    }

def binaryReleases(ctx):
    pipelines = []
    depends_on = getPipelineNames(getGoBinForTesting(ctx))

    for os in config["binaryReleases"]["os"]:
        pipelines.append(binaryRelease(ctx, os, depends_on))

    return pipelines

def binaryRelease(ctx, arch, depends_on = []):
    return {
        "name": "binaries-%s" % arch,
        "steps": makeNodeGenerate("") +
                 makeGoGenerate("") + [
            {
                "name": "build",
                "image": OC_CI_GOLANG,
                "environment": {
                    "VERSION": (ctx.build.ref.replace("refs/tags/", "") if ctx.build.event == "tag" else "daily"),
                    "HTTP_PROXY": {
                        "from_secret": "ci_http_proxy",
                    },
                    "HTTPS_PROXY": {
                        "from_secret": "ci_http_proxy",
                    },
                },
                "commands": [
                    "make -C opencloud release-%s" % arch,
                ],
            },
            {
                "name": "finish",
                "image": OC_CI_GOLANG,
                "environment": CI_HTTP_PROXY_ENV,
                "commands": [
                    "make -C opencloud release-finish",
                ],
                "when": [
                    event["cron"],
                    event["base"],
                    event["tag"],
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
                        "opencloud/dist/release/*",
                    ],
                    "title": ctx.build.ref.replace("refs/tags/v", ""),
                    "prerelease": len(ctx.build.ref.split("-")) > 1,
                },
                "when": [
                    event["tag"],
                ],
            },
        ],
        "depends_on": depends_on,
        "when": [
            event["cron"],
            event["base"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "build-binary"),
                },
            },
            event["tag"],
        ],
    }

def licenseCheck(ctx):
    return {
        "name": "check-licenses",
        "steps": restoreGoBinCache() + [
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
                    "prerelease": len(ctx.build.ref.split("-")) > 1,
                },
                "when": [
                    event["tag"],
                ],
            },
        ],
        "when": [
            event["base"],
            event["pull_request"],
            event["tag"],
        ],
        "workspace": workspace,
    }

def readyReleaseGo():
    return [{
        "name": "ready-release-go",
        "steps": [
            {
                "name": "release-helper",
                "image": READY_RELEASE_GO,
                "settings": {
                    "git_email": "devops@opencloud.eu",
                    "forge_type": "github",
                    "forge_token": {
                        "from_secret": "github_token",
                    },
                },
            },
        ],
        "when": [event["base"]],
    }]

def releaseDockerReadme(repo, build_type):
    return {
        "name": "readme-%s" % build_type,
        "steps": [
            {
                "name": "push-docker",
                "image": CHKO_DOCKER_PUSHRM,
                "environment": {
                    "DOCKER_USER": {
                        "from_secret": "docker_username",
                    },
                    "DOCKER_PASS": {
                        "from_secret": "docker_password",
                    },
                    "PUSHRM_TARGET": repo,
                    "PUSHRM_SHORT": "Docker images for %s" % repo,
                    "PUSHRM_FILE": "README.md",
                },
            },
            {
                "name": "push-quay",
                "image": CHKO_DOCKER_PUSHRM,
                "environment": {
                    "APIKEY__QUAY_IO": {
                        "from_secret": "quay_apikey",
                    },
                    "PUSHRM_TARGET": "quay.io/%s" % repo,
                    "PUSHRM_FILE": "README.md",
                    "PUSHRM_PROVIDER": "quay",
                },
            },
        ],
        "when": [
            event["cron"],
            event["base"],
            event["tag"],
        ],
    }

def makeNodeGenerate(module):
    if module == "":
        make = "make"
    else:
        make = "make -C %s" % module
    return [
        {
            "name": "generate nodejs",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "environment": {
                "CHROMEDRIVER_SKIP_DOWNLOAD": True,  # install fails on arm and chromedriver is a test only dependency
            },
            "commands": [
                "pnpm config set store-dir ./.pnpm-store",
                "for i in $(seq 3); do %s node-generate-prod && break || sleep 1; done" % make,
            ],
        },
    ]

def makeGoGenerate(module):
    if module == "":
        make = "make"
    else:
        make = "make -C %s" % module
    return [
        {
            "name": "generate go",
            "image": OC_CI_GOLANG,
            "commands": [
                "for i in $(seq 3); do %s go-generate && break || sleep 1; done" % make,
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
    ]

def notifyMatrix(ctx):
    result = [{
        "name": "chat-notifications",
        "skip_clone": True,
        "runs_on": ["success", "failure"],
        "depends_on": getPipelineNames(testPipelines(ctx)),
        "steps": [
            {
                "name": "notify-matrix",
                "image": OC_CI_GOLANG,
                "environment": {
                    "HTTP_PROXY": {
                        "from_secret": "ci_http_proxy",
                    },
                    "HTTPS_PROXY": {
                        "from_secret": "ci_http_proxy",
                    },
                    "MATRIX_HOME_SERVER": "matrix.org",
                    "MATRIX_ROOM_ALIAS": {
                        "from_secret": "opencloud-notifications-channel",
                    },
                    "MATRIX_USER": {
                        "from_secret": "opencloud-notifications-user",
                    },
                    "MATRIX_PASSWORD": {
                        "from_secret": "opencloud-notifications-user-password",
                    },
                    "QA_REPO": "https://github.com/opencloud-eu/qa.git",
                    "QA_REPO_BRANCH": "main",
                    "CI_WOODPECKER_URL": "https://ci.opencloud.eu/",
                    "CI_REPO_ID": "3",
                    "CI_WOODPECKER_TOKEN": "no-auth-needed-on-this-repo",
                },
                "commands": [
                    "git clone --single-branch --branch $QA_REPO_BRANCH $QA_REPO /tmp/qa",
                    "cd /tmp/qa/scripts/matrix-notification/",
                    "go run matrix-notification.go",
                ],
            },
        ],
        "when": [
            event["cron"],
            event["base"],
            event["pull_request"],
        ],
    }]

    return result

def opencloudServer(storage = "decomposed", accounts_hash_difficulty = 4, depends_on = [], deploy_type = "", extra_server_environment = {}, with_wrapper = False, tika_enabled = False, watch_fs_enabled = False):
    user = "0:0"
    container_name = OC_SERVER_NAME
    environment = {
        "OC_URL": OC_URL,
        "OC_CONFIG_DIR": "/root/.opencloud/config",  # needed for checking config later
        "STORAGE_USERS_DRIVER": "%s" % storage,
        "PROXY_ENABLE_BASIC_AUTH": True,
        "WEB_UI_CONFIG_FILE": "%s/%s" % (dirs["base"], dirs["opencloudConfig"]),
        "OC_LOG_LEVEL": "error",
        "IDM_CREATE_DEMO_USERS": True,  # needed for litmus and cs3api-validator tests
        "IDM_ADMIN_PASSWORD": "admin",  # override the random admin password from `opencloud init`
        "FRONTEND_SEARCH_MIN_LENGTH": "2",
        "OC_ASYNC_UPLOADS": True,
        "OC_EVENTS_ENABLE_TLS": False,
        "NATS_NATS_HOST": "0.0.0.0",
        "NATS_NATS_PORT": 9233,
        "OC_JWT_SECRET": "some-opencloud-jwt-secret",
        "EVENTHISTORY_STORE": "memory",
        "OC_TRANSLATION_PATH": "%s/tests/config/translations" % dirs["base"],
        "ACTIVITYLOG_WRITE_BUFFER_DURATION": "0",  # Disable write buffer so that test expectations are met in time
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

    if storage == "posix":
        environment["STORAGE_USERS_ID_CACHE_STORE"] = "nats-js-kv"

    if deploy_type == "":
        environment["FRONTEND_OCS_ENABLE_DENIALS"] = True

        # fonts map for txt thumbnails (including unicode support)
        environment["THUMBNAILS_TXT_FONTMAP_FILE"] = "%s/tests/config/woodpecker/fontsMap.json" % (dirs["base"])

    if deploy_type == "cs3api_validator":
        environment["GATEWAY_GRPC_ADDR"] = "0.0.0.0:9142"  #  make gateway available to cs3api-validator
        environment["OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD"] = False

    if deploy_type == "wopi_validator":
        environment["GATEWAY_GRPC_ADDR"] = "0.0.0.0:9142"  # make gateway available to wopi server
        environment["APP_PROVIDER_EXTERNAL_ADDR"] = "eu.opencloud.api.app-provider"
        environment["APP_PROVIDER_DRIVER"] = "wopi"
        environment["APP_PROVIDER_WOPI_APP_NAME"] = "FakeOffice"
        environment["APP_PROVIDER_WOPI_APP_URL"] = "http://fakeoffice:8080"
        environment["APP_PROVIDER_WOPI_INSECURE"] = True
        environment["APP_PROVIDER_WOPI_WOPI_SERVER_EXTERNAL_URL"] = "http://wopi-fakeoffice:9300"
        environment["APP_PROVIDER_WOPI_FOLDER_URL_BASE_URL"] = OC_URL

    if deploy_type == "federation":
        environment["OC_URL"] = OC_FED_URL
        environment["PROXY_HTTP_ADDR"] = OC_FED_DOMAIN
        container_name = FED_OC_SERVER_NAME

    if tika_enabled:
        environment["FRONTEND_FULL_TEXT_SEARCH_ENABLED"] = True
        environment["SEARCH_EXTRACTOR_TYPE"] = "tika"
        environment["SEARCH_EXTRACTOR_TIKA_TIKA_URL"] = "http://tika:9998"
        environment["SEARCH_EXTRACTOR_CS3SOURCE_INSECURE"] = True

    if watch_fs_enabled:
        environment["STORAGE_USES_POSIX_WATCH_FS"] = True

    # Pass in "default" accounts_hash_difficulty to not set this environment variable.
    # That will allow OpenCloud to use whatever its built-in default is.
    # Otherwise pass in a value from 4 to about 11 or 12 (default 4, for making regular tests fast)
    # The high values cause lots of CPU to be used when hashing passwords, and really slow down the tests.
    if accounts_hash_difficulty != "default":
        environment["ACCOUNTS_HASH_DIFFICULTY"] = accounts_hash_difficulty

    for item in extra_server_environment:
        environment[item] = extra_server_environment[item]

    server_commands = [
        "env | sort",
    ]
    if with_wrapper:
        server_commands += [
            "make -C %s build" % dirs["ocWrapper"],
            "%s/bin/ocwrapper serve --bin %s --url %s --admin-username admin --admin-password admin" % (dirs["ocWrapper"], dirs["opencloudBin"], environment["OC_URL"]),
        ]
    else:
        server_commands += [
            "%s server" % dirs["opencloudBin"],
        ]

    wait_for_opencloud = {
        "name": "wait-for-%s" % container_name,
        "image": OC_CI_ALPINE,
        "commands": [
            # wait for opencloud-server to be ready (5 minutes)
            "timeout 300 bash -c 'while [ $(curl -sk -uadmin:admin " +
            "%s/graph/v1.0/users/admin " % environment["OC_URL"] +
            "-w %{http_code} -o /dev/null) != 200 ]; do sleep 1; done'",
        ],
    }

    opencloud_server = {
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
            "apt-get update",
            "apt-get install -y inotify-tools xattr",
            INSTALL_LIBVIPS_COMMAND,
            "%s init --insecure true" % dirs["opencloudBin"],
            "cat $OC_CONFIG_DIR/opencloud.yaml",
            "cp tests/config/woodpecker/app-registry.yaml $OC_CONFIG_DIR/app-registry.yaml",
        ] + server_commands,
    }

    steps = [
        opencloud_server,
        wait_for_opencloud,
    ]

    # empty depends_on list makes steps to run in parallel, what we don't want
    if depends_on:
        steps[0]["depends_on"] = depends_on
        steps[1]["depends_on"] = depends_on

    return steps

def startOpenCloudService(service = None, name = None, environment = {}):
    """
    Starts an OpenCloud service in a detached container.

    Args:
        service (str): The name of the service to start.
        name (str): The name of the container.
        environment (dict): The environment variables to set in the container.

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
                INSTALL_LIBVIPS_COMMAND,
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

def redisForOCStorage(storage = "decomposed"):
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
                "apt-get update; apt-get install libvips-dev -y",
                "for i in $(seq 3); do make -C opencloud build ENABLE_VIPS=1 && break || sleep 1; done",
            ],
            "environment": CI_HTTP_PROXY_ENV,
        },
    ]

def skipIfUnchanged(ctx, type):
    if "full-ci" in ctx.build.title.lower() or ctx.build.event == "tag" or ctx.build.event == "cron":
        return []

    base = [
        ".github/**",
        ".vscode/**",
        "docs/**",
        "deployments/**",
        "CHANGELOG.md",
        "CONTRIBUTING.md",
        "LICENSE",
        "README.md",
    ]
    unit = [
        "**/*_test.go",
    ]
    acceptance = [
        "tests/acceptance/**",
    ]

    skip = []
    if type == "acceptance-tests" or type == "e2e-tests" or type == "lint":
        skip = base + unit
    elif type == "unit-tests":
        skip = base + acceptance
    elif type == "build-binary" or type == "build-docker" or type == "litmus":
        skip = base + unit + acceptance
    elif type == "cache" or type == "base":
        skip = base

    return skip

def translation_sync(ctx):
    return [{
        "name": "translation-sync",
        "steps": [
            {
                "name": "translation-update",
                "image": OC_CI_GOLANG,
                "commands": [
                    "make l10n-read",
                    "curl -o- https://raw.githubusercontent.com/transifex/cli/master/install.sh | bash",
                    ". ~/.profile",
                    "make l10n-push",
                    "make l10n-pull",
                    "rm tx",
                    "make l10n-clean",
                ],
                "environment": {
                    "TX_TOKEN": {
                        "from_secret": "tx_token",
                    },
                },
            },
            {
                "name": "translation-push",
                "image": PLUGINS_GIT_ACTION,
                "settings": {
                    "action": ["commit", "push"],
                    "branch": ctx.build.branch,
                    "message": "[tx] updated from transifex",
                    "author_name": "opencloudeu",
                    "author_email": "devops@opencloud.eu",
                    "netrc_username": {
                        "from_secret": "github_username",
                    },
                    "netrc_password": {
                        "from_secret": "github_token",
                    },
                    "empty_commit": False,
                },
            },
        ],
        "when": [
            {
                "event": "cron",
                "cron": "translation-sync",
            },
        ],
    }]

def checkStarlark(ctx):
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
            event["cron"],
            event["base"],
            {
                "event": "pull_request",
                "path": {
                    "exclude": skipIfUnchanged(ctx, "base"),
                },
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
            "endpoint": CACHE_S3_SERVER,
            "rebuild": rebuild,
            "restore": restore,
            "mount": mounts,
            "access_key": {
                "from_secret": "cache_s3_access_key",
            },
            "secret_key": {
                "from_secret": "cache_s3_secret_key",
            },
            "filename": "%s.tar" % name,
            "path": cache_path,
            "fallback_path": cache_path,
        },
    }
    return step

def purgeCache(name, flush_path, flush_age):
    return {
        "name": name,
        "skip_clone": True,
        "when": [
            event["cron"],
            event["base"],
            event["pull_request"],
        ],
        "runs_on": ["success", "failure"],
        "steps": [
            {
                "name": "purge",
                "image": MINIO_MC,
                "environment": MINIO_MC_ENV,
                "commands": [
                    "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                    "to_delete=$(mc find s3/%s/ --older-than %sd)" % (flush_path, flush_age),
                    'if [ -z "$to_delete" ]; then exit 0; fi',
                    "mc rm $to_delete",
                ],
            },
        ],
    }

def genericBuildArtifactCache(ctx, name, action, path):
    if action == "rebuild" or action == "restore":
        cache_path = "%s/%s/%s" % ("cache", repo_slug, ctx.build.commit + "-${CI_PIPELINE_NUMBER}")
        name = "%s_build_artifact_cache" % name
        return genericCache(name, action, [path], cache_path)

    if action == "purge":
        return purgeCache("purge_opencloud_build_artifact_cache", "cache/opencloud-eu/opencloud", 1)
    return []

def restoreBuildArtifactCache(ctx, name, path):
    return [genericBuildArtifactCache(ctx, name, "restore", path)]

def rebuildBuildArtifactCache(ctx, name, path):
    return [genericBuildArtifactCache(ctx, name, "rebuild", path)]

def purgeBuildArtifactCache(ctx):
    return genericBuildArtifactCache(ctx, "", "purge", [])

def purgeBrowserCache(ctx):
    return purgeCache("purge_browser_build_cache", "dev/web", 14)

def purgeTracingCache(ctx):
    return purgeCache("purge_playwright_tracing_cache", "public/web/tracing", 14)

def purgeOpencloudWebBuildCache(ctx):
    return purgeCache("purge_opencloud_web_build_cache", "dev/opencloud/web-test-runner", 14)

def purgeGoBinCache(ctx):
    return purgeCache("purge_go_bin_cache", "dev/opencloud/go-bin", 14)

def pipelineSanityChecks(pipelines):
    """pipelineSanityChecks helps the CI developers to find errors before running it

    These sanity checks are only executed on when converting starlark to yaml.
    Error outputs are only visible when the conversion is done with the woodpecker cli.

    Args:
      pipelines: pipelines to be checked, normally you should run this on the return value of main()

    Returns:
      none
    """

    # check if name length of pipeline and steps are exceeded.
    max_name_length = 50
    for pipeline in pipelines:
        pipeline_name = pipeline["name"]
        if len(pipeline_name) > max_name_length:
            print("Error: pipeline name %s is longer than 50 characters" % pipeline_name)

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

    if not config["litmus"]:
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
                 opencloudServer(storage) +
                 setupForLitmus() +
                 [
                     {
                         "name": "old-endpoint",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/webdav"' % OC_URL,
                             litmusCommand,
                         ],
                     },
                     {
                         "name": "new-endpoint",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/dav/files/admin"' % OC_URL,
                             litmusCommand,
                         ],
                     },
                     {
                         "name": "new-shared",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/dav/files/admin/Shares/new_folder/"' % OC_URL,
                             litmusCommand,
                         ],
                     },
                     {
                         "name": "old-shared",
                         "image": OC_LITMUS,
                         "environment": environment,
                         "commands": [
                             "source .env",
                             'export LITMUS_URL="%s/remote.php/webdav/Shares/new_folder/"' % OC_URL,
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
                             "export LITMUS_URL='%s/remote.php/dav/spaces/'$SPACE_ID" % OC_URL,
                             litmusCommand,
                         ],
                     },
                 ],
        "services": redisForOCStorage(storage),
        "depends_on": getPipelineNames(buildOpencloudBinaryForTesting(ctx)),
        "when": [
            event["cron"],
            event["base"],
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
            "TEST_SERVER_URL": OC_URL,
        },
        "commands": [
            "bash ./tests/config/woodpecker/setup-for-litmus.sh",
            "cat .env",
        ],
    }]

def getWoodpeckerEnvAndCheckScript(ctx):
    opencloud_git_base_url = "https://raw.githubusercontent.com/opencloud-eu/opencloud"
    path_to_woodpecker_env = "%s/%s/.woodpecker.env" % (opencloud_git_base_url, ctx.build.commit)
    path_to_check_script = "%s/%s/tests/config/woodpecker/check_web_cache.sh" % (opencloud_git_base_url, ctx.build.commit)
    return {
        "name": "get-woodpecker-env-and-check-script",
        "image": OC_UBUNTU,
        "commands": [
            "curl -s -o .woodpecker.env %s" % path_to_woodpecker_env,
            "curl -s -o check_web_cache.sh %s" % path_to_check_script,
        ],
    }

def checkForWebCache(name):
    return {
        "name": "check-for-%s-cache" % name,
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": [
            "bash -x check_web_cache.sh %s" % name,
        ],
    }

def cloneWeb():
    return {
        "name": "clone-web",
        "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
        "commands": [
            ". ./.woodpecker.env",
            "if $WEB_CACHE_FOUND; then exit 0; fi",
            "rm -rf %s" % dirs["web"],
            "git clone -b $WEB_BRANCH --single-branch --no-tags https://github.com/opencloud-eu/web.git %s" % dirs["web"],
            "cd %s && git checkout $WEB_COMMITID" % dirs["web"],
        ],
    }

def generateWebPnpmCache(ctx):
    return [
        getWoodpeckerEnvAndCheckScript(ctx),
        checkForWebCache("web-pnpm"),
        cloneWeb(),
        {
            "name": "install-pnpm",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "commands": [
                ". ./.woodpecker.env",
                "if $WEB_CACHE_FOUND; then exit 0; fi",
                "cd %s" % dirs["web"],
                'npm install --silent --global --force "$(jq -r ".packageManager" < package.json)"',
                "pnpm config set store-dir ./.pnpm-store",
                "for i in $(seq 3); do pnpm install && break || sleep 1; done",
            ],
        },
        {
            "name": "zip-pnpm",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "commands": [
                ". ./.woodpecker.env",
                "if $WEB_CACHE_FOUND; then exit 0; fi",
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
                ". ./.woodpecker.env",
                "if $WEB_CACHE_FOUND; then exit 0; fi",
                # cache using the minio/mc client to the public bucket (long term bucket)
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r -a %s s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID" % dirs["webPnpmZip"],
            ],
        },
    ]

def cacheBrowsers(ctx):
    e2e_trigger = [
        event["base"],
        event["cron"],
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

    check_browser_step = [{
        "name": "check-browsers-cache",
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": [
            "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
            "mc ls --recursive s3/$CACHE_BUCKET/web",
            "cd %s" % dirs["web"],
            "bash tests/woodpecker/script.sh check_browsers_cache",
        ],
    }]

    webPnpmCacheSteps = restoreWebPnpmCache(extra_commands = [
        "cd %s" % dirs["web"],
        ". ./.woodpecker.env",
        "if $BROWSER_CACHE_FOUND; then exit 0; fi",
        "cd %s" % dirs["base"],
    ])

    browser_cache_steps = [
        {
            "name": "install-browsers",
            "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
            "environment": {
                "PLAYWRIGHT_BROWSERS_PATH": ".playwright",
            },
            "commands": [
                "cd %s" % dirs["web"],
                ". ./.woodpecker.env",
                "if $BROWSER_CACHE_FOUND; then exit 0; fi",
                "pnpm exec playwright install --with-deps",
                "pnpm exec playwright install --list",
                "tar -czf %s .playwright" % dirs["playwrightBrowsersArchive"],
            ],
        },
        {
            "name": "upload-browsers-cache",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                "cd %s" % dirs["web"],
                ". ./.woodpecker.env",
                "if $BROWSER_CACHE_FOUND; then exit 0; fi",
                "playwright_version=$(bash tests/woodpecker/script.sh get_playwright_version)",
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r -a %s s3/$CACHE_BUCKET/web/browsers-cache/$playwright_version/" % dirs["playwrightBrowsersArchive"],
                "mc ls --recursive s3/$CACHE_BUCKET/web",
            ],
        },
    ]

    return [{
        "name": "cache-browsers",
        "depends_on": getPipelineNames(buildWebCache(ctx)),
        "steps": restoreWebCache() + check_browser_step + webPnpmCacheSteps + browser_cache_steps,
        "when": e2e_trigger,
    }]

def generateWebCache(ctx):
    return [
        getWoodpeckerEnvAndCheckScript(ctx),
        checkForWebCache("web"),
        cloneWeb(),
        {
            "name": "zip-web",
            "image": OC_UBUNTU,
            "commands": [
                ". ./.woodpecker.env",
                "if $WEB_CACHE_FOUND; then exit 0; fi",
                "if [ ! -d '%s' ]; then mkdir -p %s; fi" % (dirs["zip"], dirs["zip"]),
                "tar -czf %s webTestRunner" % dirs["webZip"],
            ],
        },
        {
            "name": "cache-web",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                ". ./.woodpecker.env",
                "if $WEB_CACHE_FOUND; then exit 0; fi",
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
            "source ./.woodpecker.env",
            "rm -rf %s" % dirs["web"],
            "mkdir -p %s" % dirs["web"],
            "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
            "mc cp -r -a s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID/web.tar.gz %s" % dirs["zip"],
        ],
    }, {
        "name": "unzip-web-cache",
        "image": OC_UBUNTU,
        "commands": [
            "tar -xf %s -C ." % dirs["webZip"],
        ],
    }]

def restoreWebPnpmCache(extra_commands = []):
    return [{
        "name": "restore-web-pnpm-cache",
        "image": MINIO_MC,
        "environment": MINIO_MC_ENV,
        "commands": extra_commands + [
            "source ./.woodpecker.env",
            "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
            "mc cp -r -a s3/$CACHE_BUCKET/opencloud/web-test-runner/$WEB_COMMITID/web-pnpm.tar.gz %s" % dirs["zip"],
        ],
    }, {
        # we need to install again because the node_modules are not cached
        "name": "unzip-and-install-pnpm",
        "image": OC_CI_NODEJS % DEFAULT_NODEJS_VERSION,
        "commands": extra_commands + [
            "cd %s" % dirs["web"],
            "rm -rf .pnpm-store",
            "tar -xf %s" % dirs["webPnpmZip"],
            'npm install --silent --global --force "$(jq -r ".packageManager" < package.json)"',
            "pnpm config set store-dir ./.pnpm-store",
            "for i in $(seq 3); do pnpm install --no-frozen-lockfile && break || sleep 1; done",
        ],
    }]

def restoreBrowsersCache():
    return [
        {
            "name": "restore-browsers-cache",
            "image": MINIO_MC,
            "environment": MINIO_MC_ENV,
            "commands": [
                "cd %s" % dirs["web"],
                "playwright_version=$(bash tests/woodpecker/script.sh get_playwright_version)",
                "mc alias set s3 $MC_HOST $AWS_ACCESS_KEY_ID $AWS_SECRET_ACCESS_KEY",
                "mc cp -r -a s3/$CACHE_BUCKET/web/browsers-cache/$playwright_version/playwright-browsers.tar.gz %s" % dirs["web"],
            ],
        },
        {
            "name": "unzip-browsers-cache",
            "image": OC_UBUNTU,
            "commands": [
                "tar -xf /woodpecker/src/github.com/opencloud-eu/opencloud/webTestRunner/playwright-browsers.tar.gz -C .",
            ],
        },
    ]

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

def ldapService():
    return [
        {
            "name": "ldap-server",
            "image": OPENLDAP,
            "detach": True,
            "environment": {
                "BITNAMI_DEBUG": "true",
                "LDAP_TLS_VERIFY_CLIENT": "never",
                "LDAP_ENABLE_TLS": "yes",
                "LDAP_TLS_CA_FILE": "/opt/bitnami/openldap/share/openldap.crt",
                "LDAP_TLS_CERT_FILE": "/opt/bitnami/openldap/share/openldap.crt",
                "LDAP_TLS_KEY_FILE": "/opt/bitnami/openldap/share/openldap.key",
                "LDAP_ROOT": "dc=opencloud,dc=eu",
                "LDAP_ADMIN_PASSWORD": "admin",
            },
            "commands": [
                "mkdir -p /opt/bitnami/openldap/share",
                "mkdir -p /tmp/custom-scripts",
                "mkdir -p /tmp/ldif-files",
                "cp tests/config/woodpecker/ldap/*.ldif /tmp/ldif-files/",
                "cp tests/config/woodpecker/ldap/docker-entrypoint-override.sh /tmp/custom-scripts/",
                "chmod +x /tmp/custom-scripts/docker-entrypoint-override.sh",
                "ls -la /tmp/ldif-files/",
                "/tmp/custom-scripts/docker-entrypoint-override.sh /opt/bitnami/scripts/openldap/run.sh",
            ],
            "backend_options": {
                "docker": {
                    "user": "0:0",
                },
            },
        },
    ]

def waitForLdapService():
    return [{
        "name": "wait-for-ldap",
        "image": OC_CI_WAIT_FOR,
        "commands": [
            "wait-for -it ldap-server:1636 -t 600",
        ],
    }]

def fakeOffice():
    return [
        {
            "name": "fakeoffice",
            "image": OC_CI_ALPINE,
            "environment": {},
            "commands": [
                "sh %s/tests/config/woodpecker/serve-hosting-discovery.sh" % (dirs["base"]),
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
        "OC_JWT_SECRET": "some-opencloud-jwt-secret",
        "COLLABORATION_WOPI_SECRET": "some-wopi-secret",
    }

    if name == "collabora":
        environment["COLLABORATION_APP_NAME"] = "Collabora"
        environment["COLLABORATION_APP_PRODUCT"] = "Collabora"
        environment["COLLABORATION_APP_ADDR"] = "https://collabora:9980"
        environment["COLLABORATION_APP_ICON"] = "https://collabora:9980/favicon.ico"
    elif name == "onlyoffice":
        environment["COLLABORATION_SERVICE_NAME"] = "collboration-onlyoffice"
        environment["COLLABORATION_APP_NAME"] = "OnlyOffice"
        environment["COLLABORATION_APP_PRODUCT"] = "OnlyOffice"
        environment["COLLABORATION_APP_ADDR"] = "https://onlyoffice"
        environment["COLLABORATION_APP_ICON"] = "https://onlyoffice/web-apps/apps/documenteditor/main/resources/img/favicon.ico"
    elif name == "fakeoffice":
        environment["COLLABORATION_SERVICE_NAME"] = "collboration-fakeoficce"
        environment["COLLABORATION_APP_NAME"] = "FakeOffice"
        environment["COLLABORATION_APP_PRODUCT"] = "Microsoft"
        environment["COLLABORATION_APP_ADDR"] = "http://fakeoffice:8080"

    environment["COLLABORATION_WOPI_SRC"] = "http://%s:9300" % service_name

    return startOpenCloudService("collaboration", service_name, environment)

def tikaService():
    return [{
        "name": "tika",
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
    opencloud_remote_environment = {
        "SSH_OC_REMOTE": {
            "from_secret": "k6_ssh_opencloud_remote",
        },
        "SSH_OC_USERNAME": {
            "from_secret": "k6_ssh_opencloud_user",
        },
        "SSH_OC_PASSWORD": {
            "from_secret": "k6_ssh_opencloud_pass",
        },
        "TEST_SERVER_URL": {
            "from_secret": "k6_ssh_opencloud_server_url",
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
    environment.update(opencloud_remote_environment)
    environment.update(k6_remote_environment)

    if "skip" in config["k6LoadTests"] and config["k6LoadTests"]["skip"]:
        return []

    opencloud_git_base_url = "https://raw.githubusercontent.com/opencloud-eu/opencloud"
    script_link = "%s/%s/tests/config/woodpecker/run_k6_tests.sh" % (opencloud_git_base_url, ctx.build.commit)

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
                "name": "opencloud-log",
                "image": OC_CI_ALPINE,
                "environment": opencloud_remote_environment,
                "commands": [
                    "curl -s -o run_k6_tests.sh %s" % script_link,
                    "apk add --no-cache openssh-client sshpass",
                    "sh %s/run_k6_tests.sh --opencloud-log" % (dirs["base"]),
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

def openCloudHealthCheck(name, services = []):
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
            "image": COLLABORA_CODE,
            "environment": {
                "DONT_GEN_SSL_CERT": "set",
                "extra_params": "--o:ssl.enable=true --o:ssl.termination=true --o:welcome.enable=false --o:net.frame_ancestors=%s" % OC_URL,
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
            "image": ONLYOFFICE_DOCUMENT_SERVER,
            "environment": {
                "WOPI_ENABLED": True,
                "USE_UNAUTHORIZED_STORAGE": True,  # self signed certificates
            },
            "commands": [
                "cp %s/tests/config/woodpecker/only-office.json /etc/onlyoffice/documentserver/local.json" % dirs["base"],
                "openssl req -x509 -newkey rsa:4096 -keyout onlyoffice.key -out onlyoffice.crt -sha256 -days 365 -batch -nodes",
                "mkdir -p /var/www/onlyoffice/Data/certs",
                "cp onlyoffice.key /var/www/onlyoffice/Data/certs/",
                "cp onlyoffice.crt /var/www/onlyoffice/Data/certs/",
                "chmod 400 /var/www/onlyoffice/Data/certs/onlyoffice.key",
                "/app/ds/run-document-server.sh",
            ],
        },
    ]
