@env-config @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
Feature: create a resources using collaborative posixfs

  Background:
    Given the config "STORAGE_USERS_POSIX_WATCH_FS" has been set to "true"
    And user "Alice" has been created with default attributes


  Scenario: create folder
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator creates folder "myFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    When the administrator lists the content of the POSIX storage folder of user "Alice"
    Then the command output should contain "myFolder"
    And as "Alice" folder "/myFolder" should exist
