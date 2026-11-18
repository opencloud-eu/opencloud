Feature: share jail only lists resources the sharee cannot reach otherwise
  As a user syncing the share jail with a desktop client
  I want every received resource to show up under exactly one path
  So that a file id maps to a single path on disk

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path

  @issue-4055
  Scenario: a share nested inside another received share is not listed as its own mountpoint
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "foo"
    And user "Alice" has created folder "foo/bar"
    And user "Alice" has uploaded file with content "hello world" to "foo/bar/baz.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | foo      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "foo" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | foo/bar  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "bar" synced
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
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": { "const": "foo" }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": { "const": "bar" }
                  }
                }
              ]
            }
          }
        }
      }
      """
    And for user "Brian" the space "Shares" should contain these entries:
      | foo |
    And for user "Brian" the space "Shares" should not contain these entries:
      | bar |
    And for user "Brian" folder "foo/bar" of the space "Shares" should contain these files:
      | baz.txt |
    When user "Brian" uploads a file inside space "Shares" with content "edited by Brian" to "foo/bar/baz.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Brian" the content of the file "foo/bar/baz.txt" of the space "Shares" should be "edited by Brian"

  @issue-4055
  Scenario: a share from a project space the sharee is a member of is not listed in the share jail
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "folder/file.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Brian" has a share "folder" synced
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
              "required": ["name"],
              "properties": {
                "name": { "const": "folder" }
              }
            }
          }
        }
      }
      """
    And for user "Brian" folder "folder" of the space "new-space" should contain these files:
      | file.txt |
    And for user "Brian" the space "Shares" should not contain these entries:
      | folder |
