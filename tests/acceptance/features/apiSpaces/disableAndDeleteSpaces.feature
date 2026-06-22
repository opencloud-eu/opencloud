Feature: Disabling and deleting space
  As a manager of space
  I want to be able to disable the space first, then delete it.
  So that a disabled spaces isn't accessible by shared users.

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
      | Carol    |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project Moon" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    And user "Alice" has sent the following space share invitation:
      | space           | Project Moon |
      | sharee          | Bob          |
      | shareType       | user         |
      | permissionsRole | Space Viewer |


  Scenario Outline: user can disable their own space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" disables a space "Project Moon"
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "Project Moon"
    And the user "Bob" should not have a space called "Project Moon"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: user with role user and user light cannot disable other space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    When user "Carol" tries to disable a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    And the user "Brian" should have a space called "Project Moon"
    And the user "Bob" should have a space called "Project Moon"
    Examples:
      | user-role  |
      | User       |
      | User Light |


  Scenario: a space manager can disable and delete space in which files and folders exist via the webDav API
    Given user "Alice" has uploaded a file inside space "Project Moon" with content "test" to "test.txt"
    And user "Alice" has created a folder "MainFolder" in space "Project Moon"
    When user "Alice" disables a space "Project Moon"
    Then the HTTP status code should be "204"
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project Moon"


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


  Scenario Outline: user cannot delete their own disabled space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "<code>"
    And the user "Alice" <shouldOrNot> have a space called "Project Moon"
    Examples:
      | user-role   | code | shouldOrNot |
      | Admin       | 204  | should not  |
      | Space Admin | 204  | should not  |
      | User        | 403  | should      |
      | User Light  | 403  | should      |


  Scenario Outline: an admin and space manager can disable other space via the Graph API
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    When user "Carol" disables a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Carol" should not have a space called "Project Moon"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |


  Scenario Outline: an admin and space manager can delete other disabled Space
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


  Scenario Outline: user with role user and user light cannot disable space
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    When user "Carol" tries to delete a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | user-role  |
      | User       |
      | User Light |


  Scenario Outline: user with role user and user light cannot delete others disabled space
    Given the administrator has assigned the role "<user-role>" to user "Carol" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Carol" tries to delete a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | user-role  |
      | User       |
      | User Light |


  Scenario Outline: viewer and space editor cannot disable space
    When user "<user>" tries to disable a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "403"
    And the user "<user>" should have a space called "Project Moon"
    Examples:
      | user  |
      | Brian |
      | Bob   |


  Scenario Outline: viewer and space editor cannot delete disabled space
    Given user "Alice" has disabled a space "Project Moon"
    When user "<user>" tries to delete a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | user  |
      | Brian |
      | Bob   |
