Feature: delete file
  As a user
  I want to be able to delete files
  So that I can remove unwanted data

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: delete a file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "to delete" to "/textfile0.txt"
    When user "Alice" deletes file "/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/textfile0.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: delete a file when 2 files exist with different case
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "to delete" to "/textfile1.txt"
    And user "Alice" has uploaded file with content "uploaded content" to "/TextFile1.txt"
    When user "Alice" deletes file "/textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/textfile1.txt" should not exist
    And as "Alice" file "/TextFile1.txt" should exist
    And the content of file "/TextFile1.txt" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: delete file from folder with dots in the path
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    And user "Alice" has uploaded file with content "uploaded content for file name with dots" to "<folder-name>/<file-name>"
    When user "Alice" deletes file "<folder-name>/<file-name>" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "<folder-name>/<file-name>" should not exist
    Examples:
      | dav-path-version | folder-name   | file-name   |
      | old              | /upload.      | abc.        |
      | old              | /upload.      | abc .       |
      | old              | /upload.1     | abc.txt     |
      | old              | /upload...1.. | abc...txt.. |
      | old              | /...          | ...         |
      | old              | /..upload     | abc         |
      | old              | /..upload     | ..abc       |
      | new              | /upload.      | abc.        |
      | new              | /upload.      | abc .       |
      | new              | /upload.1     | abc.txt     |
      | new              | /upload...1.. | abc...txt.. |
      | new              | /...          | ...         |
      | new              | /..upload     | abc         |
      | new              | /..upload     | ..abc       |
      | spaces           | /upload.      | abc.        |
      | spaces           | /upload...1.. | abc...txt.. |
      | spaces           | /upload.1     | abc.txt     |
      | spaces           | /upload.      | abc .       |
      | spaces           | /...          | ...         |
      | spaces           | /..upload     | abc         |
      | spaces           | /..upload     | ...abc      |


  Scenario Outline: delete a file with comma in the filename
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "file with comma in filename" to <file-name>
    When user "Alice" deletes file <file-name> using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file <file-name> should not exist
    Examples:
      | dav-path-version | file-name      |
      | old              | "sample,1.txt" |
      | old              | ",,,.txt"      |
      | old              | ",,,.,"        |
      | new              | "sample,1.txt" |
      | new              | ",,,.txt"      |
      | new              | ",,,.,"        |
      | spaces           | "sample,1.txt" |
      | spaces           | ",,,.txt"      |
      | spaces           | ",,,.,"        |


  Scenario Outline: delete a file with special characters in the filename
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "special file" to <file-name>
    When user "Alice" deletes file <file-name> using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file <file-name> should not exist
    Examples:
      | dav-path-version | file-name      |
      | old              | "'single'.txt" |
      | old              | '"double".txt' |
      | old              | "question?"    |
      | old              | "&and#hash"    |
      | new              | "'single'.txt" |
      | new              | '"double".txt' |
      | new              | "question?"    |
      | new              | "&and#hash"    |
      | spaces           | "'single'.txt" |
      | spaces           | '"double".txt' |
      | spaces           | "question?"    |
      | spaces           | "&and#hash"    |


  Scenario Outline: delete a hidden file
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded the following files with content "hidden file"
      | path                 |
      | .hidden_file         |
      | /FOLDER/.hidden_file |
    When user "Alice" deletes the following files
      | path                 |
      | .hidden_file         |
      | /FOLDER/.hidden_file |
    Then the HTTP status code of responses on all endpoints should be "204"
    And as "Alice" the following files should not exist
      | path                 |
      | .hidden_file         |
      | /FOLDER/.hidden_file |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: delete a file of size zero byte
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/zerobyte.txt" to "/zerobyte.txt"
    When user "Alice" deletes file "/zerobyte.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/zerobyte.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-9619
  Scenario: delete a file using file-id
    Given using spaces DAV path
    And user "Alice" has uploaded file with content "special file" to "/textfile.txt"
    And we save it into "FILEID"
    When user "Alice" deletes file "/textfile.txt" from space "Personal" using file-id "<<FILEID>>"
    Then the HTTP status code should be "204"
    And as "Alice" file "/textfile.txt" should not exist
