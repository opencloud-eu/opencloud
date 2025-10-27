Feature: Create a link share for a resource
  https://docs.opencloud.eu/libre-graph-api/#/drives.permissions/CreateLink

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |

  @owncloud-issue-7879
  Scenario Outline: create a link share of a folder using permissions endpoint
    Given user "Alice" has created folder "folder"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folder             |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |

  @owncloud-issue-8619
  Scenario: create an internal link share of a folder using permissions endpoint
    Given user "Alice" has created folder "folder"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folder   |
      | space           | Personal |
      | permissionsRole | internal |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """

  @owncloud-issue-8619
  Scenario: try to create an internal link share of a folder with password using permissions endpoint
    Given user "Alice" has created folder "folder"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folder   |
      | space           | Personal |
      | permissionsRole | internal |
      | password        | %public% |
    Then the HTTP status code should be "400"

  @owncloud-issue-7879
  Scenario Outline: create a link share of a file using permissions endpoint
    Given user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile1.txt      |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |

  @owncloud-issue-8619
  Scenario: create an internal link share of a file using permissions endpoint
    Given user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | permissionsRole | internal      |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """

  @owncloud-issue-8619
  Scenario: try to create an internal link share of a file with password using permissions endpoint
    Given user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | permissionsRole | internal      |
      | password        | %public%      |
    Then the HTTP status code should be "400"

  @owncloud-issue-7879
  Scenario Outline: create a link share of a folder with display name and expiry date using permissions endpoint
    Given user "Alice" has created folder "folder"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource           | folder                   |
      | space              | Personal                 |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
      | displayName        | Homework                 |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link",
          "expirationDateTime"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          },
          "id": {
            "type": "string",
            "pattern": "^[a-zA-Z]{15}$"
          },
          "expirationDateTime": {
            "const": "2200-07-15T23:59:59Z"
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
                "const": "Homework"
              },
              "@libre.graph.quickLink": {
                "const": false
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |

  @owncloud-issue-7879
  Scenario Outline: create a link share of a file with display name and expiry date using permissions endpoint
    Given user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource           | textfile1.txt            |
      | space              | Personal                 |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
      | displayName        | Homework                 |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link",
          "expirationDateTime"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          },
          "id": {
            "type": "string",
            "pattern": "^[a-zA-Z]{15}$"
          },
          "expirationDateTime": {
            "const": "2200-07-15T23:59:59Z"
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
                "const": "Homework"
              },
              "@libre.graph.quickLink": {
                "const": false
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |

  @env-config @owncloud-issue-7879
  Scenario Outline: create a link share of a file without password using permissions endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile1.txt      |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | internal         |

  @env-config @owncloud-issue-9724 @owncloud-issue-10331
  Scenario: set password on a file's link share using permissions endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | permissionsRole | view          |
    When user "Alice" sets the following password for the last link share using the Graph API:
      | resource | textfile1.txt |
      | space    | Personal      |
      | password | %public%      |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          }
        }
      }
      """
    And the public should be able to download file "textfile1.txt" from the last link share with password "%public%" and the content should be "other data"


  Scenario Outline: create a file's link share with a password that is listed in the Banned-Password-List using permissions endpoint
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And user "Alice" has uploaded file with content "other data" to "text.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | text.txt          |
      | space           | Personal          |
      | permissionsRole | view              |
      | password        | <banned-password> |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety"
              }
            }
          }
        }
      }
      """
    Examples:
      | banned-password |
      | 123             |
      | password        |
      | OpenCloud        |

  @owncloud-issue-7879
  Scenario Outline: create a link share of a folder inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare      |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: create an internal link share of a folder inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare |
      | space           | projectSpace  |
      | permissionsRole | internal      |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """

  @owncloud-issue-8619
  Scenario: try to create an internal link share of a folder inside project-space with password using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare |
      | space           | projectSpace  |
      | permissionsRole | internal      |
      | password        | %public%      |
    Then the HTTP status code should be "400"

  @owncloud-issue-7879
  Scenario Outline: create a link share of a folder inside project-space with display name and expiry date using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource           | folderToShare            |
      | space              | projectSpace             |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
      | displayName        | Homework                 |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link",
          "expirationDateTime"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          },
          "id": {
            "type": "string",
            "pattern": "^[a-zA-Z]{15}$"
          },
          "expirationDateTime": {
            "const": "2200-07-15T23:59:59Z"
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
                "const": "Homework"
              },
              "@libre.graph.quickLink": {
                "const": false
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario Outline: create a link share of a folder inside project-space with a password that is listed in the Banned-Password-List using permissions endpoint
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare      |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | <banned-password>  |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety"
              }
            }
          }
        }
      }
      """
    Examples:
      | banned-password | permissions-role |
      | 123             | view             |
      | password        | view             |
      | OpenCloud       | view             |
      | 123             | edit             |
      | password        | edit             |
      | OpenCloud       | edit             |
      | 123             | upload           |
      | password        | upload           |
      | OpenCloud       | upload           |
      | 123             | createOnly       |
      | password        | createOnly       |
      | OpenCloud       | createOnly       |

  @env-config @owncloud-issue-7879
  Scenario Outline: create a link share of a file inside project-space without password using permissions endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare      |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |

  @owncloud-issue-7879
  Scenario Outline: create a link share of a file inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt       |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |

  @owncloud-issue-8619
  Scenario: create an internal link share of a file inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | permissionsRole | internal     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """

  @owncloud-issue-8619
  Scenario: try to create an internal link share of a file inside project-space with password from project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | permissionsRole | internal     |
      | password        | %public%     |
    Then the HTTP status code should be "400"

  @owncloud-issue-7879
  Scenario Outline: create a link share of a file inside project-space with display name and expiry date using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource           | textfile.txt             |
      | space              | projectSpace             |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
      | displayName        | Homework                 |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link",
          "expirationDateTime"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          },
          "id": {
            "type": "string",
            "pattern": "^[a-zA-Z]{15}$"
          },
          "expirationDateTime": {
            "const": "2200-07-15T23:59:59Z"
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
                "const": "Homework"
              },
              "@libre.graph.quickLink": {
                "const": false
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |

  @env-config @owncloud-issue-7879
  Scenario Outline: create a link share of a file inside project-space without password using permissions endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt       |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | internal         |


  Scenario Outline: create a link share of a file inside project-space with a password that is listed in the Banned-Password-List using permissions endpoint
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt       |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | <banned-password>  |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety"
              }
            }
          }
        }
      }
      """
    Examples:
      | banned-password | permissions-role |
      | 123             | view             |
      | password        | view             |
      | OpenCloud       | view             |
      | 123             | edit             |
      | password        | edit             |
      | OpenCloud       | edit             |

  @env-config @owncloud-issue-9724 @owncloud-issue-10331
  Scenario: set password on a existing link share of a file inside project-space using permissions endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | permissionsRole | view         |
    When user "Alice" sets the following password for the last link share using the Graph API:
      | resource | textfile.txt |
      | space    | projectSpace |
      | password | %public%     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          }
        }
      }
      """
    And the public should be able to download file "textfile.txt" from the last link share with password "%public%" and the content should be "to share"

  @owncloud-issue-7879
  Scenario Outline: try to create a link share of a Personal and Shares drives using permissions endpoint
    When user "Alice" tries to create the following space link share using permissions endpoint of the Graph API:
      | space           | <drive>            |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
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
                "const": "<message>"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | drive    | message                                   |
      | view             | Shares   | no share permission                       |
      | edit             | Shares   | no share permission                       |
      | upload           | Shares   | no share permission                       |
      | createOnly       | Shares   | no share permission                       |
      | view             | Personal | cannot create link on personal space root |
      | edit             | Personal | cannot create link on personal space root |
      | upload           | Personal | cannot create link on personal space root |
      | createOnly       | Personal | cannot create link on personal space root |

  @owncloud-issue-7879
  Scenario Outline: create a link share of a project-space drive using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario Outline: try to create an internal link share of a Personal and Shares drives using permissions endpoint
    When user "Alice" tries to create the following space link share using permissions endpoint of the Graph API:
      | space           | <drive>  |
      | permissionsRole | internal |
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
                "const": "<message>"
              }
            }
          }
        }
      }
      """
    Examples:
      | drive    | message                                   |
      | Personal | cannot create link on personal space root |


  Scenario Outline: try to create an internal link share with password of a Personal and Shares drives using permissions endpoint
    When user "Alice" tries to create the following space link share using permissions endpoint of the Graph API:
      | space           | <drive>  |
      | permissionsRole | internal |
      | password        | %public% |
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
                "const": "password is redundant for the internal link"
              }
            }
          }
        }
      }
      """
    Examples:
      | drive    |
      | Personal |
      | Shares   |


  Scenario: create an internal link share of a project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace  |
      | permissionsRole | internal      |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """


  Scenario: try to create an internal link share of a project-space with password using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace  |
      | permissionsRole | internal      |
      | password        | %public%      |
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
                "const": "password is redundant for the internal link"
              }
            }
          }
        }
      }
      """


  Scenario Outline: create a link share of a project-space with display name and expiry date using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space              | projectSpace             |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
      | displayName        | Homework                 |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link",
          "expirationDateTime"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          },
          "id": {
            "type": "string",
            "pattern": "^[a-zA-Z]{15}$"
          },
          "expirationDateTime": {
            "const": "2200-07-15T23:59:59Z"
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
                "const": "Homework"
              },
              "@libre.graph.quickLink": {
                "const": false
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario Outline: try to create a link share of a project-space with a password that is listed in the Banned-Password-List using permissions endpoint
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | <banned-password>  |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety"
              }
            }
          }
        }
      }
      """
    Examples:
      | banned-password | permissions-role |
      | 123             | view             |
      | password        | view             |
      | OpenCloud        | view             |
      | 123             | edit             |
      | password        | edit             |
      | OpenCloud        | edit             |
      | 123             | upload           |
      | password        | upload           |
      | OpenCloud        | upload           |
      | 123             | createOnly       |
      | password        | createOnly       |
      | OpenCloud        | createOnly       |


  Scenario Outline: create a link share of a project-space without password using permissions endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario Outline: create a quick link share of a folder using permissions endpoint
    Given user "Alice" has created folder "folder"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folder             |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
      | quickLink       | true               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: create an internal quick link share of a folder using permissions endpoint
    Given user "Alice" has created folder "folder"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folder   |
      | space           | Personal |
      | permissionsRole | internal |
      | quickLink       | true     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """


  Scenario Outline: create a quick link share of a file using permissions endpoint
    Given user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile1.txt      |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
      | quickLink       | true               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |


  Scenario: create an internal quick link share of a file using permissions endpoint
    Given user "Alice" has uploaded file with content "other data" to "textfile1.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | permissionsRole | internal      |
      | quickLink       | true          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """


  Scenario Outline: create a quick link share of a folder inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare      |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
      | quickLink       | true               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: create an internal quick link share of a folder inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | folderToShare |
      | space           | projectSpace  |
      | permissionsRole | internal      |
      | quickLink       | true          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """


  Scenario Outline: create a quick link share of a file inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt       |
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
      | quickLink       | true               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |

  @owncloud-issue-8619
  Scenario: create an internal quick link share of a file inside project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | permissionsRole | internal     |
      | quickLink       | true         |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """


  Scenario Outline: create a quick link share of a project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
      | quickLink       | true               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "<permissions-role>"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: create an internal quick link share of a project-space using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace  |
      | permissionsRole | internal      |
      | quickLink       | true          |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
            }
          }
        }
      }
      """

  @owncloud-issue-8960
  Scenario Outline: create an internal link share by a member of a project space drive using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following space share invitation:
      | space           | projectSpace       |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates the following space link share using root endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
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

  @owncloud-issue-8960
  Scenario Outline: create an internal link share by a member of a project space drive using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following space share invitation:
      | space           | projectSpace       |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
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

  @owncloud-issue-8960
  Scenario Outline: create an internal quick link share by a member of a project space drive using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following space share invitation:
      | space           | projectSpace       |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
      | quickLink       | true               |
    When user "Brian" creates the following space link share using root endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
      | quickLink       | true         |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
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

  @owncloud-issue-8960
  Scenario Outline: create an internal quick link share by a member of a project space drive using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following space share invitation:
      | space           | projectSpace       |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
      | quickLink       | true               |
    When user "Brian" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
      | quickLink       | true         |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": {
            "const": false
          },
          "id": {
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
                "const": true
              },
              "preventsDownload": {
                "const": false
              },
              "type": {
                "const": "internal"
              },
              "webUrl": {
                "type": "string",
                "pattern": "^%base_url%/s/[a-zA-Z]{15}$"
              }
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

  @owncloud-issue-8960
  Scenario Outline: try to create an internal link share by a member of a project space drive with password using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following space share invitation:
      | space           | projectSpace       |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates the following space link share using root endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
      | quickLink       | true         |
      | password        | %public%     |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "password is redundant for the internal link"
              }
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

  @owncloud-issue-8960
  Scenario Outline: try to create an internal link share by a member of a project space drive with password using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following space share invitation:
      | space           | projectSpace       |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates the following space link share using permissions endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
      | quickLink       | true         |
      | password        | %public%     |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "password is redundant for the internal link"
              }
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

  @env-config
  Scenario: try to create a public link share of a folder with denied permissions role
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And user "Alice" has created folder "FolderToShare"
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | FolderToShare |
      | space           | Personal      |
      | permissionsRole | denied        |
      | password        | %public%      |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [ "code", "innererror", "message" ],
            "properties": {
              "code": {
                "const": "invalidRequest"
              },
              "innererror": {
                "type": "object",
                "required": [ "date", "request-id" ]
              },
              "message": {
                "const": "invalid body schema definition"
              }
            }
          }
        }
      }
      """
