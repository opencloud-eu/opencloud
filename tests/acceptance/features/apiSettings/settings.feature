Feature: settings api
  As a user,
  I want to use settings api
  So that I can manage user specific settings

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario: disable auto-sync share
    When user "Brian" disables the auto-sync share using the settings API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value" : {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier": {
                "type": "object",
                "required": [ "extension", "bundle", "setting"],
                "properties": {
                  "extension": { "const": "opencloud-accounts" },
                  "bundle": { "const": "profile" },
                  "setting": { "const": "auto-accept-shares" }
                }
              },
              "value": {
                "type": "object",
                "required": [ "id", "bundleId", "settingId", "accountUuid", "resource", "boolValue" ],
                "properties": {
                  "id": { "pattern": "^%user_id_pattern%$" },
                  "bundleId": { "pattern": "^%user_id_pattern%$" },
                  "settingId": { "pattern": "^%user_id_pattern%$" },
                  "accountUuid": { "pattern": "^%user_id_pattern%$" },
                  "resource": {
                    "type": "object",
                    "required": ["type"],
                    "properties": {
                      "type": { "const": "TYPE_USER" }
                    }
                  },
                  "boolValue": { "const": false }
                }
              }
            }
          }
        }
      }
      """


  Scenario: assign role to user
    When user "Admin" assigns the role "Admin" to user "Alice" using the settings API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["assignment"],
        "properties": {
          "assignment" : {
            "type": "object",
            "required": [ "id", "accountUuid", "roleId" ],
            "properties": {
              "id": { "pattern": "^%user_id_pattern%$" },
              "accountUuid": { "pattern": "^%user_id_pattern%$" },
              "roleId": { "pattern": "^%user_id_pattern%$" }
            }
          }
        }
      }
      """

  @owncloud-issue-5032
  Scenario: user lists assignments
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When user "Alice" tries to get list of assignment using the settings API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["assignments"],
        "properties": {
          "assignments" : {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "uniqueItems": true,
            "items": {
              "type": "object",
              "required": ["id","accountUuid","roleId"],
              "properties": {
                "id": { "pattern": "^%user_id_pattern%$" },
                "accountUuid": { "pattern": "^%user_id_pattern%$" },
                "roleId": { "pattern": "^%user_id_pattern%$" }
              }
            }
          }
        }
      }
      """


  Scenario: switch language
    When user "Alice" switches the system language to "de" using the settings API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value" : {
            "type": "object",
            "required": [ "identifier", "value"],
            "properties": {
              "identifier": {
                "type": "object",
                "required": [ "extension", "bundle", "setting"],
                "properties": {
                  "extension": { "const": "opencloud-accounts" },
                  "bundle": { "const": "profile" },
                  "setting": { "const": "language" }
                }
              },
              "value": {
                "type": "object",
                "required": [ "id", "bundleId", "settingId", "accountUuid", "resource", "listValue" ],
                "properties": {
                  "id": { "pattern": "^%user_id_pattern%$" },
                  "bundleId": { "pattern": "^%user_id_pattern%$" },
                  "settingId": { "pattern": "^%user_id_pattern%$" },
                  "accountUuid": { "pattern": "^%user_id_pattern%$" },
                  "resource": {
                    "type": "object",
                    "required": ["type"],
                    "properties": {
                      "type": { "const": "TYPE_USER" }
                    }
                  },
                  "listValue": {
                    "type": "object",
                    "required": ["values"],
                    "properties": {
                      "values": {
                        "type": "array",
                        "minItems": 1,
                        "maxItems": 1,
                        "items": {
                          "type": "object",
                          "required": ["stringValue"],
                          "properties": {
                            "stringValue": { "const": "de" }
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

  @owncloud-issue-5079
  Scenario Outline: user lists existing roles
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to get all existing roles using the settings API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["bundles"],
        "properties": {
          "bundles" : {
            "type": "array",
            "minItems": 4,
            "maxItems": 4,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id", "name", "type", "extension", "displayName", "settings", "resource"],
                  "properties": {
                    "id": { "pattern": "^%user_id_pattern%$" },
                    "name": { "const": "spaceadmin" },
                    "type": { "const": "TYPE_ROLE" },
                    "extension": { "const": "opencloud-roles" },
                    "displayName": { "const": "Space Admin" },
                    "resource": {
                      "type": "object",
                      "required": ["type"],
                      "properties": {
                        "type": { "const": "TYPE_SYSTEM" }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id", "name", "type", "extension", "displayName", "settings", "resource"],
                  "properties": {
                    "id": { "pattern": "^%user_id_pattern%$" },
                    "name": { "const": "admin"},
                    "type": { "const": "TYPE_ROLE" },
                    "extension": { "const": "opencloud-roles" },
                    "displayName": { "const": "Admin" },
                    "resource": {
                      "type": "object",
                      "required": ["type"],
                      "properties": {
                        "type": { "const": "TYPE_SYSTEM" }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id", "name", "type", "extension", "displayName", "settings", "resource"],
                  "properties": {
                    "id": { "pattern": "^%user_id_pattern%$" },
                    "name": { "const": "user-light" },
                    "type": { "const": "TYPE_ROLE" },
                    "extension": { "const": "opencloud-roles" },
                    "displayName": { "const": "User Light" },
                    "resource": {
                      "type": "object",
                      "required": ["type"],
                      "properties": {
                        "type": { "const": "TYPE_SYSTEM" }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id", "name", "type", "extension", "displayName", "settings", "resource"],
                  "properties": {
                    "id": { "pattern": "^%user_id_pattern%$" },
                    "name": { "const": "user" },
                    "type": { "const": "TYPE_ROLE" },
                    "extension": { "const": "opencloud-roles" },
                    "displayName": { "const": "User" },
                    "resource": {
                      "type": "object",
                      "required": ["type"],
                      "properties": {
                        "type": { "const": "TYPE_SYSTEM" }
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
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
