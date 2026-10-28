# Storage-Users

Purpose and description to be added

## Graceful Shutdown

You can define a graceful shutdown period for the `storage-users` service.

IMPORTANT: The graceful shutdown period is only applicable if the `storage-users` service runs as standalone service. It does not apply if the `storage-users` service runs as part of the single binary or as single Docker environment. To build an environment where the `storage-users` service runs as a standalone service, you must start two instances, one _without_ the `storage-users` service and one _only with_ the the `storage-users` service. Note that both instances must be able to communicate on the same network.

When hard-stopping OpenCloud, for example with the `kill <pid>` command (SIGKILL), it is possible and likely that not all data from the decomposedfs (metadata) has been written to the storage which may result in an inconsistent decomposedfs. When gracefully shutting down OpenCloud, using a command like SIGTERM, the process will no longer accept any write requests from _other_ services and will try to write the internal open  requests which can take an undefined duration based on many factors. To mitigate that situation, the following things have been implemented:

*   With the value of the environment variable `STORAGE_USERS_GRACEFUL_SHUTDOWN_TIMEOUT`, the `storage-users` service will delay its shutdown giving it time to finalize writing necessary data. This delay can be necessary if there is a lot of data to be saved and/or if storage access/thruput is slow. In such a case you would receive an error log entry informing you that not all data could be saved in time. To prevent such occurrences, you must increase the default value.

*   If a shutdown error has been logged, the command-line maintenance tool to inspect and manipulate node metadata can help to fix the issue. Please contact support for details.

## CLI Commands

For any command listed, use `--help` to get more details and possible options and arguments.

To authenticate CLI commands use:

*   `OC_SERVICE_ACCOUNT_SECRET=<acc-secret>` and
*   `OC_SERVICE_ACCOUNT_ID=<acc-id>`.

The `storage-users` CLI tool uses the default address to establish the connection to the `gateway` service. If the connection fails, check your custom `gateway` service `GATEWAY_GRPC_ADDR` configuration and set the same address in `storage-users` `OC_GATEWAY_GRPC_ADDR` or `STORAGE_USERS_GATEWAY_GRPC_ADDR`.

### Manage Unfinished Uploads

<!-- referencing: [oCIS FS] clean up aborted uploads https://github.com/owncloud/ocis/issues/2622 -->

