@env-config
Feature: clean up old file versions according to a cleanup strategy
  As an administrator
  I want to delete old file versions according to a cleanup strategy
  So that files that are not changed anymore don't keep all their old versions

  # The strategies are written without spaces, as the CLI wrapper splits commands on spaces;
  # "2,auto,10" is the same as "2, auto, 10".

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: keep only the newest version of every file in all spaces
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 4" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 1" to "anotherFile.txt"
    And user "Alice" has uploaded file with content "version 2" to "anotherFile.txt"
    When the administrator cleans up the file versions of all spaces using the CLI with flags "--strategy 1"
    Then the command should be successful
    And the command output should contain "/textfile.txt: deleted 2 of 3 revisions"
    And the command output should not contain "/anotherFile.txt"
    And the command output should contain "deleted 2 revisions"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Alice" gets the number of versions of file "anotherFile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    And the content of file "textfile.txt" for user "Alice" should be "version 4"


  Scenario: a dry run only prints what would be deleted
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    When the administrator cleans up the file versions of all spaces using the CLI with flags "-s 0 --dry-run"
    Then the command should be successful
    And the command output should contain "/textfile.txt: would delete 2 of 2 revisions"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: clean up only a folder within the personal space
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "version 1" to "folder/inFolder.txt"
    And user "Alice" has uploaded file with content "version 2" to "folder/inFolder.txt"
    And user "Alice" has uploaded file with content "version 3" to "folder/inFolder.txt"
    And user "Alice" has uploaded file with content "version 1" to "outside.txt"
    And user "Alice" has uploaded file with content "version 2" to "outside.txt"
    And user "Alice" has uploaded file with content "version 3" to "outside.txt"
    When the administrator cleans up the file versions of "/folder" in space "Personal" of user "Alice" using the CLI with flags "-s 0"
    Then the command should be successful
    And the command output should contain "/folder/inFolder.txt: deleted 2 of 2 revisions"
    When user "Alice" gets the number of versions of file "folder/inFolder.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
    When user "Alice" gets the number of versions of file "outside.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: clean up only a project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded file with content "version 1" to "personal.txt"
    And user "Alice" has uploaded file with content "version 2" to "personal.txt"
    And user "Alice" has uploaded file with content "version 3" to "personal.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "version 1" to "project.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "version 2" to "project.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "version 3" to "project.txt"
    And we save it into "PROJECT_FILEID"
    When the administrator cleans up the file versions of space "projectSpace" of user "Alice" using the CLI with flags "-s 0"
    Then the command should be successful
    And the command output should contain "/project.txt: deleted 2 of 2 revisions"
    When user "Alice" gets the number of versions of file "project.txt" using file-id "<<PROJECT_FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
    When user "Alice" gets the number of versions of file "personal.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: print the results as JSON
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    When the administrator cleans up the file versions of all spaces using the CLI with flags "-s 0 --json"
    Then the command should be successful
    And the JSON output of the command should contain an object of type "file" with:
      | key       | value         |
      | path      | /textfile.txt |
      | dry_run   | false         |
      | revisions | 2             |
      | deleted   | 2             |
      | failed    | 0             |
    And the JSON output of the command should contain an object of type "summary" with:
      | key     | value |
      | deleted | 2     |
      | failed  | 0     |


  Scenario: a cleanup without a strategy fails when the configured one is disabled
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    When the administrator cleans up the file versions using the CLI with flags "--all"
    Then the command should exit with code "1"
    And the command output should contain "no strategy"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"


  Scenario: a cleanup stopped by its timeout exits with code 2
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    When the administrator cleans up the file versions of all spaces using the CLI with flags "-s 0 --timeout 1ns"
    Then the command should exit with code "2"
    And the command output should contain "stopped after the timeout of 1ns"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"


  Scenario: a strategy without spaces after its commas
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 4" to "textfile.txt"
    When the administrator cleans up the file versions of all spaces using the CLI with flags "-s auto,1"
    Then the command should be successful
    And the command output should contain "/textfile.txt: deleted 2 of 3 revisions"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"

  @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
  Scenario: clean up a folder given by its path on the POSIX filesystem
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "version 1" to "folder/inFolder.txt"
    And user "Alice" has uploaded file with content "version 2" to "folder/inFolder.txt"
    And user "Alice" has uploaded file with content "version 3" to "folder/inFolder.txt"
    And user "Alice" has uploaded file with content "version 1" to "outside.txt"
    And user "Alice" has uploaded file with content "version 2" to "outside.txt"
    And user "Alice" has uploaded file with content "version 3" to "outside.txt"
    When the administrator cleans up the file versions of "folder" of user "Alice" by its path on the POSIX filesystem using the CLI with flags "-s 0"
    Then the command should be successful
    And the command output should contain "/folder/inFolder.txt: deleted 2 of 2 revisions"
    When user "Alice" gets the number of versions of file "folder/inFolder.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
    When user "Alice" gets the number of versions of file "outside.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: the configured strategy cleans up a file whenever it gets a new version
    Given the following configs have been set:
      | config                        | value |
      | OC_REVISIONS_CLEANUP_STRATEGY | 2     |
      | OC_REVISIONS_CLEANUP_ASYNC    | false |
    And user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 4" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 5" to "textfile.txt"
    Then the number of versions of file "textfile.txt" for user "Alice" should eventually be "2"
    And the content of file "textfile.txt" for user "Alice" should be "version 5"


  Scenario: the configured strategy cleans up in the background by default
    Given the following configs have been set:
      | config                        | value |
      | OC_REVISIONS_CLEANUP_STRATEGY | 2     |
    And user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 4" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 5" to "textfile.txt"
    Then the number of versions of file "textfile.txt" for user "Alice" should eventually be "2"
    And the content of file "textfile.txt" for user "Alice" should be "version 5"


  Scenario: the configured strategy cleans up a file uploaded with TUS
    Given the following configs have been set:
      | config                        | value |
      | OC_REVISIONS_CLEANUP_STRATEGY | 2     |
    And user "Alice" has uploaded a file with content "version 1" to "textfile.txt" via TUS inside of the space "Personal"
    And user "Alice" has uploaded a file with content "version 2" to "textfile.txt" via TUS inside of the space "Personal"
    And user "Alice" has uploaded a file with content "version 3" to "textfile.txt" via TUS inside of the space "Personal"
    And user "Alice" has uploaded a file with content "version 4" to "textfile.txt" via TUS inside of the space "Personal"
    And user "Alice" has uploaded a file with content "version 5" to "textfile.txt" via TUS inside of the space "Personal"
    Then the number of versions of file "textfile.txt" for user "Alice" should eventually be "2"
    And the content of file "textfile.txt" for user "Alice" should be "version 5"

  # The age of a version is the time its content was written, i.e. the modification time the
  # file had before it was changed, not the time of that change: "version 1" is 60 days old
  # as soon as it becomes a version.
  Scenario Outline: the configured strategy measures the age of a version by the time its content was written
    Given the following configs have been set:
      | config                        | value      |
      | OC_REVISIONS_CLEANUP_STRATEGY | <strategy> |
      | OC_REVISIONS_CLEANUP_ASYNC    | false      |
    And user "Alice" has uploaded a file with content "version 1" and mtime "-60 days" to "textfile.txt"
    And user "Alice" has uploaded a file with content "version 2" and mtime "-50 days" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "<versions>"
    And the content of file "textfile.txt" for user "Alice" should be "version 3"
    Examples:
      | strategy | versions |
      | 30d      | 0        |
      | 1,30d    | 1        |
      | 90d      | 2        |

  # A restore makes the content from before the restore a version, which the cleanup keeps
  # whatever the strategy, so that the restore can be undone. The restored version itself
  # becomes the current content and is no longer a version.
  Scenario: restoring a version keeps the content from before the restore, so that the restore can be undone
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    And the following configs have been set:
      | config                        | value |
      | OC_REVISIONS_CLEANUP_STRATEGY | 0     |
      | OC_REVISIONS_CLEANUP_ASYNC    | false |
    When user "Alice" restores version index "1" of file "textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "textfile.txt" for user "Alice" should be "version 2"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Alice" restores version index "1" of file "textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "textfile.txt" for user "Alice" should be "version 3"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
