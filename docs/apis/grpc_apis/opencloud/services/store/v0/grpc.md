---
title: "opencloud.services.store.v0"
url: /apis/grpc_apis/opencloud_services_store_v0
date: 2025-03-27T14:44:40+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/services/store/v0/store.proto

### DatabasesRequest




### DatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [string](#string) | repeated |  |

### DeleteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| options | [opencloud.messages.store.v0.DeleteOptions](/apis/grpc_apis/opencloud_messages_store_v0/#deleteoptions) |  |  |

### DeleteResponse




### ListRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| options | [opencloud.messages.store.v0.ListOptions](/apis/grpc_apis/opencloud_messages_store_v0/#listoptions) |  |  |

### ListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| keys | [string](#string) | repeated |  |

### ReadRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| options | [opencloud.messages.store.v0.ReadOptions](/apis/grpc_apis/opencloud_messages_store_v0/#readoptions) |  |  |

### ReadResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| records | [opencloud.messages.store.v0.Record](/apis/grpc_apis/opencloud_messages_store_v0/#record) | repeated |  |

### TablesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  |  |

### TablesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tables | [string](#string) | repeated |  |

### WriteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record | [opencloud.messages.store.v0.Record](/apis/grpc_apis/opencloud_messages_store_v0/#record) |  |  |
| options | [opencloud.messages.store.v0.WriteOptions](/apis/grpc_apis/opencloud_messages_store_v0/#writeoptions) |  |  |

### WriteResponse





### Store



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Read | [ReadRequest](#readrequest) | [ReadResponse](#readresponse) |  |
| Write | [WriteRequest](#writerequest) | [WriteResponse](#writeresponse) |  |
| Delete | [DeleteRequest](#deleterequest) | [DeleteResponse](#deleteresponse) |  |
| List | [ListRequest](#listrequest) | [ListResponse](#listresponse) stream |  |
| Databases | [DatabasesRequest](#databasesrequest) | [DatabasesResponse](#databasesresponse) |  |
| Tables | [TablesRequest](#tablesrequest) | [TablesResponse](#tablesresponse) |  |

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

