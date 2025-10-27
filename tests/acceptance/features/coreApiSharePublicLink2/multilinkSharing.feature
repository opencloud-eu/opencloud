@skipOnReva
Feature: multi-link sharing
  As a user
  I want to create multiple public links for a single resource
  So that I can share them with various permissions and/or different groups of people

  Background:
    Given user "Alice" has been created with default attributes

  @smokeTest
  Scenario Outline: creating three public shares of a folder
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "FOLDER"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink1 |
      | password           | %public%    |
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink2 |
      | password           | %public%    |
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink3 |
      | password           | %public%    |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | read |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as user "Alice" the public shares of folder "/FOLDER" should be
      | path    | permissions | name        |
      | /FOLDER | 15          | sharedlink2 |
      | /FOLDER | 15          | sharedlink1 |
      | /FOLDER | 1           | sharedlink3 |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: creating three public shares of a file
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink1   |
      | password           | %public%      |
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink2   |
      | password           | %public%      |
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink3   |
      | password           | %public%      |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | read |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as user "Alice" the public shares of file "/textfile0.txt" should be
      | path           | permissions | name        |
      | /textfile0.txt | 1           | sharedlink2 |
      | /textfile0.txt | 1           | sharedlink1 |
      | /textfile0.txt | 1           | sharedlink3 |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: check that updating password doesn't remove name of links
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "FOLDER"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink1 |
      | password           | %public%    |
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink2 |
      | password           | %public%    |
    When user "Alice" updates the last public link share using the sharing API with
      | password | New-StronPass1 |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as user "Alice" the public shares of folder "/FOLDER" should be
      | path    | permissions | name        |
      | /FOLDER | 15          | sharedlink2 |
      | /FOLDER | 15          | sharedlink1 |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: deleting a file also deletes its public links
    Given using OCS API version "1"
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink1   |
      | password           | %public%      |
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink2   |
      | password           | %public%      |
    And user "Alice" has deleted file "/textfile0.txt"
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as user "Alice" the file "/textfile0.txt" should not have any shares
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: deleting one public link share of a file doesn't affect the rest
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink1   |
      | password           | %public%      |
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink2   |
      | password           | %public%      |
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink3   |
      | password           | %public%      |
    When user "Alice" deletes public link share named "sharedlink2" in file "/textfile0.txt" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as user "Alice" the public shares of file "/textfile0.txt" should be
      | path           | permissions | name        |
      | /textfile0.txt | 1           | sharedlink1 |
      | /textfile0.txt | 1           | sharedlink3 |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: overwriting a file doesn't remove its public shares
    Given using OCS API version "1"
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink1   |
      | password           | %public%      |
    And user "Alice" has created the following resource link share:
      | resource           | textfile0.txt |
      | space              | Personal      |
      | permissionsRole    | view          |
      | expirationDateTime | +3 days       |
      | displayName        | sharedlink2   |
      | password           | %public%      |
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as user "Alice" the public shares of file "/textfile0.txt" should be
      | path           | permissions | name        |
      | /textfile0.txt | 1           | sharedlink1 |
      | /textfile0.txt | 1           | sharedlink2 |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1251
  Scenario Outline: renaming a folder doesn't remove its public shares
    Given using OCS API version "1"
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink1 |
      | password           | %public%    |
    And user "Alice" has created the following resource link share:
      | resource           | FOLDER      |
      | space              | Personal    |
      | permissionsRole    | edit        |
      | expirationDateTime | +3 days     |
      | displayName        | sharedlink2 |
      | password           | %public%    |
    When user "Alice" moves folder "/FOLDER" to "/FOLDER_RENAMED" using the WebDAV API
    Then the HTTP status code should be "201"
    And as user "Alice" the public shares of file "/FOLDER_RENAMED" should be
      | path            | permissions | name        |
      | /FOLDER_RENAMED | 15          | sharedlink1 |
      | /FOLDER_RENAMED | 15          | sharedlink2 |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
