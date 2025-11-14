@env-config
Feature: remove file versions via CLI command

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: remove all file versions
    Given user "Alice" has uploaded file with content "This is version 1" to "textfile.txt"
    And user "Alice" has uploaded file with content "This is version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "This is version 3" to "textfile.txt"
    When the administrator removes all the file versions using the CLI
    Then the command should be successful
    And the command output should contain "✅ Deleted 2 revisions (6 files / 2 blobs)"
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"


  Scenario: remove all versions of file using file-id
    Given user "Alice" has uploaded file with content "This is version 1" to "randomFile.txt"
    And user "Alice" has uploaded file with content "This is version 2" to "randomFile.txt"
    And user "Alice" has uploaded file with content "This is version 3" to "randomFile.txt"
    And user "Alice" has uploaded file with content "This is version 1" to "anotherFile.txt"
    And user "Alice" has uploaded file with content "This is version 2" to "anotherFile.txt"
    And user "Alice" has uploaded file with content "This is version 3" to "anotherFile.txt"
    When the administrator removes the versions of file "randomFile.txt" of user "Alice" from space "Personal" using the CLI
    Then the command should be successful
    And the command output should contain "✅ Deleted 2 revisions (6 files / 2 blobs)"
    When user "Alice" gets the number of versions of file "randomFile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
    When user "Alice" gets the number of versions of file "anotherFile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: remove all versions of files from a space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded file with content "This is version 1" to "file.txt"
    And user "Alice" has uploaded file with content "This is version 2" to "file.txt"
    And user "Alice" has uploaded file with content "This is version 3" to "file.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "This is version 1" to "lorem.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "This is version 2" to "lorem.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "This is version 3" to "lorem.txt"
    And we save it into "LOREM_FILEID"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "This is version 1" to "epsum.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "This is version 2" to "epsum.txt"
    And user "Alice" has uploaded a file inside space "projectSpace" with content "This is version 3" to "epsum.txt"
    And we save it into "EPSUM_FILEID"
    When the administrator removes the file versions of space "projectSpace" using the CLI
    Then the command should be successful
    And the command output should contain "✅ Deleted 4 revisions (12 files / 4 blobs)"
    When user "Alice" gets the number of versions of file "file.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"
    When user "Alice" gets the number of versions of file "lorem.txt" using file-id "<<LOREM_FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
    When user "Alice" gets the number of versions of file "epsum.txt" using file-id "<<EPSUM_FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"
