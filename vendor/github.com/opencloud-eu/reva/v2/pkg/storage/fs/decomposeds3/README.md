# S3 customer-provided encryption keys (SSE-C)

Both `decomposeds3` and the legacy `s3ng` driver accept an optional
`s3.sse_c_key` alongside the bucket configuration:

```toml
[grpc.services.storageprovider.drivers.decomposeds3]
"s3.endpoint" = "https://s3.example.com"
"s3.region" = "us-east-1"
"s3.bucket" = "files"
"s3.access_key" = "access-key"
"s3.secret_key" = "secret-key"
"s3.sse_c_key" = "<base64-encoded 32-byte encryption key>"
```

Generate a random key with `openssl rand -base64 32`. Keep it in secret
configuration and retain a secure backup: the same key is needed for subsequent
reads, including metadata requests and historical file versions. It is separate
from the S3 access credentials. The endpoint must use HTTPS and the S3 service
must permit SSE-C uploads.

An empty setting preserves the existing behavior. A configured key applies to
both upload paths, including multipart uploads, and to downloads and their
metadata requests. The SDK generates AES256, key and key-MD5 headers. Listing
and deleting objects do not send customer encryption headers. Incorrect or
missing keys fail without an unencrypted retry.

Setting or changing the key does not migrate or re-encrypt existing objects.
Before enabling SSE-C on an existing bucket, plan its data migration separately.
Key rotation and multiple keys within one bucket configuration are not provided.
