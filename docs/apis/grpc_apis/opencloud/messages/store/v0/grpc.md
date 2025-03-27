---
title: "opencloud.messages.store.v0"
url: /apis/grpc_apis/opencloud_messages_store_v0
date: 2025-03-27T14:44:40+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/messages/store/v0/store.proto

### DeleteOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  |  |
| table | [string](#string) |  |  |

### Field



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type of value e.g string, int, int64, bool, float64 |
| value | [string](#string) |  | the actual value |

### ListOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  |  |
| table | [string](#string) |  |  |
| prefix | [string](#string) |  |  |
| suffix | [string](#string) |  |  |
| limit | [uint64](#uint64) |  |  |
| offset | [uint64](#uint64) |  |  |

### ReadOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  |  |
| table | [string](#string) |  |  |
| prefix | [bool](#bool) |  |  |
| suffix | [bool](#bool) |  |  |
| limit | [uint64](#uint64) |  |  |
| offset | [uint64](#uint64) |  |  |
| where | [ReadOptions.WhereEntry](#readoptionswhereentry) | repeated |  |

### ReadOptions.WhereEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [Field](#field) |  |  |

### Record



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  | key of the recorda |
| value | [bytes](#bytes) |  | value in the record |
| expiry | [int64](#int64) |  | time.Duration (signed int64 nanoseconds) |
| metadata | [Record.MetadataEntry](#recordmetadataentry) | repeated | the associated metadata |

### Record.MetadataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [Field](#field) |  |  |

### WriteOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  |  |
| table | [string](#string) |  |  |
| expiry | [int64](#int64) |  | time.Time |
| ttl | [int64](#int64) |  | time.Duration |


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

