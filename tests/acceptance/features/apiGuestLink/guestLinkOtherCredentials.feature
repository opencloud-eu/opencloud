@email
Feature: guest session does not override other credentials
  As a user who has opened a guest link in the same browser
  I want requests with other credentials to never be authenticated as the guest
  So that I get my own data, or a 401 to renew invalid credentials

  Background:
    Given user "Alice" has been created with default attributes
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "for guest" to "guestFile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | guestFile.txt     |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    And guest "guest@example.org" has verified the guest link from the last email


  Scenario: request with an invalid bearer token and a guest session is rejected
    When guest "guest@example.org" lists the shares shared with them with authorization header "Bearer invalid-token" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: request with invalid basic auth credentials and a guest session is rejected
    When user "Brian" lists the shares shared with them with password "invalid-password" and the guest session of guest "guest@example.org" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: request with valid user credentials and a guest session is authenticated as the user
    When user "Brian" lists the shares shared with them with the guest session of guest "guest@example.org" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 0
          }
        }
      }
      """


  Scenario: public link request with a guest session is authenticated as the public link
    Given user "Alice" has created folder "publicFolder"
    And user "Alice" has uploaded file with content "public content" to "publicFolder/public.txt"
    And user "Alice" has created the following resource link share:
      | resource        | publicFolder |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    When guest "guest@example.org" downloads file "public.txt" from the last public link share with password "%public%" and the guest session using the public WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "public content"


  Scenario: public link request with an invalid guest session is authenticated as the public link
    Given user "Alice" has created folder "publicFolder"
    And user "Alice" has uploaded file with content "public content" to "publicFolder/public.txt"
    And user "Alice" has created the following resource link share:
      | resource        | publicFolder |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    And the signature of the guest session of guest "guest@example.org" has been changed
    When guest "guest@example.org" downloads file "public.txt" from the last public link share with password "%public%" and the guest session using the public WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "public content"
