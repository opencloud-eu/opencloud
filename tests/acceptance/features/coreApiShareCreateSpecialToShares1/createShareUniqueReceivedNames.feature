@skipOnReva
Feature: resources shared with the same name are received with unique names
  As a user
  I want to share resources with same name
  So that I can make sure the naming is handled properly by the server

  Background:
    Given using OCS API version "1"
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |

  @smokeTest @owncloud-issue-2131
  Scenario: unique target names for incoming shares
    Given user "Alice" has created folder "/foo"
    And user "Brian" has created folder "/foo"
    When user "Alice" shares folder "/foo" with user "Carol" using the sharing API
    And user "Brian" shares folder "/foo" with user "Carol" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Carol" should see the following elements
      | Shares/foo      |
      | /Shares/foo (1) |

  @smokeTest @owncloud-issue-2131
  Scenario: unique target names for incoming shares when auto-accepting is disabled
    Given user "Brian" has disabled auto-accepting
    And user "Alice" has created folder "/foo"
    And user "Brian" has created folder "/foo"
    When user "Alice" shares folder "/foo" with user "Carol" using the sharing API
    And user "Brian" shares folder "/foo" with user "Carol" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Carol" should see the following elements
      | Shares/foo      |
      | /Shares/foo (1) |
