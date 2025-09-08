# Proxy

The proxy service is an API-Gateway for the OpenCloud microservices. Every HTTP request goes through this service. Authentication, logging and other preprocessing of requests also happens here. Mechanisms like request rate limiting or intrusion prevention are **not** included in the proxy service and must be setup in front like with an external reverse proxy.

The proxy service is the only service communicating to the outside and needs therefore usual protections against DDOS, Slow Loris or other attack vectors. All other services are not exposed to the outside, but also need protective measures when it comes to distributed setups like when using container orchestration over various physical servers.

## Authentication

The following request authentication schemes are implemented:

-   Basic Auth (Only use in development, **never in production** setups!)
-   OpenID Connect
-   Signed URL
-   Public Share Token

## Configuring Routes

The proxy handles routing to all endpoints that OpenCloud offers. The currently availabe default routes can be found [in the code](https://github.com/opencloud-eu/opencloud/blob/main/services/proxy/pkg/config/defaults/defaultconfig.go). Changing or adding routes can be necessary when writing own OpenCloud extensions.

Due to the complexity when defining routes, these can only be defined in the yaml file but not via environment variables.

For _overwriting_ default routes, use the following yaml example:

```yaml
policies:
  - name: opencloud
    routes:
      - endpoint: /
        service: eu.opencloud.web.web
      - endpoint: /dav/
        service: eu.opencloud.web.ocdav
```

For adding _additional_ routes to the default routes use:

```yaml
additional_policies:
  - name: opencloud
    routes:
      - endpoint: /custom/endpoint
        service: eu.opencloud.custom.custom
```

A route has the following configurable parameters:

```yaml
endpoint: ""       # the url that should be routed
service: ""        # the service the url should be routed to
unprotected: false # with false (default), calling the endpoint requires authorization.
                   # with true, anyone can call the endpoint without authorisation.
```

## Automatic User and Group Provisioning

When using an external OpenID Connect IDP, the proxy can be configured to automatically provision
users upon their first login.

### Prequisites

A number of prerequisites must be met for automatic user provisioning to work:

* OpenCloud must be configured to use an external OpenID Connect IDP
* The `graph` service must be configured to allow updating users and groups
  (`GRAPH_LDAP_SERVER_WRITE_ENABLED`).
* One of the claim values returned by the IDP as part of the userinfo response
  or the access token must be unique and stable for the user. I.e. the value
  must not change for the whole lifetime of the user. This claim is configured
  via the `PROXY_USER_OIDC_CLAIM` environment variable (see below). A natural
  choice would e.g. be the `sub` claim which is guaranteed to be unique and
  stable per IDP. If a claim like `email` or `preferred_username` is used, you
  have to ensure that the user's email address or username never changes.

### Configuration

To enable automatic user provisioning, the following environment variables must
be set for the proxy service:

* `PROXY_AUTOPROVISION_ACCOUNTS`\
Set to `true` to enable automatic user provisioning.
* `PROXY_AUTOPROVISION_CLAIM_USERNAME`\
The name of an OIDC claim whose value should be used as the username for the
autoprovsioned user in OpenCloud. Defaults to `preferred_username`.
Can also be set to e.g. `sub` to guarantee a unique and stable username.
* `PROXY_AUTOPROVISION_CLAIM_EMAIL`\
The name of an OIDC claim whose value should be used for the `mail` attribute
of the autoprovisioned user in OpenCloud. Defaults to `email`.
* `PROXY_AUTOPROVISION_CLAIM_DISPLAYNAME`\
The name of an OIDC claim whose value should be used for the `displayname`
attribute of the autoprovisioned user in OpenCloud. Defaults to `name`.
* `PROXY_AUTOPROVISION_CLAIM_GROUPS`\
The name of an OIDC claim whose value should be used to maintain a user's group
membership. The claim value should contain a list of group names the user should
be a member of. Defaults to `groups`.
* `PROXY_USER_OIDC_CLAIM`\
When resolving and authenticated OIDC user, the value of this claims is used to
lookup the user in the users service. For auto provisioning setups this usually is the
same claims as set via `PROXY_AUTOPROVISION_CLAIM_USERNAME`.
* `PROXY_USER_CS3_CLAIM`\
This is the name of the user attribute in OpenCloud that is used to lookup the user by the
value of the `PROXY_USER_OIDC_CLAIM`. For auto provisioning setups this usually
needs to be set to `username`.

### How it Works

When a user logs into OpenCloud for the first time, the proxy
checks if that user already exists. This is done by querying the `users` service for users,
where the attribute set in `PROXY_USER_CS3_CLAIM` matches the value of the OIDC
claim configured in `PROXY_USER_OIDC_CLAIM`.

If the users does not exist, the proxy will create a new user via the `graph`
service using the claim values configured in
`PROXY_AUTOPROVISION_CLAIM_USERNAME`, `PROXY_AUTOPROVISION_CLAIM_EMAIL` and
`PROXY_AUTOPROVISION_CLAIM_DISPLAYNAME`.

If the user does already exist, the proxy checks if the displayname has changed
and updates that accordingly via `graph` service.

Unless the claim configured via `PROXY_AUTOPROVISION_CLAIM_EMAIL` is the same
as the one set via `PROXY_USER_OIDC_CLAIM` the proxy will also check if the
email address has changed and update that as well.

Next, the proxy will check if the user is a member of the groups configured in
`PROXY_AUTOPROVISION_CLAIM_GROUPS`. It will add the user to the groups listed
via the OIDC claim that holds the groups defined in the envvar and removes it from
all other groups that he is currently a member of.
Groups that do not exist in the external IDP yet will be created. Note: This can be a
somewhat costly operation, especially if the user is a member of a large number of
groups. If the group memberships of a user are changed in the IDP after the
first login, it can take up to 5 minutes until the changes are reflected in OpenCloud.

### Claim Updates

OpenID Connect (OIDC) scopes are used by an application during authentication to authorize access to a user's detail, like name, email or picture information. A scope can also contain among other things groups, roles, and permissions data. Each scope returns a set of attributes, which are called claims. The scopes an application requests, depends on which  attributes the application needs. Once the user authorizes the requested scopes, the claims are returned in a token.

These issued JWT tokens are immutable and integrity-protected. Which means, any change in the source requires issuing a new token containing updated claims. On the other hand side, there is no active synchronisation process between the identity provider (IDP) who issues the token and OpenCloud. The earliest possible time that OpenCloud will notice changes is, when the current access token has expired and a new access token is issued by the IDP, or the user logs out and relogs in.

**NOTES**

* For resource optimisation, OpenCloud skips any checks and updates on groupmemberships, if the last update happened less than 5min ago.

* OpenCloud can't differentiate between a group being renamed in the IDP and users being reassigned to a different group.

* OpenCloud does not get aware when a group is being deleted in the IDP, a new claim will not hold any information from the deleted group. OpenCloud does not track a claim history to compare.

#### Impacts

For shares or space memberships based on groups, a renamed or deleted group will impact accessing the resource:

* There is no user notification about the inability accessing the resource.
* The user will only experience rejected access.
* This also applies for connected apps like the Desktop, iOS or Android app!

To give access for rejected users on a resource, one with rights to share must update the group information.

## Automatic Quota Assignments

It is possible to automatically assign a specific quota to new users depending on their role.
To do this, you need to configure a mapping between roles defined by their ID and the quota in bytes.
The assignment can only be done via a `yaml` configuration and not via environment variables.
See the following `proxy.yaml` config snippet for a configuration example.

```yaml
role_quotas:
    <role ID1>: <quota1>
    <role ID2>: <quota2>
```

## Automatic Role Assignments

When users login, they do automatically get a role assigned. The automatic role assignment can be
configured in different ways. The `PROXY_ROLE_ASSIGNMENT_DRIVER` environment variable (or the `driver`
setting in the `role_assignment` section of the configuration file select which mechanism to use for
the automatic role assignment.

When set to `default`, all users which do not have a role assigned at the time for the first login will
get the role 'user' assigned. (This is also the default behavior if `PROXY_ROLE_ASSIGNMENT_DRIVER`
is unset.

When `PROXY_ROLE_ASSIGNMENT_DRIVER` is set to `oidc` the role assignment for a user will happen
based on the values of an OpenID Connect Claim of that user. The name of the OpenID Connect Claim to
be used for the role assignment can be configured via the `PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM`
environment variable. It is also possible to define a mapping of claim values to role names defined
in OpenCloud via a `yaml` configuration. See the following `proxy.yaml` snippet for an example.

```yaml
role_assignment:
    driver: oidc
    oidc_role_mapper:
        role_claim: opencloudRoles
        role_mapping:
            - role_name: admin
              claim_value: myAdminRole
            - role_name: spaceadmin
              claim_value: mySpaceAdminRole
            - role_name: user
              claim_value: myUserRole
            - role_name: guest
              claim_value: myGuestRole
```

This would assign the role `admin` to users with the value `myAdminRole` in the claim `opencloudRoles`.
The role `user` to users with the values `myUserRole` in the claims `opencloudRoles` and so on.

Claim values that are not mapped to a specific OpenCloud role will be ignored.

Note: An OpenCloud user can only have a single role assigned. If the configured
`role_mapping` and a user's claim values result in multiple possible roles for a user, the order in
which the role mappings are defined in the configuration is important. The first role in the
`role_mappings` where the `claim_value` matches a value from the user's roles claim will be assigned
to the user. So if e.g. a user's `opencloudRoles` claim has the values `myUserRole` and
`mySpaceAdminRole` that user will get the OpenCloud role `spaceadmin` assigned (because `spaceadmin`
appears before `user` in the above sample configuration).

If a user's claim values don't match any of the configured role mappings an error will be logged and
the user will not be able to login.

The default `role_claim` (or `PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM`) is `roles`. The default `role_mapping` is:

```yaml
- role_name: admin
  claim_value: opencloudAdmin
- role_name: spaceadmin
  claim_value: opencloudSpaceAdmin
- role_name: user
  claim_value: opencloudUser
- role_name: guest
  claim_value: opencloudcloudGuest
```

## Recommendations for Production Deployments

In a production deployment, you want to have basic authentication (`PROXY_ENABLE_BASIC_AUTH`) disabled which is the default state. You also want to setup a firewall to only allow requests to the proxy service or the reverse proxy if you have one. Requests to the other services should be blocked by the firewall.

### Content Security Policy

For OpenCloud, external resources like an IDP (e.g. Keycloak) or when using web office documents or web apps, require defining a CSP. If not defined, the referenced services will not work.

To create a Content Security Policy (CSP), you need to create a yaml file containing the CSP definitions. To activate the settings, reference the file as value in the `PROXY_CSP_CONFIG_FILE_LOCATION` environment variable. For each change, a restart of the OpenCloud deployment or the proxy service is required.

A working example for a CSP can be found in a sub path of the `config` directory of the [opencloud-compose](https://github.com/opencloud-eu/opencloud-compose/tree/main/config) deployment example.

See the [Content Security Policy (CSP) Quick Reference Guide](https://content-security-policy.com) for a description of directives.

## Caching

The `proxy` service can use a configured store via `PROXY_OIDC_USERINFO_CACHE_STORE`. Possible stores are:
  -   `memory`: Basic in-memory store and the default.
  -   `redis-sentinel`: Stores data in a configured Redis Sentinel cluster.
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `noop`: Stores nothing. Useful for testing. Not recommended in production environments.

Other store types may work but are not supported currently.

Note: The service can only be scaled if not using `memory` store and the stores are configured identically over all instances!

Note that if you have used one of the deprecated stores, you should reconfigure to one of the supported ones as the deprecated stores will be removed in a later version.

Store specific notes:
  -   When using `redis-sentinel`, the Redis master to use is configured via e.g. `OC_CACHE_STORE_NODES` in the form of `<sentinel-host>:<sentinel-port>/<redis-master>` like `10.10.0.200:26379/mymaster`.
  -   When using `nats-js-kv` it is recommended to set `OC_CACHE_STORE_NODES` to the same value as `OC_EVENTS_ENDPOINT`. That way the cache uses the same nats instance as the event bus.
  -   When using the `nats-js-kv` store, it is possible to set `OC_CACHE_DISABLE_PERSISTENCE` to instruct nats to not persist cache data on disc.


## Presigned Urls

To authenticate presigned URLs the proxy service needs to read signing keys from a store that is populated by the ocs service. Possible stores are:
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `redis-sentinel`: Stores data in a configured Redis Sentinel cluster.
  -   `opencloudstoreservice`:  Stores data in the legacy OpenCloud store service. Requires setting `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` to `eu.opencloud.api.store`.

The `memory` store cannot be used as it does not share the memory from the ocs service signing key memory store, even in a single process.

Make sure to configure the same store in the ocs service.

Store specific notes:
  -   When using `redis-sentinel`, the Redis master to use is configured via e.g. `OC_CACHE_STORE_NODES` in the form of `<sentinel-host>:<sentinel-port>/<redis-master>` like `10.10.0.200:26379/mymaster`.
  -   When using `nats-js-kv` it is recommended to set `OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` to the same value as `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES`. That way the ocs uses the same nats instance as the proxy service.
  -   When using the `nats-js-kv` store, it is possible to set `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_DISABLE_PERSISTENCE` to instruct nats to not persist signing key data on disc.
  -   When using `opencloudstoreservice` the `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` must be set to the service name `eu.opencloud.api.store`. It does not support TTL and stores the presigning keys indefinitely. Also, the store service needs to be started.


## Special Settings

When using the OpenCloud IDP service instead of an external IDP:

-   Use the environment variable `OC_URL` to define how OpenCloud can be accessed, mandatory use `https` as protocol for the URL.
-   If no reverse proxy is set up, the `PROXY_TLS` environment variable **must** be set to `true` because the embedded `libreConnect` shipped with the IDP service has a hard check if the connection is on TLS and uses the HTTPS protocol. If this mismatches, an error will be logged and no connection from the client can be established.
-   `PROXY_TLS` **can** be set to `false` if a reverse proxy is used and the https connection is terminated at the reverse proxy. When setting to `false`, the communication between the reverse proxy and OpenCloud is not secured. If set to `true`, you must provide certificates.

## Metrics

The proxy service in OpenCloud has the ability to expose metrics in the prometheus format. The metrics are exposed on the `/metrics` endpoint. There are two ways to run the OpenCloud proxy service which has an impact on the number of metrics exposed.

### 1) Single Process Mode
In the single process mode, all OpenCloud services are running inside a single process. This is the default mode when using the `opencloud server` command to start the services. In this mode, the proxy service exposes metrics about the proxy service itself and about the OpenCloud services it is proxying. This is due to the nature of the prometheus registry which is a singleton. The metrics exposed by the proxy service itself are prefixed with `opencloud_proxy_` and the metrics exposed by other opencloud services are prefixed with `opencloud_<service-name>_`.

### 2) Standalone Mode
In this mode, the proxy service only exposes its own metrics. The metrics of the other OpenCloud services are exposed on their own metrics endpoints.

### Available Metrics
The following metrics are exposed by the proxy service:

| Metric Name                      | Description                                                                                                                                                                                                                   | Labels                                |
|----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| `opencloud_proxy_requests_total`      | [Counter](https://prometheus.io/docs/tutorials/understanding_metric_types/#counter) metric which reports the total number of HTTP requests.                                                                                   | `method`: HTTP method of the request  |
| `opencloud_proxy_errors_total`        | [Counter](https://prometheus.io/docs/tutorials/understanding_metric_types/#counter) metric which reports the total number of HTTP requests which have failed. That counts all response codes >= 500                           | `method`: HTTP method of the request  |
| `opencloud_proxy_duration_seconds`    | [Histogram](https://prometheus.io/docs/tutorials/understanding_metric_types/#histogram) of the time (in seconds) each request took. A histogram metric uses buckets to count the number of events that fall into each bucket. | `method`: HTTP method of the request  |
| `opencloud_proxy_build_info{version}` | A metric with a constant `1` value labeled by version, exposing the version of the OpenCloud proxy service.                                                                                                                        | `version`: Build version of the proxy |

### Prometheus Configuration
The following is an example prometheus configuration for the single process mode. It assumes that the proxy debug address is configured to bind on all interfaces `PROXY_DEBUG_ADDR=0.0.0.0:9205` and that the proxy is available via the `opencloud` service name (typically in docker-compose). The prometheus service detects the `/metrics` endpoint automatically and scrapes it every 15 seconds.

```yaml
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: opencloud_proxy
    static_configs:
    - targets: ["opencloud:9205"]
```
