@env-config
Feature: purge expired trash-bin items via CLI command
  As an admin
  I want to purge expired trash-bin items
  So that the storage does not fill up with old deleted files

  Background:
    Given the following configs have been set:
      | config                                                | value |
      | STORAGE_USERS_PURGE_TRASH_BIN_PERSONAL_DELETE_BEFORE  | 1s    |
      | STORAGE_USERS_PURGE_TRASH_BIN_PROJECT_DELETE_BEFORE   | 1s    |
    And user "Alice" has been created with default attributes


  Scenario: purge expired trash-bin items of a personal space
    Given user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And user "Alice" has deleted file "textfile.txt"
    And the user waits for 2 seconds
    When the administrator purges the expired trash-bin items using the CLI
    Then the command should be successful
    And user "Alice" lists the resources in the trashbin with depth "1" using the WebDAV API
    And as "Alice" file "textfile.txt" should not exist in the trashbin of the space "Personal"


  Scenario: purge expired trash-bin items of a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" deletes file "textfile.txt" from space "new-space" using file-id "<<FILEID>>"
    And the user waits for 2 seconds
    When the administrator purges the expired trash-bin items using the CLI
    Then the command should be successful
    And as "Alice" file "textfile.txt" should not exist in the trashbin of the space "new-space"
