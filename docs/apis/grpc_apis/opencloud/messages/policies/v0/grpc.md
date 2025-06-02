---
title: "opencloud.messages.policies.v0"
url: /apis/grpc_apis/opencloud_messages_policies_v0
date: 2025-01-21T14:22:28+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/messages/policies/v0/policies.proto

### Environment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [Stage](#stage) |  |  |
| user | [User](#user) |  |  |
| request | [Request](#request) |  |  |
| resource | [Resource](#resource) |  |  |

### Request



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  |  |
| path | [string](#string) |  |  |

### Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Resource.ID](#resourceid) |  |  |
| name | [string](#string) |  |  |
| size | [uint64](#uint64) |  |  |
| url | [string](#string) |  |  |

### Resource.ID



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| storage_id | [string](#string) |  |  |
| opaque_id | [string](#string) |  |  |
| space_id | [string](#string) |  |  |

### User



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [User.ID](#userid) |  |  |
| username | [string](#string) |  |  |
| mail | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| groups | [string](#string) | repeated |  |

### User.ID



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| opaque_id | [string](#string) |  |  |

### Stage



| Name | Number | Description |
| ---- | ------ | ----------- |
| STAGE_PP | 0 |  |
| STAGE_HTTP | 1 |  |

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

