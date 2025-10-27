Feature: Share a file or folder that is inside a space
  As a user with manager space role
  I want to be able to share the data inside the space
  So that other users can have access to it

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "share sub-item" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "share sub-item"
    And user "Alice" has uploaded a file inside space "share sub-item" with content "some content" to "file.txt"
    And using new DAV path


  Scenario Outline: manager of the space can share an entity inside project space to another user with role
    When user "Alice" creates a share inside of space "share sub-item" with settings:
      | path       | <resource>    |
      | shareWith  | Brian         |
      | role       | <space-role>  |
      | expireDate | <expire-date> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And as "Brian" <resource-type> "Shares/<resource>" should exist
    And as user "Brian" the last share should include the following properties:
      | expiration | <expiration> |
    Examples:
      | resource | resource-type | space-role | expire-date              | expiration |
      | folder   | folder        | viewer     |                          |            |
      | folder   | folder        | editor     | 2042-03-25T23:59:59+0100 | 2042-03-25 |
      | file.txt | file          | viewer     |                          |            |
      | file.txt | file          | editor     | 2042-03-25T23:59:59+0100 | 2042-03-25 |


  Scenario Outline: user participant of the project space with manager role can share an entity to another user
    Given user "Alice" has sent the following space share invitation:
      | space           | share sub-item |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | Manager        |
    When user "Brian" creates a share inside of space "share sub-item" with settings:
      | path       | <resource>    |
      | shareWith  | Bob           |
      | role       | <space-role>  |
      | expireDate | <expire-date> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And as "Bob" <resource-type> "Shares/<resource>" should exist
    And as user "Brian" the last share should include the following properties:
      | expiration | <expiration> |
    Examples:
      | resource | resource-type | space-role | expire-date              | expiration |
      | folder   | folder        | viewer     | 2042-03-25T23:59:59+0100 | 2042-03-25 |
      | folder   | folder        | editor     |                          |            |
      | file.txt | file          | viewer     | 2042-03-25T23:59:59+0100 | 2042-03-25 |
      | file.txt | file          | editor     |                          |            |


  Scenario Outline: user participant of the project space without space manager role cannot share an entity to another user
    Given user "Alice" has sent the following space share invitation:
      | space           | share sub-item |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | <space-role>   |
    When user "Brian" creates a share inside of space "share sub-item" with settings:
      | path      | <resource> |
      | shareWith | Bob        |
      | role      | editor     |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the OCS status message should be "No share permission"
    Examples:
      | resource | space-role   |
      | folder   | Space Editor |
      | file.txt | Space Editor |
      | file.txt | Space Viewer |
      | folder   | Space Viewer |


  Scenario Outline: user participant of the project space can see the created resources share
    Given user "Alice" has sent the following space share invitation:
      | space           | share sub-item |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | <space-role>   |
    When user "Alice" creates a share inside of space "share sub-item" with settings:
      | path      | file.txt |
      | shareWith | Bob      |
      | role      | editor   |
    Then for user "Alice" the space "share sub-item" should contain the last created share of the file "file.txt"
    And for user "Brian" the space "share sub-item" should contain the last created share of the file "file.txt"
    Examples:
      | space-role   |
      | Space Editor |
      | Space Viewer |
      | Manager      |


  Scenario: user shares the folder to the group
    Given group "sales" has been created
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    When user "Alice" creates a share inside of space "share sub-item" with settings:
      | path       | folder                   |
      | shareWith  | sales                    |
      | shareType  | 1                        |
      | role       | viewer                   |
      | expireDate | 2042-01-01T23:59:59+0100 |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And as "Brian" folder "Shares/folder" should exist
    And as user "Brian" the last share should include the following properties:
      | expiration | 2042-01-01 |


  Scenario: user changes the expiration date
    Given using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource           | folder                   |
      | space              | share sub-item           |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | Viewer                   |
      | expirationDateTime | 2042-01-01T23:59:59.000Z |
    When user "Alice" changes the last share with settings:
      | expireDate | 2044-01-01T23:59:59.999+01:00 |
      | role       | viewer                        |
    Then the HTTP status code should be "200"
    And as user "Brian" the last share should include the following properties:
      | expiration | 2044-01-01 |


  Scenario: user deletes the expiration date
    Given using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource           | folder                   |
      | space              | share sub-item           |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | Viewer                   |
      | expirationDateTime | 2042-01-01T23:59:59.000Z |
    When user "Alice" changes the last share with settings:
      | expireDate |        |
      | role       | viewer |
    Then the HTTP status code should be "200"
    And as user "Brian" the last share should include the following properties:
      | expiration |  |

  @owncloud-issue-8747
  Scenario: user cannot delete share role
    Given using OCS API version "<ocs_api_version>"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource           | folder                   |
      | space              | share sub-item           |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | Viewer                   |
      | expirationDateTime | 2042-01-01T23:59:59.000Z |
    When user "Alice" changes the last share with settings:
      | role |  |
    Then the HTTP status code should be "400"


  Scenario: check the end of expiration date in user share
    Given using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource           | folder                   |
      | space              | share sub-item           |
      | sharee             | Brian                    |
      | shareType          | user                     |
      | permissionsRole    | Viewer                   |
      | expirationDateTime | 2042-01-01T23:59:59.000Z |
    When user "Alice" expires the last share of resource "folder" inside of the space "share sub-item"
    Then the HTTP status code should be "200"
    And as "Brian" folder "Shares/folder" should not exist

  @owncloud-issue-5823
  Scenario: check the end of expiration date in group share
    Given group "sales" has been created
    And using SharingNG
    And the administrator has added a user "Brian" to the group "sales" using the Graph API
    And user "Alice" has sent the following resource share invitation:
      | resource           | folder                   |
      | space              | share sub-item           |
      | sharee             | sales                    |
      | shareType          | group                    |
      | permissionsRole    | Viewer                   |
      | expirationDateTime | 2042-01-01T23:59:59.000Z |
    When user "Alice" expires the last share of resource "folder" inside of the space "share sub-item"
    Then the HTTP status code should be "200"
    And as "Brian" folder "Shares/folder" should not exist
