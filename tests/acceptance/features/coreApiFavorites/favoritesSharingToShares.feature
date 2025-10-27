@skipOnReva
Feature: favorite
  As a user
  I want to favorite the shared resources
  So that I can access them quickly

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/parent.txt"


  Scenario Outline: favorite a file inside of a received share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" favorites element "/Shares/PARENT/parent.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" file "/Shares/PARENT/parent.txt" should be favorited
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: favorite a folder inside of a received share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "/PARENT/sub-folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" favorites element "/Shares/PARENT/sub-folder" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" folder "/Shares/PARENT/sub-folder" should be favorited
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: favorite a received share itself
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" favorites element "/Shares/PARENT" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" folder "/Shares/PARENT" should be favorited
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1228
  Scenario Outline: moving a favorite file out of a share keeps favorite state
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    And user "Brian" has favorited element "/Shares/PARENT/parent.txt"
    When user "Brian" moves file "/Shares/PARENT/parent.txt" to "/taken_out.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/taken_out.txt" should exist
    And as user "Brian" file "/taken_out.txt" should be favorited
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: sharee file favorite state should not change the favorite state of sharer
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT/parent.txt |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | File Editor       |
    And user "Brian" has a share "parent.txt" synced
    When user "Brian" favorites element "/Shares/parent.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Brian" file "/Shares/parent.txt" should be favorited
    And as user "Alice" file "/PARENT/parent.txt" should not be favorited
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
