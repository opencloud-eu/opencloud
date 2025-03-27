---
title: "opencloud.services.thumbnails.v0"
url: /apis/grpc_apis/opencloud_services_thumbnails_v0
date: 2025-03-27T14:44:40+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/services/thumbnails/v0/thumbnails.proto

### GetThumbnailRequest

A request to retrieve a thumbnail

| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filepath | [string](#string) |  | The path to the source image |
| thumbnail_type | [opencloud.messages.thumbnails.v0.ThumbnailType](/apis/grpc_apis/opencloud_messages_thumbnails_v0/#thumbnailtype) |  | The type to which the thumbnail should get encoded to. |
| width | [int32](#int32) |  | The width of the thumbnail |
| height | [int32](#int32) |  | The height of the thumbnail |
| processor | [string](#string) |  | Indicates which image processor to use |
| webdav_source | [opencloud.messages.thumbnails.v0.WebdavSource](/apis/grpc_apis/opencloud_messages_thumbnails_v0/#webdavsource) |  |  |
| cs3_source | [opencloud.messages.thumbnails.v0.CS3Source](/apis/grpc_apis/opencloud_messages_thumbnails_v0/#cs3source) |  |  |

### GetThumbnailResponse

The service response

| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data_endpoint | [string](#string) |  | The endpoint where the thumbnail can be downloaded. |
| transfer_token | [string](#string) |  | The transfer token to be able to download the thumbnail. |
| mimetype | [string](#string) |  | The mimetype of the thumbnail |


### ThumbnailService

A Service for handling thumbnail generation

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetThumbnail | [GetThumbnailRequest](#getthumbnailrequest) | [GetThumbnailResponse](#getthumbnailresponse) | Generates the thumbnail and returns it. |

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

