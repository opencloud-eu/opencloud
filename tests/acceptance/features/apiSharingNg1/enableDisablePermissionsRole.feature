@env-config
Feature: enable disable permissions role
  As a user
  I want to enable/disable permissions role on shared resources
  So that I can control the accessibility of shared resources to sharee

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: users list the shares shared with Secure Viewer after the role is disabled (Personal Space)
    Given the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has uploaded file with content "some content" to "textfile.txt"
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "hello world" to "folderToShare/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource>    |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And the administrator has disabled the permissions role "Secure Viewer"
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "<resource>" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@libre.graph.permissions.actions",
                "grantedToV2",
                "id",
                "invitation"
              ],
              "properties": {
                "@libre.graph.permissions.actions": {
                  "const": [
                      "libre.graph/driveItem/path/read",
                      "libre.graph/driveItem/children/read",
                      "libre.graph/driveItem/basic/read"
                  ]
                },
                "roles": { "const": null }
              }
            }
          }
        }
      }
      """
    When user "Brian" lists the shares shared with him using the Graph API
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
                "<resource-type>",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "<resource-type>",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "name": { "const": "<resource>" },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "@libre.graph.permissions.actions",
                          "grantedToV2",
                          "id",
                          "invitation"
                        ],
                        "properties": {
                          "@libre.graph.permissions.actions": {
                            "const": [
                                "libre.graph/driveItem/path/read",
                                "libre.graph/driveItem/children/read",
                                "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "roles": { "const": null }
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
    And user "Brian" should have a share "<resource>" shared by user "Alice" from space "Personal"
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "<resource>" with depth "0" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "<resource>" with these key and value pairs:
      | key            | value      |
      | oc:name        | <resource> |
      | oc:permissions | SX         |
    And user "Brian" should not be able to download file "<resource-to-download>" from space "Shares"
    Examples:
      | resource      | resource-type | resource-to-download        |
      | textfile.txt  | file          | textfile.txt                |
      | folderToShare | folder        | folderToShare/textfile1.txt |


  Scenario: users list the shares shared with Denied after the role is disabled (Personal Space)
    Given the administrator has enabled the permissions role "Denied"
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "hello world" to "folderToShare/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Denied        |
    And the administrator has disabled the permissions role "Denied"
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "folderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@libre.graph.permissions.actions",
                "grantedToV2",
                "id",
                "invitation"
              ],
              "properties": {
                "@libre.graph.permissions.actions": {
                  "const": ["none"]
                },
                "roles": { "const": null }
              }
            }
          }
        }
      }
      """
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 0,
            "maxItems": 0
          }
        }
      }
      """
    And user "Brian" should not have a share "folderToShare" shared by user "Alice" from space "Personal"
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "folderToShare" with depth "0" using the WebDAV API
    Then the HTTP status code should be "404"
    And user "Brian" should not be able to download file "folderToShare/textfile1.txt" from space "Shares"


  Scenario Outline: users list the shares shared with Secure Viewer after the role is disabled (Project Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has created a folder "folderToShare" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "folderToShare/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource>    |
      | space           | new-space     |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And the administrator has disabled the permissions role "Secure Viewer"
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "<resource>" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@libre.graph.permissions.actions",
                "grantedToV2",
                "id",
                "invitation"
              ],
              "properties": {
                "@libre.graph.permissions.actions": {
                  "const": [
                      "libre.graph/driveItem/path/read",
                      "libre.graph/driveItem/children/read",
                      "libre.graph/driveItem/basic/read"
                  ]
                },
                "roles": { "const": null }
              }
            }
          }
        }
      }
      """
    When user "Brian" lists the shares shared with him using the Graph API
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
                "eTag",
                "<resource-type>",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "<resource-type>",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "name": { "const": "<resource>" },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "@libre.graph.permissions.actions",
                          "grantedToV2",
                          "id",
                          "invitation"
                        ],
                        "properties": {
                          "@libre.graph.permissions.actions": {
                            "const": [
                                "libre.graph/driveItem/path/read",
                                "libre.graph/driveItem/children/read",
                                "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "roles": { "const": null }
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
    And user "Brian" should have a share "<resource>" shared by user "Alice" from space "new-space"
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "<resource>" with depth "0" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "<resource>" with these key and value pairs:
      | key            | value      |
      | oc:name        | <resource> |
      | oc:permissions | SX         |
    And user "Brian" should not be able to download file "<resource-to-download>" from space "Shares"
    Examples:
      | resource      | resource-type | resource-to-download        |
      | textfile.txt  | file          | textfile.txt                |
      | folderToShare | folder        | folderToShare/textfile1.txt |


  Scenario: users list the shares shared with Denied after the role is disabled (Project Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "folderToShare/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | new-space     |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Denied        |
    And the administrator has disabled the permissions role "Denied"
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "folderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@libre.graph.permissions.actions",
                "grantedToV2",
                "id",
                "invitation"
              ],
              "properties": {
                "@libre.graph.permissions.actions": {
                  "const": ["none"]
                },
                "roles": { "const": null }
              }
            }
          }
        }
      }
      """
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 0,
            "maxItems": 0
          }
        }
      }
      """
    And user "Brian" should not have a share "folderToShare" shared by user "Alice" from space "Personal"
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "folderToShare" with depth "0" using the WebDAV API
    Then the HTTP status code should be "404"
    And user "Brian" should not be able to download file "folderToShare/textfile1.txt" from space "Shares"


  Scenario: sharee lists drives after the share role Space Editor Without Versions has been disabled
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Space Editor Without Versions"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile1.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space                     |
      | sharee          | Brian                         |
      | shareType       | user                          |
      | permissionsRole | Space Editor Without Versions |
    And the administrator has disabled the permissions role "Space Editor Without Versions"
    When user "Brian" lists all available spaces via the Graph API
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "new-space" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "driveAlias",
          "name",
          "id",
          "quota",
          "root",
          "webUrl"
        ],
        "properties": {
          "root": {
            "type": "object",
            "required": [
              "eTag",
              "id",
              "webDavUrl"
            ]
          }
        }
      }
      """
    When user "Brian" sends PROPFIND request to space "new-space" with depth "0" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a space "new-space" with these key and value pairs:
      | key            | value     |
      | oc:name        | new-space |
      | oc:permissions | DNVCK     |
    And user "Brian" should be able to download file "textfile1.txt" from space "new-space"
