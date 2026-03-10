Enhancement: Add blobstore CLI commands to storage-users service

Added two new CLI commands under `ocis storage-users blobstore` to help
operators verify and inspect the configured blobstore without needing
direct access to the underlying storage system.

`blobstore check` performs a full upload/download/delete round-trip using
a random payload. The payload size is configurable via `--blob-size` and
accepts human-readable values such as `64`, `1KB` or `4MiB` (default: 64 bytes).

`blobstore get` downloads a specific blob by its ID to verify it is
readable. The blob can be identified either with `--blob-id` and
`--space-id`, or by passing the raw blob path from a log line directly
via `--path`. Both the s3ng key format
(`<spaceID>/<pathified_blobID>`) and the ocis filesystem path format
(`…/spaces/<pathified_spaceID>/blobs/<pathified_blobID>`) are accepted.
When using the s3ng driver without a known blob size, an automatic retry
with the actual size is performed on a size mismatch.

Both commands read the existing service configuration, so they always
target the same blobstore as the running service. Only the `ocis` and
`s3ng` storage drivers are supported.

https://github.com/owncloud/ocis/pull/12102
