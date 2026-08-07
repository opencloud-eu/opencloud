@env-config @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
Feature: posixfs consistency CLI command
  As an administrator
  I want to check and repair the consistency of the posix filesystem metadata
  So that files with broken metadata (name, blobsize, parent id, ...) get fixed

  # WATCH_FS=false so the watcher does not re-assimilate the corrupted metadata before the command runs
  Background:
    Given the config "STORAGE_USERS_POSIX_WATCH_FS" has been set to "false"
    And user "Alice" has been created with default attributes


  Scenario: consistency fixes a corrupted name attribute
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator sets the extended attribute "user.oc.name" of file "textfile.txt" of user "Alice" to "corrupted.txt" on the POSIX filesystem
    Then the command should be successful
    When the administrator gets the extended attribute "user.oc.name" of file "textfile.txt" of user "Alice" on the POSIX filesystem
    Then the command output should contain "corrupted.txt"
    When the administrator checks the posixfs consistency using the CLI
    Then the command should be successful
    And the command output should contain "Fixed name attribute"
    When the administrator gets the extended attribute "user.oc.name" of file "textfile.txt" of user "Alice" on the POSIX filesystem
    Then the command output should contain "textfile.txt"
    And the command output should not contain "corrupted.txt"


  Scenario: consistency fixes a corrupted blobsize attribute
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator sets the extended attribute "user.oc.blobsize" of file "textfile.txt" of user "Alice" to "0" on the POSIX filesystem
    Then the command should be successful
    When the administrator checks the posixfs consistency using the CLI
    Then the command should be successful
    And the command output should contain "Fixed blobsize"
    When the administrator gets the extended attribute "user.oc.blobsize" of file "textfile.txt" of user "Alice" on the POSIX filesystem
    Then the command output should contain "7"


  Scenario: consistency fixes corrupted checksums only with the --fix-checksums flag
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator sets the extended attribute "user.oc.cs.sha1" of file "textfile.txt" of user "Alice" to "corrupted" on the POSIX filesystem
    Then the command should be successful
    And the administrator checks the posixfs consistency using the CLI
    And the command should be successful
    And the command output should not contain "Fixed checksum"
    And the administrator checks the posixfs consistency using the CLI with flag "--fix-checksums"
    And the command should be successful
    And the command output should contain "Fixed checksum"
    And the administrator checks the posixfs consistency using the CLI with flag "--fix-checksums"
    And the command should be successful
    And the command output should not contain "Fixed checksum"
