# Policies

The policies service provides a gRPC and an Events API which can be used to check whether a requested operation is allowed or not. To do so, [Open Policy Agent (OPA)](https://www.openpolicyagent.org/) is used to define the set of rules of what is permitted and what is not.

Policies are written in the [rego query language](https://www.openpolicyagent.org/docs/latest/policy-language/). The location of the rego files can be configured via yaml or via the environment variable `POLICIES_ENGINE_FILES`.

## General Information

The policies service consists of the following modules that are able to process policy authorization requests for different API use-cases:

* Proxy authorization (middleware)
* Event authorization (async post-processing)
* gRPC API (can be used by other services)

To configure the policies service, three environment variables need to be defined:

* `POLICIES_ENGINE_TIMEOUT`
* `POLICIES_POSTPROCESSING_QUERY`
* `PROXY_POLICIES_QUERY`

Note that each query setting defines the [Complete Rules](https://www.openpolicyagent.org/docs/latest/#complete-rules)
variable defined in the rego rule set the corresponding step uses for the evaluation. If the variable is mistyped or not
found, the evaluation defaults to deny. Individual query definitions can be defined for each module.

To activate it for a module, the `policies` service must be started with a yaml configuration or by setting the environment variable `POLICIES_ENGINE_FILES` that may contain a comma-separated list of file paths, either of which points to at least one rego file that contains the complete rule variable to be queried.

The rego files are read once when the service starts. A changed rule set takes effect after a restart, and a configured
file that is missing or does not parse keeps the service from starting, as it will abort with an error.

Note that if the service is scaled horizontally, each instance should have access to the same rego files to avoid unpredictable results. If a file path has been configured but the file is not present or accessible, the evaluation defaults to deny.

If a directory is specified in the list of paths, `.rego` files will be looked up and loaded recursively in that directory.

When using async post-processing, which is done via the `postprocessing` service, the value `policies` must be added to the `POSTPROCESSING_STEPS` configuration in the `postprocessing` service in the order in which the policies evaluation should take place.

Example: First check if a file contains questionable content via policies. If it looks okay, continue to check for viruses.

Configuration examples may be found in the [Example Policies](#example-policies) section below.

## Modules

### gRPC API

The gRPC API can be used by any other internal service. It can also be used for example by third parties to find out if an action is allowed or not. This layer is already used by the `proxy` middleware.

No configuration is necessary, because the query setting (complete rule variable) that should be evaluated is part of the request that is sent to the gRPC API.

Note that the gRPC API handler may be disabled via the configuration setting `disabled` in the `grpc` section, or via the environment variable `POLICIES_GRPC_DISABLED`, which ought to be set to `true` to be disabled, and defaults to `false`.

The purpose of disabling the gRPC API is to set up pools of `policies` service processes (or containers, or pods) that specialize in serving only the async Events API.

### Proxy Middleware

The `proxy` service already includes a middleware which uses the internal [gRPC API](#grpc-api) to evaluate the policies. Since the proxy is in heavy use and every inbound HTTP request is processed by the `policies` service, only simple and quick decisions should be evaluated. More complex queries such as file content evaluation are _strongly_ discouraged.

The `proxy` middleware only denies based on the outcome of the policy, it makes no decision of its own. Where the file name is not part of the request path, for example when a single shared resource is uploaded by its identifier, the middleware stats the resource to obtain it. If that stat fails, `input.resource.name` is an empty string and the policy may still decide what that means and how to proceed.

Prefer policies that state which files are allowed over policies that list what is forbidden, a rule matching on a specific extension does not match an empty name:

```rego
granted = false if {
    input.resource.name == ""
}
```

If the outcome of the evaluation in the `proxy` results in access being denied, the response will return a `403 Permission Denied` with the following response body:

```json
{
  "error": {
    "code": "deniedByPolicy",
    "message": "Operation denied due to security policies",
    "innererror": {
      "date": "2023-09-19T13:22:20Z",
      "filename": "File",
      "method": "POST",
      "path": "/dav/spaces/some-space-id/Folder/",
      "request-id": "9CFCE925-F9D9-4F26-AB3B-2C1C40A9CD0C"
    }
  }
}
```

### Event Service (Postprocessing)

This layer is event-based and part of the `postprocessing` service. Since processing at this point is asynchronous, the operations can also take longer and be more expensive, like evaluating the contents of a file.

For processing asynchronous requests that come in through the Events API, one must set the `POLICIES_POSTPROCESSING_QUERY` environment variable, or the `query` string in the `postprocessing` section in the YAML configuration.

Note that the Events API handler may be disabled via the configuration setting `disabled` in the `events` section, or via the environment variable `POLICIES_EVENTS_DISABLED`, which ought to be set to `true` to be disabled, and defaults to `false`.

The purpose of disabling the Events API is to set up pools of `policies` service processes (or containers, or pods) that specialize in serving only the gRPC API.

## Defining Policies to Evaluate

Each module can have as many policy files as needed for evaluation. Files can also include other files if necessary.

To use policies, they have to be saved to a location that is accessible to the policies service. As a good starting point, take the config directory and use a subdirectory collecting all the `.rego` files, though any other directory can be defined. The config directory is already accessible by all services and usually is included in a backup plan.

The list of files or directories specified there (or as comma-separated strings in the environment variable `POLICIES_ENGINE_FILES`) may make use of two magic prefixes that are resolved at runtime, for convenience:

* in paths starting with `config:`, that string is replaced by the OpenCloud configuration directory
* in paths starting with `data:`, that string is replaced by the OpenCloud base directory

If this is done, it's required to configure the policies service to use these files:

NOTE: It is important that _all_ necessary files are added to the list of files the policies service uses.

```yaml
policies:
  engine:
    policies:
      - your_path_to_policies/proxy.rego
      - your_path_to_policies/postprocessing.rego
      - your_path_to_policies/util.rego
```

Alternatively, using the environment variable:

```bash
POLICIES_ENGINE_FILES="data:policies/proxy.rego,data:policies/postprocessing.rego,data:policies/util.rego"
```

Although it would be more convenient to make use of the directory recuring support by just specifying the directory that contains all the `.rego` files instead:

```shell
export POLICIES_ENGINE_FILES="data:policies"
```

Once the references to policy files are configured correctly, the `_QUERY` configuration needs to be defined for the `proxy` middleware and for the events service.

## Setting the Query Configuration

To define a value for the query evaluation, the following scheme must be used:

`data.<package-name>.<complete-rule-variable-name>`

* The keyword `data` is mandatory and must be present.
* The `package-name` is defined in one `.rego` file like `package postprocessing`. It is not related to the filename. For more details, see the [packages](https://www.openpolicyagent.org/docs/latest/policy-language/#packages) documentation.
* The `complete-rule-variable-name` is the variable providing the result of the evaluation.
* Exact one of the defined files, which is responsible for returning the evaluation result, must contain the combination
  of `<package-name>` and `<complete-rule-variable-name>`.

### Proxy

Note that this setting has to be part of the configuration of the `proxy` service:

```yaml
proxy:
  policies_middleware:
    query: data.proxy.granted
```

The same can be achieved by setting the following environment variable:

```shell
export PROXY_POLICIES_QUERY=data.proxy.granted
```

### Postprocessing

```yaml
policies:
  postprocessing:
    query: data.postprocessing.granted
```

The same can be achieved by setting the following environment variable:

```shell
export POLICIES_POSTPROCESSING_QUERY=data.postprocessing.granted
```

As soon as that query is configured, the `postprocessing` service must be informed to use the policies step by setting the environment variable:

```shell
export POSTPROCESSING_STEPS=policies
```

Note that additional steps can be configured and their position in the list defines the order of processing. For details see the `postprocessing` service documentation.

## Rego Key Match

To identify available keys for OPA, you need to look at [`engine.go`](https://github.com/opencloud-eu/opencloud/blob/main/services/policies/pkg/engine/engine.go) and the [`policies.swagger.json`](https://github.com/opencloud-eu/opencloud/blob/master/protogen/gen/opencloud/services/policies/v0/policies.swagger.json) file.

Note that which keys are available depends on the module it is used in.

## Rego Extensions

Besides the standard rego built-in functions, the following functions are added on top:

| Function | Result | Description |
| -------- | ------ | ----------- |
| `opencloud.mimetype.extensions("application/pdf")` | `[".pdf"]` | Lists the file extensions associated with a mimetype. See [Extend Mimetype File Extension Mapping](#extend-mimetype-file-extension-mapping). |
| `opencloud.mimetype.detect(bytes)` | `"application/pdf"` | Detects a mimetype from content. The list of known mimetypes is limited. |
| `opencloud.resource.download(input.resource.url)` | bytes | Downloads a resource. Available in the event service (postprocessing) where `input.resource.url` is set. |

Rego has no byte type, so `opencloud.resource.download` hands the content to the policy as base64.
`opencloud.mimetype.detect` takes that value as it is, a policy working on the content itself has to run it through `base64.decode` first.

Note that `opencloud.resource.download` performs an HTTP request and holds the whole resource in memory. Use it in postprocessing policies only, not in policies evaluated by the proxy middleware.

## Extend Mimetype File Extension Mapping

In the extended set of the rego query language, it is possible to get a list of associated file extensions based on a
mimetype, for example `opencloud.mimetype.extensions("application/pdf")`.

The list of mappings is restricted by default and is provided by the host system OpenCloud is installed on.

In order to extend this list, OpenCloud must be provided with the path to a custom `mime.types` file that maps mimetypes
to extensions. The location for the file must be accessible by all instances of the policy service. As a rule of thumb,
use the directory where the OpenCloud configuration files are stored. Note that existing mappings from the host are
extended by the definitions from the mime types file, but not replaced.

The path to that file can be provided via a yaml configuration or an environment variable. Note that the `config:` file path prefix is replaced by `$OC_CONFIG_DIR`.

```shell
export POLICIES_ENGINE_MIMES=config:mime.types
```

```yaml
policies:
  engine:
    mimes: config:mime.types
```

A good example of how such a file should be formatted can be found in the [Apache SVN repository](https://svn.apache.org/repos/asf/httpd/httpd/trunk/docs/conf/mime.types).

## Example Policies

The `policies` service contains a set of preconfigured example policies. See the [devtools policies](https://github.com/opencloud-eu/opencloud/tree/main/devtools/deployments/service_policies/policies/) directory for details. The contained policies disallow OpenCloud from creating certain file types, both via the `proxy` middleware and the events service via `postprocessing`.

## Metrics

The `policies` service provides the following metrics:

| Name | Description |
| ---- | ----------- |
| `opencloud_policies_build_info{version=...}` | Contains a label `version` that is set to the current version of the service, and always has a value of `1` |
| `opencloud_policies_events_enabled` | Is set to `1` if the Events API handler is enabled, or `0` if not |
| `opencloud_policies_grpc_enabled` | Is set to `1` if the gRPC API handler is enabled, or `0` if not |
| `opencloud_policies_policies_processed{result=...,origin=...}` | A counter with the number of policy rule evaluations. Has a label `result` that is set to `allowed` or `not-allowed` depending on the outcome, as well as a label `origin` that is set to `grpc` or `events` depending on how the request came in. |
| `opencloud_policies_policies_failures{origin=...}` | A counter with the number of policy rule evaluation errors. Has a label `origin` that is set to `grpc` or `events` depending on how the request came in. |
| `opencloud_policies_policies_requests{origin=...}` | A counter with the number of received requests for evaluating policies. Has a label `origin` that is set to `grpc` or `events` depending on how the request came in. |

## Testing

As a developer, to test the `policies` service, one approach may be to set the following environment variables:

```yaml
OC_ADD_RUN_SERVICES: "policies"
POLICIES_EVENTS_DISABLED: "false"
POLICIES_GRPC_DISABLED: "false"
POLICIES_ENGINE_TIMEOUT: "30s"
POLICIES_ENGINE_FILES: "data:policies"
PROXY_POLICIES_QUERY: "data.proxy.granted"
```

Create a directory `policies` under your `OC_CONFIG_DIR`, typically `~/.opencloud/policies/`, and then store the following file content in a file `proxy.rego` under that directory:

```rego
package proxy

import future.keywords.if

default granted := true

granted = false if {
  input.user.username == "alan"
}
```

To inspect metrics, use the following request:

```shell
curl -sSLf http://localhost:9129/metrics
```

To also inspect the metrics using a Prometheus container (and possibly even e.g. Grafana), set the following environment variable:

```yaml
POLICIES_DEBUG_ADDR: "0.0.0.0:9129"
```

Create a `prometheus.yml` file with the following content:

```yaml
scrape_configs:
  - job_name: 'opencloud'
    scrape_interval: 5s
    static_configs:
    - targets:
      - 'host.docker.internal:9129'
      labels:
        service: 'policies'
```

```shell
docker run --rm -p 9090:9090 \
  -v "$PWD/prometheus.yml:/etc/prometheus/prometheus.yml" \
  --add-host host.docker.internal=host-gateway \
  prom/prometheus:latest
```
