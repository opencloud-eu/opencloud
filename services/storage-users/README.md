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
