Feature: create user
  As a admin
  I want to create a user
  So that the user can use the application

  Background:
    Given user "Alice" has been created with default attributes

  @owncloud-issue-3516
  Scenario Outline: admin creates a user
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When the user "Alice" creates a new user with the following attributes using the Graph API:
      | userName       | <user>         |
      | displayName    | <display-name> |
      | email          | <email>        |
      | password       | <password>     |
      | accountEnabled | <enable>       |
    Then the HTTP status code should be "<http-status-code>"
    And user "<user>" <should-or-not> exist
    Examples:
      | user                         | display-name                            | email                   | password                     | http-status-code | enable | should-or-not |
      | SameDisplayName              | Alice Hansen                            | new@example.org         | containsCharacters(*:!;_+-&) | 201              | true   | should        |
      | withoutPassSameEmail         | without pass                            | alice@example.org       |                              | 201              | true   | should        |
      | name                         | pass with space                         | example@example.org     | my pass                      | 201              | true   | should        |
      | user1                        | user names must not start with a number | example@example.org     | my pass                      | 201              | true   | should        |
      | nameWithCharacters(*:!;_+-&) | user                                    | new@example.org         | 123                          | 400              | true   | should not    |
      | name with space              | name with space                         | example@example.org     | 123                          | 400              | true   | should not    |
      | createDisabledUser           | disabled user                           | example@example.org     | 123                          | 201              | false  | should        |
      | nameWithNumbers0123456       | user                                    | name0123456@example.org | 123                          | 201              | true   | should        |
      | name.with.dots               | user                                    | name.w.dots@example.org | 123                          | 201              | true   | should        |
      | 123456789                    | user                                    | 123456789@example.org   | 123                          | 400              | true   | should not    |
      | 0.0                          | user                                    | float@example.org       | 123                          | 400              | true   | should not    |
      | withoutEmail                 | without email                           |                         | 123                          | 201              | true   | should        |
      | Alice                        | same userName                           | new@example.org         | 123                          | 409              | true   | should        |


  Scenario: user cannot be created with empty name
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When the user "Alice" creates a new user with the following attributes using the Graph API:
      | userName       |              |
      | displayName    | emptyName    |
      | email          | @example.org |
      | password       | 123          |
      | accountEnabled | true         |
    Then the HTTP status code should be "400"


  Scenario Outline: user without admin right cannot create a user
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When the user "Alice" creates a new user with the following attributes using the Graph API:
      | userName       | user         |
      | displayName    | user         |
      | email          | @example.org |
      | password       | 123          |
      | accountEnabled | true         |
    Then the HTTP status code should be "403"
    And user "user" should not exist
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: user cannot be created with the name of the disabled user
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the user "Alice" has disabled user "Brian"
    When the user "Alice" creates a new user with the following attributes using the Graph API:
      | userName       | Brian                 |
      | displayName    | This is another Brian |
      | email          | brian@example.com     |
      | password       | 123                   |
      | accountEnabled | true                  |
    Then the HTTP status code should be "409"


  Scenario: user can be created with the name of the deleted user
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the user "Alice" has deleted a user "Brian"
    When the user "Alice" creates a new user with the following attributes using the Graph API:
      | userName       | Brian                 |
      | displayName    | This is another Brian |
      | email          | brian@example.com     |
      | password       | 123                   |
      | accountEnabled | true                  |
    Then the HTTP status code should be "201"
    And user "Brian" should exist

  @env-config
  Scenario Outline: create user with setting OpenCloud no restriction on the user name
    Given the config "GRAPH_USERNAME_MATCH" has been set to "none"
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When the user "Alice" creates a new user with the following attributes using the Graph API:
      | userName       | <user>          |
      | displayName    | test user       |
      | email          | new@example.org |
      | password       | 123             |
      | accountEnabled | true            |
    Then the HTTP status code should be "201"
    And user "<user>" should exist
    Examples:
      | user              | description                                 |
      | 1248Bob           | user names starts with the number           |
      | (*:!;+-&$%)_alice | user names starts with the ASCII characters |

  @env-config
  Scenario: create user with setting OpenCloud not to assign the default user role
    Given the config "GRAPH_ASSIGN_DEFAULT_USER_ROLE" has been set to "false"
    When the user "admin" creates a new user with the following attributes using the Graph API:
      | userName       | sam             |
      | displayName    | test user       |
      | email          | new@example.org |
      | password       | 123             |
      | accountEnabled | true            |
    Then the HTTP status code should be "201"
    And user "sam" should exist
    When the administrator retrieves the assigned role of user "sam" using the Graph API
    Then the HTTP status code should be "200"
    And the Graph API response should have no role

  @env-config
  Scenario: create user with setting OpenCloud assign the default user role
    Given the config "GRAPH_ASSIGN_DEFAULT_USER_ROLE" has been set to "true"
    When the user "admin" creates a new user with the following attributes using the Graph API:
      | userName       | sam             |
      | displayName    | test user       |
      | email          | new@example.org |
      | password       | 123             |
      | accountEnabled | true            |
    Then the HTTP status code should be "201"
    And user "sam" should exist
    And user "sam" should have the role "User" assigned


  Scenario: user is created with the default User role
    When the user "admin" creates a new user with the following attributes using the Graph API:
      | userName       | sam             |
      | displayName    | test user       |
      | email          | new@example.org |
      | password       | 123             |
      | accountEnabled | true            |
    Then the HTTP status code should be "201"
    And user "sam" should exist
    And user "sam" should have the role "User" assigned
