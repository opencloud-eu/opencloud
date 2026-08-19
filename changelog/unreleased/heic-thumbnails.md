Enhancement: Optionally generate thumbnails for HEIC, HEIF and AVIF images

Photos from recent Apple devices are stored as HEIC and there was no way to show
a preview up until now. A HEVC decoder cannot be shipped as part of the
OpenCloud binaries or images because of the patent situation around HEVC.

But: Builds with libvips support now look for the optional libvips heif loader
on startup. That loader is packaged separately from libvips and is not part of
the images we build, so admins who may want to use HEVC can install it
themselves. When it is there, the thumbnails service registers image/heic,
image/heic-sequence, image/heif, image/heif-sequence and image/avif and renders
them like any other image. When it is not, the mimetypes stay unregistered and
oc:has-preview keeps reporting 0 for those files, so clients do not ask for
thumbnails the server cannot produce. 

AVIF is registered along with the others because the same loader reads it, so
it comes along with HEIC at no extra cost.

Displaying HEIC in the web UI also needs the mimetypes in the preview app of
OpenCloud Web (still an open follow-up task), but file list thumbnails already
work without it.

https://github.com/opencloud-eu/opencloud/issues/630
