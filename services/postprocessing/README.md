# Postprocessing

The `postprocessing` service handles the coordination of asynchronous postprocessing steps.

## General Prerequisites

To use the postprocessing service, an event system needs to be configured for all services. By default, `OpenCloud` ships with a preconfigured `nats` service.

## Postprocessing Functionality

The storageprovider service (`storage-users`) can be configured to initiate asynchronous postprocessing by setting the `OC_ASYNC_UPLOADS` environment variable to `true`. If this is the case, postprocessing will get initiated *after* uploading a file and all bytes have been received.

The `postprocessing` service will then coordinate configured postprocessing steps like scanning the file for viruses. During postprocessing, the file will be in a `processing state` where only a limited set of actions are available. Note that this processing state excludes file accessibility by users.

When all postprocessing steps have completed successfully, the file will be made accessible for users.

## Storing Postprocessing Data

The `postprocessing` service needs to store some metadata about uploads to be able to orchestrate post-processing. When running in single binary mode, the default in-memory implementation will be just fine. In distributed deployments it is recommended to use a persistent store, see below for more details.

The `postprocessing` service stores its metadata via the configured store in `POSTPROCESSING_STORE`. Possible stores are:
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

## Additional Prerequisites for the Postprocessing Service

When postprocessing has been enabled, configuring any postprocessing step will require the requested services to be enabled and pre-configured. For example, to use the `virusscan` step, one needs to have an enabled and configured `antivirus` service.

## Postprocessing Steps

The postporcessing service is individually configurable. This is achieved by allowing a list of postprocessing steps that are processed in order of their appearance in the `POSTPROCESSING_STEPS` envvar. This envvar expects a comma separated list of steps that will be executed. Currently known steps to the system are `virusscan` and `delay`. Custom steps can be added but need an existing target for processing.

### Virus Scanning

To enable virus scanning as a postprocessing step after uploading a file, the environment variable `POSTPROCESSING_STEPS` needs to contain the word `virusscan` at one location in the list of steps. As a result, each uploaded file gets virus scanned as part of the postprocessing steps. Note that the `antivirus` service is required to be enabled and configured for this to work.

### Delay

Though this is for development purposes only and NOT RECOMMENDED on production systems, setting the environment variable `POSTPROCESSING_DELAY` to a duration not equal to zero will add a delay step with the configured amount of time. OpenCloud will continue postprocessing the file after the configured delay. Use the environment variable `POSTPROCESSING_STEPS` and the keyword `delay` if you have multiple postprocessing steps and want to define their order. If `POSTPROCESSING_DELAY` is set but the keyword `delay` is not contained in `POSTPROCESSING_STEPS`, it will be processed as last postprocessing step without being listed there. In this case, a log entry will be written on service startup to notify the admin about that situation. That log entry can be avoided by adding the keyword `delay` to `POSTPROCESSING_STEPS`.

### Custom Postprocessing Steps
By using the envvar `POSTPROCESSING_STEPS`, custom postprocessing steps can be added. Any word can be used as step name but be careful not to conflict with exising keywords like `virusscan` and `delay`. In addition, if a keyword is misspelled or the corresponding service does either not exist or does not follow the necessary event communication, the postprocessing service will wait forever getting the required response to proceed and does not continue any other processing.

#### Prerequisites
For using custom postprocessing steps you need a custom service listening to the configured event system (see `General Prerequisites`)

#### Workflow
When defining a custom postprocessing step (eg. `"customstep"`), the postprocessing service will eventually send an event during postprocessing. The event will be of type `StartPostprocessingStep` with its field `StepToStart` set to `"customstep"`. When the service defined as custom step receives this event, it can safely execute its actions. The postprocessing service will wait until it has finished its work. The event contains further information (filename, executing user, size, ...) and also requires tokens and URLs to download the file in case byte inspection is necessary.

Once the service defined as custom step has finished its work, it should send an event of type `PostprocessingFinished` via the configured events system back to the postprocessing service. This event needs to contain a `FinishedStep` field set to `"customstep"`. It also must contain the outcome of the step, which can be one of the following:

