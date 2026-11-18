Feature: propfind a file using file id
  As a user
  I want to check the PROPFIND response of file using their file id
  So that I can make sure that the response contains all the relevant values

  Background:
    Given using spaces DAV path
    And user "Alice" has been created with default attributes


  Scenario Outline: send a PROPFIND request to a file inside root of personal space
    Given user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Alice" the PROPFIND response should contain a resource "<<FILEID>>" with these key and value pairs:
      | key            | value        |
      | oc:name        | textfile.txt |
      | oc:permissions | RDNVWZP      |
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: send a PROPFIND request to a file inside a folder of personal space
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "some data" to "folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Alice" the PROPFIND response should contain a resource "<<FILEID>>" with these key and value pairs:
      | key            | value        |
      | oc:name        | textfile.txt |
      | oc:permissions | RDNVWZP      |
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: send a PROPFIND request to a file in personal space owned by another user
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "textfile.txt"
    And we save it into "FILEID"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "404"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: send a PROPFIND request to a file of inside project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Alice" the PROPFIND response should contain a resource "<<FILEID>>" with these key and value pairs:
      | key            | value        |
      | oc:name        | textfile.txt |
      | oc:permissions | RDNVWZP      |
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: send a PROPFIND request to a file inside a folder of project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "/folder/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Alice" the PROPFIND response should contain a resource "<<FILEID>>" with these key and value pairs:
      | key            | value        |
      | oc:name        | textfile.txt |
      | oc:permissions | RDNVWZP      |
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: send a PROPFIND request to a file inside project space owned by another user
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "404"
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: send a PROPFIND request to a shared file
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textfile.txt" synced
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "<<FILEID>>" with these key and value pairs:
      | key            | value        |
      | oc:name        | textfile.txt |
      | oc:permissions | SW           |
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |


  Scenario Outline: sharee sends a PROPFIND request to a file inside of a shared folder
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "/folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "folder" synced
    And user "Alice" has uploaded file with content "some data" to "/folder/textfile.txt"
    And we save it into "FILEID"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "<<FILEID>>" with these key and value pairs:
      | key            | value        |
      | oc:name        | textfile.txt |
      | oc:permissions | DNVW         |
    Examples:
      | dav-path               |
      | /dav/spaces/<<FILEID>> |
