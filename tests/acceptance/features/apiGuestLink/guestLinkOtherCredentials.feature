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


  Scenario Outline: request with invalid credentials and a guest session is rejected
    When guest "guest@example.org" lists the shares shared with them with authorization header "<authorization>" using the Graph API
    Then the HTTP status code should be "401"
    Examples:
      | authorization              |
      | Bearer invalid-token       |
      | Basic aW52YWxpZDppbnZhbGlk |


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
            "minItems": 0,
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


  Scenario: guest downloads a shared file via its signed download URL
    When guest "guest@example.org" downloads file "guestFile.txt" shared from space "Personal" of user "Alice" via its signed download URL
    Then the HTTP status code should be "200"
    And the downloaded content should be "for guest"


  Scenario: signed download URL of a user with a guest session is authenticated as the user
    Given user "Brian" has uploaded file with content "from Brian" to "brianFile.txt"
    When user "Brian" downloads file "brianFile.txt" via its signed download URL with the guest session of guest "guest@example.org"
    Then the HTTP status code should be "200"
    And the downloaded content should be "from Brian"


  Scenario: guest uploads a file using TUS with an unusable authorization header
    Given user "Alice" has created folder "uploadFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder      |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Editor            |
    When guest "guest@example.org" uploads content "uploaded by guest" to "tus.txt" inside folder "uploadFolder" of space "Personal" of user "Alice" using the TUS protocol with authorization header "undefined"
    Then the HTTP status code should be "204"
    And the content of file "uploadFolder/tus.txt" for user "Alice" should be "uploaded by guest"
