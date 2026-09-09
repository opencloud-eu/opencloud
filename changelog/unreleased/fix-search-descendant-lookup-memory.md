Bugfix: Bound memory of the search descendant lookup

Deleting, moving, restoring or purging a folder made the search service look
up every descendant of that folder with a Path wildcard query. Path is a
keyword field, so bleve expanded the wildcard into one term searcher per
descendant, all alive at once. Peak live memory scaled with the number of
descendants and the kernel OOM-killed the whole server on folder deletes; on
one production instance a routine delete held 2.24 GB of a 2.29 GB live heap
in this single query.

The lookup now enumerates the matching path terms from the field dictionary
and fetches the documents in bounded batches of exact term queries, returning
the same result set with O(1) live searcher memory: a 100k-file folder went
from 1194 MB peak to 102 MB, slightly faster than before.

https://github.com/opencloud-eu/opencloud/issues/1269
https://github.com/opencloud-eu/opencloud/issues/3469
