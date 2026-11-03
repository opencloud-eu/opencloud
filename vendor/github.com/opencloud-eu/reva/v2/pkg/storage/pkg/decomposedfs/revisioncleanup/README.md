# Revision cleanup

Every change to a file keeps the previous content as a revision (version).
Without a cleanup, revisions accumulate forever and use up storage.
The revision cleanup deletes old revisions according to a configurable strategy, whenever a file gets a new revision.

It works for the `decomposed`, `decomposeds3` and `posix` storage drivers.
With posix it also covers the revisions created from changes made directly on disk (`enable_fs_revisions`).
For how those drivers are structured, see [`../README.md`](../README.md); for implementation details, see the package documentation in `cleanup.go`.

## Configuration

The options are part of the decomposedfs options and therefore apply to all three drivers.

| Option | Default | Meaning |
|---|---|---|
| `revisions_cleanup_strategy` | `""` (disabled) | Which revisions to delete, see below. Empty or `disabled` keeps all revisions. |
| `revisions_cleanup_async` | `true` | Run the cleanup in background workers, so that it doesn't slow down the upload, restore or change that created the new revision. `false` runs it right away. |
| `revisions_cleanup_async_workers` | `0` (automatic) | Number of background workers. Automatic: half the available CPUs (respecting container CPU limits), at least 2. |
| `revisions_cleanup_async_queue_size` | `0` (automatic) | Number of files that can wait for a background cleanup. Automatic: 64 per worker. |
| `revisions_cleanup_async_queue_full` | `drop` | What happens when the queue is full: `drop` skips the cleanup of that file (logged), it is cleaned up with its next revision; `block` waits for room, up to `revisions_cleanup_async_queue_full_timeout`. |
| `revisions_cleanup_async_queue_full_timeout` | `5s` | How long `block` waits for room at most, after which the cleanup of that file is skipped as with `drop`. `0` waits as long as it takes, so that no cleanup is ever skipped, but then writes can slow down without bound under heavy load. |

An invalid value in any of these options makes the storage driver fail at startup.

## Strategies

A strategy is a comma separated list of up to three parts.
Each `X` is either

- a plain number, which is a **number of revisions**, always counting from the newest one, or
- a number with a unit suffix, which is a **duration**: `w` weeks, `d` days, `h` hours, `m` minutes, `s` seconds.
  Durations in days or weeks can use an uppercase unit, `D` or `W`, to count from midnight (server time) instead of from now:
  `1d` means the last 24 hours, `1D` means since yesterday 00:00, `0D` since today 00:00.

| Strategy | Meaning | Example |
|---|---|---|
| `auto` | Thin out revisions the older they get (see below) | `auto` |
| `X` | Delete everything beyond `X`, no thinning | `10` keeps the 10 newest revisions, `30d` deletes revisions older than 30 days, `0` deletes all revisions |
| `X, auto` | Keep everything within `X`, thin out the older revisions | `14d, auto` |
| `auto, X` | Delete everything beyond `X`, thin out the rest | `auto, 90d`, `auto, 50` |
| `X1, X2` | Keep everything within `X1`, delete everything beyond `X2`; one must be a number and the other a duration | `5, 30d` deletes revisions older than 30 days but always keeps the 5 newest; `7d, 10` keeps at most 10 revisions but never deletes revisions younger than 7 days |
| `X1, auto, X2` | Keep everything within `X1`, delete everything beyond `X2`, thin out in between | `7d, auto, 90d`, `2, auto, 50`, `1D, auto, 30d` |
| `disabled` | Never delete revisions (same as empty) | `disabled` |

Rules worth knowing:

- **`auto` thins out revisions** by keeping one revision per time slot. The slots get longer the older the revisions are:

  | Age of the revision | Kept |
  |---|---|
  | last minute | one per 10 seconds |
  | last hour | one per minute |
  | last 24 hours | one per hour |
  | last 30 days | one per day |
  | older | one per week |

