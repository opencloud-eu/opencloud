@env-config
Feature: reindex space via CLI command
    As an admin
    I want to reindex space
    So that I can improve search performance by ensuring that the index is up-to-date

  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some data" to "textfile.txt"

  @issue-10329
  Scenario: reindex all spaces
    When the administrator reindexes all spaces using the CLI
    Then the command should be successful
    When user "Alice" searches for "textfile.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | textfile.txt |


  Scenario: reindex a space
    Given using spaces DAV path
    And user "Alice" has created the following tags for file "textfile.txt" of the space "new-space":
      | tag1 |
    And user "Alice" has removed the following tags for file "textfile.txt" of space "new-space":
      | tag1 |
    When the administrator reindexes a space "new-space" using the CLI
    Then the command should be successful
    When user "Alice" searches for "Tags:tag1" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result should contain "0" entries
