Feature: Remove access to a drive
  https://docs.opencloud.eu/libre-graph-api/#/drives.root/DeletePermissionSpaceRoot

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And using spaces DAV path


  Scenario Outline: user removes user member from project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" removes the access of user "Brian" from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "NewSpace"
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |

  @issue-8768
  Scenario Outline: user removes group from project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | group1             |
      | shareType       | group              |
      | permissionsRole | <permissions-role> |
    When user "Alice" removes the access of group "group1" from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "NewSpace"
    Examples:
      | permissions-role |
      | Space Viewer     |
      | Space Editor     |
      | Manager          |


  Scenario Outline: user of a group removes another user from project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | group1             |
      | shareType       | group              |
      | permissionsRole | <permissions-role> |
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace     |
      | sharee          | Carol        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" removes the access of user "Carol" from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "<status-code>"
    And the user "Carol" <shouldOrNot> have a space called "NewSpace"
    Examples:
      | permissions-role | status-code | shouldOrNot |
      | Space Viewer     | 403         | should      |
      | Space Editor     | 403         | should      |
      | Manager          | 204         | should not  |


  Scenario Outline: user of a group removes own group from project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | group1             |
      | shareType       | group              |
      | permissionsRole | <permissions-role> |
    When user "Brian" removes the access of group "group1" from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "<status-code>"
    And the user "Brian" <shouldOrNot> have a space called "NewSpace"
    Examples:
      | permissions-role | status-code | shouldOrNot |
      | Space Viewer     | 403         | should      |
      | Space Editor     | 403         | should      |
      | Manager          | 204         | should not  |

  @issue-8819
  Scenario Outline: user removes himself from the project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" removes the access of user "Brian" from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "<status-code>"
    And the user "Brian" <shouldOrNot> have a space called "NewSpace"
    Examples:
      | permissions-role | status-code | shouldOrNot |
      | Space Viewer     | 403         | should      |
      | Space Editor     | 403         | should      |
      | Manager          | 204         | should not  |

  @issue-8819
  Scenario Outline: user removes another user from project space using permissions endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace     |
      | sharee          | Carol        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" removes the access of user "Carol" from space "NewSpace" using permissions endpoint of the Graph API
    Then the HTTP status code should be "<status-code>"
    And the user "Carol" <shouldOrNot> have a space called "NewSpace"
    Examples:
      | permissions-role | status-code | shouldOrNot |
      | Space Viewer     | 403         | should      |
      | Space Editor     | 403         | should      |
      | Manager          | 204         | should not  |


  Scenario: user cannot remove himself from the project space if he is the last manager using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    When user "Alice" tries to remove own access from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "403"
    And the user "Alice" should have a space called "NewSpace"


  Scenario: user of a group cannot remove own group from project space if it is the last manager using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | NewSpace |
      | sharee          | group1   |
      | shareType       | group    |
      | permissionsRole | Manager  |
    And user "Alice" has removed own access from space "NewSpace"
    When user "Brian" tries to remove the access of group "group1" from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "403"
    And the user "Brian" should have a space called "NewSpace"

  @issue-7879
  Scenario Outline: user removes link share from project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | NewSpace           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" removes the link from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "204"
    And user "Alice" should not have any "link" permissions on space "NewSpace"
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: user removes internal link share from project space using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | NewSpace |
      | permissionsRole | internal |
    When user "Alice" removes the link from space "NewSpace" using root endpoint of the Graph API
    Then the HTTP status code should be "204"
    And user "Alice" should not have any "link" permissions on space "NewSpace"

  @issue-7879
  Scenario Outline: user tries to remove link share of project space owned by next user using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | NewSpace           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Brian" tries to remove the link from space "NewSpace" owned by "Alice" using root endpoint of the Graph API
    Then the HTTP status code should be "404"
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: user tries to remove internal link share of project space owned by next user using root endpoint
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | NewSpace |
      | permissionsRole | internal |
    When user "Brian" tries to remove the link from space "NewSpace" owned by "Alice" using root endpoint of the Graph API
    Then the HTTP status code should be "404"


  Scenario Outline: user removes link share of a project drive using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | projectSpace       |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" removes the last link share of space "projectSpace" using permissions endpoint of the Graph API
    Then the HTTP status code should be "204"
    And user "Alice" should not have any "link" permissions on space "projectSpace"
    Examples:
      | permissions-role |
      | view             |
      | edit             |
      | upload           |
      | createOnly       |


  Scenario: user removes internal link share of a project drive using permissions endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | projectSpace |
      | permissionsRole | internal     |
    When user "Alice" removes the last link share of space "projectSpace" using permissions endpoint of the Graph API
    Then the HTTP status code should be "204"
    And user "Alice" should not have any "link" permissions on space "projectSpace"

  @env-config
  Scenario: remove space share after the share role Space Editor Without Versions has been disabled
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Space Editor Without Versions"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space                     |
      | sharee          | Brian                         |
      | shareType       | user                          |
      | permissionsRole | Space Editor Without Versions |
    And the administrator has disabled the permissions role "Space Editor Without Versions"
    When user "Alice" removes the access of user "Brian" from space "new-space" using root endpoint of the Graph API
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "NewSpace"

  @env-config
  Scenario: remove space share from group after the share role Space Editor Without Versions has been disabled
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Space Editor Without Versions"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space                     |
      | sharee          | group1                        |
      | shareType       | group                         |
      | permissionsRole | Space Editor Without Versions |
    And the administrator has disabled the permissions role "Space Editor Without Versions"
    When user "Alice" removes the access of group "group1" from space "new-space" using root endpoint of the Graph API
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "new-space"
