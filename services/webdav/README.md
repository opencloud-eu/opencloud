# Webdav

The webdav service, like the [frontend](../frontend) service, provides a HTTP API following the webdav protocol. It receives HTTP calls from requestors like clients and issues gRPC calls to other services executing these requests. After the called service has finished the request, the webdav service will render their responses in `xml` and sends them back to the requestor.

## Endpoints Overview

Currently, the webdav service handles request for two functionalities, which are `Thumbnails` and `Search`.

### Thumbnails

The webdav service provides various `GET` endpoints to get the thumbnails of a file in authenticated and unauthenticated contexts. It also provides thumbnails for spaces on different endpoints.

#### Who Makes a Thumbnail

Three parties can turn a file into a thumbnail:

- the **generator** (imagor, or the thumbnails service in its place) scales images
- the **built-in converters** in the webdav service render plain text and extract audio cover art and GeoGebra thumbnails. The audio and GeoGebra extractors are deprecated in favour of Tika.
- an **Apache Tika server**, when configured, extracts a preview image from documents, ebooks, office files, audio, camera raws and more

A thumbnail is only attempted for a type that is announced (`OC_THUMBNAILS_ADDITIONAL_SOURCE_TYPES` plus the built-in list) or that Tika takes (named in the Tika list, or any type when the list contains `*`). Every other type answers `404` without the file being read. For the types that pass, the first rule that applies wins:

1. a type the Tika list names goes to Tika, even an image (that is how camera raws get a preview: reva types them as images, and no generator reads them)
2. text, audio and GeoGebra go to the built-in converters
3. an image goes to the generator as it is
4. anything else goes to Tika if the Tika list contains `*`
5. what is left is an announced type, and it goes to the generator as it is, which answers whether it can read it

So `*` never sends an image to Tika. An image type that only Tika can read has to be named in the Tika list.

#### Typical Setups

| File                  | No Tika          | Tika, default list | Default list plus `*` | List is only `*`         | Empty list       |
| --------------------- | ---------------- | ------------------ | --------------------- | ------------------------ | ---------------- |
| `photo.jpg`           | generator        | generator          | generator             | generator                | generator        |
| `photo.nef` (raw)     | no preview       | Tika               | Tika                  | generator, 404           | no preview       |
| `report.pdf`, `.docx` | no preview       | Tika               | Tika                  | Tika                     | no preview       |
| `song.mp3`            | built-in         | Tika               | Tika                  | built-in                 | built-in         |
| `board.ggs`           | built-in         | Tika               | Tika                  | built-in                 | built-in         |
| `notes.txt`           | built-in         | built-in           | built-in              | built-in                 | built-in         |
| `archive.zip`         | no preview       | no preview         | Tika                  | Tika                     | no preview       |

- **Default list plus `*`**: `OC_TIKA_ADDITIONAL_THUMBNAIL_MIME_TYPES=*`
- **List is only `*`**: `OC_TIKA_THUMBNAIL_MIME_TYPES=*`. This replaces the default list, so the raws are no longer named and fall to the generator.
- **Empty list**: `OC_TIKA_THUMBNAIL_MIME_TYPES=""`. This keeps a Tika server that is shared with the search service for search alone.
- "no preview" means no attempt is made, and the request answers `404`.

#### Tika and Thumbnails

Set `OC_TIKA_URL` (or `WEBDAV_TIKA_URL`) to an Apache Tika server 4.2.0 or newer, started with `TIKA_PRESETS=thumbnail`. The webdav service calls `PUT /unpack/preset/thumbnail`, which only exists with that preset active.

The Tika list starts from a default list: the formats a Tika release serves a thumbnail for, which are documents, ebooks, office and iWork files, GeoGebra, audio cover art and camera raws. It is configured with these variables:

- `OC_TIKA_ADDITIONAL_THUMBNAIL_MIME_TYPES` adds to the default list. This is the usual way to send more to Tika, including `*` for every type that no other rule claims.
- `OC_TIKA_THUMBNAIL_MIME_TYPES` replaces the default list. Use it to remove types from the default list, or set it empty to send nothing. Whatever it holds is the whole list, so a value of `*` alone drops the camera raws (see the table above).

An entry can map a type to the name Tika uses for it, for example `image/x-nikon-nef:image/x-raw-nikon`. That name decides which Tika parser reads the file. A raw photo that Tika only detects as a plain tiff is parsed as a tiff, and its embedded preview is never found. Every format in the default list is detected correctly without a mapping. The mapping is meant for the next format whose detection stops one level short.

