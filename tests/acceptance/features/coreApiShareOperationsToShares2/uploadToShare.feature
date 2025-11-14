@skipOnReva
Feature: sharing
  As a user
  I want to upload resources to a folder shared to me
  So that other people can access the resource

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: uploading file to a user read-only share folder does not work
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading file to a group read-only share folder does not work
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading file to a user upload-only share folder works
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the following headers should match these regular expressions for user "Brian"
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading file to a group upload-only share folder works
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the following headers should match these regular expressions for user "Brian"
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest
  Scenario Outline: uploading file to a user read/write share folder works
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading file to a group read/write share folder works
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest
  Scenario Outline: check quota of owners parent directory of a shared file
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Admin" has changed the quota of the personal space of user "Brian" to "0"
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/myfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | myfile.txt  |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Brian" has a share "myfile.txt" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/myfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the following headers should match these regular expressions for user "Brian"
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    And the content of file "/myfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading to a user shared folder with read/write permission when the sharer has insufficient quota does not work
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" to "1"
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/myfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/myfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading to a group shared folder with read/write permission when the sharer has insufficient quota does not work
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" to "1"
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/myfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/myfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading to a user shared folder with upload-only permission when the sharer has insufficient quota does not work
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" to "1"
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/myfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/myfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: uploading to a group shared folder with upload-only permission when the sharer has insufficient quota does not work
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" to "1"
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/FOLDER/myfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/myfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharer can download file uploaded with different permission by sharee to a shared folder
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER             |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file with content "some content" to "/Shares/FOLDER/textFile.txt" using the WebDAV API
    And user "Alice" downloads file "/FOLDER/textFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | dav-path-version | permissions-role |
      | old              | Editor           |
      | old              | Uploader         |
      | new              | Editor           |
      | new              | Uploader         |
      | spaces           | Editor           |
      | spaces           | Uploader         |


  Scenario Outline: upload an empty file (size zero byte) to a shared folder
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/folder-to-share"
    And user "Brian" has sent the following resource share invitation:
      | resource        | folder-to-share |
      | space           | Personal        |
      | sharee          | Alice           |
      | shareType       | user            |
      | permissionsRole | Editor          |
    And user "Alice" has a share "folder-to-share" synced
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "/Shares/folder-to-share/zerobyte.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/Shares/folder-to-share/zerobyte.txt" should exist
    And the content of file "/Shares/folder-to-share/zerobyte.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
