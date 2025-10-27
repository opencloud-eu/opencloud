@skipOnReva @owncloud-issue-1327
Feature: shares are received in the default folder for received shares
  As a user
  I want to share the default Shares folder
  So that I can make sure it does not work

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: do not allow sharing of the entire share folder
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "FOLDER"
    When user "Alice" shares folder "/FOLDER" with user "Brian" using the sharing API
    And user "Brian" declines share "/Shares/FOLDER" offered by user "Alice" using the sharing API
    And user "Brian" shares folder "/Shares" with user "Alice" using the sharing API
    Then the OCS status code of responses on each endpoint should be "<ocs-status-code>" respectively
    And the HTTP status code of responses on each endpoint should be "<http-status-code>" respectively
    Examples:
      | ocs-api-version | ocs-status-code | http-status-code |
      | 1               | 100, 100, 403   | 200, 200, 200    |
      | 2               | 200, 200, 403   | 200, 200, 403    |
