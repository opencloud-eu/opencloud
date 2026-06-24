Feature: Disabling, restoring and deleting space
  As a manager of space I want to be able to disable the space first, then enable it again.
  So that a disabled space isn't accessible by shared users until it is restored,
  and so that data is protected from accidental deletion by a mandatory disable step first.
  Only a space administrator can delete a space.

  | action                           | space admin (user role) | space manager (space role) | space viewer/editor (space role) |
  | disable space                    | allowed (204)           | allowed (204)              | denied (403)                     |
  | list disabled space (/me/drives) | yes                     | yes                        | no                               |
  | enable space                     | allowed (200)           | allowed (200)              | denied (404)                     |
  | delete space                     | allowed (204)           | denied (403)               | denied (404)                     |

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project Moon" with the default quota using the Graph API
  ##############################################################################
  # DISABLE SPACE
  ##############################################################################

  Scenario Outline: disable space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Brian" disables a space "Project Moon"
    Then the HTTP status code should be "<code>"
    And the user "Brian" <shouldOrNot> have a space called "Project Moon"
    Examples:
      | user-role   | space-role   | code | shouldOrNot |
      | Space Admin | Space Viewer | 204  | should not  |
      | User        | Manager      | 204  | should      |
      | User        | Space Editor | 403  | should      |
      | User Light  | Space Viewer | 403  | should      |


  Scenario: user can disable their own space via the Graph API
    When user "Alice" has disabled a space "Project Moon"
    And user "Alice" lists all available spaces via the Graph API with query "$filter=driveType eq 'project'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Moon" and match
      """
      {
        "type": "object",
        "required": [
          "name",
          "driveType"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Moon"]
          },
          "driveType": {
            "type": "string",
            "enum": ["project"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          },
          "root": {
            "type": "object",
            "required": [
              "deleted"
            ],
            "properties": {
              "deleted": {
                "type": "object",
                "required": [
                  "state"
                ],
                "properties": {
                  "state": {
                    "type": "string",
                    "enum": ["trashed"]
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario Outline: an admin and space manager can disable other space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    When user "Carol" disables a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Carol" should not have a space called "Project Moon"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
  ##############################################################################
  # LIST DISABLED SPACE (/me/drives)
  ##############################################################################

  Scenario Outline: list disabled space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" has disabled a space "Project Moon"
    And the user "Brian" <shouldOrNot> have a space called "Project Moon"
    Examples:
      | user-role   | space-role   | shouldOrNot |
      | Space Admin | Space Viewer | should not  |
      | User        | Manager      | should      |
      | User        | Space Editor | should not  |
      | User Light  | Space Viewer | should not  |
  ##############################################################################
  # ENABLE SPACE
  ##############################################################################

  Scenario Outline: enable space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Alice" has disabled a space "Project Moon"
    When user "Brian" restores a disabled space "Project Moon"
    Then the HTTP status code should be "<code>"
    Examples:
      | user-role   | space-role   | code |
      | Space Admin | Space Viewer | 200  |
      | User        | Manager      | 200  |
      | User        | Space Editor | 404  |
      | User Light  | Space Viewer | 404  |


  Scenario: participants can see and create the data after the space is restored
    Given using spaces DAV path
    And user "Alice" has created a folder "mainFolder" in space "Project Moon"
    And user "Alice" has uploaded a file inside space "Project Moon" with content "example" to "test.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Carol        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Alice" has disabled a space "Project Moon"
    When user "Alice" restores a disabled space "Project Moon"
    Then for user "Alice" the space "Project Moon" should contain these entries:
      | test.txt   |
      | mainFolder |
    When user "Brian" creates a folder "newFolder" in space "Project Moon" using the WebDav Api
    And user "Brian" uploads a file inside space "Project Moon" with content "test" to "new.txt" using the WebDAV API
    And for user "Brian" the space "Project Moon" should contain these entries:
      | test.txt   |
      | mainFolder |
      | new.txt    |
      | newFolder  |
    Then for user "Carol" the space "Project Moon" should contain these entries:
      | test.txt   |
      | mainFolder |
      | new.txt    |
      | newFolder  |


  Scenario Outline: try to restore other space
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Brian" restores a disabled space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "<code>"
    Examples:
      | user-role   | code |
      | Admin       | 200  |
      | Space Admin | 200  |
      | User        | 404  |
      | User Light  | 404  |
  ##############################################################################
  # DELETE SPACE
  ##############################################################################

  Scenario Outline: delete space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Alice" has disabled a space "Project Moon"
    When user "Brian" deletes a space "Project Moon"
    Then the HTTP status code should be "<code>"
    Examples:
      | user-role   | space-role   | code |
      | Space Admin | Space Viewer | 204  |
      | User        | Manager      | 403  |
      | User        | Space Editor | 404  |
      | User Light  | Space Viewer | 404  |


  Scenario Outline: user cannot delete their own space without first disabling it
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "<code>"
    And the user "Alice" should have a space called "Project Moon"
    Examples:
      | user-role   | code |
      | Admin       | 400  |
      | Space Admin | 400  |
      | User        | 403  |
      | User Light  | 403  |


  Scenario: user can delete their own disabled space via the Graph API
    Given user "Alice" has disabled a space "Project Moon"
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project Moon"


  Scenario Outline: admin and space manager can delete other disabled Space
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Carol" deletes a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project Moon"
    And the user "Carol" should not have a space called "Project Moon"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |


  Scenario Outline: user cannot delete other disabled Space
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Carol" deletes a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | user-role  |
      | User       |
      | User Light |
