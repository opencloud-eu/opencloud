@skipOnReva
Feature: sharing
  As a user
  I want to update share permissions
  So that I can decide what resources can be shared with which permission

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes

  @owncloud-issue-1289 @owncloud-issue-7555
  Scenario Outline: keep group permissions in sync when the share is renamed by the receiver and then the permissions are updated by sharer
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | File Editor   |
    And user "Brian" has a share "textfile0.txt" synced
    And using SharingNG
    And user "Brian" has moved file "/Shares/textfile0.txt" to "/Shares/textfile_new.txt"
    When user "Alice" updates the last share using the sharing API with
      | permissions | read |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" sharing with group "grp1" should include
      | id                | A_STRING              |
      | item_type         | file                  |
      | item_source       | A_STRING              |
      | share_type        | group                 |
      | file_source       | A_STRING              |
      | file_target       | /Shares/textfile0.txt |
      | permissions       | read                  |
      | stime             | A_NUMBER              |
      | storage           | A_STRING              |
      | mail_send         | 0                     |
      | uid_owner         | %username%            |
      | displayname_owner | %displayname%         |
      | mimetype          | text/plain            |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: cannot set permissions to zero
    Given using OCS API version "<ocs-api-version>"
    And group "grp1" has been created
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | 0 |
    Then the OCS status code should be "400"
    And the HTTP status code should be "<http-status-code>"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 400              |

  @owncloud-issue-2173
  Scenario Outline: cannot update a share of a file with a user to have only create and/or delete permission
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | <permissions> |
    Then the OCS status code should be "400"
    And the HTTP status code should be "<http-status-code>"
    # Brian should still have at least read access to the shared file
    And as "Brian" entry "/Shares/textfile0.txt" should exist
    Examples:
      | ocs-api-version | http-status-code | permissions   |
      | 1               | 200              | create        |
      | 2               | 400              | create        |
      | 1               | 200              | delete        |
      | 2               | 400              | delete        |
      | 1               | 200              | create,delete |
      | 2               | 400              | create,delete |

  @owncloud-issue-2173
  Scenario Outline: cannot update a share of a file with a group to have only create and/or delete permission
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | <permissions> |
    Then the OCS status code should be "400"
    And the HTTP status code should be "<http-status-code>"
    # Brian in grp1 should still have at least read access to the shared file
    And as "Brian" entry "/Shares/textfile0.txt" should exist
    Examples:
      | ocs-api-version | http-status-code | permissions   |
      | 1               | 200              | create        |
      | 2               | 400              | create        |
      | 1               | 200              | delete        |
      | 2               | 400              | delete        |
      | 1               | 200              | create,delete |
      | 2               | 400              | create,delete |

  @owncloud-issue-2442
  Scenario Outline: share ownership change after moving a shared file to another share
    Given using <dav-path-version> DAV path
    And these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    And user "Alice" has created folder "/Alice-folder"
    And user "Alice" has created folder "/Alice-folder/folder2"
    And user "Carol" has created folder "/Carol-folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | Alice-folder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Brian" has a share "Alice-folder" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | Carol-folder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Brian" has a share "Carol-folder" synced
    When user "Brian" moves folder "/Shares/Alice-folder/folder2" to "/Shares/Carol-folder/folder2" using the WebDAV API
    Then the HTTP status code should be "502"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1253 @owncloud-issue-1224 @owncloud-issue-1225
  Scenario Outline: change the permission of the share and check the API response
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "/Alice-folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | Alice-folder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "Alice-folder" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | all |
    Then the OCS status code should be "<ocs-status-code>"
    And the OCS status message should be "OK"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | id                         | A_STRING             |
      | share_type                 | user                 |
      | uid_owner                  | %username%           |
      | displayname_owner          | %displayname%        |
      | permissions                | all                  |
      | stime                      | A_NUMBER             |
      | parent                     |                      |
      | expiration                 |                      |
      | token                      |                      |
      | uid_file_owner             | %username%           |
      | displayname_file_owner     | %displayname%        |
      | additional_info_owner      | %emailaddress%       |
      | additional_info_file_owner | %emailaddress%       |
      | item_type                  | folder               |
      | item_source                | A_STRING             |
      | path                       | /Alice-folder        |
      | mimetype                   | httpd/unix-directory |
      | storage_id                 | A_STRING             |
      | storage                    | A_STRING             |
      | file_source                | A_STRING             |
      | file_target                | /Shares/Alice-folder |
      | share_with                 | %username%           |
      | share_with_displayname     | %displayname%        |
      | share_with_additional_info | %emailaddress%       |
      | mail_send                  | 0                    |
      | name                       |                      |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: increasing permissions is allowed for owner
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has been created with default attributes
    And user "Carol" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Carol" has been added to group "grp1"
    And user "Carol" has created folder "/FOLDER"
    And user "Carol" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And using SharingNG
    And user "Carol" has updated the last share with
      | permissions | read |
    When user "Carol" updates the last share using the sharing API with
      | permissions | all |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should be able to upload file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: sharer deletes file uploaded with upload-only permission by sharee to a shared folder
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has uploaded file with content "some content" to "/Shares/FOLDER/textFile.txt"
    When user "Alice" deletes file "/FOLDER/textFile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Brian" file "/Shares/FOLDER/textFile.txt" should not exist
    And as "Alice" file "/textFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
