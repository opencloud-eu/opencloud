Feature: Public can download folders from project space public link
  As a public
  I want to be able to download folder from public link
  So that I can gain access to it's contents

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API

  @env-config @owncloud-issue-9724 @owncloud-issue-10331
  Scenario: download a folder from public link of a space
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using SharingNG
    And user "Alice" has created a folder "NewFolder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "NewFolder/test.txt"
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | displayName     | someName  |
      | permissionsRole | view      |
    When public downloads the folder "NewFolder" from the last created public link using the public files API
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name               | content      |
      | NewFolder/test.txt | some content |

  @env-config @owncloud-issue-5229 @owncloud-issue-9724 @owncloud-issue-10331
  Scenario: download a folder from public link of a folder inside a space
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using SharingNG
    And user "Alice" has created a folder "NewFolder" in space "new-space"
    And user "Alice" has created a folder "NewFolder/folder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "NewFolder/folder/test.txt"
    And user "Alice" has created the following resource link share:
      | resource        | NewFolder   |
      | space           | new-space   |
      | displayName     | public link |
      | permissionsRole | view        |
    When public downloads the folder "folder" from the last created public link using the public files API
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name            | content      |
      | folder/test.txt | some content |
