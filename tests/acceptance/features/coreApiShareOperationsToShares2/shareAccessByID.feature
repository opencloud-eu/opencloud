@skipOnReva
Feature: share access by ID
  As an API consumer (app)
  I want to access a share by its id
  So that the app can more easily manage shares

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: get a share with a valid share ID
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    When user "Alice" shares file "textfile0.txt" with user "Brian" using the sharing API
    And user "Alice" gets share with id "%last_share_id%" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | share_with             | %username%            |
      | share_with_displayname | %displayname%         |
      | file_target            | /Shares/textfile0.txt |
      | path                   | /textfile0.txt        |
      | permissions            | read,update           |
      | uid_owner              | %username%            |
      | displayname_owner      | %displayname%         |
      | item_type              | file                  |
      | mimetype               | text/plain            |
      | storage_id             | ANY_VALUE             |
      | share_type             | user                  |
    And the content of file "/Shares/textfile0.txt" for user "Brian" should be "OpenCloud test text file 0"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: get a share with an invalid share id
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" gets share with id "<share-id>" using the sharing API
    Then the OCS status code should be "404"
    And the HTTP status code should be "<http-status-code>"
    And the API should not return any data
    Examples:
      | ocs-api-version | share-id   | http-status-code |
      | 1               | 2333311    | 200              |
      | 2               | 2333311    | 404              |
      | 1               | helloshare | 200              |
      | 2               | helloshare | 404              |
      | 1               | $#@r3      | 200              |
      | 2               | $#@r3      | 404              |
      | 1               | 0          | 200              |
      | 2               | 0          | 404              |


  Scenario Outline: accept a share using the share Id
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    When user "Alice" shares file "textfile0.txt" with user "Brian" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should see the following elements
      | /Shares/textfile0.txt |
    And the sharing API should report to user "Brian" that these shares are in the accepted state
      | path                  |
      | /Shares/textfile0.txt |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: accept a share using the invalid share Id
    Given using OCS API version "<ocs-api-version>"
    When user "Brian" accepts share with ID "<share-id>" using the sharing API
    Then the OCS status code should be "998"
    And the HTTP status code should be "<http-status-code>"
    And the API should not return any data
    Examples:
      | ocs-api-version | share-id   | http-status-code |
      | 1               | 2333311    | 200              |
      | 2               | 2333311    | 404              |
      | 1               | helloshare | 200              |
      | 2               | helloshare | 404              |
      | 1               | $#@r3      | 200              |
      | 2               | $#@r3      | 404              |
      | 1               | 0          | 200              |
      | 2               | 0          | 404              |


  Scenario Outline: accept a share using empty share Id
    Given using OCS API version "<ocs-api-version>"
    When user "Brian" accepts share with ID "" using the sharing API
    Then the HTTP status code should be "<http-status-code>"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 405              |
      | 2               | 405              |


  Scenario Outline: decline a share using the share Id
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    And using SharingNG
    When user "Brian" declines share with ID "%last_share_id%" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should not see the following elements
      | /Shares/textfile0.txt |
    And the sharing API should report to user "Brian" that these shares are in the declined state
      | path          |
      | textfile0.txt |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: decline a share using empty share Id
    Given using OCS API version "<ocs-api-version>"
    When user "Brian" declines share with ID "" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "<http-status-code>"
    And the API should not return any data
    Examples:
      | ocs-api-version | http-status-code | ocs-status-code |
      | 1               | 200              | 998             |
      | 2               | 404              | 998             |
