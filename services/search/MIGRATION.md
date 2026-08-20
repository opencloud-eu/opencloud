# Migrating the OpenSearch index

Our index definition id versioned. The service does not start on an index that
was built with an older one. A new installation is not affected.

There are two ways out.

> `$OS` (the OpenSearch address), `opencloud-resource` (default) index name.

## Throw it away and index again

Recommended for small installations. Works for every version, and it also
removes records that were already orphaned. Every file is read and extracted again,
so it takes longer on a big installation and search is incomplete until it is
done.

Writing to a missing index creates one without our analyzer and mappings, so
opencloud has to be down while it is gone.

```shell
# stop opencloud
curl -XDELETE "$OS/opencloud-resource"
# start opencloud, it creates the index from the current definition
opencloud search index --all-spaces
```

## v2 to v3

Keeps the documents, updating the index without reindexing the whole file tree.

```shell
# stop opencloud

curl -XPOST "$OS/opencloud-resource/_close"
curl -XPUT "$OS/opencloud-resource/_settings" -H 'Content-Type: application/json' -d '{"analysis":{"analyzer":{"path_hierarchy":{"filter":null}}}}'
curl -XPOST "$OS/opencloud-resource/_open"

# the updated v3 fields
curl -XPUT "$OS/opencloud-resource/_mapping" -H 'Content-Type: application/json' -d '
{"properties":{"Name":{"type":"text","fields":{"wildcard":{"type":"wildcard"}}},
               "Title":{"type":"text","fields":{"wildcard":{"type":"wildcard"}}},
               "Tags":{"type":"text","fields":{"wildcard":{"type":"wildcard"}}},
               "Mtime":{"type":"date","ignore_malformed":true}}}'

# start opencloud

curl -XPOST "$OS/opencloud-resource/_update_by_query?conflicts=proceed&wait_for_completion=false"
```

The last command answers with a task id, watch it with `GET $OS/_tasks/<task-id>`.
A million documents take a few minutes. Search keeps working while it runs,
documents get correct one by one, and the run can be repeated at any time.
