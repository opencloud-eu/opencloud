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
        "required": ["folder", "children"],
        "properties": {
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
              "properties": {
                "parentReference": {
                  "type": "object",
                  "properties": {
                    "path": {
                      "not": { "pattern": "deep" }
                    }
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

  Scenario: listing a password protected public link without the password fails
    When the public lists the children of the last created public link using the Graph API
    Then the HTTP status code should be "401"

  Scenario: listing a password protected public link with a wrong password fails
    When the public lists the children of the last created public link with password "wrong" using the Graph API
    Then the HTTP status code should be "401"

  Scenario: a resource outside the public link is not readable through its token
    When the public tries to get the resource "private.txt" of user "Alice" through the last created public link with password "%public%" using the Graph API
    Then the HTTP status code should be "404"
