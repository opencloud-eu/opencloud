# Search

The search service is responsible for metadata and content extraction, stores that data as index and makes it searchable. The following clarifies the extraction terms _metadata_ and _content_:

*   Metadata: all data that _describes_ the file like `Name`, `Size`, `MimeType`, `Tags` and `Mtime`.
*   Content: all data that _relates to content_ of the file like `words`, `geo data`, `exif data` etc.

## General Considerations

*   To use the search service, an event system needs to be configured for all services like NATS, which is shipped and preconfigured.
*   The search service consumes events and does not block other tasks.
*   When looking for content extraction, [Apache Tika - a content analysis toolkit](https://tika.apache.org) can be used but needs to be installed separately.

Extractions are stored as index via the search service. Consider that indexing requires adequate storage capacity - and the space requirement will grow. To avoid filling up the filesystem with the index and rendering OpenCloud unusable, the index should reside on its own filesystem.

You can change the path to where search maintains its data in case the filesystem gets close to full and you need to relocate the data. Stop the service, move the data, reconfigure the path in the environment variable and restart the service.

When using content extraction, more resources and time are needed, because the content of the file needs to be analyzed. This is especially true for big and multiple concurrent files.

The search service runs out of the box with the shipped default `basic` configuration. No further configuration is needed, except when using content extraction.

Note that as of now, the search service can not be scaled. Consider using a dedicated hardware for this service in case more resources are needed.

## Search engines

By default, the search service is shipped with [bleve](https://github.com/blevesearch/bleve) as its primary search engine. The available engines can be extended by implementing the [Engine](pkg/engine/engine.go) interface and making that engine available.

## Query language

By default, [KQL](https://learn.microsoft.com/en-us/sharepoint/dev/general-development/keyword-query-language-kql-syntax-reference) is used as query language,
for an overview of how the syntax works, please read the [microsoft documentation](https://learn.microsoft.com/en-us/sharepoint/dev/general-development/keyword-query-language-kql-syntax-reference).

Not all parts are supported, the following list gives an overview of parts that are not implemented yet:

*   Synonym operators
*   Inclusion and exclusion operators
*   Dynamic ranking operator
*   ONEAR operator
*   NEAR operator
*   Date intervals

 In [this ADR](https://github.com/owncloud/ocis/blob/docs/ocis/adr/0020-file-search-query-language.md) you can read why KQL whas chosen.

## Extraction Engines

The search service provides the following extraction engines and their results are used as index for searching:

*   The embedded `basic` configuration provides metadata extraction which is always on.
*   The `tika` configuration, which _additionally_ provides content extraction, if installed and configured.

## Content Extraction

The search service is able to manage and retrieve many types of information. For this purpose the following content extractors are included:

### Basic Extractor

This extractor is the most simple one and just uses the resource information provided by OpenCloud. It does not do any further analysis. The following fields are included in the index: `Name`, `Size`, `MimeType`, `Tags`, `Mtime`.

### Tika Extractor

This extractor is more advanced compared to the [Basic extractor](#basic-extractor). The main difference is that this extractor is able to search file contents.
However, [Apache Tika](https://tika.apache.org/) is required for this task. Read the [Getting Started with Apache Tika](https://tika.apache.org/2.6.0/gettingstarted.html) guide on how to install and run Tika or use a ready to run [Tika container](https://hub.docker.com/r/apache/tika). See the [Tika container usage document](https://github.com/apache/tika-docker#usage) for a quickstart. Note that at the time of writing, containers are only available for the amd64 platform.

As soon as Tika is installed and accessible, the search service must be configured for the use with Tika. The following settings must be set:

*   `SEARCH_EXTRACTOR_TYPE=tika`
*   `SEARCH_EXTRACTOR_TIKA_TIKA_URL=http://YOUR-TIKA.URL`

When the search service can reach Tika, it begins to read out the content on demand. Note that files must be downloaded during the process, which can lead to delays with larger documents.

Content extraction and handling the extracted content can be very resource intensive. Content extraction is therefore limited to files with a certain file size. The default limit is 20MB and can be configured using the `SEARCH_CONTENT_EXTRACTION_SIZE_LIMIT` variable.

When extracting content, you can specify whether [stop words](https://en.wikipedia.org/wiki/Stop_word) like `I`, `you`, `the` are ignored or not. Noramlly, these stop words are removed automatically. To keep them, the environment variable `SEARCH_EXTRACTOR_TIKA_CLEAN_STOP_WORDS` must be set to `false`.

When using the Tika container and docker-compose, consider the following:

*   See the [opencloud_full](https://github.com/opencloud-eu/opencloud/tree/main/deployments/examples/opencloud_full) example.
*   Containers for the linked service are reachable at a hostname identical to the alias or the service name if no alias was specified.

If using the `tika` extractor, make sure to also set `FRONTEND_FULL_TEXT_SEARCH_ENABLED` in the frontend service to `true`. This will tell the webclient that full-text search has been enabled.

## Search Functionality

The search service consists of two main parts which are file `indexing` and file `search`.

### Indexing

Every time a resource changes its state, a corresponding event is triggered. Based on the event, the search service processes the file and adds the result to its index. There are a few more steps between accepting the file and updating the index.

### Search

A query via the search service will return results based on the index created.

### State Changes which Trigger Indexing

The following state changes in the life cycle of a file can trigger the creation of an index or an update:

#### Resource Trashed

The service checks its index to see if the file has been processed. If an index entry exists, the index will be marked as deleted. In consequence, the file won't appear in search requests anymore. The index entry stays intact and could be restored via [Resource Restored](#resource-restored).

#### Resource Deleted

The service checks its index to see if the file has been processed. If an index entry exists, the index will be finally deleted. In consequence, the file won't appear in search requests anymore.

#### Resource Restored

This step is the counterpart of [Resource Trashed](#resource-trashed). When a file is deleted, is isn't removed from the index, instead the service just marks it as deleted. This mark is removed when the file has been restored, and it shows up in search results again.

#### Resource Moved

This comes into play whenever a file or folder is renamed or moved. The search index then updates the resource location path or starts indexing if no index has been created so far for all items affected. See [Notes](#notes) for an example.

#### Folder Created

The creation of a folder always triggers indexing. The search service extracts all necessary information and stores it in the search index

#### File Created

This case is similar to [Folder created](#folder-created) with the difference that a file can contain far more valuable information. This gets interesting but time-consuming when data content needs to be analyzed and indexed. Content extraction is part of the search service if configured.

#### File Version Restored

Since OpenCloud is capable of storing multiple versions of the same file, the search service also needs to take care of those versions. When a file version is restored, the service starts to extract all needed information, creates the index and makes the file discoverable.

#### Resource Tag Added

Whenever a resource gets a new tag, the service takes care of it and makes that resource discoverable by the tag.

#### Resource Tag Removed

This is the counterpart of [Resource tag added](#resource-tag-added). It takes care that a tag gets unassigned from the referenced resource.

#### File Uploaded - Synchronous

This case only triggers indexing if `async post processing` is disabled. If so, the service starts to extract all needed file information, stores it in the index and makes it discoverable.

#### File Uploaded - Asynchronous

This is exactly the same as [File uploaded - synchronous](#file-uploaded---synchronous) with the only difference that it is used for asynchronous uploads.

## Manually Trigger Re-Indexing a Space

The service includes a command-line interface to trigger re-indexing a space:

```shell
opencloud search index --space $SPACE_ID
```

It can also be used to re-index all spaces:

```shell
opencloud search index --all-spaces
```

Note that either `--space $SPACE_ID` or `--all-spaces` must be set.

## Notes

The indexing process tries to be self-healing in some situations.

In the following example, let's assume a file tree `foo/bar/baz` exists.
If the folder `bar` gets renamed to `new-bar`, the path to `baz` is no longer `foo/bar/baz` but `foo/new-bar/baz`.
The search service checks the change and either just updates the path in the index or creates a new index for all items affected if none was present.

## Metrics

The search service exposes the following prometheus metrics at `<debug_endpoint>/metrics` (as configured using the `SEARCH_DEBUG_ADDR` env var):

| Metric Name | Type | Description | Labels |
| --- | --- | --- | --- |
| `opencloud_search_build_info` | Gauge | Build information | `version` |
| `opencloud_search_events_outstanding_acks` | Gauge | Number of outstanding acks for events | |
| `opencloud_search_events_unprocessed` | Gauge | Number of unprocessed events | |
| `opencloud_search_events_redelivered` | Gauge | Number of redelivered events | |
| `opencloud_search_search_duration_seconds` | Histogram | Duration of search operations in seconds | `status` |
| `opencloud_search_index_duration_seconds` | Histogram | Duration of indexing operations in seconds | `status` |
