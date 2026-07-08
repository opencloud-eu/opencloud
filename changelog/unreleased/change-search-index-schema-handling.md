Change: Check the search index schema on startup (existing indexes need a rebuild)

The search service now compares the schema of an existing search index with
the schema expected by the code at startup, for both the bleve and the
OpenSearch engine. A purely additive change (new fields that have never been
indexed) is applied in place and the service starts; documents indexed before
the upgrade do not contain the new fields until they are re-indexed. Any other
difference (changed field definitions, changed analyzers, removed or renamed
fields, or new fields that already contain data of unknown form) makes the
service refuse to start instead of silently returning wrong or incomplete
search results.

Upgrading to this version is a breaking change for BOTH engines: every search
index built by a previous version differs from the new schema and the service
will refuse to start. To rebuild: stop the service, delete the search index
(the bleve directory or the OpenSearch index), start the service (an empty
index with the new schema is created) and run
"opencloud search index --all-spaces" to re-index all files. To bring an
instance up without search until a maintenance window, set
OC_EXCLUDE_RUN_SERVICES=search.

https://github.com/opencloud-eu/opencloud/issues/3092
