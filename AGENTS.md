# AGENTS.md - AI Assistant Guide for the OpenCloud server

This file holds conventions of this repository that are easy to miss when changing the code. Skills for specific tasks, such as bumping reva or web, are in `.agents/skills/`. For commit messages, branch names and pull requests, follow [CONTRIBUTING.md](CONTRIBUTING.md).

## Configuration options

Each service defines its configuration as Go structs in `services/<service>/pkg/config/`. A field is set from the YAML configuration file through its `yaml` tag, and from environment variables through its `env` tag. Every field with an `env` tag also needs a `desc` and an `introductionVersion` tag:

```go
TransferTimeout time.Duration `yaml:"transfer_timeout" env:"PROXY_TRANSFER_TIMEOUT" desc:"..." introductionVersion:"..."`
```

### Names of environment variables

The prefix of an environment variable says which services it applies to, as described in the [configuration documentation](https://docs.opencloud.eu/docs/dev/server/configuration/config-system/):

- **Used by one service only:** the name of the service as the prefix, in capitals and with `_` instead of `-`, never `OC_`. For example `PROXY_TRANSFER_TIMEOUT` in the `proxy` service, or `STORAGE_USERS_GRACEFUL_SHUTDOWN_TIMEOUT` in `storage-users`.
- **Shared by more than one service:** the prefix `OC_`, for example `OC_REVA_GATEWAY`, which every service that talks to the gateway reads.
- **Shared, but adjustable per service:** both names in the `env` tag, separated by `;`, the `OC_` one first, for example `env:"OC_LOG_LEVEL;ACTIVITYLOG_LOG_LEVEL"`. The names are read in order, and the last one that is set wins (see `pkg/config/envdecode`), so the name of the service overrides the shared one for that service.

When adding an option, decide first whether other services need the same setting, and look at how the existing options of the same kind are named in the other services.

### `desc`

The `desc` tag describes the option for administrators: the documentation of the configuration options is generated from it. Write it as one or more full sentences that say what the option does, its unit (seconds, bytes, ...), the valid values, and what happens when it isn't set. It must be at least 10 characters long.

### `introductionVersion`

The `introductionVersion` tag says in which release the option first appeared. For a new option, it is always the placeholder `%%NEXT%%`:

```go
introductionVersion:"%%NEXT%%"
```

Never guess the version of the next release: the placeholders are replaced with the actual version as part of the release process, and the CI of the release pull request fails while any are left.

### Checking the tags

`make check-env-var-annotations` checks every `env` tag for a valid `introductionVersion` (a version or `%%NEXT%%`) and a `desc` of at least 10 characters. The CI runs the same check on every pull request.
