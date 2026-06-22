Feature: Space management
  As a user with space admin permission
  I want to be able to manage all existing project spaces
  So that
  - I can get all project space where I am not member using "graph/v1.0/drives" endpoint
  - I can edit space: change quota, name, description
  - I can enable, disable, delete space

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Alice" has created a space "Project" of type "project" with quota "10"


  Scenario: space admin user can see another project space even if he is not member of the space
    When user "Brian" lists all spaces via the Graph API with query "$filter=driveType eq 'project'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "name",
          "id"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project"]
          },
          "driveType": {
            "type": "string",
            "enum": ["project"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          }
        }
      }
      """
    And the json response should not contain a space with name "Alice Hansen"


  Scenario: space admin user can see another personal spaces
    When user "Brian" lists all spaces via the Graph API with query "$filter=driveType eq 'personal'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Alice Hansen" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "name",
          "id"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Alice Hansen"]
          },
          "driveType": {
            "type": "string",
            "enum": ["personal"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          }
        }
      }
      """
    And the json response should not contain a space with name "Project"


  Scenario: user without space admin permission cannot see another spaces
    When user "Carol" tries to list all spaces via the Graph API
    Then the HTTP status code should be "200"
    And the json response should not contain a space with name "Project"
    And the json response should not contain a space with name "Alice Hansen"


  Scenario: space admin user changes the name of the project space
    When user "Brian" changes the name of the "Project" space to "New Name" owned by user "Alice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "name"
        ],
        "properties": {
          "name": {
             "type": "string",
             "enum": ["New Name"]
          }
        }
      }
      """


  Scenario: user without space admin permission tries to change the name of the project space
    When user "Carol" tries to change the name of the "Project" space to "New Name" owned by user "Alice"
    Then the HTTP status code should be "404"
    And the user "Alice" should have a space called "Project"


  Scenario: space admin user changes the description of the project space
    When user "Brian" changes the description of the "Project" space to "New description" owned by user "Alice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "description"
        ],
        "properties": {
          "description": {
             "type": "string",
             "enum": ["New description"]
          }
        }
      }
      """


  Scenario: user without space admin permission tries to change the description of the project space
    Given user "Alice" has changed the description of the "Project" space to "old description"
    When user "Carol" tries to change the description of the "Project" space to "New description" owned by user "Alice"
    Then the HTTP status code should be "404"


  Scenario: space admin user disables the project space
    When user "Brian" disables a space "Project" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "Project"


  Scenario: user without space admin permission tries to disable the project space
    When user "Carol" tries to disable a space "Project" owned by user "Alice"
    Then the HTTP status code should be "404"


  Scenario Outline: space admin user tries to disable the personal space
    When user "<user>" disables a space "Alice Hansen" owned by user "Alice"
    Then the HTTP status code should be "<code>"
    Examples:
      | user  | code |
      # Brian is Space Admin and can currently list all Personal Spaces, so he
      # gets a 403 instead of a 404 here. This might change in the future see:
      # https://github.com/opencloud-eu/opencloud/issues/2979
      | Brian | 403  |
      | Carol | 404  |


  Scenario: space admin user deletes the project space
    Given user "Alice" has disabled a space "Project"
    When user "Brian" deletes a space "Project" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project"


  Scenario: user without space admin permission tries to delete the project space
    Given user "Alice" has disabled a space "Project"
    When user "Carol" tries to delete a space "Project" owned by user "Alice"
    Then the HTTP status code should be "404"


  Scenario: space admin user enables the project space
    Given user "Alice" has disabled a space "Project"
    When user "Brian" restores a disabled space "Project" owned by user "Alice"
    Then the HTTP status code should be "200"


  Scenario: user without space admin permission tries to enable the project space
    Given user "Alice" has disabled a space "Project"
    When user "Carol" tries to restore a disabled space "Project" owned by user "Alice"
    Then the HTTP status code should be "404"
    And the user "Alice" should have a space "Project" in the disable state
