Feature: delete folder
  As a user
  I want to be able to delete folders
  So that I can quickly remove unwanted data

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" creates folder "/PARENT" using the WebDAV API


  Scenario Outline: delete a folder
    Given using <dav-path-version> DAV path
    When user "Alice" deletes folder "/PARENT" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/PARENT" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: delete a folder when 2 folder exist with different case
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/parent"
    When user "Alice" deletes folder "/PARENT" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/PARENT" should not exist
    But as "Alice" folder "/parent" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: delete a sub-folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/PARENT/CHILD"
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/PARENT/parent.txt"
    When user "Alice" deletes folder "/PARENT/CHILD" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/PARENT/CHILD" should not exist
    But as "Alice" folder "/PARENT" should exist
    And as "Alice" file "/PARENT/parent.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: deleting folder with dot in the name
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    When user "Alice" deletes folder "<folder-name>" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "<folder-name>" should not exist
    Examples:
      | dav-path-version | folder-name |
      | old              | /fo.        |
      | old              | /fo.1       |
      | old              | /fo...1..   |
      | old              | /...        |
      | old              | /..fo       |
      | old              | /fo.xyz     |
      | old              | /fo.exe     |
      | new              | /fo.        |
      | new              | /fo.1       |
      | new              | /fo...1..   |
      | new              | /...        |
      | new              | /..fo       |
      | new              | /fo.xyz     |
      | new              | /fo.exe     |
      | spaces           | /fo.        |
      | spaces           | /fo.1       |
      | spaces           | /fo...1..   |
      | spaces           | /...        |
      | spaces           | /..fo       |
      | spaces           | /fo.xyz     |
      | spaces           | /fo.exe     |
