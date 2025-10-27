Feature: Share a file or folder that is inside a space via public link
  As a user with manager space role
  I want to be able to share the data inside the space via public link
  So that an anonymous user can have access to certain resources

  folder permissions:
  | role        | permissions |
  | internal    | 0           |
  | viewer      | 1           |
  | uploader    | 4           |
  | contributor | 5           |
  | editor      | 15          |

  file permissions:
  | role     | permissions |
  | internal | 0           |
  | viewer   | 1           |
  | editor   | 3           |

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "share sub-item" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "share sub-item"
    And user "Alice" has uploaded a file inside space "share sub-item" with content "some content" to "folder/file.txt"

  @owncloud-issue-5139
  Scenario Outline: manager of the space can share an entity inside project space via public link
    When user "Alice" creates a public link share inside of space "share sub-item" with settings:
      | path        | <resource>    |
      | shareType   | 3             |
      | permissions | <permissions> |
      | password    | <password>    |
      | name        | <link-name>   |
      | expireDate  | <expire-date> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the fields of the last response to user "Alice" and space "share sub-item" should include
      | item_type         | <resource-type> |
      | mimetype          | <mime-type>     |
      | file_target       | <file-target>   |
      | path              | /<resource>     |
      | permissions       | <permissions>   |
      | share_type        | public_link     |
      | displayname_owner | %displayname%   |
      | name              | <link-name>     |
      | uid_file_owner    | %space_id%      |
      | space_id          | %space_id%      |
    Examples:
      | resource        | file-target | permissions | password | link-name | expire-date              | resource-type | mime-type            |
      | folder          | /folder     | 0           |          | link      |                          | folder        | httpd/unix-directory |
      | folder          | /folder     | 1           | %public% | link      | 2042-03-25T23:59:59+0100 | folder        | httpd/unix-directory |
      | folder          | /folder     | 4           | %public% |           |                          | folder        | httpd/unix-directory |
      | folder          | /folder     | 5           | %public% |           | 2042-03-25T23:59:59+0100 | folder        | httpd/unix-directory |
      | folder          | /folder     | 15          | %public% | link      |                          | folder        | httpd/unix-directory |
      | folder/file.txt | /file.txt   | 0           |          | link      | 2042-03-25T23:59:59+0100 | file          | text/plain           |
      | folder/file.txt | /file.txt   | 1           | %public% | link      | 2042-03-25T23:59:59+0100 | file          | text/plain           |
      | folder/file.txt | /file.txt   | 3           | %public% | link      | 2042-03-25T23:59:59+0100 | file          | text/plain           |

  @owncloud-issue-5139
  Scenario Outline: user participant of the project space with space manager role can share an entity inside project space via public link
    Given user "Alice" has sent the following space share invitation:
      | space           | share sub-item |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | Manager        |
    When user "Brian" creates a public link share inside of space "share sub-item" with settings:
      | path        | <resource>               |
      | shareType   | 3                        |
      | permissions | 1                        |
      | password    | %public%                 |
      | name        | public link              |
      | expireDate  | 2042-03-25T23:59:59+0100 |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the fields of the last response to user "Brian" and space "share sub-item" should include
      | item_type         | <resource-type> |
      | mimetype          | <mime-type>     |
      | file_target       | <file-target>   |
      | path              | /<resource>     |
      | share_type        | public_link     |
      | displayname_owner | %displayname%   |
      | name              | public link     |
      | uid_file_owner    | %space_id%      |
      | space_id          | %space_id%      |
    Examples:
      | resource        | file-target | resource-type | mime-type            |
      | folder          | /folder     | folder        | httpd/unix-directory |
      | folder/file.txt | /file.txt   | file          | text/plain           |


  Scenario Outline: user participant of the project space without space manager role cannot share an entity inside project space via public link
    Given user "Alice" has sent the following space share invitation:
      | space           | share sub-item |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | <space-role>   |
    When user "Brian" creates a public link share inside of space "share sub-item" with settings:
      | path        | <resource>               |
      | shareType   | 3                        |
      | permissions | 1                        |
      | password    | %public%                 |
      | name        | public link              |
      | expireDate  | 2042-03-25T23:59:59+0100 |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the OCS status message should be "No share permission"
    Examples:
      | resource        | space-role   |
      | folder          | Space Editor |
      | folder          | Space Viewer |
      | folder/file.txt | Space Editor |
      | folder/file.txt | Space Viewer |

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: user creates a new public link share of a file inside the personal space with edit permissions
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "Random data" to "/file.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | file.txt    |
      | permissions | read,update |
      | password    | %public%    |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | item_type              | file          |
      | mimetype               | text/plain    |
      | file_target            | /file.txt     |
      | path                   | /file.txt     |
      | permissions            | read,update   |
      | share_type             | public_link   |
      | displayname_file_owner | %displayname% |
      | displayname_owner      | %displayname% |
      | uid_file_owner         | %username%    |
      | uid_owner              | %username%    |
    And the public should be able to download the last publicly shared file using the public WebDAV API with password "%public%" and the content should be "Random data"
    And the public upload to the last publicly shared file using the public WebDAV API with password "%public%" should pass with HTTP status code "204"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-5139
  Scenario Outline: user participant of the project space can see the created public resources link
    Given user "Alice" has sent the following space share invitation:
      | space           | share sub-item |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | <space-role>   |
    When user "Alice" creates a public link share inside of space "share sub-item" with settings:
      | path        | folder/file.txt |
      | shareType   | 3               |
      | permissions | 1               |
      | password    | %public%        |
    Then the fields of the last response to user "Alice" and space "share sub-item" should include
      | item_type              | file             |
      | mimetype               | text/plain       |
      | file_target            | /file.txt        |
      | path                   | /folder/file.txt |
      | permissions            | 1                |
      | share_type             | public_link      |
      | displayname_file_owner |                  |
      | displayname_owner      | %displayname%    |
      | uid_owner              | %username%       |
      | uid_file_owner         | %space_id%       |
      | space_id               | %space_id%       |
    And for user "Brian" the space "share sub-item" should contain the last created public link of the file "folder/file.txt"
    Examples:
      | space-role   |
      | Space Editor |
      | Space Viewer |
      | Manager      |
