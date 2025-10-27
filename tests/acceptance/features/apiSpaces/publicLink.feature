@env-config
Feature: public link for a space

  Background:
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And these users have been created with default attributes:
      | username |
      | Alice    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "public space" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | public space |
      | permissionsRole | view         |
    And using SharingNG

  @owncloud-issue-10331
  Scenario: public tries to upload a file in the public space
    When the public uploads file "test.txt" with content "test" using the public WebDAV API
    And the HTTP status code should be "403"

  @owncloud-issue-10331
  Scenario: public tries to create a folder in the public space
    When the public creates folder "created-by-public" using the public WebDAV API
    And the HTTP status code should be "403"

  @owncloud-issue-10331
  Scenario: public tries to delete a file in the public space
    Given user "Alice" has uploaded a file inside space "public space" with content "some content" to "test.txt"
    When the public deletes file "test.txt" from the last public link share using the public WebDAV API
    And the HTTP status code should be "403"

  @owncloud-issue-10331
  Scenario: public tries to delete a folder in the public space
    And user "Alice" has created a folder "/public-folder" in space "public space"
    When the public deletes folder "public-folder" from the last public link share using the public WebDAV API
    And the HTTP status code should be "403"

  @owncloud-issue-10331
  Scenario: public tries to change content of a resources in the public space
    Given user "Alice" has uploaded a file inside space "public space" with content "some content" to "test.txt"
    When the public overwrites file "test.txt" with content "public content" using the public WebDAV API
    And the HTTP status code should be "403"
