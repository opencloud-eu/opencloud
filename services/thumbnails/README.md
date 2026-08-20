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

Builds with libvips support additionally handle `webp`, see [Using libvips for Thumbnail Generation](#using-libvips-for-thumbnail-generation), and can optionally be extended with `heic`/`heif` and `avif` at runtime, see [HEIF, HEIC and AVIF Images](#heif-heic-and-avif-images).

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

### HEIF, HEIC and AVIF Images

Photos taken with recent Apple devices are usually stored as HEIC (which is a HEIF container holding HEVC coded image data). OpenCloud does not ship a HEVC decoder by default due to legal reasons: HEVC is covered by patent pools that require licensing, when distributing/shipping a decoder as part of the OpenCloud binaries or images.

However, the `libvips` library, that OpenCloud (optionally) uses to generate thumbnails can handle these files easily – if it is extended with an additional HEIF package.

To add HEIF-support based on the official OpenCloud Docker images, which are Alpine-based, one tiny package has to be added to the image: `vips-heif`. This means, due to legal reasons, you have to build your own Docker image and you cannot use the existing images from the public Docker repository.

The `vips-heif` package needs to be either added to the existing Dockerfile or it can be added by creating a second, small Dockerfile wrapper around the official image from the registry. Save this wrapper as `Dockerfile.heif` next to your existing `Dockerfile`:

```Dockerfile
FROM opencloudeu/opencloud:latest

USER root
RUN apk add --no-cache vips-heif
USER 1000
```

Then build this new Dockerfile and use the resulting image in place of the stock one. Without Compose, tag it yourself and reference that tag wherever the stock image is used:

```shell
docker build -f Dockerfile.heif -t opencloud-heif:latest .
```

With Compose, replace the `image:` line of the `opencloud` service with a `build:` section. Keeping an `image:` line as well is useful, it gives the image Compose builds a name of its own:

```yaml
services:
  opencloud:
    build:
      context: .
      dockerfile: Dockerfile.heif
    image: opencloud-heif:latest
```

Build it and start the stack:

```shell
docker compose up -d --build
```

Note that `FROM opencloudeu/opencloud:latest` is resolved while the image is built, not when the container starts, so your image does not follow new OpenCloud releases by itself. Rebuild it whenever you would have pulled a new stock image, with `--pull` so the base image is refreshed as well:

```shell
docker compose build --pull && docker compose up -d
```

To verify that the module actually ended up in your image, run this:

```shell
docker compose exec opencloud sh -c 'ls /usr/lib/vips-modules-*/'
```

The output should list `vips-heif.so`.

OpenCloud verifies the HEVC and AV1 decoders independently when thumbnail support is first queried by decoding a tiny image with each codec. It only advertises previews for the formats whose decoder actually works, even on systems where the codec backends are packaged separately.

AVIF is not affected by the HEVC patent situation. It is listed here because libvips reads it through the same module. Alpine's `vips-heif` dependencies provide the required decoders, so there is nothing extra to install for it when using the wrapper image above.
