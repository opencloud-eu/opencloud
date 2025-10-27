Feature: re-share resources
  As a user
  I cannot to re-share resources
  This feature has been removed from OpenCloud

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And user "Alice" has created folder "test"


  Scenario Outline: share folder with different roles
    Given using <dav-path-version> DAV path
    When user "Alice" creates a share inside of space "Personal" with settings:
      | path      | test   |
      | shareWith | Brian  |
      | role      | <role> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | permissions | <expected-permissions> |
    Examples:
      | dav-path-version | role   | expected-permissions |
      | old              | editor | 15                   |
      | old              | viewer | 1                    |
      | new              | editor | 15                   |
      | new              | viewer | 1                    |
      | spaces           | editor | 15                   |
      | spaces           | viewer | 1                    |


  Scenario Outline: try to re-share folder
    Given using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | test               |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "test" synced
    When user "Brian" creates a share inside of space "Shares" with settings:
      | path      | test   |
      | shareWith | Carol  |
      | role      | <role> |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the OCS status message should be "No share permission"
    Examples:
      | dav-path-version | role   | permissions-role |
      | old              | editor | Editor           |
      | old              | viewer | Viewer           |
      | new              | editor | Editor           |
      | new              | viewer | Viewer           |
      | spaces           | editor | Editor           |
      | spaces           | viewer | Viewer           |


  Scenario Outline: try to re-share file
    Given user "Alice" has uploaded file with content "other data" to "/textfile1.txt"
    And using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "textfile1.txt" synced
    When user "Brian" creates a share inside of space "Shares" with settings:
      | path      | textfile1.txt |
      | shareWith | Carol         |
      | role      | <role>        |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the OCS status message should be "No share permission"
    Examples:
      | dav-path-version | role   | permissions-role |
      | old              | editor | File Editor      |
      | old              | viewer | Viewer           |
      | new              | editor | File Editor      |
      | new              | viewer | Viewer           |
      | spaces           | editor | File Editor      |
      | spaces           | viewer | Viewer           |


  Scenario Outline: try to create a link to the shared folder
    Given using OCS API version "<ocs_api_version>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | test     |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "test" synced
    When user "Brian" creates a public link share using the sharing API with settings
      | path        | /Shares/test |
      | permissions | 1            |
      | password    | %public%     |
    Then the OCS status code should be "<ocs_status_code>"
    And the HTTP status code should be "<http_status_code>"
    Examples:
      | ocs_api_version | ocs_status_code | http_status_code |
      | 1               | 403             | 200              |
      | 2               | 403             | 403              |

  @owncloud-issue-enterprise-6423
  Scenario Outline: user cannot share items in the project space with share permission
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "project1" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "project1"
    And user "Alice" has sent the following space share invitation:
      | space           | project1     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" creates a share inside of space "project1" with settings:
      | path        | folder        |
      | shareWith   | Brian         |
      | role        | custom        |
      | permissions | <permissions> |
    Then the HTTP status code should be "400"
    And the OCS status code should be "400"
    And the OCS status message should be "resharing not supported"
    Examples:
      | permissions | description                           |
      | 19          | share + view + edit                   |
      | 21          | share + view + create                 |
      | 23          | share + view + create + edit          |
      | 25          | share + view + delete                 |
      | 27          | share + view + edit + delete          |
      | 29          | share + view + create + delete        |
      | 31          | share + view + create + edit + delete |

  @owncloud-issue-enterprise-6423
  Scenario Outline: user cannot share items in the personal space with share permission
    Given user "Alice" has uploaded file with content "some content" to "/file.txt"
    When user "Alice" creates a share inside of space "Alice Hansen" with settings:
      | path        | file.txt      |
      | shareWith   | Brian         |
      | role        | custom        |
      | permissions | <permissions> |
    Then the HTTP status code should be "400"
    And the OCS status code should be "400"
    And the OCS status message should be "resharing not supported"
    Examples:
      | permissions | description                           |
      | 19          | share + view + edit                   |
      | 21          | share + view + create                 |
      | 23          | share + view + create + edit          |
      | 25          | share + view + delete                 |
      | 27          | share + view + edit + delete          |
      | 29          | share + view + create + delete        |
      | 31          | share + view + create + edit + delete |