- **A number of revisions is a total**, the protected revisions included: `7d, 10` keeps 10 revisions in total, not 10 on top of the ones from the last 7 days. Revisions within the protected range are never deleted, though, even if there are more of them.
- **In `auto, N`, the number `N` applies first, then `auto` thins out what is left.** So `auto, 10` keeps *at most* 10 revisions, usually fewer, and sometimes fewer than `auto` alone: if the 10 newest revisions were all saved within the same minute yesterday, `auto, 10` keeps a single revision, while `auto` keeps one of those plus one per day before. It does *not* mean "thin out first, then keep the 10 newest of what is left". The same goes for a number in `X1, auto, N`.
- **Two values of the same kind are rejected** in `X1, X2`, because one of them would have no effect: `5, 10` is the same as `10`, and `7d, 30d` is the same as `30d`. The error message names the equivalent single value.
- **In `X1, auto, X2` with two values of the same kind, `X1` must be smaller than `X2`**, otherwise `auto` would never apply.
- **`0` deletes all revisions.** Unlike `disable_versioning`, which only stops creating new revisions, this also removes the ones that have accumulated, over time (see [When the cleanup runs](#when-the-cleanup-runs) below).

## How the age of a revision is measured

A revision's age is the time **its content was written**, i.e. the modification time the file had before it was changed.
That is also the time the revision is named after on disk, and the date the version list shows for it.
It is **not** the time the file was changed, which turned that content into a revision; that time is not recorded anywhere.
Revisions are never modified after they were created; the point is only which of the two times the cleanup looks at.

For files that change often, both dates are close together. For files that sit unchanged for a long time, they are far apart:

1. On August 14, a document is saved.
2. Nobody touches it for two months.
3. On October 14, it is changed. A revision is created to keep the previous content, dated August 14.
4. The cleanup runs right after the change. The revision counts as 61 days old.

With a strategy that deletes by duration, e.g. `30d`, `auto, 30d` or `7d, 30d`, that revision is deleted right away, and the content the document had until that morning is gone.
To always keep the most recent versions of a file regardless of their age, combine the duration with a number of revisions, e.g. `5, 30d`, or use `auto`, which keeps one revision per week even for old content.

## When the cleanup runs

The cleanup of a file runs after the file got a new revision:

- after an upload replaced an existing file, once the upload is complete (with async postprocessing: once postprocessing finished successfully),
- after a revision was restored; the revision holding the content from before the restore is always kept,
- with posix and `enable_fs_revisions`, after a change on disk created a revision.

The cleanup skips a file while an upload to it is still being processed, as that upload may still need its revisions.
Only the latest upload of a file is tracked, though: if an older upload is still being processed after a newer one finished, the cleanup runs anyway.
That is safe: if it deletes the revision the older upload writes its content into, the older upload creates it again when it finishes, and the revision the older upload would roll back to after a failed postprocessing must not be restored anyway once a newer upload finished, as that would discard the newer upload's content.

Files that are never changed again keep their revisions. To clean those up as well, or to apply a different strategy once, the cleanup can also run on demand for whole spaces, folders or files: in OpenCloud with the command `opencloud storage-users revisions cleanup`.

## Metrics

The cleanup exports Prometheus metrics, all named `reva_revision_cleanup_*`.

| Metric | Type | Meaning |
|---|---|---|
| `reva_revision_cleanup_enabled` | gauge | 1 if a strategy is configured, 0 otherwise |
| `reva_revision_cleanup_revisions_deleted_total` | counter | Deleted revisions |
| `reva_revision_cleanup_bytes_deleted_total` | counter | Total size of the deleted revisions |
| `reva_revision_cleanup_purge_failures_total` | counter | Revisions that could not be deleted, which are retried with the file's next revision, and revisions whose blob could not be deleted afterwards, which is left behind (logged with its blob id) |
| `reva_revision_cleanup_queue_full_total{outcome}` | counter | Requests that found the queue full, by `outcome`: `blocked` (the writer waited) or `dropped` (cleanup skipped); a writer that waited until the timeout of `block` counts as both |
| `reva_revision_cleanup_queue_length` | gauge | Files waiting for a background cleanup, including writers waiting for room in a full queue |
| `reva_revision_cleanup_queue_capacity` | gauge | Size of the queue |
| `reva_revision_cleanup_workers` | gauge | Background workers |
| `reva_revision_cleanup_running` | gauge | Files being cleaned up by workers right now |

Things to keep in mind:

- **The queue holds files, not revisions.** A file waits at most once, however many revisions it got in the meantime, and how many of its revisions get deleted is only decided when a worker processes it.
- **One process can have several queues.** The storageprovider and the dataprovider each create their own storage driver, and with it their own cleanup. The queue and worker gauges are the sum over all of them.
  Because a sum can hide one queue being full while another one is empty, `queue_full_total` counts every request that found its queue full.
  It is the most direct sign that there are too few workers or the queue is too small.
- **On-demand cleanups only count their deletions.** A cleanup of whole spaces, folders or files on demand uses a queue and workers of its own, which the queue, worker and `enabled` metrics leave out, as they describe the cleanup that runs whenever a file gets a new revision. The revisions it deletes are counted in `revisions_deleted_total`, `bytes_deleted_total` and `purge_failures_total`.

Example queries:

```promql
# revisions deleted per second, and storage freed per second
rate(reva_revision_cleanup_revisions_deleted_total[5m])
rate(reva_revision_cleanup_bytes_deleted_total[5m])

# how full the queues are (0..1, can exceed 1 while writers wait for room)
reva_revision_cleanup_queue_length / reva_revision_cleanup_queue_capacity

# requests hitting a full queue, worth an alert if it doesn't stay at 0
sum by (outcome) (rate(reva_revision_cleanup_queue_full_total[5m]))

# how busy the workers are (0..1)
reva_revision_cleanup_running / reva_revision_cleanup_workers

# failed deletions, worth an alert
increase(reva_revision_cleanup_purge_failures_total[1h]) > 0
```

## Logging

- At startup with background workers: one info line `revision cleanup: running in the background` with the strategy, workers, queue size, queue full policy and timeout.
- Per cleaned file: one info line `revision cleanup: purged old revisions` with the strategy (`config`), the number and size of the deleted revisions, and how many revisions the file had. Nothing is logged for files where nothing was deleted.
- Every deleted revision is logged at debug level (`revision cleanup: purged revision`), with its `size` and the `reason`: which rule of the strategy deleted it, and the boundary it was evaluated with. This is particularly useful to understand the time slots of `auto`. Examples:
  - `deleted by "30d": older than 2026-09-14T10:00:00Z`
  - `deleted by "10": beyond the 10 newest, counting the 3 protected ones`
  - `deleted by "auto" (last hour): its slot of 1 minute starting 2026-10-14T09:05:00Z already keeps a newer revision`
- A revision dated in the future is logged as a warning (`revision cleanup: revision dated in the future, it counts as the newest one`), every time its file is cleaned up, and its reason ends in `dated in the future (...), so it counts as the newest`. Its time is the modification time of its content, which comes from the client or from the change on disk, e.g. a client with a wrong clock. Such a revision takes the first places of a number of revisions to keep, and no duration deletes it before its date has passed, which can make the cleanup delete the actual newest revisions instead: check the client or the clock.
- With `drop`, or when `block` gives up after its timeout, a skipped cleanup is logged as a warning; on shutdown, the number of abandoned queued cleanups is logged.

## Limitations

- **Shutdown:** cleanups still waiting in the queue are abandoned when the service stops. They are picked up again with the files' next revisions.
