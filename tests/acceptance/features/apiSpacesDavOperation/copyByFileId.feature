Feature: copying file using file id
  As a user
  I want to copy the file using file id
  So that I can manage my resource

  Background:
    Given using spaces DAV path
    And user "Alice" has been created with default attributes


  Scenario: copy a file into a folder in personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | textfile.txt |


  Scenario: copy a file into a sub-folder in personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "textfile.txt" into folder "/folder/sub-folder" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "Personal" should contain these files:
      | textfile.txt |


  Scenario: copy a file from a folder into root of personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | textfile.txt |


  Scenario: copy a file from sub-folder into root of personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "folder/sub-folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "/textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "Personal" should contain these files:
      | textfile.txt |


  Scenario: copy a file into a folder in project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "/folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder" of the space "project-space" should contain these files:
      | textfile.txt |


  Scenario: copy a file into a sub-folder in project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder/sub-folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder/sub-folder" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "project-space" should contain these files:
      | textfile.txt |


  Scenario: copy a file from a folder into root of project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder" of the space "project-space" should contain these files:
      | textfile.txt |


  Scenario: copy a file from sub-folder into root of project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder/sub-folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "folder/sub-folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "project-space" should contain these files:
      | textfile.txt |


  Scenario: copy a file from personal to project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" copies file with id "<<FILEID>>" as "/textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |


  Scenario: copy a file from sub-folder to root folder inside Shares space
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/sub-folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    When user "Brian" copies file with id "<<FILEID>>" as "test.txt" into folder "Shares/folder" inside space "Shares"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "folder/sub-folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | test.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "Personal" should contain these files:
      | test.txt |


  Scenario: copy a file from personal to share space
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has uploaded file with content "some data" to "/test.txt"
    And we save it into "FILEID"
    And user "Brian" has a share "folder" synced
    When user "Brian" copies file with id "<<FILEID>>" as "/test.txt" into folder "Shares/folder" inside space "Shares"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "/" of the space "Personal" should contain these files:
      | test.txt |
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | test.txt |


  Scenario Outline: copy a file from share to personal space
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder            |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <permission-role> |
    When user "Brian" copies file with id "<<FILEID>>" as "/test.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "/" of the space "Personal" should contain these files:
      | test.txt |
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | test.txt |
    Examples:
      | permission-role |
      | Editor          |
      | Viewer          |
      | Uploader        |


  Scenario: sharee tries to copy a file from shares space with secure viewer to personal space
    Given user "Brian" has been created with default attributes
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder        |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    When user "Brian" copies file with id "<<FILEID>>" as "/test.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "403"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "/" of the space "Personal" should not contain these files:
      | test.txt |


  Scenario Outline: sharee copies a file from shares to project space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder            |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <permission-role> |
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" copies file with id "<<FILEID>>" as "test.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "/" of the space "project-space" should contain these files:
      | test.txt |
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | test.txt |
    Examples:
      | permission-role | space-role    |
      | Viewer          | Manager       |
      | Viewer          | Space Editor  |
      | Editor          | Manager       |
      | Editor          | Space Editor  |
      | Uploader        | Manager       |
      | Uploader        | Space Editor  |

  @env-config
  Scenario Outline: sharee tries to copy a file from shares to project space
    Given user "Brian" has been created with default attributes
    And the administrator has enabled the permissions role "Secure Viewer"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder            |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <permission-role> |
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" copies file with id "<<FILEID>>" as "/test.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "403"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "/" of the space "project-space" should not contain these files:
      | test.txt |
    And for user "Alice" folder "/" of the space "project-space" should not contain these files:
      | test.txt |
    Examples:
      | permission-role | space-role    |
      | Secure Viewer   | Manager       |
      | Secure Viewer   | Space Viewer  |
      | Secure Viewer   | Space Editor  |
      | Editor          | Space Viewer  |
      | Viewer          | Space Viewer  |
      | Uploader        | Space Viewer  |


  Scenario Outline: sharee copies a file between shares spaces
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/share1"
    And user "Alice" has created folder "/share2"
    And user "Alice" has uploaded file with content "some data" to "/share1/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | share1            |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <from-share-role> |
    And user "Alice" has sent the following resource share invitation:
      | resource        | share2          |
      | space           | Personal        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | <to-share-role> |
    When user "Brian" copies file with id "<<FILEID>>" as "/test.txt" into folder "share2" inside space "Shares"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "share1" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "share2" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Alice" folder "share1" of the space "Personal" should contain these files:
      | test.txt |
    And for user "Alice" folder "share2" of the space "Personal" should contain these files:
      | test.txt |
    Examples:
      | from-share-role | to-share-role |
      | Viewer          | Editor        |
      | Viewer          | Uploader      |
      | Editor          | Editor        |
      | Editor          | Uploader      |
      | Uploader        | Editor        |
      | Uploader        | Uploader      |

  @env-config
  Scenario Outline: sharee tries to copy a file between shares space
    Given user "Brian" has been created with default attributes
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has created folder "/share1"
    And user "Alice" has created folder "/share2"
    And user "Alice" has uploaded file with content "some data" to "/share1/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | share1            |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <from-share-role> |
    And user "Alice" has sent the following resource share invitation:
      | resource        | share2          |
      | space           | Personal        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | <to-share-role> |
    When user "Brian" copies file with id "<<FILEID>>" as "test.txt" into folder "share2" inside space "Shares"
    Then the HTTP status code should be "403"
    And for user "Brian" folder "share1" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "share2" of the space "Shares" should not contain these files:
      | test.txt |
    And for user "Alice" folder "share1" of the space "Personal" should contain these files:
      | test.txt |
    And for user "Alice" folder "share2" of the space "Personal" should not contain these files:
      | test.txt |
    Examples:
      | from-share-role | to-share-role |
      | Secure Viewer   | Viewer        |
      | Secure Viewer   | Editor        |
      | Secure Viewer   | Uploader      |
      | Secure Viewer   | Secure Viewer |
      | Viewer          | Viewer        |
      | Editor          | Viewer        |
      | Uploader        | Viewer        |
      | Viewer          | Secure Viewer |
      | Editor          | Secure Viewer |
      | Uploader        | Secure Viewer |


  Scenario Outline: copy a file from project to personal space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" copies file with id "<<FILEID>>" as "/textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Brian" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: copy a file between two project spaces
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "first-project-space" with the default quota using the Graph API
    And user "Alice" has created a space "second-project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "first-project-space" with content "first project space" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | first-project-space |
      | sharee          | Brian               |
      | shareType       | user                |
      | permissionsRole | <from-space-role>   |
    And user "Alice" has sent the following space share invitation:
      | space           | second-project-space |
      | sharee          | Brian                |
      | shareType       | user                 |
      | permissionsRole | <to-space-role>      |
    When user "Brian" copies file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "second-project-space"
    Then the HTTP status code should be "201"
    And for user "Brian" the space "second-project-space" should contain these entries:
      | textfile.txt |
    And for user "Brian" the space "first-project-space" should contain these entries:
      | textfile.txt |
    And for user "Alice" the space "second-project-space" should contain these entries:
      | textfile.txt |
    Examples:
      | from-space-role | to-space-role |
      | Manager         | Manager       |
      | Manager         | Space Editor  |
      | Space Editor    | Manager       |
      | Space Editor    | Space Editor  |
      | Space Viewer    | Manager       |
      | Space Viewer    | Space Editor  |


  Scenario Outline: try to copy a file from a project to another project space with read permission
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "first-project-space" with the default quota using the Graph API
    And user "Alice" has created a space "second-project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "first-project-space" with content "first project space" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | first-project-space |
      | sharee          | Brian               |
      | shareType       | user                |
      | permissionsRole | <from-space-role>   |
    And user "Alice" has sent the following space share invitation:
      | space           | second-project-space |
      | sharee          | Brian                |
      | shareType       | user                 |
      | permissionsRole | <to-space-role>      |
    When user "Brian" copies file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "second-project-space"
    Then the HTTP status code should be "403"
    And for user "Brian" the space "second-project-space" should not contain these entries:
      | textfile.txt |
    And for user "Brian" the space "first-project-space" should contain these entries:
      | textfile.txt |
    But for user "Alice" the space "second-project-space" should not contain these entries:
      | textfile.txt |
    Examples:
      | from-space-role | to-space-role |
      | Manager         | Space Viewer  |
      | Space Editor    | Space Viewer  |
      | Space Viewer    | Space Viewer  |


  Scenario Outline: copy a file from project to shares space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    And user "Alice" has created folder "testshare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare     |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <permissions> |
    When user "Brian" copies file with id "<<FILEID>>" as "textfile.txt" into folder "testshare" inside space "Shares"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    And for user "Brian" folder "testshare" of the space "Shares" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "testshare" of the space "Personal" should contain these files:
      | textfile.txt |
    Examples:
      | space-role   | permissions |
      | Manager      | Editor      |
      | Manager      | Uploader    |
      | Space Editor | Editor      |
      | Space Editor | Uploader    |
      | Space Viewer | Editor      |
      | Space Viewer | Uploader    |

  @env-config
  Scenario Outline: try to copy a file from project to shares space with read permission
    Given user "Brian" has been created with default attributes
    And the administrator has enabled the permissions role "Secure Viewer"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    And user "Alice" has created folder "testshare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare     |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <permissions> |
    When user "Brian" copies file with id "<<FILEID>>" as "textfile.txt" into folder "testshare" inside space "Shares"
    Then the HTTP status code should be "403"
    And for user "Brian" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    But for user "Brian" folder "testshare" of the space "Shares" should not contain these files:
      | textfile.txt |
    And for user "Alice" folder "testshare" of the space "Personal" should not contain these files:
      | textfile.txt |
    Examples:
      | space-role   | permissions   |
      | Manager      | Viewer        |
      | Manager      | Secure Viewer |
      | Space Editor | Viewer        |
      | Space Editor | Secure Viewer |
      | Space Viewer | Viewer        |
      | Space Viewer | Secure Viewer |
