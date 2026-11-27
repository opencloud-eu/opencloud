@skipOnReva
Feature: moving a share inside another share
  As a user
  I want to move a shared resource inside another shared resource
  So that I have full flexibility when managing resources

  Background:
    Given using OCS API version "1"
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has created folder "folderA"
    And user "Alice" has created folder "folderB"
    And user "Alice" has uploaded file with content "text A" to "/folderA/fileA.txt"
    And user "Alice" has uploaded file with content "text B" to "/folderB/fileB.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderA  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderB  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |


  Scenario Outline: share receiver cannot move a whole share inside another share
    Given using <dav-path-version> DAV path
    When user "Brian" moves folder "Shares/folderB" to "Shares/folderA/folderB" using the WebDAV API
    Then the HTTP status code should be "502"
    And as "Alice" folder "/folderB" should exist
    And as "Brian" folder "/Shares/folderB" should exist
    And as "Alice" file "/folderB/fileB.txt" should exist
    And as "Brian" file "/Shares/folderB/fileB.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: share owner moves a whole share inside another share
    Given using <dav-path-version> DAV path
    When user "Alice" moves folder "folderB" to "folderA/folderB" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "/folderB" should not exist
    And as "Alice" folder "/folderA/folderB" should exist
    And as "Brian" folder "/Shares/folderB" should exist
    And as "Alice" file "/folderA/folderB/fileB.txt" should exist
    And as "Brian" file "/Shares/folderA/folderB/fileB.txt" should exist
    And as "Brian" file "/Shares/folderB/fileB.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: share receiver moves a local folder inside a received share (local folder does not have a share in it)
    Given using <dav-path-version> DAV path
    And user "Brian" has created folder "localFolder"
    And user "Brian" has created folder "localFolder/subFolder"
    And user "Brian" has uploaded file with content "local text" to "/localFolder/localFile.txt"
    When user "Brian" moves folder "localFolder" to "Shares/folderA/localFolder" using the WebDAV API
    Then the HTTP status code should be "502"
    And as "Brian" folder "/Shares/folderA/localFolder" should not exist
    And as "Alice" folder "/folderA/localFolder" should not exist
    And as "Brian" folder "/localFolder" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: share receiver tries to move a whole share inside a local folder
    Given using <dav-path-version> DAV path
    And user "Brian" has created folder "localFolder"
    And user "Brian" has uploaded file with content "local text" to "/localFolder/localFile.txt"
    When user "Brian" moves folder "Shares/folderB" to "localFolder/folderB" using the WebDAV API
    Then the HTTP status code should be "502"
    And as "Alice" file "/folderB/fileB.txt" should exist
    And as "Brian" file "/Shares/folderB/fileB.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
