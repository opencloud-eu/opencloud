Feature: upload resources using TUS protocol
  As a user
  I want to be able to upload files
  So that I can store and share files between multiple client systems

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path


  Scenario: upload a file within the set quota to a project space
    Given user "Alice" has created a space "Project Jupiter" of type "project" with quota "10000"
    When user "Alice" uploads a file with content "uploaded content" to "/upload.txt" via TUS inside of the space "Project Jupiter" using the WebDAV API
    Then for user "Alice" the space "Project Jupiter" should contain these entries:
      | upload.txt |


  Scenario: upload a file bigger than the set quota to a project space
    Given user "Alice" has created a space "Project Jupiter" of type "project" with quota "10"
    When user "Alice" creates a new TUS resource for the space "Project Jupiter" with content "file content is 24 bytes" using the WebDAV API with these headers:
      | Upload-Length   | 24                              |
      # dXBsb2FkLnR4dA== is the base64 encoded value of filename upload.txt
      | Upload-Metadata | filename dXBsb2FkLnR4dA==       |
      | Content-Type    | application/offset+octet-stream |
      | Tus-Resumable   | 1.0.0                           |
      | Tus-Extension   | creation-with-upload            |
    Then the HTTP status code should be "507"
    And for user "Alice" the space "Project Jupiter" should not contain these entries:
      | upload.txt |


  Scenario: upload the same file after renaming the first one
    Given user "Alice" has uploaded a file with content "uploaded content" to "/upload.txt" via TUS inside of the space "Alice Hansen"
    And user "Alice" has moved file "upload.txt" to "test.txt" in space "Personal"
    When user "Alice" uploads a file with content "uploaded content" to "/upload.txt" via TUS inside of the space "Alice Hansen" using the WebDAV API
    Then for user "Alice" the space "Personal" should contain these entries:
      | test.txt   |
      | upload.txt |

  @issue-10346
  Scenario Outline: upload a zero-byte file inside a shared folder
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "testFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Brian" has a share "testFolder" synced
    When user "Brian" uploads file "filesForUpload/zerobyte.txt" to "Shares/testFolder/textfile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "Shares/testFolder/textfile.txt" for user "Brian" should be ""
    And the content of file "testFolder/textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |


  Scenario: upload a zero-byte file inside a shared folder (spaces dav path)
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "testFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Brian" has a share "testFolder" synced
    When user "Brian" uploads a file from "filesForUpload/zerobyte.txt" to "testFolder/textfile.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" the content of the file "testFolder/textfile.txt" of the space "Shares" should be ""
    And for user "Alice" the content of the file "testFolder/textfile.txt" of the space "Personal" should be ""


  Scenario: upload a zero-byte file inside a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" uploads a file from "filesForUpload/zerobyte.txt" to "textfile.txt" via TUS inside of the space "new-space" using the WebDAV API
    Then for user "Alice" the content of the file "textfile.txt" of the space "new-space" should be ""

  @issue-8003 @issue-10346
  Scenario Outline: replace a shared file with zero-byte file
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "This is TUS upload" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textfile.txt" synced
    When user "Brian" uploads file "filesForUpload/zerobyte.txt" to "Shares/textfile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "Shares/textfile.txt" for user "Brian" should be ""
    And the content of file "textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-8003
  Scenario: replace a shared file with zero-byte file (spaces dav path)
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "This is TUS upload" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textfile.txt" synced
    When user "Brian" uploads a file from "filesForUpload/zerobyte.txt" to "textfile.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" the content of the file "textfile.txt" of the space "Shares" should be ""
    And for user "Alice" the content of the file "textfile.txt" of the space "Personal" should be ""

  @issue-8003
  Scenario: replace a file inside a project space with zero-byte file
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "This is TUS upload" to "textfile.txt"
    When user "Alice" uploads a file from "filesForUpload/zerobyte.txt" to "textfile.txt" via TUS inside of the space "new-space" using the WebDAV API
    Then for user "Alice" the content of the file "textfile.txt" of the space "new-space" should be ""

  @issue-8003
  Scenario: replace a file inside a shared project space with zero-byte file
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "This is TUS upload" to "textfile.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    When user "Brian" uploads a file from "filesForUpload/zerobyte.txt" to "textfile.txt" via TUS inside of the space "new-space" using the WebDAV API
    Then for user "Brian" the content of the file "textfile.txt" of the space "new-space" should be ""
    And for user "Alice" the content of the file "textfile.txt" of the space "new-space" should be ""
