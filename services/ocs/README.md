# OCS Service

The `ocs` service (open collaboration services) serves one purpose: it has an endpoint for signing keys which the web frontend accesses when uploading data.

## Signing-Keys Endpoint

The `ocs` service contains an endpoint `/cloud/user/signing-key` on which a user can GET a signing key. Note, this functionality might be deprecated or moved in the future.

## Signing-Keys Store

To authenticate presigned URLs the proxy service needs to read the signing keys from a store that is populated by the ocs service.
Possible stores that can be configured via `OCS_PRESIGNEDURL_SIGNING_KEYS_STORE` are:
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `opencloudstoreservice`:  Stores data in the legacy OpenCloud store service. Requires setting `OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` to `eu.opencloud.api.store`.

The `memory` store cannot be used as it does not share the memory from the ocs service signing key memory store, even in a single process.

Make sure to configure the same store in the proxy service.

Store specific notes:
  -   When using `nats-js-kv` it is recommended to set `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` to the same value as `OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES`. That way the proxy uses the same nats instance as the ocs service.
  -   When using `opencloudstoreservice` the `OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` must be set to the service name `eu.opencloud.api.store`. It does not support TTL and stores the presigning keys indefinitely. Also, the store service needs to be started.
