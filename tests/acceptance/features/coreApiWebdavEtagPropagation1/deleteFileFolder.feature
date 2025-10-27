Feature: propagation of etags when deleting a file or folder
  As a client app
  I want metadata (etags) of parent folders to change when a file or folder is deleted
  So that the client app can know to re-scan and sync the content of the folder(s)

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/upload"


  Scenario Outline: deleting a file changes the etags of all parents
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/file.txt"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    When user "Alice" deletes file "/upload/sub/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path        |
      | Alice | /           |
      | Alice | /upload     |
      | Alice | /upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-4251
  Scenario Outline: deleting a folder changes the etags of all parents
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has created folder "/upload/sub/toDelete"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    When user "Alice" deletes folder "/upload/sub/toDelete" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path        |
      | Alice | /           |
      | Alice | /upload     |
      | Alice | /upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: deleting a folder with content changes the etags of all parents
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has created folder "/upload/sub/toDelete"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/toDelete/file.txt"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    When user "Alice" deletes folder "/upload/sub/toDelete" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path        |
      | Alice | /           |
      | Alice | /upload     |
      | Alice | /upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: sharee deleting a file changes the etags of all parents for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    And user "Brian" has stored etag of element "/Shares/upload/sub"
    When user "Brian" deletes file "/Shares/upload/sub/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path               |
      | Alice | /                  |
      | Alice | /upload            |
      | Alice | /upload/sub        |
      | Brian | /                  |
      | Brian | /Shares            |
      | Brian | /Shares/upload     |
      | Brian | /Shares/upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @skipOnReva
  Scenario Outline: sharer deleting a file changes the etags of all parents for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    And user "Brian" has stored etag of element "/Shares/upload/sub"
    When user "Alice" deletes file "/upload/sub/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path               |
      | Alice | /                  |
      | Alice | /upload            |
      | Alice | /upload/sub        |
      | Brian | /                  |
      | Brian | /Shares            |
      | Brian | /Shares/upload     |
      | Brian | /Shares/upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-4251 @skipOnReva
  Scenario Outline: sharee deleting a folder changes the etags of all parents for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has created folder "/upload/sub/toDelete"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    And user "Brian" has stored etag of element "/Shares/upload/sub"
    When user "Brian" deletes folder "/Shares/upload/sub/toDelete" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path               |
      | Alice | /                  |
      | Alice | /upload            |
      | Alice | /upload/sub        |
      | Brian | /                  |
      | Brian | /Shares            |
      | Brian | /Shares/upload     |
      | Brian | /Shares/upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-4251 @skipOnReva
  Scenario Outline: sharer deleting a folder changes the etags of all parents for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has created folder "/upload/sub/toDelete"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "upload" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/upload"
    And user "Brian" has stored etag of element "/Shares/upload/sub"
    When user "Alice" deletes folder "/upload/sub/toDelete" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path               |
      | Alice | /                  |
      | Alice | /upload            |
      | Alice | /upload/sub        |
      | Brian | /                  |
      | Brian | /Shares            |
      | Brian | /Shares/upload     |
      | Brian | /Shares/upload/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-4251 @skipOnReva @owncloud-issue-10331
  Scenario Outline: deleting a file in a publicly shared folder changes its etag for the sharer
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | upload   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    When the public deletes file "file.txt" from the last link share with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path    |
      | Alice | /       |
      | Alice | /upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-4251 @skipOnReva @owncloud-issue-10331
  Scenario Outline: deleting a folder in a publicly shared folder changes its etag for the sharer
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/upload/sub"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | upload   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    When the public deletes folder "sub" from the last link share with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path    |
      | Alice | /       |
      | Alice | /upload |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
