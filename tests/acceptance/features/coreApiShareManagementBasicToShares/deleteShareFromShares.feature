@skipOnReva @issue-1328 @issue-1289
Feature: sharing
  As a user
  I want to delete shares
  So that I don't have redundant shares

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "example test text file 0" to "/textfile0.txt"

  @issue-7555
  Scenario Outline: delete all group shares
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | File Editor   |
    And user "Brian" has a share "textfile0.txt" synced
    And user "Brian" has moved file "/Shares/textfile0.txt" to "/Shares/anotherName.txt"
    When user "Alice" deletes the last share using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should not see the share id of the last share
    And as "Brian" file "/Shares/textfile0.txt" should not exist
    And as "Brian" file "/Shares/anotherName.txt" should not exist
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest
  Scenario Outline: delete a share
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" deletes the last share using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the last share id should not be included in the response
    And as "Brian" file "/Shares/textfile0.txt" should not exist
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: orphaned shares
    Given using <dav-path-version> DAV path
    And using OCS API version "1"
    And user "Alice" has created folder "/common"
    And user "Alice" has created folder "/common/sub"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /common/sub |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | Viewer      |
    When user "Alice" deletes folder "/common" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Brian" folder "/Shares/sub" should not exist
    And as "Brian" folder "/sub" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest
  Scenario Outline: deleting a file out of a share as recipient creates a backup for the owner
    Given using <dav-path-version> DAV path
    And using OCS API version "1"
    And user "Alice" has created folder "/shared"
    And user "Alice" has moved file "/textfile0.txt" to "/shared/shared_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "shared" synced
    When user "Brian" deletes file "/Shares/shared/shared_file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Brian" file "/Shares/shared/shared_file.txt" should not exist
    And as "Alice" file "/shared/shared_file.txt" should not exist
    And as "Alice" file "/shared_file.txt" should exist in the trashbin
    And as "Brian" the file with original path "/shared_file.txt" should not exist in the trashbin
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: deleting a folder out of a share as recipient creates a backup for the owner
    Given using <dav-path-version> DAV path
    And using OCS API version "1"
    And user "Alice" has created folder "/shared"
    And user "Alice" has created folder "/shared/sub"
    And user "Alice" has moved file "/textfile0.txt" to "/shared/sub/shared_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "shared" synced
    When user "Brian" deletes folder "/Shares/shared/sub" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Brian" folder "/Shares/shared/sub" should not exist
    And as "Alice" folder "/shared/sub" should not exist
    And as "Alice" folder "/sub" should exist in the trashbin
    And as "Alice" file "/sub/shared_file.txt" should exist in the trashbin
    And as "Brian" the folder with original path "/sub" should not exist in the trashbin
    And as "Brian" the file with original path "/sub/shared_file.txt" should not exist in the trashbin
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest
  Scenario: unshare from self
    And group "grp1" has been created
    And these users have been created with default attributes:
      | username |
      | Carol    |
    And user "Brian" has been added to group "grp1"
    And user "Carol" has been added to group "grp1"
    And user "Carol" has created folder "PARENT"
    And user "Carol" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | /PARENT/parent.txt |
      | space           | Personal           |
      | sharee          | grp1               |
      | shareType       | group              |
      | permissionsRole | Viewer             |
    And user "Carol" has stored etag of element "/PARENT"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    When user "Brian" declines share "/Shares/parent.txt" offered by user "Carol" using the sharing API
    Then the HTTP status code should be "200"
    And the etag of element "/" of user "Brian" should have changed
    And the etag of element "/Shares" of user "Brian" should have changed
    And the etag of element "/PARENT" of user "Carol" should not have changed


  Scenario Outline: sharee of a read-only share folder tries to delete the shared folder
    Given using <dav-path-version> DAV path
    And using OCS API version "1"
    And user "Alice" has created folder "/shared"
    And user "Alice" has moved file "/textfile0.txt" to "/shared/shared_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "shared" synced
    When user "Brian" deletes file "/Shares/shared/shared_file.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/shared/shared_file.txt" should exist
    And as "Brian" file "/Shares/shared/shared_file.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharee of a upload-only shared folder tries to delete a file in the shared folder
    Given using <dav-path-version> DAV path
    And using OCS API version "1"
    And user "Alice" has created folder "/shared"
    And user "Alice" has moved file "/textfile0.txt" to "/shared/shared_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "shared" synced
    When user "Brian" deletes file "/Shares/shared/shared_file.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/shared/shared_file.txt" should exist
    And as "Brian" file "/Shares/shared/shared_file.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharee of an upload-only shared folder tries to delete their file in the folder
    Given using <dav-path-version> DAV path
    And using OCS API version "1"
    And user "Alice" has created folder "/shared"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "shared" synced
    And user "Brian" has uploaded file "filesForUpload/textfile.txt" to "/Shares/shared/textfile.txt"
    When user "Brian" deletes file "/Shares/shared/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/shared/textfile.txt" should exist
    And as "Brian" file "/Shares/shared/textfile.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: group share recipient tries to delete the share
    Given using OCS API version "<ocs-api-version>"
    And group "grp1" has been created
    And these users have been created with default attributes:
      | username |
      | Carol    |
    And user "Brian" has been added to group "grp1"
    And user "Carol" has been added to group "grp1"
    And user "Alice" has created folder "/shared"
    And user "Alice" has moved file "/textfile0.txt" to "/shared/shared_file.txt"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | <entry-to-share>  |
      | space           | Personal          |
      | sharee          | grp1              |
      | shareType       | group             |
      | permissionsRole | <permission-role> |
    And user "Brian" has a share "<sync-entry>" synced
    When user "Brian" deletes the last share of user "Alice" using the sharing API
    Then the OCS status code should be "996"
    And the HTTP status code should be "<http-status-code>"
    And as "Alice" entry "<entry-to-share>" should exist
    And as "Brian" entry "<received-entry>" should exist
    And as "Carol" entry "<received-entry>" should exist
    Examples:
      | entry-to-share          | permission-role | ocs-api-version | http-status-code | received-entry          | sync-entry      |
      | /shared/shared_file.txt | File Editor     | 1               | 200              | /Shares/shared_file.txt | shared_file.txt |
      | /shared/shared_file.txt | File Editor     | 2               | 500              | /Shares/shared_file.txt | shared_file.txt |
      | /shared                 | Editor          | 1               | 200              | /Shares/shared          | shared          |
      | /shared                 | Editor          | 2               | 500              | /Shares/shared          | shared          |


  Scenario Outline: individual share recipient tries to delete the share
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created folder "/shared"
    And user "Alice" has moved file "/textfile0.txt" to "/shared/shared_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <entry-to-share>  |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <permission-role> |
    And user "Brian" has a share "<sync-entry>" synced
    When user "Brian" deletes the last share of user "Alice" using the sharing API
    Then the OCS status code should be "996"
    And the HTTP status code should be "<http-status-code>"
    And as "Alice" entry "<entry-to-share>" should exist
    And as "Brian" entry "<received-entry>" should exist
    Examples:
      | entry-to-share          | permission-role | ocs-api-version | http-status-code | received-entry          | sync-entry      |
      | /shared/shared_file.txt | File Editor     | 1               | 200              | /Shares/shared_file.txt | shared_file.txt |
      | /shared/shared_file.txt | File Editor     | 2               | 500              | /Shares/shared_file.txt | shared_file.txt |
      | /shared                 | Editor          | 1               | 200              | /Shares/shared          | shared          |
      | /shared                 | Editor          | 2               | 500              | /Shares/shared          | shared          |

  @issue-720
  Scenario Outline: request PROPFIND after sharer deletes the collaborator
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    And user "Brian" has a share "textfile0.txt" synced
    When user "Alice" deletes the last share using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    When user "Brian" requests "/dav/files/%username%" with "PROPFIND" using basic auth
    Then the HTTP status code should be "207"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @issue-1229
  Scenario Outline: delete a share with wrong authentication
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    And user "Brian" has a share "textfile0.txt" synced
    When user "Brian" tries to delete the last share of user "Alice" using the sharing API
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "996"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 500              |


  Scenario Outline: unshare a shared resources
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    When user "Alice" unshares file "textfile0.txt" shared to "Brian"
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as "Brian" file "/Shares/textfile0.txt" should not exist
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
