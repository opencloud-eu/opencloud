Feature: share a project space with a guest via email address
  As a space manager
  I want to invite a person without an account to a project space using their email address
  So that the person can access the space through a guest link

  Background:
    Given using spaces DAV path
    And using SharingNG
    And user "Alice" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "space content" to "textfile.txt"


  Scenario Outline: invite a guest to a project space using an email address
    When user "Alice" sends the following space share invitation using root endpoint of the Graph API:
      | space           | new-space          |
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
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |

  @email
  Scenario: guest invited to a project space gets access through the guest link
    Given user "Alice" has sent the following space share invitation:
      | space           | new-space         |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Space Viewer      |
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "new-space" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "space content"

  @email
  Scenario: guest gets access to a file shared from a project space
    Given user "Alice" has uploaded a file inside space "new-space" with content "not shared" to "private.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | new-space         |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "new-space" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "space content"
    When guest "guest@example.org" downloads file "private.txt" shared from space "new-space" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "404"
