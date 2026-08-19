# Thumbnails

The thumbnails service provides methods to generate thumbnails for various files and resolutions based on requests. It retrieves the sources at the location where the user files are stored and saves the thumbnails where system files are stored. Those locations have defaults but can be manually defined via environment variables.

## File Locations Overview

The relevant environment variables defining file locations are:

-   (1) `OC_BASE_DATA_PATH`
-   (2) `STORAGE_USERS_DECOMPOSED_ROOT`
-   (3) `THUMBNAILS_FILESYSTEMSTORAGE_ROOT`

(1) ... Having a default set by the OpenCloud code, but if defined, used as base path for other services.
(2) ... Source files, defaults to (1) plus path component, but can be freely defined if required.
(3) ... Target files, defaults to (1) plus path component, but can be freely defined if required.

For details and defaults for these environment variables see the OpenCloud admin documentation.

## Thumbnail Location

It may be beneficial to define the location of the thumbnails to be other than the default (with system files). This is due the fact that storing thumbnails can consume a lot of space over time which not necessarily needs to reside on the same partition or mount or expensive drives.

## Thumbnail Source File Types

Thumbnails can be generated from the following source file types:

-   png
-   jpg
-   gif
-   tiff
-   bmp
-   txt
-   camera raw files: nef, cr2, pef, arw, sr2, srf, dng

For camera raw files the thumbnail is generated from the JPEG preview the camera embedded in the file, the raw sensor data itself is not developed. Raw files without an embedded preview cannot be thumbnailed. Preview extraction is done by an Apache Tika server, so raw thumbnails require `THUMBNAILS_PREPROCESSOR_TIKA_TIKA_URL` (or the shared `OC_TIKA_URL`, which the search service reads too) to be set.

The thumbnail service retrieves source files using the information provided by the backend. The Linux backend identifies source files usually based on the extension.

If a file type was not properly assigned or the type identification failed, thumbnail generation will fail and an error will be logged.

## Thumbnail Target File Types

Thumbnails can either be generated as `png`, `jpg` or `gif` files. These types are hardcoded and no other types can be requested. A requestor, like another service or a client, can request one of the available types to be generated. If more than one type is required, each type must be requested individually.

## Thumbnail Query String Parameters

Clients can request thumbnail previews for files by adding `?preview=1` to the file URL. Requests for files with no thumbnail available respond with HTTP status `404`.

The following query parameters are supported:

| Parameter | Required | Default Value                                        | Description                                                                     |
|-----------|----------|------------------------------------------------------|---------------------------------------------------------------------------------|
| preview   | YES      | 1                                                    | generates preview                                                               |
| x         | YES      | first x-value configured in `THUMBNAILS_RESOLUTIONS` | horizontal target size                                                          |
| y         | YES      | first y-value configured in `THUMBNAILS_RESOLUTIONS` | vertical target size                                                            |
| scalingup | NO       | 0                                                    | prevents up-scaling of small images                                             |
| a         | NO       | 1                                                    | aspect ratio                                                                    |
| c         | NO       | Caching string                                       | Clients should send the etag, so they get a fresh thumbnail after a file change |
| processor | NO       | `resize` for gifs and `thumbnail` for all others     | preferred thumbnail processor                                                   |

## Thumbnail Resolution

Various resolutions can be defined via `THUMBNAILS_RESOLUTIONS`. A requestor can request any arbitrary resolution and the thumbnail service will use the one closest to the requested resolution. If more than one resolution is required, each resolution must be requested individually.

Example:

Requested: 18x12\
Available: 30x20, 15x10, 9x6\
Returned: 15x10

## Thumbnail Processors

Normally, an image might get cropped when creating a preview, depending on the aspect ratio of the original image. This can have negative
impacts on previews as only a part of the image will be shown. When using an _optional_ processor in the request, cropping can be avoided by defining on how the preview image generation will be done. The following processors are available:

*   `resize` resizes the image to the specified width and height and returns the transformed image. If one of width or height is 0, the image aspect ratio is preserved.
*   `fit` scales down the image to fit the specified maximum width and height and returns the transformed image.
*   `fill`: creates an image with the specified dimensions and fills it with the scaled source image. To achieve the correct aspect ratio without stretching, the source image will be cropped.
*   `thumbnail` scales the image up or down, crops it to the specified width and height and returns the transformed image.

To apply one of those, a query parameter has to be added to the request, like `?processor=fit`. If no query parameter or processor is added, the default behaviour applies which is `resize` for gifs and `thumbnail` for all others.

## Deleting Thumbnails

As of now, there is no automated thumbnail deletion. This is especially true when a source file gets deleted or moved. This situation will be solved at a later stage. For the time being, if you run short on physical thumbnails space, you have to manually delete the thumbnail store to free space. Thumbnails will then be recreated on request.

## Memory Considerations

Since source files need to be loaded into memory when generating thumbnails, large source files could potentially crash this service if there is insufficient memory available. For bigger instances when using container orchestration deployment methods, this service can be dedicated to its own server(s) with more memory.
To have more control over memory (and CPU) consumption the maximum number of concurrent requests can be limited by setting the environment variable `THUMBNAILS_MAX_CONCURRENT_REQUESTS`. The default value is 0 which does not apply any restrictions to the number of concurrent requests. As soon as the number of concurrent requests is reached any further request will be responded with `429/Too Many Requests` and the client can retry at a later point in time.

## Thumbnails and SecureView

If a resource is shared using SecureView, the share reciever will get a 403 (forbidden) response when requesting a thumbnail. The requesting client needs to decide what to show and usually a placeholder thumbnail is used.

## Using libvips for Thumbnail Generation

To improve performance and to support a wider range of images formats, the thumbnails service is able to utilize the [libvips library](https://www.libvips.org/) for thumbnail generation. Support for libvips needs to be
enabled at buildtime and has a couple of implications:

*  With libvips support enabled, it is not possible to create a statically linked OpenCloud binary.
*  Therefore, the libvips shared libraries need to be available at runtime in the same release that was used to build the OpenCloud binary.
*  When using the OpenCloud docker images, the libvips shared libraries are included in the image and are correctly embedded.

Support of libvips is disabled by default. To enable it, make sure libvips and its buildtime dependencies are installed in your build environment. For macOS users, add the build time dependencies via:

```shell
brew install vips pkg-config
export PKG_CONFIG_PATH="/usr/local/opt/libffi/lib/pkgconfig"
```

Then you just need to set the `ENABLE_VIPS` variable on the `make` command:

```shell
make -C opencloud build ENABLE_VIPS=1
```

Or include the `enable_vips` build tag in the `go build` command:

```shell
go build -tags enable_vips -o opencloud -o bin/opencloud ./cmd/opencloud
```

When building a docker image using the Dockerfile in the top-level directory of OpenCloud, libvips support is enabled and the libvips shared libraries are included
in the resulting docker image.

