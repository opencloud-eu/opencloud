# Thumbnails

The thumbnails service is a stateless image resizer. It exposes an imagor-compatible push endpoint that accepts an original image as a multipart upload and returns the resized thumbnail encoded in the requested format.

## Push Endpoint

The webdav service (which owns the complete thumbnail workflow) POSTs the source file to this endpoint and receives the processed image back.

| Route | Description |
|---|---|
| `POST /unsafe/{width}x{height}` (optionally `/filters:format({format})`) | Fill the exact width x height (center crop, may upscale) — the default |
| `POST /unsafe/fit-in/{width}x{height}` (optionally `/filters:format({format})`) | Scale to fit within width x height, preserving aspect ratio and never upscaling (letterboxed) |
| `POST /unsafe/stretch/{width}x{height}` (optionally `/filters:format({format})`) | Resize to the exact width x height without preserving aspect ratio (distorts) |

The filters segment is captured whole and parsed; only the `format` filter is meaningful to this executor, other filters are ignored. The `format` filter is optional: when absent the input's own format is preserved (detected via the imaging library), matching imagor. Inputs we cannot re-encode (e.g. webp, tiff, bmp) fall back to JPEG, mirroring imagor's default for unsavable sources.

The webdav service selects the route based on the requested processor. The full mapping is:

| Request | Operation |
|---|---|
| `processor=resize` | stretch (distort to the exact box) |
| `processor=fill` or `processor=thumbnail` | fill (center-crop to the exact box, may upscale) |
| `processor=fit` / `processor=fit-in` | fit-in (preserve aspect, fit in box, never upscale) |
| no processor, gif source | stretch (resize for gifs) |
| no processor, other sources | fill (default = thumbnail = fill) |

By default (no processor, non-gif) the fill form is used, which center-crops to the exact box and upscales small sources — this matches the legacy `thumbnail` processor behavior (e.g. a 200x100 image requested at 100x100 returns a square 100x100 image). Requesting `processor=fit` switches to the `fit-in` form, which preserves aspect ratio and never upscales, letterboxing a non-square source into the box (the same image returns 100x50).

### Legacy `a` parameter

The webdav preview endpoint also accepts a legacy `a` flag: `a=1` (or absent) means "preserve aspect" (fit-in), `a=0` means "fill". An explicit `processor` always wins over `a`. When an explicit processor overrides a contradictory `a`, the thumbnail response includes the header `X-OpenCloud-Thumbnail-Aspect-Ignored` so developers can tell their client to send a consistent request.

The request body is a `multipart/form-data` upload with a single file field named `image`. Supported output formats are `jpg`, `png`, and `gif`.

## Configuration

| Environment variable | Description |
|---|---|
| `THUMBNAILS_HTTP_ADDR` | Bind address of the HTTP service (default `127.0.0.1:9186`) |
| `THUMBNAILS_LOG_LEVEL` | Log level (`panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace`) |

## Using libvips for Image Processing


To improve performance and to support a wider range of image formats, the thumbnails service is able to utilize the [libvips library](https://www.libvips.org/) for image processing. Support for libvips needs to be enabled at buildtime and has a couple of implications:

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
