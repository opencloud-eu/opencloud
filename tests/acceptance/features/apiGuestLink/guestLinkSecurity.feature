@email
Feature: guest link access is limited to the shared resource
  As a user sharing a folder with a guest
  I want the guest to only access the shared folder
  So that the rest of my files stay private

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "inside" to "folderToShare/inside.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare     |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Editor            |
    And guest "guest@example.org" has verified the guest link from the last email


  Scenario: guest works inside the shared folder
    When guest "guest@example.org" lists the content of folder "folderToShare" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "207"
    When guest "guest@example.org" uploads content "uploaded by guest" to "new.txt" inside folder "folderToShare" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "201"
    When guest "guest@example.org" creates folder "newFolder" inside folder "folderToShare" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "201"
    When guest "guest@example.org" renames "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to "renamed.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the content of the file "folderToShare/new.txt" of the space "Personal" should be "uploaded by guest"
    And as "Alice" folder "folderToShare/newFolder" should exist
    And as "Alice" file "folderToShare/renamed.txt" should exist
    And as "Alice" file "folderToShare/inside.txt" should not exist


  Scenario: guest cannot move a file out of the shared folder
    When guest "guest@example.org" tries to move "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to "stolen.txt" in the root of that space using the WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" file "folderToShare/inside.txt" should exist
    And as "Alice" file "stolen.txt" should not exist


  Scenario: guest session gives access to all shares of the same guest
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "from Alice" to "aliceFile.txt"
    And user "Brian" has uploaded file with content "from Brian" to "brianFile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | aliceFile.txt     |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And user "Brian" has sent the following resource share invitation:
      | resource        | brianFile.txt     |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When guest "guest@example.org" downloads file "aliceFile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "from Alice"
    When guest "guest@example.org" downloads file "brianFile.txt" shared from space "Personal" of user "Brian" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "from Brian"
    When guest "guest@example.org" uploads content "changed by guest" to file "aliceFile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    When guest "guest@example.org" lists the shares shared with them using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "type": "object",
              "required": ["name"],
              "properties": {
                "name": { "enum": ["folderToShare", "aliceFile.txt", "brianFile.txt"] }
              }
            }
          }
        }
      }
      """


  Scenario Outline: guest cannot manage the share
    When guest "guest@example.org" sends HTTP method "<method>" to the Graph endpoint "<endpoint>" of resource "folderToShare" of space "Personal" of user "Alice"
    Then the HTTP status code should be "401"
    Examples:
      | method | endpoint    |
      | GET    | permissions |
      | POST   | invite      |
      | POST   | createLink  |


  Scenario: guest session with an invalid signature is rejected
    Given the signature of the guest session of guest "guest@example.org" has been changed
    When guest "guest@example.org" lists the shares shared with them using the Graph API
    Then the HTTP status code should be "401"
    When guest "guest@example.org" lists the content of folder "folderToShare" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "401"

