Feature: A manager of the space can edit public link
  As an user with manager space role
  I want to be able to edit a public link.
  So that I can remove or change permission, password, expireDate, and name attributes
  Users without the manager role cannot see or edit the public link

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "edit space" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space              | edit space               |
      | permissionsRole    | view                     |
      | password           | %public%                 |
      | expirationDateTime | 2040-01-01T23:59:59.000Z |
      | displayName        | someName                 |
    And user "Alice" has uploaded a file inside space "edit space" with content "some content" to "test.txt"
    And using SharingNG

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: manager of the space can edit public link.
    Given using OCS API version "2"
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | <permissions> |
      | password    | <password>    |
      | name        | <link-name>   |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the fields of the last response to user "Alice" should include
      | item_type         | folder                 |
      | mimetype          | httpd/unix-directory   |
      | file_target       | /                      |
      | path              | /                      |
      | permissions       | <expected-permissions> |
      | share_type        | public_link            |
      | displayname_owner | %displayname%          |
      | name              | <link-name>            |
    When the public downloads file "/test.txt" from inside the last public link shared folder with password "<password>" using the public WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | permissions | expected-permissions      | password   | link-name |
      | 5           | read,create               | newPass:12 |           |
      | 15          | read,update,create,delete | newPass:12 | newName   |


  Scenario Outline: members can see a created public link
    Given using OCS API version "2"
    When user "Alice" shares a space "edit space" with settings:
      | shareWith | Brian        |
      | role      | <space-role> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And for user "Alice" the space "edit space" should contain the last created public link
    And for user "Brian" the space "edit space" should contain the last created public link
    Examples:
      | space-role |
      | manager    |
      | editor     |
      | viewer     |


  Scenario Outline: members of the space try to edit a public link
    Given using OCS API version "2"
    And user "Alice" has sent the following space share invitation:
      | space           | edit space   |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Brian" updates the last public link share using the sharing API with
      | permissions | 15 |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "<ocs-status-code>"
    Examples:
      | space-role   | http-status-code | ocs-status-code |
      | Manager      | 200              | 200             |
      | Space Editor | 401              | 997             |
      | Space Viewer | 401              | 997             |
