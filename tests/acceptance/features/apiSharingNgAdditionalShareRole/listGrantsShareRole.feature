@env-config
Feature: ListGrants role
  As a user
  I want to share resources with listGrants role
  So that sharee can view activities and grants list of shared resources

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And the administrator has enabled the following share permissions roles:
      | permissions-role        |
      | Viewer List Grants      |
      | File Editor List Grants |
      | Editor List Grants      |


  Scenario: user shares personal resources with ListGrants role
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "to share" to "textfile1.txt"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | textfile1.txt           |
      | space           | Personal                |
      | sharee          | Brian                   |
      | shareType       | user                    |
      | permissionsRole | File Editor List Grants |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": ["createdDateTime","id","roles","grantedToV2"],
              "properties": {
                "createdDateTime": { "format": "date-time" },
                "id": {"pattern": "^%permissions_id_pattern%$"},
                "roles": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "items": {"pattern": "^%role_id_pattern%$"}
                },
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["id","displayName"],
                      "properties": {
                        "id": {"pattern": "^%user_id_pattern%$"},
                        "displayName": {"const": "Brian Murphy"}
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And for user "Brian" file "textfile1.txt" should have the following shares:
      | sharee | shareType | permissionsRole         |
      | Brian  | user      | File Editor List Grants |
    And for user "Brian" file "textfile1.txt" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}    |
      | {user} shared {resource} with {sharee} |
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | FolderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | Editor List Grants |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": ["createdDateTime","id","roles","grantedToV2"],
              "properties": {
                "createdDateTime": { "format": "date-time" },
                "id": {"pattern": "^%permissions_id_pattern%$"},
                "roles": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "items": {"pattern": "^%role_id_pattern%$"}
                },
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["id","displayName"],
                      "properties": {
                        "id": {"pattern": "^%user_id_pattern%$"},
                        "displayName": {"const": "Brian Murphy"}
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And for user "Brian" folder "FolderToShare" should have the following shares:
      | sharee | shareType | permissionsRole    |
      | Brian  | user      | Editor List Grants |
    And for user "Brian" folder "FolderToShare" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}    |
      | {user} shared {resource} with {sharee} |


  Scenario: user shares project resources with ListGrants role
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "textfile1.txt"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | textfile1.txt           |
      | space           | NewSpace                |
      | sharee          | Brian                   |
      | shareType       | user                    |
      | permissionsRole | File Editor List Grants |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": ["createdDateTime","id","roles","grantedToV2"],
              "properties": {
                "createdDateTime": { "format": "date-time" },
                "id": {"pattern": "^%permissions_id_pattern%$"},
                "roles": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "items": {"pattern": "^%role_id_pattern%$"}
                },
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["id","displayName"],
                      "properties": {
                        "id": {"pattern": "^%user_id_pattern%$"},
                        "displayName": {"const": "Brian Murphy"}
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And for user "Brian" file "textfile1.txt" should have the following shares:
      | sharee | shareType | permissionsRole         |
      | Brian  | user      | File Editor List Grants |
    And for user "Brian" file "textfile1.txt" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}    |
      | {user} shared {resource} with {sharee} |
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | FolderToShare      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | Editor List Grants |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": ["createdDateTime","id","roles","grantedToV2"],
              "properties": {
                "createdDateTime": { "format": "date-time" },
                "id": {"pattern": "^%permissions_id_pattern%$"},
                "roles": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "items": {"pattern": "^%role_id_pattern%$"}
                },
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["id","displayName"],
                      "properties": {
                        "id": {"pattern": "^%user_id_pattern%$"},
                        "displayName": {"const": "Brian Murphy"}
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And for user "Brian" folder "FolderToShare" should have the following shares:
      | sharee | shareType | permissionsRole    |
      | Brian  | user      | Editor List Grants |
    And for user "Brian" folder "FolderToShare" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}    |
      | {user} shared {resource} with {sharee} |


  Scenario Outline: sharer updates shared file roles to ListGrants roles (Personal space)
    Given the administrator has enabled the permissions role "<new-permissions-role>"
    And user "Alice" has uploaded file with content "to share" to "textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | Personal               |
      | resource        | textfile1.txt          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" file "textfile1.txt" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" file "textfile1.txt" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}    |
      | {user} shared {resource} with {sharee} |
    Examples:
      | permissions-role | new-permissions-role    |
      | Viewer           | File Editor List Grants |
      | File Editor      | Viewer List Grants      |


  Scenario Outline: sharer updates shared folder roles to ListGrants roles (Personal space)
    Given the administrator has enabled the permissions role "<new-permissions-role>"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | Personal               |
      | resource        | FolderToShare          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" folder "FolderToShare" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" folder "FolderToShare" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}    |
      | {user} shared {resource} with {sharee} |
    Examples:
      | permissions-role | new-permissions-role |
      | Viewer           | Viewer List Grants   |
      | Editor           | Editor List Grants   |


  Scenario Outline: sharer updates shared file roles to ListGrants roles (Project space)
    Given the administrator has enabled the permissions role "<new-permissions-role>"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
      | resource        | textfile1.txt          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" file "textfile1.txt" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" file "textfile1.txt" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}       |
      | {user} shared {resource} with {sharee}    |
      | {user} updated {field} for the {resource} |
    Examples:
      | permissions-role | new-permissions-role    |
      | Viewer           | Viewer List Grants      |
      | File Editor      | File Editor List Grants |


  Scenario Outline: sharer updates shared folder roles to ListGrants roles (Project space)
    Given the administrator has enabled the permissions role "<new-permissions-role>"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
      | resource        | FolderToShare          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" folder "FolderToShare" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" folder "FolderToShare" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}       |
      | {user} shared {resource} with {sharee}    |
      | {user} updated {field} for the {resource} |
    Examples:
      | permissions-role | new-permissions-role |
      | Editor           | Viewer List Grants   |
      | Viewer           | Editor List Grants   |


  Scenario Outline: sharer updates shared file roles from ListGrants roles to other roles (Personal space)
    Given the administrator has enabled the permissions role "<permissions-role>"
    And user "Alice" has uploaded file with content "to share" to "textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | Personal               |
      | resource        | textfile1.txt          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" file "textfile1.txt" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" file "textfile1.txt" of the space "Shares" should not have any activity
    Examples:
      | permissions-role        | new-permissions-role |
      | Viewer List Grants      | Viewer               |
      | File Editor List Grants | File Editor          |


  Scenario Outline: sharer updates shared folder roles from ListGrants roles to other roles (Personal space)
    Given the administrator has enabled the permissions role "<permissions-role>"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | Personal               |
      | resource        | FolderToShare          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" folder "FolderToShare" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" folder "FolderToShare" of the space "Shares" should not have any activity
    Examples:
      | permissions-role   | new-permissions-role |
      | Viewer List Grants | Editor               |
      | Editor List Grants | Viewer               |


  Scenario Outline: sharer updates shared file roles from ListGrants roles to other roles (Project space)
    Given the administrator has enabled the permissions role "<new-permissions-role>"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
      | resource        | textfile1.txt          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" file "textfile1.txt" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" file "textfile1.txt" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}       |
      | {user} shared {resource} with {sharee}    |
      | {user} updated {field} for the {resource} |
    Examples:
      | new-permissions-role    | permissions-role |
      | Viewer List Grants      | File Editor      |
      | File Editor List Grants | Viewer           |


  Scenario Outline: sharer updates shared folder roles to ListGrants roles to other roles (Project space)
    Given the administrator has enabled the permissions role "<new-permissions-role>"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
      | resource        | FolderToShare          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["grantedToV2","id","roles"],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": ["displayName","id"],
                "properties": {
                  "displayName": {"const": "Brian Murphy"},
                  "id": {"pattern": "^%user_id_pattern%$"}
                }
              }
            }
          },
          "id": {"pattern": "^%permissions_id_pattern%$"},
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {"pattern": "^%role_id_pattern%$"}
          }
        }
      }
      """
    And for user "Brian" folder "FolderToShare" should have the following shares:
      | sharee | shareType | permissionsRole        |
      | Brian  | user      | <new-permissions-role> |
    And for user "Brian" folder "FolderToShare" of the space "Shares" should have the following activities:
      | {user} added {resource} to {folder}       |
      | {user} shared {resource} with {sharee}    |
      | {user} updated {field} for the {resource} |
    Examples:
      | new-permissions-role | permissions-role |
      | Viewer List Grants   | Viewer           |
      | Editor List Grants   | Uploader         |


  Scenario: sharer lists shared-by-me (Personal space)
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder             |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | Editor List Grants |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt            |
      | space           | Personal                |
      | sharee          | grp1                    |
      | shareType       | group                   |
      | permissionsRole | File Editor List Grants |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "folder" with the following data:
      """
      {
        "type": "object",
        "required": ["parentReference","permissions","name"],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": ["driveId","driveType","path","name","id"],
            "properties": {
              "driveId": {"pattern": "^%space_id_pattern%$"},
              "driveType": {"const": "personal"},
              "path": {"const": "/"},
              "name": {"const": "/"},
              "id": {"pattern": "^%file_id_pattern%$"}
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": ["grantedToV2","id","roles"],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName","id"],
                      "properties": {
                        "id": {"pattern": "^%user_id_pattern%$"},
                        "displayName": {"const": "Brian Murphy"}
                      }
                    }
                  }
                },
                "id": {"pattern": "^%permissions_id_pattern%$"},
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {"pattern": "^%role_id_pattern%$"}
                }
              }
            }
          },
          "name": {"const": "folder"}
        }
      }
      """
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": ["parentReference","permissions","name","size"],
        "properties": {
          "name": {"const": "textfile.txt"},
          "size": {"const": 8},
          "parentReference": {
            "type": "object",
            "required": ["driveId","driveType","path","name","id"],
            "properties": {
              "driveId": {"pattern": "^%space_id_pattern%$"},
              "driveType": {"const": "personal"},
              "path": {"const": "/"},
              "name": {"const": "/"},
              "id": {"pattern": "^%file_id_pattern%$"}
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": ["createdDateTime","grantedToV2","id","roles","invitation"],
              "properties": {
                "createdDateTime": { "format": "date-time" },
                "id": {"pattern": "^%permissions_id_pattern%$"},
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {"pattern": "^%role_id_pattern%$"}
                },
                "invitation": {
                  "type": "object",
                  "required": ["invitedBy"],
                  "properties": {
                    "invitedBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName", "id", "@libre.graph.userType"],
                          "properties": {
                            "displayName": { "const": "Alice Hansen" },
                            "id": { "pattern": "^%user_id_pattern%$" },
                            "@libre.graph.userType": { "const": "Member" }
                          }
                        }
                      }
                    }
                  }
                },
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": ["displayName","id"],
                      "properties": {
                        "id": {"pattern": "^%group_id_pattern%$"},
                        "displayName": {"const": "grp1"}
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee list shared-with-me (Personal space)
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And  user "Alice" has created folder "folder"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt       |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | Viewer List Grants |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder             |
      | space           | Personal           |
      | sharee          | grp1               |
      | shareType       | group              |
      | permissionsRole | Editor List Grants |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": ["@UI.Hidden","@client.synchronize","createdBy","eTag","file",
          "id","lastModifiedDateTime","name","parentReference","remoteItem","size"],
        "properties": {
          "lastModifiedDateTime": { "format": "date-time" },
          "eTag": {"pattern": "%etag_pattern%"},
          "id": {"pattern": "^%share_id_pattern%$"},
          "name": {"const": "textfile.txt"},
          "remoteItem": {
            "type": "object",
            "required": ["createdBy","eTag","file","id","lastModifiedDateTime","name","parentReference","permissions","size", "spaceId"],
            "properties": {
              "eTag": {"pattern": "%etag_pattern%"},
              "id": {"pattern": "^%file_id_pattern%$"},
              "name": {"const": "textfile.txt"},
              "permissions": {
                "type": "array",
                "minItems": 1,
                "maxItems": 1,
                "uniqueItems": true,
                "items": {
                  "type": "object",
                  "required": ["grantedToV2","id","invitation","roles"],
                  "properties": {
                    "grantedToV2": {
                      "type": "object",
                      "required": ["user"],
                      "properties":{
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "displayName": {"const": "Brian Murphy"},
                            "id": {"pattern": "^%user_id_pattern%$"}
                          }
                        }
                      }
                    },
                    "id": {"pattern": "^%permissions_id_pattern%$"},
                    "invitation": {
                      "type": "object",
                      "required": ["invitedBy"],
                      "properties": {
                        "user":{
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "displayName": {"const": "Alice Hansen"},
                            "id": {"pattern": "^%user_id_pattern%$"}
                          }
                        }
                      }
                    },
                    "roles": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {"pattern": "^%role_id_pattern%$"}
                    }
                  }
                }
              },
              "spaceId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              }
            }
          }
        }
      }
      """
    And the JSON data of the response should contain resource "folder" with the following data:
      """
      {
        "type": "object",
        "required": ["@UI.Hidden","@client.synchronize","createdBy","eTag","folder",
          "id","lastModifiedDateTime","name","parentReference","remoteItem"],
        "properties": {
          "lastModifiedDateTime": { "format": "date-time" },
          "eTag": {"pattern": "%etag_pattern%"},
          "id": {"pattern": "^%share_id_pattern%$"},
          "name": {"const": "folder"},
          "remoteItem": {
            "type": "object",
            "required": ["createdBy","eTag","folder","id","lastModifiedDateTime","name","parentReference","permissions", "spaceId"],
            "properties": {
              "eTag": {"pattern": "%etag_pattern%"},
              "file": {},
              "id": {"pattern": "^%file_id_pattern%$"},
              "name": {"const": "folder"},
              "permissions": {
                "type": "array",
                "minItems": 1,
                "maxItems": 1,
                "uniqueItems": true,
                "items": {
                  "type": "object",
                  "required": ["grantedToV2","id","invitation","roles"],
                  "properties": {
                    "grantedToV2": {
                      "type": "object",
                      "required": ["group"],
                      "properties":{
                        "group": {
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "displayName": {"const": "grp1"},
                            "id": {"pattern": "^%user_id_pattern%$"}
                          }
                        }
                      }
                    },
                    "id": {"pattern": "^%permissions_id_pattern%$"},
                    "invitation": {
                      "type": "object",
                      "required": ["invitedBy"],
                      "properties": {
                        "user":{
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "displayName": {"const": "Alice Hansen"},
                            "id": {"pattern": "^%user_id_pattern%$"}
                          }
                        }
                      }
                    },
                    "roles": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {"pattern": "^%role_id_pattern%$"}
                    }
                  }
                }
              },
              "spaceId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              }
            }
          }
        }
      }
      """


  Scenario Outline: list activities of folder shared with listGrant roles (Personal space)
    Given the administrator has enabled the permissions role "<permissions-role>"
    And using spaces DAV path
    And using SharingNG
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder             |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "folder" synced
    When user "Brian" lists the activities of folder "folder" from space "Shares" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {"pattern": "^%user_id_pattern%$"},
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {"const": "{user} added {resource} to {folder}"},
                        "variables":{
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties":{
                                "id": {"const": ""},
                                "name": {"const": "shared-with-me"}
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {"name": {"const": "folder"}}
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties":{"displayName": {"const": "Alice Hansen"}}
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {"pattern": "^%user_id_pattern%$"},
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {"const": "{user} shared {resource} with {sharee}"},
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","sharee","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {"name": {"const": "folder"}}
                            },
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {"displayName": {"const": "Brian"}}
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {"displayName": {"const": "Alice Hansen"}}
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": { "format": "date-time" }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """
    Examples:
      | permissions-role   |
      | Viewer List Grants |
      | Editor List Grants |


  Scenario: user lists permissions of a folder after enabling 'Viewer List Grants' role (Personal space)
    Given the administrator has enabled the permissions role "Viewer List Grants"
    And user "Alice" has created folder "folder"
    When user "Alice" gets permissions list for folder "folder" of the space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues",
          "@libre.graph.permissions.roles.allowedValues"
        ],
        "properties": {
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 4,
            "maxItems": 4,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 10 },
                    "description": { "const": "View and download." },
                    "displayName": { "const": "Can view" },
                    "id": { "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5" }
                  }
                },
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 30 },
                    "description": { "const": "View, download and show all invited people." },
                    "displayName": { "const": "Can view" },
                    "id": { "const": "d5041006-ebb3-4b4a-b6a4-7c180ecfb17d" }
                  }
                },
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 50 },
                    "description": { "const": "View, download and upload." },
                    "displayName": { "const": "Can upload" },
                    "id": { "const": "1c996275-f1c9-4e71-abdf-a42f6495e960" }
                  }
                },
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 60 },
                    "description": { "const": "View, download, upload, edit, add and delete." },
                    "displayName": { "const": "Can edit" },
                    "id": { "const": "fb6c3e19-e378-47e5-b277-9732f9de6e21" }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: user lists permissions of a file after enabling 'File Editor List Grants' role (Project space)
    Given the administrator has enabled the permissions role "File Editor List Grants"
    And using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile0.txt"
    When user "Alice" gets permissions list for file "textfile0.txt" of the space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues",
          "@libre.graph.permissions.roles.allowedValues"
        ],
        "properties": {
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 10 },
                    "description": { "const": "View and download." },
                    "displayName": { "const": "Can view" },
                    "id": { "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5" }
                  }
                },
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 100 },
                    "description": { "const": "View, download and edit." },
                    "displayName": { "const": "Can edit" },
                    "id": { "const": "2d00ce52-1fc2-4dbc-8b95-a73b73395f5a" }
                  }
                },
                {
                  "type": "object",
                  "required": ["@libre.graph.weight", "description", "displayName", "id"],
                  "properties": {
                    "@libre.graph.weight": { "const": 110 },
                    "description": { "const": "View, download, edit and show all invited people." },
                    "displayName": { "const": "Can edit" },
                    "id": { "const": "c1235aea-d106-42db-8458-7d5610fb0a67" }
                  }
                }
              ]
            }
          }
        }
      }
      """