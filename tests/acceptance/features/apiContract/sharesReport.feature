Feature: REPORT request to Shares space
  As a user
  I want to check the share REPORT response
  So that I can make sure that the response contains all the relevant details for shares

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has created folder "/folderMain"
    And user "Alice" has created folder "/folderMain/SubFolder1"
    And user "Alice" has created folder "/folderMain/SubFolder1/subFOLDER2"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /folderMain |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | Viewer      |
    And user "Brian" has a share "folderMain" synced


  Scenario Outline: check the REPORT response of the found folder
    Given using <dav-path-version> DAV path
    When user "Brian" searches for "SubFolder1" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Brian" the REPORT response should contain a resource "SubFolder1" with these key and value pairs:
      | key               | value                     |
      | oc:fileid         | %file_id_pattern%         |
      | oc:file-parent    | %file_id_pattern%         |
      | oc:shareroot      | /folderMain               |
      | oc:name           | SubFolder1                |
      | d:getcontenttype  | httpd/unix-directory      |
      | oc:permissions    | S                         |
      | oc:privatelink    | %base_url%/f/[0-9a-z-$%]+ |
      | oc:remote-item-id | %file_id_pattern%         |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: check the REPORT response of the found file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "Not all those who wander are lost." to "/folderMain/SubFolder1/subFOLDER2/frodo.txt"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/folderMain/SubFolder1/subFOLDER2/testavatar.jpg"
    When user "Brian" searches for "frodo.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Brian" the REPORT response should contain a resource "frodo.txt" with these key and value pairs:
      | key                | value                     |
      | oc:fileid          | %file_id_pattern%         |
      | oc:file-parent     | %file_id_pattern%         |
      | oc:shareroot       | /folderMain               |
      | oc:name            | frodo.txt                 |
      | d:getcontenttype   | text/plain                |
      | oc:permissions     | S                         |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | d:getcontentlength | 34                        |
      | oc:remote-item-id  | %file_id_pattern%         |
    When user "Brian" searches for "testavatar.jpg" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Brian" the REPORT response should contain a resource "testavatar.jpg" with these key and value pairs:
      | key                | value                     |
      | oc:fileid          | %file_id_pattern%         |
      | oc:file-parent     | %file_id_pattern%         |
      | oc:shareroot       | /folderMain               |
      | oc:name            | testavatar.jpg            |
      | d:getcontenttype   | image/jpeg                |
      | oc:permissions     | S                         |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | oc:has-preview     | 1                         |
      | oc:remote-item-id  | %file_id_pattern%         |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: search for the shared folder when share is not accepted
    Given user "Brian" has disabled auto-accepting
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/folderToBrian"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /folderToBrian |
      | space           | Personal       |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | Viewer         |
    When user "Brian" searches for "folderToBrian" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And the search result should contain "0" entries
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-9607 @issue-10329 @env-config
  Scenario Outline: check the REPORT response of a folder shared with secure viewer role
    Given using <dav-path-version> DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has created folder "/secureFolder"
    And user "Alice" has uploaded file with content "secure content" to "/secureFolder/secure.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | secureFolder  |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And user "Brian" has a share "secureFolder" synced
    When user "Brian" searches for "secureFolder" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Brian" the REPORT response should contain a resource "secureFolder" with these key and value pairs:
      | key               | value                     |
      | oc:shareroot      | /secureFolder             |
      | oc:name           | secureFolder              |
      | d:getcontenttype  | httpd/unix-directory      |
      | oc:permissions    | SMX                       |
      | oc:privatelink    | %base_url%/f/[0-9a-z-$%]+ |
      | oc:size           | 14                        |
      | oc:remote-item-id | %file_id_pattern%         |
    When user "Brian" searches for "secure.txt" using the WebDAV API
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    Then the HTTP status code should be "207"
    And as user "Brian" the REPORT response should contain a resource "secure.txt" with these key and value pairs:
      | key                | value                     |
      | oc:shareroot       | /secureFolder             |
      | oc:name            | secure.txt                |
      | d:getcontenttype   | text/plain                |
      | oc:permissions     | SX                        |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | d:getcontentlength | 14                        |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-9607 @issue-10329 @env-config
  Scenario Outline: check the REPORT response of a file shared with secure viewer role
    Given using <dav-path-version> DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has uploaded file with content "secure content" to "/secure.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | secure.txt    |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And user "Brian" has a share "secure.txt" synced
    When user "Brian" searches for "secure.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Brian" the REPORT response should contain a resource "secure.txt" with these key and value pairs:
      | key                | value                     |
      | oc:shareroot       | /secure.txt               |
      | oc:name            | secure.txt                |
      | d:getcontenttype   | text/plain                |
      | oc:permissions     | SMX                       |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | d:getcontentlength | 14                        |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
