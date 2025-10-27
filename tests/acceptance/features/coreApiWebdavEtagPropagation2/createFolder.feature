Feature: propagation of etags when creating folders
  As a client app
  I want metadata (etags) of parent folders to change when a sub-folder is created
  So that the client app can know to re-scan and sync the content of the folder(s)

  Background:
    Given user "Alice" has been created with default attributes

  @owncloud-issue-4251
  Scenario Outline: creating a folder inside a folder changes its etag
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/folder"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/folder"
    When user "Alice" creates folder "/folder/new" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path    |
      | Alice | /       |
      | Alice | /folder |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: creating an invalid folder inside a folder should not change any etags
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/folder"
    And user "Alice" has created folder "/folder/sub"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/folder"
    And user "Alice" has stored etag of element "/folder/sub"
    When user "Alice" creates folder "/folder/sub/.." using the WebDAV API
    Then the HTTP status code should be "405"
    And these etags should not have changed:
      | user  | path        |
      | Alice | /           |
      | Alice | /folder     |
      | Alice | /folder/sub |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-4251 @skipOnReva
  Scenario Outline: sharee creating a folder inside a folder received as a share changes its etag for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "folder" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/folder"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/folder"
    When user "Brian" creates folder "/Shares/folder/new" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path           |
      | Alice | /              |
      | Alice | /folder        |
      | Brian | /              |
      | Brian | /Shares        |
      | Brian | /Shares/folder |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-4251 @skipOnReva
  Scenario Outline: sharer creating a folder inside a shared folder changes etag for all collaborators
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "/folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "folder" synced
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/folder"
    And user "Brian" has stored etag of element "/"
    And user "Brian" has stored etag of element "/Shares"
    And user "Brian" has stored etag of element "/Shares/folder"
    When user "Alice" creates folder "/folder/new" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path           |
      | Alice | /              |
      | Alice | /folder        |
      | Brian | /              |
      | Brian | /Shares        |
      | Brian | /Shares/folder |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @env-config @owncloud-issue-4251 @owncloud-issue-10331
  Scenario: creating a folder in a publicly shared folder changes its etag for the sharer
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And user "Alice" has created folder "/folder"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | folder     |
      | space           | Personal   |
      | permissionsRole | createOnly |
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/folder"
    When the public creates folder "created-by-public" using the public WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed:
      | user  | path    |
      | Alice | /       |
      | Alice | /folder |
