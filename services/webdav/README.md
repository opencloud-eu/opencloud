# Webdav

The webdav service, like the [frontend](../frontend) service, provides a HTTP API following the webdav protocol. It receives HTTP calls from requestors like clients and issues gRPC calls to other services executing these requests. After the called service has finished the request, the webdav service will render their responses in `xml` and sends them back to the requestor.

## Endpoints Overview

Currently, the webdav service handles request for two functionalities, which are `Thumbnails` and `Search`.

### Thumbnails

The webdav service provides various `GET` endpoints to get the thumbnails of a file in authenticated and unauthenticated contexts. It also provides thumbnails for spaces on different endpoints.

Generated thumbnails are cached by the webdav service itself. The cache backend defaults to `file`, storing entries under `$OC_BASE_DATA_PATH/thumbnails/files` (override with `WEBDAV_THUMBNAIL_CACHE_BACKEND` and `WEBDAV_THUMBNAIL_CACHE_DIR`). Use the `s3` backend when running multiple instances behind a load balancer so they share one cache.

#### Thumbnail Query String Parameters

Clients can request thumbnail previews for files by adding `?preview=1` to the file URL. Requests for files with no thumbnail available respond with HTTP status `404`.

The following query parameters are supported:

| Parameter | Required | Default Value                                        | Description                                                                     |
|-----------|----------|------------------------------------------------------|---------------------------------------------------------------------------------|
| preview   | YES      | 1                                                    | generates preview                                                               |
| x         | YES      | first x-value configured in `WEBDAV_THUMBNAIL_RESOLUTIONS` | horizontal target size                                                  |
| y         | YES      | first y-value configured in `WEBDAV_THUMBNAIL_RESOLUTIONS` | vertical target size                                                  |
| scalingup | NO       | 0                                                    | accepted for compatibility but ignored; the generator never upscales via this flag |
| a         | NO       | 1                                                    | aspect ratio (legacy; only honored when no explicit `processor` is given)      |
| c         | NO       | Caching string                                       | Clients should send the etag, so they get a fresh thumbnail after a file change |
| processor | NO       | `resize` for gifs and `thumbnail` for all others     | preferred thumbnail processor                                                   |

#### Thumbnail Resolution

Various resolutions can be defined via `WEBDAV_THUMBNAIL_RESOLUTIONS`. A requestor can request any arbitrary resolution and the webdav service will use the one closest to the requested resolution. If more than one resolution is required, each resolution must be requested individually.

Example:

Requested: 18x12\
Available: 30x20, 15x10, 9x6\
Returned: 15x10

#### Thumbnail Processors

Normally, an image might get cropped when creating a preview, depending on the aspect ratio of the original image. This can have negative impacts on previews as only a part of the image will be shown. When using an _optional_ processor in the request, cropping can be avoided by defining on how the preview image generation will be done. The following processors are available:

*   `resize` resizes the image to the specified width and height and returns the transformed image (distorts to the exact box).
*   `fit-in` scales the image down to fit within the specified maximum width and height, preserving aspect ratio and never upscaling (letterboxed).
*   `fill`: creates an image with the specified dimensions and fills it with the scaled source image. To achieve the correct aspect ratio without stretching, the source image will be cropped.

The following are **legacy aliases** kept for compatibility. They map onto the processors above but **ignore the `a` and `scalingup` query parameters**; callers that need aspect or upscaling control should use the imagor names directly (`fill`, `fit-in`, `resize`):

*   `thumbnail` is a legacy alias for `fill` (center-crop to the exact box, may upscale).
*   `fit` is a legacy alias for `fit-in`.

To apply one of those, a query parameter has to be added to the request, like `?processor=fit-in`. If no processor is given, the default behaviour applies which is `resize` for gifs and `thumbnail` (i.e. `fill`) for all others. When an explicit processor overrides a contradictory legacy `a`, the thumbnail response includes the header `X-OpenCloud-Thumbnail-Aspect-Ignored` so developers can fix their client to send a consistent request.

### Search

The webdav service provides access to the search functionality. It offers multiple `REPORT` endpoints for getting search results.

See the [search](https://github.com/opencloud-eu/opencloud/tree/main/services/search) service for more details about search functionality.

## Scalability

The webdav service persists generated thumbnails to its thumbnail cache (file backend by default). When running multiple instances behind a load balancer, point `WEBDAV_THUMBNAIL_CACHE_BACKEND` at a shared `s3` bucket so all instances read and write the same cache; otherwise each instance keeps its own on-disk cache.
