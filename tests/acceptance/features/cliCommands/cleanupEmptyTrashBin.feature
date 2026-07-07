@env-config @skipOnOpencloud-posix-Storage
Feature: delete empty trash bin folder via CLI command


  Scenario: delete empty trashbin folders
    Given user "Alice" has been created with default attributes
    And user "Alice" has created the following folders
      | path              |
      | folder-to-delete  |
      | folder-to-restore |
    And user "Alice" has deleted the following resources
      | path              |
      | folder-to-delete  |
      | folder-to-restore |
    When the administrator deletes the empty trashbin folders using the CLI
    Then the command should be successful
