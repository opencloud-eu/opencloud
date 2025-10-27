@owncloud-issue-1289 @owncloud-issue-1328 @skipOnReva
Feature: updating shares to users and groups that have the same name
  As a user
  I want to update share permissions
  So that I can decide what resources can be shared with which permission

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And group "Brian" has been created
    And user "Carol" has been added to group "Brian"
    And user "Alice" has created folder "/TMP"
    And user "Alice" has uploaded file with content "Random data" to "/TMP/randomfile.txt"


  Scenario Outline: update permissions of a user share with a user and a group having the same name
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | TMP      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Carol" has a share "TMP" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | TMP      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "TMP" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | read |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the content of file "/Shares/TMP/randomfile.txt" for user "Brian" should be "Random data"
    And the content of file "/Shares/TMP/randomfile.txt" for user "Carol" should be "Random data"
    And user "Carol" should be able to upload file "filesForUpload/textfile.txt" to "Shares/TMP/textfile-by-Carol.txt"
    But user "Brian" should not be able to upload file "filesForUpload/textfile.txt" to "Shares/TMP/textfile-by-Brian.txt"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: update permissions of a group share with a user and a group having the same name
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | TMP      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "TMP" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | TMP      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Carol" has a share "TMP" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | read |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the content of file "/Shares/TMP/randomfile.txt" for user "Brian" should be "Random data"
    And the content of file "/Shares/TMP/randomfile.txt" for user "Carol" should be "Random data"
    And user "Brian" should be able to upload file "filesForUpload/textfile.txt" to "Shares/TMP/textfile-by-Carol.txt"
    But user "Carol" should not be able to upload file "filesForUpload/textfile.txt" to "Shares/TMP/textfile-by-Brian.txt"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
