Feature: deleting resources using file id
  As a user
  I want to be able to delete resources using file id
  So that I can remove unwanted data

  Background:
    Given using spaces DAV path
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |

  @issue-3770
  Scenario: sharee with the Editor role tries to delete the shared folder using its id
    Given user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "inside" to "folderToShare/inside.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Brian" has a share "folderToShare" synced
    When user "Brian" tries to delete the shared folder "folderToShare" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" folder "folderToShare" should exist
    And as "Alice" file "folderToShare/inside.txt" should exist
    And as "Brian" folder "/Shares/folderToShare" should exist
