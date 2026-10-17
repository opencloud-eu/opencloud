# Webdav

The webdav service, like the [frontend](../frontend) service, provides a HTTP API following the webdav protocol. It receives HTTP calls from requestors like clients and issues gRPC calls to other services executing these requests. After the called service has finished the request, the webdav service will render their responses in `xml` and sends them back to the requestor.

## Endpoints Overview

Currently, the webdav service handles request for two functionalities, which are `Thumbnails` and `Search`.

### Thumbnails

The webdav service provides various `GET` endpoints to get the thumbnails of a file in authenticated and unauthenticated contexts. It also provides thumbnails for spaces on different endpoints.

Generated thumbnails are cached by the webdav service itself. The cache backend defaults to `file`, storing entries under `$OC_BASE_DATA_PATH/thumbnails/files` (override with `WEBDAV_THUMBNAIL_CACHE_BACKEND` and `WEBDAV_THUMBNAIL_CACHE_DIR`). Use the `s3` backend when running multiple instances behind a load balancer so they share one cache. A backend nobody can serve, an unknown name or an `s3` backend without bucket, region and credentials, fails at startup instead of serving without a cache, as long as previews are enabled. A service that is not AWS needs `WEBDAV_THUMBNAIL_CACHE_S3_ENDPOINT` too, and it is always addressed over https. The `memory` backend is bounded and drops its oldest entries; `none` disables caching on purpose.

#### Thumbnail Query String Parameters

Clients can request thumbnail previews for files by adding `?preview=1` to the file URL. Requests for files with no thumbnail available respond with HTTP status `404`.

The following query parameters are supported:

| Parameter | Required | Default Value                                              | Description                                                                     |
| --------- | -------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------- |
| preview   | YES      | 1                                                          | generates preview                                                               |
| x         | NO       | 32, when y is absent too                                   | horizontal target size, absent leaves the width unconstrained, `0` is an error  |
| y         | NO       | 32, when x is absent too                                   | vertical target size, absent leaves the height unconstrained, `0` is an error   |
| scalingup | NO       | 0                                                          | accepted for compatibility but ignored; not forwarded to the generator          |
| a         | NO       | none, the chain below decides                              | aspect ratio (legacy; only honored when no explicit `processor` is given)       |
| c         | NO       | Caching string                                             | Clients should send the etag, so they get a fresh thumbnail after a file change |
| processor | NO       | none, the chain below decides                              | preferred thumbnail processor                                                   |

#### Thumbnail Resolution

`WEBDAV_THUMBNAIL_RESOLUTIONS` is a ladder of edge lengths, not a list of boxes. Each axis of a request is rounded up to the next length on the ladder, on its own, so a thumbnail never comes back in a different orientation than it was asked for. The largest length is the cap: a request above it, and an axis left unconstrained, get that one.

Example with the default ladder (16, 32, 64, 128, 192, 256, 320, 384, 448, 512, 640, 768, 1024, 1280, 1920, 2560, 3840, 7680):

Requested: 18x12\
Returned: 32x16

Entries written as `WidthxHeight` are still accepted and contribute both of their sides to the ladder, so a configuration written for the old box list keeps working.

Rounding onto the ladder is also what the cache is keyed on, so requests that produce the same image share one entry.

The response is therefore not exactly the requested box: rounding up makes it larger, and an aspect-preserving operation makes it smaller on one axis. A client has to scale what it gets instead of trusting its own numbers.

#### Thumbnail Processors

Normally, an image might get cropped when creating a preview, depending on the aspect ratio of the original image. This can have negative impacts on previews as only a part of the image will be shown. When using an _optional_ processor in the request, cropping can be avoided by defining on how the preview image generation will be done. The following processors are available:

- `resize` resizes the image to the specified width and height and returns the transformed image (distorts to the exact box).
- `fit-in` scales the image down to fit within the specified maximum width and height, preserving aspect ratio and never upscaling. The result is smaller than the box on one axis, it is not padded to it.
- `fill`: creates an image with the specified dimensions and fills it with the scaled source image. To achieve the correct aspect ratio without stretching, the source image will be cropped.

The following are **legacy aliases** kept for compatibility:

- `thumbnail` is a legacy alias for `fill` (center-crop to the exact box, may upscale).
- `fit` is a legacy alias for `fit-in`.

