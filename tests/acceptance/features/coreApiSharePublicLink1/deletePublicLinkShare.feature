@skipOnReva
Feature: delete a public link share
  As a user
  I want to delete a public link
  So that the public won't have access to the resource inside it

  Background:
    Given user "Alice" has been created with default attributes

  @owncloud-issue-1275
  Scenario Outline: deleting a public link of a file
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "This is a test file" to "test-file.txt"
    And user "Alice" has created the following resource link share:
      | resource        | test-file.txt |
      | space           | Personal      |
      | permissionsRole | view          |
      | password        | %public%      |
      | displayName     | sharedlink    |
    When user "Alice" deletes public link share named "sharedlink" in file "test-file.txt" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as user "Alice" the file "test-file.txt" should not have any shares
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1275
  Scenario Outline: deleting a public link after renaming a file
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "This is a test file" to "test-file.txt"
    And user "Alice" has created the following resource link share:
      | resource        | test-file.txt |
      | space           | Personal      |
      | permissionsRole | view          |
      | password        | %public%      |
      | displayName     | sharedlink    |
    And user "Alice" has moved file "/test-file.txt" to "/renamed-test-file.txt"
    When user "Alice" deletes public link share named "sharedlink" in file "renamed-test-file.txt" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as user "Alice" the file "renamed-test-file.txt" should not have any shares
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1275
  Scenario Outline: deleting a public link of a folder
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "test-folder"
    And user "Alice" has created the following resource link share:
      | resource        | test-folder |
      | space           | Personal    |
      | permissionsRole | view        |
      | password        | %public%    |
      | displayName     | sharedlink  |
    When user "Alice" deletes public link share named "sharedlink" in folder "test-folder" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as user "Alice" the folder "test-folder" should not have any shares
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1275
  Scenario Outline: deleting a public link of a file in a folder
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "test-folder"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/test-folder/testfile.txt" using the WebDAV API
    And user "Alice" has created the following resource link share:
      | resource        | test-folder/testfile.txt |
      | space           | Personal                 |
      | permissionsRole | view                     |
      | password        | %public%                 |
      | displayName     | sharedlink               |
    And user "Alice" deletes public link share named "sharedlink" in file "/test-folder/testfile.txt" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as user "Alice" the file "/test-folder/testfile.txt" should not have any shares
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

