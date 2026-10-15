@env-config @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
Feature: clean up old file versions that changes on the POSIX filesystem created
  As an administrator
  I want versions created by changes on the POSIX filesystem to be cleaned up like any other version
  So that files that are changed directly on disk don't keep all their old versions

  # With STORAGE_USERS_POSIX_ENABLE_FS_REVISIONS, a change on disk turns the previous content into a
  # version. That needs a copy of the previous content, which posix only keeps of content it wrote
  # or picked up while the setting was on, so it is set before the files are uploaded.

  Background:
    Given the following configs have been set:
      | config                                  | value |
      | STORAGE_USERS_POSIX_WATCH_FS            | true  |
      | STORAGE_USERS_POSIX_ENABLE_FS_REVISIONS | true  |
    And user "Alice" has been created with default attributes

  # The strategy is only set after the uploads, so that the file has 3 versions when it is changed
  # on disk: only the cleanup that the change on disk triggers can bring it down to 1.
  Scenario Outline: the configured strategy cleans up a file whenever a change on disk creates a version
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 3" to "textfile.txt"
    And user "Alice" has uploaded file with content "version 4" to "textfile.txt"
    And the following configs have been set:
      | config                        | value   |
      | OC_REVISIONS_CLEANUP_STRATEGY | 1       |
      | OC_REVISIONS_CLEANUP_ASYNC    | <async> |
    When the administrator puts the content " changed on disk" into the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command should be successful
    And as "Alice" the final content of file "textfile.txt" should be "version 4 changed on disk"
    And the number of versions of file "textfile.txt" for user "Alice" should eventually be "1"
    And the content of version index "1" of file "textfile.txt" for user "Alice" should be "version 4"
    Examples:
      | async |
      | true  |
      | false |


  Scenario: the cleanup command deletes versions that changes on disk created
    Given user "Alice" has uploaded file with content "version 1" to "textfile.txt"
    When the administrator puts the content ", changed on disk" into the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command should be successful
    And as "Alice" the final content of file "textfile.txt" should be "version 1, changed on disk"
    When the administrator puts the content ", changed again" into the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command should be successful
    And as "Alice" the final content of file "textfile.txt" should be "version 1, changed on disk, changed again"
    And the number of versions of file "textfile.txt" for user "Alice" should eventually be "2"
    When the administrator cleans up the file versions of all spaces using the CLI with flags "-s 0"
    Then the command should be successful
    And the command output should contain "/textfile.txt: deleted 2 of 2 revisions"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
    And the content of file "textfile.txt" for user "Alice" should be "version 1, changed on disk, changed again"