Naming one of them decides the behavior on its own, legacy alias or imagor name: `a` is overruled and `scalingup` is ignored either way. A request that wants `a` to decide has to leave `processor` out, a name nobody knows is treated the same way.

To apply one of those, a query parameter has to be added to the request, like `?processor=fit-in`.

Without a processor three things decide, in this order. An unconstrained axis is always fitted, because there is no box to crop to, so `?y=512` needs nothing else. Then `a` decides when the client sent it: `a=1` fits in, `a=0` fills. Otherwise the source type decides, `resize` for gifs and `fill` for everything else, which is what a request without `a` has always got.

When what a request resolves to contradicts an `a` the client sent, the response carries the header `X-OpenCloud-Thumbnail-Aspect-Ignored` so developers can fix their client. A request that does not send `a` never gets the header.

#### Thumbnail Generator

The thumbnails themselves are made by a generator the webdav service POSTs to, configured with `WEBDAV_THUMBNAIL_GENERATOR_URL` (and `WEBDAV_THUMBNAIL_GENERATOR_TIMEOUT`). The built-in thumbnails service speaks the imagor image endpoint, so a real [imagor](https://imagor.net) can be put in its place without changing anything else. A source the generator cannot read answers `404`, one that is too large `403`, and `WEBDAV_THUMBNAIL_MAX_INPUT_IMAGE_FILE_SIZE` caps what is sent at all. A generator that refuses because it is busy answers `429` with its own `Retry-After`, so a client can come back instead of treating it as a missing preview.

Three settings say what is made of what. They are merged onto the built-in defaults, so naming one type does not take the answer away from the others:

| Environment variable                 | Description                                                                                                                                                  |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `OC_THUMBNAIL_SOURCE_TYPES`          | Types a preview is announced and attempted for, e.g. `image/vnd.microsoft.icon` for a generator that reads icons. The webdav, graph and ocdav side all read this one name |
| `OC_THUMBNAIL_TYPE_MAPPING`          | The type a thumbnail is returned in, per source type, as `mime:mime` entries, e.g. `image/vnd.microsoft.icon:image/png`                                      |
| `OC_THUMBNAIL_GENERATOR_FORMATS`     | The formats the generator writes, as `mime:format` entries, e.g. `image/webp:webp`. The format is the name its own `format(...)` filter uses                 |

Reading them as a chain: a source type is mapped to the type its thumbnail is returned in, and that type is asked for by the name the generator knows it under. An icon becomes `image/png`, which the generator is asked for as `format(png)`. A type nobody mapped becomes a JPEG, and a mapping onto a type the generator has no format name for is refused at startup, as long as previews are enabled.

The defaults return `image/png` for png and icon sources, keep `image/gif` for a gif and return `image/jpeg` for everything else, so a client must read the response's `Content-Type` instead of assuming the type it got before.

Previews can be switched off entirely with `OC_DISABLE_PREVIEWS`. Search keeps working, every thumbnail path answers `404`, and nothing here is read or checked: a typo in the resolutions, the cache backend or the type mapping surfaces when previews are switched back on, not before.

In the single binary nothing has to be set: the generator URL defaults to the built-in thumbnails service. A deployment that runs the services in separate containers has to point `WEBDAV_THUMBNAIL_GENERATOR_URL` at its thumbnails service or at an imagor, because `127.0.0.1` no longer reaches it. The settings that used to live on the thumbnails service now belong to webdav, `THUMBNAILS_RESOLUTIONS` and `THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE` keep working there, and the thumbnail cache moves with them: the directory behind `THUMBNAILS_FILESYSTEMSTORAGE_ROOT`, now `WEBDAV_THUMBNAIL_CACHE_DIR`, has to be reachable by the webdav container.

### Search

The webdav service provides access to the search functionality. It offers multiple `REPORT` endpoints for getting search results.

See the [search](https://github.com/opencloud-eu/opencloud/tree/main/services/search) service for more details about search functionality.

## Scalability

The webdav service persists generated thumbnails to its thumbnail cache (file backend by default). When running multiple instances behind a load balancer, point `WEBDAV_THUMBNAIL_CACHE_BACKEND` at a shared `s3` bucket so all instances read and write the same cache; otherwise each instance keeps its own on-disk cache.