##### Tika Before 4.2.0

`TIKA_PRESETS` is read by Tika 4.2.0 and newer. A 4.1.x server serves the same route when its config file activates the preset:

```json
{
  "server": { "port": 9998, "host": "0.0.0.0" },
  "presets": { "thumbnail": true }
}
```

It is passed with `-c /tika-config.json`. The `server` block has to be in the file, because the file replaces what the command line would set.

#### Announced Types and Output Types

Announcing a type and sending it to Tika are separate settings:

- `OC_THUMBNAILS_ADDITIONAL_SOURCE_TYPES` decides what `oc:has-preview` reports in a listing.
- The Tika list decides who makes the image.

A type that Tika takes is attempted even when it is not announced. A direct request can therefore get a preview that a listing reports as absent.

The thumbnail is written in the type that the **file's** own type maps to (`OC_THUMBNAILS_ADDITIONAL_OUTPUT_TYPES`), not in whatever Tika returns. The answer is the same before and after a cache hit. A Windows binary maps to `image/png` by default, because the icon Tika reads from it has transparency that a JPEG would flatten onto black.

#### Thumbnail Cache

The webdav service caches the thumbnails it generates. Choose a backend with `WEBDAV_THUMBNAILS_CACHE_BACKEND`:

| Backend          | Use case                                                                   |
| ---------------- | -------------------------------------------------------------------------- |
| `file` (default) | single instance, entries under `$OC_BASE_DATA_PATH/thumbnails/cache`, change with `WEBDAV_THUMBNAILS_CACHE_DIR` |
| `s3`             | several instances behind a load balancer that share one cache              |
| `memory`         | bounded, drops its oldest entries first                                    |
| `none`           | turns caching off on purpose                                               |

`s3` needs a bucket, a region and credentials. A service other than AWS also needs `WEBDAV_THUMBNAILS_CACHE_S3_ENDPOINT`, and it is always addressed over https. While previews are enabled, an unknown backend or an incomplete `s3` configuration fails at startup rather than running without a cache.

##### What Is Cached

- every generated thumbnail, one entry per size
- once per file, the image an extractor (Tika or a built-in converter) got from it, so a second size does not extract again
- the fact that a file holds no image at all, so the extractor is asked only once

A failed extraction (Tika unreachable, preset not active, a broken file) is not cached and is retried on the next request.

##### Expiry

Entries never expire. A changed file gets new entries, because the key contains the file's checksum. An extractor that learns a new format, however, does not revisit files it has already answered for. After such an upgrade (a new Apache Tika release, for example), empty the cache (`WEBDAV_THUMBNAILS_CACHE_DIR`, or the bucket) so those files are asked again.

#### Thumbnail Query String Parameters

Clients can request thumbnail previews for files by adding `?preview=1` to the file URL. Requests for files with no thumbnail available respond with HTTP status `404`.

The following query parameters are supported:

| Parameter | Required | Default Value                                              | Description                                                                     |
| --------- | -------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------- |
| preview   | YES      | 1                                                          | generates preview                                                               |
| x         | NO       | 32, when y is absent too                                   | horizontal target size, absent leaves the width unconstrained, `0` is an error  |
| y         | NO       | 32, when x is absent too                                   | vertical target size, absent leaves the height unconstrained, `0` is an error   |
| scalingup | NO       | 0                                                          | accepted for compatibility but ignored; not forwarded to the generator          |
| a         | NO       | none                                                       | accepted for compatibility but ignored; this service has never honored it, so the chain below decides |
| c         | NO       | none                                                       | accepted for compatibility but ignored; the cache key holds the file's checksum, so a changed file gets a new thumbnail without it |
| processor | NO       | none, the chain below decides                              | preferred thumbnail processor                                                   |

#### Thumbnail Resolution

`WEBDAV_THUMBNAILS_RESOLUTIONS` is a ladder of edge lengths, not a list of boxes. Its largest entry is also the largest thumbnail this service asks a generator for, which is why it tops out at 2048: `fill` and `stretch` enlarge a small source to fill the box, so a higher rung lets a tiny image cost whatever that box costs. Each axis of a request is rounded up to the next length on the ladder, on its own, so a thumbnail never comes back in a different orientation than it was asked for. The largest length is the cap: a request above it, and an axis left unconstrained, get that one.

Example with the default ladder (16, 32, 64, 128, 192, 256, 320, 384, 448, 512, 640, 768, 1024, 1280, 1920, 2048):

Requested: 18x12\
Returned: 32x16

