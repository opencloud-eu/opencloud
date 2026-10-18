Feature: renew a guest link
  As a guest without an account
  I want to request a new guest link and PIN when my link or session is no longer valid
  So that I can access the shared resource again

  Background:
    Given using SharingNG
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |

  @email
  Scenario: renewing without a session does nothing while the guest link is unused
    When guest "guest@example.org" requests a new guest link without a guest session using the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain a nonce cookie
    And guest "guest@example.org" should have received 1 email in total
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"

  @email
  Scenario: guest renews a used guest link without a session and receives a new link without a PIN
    Given guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" requests a new guest link without a guest session using the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain a guest session cookie
    And the response should not contain a nonce cookie
    And guest "guest@example.org" should have received the following email from user "Alice"
      """
      Hello,

      A new link to "textfile.txt", shared with you by %displayname%, has been requested.

      The link below is personal and can only be used once. Please do not forward it.

      The link is only valid for 30 minutes.

      Click here to view it: %base_url%/g/%guest_link_token%
      """
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie

  @email
  Scenario: guest renews the guest link with the guest session and receives a new link and a PIN
    Given guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" requests a new guest link with the guest session using the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain a guest session cookie
    And the response should contain a nonce cookie
    And guest "guest@example.org" should have received the following email from user "Alice"
      """
      Hello,

      A new link to "textfile.txt", shared with you by %displayname%, has been requested.

      Your PIN: %guest_pin%

      Open the link below or enter the PIN on the page where you requested the new link. The link and the PIN are personal and can only be used once. Please do not forward them.

      The link and the PIN are only valid for 30 minutes.

      Click here to view it: %base_url%/g/%guest_link_token%
      """

  @email
  Scenario: guest gets a session with the PIN from the renewal email
    Given guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    When guest "guest@example.org" verifies the PIN from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["permissionId", "expirationDateTime"],
        "properties": {
          "permissionId": { "const": "%last_share_id%" },
          "expirationDateTime": { "type": "string", "format": "date-time" }
        }
      }
      """
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "to share"

  @email
  Scenario: PIN cannot be verified without the nonce cookie of the browser that renewed
    Given guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    When guest "guest@example.org" verifies the PIN from the last email without the nonce cookie using the Graph API
    Then the HTTP status code should be "400"
    And the response should not contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType"],
        "properties": {
          "errorType": { "const": "invalidRequest" }
        }
      }
      """

  @email
  Scenario: verifying a wrong PIN fails
    Given guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    When guest "guest@example.org" verifies the PIN "000000" using the Graph API
    Then the HTTP status code should be "401"
    And the response should not contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType"],
        "properties": {
          "errorType": { "const": "pinInvalid" }
        }
      }
      """

  @email
  Scenario: wrong PINs are throttled but the guest link keeps working
    Given guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    And guest "guest@example.org" has tried to verify the PIN "000000"
    When guest "guest@example.org" verifies the PIN "000000" using the Graph API
    Then the HTTP status code should be "429"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType", "retryAfter"],
        "properties": {
          "errorType": { "const": "tooManyRequests" },
          "retryAfter": { "type": "integer", "minimum": 1 }
        }
      }
      """
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie

  @email
  Scenario: PIN can only be used once
    Given guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    And guest "guest@example.org" has verified the PIN from the last email
    When guest "guest@example.org" verifies the PIN from the last email using the Graph API
    Then the HTTP status code should be "400"
    And the response should not contain a guest session cookie

  @email
  Scenario: previous guest link is invalid after renewal
    Given guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    When guest "guest@example.org" verifies the previous guest link using the Graph API
    Then the HTTP status code should be "401"
    And the response should not contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType"],
        "properties": {
          "errorType": { "const": "tokenInvalid" }
        }
      }
      """

  @email @env-config
  Scenario: PIN of a previous renewal is invalid after renewing again
    Given the config "AUTH_GUEST_SESSION_QUOTA_COOLDOWN" has been set to "1ms"
    And guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    And guest "guest@example.org" has requested a new guest link with the guest session
    When guest "guest@example.org" verifies the PIN from the previous email using the Graph API
    Then the HTTP status code should be "401"
    And the response should not contain a guest session cookie
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType"],
        "properties": {
          "errorType": { "const": "pinInvalid" }
        }
      }
      """

  @email
  Scenario: guest session of another share does not authorize a renewal
    Given guest "guest@example.org" has verified the guest link from the last email
    And user "Alice" has uploaded file with content "other" to "otherFile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | otherFile.txt     |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has received a new guest link
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" requests a new guest link for resource "textfile.txt" of space "Personal" of user "Alice" with the guest session using the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain a nonce cookie
    And guest "guest@example.org" should have received the following email from user "Alice"
      """
      A new link to "textfile.txt", shared with you by %displayname%, has been requested.

      The link below is personal and can only be used once. Please do not forward it.
      """

  @email
  Scenario Outline: renewing the guest link after the share is removed fails
    Given guest "guest@example.org" has verified the guest link from the last email
    And user "Alice" has removed the access of guest "guest@example.org" from resource "textfile.txt" of space "Personal"
    When guest "guest@example.org" requests a new guest link <session> using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType"],
        "properties": {
          "errorType": { "const": "tokenNotFound" }
        }
      }
      """
    Examples:
      | session                 |
      | with the guest session  |
      | without a guest session |

  @email @env-config
  Scenario: renewing with the guest session is rate limited
    Given the config "AUTH_GUEST_SESSION_QUOTA_COOLDOWN" has been set to "1h"
    And guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    When guest "guest@example.org" requests a new guest link with the guest session using the Graph API
    Then the HTTP status code should be "429"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType", "retryAfter"],
        "properties": {
          "errorType": { "const": "tooManyRequests" },
          "retryAfter": { "type": "integer", "minimum": 1 }
        }
      }
      """

  @email @env-config
  Scenario: renewing without a session is rate limited
    Given the config "AUTH_GUEST_QUOTA_COOLDOWN" has been set to "1h"
    And guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link without a guest session
    And guest "guest@example.org" has verified the guest link from the last email
    When guest "guest@example.org" requests a new guest link without a guest session using the Graph API
    Then the HTTP status code should be "429"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["errorType", "retryAfter"],
        "properties": {
          "errorType": { "const": "tooManyRequests" },
          "retryAfter": { "type": "integer", "minimum": 1 }
        }
      }
      """
