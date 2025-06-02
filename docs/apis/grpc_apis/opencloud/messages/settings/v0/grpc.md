---
title: "opencloud.messages.settings.v0"
url: /apis/grpc_apis/opencloud_messages_settings_v0
date: 2025-01-21T14:22:28+01:00
weight: 50
geekdocRepo: https://github.com/opencloud-eu/opencloud
---

{{< toc >}}



## opencloud/messages/settings/v0/settings.proto

### Bool



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| default | [bool](#bool) |  | @gotags: yaml:"default" |
| label | [string](#string) |  | @gotags: yaml:"label" |

### Bundle



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | @gotags: yaml:"id" |
| name | [string](#string) |  | @gotags: yaml:"name" |
| type | [Bundle.Type](#bundletype) |  | @gotags: yaml:"type" |
| extension | [string](#string) |  | @gotags: yaml:"extension" |
| display_name | [string](#string) |  | @gotags: yaml:"display_name" |
| settings | [Setting](#setting) | repeated | @gotags: yaml:"settings" |
| resource | [Resource](#resource) |  | @gotags: yaml:"resource" |

### CollectionOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  | required<br><br>@gotags: yaml:"key" |
| int_value | [int64](#int64) |  | @gotags: yaml:"int_value" |
| string_value | [string](#string) |  | @gotags: yaml:"string_value" |
| bool_value | [bool](#bool) |  | @gotags: yaml:"bool_value" |

### CollectionValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| values | [CollectionOption](#collectionoption) | repeated | @gotags: yaml:"values" |

### Identifier



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| extension | [string](#string) |  |  |
| bundle | [string](#string) |  |  |
| setting | [string](#string) |  |  |

### Int



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| default | [int64](#int64) |  | @gotags: yaml:"default" |
| min | [int64](#int64) |  | @gotags: yaml:"min" |
| max | [int64](#int64) |  | @gotags: yaml:"max" |
| step | [int64](#int64) |  | @gotags: yaml:"step" |
| placeholder | [string](#string) |  | @gotags: yaml:"placeholder" |

### ListOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [ListOptionValue](#listoptionvalue) |  | @gotags: yaml:"value" |
| default | [bool](#bool) |  | @gotags: yaml:"default" |
| display_value | [string](#string) |  | @gotags: yaml:"display_value" |

### ListOptionValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| string_value | [string](#string) |  | @gotags: yaml:"string_value" |
| int_value | [int64](#int64) |  | @gotags: yaml:"int_value" |
| bool_value | [bool](#bool) |  | @gotags: yaml:"bool_value" |

### ListValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| values | [ListOptionValue](#listoptionvalue) | repeated | @gotags: yaml:"values" |

### MultiChoiceCollection



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| options | [MultiChoiceCollectionOption](#multichoicecollectionoption) | repeated | @gotags: yaml:"options" |

### MultiChoiceCollectionOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [MultiChoiceCollectionOptionValue](#multichoicecollectionoptionvalue) |  | @gotags: yaml:"value" |
| key | [string](#string) |  | @gotags: yaml:"key" |
| attribute | [string](#string) |  | @gotags: yaml:"attribute" |
| display_value | [string](#string) |  | @gotags: yaml:"display_value" |

### MultiChoiceCollectionOptionValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| int_value | [Int](#int) |  | @gotags: yaml:"int_value" |
| string_value | [String](#string) |  | @gotags: yaml:"string_value" |
| bool_value | [Bool](#bool) |  | @gotags: yaml:"bool_value" |

### MultiChoiceList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| options | [ListOption](#listoption) | repeated | @gotags: yaml:"options" |

### Permission



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| operation | [Permission.Operation](#permissionoperation) |  | @gotags: yaml:"operation" |
| constraint | [Permission.Constraint](#permissionconstraint) |  | @gotags: yaml:"constraint" |

### Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Resource.Type](#resourcetype) |  |  |
| id | [string](#string) |  |  |

### Setting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | @gotags: yaml:"id" |
| name | [string](#string) |  | @gotags: yaml:"name" |
| display_name | [string](#string) |  | @gotags: yaml:"display_name" |
| description | [string](#string) |  | @gotags: yaml:"description" |
| int_value | [Int](#int) |  | @gotags: yaml:"int_value" |
| string_value | [String](#string) |  | @gotags: yaml:"string_value" |
| bool_value | [Bool](#bool) |  | @gotags: yaml:"bool_value" |
| single_choice_value | [SingleChoiceList](#singlechoicelist) |  | @gotags: yaml:"single_choice_value" |
| multi_choice_value | [MultiChoiceList](#multichoicelist) |  | @gotags: yaml:"multi_choice_value" |
| permission_value | [Permission](#permission) |  | @gotags: yaml:"permission_value" |
| multi_choice_collection_value | [MultiChoiceCollection](#multichoicecollection) |  | @gotags: yaml:"multi_choice_collection_value" |
| resource | [Resource](#resource) |  | @gotags: yaml:"resource" |

### SingleChoiceList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| options | [ListOption](#listoption) | repeated | @gotags: yaml:"options" |

### String



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| default | [string](#string) |  | @gotags: yaml:"default" |
| required | [bool](#bool) |  | @gotags: yaml:"required" |
| min_length | [int32](#int32) |  | @gotags: yaml:"min_length" |
| max_length | [int32](#int32) |  | @gotags: yaml:"max_length" |
| placeholder | [string](#string) |  | @gotags: yaml:"placeholder" |

### UserRoleAssignment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is generated upon saving the assignment |
| account_uuid | [string](#string) |  |  |
| role_id | [string](#string) |  | the role_id is a bundle_id internally |

### UserRoleAssignmentFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [UserRoleAssignmentFilter.Type](#userroleassignmentfiltertype) |  |  |
| account_uuid | [string](#string) |  |  |
| role_id | [string](#string) |  |  |

### Value



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the id of the Value. It is generated on saving it.<br><br>@gotags: yaml:"id" |
| bundle_id | [string](#string) |  | @gotags: yaml:"bundle_id" |
| setting_id | [string](#string) |  | setting_id is the id of the setting from within its bundle.<br><br>@gotags: yaml:"setting_id" |
| account_uuid | [string](#string) |  | @gotags: yaml:"account_uuid" |
| resource | [Resource](#resource) |  | @gotags: yaml:"resource" |
| bool_value | [bool](#bool) |  | @gotags: yaml:"bool_value" |
| int_value | [int64](#int64) |  | @gotags: yaml:"int_value" |
| string_value | [string](#string) |  | @gotags: yaml:"string_value" |
| list_value | [ListValue](#listvalue) |  | @gotags: yaml:"list_value" |
| collection_value | [CollectionValue](#collectionvalue) |  | @gotags: yaml:"collection_value" |

### ValueWithIdentifier



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [Identifier](#identifier) |  |  |
| value | [Value](#value) |  |  |

### Bundle.Type



| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNKNOWN | 0 |  |
| TYPE_DEFAULT | 1 |  |
| TYPE_ROLE | 2 |  |
### Permission.Constraint



| Name | Number | Description |
| ---- | ------ | ----------- |
| CONSTRAINT_UNKNOWN | 0 |  |
| CONSTRAINT_OWN | 1 |  |
| CONSTRAINT_SHARED | 2 |  |
| CONSTRAINT_ALL | 3 |  |
### Permission.Operation



| Name | Number | Description |
| ---- | ------ | ----------- |
| OPERATION_UNKNOWN | 0 |  |
| OPERATION_CREATE | 1 |  |
| OPERATION_READ | 2 |  |
| OPERATION_UPDATE | 3 |  |
| OPERATION_DELETE | 4 |  |
| OPERATION_WRITE | 5 | WRITE is a combination of CREATE and UPDATE |
| OPERATION_READWRITE | 6 | READWRITE is a combination of READ and WRITE |
### Resource.Type



| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNKNOWN | 0 |  |
| TYPE_SYSTEM | 1 |  |
| TYPE_FILE | 2 |  |
| TYPE_SHARE | 3 |  |
| TYPE_SETTING | 4 |  |
| TYPE_BUNDLE | 5 |  |
| TYPE_USER | 6 |  |
| TYPE_GROUP | 7 |  |
### UserRoleAssignmentFilter.Type



| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNKNOWN | 0 |  |
| TYPE_ACCOUNT | 1 |  |
| TYPE_ROLE | 2 |  |

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