When using OpenCloud as user storage, a directory named `storage/users/uploads` can be found in the OpenCloud data folder. This is an intermediate directory based on [TUS](https://tus.io) which is an open protocol for resumable uploads. Each upload consists of a _blob_ and a _blob.info_ file. Note that the term _blob_ is just a placeholder.

*   **If an upload succeeds**, the _blob_ file will be moved to the target and the _blob.info_ file will be deleted.

*   **In case of incomplete uploads**, the _blob_ and _blob.info_ files will continue to receive data until either the upload succeeds in time or the upload expires based on the `STORAGE_USERS_UPLOAD_EXPIRATION` variable, see the table below for details.

*   **In case of expired uploads**, the _blob_ and _blob.info_ files will _not_ be removed automatically. Thus a lot of data can pile up over time wasting storage space.

*   **In the rare case of a failure**, after the upload succeeded but the file was not moved to its target location, which can happen when postprocessing fails, the situation is the same as with expired uploads.

Example cases for expired uploads:

*   When a user uploads a big file but the file exceeds the user-quota, the upload can't be moved to the target after it has finished. The file stays at the upload location until it is manually cleared.

*   If the bandwidth is limited and the file to transfer can't be transferred completely before the upload expiration time is reached, the file expires and can't be processed.

*   If the upload was technically successful, but the postprocessing step failed due to an internal error, it will not get further processed. See the procedure **Resume Post-Processing** in the `postprocessing` service documentation for details how to solve this.

The following commands are available to manage unfinished uploads:

```bash
opencloud storage-users uploads <command>
```

```plaintext
COMMANDS:
   delete-stale-nodes Delete (or revert) all nodes in processing state that are not referenced by any upload session
   sessions           Print a list of upload sessions
```

#### Sessions command

The `sessions` command is the entry point for listing, restarting and cleaning unfinished uploads.

**IMPORTANT**
> If not noted otherwise, commands with the `restart` option can also use the `resume` option. This changes behaviour slightly.
>
> * `restart`\
> When restarting an upload, all steps for open items will be restarted, except if otherwise defined.
> * `resume`\
> When resuming an upload, processing will continue unfinished items from their last completed step.

```bash
opencloud storage-users uploads sessions <commandoptions>
```

```
NAME:
   opencloud storage-users uploads sessions - Print a list of upload sessions

USAGE:
   opencloud storage-users uploads sessions [command options]

OPTIONS:
   --id value    filter sessions by upload session id (default: unset)
   --processing  filter sessions by processing status (default: unset)
   --expired     filter sessions by expired status (default: unset)
   --has-virus   filter sessions by virus scan result (default: unset)
   --json        output as json (default: false)
   --restart     send restart event for all listed sessions (default: false)
   --resume      send resume event for all listed sessions (default: false)
   --clean       remove uploads (default: false)
   --help, -h    show help
```

This will always output a list of uploads that match the criteria. See Command Examples section.

Some additional information on returned information:
  -  `Offset` is the amount of bytes the server has already received. If `Offset` == `Size` the server has reveived all bytes of the upload.
  -  `Processing` indicates if the uploaded file is currently going through postprocessing.
  -  `Scan Date` and `Scan Result` indicate the scanning status. If `Scan Date` is set and `Scan Result` is empty the file is not virus infected.

#### Command Examples

Command to list ongoing upload sessions

```bash
opencloud storage-users uploads sessions --expired=false --processing=false
```

```plaintext
Not expired sessions:
+--------------------------------------+--------------------------------------+---------+--------+------+--------------------------------------+--------------------------------------+---------------------------+------------+---------------------------+-----------------------+
|                Space                 |              Upload Id               |  Name   | Offset | Size |              Executant               |                Owner                 |          Expires          | Processing |         Scan Date         |      Scan Result      |
+--------------------------------------+--------------------------------------+---------+--------+------+--------------------------------------+--------------------------------------+---------------------------+------------+---------------------------+-----------------------+
| f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c | 5e387954-7313-4223-a904-bf996da6ec0b | foo.txt |      0 | 1234 | f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c | f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c | 2024-01-26T13:04:31+01:00 | false      | 2024-04-24T11:24:14+02:00 |   infected: virus A   |
| f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c | f066244d-97b2-48e7-a30d-b40fcb60cec6 | bar.txt |      0 | 4321 | f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c | f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c | 2024-01-26T13:18:47+01:00 | false      | 2024-04-24T14:38:29+02:00 |                       |
+--------------------------------------+--------------------------------------+---------+--------+------+--------------------------------------+--------------------------------------+---------------------------+------------+---------------------------+-----------------------+
```

The sessions command can also output json

```bash
opencloud storage-users uploads sessions --expired=false --processing=false --json
```

```json
{"id":"5e387954-7313-4223-a904-bf996da6ec0b","space":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","filename":"foo.txt","offset":0,"size":1234,"executant":{"idp":"https://cloud.opencloud.test","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"spaceowner":{"opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"expires":"2024-01-26T13:04:31+01:00","processing":false, "scanDate": "2024-04-24T11:24:14+02:00", "scanResult": "infected: virus A"}
{"id":"f066244d-97b2-48e7-a30d-b40fcb60cec6","space":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","filename":"bar.txt","offset":0,"size":4321,"executant":{"idp":"https://cloud.opencloud.test","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"spaceowner":{"opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"expires":"2024-01-26T13:18:47+01:00","processing":false, "scanDate": "2024-04-24T14:38:29+02:00", "scanResult": ""}
```

The sessions command can also clear and restart/resume uploads. The output is the same as if run without `--clean` or `--restart` flag.
Note: It is recommended to run to command first without the `--clean` (`--processing`) flag to double check which uploads get cleaned (restarted/resumed).

```bash
# cleans all expired uploads regardless of processing and virus state.
opencloud storage-users uploads sessions --expired=true --clean

# resumes all uploads that are processing and are not virus infected
opencloud storage-users uploads sessions --processing=false --has-virus=false --resume
```


#### Delete Stale Nodes command

This command allows to remove (or revert) nodes that are stale, meaning they are in postprocessing but their upload session is gone.
It will check all nodes that are in postprocessing and find those without an upload session. Then it will delete the node if there are no other versions. If there are other versions, it will instead revert the node to the previous version.
The command reads the decomposed storage directly, so it only finds nodes when the msgpack metadata backend (`.mpk` files) is used. The stale nodes are cleaned up by `DeleteRevision` events, which are consumed by the running `storage-users` service: it needs to be up for the cleanup to actually happen.

```bash
    opencloud storage-users uploads delete-stale-nodes <commandoptions>
```
```plaintext
Delete (or revert) all nodes in processing state that are not referenced by any upload session

Usage:
  opencloud storage-users uploads delete-stale-nodes [flags]

Flags:
      --dry-run          Only show what would be deleted without actually deleting (default true)
  -h, --help             help for delete-stale-nodes
      --spaceid string   Space ID to check for processing nodes (omit to check all spaces)
      --verbose          Enable verbose logging
```

#### Command Examples

Dry run to see what would be deleted (recommended first step)

```bash
opencloud storage-users uploads delete-stale-nodes
```

Set `--dry-run=false` to actually delete the stale nodes

```bash
opencloud storage-users uploads delete-stale-nodes --dry-run=false
```

Use `--verbose` to get more information about what is happening

```bash
opencloud storage-users uploads delete-stale-nodes --dry-run=false --verbose
```


### Check and Inspect the Blobstore

The `blobstore` command group verifies connectivity to the configured blobstore and inspects individual blobs. The `ocis`/`decomposed` and `s3ng`/`decomposeds3` storage drivers are supported.

```bash
opencloud storage-users blobstore <command>
```

```plaintext
COMMANDS:
   check  check blobstore connectivity via an upload/download/delete round-trip
   get    get a blob from the blobstore by ID
```

#### Check

Uploads a random blob, downloads and verifies it, then deletes it. All three steps must succeed.

```bash
opencloud storage-users blobstore check [flags]
```

`--blob-size` sets the random payload size and accepts values such as `64`, `1KB`, `1MB`, or `4MiB` (default: `64` bytes).

```bash
# Check the configured blobstore using the default payload
opencloud storage-users blobstore check

# Use a larger payload
opencloud storage-users blobstore check --blob-size=4MiB
```

#### Get

Downloads a blob to verify that it is readable. Identify the blob by passing the raw path from a log line with `--path`, or by setting `--blob-id` and `--space-id`.

```bash
opencloud storage-users blobstore get [flags]
```

For S3-backed drivers, `--blob-size` can specify the expected size when known. If it is wrong or omitted, the command retries once using the actual size returned by S3.

```bash
# Use a path from an s3ng/decomposeds3 log line
opencloud storage-users blobstore get --path="b19ec764-5398-458a-8ff1-1925bd906999/61/03/ab/c3/-b08a-4556-9937-2bf3065c1202"

# Use a full filesystem path from an ocis/decomposed log line
opencloud storage-users blobstore get --path="/var/lib/opencloud/storage/users/spaces/b1/9ec764-5398-458a-8ff1-1925bd906999/blobs/61/03/ab/c3/-b08a-4556-9937-2bf3065c1202"

# Or provide the space and blob IDs directly
opencloud storage-users blobstore get --space-id=b19ec764-5398-458a-8ff1-1925bd906999 --blob-id=6103abc3-b08a-4556-9937-2bf3065c1202
```

### Manage Trash-Bin Items

This command set provides commands to get an overview of trash-bin items, restore items and purge old items of `personal` spaces and `project` spaces (spaces that have been created manually). `trash-bin` commands require a `spaceID` as parameter.

```bash
opencloud storage-users trash-bin <command>
```

```plaintext
COMMANDS:
   purge-expired  Purge expired trash-bin items
   list           Print a list of all trash-bin items of a space.
   restore-all    Restore all trash-bin items for a space.
   restore        Restore a trash-bin item by ID.
```

#### Purge Expired

*   Purge all expired items from the trash-bin.
    ```bash
    opencloud storage-users trash-bin purge-expired
    ```

The behaviour of the `purge-expired` command can be configured by using the following environment variables.

*   `STORAGE_USERS_PURGE_TRASH_BIN_USER_ID`\
Used to obtain space trash-bin information and takes the system admin user as the default which is the `OC_ADMIN_USER_ID` but can be set individually. It should be noted, that the `OC_ADMIN_USER_ID` is only assigned automatically when using the single binary deployment and must be manually assigned in all other deployments. The command only considers spaces to which the assigned user has access and delete permission.

*   `STORAGE_USERS_PURGE_TRASH_BIN_PERSONAL_DELETE_BEFORE`\
Has a default value of `720h` which equals `30 days`. This means, the command will delete all files older than `30 days`. The value is human-readable. A value of `0` is equivalent to disable and prevents the deletion of `personal space` trash-bin files.

*   `STORAGE_USERS_PURGE_TRASH_BIN_PROJECT_DELETE_BEFORE`\
Has a default value of `720h` which equals `30 days`. This means, the command will delete all files older than `30 days`. The value is human-readable. A value of `0` is equivalent to disable and prevents the deletion of `project space` trash-bin files.

#### List and Restore Trash-Bins Items

Restoring is possible only to the original location. The personal or project `spaceID` is required for the items to be restored. To authenticate the CLI tool use:

```bash
OC_SERVICE_ACCOUNT_SECRET=<acc-secret>
OC_SERVICE_ACCOUNT_ID=<acc-id>
```

The `storage-users` CLI tool uses the default address to establish the connection to the `gateway` service. If the connection fails, check the `GATEWAY_GRPC_ADDR` configuration from your `gateway` service and set the same address to the `storage-users` variable `STORAGE_USERS_GATEWAY_GRPC_ADDR` or globally with `OC_GATEWAY_GRPC_ADDR`.

*   Export the gateway address if your configuration differs from the default
    ```bash
    export STORAGE_USERS_GATEWAY_GRPC_ADDR=127.0.0.1:9142
    ```

*   Print a list of all trash-bin items of a space
    ```bash
    opencloud storage-users trash-bin list [command options] ['spaceID' required]
    ```

The restore option defines the behavior for an item to be restored, when the item name already exists in the target space. Supported options are: `skip`, `replace` and `keep-both`. The default value is `skip`.

When the CLI tool restores the item with the `replace` option, the existing item will be moved to a trash-bin. When the CLI tool restores the item with the `keep-both` option and the designated item already exists, the name of the restored item will be changed by adding a numeric suffix in parentheses. The variable `STORAGE_USERS_CLI_MAX_ATTEMPTS_RENAME_FILE` defines a maximum number of attempts to rename an item.

* Restore all trash-bin items for a space
    ```bash
    opencloud storage-users trash-bin restore-all [command options] ['spaceID' required]
    ```

* Restore a trash-bin item by ID
    ```bash
    opencloud storage-users trash-bin restore [command options] ['spaceID' required] ['itemID' required]
    ```

### Clean Up Old Revisions

Every change to a file keeps its previous content as a revision (version). With `OC_REVISIONS_CLEANUP_STRATEGY`, the `storage-users` service deletes old revisions according to a cleanup strategy, whenever a file gets a new revision. The `decomposed`, `decomposeds3` and `posix` storage drivers are supported, with `posix` also for the revisions created from changes on disk. The default strategy `disabled` keeps all revisions and, as one would expect, does not perform any cleanup process for revisions at all.

The service only cleans up a file when it gets a new revision, so files that are never changed again keep their revisions. The `revisions cleanup` command cleans up the revisions of whole spaces, folders or files on demand, to cover those files as well, or to apply a different strategy once, e.g. after changing the strategy or to free storage.

```bash
opencloud storage-users revisions cleanup [flags]
```

The command uses the storage directly, with the configuration of the `storage-users` service, and can run while the service is running. It needs access to the storage and, for the `posix` driver, to NATS. With the single binary, NATS runs within the server: without a running server, the command gives up after 10 seconds. It needs no service account.

#### Strategies

A strategy is a comma-separated list of up to three parts.

Each `X` is either
* a number, which is a number of revisions, always counting from the newest one,
* or a duration with one of the units `w` (weeks), `d` (days), `h` (hours), `m` (minutes) or `s` (seconds).

Durations in days or weeks can use an uppercase unit, `D` or `W`, to count from midnight (server time) instead of from now: `1d` means the last 24 hours, `1D` since yesterday 00:00.

| Strategy | Meaning | Examples |
|---|---|---|
| `auto` | Thin out revisions the older they get: one per 10 seconds in the last minute, one per minute in the last hour, one per hour in the last 24 hours, one per day in the last 30 days, one per week beyond | `auto` |
| `X` | Delete everything beyond `X` | `10`, `30d`, `0` deletes all revisions |
| `X, auto` | Keep everything within `X`, thin out the older revisions | `14d, auto` |
| `auto, X` | Delete everything beyond `X`, thin out the rest | `auto, 90d`, `auto, 50` |
| `X1, X2` | Keep everything within `X1`, delete everything beyond `X2`; one must be a number, the other a duration | `5, 30d`, `7d, 10` |
| `X1, auto, X2` | Keep everything within `X1`, delete everything beyond `X2`, thin out in between | `7d, auto, 90d`, `1D, auto, 30d` |
| `disabled` | Never delete revisions | `disabled` |

Note that the age of a revision is the time its content was written, i.e. the modification time the file had before it was changed, which is also the date the version list shows for it. It is not the time the file was changed, which turned that content into a revision. A document that was unchanged for two months gets a revision that is two months old when it is changed, and e.g. `30d` deletes that revision right away. Combine a duration with a number of revisions, e.g. `5, 30d`, to always keep the newest ones.

#### Selecting the Files

Exactly one of these selects the files to clean up:

* `--space <id>`: all files of a space. It can be given several times. The space IDs are accepted as the APIs show them, e.g. `<storage id>$<space id>`.
* `--space <id> --path <path>`: only a file or a folder (recursively) within one space, e.g. `--path /Documents`.
* `--all` (`-a`): all files of all spaces, including disabled spaces, which are skipped.
* `--fs-path <path>` (`-p`): a file or a folder (recursively) by its path on disk. Only supported by the `posix` driver.

#### Options

The strategy and the settings of the background workers default to the configured ones and can be overridden for a run. Only `--queue-full` defaults to `block` without a timeout, whatever is configured: the command finds files faster than the workers clean them up, so dropping the files that find the queue full would skip most of them.

| Flag | Default | Meaning |
|---|---|---|
| `--strategy`, `-s` | `OC_REVISIONS_CLEANUP_STRATEGY` | The strategy, see above. The command fails if there is none. |
| `--async` | `OC_REVISIONS_CLEANUP_ASYNC` | Clean up files in background workers while looking for more. |
| `--workers` | `OC_REVISIONS_CLEANUP_ASYNC_WORKERS` | The number of background workers, `0` for half the available CPUs, at least 2. |
| `--queue-size` | `OC_REVISIONS_CLEANUP_ASYNC_QUEUE_SIZE` | The number of files that can wait for a worker, `0` for 64 per worker. |
| `--queue-full` | `block` | `block` waits for room in a full queue, `drop` skips the file. |
| `--queue-full-timeout` | `0` | With `block`, skip a file after waiting this long for room, e.g. `10s`. `0` waits as long as it takes. |
| `--dry-run`, `-n` | off | Delete nothing, only print what would be deleted. |
| `--verbose`, `-v` | off | Print every file and every revision, with what was decided about it and why. |
| `--json`, `-j` | off | Print JSON objects, one per line, instead of text. |
| `--timeout`, `-t` | none | Stop the cleanup gracefully after this time, e.g. `30m` or `2h`. |
| `--shuffle` | off | Clean up the spaces, and the files within each folder, in a random order. |

By default, the command prints a line for every file whose revisions it deleted, that it could not clean up or that it skipped, with the space ID and the path of the file within the space, and a summary:

```plaintext
25b69780-5f39-43be-a7ac-a9b9e9fe4230 /Documents/report.odt: deleted 3 of 12 revisions (4.2 MiB)
25b69780-5f39-43be-a7ac-a9b9e9fe4230 /Photos/holiday.jpg: skipped: an upload to the file is still being processed
1520 files in 3 spaces: deleted 3 revisions (4.2 MiB) of 1 file, 1 skipped, 0 failed
```

With `--verbose`, every revision is listed with the decision (`deleted`, `kept` or `protected`), its time, its size and the reason:

```plaintext
25b69780-5f39-43be-a7ac-a9b9e9fe4230 /dir1/file1 (node 43ca5408-c77b-4d9a-8a3b-d4470b1253b8): would delete 1 of 3 revisions (1.2 KiB)
  protected 2026-10-15T13:24:35+02:00   1.2 KiB  protected by "1": one of the 1 newest
  kept      2026-10-13T15:24:35+02:00   1.2 KiB  kept by "30d": not older than 2026-09-15T15:24:35+02:00
  deleted   2026-03-29T15:24:35+02:00   1.2 KiB  deleted by "30d": older than 2026-09-15T15:24:35+02:00
```

With `--json`, every file is an object of the `type` `file`, with the revisions in `details` when `--verbose` is set, and the last line is an object of the `type` `summary`. The sizes are in bytes. The same run as above:

```json
{"type":"file","space_id":"25b69780-5f39-43be-a7ac-a9b9e9fe4230","node_id":"8a1c0e2f-6b4d-4f7a-9c3e-2d5b7f1a0c94","path":"/Documents/report.odt","dry_run":false,"revisions":12,"deleted":3,"deleted_bytes":4404019,"failed":0}
{"type":"file","space_id":"25b69780-5f39-43be-a7ac-a9b9e9fe4230","node_id":"f3e9d2c1-0a7b-4c8e-b6d5-9e4f1a2b3c7d","path":"/Photos/holiday.jpg","dry_run":false,"skipped":"an upload to the file is still being processed","revisions":0,"deleted":0,"deleted_bytes":0,"failed":0}
{"type":"summary","dry_run":false,"spaces":3,"files":1520,"cleaned":1,"deleted":3,"deleted_bytes":4404019,"skipped":1,"failed":0}
```

With `--verbose` as well, a file object lists its revisions in `details`, here from the dry run above:

```json
{"type":"file","space_id":"25b69780-5f39-43be-a7ac-a9b9e9fe4230","node_id":"43ca5408-c77b-4d9a-8a3b-d4470b1253b8","path":"/dir1/file1","dry_run":true,"revisions":3,"deleted":1,"deleted_bytes":1234,"failed":0,"details":[{"key":"43ca5408-c77b-4d9a-8a3b-d4470b1253b8.REV.2026-10-15T11:24:35.118424081Z","time":"2026-10-15T11:24:35.118424081Z","size":1234,"decision":"protected","reason":"protected by \"1\": one of the 1 newest"},{"key":"43ca5408-c77b-4d9a-8a3b-d4470b1253b8.REV.2026-10-13T13:24:35.118424081Z","time":"2026-10-13T13:24:35.118424081Z","size":1234,"decision":"kept","reason":"kept by \"30d\": not older than 2026-09-15T15:24:35+02:00"},{"key":"43ca5408-c77b-4d9a-8a3b-d4470b1253b8.REV.2026-03-29T13:24:35.118424081Z","time":"2026-03-29T13:24:35.118424081Z","size":1234,"decision":"deleted","reason":"deleted by \"30d\": older than 2026-09-15T15:24:35+02:00"}]}
```

A file that failed has an `error`, and so has a revision in `details` that could not be deleted. When `--timeout` stopped the cleanup, the summary has a `stopped` field with the reason.

Things to keep in mind:

*   **A dry run deletes nothing**, but it still takes the locks of the files, which can create lock files.
*   **`--timeout` stops the cleanup gracefully:** the files being cleaned up are finished, the files still waiting for a worker are skipped. It can't interrupt a file that waits for a lock held by another process, though, so the command can run longer than the timeout. Ctrl-C stops the cleanup the same way.
*   **Every run starts over from the beginning.** For regular runs with a timeout, e.g. by a cron job, add `--shuffle`, so that every run starts in a different place and the runs eventually reach all files, also in spaces too large for a single run.

The exit code is `0` when the cleanup is done, `1` if the cleanup or any file failed, and `2` if `--timeout` stopped the cleanup before all files were cleaned up, and nothing failed.

#### Command Examples

```bash
# show what a strategy would delete in all spaces, with the reason for every revision
opencloud storage-users revisions cleanup --all --strategy "auto, 90d" --dry-run --verbose

# keep only the 10 newest revisions of the files in a folder of a space
opencloud storage-users revisions cleanup --space 25b69780-5f39-43be-a7ac-a9b9e9fe4230 --path /Documents --strategy 10

# clean up a folder by its path on disk, with the posix driver
opencloud storage-users revisions cleanup --fs-path /var/lib/opencloud/storage/users/projects/25b69780-5f39-43be-a7ac-a9b9e9fe4230/Documents

# a nightly cron job: the configured strategy, at most one hour, in a random order
opencloud storage-users revisions cleanup --all --timeout 1h --shuffle

# the same with the short flags
opencloud storage-users revisions cleanup -a -t 1h --shuffle
```

## Caching

The `storage-users` service caches stat, metadata and uuids of files and folders via the configured store in `STORAGE_USERS_FILEMETADATA_CACHE_STORE` and `STORAGE_USERS_ID_CACHE_STORE`. Possible stores are:
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
