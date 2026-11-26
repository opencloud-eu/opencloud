@ocm
Feature: accepting invitation
  As a user
  I can accept invitations from users of other OpenCloud instances

  Background:
    Given user "Alice" has been created with default attributes
    And using server "REMOTE"
    And these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |


  Scenario: user accepts invitation
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    When using server "REMOTE"
    And "Brian" accepts the last federation share invitation
    Then the HTTP status code should be "200"


  Scenario: user accepts invitation sent with email and description
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation with email "brian@example.com" and description "a share invitation from Alice"
    When using server "REMOTE"
    And "Brian" accepts the last federation share invitation
    Then the HTTP status code should be "200"


  Scenario: two users can accept one invitation
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    When "Carol" accepts the last federation share invitation
    Then the HTTP status code should be "200"


  Scenario: user tries to accept the invitation twice
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    When "Brian" tries to accept the last federation share invitation
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "ALREADY_EXIST"
          },
          "message": {
            "const": "user already known"
          }
        }
      }
      """


  Scenario: users try to accept each other's invitation
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And "Brian" has created the federation share invitation
    When using server "LOCAL"
    And "Alice" tries to accept the last federation share invitation
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "ALREADY_EXIST"
          },
          "message": {
            "const": "user already known"
          }
        }
      }
      """

  @env-config
  Scenario: user cannot accept expired invitation tokens
    Given using server "LOCAL"
    And the config "OCM_OCM_INVITE_MANAGER_TOKEN_EXPIRATION" has been set to "1s"
    And "Alice" has created the federation share invitation
    When using server "REMOTE"
    And the user waits "2" seconds for the invitation token to expire
    And "Brian" tries to accept the last federation share invitation
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "INVALID_PARAMETER"
          },
          "message": {
            "const": "token not valid"
          }
        }
      }
      """


  Scenario: user cannot accept invalid invitation token
    Given using server "LOCAL"
    And "Alice" tries to accept the invitation with invalid token
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "RESOURCE_NOT_FOUND"
          },
          "message": {
            "const": "token not found"
          }
        }
      }
      """

