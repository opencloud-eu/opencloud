Feature: create files and folder
  As a user
  I want to be able to create files and folders
  So that I can organise the files in my file system

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes


  Scenario Outline: create a folder
    Given using <dav-path-version> DAV path
    When user "Alice" creates folder <folder-name> using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder <folder-name> should exist
    Examples:
      | dav-path-version | folder-name      |
      | old              | "upload"         |
      | old              | "strängé folder" |
      | old              | "C++ folder.cpp" |
      | old              | "नेपाली"         |
      | old              | "folder #2"      |
      | old              | "folder ?2"      |
      | old              | "😀 🤖"          |
      | old              | "new&folder"     |
      | old              | "Sample,comma"   |
      | old              | "'single'"       |
      | old              | '"double"'       |
      | new              | "upload"         |
      | new              | "strängé folder" |
      | new              | "C++ folder.cpp" |
      | new              | "नेपाली"         |
      | new              | "folder #2"      |
      | new              | "folder ?2"      |
      | new              | "😀 🤖"          |
      | new              | "new&folder"     |
      | new              | "Sample,comma"   |
      | new              | "'single'"       |
      | new              | '"double"'       |
      | new              | "नेपाली"         |
      | spaces           | "upload"         |
      | spaces           | "strängé folder" |
      | spaces           | "C++ folder.cpp" |
      | spaces           | "नेपाली"         |
      | spaces           | "folder #2"      |
      | spaces           | "folder ?2"      |
      | spaces           | "😀 🤖"          |
      | spaces           | "new&folder"     |
      | spaces           | "Sample,comma"   |
      | spaces           | "'single'"       |
      | spaces           | '"double"'       |

  @smokeTest
  Scenario Outline: get resourcetype property of a folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test_folder"
    When user "Alice" gets the following properties of folder "/test_folder" using the WebDAV API
      | propertyName   |
      | d:resourcetype |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:resourcetype" with a child property "d:collection"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: get resourcetype property of a folder with special chars
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test_folder:5"
    When user "Alice" gets the following properties of folder "/test_folder:5" using the WebDAV API
      | propertyName   |
      | d:resourcetype |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:resourcetype" with a child property "d:collection"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1345 @owncloud-issue-2177
  Scenario Outline: creating a directory which contains .part should not be possible
    Given using <dav-path-version> DAV path
    When user "Alice" creates folder "/folder.with.ext.part" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "folder.with.ext.part" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1283
  Scenario Outline: try to create a folder that already exists
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "my-data"
    When user "Alice" creates folder "my-data" using the WebDAV API
    Then the HTTP status code should be "405"
    And as "Alice" folder "my-data" should exist
    And the DAV exception should be "Sabre\DAV\Exception\MethodNotAllowed"
    And the DAV message should be "The resource you tried to create already exists"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1283
  Scenario Outline: try to create a folder with a name of an existing file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded data" to "/my-data.txt"
    When user "Alice" creates folder "my-data.txt" using the WebDAV API
    Then the HTTP status code should be "405"
    And the DAV exception should be "Sabre\DAV\Exception\MethodNotAllowed"
    And the DAV message should be "The resource you tried to create already exists"
    And the content of file "/my-data.txt" for user "Alice" should be "uploaded data"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: create a file
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "some text" to <file-name> using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file <file-name> should exist
    And the content of file <file-name> for user "Alice" should be "some text"
    Examples:
      | dav-path-version | file-name          |
      | old              | "upload.txt"       |
      | old              | "strängéfile.txt"  |
      | old              | "C++ file.cpp"     |
      | old              | "नेपाली"           |
      | old              | "file #2.txt"      |
      | old              | "file ?2.pdf"      |
      | old              | "😀 🤖.txt"        |
      | old              | "new&file.txt"     |
      | old              | "Sample,comma.txt" |
      | old              | "'single'.txt"     |
      | old              | '"double".txt'     |
      | new              | "upload.txt"       |
      | new              | "strängéfile.txt"  |
      | new              | "C++ file.cpp"     |
      | new              | "नेपाली"           |
      | new              | "file #2.txt"      |
      | new              | "file ?2.pdf"      |
      | new              | "😀 🤖.txt"        |
      | new              | "new&file.txt"     |
      | new              | "Sample,comma.txt" |
      | new              | "'single'.txt"     |
      | new              | '"double".txt'     |
      | spaces           | "upload.txt"       |
      | spaces           | "strängéfile.txt"  |
      | spaces           | "C++ file.cpp"     |
      | spaces           | "नेपाली"           |
      | spaces           | "file #2.txt"      |
      | spaces           | "file ?2.pdf"      |
      | spaces           | "😀 🤖.txt"        |
      | spaces           | "new&file.txt"     |
      | spaces           | "Sample,comma.txt" |
      | spaces           | "'single'.txt"     |
      | spaces           | '"double".txt'     |

  @owncloud-issue-10339 @owncloud-issue-9568
  Scenario Outline: try to create file with '.', '..' and 'empty'
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "some text" to "<file-name>" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    Examples:
      | dav-path-version | file-name | http-status-code |
      | old              | /.        | 400              |
      | old              | /..       | 404              |
      | old              | /../lorem | 404              |
      | old              |           | 400              |
      | new              | /.        | 400              |
      | new              | /..       | 405              |
      | new              | /../lorem | 400              |
      | new              |           | 400              |
      | spaces           | /.        | 400              |
      | spaces           | /..       | 405              |
      | spaces           | /../lorem | 400              |
      | spaces           |           | 400              |

  @owncloud-issue-10339 @owncloud-issue-9568
  Scenario Outline: try to create folder with '.', '..' and 'empty'
    Given using <dav-path-version> DAV path
    When user "Alice" creates folder "<folder-name>" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    Examples:
      | dav-path-version | folder-name | http-status-code |
      | old              | /.          | 400              |
      | old              | /..         | 404              |
      | old              | /../lorem   | 404              |
      | old              |             | 400              |
      | new              | /.          | 400              |
      | new              | /..         | 405              |
      | new              | /../lorem   | 400              |
      | new              |             | 400              |
      | spaces           | /.          | 400              |
      | spaces           | /..         | 405              |
      | spaces           | /../lorem   | 404              |
      | spaces           |             | 400              |


  Scenario Outline: create a file with dots in the name
    Given using <dav-path-version> DAV path
    And user "Alice" uploads file with content "some text" to "<file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "<file-name>" for user "Alice" should be "some text"
    Examples:
      | dav-path-version | file-name |
      | old              | /fo.      |
      | old              | /fo.1     |
      | old              | /fo...1.. |
      | old              | /...      |
      | old              | /..fo     |
      | old              | /fo.xyz   |
      | old              | /fo.exe   |
      | new              | /fo.      |
      | new              | /fo.1     |
      | new              | /fo...1.. |
      | new              | /...      |
      | new              | /..fo     |
      | new              | /fo.xyz   |
      | new              | /fo.exe   |
      | spaces           | /fo.      |
      | spaces           | /fo.1     |
      | spaces           | /fo...1.. |
      | spaces           | /...      |
      | spaces           | /..fo     |
      | spaces           | /fo.xyz   |
      | spaces           | /fo.exe   |


  Scenario Outline: create a folder with dots in the name
    Given using <dav-path-version> DAV path
    When user "Alice" creates folder "<folder-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "<folder-name>" should exist
    Examples:
      | dav-path-version | folder-name |
      | old              | /fo.        |
      | old              | /fo.1       |
      | old              | /fo...1..   |
      | old              | /...        |
      | old              | /..fo       |
      | old              | /fo.xyz     |
      | old              | /fo.exe     |
      | new              | /fo.        |
      | new              | /fo.1       |
      | new              | /fo...1..   |
      | new              | /...        |
      | new              | /..fo       |
      | new              | /fo.xyz     |
      | new              | /fo.exe     |
      | spaces           | /fo.        |
      | spaces           | /fo.1       |
      | spaces           | /fo...1..   |
      | spaces           | /...        |
      | spaces           | /..fo       |
      | spaces           | /fo.xyz     |
      | spaces           | /fo.exe     |
