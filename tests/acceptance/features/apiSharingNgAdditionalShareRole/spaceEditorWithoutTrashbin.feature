@env-config
Feature: an user shares resources
  As a user
  I don't want space editor to access deleted files
  So that they can't restore them


  Scenario: sharee checks trashbin after file is deleted
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has enabled the permissions role "Space Editor Without Trashbin"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space                     |
      | sharee          | Brian                         |
      | shareType       | user                          |
      | permissionsRole | Space Editor Without Trashbin |
    And user "Brian" has removed the file "textfile.txt" from space "new-space"
    When user "Brian" tries to list all deleted files in the trash bin of the space "new-space"
    Then the HTTP status code should be "403"
    When user "Brian" tries to restore the file "textfile.txt" from the trash of the space "new-space" to "/textfile.txt"
    Then the HTTP status code should be "403"
    And as "Alice" file "textfile.txt" should exist in the trashbin of the space "new-space"
