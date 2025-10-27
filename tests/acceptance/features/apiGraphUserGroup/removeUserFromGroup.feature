Feature: remove a user from a group
  As an admin
  I want to be able to remove a user from a group
  So that I can manage user access to group resources

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: admin removes a user from a group
    Given these groups have been created:
      | groupname       | comment                               |
      | brand-new-group | nothing special here                  |
      | España§àôœ€     | special European and other characters |
      | नेपाली          | Unicode group name                    |
    And the following users have been added to the following groups
      | username | groupname       |
      | Alice    | brand-new-group |
      | Alice    | España§àôœ€     |
      | Alice    | नेपाली          |
    When the administrator removes the following users from the following groups using the Graph API
      | username | groupname       |
      | Alice    | brand-new-group |
      | Alice    | España§àôœ€     |
      | Alice    | नेपाली          |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should not belong to the following groups
      | username | groupname       |
      | Alice    | brand-new-group |
      | Alice    | España§àôœ€     |
      | Alice    | नेपाली          |


  Scenario: admin removes a user from a group with special characters
    Given these groups have been created:
      | groupname           | comment            |
      | brand-new-group     | dash               |
      | the.group           | dot                |
      | left,right          | comma              |
      | 0                   | The "false" group  |
      | Finance (NP)        | Space and brackets |
      | Admin&Finance       | Ampersand          |
      | admin:Pokhara@Nepal | Colon and @        |
      | maint+eng           | Plus sign          |
      | $x<=>[y*z^2]!       | Maths symbols      |
      | Mgmt\Middle         | Backslash          |
      | 😁 😂               | emoji              |
    And the following users have been added to the following groups
      | username | groupname           |
      | Alice    | brand-new-group     |
      | Alice    | the.group           |
      | Alice    | left,right          |
      | Alice    | 0                   |
      | Alice    | Finance (NP)        |
      | Alice    | Admin&Finance       |
      | Alice    | admin:Pokhara@Nepal |
      | Alice    | maint+eng           |
      | Alice    | $x<=>[y*z^2]!       |
      | Alice    | Mgmt\Middle         |
      | Alice    | 😁 😂               |
    When the administrator removes the following users from the following groups using the Graph API
      | username | groupname           |
      | Alice    | brand-new-group     |
      | Alice    | the.group           |
      | Alice    | left,right          |
      | Alice    | 0                   |
      | Alice    | Finance (NP)        |
      | Alice    | Admin&Finance       |
      | Alice    | admin:Pokhara@Nepal |
      | Alice    | maint+eng           |
      | Alice    | $x<=>[y*z^2]!       |
      | Alice    | Mgmt\Middle         |
      | Alice    | 😁 😂               |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should not belong to the following groups
      | username | groupname           |
      | Alice    | brand-new-group     |
      | Alice    | the.group           |
      | Alice    | left,right          |
      | Alice    | 0                   |
      | Alice    | Finance (NP)        |
      | Alice    | Admin&Finance       |
      | Alice    | admin:Pokhara@Nepal |
      | Alice    | maint+eng           |
      | Alice    | $x<=>[y*z^2]!       |
      | Alice    | Mgmt\Middle         |
      | Alice    | 😁 😂               |


  Scenario: admin removes a user from a group having % and # in their names
    Given these groups have been created:
      | groupname       | comment                                 |
      | maintenance#123 | Hash sign                               |
      | 50%25=0         | %25 literal looks like an escaped "%"   |
      | staff?group     | Question mark                           |
      | 50%pass         | Percent sign (special escaping happens) |
      | 50%2Eagle       | %2E literal looks like an escaped "."   |
      | 50%2Fix         | %2F literal looks like an escaped slash |
    And the following users have been added to the following groups
      | username | groupname       |
      | Alice    | maintenance#123 |
      | Alice    | 50%25=0         |
      | Alice    | staff?group     |
      | Alice    | 50%pass         |
      | Alice    | 50%2Eagle       |
      | Alice    | 50%2Fix         |
    When the administrator removes the following users from the following groups using the Graph API
      | username | groupname       |
      | Alice    | maintenance#123 |
      | Alice    | 50%25=0         |
      | Alice    | staff?group     |
      | Alice    | 50%pass         |
      | Alice    | 50%2Eagle       |
      | Alice    | 50%2Fix         |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should not belong to the following groups
      | username | groupname       |
      | Alice    | maintenance#123 |
      | Alice    | 50%25=0         |
      | Alice    | staff?group     |
      | Alice    | 50%pass         |
      | Alice    | 50%2Eagle       |
      | Alice    | 50%2Fix         |


  Scenario: admin removes a user from a group that has forward-slash(s) in the group name
    Given these groups have been created:
      | groupname        | comment                            |
      | Mgmt/Sydney      | Slash (special escaping happens)   |
      | Mgmt//NSW/Sydney | Multiple slash                     |
      | priv/subadmins/1 | Subadmins mentioned not at the end |
      | var/../etc       | using slash-dot-dot                |
    And the following users have been added to the following groups
      | username | groupname        |
      | Alice    | Mgmt/Sydney      |
      | Alice    | Mgmt//NSW/Sydney |
      | Alice    | priv/subadmins/1 |
      | Alice    | var/../etc       |
    When the administrator removes the following users from the following groups using the Graph API
      | username | groupname        |
      | Alice    | Mgmt/Sydney      |
      | Alice    | Mgmt//NSW/Sydney |
      | Alice    | priv/subadmins/1 |
      | Alice    | var/../etc       |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should not belong to the following groups
      | username | groupname        |
      | Alice    | Mgmt/Sydney      |
      | Alice    | Mgmt//NSW/Sydney |
      | Alice    | priv/subadmins/1 |
      | Alice    | var/../etc       |


  Scenario: admin tries to remove a user from a nonexistent group
    When the administrator tries to remove user "Alice" from a nonexistent group using the Graph API
    Then the HTTP status code should be "404"


  Scenario Outline: non-admin user tries to remove a user from a nonexistent group
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to remove user "Brian" from a nonexistent group using the Graph API
    Then the HTTP status code should be "403"
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |

  @owncloud-issue-5938
  Scenario Outline: user other than the admin can't remove a user from their group
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    When user "Alice" tries to remove user "Brian" from group "grp1" using the Graph API
    Then the HTTP status code should be "403"
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
              "message"
            ],
            "properties": {
              "message": {
                "type": "string",
                "enum": ["Forbidden"]
              }
            }
          }
        }
      }
      """
    And user "Brian" should belong to group "grp1"
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: admin removes a disabled user from a group
    Given these groups have been created:
      | groupname       | comment              |
      | brand-new-group | nothing special here |
    And the following users have been added to the following groups
      | username | groupname       |
      | Alice    | brand-new-group |
    And the user "Admin" has disabled user "Alice"
    When the administrator removes the following users from the following groups using the Graph API
      | username | groupname       |
      | Alice    | brand-new-group |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should not belong to the following groups
      | username | groupname       |
      | Alice    | brand-new-group |
