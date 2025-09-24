package proxy

import future.keywords.if
import data.utils

default granted := true

granted = false if {
    input.request.method == "PUT"
    pathPrefixes := [
        "/dav",
        "/remote.php/webdav",
        "/remote.php/dav",
        "/webdav",
    ]
    restricted := pathPrefixes[_]
    startswith(input.request.path, restricted)
    not utils.is_extension_allowed(input.resource.name)
}

granted = false if {
    input.request.method == "POST"
    pathPrefixes := [
        "/data",
        "/dav",
        "/remote.php/webdav",
        "/remote.php/dav",
        "/webdav",
    ]
    restricted := pathPrefixes[_]
    startswith(input.request.path, restricted)
    not utils.is_extension_allowed(input.resource.name)
}