Entries written as `WidthxHeight` are still accepted and contribute both of their sides to the ladder, each capped at 2048, the largest box the built-in generator accepts by default. A configuration written for the old box list, which reached 7680x4320, keeps working that way. A larger length has to be a plain one, with the generator's `THUMBNAILS_MAX_OUTPUT_WIDTH` and `THUMBNAILS_MAX_OUTPUT_HEIGHT` raised to match.

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

Naming one of them decides the behavior on its own, legacy alias or imagor name. A name nobody knows is treated like no processor at all.

To apply one of those, a query parameter has to be added to the request, like `?processor=fit-in`.

Without a processor two things decide, in this order. An unconstrained axis is always fitted, because there is no box to crop to, so `?y=512` needs nothing else. Otherwise the source type decides, `resize` for gifs and `fill` for everything else.

#### Thumbnail Generator

The thumbnails themselves are made by a generator the webdav service POSTs to, configured with `WEBDAV_THUMBNAILS_GENERATOR_URL` (and `WEBDAV_THUMBNAILS_GENERATOR_TIMEOUT`). The built-in thumbnails service speaks the imagor image endpoint, so a real [imagor](https://imagor.net) can be put in its place without changing anything else. A source the generator cannot read answers `404`, one that is too large `403`, and `WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE` caps what is sent at all; `OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE` sets this and the generator's own cap together, so one does not send what the other refuses. A generator that refuses because it is busy answers `429` with its own `Retry-After`, so a client can come back instead of treating it as a missing preview. The same cap bounds an image webdav extracts itself, the cover of an audio file or the thumbnail inside a geogebra container, because a compressed entry says nothing about its size until it is read. A cropping resize asks the generator for a smart crop, so what survives is the region that carries the image; the built-in thumbnails service answers the same request with a centre crop.

Three settings say what is made of what. Each one is merged onto the built-in defaults, which is what their `ADDITIONAL` says: naming one type does not take the answer away from the others, and an entry for a type the defaults already answer for replaces that one entry. The first takes bare mime types, the other two take `key:value` entries:

| Environment variable                 | Description                                                                                                                                                  |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `OC_THUMBNAILS_ADDITIONAL_SOURCE_TYPES`      | Types a preview is announced and attempted for, e.g. `image/vnd.microsoft.icon` for a generator that reads icons. The webdav, graph and ocdav side all read this one name |
| `OC_THUMBNAILS_ADDITIONAL_OUTPUT_TYPES`      | The type a thumbnail is returned in, per source type, as `mime:mime` entries, e.g. `image/vnd.microsoft.icon:image/png`                                      |
| `OC_THUMBNAILS_ADDITIONAL_GENERATOR_FORMATS` | The formats the generator writes, as `mime:format` entries, e.g. `image/webp:webp`. The format is the name its own `format(...)` filter uses                 |

Reading them as a chain: a source type is mapped to the type its thumbnail is returned in, and that type is asked for by the name the generator knows it under. An icon becomes `image/png`, which the generator is asked for as `format(png)`. A type nobody mapped becomes a JPEG, and so does a mapping onto a type this service has no format name for: what a generator behind the URL really writes is its own business, and a wrong guess here must not keep the service from starting.

The defaults return `image/png` for png, icon and windows binary sources, keep `image/gif` for a gif and return `image/jpeg` for everything else, so a client must read the response's `Content-Type` instead of assuming the type it got before.

Previews can be switched off entirely with `OC_DISABLE_PREVIEWS`. Search keeps working, every thumbnail path answers `404`, and nothing here is read or checked: a typo in the resolutions, the cache backend or the type mapping surfaces when previews are switched back on, not before.

In the single binary nothing has to be set: the generator URL defaults to the built-in thumbnails service. A deployment that runs the services in separate containers has to point `WEBDAV_THUMBNAILS_GENERATOR_URL` at its thumbnails service or at an imagor, because `127.0.0.1` does not reach it from there. [MIGRATION.md](MIGRATION.md) says what else an existing deployment has to change.

### Search

The webdav service provides access to the search functionality. It offers multiple `REPORT` endpoints for getting search results.

See the [search](https://github.com/opencloud-eu/opencloud/tree/main/services/search) service for more details about search functionality.

## Scalability

The webdav service persists generated thumbnails to its thumbnail cache (file backend by default). When running multiple instances behind a load balancer, point `WEBDAV_THUMBNAILS_CACHE_BACKEND` at a shared `s3` bucket so all instances read and write the same cache; otherwise each instance keeps its own on-disk cache.
