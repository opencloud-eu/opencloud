Feature: move (rename) folder
  As a user
  I want to be able to move and rename folders
  So that I can quickly manage my file system

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes


  Scenario Outline: rename a folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" moves folder "testFolder" to "renamedFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "renamedFolder" should exist
    But as "Alice" folder "testFolder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-3099
  Scenario Outline: renaming a folder to a backslash should return an error
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    When user "Alice" moves folder "/testshare" to "\" using the WebDAV API
    Then the HTTP status code should be "400"
    And user "Alice" should see the following elements
      | /testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-3099
  Scenario Outline: renaming a folder beginning with a backslash should return an error
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    When user "Alice" moves folder "/testshare" to "\testshare" using the WebDAV API
    Then the HTTP status code should be "400"
    And user "Alice" should see the following elements
      | /testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-3099
  Scenario Outline: renaming a folder including a backslash encoded should return an error
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    When user "Alice" moves folder "/testshare" to "/hola\hola" using the WebDAV API
    Then the HTTP status code should be "400"
    And user "Alice" should see the following elements
      | /testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: move a folder into an other folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    And user "Alice" has created folder "/an-other-folder"
    When user "Alice" moves folder "/testshare" to "/an-other-folder/testshare" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should not see the following elements
      | /testshare |
    And user "Alice" should see the following elements
      | /an-other-folder/testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: move a folder into a nonexistent folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    When user "Alice" moves folder "/testshare" to "/not-existing/testshare" using the WebDAV API
    Then the HTTP status code should be "409"
    And user "Alice" should see the following elements
      | /testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: renaming folder with dots in the path
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    And user "Alice" has uploaded file with content "uploaded content for file name ending with a dot" to "<folder-name>/abc.txt"
    When user "Alice" moves folder "<folder-name>" to "/uploadFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/uploadFolder/abc.txt" for user "Alice" should be "uploaded content for file name ending with a dot"
    Examples:
      | dav-path-version | folder-name   |
      | old              | /upload.      |
      | old              | /upload.1     |
      | old              | /upload...1.. |
      | old              | /...          |
      | old              | /..upload     |
      | new              | /upload.      |
      | new              | /upload.1     |
      | new              | /upload...1.. |
      | new              | /...          |
      | new              | /..upload     |
      | spaces           | /upload.      |
      | spaces           | /upload.1     |
      | spaces           | /upload...1.. |
      | spaces           | /...          |
      | spaces           | /..upload     |

  @issue-3023
  Scenario Outline: moving a folder into a sub-folder of itself
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has created folder "PARENT/CHILD"
    And user "Alice" has uploaded file with content "parent text" to "/PARENT/parent.txt"
    And user "Alice" has uploaded file with content "child text" to "/PARENT/CHILD/child.txt"
    When user "Alice" moves folder "/PARENT" to "/PARENT/CHILD/PARENT" using the WebDAV API
    Then the HTTP status code should be "409"
    And the content of file "/PARENT/parent.txt" for user "Alice" should be "parent text"
    And the content of file "/PARENT/CHILD/child.txt" for user "Alice" should be "child text"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: rename folder to/from special characters
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder <from-folder-name>
    When user "Alice" moves folder <from-folder-name> to <to-folder-name> using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder <to-folder-name> should exist
    But as "Alice" folder <from-folder-name> should not exist
    Examples:
      | dav-path-version | from-folder-name           | to-folder-name             |
      | old              | "testFolder"               | "'single'quotes"           |
      | old              | "testFolder"               | '"double"quotes'           |
      | old              | "testFolder"               | "strängé नेपाली folder"    |
      | old              | "testFolder"               | "$%#?&@"                   |
      | old              | "testFolder"               | "Sample,Folder,With,Comma" |
      | old              | "testFolder"               | " start with space"        |
      | old              | "testFolder"               | "renamed.part"             |
      | old              | "'single'quotes"           | "testFolder"               |
      | old              | '"double"quotes'           | "testFolder"               |
      | old              | "strängé नेपाली folder"    | "testFolder"               |
      | old              | "$%#?&@"                   | "testFolder"               |
      | old              | "Sample,Folder,With,Comma" | "testFolder"               |
      | old              | " start with space"        | "testFolder"               |
      | old              | "renamed.part"             | "testFolder"               |
      | new              | "testFolder"               | "'single'quotes"           |
      | new              | "testFolder"               | '"double"quotes'           |
      | new              | "testFolder"               | "strängé नेपाली folder"    |
      | new              | "testFolder"               | "$%#?&@"                   |
      | new              | "testFolder"               | "Sample,Folder,With,Comma" |
      | new              | "testFolder"               | " start with space"        |
      | new              | "testFolder"               | "renamed.part"             |
      | new              | "'single'quotes"           | "testFolder"               |
      | new              | '"double"quotes'           | "testFolder"               |
      | new              | "strängé नेपाली folder"    | "testFolder"               |
      | new              | "$%#?&@"                   | "testFolder"               |
      | new              | "Sample,Folder,With,Comma" | "testFolder"               |
      | new              | " start with space"        | "testFolder"               |
      | new              | "renamed.part"             | "testFolder"               |
      | spaces           | "testFolder"               | "'single'quotes"           |
      | spaces           | "testFolder"               | '"double"quotes'           |
      | spaces           | "testFolder"               | "strängé नेपाली folder"    |
      | spaces           | "testFolder"               | "$%#?&@"                   |
      | spaces           | "testFolder"               | "Sample,Folder,With,Comma" |
      | spaces           | "testFolder"               | " start with space"        |
      | spaces           | "testFolder"               | "renamed.part"             |
      | spaces           | "'single'quotes"           | "testFolder"               |
      | spaces           | '"double"quotes'           | "testFolder"               |
      | spaces           | "strängé नेपाली folder"    | "testFolder"               |
      | spaces           | "$%#?&@"                   | "testFolder"               |
      | spaces           | "Sample,Folder,With,Comma" | "testFolder"               |
      | spaces           | " start with space"        | "testFolder"               |
      | spaces           | "renamed.part"             | "testFolder"               |


  Scenario Outline: try to rename folder to name having white space at the end
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" moves folder "testFolder" to "space at end " using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "space at end" should exist
    But as "Alice" folder "testFolder" should not exist
    And as "Alice" folder "space at end " should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1976
  Scenario Outline: try to rename folder to same name
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" moves folder "testFolder" to "testFolder" using the WebDAV API
    Then the HTTP status code should be "404"
    And as "Alice" the folder with original path "testFolder" should not exist in the trashbin
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: rename a folder to existing folder name
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    And user "Alice" has uploaded file with content "some content" to "testFolder/lorem.txt"
    And user "Alice" has created folder "renamedFolder"
    When user "Alice" moves folder "testFolder" to "renamedFolder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "renamedFolder" should exist
    And the content of file "renamedFolder/lorem.txt" for user "Alice" should be "some content"
    And as "Alice" the folder with original path "renamedFolder" should exist in the trashbin
    But as "Alice" folder "testFolder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: try to rename folder to . and ..
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" moves folder "testFolder" to "<folder-name>" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    Examples:
      | dav-path-version | folder-name | http-status-code |
      | old              | /.          | 409              |
      | old              | /..         | 404              |
      | new              | /.          | 409              |
      | new              | /..         | 404              |
      | spaces           | /.          | 409              |
      | spaces           | /..         | 400              |


  Scenario Outline: rename a folder to .htaccess
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    When user "Alice" moves folder "/testshare" to "/.htaccess" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should see the following elements
      | /.htaccess |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
