Feature: checking file versions using file id
  As a user
  I want the versions of files to be available
  So that I can manage the changes made to the files

  ┌───────────────────────────────┬──────┬────────┬─────────┐
  │ role                          │ view │ upload │ restore │
  ├───────────────────────────────┼──────┼────────┼─────────┤
  │ Viewer                        │ no   │ no     │ no      │
  │ Viewer With Versions          │ yes  │ no     │ no      │
  │ File Editor                   │ no   │ yes    │ no      │
  │ File Editor With Versions     │ yes  │ yes    │ yes     │
  │ Editor                        │ no   │ yes    │ no      │
  │ Editor With Versions          │ yes  │ yes    │ yes     │
  │ Space Viewer                  │ no   │ no     │ no      │
  │ Space Viewer With Versions    │ yes  │ no     │ no      │
  │ Space Editor Without Versions │ no   │ yes    │ no      │
  │ Space Editor                  │ yes  │ yes    │ yes     │
  │ Manager                       │ yes  │ yes    │ yes     │
  └───────────────────────────────┴──────┴────────┴─────────┘

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project1" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "Project1" with content "hello world version 1" to "text.txt"
    And we save it into "FILEID"
    And user "Alice" has uploaded a file inside space "Project1" with content "hello world version 1.1" to "text.txt"


  Scenario Outline: check the file versions of a file shared from project space
    Given user "Alice" has sent the following resource share invitation:
      | resource        | text.txt |
      | space           | Project1 |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | <role>   |
    And user "Brian" has a share "text.txt" synced
    When user "Alice" gets the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" tries to get the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "403"
    Examples:
      | role        |
      | File Editor |
      | Viewer      |


  Scenario Outline: check the versions of a file in a shared space as editor/manager
    Given user "Alice" has sent the following space share invitation:
      | space           | Project1     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" gets the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" gets the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    Examples:
      | space-role   |
      | Space Editor |
      | Manager      |


  Scenario: check the versions of a file in a shared space as viewer
    Given user "Alice" has sent the following space share invitation:
      | space           | Project1     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" tries to get the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "403"

  @env-config
  Scenario: sharee can view file versions in a shared space as space viewer with versions role
    Given the administrator has enabled the permissions role "Space Viewer With Versions"
    And user "Alice" has sent the following space share invitation:
      | space           | Project1                   |
      | sharee          | Brian                      |
      | shareType       | user                       |
      | permissionsRole | Space Viewer With Versions |
    When user "Brian" gets the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" restores version index "1" of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "403"

  @env-config
  Scenario Outline: sharee can view file versions of a file inside a shared folder while upload and restore depend on the role
    Given the administrator has enabled the permissions role "<role>"
    And user "Alice" has created a folder "folderToShare" in space "Project1"
    And user "Alice" has uploaded a file inside space "Project1" with content "folder file version 1" to "folderToShare/insideText.txt"
    And user "Alice" has uploaded a file inside space "Project1" with content "folder file version 1.1" to "folderToShare/insideText.txt"
    And we save it into "FOLDERFILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Project1      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <role>        |
    And user "Brian" has a share "folderToShare" synced
    When user "Brian" gets the number of versions of file "/Shares/folderToShare/insideText.txt" using file-id "<<FOLDERFILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" uploads file with content "folder file new version" to "/Shares/folderToShare/insideText.txt" using the WebDAV API
    Then the HTTP status code should be "<upload-code>"
    When user "Brian" restores version index "1" of file "/Shares/folderToShare/insideText.txt" using file-id "<<FOLDERFILEID>>"
    Then the HTTP status code should be "<restore-code>"
    Examples:
      | role                 | upload-code | restore-code |
      | Viewer With Versions | 403         | 403          |
      | Editor With Versions | 204         | 204          |

  @issue-7738
  Scenario Outline: check the versions of a file after moving to a shared folder inside a project space as editor/viewer
    Given user "Alice" has created a folder "testFolder" in space "Project1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | text.txt |
      | space           | Project1 |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | <role>   |
    And user "Brian" has a share "text.txt" synced
    And user "Alice" has moved file "text.txt" to "/testFolder/movedText.txt" in space "Project1"
    When user "Alice" gets the number of versions of file "/testFolder/movedText.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" tries to get the number of versions of file "/Shares/testFolder/movedText.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "403"
    Examples:
      | role        |
      | File Editor |
      | Viewer      |

  @issue-7738
  Scenario: check the versions of a file after moving it to a shared folder inside a project space as manager
    Given user "Alice" has created a folder "testFolder" in space "Project1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder |
      | space           | Project1   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "testFolder" synced
    And user "Alice" has sent the following space share invitation:
      | space           | Project1 |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Manager  |
    And user "Alice" has moved file "text.txt" to "/testFolder/movedText.txt" in space "Project1"
    When user "Brian" gets the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"

  @issue-777
  Scenario Outline: check file versions after moving to-and-from folder in personal space
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "some data" to "<source>textfile.txt"
    And user "Alice" has uploaded file with content "some data - edited" to "<source>textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has moved file "<source>textfile.txt" to "<destination>textfile.txt" in space "Personal"
    When user "Alice" gets the number of versions of file "<destination>textfile.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    Examples:
      | source  | destination |
      | /       | folder/     |
      | folder/ | /           |

  @issue-777
  Scenario Outline: check file versions after moving to-and-from folder in personal space (MOVE using file-id)
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "some data" to "<source>textfile.txt"
    And user "Alice" has uploaded file with content "some data - edited" to "<source>textfile.txt"
    And we save it into "FILEID"
    And using spaces DAV path
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "<destination>" inside space "Personal"
    Then the HTTP status code should be "201"
    When user "Alice" gets the number of versions of file "<destination>textfile.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    Examples:
      | source  | destination |
      | /       | folder/     |
      | folder/ | /           |

  @issue-777
  Scenario Outline: check file versions after moving to-and-from folder in project space
    Given user "Alice" has created a folder "folder" in space "Project1"
    And user "Alice" has uploaded a file inside space "Project1" with content "some data" to "<source>textfile.txt"
    And user "Alice" has uploaded a file inside space "Project1" with content "some data - edited" to "<source>textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has moved file "<source>textfile.txt" to "<destination>textfile.txt" in space "Project1"
    When user "Alice" gets the number of versions of file "<source>textfile.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    Examples:
      | source  | destination |
      | /       | folder/     |
      | folder/ | /           |

  @issue-777
  Scenario Outline: check file versions after moving to-and-from folder in project space (MOVE using file-id)
    And user "Alice" has created a folder "folder" in space "Project1"
    And user "Alice" has uploaded a file inside space "Project1" with content "some data" to "<source>textfile.txt"
    And user "Alice" has uploaded a file inside space "Project1" with content "some data - edited" to "<source>textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "<destination>" inside space "Project1"
    Then the HTTP status code should be "201"
    When user "Alice" gets the number of versions of file "<destination>textfile.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    Examples:
      | source  | destination |
      | /       | folder/     |
      | folder/ | /           |

  @env-config
  Scenario: check the versions of a file in a shared space as editor without versions role
    Given the administrator has enabled the permissions role "Space Editor Without Versions"
    And user "Alice" has sent the following space share invitation:
      | space           | Project1                      |
      | sharee          | Brian                         |
      | shareType       | user                          |
      | permissionsRole | Space Editor Without Versions |
    When user "Brian" gets the number of versions of file "/text.txt" using file-id "<<FILEID>>"
    Then the HTTP status code should be "403"


  Scenario Outline: public tries to check the versions of a file in a project space shared via link as viewer/editor
    Given user "Alice" has created the following space link share:
      | space           | Project1           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When the public tries to get the number of versions of file "/text.txt" with password "%public%" using file-id "<<FILEID>>"
    Then the HTTP status code should be "401"
    Examples:
      | permissions-role |
      | view             |
      | edit             |


  Scenario Outline: public tries to check the versions of a file in a personal space shared via link as viewer/editor
    Given user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/parent.txt"
    And user "Alice" has uploaded file with content "some updated data" to "/PARENT/parent.txt"
    And we save it into "FILEID"
    And user "Alice" has created the following resource link share:
      | resource           | PARENT             |
      | space              | Personal           |
      | permissionsRole    | <permissions-role> |
      | password           | %public%           |
    When the public tries to get the number of versions of file "/parent.txt" with password "%public%" using file-id "<<FILEID>>"
    Then the HTTP status code should be "401"
    Examples:
      | permissions-role |
      | view             |
      | edit             |
