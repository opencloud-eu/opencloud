@env-config @skipOnReva
Feature: checking file versions
  As a user
  I want the versions of files to be available
  So that I can manage the changes made to the files

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario: check version number of a file when versioning is disabled
    Given the config "OC_DISABLE_VERSIONING" has been set to "true"
    And user "Alice" has uploaded file with content "test file version 1" to "/testfile.txt"
    And user "Alice" has uploaded file with content "test file version 2" to "/testfile.txt"
    When user "Alice" gets the number of versions of file "/testfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "0"


  Scenario: file version number should not be added after disabling versioning
    Given user "Alice" has uploaded file with content "test file version 1" to "/testfile.txt"
    And user "Alice" has uploaded file with content "test file version 2" to "/testfile.txt"
    And the config "OC_DISABLE_VERSIONING" has been set to "true"
    And user "Alice" has uploaded file with content "test file version 3" to "/testfile.txt"
    And user "Alice" has uploaded file with content "test file version 4" to "/testfile.txt"
    When user "Alice" gets the number of versions of file "/testfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"


  Scenario Outline: sharee tries to check version number of a file shared from project space when versioning is disabled
    Given the config "OC_DISABLE_VERSIONING" has been set to "true"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project1" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "Project1" with content "hello world version 1" to "text.txt"
    And user "Alice" has uploaded a file inside space "Project1" with content "hello world version 1.1" to "text.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | text.txt |
      | space           | Project1 |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | <role>   |
    When user "Brian" tries to get the number of versions of file "/text.txt" from space "Shares"
    Then the HTTP status code should be "403"
    Examples:
      | role        |
      | File Editor |
      | Viewer      |


  Scenario Outline: sharee tries to check version number of a file shared from personal space when versioning is disabled
    Given the config "OC_DISABLE_VERSIONING" has been set to "true"
    And user "Alice" has uploaded file with content "test file version 2" to "/text.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | text.txt          |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | <permissionsRole> |
    When user "Brian" tries to get the number of versions of file "/text.txt" from space "Shares"
    Then the HTTP status code should be "403"
    Examples:
      | permissionsRole |
      | File Editor     |
      | Viewer          |


  Scenario: check file version number after disabling versioning, creating versions and then enabling versioning
    Given the config "OC_DISABLE_VERSIONING" has been set to "true"
    And user "Alice" has uploaded file with content "test file version 1" to "/testfile.txt"
    And user "Alice" has uploaded file with content "test file version 2" to "/testfile.txt"
    And the config "OC_DISABLE_VERSIONING" has been set to "false"
    And user "Alice" has uploaded file with content "test file version 3" to "/testfile.txt"
    And user "Alice" has uploaded file with content "test file version 4" to "/testfile.txt"
    When user "Alice" gets the number of versions of file "/testfile.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "2"
