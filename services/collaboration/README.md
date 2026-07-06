# Collaboration

The collaboration service connects opencloud with document servers such as Collabora, ONLYOFFICE or Microsoft using the WOPI protocol.

Since this service requires an external document server, it is not enabled by default. You need to add it to `OC_RUN_SERVICES`, `OC_ADD_RUN_SERVICES` or start it in a separate process with the `opencloud collaboration server` command.

Because the collaboration service needs to be started manually, the following prerequisite applies: On collaboration service startup, particular environment variables are required to be populated. If environment variables have a default like the `MICRO_REGISTRY_ADDRESS`, the default will be used, if not set otherwise. Use for all others the instance values as defined. If these environment variables are not provided or misconfigured, the collaboration service will not start up.

Required environment variables:
* `OC_URL`
* `OC_JWT_SECRET`
* `OC_REVA_GATEWAY`
* `MICRO_REGISTRY_ADDRESS`

## Requirements

The collaboration periodically checks the target document server (ONLYOFFICE, Collabora, etc.) to update the available edit options. Additionally, some OpenCloud services are required to be running in order to register the GRPC service for the `open in app` action in the webUI. The following internal and external services need to be available:

* The gateway service.
* The app-registry service.

If any of the named services above have not been started or are not reachable, the collaboration service won't start. For the binary or the docker release of OpenCloud, check with the `opencloud list` command if they have been started. If not, you must start them manually upfront before starting the collaboration service.

## WOPI Configuration

There are a few variables that you need to set:

* `COLLABORATION_APP_NAME`:\
  The name of the app which is shown to the user. You can chose freely but you are limited to a single word without special characters or whitespaces. We recommend to use pascalCase like 'CollaboraOnline'.

* `COLLABORATION_APP_PRODUCT`:\
  The product name of the connected WebOffice app, which can be one of the following:\
  `Collabora`, `OnlyOffice`, `Microsoft365` or `MicrosoftOfficeOnline`. This is used to internally control the behavior according to the different features of the used products.

* `COLLABORATION_APP_ADDR`:\
  The URL of the collaborative editing app (onlyoffice, collabora, etc).\
  For example: `https://office.example.com`.

* `COLLABORATION_APP_INSECURE`:\
  In case you are using a self signed certificate for the WOPI app you can tell the collaboration service to allow an insecure connection.

* `COLLABORATION_WOPI_SRC`:\
  The external address of the collaboration service. The target app (onlyoffice, collabora, etc) will use this address to read and write files from OpenCloud.\
  For example: `https://wopi.example.com`.

* `COLLABORATION_WOPI_SHORTTOKENS`:\
  Needs to be set if the office application like `Microsoft Office Online` complains about the URL is too long  (which contains the access token) and refuses to work. If enabled, a store must be configured.

The application can be customized further by changing the `COLLABORATION_APP_*` options to better describe the application.

## Storing

The `collaboration` service persists information via the configured store in `COLLABORATION_STORE`. Possible stores are:
  -   `memory`: Basic in-memory store. Will not survive a restart. This is not recommended for this service.
  -   `redis-sentinel`: Stores data in a configured Redis Sentinel cluster.
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store). This is the default value.
  -   `noop`: Stores nothing. Useful for testing. Not recommended in production environments.

Other store types may work but are not supported currently.

Note: The service can only be scaled if not using `memory` store and the stores are configured identically over all instances!

Note that if you have used one of the deprecated stores, you should reconfigure to one of the supported ones as the deprecated stores will be removed in a later version.

Store specific notes:
  -   When using `redis-sentinel`, the Redis master to use is configured via e.g. `OC_CACHE_STORE_NODES` in the form of `<sentinel-host>:<sentinel-port>/<redis-master>` like `10.10.0.200:26379/mymaster`.
  -   When using `nats-js-kv` it is recommended to set `OC_CACHE_STORE_NODES` to the same value as `OC_EVENTS_ENDPOINT`. That way the cache uses the same nats instance as the event bus.
  -   When using the `nats-js-kv` store, it is possible to set `OC_CACHE_DISABLE_PERSISTENCE` to instruct nats to not persist cache data on disc.

