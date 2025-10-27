Feature: move (rename) file
  As a user
  I want to be able to move and rename files
  So that I can manage my file system

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario Outline: moving a file within same space project with role Manager and editor
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created a folder "newfolder" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "insideSpace.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" moves file "insideSpace.txt" to "newfolder/insideSpace.txt" in space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "newfolder" of the space "Project" should contain these entries:
      | insideSpace.txt |
    But for user "Alice" the space "Project" should not contain these entries:
      | insideSpace.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |


  Scenario: moving a file within same space project with role viewer
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created a folder "newfolder" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "insideSpace.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" moves file "insideSpace.txt" to "newfolder/insideSpace.txt" in space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" folder "newfolder" of the space "Project" should not contain these entries:
      | insideSpace.txt |
    But for user "Alice" the space "Project" should contain these entries:
      | insideSpace.txt |

  @owncloud-issue-1976
  Scenario Outline: try to move a file within a project space into a folder with same name
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "insideSpace.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" moves file "insideSpace.txt" from space "Project" to "insideSpace.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Brian" file "insideSpace.txt" should not exist in the trashbin of the space "Project"
    And for user "Alice" the content of the file "insideSpace.txt" of the space "Project" should be "some content"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |

  @owncloud-issue-8116
  Scenario Outline: user moves a file from a space project with different a role to a space project with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project1" with the default quota using the Graph API
    And user "Brian" has created a space "Project2" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project1" with content "Project1 content" to "project1.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project1        |
      | sharee          | Alice           |
      | shareType       | user            |
      | permissionsRole | <to-space-role> |
    And user "Brian" has sent the following space share invitation:
      | space           | Project2          |
      | sharee          | Alice             |
      | shareType       | user              |
      | permissionsRole | <from-space-role> |
    When user "Alice" moves file "project1.txt" from space "Project1" to "project1.txt" inside space "Project2" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "Alice" the space "Project1" should contain these entries:
      | project1.txt |
    And for user "Alice" the space "Project2" should not contain these entries:
      | project1.txt |
    Examples:
      | from-space-role | to-space-role | http-status-code |
      | Manager         | Manager       | 502              |
      | Space Editor    | Manager       | 502              |
      | Manager         | Space Editor  | 502              |
      | Space Editor    | Space Editor  | 502              |
      | Manager         | Space Viewer  | 403              |
      | Space Editor    | Space Viewer  | 403              |
      | Space Viewer    | Manager       | 403              |
      | Space Viewer    | Space Editor  | 403              |
      | Space Viewer    | Space Viewer  | 403              |

  @owncloud-issue-7618
  Scenario Outline: user moves a file from a space project with different role to a space personal
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "project.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" moves file "project.txt" from space "Project" to "project.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "Alice" the space "Project" should contain these entries:
      | project.txt |
    And for user "Alice" the space "Personal" should not contain these entries:
      | project.txt |
    Examples:
      | space-role   | http-status-code |
      | Manager      | 502              |
      | Space Editor | 502              |
      | Space Viewer | 403              |


  Scenario Outline: user moves a file from space project with different role to space Shares with different role (permission)
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "project.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" moves file "project.txt" from space "Project" to "/testshare/project.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Project" should contain these entries:
      | project.txt |
    But for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | project.txt |
    Examples:
      | space-role   | permissions-role |
      | Manager      | Editor           |
      | Space Editor | Editor           |
      | Space Viewer | Editor           |
      | Manager      | Uploader         |
      | Space Editor | Uploader         |
      | Space Viewer | Uploader         |
      | Manager      | Viewer           |
      | Space Editor | Viewer           |
      | Space Viewer | Viewer           |

  @owncloud-issue-7618
  Scenario Outline: user moves a file from space personal to space project with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Alice" has uploaded file with content "personal space content" to "/personal.txt"
    When user "Alice" moves file "personal.txt" from space "Personal" to "personal.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "Alice" the space "Personal" should contain these entries:
      | personal.txt |
    And for user "Alice" the space "Project" should not contain these entries:
      | personal.txt |
    Examples:
      | space-role   | http-status-code |
      | Manager      | 502              |
      | Space Editor | 502              |
      | Space Viewer | 403              |


  Scenario Outline: user moves a file from space personal to space Shares with different role (permission)
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    And user "Alice" has uploaded file with content "personal content" to "personal.txt"
    When user "Alice" moves file "personal.txt" from space "Personal" to "/testshare/personal.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Personal" should contain these entries:
      | personal.txt |
    But for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | personal.txt |
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |
      | Viewer           |


  Scenario Outline: user moves a file from space Shares with different role (permissions) to space personal
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" moves file "/testshare/testshare.txt" from space "Shares" to "testshare.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Personal" should not contain these entries:
      | testshare.txt |
    And for user "Alice" folder "testshare" of the space "Shares" should contain these entries:
      | testshare.txt |
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |
      | Viewer           |


  Scenario Outline: user moves a file from space Shares with different role (permissions) to space project with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" moves file "/testshare/testshare.txt" from space "Shares" to "testshare.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Project" should not contain these entries:
      | /testshare.txt |
    And for user "Alice" folder "testshare" of the space "Shares" should contain these entries:
      | testshare.txt |
    Examples:
      | space-role   | permissions-role |
      | Manager      | Editor           |
      | Space Editor | Editor           |
      | Space Viewer | Editor           |
      | Manager      | Uploader         |
      | Space Editor | Uploader         |
      | Space Viewer | Uploader         |
      | Manager      | Viewer           |
      | Space Editor | Viewer           |
      | Space Viewer | Viewer           |


  Scenario Outline: user moves a file from space Shares to another space Shares with different role (permissions)
    Given user "Brian" has created folder "/testshare1"
    And user "Brian" has created folder "/testshare2"
    And user "Brian" has uploaded file with content "testshare1 content" to "/testshare1/testshare1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare1              |
      | space           | Personal                |
      | sharee          | Alice                   |
      | shareType       | user                    |
      | permissionsRole | <from-permissions-role> |
    And user "Alice" has a share "testshare1" synced
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare2            |
      | space           | Personal              |
      | sharee          | Alice                 |
      | shareType       | user                  |
      | permissionsRole | <to-permissions-role> |
    And user "Alice" has a share "testshare2" synced
    When user "Alice" moves file "/testshare1/testshare1.txt" from space "Shares" to "/testshare2/testshare1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" folder "testshare1" of the space "Shares" should contain these entries:
      | testshare1.txt |
    But for user "Alice" folder "testshare2" of the space "Shares" should not contain these entries:
      | testshare1.txt |
    Examples:
      | from-permissions-role | to-permissions-role |
      | Editor                | Editor              |
      | Editor                | Uploader            |
      | Editor                | Viewer              |
      | Uploader              | Editor              |
      | Uploader              | Uploader            |
      | Uploader              | Viewer              |
      | Viewer                | Editor              |
      | Viewer                | Uploader            |
      | Viewer                | Viewer              |


  Scenario Outline: moving a file out of a shared folder as a sharer
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "test data" to "/testshare/testfile.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Brian" moves file "/testshare/testfile.txt" from space "Personal" to "/testfile.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/testfile.txt" for user "Brian" should be "test data"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | testfile.txt |
    And for user "Brian" folder "testshare" of the space "Personal" should not contain these entries:
      | testfile.txt |
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |
      | Viewer           |


  Scenario Outline: moving a folder out of a shared folder as a sharer
    Given user "Brian" has created the following folders
      | path                     |
      | /testshare               |
      | /testshare/testsubfolder |
    And user "Brian" has uploaded file with content "test data" to "/testshare/testsubfolder/testfile.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Brian" moves folder "/testshare/testsubfolder" from space "Personal" to "/testsubfolder" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/testsubfolder/testfile.txt" for user "Brian" should be "test data"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | testsubfolder |
    And for user "Brian" folder "testshare" of the space "Personal" should not contain these entries:
      | testsubfolder |
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |
      | Viewer           |


  Scenario Outline: sharee moves a file within a Shares space (Editor/Uploader permissions)
    Given user "Brian" has created folder "testshare"
    And user "Brian" has created folder "testshare/child"
    And user "Brian" has uploaded file with content "test file content" to "/testshare/testfile.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" moves file "testshare/testfile.txt" from space "Shares" to "testshare/child/testfile.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the content of the file "testshare/child/testfile.txt" of the space "Shares" should be "test file content"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | testfile.txt |
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |


  Scenario: sharee moves a file within a Shares space (viewer permissions)
    Given user "Brian" has created folder "testshare"
    And user "Brian" has created folder "testshare/child"
    And user "Brian" has uploaded file with content "test file content" to "/testshare/testfile.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    When user "Alice" moves file "testshare/testfile.txt" from space "Shares" to "testshare/child/testfile.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" folder "testshare/child" of the space "Shares" should not contain these entries:
      | testfile.txt |
    But for user "Alice" folder "testshare" of the space "Shares" should contain these entries:
      | testfile.txt |

  @owncloud-issue-1976
  Scenario Outline: sharee tries to move a file into same shared folder with same name
    Given user "Brian" has created folder "testshare"
    And user "Brian" has uploaded file with content "test file content" to "testshare/testfile.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" moves file "testshare/testfile.txt" from space "Shares" to "testshare/testfile.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Brian" file "testfile.txt" should not exist in the trashbin of the space "Personal"
    And for user "Alice" the content of the file "testshare/testfile.txt" of the space "Shares" should be "test file content"
    And for user "Brian" the content of the file "testshare/testfile.txt" of the space "Personal" should be "test file content"
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |
      | Viewer           |


  Scenario: overwrite a file while moving in project space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created a folder "folder" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "root file v1" to "testfile.txt"
    And user "Brian" has uploaded a file inside space "Project" with content "root file v2" to "testfile.txt"
    And user "Brian" has uploaded a file inside space "Project" with content "same name file" to "folder/testfile.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    When user "Alice" overwrites file "testfile.txt" from space "Project" to "folder/testfile.txt" inside space "Project" while moving using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" the content of the file "folder/testfile.txt" of the space "Project" should be "root file v2"
    And for user "Alice" the space "Project" should not contain these entries:
      | testfile.txt |
    When user "Brian" downloads version of the file "folder/testfile.txt" with the index "1" of the space "Project" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "root file v1"
