Feature: Create a link share for a resource
  https://docs.opencloud.eu/libre-graph-api/#/drives.permissions/CreateLink

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |

  @owncloud-issue-7879
  Scenario Outline: try to create a link share of a Personal and Share drive using root endpoint
    When user "Alice" tries to create the following space link share using root endpoint of the Graph API:
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
                "const": "unsupported space type"
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role | drive    |
      | view             | Shares   |
      | edit             | Shares   |
      | upload           | Shares   |
      | createOnly       | Shares   |
      | view             | Personal |
      | edit             | Personal |
      | upload           | Personal |
      | createOnly       | Personal |


  Scenario Outline: try to create an internal link share of a Personal and Share drives using root endpoint
    When user "Alice" tries to create the following space link share using root endpoint of the Graph API:
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
                "const": "unsupported space type"
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


  Scenario Outline: try to create an internal link share with password of a Personal and Share drive using root endpoint
    When user "Alice" tries to create the following space link share using root endpoint of the Graph API:
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
                "const": "unsupported space type"
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

  @owncloud-issue-7879
  Scenario Outline: create a link share of a project-space drive with different permissions using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
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


  Scenario: create an internal link share of a project-space using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
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


  Scenario: try to create an internal link share of a project-space drive with password
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" tries to create the following space link share using root endpoint of the Graph API:
      | space           | projectSpace  |
      | permissionsRole | internal      |
      | password        | %public%      |
    Then the HTTP status code should be "400"

  @owncloud-issue-7879
  Scenario Outline: create a link share of a project-space drive with display name and expiry date using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
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

  @owncloud-issue-7879 @skip-local-run
  Scenario Outline: try to create a link share of a project-space drive with a password that is listed in the Banned-Password-List using root endpoint
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" tries to create the following space link share using root endpoint of the Graph API:
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
  Scenario Outline: create a link share of a project-space drive without password using root endpoint
    Given the following configs have been set:
      | config                                     | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
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
  Scenario Outline: create a link share of a project-space drive with display name using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
      | space              | projectSpace             |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
      | displayName        | Homework                 |
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
  Scenario Outline: create a link share of a project-space drive with expiry date using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
      | space              | projectSpace             |
      | permissionsRole    | <permissions-role>       |
      | password           | %public%                 |
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


  Scenario Outline: create quick link share of a project space drive using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
      | displayName     | Link               |
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
                "const": "Link"
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
      | upload           |
      | edit             |
      | createOnly       |


  Scenario: create an internal quick link share of a project space drive using root endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    When user "Alice" creates the following space link share using root endpoint of the Graph API:
      | space           | projectSpace |
      | permissionsRole | internal     |
      | displayName     | Link         |
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
                "const": "Link"
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
