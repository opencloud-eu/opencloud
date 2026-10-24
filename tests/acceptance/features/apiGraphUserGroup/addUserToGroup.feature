Feature: add users to group
  As a admin
  I want to be able to add users to a group
  So that I can give a user access to the resources of the group

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: adding a user to a group
    Given these groups have been created:
      | groupname   | comment                               |
      | simplegroup | nothing special here                  |
      | España§àôœ€ | special European and other characters |
      | नेपाली      | Unicode group name                    |
    When the administrator adds the following users to the following groups using the Graph API
      | username | groupname   |
      | Alice    | simplegroup |
      | Alice    | España§àôœ€ |
      | Alice    | नेपाली      |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should be listed in the following groups
      | username | groupname   |
      | Alice    | simplegroup |
      | Alice    | España§àôœ€ |
      | Alice    | नेपाली      |


  Scenario: adding a user to a group with special character in its name
    Given these groups have been created:
      | groupname           | comment            |
      | brand-new-group     | dash               |
      | the.group           | dot                |
      | left,right          | comma              |
      | 0                   | The "false" group  |
      | Finance (NP)        | Space and brackets |
      | Admin&Finance       | Ampersand          |
      | maint+eng           | Plus sign          |
      | $x<=>[y*z^2]!       | Maths symbols      |
      | 😁 😂               | emoji              |
      | admin:Pokhara@Nepal | Colon and @        |
    When the administrator adds the following users to the following groups using the Graph API
      | username | groupname           |
      | Alice    | brand-new-group     |
      | Alice    | the.group           |
      | Alice    | left,right          |
      | Alice    | 0                   |
      | Alice    | Finance (NP)        |
      | Alice    | Admin&Finance       |
      | Alice    | maint+eng           |
      | Alice    | $x<=>[y*z^2]!       |
      | Alice    | 😁 😂               |
      | Alice    | admin:Pokhara@Nepal |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should be listed in the following groups
      | username | groupname           |
      | Alice    | brand-new-group     |
      | Alice    | the.group           |
      | Alice    | left,right          |
      | Alice    | 0                   |
      | Alice    | Finance (NP)        |
      | Alice    | Admin&Finance       |
      | Alice    | maint+eng           |
      | Alice    | $x<=>[y*z^2]!       |
      | Alice    | 😁 😂               |
      | Alice    | admin:Pokhara@Nepal |


  Scenario: adding a user to a group with % and # in its name
    Given these groups have been created:
      | groupname       | comment                                 |
      | maintenance#123 | Hash sign                               |
      | 50%pass         | Percent sign (special escaping happens) |
      | 50%25=0         | %25 literal looks like an escaped "%"   |
      | 50%2Eagle       | %2E literal looks like an escaped "."   |
      | 50%2Fix         | %2F literal looks like an escaped slash |
      | Mgmt\Middle     | Backslash                               |
      | staff?group     | Question mark                           |
    When the administrator adds the following users to the following groups using the Graph API
      | username | groupname       |
      | Alice    | maintenance#123 |
      | Alice    | 50%pass         |
      | Alice    | 50%25=0         |
      | Alice    | 50%2Eagle       |
      | Alice    | 50%2Fix         |
      | Alice    | Mgmt\Middle     |
      | Alice    | staff?group     |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should be listed in the following groups
      | username | groupname       |
      | Alice    | maintenance#123 |
      | Alice    | 50%pass         |
      | Alice    | 50%25=0         |
      | Alice    | 50%2Eagle       |
      | Alice    | 50%2Fix         |
      | Alice    | Mgmt\Middle     |
      | Alice    | staff?group     |


  Scenario: adding a user to a group that has a forward-slash in the group name
    Given these groups have been created:
      | groupname        | comment                            |
      | Mgmt/Sydney      | Slash (special escaping happens)   |
      | Mgmt//NSW/Sydney | Multiple slash                     |
      | priv/subadmins/1 | Subadmins mentioned not at the end |
      | var/../etc       | using slash-dot-dot                |
    When the administrator adds the following users to the following groups using the Graph API
      | username | groupname        |
      | Alice    | Mgmt/Sydney      |
      | Alice    | Mgmt//NSW/Sydney |
      | Alice    | priv/subadmins/1 |
      | Alice    | var/../etc       |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should be listed in the following groups
      | username | groupname        |
      | Alice    | Mgmt/Sydney      |
      | Alice    | Mgmt//NSW/Sydney |
      | Alice    | priv/subadmins/1 |
      | Alice    | var/../etc       |

  @issue-5938
  Scenario Outline: user other than the admin tries to add herself to a group
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    And group "groupA" has been created
    When user "Alice" tries to add herself to group "groupA" using the Graph API
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
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |

  @issue-5938
  Scenario Outline: user other than the admin tries to add other user to a group
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    And group "groupA" has been created
    When user "Alice" tries to add user "Brian" to group "groupA" using the Graph API
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
              "message" : {
                "type": "string",
                "enum": ["Forbidden"]
              }
            }
          }
        }
      }
      """
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: admin tries to add user to a nonexistent group
    When the administrator tries to add user "Alice" to a nonexistent group using the Graph API
    Then the HTTP status code should be "404"

  @issue-5939
  Scenario Outline: user other than the admin tries to add user to a nonexistent group
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to add user "Brian" to a nonexistent group using the Graph API
    Then the HTTP status code should be "403"
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: admin tries to add a nonexistent user to a group
    Given group "groupA" has been created
    When the administrator tries to add nonexistent user to group "groupA" using the Graph API
    Then the HTTP status code should be "404"


  Scenario: non-admin user tries to add a nonexistent user to a group
    Given group "groupA" has been created
    When user "Alice" tries to add nonexistent user to group "groupA" using the Graph API
    Then the HTTP status code should be "403"


  Scenario: admin tries to add user to a group without sending the group
    When the administrator tries to add user "Alice" to a nonexistent group using the Graph API
    Then the HTTP status code should be "404"


  Scenario: add multiple users to a group at once
    Given these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    And the administrator has created a group "grp1" using the Graph API
    When the administrator adds the following users to the following groups using the Graph API
      | username | groupname |
      | Brian    | grp1      |
      | Carol    | grp1      |
    Then the HTTP status code should be "204"
    And the following users should be listed in the following groups
      | username | groupname |
      | Brian    | grp1      |
      | Carol    | grp1      |


  Scenario: admin tries to add users to a nonexistent group at once
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    When the administrator "Alice" tries to add the following users to a nonexistent group at once using the Graph API
      | username |
      | Brian    |
      | Carol    |
    Then the HTTP status code should be "404"


  Scenario: admin tries to add multiple nonexistent users to a group at once
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a group "grp1" using the Graph API
    When the administrator "Alice" tries to add the following nonexistent users to a group "grp1" at once using the Graph API
      | username |
      | Brian    |
      | Carol    |
    Then the HTTP status code should be "404"


  Scenario: admin tries to add nonexistent and existing users to a group at once
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
    And user "Alice" has created a group "grp1" using the Graph API
    When the administrator "Alice" tries to add the following existent and nonexistent users to a group "grp1" at once using the Graph API
      | username |
      | Brian    |
      | Carol    |
    Then the HTTP status code should be "404"


  Scenario: adding a disabled user to a group
    Given these groups have been created:
      | groupname | comment      |
      | sales     | normal group |
    And the user "Admin" has disabled user "Alice"
    When the administrator adds the following users to the following groups using the Graph API
      | username | groupname |
      | Alice    | sales     |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the following users should be listed in the following groups
      | username | groupname |
      | Alice    | sales     |

  @issue-5702
  Scenario: try to add users to a group twice
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    And user "Alice" has created a group "grp1" using the Graph API
    And the administrator "Alice" has added the following users to a group "grp1" at once using the Graph API
      | username |
      | Brian    |
      | Carol    |
    When the administrator "Alice" adds the following users to a group "grp1" at once using the Graph API
      | username |
      | Brian    |
      | Carol    |
    Then the HTTP status code should be "400"
    And the following users should be listed in the following groups
      | username | groupname |
      | Brian    | grp1      |
      | Carol    | grp1      |

  @issue-5793
  Scenario: try to add a group to another group with PATCH request
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
    And these groups have been created:
      | groupname |
      | student   |
      | music     |
    And user "Brian" has been added to group "music"
    When the administrator "Alice" tries to add a group "music" to another group "student" with PATCH request using the Graph API
    Then the HTTP status code should be "400"

  @issue-5793
  Scenario: try to add a group to another group with POST request
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
    And these groups have been created:
      | groupname |
      | student   |
      | music     |
    And user "Brian" has been added to group "music"
    When the administrator "Alice" tries to add a group "music" to another group "student" with POST request using the Graph API
    Then the HTTP status code should be "400"


  Scenario Outline: admin tries to add a user to a group with invalid JSON
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
    And user "Alice" has created a group "grp1" using the Graph API
    When user "Alice" tries to add user "Brian" to group "grp1" with invalid JSON "<invalid-json>" using the Graph API
    Then the HTTP status code should be "400"
    Examples:
      | invalid-json                                                        |
      | {'@odata.id': 'https://localhost:9200/graph/v1.0/users/%user_id%',} |
      | {'@odata.id'- 'https://localhost:9200/graph/v1.0/users/%user_id%'}  |
      | {@odata.id: https://localhost:9200/graph/v1.0/users/%user_id%}      |


  Scenario Outline: admin tries to add multiple users to a group at once with invalid JSON
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    And user "Alice" has created a group "grp1" using the Graph API
    When user "Alice" tries to add the following users to a group "grp1" at once with invalid JSON "<invalid-json>" using the Graph API
      | username |
      | Brian    |
      | Carol    |
    Then the HTTP status code should be "400"
    Examples:
      | invalid-json                                                                                                                       |
      | {'members@odata.bind': ['https://localhost:9200/graph/v1.0/users/%user_id%',,'https://localhost:9200/graph/v1.0/users/%user_id%']} |
      | {'members@odata.bind'- ['https://localhost:9200/graph/v1.0/users/%user_id%','https://localhost:9200/graph/v1.0/users/%user_id%']}  |
      | {'members@odata.bind': ['https://localhost:9200/graph/v1.0/users/%user_id%'.'https://localhost:9200/graph/v1.0/users/%user_id%']}  |

  @issue-5871
  Scenario: admin tries to add multiple users with wrong host
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
      | Carol    |
    And user "Alice" has created a group "grp1" using the Graph API
    When user "Alice" tries to add the following users to a group "grp1" at once with an invalid host using the Graph API
      | username |
      | Brian    |
      | Carol    |
    Then the HTTP status code should be "400"

  @issue-5871
  Scenario: admin tries to add single user with wrong host
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
    And user "Alice" has created a group "grp1" using the Graph API
    When user "Alice" tries to add user "Brian" to group "grp1" with an invalid host using the Graph API
    Then the HTTP status code should be "400"


  Scenario Outline: try to add invalid user id to a group
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a group "grp1" using the Graph API
    When the administrator "Alice" tries to add an invalid user id "<invalid-uuidv4>" to a group "grp1" using the Graph API
    Then the HTTP status code should be "404"
    Examples:
      | invalid-uuidv4                        | comment                                                |
      | �ϰ�Ϧ-@$@^-¶Ëøœ-ɧɸɱʨΌϖЁϿ               | UTF characters                                         |
      | b1f74ec4-dd7e-11ef-a543-03775734d0f71 | adding an extra character at end of valid UUID pattern |
      | 4c510adac8-6b-4815-882042cdf-82c3d51  | invalid UUID pattern                                   |


  Scenario Outline: try to add invalid user ids to a group at once
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a group "grp1" using the Graph API
    When the administrator "Alice" tries to add the following invalid user ids to a group "grp1" at once using the Graph API
      | user-id          |
      | <invalid-uuidv4> |
      | <invalid-uuidv4> |
    Then the HTTP status code should be "404"
    Examples:
      | invalid-uuidv4                        | comment                                                |
      | �ϰ�Ϧ-@$@^-¶Ëøœ-ɧɸɱʨΌϖЁϿ               | UTF characters                                         |
      | b1f74ec4-dd7e-11ef-a543-03775734d0f71 | adding an extra character at end of valid UUID pattern |
      | 4c510adac8-6b-4815-882042cdf-82c3d51  | invalid UUID pattern                                   |

  @issue-5855
  Scenario: add same user twice to a group at once
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And these users have been created with default attributes:
      | username |
      | Brian    |
    And user "Alice" has created a group "grp1" using the Graph API
    When the administrator "Alice" adds the following users to a group "grp1" at once using the Graph API
      | username |
      | Brian    |
      | Brian    |
    Then the HTTP status code should be "204"
    And the user "Brian" should be listed once in the group "grp1"
