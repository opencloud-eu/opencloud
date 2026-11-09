@env-config
Feature: an user shares resources
  As a user
  I don't want space editor to access file versions or the trash bin
  So that they can't see the versions or restore deleted files


  Scenario: space editor without versions without trash bin permissions cannot access versions or restore deleted files
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has enabled the permissions role "Space Editor Without Versions Without Trashbin"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "new content" to "textfile.txt"
    And user "Alice" has uploaded a file inside space "new-space" with content "newest content" to "textfile.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space                                      |
      | sharee          | Brian                                          |
      | shareType       | user                                           |
      | permissionsRole | Space Editor Without Versions Without Trashbin |
    When user "Brian" tries to get versions of the file "textfile.txt" from the space "new-space" using the WebDAV API
    Then the HTTP status code should be "403"
    When user "Brian" tries to download version of the file "textfile.txt" with the index "1" of the space "new-space" using the WebDAV API
    Then the HTTP status code should be "403"
    When user "Brian" removes the file "textfile.txt" from space "new-space"
    And user "Brian" tries to list all deleted files in the trash bin of the space "new-space"
    Then the HTTP status code should be "403"
    When user "Brian" tries to restore the file "textfile.txt" from the trash of the space "new-space" to "/textfile.txt"
    Then the HTTP status code should be "403"
    And as "Alice" file "textfile.txt" should exist in the trashbin of the space "new-space"