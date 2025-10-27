Feature: propagation of etags when uploading data
  As a client app
  I want metadata (etags) of parent folders to change when a file is uploaded
  So that the client app can know to re-scan and sync the content of the folder(s)

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/upload"

  @owncloud-issue-4251
  Scenario Outline: uploading a file inside a folder changes its etag
    Given using <dav-path-version> DAV path
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    When user "Alice" uploads file with content "uploaded content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path    |
      | Alice | /       |
      | Alice | /upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: overwriting a file inside a folder changes its etag
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/file.txt"
    When user "Alice" uploads file with content "new content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path             |
      | Alice | /                |
      | Alice | /upload          |
      | Alice | /upload/file.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-4251 @skipOnReva
  Scenario Outline: sharee uploading a file inside a received shared folder should update etags for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    When user "Brian" uploads file with content "uploaded content" to "/Shares/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path           |
      | Alice | /              |
      | Alice | /upload        |
      | Brian | /              |
      | Brian | /Shares        |
      | Brian | /Shares/upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-4251 @skipOnReva
  Scenario Outline: sharer uploading a file inside a shared folder should update etags for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    When user "Alice" uploads file with content "uploaded content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path           |
      | Alice | /              |
      | Alice | /upload        |
      | Brian | /              |
      | Brian | /Shares        |
      | Brian | /Shares/upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @skipOnReva
  Scenario Outline: sharee overwriting a file inside a received shared folder should update etags for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    When user "Brian" uploads file with content "new content" to "/Shares/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path           |
      | Alice | /              |
      | Alice | /upload        |
      | Brian | /              |
      | Brian | /Shares        |
      | Brian | /Shares/upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @skipOnReva
  Scenario Outline: sharer overwriting a file inside a shared folder should update etags for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    When user "Alice" uploads file with content "new content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path           |
      | Alice | /              |
      | Alice | /upload        |
      | Brian | /              |
      | Brian | /Shares        |
      | Brian | /Shares/upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-4251 @skipOnReva @owncloud-issue-10331
  Scenario Outline: uploading a file into a publicly shared folder changes its etag for the sharer
    Given using <dav-path-version> DAV path
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | upload     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    When the public uploads file "file.txt" with password "%public%" and content "new content" using the public WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path    |
      | Alice | /       |
      | Alice | /upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
