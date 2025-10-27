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


  Scenario Outline: create a public link with edit permission without a password when enforce-password is enabled
    Given the following configs have been set:
      | config                                                 | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 3             |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "400"
    And the OCS status message should be "missing required password"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 400              |


  Scenario Outline: create a public link with viewer permission without a password when enforce-password is enabled
    Given the following configs have been set:
      | config                                                 | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 1             |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: updates a public link to edit permission with a password
    Given the following configs have been set:
      | config                                                 | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | 3        |
      | password    | %public% |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the OCS status message should be "OK"
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API without a password
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "wrong pass"
    But the public should be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "%public%"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: create a public link with a password in accordance with the password policy
    Given the following configs have been set:
      | config                                                 | value |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS                    | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS          | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS          | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS                        | 2     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS            | 2     |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 3             |
      | password    | 3s:5WW9uE5h=A |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the OCS status message should be "OK"
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API without a password
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "wrong pass"
    But the public should be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "3s:5WW9uE5h=A"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: try to create a public link with a password that does not comply with the password policy
    Given the following configs have been set:
      | config                                        | value |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS               | 2     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 2     |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 3             |
      | password    | Pas1          |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "400"
    And the OCS status message should be:
      """
      At least 13 characters are required
      at least 3 lowercase letters are required
      at least 2 uppercase letters are required
      at least 2 numbers are required
      at least 2 special characters are required  !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~
      """
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 400              |

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: update a public link with a password in accordance with the password policy
    Given the following configs have been set:
      | config                                                 | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS                    | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS          | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS          | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS                        | 1     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS            | 2     |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | 3             |
      | password    | 6a0Q;A3 +i^m[ |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the OCS status message should be "OK"
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API without a password
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "wrong pass"
    But the public should be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "6a0Q;A3 +i^m["
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: try to update a public link with a password that does not comply with the password policy
    Given the following configs have been set:
      | config                                                 | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD           | false |
      | OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD | true  |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS                    | 13    |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS          | 3     |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS          | 2     |
      | OC_PASSWORD_POLICY_MIN_DIGITS                        | 1     |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS            | 2     |
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | 3    |
      | password    | Pws^ |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "400"
    And the OCS status message should be:
      """
      At least 13 characters are required
      at least 3 lowercase letters are required
      at least 2 uppercase letters are required
      at least 1 numbers are required
      at least 2 special characters are required  !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~
      """
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 400              |

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: create a public link with a password in accordance with the password policy (valid cases)
    Given the config "<config>" has been set to "<config-value>"
    And using OCS API version "2"
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 1             |
      | password    | <password>    |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API without a password
    And the public should not be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "wrong pass"
    But the public should be able to download file "/testfile.txt" from inside the last public link shared folder using the public WebDAV API with password "<password>"
    Examples:
      | config                                        | config-value | password                             |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 4            | Ps-1                                 |
      | OC_PASSWORD_POLICY_MIN_CHARACTERS           | 14           | Ps1:with space                       |
      | OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS | 4            | PS1:test                             |
      | OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS | 3            | PS1:TeƒsT                            |
      | OC_PASSWORD_POLICY_MIN_DIGITS               | 2            | PS1:test2                            |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 2            | PS1:test pass                        |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 33           | pS1! #$%&'()*+,-./:;<=>?@[\]^_`{  }~ |
      | OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS   | 5            | 1sameCharacterShouldWork!!!!!        |


  Scenario Outline: try to create a public link with a password that does not comply with the password policy (invalid cases)
    Given using OCS API version "2"
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 3             |
      | password    | <password>    |
    Then the HTTP status code should be "400"
    And the OCS status code should be "400"
    And the OCS status message should be "<message>"
    Examples:
      | password | message                                   |
      | 1Pw:     | At least 8 characters are required        |
      | 1P:12345 | At least 1 lowercase letters are required |
      | test-123 | At least 1 uppercase letters are required |
      | Test-psw | At least 1 numbers are required           |


  Scenario Outline: update a public link with a password that is listed in the Banned-Password-List
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And using OCS API version "2"
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testfile.txt |
      | space           | Personal     |
      | permissionsRole | internal     |
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | 3          |
      | password    | <password> |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "<ocs-status-code>"
    And the OCS status message should be "<message>"
    Examples:
      | password  | http-status-code | ocs-status-code | message                                                                                               |
      | 123       | 400              | 400             | Unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | password  | 400              | 400             | Unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | OpenCloud | 400              | 400             | Unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |


  Scenario Outline: create  a public link with a password that is listed in the Banned-Password-List
    Given the config "OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" has been set to path "config/woodpecker/banned-password-list.txt"
    And using OCS API version "2"
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "test file" to "/testfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testfile.txt |
      | permissions | 3             |
      | password    | <password>    |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "<ocs-status-code>"
    And the OCS status message should be "<message>"
    Examples:
      | password  | http-status-code | ocs-status-code | message                                                                                               |
      | 123       | 400              | 400             | Unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | password  | 400              | 400             | Unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
      | OpenCloud | 400              | 400             | Unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety |
