Feature: listing the content of a public link via the Graph API
  As an anonymous visitor of a public link
  I want to list the shared folder through the Graph API
  So that clients can browse public links without WebDAV

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "publicfolder"
    And user "Alice" has created folder "publicfolder/sub"
    And user "Alice" has uploaded file with content "hello public" to "publicfolder/a.txt"
    And user "Alice" has uploaded file with content "nested" to "publicfolder/sub/b.txt"
    And user "Alice" has uploaded file with content "not shared" to "private.txt"
    And user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |


  Scenario: the public lists the children of a public link
    When the public lists the children of the last created public link with password "%public%" using the Graph API
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
                  "required": ["name", "folder"],
                  "properties": {
                    "name": { "const": "sub" }
                  }
                },
                {
                  "type": "object",
                  "required": ["name", "file", "size"],
                  "properties": {
                    "name": { "const": "a.txt" },
                    "size": { "const": 12 }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: the public lists a subfolder of a public link by path
    When the public lists the children of path "sub" of the last created public link with password "%public%" using the Graph API
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
              "required": ["name", "file"],
              "properties": {
                "name": { "const": "b.txt" }
              }
            }
          }
        }
      }
      """


  Scenario: the public expands the children of the public link root
    When the public gets the root of the last created public link expanding its children with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["name", "folder", "children"],
        "not": { "required": ["parentReference"] },
        "properties": {
          "name": { "const": "publicfolder" },
          "children": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "type": "object",
              "required": ["name"]
            }
          }
        }
      }
      """


  Scenario: the public gets a file through an item anchored colon path
    When the public gets the drive item "b.txt" below the child "sub" of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["name", "file", "parentReference"],
        "properties": {
          "name": { "const": "b.txt" },
          "parentReference": {
            "type": "object",
            "required": ["path"],
            "properties": {
              "path": { "const": "/sub" }
            }
          }
        }
      }
      """


  Scenario: the public must not see the owner's paths above the share
    Given user "Alice" has created folder "deep"
    And user "Alice" has created folder "deep/shared"
    And user "Alice" has uploaded file with content "x" to "deep/shared/c.txt"
    And user "Alice" has created the following resource link share:
      | resource        | deep/shared |
      | space           | Personal    |
      | permissionsRole | view        |
      | password        | %public%    |
    When the public lists the children of the last created public link with password "%public%" using the Graph API
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
            "uniqueItems": true,
            "items": {
              "type": "object",
              "required": ["parentReference"],
              "properties": {
                "parentReference": {
                  "type": "object",
                  "required": ["path", "driveId"],
                  "properties": {
                    "path": { "const": "/" },
                    "driveId": { "pattern": "^7993447f-687f-490d-875c-ac95e89a62a4\\$7993447f-687f-490d-875c-ac95e89a62a4!" }
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario: advertised permissions inside a public link stay within the link role
    When the public gets the drive item of path "sub" of the last created public link selecting the allowed actions with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["@libre.graph.permissions.actions.allowedValues"],
        "properties": {
          "@libre.graph.permissions.actions.allowedValues": {
            "type": "array",
            "minItems": 6,
            "maxItems": 6,
            "uniqueItems": true,
            "items": {
              "type": "string",
              "not": { "pattern": "/(delete|create|update|deny)$" }
            }
          }
        }
      }
      """


  Scenario: listing a password protected public link without the password reports that a password is required
    When the public lists the children of the last created public link using the Graph API
    Then the HTTP status code should be "401"
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
              "code": { "const": "publicLinkPasswordRequired" }
            }
          }
        }
      }
      """


  Scenario: listing a password protected public link with a wrong password reports an invalid password
    When the public lists the children of the last created public link with password "wrong" using the Graph API
    Then the HTTP status code should be "401"
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
              "code": { "const": "publicLinkPasswordInvalid" }
            }
          }
        }
      }
      """


  Scenario: an editable public link grants an upload session
    Given user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | edit         |
      | password        | %public%     |
    When the public creates an upload session for "up.txt" in the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["UploadURL"],
        "properties": {
          "UploadURL": {
            "type": "string",
            "pattern": "/data/"
          }
        }
      }
      """


  Scenario: a view only public link does not grant an upload session
    When the public creates an upload session for "up.txt" in the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "404"


  Scenario: a resource outside the public link is not readable through its token
    When the public tries to get the resource "private.txt" of user "Alice" through the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "404"


  Scenario: a public token cannot list drives
    When the public tries to list the drives using the token of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: a public token cannot read the owner's personal drive
    When the public tries to get the personal drive of user "Alice" through the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: an in-share item is not addressable through its real drive id
    When the public tries to get the child "sub" of the last created public link through its real drive id with password "%public%" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: a token does not open another link's drive
    When the public tries to list the children of a foreign public link drive using the last created token with password "%public%" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: collaborative share types are not disclosed to the public
    Given user "Brian" has been created with default attributes
    And user "Alice" has sent the following resource share invitation:
      | resource        | publicfolder/sub |
      | space           | Personal         |
      | sharee          | Brian            |
      | shareType       | user             |
      | permissionsRole | Viewer           |
    When the public lists the children of the last created public link selecting the share types with password "%public%" using the Graph API
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
              "type": "object",
              "not": { "required": ["@libre.graph.shareTypes"] }
            }
          }
        }
      }
      """


  Scenario: the owner's favorite flag is not disclosed to the public
    Given user "Alice" has marked file "publicfolder/a.txt" as favorite from space "Personal"
    When the public lists the children of the last created public link with password "%public%" using the Graph API
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
            "contains": {
              "type": "object",
              "required": ["name", "@libre.graph.me.following"],
              "properties": {
                "name": { "const": "a.txt" }
              }
            },
            "items": {
              "type": "object",
              "properties": {
                "@libre.graph.me.following": { "const": false }
              }
            }
          }
        }
      }
      """


  Scenario: the public sees a lock but not who holds it
    Given user "Alice" has locked file "publicfolder/a.txt" setting the following properties
      | lockscope | exclusive |
    When the public lists the children of the last created public link with password "%public%" using the Graph API
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
            "contains": {
              "type": "object",
              "required": ["name", "lockInfo"],
              "properties": {
                "name": { "const": "a.txt" }
              }
            },
            "items": {
              "type": "object",
              "properties": {
                "lockInfo": {
                  "type": "object",
                  "not": { "required": ["owners"] }
                }
              }
            }
          }
        }
      }
      """


  Scenario: a file drop link does not list its folder through the real id
    Given user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | createOnly   |
      | password        | %public%     |
    When the public tries to list the children of resource "publicfolder" of user "Alice" through the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "403"


  Scenario: a file drop link still reaches its drive and upload sessions
    Given user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | createOnly   |
      | password        | %public%     |
    When the public gets the drive of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    When the public creates an upload session for "drop.txt" in the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "200"


  Scenario: an internal link does not list its folder through the real id
    Given user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | internal     |
    When the public tries to list the children of resource "publicfolder" of user "Alice" through the last created public link using the Graph API
    Then the HTTP status code should be "403"


  Scenario Outline: a colon path does not climb above the public link
    When the public tries to get the drive item of path "<path>" of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "404"
    Examples:
      | path                  |
      | ../private.txt        |
      | %2E%2E/private.txt    |
      | sub/../../private.txt |


  Scenario: an upload session name does not climb above the public link
    Given user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | edit         |
      | password        | %public%     |
    When the public creates an upload session for "../evil.txt" in the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "400"


  Scenario: a single item hides the owner's favorite flag and lock holder
    Given user "Alice" has marked file "publicfolder/a.txt" as favorite from space "Personal"
    And user "Alice" has locked file "publicfolder/a.txt" setting the following properties
      | lockscope | exclusive |
    When the public tries to get the drive item of path "a.txt" of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["name", "@libre.graph.me.following", "lockInfo"],
        "properties": {
          "name": { "const": "a.txt" },
          "@libre.graph.me.following": { "const": false },
          "lockInfo": {
            "type": "object",
            "not": { "required": ["owners"] }
          }
        }
      }
      """


  Scenario Outline: writes through the public link surface are rejected
    Given user "Alice" has created the following resource link share:
      | resource        | publicfolder |
      | space           | Personal     |
      | permissionsRole | edit         |
      | password        | %public%     |
    When the public tries to <action> the child "a.txt" of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "<code>"
    And as "Alice" file "publicfolder/a.txt" should exist

    Examples:
      | action                  | code |
      | create a link for       | 404  |
      | delete                  | 400  |
      | rename                  | 400  |
      | list the permissions of | 404  |


  Scenario: the public sees who shared the link on the drive
    When the public gets the drive of the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["driveType", "owner"],
        "properties": {
          "driveType": { "const": "mountpoint" },
          "owner": {
            "type": "object",
            "required": ["user"],
            "properties": {
              "user": {
                "type": "object",
                "required": ["id", "displayName"],
                "properties": {
                  "id": { "pattern": "^%user_id_pattern%$" },
                  "displayName": { "const": "Alice Hansen" }
                }
              }
            }
          }
        }
      }
      """
