@skipOnReva
Feature: sharing
  As a user
  I want to get all the shares
  So that I can know I have proper access to them

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |

  @smokeTest @owncloud-issue-1258
  Scenario Outline: getting all shares from a user
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "some data" to "/file_to_share.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | file_to_share.txt |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | Viewer            |
    When user "Alice" gets all shares shared by her using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And file "/Shares/file_to_share.txt" should be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1319
  Scenario Outline: getting all shares of a user using another user
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When the administrator gets all shares shared by him using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And file "/Shares/textfile0.txt" should not be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest
  Scenario Outline: getting all shares of a file
    Given using OCS API version "<ocs-api-version>"
    And these users have been created with default attributes:
      | username |
      | Carol    |
      | David    |
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Carol         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" gets all the shares of the file "textfile0.txt" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should be included in the response
    And user "Carol" should be included in the response
    And user "David" should not be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest @owncloud-issue-1226 @owncloud-issue-1270 @owncloud-issue-1271
  Scenario Outline: getting share info of a share
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "some data" to "/file_to_share.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | file_to_share.txt |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | File Editor       |
    And using SharingNG
    When user "Alice" gets the info of the last share using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | id                     | A_STRING                  |
      | item_type              | file                      |
      | item_source            | A_STRING                  |
      | share_type             | user                      |
      | share_with             | %username%                |
      | file_source            | A_STRING                  |
      | file_target            | /Shares/file_to_share.txt |
      | path                   | /file_to_share.txt        |
      | permissions            | read,update               |
      | stime                  | A_NUMBER                  |
      | storage                | A_STRING                  |
      | mail_send              | 0                         |
      | uid_owner              | %username%                |
      | share_with_displayname | %displayname%             |
      | displayname_owner      | %displayname%             |
      | mimetype               | text/plain                |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1233
  Scenario Outline: get a share with a user that didn't receive the share
    Given using OCS API version "<ocs-api-version>"
    And user "Carol" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And using SharingNG
    When user "Carol" gets the info of the last share using the sharing API
    Then the OCS status code should be "404"
    And the HTTP status code should be "<http_status_code>"
    Examples:
      | ocs-api-version | http_status_code |
      | 1               | 200              |
      | 2               | 404              |

  @owncloud-issue-1289
  Scenario: share a folder to a group, and remove user from that group
    Given using OCS API version "1"
    And user "Carol" has been created with default attributes
    And group "group0" has been created
    And user "Brian" has been added to group "group0"
    And user "Carol" has been added to group "group0"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/parent.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /PARENT  |
      | space           | Personal |
      | sharee          | group0   |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    When the administrator removes user "Carol" from group "group0" using the provisioning API
    Then the HTTP status code should be "204"
    And user "Brian" should see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
    But user "Carol" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |

  @smokeTest @owncloud-issue-1226 @owncloud-issue-1270 @owncloud-issue-1271 @owncloud-issue-1231
  Scenario Outline: getting share info of a share shared from inside folder
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/file_to_share.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /PARENT/file_to_share.txt |
      | space           | Personal                  |
      | sharee          | Brian                     |
      | shareType       | user                      |
      | permissionsRole | File Editor               |
    When user "Alice" gets all shares shared by her using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | id                     | A_STRING                  |
      | item_type              | file                      |
      | item_source            | A_STRING                  |
      | share_type             | user                      |
      | share_with             | %username%                |
      | file_source            | A_STRING                  |
      | file_target            | /Shares/file_to_share.txt |
      | path                   | /PARENT/file_to_share.txt |
      | permissions            | read,update               |
      | stime                  | A_NUMBER                  |
      | storage                | A_STRING                  |
      | mail_send              | 0                         |
      | uid_owner              | %username%                |
      | share_with_displayname | %displayname%             |
      | displayname_owner      | %displayname%             |
      | mimetype               | text/plain                |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
