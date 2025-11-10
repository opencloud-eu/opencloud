# Userlog

The `userlog` service is a mediator between the `eventhistory` service and clients who want to be informed about user related events. It provides an API to retrieve those.

## The Log Service Ecosystem

Log services like the `userlog`, `clientlog` and `sse` are responsible for composing notifications for a certain audience.
  -   The `userlog` service translates and adjusts messages to be human readable.
  -   The `clientlog` service composes machine readable messages, so clients can act without the need to query the server.
  -   The `sse` service is only responsible for sending these messages. It does not care about their form or language.

## Prerequisites

Running the `userlog` service without running the `eventhistory` service is not possible.

## Storing

The `userlog` service persists information via the configured store in `USERLOG_STORE`. Possible stores are:
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

## Configuring

For the time being, the configuration which user related events are of interest is hardcoded and cannot be changed.

## Retrieving

The `userlog` service provides an API to retrieve configured events. For now, this API is mostly following the [oc10 notification GET API](https://doc.owncloud.com/server/next/developer_manual/core/apis/ocs-notification-endpoint-v1.html#get-user-notifications).

## Posting

The userlog service is able to store global messages that will be displayed in the Web UI to all users. If a user deletes the message in the Web UI, it reappears on reload. Global messages use the endpoint `/ocs/v2.php/apps/notifications/api/v1/notifications/global` and are activated by sending a `POST` request. Note that sending another `POST` request of the same type overwrites the previous one. For the time being, only the type `deprovision` is supported.

### Authentication

`POST` and `DELETE` endpoints provide notifications to all users. Therefore only certain users can configure them. Two authentication methods for this endpoint are provided. Users with the `admin` role can always access these endpoints. Additionally, a static secret via the `USERLOG_GLOBAL_NOTIFICATIONS_SECRET` can be defined to enable access for users knowing this secret, which has to be sent with the header containing the request.

### Deprovisioning

Deprovision messages announce a deprovision text including a deprovision date of the instance to all users. With this message, users get informed that the instance will be shut down and deprovisioned and no further access to their data is possible past the given date. This implies that users must download their data before the given date. The text shown to users refers to this information. Note that the task to deprovision the instance does not depend on the message. The text of the message can be translated according to the translation settings, see section [Translations](#translations). The endpoint only expects a `deprovision_date` parameter in the `POST` request body as the final text is assembled automatically. The string hast to be in `RFC3339` format, however, this format can be changed by using `deprovision_date_format`. See the [go time formating](https://pkg.go.dev/time#pkg-constants) for more details.

## Deleting

To delete events for an user, use a `DELETE` request to `ocs/v2.php/apps/notifications/api/v1/notifications` containing the IDs to delete.

Sending a `DELETE` request to the `ocs/v2.php/apps/notifications/api/v1/notifications/global` endpoint to remove a global message is a restricted action, see the [Authentication](#authentication) section for more details.)

## Translations

The `userlog` service has embedded translations sourced via transifex to provide a basic set of translated languages. These embedded translations are available for all deployment scenarios. In addition, the service supports custom translations, though it is currently not possible to just add custom translations to embedded ones. If custom translations are configured, the embedded ones are not used. To configure custom translations, the `USERLOG_TRANSLATION_PATH` environment variable needs to point to a base folder that will contain the translation files. This path must be available from all instances of the userlog service, a shared storage is recommended. Translation files must be of type  [.po](https://www.gnu.org/software/gettext/manual/html_node/PO-Files.html#PO-Files) or [.mo](https://www.gnu.org/software/gettext/manual/html_node/Binaries.html). For each language, the filename needs to be `userlog.po` (or `userlog.mo`) and stored in a folder structure defining the language code. In general the path/name pattern for a translation file needs to be:

```text
{USERLOG_TRANSLATION_PATH}/{language-code}/LC_MESSAGES/userlog.po
```

The language code pattern is composed of `language[_territory]` where  `language` is the base language and `_territory` is optional and defines a country.

For example, for the language `de`, one needs to place the corresponding translation files to `{USERLOG_TRANSLATION_PATH}/de_DE/LC_MESSAGES/userlog.po`.

<!-- also see the notifications readme -->

Important: For the time being, the embedded OpenCloud Web frontend only supports the main language code but does not handle any territory. When strings are available in the language code `language_territory`, the web frontend does not see it as it only requests `language`. In consequence, any translations made must exist in the requested `language` to avoid a fallback to the default.

### Translation Rules

*   If a requested language code is not available, the service tries to fall back to the base language if available. For example, if the requested language-code `de_DE` is not available, the service tries to fall back to translations in the `de` folder.
*   If the base language `de` is also not available, the service falls back to the system's default English (`en`),
which is the source of the texts provided by the code.

## Default Language

The default language can be defined via the `OC_DEFAULT_LANGUAGE` environment variable. See the `settings` service for a detailed description.