-   `delete`: Abort postprocessing, delete the file.
-   `abort`: Abort postprocessing, keep the file.
-   `retry`: There was a problem that was most likely temporary and may be solved by trying again after some backoff duration. Retry runs automatically and is defined by the backoff behavior as described below.
-   `continue`: Continue postprocessing, this is the success case.

The backoff behavior as mentioned in the `retry` outcome can be configured using the `POSTPROCESSING_RETRY_BACKOFF_DURATION` and `POSTPROCESSING_MAX_RETRIES` environment variables. The backoff duration is calculated using the following formula after each failure: `backoff_duration = POSTPROCESSING_RETRY_BACKOFF_DURATION * 2^(number of failures - 1)`. This means that the time between the next round grows exponentially limited by the number of retries. Steps that still don't succeed after the maximum number of retries will be automatically moved to the `abort` state.

See the [cs3 org](https://github.com/cs3org/reva/blob/edge/pkg/events/postprocessing.go) for up-to-date information of reserved step names and event definitions.

## CLI Commands

### Resume Postprocessing

**IMPORTANT**
> If not noted otherwise, commands with the `restart` option can also use the `resume` option. This changes behaviour slightly.
>
> * `restart`\
> When restarting an upload, all steps for open items will be restarted, except if otherwise defined.
> * `resume`\
> When resuming an upload, processing will continue unfinished items from their last completed step.

If post-processing fails in one step due to an unforeseen error, current uploads will not be resumed automatically. A system administrator can instead run CLI commands to resume the failed upload manually which is at minimum a two step process.

For details on the `storage-users` command see the **Manage Unfinished Uploads** documentation in the `storage-users` service documentation.

Depending if you want to restart/resume all or defined failed uploads, different commands are used.

-   First, list ongoing upload sessions to identify possibly failed ones.\
    Note that there never can be a clear identification of a failed upload session due to various reasons causing them. You need to apply more critera like free space on disk, a failed service like antivirus etc. to declare an upload as failed.

    ```bash
    opencloud storage-users uploads sessions
    ```

-   **All failed uploads**\
    If you want to restart/resume all failed uploads, just rerun the command with the relevant flag. Note that this is the preferred command to handle failed processing steps:
    ```bash
    opencloud storage-users uploads sessions --resume
    ```

-   **Particular failed uploads**\
    Use the `postprocessing` command to resume defined failed uploads. For postprocessing steps, the default is to resume . Note that at the moment, `resume` is an alias for `restart` to keep old functionality. `restart` is subject of change and will most likely be removed in a later version.

    - **Defined by ID**\
      If you want to resume only a specific upload, use the postprocessing resume command with the ID selected:
      ```bash
      opencloud postprocessing resume -u <uploadID>
      ```

    - **Defined by step**\
      Alternatively, instead of restarting one specific upload, a system admin can also resume all uploads that are currently in a specific step.\
      Examples:\
      ```bash
      opencloud postprocessing resume                # Resumes all uploads where postprocessing is finished, but upload is not finished
      opencloud postprocessing resume -s "finished"  # Equivalent to the above
      opencloud postprocessing resume -s "virusscan" # Resume all uploads currently in virusscan step
      ```

## Metrics

The postprocessing service exposes the following prometheus metrics at `<debug_endpoint>/metrics` (as configured using the `POSTPROCESSING_DEBUG_ADDR` env var):

| Metric Name | Type | Description | Labels |
| --- | --- | --- | --- |
| `opencloud_postprocessing_build_info` | Gauge | Build information | `version` |
| `opencloud_postprocessing_events_outstanding_acks` | Gauge | Number of outstanding acks for events | |
| `opencloud_postprocessing_events_unprocessed` | Gauge | Number of unprocessed events | |
| `opencloud_postprocessing_events_redelivered` | Gauge | Number of redelivered events | |
| `opencloud_postprocessing_in_progress` | Gauge | Number of postprocessing events in progress | |
| `opencloud_postprocessing_finished` | Counter | Number of finished postprocessing events | `status` |
| `opencloud_postprocessing_duration_seconds` | Histogram | Duration of postprocessing operations in seconds | `status` |
