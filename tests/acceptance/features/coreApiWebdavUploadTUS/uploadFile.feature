Feature: upload file
  As a user
  I want to be able to upload files
  So that I can store and share files between multiple client systems

  Background:
    Given user "Alice" has been created with default attributes

  @owncloud-issue-10346
  Scenario Outline: upload a file and check download content
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "<file-name>" using the TUS protocol on the WebDAV API
    Then the content of file "<file-name>" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | file-name         |
      | old              | /upload.txt       |
      | old              | /नेपाली.txt       |
      | old              | /strängé file.txt |
      | old              | /s,a,m,p,l,e.txt  |
      | old              | /C++ file.cpp     |
      | old              | /?fi=le&%#2 . txt |
      | old              | /# %ab ab?=ed     |
      | new              | /upload.txt       |
      | new              | /strängé file.txt |
      | new              | /नेपाली.txt       |
      | new              | /s,a,m,p,l,e.txt  |
      | new              | /C++ file.cpp     |
      | new              | /?fi=le&%#2 . txt |
      | new              | /# %ab ab?=ed     |
      | spaces           | /upload.txt       |
      | spaces           | /strängé file.txt |
      | spaces           | /नेपाली.txt       |
      | spaces           | /s,a,m,p,l,e.txt  |
      | spaces           | /C++ file.cpp     |
      | spaces           | /?fi=le&%#2 . txt |
      | spaces           | /# %ab ab?=ed     |

  @owncloud-issue-10346
  Scenario Outline: upload a file into a folder and check download content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    When user "Alice" uploads file with content "uploaded content" to "<folder-name>/<file-name>" using the TUS protocol on the WebDAV API
    Then the content of file "<folder-name>/<file-name>" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version | folder-name                      | file-name                     |
      | old              | /upload                          | abc.txt                       |
      | old              | /strängé folder                  | strängé file.txt              |
      | old              | /C++ folder                      | C++ file.cpp                  |
      | old              | /नेपाली                          | नेपाली                        |
      | old              | /folder #2.txt                   | file #2.txt                   |
      | old              | /folder ?2.txt                   | file ?2.txt                   |
      | old              | /?fi=le&%#2 . txt                | # %ab ab?=ed                  |
      | new              | /upload                          | abc.txt                       |
      | new              | /strängé folder (duplicate #2 &) | strängé file (duplicate #2 &) |
      | new              | /C++ folder                      | C++ file.cpp                  |
      | new              | /नेपाली                          | नेपाली                        |
      | new              | /folder #2.txt                   | file #2.txt                   |
      | new              | /folder ?2.txt                   | file ?2.txt                   |
      | new              | /?fi=le&%#2 . txt                | # %ab ab?=ed                  |
      | spaces           | /upload                          | abc.txt                       |
      | spaces           | /strängé folder (duplicate #2 &) | strängé file (duplicate #2 &) |
      | spaces           | /C++ folder                      | C++ file.cpp                  |
      | spaces           | /नेपाली                          | नेपाली                        |
      | spaces           | /folder #2.txt                   | file #2.txt                   |
      | spaces           | /folder ?2.txt                   | file ?2.txt                   |
      | spaces           | /?fi=le&%#2 . txt                | # %ab ab?=ed                  |

  @owncloud-issue-10346
  Scenario Outline: upload chunked file with TUS
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" in 3 chunks to "/myChunkedFile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "/myChunkedFile.txt" for user "Alice" should be "uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload 1 byte chunks with TUS
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "0123456789" in 10 chunks to "/myChunkedFile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "/myChunkedFile.txt" for user "Alice" should be "0123456789"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload to overwriting a file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "original content" to "textfile.txt"
    When user "Alice" uploads file with content "overwritten content" to "textfile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "textfile.txt" for user "Alice" should be "overwritten content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload a file and no version is available
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "/upload.txt" using the TUS protocol on the WebDAV API
    Then the version folder of file "/upload.txt" for user "Alice" should contain "0" elements
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload a file twice and versions are available
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "/upload.txt" using the TUS protocol on the WebDAV API
    And user "Alice" uploads file with content "re-uploaded content" to "/upload.txt" using the TUS protocol on the WebDAV API
    Then the version folder of file "/upload.txt" for user "Alice" should contain "1" element
    And the content of file "/upload.txt" for user "Alice" should be "re-uploaded content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload a file in chunks with TUS and no version is available
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "0123456789" in 10 chunks to "/myChunkedFile.txt" using the TUS protocol on the WebDAV API
    Then the version folder of file "/myChunkedFile.txt" for user "Alice" should contain "0" elements
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10346
  Scenario Outline: upload a twice file in chunks with TUS and versions are available
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "0123456789" in 10 chunks to "/myChunkedFile.txt" using the TUS protocol on the WebDAV API
    And user "Alice" uploads file with content "01234" in 5 chunks to "/myChunkedFile.txt" using the TUS protocol on the WebDAV API
    Then the version folder of file "/myChunkedFile.txt" for user "Alice" should contain "1" elements
    And the content of file "/myChunkedFile.txt" for user "Alice" should be "01234"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-10334
  Scenario Outline: upload a file with invalid-name
    Given using <dav-path-version> DAV path
    When user "Alice" creates a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 100                 |
      | Upload-Metadata | filename <metadata> |
      | Tus-Resumable   | 1.0.0               |
    Then the HTTP status code should be "412"
    And the following headers should not be set
      | header   |
      | Location |
    And as "Alice" file <file-name> should not exist
    Examples:
      | dav-path-version | file-name               | metadata                     |
      | old              | " "                     | IA==                         |
      | old              | "filewithLF-and-CR\r\n" | ZmlsZXdpdGhMRi1hbmQtQ1INCgo= |
      | old              | "folder/file"           | Zm9sZGVyL2ZpbGU=             |
      | old              | "my\\file"              | bXkMaWxl                     |
      | old              | ".."                    | Li4=                         |
      | new              | " "                     | IA==                         |
      | new              | "filewithLF-and-CR\r\n" | ZmlsZXdpdGhMRi1hbmQtQ1INCgo= |
      | new              | "folder/file"           | Zm9sZGVyL2ZpbGU=             |
      | new              | "my\\file"              | bXkMaWxl                     |
      | new              | ".."                    | Li4=                         |
      | spaces           | " "                     | IA==                         |
      | spaces           | "filewithLF-and-CR\r\n" | ZmlsZXdpdGhMRi1hbmQtQ1INCgo= |
      | spaces           | "folder/file"           | Zm9sZGVyL2ZpbGU=             |
      | spaces           | "my\\file"              | bXkMaWxl                     |
      | spaces           | ".."                    | Li4=                         |

  @owncloud-issue-10346
  Scenario Outline: upload a zero-byte file
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "textfile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-8003 @owncloud-issue-10346
  Scenario Outline: replace a file with zero-byte file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "This is TUS upload" to "textfile.txt"
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "textfile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-8003 @owncloud-issue-10346
  Scenario Outline: replace a file inside a folder with zero-byte file
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    And user "Alice" has uploaded file with content "This is TUS upload" to "testFolder/textfile.txt"
    When user "Alice" uploads file "filesForUpload/zerobyte.txt" to "testFolder/textfile.txt" using the TUS protocol on the WebDAV API
    Then the content of file "testFolder/textfile.txt" for user "Alice" should be ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-8804
  Scenario Outline: multiple upload locations of the same file
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 5                     |
      #    bG9yZW0udHh0 is the base64 encode of lorem.txt
      | Upload-Metadata | filename bG9yZW0udHh0 |
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 5                     |
      #    bG9yZW0udHh0 is the base64 encode of lorem.txt
      | Upload-Metadata | filename bG9yZW0udHh0 |
    When user "Alice" uploads content "lorem" with checksum "MD5 d2e16e6ef52a45b7468f1da56bba1953" and offset "0" to the index "1" location of file "lorem.txt" using the TUS protocol
    Then the HTTP status code should be "204"
    And the content of file "lorem.txt" for user "Alice" should be "lorem"
    When user "Alice" tries to upload content "epsum" with checksum "MD5 d6145e3d2ced88009796acae1dc7929f" and offset "0" to the index "0" location of file "lorem.txt" using the TUS protocol
    Then the HTTP status code should be "409"
    And the content of file "lorem.txt" for user "Alice" should be "lorem"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
