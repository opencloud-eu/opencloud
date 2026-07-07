@env-config @skipOnOpencloud-posix-Storage
Feature: backup consistency
  As a user
  I want to check my data for inconsistencies
  So that I can backup my data

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: check backup consistency via CLI command
    Given these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    And user "Alice" has created folder "/uploadFolder"
    And user "Carol" has created folder "/uploadFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Carol" has deleted file "/uploadFolder"
    And the administrator has stopped the server
    When the administrator checks the backup consistency using the CLI
    Then the command should be successful
    And the command output should contain "💚 No inconsistency found. The backup in '%storage_path%' seems to be valid."

  @issue-9498 @issue-391
  Scenario: check backup consistency after uploading file multiple times via TUS
    Given user "Alice" uploads a file "filesForUpload/textfile.txt" to "/today.txt" with mtime "today" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" uploads a file "filesForUpload/textfile.txt" to "/today.txt" with mtime "today" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" uploads a file "filesForUpload/textfile.txt" to "/today.txt" with mtime "today" via TUS inside of the space "Personal" using the WebDAV API
    And the administrator has stopped the server
    When the administrator checks the backup consistency using the CLI
    Then the command should be successful
    And the command output should contain "💚 No inconsistency found. The backup in '%storage_path%' seems to be valid."
    And the administrator has started the server
    When user "Alice" gets the number of versions of file "today.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"

  @issue-9498 @issue-428
  Scenario: check backup consistency after uploading a file multiple times
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And the administrator has stopped the server
    When the administrator checks the backup consistency using the CLI
    Then the command should be successful
    And the command output should contain "💚 No inconsistency found. The backup in '%storage_path%' seems to be valid."
    And the administrator has started the server
    When user "Alice" gets the number of versions of file "/textfile0.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"
