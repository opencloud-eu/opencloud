# Eventhistory

The `eventhistory` consumes all events from the configured event system like NATS, stores them and allows other services to retrieve them via an event ID.

## Prerequisites

Running the eventhistory service without an event system like NATS is not possible.

## Consuming

The `eventhistory` services consumes all events from the configured event system.

## Storing

The `eventhistory` service stores each consumed event via the configured store in `EVENTHISTORY_STORE`. Possible stores are:
  -   `memory`: Basic in-memory store and the default.
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `noop`: Stores nothing. Useful for testing. Not recommended in production environments.

Other store types may work but are not supported currently.

Note: The service can only be scaled if not using `memory` store and the stores are configured identically over all instances!

Note that if you have used one of the deprecated stores, you should reconfigure to one of the supported ones as the deprecated stores will be removed in a later version.

Store specific notes:
  -   When using `nats-js-kv` it is recommended to set `OC_CACHE_STORE_NODES` to the same value as `OC_EVENTS_ENDPOINT`. That way the cache uses the same nats instance as the event bus.
  -   When using the `nats-js-kv` store, it is possible to set `OC_CACHE_DISABLE_PERSISTENCE` to instruct nats to not persist cache data on disc.

## Retrieving

Other services can call the `eventhistory` service via a gRPC call to retrieve events. The request must contain the event ID that should be retrieved.
