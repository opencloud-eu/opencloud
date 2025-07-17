Feature: Update permission of a share
  As a user
  I want to update resources shared with me
  So that I can have more control over my shares and manage it
  https://docs.opencloud.eu/libre-graph-api/#/drives.permissions/UpdatePermission

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario: user updates expiration date of a share
    Given user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource           | testfile.txt         |
      | space              | Personal             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | Viewer               |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | space              | Personal             |
      | resource           | testfile.txt         |
      | expirationDateTime | 2200-07-15T14:00:00Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expirationDateTime",
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "expirationDateTime": {
            "type": "string",
            "enum": ["2200-07-15T14:00:00Z"]
          },
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%permissions_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """


  Scenario Outline: user removes expiration date of a share
    Given user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource           | <resource>           |
      | space              | Personal             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | Viewer               |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | space              | Personal   |
      | resource           | <resource> |
      | expirationDateTime |            |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%permissions_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | resource     |
      | testfile.txt |
      | folder       |


  Scenario Outline: user updates role of a share
    Given user "Alice" has uploaded file with content "to share" to "/textfile1.txt"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource>         |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | Personal               |
      | resource        | <resource>             |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%permissions_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | resource      | new-permissions-role |
      | Viewer           | textfile1.txt | File Editor          |
      | File Editor      | textfile1.txt | Viewer               |
      | Viewer           | FolderToShare | Uploader             |
      | Viewer           | FolderToShare | Editor               |
      | Editor           | FolderToShare | Viewer               |
      | Editor           | FolderToShare | Uploader             |
      | Uploader         | FolderToShare | Editor               |
      | Uploader         | FolderToShare | Viewer               |


  Scenario Outline: space admin updates role of a member in project space (permissions endpoint)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the space share for user "Brian" with the following using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^u:%user_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Editor         |
      | Manager          | Space Viewer         |


  Scenario Outline: user updates role of a shared resource of project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "textfile1.txt"
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource>         |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
      | resource        | <resource>             |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%permissions_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role | resource      |
      | Viewer           | File Editor          | textfile1.txt |
      | File Editor      | Viewer               | textfile1.txt |
      | Viewer           | Editor               | FolderToShare |
      | Viewer           | Uploader             | FolderToShare |
      | Editor           | Viewer               | FolderToShare |
      | Editor           | Uploader             | FolderToShare |
      | Uploader         | Viewer               | FolderToShare |
      | Uploader         | Editor               | FolderToShare |

  @issue-8905
  Scenario Outline: update role of a shared project space to group with different roles using root endpoint
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | grp1                 |
      | shareType          | group                |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | group                  |
      | sharee             | grp1                   |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "group"
            ],
            "properties":{
              "group": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "grp1"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%group_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^g:%group_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |

  @issue-8905
  Scenario Outline: remove expiration date of a shared project space to group using root endpoint
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | grp1                 |
      | shareType          | group                |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime |          |
      | space              | NewSpace |
      | shareType          | group    |
      | sharee             | grp1     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "group"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "grp1"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^g:%group_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Viewer     |
      | Space Editor     |
      | Space Editor     |
      | Manager          |
      | Manager          |

  @issue-8905
  Scenario Outline: update expiration date of a shared project space to group with different roles using root endpoint
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | grp1                 |
      | shareType          | group                |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime | 2200-07-15T14:00:00Z |
      | space              | NewSpace             |
      | shareType          | group                |
      | sharee             | grp1                 |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expirationDateTime",
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "expirationDateTime": {
            "type": "string",
            "enum": ["2200-07-15T14:00:00Z"]
          },
          "grantedToV2": {
            "type": "object",
            "required": [
              "group"
            ],
            "properties":{
              "group": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "grp1"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%group_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^g:%group_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |


  Scenario Outline: remove expiration date and update role of a shared project space at once to group with different roles at once using root endpoint
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | grp1                 |
      | shareType          | group                |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime |                        |
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | group                  |
      | sharee             | grp1                   |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "group"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "grp1"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^g:%group_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |


  Scenario Outline: update expiration date and role of a shared project space at once to group with different roles at once using root endpoint
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | grp1                 |
      | shareType          | group                |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime | 2200-07-15T14:00:00Z   |
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | group                  |
      | sharee             | grp1                   |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expirationDateTime",
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "expirationDateTime": {
            "type": "string",
            "enum": ["2200-07-15T14:00:00Z"]
          },
          "grantedToV2": {
            "type": "object",
            "required": [
              "group"
            ],
            "properties":{
              "group": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "grp1"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%group_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^g:%group_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |

  @issue-8905
  Scenario Outline: update role of a shared project space to user with different roles using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | user                   |
      | sharee             | Brian                  |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^u:%user_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |

  @issue-8905
  Scenario Outline: remove expiration date of a shared project space to user using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime |                        |
      | space              | NewSpace               |
      | shareType          | user                   |
      | sharee             | Brian                  |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^u:%user_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Viewer     |
      | Space Editor     |
      | Space Editor     |
      | Manager          |
      | Manager          |


  Scenario Outline: update expiration date of a share to user with different roles at once using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime | 2200-07-15T14:00:00Z   |
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | user                   |
      | sharee             | Brian                  |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expirationDateTime",
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "expirationDateTime": {
            "type": "string",
            "enum": ["2200-07-15T14:00:00Z"]
          },
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^u:%user_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |


  Scenario Outline: remove expiration and update role of a share to user with different roles at once using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime |                        |
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | user                   |
      | sharee             | Brian                  |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^u:%user_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |


  Scenario Outline: update expiration date and role of a share to user with different roles at once using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | NewSpace             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | <permissions-role>   |
      | expirationDateTime | tomorrow             |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | expirationDateTime | 2200-07-15T14:00:00Z   |
      | permissionsRole    | <new-permissions-role> |
      | space              | NewSpace               |
      | shareType          | user                   |
      | sharee             | Brian                  |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expirationDateTime",
          "grantedToV2",
          "id",
          "roles"
        ],
        "properties": {
          "expirationDateTime": {
            "type": "string",
            "enum": ["2200-07-15T14:00:00Z"]
          },
          "grantedToV2": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^u:%user_id_pattern%$"
          },
          "roles": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%role_id_pattern%$"
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Viewer         |
      | Manager          | Space Editor         |

  @issue-10768
  Scenario Outline: sharer updates share permissions role of a resource to Secure Viewer without enabling it
    Given user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource>         |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | Secure Viewer |
      | space           | Personal      |
      | resource        | <resource>    |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions",
          "grantedToV2",
          "id",
          "invitation"
        ],
        "properties": {
          "@libre.graph.permissions.actions": {
            "const": [
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/basic/read"
            ]
          },
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties": {
              "user": {
                "type": "object",
                "required": ["displayName", "id"],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "pattern": "^%permissions_id_pattern%$"
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
                    "required": ["displayName", "id"],
                    "properties": {
                      "displayName": {
                        "const": "Alice Hansen"
                      },
                      "id": {
                        "pattern": "^%user_id_pattern%$"
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
    Examples:
      | resource      | permissions-role |
      | folderToShare | Uploader         |
      | folderToShare | Editor           |
      | folderToShare | Viewer           |
      | textfile.txt  | File Editor      |
      | textfile.txt  | Viewer           |

  @issue-10768
  Scenario Outline: sharer updates share permissions role of a resource to Denied without enabling it
    Given user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last resource share with the following properties using the Graph API:
      | permissionsRole | Denied        |
      | space           | Personal      |
      | resource        | folderToShare |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions",
          "grantedToV2",
          "id",
          "invitation"
        ],
        "properties": {
          "@libre.graph.permissions.actions": {
            "const": ["none"]
          },
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties": {
              "user": {
                "type": "object",
                "required": ["displayName", "id"],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "pattern": "^%permissions_id_pattern%$"
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
                    "required": ["displayName", "id"],
                    "properties": {
                      "displayName": {
                        "const": "Alice Hansen"
                      },
                      "id": {
                        "pattern": "^%user_id_pattern%$"
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
    Examples:
      | permissions-role |
      | Uploader         |
      | Editor           |
      | Viewer           |

  @issue-10768
  Scenario Outline: sharer updates share permissions role of space to Space Editor Without Versions without enabling it
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last drive share with the following using root endpoint of the Graph API:
      | permissionsRole | Space Editor Without Versions |
      | space           | new-space                     |
      | shareType       | user                          |
      | sharee          | Brian                         |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions",
          "grantedToV2",
          "id"
        ],
        "properties": {
          "@libre.graph.permissions.actions": {
            "const": [
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read"
            ]
          },
          "grantedToV2": {
            "type": "object",
            "required": ["user"],
            "properties":{
              "user": {
                "type": "object",
                "required": [
                  "displayName",
                  "id"
                ],
                "properties": {
                  "displayName": {
                    "const": "Brian Murphy"
                  },
                  "id": {
                    "pattern": "^%user_id_pattern%$"
                  }
                }
              }
            }
          },
          "id": {
            "pattern": "^u:%user_id_pattern%$"
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |
