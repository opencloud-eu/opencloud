---
title: "opencloud.services.search.v0"
url: /apis/grpc_apis/opencloud_services_search_v0
date: 2025-03-27T14:44:40+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/services/search/v0/search.proto

### IndexSpaceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| space_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |

### IndexSpaceResponse




### SearchIndexRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Optional. The maximum number of entries to return in the response |
| page_token | [string](#string) |  | Optional. A pagination token returned from a previous call to `Get`<br>that indicates from where search should continue |
| query | [string](#string) |  |  |
| ref | [opencloud.messages.search.v0.Reference](/apis/grpc_apis/opencloud_messages_search_v0/#reference) |  |  |

### SearchIndexResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matches | [opencloud.messages.search.v0.Match](/apis/grpc_apis/opencloud_messages_search_v0/#match) | repeated |  |
| next_page_token | [string](#string) |  | Token to retrieve the next page of results, or empty if there are no<br>more results in the list |
| total_matches | [int32](#int32) |  |  |

### SearchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Optional. The maximum number of entries to return in the response |
| page_token | [string](#string) |  | Optional. A pagination token returned from a previous call to `Get`<br>that indicates from where search should continue |
| query | [string](#string) |  |  |
| ref | [opencloud.messages.search.v0.Reference](/apis/grpc_apis/opencloud_messages_search_v0/#reference) |  |  |

### SearchResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matches | [opencloud.messages.search.v0.Match](/apis/grpc_apis/opencloud_messages_search_v0/#match) | repeated |  |
| next_page_token | [string](#string) |  | Token to retrieve the next page of results, or empty if there are no<br>more results in the list |
| total_matches | [int32](#int32) |  |  |


### IndexProvider



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Search | [SearchIndexRequest](#searchindexrequest) | [SearchIndexResponse](#searchindexresponse) |  |

### SearchProvider



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Search | [SearchRequest](#searchrequest) | [SearchResponse](#searchresponse) |  |
| IndexSpace | [IndexSpaceRequest](#indexspacerequest) | [IndexSpaceResponse](#indexspaceresponse) |  |

## Scalar Value Types

| .proto Type | Notes | C++ | Java |
| ----------- | ----- | --- | ---- |
| {{< div id="double" content="double" >}} |  | double | double |
| {{< div id="float" content="float" >}} |  | float | float |
| {{< div id="int32" content="int32" >}} | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int |
| {{< div id="int64" content="int64" >}} | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long |
| {{< div id="uint32" content="uint32" >}} | Uses variable-length encoding. | uint32 | int |
| {{< div id="uint64" content="uint64" >}} | Uses variable-length encoding. | uint64 | long |
| {{< div id="sint32" content="sint32" >}} | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int |
| {{< div id="sint64" content="sint64" >}} | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long |
| {{< div id="fixed32" content="fixed32" >}} | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int |
| {{< div id="fixed64" content="fixed64" >}} | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long |
| {{< div id="sfixed32" content="sfixed32" >}} | Always four bytes. | int32 | int |
| {{< div id="sfixed64" content="sfixed64" >}} | Always eight bytes. | int64 | long |
| {{< div id="bool" content="bool" >}} |  | bool | boolean |
| {{< div id="string" content="string" >}} | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String |
| {{< div id="bytes" content="bytes" >}} | May contain any arbitrary sequence of bytes. | string | ByteString |

