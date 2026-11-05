Feature: access the space of the share owner using the drive id
  As a user
  I want the sharee to only access the shared resource
  So that the rest of my space stays private

  Background:
    Given using spaces DAV path
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "inside" to "folderToShare/inside.txt"
    And user "Alice" has uploaded file with content "private" to "private.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Brian" has a share "folderToShare" synced


  Scenario: sharee cannot access the space of the owner using its drive id
    When user "Brian" lists the content of the space of the owner of share "folderToShare" using the WebDAV API
    Then the HTTP status code should be "404"
    When user "Brian" downloads file "private.txt" from the space of the owner of share "folderToShare" using the WebDAV API
    Then the HTTP status code should be "404"
    When user "Brian" uploads content "uploaded by Brian" to file "brian.txt" in the space of the owner of share "folderToShare" using the WebDAV API
    Then the HTTP status code should be "404"
    When user "Brian" creates folder "brianFolder" in the space of the owner of share "folderToShare" using the WebDAV API
    Then the HTTP status code should be "404"
    When user "Brian" deletes file "private.txt" from the space of the owner of share "folderToShare" using the WebDAV API
    Then the HTTP status code should be "404"
    And for user "Alice" the content of the file "private.txt" of the space "Personal" should be "private"
    And for user "Alice" folder "/" of the space "Personal" should not contain these files:
      | brian.txt   |
      | brianFolder |


  Scenario: sharee cannot copy or move a file out of the shared folder into the space of the owner
    When user "Brian" tries to copy file "inside.txt" from share "folderToShare" to "copied.txt" in the space of the owner using the WebDAV API
    Then the HTTP status code should be "409"
    When user "Brian" tries to move file "inside.txt" from share "folderToShare" to "moved.txt" in the space of the owner using the WebDAV API
    Then the HTTP status code should be "409"
    And for user "Alice" the content of the file "folderToShare/inside.txt" of the space "Personal" should be "inside"
    And for user "Alice" folder "/" of the space "Personal" should not contain these files:
      | copied.txt |
      | moved.txt  |
