---
title: "opencloud.services.settings.v0"
url: /apis/grpc_apis/opencloud_services_settings_v0
date: 2025-01-21T14:22:28+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/services/settings/v0/settings.proto

### AddSettingToBundleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle_id | [string](#string) |  |  |
| setting | [opencloud.messages.settings.v0.Setting](/apis/grpc_apis/opencloud_messages_settings_v0/#setting) |  |  |

### AddSettingToBundleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [opencloud.messages.settings.v0.Setting](/apis/grpc_apis/opencloud_messages_settings_v0/#setting) |  |  |

### AssignRoleToUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_uuid | [string](#string) |  |  |
| role_id | [string](#string) |  | the role_id is a bundle_id internally |

### AssignRoleToUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| assignment | [opencloud.messages.settings.v0.UserRoleAssignment](/apis/grpc_apis/opencloud_messages_settings_v0/#userroleassignment) |  |  |

### GetBundleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle_id | [string](#string) |  |  |

### GetBundleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle | [opencloud.messages.settings.v0.Bundle](/apis/grpc_apis/opencloud_messages_settings_v0/#bundle) |  |  |

### GetPermissionByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| permission_id | [string](#string) |  |  |

### GetPermissionByIDResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| permission | [opencloud.messages.settings.v0.Permission](/apis/grpc_apis/opencloud_messages_settings_v0/#permission) |  |  |

### GetValueByUniqueIdentifiersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_uuid | [string](#string) |  |  |
| setting_id | [string](#string) |  |  |

### GetValueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |

### GetValueResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [opencloud.messages.settings.v0.ValueWithIdentifier](/apis/grpc_apis/opencloud_messages_settings_v0/#valuewithidentifier) |  |  |

### ListBundlesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle_ids | [string](#string) | repeated |  |
| locale | [string](#string) |  |  |

### ListBundlesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundles | [opencloud.messages.settings.v0.Bundle](/apis/grpc_apis/opencloud_messages_settings_v0/#bundle) | repeated |  |

### ListPermissionsByResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [opencloud.messages.settings.v0.Resource](/apis/grpc_apis/opencloud_messages_settings_v0/#resource) |  |  |

### ListPermissionsByResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| permissions | [opencloud.messages.settings.v0.Permission](/apis/grpc_apis/opencloud_messages_settings_v0/#permission) | repeated |  |

### ListPermissionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_uuid | [string](#string) |  |  |

### ListPermissionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| permissions | [string](#string) | repeated |  |

### ListRoleAssignmentsFilteredRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filters | [opencloud.messages.settings.v0.UserRoleAssignmentFilter](/apis/grpc_apis/opencloud_messages_settings_v0/#userroleassignmentfilter) | repeated |  |

### ListRoleAssignmentsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_uuid | [string](#string) |  |  |

### ListRoleAssignmentsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| assignments | [opencloud.messages.settings.v0.UserRoleAssignment](/apis/grpc_apis/opencloud_messages_settings_v0/#userroleassignment) | repeated |  |

### ListValuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle_id | [string](#string) |  |  |
| account_uuid | [string](#string) |  |  |

### ListValuesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| values | [opencloud.messages.settings.v0.ValueWithIdentifier](/apis/grpc_apis/opencloud_messages_settings_v0/#valuewithidentifier) | repeated |  |

### RemoveRoleFromUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |

### RemoveSettingFromBundleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle_id | [string](#string) |  |  |
| setting_id | [string](#string) |  |  |

### SaveBundleRequest

---
requests and responses for settings bundles
---

| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle | [opencloud.messages.settings.v0.Bundle](/apis/grpc_apis/opencloud_messages_settings_v0/#bundle) |  |  |

### SaveBundleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bundle | [opencloud.messages.settings.v0.Bundle](/apis/grpc_apis/opencloud_messages_settings_v0/#bundle) |  |  |

### SaveValueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [opencloud.messages.settings.v0.Value](/apis/grpc_apis/opencloud_messages_settings_v0/#value) |  |  |

### SaveValueResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [opencloud.messages.settings.v0.ValueWithIdentifier](/apis/grpc_apis/opencloud_messages_settings_v0/#valuewithidentifier) |  |  |


### BundleService



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SaveBundle | [SaveBundleRequest](#savebundlerequest) | [SaveBundleResponse](#savebundleresponse) |  |
| GetBundle | [GetBundleRequest](#getbundlerequest) | [GetBundleResponse](#getbundleresponse) |  |
| ListBundles | [ListBundlesRequest](#listbundlesrequest) | [ListBundlesResponse](#listbundlesresponse) |  |
| AddSettingToBundle | [AddSettingToBundleRequest](#addsettingtobundlerequest) | [AddSettingToBundleResponse](#addsettingtobundleresponse) |  |
| RemoveSettingFromBundle | [RemoveSettingFromBundleRequest](#removesettingfrombundlerequest) | [.google.protobuf.Empty](#googleprotobufempty) |  |

### PermissionService



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListPermissions | [ListPermissionsRequest](#listpermissionsrequest) | [ListPermissionsResponse](#listpermissionsresponse) |  |
| ListPermissionsByResource | [ListPermissionsByResourceRequest](#listpermissionsbyresourcerequest) | [ListPermissionsByResourceResponse](#listpermissionsbyresourceresponse) |  |
| GetPermissionByID | [GetPermissionByIDRequest](#getpermissionbyidrequest) | [GetPermissionByIDResponse](#getpermissionbyidresponse) |  |

### RoleService



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListBundlesRequest](#listbundlesrequest) | [ListBundlesResponse](#listbundlesresponse) |  |
| ListRoleAssignments | [ListRoleAssignmentsRequest](#listroleassignmentsrequest) | [ListRoleAssignmentsResponse](#listroleassignmentsresponse) |  |
| ListRoleAssignmentsFiltered | [ListRoleAssignmentsFilteredRequest](#listroleassignmentsfilteredrequest) | [ListRoleAssignmentsResponse](#listroleassignmentsresponse) |  |
| AssignRoleToUser | [AssignRoleToUserRequest](#assignroletouserrequest) | [AssignRoleToUserResponse](#assignroletouserresponse) |  |
| RemoveRoleFromUser | [RemoveRoleFromUserRequest](#removerolefromuserrequest) | [.google.protobuf.Empty](#googleprotobufempty) |  |

### ValueService



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SaveValue | [SaveValueRequest](#savevaluerequest) | [SaveValueResponse](#savevalueresponse) |  |
| GetValue | [GetValueRequest](#getvaluerequest) | [GetValueResponse](#getvalueresponse) |  |
| ListValues | [ListValuesRequest](#listvaluesrequest) | [ListValuesResponse](#listvaluesresponse) |  |
| GetValueByUniqueIdentifiers | [GetValueByUniqueIdentifiersRequest](#getvaluebyuniqueidentifiersrequest) | [GetValueResponse](#getvalueresponse) |  |

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

