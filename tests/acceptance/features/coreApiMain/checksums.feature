Feature: checksums
  As a user
  I want to upload files with checksum
  So that I can make sure that the files are uploaded with correct checksums

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: uploading a file with checksum should work
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a" using the WebDAV API
    Then the HTTP status code should be "201"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest @owncloud-issue-1291
  Scenario Outline: uploading a file with checksum should return the checksum in the propfind
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    When user "Alice" requests the checksum of "/myChecksumFile.txt" via propfind
    Then the webdav checksum should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f MD5:d70b40f177b14b470d1756a3c12b963a ADLER32:8ae90960"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest @owncloud-issue-1316
  Scenario Outline: uploading a file with checksum should return the checksum in the download header
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    When user "Alice" downloads file "/myChecksumFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the header checksum should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1291
  Scenario Outline: moving a file with checksum should return the checksum in the propfind
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    When user "Alice" moves file "/myChecksumFile.txt" to "/myMovedChecksumFile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as user "Alice" the webdav checksum of "/myMovedChecksumFile.txt" via propfind should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f MD5:d70b40f177b14b470d1756a3c12b963a ADLER32:8ae90960"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1316
  Scenario Outline: downloading a file with checksum should return the checksum in the download header
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    And user "Alice" has moved file "/myChecksumFile.txt" to "/myMovedChecksumFile.txt"
    When user "Alice" downloads file "/myMovedChecksumFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the header checksum should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1291
  Scenario Outline: uploading a chunked file with checksum should return the checksum in the propfind
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded chunk file "1" of "3" with "AAAAA" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e"
    And user "Alice" has uploaded chunk file "2" of "3" with "BBBBB" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e"
    And user "Alice" has uploaded chunk file "3" of "3" with "CCCCC" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e"
    When user "Alice" requests the checksum of "/myChecksumFile.txt" via propfind
    Then the HTTP status code should be "207"
    And the webdav checksum should match "SHA1:acfa6b1565f9710d4d497c6035d5c069bd35a8e8 MD5:45a72715acdd5019c5be30bdbb75233e ADLER32:1ecd03df"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1343
  Scenario Outline: uploading a chunked file with checksum should return the checksum in the download header
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded chunk file "1" of "3" with "AAAAA" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e"
    And user "Alice" has uploaded chunk file "2" of "3" with "BBBBB" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e"
    And user "Alice" has uploaded chunk file "3" of "3" with "CCCCC" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e"
    When user "Alice" downloads file "/myChecksumFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the header checksum should match "SHA1:acfa6b1565f9710d4d497c6035d5c069bd35a8e8"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: moving file with checksum should return the checksum in the download header
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    When user "Alice" moves file "/myChecksumFile.txt" to "/myMovedChecksumFile.txt" using the WebDAV API
    And user "Alice" downloads file "/myMovedChecksumFile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the header checksum should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1291
  Scenario Outline: copying a file with checksum should return the checksum in the propfind
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    When user "Alice" copies file "/myChecksumFile.txt" to "/myChecksumFileCopy.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as user "Alice" the webdav checksum of "/myChecksumFileCopy.txt" via propfind should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f MD5:d70b40f177b14b470d1756a3c12b963a ADLER32:8ae90960"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1316
  Scenario Outline: copying file with checksum should return the checksum in the download header
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    When user "Alice" copies file "/myChecksumFile.txt" to "/myChecksumFileCopy.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the header checksum when user "Alice" downloads file "/myChecksumFileCopy.txt" using the WebDAV API should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1291 @skipOnReva
  Scenario Outline: sharing a file with checksum should return the checksum in the propfind using new DAV path
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    And user "Alice" has sent the following resource share invitation:
      | resource        | myChecksumFile.txt |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | File Editor        |
    And user "Brian" has a share "myChecksumFile.txt" synced
    When user "Brian" requests the checksum of "/Shares/myChecksumFile.txt" via propfind
    Then the HTTP status code should be "207"
    And the webdav checksum should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f MD5:d70b40f177b14b470d1756a3c12b963a ADLER32:8ae90960"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1291 @skipOnReva
  Scenario Outline: modifying a shared file should return correct checksum in the propfind using new DAV path
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/myChecksumFile.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a"
    And user "Alice" has sent the following resource share invitation:
      | resource        | myChecksumFile.txt |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | File Editor        |
    And user "Brian" has a share "myChecksumFile.txt" synced
    When user "Brian" uploads file with checksum "SHA1:ce5582148c6f0c1282335b87df5ed4be4b781399" and content "Some Text" to "/Shares/myChecksumFile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the webdav checksum of "/myChecksumFile.txt" via propfind should match "SHA1:ce5582148c6f0c1282335b87df5ed4be4b781399 MD5:56e57920c3c8c727bfe7a5288cdf61c4 ADLER32:1048035a"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1315
  Scenario Outline: upload a file where checksum does not match
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with checksum "SHA1:f005ba11" and content "Some Text" to "/chksumtst.txt" using the WebDAV API
    Then the HTTP status code should be "400"
    And user "Alice" should not see the following elements
      | /chksumtst.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a file where checksum does match
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with checksum "SHA1:ce5582148c6f0c1282335b87df5ed4be4b781399" and content "Some Text" to "/chksumtst.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1315
  Scenario Outline: uploaded file should have the same checksum when downloaded
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with checksum "SHA1:ce5582148c6f0c1282335b87df5ed4be4b781399" and content "Some Text" to "/chksumtst.txt"
    When user "Alice" downloads file "/chksumtst.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the following headers should be set
      | header      | value                                         |
      | OC-Checksum | SHA1:ce5582148c6f0c1282335b87df5ed4be4b781399 |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  ## Validation Plugin or Old Endpoint Specific
  @owncloud-issue-1343 @owncloud-issue-2323
  Scenario Outline: uploading an old method chunked file with checksum should fail using new DAV path
    Given using <dav-path-version> DAV path
    When user "Alice" uploads chunk file "1" of "3" with "AAAAA" to "/myChecksumFile.txt" with checksum "MD5:45a72715acdd5019c5be30bdbb75233e" using the WebDAV API
    Then the HTTP status code should be "503"
    And user "Alice" should not see the following elements
      | /myChecksumFile.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  ## upload overwriting
  @owncloud-issue-1291
  Scenario Outline: uploading a file with MD5 checksum overwriting an existing file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "some data" to "textfile0.txt"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/textfile0.txt" with checksum "MD5:d70b40f177b14b470d1756a3c12b963a" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the webdav checksum of "/textfile0.txt" via propfind should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f MD5:d70b40f177b14b470d1756a3c12b963a ADLER32:8ae90960"
    And the content of file "/textfile0.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1291
  Scenario Outline: uploading a file with SHA1 checksum overwriting an existing file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "some data" to "textfile0.txt"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/textfile0.txt" with checksum "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the webdav checksum of "/textfile0.txt" via propfind should match "SHA1:3ee962b839762adb0ad8ba6023a4690be478de6f MD5:d70b40f177b14b470d1756a3c12b963a ADLER32:8ae90960"
    And the content of file "/textfile0.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnStorage:ceph @skipOnStorage:scality @owncloud-issue-1291
  Scenario Outline: uploading a file with invalid SHA1 checksum overwriting an existing file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/textfile0.txt" with checksum "SHA1:f005ba11f005ba11f005ba11f005ba11f005ba11" using the WebDAV API
    Then the HTTP status code should be "400"
    And as user "Alice" the webdav checksum of "/textfile0.txt" via propfind should match "SHA1:294acb3a34f7e11f8eac865183d30aac04fba0c7 MD5:7f41994bdd25f722cd715bb376eff10e ADLER32:83ca095f"
    And the content of file "/textfile0.txt" for user "Alice" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1296
  Scenario Outline: uploading a file with checksum should work for file with special characters
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to <renamed-file> with checksum "MD5:d70b40f177b14b470d1756a3c12b963a" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file <renamed-file> for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version | renamed-file      |
      | old              | " oc?test=ab&cd " |
      | old              | "# %ab ab?=ed"    |
      | new              | " oc?test=ab&cd " |
      | new              | "# %ab ab?=ed"    |
      | spaces           | " oc?test=ab&cd " |
      | spaces           | "# %ab ab?=ed"    |
