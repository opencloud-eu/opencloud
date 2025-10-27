Feature: change shared resource
  As a user
  I want to change the shared resource
  So that I can organize them as per my necessity

  Background:
    Given using spaces DAV path
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |

  @owncloud-issue-4421
  Scenario: move files between shares by different users
    Given user "Carol" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has created folder "/PARENT"
    And user "Brian" has created folder "/PARENT"
    And user "Alice" has moved file "textfile0.txt" to "PARENT/from_alice.txt" in space "Personal"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Carol    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Carol" has a share "PARENT" synced
    And user "Brian" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Carol    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Carol" has a share "PARENT (1)" synced
    When user "Carol" moves file "PARENT/from_alice.txt" to "PARENT (1)/from_alice.txt" in space "Shares" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Carol" folder "PARENT" of the space "Shares" should contain these entries:
      | from_alice.txt |
    And for user "Carol" folder "PARENT (1)" of the space "Shares" should not contain these entries:
      | from_alice.txt |


  Scenario Outline: overwrite a received file share
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And user "Alice" has uploaded file with content "old content version 1" to "/textfile1.txt"
    And user "Alice" has uploaded file with content "old content version 2" to "/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    And user "Brian" has a share "textfile1.txt" synced
    When user "Brian" uploads a file inside space "Shares" with content "this is a new content" to "textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Brian" the space "Shares" should contain these entries:
      | textfile1.txt |
    And for user "Brian" the content of the file "/textfile1.txt" of the space "Shares" should be "this is a new content"
    And for user "Alice" the content of the file "/textfile1.txt" of the space "Personal" should be "this is a new content"
    When user "Alice" downloads version of the file "/textfile1.txt" with the index "2" of the space "Personal" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "old content version 1"
    When user "Alice" downloads version of the file "/textfile1.txt" with the index "1" of the space "Personal" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "old content version 2"
    When user "Brian" tries to get versions of the file "/textfile1.txt" from the space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |
