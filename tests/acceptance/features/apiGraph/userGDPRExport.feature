Feature: user GDPR (General Data Protection Regulation) report
  As a user
  I want to generate my GDPR report
  So that I can review what events are stored by the server

  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path


  Scenario: generate a GDPR report and check user data in the downloaded report
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain key 'user' and match
      """
      {
        "type": "object",
        "required": [
          "id",
          "username",
          "mail",
          "display_name",
          "uid_number",
          "gid_number"
        ],
        "properties": {
          "id": {
            "type": "object",
            "required": [
              "idp",
              "opaque_id",
              "type"
            ],
            "properties": {
              "idp": {
                "type": "string",
                "pattern": "^%base_url%$"
              },
              "opaque_id": {
                "type": "string",
                "pattern": "^%user_id_pattern%$"
              },
              "type": {
                "type": "number",
                "enum": [1]
              }
            }
          },
          "username": {
            "type": "string",
            "enum": ["Alice"]
          },
          "mail": {
            "type": "string",
            "enum": ["alice@example.org"]
          },
          "display_name": {
            "type": "string",
            "enum": ["Alice Hansen"]
          },
          "uid_number": {
            "type": "number",
            "enum": [99]
          },
          "gid_number": {
            "type": "number",
            "enum": [99]
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when a user is created
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.UserCreated" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "UserID"
            ],
            "properties": {
              "Executant": {
                "type": "object",
                "required": [
                  "idp",
                  "opaque_id",
                  "type"
                ],
                "properties": {
                  "idp": {
                    "type": "string",
                    "pattern": "^%base_url%$"
                  },
                  "opaque_id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  },
                  "type": {
                    "type": "number",
                    "enum": [1]
                  }
                }
              },
              "UserID": {
                "type": "string",
                "pattern": "^%user_id_pattern%$"
              }
            }
          }
        }
      }
      """
    And the downloaded JSON content should contain event type "events.SpaceCreated" for "personal" space and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "Name",
              "Type",
              "Quota"
            ],
            "properties": {
              "Executant": {
                "type": "object",
                "required": [
                  "idp",
                  "opaque_id",
                  "type"
                ],
                "properties": {
                  "idp": {
                    "type": "string",
                    "pattern": "^%base_url%$"
                  },
                  "opaque_id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  },
                  "type": {
                    "type": "number",
                    "enum": [1]
                  }
                }
              },
              "Name": {
                "type": "string",
                "enum": ["Alice Hansen"]
              },
              "Type": {
                "type": "string",
                "enum": ["personal"]
              },
              "Quota": {
                "type": ["number", "null"],
                "enum": [null]
              }
            }
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when user uploads a file
    Given user "Alice" has uploaded file with content "sample text" to "lorem.txt"
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.BytesReceived" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "ExecutingUser",
              "Filename",
              "ResourceID",
              "Filesize",
              "UploadID",
              "SpaceOwner"
            ],
            "properties": {
              "ExecutingUser": {
                "type": "object",
                "required": [
                  "username"
                ],
                "properties": {
                  "username": {
                    "type": "string",
                    "enum": ["Alice"]
                  }
                }
              },
              "Filename": {
                "type": "string",
                "enum": ["lorem.txt"]
              },
              "Filesize": {
                "type": "number",
                "enum": [11]
              },
              "Quota": {
                "type": ["number", "null"],
                "enum": [null]
              }
            }
          }
        }
      }
      """
    And the downloaded JSON content should contain event type "events.FileUploaded" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "Owner",
              "Ref",
              "SpaceOwner"
            ],
            "properties": {
              "Ref": {
                "type": "object",
                "required": [
                  "path"
                ],
                "properties": {
                  "path" : {
                    "type": "string",
                    "enum": ["./lorem.txt"]
                  }
                }
              }
            }
          }
        }
      }
      """
    And the downloaded JSON content should contain event type "events.PostprocessingFinished" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "ExecutingUser",
              "Filename"
            ],
            "properties": {
              "ExecutingUser": {
                "type": "object",
                "required": [
                  "username"
                ],
                "properties": {
                  "username": {
                    "type": "string",
                    "enum": ["Alice"]
                  }
                }
              },
              "Filename": {
                "type": "string",
                "enum": ["lorem.txt"]
              }
            }
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when a user is added to a group
    Given group "tea-lover" has been created
    And user "Alice" has been added to group "tea-lover"
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.GroupMemberAdded" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "GroupID",
              "UserID"
            ],
            "properties": {
              "Executant": {
                "type": "object",
                "required": [
                  "idp",
                  "opaque_id",
                  "type"
                ],
                "properties": {
                  "idp": {
                    "type": "string",
                    "pattern": "^%base_url%$"
                  },
                  "opaque_id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  },
                  "type": {
                    "type": "number",
                    "enum": [1]
                  }
                }
              },
              "GroupID": {
                "type": "string",
                "pattern": "^%group_id_pattern%$"
              },
              "UserID": {
                "type": "string",
                "pattern": "^%user_id_pattern%$"
              }
            }
          }
        }
      }
      """
    And the downloaded JSON content should contain key 'user' and match
      """
      {
        "type": "object",
        "required": [
          "id",
          "username",
          "mail",
          "display_name",
          "groups",
          "uid_number",
          "gid_number"
        ],
        "properties": {
          "username": {
            "type": "string",
            "enum": ["Alice"]
          },
          "mail": {
            "type": "string",
            "enum": ["alice@example.org"]
          },
          "display_name": {
            "type": "string",
            "enum": ["Alice Hansen"]
          },
          "groups": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "string",
              "pattern": "^%group_id_pattern%$"
            }
          },
          "uid_number": {
            "type": "number",
            "enum": [99]
          },
          "gid_number": {
            "type": "number",
            "enum": [99]
          }
        }
      }
      """


  Scenario: generate a GDPR report after the admin updates the quota of personal space
    Given user "Admin" has changed the quota of the personal space of user "Alice" to "10000"
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.SpaceUpdated" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "Space"
            ],
            "properties": {
              "Executant": {
                "type": "object",
                "required": [
                  "idp",
                  "opaque_id",
                  "type"
                ],
                "properties": {
                  "idp": {
                    "type": "string",
                    "pattern": "^%base_url%$"
                  },
                  "opaque_id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  },
                  "type": {
                    "type": "number",
                    "enum": [1]
                  }
                }
              },
              "Space": {
                "type": "object",
                "required": [
                  "name",
                  "quota",
                  "space_type"
                ],
                "properties": {
                  "name": {
                    "type": "string",
                    "enum": ["Alice Hansen"]
                  },
                  "quota": {
                    "type": "object",
                    "required": [
                      "quota_max_bytes",
                      "quota_max_files"
                    ],
                    "properties": {
                      "quota_max_bytes": {
                        "type": "number",
                        "enum": [10000]
                      },
                      "quota_max_files": {
                        "type": "number",
                        "enum": [18446744073709552000]
                      }
                    }
                  },
                  "space_type": {
                    "type": "string",
                    "enum": ["personal"]
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario Outline: user tries to generate GDPR report of other users
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    And the administrator has assigned the role "<new-user-role>" to user "Brian" using the Graph API
    When user "Alice" tries to export GDPR report of user "Brian" to "/.personal_data_export.json" using Graph API
    Then the HTTP status code should be "400"
    Examples:
      | user-role   | new-user-role |
      | Space Admin | Space Admin   |
      | Space Admin | User          |
      | Space Admin | User Light    |
      | Space Admin | Admin         |
      | User        | Space Admin   |
      | User        | User          |
      | User        | User Light    |
      | User        | Admin         |
      | User Light  | Space Admin   |
      | User Light  | User          |
      | User Light  | User Light    |
      | User Light  | Admin         |
      | Admin       | Space Admin   |
      | Admin       | User          |
      | Admin       | User Light    |
      | Admin       | Admin         |


  Scenario: generate a GDPR report and check events when user creates a folder
    Given user "Alice" has created folder "/folderMain"
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.ContainerCreated" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "Owner",
              "Ref",
              "SpaceOwner"
            ],
            "properties": {
              "Executant": {
                  "type": "object",
                  "required": [
                    "idp",
                    "opaque_id",
                    "type"
                  ],
                  "properties": {
                    "idp": {
                      "type": "string",
                      "pattern": "^%base_url%$"
                    },
                    "opaque_id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "type": {
                      "type": "number",
                      "enum": [1]
                    }
                  }
                },
              "Ref": {
                "type": "object",
                "required": [
                  "path",
                  "resource_id"
                ],
                "properties": {
                  "path" : {
                    "type": "string",
                    "enum": ["./folderMain"]
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when a user shares a folder
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folderMain"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderMain |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "folderMain" synced
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.ShareCreated" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "CTime",
              "Executant",
              "GranteeGroupID",
              "GranteeUserID",
              "ItemID",
              "Permissions",
              "ShareID",
              "Sharee",
              "Sharer"
            ],
            "properties": {
              "GranteeGroupID": {
                "type": ["number", "null"],
                "enum": [null]
              },
              "Permissions": {
                "type": "object",
                "required": [
                  "permissions"
                ],
                "properties": {
                  "permissions": {
                    "type": "object",
                    "required": [
                      "get_path",
                      "get_quota",
                      "initiate_file_download",
                      "list_container",
                      "list_recycle",
                      "stat"
                    ],
                    "properties": {
                      "get_path" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "get_quota" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "initiate_file_download" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "list_container" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "list_recycle" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "stat" : {
                        "type": "boolean",
                        "enum": [true]
                      }
                    }
                  }
                }
              },
              "ShareID": {
                "type": "object",
                "required": [
                  "opaque_id"
                ],
                "properties": {
                  "opaque_id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%:%user_id_pattern%:%user_id_pattern%$"
                  }
                }
              },
              "Sharee": {
                "type": ["number", "null"],
                "enum": [null]
              },
              "Sharer": {
                "type": "object",
                  "required": [
                    "idp",
                    "opaque_id",
                    "type"
                  ],
                  "properties": {
                    "idp": {
                      "type": "string",
                      "pattern": "^%base_url%$"
                    },
                    "opaque_id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "type": {
                      "type": "number",
                      "enum": [1]
                    }
                  }
              }
            }
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when a user creates a public link share
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folderMain"
    And user "Alice" has created the following resource link share:
      | resource        | folderMain |
      | space           | Personal   |
      | permissionsRole | view       |
      | password        | %public%   |
      | displayName     | sharedlink |
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.LinkCreated" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "CTime",
              "DisplayName",
              "Executant",
              "Expiration",
              "ItemID",
              "PasswordProtected",
              "Permissions",
              "ShareID",
              "Sharer",
              "Token"
            ],
            "properties": {
              "DisplayName": {
                "type": "string",
                "enum": ["sharedlink"]
              },
              "Expiration": {
                "type": ["string", "null"],
                "enum": [null]
              },
              "PasswordProtected": {
                "type": "boolean",
                "enum": [true]
              },
              "Permissions": {
                "type": "object",
                "required": [
                  "permissions"
                ],
                "properties": {
                  "permissions": {
                    "type": "object",
                    "required": [
                      "get_path",
                      "get_quota",
                      "initiate_file_download",
                      "list_container",
                      "list_recycle",
                      "stat"
                    ],
                    "properties": {
                      "get_path" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "get_quota" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "initiate_file_download" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "list_container" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "list_recycle" : {
                        "type": "boolean",
                        "enum": [true]
                      },
                      "stat" : {
                        "type": "boolean",
                        "enum": [true]
                      }
                    }
                  }
                }
              },
              "Token": {
                "type": "string",
                "pattern": "^[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when user deletes a resource
    Given user "Alice" has created folder "/folderMain"
    And user "Alice" has deleted folder "/folderMain"
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.ItemTrashed" in item 'events' and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "Owner",
              "ID",
              "Ref",
              "SpaceOwner"
            ],
            "properties": {
              "Ref": {
                "type": "object",
                "required": [
                  "path"
                ],
                "properties": {
                  "path" : {
                    "type": "string",
                    "enum": ["./folderMain"]
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when a space is shared
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "GDPR Space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | GDPR Space   |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.SpaceShared" in item 'events' and should match
      """
      {
        "type": "object",
        "required": ["event", "type"],
        "properties": {
          "event": {
            "type": "object",
            "required": [
              "Creator",
              "Executant",
              "GranteeGroupID",
              "GranteeUserID"
            ],
            "properties": {
              "Creator": {
                  "type": "object",
                  "required": ["idp", "opaque_id", "type"],
                  "properties": {
                      "idp": {
                        "type": "string",
                        "pattern": "^%base_url%$"
                      },
                      "opaque_id": {
                        "type": "string",
                        "pattern": "^%user_id_pattern%$"
                      },
                      "type": {
                        "const": 1
                      }
                  }
              },
              "Executant": {
                  "type": "object",
                  "required": ["idp", "opaque_id", "type"],
                  "properties": {
                      "idp": {
                        "type": "string",
                        "pattern": "^%base_url%$"
                      },
                      "opaque_id": {
                        "type": "string",
                        "pattern": "^%user_id_pattern%$"
                      },
                      "type": {
                        "const": 1
                      }
                  }
              },
              "GranteeGroupID": {
                "const": null
              },
              "GranteeUserID": {
                "type": "object",
                "required": ["opaque_id"],
                "properties": {
                    "opaque_id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    }
                }
              }
            }
          },
          "type": {
            "const": "events.SpaceShared"
          }
        }
      }
      """


  Scenario: generate a GDPR report and check events when a space is created
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "GDPR Space" with the default quota using the Graph API
    When user "Alice" exports her GDPR report to "/.personal_data_export.json" using the Graph API
    And user "Alice" downloads the content of GDPR report ".personal_data_export.json"
    Then the HTTP status code of responses on each endpoint should be "202, 200" respectively
    And the downloaded JSON content should contain event type "events.SpaceCreated" for "project" space and should match
      """
      {
        "type": "object",
        "required": [
          "event"
        ],
        "properties": {
          "event" : {
            "type": "object",
            "required": [
              "Executant",
              "Name",
              "Type"
            ],
            "properties": {
              "Executant": {
                "type": "object",
                "required": [
                  "idp",
                  "opaque_id",
                  "type"
                ],
                "properties": {
                  "idp": {
                    "type": "string",
                    "pattern": "^%base_url%$"
                  },
                  "opaque_id": {
                    "type": "string",
                    "pattern": "^%user_id_pattern%$"
                  },
                  "type": {
                    "type": "number",
                    "enum": [1]
                  }
                }
              },
              "Name": {
                "type": "string",
                "enum": ["GDPR Space"]
              },
              "Type": {
                "type": "string",
                "enum": ["project"]
              }
            }
          }
        }
      }
      """
