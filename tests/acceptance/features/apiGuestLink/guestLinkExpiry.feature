@email @env-config
Feature: guest session and PIN expire
  As a user sharing a resource with a guest
  I want the guest session and the PIN to expire
  So that a leaked session or PIN cannot be used forever

  Background:
    Given using SharingNG
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"


  Scenario: guest loses access when the guest session expires
    Given the config "AUTH_GUEST_JWT_TTL" has been set to "5s"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    And the guest session of guest "guest@example.org" has expired
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "401"
    When guest "guest@example.org" lists the shares shared with them using the Graph API
    Then the HTTP status code should be "401"


  Scenario: guest with an expired session gets a new link without a PIN
    Given the config "AUTH_GUEST_JWT_TTL" has been set to "5s"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    And the guest session of guest "guest@example.org" has expired
    When guest "guest@example.org" requests a new guest link with the guest session using the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain a nonce cookie
    And guest "guest@example.org" should have received the following email from user "Alice"
      """
      The link below is personal and can only be used once. Please do not forward it.
      """
    When guest "guest@example.org" verifies the guest link from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie
    When guest "guest@example.org" downloads file "textfile.txt" shared from space "Personal" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "200"


  Scenario: expired PIN cannot be used
    Given the following configs have been set:
      | config                            | value |
      | AUTH_GUEST_PIN_AND_NONCE_TTL      | 5s    |
      | AUTH_GUEST_SESSION_QUOTA_COOLDOWN | 1ms   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email
    And guest "guest@example.org" has requested a new guest link with the guest session
    And the PIN of guest "guest@example.org" has expired
    When guest "guest@example.org" verifies the PIN from the last email using the Graph API
    Then the HTTP status code should be "401"
    And the response should not contain a guest session cookie
    When guest "guest@example.org" requests a new guest link with the guest session using the Graph API
    Then the HTTP status code should be "200"
    And guest "guest@example.org" has received a new guest link
    When guest "guest@example.org" verifies the PIN from the last email using the Graph API
    Then the HTTP status code should be "200"
    And the response should contain a guest session cookie
