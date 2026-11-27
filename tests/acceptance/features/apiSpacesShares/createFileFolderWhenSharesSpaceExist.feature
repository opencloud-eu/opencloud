Feature: create file or folder named similar to Shares folder
  As a user
  I want to be able to create files and folders when the Shares folder exists
  So that I can organise the files in my file system

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |


  Scenario Outline: create a folder with a name similar to Shares
    Given using spaces DAV path
    When user "Brian" creates folder "<folder-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Brian" the space "Personal" should contain these entries:
      | <folder-name>/ |
    Examples:
      | folder-name |
      | Share       |
      | shares      |
      | Share1      |


  Scenario Outline: create a file with a name similar to Shares
    Given using spaces DAV path
    When user "Brian" uploads file with content "some text" to "<file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "<file-name>" for user "Brian" should be "some text"
    And for user "Brian" the space "Personal" should contain these entries:
      | <file-name> |
    And for user "Brian" the space "Shares" should contain these entries:
      | FOLDER/ |
    Examples:
      | file-name |
      | Share     |
      | shares    |
      | Share1    |


  Scenario: try to create a folder named Shares
    Given using spaces DAV path
    When user "Brian" creates folder "/Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Brian" the space "Shares" should contain these entries:
      | FOLDER/ |


  Scenario: try to create a file named Shares
    Given using spaces DAV path
    When user "Brian" uploads file with content "some text" to "/Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Brian" the space "Shares" should contain these entries:
      | FOLDER/ |
