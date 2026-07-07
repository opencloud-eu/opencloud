@env-config
Feature: restore trash-bin items via CLI command
  As an admin
  I want to restore a specific trash-bin item or all trash-bin items of a space
  So that accidentally deleted files can be recovered individually or in bulk

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: restore a single trash-bin item of a personal space
    Given user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has deleted file "textfile.txt"
    When the administrator restores the trash-bin item with file-id "<<FILEID>>" of the personal space of user "Alice" using the CLI
    Then the command should be successful
    And the command output should contain "textfile.txt"
    When the administrator lists the trash-bin items of the personal space of user "Alice" using the CLI
    Then the command should be successful
    And the command output should contain "total count: 0"


  Scenario: restore a single trash-bin item of a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" deletes file "textfile.txt" from space "new-space" using file-id "<<FILEID>>"
    When the administrator restores the trash-bin item with file-id "<<FILEID>>" of space "new-space" using the CLI
    Then the command should be successful
    And the command output should contain "textfile.txt"
    When the administrator lists the trash-bin items of space "new-space" using the CLI
    Then the command should be successful
    And the command output should contain "total count: 0"


  Scenario: restore all trash-bin items of a personal space
    Given using spaces DAV path
    And user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And user "Alice" has uploaded file with content "some more data" to "anotherfile.txt"
    And user "Alice" has deleted file "textfile.txt"
    And user "Alice" has deleted file "anotherfile.txt"
    When the administrator restores all the trash-bin items of the personal space of user "Alice" using the CLI
    Then the command should be successful
    And the command output should contain "textfile.txt"
    And the command output should contain "anotherfile.txt"
    And for user "Alice" the space "Personal" should contain these entries:
      | textfile.txt |
      | anotherfile.txt |


  Scenario: restore all trash-bin items of a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has uploaded a file inside space "new-space" with content "some more data" to "anotherfile.txt"
    And we save it into "ANOTHER_FILEID"
    And user "Alice" deletes file "textfile.txt" from space "new-space" using file-id "<<FILEID>>"
    And user "Alice" deletes file "anotherfile.txt" from space "new-space" using file-id "<<ANOTHER_FILEID>>"
    When the administrator restores all the trash-bin items of space "new-space" using the CLI
    Then the command should be successful
    And for user "Alice" the space "new-space" should contain these entries:
      | textfile.txt    |
      | anotherfile.txt |
