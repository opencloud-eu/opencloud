# opencloud

The opencloud package contains the OpenCloud runtime and the commands for the OpenCloud CLI.

## Service Registry

This package also configures the service registry which will be used to look up the service addresses.

Available registries are:

-   nats-js-kv (default)
-   memory

To configure which registry to use, you have to set the environment variable `MICRO_REGISTRY`, and for all except `memory` you also have to set the registry address via `MICRO_REGISTRY_ADDRESS`.

## Memory limits

OpenCloud will automatically set the go native `GOMEMLIMIT` to `0.9`. To disable the limit set `AUTOMEMEMLIMIT=off`. For more information take a look at the official [Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide).

## CLI Commands

The opencloud package offers a variety of cli commands to monitor or repair OpenCloud installations. All these commands have a common mandatory parameter: `--basePath` (or `-p`) which needs to point to a storage provider. Example paths are:

```bash
.opencloud/storage/users          # bare metal installation
/var/tmp/opencloud/storage/users  # docker installation
...
```

These paths can vary depending on your OpenCloud installation.

### Backup CLI

The backup command allows inspecting the consistency of an OpenCloud storage:

```bash
opencloud backup consistency -p /base/path/storage/users
```

This will check the consistency of the storage and output a list of inconsistencies. Inconsistencies can be:

* **Orphaned Blobs**\
A blob in the blobstore that is not referenced by any file metadata.
* **Missing Blobs**\
A blob referenced by file metadata that is not present in the blobstore.
* **Missing Nodes**\
A node that is referenced by a symlink but doesn't exist.
* **Missing Link**\
A node that is not referenced by any symlink but should be.
* **Missing Files**\
A node that is missing essential files (such as the `.mpk` metadata file).
* **Missing/Malformed Metadata**\
A node that doesn't have any (or malformed) metadata.

This command provides additional options:

* `-b` / `--blobstore`\
Allows specifying the blobstore to use. Defaults to `decomposed`. Empty blobs will not be checked. Can also be switched to `decomposeds3`, but needs addtional envvar configuration (see the `storage-users` service for more details).
* `--fail`\
Exits with non-zero exit code if inconsistencies are found. Useful for automation.

### Cleanup Orphaned Shares

When a shared space or directory got deleted, use the `shares cleanup` command to remove those share orphans. This can't be done automatically at the moment.

```bash
opencloud shares cleanup
```

### Revisions CLI

The revisions command allows removing the revisions of files in the storage.

```bash
opencloud revisions purge -p /base/path/storage/users
```

It takes the `--resource-id` (or `--r`) parameter which specify the scope of the command:

* An empty string (default) removes all revisions from all spaces.
* A spaceID (like `d419032c-65b9-4f4e-b1e4-0c69a946181d\$44b5a63b-540c-4002-a674-0e9c833bbe49`) removes all revisions in that space.
* A resourceID (e.g. `d419032c-65b9-4f4e-b1e4-0c69a946181d\$44b5a63b-540c-4002-a674-0e9c833bbe49\!e8a73d49-2e00-4322-9f34-9d7f178577b2`) removes all revisions from that specific file.

This command provides additional options:

* `--dry-run` (default: `true`)\
Do not remove any revisions but print the revisions that would be removed.
* `-b` / `--blobstore`\
Allows specifying the blobstore to use. Defaults to `decomposed`. Can be switched to `decomposeds3` but needs addtional envvar configuration (see the `storage-users` service for more details).
* `-v` / `--verbose`\
Prints additional information about the revisions that are removed.
* `--glob-mechanism` (default: `glob`\
(advanced) Allows specifying the mechanism to use for globbing. Can be `glob`, `list` or `workers`. In most cases the default `glob` does not need to be changed. If large spaces need to be purged, `list` or `workers` can be used to improve performance at the cost of higher cpu and ram usage. `list` will spawn 10 threads that list folder contents in parallel. `workers` will use a special globbing mechanism and multiple threads to achieve the best performance for the highest cost.

### Trash CLI

The trash cli allows removing empty folders from the trashbin. This should be used to speed up trash bin operations.

```bash
opencloud trash purge-empty-dirs -p /base/path/storage/users
```

This command provides additional options:

* `--dry-run` (default: `true`)\
Do not remove any empty folders but print the empty folders that would be removed.

### List Unified Roles

This command simplifies the process of finding out which UID belongs to which role. The command is:

```bash
opencloud graph list-unified-roles
```

The output of this command includes the following information for each role:

* `Label`\
  A short, stable name of the role.
* `UID`\
  The unique identifier of the role.
* `Enabled`\
  Whether the role is enabled or not.
* `Description`\
  A short description of the role.
* `Condition`
* `Allowed Resource Action`

**Example output (shortned)**

```bash
+----------------------------+--------------------------------------+----------+--------------------------------+--------------------------------+------------------------------------------+
|           LABEL            |                 UID                  | ENABLED  |          DESCRIPTION           |           CONDITION            |         ALLOWED RESOURCE ACTIONS         |
+----------------------------+--------------------------------------+----------+--------------------------------+--------------------------------+------------------------------------------+
| Viewer                     | b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5 | enabled  | View and download.             | exists @Resource.File          | libre.graph/driveItem/path/read          |
|                            |                                      |          |                                | exists @Resource.Folder        | libre.graph/driveItem/quota/read         |
|                            |                                      |          |                                | exists @Resource.File &&       | libre.graph/driveItem/content/read       |
|                            |                                      |          |                                | @Subject.UserType=="Federated" | libre.graph/driveItem/children/read      |
|                            |                                      |          |                                | exists @Resource.Folder &&     | libre.graph/driveItem/deleted/read       |
|                            |                                      |          |                                | @Subject.UserType=="Federated" | libre.graph/driveItem/basic/read         |
+----------------------------+--------------------------------------+----------+--------------------------------+--------------------------------+------------------------------------------+
```
