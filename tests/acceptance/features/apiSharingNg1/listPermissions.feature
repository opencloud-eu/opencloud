Feature: List a sharing permissions
  https://docs.opencloud.eu/libre-graph-api/#/drives.permissions/ListPermissions

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |


  Scenario: user lists permissions of a folder in personal space
    Given user "Alice" has created folder "folder"
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
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 10
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 50
                    },
                    "description": {
                      "const": "View, download and upload."
                    },
                    "displayName": {
                      "const": "Can upload"
                    },
                    "id": {
                      "const": "1c996275-f1c9-4e71-abdf-a42f6495e960"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 60
                    },
                    "description": {
                      "const": "View, download, upload, edit, add and delete."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "fb6c3e19-e378-47e5-b277-9732f9de6e21"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: user lists permissions of a project space
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" lists the permissions of space "new-space" using permissions endpoint of the Graph API
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
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8352
  Scenario Outline: sharer lists permissions of a shared project space
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | permissionsRole | view      |
      | password        | %public%  |
    When user "Alice" lists the permissions of space "new-space" using permissions endpoint of the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues",
          "@libre.graph.permissions.roles.allowedValues",
          "value"
        ],
        "properties": {
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          },
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":[
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "const": "Brian Murphy"
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
                },
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "const": "Alice Hansen"
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
                },
                {
                  "type": "object",
                  "required": [
                    "hasPassword",
                    "id",
                    "link"
                  ],
                  "properties": {
                    "hasPassword": {
                      "const": true
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^[a-zA-Z]{15}$"
                    },
                    "link": {
                      "type": "object",
                      "required": [
                        "@libre.graph.displayName",
                        "@libre.graph.quickLink",
                        "preventsDownload",
                        "type",
                        "webUrl"
                      ],
                      "properties": {
                        "@libre.graph.displayName": {
                          "const": ""
                        },
                        "@libre.graph.quickLink": {
                          "const": false
                        },
                        "preventsDownload": {
                          "const": false
                        },
                        "type": {
                          "const": "view"
                        },
                        "webUrl": {
                          "type": "string",
                          "pattern": "^%base_url%\/s\/[a-zA-Z]{15}$"
                        }
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
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |

  @issues-8331
  Scenario: user lists permissions of a file in personal space
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" gets permissions list for file "textfile0.txt" of the space "Personal" using the Graph API
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
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf":[
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 10
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 100
                    },
                    "description": {
                      "const": "View, download and edit."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "2d00ce52-1fc2-4dbc-8b95-a73b73395f5a"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8331
  Scenario: user lists permissions of a folder in project space
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    When user "Alice" gets permissions list for folder "folder" of the space "new-space" using the Graph API
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
            "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
            },
            "@libre.graph.permissions.roles.allowedValues": {
              "type": "array",
              "minItems": 3,
              "maxItems": 3,
              "uniqueItems": true,
              "items": {
                "oneOf":[
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 10
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 50
                    },
                    "description": {
                      "const": "View, download and upload."
                    },
                    "displayName": {
                      "const": "Can upload"
                    },
                    "id": {
                      "const": "1c996275-f1c9-4e71-abdf-a42f6495e960"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 60
                    },
                    "description": {
                      "const": "View, download, upload, edit, add and delete."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "fb6c3e19-e378-47e5-b277-9732f9de6e21"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8331
  Scenario: user lists permissions of a file in project space
    Given using spaces DAV path
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
            "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
            },
            "@libre.graph.permissions.roles.allowedValues": {
              "type": "array",
              "minItems": 2,
              "maxItems": 2,
              "uniqueItems": true,
              "items": {
                "oneOf":[
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 10
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 100
                    },
                    "description": {
                      "const": "View, download and edit."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "2d00ce52-1fc2-4dbc-8b95-a73b73395f5a"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8331
  Scenario: user sends share invitation with all allowed roles for a file
    Given user "Alice" has uploaded file with content "hello text" to "textfile.txt"
    And user "Brian" has been created with default attributes
    When user "Alice" gets permissions list for file "textfile.txt" of the space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And user "Alice" should be able to send the following resource share invitation with all allowed permission roles
      | resource     | textfile.txt |
      | space        | Personal     |
      | sharee       | Brian        |
      | shareType    | user         |

  @issues-8331
  Scenario: user sends share invitation with all allowed roles for a folder
    Given user "Alice" has created folder "folder"
    And user "Brian" has been created with default attributes
    When user "Alice" gets permissions list for folder "folder" of the space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And user "Alice" should be able to send the following resource share invitation with all allowed permission roles
      | resource     | folder   |
      | space        | Personal |
      | sharee       | Brian    |
      | shareType    | user     |

  @issues-8351
  Scenario: user lists permissions of a project space using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" lists the permissions of space "new-space" using root endpoint of the Graph API
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
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: try to lists the permissions of a Personal drive using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" tries to list the permissions of space "Personal" using root endpoint of the Graph API
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
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: try to lists the permissions of a Shares drive using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" tries to list the permissions of space "Shares" using root endpoint of the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.roles.allowedValues"
        ],
        "properties": {
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 0,
            "maxItems": 0
          }
        }
      }
      """


  Scenario: space admin invites to a project space with all allowed roles
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Brian" has been created with default attributes
    When user "Alice" lists the permissions of space "new-space" using permissions endpoint of the Graph API
    Then the HTTP status code should be "200"
    And user "Alice" should be able to send the following resource share invitation with all allowed permission roles
      | space        | new-space    |
      | sharee       | Brian        |
      | shareType    | user         |


  Scenario: user sends share invitation with all allowed roles for a file in project space
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile.txt"
    And user "Brian" has been created with default attributes
    When user "Alice" gets permissions list for file "textfile.txt" of the space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And user "Alice" should be able to send the following resource share invitation with all allowed permission roles
      | resource     | textfile.txt |
      | space        | new-space    |
      | sharee       | Brian        |
      | shareType    | user         |


  Scenario: non-member user tries to list the permissions of a project space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Brian" tries to list the permissions of space "new-space" owned by "Alice" using permissions endpoint of the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code": {
                "const": "itemNotFound"
              },
              "innererror": {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message": {
                "type": "string",
                "pattern": "stat: error: not found: %file_id_pattern%$"
              }
            }
          }
        }
      }
      """

  @issues-8331
  Scenario: user sends share invitation with all allowed roles for a folder in project space
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Brian" has been created with default attributes
    When user "Alice" gets permissions list for folder "folder" of the space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And user "Alice" should be able to send the following resource share invitation with all allowed permission roles
      | resource  | folder    |
      | space     | new-space |
      | sharee    | Brian     |
      | shareType | user      |


  Scenario: try to list the permissions of other user's personal space
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    When user "Brian" tries to list the permissions of space "Personal" owned by "Alice" using permissions endpoint of the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code": {
                "const": "itemNotFound"
              },
              "innererror": {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message": {
                "type": "string",
                "pattern": "stat: error: not found: %file_id_pattern%$"
              }
            }
          }
        }
      }
      """


  Scenario Outline: sharer lists permissions of a shared project space using root endpoint
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | permissionsRole | view      |
      | password        | %public%  |
    When user "Alice" lists the permissions of space "new-space" using root endpoint of the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues",
          "@libre.graph.permissions.roles.allowedValues",
          "value"
        ],
        "properties": {
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          },
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":[
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "const": "Brian Murphy"
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
                },
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "const": "Alice Hansen"
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
                },
                {
                  "type": "object",
                  "required": [
                    "hasPassword",
                    "id",
                    "link"
                  ],
                  "properties": {
                    "hasPassword": {
                      "const": true
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^[a-zA-Z]{15}$"
                    },
                    "link": {
                      "type": "object",
                      "required": [
                        "@libre.graph.displayName",
                        "@libre.graph.quickLink",
                        "preventsDownload",
                        "type",
                        "webUrl"
                      ],
                      "properties": {
                        "@libre.graph.displayName": {
                          "const": ""
                        },
                        "@libre.graph.quickLink": {
                          "const": false
                        },
                        "preventsDownload": {
                          "const": false
                        },
                        "type": {
                          "const": "view"
                        },
                        "webUrl": {
                          "type": "string",
                          "pattern": "^%base_url%\/s\/[a-zA-Z]{15}$"
                        }
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
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |


  Scenario: user sends share invitation with all allowed roles for a project space using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Brian" has been created with default attributes
    When user "Alice" lists the permissions of space "new-space" using root endpoint of the Graph API
    Then the HTTP status code should be "200"
    And user "Alice" should be able to send the following space share invitation with all allowed permission roles using root endpoint of the Graph API
      | space     | new-space |
      | sharee    | Brian     |
      | shareType | user      |

  @issue-9151
  Scenario: non-member user tries to list the permissions of a project space using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Brian" tries to list the permissions of space "new-space" owned by "Alice" using root endpoint of the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code": {
                "const": "itemNotFound"
              },
              "innererror": {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message": {
                "const": "getting space"
              }
            }
          }
        }
      }
      """

  @issue-8922
  Scenario: user lists the permissions of Shares drive using permissions endpoint
    When user "Alice" lists the permissions of space "Shares" using permissions endpoint of the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.roles.allowedValues"
        ],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 0,
            "maxItems": 0
          }
        }
      }
      """

  @issue-8922
  Scenario: list the permissions of a Personal drive using permissions endpoint
    When user "Alice" lists the permissions of space "Personal" using permissions endpoint of the Graph API
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
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          },
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8428
  Scenario: user lists permissions of a shared folder in personal space
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "folder" synced
    And user "Alice" has created the following resource link share:
      | resource        | folder   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When user "Alice" gets permissions list for folder "folder" of the space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues",
          "@libre.graph.permissions.roles.allowedValues",
          "value"
        ],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf":[
                {
                  "type": "object",
                  "required": [
                    "grantedToV2",
                    "id",
                    "invitation",
                    "roles"
                  ],
                  "properties": {
                    "id": {
                      "type": "string",
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
                              "required": ["displayName","id"],
                              "properties": {
                                "displayName": {
                                  "const": "Alice Hansen"
                                },
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["hasPassword", "id", "link"],
                  "properties": {
                    "hasPassword": {
                      "const": true
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^[a-zA-Z]{15}$"
                    },
                    "link": {
                      "type": "object",
                      "required": [
                        "@libre.graph.displayName",
                        "@libre.graph.quickLink",
                        "preventsDownload",
                        "type",
                        "webUrl"
                      ]
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8428
  Scenario: user lists permissions of a shared file in personal space
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And user "Alice" has created the following resource link share:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | permissionsRole | view          |
      | password        | %public%      |
    When user "Alice" gets permissions list for file "textfile0.txt" of the space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues",
          "@libre.graph.permissions.roles.allowedValues",
          "value"
        ],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf":[
                {
                  "type": "object",
                  "required": [
                    "grantedToV2",
                    "id",
                    "invitation",
                    "roles"
                  ],
                  "properties": {
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
                              "required": ["displayName","id"],
                              "properties": {
                                "displayName": {
                                  "const": "Alice Hansen"
                                },
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["hasPassword", "id", "link"],
                  "properties": {
                    "hasPassword": {
                      "const": true
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^[a-zA-Z]{15}$"
                    },
                    "link": {
                      "type": "object",
                      "required": [
                        "@libre.graph.displayName",
                        "@libre.graph.quickLink",
                        "preventsDownload",
                        "type",
                        "webUrl"
                      ]
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issues-8428
  Scenario: user lists permissions of a shared folder in project space
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Brian" has a share "folder" synced
    And user "Alice" has created the following resource link share:
      | resource        | folder    |
      | space           | new-space |
      | permissionsRole | view      |
      | password        | %public%  |
    When user "Alice" gets permissions list for folder "folder" of the space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
          "required": [
            "@libre.graph.permissions.actions.allowedValues",
            "@libre.graph.permissions.roles.allowedValues",
            "value"
          ],
          "properties": {
            "value": {
              "type": "array",
              "minItems": 2,
              "maxItems": 2,
              "uniqueItems": true,
              "items": {
                "oneOf":[
                  {
                    "type": "object",
                    "required": [
                      "grantedToV2",
                      "id",
                      "invitation",
                      "roles"
                    ],
                    "properties": {
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
                                "required": ["displayName","id"],
                                "properties": {
                                  "displayName": {
                                    "const": "Alice Hansen"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  },
                  {
                    "type": "object",
                    "required": [
                      "hasPassword",
                      "id",
                      "link"
                    ],
                    "properties": {
                      "hasPassword": {
                        "const": true
                      },
                      "id": {
                        "type": "string",
                        "pattern": "^[a-zA-Z]{15}$"
                      },
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ]
                      }
                    }
                  }
                ]
              }
            }
          }
      }
      """

  @issues-8428
  Scenario: user lists permissions of a shared file in project space
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | new-space     |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And user "Alice" has created the following resource link share:
      | resource        | textfile0.txt |
      | space           | new-space     |
      | permissionsRole | view          |
      | password        | %public%      |
    When user "Alice" gets permissions list for file "textfile0.txt" of the space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
          "required": [
            "@libre.graph.permissions.actions.allowedValues",
            "@libre.graph.permissions.roles.allowedValues",
            "value"
          ],
          "properties": {
            "value": {
              "type": "array",
              "minItems": 2,
              "maxItems": 2,
              "uniqueItems": true,
              "items": {
                "oneOf":[
                  {
                    "type": "object",
                    "required": [
                      "grantedToV2",
                      "id",
                      "invitation",
                      "roles"
                    ],
                    "properties": {
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
                                "required": ["displayName","id"],
                                "properties": {
                                  "displayName": {
                                    "const": "Alice Hansen"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  },
                  {
                    "type": "object",
                    "required": [
                      "hasPassword",
                      "id",
                      "link"
                    ],
                    "properties": {
                      "hasPassword": {
                        "const": true
                      },
                      "id": {
                        "type": "string",
                        "pattern": "^[a-zA-Z]{15}$"
                      },
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ]
                      }
                    }
                  }
                ]
              }
            }
          }
      }
      """

  @env-config
  Scenario: user lists permissions of a folder in personal space after enabling secure viewer role
    Given user "Alice" has created folder "folder"
    And the administrator has enabled the permissions role "Secure Viewer"
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
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 20
                    },
                    "description": {
                      "const": "View only documents, images and PDFs. Watermarks will be applied."
                    },
                    "displayName": {
                      "const": "Can view (secure)"
                    },
                    "id": {
                      "const": "aa97fe03-7980-45ac-9e50-b325749fd7e6"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can view"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can upload"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can edit"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @env-config
  Scenario: user lists permissions of a space after enabling 'Space Editor Without Versions' role
    Given the administrator has enabled the permissions role "Space Editor Without Versions"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" lists the permissions of space "new-space" using root endpoint of the Graph API
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
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can view"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 80
                    },
                    "description": {
                      "const": "View, download, upload, edit, add and delete."
                    },
                    "displayName": {
                      "const": "Can edit without versions"
                    },
                    "id": {
                      "const": "3284f2d5-0070-4ad8-ac40-c247f7c1fb27"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can edit"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can manage"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @env-config
  Scenario: user lists permissions of a folder after enabling 'Denied' role
    Given the administrator has enabled the permissions role "Denied"
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
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 200
                    },
                    "description": {
                      "const": "Deny all access."
                    },
                    "displayName": {
                      "const": "Cannot access"
                    },
                    "id": {
                      "const": "63e64e19-8d43-42ec-a738-2b6af2610efa"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can view"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can upload"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can edit"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @env-config
  Scenario: user lists permissions of a folder inside a space after enabling 'Denied' role
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    When user "Alice" gets permissions list for folder "folder" of the space "new-space" using the Graph API
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
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 200
                    },
                    "description": {
                      "const": "Deny all access."
                    },
                    "displayName": {
                      "const": "Cannot access"
                    },
                    "id": {
                      "const": "63e64e19-8d43-42ec-a738-2b6af2610efa"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can view"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can upload"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Can edit"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issue-9764
  Scenario: user tries to list permissions of a disabled project space using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has disabled a space "new-space"
    When user "Alice" tries to list the permissions of space "new-space" using root endpoint of the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": ["code", "innererror", "message"],
            "properties": {
              "code": { "const": "itemNotFound" },
              "innererror": {
                "type": "object",
                "required": ["date", "request-id"]
              },
              "message": {
                "pattern": "stat: error: not found: %user_id_pattern%$"
              }
            }
          }
        }
      }
      """


  Scenario: user lists permissions of a folder in personal space with select filter
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    When user "Alice" gets the permittion list of folder "folder" from the space "Personal" using the Graph API with query "$select=@libre.graph.permissions.roles.allowedValues"
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain the following keys:
      | @libre.graph.permissions.roles.allowedValues |
    And the JSON data of the response should not contain the following keys:
      | @libre.graph.permissions.actions.allowedValues |
      | value                                          |
    When user "Alice" gets the permittion list of folder "folder" from the space "Personal" using the Graph API with query "$select=@libre.graph.permissions.actions.allowedValues"
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain the following keys:
      | @libre.graph.permissions.actions.allowedValues |
    And the JSON data of the response should not contain the following keys:
      | @libre.graph.permissions.roles.allowedValues   |
      | value                                          |
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.actions.allowedValues"
        ],
        "properties": {
          "@libre.graph.permissions.actions.allowedValues": {
            "const": [
              "libre.graph/driveItem/permissions/create",
              "libre.graph/driveItem/children/create",
              "libre.graph/driveItem/standard/delete",
              "libre.graph/driveItem/path/read",
              "libre.graph/driveItem/quota/read",
              "libre.graph/driveItem/content/read",
              "libre.graph/driveItem/upload/create",
              "libre.graph/driveItem/permissions/read",
              "libre.graph/driveItem/children/read",
              "libre.graph/driveItem/versions/read",
              "libre.graph/driveItem/deleted/read",
              "libre.graph/driveItem/path/update",
              "libre.graph/driveItem/permissions/delete",
              "libre.graph/driveItem/deleted/delete",
              "libre.graph/driveItem/versions/update",
              "libre.graph/driveItem/deleted/update",
              "libre.graph/driveItem/basic/read",
              "libre.graph/driveItem/permissions/update",
              "libre.graph/driveItem/permissions/deny"
            ]
          }
        }
      }
      """


  Scenario: user lists permissions of a project space with select filter
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" gets the drive permittion list of the space "new-space" using the Graph API with query "$select=@libre.graph.permissions.actions.allowedValues"
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain the following keys:
      | @libre.graph.permissions.actions.allowedValues |
    And the JSON data of the response should not contain the following keys:
      | @libre.graph.permissions.roles.allowedValues |
      | value                                        |
    When user "Alice" gets the drive permittion list of the space "new-space" using the Graph API with query "$select=@libre.graph.permissions.roles.allowedValues"
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain the following keys:
      | @libre.graph.permissions.roles.allowedValues |
    And the JSON data of the response should not contain the following keys:
      | @libre.graph.permissions.actions.allowedValues |
      | value                                        |
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.permissions.roles.allowedValues"
        ],
        "properties": {
          "@libre.graph.permissions.roles.allowedValues": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf":  [
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 40
                    },
                    "description": {
                      "const": "View and download."
                    },
                    "displayName": {
                      "const": "Can view"
                    },
                    "id": {
                      "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 90
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete including the history."
                    },
                    "displayName": {
                      "const": "Can edit"
                    },
                    "id": {
                      "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@libre.graph.weight",
                    "description",
                    "displayName",
                    "id"
                  ],
                  "properties": {
                    "@libre.graph.weight": {
                      "const": 120
                    },
                    "description": {
                      "const": "View, download, upload, edit, add, delete and manage members."
                    },
                    "displayName": {
                      "const": "Can manage"
                    },
                    "id": {
                      "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: user lists permissions of a project space with count filter
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" gets the drive permittion list of the space "new-space" using the Graph API with query "$count=true&$top=0"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@odata.count"
        ],
        "properties": {
          "@odata.count": {
            "const": 2
          }
        }
      }
      """