@skipOnReva
Feature: sharing
  As a user
  I want to share resources with other users
  So that they can have access to the resources

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"

  @smokeTest
  Scenario Outline: sharee can see the share
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And using SharingNG
    And user "Brian" has a share "textfile0.txt" synced
    When user "Brian" gets all the shares shared with him using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the last share_id should be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest
  Scenario Outline: sharee can see the filtered share
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "some data" to "/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And using SharingNG
    And user "Brian" has a share "textfile1.txt" synced
    And user "Brian" has a share "textfile0.txt" synced
    When user "Brian" gets all the shares shared with him that are received as file "/Shares/textfile1.txt" using the provisioning API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the last share_id should be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest @issue-1257
  Scenario Outline: sharee can't see the share that is filtered out
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "some data" to "/textfile1.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And using SharingNG
    And user "Brian" has a share "textfile0.txt" synced
    And user "Brian" has a share "textfile1.txt" synced
    When user "Brian" gets all the shares shared with him that are received as file "/Shares/textfile0.txt" using the provisioning API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the last share id should not be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest @issue-1289
  Scenario Outline: sharee can see the group share
    Given using OCS API version "<ocs-api-version>"
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And using SharingNG
    And user "Brian" has a share "textfile0.txt" synced
    When user "Brian" gets all the shares shared with him using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the last share_id should be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
