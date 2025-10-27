@skipOnReva @owncloud-issue-1289 @owncloud-issue-1328
Feature: sharing

  Background:
    Given using OCS API version "1"
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |

  @owncloud-issue-8242 @owncloud-issue-10334
  Scenario Outline: sharer renames the shared item (old/new webdav)
    Given user "Alice" has uploaded file with content "foo" to "sharefile.txt"
    And using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "sharefile.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt |
      | space           | Personal      |
      | sharee          | Carol         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Carol" has a share "sharefile.txt" synced
    When user "Alice" moves file "sharefile.txt" to "renamedsharefile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "renamedsharefile.txt" should exist
    And as "Brian" file "Shares/sharefile.txt" should exist
    And as "Carol" file "Shares/sharefile.txt" should exist
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Alice" the value of the item "//oc:name" of path "<dav-path>/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    And as user "Alice" the value of the item "//d:displayname" of path "<dav-path>/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>/Shares"
    Then the HTTP status code should be "207"
    And as user "Brian" the value of the item "//oc:name" of path "<dav-path>/Shares/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Brian" the value of the item "//d:displayname" of path "<dav-path>/Shares/sharefile.txt" in the response should be "sharefile.txt"
    When user "Carol" sends HTTP method "PROPFIND" to URL "<dav-path>/Shares"
    Then the HTTP status code should be "207"
    And as user "Carol" the value of the item "//oc:name" of path "<dav-path>/Shares/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Carol" the value of the item "//d:displayname" of path "<dav-path>/Shares/sharefile.txt" in the response should be "sharefile.txt"
    Examples:
      | dav-path-version | dav-path              |
      | old              | /webdav               |
      | new              | /dav/files/%username% |

  @owncloud-issue-8242
  Scenario Outline: sharer renames the shared item (spaces webdav)
    Given user "Alice" has uploaded file with content "foo" to "sharefile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "sharefile.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt |
      | space           | Personal      |
      | sharee          | Carol         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Carol" has a share "sharefile.txt" synced
    When user "Alice" moves file "sharefile.txt" to "renamedsharefile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "renamedsharefile.txt" should exist
    And as "Brian" file "Shares/sharefile.txt" should exist
    And as "Carol" file "Shares/sharefile.txt" should exist
    And using spaces DAV path
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path-personal>"
    Then the HTTP status code should be "207"
    And as user "Alice" the value of the item "//oc:name" of path "<dav-path-personal>/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    And as user "Alice" the value of the item "//d:displayname" of path "<dav-path-personal>/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Brian" the value of the item "//oc:name" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Brian" the value of the item "//d:displayname" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    When user "Carol" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Carol" the value of the item "//oc:name" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Carol" the value of the item "//d:displayname" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    Examples:
      | dav-path                                 | dav-path-personal     |
      | /dav/spaces/%shares_drive_id%            | /dav/spaces/%spaceid% |

  @owncloud-issue-8242 @owncloud-issue-10334 @env-config
  Scenario Outline: share receiver renames the shared item (old/new webdav)
    Given user "Alice" has uploaded file with content "foo" to "/sharefile.txt"
    And the administrator has enabled the permissions role "Secure Viewer"
    And using <dav-path-version> DAV path
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "sharefile.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt      |
      | space           | Personal           |
      | sharee          | Carol              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Carol" has a share "sharefile.txt" synced
    When user "Carol" moves file "Shares/sharefile.txt" to "Shares/renamedsharefile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Carol" file "Shares/renamedsharefile.txt" should exist
    And as "Brian" file "Shares/sharefile.txt" should exist
    And as "Alice" file "sharefile.txt" should exist
    When user "Carol" sends HTTP method "PROPFIND" to URL "<dav-path>/Shares"
    Then the HTTP status code should be "207"
    And as user "Carol" the value of the item "//oc:name" of path "<dav-path>/Shares/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    And as user "Carol" the value of the item "//d:displayname" of path "<dav-path>/Shares/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Alice" the value of the item "//oc:name" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Alice" the value of the item "//d:displayname" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>/Shares"
    Then the HTTP status code should be "207"
    And as user "Brian" the value of the item "//oc:name" of path "<dav-path>/Shares/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Brian" the value of the item "//d:displayname" of path "<dav-path>/Shares/sharefile.txt" in the response should be "sharefile.txt"
    Examples:
      | dav-path-version | dav-path              | permissions-role |
      | old              | /webdav               | Viewer           |
      | old              | /webdav               | Secure Viewer    |
      | new              | /dav/files/%username% | Viewer           |
      | new              | /dav/files/%username% | Secure Viewer    |

  @owncloud-issue-8242 @env-config
  Scenario Outline: share receiver renames the shared item (spaces webdav)
    Given user "Alice" has uploaded file with content "foo" to "/sharefile.txt"
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "sharefile.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | sharefile.txt      |
      | space           | Personal           |
      | sharee          | Carol              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Carol" has a share "sharefile.txt" synced
    When user "Carol" moves file "Shares/sharefile.txt" to "Shares/renamedsharefile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Carol" file "Shares/renamedsharefile.txt" should exist
    And as "Brian" file "Shares/sharefile.txt" should exist
    And as "Alice" file "sharefile.txt" should exist
    And using spaces DAV path
    When user "Carol" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Carol" the value of the item "//oc:name" of path "<dav-path>/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    And as user "Carol" the value of the item "//d:displayname" of path "<dav-path>/renamedsharefile.txt" in the response should be "renamedsharefile.txt"
    When user "Alice" sends HTTP method "PROPFIND" to URL "<dav-path-personal>"
    Then the HTTP status code should be "207"
    And as user "Alice" the value of the item "//oc:name" of path "<dav-path-personal>/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Alice" the value of the item "//d:displayname" of path "<dav-path-personal>/sharefile.txt" in the response should be "sharefile.txt"
    When user "Brian" sends HTTP method "PROPFIND" to URL "<dav-path>"
    Then the HTTP status code should be "207"
    And as user "Brian" the value of the item "//oc:name" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    And as user "Brian" the value of the item "//d:displayname" of path "<dav-path>/sharefile.txt" in the response should be "sharefile.txt"
    Examples:
      | dav-path                                 | dav-path-personal     | permissions-role |
      | /dav/spaces/%shares_drive_id%            | /dav/spaces/%spaceid% | Viewer           |
      | /dav/spaces/%shares_drive_id%            | /dav/spaces/%spaceid% | Secure Viewer    |


  Scenario Outline: keep group share when the one user renames the share and the user is deleted
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Carol" has been added to group "grp1"
    And user "Alice" has created folder "/TMP"
    And user "Alice" has sent the following resource share invitation:
      | resource        | TMP      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Carol" has a share "/TMP" synced
    When user "Carol" moves folder "/Shares/TMP" to "/Shares/new" using the WebDAV API
    And the administrator deletes user "Carol" using the provisioning API
    Then the HTTP status code of responses on each endpoint should be "201, 204" respectively
    And as "Alice" file "Shares/TMP" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received share with read, change permissions inside the Shares folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has uploaded file with content "thisIsAFileInsideTheSharedFolder" to "/folderToShare/fileInside"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
    And user "Brian" has a share "folderToShare" synced
    When user "Brian" moves folder "/Shares/folderToShare" to "/Shares/myFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" folder "/Shares/myFolder" should exist
    But as "Alice" folder "/Shares/myFolder" should not exist
    When user "Brian" moves file "/Shares/myFolder/fileInside" to "/Shares/myFolder/renamedFile" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/Shares/myFolder/renamedFile" should exist
    And as "Alice" file "/folderToShare/renamedFile" should exist
    But as "Alice" file "/folderToShare/fileInside" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @env-config
  Scenario Outline: receiver tries to rename a received share with read permissions inside the Shares folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "folderToShare"
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has created folder "folderToShare/folderInside"
    And user "Alice" has uploaded file with content "thisIsAFileInsideTheSharedFolder" to "/folderToShare/fileInside"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "folderToShare" synced
    When user "Brian" moves folder "/Shares/folderToShare" to "/Shares/myFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" folder "/Shares/myFolder" should exist
    But as "Alice" folder "/Shares/myFolder" should not exist
    When user "Brian" moves file "/Shares/myFolder/fileInside" to "/Shares/myFolder/renamedFile" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Brian" file "/Shares/myFolder/renamedFile" should not exist
    But as "Brian" file "Shares/myFolder/fileInside" should exist
    When user "Brian" moves folder "/Shares/myFolder/folderInside" to "/Shares/myFolder/renamedFolder" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Brian" folder "/Shares/myFolder/renamedFolder" should not exist
    But as "Brian" folder "Shares/myFolder/folderInside" should exist
    Examples:
      | permissions-role | dav-path-version |
      | Viewer           | old              |
      | Secure Viewer    | old              |
      | Viewer           | new              |
      | Secure Viewer    | new              |
      | Viewer           | spaces           |
      | Secure Viewer    | spaces           |


  Scenario Outline: receiver renames a received folder share to a different name on the same folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" moves folder "/Shares/PARENT" to "/Shares/myFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" folder "/Shares/myFolder" should exist
    But as "Alice" folder "myFolder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received file share to different name on the same folder
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "fileToShare.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | fileToShare.txt |
      | space           | Personal        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | File Editor     |
    And user "Brian" has a share "fileToShare.txt" synced
    When user "Brian" moves file "/Shares/fileToShare.txt" to "/Shares/newFile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/Shares/newFile.txt" should exist
    But as "Alice" file "newFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received file share to different name on the same folder for group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "fileToShare.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | fileToShare.txt |
      | space           | Personal        |
      | sharee          | grp1            |
      | shareType       | group           |
      | permissionsRole | File Editor     |
    And user "Brian" has a share "fileToShare.txt" synced
    When user "Brian" moves file "/Shares/fileToShare.txt" to "/Shares/newFile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/Shares/newFile.txt" should exist
    But as "Alice" file "newFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received folder share to different name on the same folder for group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Alice" has created folder "PARENT"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" moves folder "/Shares/PARENT" to "/Shares/myFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" folder "/Shares/myFolder" should exist
    But as "Alice" folder "myFolder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received file share with read,update permissions inside the Shares folder in group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "fileToShare.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | fileToShare.txt |
      | space           | Personal        |
      | sharee          | grp1            |
      | shareType       | group           |
      | permissionsRole | File Editor     |
    And user "Brian" has a share "fileToShare.txt" synced
    When user "Brian" moves folder "/Shares/fileToShare.txt" to "/Shares/newFile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/Shares/newFile.txt" should exist
    But as "Alice" file "/Shares/newFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received folder share with read, change permissions inside the Shares folder in group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Alice" has created folder "PARENT"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" moves folder "/Shares/PARENT" to "/Shares/myFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" folder "/Shares/myFolder" should exist
    But as "Alice" folder "/Shares/myFolder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received file share with share, read permissions inside the Shares folder in group sharing)
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "fileToShare.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | fileToShare.txt |
      | space           | Personal        |
      | sharee          | grp1            |
      | shareType       | group           |
      | permissionsRole | Viewer          |
    And user "Brian" has a share "fileToShare.txt" synced
    When user "Brian" moves file "/Shares/fileToShare.txt" to "/Shares/newFile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/Shares/newFile.txt" should exist
    But as "Alice" file "/Shares/newFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: receiver renames a received folder share with read permissions inside the Shares folder in group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Alice" has created folder "PARENT"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" moves folder "/Shares/PARENT" to "/Shares/myFolder" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" folder "/Shares/myFolder" should exist
    But as "Alice" folder "/Shares/myFolder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-2141
  Scenario Outline: receiver renames a received folder share to name with special characters in group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Carol" has been added to group "grp1"
    And user "Alice" has created folder "<sharer-folder>"
    And user "Alice" has created folder "<group-folder>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <sharer-folder> |
      | space           | Personal        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | Editor          |
    And user "Brian" has a share "<sharer-folder>" synced
    When user "Brian" moves folder "/Shares/<sharer-folder>" to "/Shares/<receiver-folder>" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "<receiver-folder>" should not exist
    And as "Brian" folder "/Shares/<receiver-folder>" should exist
    When user "Alice" shares folder "<group-folder>" with group "grp1" using the sharing API
    And user "Carol" moves folder "/Shares/<group-folder>" to "/Shares/<receiver-folder>" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "<receiver-folder>" should not exist
    But as "Carol" folder "/Shares/<receiver-folder>" should exist
    Examples:
      | sharer-folder | group-folder    | receiver-folder | dav-path-version |
      | ?abc=oc #     | ?abc=oc g%rp#   | # oc?test=oc&a  | old              |
      | @a#8a=b?c=d   | @a#8a=b?c=d grp | ?a#8 a=b?c=d    | old              |
      | ?abc=oc #     | ?abc=oc g%rp#   | # oc?test=oc&a  | new              |
      | @a#8a=b?c=d   | @a#8a=b?c=d grp | ?a#8 a=b?c=d    | new              |
      | ?abc=oc #     | ?abc=oc g%rp#   | # oc?test=oc&a  | spaces           |
      | @a#8a=b?c=d   | @a#8a=b?c=d grp | ?a#8 a=b?c=d    | spaces           |

  @owncloud-issue-2141
  Scenario Outline: receiver renames a received file share to name with special characters with read, change permissions in group sharing
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Carol" has been added to group "grp1"
    And user "Alice" has created folder "<sharer-folder>"
    And user "Alice" has created folder "<group-folder>"
    And user "Alice" has uploaded file with content "thisIsAFileInsideTheSharedFolder" to "/<sharer-folder>/fileInside"
    And user "Alice" has uploaded file with content "thisIsAFileInsideTheSharedFolder" to "/<group-folder>/fileInside"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <sharer-folder> |
      | space           | Personal        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | Editor          |
    And user "Brian" has a share "<sharer-folder>" synced
    When user "Brian" moves folder "/Shares/<sharer-folder>/fileInside" to "/Shares/<sharer-folder>/<receiver_file>" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "<sharer-folder>/<receiver_file>" should exist
    And as "Brian" file "/Shares/<sharer-folder>/<receiver_file>" should exist
    When user "Alice" shares folder "<group-folder>" with group "grp1" with permissions "read,change" using the sharing API
    And user "Carol" moves folder "/Shares/<group-folder>/fileInside" to "/Shares/<group-folder>/<receiver_file>" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "<group-folder>/<receiver_file>" should exist
    And as "Carol" file "/Shares/<group-folder>/<receiver_file>" should exist
    Examples:
      | sharer-folder | group-folder    | receiver_file  | dav-path-version |
      | ?abc=oc #     | ?abc=oc g%rp#   | # oc?test=oc&a | old              |
      | @a#8a=b?c=d   | @a#8a=b?c=d grp | ?a#8 a=b?c=d   | old              |
      | ?abc=oc #     | ?abc=oc g%rp#   | # oc?test=oc&a | new              |
      | @a#8a=b?c=d   | @a#8a=b?c=d grp | ?a#8 a=b?c=d   | new              |
      | ?abc=oc #     | ?abc=oc g%rp#   | # oc?test=oc&a | spaces           |
      | @a#8a=b?c=d   | @a#8a=b?c=d grp | ?a#8 a=b?c=d   | spaces           |
