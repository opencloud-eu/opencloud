Feature: REPORT request to project space
  As a user
  I want to check the REPORT response of project spaces
  So that I can make sure that the response contains all the relevant details

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "findData" with the default quota using the Graph API

  @issue-10329
  Scenario: check the response of the searched file
    Given user "Alice" has uploaded a file inside space "findData" with content "some content" to "testFile.txt"
    When user "Alice" searches for "testFile.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | testFile.txt |
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Alice" the REPORT response should contain a resource "testFile.txt" with these key and value pairs:
      | key                | value                     |
      | oc:fileid          | %file_id_pattern%         |
      | oc:file-parent     | %file_id_pattern%         |
      | oc:name            | testFile.txt              |
      | d:getcontenttype   | text/plain                |
      | oc:permissions     | RDNVW                     |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | d:getcontentlength | 12                        |

  @issue-10329
  Scenario: check the response of the searched sub-file
    Given user "Alice" has created a folder "folderMain/SubFolder1/subFOLDER2" in space "findData"
    And user "Alice" has uploaded a file inside space "findData" with content "some content" to "folderMain/SubFolder1/subFOLDER2/insideTheFolder.txt"
    When user "Alice" searches for "insideTheFolder.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | insideTheFolder.txt |
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Alice" the REPORT response should contain a resource "insideTheFolder.txt" with these key and value pairs:
      | key                | value                     |
      | oc:fileid          | %file_id_pattern%         |
      | oc:file-parent     | %file_id_pattern%         |
      | oc:name            | insideTheFolder.txt       |
      | d:getcontenttype   | text/plain                |
      | oc:permissions     | RDNVW                     |
      | oc:privatelink     | %base_url%/f/[0-9a-z-$%]+ |
      | d:getcontentlength | 12                        |

  @issue-10329
  Scenario: check the response of the searched folder
    Given user "Alice" has created a folder "folderMain" in space "findData"
    When user "Alice" searches for "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | folderMain |
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And as user "Alice" the REPORT response should contain a resource "folderMain" with these key and value pairs:
      | key              | value                     |
      | oc:fileid        | %file_id_pattern%         |
      | oc:file-parent   | %file_id_pattern%         |
      | oc:name          | folderMain                |
      | d:getcontenttype | httpd/unix-directory      |
      | oc:permissions   | RDNVCK                    |
      | oc:privatelink   | %base_url%/f/[0-9a-z-$%]+ |
      | oc:size          | 0                         |

  @issue-10329
  Scenario: check the response of the searched sub-folder
    Given user "Alice" has created a folder "folderMain/sub-folder" in space "findData"
    When user "Alice" searches for "*sub*" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | sub-folder |
    And the following headers should match these regular expressions
      | X-Request-Id | %request_id_pattern% |
    And the HTTP status code should be "207"
    And as user "Alice" the REPORT response should contain a resource "sub-folder" with these key and value pairs:
      | key              | value                     |
      | oc:fileid        | %file_id_pattern%         |
      | oc:file-parent   | %file_id_pattern%         |
      | oc:name          | sub-folder                |
      | d:getcontenttype | httpd/unix-directory      |
      | oc:permissions   | RDNVCK                    |
      | oc:privatelink   | %base_url%/f/[0-9a-z-$%]+ |
      | oc:size          | 0                         |
