- [Developer Guide](#developer-guide)
  - [Getting Started](#getting-started)
    - [Git Clone OpenSearch Go Client Repository](#git-clone-opensearch-go-client-repository)
    - [Install Prerequisites](#install-prerequisites)
      - [Go 1.24](#go-124)
      - [Docker](#docker)
      - [Windows](#windows)
    - [Unit Testing](#unit-testing)
    - [Integration Testing](#integration-testing)
    - [Composing an OpenSearch Docker Container](#composing-an-opensearch-docker-container)
      - [Container Providers](#container-providers)
      - [Execute integration tests from your terminal](#execute-integration-tests-from-your-terminal)
  - [Advanced Cluster Configuration](#advanced-cluster-configuration)
    - [Cluster Scaling](#cluster-scaling)
    - [Heterogeneous Clusters](#heterogeneous-clusters)
      - [CPU Limits](#cpu-limits)
      - [Node Roles](#node-roles)
      - [Combining Overrides](#combining-overrides)
      - [Resetting to Defaults](#resetting-to-defaults)
    - [Testing Specific OpenSearch Versions](#testing-specific-opensearch-versions)
    - [Cluster Status and Troubleshooting](#cluster-status-and-troubleshooting)
  - [Network Latency Simulation](#network-latency-simulation)
    - [Latency Profiles](#latency-profiles)
    - [Inspecting and Clearing Latency](#inspecting-and-clearing-latency)
  - [Code Generation](#code-generation)
  - [Demo](#demo)
  - [Verification Matrix](#verification-matrix)
    - [Individual Verifications](#individual-verifications)
    - [Workload and Fanout Comparisons](#workload-and-fanout-comparisons)
    - [Full Matrix](#full-matrix)
    - [Reviewing Results](#reviewing-results)
  - [Inspecting CI Failures](#inspecting-ci-failures)
    - [Quick Summary](#quick-summary)
    - [Detailed Failure Output](#detailed-failure-output)
    - [Failure Context](#failure-context)
    - [Full Logs](#full-logs)
    - [Inspecting a Specific Run](#inspecting-a-specific-run)
  - [Lint](#lint)
    - [Markdown lint](#markdown-lint)
    - [Go lint](#go-lint)
  - [Coverage](#coverage)
  - [Use an Editor](#use-an-editor)
    - [GoLand](#goland)
    - [Vim](#vim)

# Developer Guide

So you want to contribute code to the OpenSearch Go Client? Excellent! We're glad you're here. Here's what you need to do:

## Getting Started

### Git Clone OpenSearch Go Client Repository

Fork [opensearch-project/opensearch-go](https://github.com/opensearch-project/opensearch-go) and clone locally, e.g. `git clone https://github.com/[your username]/opensearch-go.git`.

### Install Prerequisites

#### Go 1.24

OpenSearch Go Client builds using [Go](https://go.dev/doc/install) 1.24 at a minimum.

#### Docker

[Docker](https://docs.docker.com/get-docker/) is required for building some OpenSearch artifacts and executing integration tests.

#### Windows

To build the project on Windows, use [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install), the compatibility layer for running Linux applications.

Install `make`

```
sudo apt install make
```

### Unit Testing

Go has a simple tool for running tests, and we simplified it further by creating this make command:

```
make test-unit
```

Individual unit tests can be run with the following command:

```
cd folder-path/to/test;
go test -v -run TestName;
```

### Integration Testing

In order to test opensearch-go client, you need a running OpenSearch cluster. You can use Docker to accomplish this. The [Docker Compose file](.ci/opensearch/docker-compose.yml) supports the ability to run integration tests for the project in local environments. If you have not installed docker-compose, you can install it from this [link](https://docs.docker.com/compose/install/).

### Composing an OpenSearch Docker Container

Ensure that Docker is installed on your local machine. You can check by running `docker --version`. Next, navigate to your local opensearch-go repository. Run the following command to build and start the OpenSearch docker container.

```
make cluster.build cluster.start
```

This command will start the OpenSearch container using the `docker-compose.yaml` configuration file. During the build process, the necessary dependencies and files will be downloaded, which may take some time depending on your internet connection and system resources.

Once the container is built and running, you can open a web browser and navigate to localhost:9200 to access the OpenSearch docker container.

#### Container Providers

The cluster targets work with any of three container providers. When you do not specify one, the Makefile auto-detects a provider by CLI presence in this order:

1. **Colima** (`colima` on `PATH`)
2. **Rancher Desktop** (`rdctl` on `PATH`)
3. **Docker** / Docker Desktop (`docker` on `PATH`)

Override the choice explicitly with `CONTAINER_PROVIDER`:

```
CONTAINER_PROVIDER=docker make cluster.start
```

Selecting a provider does four things:

- Pins the docker context commands talk to (`colima` for Colima, `rancher-desktop` for Rancher; the Docker provider leaves your active context alone). If `DOCKER_CONTEXT` is already set in your environment, it is respected and not overridden.
- Resolves the CLI runtime `$(CTR)` (defaults to `docker` for every provider). For advanced containerd use you can force `CONTAINER_RUNTIME=nerdctl`, though docker-context pinning does not apply to nerdctl.
- Ensures the backing VM/daemon is running (`make cluster.provider.ensure`, run automatically by `cluster.start`), starting it if needed.
- Sets `vm.max_map_count=262144` through the provider's VM (`colima ssh` / `rdctl shell`) or, for the Docker provider, a privileged helper container.

Run `make cluster.runtime` to see the detected provider, docker context, and runtime.

In order to differentiate unit tests from integration tests, Go has a built-in mechanism for allowing you to logically separate your tests with [build tags](https://pkg.go.dev/cmd/go#hdr-Build_constraints). The build tag needs to be placed as close to the top of the file as possible, and must have a blank line beneath it. Hence, create all integration tests with build tag 'integration'.

#### Execute integration tests from your terminal

1. Run below command to start containers. By default, it will launch latest OpenSearch cluster.
   ```
   make cluster.build cluster.start
   ```
2. Run all integration tests.
   ```
   make test-integ race=true
   ```
3. Stop and clean containers.
   ```
   make cluster.stop cluster.clean
   ```

## Advanced Cluster Configuration

By default, `make cluster.start` launches a 3-node cluster where every node has the same resources and roles (`cluster_manager,data,ingest`). The targets below let you customize the cluster for testing weighted round-robin routing, role-based request routing, and other behavior that only surfaces with non-uniform nodes.

### Cluster Scaling

Scale the running cluster to a different number of nodes without rebuilding:

```
make cluster.scale.1    # Single-node cluster
make cluster.scale.2    # 2-node cluster
make cluster.scale.3    # Full 3-node cluster (default)
```

### Heterogeneous Clusters

Override files let you change CPU limits and node roles independently. Each target writes a Docker Compose override file under `.ci/opensearch/` that is automatically merged with `docker-compose.yml` on the next `cluster.build` or `cluster.start`. The override files are not checked into source control.

The override file paths are:

| Override   | File                                               |
| ---------- | -------------------------------------------------- |
| CPU limits | `.ci/opensearch/docker-compose.cpu-override.yml`   |
| Node roles | `.ci/opensearch/docker-compose.roles-override.yml` |

These are standard Docker Compose files. You can hand-edit them for custom configurations (e.g., different CPU ratios or role combinations not covered by the Make targets), or remove individual files to selectively reset one dimension while keeping the other:

```
# Remove only the CPU override, keep roles
rm .ci/opensearch/docker-compose.cpu-override.yml

# Remove only the roles override, keep CPU limits
rm .ci/opensearch/docker-compose.roles-override.yml
```

#### CPU Limits

Set per-node CPU limits so the client's weighted round-robin allocates proportional traffic. The `deploy.resources.limits.cpus` value is reported by each node via `GET /_nodes/os` as `allocated_processors`, which the client uses to compute connection weights.

```
# Balanced weights [1,1,2]
make cluster.heterogeneous.cpu.1    # node1=2, node2=2, node3=4 CPUs

# Skewed weights [1,2,4]
make cluster.heterogeneous.cpu.2    # node1=1, node2=2, node3=4 CPUs
```

#### Node Roles

Assign different roles to each node so the client's role-based routing policy can direct requests to the correct nodes (e.g., bulk requests to ingest-capable nodes, search requests to data nodes):

```
make cluster.heterogeneous.roles    # node1=cluster_manager+ingest, node2=data+ingest, node3=data
```

#### Combining Overrides

CPU and role overrides are independent files and can be combined. Set both, then rebuild:

```
make cluster.heterogeneous.cpu.1 cluster.heterogeneous.roles
make cluster.stop cluster.clean cluster.build cluster.start
```

Verify the resulting configuration:

```
curl -sk 'https://admin:myStrongPassword123%21@localhost:9200/_nodes/http,os?pretty' \
  | jq '.nodes[] | {name, roles: .roles, processors: .os.allocated_processors}'
```

#### Resetting to Defaults

Remove all override files and return to the default homogeneous 3-node cluster:

```
make cluster.homogeneous
make cluster.stop cluster.clean cluster.build cluster.start
```

### Testing Specific OpenSearch Versions

The cluster supports any published OpenSearch Docker image version. Always clean before switching versions to avoid stale data or cached images:

```
make cluster.stop
OPENSEARCH_VERSION=3.6.0 make cluster.clean cluster.build cluster.start
make test-integ
```

Set `SECURE_INTEGRATION=false` to disable TLS and basic auth:

```
SECURE_INTEGRATION=false OPENSEARCH_VERSION=3.6.0 make cluster.clean cluster.build cluster.start
```

### Cluster Status and Troubleshooting

Use `make cluster.status` to display cluster health, node info, Docker container state, and index/shard details. It auto-detects whether the cluster is running in secure or insecure mode.

```
make cluster.status
```

## Network Latency Simulation

The `cluster.latency.*` targets inject artificial network delay into running Docker containers using Linux `tc` (traffic control). This lets you test the client's RTT-bucketed connection scoring and congestion-aware routing under realistic network topologies without leaving your development machine.

All latency targets use `tc qdisc` with `netem` delay and require the cluster to be running. They operate on the Docker container network interfaces directly.

### Latency Profiles

Four predefined profiles cover common deployment topologies:

```
# Asymmetric — simulates 3 availability zones at different distances
make cluster.latency.asymmetric    # node1=0ms, node2=50ms ±5ms, node3=150ms ±15ms

# Symmetric — single data center, all nodes equidistant
make cluster.latency.symmetric     # all nodes 1ms ±100us

# Bimodal — 2 local nodes + 1 remote node
make cluster.latency.bimodal       # node1=1ms, node2=1ms, node3=20ms

# Graduated — wide spread across 3 tiers
make cluster.latency.graduated     # node1=1ms, node2=10ms, node3=100ms
```

### Inspecting and Clearing Latency

```
# Show current tc qdisc rules on each node
make cluster.latency.show

# Remove all artificial latency
make cluster.latency.clear
```

## Demo

The `cmd/demo` binary exercises the client with affinity routing enabled, printing live per-node request distribution and latency statistics. It connects to a running cluster and generates configurable workloads.

```
make demo.build     # Build bin/demo
make demo.run       # Build and run with default affinity-visible settings
```

`demo.run` connects to all three local nodes with request routing, node discovery, and periodic stats output. The binary accepts flags for concurrency, workload type, fanout, and report output — run `bin/demo -help` for the full list.

## Verification Matrix

The `verify.*` targets automate end-to-end validation of the client's affinity routing behavior. Each target applies a latency profile, runs a workload through the demo binary, and writes a JSON report to `reports/verify/`.

### Individual Verifications

Run a single latency profile with the default search workload:

```
make verify.symmetric     # Equal distribution under symmetric latency
make verify.asymmetric    # Tier equalization under asymmetric latency (0/50/150ms)
make verify.bimodal       # 2-local + 1-remote equalization (1/1/20ms)
make verify.graduated     # Wide tier-spread equalization (1/10/100ms)
```

### Workload and Fanout Comparisons

```
# Compare fanout=1 vs fanout=2 vs fanout=3 under bimodal latency
make verify.fanout

# Compare search, read-write, bulk-write, and write-only across all latency profiles
make verify.workloads

# Verify CPU-proportional distribution with heterogeneous CPUs [1,2,4]
make verify.hetero.cpu
```

### Full Matrix

Run every combination of latency profile, workload, fanout, and hardware configuration:

```
make verify.matrix
```

Reports are written to `reports/verify/`. The full matrix takes several minutes to complete.

### Reviewing Results

```
# Print a tabular summary of all reports (name, total, RPS, P50, P99, connections)
make verify.summary

# Validate that distribution expectations are met
make verify.check
```

## Inspecting CI Failures

When a GitHub Actions check fails, searching the raw logs for keywords like "fail" or "error" produces many false positives from test names that legitimately contain those words. The `gh.fail.*` Makefile targets use the `gh` CLI to fetch only the failed steps and extract Go's structured failure markers (`--- FAIL:`, `FAIL\t`, `panic:`), giving clean output with no noise.

All targets auto-detect the most recent failed run on the current branch. Requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated.

### Quick Summary

Show which jobs failed and a deduplicated list of failed test names:

```
make gh.fail.summary
```

Example output:

```
Run 22722089811 — failed jobs:
  ✗ integ-test-compat (true, 2.19.5)
  ✗ integ-test-compat (false, 3.6.0)

Failed tests:
  --- FAIL: TestShardExactRouting_FullPipeline_Integration (2.56s)
      --- FAIL: TestShardExactRouting_FullPipeline_Integration/pipeline/routing=0 (0.01s)
      --- FAIL: TestShardExactRouting_FullPipeline_Integration/pipeline/routing=order-12345 (0.01s)
```

### Detailed Failure Output

Show all `--- FAIL:`, `FAIL`, and `panic:` lines with the job/step/timestamp prefix stripped:

```
make gh.fail
```

### Failure Context

Show 5 lines of context before each `--- FAIL:` or `panic:` line. This surfaces the assertion error messages (e.g., `Error:`, `Messages:`, `Error Trace:`) that explain why the test failed:

```
make gh.fail.context
```

### Full Logs

Dump the complete log output from all failed steps (raw `gh run view --log-failed`):

```
make gh.fail.full
```

### Inspecting a Specific Run

All targets accept a `GH_RUN_ID` override to inspect any run, not just the most recent failure:

```
make gh.fail.summary GH_RUN_ID=22705106053
```

To find run IDs, list recent checks:

```
make gh.checks            # All checks on the current branch
make gh.checks.failed     # Only failed checks
```

## Code Generation

The `cmd/osgen` tool generates typed path builder structs (`internal/path/`) and API consumer files (`v5preview/opensearchapi/`, `v5preview/opensearchapi/plugins/`) from the published [OpenSearch API specification](https://github.com/opensearch-project/opensearch-api-specification). It reads `x-operation-group`, `x-version-added`, `x-version-deprecated`, `x-version-removed`, and `x-error-responses` extensions from the spec to produce version-aware Go source.

The `v5preview/opensearchapi/` package is the v5-track API surface and coexists with the hand-written `opensearchapi/` package during the v4 -> v5 transition. New code should target `v5preview/opensearchapi/`; see `v5preview/opensearchapi/README.md` for usage and `UPGRADING.md` for migration guidance.

> **PRs that edit `*_gen.go` will be rejected.** These files are generated. To change them, send a PR against `cmd/osgen` (the generator) or against the [OpenSearch API specification](https://github.com/opensearch-project/opensearch-api-specification) (the input).

> **Spec-extension status.** Some `x-*` extensions osgen reads (notably `x-error-responses`) are still in flight upstream. Until they merge, the build pulls from the local `opensearch-openapi.yaml` checkout in this repo rather than the published spec. Once upstream catches up, the local file goes away.

### Partial-failure error generation

The `x-error-responses` extension on a spec operation declares the categories of partial failure that operation can produce. For each category named on an operation, osgen emits:

- A typed Go error (e.g. `*PartialBulkError`, `*PartialSearchError`, `*ShardFailureError`, `*MultiSearchItemError`) decoded from the response body when that category fires.
- A bit on `errmask.ErrorMask` (PascalCase, e.g. `errmask.BulkItems`) plus the corresponding env-var token (`bulk_items`) so callers can suppress or surface it via `Config.Errors` or `OPENSEARCH_GO_ERROR_MASK`.
- A per-Resp helper method on the operation's typed response (e.g. `BulkResp.BulkItemFailures()`, `SearchResp.SearchShardFailures()`). These exist as engine machinery for the dispatch and are not the recommended call-site pattern; user docs point callers at a `for`/`switch` over `opensearchapi.Errors(err)` instead.
- A `PartialFailures(mask)` aggregator on the same Resp.

Operations that declare two or more categories also get a per-op error container (e.g. `*MSearchErrors`) implementing `Unwrap() []error`, used when more than one category fires on a single response.

The user-facing partial-failure model and best-practices guidance live in [`v5preview/opensearchapi/README.md`](v5preview/opensearchapi/README.md#partial-failure-errors) and [`guides/error_handling.md`](guides/error_handling.md). They deliberately omit `x-error-responses` terminology because callers don't need to read the spec to use the resulting errors. The spec-driven mechanics are documented here and in [`cmd/osgen/README.md`](cmd/osgen/README.md).

To regenerate (downloads the spec automatically if not cached):

```
make gen
```

To force a fresh spec download, remove the cached file first:

```
rm -f opensearch-openapi.yaml
make gen
```

### Version-Scoped Generation

By default, `make gen` includes all operations and fields across every OpenSearch version. Three Makefile variables scope the generated output to a version window:

| Variable                | Default  | Effect                                                   |
| ----------------------- | -------- | -------------------------------------------------------- |
| `GEN_MIN_VERSION`       | `epoch`  | Exclude operations removed before this floor             |
| `GEN_MAX_VERSION`       | `latest` | Exclude operations added after this ceiling              |
| `GEN_REMOVE_DEPRECATED` | `epoch`  | Treat operations deprecated at or before this as removed |

Override on the command line:

```
# Generate a client targeting only OpenSearch 2.x+, dropping deprecated APIs
make gen GEN_MIN_VERSION=2.0 GEN_REMOVE_DEPRECATED=2.0

# Generate a client for a specific version range
make gen GEN_MIN_VERSION=2.0 GEN_MAX_VERSION=2.99

# Strict less-than ceiling (excludes 3.0.0 itself)
make gen GEN_MAX_VERSION="<3.0.0"
```

Version flags accept an optional operator prefix (`>=`, `>`, `<=`, `<`). When omitted, `--min-version` defaults to `>=` and `--max-version` defaults to `<=`. The magic values `epoch` (no floor) and `latest` (no ceiling) mean "include everything".

When items are excluded by version filtering, breadcrumb comments are left in the generated code explaining the reason (e.g., `// cat.masterPath: deprecated in OpenSearch 2.0.0 (treated as removed).`). Breadcrumb visibility is configurable per category via `-version-breadcrumb-*` flags.

See [`cmd/osgen/README.md`](cmd/osgen/README.md) for the full flag reference and subcommand details.

## Lint

To keep all the code in a certain uniform format, it was decided to use some writing rules. If you wrote something wrong, it's okay, you can simply run the script to check the necessary files, and optionally format the content. But keep in mind that all these checks are repeated on the pipeline, so it's better to check locally.

### Markdown lint

To check the markdown files, run the following command:

```
make lint.markdown
```

### Go lint

To check all go files, run the following command:

```
make linters
```

## Coverage

To get the repository test coverage, run the following command:

For the results to be display in your terminal:

```
make coverage
```

For the results to be display in your browser:

```
make coverage-html
```

## Use an Editor

### GoLand

You can import the OpenSearch project into GoLand as follows:

1. Select **File | Open**
2. In the subsequent dialog navigate to the ~/go/src/opensearch-go and click **Open**

After you have opened your project, you need to specify the location of the Go SDK. You can either specify a local path to the SDK or download it. To set the Go SDK, navigate to **Go | GOROOT** and set accordingly.

### Vim

To improve your vim experience with Go, you might want to check out [fatih/vim-go](https://github.com/fatih/vim-go). For example it correctly formats the file and validates it on save.
