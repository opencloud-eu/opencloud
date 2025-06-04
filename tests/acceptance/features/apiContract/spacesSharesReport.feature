Feature: Report test
  As a user
  I want to check the share REPORT response
  So that I can make sure that the response contains all the relevant details for shares

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "find data" with the default quota using the Graph API
    And user "Alice" has created a folder "folderMain/SubFolder1/subFOLDER2" in space "find data"
    And user "Alice" has uploaded a file inside space "find data" with content "some content" to "folderMain/SubFolder1/subFOLDER2/insideTheFolder.txt"


  Scenario Outline: check the response of the found folder
    Given using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderMain |
      | space           | find data  |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "folderMain" synced
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
      | oc:size           | 12                        |
      | oc:remote-item-id | %file_id_pattern%         |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: check the response of the found file
    Given using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderMain |
      | space           | find data  |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Brian" has a share "folderMain" synced
    When user "Brian" searches for "insideTheFolder.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Brian" the REPORT response should contain a resource "insideTheFolder.txt" with these key and value pairs:
      | key                | value                     |
      | oc:fileid          | %file_id_pattern%         |
      | oc:file-parent     | %file_id_pattern%         |
      | oc:shareroot       | /folderMain               |
      | oc:name            | insideTheFolder.txt       |
      | d:getcontenttype   | text/plain                |
      | oc:permissions     | SD                        |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | d:getcontentlength | 12                        |
      | oc:remote-item-id  | %file_id_pattern%         |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: search for the shared folder when the share is not accepted
    Given using <dav-path-version> DAV path
    And user "Brian" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderMain |
      | space           | find data  |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" searches for "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And the search result should contain "0" entries
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
