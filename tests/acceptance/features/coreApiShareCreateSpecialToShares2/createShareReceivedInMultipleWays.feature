@skipOnReva
Feature: share resources where the sharee receives the share in multiple ways
  As a user
  I want to receives the same resource share from multiple channels
  So that I can make sure that the sharing works

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: creating and accepting a new share with user who already received a share through their group
    Given using OCS API version "<ocs-api-version>"
    And user "Brian" has disabled auto-accepting
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | File Editor   |
    When user "Alice" shares file "/textfile0.txt" with user "Brian" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should be able to accept pending share "/textfile0.txt" offered by user "Alice"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | share_with             | %username%            |
      | share_with_displayname | %displayname%         |
      | path                   | /textfile0.txt        |
      | file_target            | /Shares/textfile0.txt |
      | permissions            | read,update           |
      | uid_owner              | %username%            |
      | displayname_owner      | %displayname%         |
      | item_type              | file                  |
      | mimetype               | text/plain            |
      | storage_id             | ANY_VALUE             |
      | share_type             | user                  |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1289
  Scenario Outline: share of folder and sub-folder to same user
    Given using OCS API version "<ocs-api-version>"
    And group "grp4" has been created
    And user "Brian" has been added to group "grp4"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has created folder "/PARENT/CHILD"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/PARENT/parent.txt"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/PARENT/CHILD/child.txt"
    When user "Alice" shares folder "/PARENT" with user "Brian" using the sharing API
    And user "Alice" shares folder "/PARENT/CHILD" with group "grp4" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And user "Brian" should see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/CHILD             |
      | /Shares/CHILD/child.txt   |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-2021
  Scenario Outline: sharing subfolder when parent already shared
    Given using OCS API version "<ocs-api-version>"
    And group "grp1" has been created
    And user "Alice" has created folder "/test"
    And user "Alice" has created folder "/test/sub"
    And user "Alice" has sent the following resource share invitation:
      | resource        | test     |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    When user "Alice" shares folder "/test/sub" with user "Brian" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as "Brian" folder "/Shares/sub" should exist
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-2021
  Scenario Outline: sharing subfolder when parent already shared with group of sharer
    Given using OCS API version "<ocs-api-version>"
    And group "grp0" has been created
    And user "Alice" has been added to group "grp0"
    And user "Alice" has created folder "/test"
    And user "Alice" has created folder "/test/sub"
    And user "Alice" has sent the following resource share invitation:
      | resource        | test     |
      | space           | Personal |
      | sharee          | grp0     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    When user "Alice" shares folder "/test/sub" with user "Brian" using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And as "Brian" folder "/Shares/sub" should exist
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-2131
  Scenario Outline: multiple users share a file with the same name but different permissions to a user
    Given using OCS API version "<ocs-api-version>"
    And user "Carol" has been created with default attributes
    And user "Brian" has uploaded file with content "First data" to "/randomfile.txt"
    And user "Carol" has uploaded file with content "Second data" to "/randomfile.txt"
    When user "Brian" shares file "randomfile.txt" with user "Alice" with permissions "read" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as "Alice" the info about the last share by user "Brian" with user "Alice" should include
      | uid_owner   | %username%             |
      | share_with  | %username%             |
      | file_target | /Shares/randomfile.txt |
      | item_type   | file                   |
      | permissions | read                   |
    When user "Carol" shares file "randomfile.txt" with user "Alice" with permissions "read,update" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as "Alice" the info about the last share by user "Carol" with user "Alice" should include
      | uid_owner   | %username%             |
      | share_with  | %username%             |
      | file_target | /Shares/randomfile.txt |
      | item_type   | file                   |
      | permissions | read,update            |
    And the content of file "/Shares/randomfile.txt" for user "Alice" should be "First data"
    And the content of file "/Shares/randomfile (1).txt" for user "Alice" should be "Second data"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-2131
  Scenario Outline: multiple users share a folder with the same name to a user
    Given using OCS API version "<ocs-api-version>"
    And user "Carol" has been created with default attributes
    And user "Brian" has created folder "/zzzfolder"
    And user "Brian" has created folder "zzzfolder/Brian"
    And user "Carol" has created folder "/zzzfolder"
    And user "Carol" has created folder "zzzfolder/Carol"
    When user "Brian" shares folder "zzzfolder" with user "Alice" with permissions "read,delete" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as "Alice" the info about the last share by user "Brian" with user "Alice" should include
      | uid_owner   | %username%        |
      | share_with  | %username%        |
      | file_target | /Shares/zzzfolder |
      | item_type   | folder            |
      | permissions | read,delete       |
    When user "Carol" shares folder "zzzfolder" with user "Alice" with permissions "read" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And as "Alice" the info about the last share by user "Carol" with user "Alice" should include
      | uid_owner   | %username%        |
      | share_with  | %username%        |
      | file_target | /Shares/zzzfolder |
      | item_type   | folder            |
      | permissions | read              |
    And as "Alice" folder "/Shares/zzzfolder/Brian" should exist
    And as "Alice" folder "/Shares/zzzfolder (1)/Carol" should exist
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @skipOnReva
  Scenario Outline: share with a group and then add a user to that group that already has a file with the shared name
    Given using OCS API version "<ocs-api-version>"
    And user "Carol" has been created with default attributes
    And user "Carol" has disabled auto-accepting
    And these groups have been created:
      | groupname |
      | grp1      |
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "Shared content" to "lorem.txt"
    And user "Carol" has uploaded file with content "My content" to "lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt   |
      | space           | Personal    |
      | sharee          | grp1        |
      | shareType       | group       |
      | permissionsRole | File Editor |
    And user "Brian" has a share "lorem.txt" synced
    When the administrator adds user "Carol" to group "grp1" using the provisioning API
    Then the HTTP status code should be "204"
    And user "Carol" should be able to accept pending share "/lorem.txt" offered by user "Alice"
    And the content of file "Shares/lorem.txt" for user "Brian" should be "Shared content"
    And the content of file "lorem.txt" for user "Carol" should be "My content"
    And the content of file "Shares/lorem.txt" for user "Carol" should be "Shared content"
    Examples:
      | ocs-api-version |
      | 1               |
      | 2               |

  @owncloud-issue-2440
  Scenario: sharing parent folder to user with all permissions and its child folder to group with read permission then check create operation
    Given group "grp1" has been created
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                  |
      | /parent               |
      | /parent/child1        |
      | /parent/child1/child2 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And user "Alice" has a share "child1" synced
    And user "Brian" has a share "child1" synced
    And user "Brian" should be able to create folder "/Shares/parent/fo1"
    And user "Brian" should be able to create folder "/Shares/parent/child1/fo2"
    And user "Alice" should not be able to create folder "/Shares/child1/fo3"

  @owncloud-issue-2440
  Scenario: sharing parent folder to user with all permissions and its child folder to group with read permission then check rename operation
    Given group "grp1" has been created
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                  |
      | /parent               |
      | /parent/child1        |
      | /parent/child1/child2 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has uploaded file with content "some data" to "/parent/child1/child2/textfile-2.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And user "Alice" has a share "child1" synced
    And user "Brian" has a share "child1" synced
    Then user "Brian" should be able to rename file "/Shares/parent/child1/child2/textfile-2.txt" to "/Shares/parent/child1/child2/rename.txt"
    And user "Brian" should be able to rename file "/Shares/child1/child2/rename.txt" to "/Shares/child1/child2/rename2.txt"
    And user "Alice" should not be able to rename file "/Shares/child1/child2/rename2.txt" to "/Shares/child1/child2/rename3.txt"

  @owncloud-issue-2440
  Scenario: sharing parent folder to user with all permissions and its child folder to group with read permission then check delete operation
    Given group "grp1" has been created
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                         |
      | /parent                      |
      | /parent/child1               |
      | /parent/child1/child2        |
      | /parent/child1/child2/child3 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has uploaded file with content "some data" to "/parent/child1/child2/child3/textfile-2.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And user "Alice" has a share "child1" synced
    And user "Brian" has a share "child1" synced
    And user "Brian" should be able to delete file "/Shares/parent/child1/child2/child3/textfile-2.txt"
    And user "Brian" should be able to delete folder "/Shares/child1/child2/child3"
    And user "Alice" should not be able to delete folder "/Shares/child1/child2"


  Scenario: sharing parent folder to group with read permission and its child folder to user with all permissions then check create operation
    Given group "grp1" has been created
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                  |
      | /parent               |
      | /parent/child1        |
      | /parent/child1/child2 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Alice" has a share "parent" synced
    And user "Brian" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Brian" has a share "child1" synced
    Then user "Brian" should be able to create folder "/Shares/child1/fo1"
    And user "Brian" should be able to create folder "/Shares/child1/child2/fo2"
    But user "Brian" should not be able to create folder "/Shares/parent/fo3"
    And user "Brian" should not be able to create folder "/Shares/parent/fo3"
    And user "Alice" should not be able to create folder "/Shares/parent/fo3"

  @owncloud-issue-2440
  Scenario: sharing parent folder to group with read permission and its child folder to user with all permissions then check rename operation
    Given group "grp1" has been created
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                  |
      | /parent               |
      | /parent/child1        |
      | /parent/child1/child2 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has uploaded file with content "some data" to "/parent/child1/child2/textfile-2.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "parent" synced
    And user "Alice" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Brian" has a share "child1" synced
    And user "Brian" should be able to rename file "/Shares/child1/child2/textfile-2.txt" to "/Shares/child1/child2/rename.txt"
    And user "Brian" should be able to rename file "/Shares/parent/child1/child2/rename.txt" to "/Shares/parent/child1/child2/rename2.txt"
    And user "Alice" should not be able to rename file "/Shares/parent/child1/child2/rename2.txt" to "/Shares/parent/child1/child2/rename3.txt"

  @owncloud-issue-2440
  Scenario: sharing parent folder to group with read permission and its child folder to user with all permissions then check delete operation
    Given group "grp1" has been created
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                  |
      | /parent               |
      | /parent/child1        |
      | /parent/child1/child2 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has uploaded file with content "some data" to "/parent/child1/child2/textfile-2.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Alice" has a share "parent" synced
    And user "Brian" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Brian" has a share "child1" synced
    And user "Brian" should be able to delete file "/Shares/child1/child2/textfile-2.txt"
    And user "Brian" should be able to delete folder "/Shares/parent/child1/child2"
    And user "Alice" should not be able to delete folder "/Shares/parent/child1"


  Scenario: sharing parent folder to one group with all permissions and its child folder to another group with read permission
    Given these groups have been created:
      | groupname |
      | grp1      |
      | grp2      |
      | grp3      |
    And user "Carol" has been created with default attributes
    And user "Carol" has created the following folders
      | path                  |
      | /parent               |
      | /parent/child1        |
      | /parent/child1/child2 |
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp2"
    And user "Carol" has uploaded file with content "some data" to "/parent/child1/child2/textfile-2.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Alice" has a share "parent" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | parent/child1 |
      | space           | Personal      |
      | sharee          | grp2          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "child1" synced
    Then user "Alice" should be able to create folder "/Shares/parent/child1/fo1"
    And user "Alice" should be able to create folder "/Shares/parent/child1/child2/fo2"
    And user "Alice" should be able to delete folder "/Shares/parent/child1/fo1"
    And user "Alice" should be able to delete folder "/Shares/parent/child1/child2/fo2"
    And user "Alice" should be able to rename file "/Shares/parent/child1/child2/textfile-2.txt" to "/Shares/parent/child1/child2/rename.txt"
    And user "Alice" should not be able to share folder "/Shares/parent/child1" with group "grp3" with permissions "all" using the sharing API
    And as "Brian" folder "/Shares/child1" should exist
    And user "Brian" should not be able to create folder "/Shares/child1/fo1"
    And user "Brian" should not be able to create folder "/Shares/child1/child2/fo2"
    And user "Brian" should not be able to rename file "/Shares/child1/child2/rename.txt" to "/Shares/child1/child2/rename2.txt"
    And user "Brian" should not be able to share folder "/Shares/child1" with group "grp3" with permissions "read" using the sharing API


  Scenario: share receiver renames the received group share and shares same folder through user share again
    Given group "grp" has been created
    And user "Brian" has been added to group "grp"
    And user "Alice" has been added to group "grp"
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/child"
    And user "Alice" has uploaded file with content "Share content" to "parent/child/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp      |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "parent" synced
    And user "Brian" should be able to rename folder "/Shares/parent" to "/Shares/sharedParent"
    And user "Alice" should be able to share folder "parent" with user "Brian" with permissions "read" using the sharing API
    # Note: Brian has already accepted the share of this resource as a member of "grp".
    #       Now he has also received the same resource shared directly to "Brian".
    #       The server should effectively "auto-accept" this new "copy" of the resource
    #       and present to Brian only the single resource "Shares/sharedParent"
    And as "Brian" folder "Shares/parent" should not exist
    And as "Brian" folder "Shares/sharedParent" should exist
    And as "Brian" file "Shares/sharedParent/child/lorem.txt" should exist

  @owncloud-issue-7555
  Scenario Outline: share receiver renames a group share and receives same resource through user share with additional permissions
    Given using OCS API version "<ocs-api-version>"
    And group "grp" has been created
    And user "Brian" has been added to group "grp"
    And user "Alice" has been added to group "grp"
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/child"
    And user "Alice" has uploaded file with content "Share content" to "parent/child/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp      |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "parent" synced
    And user "Brian" has moved folder "/Shares/parent" to "/Shares/sharedParent"
    When user "Alice" shares folder "parent" with user "Brian" with permissions "all" using the sharing API
    # Note: Brian has already accepted the share of this resource as a member of "grp".
    #       Now he has also received the same resource shared directly to "Brian".
    #       The server should effectively "auto-accept" this new "copy" of the resource
    #       and present to Brian only the single resource "Shares/sharedParent"
    Then as "Brian" folder "Shares/parent" should not exist
    And as "Brian" folder "Shares/sharedParent" should exist
    And as "Brian" file "Shares/sharedParent/child/lorem.txt" should exist
    Examples:
      | ocs-api-version |
      | 1               |
      | 2               |


  Scenario: share receiver renames a group share and receives same resource through user share with less permissions
    Given group "grp" has been created
    And user "Brian" has been added to group "grp"
    And user "Alice" has been added to group "grp"
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/child"
    And user "Alice" has uploaded file with content "Share content" to "parent/child/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | grp      |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "parent" synced
    And user "Brian" should be able to rename folder "/Shares/parent" to "/Shares/sharedParent"
    And user "Alice" should be able to share folder "parent" with user "Brian" with permissions "read" using the sharing API
    # Note: Brian has already accepted the share of this resource as a member of "grp".
    #       Now he has also received the same resource shared directly to "Brian".
    #       The server should effectively "auto-accept" this new "copy" of the resource
    #       and present to Brian only the single resource "Shares/sharedParent"
    And as "Brian" folder "Shares/parent" should not exist
    And as "Brian" folder "Shares/sharedParent" should exist
    And as "Brian" file "Shares/sharedParent/child/lorem.txt" should exist
