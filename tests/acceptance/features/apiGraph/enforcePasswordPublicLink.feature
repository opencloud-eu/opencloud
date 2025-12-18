@env-config
Feature: enforce password on public link
  As a user
  I want to enforce passwords on public links shared with upload, edit, or contribute permission
  So that the password is required to access the contents of the link

  Password requirements. set by default:
  | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD  | true |
  | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 8    |
  | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS | 1    |
  | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS | 1    |
  | OC_PASSWORD_POLICY_MIN_DIGITS               | 1    |
  | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 1    |

  Background:
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"


  Scenario Outline: create a public link without a password when enforce-password for writable share is enabled
    Given the following configs have been set:
      | config                                               | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | testfile.txt       |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
    Then the HTTP status code should be "<status-code>"
    Examples:
      | permissions-role | status-code |
      | view             | 200         |
      | edit             | 400         |


  Scenario: try to update a public link to edit permission without a password
    Given the following configs have been set:
      | config                                               | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" tries to update the last public link share using the permissions endpoint of the Graph API:
      | resource           | testfile.txt |
      | space              | Personal     |
      | permissionsRole    | edit         |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "type": "string",
                "pattern": "invalidRequest"
              },
              "message": {
                "const": "password protection is enforced"
              }
            }
          }
        }
      }
      """

  @issue-2048
  Scenario: update a public link to edit permission. Need set pasword first
    Given the following configs have been set:
      | config                                               | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" sets the following password for the last link share using the Graph API:
      | resource | testfile.txt |
      | space    | Personal     |
      | password | %public%     |
    Then the HTTP status code should be "200"
    And user "Alice" updates the last public link share using the permissions endpoint of the Graph API:
      | resource           | testfile.txt |
      | space              | Personal     |
      | permissionsRole    | edit         |
    And the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "link"
        ],
        "properties": {
          "hasPassword": { "const": true },
          "link": {
            "type": "object",
            "required": [
              "type"
            ],
            "properties": {
              "type": { "const": "edit" }
            }
          }
        }
      }
      """

  @issue-9724 @issue-10331
  Scenario: create a public link with a password in accordance with the password policy
    Given the following configs have been set:
      | config                                               | value |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS                    | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS          | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS          | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS                        | 2     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS            | 2     |
    When user "Alice" creates the following resource link share using the Graph API:
      | resource        | testfile.txt  |
      | space           | Personal      |
      | permissionsRole | edit          |
      | password        | 3s:5WW9uE5h=A |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword",
          "id",
          "link"
        ],
        "properties": {
          "hasPassword": { "const": true },
          "id": {
            "type": "string",
            "pattern": "^[a-zA-Z]{15}$"
          },
          "link": {
            "type": "object",
            "required": [
              "type"
            ],
            "properties": {
              "type": { "const": "edit" }
            }
          }
        }
      }
      """


  Scenario: try to create a public link with a password that does not comply with the password policy
    Given the following configs have been set:
      | config                                      | value |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS               | 2     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 2     |
    When user "Alice" tries to create the following resource link share using the Graph API:
      | space           | Personal     |
      | resource        | testfile.txt |
      | permissionsRole | edit         |
      | password        | Pas1         |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code": { "const": "invalidRequest" },
              "innererror": {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message": {
                "type": "string",
                "pattern": "at least 13 characters are required\\s+at least 3 lowercase letters are required\\s+at least 2 uppercase letters are required\\s+at least 2 numbers are required\\s+at least 2 special characters are required\\s+[!\"#$%&'()*+,\\-./:;<=>?@\\[\\\\\\]^_`{|}~]+"
              }
            }
          }
        }
      }
      """

  @issue-9724 @issue-10331
  Scenario: update a public link with a password in accordance with the password policy
    Given the following configs have been set:
      | config                                               | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS                    | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS          | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS          | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS                        | 1     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS            | 2     |
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" sets the following password for the last link share using the Graph API:
      | resource | testfile.txt  |
      | space    | Personal      |
      | password | 6a0Q;A3 +i^m[ |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [ "hasPassword" ],
        "properties": {
          "hasPassword": { "const": true }
        }
      }
      """


  Scenario Outline: try to update a public link with a password that does not comply with the password policy
    Given the following configs have been set:
      | config                                               | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS                    | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS          | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS          | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS                        | 1     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS            | 2     |
    And using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | 3    |
      | password    | Pws^ |
    And user "Alice" tries to set the following password for the last link share using the Graph API:
      | resource | testfile.txt |
      | space    | Personal     |
      | password | Pws^         |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [ "error" ],
        "properties": {
          "error": {
            "type": "object",
            "required": [  "message"  ],
            "properties": {
              "message": {
                "type": "string",
                "pattern": "at least 13 characters are required\\s+at least 3 lowercase letters are required\\s+at least 2 uppercase letters are required\\s+at least 1 numbers are required\\s+at least 2 special characters are required\\s+[!\"#$%&'()*+,\\-./:;<=>?@\\[\\\\\\]^_`{|}~]+"
              }
            }
          }
        }
      }
      """

  @issue-9724 @issue-10331
  Scenario Outline: create a public link with a password in accordance with the password policy (valid cases)
    Given the config "<config>" has been set to "<config-value>"
    When user "Alice" creates the following resource link share using the Graph API:
      | space           | Personal     |
      | resource        | testfile.txt |
      | permissionsRole | view         |
      | password        | <password>   |
    Then the HTTP status code should be "200"
    Examples:
      | config                                      | config-value | password                             |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 4            | Ps-1                                 |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 14           | Ps1:with space                       |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS | 4            | PS1:test                             |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS | 3            | PS1:TeƒsT                            |
      | OC_PASSWORD_POLICY_MIN_DIGITS               | 2            | PS1:test2                            |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 2            | PS1:test pass                        |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 33           | pS1! #$%&'()*+,-./:;<=>?@[\]^_`{  }~ |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 5            | 1sameCharacterShouldWork!!!!!        |


  Scenario Outline: try to create a public link with a password that does not comply with the password policy (invalid cases)
    When user "Alice" tries to create the following resource link share using the Graph API:
      | space           | Personal     |
      | resource        | testfile.txt |
      | permissionsRole | view         |
      | password        | <password>   |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [ "error" ],
        "properties": {
          "error": {
            "type": "object",
            "required": [  "message"  ],
            "properties": {
               "message": {
                "const": "<message>"
              }
            }
          }
        }
      }
      """
    Examples:
      | password | message                                   |
      | 1Pw:     | at least 8 characters are required        |
      | 1P:12345 | at least 1 lowercase letters are required |
      | test-123 | at least 1 uppercase letters are required |
      | Test-psw | at least 1 numbers are required           |


  Scenario Outline: update a public link with a password that is listed in the Banned-Password-List
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" tries to set the following password for the last link share using the Graph API:
      | resource | testfile.txt |
      | space    | Personal     |
      | password | <password>   |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [ "error" ],
        "properties": {
          "error": {
            "type": "object",
            "required": [ "message" ],
            "properties": {
               "message": { "const": "<message>" }
            }
          }
        }
      }
      """
    Examples:
      | password  | message                                                                                               |
      | 123       | unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | password  | unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | OpenCloud | unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |


  Scenario Outline: create  a public link with a password that is listed in the Banned-Password-List
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    When user "Alice" tries to create the following resource link share using the Graph API:
      | space           | Personal     |
      | resource        | testfile.txt |
      | permissionsRole | view         |
      | password        | <password>   |
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [ "error" ],
        "properties": {
          "error": {
            "type": "object",
            "required": [ "message" ],
            "properties": {
               "message": { "const": "<message>" }
            }
          }
        }
      }
      """
    Examples:
      | password  | message                                                                                               |
      | 123       | unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | password  | unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | OpenCloud | unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
