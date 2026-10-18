Feature: access a shared resource as a guest
  As a guest without an account
  I want to open the guest link I received by email
  So that I can access the resource that was shared with me

  Background:
    Given using SharingNG
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"

  @email
  Scenario: guest verifies the guest link and gets a session
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie
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

  @email
  Scenario: guest link can only be used once
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "409"
    And the response should not contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType", "permissionId"],
        "properties": {
          "errorType": { "const": "tokenAlreadyRedeemed" },
          "permissionId": { "const": "%last_share_id%" }
        }
      }
      """


  Scenario Outline: verifying an invalid guest link token fails
    When guest "guest@example.org" verifies the guest link token "<token>" using the Graph API
    Then the HTTP status code should be "<http-status-code>"
    And the response should not contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType", "permissionId"],
        "properties": {
          "errorType": { "const": "<error-type>" },
          "permissionId": { "const": "" }
        }
      }
      """
    Examples:
      | token                                                                                     | http-status-code | error-type     |
      |                                                                                           | 400              | invalidRequest |
      | invalid                                                                                   | 401              | tokenInvalid   |
      | v2.bJTW-HnCYqpGxkk1S1PQUOMm1vlt8W2FgZDlFSGtyu0.DXNg71vlfMV80QIp1qxl1G0MCi4RbRFSfM4rJk5Ibmw | 401              | tokenInvalid   |
      | v1.a.b                                                                                    | 401              | tokenInvalid   |
      | v1.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA.DXNg71vlfMV80QIp1qxl1G0MCi4RbRFSfM4rJk5Ibmw | 404              | tokenNotFound  |

  @email
  Scenario: guest lists the resource shared with them
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" lists the shares shared with them using the Graph API
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
              "required": ["name", "remoteItem"],
              "properties": {
                "name": { "const": "textfile.txt" },
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
                        "required": ["id", "grantedToV2"],
                        "properties": {
                          "id": { "const": "%last_share_id%" },
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["@libre.graph.userType", "id"],
                                "properties": {
                                  "@libre.graph.userType": { "const": "Guest" },
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
              }
            }
          }
        }
      }
      """

  @email
  Scenario: guest downloads the shared file
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "to share"

  @email
  Scenario Outline: guest updates the shared file depending on the role
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt       |
      | space           | Personal           |
      | sharee          | guest@example.org  |
      | shareType       | guest              |
      | permissionsRole | <permissions-role> |
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" uploads content "changed by guest" to file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "Alice" the content of the file "textfile.txt" of the space "Personal" should be "<content>"
    Examples:
      | permissions-role | http-status-code | content          |
      | Viewer           | 403              | to share         |
      | File Editor      | 204              | changed by guest |

  @email
  Scenario Outline: guest session only grants access to the guest endpoints
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" sends HTTP method "<method>" to URL "<url>" with the guest session
    Then the HTTP status code should be "401"
    Examples:
      | method | url                          |
      | GET    | /graph/v1.0/me               |
      | GET    | /graph/v1.0/me/drives        |
      | GET    | /graph/v1.0/users            |
      | GET    | /ocs/v2.php/cloud/user       |
      | GET    | /graph/v1beta1/me/drive/root |

  @email
  Scenario: guest cannot access other resources of the sharer
    Given user "Alice" has uploaded file with content "not shared" to "private.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" downloads file "private.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "404"

  @email
  Scenario: guest loses access when the share is removed
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    And user "Alice" has removed the access of guest "guest@example.org" from resource "textfile.txt" of space "Personal"
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "401"
    When guest "guest@example.org" lists the shares shared with them using the Graph API
    Then the HTTP status code should be "401"

  @email
  Scenario: guest link cannot be used after the share is removed
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And user "Alice" has removed the access of guest "guest@example.org" from resource "textfile.txt" of space "Personal"
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "404"
    And the response should not contain a guest session cookie

  @email
  Scenario: guest loses write access when the owner downgrades the role
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | File Editor       |
    And guest "guest@example.org" has verified the guest link from the last email
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole | Viewer       |
      | space           | Personal     |
      | resource        | textfile.txt |
    When guest "guest@example.org" uploads content "changed by guest" to file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the content of the file "textfile.txt" of the space "Personal" should be "to share"
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "to share"

  @email
  Scenario: guest keeps access when the owner renames the shared resource
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    And user "Alice" has moved file "textfile.txt" to "renamed.txt"
    When guest "guest@example.org" downloads file "renamed.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "to share"
    When guest "guest@example.org" lists the shares shared with them using the Graph API
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
              "required": ["name"],
              "properties": {
                "name": { "const": "renamed.txt" }
              }
            }
          }
        }
      }
      """
