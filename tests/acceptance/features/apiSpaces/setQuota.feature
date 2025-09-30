Feature: Set quota
  As a user
  I want to set quota to different users
  So that users can only take up a certain amount of storage space

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: admin sets personal space quota of user with different role
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When user "Alice" changes the quota of the personal space of user "Brian" to "100" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "quota"
        ],
        "properties": {
          "quota": {
            "type": "object",
            "required": [
              "total"
            ],
            "properties": {
              "total" : {
                "type": "number",
                "enum": [100]
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


  Scenario Outline: non-admin user tries to set the personal space quota of other users
    Given the administrator has assigned the role "<user-role-2>" to user "Alice" using the Graph API
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When user "Alice" changes the quota of the personal space of user "Brian" to "100" using the Graph API
    Then the HTTP status code should be "403"
    Examples:
      | user-role-2 | user-role   |
      | Space Admin | Admin       |
      | Space Admin | Space Admin |
      | Space Admin | User        |
      | Space Admin | User Light  |
      | User        | Admin       |
      | User        | Space Admin |
      | User        | User        |
      | User        | User Light  |
      | User Light  | Admin       |
      | User Light  | Space Admin |
      | User Light  | User        |
      | User Light  | User Light  |


  Scenario Outline: admin or space admin user sets a quota of a project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has created a space "Project Jupiter" of type "project" with quota "20"
    When user "Brian" changes the quota of the "Project Jupiter" space to "100" owned by user "Alice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "name",
          "quota"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "quota": {
            "type": "object",
            "required": [
              "total"
            ],
            "properties": {
              "total" : {
                "type": "number",
                "enum": [100]
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


  Scenario Outline: normal or user light user tries to set quota of a space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has created a space "Project Jupiter" of type "project" with quota "20"
    And user "Alice" has sent the following space share invitation:
      | space           | Project Jupiter |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | <space-role>    |
    When user "Brian" changes the quota of the "Project Jupiter" space to "100"
    Then the HTTP status code should be "403"
    Examples:
      | user-role  | space-role   |
      | User       | Space Viewer |
      | User       | Space Editor |
      | User       | Manager      |
      | User Light | Space Viewer |
      | User Light | Space Editor |
      | User Light | Manager      |


  Scenario: admin user can set their own personal space quota
    When user "Admin" changes the quota of the personal space of user "Alice" to "100" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "quota"
        ],
        "properties": {
          "quota": {
            "type": "object",
            "required": [
              "total"
            ],
            "properties": {
              "total" : {
                "type": "number",
                "enum": [100]
              }
            }
          }
        }
      }
      """


  Scenario Outline: non-admin user tries to set their own personal space quota
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" changes the quota of the personal space of user "Alice" to "100" using the Graph API
    Then the HTTP status code should be "403"
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |
