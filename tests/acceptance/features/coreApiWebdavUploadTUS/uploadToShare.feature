@skipOnReva
Feature: upload file to shared folder
  As a user
  I want to upload files on a shared folder
  So that other user with access on the shared folder can access the resource

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |

  @issue-10346
  Scenario Outline: uploading file to a received share folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file with content "uploaded content" to "/Shares/FOLDER/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Alice" file "/FOLDER/textfile.txt" should exist
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-10346
  Scenario Outline: uploading file to a user read/write share folder works
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file with content "uploaded content" to "/Shares/FOLDER/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Alice" file "/FOLDER/textfile.txt" should exist
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-10346
  Scenario Outline: uploading a file into a group share as share receiver
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file with content "uploaded content" to "/Shares/FOLDER/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Alice" file "/FOLDER/textfile.txt" should exist
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-10346
  Scenario Outline: overwrite file to a received share folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded file with content "original content" to "/FOLDER/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file with content "overwritten content" to "/Shares/FOLDER/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Alice" file "/FOLDER/textfile.txt" should exist
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be "overwritten content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-10346
  Scenario Outline: attempt to upload a file into a folder within correctly received read only share
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads file with content "uploaded content" to "/Shares/FOLDER/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Brian" file "/Shares/FOLDER/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a file to shared folder with checksum should return the checksum in the propfind for sharee
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 5                                     |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" using the TUS protocol on the WebDAV API
    When user "Brian" requests the checksum of "/Shares/FOLDER/textFile.txt" via propfind
    Then the HTTP status code should be "207"
    And the webdav checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964 MD5:827ccb0eea8a706c4c34a16891f84e7b ADLER32:02f80100"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a file to shared folder with checksum should return the checksum in the download header for sharee
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 5                                     |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d069ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" using the TUS protocol on the WebDAV API
    When user "Brian" downloads file "/Shares/FOLDER/textFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the header checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharer shares a file with correct checksum should return the checksum in the propfind for sharee
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" using the TUS protocol on the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" requests the checksum of "/Shares/textFile.txt" via propfind
    Then the HTTP status code should be "207"
    And the webdav checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964 MD5:827ccb0eea8a706c4c34a16891f84e7b ADLER32:02f80100"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharer shares a file with correct checksum should return the checksum in the download header for sharee
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
    And user "Alice" has uploaded file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513964" to the last created TUS Location with offset "0" and content "12345" using the TUS protocol on the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" downloads file "/Shares/textFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the header checksum should match "SHA1:8cb2237d0679ca88db6464eac60da96345513964"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharee uploads a file to a received share folder with correct checksum
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has created a new TUS resource on the WebDAV API with these headers:
      | Tus-Resumable   | 1.0.0                                         |
      | Upload-Length   | 16                                            |
      #    L1NoYXJlcy9GT0xERVIvdGV4dEZpbGUudHh0 is the base64 encode of /Shares/FOLDER/textFile.txt
      | Upload-Metadata | filename L1NoYXJlcy9GT0xERVIvdGV4dEZpbGUudHh0 |
    When user "Brian" uploads file with checksum "MD5 8a4e0407dcda7872d44dada38887b8ae" to the last created TUS Location with offset "0" and content "uploaded content" using the TUS protocol on the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "FOLDER/textFile.txt" for user "Alice" should be "uploaded content"
    And the content of file "Shares/FOLDER/textFile.txt" for user "Brian" should be "uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-1755
  Scenario Outline: sharee uploads a file to a received share folder with wrong checksum should not work
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has created a new TUS resource on the WebDAV API with these headers:
      | Tus-Resumable   | 1.0.0                                         |
      | Upload-Length   | 16                                            |
      #    L1NoYXJlcy9GT0xERVIvdGV4dEZpbGUudHh0 is the base64 encode of /Shares/FOLDER/textFile.txt
      | Upload-Metadata | filename L1NoYXJlcy9GT0xERVIvdGV4dEZpbGUudHh0 |
    And user "Brian" uploads file with checksum "MD5 827ccb0eea8a706c4c34a16891f84e8c" to the last created TUS Location with offset "0" and content "uploaded content" using the TUS protocol on the WebDAV API
    Then the HTTP status code should be "460"
    And as "Alice" file "/FOLDER/textFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-1755
  Scenario Outline: sharer uploads a file to shared folder with wrong checksum should not work
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 16                                    |
      #    L0ZPTERFUi90ZXh0RmlsZS50eHQ= is the base64 encode of /FOLDER/textFile.txt
      | Upload-Metadata | filename L0ZPTERFUi90ZXh0RmlsZS50eHQ= |
    When user "Alice" uploads file with checksum "SHA1 8cb2237d0679ca88db6464eac60da96345513954" to the last created TUS Location with offset "0" and content "uploaded content" using the TUS protocol on the WebDAV API
    Then the HTTP status code should be "460"
    And as "Alice" file "/FOLDER/textFile.txt" should not exist
    And as "Brian" file "/Shares/FOLDER/textFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |


  Scenario Outline: sharer uploads a chunked file with correct checksum and share it with sharee should work
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 10                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
    When user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "01234" with checksum "MD5 4100c4d44da9177247e44a5fc1546778" using the TUS protocol on the WebDAV API
    And user "Alice" sends a chunk to the last created TUS Location with offset "5" and data "56789" with checksum "MD5 099ebea48ea9666a7da2177267983138" using the TUS protocol on the WebDAV API
    And user "Alice" shares file "textFile.txt" with user "Brian" using the sharing API
    Then the HTTP status code should be "200"
    And user "Brian" has a share "textFile.txt" synced
    And the content of file "/Shares/textFile.txt" for user "Brian" should be "0123456789"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharee uploads a chunked file with correct checksum to a received share folder should work
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has created a new TUS resource on the WebDAV API with these headers:
      | Tus-Resumable   | 1.0.0                                         |
      | Upload-Length   | 10                                            |
      #    L1NoYXJlcy9GT0xERVIvdGV4dEZpbGUudHh0 is the base64 encode of /Shares/FOLDER/textFile.txt
      | Upload-Metadata | filename L1NoYXJlcy9GT0xERVIvdGV4dEZpbGUudHh0 |
    When user "Brian" sends a chunk to the last created TUS Location with offset "0" and data "01234" with checksum "MD5 4100c4d44da9177247e44a5fc1546778" using the TUS protocol on the WebDAV API
    And user "Brian" sends a chunk to the last created TUS Location with offset "5" and data "56789" with checksum "MD5 099ebea48ea9666a7da2177267983138" using the TUS protocol on the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/FOLDER/textFile.txt" for user "Alice" should be "0123456789"
    And the content of file "Shares/FOLDER/textFile.txt" for user "Brian" should be "0123456789"
    Examples:
      | dav-path-version |
      | old              |
      | new              |


  Scenario Outline: sharer uploads a file with checksum and as a sharee overwrites the shared file with new data and correct checksum
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 16                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
    And user "Alice" has uploaded file with checksum "SHA1 c1dab0c0864b6ac9bdd3743a1408d679f1acd823" to the last created TUS Location with offset "0" and content "original content" using the TUS protocol on the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" overwrites recently shared file with offset "0" and data "overwritten content" with checksum "SHA1 fe990d2686a0fc86004efc31f5bf2475a45d4905" using the TUS protocol on the WebDAV API with these headers:
      | Upload-Length   | 19                                    |
        #    dGV4dEZpbGUudHh0 is the base64 encode of /Shares/textFile.txt
      | Upload-Metadata | filename L1NoYXJlcy90ZXh0RmlsZS50eHQ= |
    Then the HTTP status code should be "204"
    And the content of file "/textFile.txt" for user "Alice" should be "overwritten content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @issue-1755
  Scenario Outline: sharer uploads a file with checksum and as a sharee overwrites the shared file with new data and invalid checksum
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 16                        |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
    And user "Alice" has uploaded file with checksum "SHA1 c1dab0c0864b6ac9bdd3743a1408d679f1acd823" to the last created TUS Location with offset "0" and content "original content" using the TUS protocol on the WebDAV API
    And user "Alice" has sent the following resource share invitation:
      | resource        | textFile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textFile.txt" synced
    When user "Brian" overwrites recently shared file with offset "0" and data "overwritten content" with checksum "SHA1 fe990d2686a0fc86004efc31f5bf2475a45d4906" using the TUS protocol on the WebDAV API with these headers:
      | Upload-Length   | 19                                    |
      #    dGV4dEZpbGUudHh0 is the base64 encode of /Shares/textFile.txt
      | Upload-Metadata | filename L1NoYXJlcy90ZXh0RmlsZS50eHQ= |
    Then the HTTP status code should be "460"
    And the content of file "/textFile.txt" for user "Alice" should be "original content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
