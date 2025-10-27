Feature: unassign user role
  As an admin
  I want to unassign the role of user
  So that the role of user is set to default

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: admin user unassigns the role of another user
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When user "Alice" unassigns the role of user "Brian" using the Graph API
    Then the HTTP status code should be "204"
    And user "Brian" should not have any role assigned
    When user "Brian" uploads file with content "this step will assign the role to default" to "assign-to-default.txt" using the WebDAV API
    And user "Brian" should have the role "User" assigned
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |

  @owncloud-issue-6035
  Scenario: admin user tries to unassign his/her own role
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When user "Alice" tries to unassign the role of user "Alice" using the Graph API
    Then the HTTP status code should be "403"
    And user "Alice" should have the role "Admin" assigned


  Scenario: non-admin user tries to unassign role of another user
    Given user "Brian" has been created with default attributes
    When user "Alice" tries to unassign the role of user "Brian" using the Graph API
    Then the HTTP status code should be "403"
    And user "Brian" should have the role "User" assigned


  Scenario: non-admin user tries to unassign role of nonexistent user
    When user "Alice" tries to unassign the role of user "nonexistent" using the Graph API
    Then the HTTP status code should be "403"
