@skipOnReva
Feature: upload file
  As a user
  I want the mtime of an uploaded file to be the creation date on upload source not the upload date
  So that I can find files by their real creation date

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |

  @owncloud-issue-10346
  Scenario Outline: upload file with mtime to a received share
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Brian" the mtime of the file "/Shares/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-10346
  Scenario Outline: upload file with mtime to a send share
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Brian" the mtime of the file "/Shares/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: overwriting a file with mtime in a received share
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    And user "Alice" has uploaded file with content "uploaded content" to "/toShare/file.txt"
    When user "Brian" uploads file "filesForUpload/textfile.txt" to "/Shares/toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Brian" the mtime of the file "/Shares/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-10346
  Scenario Outline: overwriting a file with mtime in a send share
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    And user "Brian" has uploaded file with content "uploaded content" to "/Shares/toShare/file.txt"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Brian" the mtime of the file "/Shares/toShare/file.txt" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
