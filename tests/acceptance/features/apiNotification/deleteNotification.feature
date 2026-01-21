Feature: Delete notification
  As a user
  I want to delete notifications
  So that I can filter notifications

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "other data" to "/textfile1.txt"
    And user "Alice" has created folder "my_data"
    And user "Alice" has sent the following resource share invitation:
      | resource        | my_data  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |


  Scenario: delete a notification
    When user "Brian" deletes a notification related to resource "my_data" with subject "Resource shared"
    Then the HTTP status code should be "200"
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                    |
      | Alice Hansen shared textfile1.txt with you |
    But user "Brian" should not get a notification related to resource "my_data" with subject "Resource shared"


  Scenario: delete all notifications
    When user "Brian" deletes all notifications
    Then the HTTP status code should be "200"
    And user "Brian" should not have any notification
