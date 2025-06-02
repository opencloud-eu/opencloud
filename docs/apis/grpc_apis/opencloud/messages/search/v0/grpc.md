---
title: "opencloud.messages.search.v0"
url: /apis/grpc_apis/opencloud_messages_search_v0
date: 2025-01-21T14:22:28+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/messages/search/v0/search.proto

### Audio



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| album | [string](#string) | optional |  |
| albumArtist | [string](#string) | optional |  |
| artist | [string](#string) | optional |  |
| bitrate | [int64](#int64) | optional |  |
| composers | [string](#string) | optional |  |
| copyright | [string](#string) | optional |  |
| disc | [int32](#int32) | optional |  |
| discCount | [int32](#int32) | optional |  |
| duration | [int64](#int64) | optional |  |
| genre | [string](#string) | optional |  |
| hasDrm | [bool](#bool) | optional |  |
| isVariableBitrate | [bool](#bool) | optional |  |
| title | [string](#string) | optional |  |
| track | [int32](#int32) | optional |  |
| trackCount | [int32](#int32) | optional |  |
| year | [int32](#int32) | optional |  |

### Entity



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ref | [Reference](#reference) |  |  |
| id | [ResourceID](#resourceid) |  |  |
| name | [string](#string) |  |  |
| etag | [string](#string) |  |  |
| size | [uint64](#uint64) |  |  |
| last_modified_time | [google.protobuf.Timestamp](#googleprotobuftimestamp) |  |  |
| mime_type | [string](#string) |  |  |
| permissions | [string](#string) |  |  |
| type | [uint64](#uint64) |  |  |
| deleted | [bool](#bool) |  |  |
| shareRootName | [string](#string) |  |  |
| parent_id | [ResourceID](#resourceid) |  |  |
| tags | [string](#string) | repeated |  |
| highlights | [string](#string) |  |  |
| audio | [Audio](#audio) |  |  |
| location | [GeoCoordinates](#geocoordinates) |  |  |
| remote_item_id | [ResourceID](#resourceid) |  |  |
| image | [Image](#image) |  |  |
| photo | [Photo](#photo) |  |  |

### GeoCoordinates



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| altitude | [double](#double) | optional |  |
| latitude | [double](#double) | optional |  |
| longitude | [double](#double) | optional |  |

### Image



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| width | [int32](#int32) | optional |  |
| height | [int32](#int32) | optional |  |

### Match



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | [Entity](#entity) |  | the matched entity |
| score | [float](#float) |  | the match score |

### Photo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cameraMake | [string](#string) | optional |  |
| cameraModel | [string](#string) | optional |  |
| exposureDenominator | [float](#float) | optional |  |
| exposureNumerator | [float](#float) | optional |  |
| fNumber | [float](#float) | optional |  |
| focalLength | [float](#float) | optional |  |
| iso | [int32](#int32) | optional |  |
| orientation | [int32](#int32) | optional |  |
| takenDateTime | [google.protobuf.Timestamp](#googleprotobuftimestamp) | optional |  |

### Reference



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceID](#resourceid) |  |  |
| path | [string](#string) |  |  |

### ResourceID



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| storage_id | [string](#string) |  |  |
| opaque_id | [string](#string) |  |  |
| space_id | [string](#string) |  |  |


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

