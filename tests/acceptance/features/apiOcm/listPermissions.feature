@ocm
Feature: List a federated sharing permissions
  As a user
  I want to list the permissions for federated share
  So that the federated share is assigned the correct permissions

  Background:
    Given user "Alice" has been created with default attributes

  @issue-9898
  Scenario: user lists permissions of a resource shared to a federated user
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And user "Brian" has been created with default attributes
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has uploaded file with content "ocm test" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Alice" gets permissions list for file "textfile.txt" of the space "Personal" using the Graph API
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
            "minItems": 1,
            "maxItems": 1,
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
                          "required": ["@libre.graph.userType","displayName","id"],
                          "properties": {
                            "@libre.graph.userType": {
                              "const": "Federated"
                            },
                            "id": {
                              "type": "string",
                              "pattern": "^%federated_user_id_pattern%$"
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
                      "pattern": "^%user_id_pattern%$"
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
                              "required": ["@libre.graph.userType", "displayName", "id"],
                              "properties": {
                                "@libre.graph.userType": {
                                  "const": "Member"
                                },
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
                        }
                      }
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
              ]
            }
          }
        }
      }
      """

  @issue-9745 @env-config
  Scenario: user lists allowed file permissions for federated user
    Given using server "LOCAL"
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has uploaded file with content "ocm test" to "/textfile.txt"
    When user "Alice" gets the allowed roles for federated user of file "textfile.txt" from the space "Personal" using the Graph API
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

  @issue-9745
  Scenario: user lists allowed folder permissions for federated user
    Given using server "LOCAL"
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has created folder "folderToShare"
    When user "Alice" gets the allowed roles for federated user of folder "folderToShare" from the space "Personal" using the Graph API
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
