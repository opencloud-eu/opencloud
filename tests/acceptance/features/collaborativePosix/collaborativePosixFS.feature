@env-config @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
Feature: create a resources using collaborative posixfs

  Background:
    Given the config "STORAGE_USERS_POSIX_WATCH_FS" has been set to "true"
    And user "Alice" has been created with default attributes


  Scenario: create folder
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator creates the folder "myFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    When the administrator lists the content of the POSIX storage folder of user "Alice"
    Then the command output should contain "myFolder"
    And as "Alice" folder "/myFolder" should exist


  Scenario: create file
    Given user "Alice" has created folder "/folder"
    When the administrator creates the file "test.txt" with content "content" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    When the administrator lists the content of the POSIX storage folder of user "Alice"
    Then the command output should contain "test.txt"
    And the content of file "/test.txt" for user "Alice" should be "content"


  Scenario: edit file
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator puts the content "new" into the file "test.txt" in the POSIX storage folder of user "Alice"
    Then the content of file "/test.txt" for user "Alice" should be "contentnew"


  Scenario: read file content
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator reads the content of the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command output should contain "content"


  Scenario: copy file to folder
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator copies the file "test.txt" to the folder "folder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/folder/test.txt" for user "Alice" should be "content"


  Scenario: move file to folder
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator moves the file "test.txt" to the folder "folder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/folder/test.txt" for user "Alice" should be "content"
    And as "Alice" file "/test.txt" should not exist


  Scenario: delete file
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator deletes the file "test.txt" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" file "/test.txt" should not exist


  Scenario: delete folder
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "content" to "/folder/test.txt"
    When the administrator deletes the folder "folder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" folder "folder" should not exist


  Scenario: copy file from personal to project space
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project space" with the default quota using the Graph API
    When the administrator copies the file "test.txt" to the space "Project space" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And using spaces DAV path
    And for user "Alice" the space "Project space" should contain these entries:
      | test.txt |


  Scenario: delete project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project space" with the default quota using the Graph API
    When the administrator deletes the project space "Project space" on the POSIX filesystem
    Then the command should be successful
    And the user "Alice" should not have a space called "Project space"
