Feature: move file
  As a user
  I want the mtime of a file to be preserved when it is moved
  So that a moved file keeps its original modification date

  Background:
    Given user "Alice" has been created with default attributes

  @issue-2378
  Scenario Outline: moving a file preserves the mtime
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol
    When user "Alice" moves file "file.txt" to "renamed.txt" using the WebDAV API
    Then as "Alice" the mtime of the file "renamed.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-2378
  Scenario Outline: moving a file into a folder preserves the mtime
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the TUS protocol
    When user "Alice" moves file "file.txt" to "/testFolder/file.txt" using the WebDAV API
    Then as "Alice" the mtime of the file "/testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
