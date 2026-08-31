# Migrating the search index

A version that changes how resources are indexed works on a new index and
leaves the old one untouched. The service starts normally, but the new index is
empty: search finds nothing until it is filled. The old index stays around
until you remove it.

## v7.x.x to %%NEXT%%

### OpenSearch

The new index is `opencloud-resource-v3`. Fill it in one of two ways:

- copy the old index, fast and keeps the extracted file contents, or
- index all spaces again, slower since every file is read once more, but drops
  documents that no longer have a resource.

The address below is the one from `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ADDRESSES`,
`opencloud-resource` the name from
`SEARCH_ENGINE_OPEN_SEARCH_RESOURCE_INDEX_NAME`.

```shell
# either copy the old index
curl -X POST "https://opensearch.example.com:9200/_reindex?wait_for_completion=false" \
  -H 'Content-Type: application/json' -d '
  {"source":{"index":"opencloud-resource"},"dest":{"index":"opencloud-resource-v3"}}'

# the answer carries a task id, watch it while it runs
curl "https://opensearch.example.com:9200/_tasks/<task-id>"

# or index all spaces again, the service keeps running while it happens
opencloud search index --all-spaces
```

Once the new index is filled, remove the old one:

```shell
curl -X DELETE "https://opensearch.example.com:9200/opencloud-resource"
```

### bleve

The new index is the `bleve-v2` directory next to the old `bleve` one, both in
`$OC_BASE_DATA_PATH/search` by default (`SEARCH_ENGINE_BLEVE_DATA_PATH`). A
bleve index cannot be copied, index all spaces again:

```shell
opencloud search index --all-spaces
```

Once the new index is filled, remove the old one:

```shell
rm -r "$OC_BASE_DATA_PATH/search/bleve"
```
