Enhancement: Add blobstore CLI commands to storage-users service

Added blobstore CLI commands to verify connectivity to the configured
blobstore and inspect individual blobs. The commands support the
`ocis`/`decomposed` and `s3ng`/`decomposeds3` storage drivers.

`blobstore check` performs an upload/download/delete round-trip with a
random payload. The payload size is configurable via `--blob-size`.

`blobstore get` downloads an individual blob, identified by its ID and
space ID or by a raw blobstore path from a log line. S3-backed drivers
retry once with the actual blob size when the requested size does not
match.

https://github.com/owncloud/ocis/pull/12102
