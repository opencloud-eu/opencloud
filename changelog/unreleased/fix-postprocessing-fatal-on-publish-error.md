Bugfix: Retry publishing postprocessing events before giving up

A single transient failure while publishing an event to the event system took
the whole server down. The postprocessing service treated every publish error
as fatal and called log.Fatal, which exits the process and so also stopped all
the other services running in the same binary. A burst of uploads was enough to
run into one nats publish timeout and lose the server with it.

Publishing is now retried using the same exponential backoff that is already
used for failed postprocessing steps. Between the attempts the source event is
marked as in progress and the backoff is capped at half the ack wait, so the
event should not be redelivered to another worker while we are still retrying.
The number of retries is configurable via POSTPROCESSING_PUBLISH_MAX_RETRIES.

Should all attempts fail, the source event is no longer acknowledged. Before,
the event was acknowledged even though its successor was never published, so
the upload was left half processed in the store and did not recover on restart.

https://github.com/opencloud-eu/opencloud/issues/3271
https://github.com/opencloud-eu/opencloud/issues/2232
https://github.com/opencloud-eu/opencloud/issues/2422
