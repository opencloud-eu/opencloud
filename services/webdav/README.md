# Webdav

The webdav service, like the [frontend](../frontend) service, provides a HTTP API following the webdav protocol. It receives HTTP calls from requestors like clients and issues gRPC calls to other services executing these requests. After the called service has finished the request, the webdav service will render their responses in `xml` and sends them back to the requestor.

## Endpoints Overview

Currently, the webdav service handles request for two functionalities, which are `Thumbnails` and `Search`.

### Thumbnails

The webdav service provides various `GET` endpoints to get the thumbnails of a file in authenticated and unauthenticated contexts. It also provides thumbnails for spaces on different endpoints. 

See the [thumbnail](https://github.com/opencloud-eu/opencloud/tree/main/services/thumbnails) service for more information about thumbnails.

### Search

The webdav service provides access to the search functionality. It offers multiple `REPORT` endpoints for getting search results. 

See the [search](https://github.com/opencloud-eu/opencloud/tree/main/services/search) service for more details about search functionality. 

## Scalability

The webdav service does not persist any data and does not cache any information. Therefore multiple instances of this service can be spawned in a bigger deployment like when using container orchestration with Kubernetes, without any extra configuration.
