@skipOnReva
Feature: sharing
  As a user
  I want to check the webdav share permissions
  So that I know the resources have proper permissions

  Background:
    Given using OCS API version "1"
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |

  @smokeTest
  Scenario Outline: check webdav share-permissions for owned file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    When user "Alice" gets the following properties of file "/tmp.txt" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "19"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: check webdav share-permissions for received file with edit
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt     |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/Shares/tmp.txt" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "3"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: check webdav share-permissions for received group shared file with edit
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt     |
      | space           | Personal    |
      | sharee          | grp1        |
      | shareType       | group       |
      | permissionsRole | File Editor |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/Shares/tmp.txt" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "3"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @owncloud-issue-2213
  Scenario Outline: check webdav share-permissions for received file without edit permissions
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/Shares/tmp.txt" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "1"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: check webdav share-permissions for owned folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/tmp"
    When user "Alice" gets the following properties of folder "/" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "31"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: check webdav share-permissions for received folder with all permissions
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Brian" gets the following properties of folder "/Shares/tmp" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "15"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: check webdav share-permissions for received group shared folder with all permissions
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Brian" gets the following properties of folder "/Shares/tmp" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "15"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @owncloud-issue-2213
  Scenario: check webdav share-permissions for received folder with all permissions but edit
    Given user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,create,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/Shares/tmp" should contain a property "ocs:share-permissions" with value "13"


  Scenario: check webdav share-permissions for received group shared folder with all permissions but edit
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,create,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/Shares/tmp" should contain a property "ocs:share-permissions" with value "13"

  @skipOnReva
  Scenario: check webdav share-permissions for received folder with all permissions but create
    Given user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/Shares/tmp" should contain a property "ocs:share-permissions" with value "11"


  Scenario: check webdav share-permissions for received group shared folder with all permissions but create
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/Shares/tmp" should contain a property "ocs:share-permissions" with value "11"

  @skipOnReva
  Scenario: check webdav share-permissions for received folder with all permissions but delete
    Given user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp" synced
    And using SharingNG
    When user "Alice" updates the last share using the sharing API with
      | permissions | create,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/Shares/tmp" should contain a property "ocs:share-permissions" with value "7"


  Scenario: check webdav share-permissions for received group shared folder with all permissions but delete
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | create,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/Shares/tmp" should contain a property "ocs:share-permissions" with value "7"
