# Migrating the search index

A version that changes how resources are indexed works on a new index and
leaves the old one untouched. The service starts normally, but the new index is
empty: search finds nothing until it is filled. The old index stays around
until you remove it.

## v7.x.x to %%NEXT%%

### OpenSearch

Fill the new index by indexing all spaces again:

```shell
# the service keeps running while it happens
opencloud search index --all-spaces
```

Once the new index is filled, every index but the one with the highest
`-v<N>` suffix can go (indexes up to 7.4 have no suffix):

```shell
curl "https://os.example.com:9200/_cat/indices/opencloud-resource*"
curl -X DELETE "https://os.example.com:9200/opencloud-resource"
```

### bleve

The new index is a directory next to the old `bleve` one, both in
`$OC_BASE_DATA_PATH/search` by default (`SEARCH_ENGINE_BLEVE_DATA_PATH`). A
bleve index cannot be copied, index all spaces again:

```shell
opencloud search index --all-spaces
```

Once the new index is filled, every directory but the one with the highest
`bleve-v<N>` suffix can go (directories up to 7.4 have no suffix):

```shell
rm -r "$OC_BASE_DATA_PATH/search/bleve"
```
