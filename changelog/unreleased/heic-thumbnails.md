Enhancement: Optionally generate thumbnails for HEIC, HEIF and AVIF images

Photos from recent Apple devices are stored as HEIC and there was no way to show
a preview up until now. A HEVC decoder cannot be shipped as part of the
OpenCloud binaries or images because of the patent situation around HEVC.

But: Builds with libvips support now test the optional HEVC and AV1 decoders
when thumbnail support is first queried. The required libvips loader and codec
backends are packaged separately and are not part of the images we build, so
admins can install them themselves.
When decoding succeeds, the thumbnails service registers the corresponding
image/heic, image/heic-sequence, image/heif, image/heif-sequence or image/avif
mimetypes and renders them like any other image. When a decoder is not
available, its mimetypes stay unregistered and oc:has-preview keeps reporting 0
for those files, so clients do not ask for thumbnails the server cannot
produce.

AVIF uses the same libvips loader but is registered independently because it
uses an AV1 decoder instead of an HEVC decoder.

Displaying HEIC in the web UI also needs the mimetypes in the preview app of
OpenCloud Web (still an open follow-up task), but file list thumbnails already
work without it.

https://github.com/opencloud-eu/opencloud/issues/630
