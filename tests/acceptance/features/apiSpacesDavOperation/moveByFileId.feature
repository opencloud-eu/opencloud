Feature: moving/renaming file using file id
  As a user
  I want to be able to move or rename files using file id
  So that I can manage my file system

  Background:
    Given using spaces DAV path
    And user "Alice" has been created with default attributes


  Scenario: move a file into a folder inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | textfile.txt |
    But for user "Alice" the space "Personal" should not contain these entries:
      | textfile.txt |


  Scenario: move a file into a sub-folder inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder/sub-folder" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder/sub-folder/" of the space "Personal" should contain these files:
      | textfile.txt |
    But for user "Alice" the space "Personal" should not contain these entries:
      | textfile.txt |


  Scenario: move a file from folder to root inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | textfile.txt |
    But for user "Alice" folder "folder" of the space "Personal" should not contain these files:
      | textfile.txt |


  Scenario: move a file from sub-folder to root inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "folder/sub-folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | textfile.txt |
    But for user "Alice" folder "folder/sub-folder" of the space "Personal" should not contain these files:
      | textfile.txt |

  @owncloud-issue-1976
  Scenario: try to move a file into same folder with same name
    And user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "403"
    And as "Alice" file "textfile.txt" should not exist in the trashbin of the space "Personal"
    And for user "Alice" the content of the file "textfile.txt" of the space "Personal" should be "some data"


  Scenario Outline: move a file from personal to share space
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder        |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <permissions> |
    And user "Brian" has a share "folder" synced
    And user "Brian" has uploaded file with content "some data" to "/test.txt"
    And we save it into "FILEID"
    When user "Brian" moves file with id "<<FILEID>>" as "test.txt" into folder "folder" inside space "Shares"
    Then the HTTP status code should be "502"
    And the value of the item "/d:error/s:message" in the response about user "Brian" should be "cross storage moves are not supported, use copy and delete"
    And for user "Brian" folder "/" of the space "Personal" should contain these files:
      | test.txt |
    But for user "Alice" folder "folder" of the space "Personal" should not contain these files:
      | test.txt |
    Examples:
      | permissions |
      | Editor      |
      | Viewer      |

  @owncloud-issue-7618
  Scenario Outline: move a file from personal to project space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Personal" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    But for user "Brian" folder "/" of the space "project-space" should not contain these files:
      | textfile.txt |
    Examples:
      | space-role   | http-status-code |
      | Manager      | 502              |
      | Space Editor | 502              |
      | Space Viewer | 403              |

  @owncloud-issue-7618
  Scenario: move a file to different name from personal space to project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "Personal" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "renamed.txt" inside space "project-space"
    Then the HTTP status code should be "502"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "move:error: not supported: cannot move across spaces"
    And for user "Alice" folder "/" of the space "Personal" should contain these files:
      | textfile.txt |
    But for user "Alice" folder "/" of the space "project-space" should not contain these files:
      | renamed.txt |


  Scenario Outline: move a file into a folder inside project space (manager/editor)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "/folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" moves file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder" of the space "project-space" should contain these files:
      | textfile.txt |
    But for user "Alice" the space "project-space" should not contain these entries:
      | textfile.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |

  @owncloud-issue-1976
  Scenario Outline: try to move a file within a project space into a folder with same name
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "403"
    And as "Alice" file "textfile.txt" should not exist in the trashbin of the space "project-space"
    And for user "Brian" the content of the file "textfile.txt" of the space "project-space" should be "some data"
    Examples:
      | space-role   |
      | Manager      |
      | Space Viewer |
      | viewer       |


  Scenario: try to move a file into a folder inside project space (viewer)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "/folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Space Viewer  |
    When user "Brian" moves file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder" inside space "project-space"
    Then the HTTP status code should be "403"
    And for user "Alice" the space "project-space" should contain these entries:
      | textfile.txt |
    But for user "Alice" folder "folder" of the space "project-space" should not contain these files:
      | textfile.txt |


  Scenario: move a file into a sub-folder inside project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder/sub-folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "/textfile.txt" into folder "/folder/sub-folder" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder/sub-folder/" of the space "project-space" should contain these files:
      | textfile.txt |
    But for user "Alice" the space "Personal" should not contain these entries:
      | textfile.txt |


  Scenario: move a file from folder to root inside project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "project-space" should contain these entries:
      | textfile.txt |
    But for user "Alice" folder "folder" of the space "project-space" should not contain these files:
      | textfile.txt |


  Scenario: move a file from sub-folder to root inside project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder/sub-folder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "folder/sub-folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "project-space" should contain these entries:
      | textfile.txt |
    But for user "Alice" folder "folder/sub-folder" of the space "project-space" should not contain these files:
      | textfile.txt |

  @owncloud-issue-8116
  Scenario Outline: move a file between two project spaces
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "first-project-space" with the default quota using the Graph API
    And user "Alice" has created a space "second-project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "first-project-space" with content "first project space" to "file.txt"
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
    When user "Brian" moves file with id "<<FILEID>>" as "file.txt" into folder "/" inside space "second-project-space"
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" the space "second-project-space" should not contain these entries:
      | file.txt |
    But for user "Brian" the space "first-project-space" should contain these entries:
      | file.txt |
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

  @owncloud-issue-8116
  Scenario: move a file to different name between project spaces
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "first-project-space" with the default quota using the Graph API
    And user "Alice" has created a space "second-project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "first-project-space" with content "data from first project space" to "firstProjectSpacetextfile.txt"
    And user "Alice" has uploaded a file inside space "second-project-space" with content "data from second project space" to "secondProjectSpacetextfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "/renamedSecondProjectSpacetextfile.txt" inside space "first-project-space"
    Then the HTTP status code should be "502"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "move:error: not supported: cannot move across spaces"
    And for user "Alice" folder "/" of the space "first-project-space" should contain these files:
      | firstProjectSpacetextfile.txt |
    And for user "Alice" folder "/" of the space "second-project-space" should contain these files:
      | secondProjectSpacetextfile.txt |
    But for user "Alice" the space "first-project-space" should not contain these entries:
      | renamedSecondProjectSpacetextfile.txt |


  Scenario Outline: move a file from project to shares space
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
    And user "Brian" has a share "testshare" synced
    When user "Brian" moves file with id "<<FILEID>>" as "textfile.txt" into folder "testshare" inside space "Shares"
    Then the HTTP status code should be "502"
    And for user "Brian" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    But for user "Brian" folder "testshare" of the space "Shares" should not contain these files:
      | textfile.txt |
    Examples:
      | space-role   | permissions |
      | Manager      | Editor      |
      | Space Editor | Editor      |
      | Space Viewer | Editor      |
      | Manager      | Viewer      |
      | Space Editor | Viewer      |
      | Space Viewer | Viewer      |

  @owncloud-issue-7618
  Scenario Outline: move a file from project to personal space
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
    When user "Brian" moves file with id "<<FILEID>>" as "/textfile.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    But for user "Brian" folder "/" of the space "Personal" should not contain these files:
      | textfile.txt |
    Examples:
      | space-role   | http-status-code |
      | Manager      | 502              |
      | Space Editor | 502              |
      | Space Viewer | 403              |

  @owncloud-issue-7618
  Scenario: move a file to different name from project space to personal space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "/renamed.txt" inside space "Personal"
    Then the HTTP status code should be "502"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "move:error: not supported: cannot move across spaces"
    And for user "Alice" folder "/" of the space "project-space" should contain these files:
      | textfile.txt |
    But for user "Alice" folder "/" of the space "Personal" should not contain these files:
      | renamed.txt |

  @owncloud-issue-7617
  Scenario: move a file into a folder within a shared folder (edit permissions)
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "folder" synced
    When user "Brian" moves file with id "<<FILEID>>" as "test.txt" into folder "folder/sub-folder" inside space "Shares"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "folder/sub-folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "Personal" should contain these files:
      | test.txt |
    But for user "Brian" folder "folder" of the space "Shares" should not contain these files:
      | test.txt |
    And for user "Alice" folder "folder" of the space "Personal" should not contain these files:
      | test.txt |

  @owncloud-issue-1976
  Scenario: sharee tries to move a file into same shared folder with same name
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "some data" to "folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "/folder" synced
    When user "Brian" moves file with id "<<FILEID>>" as "test.txt" into folder "folder" inside space "Shares"
    Then the HTTP status code should be "403"
    And as "Alice" file "test.txt" should not exist in the trashbin of the space "Personal"
    And for user "Brian" the content of the file "folder/test.txt" of the space "Shares" should be "some data"
    And for user "Alice" the content of the file "folder/test.txt" of the space "Personal" should be "some data"


  Scenario: try to move a file into a folder within a shared folder (read permissions)
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "folder" synced
    When user "Brian" moves file with id "<<FILEID>>" as "test.txt" into folder "folder/sub-folder" inside space "Shares"
    Then the HTTP status code should be "502"
    And for user "Brian" folder "folder/sub-folder" of the space "Shares" should not contain these files:
      | test.txt |
    And for user "Alice" folder "folder/sub-folder" of the space "Personal" should not contain these files:
      | test.txt |
    But for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | test.txt |


  Scenario Outline: move a file from one shared folder to another shared folder
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "testshare1"
    And user "Alice" has created folder "testshare2"
    And user "Alice" has uploaded file with content "some data" to "testshare1/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare1         |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <from-permissions> |
    And user "Brian" has a share "testshare1" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare2       |
      | space           | Personal         |
      | sharee          | Brian            |
      | shareType       | user             |
      | permissionsRole | <to-permissions> |
    And user "Brian" has a share "testshare2" synced
    When user "Brian" moves file with id "<<FILEID>>" as "textfile.txt" into folder "testshare2" inside space "Shares"
    Then the HTTP status code should be "502"
    And for user "Brian" folder "testshare1" of the space "Shares" should contain these files:
      | textfile.txt |
    But for user "Brian" folder "testshare2" of the space "Shares" should not contain these files:
      | textfile.txt |
    Examples:
      | from-permissions | to-permissions |
      | Editor           | Editor         |
      | Editor           | Viewer         |
      | Viewer           | Editor         |
      | Viewer           | Viewer         |

  @owncloud-issue-8124
  Scenario Outline: move a file from share to personal space
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder        |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <permissions> |
    And user "Brian" has a share "folder" synced
    When user "Brian" moves file with id "<<FILEID>>" as "test.txt" into folder "/" inside space "Personal"
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    And for user "Brian" folder "/" of the space "Personal" should not contain these files:
      | test.txt |
    Examples:
      | permissions | http-status-code |
      | Editor      | 502              |
      | Viewer      | 403              |

  @owncloud-issue-8125
  Scenario Outline: move a file from shares to project space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    And user "Alice" has created folder "testshare"
    And user "Alice" has uploaded file with content "some data" to "testshare/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare     |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <permissions> |
    And user "Brian" has a share "testshare" synced
    When user "Brian" moves file with id "<<FILEID>>" as "textfile.txt" into folder "/" inside space "project-space"
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" folder "testshare" of the space "Shares" should contain these files:
      | textfile.txt |
    But for user "Brian" folder "/" of the space "project-space" should not contain these files:
      | textfile.txt |
    Examples:
      | space-role   | permissions | http-status-code |
      | Manager      | Editor      | 502              |
      | Space Editor | Editor      | 502              |
      | Space Viewer | Editor      | 403              |
      | Manager      | Viewer      | 403              |
      | Space Editor | Viewer      | 403              |
      | Space Viewer | Viewer      | 403              |


  Scenario: rename a root file inside personal space
    Given user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "renamed.txt" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | renamed.txt |
    But for user "Alice" the space "Personal" should not contain these entries:
      | textfile.txt |


  Scenario: rename a file and move into a folder inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "/folder/renamed.txt" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder" of the space "Personal" should contain these files:
      | renamed.txt |
    But for user "Alice" the space "Personal" should not contain these entries:
      | textfile.txt |


  Scenario: rename a file and move into a sub-folder inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "/folder/sub-folder/renamed.txt" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder/sub-folder" of the space "Personal" should contain these files:
      | renamed.txt |
    But for user "Alice" the space "Personal" should not contain these entries:
      | textfile.txt |


  Scenario: rename a file and move from a folder to root inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has uploaded file with content "some data" to "folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "/renamed.txt" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | renamed.txt |
    But for user "Alice" folder "folder" of the space "Personal" should not contain these files:
      | renamed.txt |


  Scenario: rename a file and move from sub-folder to root inside personal space
    Given user "Alice" has created folder "/folder"
    And user "Alice" has created folder "folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "folder/sub-folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" renames file with id "<<FILEID>>" to "/renamed.txt" inside space "Personal"
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these files:
      | renamed.txt |
    But for user "Alice" folder "folder/sub-folder" of the space "Personal" should not contain these files:
      | textfile.txt |

  @owncloud-issue-7617
  Scenario: move a file to a different name into a sub-folder inside share space (editor permissions)
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has created folder "/folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "folder" synced
    When user "Brian" renames file with id "<<FILEID>>" to "folder/sub-folder/renamed.txt" inside space "Shares"
    Then the HTTP status code should be "201"
    And for user "Brian" folder "folder/sub-folder" of the space "Shares" should contain these files:
      | renamed.txt |
    But for user "Brian" folder "folder" of the space "Shares" should not contain these files:
      | test.txt |


  Scenario: move a file to a different name into a sub-folder inside share space (read permissions)
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has created folder "/folder/sub-folder"
    And user "Alice" has uploaded file with content "some data" to "/folder/test.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "folder" synced
    When user "Brian" renames file with id "<<FILEID>>" to "folder/sub-folder/renamed.txt" inside space "Shares"
    Then the HTTP status code should be "502"
    And for user "Brian" folder "folder" of the space "Shares" should contain these files:
      | test.txt |
    But for user "Brian" folder "folder/sub-folder" of the space "Shares" should not contain these files:
      | renamed.txt |

  @owncloud-issue-6739
  Scenario Outline: try to move personal file to other spaces using its id as the destination
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "myspace" with the default quota using the Graph API
    And user "Alice" has uploaded file with content "some data" to "textfile.txt"
    When user "Alice" tries to move file "textfile.txt" of space "Personal" to space "<space-name>" using its id in destination path
    Then the HTTP status code should be "<http-status-code>"
    And for user "Alice" the space "Personal" should contain these entries:
      | textfile.txt |
    Examples:
      | space-name | http-status-code |
      | Personal   | 409              |
      | myspace    | 400              |
      | Shares     | 404              |

  @owncloud-issue-6739
  Scenario Outline: try to move project file to other spaces using its id as the destination
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "myspace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "myspace" with content "some data" to "textfile.txt"
    When user "Alice" tries to move file "textfile.txt" of space "myspace" to space "<space-name>" using its id in destination path
    Then the HTTP status code should be "<http-status-code>"
    And for user "Alice" the space "myspace" should contain these entries:
      | textfile.txt |
    Examples:
      | space-name | http-status-code |
      | Personal   | 400              |
      | myspace    | 409              |
      | Shares     | 404              |

  @owncloud-issue-6739
  Scenario: move a file to folder using its id as the destination (Personal space)
    Given user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And user "Alice" has created folder "docs"
    When user "Alice" moves file "textfile.txt" of space "Personal" to folder "docs" using its id in destination path
    Then the HTTP status code should be "204"
    And the content of file "docs" for user "Alice" should be "some data"
    And as "Alice" file "textfile.txt" should not exist
    And as "Alice" folder "docs" should not exist
    And as "Alice" folder "docs" should exist in the trashbin of the space "Personal"

  @owncloud-issue-6739
  Scenario: move a file to folder using its id as the destination (Project space)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "myspace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "myspace" with content "some data" to "textfile.txt"
    And user "Alice" has created a folder "docs" in space "myspace"
    When user "Alice" moves file "textfile.txt" of space "myspace" to folder "docs" using its id in destination path
    Then the HTTP status code should be "204"
    And for user "Alice" the content of the file "docs" of the space "myspace" should be "some data"
    And as "Alice" folder "docs" should exist in the trashbin of the space "myspace"
    And for user "Alice" folder "/" of the space "myspace" should not contain these files:
      | textfile.txt |
