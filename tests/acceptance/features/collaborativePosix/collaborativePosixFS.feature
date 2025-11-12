@env-config @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
Feature: create a resources using collaborative posixfs

  Background:
    Given the config "STORAGE_USERS_POSIX_WATCH_FS" has been set to "true"
    And user "Alice" has been created with default attributes
    And user "Alice" has created folder "/firstFolder"


  Scenario: administrator lists the content of the POSIX storage
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator lists the content of the POSIX storage folder of user "Alice"
    Then the command output should contain "firstFolder"
    And the command output should contain "test.txt"


  Scenario: create folder
    When the administrator creates the folder "myFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" folder "/myFolder" should exist


  Scenario: create nested folder
    When the administrator creates the folder "deep/nested/structure/myFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" folder "/deep/nested/structure/myFolder" should exist


  Scenario: create file
    When the administrator creates the file "test.txt" with content "content" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/test.txt" for user "Alice" should be "content"


  Scenario: create large file
    When the administrator creates the file "largefile.txt" with size "1gb" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" file "/largefile.txt" should exist


  Scenario: creates files sequentially in a folder
    When the administrator creates 50 files sequentially in the directory "firstFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/firstFolder/file_1.txt" for user "Alice" should be "file 1 content"
    And the content of file "/firstFolder/file_50.txt" for user "Alice" should be "file 50 content"


  Scenario: creates files in parallel in a folder
    When the administrator creates 100 files in parallel in the directory "firstFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/firstFolder/parallel_1.txt" for user "Alice" should be "parallel file 1 content"
    And the content of file "/firstFolder/parallel_100.txt" for user "Alice" should be "parallel file 100 content"


  Scenario: edit file
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator puts the content "new" into the file "test.txt" in the POSIX storage folder of user "Alice"
    Then the content of file "/test.txt" for user "Alice" should be "contentnew"


  Scenario: read file content
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator reads the content of the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command output should contain "content"


  Scenario: copy file to folder
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator copies the file "test.txt" to the folder "firstFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/firstFolder/test.txt" for user "Alice" should be "content"


  Scenario: rename file
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator renames the file "test.txt" to "new-name.txt" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/new-name.txt" for user "Alice" should be "content"


  Scenario: rename a created file
    Given the administrator has created the file "test.txt" with content "content" for user "Alice" on the POSIX filesystem
    When the administrator renames the file "test.txt" to "test.md" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/test.md" for user "Alice" should be "content"


  Scenario: move file to folder
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator moves the file "test.txt" to the folder "firstFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And the content of file "/firstFolder/test.txt" for user "Alice" should be "content"
    And as "Alice" file "/test.txt" should not exist


  Scenario: delete file
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    When the administrator deletes the file "test.txt" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" file "/test.txt" should not exist


  Scenario: delete folder
    And user "Alice" has uploaded file with content "content" to "/firstFolder/test.txt"
    When the administrator deletes the folder "firstFolder" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And as "Alice" folder "firstFolder" should not exist


  Scenario: copy file from personal to project space
    Given user "Alice" has uploaded file with content "content" to "test.txt"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project space" with the default quota using the Graph API
    When the administrator copies the file "test.txt" to the space "Project space" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And using spaces DAV path
    And for user "Alice" the space "Project space" should contain these entries:
      | test.txt |


  Scenario: user doesn't lose file versions after renaming the file
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "content" to "textfile.txt"
    And user "Alice" has uploaded file with content "new content version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "new content version 3" to "textfile.txt"
    When the administrator renames the file "textfile.txt" to "new-name.txt" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    When user "Alice" gets the number of versions of file "new-name.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: user doesn't lose file versions after changing the file content
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    And user "Alice" has uploaded file with content "new content version 2" to "textfile.txt"
    And user "Alice" has uploaded file with content "new content version 3" to "textfile.txt"
    When the administrator puts the content "new" into the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command should be successful
    When user "Alice" gets the number of versions of file "textfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"


  Scenario: user doesn't lose share and public link after renaming the file
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "content" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "textfile.txt" synced
    When the administrator renames the file "textfile.txt" to "new-name.txt" for user "Alice" on the POSIX filesystem
    Then the command should be successful
    And user "Brian" should have a share "textfile.txt" shared by user "Alice" from space "Personal"
    And the public should be able to download file "textfile.txt" from the last link share with password "%public%" and the content should be "content"


  Scenario: user doesn't lose share and public link after changing the file content
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "content" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When the administrator puts the content "new" into the file "textfile.txt" in the POSIX storage folder of user "Alice"
    Then the command should be successful
    And for user "Brian" the content of the file "textfile.txt" of the space "Shares" should be "contentnew"
    And the public should be able to download file "textfile.txt" from the last link share with password "%public%" and the content should be "contentnew"

  @issue-1100
  Scenario: upload and rename file
    When the administrator creates the file "test.txt" with content "content" for user "Alice" on the POSIX filesystem
    And the administrator renames the file "test.txt" to "renamed.txt" for user "Alice" on the POSIX filesystem
    And the administrator checks the attribute "user.oc.name" of file "renamed.txt" for user "Alice" on the POSIX filesystem
    Then the command output should contain "renamed.txt"
    And the content of file "/renamed.txt" for user "Alice" should be "content"
