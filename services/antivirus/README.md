# Antivirus

The `antivirus` service is responsible for scanning files for viruses.

## Memory Considerations

The antivirus service can consume considerable amounts of memory.
This is relevant to provide or define sufficient memory for the deployment selected.
To avoid out of memory (OOM) situations, the following equation gives a rough overview based on experiences made.
The memory calculation comes without any guarantee, is intended as overview only and subject of change.

`memory limit` = `max file size` x `workers` x `factor 8 - 14`

With:
`ANTIVIRUS_WORKERS` == 1
```plaintext
 50MB file --> factor 14   --> 700MB memory
844MB file --> factor  8,3 -->   7GB memory
```

## Configuration

### Antivirus Scanner Type

The antivirus service currently supports [ICAP](https://tools.ietf.org/html/rfc3507) and [ClamAV](http://www.clamav.net/index.html) as antivirus scanners.
The `ANTIVIRUS_SCANNER_TYPE` environment variable is used to select the scanner.
The detailed configuration for each scanner heavily depends on the scanner type selected.
See the environment variables for more details.

  -   For `icap`, only scanners using the `X-Infection-Found` header are currently supported.
  -   For `clamav` only local sockets can currently be configured.

### Maximum Scan Size

Several factors can make it necessary to limit the maximum filesize the antivirus service uses for scanning.
Use the `ANTIVIRUS_MAX_SCAN_SIZE` environment variable to scan only a given number of bytes,
or to skip the whole resource.

Even if it is recommended to scan the whole file, several factors like scanner type and version,
bandwidth, performance issues, etc. might make a limit necessary.

In such cases, the antivirus max scan size mode can be handy, the following modes are available:

  -   `partial`: The file is scanned up to the given size. The rest of the file is not scanned. This is the default mode `ANTIVIRUS_MAX_SCAN_SIZE_MODE=partial`
  -   `skip`: The file is skipped and not scanned. `ANTIVIRUS_MAX_SCAN_SIZE_MODE=skip`

**IMPORTANT**
> Streaming of files to the virus scan service still [needs to be implemented](https://github.com/owncloud/ocis/issues/6803).
> To prevent OOM errors `ANTIVIRUS_MAX_SCAN_SIZE` needs to be set lower than available ram and or the maximum file size that can be scanned by the virus scanner.

### Antivirus Workers

The number of concurrent scans can be increased by setting `ANTIVIRUS_WORKERS`. Be aware that this will also increase memory usage.

### Scan queue priority

Antivirus requests are copied from the existing event stream into durable
JetStream high- and low-priority queues. A user's jobs stay high priority until
they exceed `ANTIVIRUS_PRIORITY_THRESHOLD` scan requests during
`ANTIVIRUS_PRIORITY_WINDOW` (more than 10 requests per second by default).
After that, the user's jobs are low priority for
`ANTIVIRUS_PRIORITY_COOLDOWN` (30 seconds by default). Continued high-rate
activity extends the cooldown. Other users' jobs are selected before queued
low-priority jobs; there is no round-robin between users.

High-lane jobs are held for the configured rate window before scanning so a
burst can be classified first. This adds up to one second of queueing by
default. If a user crosses the threshold during that window, their queued
high-lane jobs are moved to the low lane. Jobs already running cannot be
demoted or preempted.

Jobs for the same storage resource stay in source-event order across both
lanes and through the corresponding `UploadReady` event. A newer high-priority
edit cannot overtake an older low-priority scan or its finalization. Retry
attempts keep their original place in that resource's order.

By default, one `ANTIVIRUS_WORKERS` slot is reserved for high-priority work
when multiple scan workers are configured. With one scan worker, the default
reservation is zero. Reserved workers never take low-priority jobs; the other
workers check for high-priority jobs first. This protects some capacity for
interactive work, at the cost of leaving that slot idle when only bulk work is
queued. Set `ANTIVIRUS_HIGH_PRIORITY_RESERVED_WORKERS=0` to disable it; the
value must be less than `ANTIVIRUS_WORKERS`.

This is prioritization, not rate limiting: uploads are accepted and low-priority
scans remain queued. Strict high-first scheduling can starve low-priority jobs
while the high-priority queue never empties. Scans already running cannot be
preempted, so the reserved worker cannot guarantee a fixed latency during a
burst of high-priority work. A user's Collabora saves are also classified by
that user's upload rate; interactive edits are not distinguished separately.

The scheduler creates the `OPENCLOUD_ANTIVIRUS_JOBS` JetStream stream and the
`ANTIVIRUS_USER_RATES` and `ANTIVIRUS_RESOURCE_ORDER` KV buckets. The NATS
account must be allowed to create and use streams, consumers, and KV buckets.
These resources are durable and shared by antivirus replicas. Set
`ANTIVIRUS_QUEUE_REPLICAS` to the desired NATS replication factor (default 1).
Job acknowledgements are refreshed while scans run; `ANTIVIRUS_QUEUE_ACK_WAIT`
controls the redelivery timeout.

Configuration:

* `ANTIVIRUS_PRIORITY_THRESHOLD` (default `10`): requests per user allowed in
  the priority window before new requests are demoted.
* `ANTIVIRUS_PRIORITY_WINDOW` (default `1s`): rolling rate window.
* `ANTIVIRUS_PRIORITY_COOLDOWN` (default `30s`): low-priority cooldown.
* `ANTIVIRUS_QUEUE_INTAKE_WORKERS` (default `10`): event intake workers,
  independent of scan concurrency.
* `ANTIVIRUS_HIGH_PRIORITY_RESERVED_WORKERS` (default `1`, or `0` with one
  scan worker): reserved high-priority workers. Must be less than
  `ANTIVIRUS_WORKERS`.
* `ANTIVIRUS_QUEUE_ACK_WAIT` (default `1m`): job redelivery timeout. Workers
  send lease heartbeats; this value must stay below `2m`.
* `ANTIVIRUS_QUEUE_REPLICAS` (default `1`): replication factor for queue state.

Metrics are available without user-ID labels:
`opencloud_antivirus_jobs_pending{priority}`,
`opencloud_antivirus_jobs_in_flight{priority}`,
`opencloud_antivirus_jobs_enqueued_total{priority}`, and
`opencloud_antivirus_queue_wait_seconds{priority}`.

When upgrading, stop old antivirus replicas before starting the new version,
then start the new replicas together. The implementation reuses the existing
`antivirus` durable consumer cursor; mixed old/new replicas can process the
same group with different queue behavior.

### Infected File Handling

The antivirus service allows three different ways of handling infected files. Those can be set via the `ANTIVIRUS_INFECTED_FILE_HANDLING` environment variable:

  -   `delete`: (default): Infected files will be deleted immediately, further postprocessing is cancelled.
  -   `abort`:  (advanced option): Infected files will be kept, further postprocessing is cancelled. Files can be manually retrieved and inspected by an admin. To identify the file for further investigation, the antivirus service logs the abort/infected state including the file ID. The file is located in the `storage/users/uploads` folder of the OpenCloud data directory and persists until it is manually deleted by the admin via the [Manage Unfinished Uploads](https://github.com/opencloud-eu/opencloud/tree/main/services/storage-users#manage-unfinished-uploads) command.
  -   `continue`:  (not recommended): Infected files will be marked via metadata as infected, but postprocessing continues normally. Note: Infected Files are moved to their final destination and therefore not prevented from download, which includes the risk of spreading viruses.

In all cases, a log entry is added declaring the infection and handling method and a notification via the `userlog` service sent.

### Scanner Inaccessibility

In case a scanner is not accessible by the antivirus service like a network outage, service outage or hardware outage, the antivirus service uses the `abort` case for further processing, independent of the actual setting made. In any case, an error is logged noting the inaccessibility of the scanner used.

## Operation Modes

The antivirus service can scan files during `postprocessing`. `on demand` scanning is currently not available and might be added in a future release.

### Postprocessing

The antivirus service will scan files during postprocessing. It listens for a postprocessing step called `virusscan`. This step can be added in the environment variable `POSTPROCESSING_STEPS`. Read the documentation of the [postprocessing service](https://github.com/opencloud-eu/opencloud/tree/main/services/postprocessing) for more details.

The number of concurrent scans can be increased by setting `ANTIVIRUS_WORKERS`, but be aware that this will also increase the memory usage.

### Scaling in Kubernetes

In kubernetes, `ANTIVIRUS_WORKERS` and `ANTIVIRUS_MAX_SCAN_SIZE` can be used to trigger the horizontal pod autoscaler by requesting a memory size that is below `ANTIVIRUS_MAX_SCAN_SIZE`. Keep in mind that `ANTIVIRUS_MAX_SCAN_SIZE` amount of memory might be held by `ANTIVIRUS_WORKERS` number of go routines.
