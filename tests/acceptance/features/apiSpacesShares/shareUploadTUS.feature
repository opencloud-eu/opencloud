Feature: upload resources on share using TUS protocol
  As a user
  I want to be able to upload files
  So that I can store and share files between multiple client systems

  Background:
    Given using spaces DAV path
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario: upload file with mtime to a received share
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file "filesForUpload/textfile.txt" to "toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" folder "toShare" of the space "Shares" should contain these entries:
      | file.txt |
    And as "Brian" the mtime of the file "/toShare/file.txt" in space "Shares" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Alice" the mtime of the file "/toShare/file.txt" in space "Personal" should be "Thu, 08 Aug 2012 04:18:13 GMT"


  Scenario: upload file with mtime to a sent share
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Alice" uploads a file "filesForUpload/textfile.txt" to "toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" via TUS inside of the space "Personal" using the WebDAV API
    Then for user "Alice" folder "toShare" of the space "Personal" should contain these entries:
      | file.txt |
    And as "Alice" the mtime of the file "/toShare/file.txt" in space "Personal" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Brian" the mtime of the file "/toShare/file.txt" in space "Shares" should be "Thu, 08 Aug 2012 04:18:13 GMT"


  Scenario: overwriting a file with mtime in a received share
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    And user "Alice" has uploaded file with content "uploaded content" to "/toShare/file.txt"
    When user "Brian" uploads a file "filesForUpload/textfile.txt" to "toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" folder "toShare" of the space "Shares" should contain these entries:
      | file.txt |
    And as "Brian" the mtime of the file "/toShare/file.txt" in space "Shares" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Alice" the mtime of the file "/toShare/file.txt" in space "Personal" should be "Thu, 08 Aug 2012 04:18:13 GMT"


  Scenario: overwriting a file with mtime in a sent share
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    And user "Brian" has uploaded a file inside space "Shares" with content "uploaded content" to "toShare/file.txt"
    When user "Alice" uploads a file "filesForUpload/textfile.txt" to "toShare/file.txt" with mtime "Thu, 08 Aug 2012 04:18:13 GMT" via TUS inside of the space "Personal" using the WebDAV API
    Then for user "Alice" folder "toShare" of the space "Personal" should contain these entries:
      | file.txt |
    And as "Alice" the mtime of the file "/toShare/file.txt" in space "Personal" should be "Thu, 08 Aug 2012 04:18:13 GMT"
    And as "Brian" the mtime of the file "/toShare/file.txt" in space "Shares" should be "Thu, 08 Aug 2012 04:18:13 GMT"


  Scenario: attempt to upload a file into a nonexistent folder within correctly received share
    Given using OCS API version "1"
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "uploaded content" to "/toShare/nonExistentFolder/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" folder "toShare" of the space "Shares" should not contain these entries:
      | nonExistentFolder |


  Scenario: attempt to upload a file into a nonexistent folder within correctly received read only share
    Given using OCS API version "1"
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "uploaded content" to "/toShare/nonExistentFolder/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" folder "toShare" of the space "Shares" should not contain these entries:
      | nonExistentFolder |


  Scenario: uploading a file to a received share folder
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "uploaded content" to "/toShare/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Alice" folder "toShare" of the space "Personal" should contain these entries:
      | file.txt |
    And for user "Alice" the content of the file "toShare/file.txt" of the space "Personal" should be "uploaded content"


  Scenario: uploading a file to a user read/write share folder
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "uploaded content" to "/toShare/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Alice" folder "toShare" of the space "Personal" should contain these entries:
      | file.txt |
    And for user "Alice" the content of the file "toShare/file.txt" of the space "Personal" should be "uploaded content"


  Scenario: uploading a file into a group share as a share receiver
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Uploader |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "uploaded content" to "/toShare/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Alice" folder "toShare" of the space "Personal" should contain these entries:
      | file.txt |
    And for user "Alice" the content of the file "toShare/file.txt" of the space "Personal" should be "uploaded content"


  Scenario: overwrite file to a received share folder
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has uploaded file with content "original content" to "/toShare/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "overwritten content" to "/toShare/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Alice" folder "toShare" of the space "Personal" should contain these entries:
      | file.txt |
    And for user "Alice" the content of the file "toShare/file.txt" of the space "Personal" should be "overwritten content"


  Scenario: attempt to upload a file into a folder within correctly received read only share
    Given user "Alice" has created folder "/toShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShare  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "toShare" synced
    When user "Brian" uploads a file with content "uploaded content" to "/toShare/file.txt" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Brian" folder "toShare" of the space "Shares" should not contain these entries:
      | file.txt |


  Scenario: upload a file to shared folder with checksum should return the checksum in the propfind for sharee
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                                     |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
      | Tus-Resumable   | 1.0.0                                 |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" via TUS inside of the space "Personal" using the WebDAV API
    When user "Brian" requests the checksum of file "/FOLDER/textFile.txt" in space "Shares" via propfind using the WebDAV API
    Then the HTTP status code should be "207"
    And the webdav checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964 MD5:827ccb0eea8a706c4c34a16891f84e7b ADLER32:02f80100"


  Scenario: upload a file to shared folder with checksum should return the checksum in the download header for sharee
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                                     |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
      | Tus-Resumable   | 1.0.0                                 |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" via TUS inside of the space "Personal" using the WebDAV API
    When user "Brian" downloads the file "/FOLDER/textFile.txt" of the space "Shares" using the WebDAV API
    Then the header checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964"


  Scenario: sharer shares a file with correct checksum should return the checksum in the propfind for sharee
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" requests the checksum of file "/textFile.txt" in space "Shares" via propfind using the WebDAV API
    Then the HTTP status code should be "207"
    And the webdav checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964 MD5:827ccb0eea8a706c4c34a16891f84e7b ADLER32:02f80100"


  Scenario: sharer shares a file with correct checksum should return the checksum in the download header for sharee
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" downloads the file "/textFile.txt" of the space "Shares" using the WebDAV API
    Then the header checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964"


  Scenario: sharee uploads a file to a received share folder with correct checksum
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" creates a new TUS resource for the space "Shares" with content "" using the WebDAV API with these headers:
      | Upload-Length   | 5                                     |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
      | Tus-Resumable   | 1.0.0                                 |
    And user "Brian" uploads file with checksum "MD5 827ccb0eea8a706c4c34a16891f84e7b" to the last created TUS Location with offset "0" and content "12345" via TUS inside of the space "Shares" using the WebDAV API
    Then for user "Alice" folder "FOLDER" of the space "Personal" should contain these entries:
      | textFile.txt |
    And for user "Alice" the content of the file "FOLDER/textFile.txt" of the space "Personal" should be "12345"

  @owncloud-issue-1755
  Scenario: sharee uploads a file to a received share folder with wrong checksum should not work
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" creates a new TUS resource for the space "Shares" with content "" using the WebDAV API with these headers:
      | Upload-Length   | 5                                     |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
      | Tus-Resumable   | 1.0.0                                 |
    And user "Brian" uploads file with checksum "MD5 827ccb0eea8a706c4c34a16891f84e7b" to the last created TUS Location with offset "0" and content "12346" via TUS inside of the space "Shares" using the WebDAV API
    Then the HTTP status code should be "460"
    And for user "Alice" folder "FOLDER" of the space "Personal" should not contain these entries:
      | textFile.txt |

  @owncloud-issue-1755
  Scenario: sharer uploads a file to shared folder with wrong checksum should not work
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 16                                    |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
      | Tus-Resumable   | 1.0.0                                 |
    When user "Alice" uploads file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12346" via TUS inside of the space "Personal" using the WebDAV API
    Then the HTTP status code should be "460"
    And for user "Alice" folder "FOLDER" of the space "Personal" should not contain these entries:
      | textFile.txt |
    And for user "Brian" folder "FOLDER" of the space "Shares" should not contain these entries:
      | textFile.txt |


  Scenario: sharer uploads a chunked file with correct checksum and share it with sharee should work
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 10                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    When user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "01234" with checksum "MD5 4100c4d44da9177247e44a5fc1546778" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" sends a chunk to the last created TUS Location with offset "5" and data "56789" with checksum "MD5 099ebea48ea9666a7da2177267983138" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" shares file "textFile.txt" with user "Brian" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "100"
    And for user "Brian" the content of the file "/textFile.txt" of the space "Shares" should be "0123456789"


  Scenario: sharee uploads a chunked file with correct checksum to a received share folder should work
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has created a new TUS resource in the space "Shares" with the following headers:
      | Upload-Length   | 10                                    |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
      | Tus-Resumable   | 1.0.0                                 |
    When user "Brian" sends a chunk to the last created TUS Location with offset "0" and data "01234" with checksum "MD5 4100c4d44da9177247e44a5fc1546778" via TUS inside of the space "Shares" using the WebDAV API
    And user "Brian" sends a chunk to the last created TUS Location with offset "5" and data "56789" with checksum "MD5 099ebea48ea9666a7da2177267983138" via TUS inside of the space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" folder "FOLDER" of the space "Personal" should contain these entries:
      | textFile.txt |
    And for user "Alice" the content of the file "/FOLDER/textFile.txt" of the space "Personal" should be "0123456789"


  Scenario: sharer uploads a file with checksum and as a sharee overwrites the shared file with new data and correct checksum
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 16                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    And user "Alice" has uploaded file with checksum "SHA1 c1dab0c0864b6ac9bdd3743a1408d679f1acd823" to the last created TUS Location with offset "0" and content "original content" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" overwrites recently shared file with offset "0" and data "overwritten content" with checksum "SHA1 fe990d2686a0fc86004efc31f5bf2475a45d4905" via TUS inside of the space "Shares" using the WebDAV API with these headers:
      | Upload-Length   | 19                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    Then the HTTP status code should be "204"
    And for user "Alice" the content of the file "/textFile.txt" of the space "Personal" should be "overwritten content"

  @owncloud-issue-1755
  Scenario: sharer uploads a file with checksum and as a sharee overwrites the shared file with new data and invalid checksum
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 16                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    And user "Alice" has uploaded file with checksum "SHA1 c1dab0c0864b6ac9bdd3743a1408d679f1acd823" to the last created TUS Location with offset "0" and content "original content" via TUS inside of the space "Personal" using the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" overwrites recently shared file with offset "0" and data "overwritten content" with checksum "SHA1 fe990d2686a0fc86004efc31f5bf2475a45d4906" via TUS inside of the space "Shares" using the WebDAV API with these headers:
      | Upload-Length   | 19                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    Then the HTTP status code should be "460"
    And for user "Alice" the content of the file "/textFile.txt" of the space "Personal" should be "original content"
