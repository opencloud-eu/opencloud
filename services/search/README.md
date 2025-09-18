# Search

The search service is responsible for metadata and content extraction,
the retrieved data is indexed and made searchable.

The search service runs out of the box with the shipped default `basic` configuration.
No further configuration is needed.

Note that as of now, the search service cannot be scaled.
Consider using dedicated hardware for this service in case more resources are needed.

## Search backends

To store and query the indexed data, a search backend is needed.

As of now, the search service supports the following backends:

- [bleve](https://github.com/blevesearch/bleve) (default)
- [opensearch](https://opensearch.org/)

### Bleve

Bleve is a lightweight, embedded full-text search engine written in Go and is the default search backend.
It is straightforward to set up and requires no additional services to run.

The following optional settings can be set:

*   `SEARCH_ENGINE_BLEVE_DATA_PATH=/path/to/bleve/index` (default: `$OC_BASE_DATA_PATH/search`): Path to store the bleve index.

### OpenSearch

OpenSearch is a distributed, RESTful search and analytics engine capable of addressing a growing number of use cases.
Additionally, it provides advanced features like clustering, replication, and sharding.

To enable OpenSearch as a backend, the following settings must be set:

*   `SEARCH_ENGINE_TYPE=open-search`
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ADDRESSES=http://YOUR-OPENSEARCH.URL:9200` (comma-separated list of OpenSearch addresses)

Additionally, the following optional settings can be set:

*   `SEARCH_ENGINE_OPEN_SEARCH_RESOURCE_INDEX_NAME=val` (default: `opencloud-resource`): Name of the OpenSearch index
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_USERNAME=val`: Username for HTTP Basic Authentication.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_PASSWORD=val`: Password for HTTP Basic Authentication.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_HEADER=val`: HTTP headers to include in requests.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_CA_CERT=val` CA certificate for TLS connections.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_RETRY_ON_STATUS=val` HTTP status codes that trigger a retry.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISABLE_RETRY=val` Disable retries on errors.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_RETRY_ON_TIMEOUT=val`: Enable retries on timeout.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_MAX_RETRIES=val`: Maximum number of retries for requests.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_COMPRESS_REQUEST_BODY=val`: Compress request bodies.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISCOVER_NODES_ON_START=val`: Discover nodes on service start.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISCOVER_NODES_INTERVAL=val`: Interval for discovering nodes.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_METRICS=val`: Enable metrics collection.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_DEBUG_LOGGER=val`: Enable debug logging.
*   `SEARCH_ENGINE_OPEN_SEARCH_CLIENT_INSECURE=val`: Skip TLS certificate verification.

## Query language

By default, [KQL](https://learn.microsoft.com/en-us/sharepoint/dev/general-development/keyword-query-language-kql-syntax-reference) is used as the query language.
For an overview of how to write kql queries, please read the [microsoft documentation](https://learn.microsoft.com/en-us/sharepoint/dev/general-development/keyword-query-language-kql-syntax-reference).

Not all parts are supported, the following list gives an overview of parts that are not implemented yet:

*   Synonym operators
*   Inclusion and exclusion operators
*   Dynamic ranking operator
*   ONEAR operator
*   NEAR operator
*   Date intervals

In [this ADR](https://github.com/owncloud/ocis/blob/docs/ocis/adr/0020-file-search-query-language.md) you can read why KQL was chosen.

## Content analysis / Extraction

The search service supports the following content extraction methods:

*   `Basic`: enabled by default, only provides metadata extraction.
*   `Tika`: needs to be installed and configured separately, provides content extraction for many file types.

Note that the file content has to be transferred to the search service internally for content extraction,
which is resource-intensive and can lead to delays with larger documents.

### Basic

This extractor is the simplest one and just uses the resource information provided by OpenCloud.
It does not do any further content analysis.

### Tika

The main difference is that this extractor is able to analyze and extract data from more advanced file types like PDF, DOCX, PPTX, etc.
However, [Apache Tika](https://tika.apache.org/) is required for this task.
Read the [Getting Started with Apache Tika](https://tika.apache.org/2.6.0/gettingstarted.html) guide on how to install and run Tika or use a ready to run [Tika container](https://hub.docker.com/r/apache/tika).
See the [Tika container usage document](https://github.com/apache/tika-docker#usage) for a quickstart.

As soon as Tika is installed and configured, the search service needs to be told to use it.

The following settings must be set:

*   `SEARCH_EXTRACTOR_TYPE=tika`
*   `SEARCH_EXTRACTOR_TIKA_TIKA_URL=http://YOUR-TIKA.URL`

Additionally, the following optional settings can be set:

*   `SEARCH_EXTRACTOR_TIKA_CLEAN_STOP_WORDS=true` (default: `true`): ignore stop words like `I`, `you`, `the` during content extraction.

## Manually Trigger Re-Indexing a Space

The service includes a command-line interface to trigger re-indexing a space:

```shell
opencloud search index --space $SPACE_ID
```

It can also be used to re-index all spaces:

```shell
opencloud search index --all-spaces
```

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
