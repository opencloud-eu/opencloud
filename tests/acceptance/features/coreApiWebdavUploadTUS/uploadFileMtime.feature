Feature: upload file
  As a user
  I want the mtime of an uploaded file to be the creation date on upload source not the upload date
  So that I can find files by their real creation date

  Background:
    Given user "Alice" has been created with default attributes

  @owncloud-issue-10346
  Scenario Outline: upload file with mtime
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload file with future mtime
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2129 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2129 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload a file with mtime in a folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/testFolder/file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "/testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: overwriting a file changes its mtime
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "first time upload content" to "file.txt"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload a file with the same mtime and same content multiple times (atleast 3 times)
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    And the version folder of file "file.txt" for user "Alice" should contain "2" element
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346 @owncloud-issue-10496
  Scenario Outline: upload a file with the same mtime and different content multiple times (atleast 3 times)
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol
    And user "Alice" has uploaded file "filesForUpload/davtest.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol
    When user "Alice" uploads file "filesForUpload/lorem.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol on the WebDAV API
    Then as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    And the version folder of file "file.txt" for user "Alice" should contain "2" element
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
