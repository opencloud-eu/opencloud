@skipOnReva @owncloud-issue-1289 @owncloud-issue-1328
Feature: sharing
  As a user
  I want to move shares that I received
  So that I can organise them according to my needs

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: move files between shares by same user
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "share1"
    And user "Alice" has created folder "share2"
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has moved file "textfile0.txt" to "share1/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /share1  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "share1" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | /share2  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "share2" synced
    When user "Brian" moves file "/Shares/share1/textfile0.txt" to "/Shares/share2/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "502"
    And as "Brian" file "/Shares/share1/textfile0.txt" should exist
    And as "Alice" file "share1/textfile0.txt" should exist
    But as "Brian" file "/Shares/share2/textfile0.txt" should not exist
    And as "Alice" file "share2/textfile0.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: overwrite a received file share
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "this is the old content" to "/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    And user "Brian" has a share "textfile1.txt" synced
    When user "Brian" uploads file with content "this is a new content" to "/Shares/textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Brian" file "Shares/textfile1.txt" should exist
    And the content of file "Shares/textfile1.txt" for user "Brian" should be "this is a new content"
    And the content of file "textfile1.txt" for user "Alice" should be "this is a new content"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

