@ocm
Feature: an user shares resources using ScienceMesh application
  As a user
  I want to share resources between different OpenCloud instances

  Background:
    Given user "Alice" has been created with default attributes
    And using server "REMOTE"
    And user "Brian" has been created with default attributes

  @issue-9534
  Scenario: local user shares a folder to federation user
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has created folder "folderToShare"
    When user "Alice" sends the following resource share invitation to federated user using the Graph API:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    Then the HTTP status code should be "200"
    When using server "REMOTE"
    And user "Brian" lists the shares shared with him without retry using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden": { "const": false },
                "@client.synchronize": { "const": false },
                "createdBy": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id", "@libre.graph.userType"],
                      "properties": {
                        "displayName": { "const": "Alice Hansen" },
                        "id": { "pattern": "^%federated_user_id_pattern%$" },
                        "@libre.graph.userType": { "const": "Federated" }
                      }
                    }
                  }
                },
                "eTag": { "pattern": "%etag_pattern%" },
                "folder": { "const": {} },
                "id": { "pattern": "^%file_id_pattern%$" },
                "name": { "const": "folderToShare" },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
                  "properties": {
                    "driveId": { "pattern": "^%space_id_pattern%$" },
                    "driveType": { "const": "virtual" },
                    "id": { "pattern": "^%file_id_pattern%$" }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "permissions"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["id", "displayName", "@libre.graph.userType"],
                          "properties": {
                            "id": { "pattern": "^%federated_user_id_pattern%$" },
                            "displayName": { "const": "Alice Hansen" },
                            "@libre.graph.userType": { "const": "Federated" }
                          }
                        }
                      }
                    },
                    "eTag": { "pattern": "%etag_pattern%" },
                    "folder": { "const": {} },
                    "id": { "pattern": "^%federated_file_id_pattern%$" },
                    "name": { "const": "folderToShare" },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "createdDateTime",
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id", "@libre.graph.userType"],
                                "properties": {
                                  "displayName": { "const": "Brian Murphy" },
                                  "id": { "pattern": "^%user_id_pattern%$" },
                                  "@libre.graph.userType": { "const": "Member" }
                                }
                              }
                            }
                          },
                          "id": { "pattern": "^%uuidv4_pattern%$" },
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
                                      "id": { "pattern": "^%federated_user_id_pattern%$" },
                                      "@libre.graph.userType": { "const": "Federated" }
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
                              "pattern": "^%role_id_pattern%$"
                            }
                          }
                        }
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

  @issue-9534
  Scenario: local user shares a file to federation user
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has uploaded file with content "ocm test" to "textfile.txt"
    When user "Alice" sends the following resource share invitation to federated user using the Graph API:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    Then the HTTP status code should be "200"
    When using server "REMOTE"
    And user "Brian" lists the shares shared with him without retry using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden": { "const": false },
                "@client.synchronize": { "const": false },
                "createdBy": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id", "@libre.graph.userType"],
                      "properties": {
                        "displayName": { "const": "Alice Hansen" },
                        "id": { "pattern": "^%federated_user_id_pattern%$" },
                        "@libre.graph.userType": { "const": "Federated" }
                      }
                    }
                  }
                },
                "eTag": { "pattern": "%etag_pattern%" },
                "file": {
                  "type": "object",
                  "required": ["mimeType"],
                  "properties": {
                    "mimeType": { "const": "text/plain" }
                  }
                },
                "id": { "pattern": "^%file_id_pattern%$" },
                "name": { "const": "textfile.txt" },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
                  "properties": {
                    "driveId": { "pattern": "^%space_id_pattern%$" },
                    "driveType": { "const": "virtual" },
                    "id": { "pattern": "^%file_id_pattern%$" }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "permissions"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["id", "displayName", "@libre.graph.userType"],
                          "properties": {
                            "id": { "pattern": "^%federated_user_id_pattern%$" },
                            "displayName": { "const": "Alice Hansen" },
                            "@libre.graph.userType": { "const": "Federated" }
                          }
                        }
                      }
                    },
                    "eTag": { "pattern": "%etag_pattern%" },
                    "file": {
                      "type": "object",
                      "required": ["mimeType"],
                      "properties": {
                        "mimeType": { "const": "text/plain" }
                      }
                    },
                    "id": { "pattern": "^%federated_file_id_pattern%$" },
                    "name": { "const": "textfile.txt" },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "createdDateTime",
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id", "@libre.graph.userType"],
                                "properties": {
                                  "displayName": { "const": "Brian Murphy" },
                                  "id": { "pattern": "^%user_id_pattern%$" },
                                  "@libre.graph.userType": { "const": "Member" }
                                }
                              }
                            }
                          },
                          "id": { "pattern": "^%uuidv4_pattern%$" },
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
                                      "id": { "pattern": "^%federated_user_id_pattern%$" },
                                      "@libre.graph.userType": { "const": "Federated" }
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
                              "pattern": "^%role_id_pattern%$"
                            }
                          }
                        }
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


  Scenario: local user shares a folder from project space to federation user
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" sends the following resource share invitation to federated user using the Graph API:
      | resource        | folderToShare |
      | space           | projectSpace  |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    Then the HTTP status code should be "200"
    When using server "REMOTE"
    And user "Brian" lists the shares shared with him without retry using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden": { "const": false },
                "@client.synchronize": { "const": false },
                "createdBy": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id", "@libre.graph.userType"],
                      "properties": {
                        "displayName": { "const": "Alice Hansen" },
                        "id": { "pattern": "^%federated_user_id_pattern%$" },
                        "@libre.graph.userType": { "const": "Federated" }
                      }
                    }
                  }
                },
                "eTag": { "pattern": "%etag_pattern%" },
                "folder": { "const": {} },
                "id": { "pattern": "^%file_id_pattern%$" },
                "name": { "const": "folderToShare" },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
                  "properties": {
                    "driveId": { "pattern": "^%space_id_pattern%$" },
                    "driveType": { "const": "virtual" },
                    "id": { "pattern": "^%file_id_pattern%$" }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "permissions"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["id", "displayName", "@libre.graph.userType"],
                          "properties": {
                            "id": { "pattern": "^%federated_user_id_pattern%$" },
                            "displayName": { "const": "Alice Hansen" },
                            "@libre.graph.userType": { "const": "Federated" }
                          }
                        }
                      }
                    },
                    "eTag": { "pattern": "%etag_pattern%" },
                    "folder": { "const": {} },
                    "id": { "pattern": "^%federated_file_id_pattern%$" },
                    "name": { "const": "folderToShare" },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "createdDateTime",
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id", "@libre.graph.userType"],
                                "properties": {
                                  "displayName": { "const": "Brian Murphy" },
                                  "id": { "pattern": "^%user_id_pattern%$" },
                                  "@libre.graph.userType": { "const": "Member" }
                                }
                              }
                            }
                          },
                          "id": { "pattern": "^%uuidv4_pattern%$" },
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
                                      "id": { "pattern": "^%federated_user_id_pattern%$" },
                                      "@libre.graph.userType": { "const": "Federated" }
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
                              "pattern": "^%role_id_pattern%$"
                            }
                          }
                        }
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

  @issue-10051
  Scenario Outline: try to add federated user as a member of a project space (permissions endpoint)
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "alice's space" with the default quota using the Graph API
    When user "Alice" tries to send the following space share invitation to federated user using permissions endpoint of the Graph API:
      | space           | alice's space      |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "400"
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
                "const": "invalidRequest"
              },
              "innererror": {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message": {
                "const": "federated user can not become a space member"
              }
            }
          }
        }
      }
      """
    And using server "REMOTE"
    And the user "Brian" should not have a space called "alice's space"
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |

  @issue-10051
  Scenario Outline: try to add federated user as a member of a project space (root endpoint)
    Given using server "LOCAL"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "alice's space" with the default quota using the Graph API
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    When user "Alice" tries to send the following space share invitation to federated user using root endpoint of the Graph API:
      | space           | alice's space      |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "400"
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
                "const": "invalidRequest"
              },
              "innererror": {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message": {
                "const": "federated user can not become a space member"
              }
            }
          }
        }
      }
      """
    And using server "REMOTE"
    And the user "Brian" should not have a space called "alice's space"
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |

  @issue-9908
  Scenario: sharer lists the shares shared to a federated user
    Given using server "LOCAL"
    And user "Alice" has uploaded file with content "ocm test" to "/textfile.txt"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "eTag",
          "file",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
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
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
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
          },
          "name": {
            "const": "textfile.txt"
          },
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "file": {
            "type": "object",
            "required": ["mimeType"],
            "properties": {
              "mimeType": {
                "const": "text/plain"
              }
            }
          },
          "size": {
            "const": 8
          }
        }
      }
      """

  @issue-10222 @issue-10495
  Scenario: local user lists multiple federation shares
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And user "Brian" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And user "Brian" has created folder "folderToShare"
    And user "Brian" has sent the following resource share invitation to federated user:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Alice         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has sent the following resource share invitation to federated user:
      | resource        | testavatar.jpg |
      | space           | Personal       |
      | sharee          | Alice          |
      | shareType       | user           |
      | permissionsRole | Viewer         |
    And using server "LOCAL"
    When user "Alice" lists the shares shared with her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
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
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "name": {
                      "const": "folderToShare"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem",
                    "size"
                  ],
                  "properties": {
                    "name": {
                      "const": "testavatar.jpg"
                    },
                    "file": {
                      "type": "object",
                      "required": ["mimeType"],
                      "properties": {
                        "mimeType": {
                          "const": "image/jpeg"
                        }
                      }
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": [
                        "createdBy",
                        "eTag",
                        "file",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "permissions",
                        "size"
                      ],
                      "properties": {
                        "file": {
                          "type": "object",
                          "required": ["mimeType"],
                          "properties": {
                            "mimeType": {
                              "const": "image/jpeg"
                            }
                          }
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

  @issue-10285 @issue-10536 @issue-10305
  Scenario: federation user uploads file to a federated shared folder via TUS
    Given using spaces DAV path
    And using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has created a folder "FOLDER" in space "Personal"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    When using server "REMOTE"
    And user "Brian" uploads a file with content "lorem" to "file.txt" inside federated share "FOLDER" via TUS using the WebDAV API
    Then for user "Brian" the content of file "file.txt" of federated share "FOLDER" should be "lorem"

  @issue-10285 @issue-10536
  Scenario: local user uploads file to a federated shared folder via TUS
    Given using spaces DAV path
    And using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And user "Brian" has created a folder "FOLDER" in space "Personal"
    And user "Brian" has sent the following resource share invitation to federated user:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    When using server "LOCAL"
    And user "Alice" uploads a file with content "lorem" to "file.txt" inside federated share "FOLDER" via TUS using the WebDAV API
    Then for user "Alice" the content of file "file.txt" of federated share "FOLDER" should be "lorem"

  @issue-10495
  Scenario: local user downloads thumbnail preview of a federated shared image
    Given using spaces DAV path
    And using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And user "Brian" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And user "Brian" has sent the following resource share invitation to federated user:
      | resource        | testavatar.jpg |
      | space           | Personal       |
      | sharee          | Alice          |
      | shareType       | user           |
      | permissionsRole | Viewer         |
    And using server "LOCAL"
    When user "Alice" downloads the preview of federated share image "testavatar.jpg" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "32" pixels wide and "32" pixels high
    And the downloaded preview content should match with "thumbnail.png" fixtures preview content

  @issue-10358
  Scenario: user edits content of a federated share file
    Given using spaces DAV path
    And using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has uploaded file with content "ocm test" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And using server "REMOTE"
    And for user "Brian" the content of file "textfile.txt" of federated share "textfile.txt" should be "ocm test"
    When user "Brian" updates the content of federated share "textfile.txt" with "this is a new content" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Brian" the content of file "textfile.txt" of federated share "textfile.txt" should be "this is a new content"
    And using server "LOCAL"
    And for user "Alice" the content of the file "textfile.txt" of the space "Personal" should be "this is a new content"

  @issue-10488
  Scenario: local user shares a folder copied from an already shared folder to federation user
    Given using server "REMOTE"
    And "Brian" has created the federation share invitation
    And using server "LOCAL"
    And "Alice" has accepted invitation
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Alice" has copied folder "folderToShare" to "folderToShareCopy"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | folderToShareCopy |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | Viewer            |
    And using server "REMOTE"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
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
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "name": {
                      "const": "folderToShare"
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": ["permissions"],
                      "properties": {
                        "permissions": {
                          "type": "array",
                          "minItems": 1,
                          "maxItems": 1,
                          "items": {
                            "type": "object",
                            "required": ["@libre.graph.permissions.actions"]
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "name": {
                      "const": "folderToShareCopy"
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": ["permissions"],
                      "properties": {
                        "permissions": {
                          "type": "array",
                          "minItems": 1,
                          "maxItems": 1,
                          "items": {
                            "type": "object",
                            "properties": {
                              "@libre.graph.permissions.actions": {
                                "type": "null"
                              }
                            }
                          }
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

  @issue-9926
  Scenario: federated user tries to update a shared file after local user updates role
    Given using spaces DAV path
    And using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    And user "Alice" has uploaded file with content "ocm test" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole    | File Editor              |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
      | space              | Personal                 |
      | resource           | textfile.txt             |
    And using server "REMOTE"
    When user "Brian" updates the content of federated share "textfile.txt" with "this is a new content" using the WebDAV API
    Then the HTTP status code should be "500"
    And using server "LOCAL"
    And the content of file "textfile.txt" for user "Alice" should be "ocm test"

  @issue-10689
  Scenario: federation user lists all the spaces
    Given using server "REMOTE"
    And "Brian" has created the federation share invitation
    And using server "LOCAL"
    And "Alice" has accepted invitation
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And using server "REMOTE"
    When user "Brian" lists all available spaces via the Graph API
    Then the HTTP status code should be "200"
    And the json response should not contain a space with name "folderToShare"

  @issue-10213
  Scenario Outline: local user removes access of federated user from a resource
    Given using spaces DAV path
    And using server "REMOTE"
    And "Brian" has created the federation share invitation
    And using server "LOCAL"
    And "Alice" has accepted invitation
    And user "Alice" has created a folder "FOLDER" in space "Personal"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | FOLDER            |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <permissionsRole> |
    When user "Alice" removes the access of user "Brian" from resource "FOLDER" of space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    And using server "REMOTE"
    And user "Brian" should not have a federated share "FOLDER" shared by user "Alice" from space "Personal"
    Examples:
      | permissionsRole |
      | Viewer          |
      | Uploader        |
      | Editor          |

  @issue-10272
  Scenario: federated user downloads shared resources as an archive
    Given using spaces DAV path
    And using server "REMOTE"
    And "Brian" has created the federation share invitation
    And using server "LOCAL"
    And "Alice" has accepted invitation
    And user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And user "Alice" has created folder "imageFolder"
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "imageFolder/testavatar.png"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | imageFolder |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | Viewer      |
    And using server "REMOTE"
    When user "Brian" downloads the archive of these items using the resource remoteItemIds
      | textfile.txt |
      | imageFolder  |
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name                       | content    |
      | textfile.txt               | some data  |
      | imageFolder                |            |
      | imageFolder/testavatar.png |            |
