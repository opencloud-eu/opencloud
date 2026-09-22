Bugfix: Generate public link thumbnails behind a compressing reverse proxy

Public link thumbnails were never generated when a reverse proxy in front of
OpenCloud compressed responses. The thumbnails service reported "thumbnails:
image is too large" and the webdav service turned that into a 500, no matter how
small the image actually was.

A missing Content-Length is not evidence of size: a proxy drops the header
whenever it compresses or re-chunks a response.

The download is now capped while it is read instead of being rejected up front,
so images of unknown length are processed and oversized ones are still refused.

Already generated thumbnails were served from the cache and were never affected,
which is why the problem only showed up for images opened for the first time
while logged out.

<https://github.com/opencloud-eu/opencloud/issues/3450>
