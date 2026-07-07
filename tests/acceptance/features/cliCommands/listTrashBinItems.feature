@env-config
Feature: list trash-bin items of a space via CLI command
  As an admin
  I want to list the trash-bin items of a space
  So that I know which items are available to restore

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: list trash-bin items of a personal space
    Given user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And user "Alice" has deleted file "textfile.txt"
    When the administrator lists the trash-bin items of the personal space of user "Alice" using the CLI
    Then the command should be successful
    And the command output should contain "textfile.txt"


  Scenario: list trash-bin items of a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" deletes file "textfile.txt" from space "new-space" using file-id "<<FILEID>>"
    When the administrator lists the trash-bin items of space "new-space" using the CLI
    Then the command should be successful
    And the command output should contain "textfile.txt"
