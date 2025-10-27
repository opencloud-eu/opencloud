Feature: Restoring space
  As a manager of space
  I want to be able to restore a disabled space
  So that I can retrieve all the data belonging to the space

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "restore a space" of type "project" with quota "10"
    And using spaces DAV path


  Scenario: owner can restore a space via the Graph API
    Given user "Alice" has disabled a space "restore a space"
    When user "Alice" restores a disabled space "restore a space"
    Then the HTTP status code should be "200"


  Scenario: participants can see the data after the space is restored
    Given user "Alice" has created a folder "mainFolder" in space "restore a space"
    And user "Alice" has uploaded a file inside space "restore a space" with content "example" to "test.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | restore a space |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | Space Editor    |
    And user "Alice" has sent the following space share invitation:
      | space           | restore a space |
      | sharee          | Bob             |
      | shareType       | user            |
      | permissionsRole | Space Viewer    |
    And user "Alice" has disabled a space "restore a space"
    When user "Alice" restores a disabled space "restore a space"
    Then for user "Alice" the space "restore a space" should contain these entries:
      | test.txt   |
      | mainFolder |
    And for user "Brian" the space "restore a space" should contain these entries:
      | test.txt   |
      | mainFolder |
    And for user "Bob" the space "restore a space" should contain these entries:
      | test.txt   |
      | mainFolder |


  Scenario: participant can create data in the space after restoring
    Given user "Alice" has sent the following space share invitation:
      | space           | restore a space |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | Space Editor    |
    And user "Alice" has disabled a space "restore a space"
    And user "Alice" has restored a disabled space "restore a space"
    When user "Brian" creates a folder "mainFolder" in space "restore a space" using the WebDav Api
    And user "Brian" uploads a file inside space "restore a space" with content "test" to "test.txt" using the WebDAV API
    Then for user "Brian" the space "restore a space" should contain these entries:
      | test.txt   |
      | mainFolder |


  Scenario Outline: user without space manager role cannot restore space
    Given user "Alice" has sent the following space share invitation:
      | space           | restore a space |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | <space-role>    |
    And user "Alice" has disabled a space "restore a space"
    When user "Brian" tries to restore a disabled space "restore a space" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | space-role   |
      | Space Viewer |
      | Space Editor |


  Scenario Outline: user with role user and user light cannot restore space
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has disabled a space "restore a space"
    When user "Brian" tries to restore a disabled space "restore a space" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | user-role  |
      | User       |
      | User Light |

  @owncloud-issue-5872
  Scenario Outline: admin and space admin can restore other space
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has disabled a space "restore a space"
    When user "Brian" restores a disabled space "restore a space" owned by user "Alice"
    Then the HTTP status code should be "200"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
