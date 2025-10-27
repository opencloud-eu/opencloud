@ocm
Feature: create invitation
  As a user
  I can create an invitations and send it to the person I want to share with

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: user creates invitation
    Given using server "LOCAL"
    When "Alice" creates the federation share invitation
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expiration",
          "token"
        ],
        "properties": {
          "expiration": {
            "type": "integer",
            "pattern": "^[0-9]{10}$"
          },
          "token": {
            "type": "string",
            "pattern": "%fed_invitation_token%"
          }
        }
      }
      """

  @owncloud-issue-9591
  Scenario: user creates invitation with valid email and description
    Given using server "LOCAL"
    When "Alice" creates the federation share invitation with email "brian@example.com" and description "a share invitation from Alice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "expiration",
          "token",
          "description"
        ],
        "properties": {
          "expiration": {
            "type": "integer",
            "pattern": "^[0-9]{10}$"
          },
          "token": {
            "type": "string",
            "pattern": "%fed_invitation_token%"
          },
          "description": {
            "const": "a share invitation from Alice"
          }
        }
      }
      """


  Scenario Outline: user creates invitation with valid/invalid email
    Given using server "LOCAL"
    When "Alice" creates the federation share invitation with email "<email>" and description "a share invitation from Alice"
    Then the HTTP status code should be "<code>"
    Examples:
      | email                             | code |
      | user@subdomain.example.longdomain | 200  |
      | user.bob+123@domain.test-123.com  | 200  |
      | user.example.com                  | 400  |
      | user@.com                         | 400  |
      | @domain.com                       | 400  |
      | user@domain..com                  | 400  |

  @email @owncloud-issue-10059
  Scenario: federated user gets an email notification if their email was specified when creating the federation share invitation
    Given using server "REMOTE"
    And user "David" has been created with default attributes
    And using server "LOCAL"
    When "Alice" has created the federation share invitation with email "david@example.com" and description "a share invitation from Alice"
    And user "David" should have received the following email from user "Alice" ignoring whitespaces
      """
      Hi,

      Alice Hansen (alice@example.org) wants to start sharing collaboration resources with you.

      Please visit your federation settings and use the following details:
        Token: %fed_invitation_token%
        ProviderDomain: %local_base_url%
      """

  @env-config
  Scenario: user cannot see expired invitation tokens
    Given using server "LOCAL"
    And the config "OCM_OCM_INVITE_MANAGER_TOKEN_EXPIRATION" has been set to "1s"
    And "Alice" has created the federation share invitation
    When the user waits "2" seconds for the invitation token to expire
    And "Alice" lists the created invitations
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "array",
        "minItems": 0,
        "maxItems": 0
      }
      """


  Scenario: user lists created invitation
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    When "Alice" lists the created invitations
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "array",
        "minItems": 1,
        "maxItems": 1,
        "items": {
          "type": "object",
          "required": [
            "expiration",
            "token"
          ],
          "properties": {
            "expiration": {
              "type": "integer",
              "pattern": "^[0-9]{10}$"
            },
            "token": {
              "type": "string",
              "pattern": "%fed_invitation_token%"
            }
          }
        }
      }
      """

  @owncloud-issue-9591
  Scenario: user lists invitation created with valid email and description
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation with email "brian@example.com" and description "a share invitation from Alice"
    When "Alice" lists the created invitations
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "array",
        "minItems": 1,
        "maxItems": 1,
        "items": {
          "type": "object",
          "required": [
            "expiration",
            "token",
            "description"
          ],
          "properties": {
            "expiration": {
              "type": "integer",
              "pattern": "^[0-9]{10}$"
            },
            "token": {
              "type": "string",
              "pattern": "%fed_invitation_token%"
            },
            "description": {
              "const": "a share invitation from Alice"
            }
          }
        }
      }
      """
