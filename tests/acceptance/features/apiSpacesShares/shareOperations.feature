Feature: sharing
  As a user
  I want to share resources with different permissions
  So that I can manage the access to the resource

  Background:
    Given using OCS API version "1"
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario: correct webdav share-permissions for received file with edit permissions
    Given user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt     |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/tmp.txt" inside space "Shares" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "3"


  Scenario: correct webdav share-permissions for received group shared file with edit permissions
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt     |
      | space           | Personal    |
      | sharee          | grp1        |
      | shareType       | group       |
      | permissionsRole | File Editor |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/tmp.txt" inside space "Shares" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "3"


  Scenario: correct webdav share-permissions for received file with read permissions
    Given user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/tmp.txt" inside space "Shares" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "1"


  Scenario: correct webdav share-permissions for received group shared file with read permissions
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "foo" to "/tmp.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp.txt  |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "tmp.txt" synced
    When user "Brian" gets the following properties of file "/tmp.txt" inside space "Shares" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "1"


  Scenario: correct webdav share-permissions for received folder with all permissions
    Given user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Brian" gets the following properties of folder "/tmp" inside space "Shares" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "15"


  Scenario: correct webdav share-permissions for received group shared folder with all permissions
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Brian" gets the following properties of folder "/tmp" inside space "Shares" using the WebDAV API
      | propertyName          |
      | ocs:share-permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "ocs:share-permissions" with value "15"


  Scenario: correct webdav share-permissions for received folder with all permissions but edit
    Given user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,create,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/tmp" inside space "Shares" should contain a property "ocs:share-permissions" with value "13"


  Scenario: correct webdav share-permissions for received group shared folder with all permissions but edit
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,create,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/tmp" inside space "Shares" should contain a property "ocs:share-permissions" with value "13"


  Scenario: correct webdav share-permissions for received folder with all permissions but create
    Given user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/tmp" inside space "Shares" should contain a property "ocs:share-permissions" with value "11"


  Scenario: correct webdav share-permissions for received group shared folder with all permissions but create
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | delete,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/tmp" inside space "Shares" should contain a property "ocs:share-permissions" with value "11"


  Scenario: correct webdav share-permissions for received folder with all permissions but delete
    Given user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | create,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/tmp" inside space "Shares" should contain a property "ocs:share-permissions" with value "7"


  Scenario: correct webdav share-permissions for received group shared folder with all permissions but delete
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/tmp"
    And using SharingNG
    And user "Alice" has sent the following resource share invitation:
      | resource        | tmp      |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "tmp" synced
    When user "Alice" updates the last share using the sharing API with
      | permissions | create,update,read |
    Then the HTTP status code should be "200"
    And as user "Brian" folder "/tmp" inside space "Shares" should contain a property "ocs:share-permissions" with value "7"


  Scenario: uploading a file to a folder received as a read-only user share
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist


  Scenario: uploading a file to a folder received as a read-only group share
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist


  Scenario: uploading a file to a folder received as a upload-only user share
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the following headers should match these regular expressions for user "Brian"
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      new description
      """


  Scenario: uploading a file to a folder received as a upload-only group share
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the following headers should match these regular expressions for user "Brian"
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      new description
      """


  Scenario: uploading a file to a folder received as a read/write user share
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      new description
      """


  Scenario: uploading a file to a folder received as a read/write group share
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/textfile.txt" for user "Alice" should be:
      """
      new description
      """


  Scenario: uploading to a user shared folder with read/write permission when the sharer has insufficient quota
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "1"
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist


  Scenario: uploading to a user shared folder with read/write permission when the sharer has insufficient quota
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "1"
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist


  Scenario: uploading to a user shared folder with upload-only permission when the sharer has insufficient quota
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "1"
    When user "Brian" uploads a file inside space "Shares" with content "new description" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist


  Scenario: uploading a file to a group shared folder with upload-only permission when the sharer has insufficient quota
    Given group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Uploader |
    And user "Brian" has a share "FOLDER" synced
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "10"
    When user "Brian" uploads a file inside space "Shares" with content "new descriptionfgshsywhhh" to "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And as "Alice" file "/FOLDER/textfile.txt" should not exist


  Scenario Outline: sharer can download file uploaded with different permission by sharee to a shared folder
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER             |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Brian" has a share "FOLDER" synced
    When user "Brian" uploads a file inside space "Shares" with content "some content" to "/FOLDER/textfile.txt" using the WebDAV API
    And user "Alice" downloads file "/FOLDER/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "some content"
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |

  @env-config
  Scenario: sharee cannot download file shared with Secure viewer permission by sharee
    Given using old DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt  |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And user "Brian" has a share "textfile.txt" synced
    And user "Brian" downloads file "/Shares/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"

  @env-config
  Scenario: sharee cannot download file inside folder shared with Secure viewer permission by sharee
    Given using old DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "FolderToShare/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And user "Brian" has a share "FolderToShare" synced
    And user "Brian" downloads file "/Shares/FolderToShare/textfile.txt" using the WebDAV API
    Then the HTTP status code should be "403"


  Scenario Outline: space admin tries to remove password of a public link share (change/create permission)
    Given using spaces DAV path
    And using OCS API version "<ocs-api-version>"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created folder "FOLDER"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER             |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" updates the last public link share using the sharing API with
      | path        | /FOLDER       |
      | permissions | <permissions> |
      | password    |               |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "400"
    And the OCS status message should be "missing required password"
    Examples:
      | ocs-api-version | permissions-role | permissions | http-status-code |
      | 1               | edit             | change      | 200              |
      | 2               | edit             | change      | 400              |
      | 1               | createOnly       | create      | 200              |
      | 2               | createOnly       | create      | 400              |


  Scenario Outline: space admin removes password of a public link share (read permission)
    Given using spaces DAV path
    And using OCS API version "<ocs-api-version>"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created folder "FOLDER"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When user "Alice" updates the last public link share using the sharing API with
      | path        | /FOLDER |
      | permissions | read    |
      | password    |         |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
