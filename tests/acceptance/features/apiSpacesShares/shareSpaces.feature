Feature: Share spaces
  As the owner of a space
  I want to be able to add members to a space, and to remove access for them
  So that I can manage the access to the space

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "share space" with the default quota using the Graph API
    And using spaces DAV path


  Scenario Outline: space admin can share a space to another user
    When user "Alice" shares a space "share space" with settings:
      | shareWith | Brian        |
      | role      | <space-role> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the user "Brian" should have a space called "share space"
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario: user can see who has been granted access
    When user "Alice" shares a space "share space" with settings:
      | shareWith | Brian  |
      | role      | viewer |
    Then the user "Alice" should have a space called "share space" granted to user "Brian" with role "viewer"


  Scenario: user can see that the group has been granted access
    Given group "sales" has been created
    When user "Alice" shares a space "share space" with settings:
      | shareWith | sales  |
      | shareType | 8      |
      | role      | viewer |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the user "Alice" should have a space called "share space" granted to group "sales" with role "viewer"


  Scenario: user can see a file in a received shared space
    Given user "Alice" has uploaded a file inside space "share space" with content "Test" to "test.txt"
    And user "Alice" has created a folder "Folder Main" in space "share space"
    When user "Alice" shares a space "share space" with settings:
      | shareWith | Brian  |
      | role      | viewer |
    Then for user "Brian" the space "share space" should contain these entries:
      | test.txt    |
      | Folder Main |


  Scenario: user unshares a space
    Given user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" unshares a space "share space" to user "Brian"
    Then the HTTP status code should be "200"
    But the user "Brian" should not have a space called "share space"


  Scenario Outline: owner of a space cannot see the space after removing his access to the space
    Given user "Alice" has sent the following space share invitation:
      | space           | share space |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | Manager     |
    When user "<user>" unshares a space "share space" to user "Alice"
    Then the HTTP status code should be "200"
    But the user "Alice" should not have a space called "share space"
    Examples:
      | user  |
      | Alice |
      | Brian |


  Scenario: user can add another user to the space managers to enable him
    Given user "Alice" has uploaded a file inside space "share space" with content "Test" to "test.txt"
    When user "Alice" shares a space "share space" with settings:
      | shareWith | Brian   |
      | role      | manager |
    Then the user "Brian" should have a space called "share space" granted to "Brian" with role "manager"
    When user "Brian" shares a space "share space" with settings:
      | shareWith | Bob    |
      | role      | viewer |
    Then the user "Bob" should have a space called "share space" granted to "Bob" with role "viewer"
    And for user "Bob" the space "share space" should contain these entries:
      | test.txt |


  Scenario Outline: user cannot share a disabled space to another user
    Given user "Alice" has disabled a space "share space"
    When user "Alice" shares a space "share space" with settings:
      | shareWith | Brian        |
      | role      | <space-role> |
    Then the HTTP status code should be "404"
    And the OCS status code should be "404"
    And the OCS status message should be "Wrong path, file/folder doesn't exist"
    But the user "Brian" should not have a space called "share space"
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario Outline: user with manager role can share a space to another user
    Given user "Alice" has sent the following space share invitation:
      | space           | share space |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | Manager     |
    When user "Brian" shares a space "share space" with settings:
      | shareWith | Bob          |
      | role      | <space-role> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the user "Bob" should have a space called "share space"
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario Outline: user with editor or viewer role cannot share a space to another user
    Given user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Brian" shares a space "share space" with settings:
      | shareWith | Bob                |
      | role      | <share-space-role> |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the OCS status message should be "No share permission"
    And the user "Bob" should not have a space called "share space"
    Examples:
      | space-role   | share-space-role |
      | Space Editor | manager          |
      | Space Editor | editor           |
      | Space Editor | viewer           |
      | Space Viewer | manager          |
      | Space Viewer | editor           |
      | Space Viewer | viewer           |


  Scenario Outline: space manager can change the role of space members
    Given user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" updates the space "share space" with settings:
      | shareWith | Brian              |
      | role      | <share-space-role> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the user "Alice" should have a space called "share space" granted to "Brian" with role "<share-space-role>"
    Examples:
      | space-role   | share-space-role |
      | Space Editor | manager          |
      | Space Editor | viewer           |
      | Space Viewer | manager          |
      | Space Viewer | editor           |
      | Manager      | editor           |
      | Manager      | viewer           |


  Scenario Outline: user without manager role cannot change the role of space members
    Given user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | Bob          |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" updates the space "share space" with settings:
      | shareWith | Bob                |
      | role      | <share-space-role> |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the user "Alice" should have a space called "share space" granted to "Bob" with role "viewer"
    Examples:
      | space-role   | share-space-role |
      | Space Editor | manager          |
      | Space Editor | viewer           |
      | Space Viewer | manager          |
      | Space Viewer | editor           |


  Scenario Outline: user shares a space with a group
    Given group "group2" has been created
    And the administrator has added a user "Brian" to the group "group2" using the Graph API
    And the administrator has added a user "Bob" to the group "group2" using the Graph API
    When user "Alice" shares a space "share space" with settings:
      | shareWith | group2       |
      | shareType | 8            |
      | role      | <space-role> |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space"
    And the user "Bob" should have a space called "share space"
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario Outline: user has no access to the space if access for the group has been removed
    Given group "group2" has been created
    And the administrator has added a user "Brian" to the group "group2" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | group2       |
      | shareType       | group        |
      | permissionsRole | <space-role> |
    When user "Alice" unshares a space "share space" to group "group2"
    Then the HTTP status code should be "200"
    And the user "Brian" should not have a space called "share space"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario: user has no access to the space if he has been removed from the group
    Given group "group2" has been created
    And the administrator has added a user "Brian" to the group "group2" using the Graph API
    And the administrator has added a user "Bob" to the group "group2" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | group2       |
      | shareType       | group        |
      | permissionsRole | Space Editor |
    When the administrator removes the following users from the following groups using the Graph API
      | username | groupname |
      | Brian    | group2    |
    Then the HTTP status code of responses on all endpoints should be "204"
    And the user "Brian" should not have a space called "share space"
    But the user "Bob" should have a space called "share space"


  Scenario: users don't have access to the space if the group has been deleted
    Given group "group2" has been created
    And the administrator has added a user "Brian" to the group "group2" using the Graph API
    And the administrator has added a user "Bob" to the group "group2" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | group2       |
      | shareType       | group        |
      | permissionsRole | Space Editor |
    When the administrator deletes group "group2" using the Graph API
    Then the HTTP status code should be "204"
    And the user "Bob" should not have a space called "share space"
    And the user "Brian" should not have a space called "share space"


  Scenario: user increases permissions for one member of the group or for the entire group
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | sales        |
      | shareType       | group        |
      | permissionsRole | Space Viewer |
    When user "Brian" uploads a file inside space "share space" with content "Test" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    When user "Alice" shares a space "share space" with settings:
      | shareWith | Brian  |
      | role      | editor |
    Then the HTTP status code should be "200"
    When user "Brian" uploads a file inside space "share space" with content "Test" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "201"


  Scenario: user increases permissions for the group, so the user's permissions are increased
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | share space  |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" uploads a file inside space "share space" with content "Test" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    When user "Alice" shares a space "share space" with settings:
      | shareWith | sales  |
      | shareType | 8      |
      | role      | editor |
    Then the HTTP status code should be "200"
    When user "Brian" uploads a file inside space "share space" with content "Test" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "201"


  Scenario Outline: space Admin can share a space to the user with an expiration date
    When user "Alice" shares a space "share space" with settings:
      | shareWith  | Brian                    |
      | role       | <space-role>             |
      | expireDate | 2042-03-25T23:59:59+0100 |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space" granted to user "Brian" with role "<space-role>" and expiration date "2042-03-25"
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario Outline: space Admin can share a space to the group with an expiration date
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    When user "Alice" shares a space "share space" with settings:
      | shareWith  | sales                    |
      | shareType  | 8                        |
      | role       | <space-role>             |
      | expireDate | 2042-03-25T23:59:59+0100 |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space" granted to group "sales" with role "<space-role>" and expiration date "2042-03-25"
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario Outline: update the expiration date of a space in user share
    Given user "Alice" has sent the following space share invitation:
      | space              | share space              |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | <space-role>             |
      | expirationDateTime | 2042-03-25T23:59:59.000Z |
    When user "Alice" updates the space "share space" with settings:
      | shareWith  | Brian                         |
      | expireDate | 2044-01-01T23:59:59.999+01:00 |
      | role       | <share-space-role>            |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space" granted to user "Brian" with role "<share-space-role>" and expiration date "2044-01-01"
    Examples:
      | space-role   | share-space-role |
      | Manager      | manager          |
      | Space Editor | editor           |
      | Space Viewer | viewer           |


  Scenario Outline: update the expiration date of a space in group share
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | share space              |
      | sharee             | sales                    |
      | shareType          | group                    |
      | permissionsRole    | <space-role>             |
      | expirationDateTime | 2042-03-25T23:59:59.000Z |
    When user "Alice" updates the space "share space" with settings:
      | shareWith  | sales                         |
      | shareType  | 8                             |
      | expireDate | 2044-01-01T23:59:59.999+01:00 |
      | role       | <share-space-role>            |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space" granted to group "sales" with role "<share-space-role>" and expiration date "2044-01-01"
    Examples:
      | space-role   | share-space-role |
      | Manager      | manager          |
      | Space Editor | editor           |
      | Space Viewer | viewer           |


  Scenario Outline: delete the expiration date of a space in user share
    Given user "Alice" has sent the following space share invitation:
      | space              | share space              |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | <space-role>             |
      | expirationDateTime | 2042-03-25T23:59:59.000Z |
    When user "Alice" updates the space "share space" with settings:
      | shareWith  | Brian              |
      | expireDate |                    |
      | role       | <share-space-role> |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space" granted to user "Brian" with role "<share-space-role>" and expiration date ""
    Examples:
      | space-role   | share-space-role |
      | Manager      | manager          |
      | Space Editor | editor           |
      | Space Viewer | viewer           |


  Scenario Outline: delete the expiration date of a space in group share
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | share space              |
      | sharee             | sales                    |
      | shareType          | group                    |
      | permissionsRole    | <space-role>             |
      | expirationDateTime | 2042-03-25T23:59:59.000Z |
    When user "Alice" updates the space "share space" with settings:
      | shareWith  | sales              |
      | shareType  | 8                  |
      | expireDate |                    |
      | role       | <share-space-role> |
    Then the HTTP status code should be "200"
    And the user "Brian" should have a space called "share space" granted to group "sales" with role "<share-space-role>" and expiration date ""
    Examples:
      | space-role   | share-space-role |
      | Manager      | manager          |
      | Space Editor | editor           |
      | Space Viewer | viewer           |


  Scenario Outline: check the end of expiration of a space in user share
    Given user "Alice" has sent the following space share invitation:
      | space              | share space              |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | <space-role>             |
      | expirationDateTime | 2042-03-25T23:59:59.000Z |
    When user "Alice" expires the user share of space "share space" for user "Brian"
    Then the HTTP status code should be "200"
    And the user "Brian" should not have a space called "share space"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: check the end of expiration of a space in group share
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space              | share space              |
      | sharee             | sales                    |
      | shareType          | group                    |
      | permissionsRole    | <space-role>             |
      | expirationDateTime | 2042-03-25T23:59:59.000Z |
    When user "Alice" expires the group share of space "share space" for group "sales"
    Then the HTTP status code should be "200"
    And the user "Brian" should not have a space called "share space"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: user cannot share the personal space to an other user
    Given the administrator has assigned the role "<user-role>" to user "Brian" using the Graph API
    When user "Brian" shares a space "Brian Murphy" with settings:
      | shareWith | Bob    |
      | role      | viewer |
    Then the HTTP status code should be "400"
    And the OCS status message should be "can not add members to personal spaces"
    And the user "Bob" should not have a space called "Brian Murphy"
    Examples:
      | user-role   |
      | Space Admin |
      | Admin       |
      | User        |


  Scenario: user cannot share the personal space to a group
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    When user "Alice" shares a space "Alice Hansen" with settings:
      | shareWith | sales   |
      | shareType | 8       |
      | role      | manager |
    Then the HTTP status code should be "400"
    And the OCS status message should be "can not add members to personal spaces"
    And the user "Brian" should not have a space called "Alice Hansen"


  Scenario: last space manager cannot change his role
    Given user "Alice" has sent the following space share invitation:
      | space           | share space |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | Manager     |
    When user "Alice" updates the space "share space" with settings:
      | shareWith | Alice  |
      | role      | editor |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    When user "Brian" updates the space "share space" with settings:
      | shareWith | Brian  |
      | role      | editor |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the user "Alice" should have a space called "share space" granted to "Brian" with role "manager"
    And the user "Alice" should have a space called "share space" granted to "Alice" with role "editor"
