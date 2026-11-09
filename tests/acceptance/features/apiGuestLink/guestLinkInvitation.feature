Feature: share a resource with a guest via email address
  As a user
  I want to share a resource with an email address of a person without an account
  So that the person can access the resource through a guest link

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"


  Scenario Outline: share a resource with a guest using an email address
    Given user "Alice" has created folder "folderToShare"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | <resource>         |
      | space           | Personal           |
      | sharee          | guest@example.org  |
      | shareType       | guest              |
      | permissionsRole | <permissions-role> |
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
              "required": ["id", "roles", "grantedToV2", "invitation"],
              "properties": {
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
                },
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["@libre.graph.userType", "displayName", "id"],
                      "properties": {
                        "@libre.graph.userType": { "const": "Guest" },
                        "displayName": { "const": "guest@example.org" },
                        "id": { "const": "guest@example.org" }
                      }
                    }
                  }
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
                            "@libre.graph.userType": { "const": "Member" },
                            "displayName": { "const": "Alice Hansen" },
                            "id": { "type": "string", "pattern": "^%user_id_pattern%$" }
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
    Examples:
      | resource      | permissions-role |
      | textfile.txt  | Viewer           |
      | textfile.txt  | File Editor      |
      | folderToShare | Viewer           |
      | folderToShare | Editor           |


  Scenario: email address of the guest is stored in its canonical lowercase form
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | Guest@Example.ORG |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
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
              "required": ["grantedToV2"],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["id"],
                      "properties": {
                        "id": { "const": "guest@example.org" }
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

  @email
  Scenario: guest receives an email with a guest link
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    Then the HTTP status code should be "200"
    And guest "guest@example.org" should have received the following email from user "Alice"
      """
      Hello,

      %displayname% has shared "textfile.txt" with you.

      The link below is personal and can only be used once. Please do not forward it.

      The link is only valid for 30 minutes.

      Click here to view it: %base_url%/g/%guest_link_token%
      """


  Scenario Outline: share the same resource with the same guest again
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | textfile.txt       |
      | space           | Personal           |
      | sharee          | <sharee>           |
      | shareType       | guest              |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": ["code"],
            "properties": {
              "code": { "const": "nameAlreadyExists" }
            }
          }
        }
      }
      """
    Examples:
      | sharee            | permissions-role |
      | guest@example.org | Viewer           |
      | guest@example.org | File Editor      |
      | Guest@Example.ORG | Viewer           |

  @email
  Scenario: share a resource with a guest again after the previous share was removed
    Given using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has received a new guest link
    And user "Alice" has removed the access of guest "guest@example.org" from resource "textfile.txt" of space "Personal"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has received a new guest link
    When guest "guest@example.org" verifies the previous guest link using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType"],
        "properties": {
          "errorType": { "const": "tokenNotFound" }
        }
      }
      """
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["permissionId"],
        "properties": {
          "permissionId": { "const": "%last_share_id%" }
        }
      }
      """
