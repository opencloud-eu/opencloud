Feature: default capabilities for normal user
  As a user
  I want to list capabilities
  So that I can make sure what capabilities are available to me

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes

  # adjust this scenario after fixing tagged issues as its just created to show difference
  # in the response items in different environment (core & opencloud-reva)
  @issue-1285 @issue-1286
  Scenario: getting default capabilities with normal user
    When user "Alice" retrieves the capabilities using the capabilities API
    Then the OCS status code should be "100"
    And the HTTP status code should be "200"
    And the ocs JSON data of the response should match
      """
      {
        "type": "object",
        "required": [ "capabilities" ],
        "properties": {
          "capabilities": {
            "type": "object",
            "required": [
              "core",
              "files",
              "files_sharing"
            ],
            "properties": {
              "core": {
                "type": "object",
                "required": [
                  "pollinterval",
                  "webdav-root",
                  "status"
                ],
                "properties": {
                  "pollinterval": {
                    "const": 60
                  },
                  "webdav-root": {
                    "const": "remote.php/webdav"
                  },
                  "status": {
                    "type": "object",
                    "required": [
                      "version",
                      "versionstring",
                      "edition",
                      "productname"
                    ],
                    "properties": {
                      "version": {
                        "const": "%version%"
                      },
                      "versionstring": {
                        "const": "%versionstring%"
                      },
                      "edition": {
                        "const": "%edition%"
                      },
                      "productname": {
                        "const": "%productname%"
                      }
                    }
                  }
                }
              },
              "files": {
                "type": "object",
                "required": [
                  "bigfilechunking",
                  "privateLinks"
                ],
                "properties": {
                  "bigfilechunking": {
                    "const": false
                  },
                  "privateLinks": {
                    "const": true
                  }
                }
              },
              "files_sharing": {
                "type": "object",
                "required": [
                  "api_enabled",
                  "default_permissions",
                  "public",
                  "resharing",
                  "federation",
                  "group_sharing",
                  "share_with_group_members_only",
                  "share_with_membership_groups_only",
                  "auto_accept_share",
                  "user_enumeration"
                ],
                "properties": {
                  "api_enabled": {
                    "const": true
                  },
                  "default_permissions": {
                    "const": 22
                  },
                  "public": {
                    "type": "object",
                    "required": [
                      "enabled",
                      "multiple",
                      "upload",
                      "supports_upload_only",
                      "send_mail",
                      "social_share"
                    ],
                    "properties": {
                      "enabled": {
                        "const": true
                      },
                      "multiple": {
                        "const": true
                      },
                      "upload": {
                        "const": true
                      },
                      "supports_upload_only": {
                        "const": true
                      },
                      "send_mail": {
                        "const": true
                      },
                      "social_share": {
                        "const": true
                      }
                    }
                  },
                  "resharing": {
                    "const": false
                  },
                  "federation": {
                    "type": "object",
                    "required": [
                      "outgoing",
                      "incoming"
                    ],
                    "properties": {
                      "outgoing": {
                        "const": false
                      },
                      "incoming": {
                        "const": false
                      }
                    }
                  },
                  "group_sharing": {
                    "const": true
                  },
                  "share_with_group_members_only": {
                    "const": true
                  },
                  "share_with_membership_groups_only": {
                    "const": true
                  },
                  "auto_accept_share": {
                    "const": true
                  },
                  "user_enumeration": {
                    "type": "object",
                    "required": [
                      "enabled",
                      "group_members_only"
                    ],
                    "properties": {
                      "enabled": {
                        "const": true
                      },
                      "group_members_only": {
                        "const": true
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
