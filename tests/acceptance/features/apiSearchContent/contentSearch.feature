@tikaServiceNeeded
Feature: content search
  As a user
  I want to do search resources by content
  So that I can find the files with the content I am looking for

  Background:
    Given user "Alice" has been created with default attributes

  @issue-10329
  Scenario Outline: search files by content
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "hello world from nepal" to "keywordAtStart.txt"
    And user "Alice" has uploaded file with content "saying hello to the world" to "keywordAtMiddle.txt"
    And user "Alice" has uploaded file with content "nepal want to say hello" to "keywordAtLast.txt"
    And user "Alice" has uploaded file with content "namaste from nepal" to "hello.txt"
    When user "Alice" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | keywordAtStart.txt  |
      | keywordAtMiddle.txt |
      | keywordAtLast.txt   |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: search files by different content types
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "Using k6, you can test the reliability and performance of your systems" to "wordWithNumber.md"
    And user "Alice" has uploaded file with content "see our web site https://opencloud.eu/en/contact-us" to "findByWebSite.txt"
    And user "Alice" has uploaded file with content "alan@example.org want to say hello" to "findByEmail.docs"
    When user "Alice" searches for "Content:k6" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | wordWithNumber.md |
    When user "Alice" searches for "Content:https://opencloud.eu/" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | findByWebSite.txt |
    When user "Alice" searches for "Content:alan@" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | findByEmail.docs |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: search files by stop words when clean_stop_words is enabled (default)
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "He has expirience, we must to have, I have to find ...." to "fileWithStopWords.txt"
    When user "Alice" searches for 'Content:"he has"' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Brian" should not contain these entries:
      | fileWithStopWords.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @env-config @issue-10329
  Scenario Outline: search files by stop words when clean_stop_words is disabled
    Given using <dav-path-version> DAV path
    And the config "SEARCH_EXTRACTOR_TIKA_CLEAN_STOP_WORDS" has been set to "false"
    And user "Alice" has uploaded file with content "He has expirience, we must to have, I have to find ...." to "fileWithStopWords.txt"
    When user "Alice" searches for 'Content:"he has"' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | fileWithStopWords.txt |
    When user "Alice" searches for 'Content:"I have"' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | fileWithStopWords.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: sharee searches files by content
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "hello world from nepal" to "uploadFolder/keywordAtStart.txt"
    And user "Alice" has uploaded file with content "saying hello to the world" to "uploadFolder/keywordAtMiddle.txt"
    And user "Alice" has uploaded file with content "nepal want to say hello" to "uploadFolder/keywordAtLast.txt"
    And user "Alice" has uploaded file with content "Namaste nepal" to "uploadFolder/hello.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "uploadFolder" synced
    When user "Brian" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Brian" should contain only these files:
      | keywordAtStart.txt  |
      | keywordAtMiddle.txt |
      | keywordAtLast.txt   |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: search deleted files by content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "hello world from nepal" to "uploadFolder/keywordAtStart.txt"
    And user "Alice" has uploaded file with content "saying hello to the world" to "keywordAtMiddle.txt"
    And user "Alice" has uploaded file with content "nepal want to say hello" to "keywordAtLast.txt"
    And user "Alice" has deleted file "keywordAtLast.txt"
    When user "Alice" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | keywordAtStart.txt  |
      | keywordAtMiddle.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: search restored files by content
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "hello world from nepal" to "keywordAtStart.txt"
    And user "Alice" has deleted file "keywordAtStart.txt"
    And user "Alice" has restored the file with original path "keywordAtStart.txt"
    When user "Alice" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | keywordAtStart.txt |
    Examples:
      | dav-path-version |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: search restored version of a file by content
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "hello world" to "test.txt"
    And user "Alice" has uploaded file with content "Namaste nepal" to "test.txt"
    And user "Alice" has restored version index "1" of file "test.txt"
    When user "Alice" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | test.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: search project space files by content
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has created a folder "spacesFolderWithFile/spacesSubFolder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "hello world from nepal" to "keywordAtStart.txt"
    And user "Alice" has uploaded a file inside space "project-space" with content "saying hello to the world" to "spacesFolderWithFile/keywordAtMiddle.txt"
    And user "Alice" has uploaded a file inside space "project-space" with content "nepal want to say hello" to "spacesFolderWithFile/spacesSubFolder/keywordAtLast.txt"
    And user "Alice" has uploaded a file inside space "project-space" with content "namaste from nepal" to "hello.txt"
    And using <dav-path-version> DAV path
    When user "Alice" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | keywordAtStart.txt  |
      | keywordAtMiddle.txt |
      | keywordAtLast.txt   |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: sharee searches shared project space files by content
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | project-space |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Space Viewer  |
    And user "Alice" has created a folder "spacesFolderWithFile/spacesSubFolder" in space "project-space"
    And user "Alice" has uploaded a file inside space "project-space" with content "hello world from nepal" to "keywordAtStart.txt"
    And user "Alice" has uploaded a file inside space "project-space" with content "saying hello to the world" to "spacesFolderWithFile/keywordAtMiddle.txt"
    And user "Alice" has uploaded a file inside space "project-space" with content "nepal wants to say hello" to "spacesFolderWithFile/spacesSubFolder/keywordAtLast.txt"
    And user "Alice" has uploaded a file inside space "project-space" with content "namaste from nepal" to "hello.txt"
    And using <dav-path-version> DAV path
    When user "Brian" searches for "Content:hello" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these files:
      | keywordAtStart.txt  |
      | keywordAtMiddle.txt |
      | keywordAtLast.txt   |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10329
  Scenario Outline: search resources using different search patterns (KQL feature)
    Given using spaces DAV path
    And user "Alice" has uploaded file with content "hello world, let start to test" to "technical task.txt"
    And user "Alice" has uploaded file with content "it's been hell" to "task comments.txt"
    And user "Alice" has tagged the following files of the space "Personal":
      | path               | tagName |
      | technical task.txt | test    |
    When user "Alice" searches for '<pattern>' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "<result-count>" entries
    And the search result of user "Alice" should contain these entries:
      | <search-result-1> |
      | <search-result-2> |
    Examples:
      | pattern                                     | result-count | search-result-1    | search-result-2   |
      | Content:hello                               | 1            | technical task.txt |                   |
      | content:hello                               | 1            | technical task.txt |                   |
      | content:"hello"                             | 1            | technical task.txt |                   |
      | content:hel*                                | 2            | technical task.txt | task comments.txt |
      | content:hel* AND tag:test                   | 1            | technical task.txt |                   |
      | (name:*task* AND content:hel*) NOT tag:test | 1            | task comments.txt  |                   |

  @issue-10329
  Scenario Outline: search across files with different format with search text highlight
    Given using <dav-path-version> DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded file with content "this is a simple text file" to "test-text-file.txt"
    And user "Alice" has uploaded file with content "this is a simple pdf file" to "test-pdf-file.pdf"
    And user "Alice" has uploaded file with content "this is a simple cpp file" to "test-cpp-file.cpp"
    And user "Alice" has uploaded file with content "this is another text file" to "testfile.txt"
    And user "Alice" has uploaded file "filesForUpload/testavatar.png" to "/testavatar.png"
    And user "Alice" has uploaded a file inside space "project-space" with content "this is a simple markdown file" to "test-md-file.md"
    And user "Alice" has uploaded a file inside space "project-space" with content "this is a simple odt file" to "test-odt-file.odt"
    When user "Alice" searches for "Content:simple" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain these entries with highlight on keyword "simple"
      | test-text-file.txt |
      | test-pdf-file.pdf  |
      | test-cpp-file.cpp  |
      | test-md-file.md    |
      | test-odt-file.odt  |
    But the search result of user "Alice" should not contain these entries:
      | testavatar.png |
      | testfile.txt   |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
