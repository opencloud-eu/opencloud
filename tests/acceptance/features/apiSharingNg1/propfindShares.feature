Feature: propfind a shares
  As a user
  I want to check the PROPFIND response
  So that I can make sure that the response contains all the relevant values

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |

  @owncloud-issue-4421 @owncloud-issue-9933
  Scenario Outline: sharee PROPFIND same name shares shared by multiple users
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has created folder "folderToShare"
    And user "Carol" has uploaded file with content "to share" to "textfile.txt"
    And user "Carol" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource-2>" synced
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "/" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a space "Shares" with these key and value pairs:
      | key       | value             |
      | oc:fileid | %file_id_pattern% |
      | oc:name   | Shares            |
    And as user "Brian" the PROPFIND response should contain a resource "<resource>" with these key and value pairs:
      | key            | value             |
      | oc:fileid      | %file_id_pattern% |
      | oc:name        | <resource>        |
      | oc:permissions | S                 |
    And as user "Brian" the PROPFIND response should contain a resource "<resource-2>" with these key and value pairs:
      | key            | value             |
      | oc:fileid      | %file_id_pattern% |
      | oc:name        | <resource-2>      |
      | oc:permissions | S                 |
    Examples:
      | dav-path-version | resource      | resource-2        |
      | old              | textfile.txt  | textfile (1).txt  |
      | old              | folderToShare | folderToShare (1) |
      | new              | textfile.txt  | textfile (1).txt  |
      | new              | folderToShare | folderToShare (1) |

  @owncloud-issue-4421 @owncloud-issue-9933
  Scenario: sharee PROPFIND a share having bracket in the name
    Given using spaces DAV path
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "to share" to "folderToShare/textfile.txt"
    And user "Carol" has created folder "folderToShare"
    And user "Carol" has uploaded file with content "to share" to "folderToShare/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "folderToShare" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "folderToShare (1)" synced
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "folderToShare (1)" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "folderToShare (1)" with these key and value pairs:
      | key            | value              |
      | oc:fileid      | %share_id_pattern% |
      | oc:name        | folderToShare      |
      | oc:permissions | S                  |
    And as user "Brian" the PROPFIND response should contain a resource "textfile.txt" with these key and value pairs:
      | key            | value             |
      | oc:fileid      | %file_id_pattern% |
      | oc:name        | textfile.txt      |
      | oc:permissions |                   |

  @owncloud-issue-8420
  Scenario Outline: check file-id from PROPFIND with shared-with-me drive-item-id
    Given using spaces DAV path
    And user "Alice" has uploaded file with content "to share" to "/textfile1.txt"
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    When user "Brian" sends PROPFIND request to space "Shares" with depth "1" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the key "oc:fileid" from PROPFIND response should match with shared-with-me drive-item-id of share "<resource>"
    Examples:
      | resource      |
      | textfile1.txt |
      | folderToShare |

  @owncloud-issue-9933
  Scenario Outline: check file-id of different PROPFIND requests to shared items
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "lorem epsum" to "folderToShare/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "folderToShare" synced
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "/" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "folderToShare" with these key and value pairs:
      | key            | value                               |
      | oc:fileid      | <pattern>                           |
      | oc:file-parent | %self::oc:spaceid%!%uuidv4_pattern% |
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "folderToShare" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "folderToShare" with these key and value pairs:
      | key            | value                               |
      | oc:fileid      | <pattern>                           |
      | oc:file-parent | %self::oc:spaceid%!%uuidv4_pattern% |
    And as user "Brian" the PROPFIND response should contain a resource "folderToShare/textfile.txt" with these key and value pairs:
      | key            | value                               |
      | oc:fileid      | %file_id_pattern%                   |
      | oc:file-parent | %self::oc:spaceid%!%uuidv4_pattern% |
    When user "Brian" sends PROPFIND request from the space "Shares" to the resource "folderToShare/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" the PROPFIND response should contain a resource "folderToShare/textfile.txt" with these key and value pairs:
      | key            | value                               |
      | oc:fileid      | %file_id_pattern%                   |
      | oc:file-parent | %self::oc:spaceid%!%uuidv4_pattern% |
    Examples:
      | dav-path-version | pattern            |
      | old              | %file_id_pattern%  |
      | new              | %file_id_pattern%  |
      | spaces           | %share_id_pattern% |
