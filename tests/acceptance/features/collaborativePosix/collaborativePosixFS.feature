Feature: create a resources using collaborative posixfs

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: create folder
    Given user "Alice" has uploaded file with content "content" to "textfile.txt"
    When the administrator creates folder "myFolder" for user "Alice"
    Then the command should be successful
    When the administrator checks storage folder for user "Alice"
    Then the command output should contain "myFolder"
    And as "Alice" folder "/myFolder" should exist
