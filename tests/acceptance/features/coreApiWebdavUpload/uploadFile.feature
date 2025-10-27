Feature: upload file
  As a user
  I want to be able to upload files
  So that I can store and share files between multiple client systems

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes

  @smokeTest
  Scenario Outline: upload a file and check etag and download content
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "<file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And the following headers should match these regular expressions for user "Alice"
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    And the content of file "<file-name>" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | file-name         |
      | old              | /upload.txt       |
      | old              | /नेपाली.txt       |
      | old              | /strängé file.txt |
      | old              | /s,a,m,p,l,e.txt  |
      | new              | /upload.txt       |
      | new              | /नेपाली.txt       |
      | new              | /strängé file.txt |
      | new              | /s,a,m,p,l,e.txt  |
      | spaces           | /upload.txt       |
      | spaces           | /नेपाली.txt       |
      | spaces           | /strängé file.txt |
      | spaces           | /s,a,m,p,l,e.txt  |


  Scenario Outline: upload a file and check download content
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to <file-name> using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file <file-name> for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | file-name      |
      | old              | "C++ file.cpp" |
      | old              | "file #2.txt"  |
      | new              | "C++ file.cpp" |
      | new              | "file #2.txt"  |
      | spaces           | "C++ file.cpp" |
      | spaces           | "file #2.txt"  |

  @owncloud-issue-1259
  #after fixing all issues delete this Scenario and merge with the one above
  Scenario Outline: upload a file with character '?' in its name and check download content
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to <file-name> using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file <file-name> for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | file-name           |
      | old              | "file ?2.txt"       |
      | old              | " ?fi=le&%#2 . txt" |
      | old              | " # %ab ab?=ed "    |
      | new              | "file ?2.txt"       |
      | new              | " ?fi=le&%#2 . txt" |
      | new              | " # %ab ab?=ed "    |
      | spaces           | "file ?2.txt"       |
      | spaces           | " ?fi=le&%#2 . txt" |
      | spaces           | " # %ab ab?=ed "    |


  Scenario Outline: upload a file with comma in the filename and check download content
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "file with comma" to <file-name> using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file <file-name> for user "Alice" should be "file with comma"
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


  Scenario Outline: upload a file into a folder and check download content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    When user "Alice" uploads file with content "uploaded content" to "<folder-name>/<file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "<folder-name>/<file-name>" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | folder-name                      | file-name                     |
      | old              | /upload                          | abc.txt                       |
      | old              | /strängé folder                  | strängé file.txt              |
      | old              | /C++ folder                      | C++ file.cpp                  |
      | old              | /नेपाली                          | नेपाली                        |
      | old              | /folder #2.txt                   | file #2.txt                   |
      | new              | /upload                          | abc.txt                       |
      | new              | /strängé folder (duplicate #2 &) | strängé file (duplicate #2 &) |
      | new              | /C++ folder                      | C++ file.cpp                  |
      | new              | /नेपाली                          | नेपाली                        |
      | new              | /folder #2.txt                   | file #2.txt                   |
      | spaces           | /strängé folder                  | strängé file.txt              |
      | spaces           | /upload                          | abc.txt                       |
      | spaces           | /C++ folder                      | C++ file.cpp                  |
      | spaces           | /नेपाली                          | नेपाली                        |
      | spaces           | /folder #2.txt                   | file #2.txt                   |

  @owncloud-issue-1259
    #after fixing all issues delete this Scenario and merge with the one above
  Scenario Outline: upload a file into a folder with character '?' in its name and check download content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    When user "Alice" uploads file with content "uploaded content" to "<folder-name>/<file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "<folder-name>/<file-name>" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | folder-name       | file-name    |
      | old              | /folder ?2.txt    | file ?2.txt  |
      | old              | /?fi=le&%#2 . txt | # %ab ab?=ed |
      | new              | /folder ?2.txt    | file ?2.txt  |
      | new              | /?fi=le&%#2 . txt | # %ab ab?=ed |
      | spaces           | /folder ?2.txt    | file ?2.txt  |
      | spaces           | /?fi=le&%#2 . txt | # %ab ab?=ed |


  Scenario Outline: attempt to upload a file into a nonexistent folder
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "nonexistent-folder/new-file.txt" using the WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" folder "nonexistent-folder" should not exist
    And as "Alice" file "nonexistent-folder/new-file.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1345 @owncloud-issue-2177
  Scenario Outline: uploading file to path with extension .part should not be possible
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/textfile.part" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should see the following elements
      | /textfile.part |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a file into a folder with dots in the path and check download content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    When user "Alice" uploads file with content "uploaded content for file name ending with a dot" to "<folder-name>/<file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/<folder-name>/<file-name>" should exist
    And the content of file "<folder-name>/<file-name>" for user "Alice" should be "uploaded content for file name ending with a dot"
    Examples:
      | dav-path-version | folder-name   | file-name   |
      | old              | /upload.      | abc.        |
      | old              | /upload.      | abc .       |
      | old              | /upload.1     | abc.txt     |
      | old              | /upload...1.. | abc...txt.. |
      | old              | /...          | ...         |
      | new              | /..upload     | ..abc       |
      | new              | /upload.      | abc.        |
      | new              | /upload.      | abc .       |
      | new              | /upload.1     | abc.txt     |
      | new              | /upload...1.. | abc...txt.. |
      | new              | /...          | ...         |
      | spaces           | /upload.      | abc.        |
      | spaces           | /upload.      | abc .       |
      | spaces           | /upload.1     | abc.txt     |
      | spaces           | /upload...1.. | abc...txt.. |
      | spaces           | /...          | ...         |


  Scenario Outline: upload file with mtime
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "file.txt" should exist
    And as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1248
  Scenario Outline: upload an empty file with mtime
    Given using <dav_version> DAV path
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "file.txt" should exist
    And as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav_version |
      | old         |
      | new         |
      | spaces      |


  Scenario Outline: upload a file with mtime in a folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/testFolder/file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/testFolder/file.txt" should exist
    And as "Alice" the mtime of the file "/testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1248
  Scenario Outline: upload an empty file with mtime in a folder
    Given using <dav_version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "/testFolder/file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/testFolder/file.txt" should exist
    And as "Alice" the mtime of the file "/testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav_version |
      | old         |
      | new         |
      | spaces      |


  Scenario Outline: moving a file does not change its mtime
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the WebDAV API
    And user "Alice" moves file "file.txt" to "/testFolder/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/testFolder/file.txt" should exist
    And as "Alice" the mtime of the file "/testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: overwriting a file changes its mtime
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "first time upload content" to "file.txt"
    When user "Alice" uploads a file with content "Overwrite file" and mtime "Thu, 08 Aug 2019 04:18:13 GMT" to "file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "file.txt" should exist
    And as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    And the content of file "file.txt" for user "Alice" should be "Overwrite file"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a hidden file and check download content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    When user "Alice" uploads the following files with content "hidden file"
      | path                 |
      | .hidden_file         |
      | /FOLDER/.hidden_file |
    Then the HTTP status code of responses on all endpoints should be "201"
    And as "Alice" the following files should exist
      | path                 |
      | .hidden_file         |
      | /FOLDER/.hidden_file |
    And the content of the following files for user "Alice" should be "hidden file"
      | path                 |
      | .hidden_file         |
      | /FOLDER/.hidden_file |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a file of size zero byte
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "/zerobyte.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "zerobyte.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-7257
  Scenario Outline: user updates a file with empty content
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "file with content" to "/textfile.txt"
    When user "Alice" uploads file with content "" to "/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-7257
  Scenario Outline: user updates a file inside a folder with empty content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    And user "Alice" has uploaded file with content "file with content" to "testFolder/textfile.txt"
    When user "Alice" uploads file with content "" to "testFolder/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "testFolder/textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @owncloud-issue-7257
  Scenario Outline: user updates a shared file with empty content
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "file with content" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textfile.txt" synced
    When user "Brian" uploads file with content "" to shared resource "Shares/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Brian" the content of the file "/test.txt" of the space "Shares" should be ""
    And the content of file "/textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @owncloud-issue-7257
  Scenario: user updates a file inside a project space with empty content
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "file with content" to "textfile.txt"
    When user "Alice" uploads a file inside space "new-space" with content "" to "textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" the content of the file "/textfile.txt" of the space "new-space" should be ""

  @skipOnReva @owncloud-issue-7257
  Scenario: user updates a file inside a shared space with empty content
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "file with content" to "textfile.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    When user "Brian" uploads a file inside space "new-space" with content "" to "textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Brian" the content of the file "/textfile.txt" of the space "new-space" should be ""
    And for user "Alice" the content of the file "/textfile.txt" of the space "new-space" should be ""

  @skipOnReva @owncloud-issue-8699 @owncloud-issue-10331
  Scenario: user updates a file inside a link shared space with empty content
    Given using SharingNG
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | permissionsRole | edit      |
      | password        | %public%  |
    When the public uploads file "test.txt" with password "%public%" and content "test-file" using the public WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the content of the file "/test.txt" of the space "new-space" should be "test-file"

  @issue-10496
  Scenario Outline: upload a file with the same mtime and different content multiple times (atleast 3 times)
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT"
    And user "Alice" has uploaded file "filesForUpload/davtest.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT"
    When user "Alice" uploads file "filesForUpload/lorem.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" the mtime of the file "file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    And the version folder of file "file.txt" for user "Alice" should contain "2" element
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
