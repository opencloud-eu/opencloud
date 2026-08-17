Bugfix: Log a missing parent id cache entry at debug level

Moving a file made the activitylog service log "could not delete parent id
cache" at error level, even though the operation succeeded. The parent id
cache is filled lazily and its entries expire, so removing a key that was
never cached returns "key not found", which is the expected case and not an
error. It is now logged at debug level.

https://github.com/opencloud-eu/opencloud/issues/1043
