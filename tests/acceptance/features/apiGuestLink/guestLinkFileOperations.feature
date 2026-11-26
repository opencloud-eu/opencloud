@email
Feature: file operations of a guest on shared resources
  As a guest
  I want to work with the resources shared with me
  So that I can collaborate without an account, while the rest of the owner's files stay private

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

  @issue-3770
  Scenario: guest with the Editor role cannot delete the shared folder
    When guest "guest@example.org" tries to delete the shared folder "folderToShare" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" folder "folderToShare" should exist
    And as "Alice" file "folderToShare/inside.txt" should exist


  Scenario: guest cannot rename the shared folder
    When guest "guest@example.org" tries to rename the shared folder "folderToShare" of space "Personal" of user "Alice" to "renamedFolder" using the WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" folder "folderToShare" should exist
    And as "Alice" folder "renamedFolder" should not exist


  Scenario: guest with the Viewer role cannot change the shared folder
    Given user "Alice" has created folder "viewerFolder"
    And user "Alice" has uploaded file with content "viewer" to "viewerFolder/viewer.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | viewerFolder      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When guest "guest@example.org" uploads content "uploaded by guest" to "new.txt" inside folder "viewerFolder" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    When guest "guest@example.org" creates folder "newFolder" inside folder "viewerFolder" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    When guest "guest@example.org" renames "viewer.txt" inside folder "viewerFolder" of space "Personal" of user "Alice" to "renamed.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    When guest "guest@example.org" deletes "viewer.txt" inside folder "viewerFolder" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "viewerFolder/viewer.txt" should exist
    And as "Alice" file "viewerFolder/new.txt" should not exist
    And as "Alice" folder "viewerFolder/newFolder" should not exist


  Scenario: guest downloads the archive of the shared folder
    Given user "Alice" has uploaded file with content "second" to "folderToShare/second.txt"
    And user "Alice" has created folder "folderToShare/subFolder"
    And user "Alice" has uploaded file with content "nested" to "folderToShare/subFolder/nested.txt"
    When guest "guest@example.org" downloads the archive of these resources using the archiver:
      | resource      | space    | owner |
      | folderToShare | Personal | Alice |
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name                               | content |
      | folderToShare/inside.txt           | inside  |
      | folderToShare/second.txt           | second  |
      | folderToShare/subFolder/nested.txt | nested  |


  Scenario: guest downloads the archive of the shared folders of different users
    Given user "Brian" has been created with default attributes
    And user "Brian" has created folder "brianFolder"
    And user "Brian" has uploaded file with content "from Brian" to "brianFolder/brian.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | brianFolder       |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When guest "guest@example.org" downloads the archive of these resources using the archiver:
      | resource      | space    | owner |
      | folderToShare | Personal | Alice |
      | brianFolder   | Personal | Brian |
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name                     | content    |
      | folderToShare/inside.txt | inside     |
      | brianFolder/brian.txt    | from Brian |


  Scenario: guest cannot download the archive of resources outside the shared folder
    Given user "Alice" has uploaded file with content "private" to "private.txt"
    When guest "guest@example.org" downloads the archive of space "Personal" of user "Alice" using the archiver
    Then the HTTP status code should be "404"
    When guest "guest@example.org" downloads the archive of these resources using the archiver:
      | resource    | space    | owner |
      | private.txt | Personal | Alice |
    Then the HTTP status code should be "404"
    When guest "guest@example.org" downloads the archive of these resources using the archiver:
      | resource      | space    | owner |
      | folderToShare | Personal | Alice |
      | private.txt   | Personal | Alice |
    Then the HTTP status code should be "404"


  Scenario: guest copies and moves files between the shares of different users
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "editorFolder"
    And user "Alice" has created folder "viewerFolder"
    And user "Alice" has uploaded file with content "viewer" to "viewerFolder/viewer.txt"
    And user "Brian" has created folder "brianFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | editorFolder      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Editor            |
    And user "Alice" has sent the following resource share invitation:
      | resource        | viewerFolder      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And user "Brian" has sent the following resource share invitation:
      | resource        | brianFolder       |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Editor            |
    When guest "guest@example.org" copies "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to folder "editorFolder" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "201"
    When guest "guest@example.org" copies "viewer.txt" inside folder "viewerFolder" of space "Personal" of user "Alice" to folder "folderToShare" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "201"
    When guest "guest@example.org" copies "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to folder "viewerFolder" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    When guest "guest@example.org" moves "viewer.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to folder "editorFolder" of space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "201"
    When guest "guest@example.org" moves "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to folder "brianFolder" of space "Personal" of user "Brian" using the WebDAV API
    Then the HTTP status code should be "502"
    When guest "guest@example.org" copies "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to folder "brianFolder" of space "Personal" of user "Brian" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the content of the file "editorFolder/inside.txt" of the space "Personal" should be "inside"
    And for user "Alice" the content of the file "editorFolder/viewer.txt" of the space "Personal" should be "viewer"
    And for user "Brian" the content of the file "brianFolder/inside.txt" of the space "Personal" should be "inside"
    And as "Alice" file "folderToShare/inside.txt" should exist
    And as "Alice" file "folderToShare/viewer.txt" should not exist
    And as "Alice" file "viewerFolder/viewer.txt" should exist
    And as "Alice" file "viewerFolder/inside.txt" should not exist

  @issue-3767
  Scenario: guest moves a file into a share of another user over an existing file
    Given user "Brian" has been created with default attributes
    And user "Brian" has created folder "brianFolder"
    And user "Brian" has uploaded file with content "from Brian" to "brianFolder/inside.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | brianFolder       |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Editor            |
    When guest "guest@example.org" moves "inside.txt" inside folder "folderToShare" of space "Personal" of user "Alice" to folder "brianFolder" of space "Personal" of user "Brian" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Brian" the content of the file "brianFolder/inside.txt" of the space "Personal" should be "from Brian"
    And for user "Alice" the content of the file "folderToShare/inside.txt" of the space "Personal" should be "inside"

