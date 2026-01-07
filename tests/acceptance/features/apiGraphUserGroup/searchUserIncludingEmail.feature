@env-config
Feature: edit/search user including email

  Background:
    Given user "Alice" has been created with default attributes
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the user "Alice" has created a new user with the following attributes:
      | userName    | Brian             |
      | displayName | Brian Murphy      |
      | email       | brian@example.com |
      | password    | 1234              |
    And the config "OC_SHOW_USER_EMAIL_IN_RESULTS" has been set to "true"


  Scenario Outline: admin user can edit another user's email
    When the user "Alice" changes the email of user "Brian" to "<new-email>" using the Graph API
    Then the HTTP status code should be "<http-status-code>"
    And the user information of "Brian" should match this JSON schema
      """
      {
        "type": "object",
        "required": ["mail"],
        "properties": {
          "mail": {
            "const": "<expected-email>"
          }
        }
      }
      """
    Examples:
      | action description        | new-email            | http-status-code | expected-email       |
      | change to a valid email   | newemail@example.com | 200              | newemail@example.com |
      | override existing mail    | brian@example.com    | 200              | brian@example.com    |
      | two users with same mail  | alice@example.org    | 200              | alice@example.org    |
      | empty mail                |                      | 400              | brian@example.com    |
      | change to a invalid email | invalidEmail         | 400              | brian@example.com    |


  Scenario Outline: normal user should not be able to change their email address
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When the user "Brian" tries to change the email of user "Brian" to "newemail@example.com" using the Graph API
    Then the HTTP status code should be "403"
    And the user information of "Brian" should match this JSON schema
      """
      {
        "type": "object",
        "required": ["mail"],
        "properties": {
          "mail": {
            "const": "brian@example.com"
          }
        }
      }
      """
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: normal user should not be able to edit another user's email
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And the user "Alice" has created a new user with the following attributes:
      | userName    | Carol             |
      | displayName | Carol King        |
      | email       | carol@example.com |
      | password    | 1234              |
    And the administrator has assigned the role "<user-role-2>" to user "Carol" using the Graph API
    When the user "Brian" tries to change the email of user "Carol" to "newemail@example.com" using the Graph API
    Then the HTTP status code should be "403"
    And the user information of "Carol" should match this JSON schema
      """
      {
        "type": "object",
        "required": ["mail"],
        "properties": {
          "mail": {
            "const": "carol@example.com"
          }
        }
      }
      """
    Examples:
      | user-role   | user-role-2 |
      | Space Admin | Space Admin |
      | Space Admin | User        |
      | Space Admin | User Light  |
      | Space Admin | Admin       |
      | User        | Space Admin |
      | User        | User        |
      | User        | User Light  |
      | User        | Admin       |
      | User Light  | Space Admin |
      | User Light  | User        |
      | User Light  | User Light  |
      | User Light  | Admin       |


  Scenario: admin user gets the information of a user
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When user "Alice" gets information of user "Brian" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "displayName",
          "id",
          "mail",
          "onPremisesSamAccountName",
          "accountEnabled",
          "userType"
        ],
        "properties": {
          "displayName": {
            "const": "Brian Murphy"
          },
          "id" : {
            "type": "string",
            "pattern": "^%user_id_pattern%$"
          },
          "mail": {
            "const": "brian@example.com"
          },
          "onPremisesSamAccountName": {
            "const": "Brian"
          },
          "accountEnabled": {
            "const": true
          },
          "userType": {
            "const": "Member"
          }
        }
      }
      """


  Scenario Outline: user gets his/her own information along with drive information
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When the user "Brian" gets his drive information using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
        {
          "type": "object",
          "required": [
            "displayName",
            "id",
            "mail",
            "onPremisesSamAccountName",
            "drive",
            "accountEnabled",
            "userType"
          ],
          "properties": {
            "displayName": {
              "const": "Brian Murphy"
            },
            "id" : {
              "type": "string",
              "pattern": "^%user_id_pattern%$"
            },
            "mail": {
              "const": "brian@example.com"
            },
            "onPremisesSamAccountName": {
              "const": "Brian"
            },
            "accountEnabled": {
              "const": true
            },
            "userType": {
              "const": "Member"
            },
            "drive": {
              "type": "object",
              "required": [
                "driveAlias",
                "id",
                "name",
                "owner",
                "quota",
                "root",
                "webUrl"
              ],
              "properties": {
                "driveType" : {
                  "const": "personal"
                },
                "driveAlias" : {
                  "const": "personal/brian"
                },
                "id" : {
                  "type": "string",
                  "pattern": "^%space_id_pattern%$"
                },
                "name": {
                  "const": "Brian Murphy"
                },
                "owner": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["id"],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "%user_id_pattern%"
                        }
                      }
                    }
                  }
                },
                "quota": {
                  "type": "object",
                  "required": ["state"],
                  "properties": {
                    "state": {
                      "const": "normal"
                    }
                  }
                },
                "root": {
                  "type": "object",
                  "required": ["id", "webDavUrl"],
                  "properties": {
                    "state": {
                      "const": "normal"
                    },
                    "webDavUrl": {
                      "type": "string",
                      "pattern": "^%base_url%/dav/spaces/%space_id_pattern%$"
                    }
                  }
                },
                "webUrl": {
                  "type": "string",
                  "pattern": "^%base_url%/f/%space_id_pattern%$"
                }
              }
            }
          }
        }
      """
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: non-admin user searches other users by display name
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When user "Brian" searches for user "ali" using Graph API
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
                "displayName",
                "id",
                "onPremisesSamAccountName",
                "mail",
                "userType"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                },
                "onPremisesSamAccountName": {
                  "const": "Alice"
                },
                "mail": {
                  "const": "alice@example.org"
                },
                "userType": {
                  "const": "Member"
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |

  @issue-7990
  Scenario: non-admin user searches other users by e-mail
    When user "Brian" searches for user "%22alice@example.org%22" using Graph API
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
                "displayName",
                "id",
                "mail",
                "userType"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                },
                "mail": {
                  "const": "alice@example.org"
                },
                "userType": {
                  "const": "Member"
                }
              }
            }
          }
        }
      }
      """


  Scenario: non-admin user searches for a disabled users
    Given the user "Admin" has disabled user "Alice"
    When user "Brian" searches for user "alice" using Graph API
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
                "displayName",
                "id",
                "onPremisesSamAccountName",
                "mail",
                "userType"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                },
                "onPremisesSamAccountName": {
                  "const": "Alice"
                },
                "mail": {
                  "const": "alice@example.org"
                },
                "userType": {
                  "const": "Member"
                }
              }
            }
          }
        }
      }
      """


  Scenario: non-admin user searches for multiple users having same displayname
    Given the user "Admin" has created a new user with the following attributes:
      | userName    | another-alice                |
      | displayName | Alice Murphy                 |
      | email       | another-alice@example.org    |
      | password    | containsCharacters(*:!;_+-&) |
    When user "Brian" searches for user "alice" using Graph API
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
                  "required": [
                    "displayName",
                    "id",
                    "onPremisesSamAccountName",
                    "mail",
                    "userType"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Alice Hansen"
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "onPremisesSamAccountName": {
                      "const": "Alice"
                    },
                    "mail": {
                      "const": "alice@example.org"
                    },
                    "userType": {
                      "const": "Member"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "displayName",
                    "id",
                    "mail",
                    "userType"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Alice Murphy"
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "onPremisesSamAccountName": {
                      "const": "another-alice"
                    },
                    "mail": {
                      "const": "another-alice@example.org"
                    },
                    "userType": {
                      "const": "Member"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario Outline: search other users when OC_SHOW_USER_EMAIL_IN_RESULTS config is disabled
    Given the config "OC_SHOW_USER_EMAIL_IN_RESULTS" has been set to "false"
    And the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" searches for user "Brian" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the search response should not contain user email
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |
