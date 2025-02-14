Feature: media type search
  As a user
  I want to search files using media type
  So that I can find the files with specific media type

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path

  @issue-10329
  Scenario Outline: search for files using media type
    Given user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/lorem.txt"
    And user "Alice" has uploaded file "filesForUpload/simple.pdf" to "/simple.pdf"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/testavatar.jpg"
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "/testavatar.png"
    And user "Alice" has uploaded file "filesForUpload/data.tar.gz" to "/data.tar.gz"
    And user "Alice" has uploaded file "filesForUpload/data.tar" to "/data.tar"
    And user "Alice" has uploaded file "filesForUpload/data.7z" to "/data.7z"
    And user "Alice" has uploaded file "filesForUpload/data.rar" to "/data.rar"
    And user "Alice" has uploaded file "filesForUpload/data.tar.bz2" to "/data.tar.bz2"
    When user "Alice" searches for "mediatype:<pattern>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | <search-result> |
    Examples:
      | pattern | search-result   |
      | *text*  | lorem.txt      |
      | *pdf*   | simple.pdf     |
      | *jpeg*  | testavatar.jpg |
      | *png*   | testavatar.png |
      | *gzip*  | data.tar.gz    |
      | *tar*   | data.tar       |
      | *7z*    | data.7z        |
      | *rar*   | data.rar       |
      | *bzip2* | data.tar.bz2   |

  @issue-10329
  Scenario Outline: search for files inside sub folders using media type
    Given user "Alice" has created folder "/uploadFolder"
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/uploadFolder/lorem.txt"
    And user "Alice" has uploaded file "filesForUpload/simple.pdf" to "/uploadFolder/simple.pdf"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/uploadFolder/testavatar.jpg"
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "/uploadFolder/testavatar.png"
    And user "Alice" has uploaded file "filesForUpload/data.tar.gz" to "/uploadFolder/data.tar.gz"
    And user "Alice" has uploaded file "filesForUpload/data.tar" to "/uploadFolder/data.tar"
    And user "Alice" has uploaded file "filesForUpload/data.7z" to "/uploadFolder/data.7z"
    And user "Alice" has uploaded file "filesForUpload/data.rar" to "/uploadFolder/data.rar"
    And user "Alice" has uploaded file "filesForUpload/data.tar.bz2" to "/uploadFolder/data.tar.bz2"
    When user "Alice" searches for "mediatype:<pattern>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | <search-result> |
    Examples:
      | pattern | search-result  |
      | *text*  | lorem.txt      |
      | *pdf*   | simple.pdf     |
      | *jpeg*  | testavatar.jpg |
      | *png*   | testavatar.png |
      | *gzip*  | data.tar.gz    |
      | *tar*   | data.tar       |
      | *7z*    | data.7z        |
      | *rar*   | data.rar       |
      | *bzip2* | data.tar.bz2   |

  @issue-10329
  Scenario Outline: search for file inside project space using media type
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project101" with the default quota using the Graph API
    And user "Alice" has uploaded a file "filesForUpload/lorem.txt" to "/lorem.txt" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/simple.pdf" to "/simple.pdf" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/testavatar.jpg" to "/testavatar.jpg" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/testavatar.png" to "/testavatar.png" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.tar.gz" to "/data.tar.gz" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.tar" to "/data.tar" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.7z" to "/data.7z" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.rar" to "/data.rar" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.tar.bz2" to "/data.tar.bz2" in space "project101"
    When user "Alice" searches for "mediatype:<pattern>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | <search-result> |
    Examples:
      | pattern | search-result   |
      | *text*  | lorem.txt      |
      | *pdf*   | simple.pdf     |
      | *jpeg*  | testavatar.jpg |
      | *png*   | testavatar.png |
      | *gzip*  | data.tar.gz    |
      | *tar*   | data.tar       |
      | *7z*    | data.7z        |
      | *rar*   | data.rar       |
      | *bzip2* | data.tar.bz2   |

  @issue-10329
  Scenario Outline: sharee searches for shared files using media type
    Given user "Alice" has created folder "/uploadFolder"
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/uploadFolder/lorem.txt"
    And user "Alice" has uploaded file "filesForUpload/simple.pdf" to "/uploadFolder/simple.pdf"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/uploadFolder/testavatar.jpg"
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "/uploadFolder/testavatar.png"
    And user "Alice" has uploaded file "filesForUpload/data.tar.gz" to "/uploadFolder/data.tar.gz"
    And user "Alice" has uploaded file "filesForUpload/data.tar" to "/uploadFolder/data.tar"
    And user "Alice" has uploaded file "filesForUpload/data.7z" to "/uploadFolder/data.7z"
    And user "Alice" has uploaded file "filesForUpload/data.rar" to "/uploadFolder/data.rar"
    And user "Alice" has uploaded file "filesForUpload/data.tar.bz2" to "/uploadFolder/data.tar.bz2"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "uploadFolder" synced
    When user "Brian" searches for "mediatype:<pattern>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | <search-result> |
    Examples:
      | pattern | search-result  |
      | *text*  | lorem.txt      |
      | *pdf*   | simple.pdf     |
      | *jpeg*  | testavatar.jpg |
      | *png*   | testavatar.png |
      | *gzip*  | data.tar.gz    |
      | *tar*   | data.tar       |
      | *7z*    | data.7z        |
      | *rar*   | data.rar       |
      | *bzip2* | data.tar.bz2   |

  @issue-10329
  Scenario Outline: space viewer searches for files using mediatype filter
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project101" with the default quota using the Graph API
    And user "Alice" has uploaded a file "filesForUpload/lorem.txt" to "/lorem.txt" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/simple.pdf" to "/simple.pdf" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/testavatar.jpg" to "/testavatar.jpg" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/testavatar.png" to "/testavatar.png" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.tar.gz" to "/data.tar.gz" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.tar" to "/data.tar" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.7z" to "/data.7z" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.rar" to "/data.rar" in space "project101"
    And user "Alice" has uploaded a file "filesForUpload/data.tar.bz2" to "/data.tar.bz2" in space "project101"
    And user "Alice" has sent the following space share invitation:
      | space           | project101   |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" searches for "mediatype:<pattern>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | <search-result> |
    Examples:
      | pattern | search-result   |
      | *text*  | lorem.txt      |
      | *pdf*   | simple.pdf     |
      | *jpeg*  | testavatar.jpg |
      | *png*   | testavatar.png |
      | *gzip*  | data.tar.gz    |
      | *tar*   | data.tar       |
      | *7z*    | data.7z        |
      | *rar*   | data.rar       |
      | *bzip2* | data.tar.bz2   |

  @issue-10329
  Scenario: search files with different mediatype filter
    Given user "Alice" has created folder "testFolder"
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "lorem.txt"
    And user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And user "Alice" has uploaded file "filesForUpload/simple.pdf" to "simple.pdf"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "testavatar.png"
    And user "Alice" has uploaded file "filesForUpload/example.gif" to "example.gif"
    And user "Alice" has uploaded file "filesForUpload/data.tar.gz" to "data.tar.gz"
    And user "Alice" has uploaded file "filesForUpload/data.tar" to "data.tar"
    And user "Alice" has uploaded file "filesForUpload/data.7z" to "data.7z"
    And user "Alice" has uploaded file "filesForUpload/data.rar" to "data.rar"
    And user "Alice" has uploaded file "filesForUpload/data.tar.bz2" to "data.tar.bz2"
    When user "Alice" searches for "mediatype:folder" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "2" entries
    And the search result of user "Alice" should contain these entries:
      | %spaceid%  |
      | testFolder |
    When user "Alice" searches for "mediatype:document" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "2" entries
    And the search result of user "Alice" should contain these entries:
      | lorem.txt  |
      | simple.odt |
    When user "Alice" searches for "mediatype:pdf" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | simple.pdf |
    When user "Alice" searches for "mediatype:image" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "3" entries
    And the search result of user "Alice" should contain these entries:
      | testavatar.jpg |
      | testavatar.png |
      | example.gif    |
    When user "Alice" searches for "mediatype:archive" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "5" entries
    And the search result of user "Alice" should contain these entries:
      | data.tar.gz  |
      | data.tar     |
      | data.7z      |
      | data.rar     |
      | data.tar.bz2 |
