Feature: search sharees
  As a user
  I want to search sharees
  So that I can find them quickly

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | sharee1  |
    And group "ShareeGroup" has been created
    And group "ShareeGroup2" has been created
    And user "Alice" has been added to group "ShareeGroup2"


  Scenario Outline: search without exact match
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | sharee |
      | itemType | file   |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be empty
    And the "users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be
      | ShareeGroup  | 1 | ShareeGroup  | ShareeGroup  |
      | ShareeGroup2 | 1 | ShareeGroup2 | ShareeGroup2 |
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search without exact match not-exact casing
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | sHaRee |
      | itemType | file   |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be empty
    And the "users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be
      | ShareeGroup  | 1 | ShareeGroup  | ShareeGroup  |
      | ShareeGroup2 | 1 | ShareeGroup2 | ShareeGroup2 |
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search only with group members - allowed
    Given using OCS API version "<ocs-api-version>"
    And user "Sharee1" has been added to group "ShareeGroup2"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | sharee |
      | itemType | file   |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be empty
    And the "users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be
      | ShareeGroup  | 1 | ShareeGroup  | ShareeGroup  |
      | ShareeGroup2 | 1 | ShareeGroup2 | ShareeGroup2 |
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search with exact match
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | Sharee1 |
      | itemType | file    |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
    And the "users" sharees returned should be empty
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be empty
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search with exact match not-exact casing
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | sharee1 |
      | itemType | file    |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
    And the "users" sharees returned should be empty
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be empty
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search with exact match not-exact casing group
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | shareegroup2 |
      | itemType | file         |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be empty
    And the "users" sharees returned should be empty
    And the "exact groups" sharees returned should be
      | ShareeGroup2 | 1 | ShareeGroup2 | ShareeGroup2 |
    And the "groups" sharees returned should be empty
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search with "self"
    Given using OCS API version "<ocs-api-version>"
    When user "Sharee1" gets the sharees using the sharing API with parameters
      | search   | Sharee1 |
      | itemType | file    |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
    And the "users" sharees returned should be empty
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be empty
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: enumerate only group members - only show partial results from member of groups
    Given using OCS API version "<ocs-api-version>"
    And these users have been created with default attributes:
      | username | displayname |
      | another  | Another     |
    And user "Another" has been added to group "ShareeGroup2"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | anot |
      | itemType | file |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be empty
    And the "users" sharees returned should be
      | Another | 0 | another | another |
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be empty
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: search without exact match such that the search string matches the user getting the sharees
    Given user "sharee2" has been created with default attributes
    And using OCS API version "<ocs-api-version>"
    When user "sharee1" gets the sharees using the sharing API with parameters
      | search   | sharee |
      | itemType | file   |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be empty
    And the "users" sharees returned should be
      | Sharee One | 0 | sharee1 | sharee1 |
      | Sharee Two | 0 | sharee2 | sharee2 |
    And the "exact groups" sharees returned should be empty
    And the "groups" sharees returned should be
      | ShareeGroup  | 1 | ShareeGroup  | ShareeGroup  |
      | ShareeGroup2 | 1 | ShareeGroup2 | ShareeGroup2 |
    And the "exact remotes" sharees returned should be empty
    And the "remotes" sharees returned should be empty
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @env-config
  Scenario Outline: search other users when OC_SHOW_USER_EMAIL_IN_RESULTS config is enabled
    Given user "Brian" has been created with default attributes
    And the config "OC_SHOW_USER_EMAIL_IN_RESULTS" has been set to "true"
    And using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | Brian |
      | itemType | file  |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be
      | Brian Murphy | 0 | Brian | brian@example.org |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @env-config
  Scenario Outline: search other users when OC_SHOW_USER_EMAIL_IN_RESULTS config is disabled
    Given user "Brian" has been created with default attributes
    And the config "OC_SHOW_USER_EMAIL_IN_RESULTS" has been set to "false"
    And using OCS API version "<ocs-api-version>"
    When user "Alice" gets the sharees using the sharing API with parameters
      | search   | Brian |
      | itemType | file  |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the "exact users" sharees returned should be
      | Brian Murphy | 0 | Brian | Brian |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
