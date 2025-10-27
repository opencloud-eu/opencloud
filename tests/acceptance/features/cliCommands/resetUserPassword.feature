@env-config
Feature: reset user password via CLI command


  Scenario: reset user password
    Given the user "Admin" has created a new user with the following attributes:
      | userName    | Alice        |
      | displayName | Alice Hansen |
      | password    | %alt1%       |
    And the administrator has stopped the server
    When the administrator resets the password of existing user "Alice" to "newpass" using the CLI
    Then the command should be successful
    And the command output should contain "Password for user 'uid=Alice,ou=users,o=libregraph-idm' updated."
    But the command output should not contain "Failed to update user password: entry does not exist"
    And the administrator has started the server
    And user "Alice" should be able to create folder "newFolder" using password "newpass"
    But user "Alice" should not be able to log in with wrong password "%alt1%"


  Scenario: try to reset password of non-existing user
    Given the administrator has stopped the server
    When the administrator resets the password of non-existing user "Alice" to "newpass" using the CLI
    Then the command should be successful
    But the command output should contain "Failed to update user password: entry does not exist"


  Scenario: reset password of admin user
    Given the user "Admin" has created a new user with the following attributes:
      | userName    | Alice        |
      | displayName | Alice Hansen |
      | password    | %alt1%       |
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the administrator has stopped the server
    When the administrator resets the password of existing user "Alice" to "newpass" using the CLI
    Then the command should be successful
    And the command output should contain "Password for user 'uid=Alice,ou=users,o=libregraph-idm' updated."
    But the command output should not contain "Failed to update user password: entry does not exist"
    And the administrator starts the server
    And user "Alice" using password "newpass" should be able to create a new user "Brian" with default attributes


  Scenario: reset password after renaming the admin user
    Given the user "Admin" has created a new user with the following attributes:
      | userName    | Alice        |
      | displayName | Alice Hansen |
      | password    | %alt1%       |
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has changed the username to "superUser"
    And the administrator has stopped the server
    When the administrator resets the password of existing user "superUser" to "newpass" using the CLI
    Then the command should be successful
    And the command output should contain "Password for user 'uid=superUser,ou=users,o=libregraph-idm' updated."
    But the command output should not contain "Failed to update user password: entry does not exist"
    And the administrator starts the server
    And user "superUser" using password "newpass" should be able to create a new user "Brian" with default attributes
