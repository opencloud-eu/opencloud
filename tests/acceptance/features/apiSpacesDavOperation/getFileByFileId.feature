Feature: accessing files using file id
  As a user
  I want to access the files using file id
  So that I can get the content of a file

  Background:
    Given using spaces DAV path
    And user "Alice" has been created with default attributes


  Scenario Outline: get content of a file
    Given user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: get content of a file inside a folder
    Given user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "some data" to "uploadFolder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: get content of a file inside a project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee gets content of a shared file
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "textfile.txt" synced
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path                          |
      | /dav/spaces/<<FILEID>>            |


  Scenario Outline: sharee gets content of a file inside a shared folder
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "some data" to "uploadFolder/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "uploadFolder" synced
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee gets content of a file inside a shared space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: user tries to get content of file owned by others
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "404"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee gets content of a shared file when sync is disable
    Given user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee gets content of a file inside a shared folder when sync is disable
    Given user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "some data" to "uploadFolder/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: user who is member of group gets content of a shared file when sync is disable
    Given user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    And the administrator has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: user who is member of group gets content of a shared folder when sync is disable
    Given user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "some data" to "uploadFolder/textfile.txt"
    And we save it into "FILEID"
    And the administrator has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some data"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee gets content of a shared file in project space when sync is disabled
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee gets content of a file inside a shared folder in project space when sync is disabled
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has created a folder "uploadFolder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "uploadFolder/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: user who is member of group gets content of a shared file in project space when sync is disabled
    Given using spaces DAV path
    And the administrator has created a group "grp1" using the Graph API
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: user who is member of group gets content of a file from shared folder in project space when sync is disabled
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Brian" has been created with default attributes
    And user "Brian" has disabled the auto-sync share
    And user "Alice" has created a folder "uploadFolder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "uploadFolder/textfile.txt"
    And we save it into "FILEID"
    And the administrator has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | new-space    |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    When user "Brian" sends HTTP method "GET" to URL "<dav-path>"
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |
