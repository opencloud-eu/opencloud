# Graph

The graph service provides the Graph API which is a RESTful web API used to access OpenCloud
resources. It is inspired by the [Microsoft Graph API](https://learn.microsoft.com/en-us/graph/use-the-api)
and can be used by clients or other services or extensions. Visit the [Libre Graph API](https://docs.opencloud.eu/swagger/libre-graph-api/)
for a detailed specification of the API implemented by the graph service.

## Sequence Diagram

The following image gives an overview of the scenario when a client requests to list available spaces the user has access to. To do so, the client is directed with his request automatically via the proxy service to the graph service.

<img src="https://raw.githubusercontent.com/opencloud-eu/opencloud/main/services/graph/images/mermaid-graph.svg" width="500" />

## Users and Groups API

The graph service provides endpoints for querying users and groups. It features two different backend implementations:
  * `ldap`: This is currently the default backend. It queries user and group information from an
    LDAP server. Depending on the configuration, it can also be used to manage (create, update,
    delete) users and groups provided by an LDAP server.
  * `cs3`: This backend queries users and groups using the CS3 identity APIs as implemented by the
    `users` and `groups` service. This backend is currently still experimental and only implements a
    subset of the Libre Graph API. It should not be used in production.

### LDAP Configuration

The LDAP backend is configured using a set of environment variables. A detailed list of all the
available configuration options can be found in the [documentation](https://docs.opencloud.eu/docs/dev/server/services/graph/environment-variables).
The LDAP related options are prefixed with `OC_LDAP_` (or `GRAPH_LDAP_` for settings specific to graph service).

#### Read-Only Access to Existing LDAP Servers

To connect the graph service to an existing LDAP server, set `OC_LDAP_SERVER_WRITE_ENABLED` to
`false` to prevent the graph service from sending write operations to the LDAP server. Also set the
various `OC_LDAP_*` environment variables to match the configuration of the LDAP server you are connecting
to. A more detailed explanation can be found [here](https://docs.opencloud.eu/docs/admin/configuration/authentication-and-user-management/.

#### Using a Write Enabled LDAP Server

To use the graph service for managing (create, update, delete) users and groups, a write enabled LDAP
server is required. In the default configuration, the graph service will use the simple LDAP server
that is bundled with OpenCloud in the `idm` service which provides all the required features.
It is also possible to setup up an external LDAP server with write access for use with OpenCloud. It is
recommend to use OpenLDAP for this. The LDAP server needs to fulfill a couple of requirements with
respect to the available schema:
  * The LDAP server must provide the `inetOrgPerson` object class for users and the `groupOfNames`
    object class for groups.
  * The graph service maintains a few additional attributes for users and groups that are not
    available in the standard LDAP schema. An schema file, ready to use with OpenLDAP, defining those
    additional attributes is available [here](https://github.com/opencloud-eu/opencloud-compose/blob/main/config/ldap/schemas/10_opencloud_schema.ldif)

## Query Filters Provided by the Graph API

Some API endpoints provided by the graph service allow to specify query filters. The filter syntax
is based on the [OData Specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionfilter).
See the [Libre Graph API](https://docs.opencloud.eu/swagger/libre-graph-api/#/users/ListUsers) for examples
on the filters supported when querying users.

## Caching

The `graph` service can use a configured store via `GRAPH_CACHE_STORE`. Possible stores are:
  -   `memory`: Basic in-memory store and the default.
  -   `redis-sentinel`: Stores data in a configured Redis Sentinel cluster.
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `noop`: Stores nothing. Useful for testing. Not recommended in production environments.

Other store types may work but are not supported currently.

Store specific notes:
  -   When using `redis-sentinel`, the Redis master to use is configured via e.g. `OC_CACHE_STORE_NODES` in the form of `<sentinel-host>:<sentinel-port>/<redis-master>` like `10.10.0.200:26379/mymaster`.
  -   When using `nats-js-kv` it is recommended to set `OC_CACHE_STORE_NODES` to the same value as `OC_EVENTS_ENDPOINT`. That way the cache uses the same nats instance as the event bus.
  -   When using the `nats-js-kv` store, it is possible to set `OC_CACHE_DISABLE_PERSISTENCE` to instruct nats to not persist cache data on disc.

## Keycloak Configuration For The Personal Data Export

If Keycloak is used for authentication, GDPR regulations require to add all personal identifiable information that Keycloak has about the user to the personal data export. To do this, the following environment variables must be set:

*   `OC_KEYCLOAK_BASE_PATH` - The URL to the keycloak instance.
*   `OC_KEYCLOAK_CLIENT_ID` - The client ID of the client that is used to authenticate with keycloak, this client has to be able to list users and get the credential data.
*   `OC_KEYCLOAK_CLIENT_SECRET` - The client secret of the client that is used to authenticate with keycloak.
*   `OC_KEYCLOAK_CLIENT_REALM` - The realm the client is defined in.
*   `OC_KEYCLOAK_USER_REALM` - The realm the OpenCloud users are defined in.
*   `OC_KEYCLOAK_INSECURE_SKIP_VERIFY` - If set to true, the TLS certificate of the keycloak instance is not verified.

### Keycloak Client Configuration

The client that is used to authenticate with keycloak has to be able to list users and get the credential data. To do this, the following  roles have to be assigned to the client and they have to be about the realm that contains the OpenCloud users:

*   `view-users`
*   `view-identity-providers`
*   `view-realm`
*   `view-clients`
*   `view-events`
*   `view-authorization`

:::note
These roles are only available to assign if the client is in the `master` realm.
:::

## Translations

The `graph` service has embedded translations sourced via transifex to provide a basic set of translated languages. These embedded translations are available for all deployment scenarios. In addition, the service supports custom translations, though it is currently not possible to just add custom translations to embedded ones. If custom translations are configured, the embedded ones are not used. To configure custom translations, the `GRAPH_TRANSLATION_PATH` environment variable needs to point to a base folder that will contain the translation files. This path must be available from all instances of the graph service, a shared storage is recommended. Translation files must be of type  [.po](https://www.gnu.org/software/gettext/manual/html_node/PO-Files.html#PO-Files) or [.mo](https://www.gnu.org/software/gettext/manual/html_node/Binaries.html). For each language, the filename needs to be `graph.po` (or `graph.mo`) and stored in a folder structure defining the language code. In general the path/name pattern for a translation file needs to be:

```text
{GRAPH_TRANSLATION_PATH}/{language-code}/LC_MESSAGES/graph.po
```

The language code pattern is composed of `language[_territory]` where  `language` is the base language and `_territory` is optional and defines a country.

For example, for the language `de`, one needs to place the corresponding translation files to `{GRAPH_TRANSLATION_PATH}/de_DE/LC_MESSAGES/graph.po`.

<!-- also see the notifications readme -->

:::warning
For the time being, the embedded OpenCloud Web frontend only supports the main language code but does not handle any territory. When strings are available in the language code `language_territory`, the web frontend does not see it as it only requests `language`. In consequence, any translations made must exist in the requested `language` to avoid a fallback to the default.
:::

### Translation Rules

*   If a requested language code is not available, the service tries to fall back to the base language if available. For example, if the requested language-code `de_DE` is not available, the service tries to fall back to translations in the `de` folder.
*   If the base language `de` is also not available, the service falls back to the system's default English (`en`),
which is the source of the texts provided by the code.

## Default Language

The default language can be defined via the `OC_DEFAULT_LANGUAGE` environment variable. See the `settings` service for a detailed description.

## Unified Role Management

Unified Roles are roles granted a user for sharing and can be enabled or disabled. A CLI command is provided to list existing roles and their state among other data.

:::info
Note that a disabled role does not lose previously assigned permissions. It only means that the role is not available for new assignments.
:::

The following roles are **enabled** by default:

- `UnifiedRoleViewerID`
- `UnifiedRoleSpaceViewer`
- `UnifiedRoleEditor`
- `UnifiedRoleSpaceEditor`
- `UnifiedRoleFileEditor`
- `UnifiedRoleEditorLite`
- `UnifiedRoleManager`

The following role is **disabled** by default:

- `UnifiedRoleSecureViewer`

To enable disabled roles like the `UnifiedRoleSecureViewer`, you must provide the UID(s) by one of the following methods:

- Using the `GRAPH_AVAILABLE_ROLES` environment variable.
- Setting the `available_roles` configuration value.

The following CLI command simplifies the process of finding out which UID belongs to which role:

```bash
opencloud graph list-unified-roles
```

The output of this command includes the following information for each role:

* `UID`\
  The unique identifier of the role.
* `Enabled`\
  Whether the role is enabled or not.
* `Description`\
  A short description of the role.
* `Condition`
* `Allowed resource actions`

**Example output (shortned)**

```bash
+--------------------------------------+----------+--------------------------------+--------------------------------+------------------------------------------+
|                 UID                  | ENABLED  |          DESCRIPTION           |           CONDITION            |         ALLOWED RESOURCE ACTIONS         |
+--------------------------------------+----------+--------------------------------+--------------------------------+------------------------------------------+
| a8d5fe5e-96e3-418d-825b-534dbdf22b99 | enabled  | View and download.             | exists @Resource.Root          | libre.graph/driveItem/path/read          |
|                                      |          |                                |                                | libre.graph/driveItem/quota/read         |
|                                      |          |                                |                                | libre.graph/driveItem/content/read       |
|                                      |          |                                |                                | libre.graph/driveItem/permissions/read   |
|                                      |          |                                |                                | libre.graph/driveItem/children/read      |
|                                      |          |                                |                                | libre.graph/driveItem/deleted/read       |
|                                      |          |                                |                                | libre.graph/driveItem/basic/read         |
+--------------------------------------+----------+--------------------------------+--------------------------------+------------------------------------------+
```

