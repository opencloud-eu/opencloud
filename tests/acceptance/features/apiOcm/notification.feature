@ocm
Feature: federated users get in-app notifications about shares
  As a federated user
  I want to be notified when a user of another instance shares a resource with me or removes the share
  So that I know which federated resources I have access to

  Background:
    Given user "Alice" has been created with default attributes
    And using server "REMOTE"
    And user "Brian" has been created with default attributes
    And "Brian" has created the federation share invitation
    And using server "LOCAL"
    And "Alice" has accepted invitation
    And user "Alice" has created folder "folderToShare"


  Scenario: federated user gets a notification when a resource is shared
    When user "Alice" sends the following resource share invitation to federated user using the Graph API:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    Then the HTTP status code should be "200"
    And using server "REMOTE"
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                    |
      | Alice Hansen shared folderToShare with you |


  Scenario: federated user gets a notification when a share is removed
    Given user "Alice" has sent the following resource share invitation to federated user:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" removes the access of user "Brian" from resource "folderToShare" of space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    And using server "REMOTE"
    And user "Brian" should get a notification with subject "Resource unshared" and message:
      | message                                      |
      | Alice Hansen unshared folderToShare with you |


  Scenario: federated user does not get a notification for a removed share when the in-app notification is disabled
    Given using server "REMOTE"
    And user "Brian" has disabled notification for the following event using the settings API:
      | event             | Share Removed |
      | notificationTypes | in-app        |
    And using server "LOCAL"
    And user "Alice" has sent the following resource share invitation to federated user:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" removes the access of user "Brian" from resource "folderToShare" of space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    And using server "REMOTE"
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                    |
      | Alice Hansen shared folderToShare with you |
    But user "Brian" should not have a notification related to resource "folderToShare" with subject "Resource unshared"
