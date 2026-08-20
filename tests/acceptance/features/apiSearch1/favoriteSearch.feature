Feature: search for favorites
  As a user
  I want to be able to search for my favorite files and folders

  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path


  Scenario: search by favorite files and folders in the Personal space
    Given user "Alice" has created folder "first"
    And user "Alice" has created folder "first/second"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "first/second/testavatar.jpg"
    And user "Alice" has marked file "first/second/testavatar.jpg" as favorite from space "Personal"
    And user "Alice" has marked folder "first" as favorite from space "Personal"
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "2" entries
    And the search result of user "Alice" should contain these entries:
      | first          |
      | testavatar.jpg |
    But the search result of user "Alice" should not contain these entries:
      | second         |


  Scenario: search for favorite files and folders in a project space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text2.txt"
    And user "Alice" has created a folder "space-folder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Brian" has marked file "text.txt" as favorite from space "new-space"
    And user "Brian" has marked folder "space-folder" as favorite from space "new-space"
    When user "Brian" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "2" entries
    And the search result of user "Brian" should contain these entries:
      | text.txt       |
      | space-folder   |
    But the search result of user "Brian" should not contain these entries:
      | text2.txt      |
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "0" entries

  @issue-2488
  Scenario: search for favorite shared files and folders
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/sub"
    And user "Alice" has created folder "parent/sub/sub2"
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "parent/sub/sub2/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent/sub/sub2/textfile.txt |
      | space           | Personal                     |
      | sharee          | Brian                        |
      | shareType       | user                         |
      | permissionsRole | Viewer                       |
    And user "Brian" has a share "textfile.txt" synced
    And user "Brian" has a share "parent" synced
    And user "Brian" has marked file "textfile.txt" as favorite from space "Shares"
    And user "Brian" has marked folder "parent/sub" as favorite from space "Shares"
    When user "Brian" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "3" entries
    And the search result of user "Brian" should contain these entries:
      | textfile.txt |
      | sub          |
    But the search result of user "Brian" should not contain these entries:
      | parent          |
      | parent/sub/sub2 |
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "0" entries


  Scenario Outline: search for favorite files by name and media type in the Personal space
    Given user "Alice" has uploaded file "filesForUpload/textfile.txt" to "test-textfile.txt"
    And user "Alice" has uploaded file "filesForUpload/simple.odt" to "test.odt"
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "test-not-favorite.txt"
    And user "Alice" has marked file "test-textfile.txt" as favorite from space "Personal"
    And user "Alice" has marked file "test.odt" as favorite from space "Personal"
    When user "Alice" searches for '<search-patern>' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "2" entries
    And the search result of user "Alice" should contain these entries:
      | test-textfile.txt |
      | test.odt          |
    But the search result of user "Alice" should not contain these entries:
      | test-not-favorite.txt |
    Examples:
      | search-patern                          |
      | name:"test*" AND is:favorite           |
      | mediatype:("document") AND is:favorite |


  Scenario: search for favorite files after unmarking a file as favorite
    Given user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile.txt"
    And user "Alice" has marked file "textfile.txt" as favorite from space "Personal"
    And user "Alice" has unmarked file "textfile.txt" as favorite from space "Personal"
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "0" entries


  Scenario: search for favorite files after deleting a file
    Given user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile.txt"
    And user "Alice" has marked file "textfile.txt" as favorite from space "Personal"
    And user "Alice" has deleted file "textfile.txt"
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "0" entries


  Scenario: search for favorite files after restoring a file
    Given user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile.txt"
    And user "Alice" has marked file "textfile.txt" as favorite from space "Personal"
    And user "Alice" has deleted file "textfile.txt"
    And user "Alice" has restored the file with original path "textfile.txt"
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | textfile.txt |


  Scenario: search for favorite files after moving a file
    Given user "Alice" has created folder "parent"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile.txt"
    And user "Alice" has marked file "textfile.txt" as favorite from space "Personal"
    And user "Alice" has moved file "textfile.txt" to "parent/textfile.txt"
    When user "Alice" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "1" entries
    And the search result of user "Alice" should contain these entries:
      | textfile.txt |


  Scenario: search for favorite shared files after removing the access to the shared file
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "textfile.txt" synced
    And user "Brian" has marked file "textfile.txt" as favorite from space "Shares"
    And user "Alice" has removed the access of user "Brian" from resource "textfile.txt" of space "Personal"
    When user "Brian" searches for 'is:favorite' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "0" entries
