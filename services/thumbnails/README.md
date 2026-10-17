# Thumbnails

The thumbnails service is a stateless image resizer. It exposes an
imagor-compatible push endpoint that accepts an original image as a multipart
upload and returns the resized thumbnail encoded in the requested format.

> **Warning:** This service performs no authentication or authorization. By
> default it binds to `127.0.0.1:9186` so only local processes (the webdav
> service) can reach it. If you change `THUMBNAILS_HTTP_ADDR` to a non-loopback
> address, put it behind your reverse proxy's auth layer or an internal network
> boundary. Anyone who can reach the endpoint can upload and retrieve
> arbitrary images through it.

## Push Endpoint

The webdav service (which owns the complete thumbnail workflow) POSTs the
source file to this endpoint and receives the processed image back.

| Route                                                     | Description                                                                       |
| --------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `POST /unsafe/{width}x{height}/filters:{filters}`         | Fill the exact width x height (center crop, may upscale), the default             |
| `POST /unsafe/fit-in/{width}x{height}/filters:{filters}`  | Scale to fit within width x height, preserving aspect ratio and never upscaling   |
| `POST /unsafe/stretch/{width}x{height}/filters:{filters}` | Resize to the exact width x height without preserving aspect ratio (distorts)     |

The path is imagor's. The filters segment is optional and may be empty
(`filters:`), and an image key behind it is accepted and ignored, because the
source arrives in the upload rather than being loaded by key. A bare box with
nothing behind it is the one place this service and imagor part: imagor reads the
last segment as the image key, so `/unsafe/100x100` is a key and nothing is
resized, while this service treats it as the box. Inside the filters segment,
`format` and `no_upscale` are the filters this executor acts on, every other
filter is ignored. `format` itself is optional: without it the input's own format is
preserved, the same way imagor does it. An input this service cannot write
(tiff, bmp) becomes a JPEG; imagor writes tiff back in its own format and only
falls back to JPEG for the ones it cannot write either. Which formats those are
depends on the build: with libvips a webp source stays a webp, the pure Go build
answers it as a JPEG because it has no webp encoder.

What a build can read is its own backend's answer rather than a list: the library
that will decode the image reads the header first, so a source it cannot read
answers `406` before anything is decoded, and the maximum dimensions are checked
on that same header.

A side given as `0` is computed from the other one, so `/unsafe/0x100/` keeps the
aspect ratio and `/unsafe/0x0/` returns the source at its own size. A free axis
leaves nothing to crop to, so the operation makes no difference to it.

Like real imagor, the default fill resize **upscales** small sources to fill
the box exactly (a 100x100 source requested at `320x320` returns a 320x320
image). Adding the `no_upscale()` filter caps the result at the source size
instead (the same request returns 100x100). The `fit-in` route never upscales,
imagor would with the `upscale()` filter, this service ignores it.

Which route a preview request ends up on, and the `processor` and `a`
parameters that decide it, are the webdav service's business and documented in
[its README](../webdav/README.md).

The request body is a `multipart/form-data` upload with a single file field
named `image`, typed `image/*`; an untyped part is refused the way imagor's
upload loader refuses it. Output formats are `jpg`, `png`, `gif` and, with
libvips, `webp`; the pure Go build answers `406` for `format(webp)` because it
has no webp encoder. An animated gif asked for a still format is flattened to
its first frame, and a still source asked for `format(gif)` is palettised.

Reading an ICO needs libvips with its magick loader; the pure Go build answers
`406` for one.

The answers are imagor's: `400` for a malformed request, `406` for a source it
cannot read, for a format this build cannot write and for an upload part that
is not typed `image/*`, `422` for an image above the configured maximum
dimensions, `429` when the concurrency limit is reached.

## Configuration

| Environment variable                 | Description                                                                                                                                                                           |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `THUMBNAILS_HTTP_ADDR`               | Bind address of the HTTP service (default `127.0.0.1:9186`)                                                                                                                           |
| `THUMBNAILS_LOG_LEVEL`               | Log level (`panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace`)                                                                                                               |
| `THUMBNAILS_MAX_CONCURRENT_REQUESTS` | Maximum number of thumbnail requests decoded and resized in parallel. Default is 0 (unlimited). Requests arriving while the limit is reached get HTTP 429 with a `Retry-After` header |
| `THUMBNAILS_MAX_INPUT_WIDTH`         | Maximum width of a source image, 7680 by default. A wider one is refused with HTTP 422, which webdav turns into its legacy 403                                                        |
| `THUMBNAILS_MAX_INPUT_HEIGHT`        | Maximum height of a source image, 7680 by default                                                                                                                                     |

## Using libvips for Image Processing

To improve performance and to support a wider range of image formats, the
thumbnails service is able to utilize the
[libvips library](https://www.libvips.org/) for image processing. Support for
libvips needs to be enabled at buildtime and has a couple of implications:

- With libvips support enabled, it is not possible to create a statically
   linked OpenCloud binary.
- Therefore, the libvips shared libraries need to be available at runtime in
   the same release that was used to build the OpenCloud binary.
- When using the OpenCloud docker images, the libvips shared libraries are
   included in the image and are correctly embedded.

Support of libvips is disabled by default. To enable it, make sure libvips
and its buildtime dependencies are installed in your build environment. For
macOS users, add the build time dependencies via:

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

When building a docker image using the Dockerfile in the top-level directory of
OpenCloud, libvips support is enabled and the libvips shared libraries are
included in the resulting docker image.
