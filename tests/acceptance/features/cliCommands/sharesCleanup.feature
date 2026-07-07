@env-config
Feature: clean up orphaned shares via CLI command
  As an admin
  I want to clean up orphaned shares
  So that stale share entries do not accumulate in the share manager after a shared resource has been permanently deleted

  Background:
    Given user "Alice" has been created with default attributes
    And user "Brian" has been created with default attributes


  Scenario: clean up orphaned shares after a shared folder has been permanently deleted
    Given user "Alice" has created folder "/uploadFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has deleted folder "/uploadFolder"
    And user "Alice" has deleted the folder with original path "/uploadFolder" from the trashbin
    When the administrator cleans up orphaned shares using the CLI
    Then the command should be successful
    And the command output should contain "shared resource does not exist anymore. cleaning up shares"
    When user "Brian" lists the shares shared with her using the Graph API
    Then the HTTP status code should be "200"
