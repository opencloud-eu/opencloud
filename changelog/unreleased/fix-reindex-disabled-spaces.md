Bugfix: Do not fail reindexing all spaces on disabled spaces

Reindexing all spaces via `opencloud search index --all-spaces` aborted as soon
as it ran into a disabled space. The storage provider answers with "not found"
for the root of a disabled space, so the tree walk failed and took the whole run
down with it. Every space that had not been processed yet was left unindexed,
which also made the post upgrade reindex job of the helm charts fail.

Disabled spaces are now skipped and logged, so they can be indexed again once
they have been enabled.

https://github.com/opencloud-eu/opencloud/issues/3559
