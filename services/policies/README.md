# Policies

The policies service provides a new gRPC API which can be used to check whether a requested operation is allowed or not. To do so, Open Policy Agent (OPA) is used to define the set of rules of what is permitted and what is not.

Policies are written in the [rego query language](https://www.openpolicyagent.org/docs/latest/policy-language/). The location of the rego files can be configured via yaml, a configuration via environment variables is not possible.

## General Information

The policies service consists of the following modules:

*   Proxy authorization (middleware)
*   Event authorization (async post-processing)
*   gRPC API (can be used by other services)

To configure the policies service, three environment variables need to be defined:

*   `POLICIES_ENGINE_TIMEOUT`
*   `POLICIES_POSTPROCESSING_QUERY`
*   `PROXY_POLICIES_QUERY`

Note that each query setting defines the [Complete Rules](https://www.openpolicyagent.org/docs/latest/#complete-rules) variable defined in the rego rule set the corresponding step uses for the evaluation. If the variable is mistyped or not found, the evaluation defaults to deny. Individual query definitions can be defined for each module.

To activate the policies service for a module, it must be started with a yaml configuration that points to one or more rego files. Note that if the service is scaled horizontally, each instance should have access to the same rego files to avoid unpredictable results. If a file path has been configured but the file is not present or accessible, the evaluation defaults to deny.

When using async post-processing which is done via the postprocessing service, the value `policies` must be added to the `POSTPROCESSING_STEPS` configuration in postprocessing service in the order where the evaluation should take place.

variable defined in the Rego rule set the corresponding step uses for the evaluation. If the variable is mistyped or not found, the evaluation defaults to deny. Individual query definitions can be defined for each module.

To activate the policies service for a module, it must be started with a yaml configuration that points to at least one Rego file that contains the complete rule variable to be queried. Note that if the service is scaled horizontally, each instance should have access to the same Rego files to avoid unpredictable results. If a file path has been configured but the file it is not present or accessible, the evaluation defaults to deny.

When using async post-processing via the postprocessing service, the value `policies` must be added to the `POSTPROCESSING_STEPS` configuration in the order in which the evaluation should take place. Example: First check if a file contains questionable content via policies. If it looks okay, continue to check for viruses.

For configuration examples, the [Example Policies](#example-policies) from below are used.

## Modules

### gRPC API

The gRPC API can be used by any other internal service. It can also be used for example by third parties to find out if an action is allowed or not. This layer is already used by the proxy middleware. There is no configuration necessary, because the query setting (complete rule variable) must be part of the request.

### Proxy Middleware

The proxy service already includes a middleware which uses the internal [gRPC API](#grpc-api) to evaluate the policies. Since the proxy is in heavy use and every HTTP request is processed here, only simple and quick decisions should be evaluated. More complex queries such as file content evaluation are _strongly_ discouraged.

If the evaluation in the proxy results in a "denied" outcome, the response will return a `403 Permission Denied` with the following response body

```json
{
    "error":
    {
        "code": "deniedByPolicy",
        "message": "Operation denied due to security policies",
        "innererror":
        {
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

This layer is event-based and part of the postprocessing service. Since processing at this point is asynchronous, the operations can also take longer and be more expensive, like evaluating the contents of a file.

## Defining Policies to Evaluate

Each module can have as many policy files as needed for evaluation. Files can also include other files if necessary. To use policies, they have to be saved to a location that is accessible to the policies service. As a good starting point, take the config directory and use a subdirectory collecting all the `.rego` files, though any other directory can be defined. The config directory is already accessible by all services and usually is included in a xref:maintenance/b-r/backup.adoc[backup] plan.

If this is done, it's required to configure the policies service to use these files:

NOTE: It is important that *all* necessary files are added to the list of files the policies service uses.

```yaml
policies:
  engine:
    policies:
      - your_path_to_policies/proxy.rego
      - your_path_to_policies/postprocessing.rego
      - your_path_to_policies/util.rego
```

Once the references to policy files are configured correctly, the `_QUERY`  configuration needs to be defined for the proxy middleware and for the events service.

## Setting the Query Configuration

To define a value for the query evaluation, the following scheme is necessary:

`data.<package-name>.<complete-rule-variable-name>`

* The keyword `data` is mandatory and must be present.
* The `package-name` is defined in one .rego file like `package postprocessing`. It is not related to the filename. For more details, see the [packages](https://www.openpolicyagent.org/docs/latest/policy-language/#packages) documentation.
* The `complete-rule-variable-name` is the variable providing the result of the evaluation.
* Exact one of the defined files, which is responsible for returning the evaluation result, must contain the combination of `<package-name>` and `<complete-rule-variable-name>`.

### Proxy

Note that this setting has to be part of the proxy configuration.

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

As soon as that query is configured, the postprocessing service must be informed to use the policies step by setting the environment variable:

```shell
export POSTPROCESSING_STEPS=policies
```

Note that additional steps can be configured and their position in the list defines the order of processing. For details see the postprocessing service documentation.

## Rego Key Match

To identify available keys for OPA, you need to look at [engine.go](https://github.com/opencloud-eu/opencloud/blob/main/services/policies/pkg/engine/engine.go) and the [policies.swagger.json](https://github.com/opencloud-eu/opencloud/blob/master/protogen/gen/opencloud/services/policies/v0/policies.swagger.json) file. Note that which keys are available depends on from which module it is used.

## Extend Mimetype File Extension Mapping

In the extended set of the rego query language, it is possible to get a list of associated file extensions based on a mimetype, for example `opencloud.mimetype.extensions("application/pdf")`.

The list of mappings is restricted by default and is provided by the host system OpenCloud is installed on.

In order to extend this list, OpenCloud must be provided with the path to a custom `mime.types` file that maps mimetypes to extensions.
The location for the file must be accessible by all instances of the policy service. As a rule of thumb, use the directory where the OpenCloud configuration files are stored.
Note that existing mappings from the host are extended by the definitions from the mime types file, but not replaced.

The path to that file can be provided via a yaml configuration or an environment variable. Note to replace the `OC_CONFIG_DIR` string by an existing path.

```shell
export POLICIES_ENGINE_MIMES=OC_CONFIG_DIR/mime.types
```

```yaml
policies:
  engine:
    mimes: OC_CONFIG_DIR/mime.types
```

A good example of how such a file should be formatted can be found in the [Apache svn repository](https://svn.apache.org/repos/asf/httpd/httpd/trunk/docs/conf/mime.types).

## Example Policies

The policies service contains a set of preconfigured example policies. See the [devtools policie](https://github.com/opencloud-eu/opencloud/tree/main/devtools/deployments/service_policies/policies/) directory for details. The contained policies disallow OpenCloud to create certain file types, both via the proxy middleware and the events service via postprocessing.
