Feature: assign role
  As an admin,
  I want to assign roles to users.
  So that users without an admin role cannot get the list of roles, assignments list and assign roles to users

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: get assigned role of a user
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When the administrator retrieves the assigned role of user "Alice" using the Graph API
    Then the HTTP status code should be "200"
    And the Graph API response should have the role "<user-role>"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |

  @owncloud-issue-5032
  Scenario Outline: get assigned role of a user via setting api
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to get list of assignment using the settings API
    Then the HTTP status code should be "<http-status-code>"
    And the setting API response should have the role "<user-role>"
    Examples:
      | user-role   | http-status-code |
      | Admin       | 201              |
      | Space Admin | 401              |
      | User        | 401              |
      | User Light  | 401              |


  Scenario Outline: get role of a user assigned via setting api
    Given the administrator has given "Alice" the role "<user-role>" using the settings api
    When the administrator retrieves the assigned role of user "Alice" using the Graph API
    Then the HTTP status code should be "200"
    And the Graph API response should have the role "<user-role>"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: non-admin user tries to get assigned role of another user
    Given user "Brian" has been created with default attributes
    When user "Alice" tries to get the assigned role of user "Brian" using the Graph API
    Then the HTTP status code should be "403"


  Scenario: non-admin user tries to get assigned role of nonexistent user
    Given user "Brian" has been created with default attributes
    When user "Alice" tries to get the assigned role of user "nonexistent" using the Graph API
    Then the HTTP status code should be "403"
