@skipOnReva
Feature: previews of files downloaded through the webdav API
  As a user
  I want to be able to download the preview of the files
  So that I can view the contents of the files

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: download previews with invalid width
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    When user "Alice" downloads the preview of "/parent.txt" with width "<width>" and height "32" using the WebDAV API
    Then the HTTP status code should be "400"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "Cannot set width of 0 or smaller!"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\BadRequest"
    Examples:
      | dav-path-version | width |
      | old              | 0     |
      | old              | 0.5   |
      | old              | -1    |
      | old              | false |
      | old              | true  |
      | old              | A     |
      | old              | %2F   |
      | new              | 0     |
      | new              | 0.5   |
      | new              | -1    |
      | new              | false |
      | new              | true  |
      | new              | A     |
      | new              | %2F   |
      | spaces           | 0     |
      | spaces           | 0.5   |
      | spaces           | -1    |
      | spaces           | false |
      | spaces           | true  |
      | spaces           | A     |
      | spaces           | %2F   |


  Scenario Outline: download previews with invalid height
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    When user "Alice" downloads the preview of "/parent.txt" with width "32" and height "<height>" using the WebDAV API
    Then the HTTP status code should be "400"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "Cannot set height of 0 or smaller!"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\BadRequest"
    Examples:
      | dav-path-version | height |
      | old              | 0      |
      | old              | 0.5    |
      | old              | -1     |
      | old              | false  |
      | old              | true   |
      | old              | A      |
      | old              | %2F    |
      | new              | 0      |
      | new              | 0.5    |
      | new              | -1     |
      | new              | false  |
      | new              | true   |
      | new              | A      |
      | new              | %2F    |
      | spaces           | 0      |
      | spaces           | 0.5    |
      | spaces           | -1     |
      | spaces           | false  |
      | spaces           | true   |
      | spaces           | A      |
      | spaces           | %2F    |


  Scenario Outline: download previews of files inside sub-folders
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "subfolder"
    And user "Alice" has uploaded file "filesForUpload/example.gif" to "subfolder/example.gif"
    When user "Alice" downloads the preview of "subfolder/example.gif" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "32" pixels wide and "32" pixels high
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: download previews of file types that don't support preview
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/<file-name>" to "/<file-name>"
    When user "Alice" downloads the preview of "/<file-name>" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "404"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\NotFound"
    Examples:
      | dav-path-version | file-name    |
      | old              | simple.pdf   |
      | old              | simple.odt   |
      | old              | new-data.zip |
      | new              | simple.pdf   |
      | new              | simple.odt   |
      | new              | new-data.zip |
      | spaces           | simple.pdf   |
      | spaces           | simple.odt   |
      | spaces           | new-data.zip |


  Scenario Outline: download previews of different image file types
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/<image-name>" to "/<image-name>"
    When user "Alice" downloads the preview of "/<image-name>" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "32" pixels wide and "32" pixels high
    Examples:
      | dav-path-version | image-name     |
      | old              | testavatar.jpg |
      | old              | testavatar.png |
      | new              | testavatar.jpg |
      | new              | testavatar.png |
      | spaces           | testavatar.jpg |
      | spaces           | testavatar.png |


  Scenario Outline: download previews of image after renaming it
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/testimage.jpg"
    And user "Alice" has moved file "/testimage.jpg" to "/testimage.txt"
    When user "Alice" downloads the preview of "/testimage.txt" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "32" pixels wide and "32" pixels high
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: download previews of shared files (to shares folder)
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/<file-name>" to "/<file-name>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <file-name> |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Brian" has a share "<file-name>" synced
    When user "Brian" downloads the preview of shared resource "/Shares/<file-name>" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "32" pixels wide and "32" pixels high
    Examples:
      | dav-path-version | file-name   |
      | old              | lorem.txt   |
      | old              | example.gif |
      | new              | lorem.txt   |
      | new              | example.gif |
      | spaces           | lorem.txt   |
      | spaces           | example.gif |


  Scenario Outline: user tries to download previews of other users files
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    When user "Brian" downloads the preview of "/parent.txt" of "Alice" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "404"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "File with name parent.txt could not be located"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\NotFound"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: download previews of folders
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "subfolder"
    When user "Alice" downloads the preview of "/subfolder/" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "400"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "Unsupported file type"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\BadRequest"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to download previews of nonexistent files
    Given using <dav-path-version> DAV path
    When user "Alice" tries to download the preview of nonexistent file "/parent.txt" with width "32" and height "32" using the WebDAV API
    Then the HTTP status code should be "404"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "File with name parent.txt could not be located"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\NotFound"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: preview content changes with the change in file content
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    And user "Alice" has downloaded the preview of "/parent.txt" with width "32" and height "32"
    When user "Alice" uploads file with content "this is a file to upload" to "/parent.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the preview of "/parent.txt" with width "32" and height "32" should have been changed
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-2538
  Scenario Outline: when owner updates a shared file, previews for sharee are also updated (to shared folder)
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent.txt  |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Brian" has a share "parent.txt" synced
    And user "Brian" has downloaded the preview of shared resource "/Shares/parent.txt" with width "32" and height "32"
    When user "Alice" uploads file with content "this is a file to upload" to "/parent.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Brian" the preview of shared resource "/Shares/parent.txt" with width "32" and height "32" should have been changed
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @need-build-with-VIPS @issue-1451
  Scenario Outline: it should update the preview content if the file content is updated (content with UTF chars)
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/lorem.txt"
    And user "Alice" has uploaded file with content "ओनक्लाउड फाएल शेरिङ्ग एन्ड सिन्किङ" to "/lorem.txt"
    When user "Alice" downloads the preview of "/lorem.txt" with width "1200" and height "1200" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded preview content should match with "unicode-fixture.png" fixtures preview content
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @need-build-with-VIPS @issue-1452
  Scenario Outline: download preview of the text file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/lorem.txt"
    When user "Alice" downloads the preview of "/lorem.txt" with width "1200" and height "1200" and processor "thumbnail" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "1200" pixels wide and "1200" pixels high
    And the downloaded preview content should match with "text-file-fixture.png" fixtures preview content
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: updates to a file should change the preview for both sharees and sharers
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file with content "file to upload" to "/FOLDER/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has downloaded the preview of "/FOLDER/lorem.txt" with width "32" and height "32"
    And user "Brian" has downloaded the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32"
    When user "Alice" uploads file "filesForUpload/lorem.txt" to "/FOLDER/lorem.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the preview of "/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    And as user "Brian" the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    When user "Brian" uploads file with content "new uploaded content" to shared resource "Shares/FOLDER/lorem.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the preview of "/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    And as user "Brian" the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: updates to a group shared file should change the preview for both sharees and sharers
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been created with default attributes
    And user "Carol" has been created with default attributes
    And user "Brian" has been added to group "grp1"
    And user "Carol" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file with content "file to upload" to "/FOLDER/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Carol" has a share "FOLDER" synced
    And user "Alice" has downloaded the preview of "/FOLDER/lorem.txt" with width "32" and height "32"
    And user "Brian" has downloaded the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32"
    And user "Carol" has downloaded the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32"
    When user "Alice" uploads file "filesForUpload/lorem.txt" to "/FOLDER/lorem.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the preview of "/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    And as user "Brian" the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    And as user "Carol" the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    When user "Brian" uploads file with content "new uploaded content" to shared resource "Shares/FOLDER/lorem.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the preview of "/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    And as user "Brian" the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    And as user "Carol" the preview of shared resource "Shares/FOLDER/lorem.txt" with width "32" and height "32" should have been changed
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: try to download preview of an image with preview set to 0
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/testavatar.jpg"
    When user "Alice" tries to download the preview of "/testavatar.jpg" with width "32" and height "32" and preview set to 0 using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be "1402" pixels wide and "500" pixels high
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: try to download preview of a text file with preview set to 0
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "to preview" to "textfile.txt"
    When user "Alice" tries to download the preview of "textfile.txt" with width "32" and height "32" and preview set to 0 using the WebDAV API
    Then the HTTP status code should be "200"
    And the content in the response should match the following content:
      """
      to preview
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @need-build-with-VIPS
  Scenario Outline: download previews of an image with different processors
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/testimage.jpg"
    When user "Alice" downloads the preview of "/testimage.jpg" with width "32" and height "32" and processor <processor> using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded preview content should match with <expected-image> fixtures preview content
    Examples:
      | dav-path-version | processor | expected-image |
      | old              | fit       | fit.png        |
      | old              | fill      | fill.png       |
      | old              | resize    | resize.png     |
      | old              | thumbnail | thumbnail.png  |
      | new              | fit       | fit.png        |
      | new              | fill      | fill.png       |
      | new              | resize    | resize.png     |
      | new              | thumbnail | thumbnail.png  |
      | spaces           | fit       | fit.png        |
      | spaces           | fill      | fill.png       |
      | spaces           | resize    | resize.png     |
      | spaces           | thumbnail | thumbnail.png  |

  @issue-10589 @env-config
  Scenario Outline: try to download an image preview when the maximum thumbnail input value in the environment is set to a small value
    Given the following configs have been set:
      | config                               | value |
      | THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE | 1KB   |
      | THUMBNAILS_MAX_INPUT_WIDTH           | 200   |
      | THUMBNAILS_MAX_INPUT_HEIGHT          | 200   |
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "/testimage.png"
    When user "Alice" downloads the preview of "/testimage.png" with width "36" and height "36" and processor thumbnail using the WebDAV API
    Then the HTTP status code should be "403"
    And the value of the item "/d:error/s:message" in the response should be "thumbnails: image is too large"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10589 @env-config
  Scenario Outline: try to download a file preview when the maximum thumbnail input value in the environment is set to a small value
    Given the following configs have been set:
      | config                               | value |
      | THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE | 1KB   |
      | THUMBNAILS_MAX_INPUT_WIDTH           | 200   |
      | THUMBNAILS_MAX_INPUT_HEIGHT          | 200   |
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem-big.txt" to "/lorem-big.txt"
    When user "Alice" downloads the preview of "/lorem-big.txt" with width "36" and height "36" and processor thumbnail using the WebDAV API
    Then the HTTP status code should be "403"
    And the value of the item "/d:error/s:message" in the response should be "thumbnails: image is too large"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
