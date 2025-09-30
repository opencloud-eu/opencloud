@skipOnReva @issue-1276 @issue-1269
Feature: changing a public link share
  As a user
  I want to set the permissions of a public link share
  So that people who have the public link only have the designated authorization

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"

  @issue-10331
  Scenario Outline: public can or cannot delete file through publicly shared link depending on having delete permissions using the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT             |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When the public deletes file "parent.txt" from the last link share with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And as "Alice" file "PARENT/parent.txt" <should-or-not> exist
    Examples:
      | permissions-role | http-status-code | should-or-not |
      | view             | 403              | should        |
      | upload           | 403              | should        |
      | createOnly       | 403              | should        |
      | edit             | 204              | should not    |

  @issue-10331
  Scenario: public link share permissions work correctly for renaming and share permissions edit using the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public renames file "parent.txt" to "newparent.txt" from the last public link share using the password "%public%" and the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/parent.txt" should not exist
    And as "Alice" file "/PARENT/newparent.txt" should exist

  @issue-10331
  Scenario: public link share permissions work correctly for upload with share permissions edit with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public uploads file "lorem.txt" with password "%public%" and content "test" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "PARENT/lorem.txt" for user "Alice" should be "test"


  Scenario: public cannot delete file through publicly shared link with password using an invalid password with public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public deletes file "parent.txt" from the last link share with password "invalid" using the public WebDAV API
    Then the HTTP status code should be "401"
    And as "Alice" file "PARENT/parent.txt" should exist

  @issue-10331
  Scenario: public can delete file through publicly shared link with password using the valid password with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public deletes file "parent.txt" from the last link share with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "PARENT/parent.txt" should not exist


  Scenario: public tries to rename a file in a password protected share using an invalid password with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public renames file "parent.txt" to "newparent.txt" from the last public link share using the password "invalid" and the public WebDAV API
    Then the HTTP status code should be "401"
    And as "Alice" file "/PARENT/newparent.txt" should not exist
    And as "Alice" file "/PARENT/parent.txt" should exist

  @issue-10331
  Scenario: public tries to rename a file in a password protected share using the valid password with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public renames file "parent.txt" to "newparent.txt" from the last public link share using the password "%public%" and the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/newparent.txt" should exist
    And as "Alice" file "/PARENT/parent.txt" should not exist


  Scenario: public tries to upload to a password protected public share using an invalid password with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public uploads file "lorem.txt" with password "invalid" and content "test" using the public WebDAV API
    Then the HTTP status code should be "401"
    And as "Alice" file "/PARENT/lorem.txt" should not exist

  @issue-10331
  Scenario: public tries to upload to a password protected public share using the valid password with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public uploads file "lorem.txt" with password "%public%" and content "test" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/lorem.txt" should exist

  @issue-10331
  Scenario: public cannot rename a file in upload-write-only public link share with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public renames file "parent.txt" to "newparent.txt" from the last public link share using the password "%public%" and the public WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/PARENT/parent.txt" should exist
    And as "Alice" file "/PARENT/newparent.txt" should not exist

  @issue-10331
  Scenario: public cannot delete a file in upload-write-only public link share with the public WebDAV API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public deletes file "parent.txt" from the last link share with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "PARENT/parent.txt" should exist


  Scenario Outline: normal user tries to remove password of a public link share (change/create permission)
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT             |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" updates the last public link share using the sharing API with
      | path        | /PARENT       |
      | permissions | <permissions> |
      | password    |               |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "400"
    And the OCS status message should be "missing required password"
    Examples:
      | ocs-api-version | permissions | permissions-role | http-status-code |
      | 1               | change      | edit             | 200              |
      | 2               | change      | edit             | 400              |
      | 1               | create      | createOnly       | 200              |
      | 2               | create      | createOnly       | 400              |

  @issue-7821
  Scenario Outline: normal user tries to remove password of a public link (update without sending permissions)
    Given using OCS API version "<ocs-api-version>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When user "Alice" updates the last public link share using the sharing API with
      | path     | /PARENT |
      | password |         |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "104"
    And the OCS status message should be "user is not allowed to delete the password from the public link"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 403              |

  @issue-9724 @issue-10331
  Scenario Outline: administrator removes password of a read-only public link
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And using OCS API version "<ocs-api-version>"
    And admin has created folder "/PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When user "Alice" updates the last public link share using the sharing API with
      | path        | /PARENT |
      | permissions | read    |
      | password    |         |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the public should be able to download file "/parent.txt" from inside the last public link shared folder using the public WebDAV API with password ""
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: administrator tries to remove password of a public link share (change/create permission)
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And using OCS API version "<ocs-api-version>"
    And admin has created folder "/PARENT"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT             |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" updates the last public link share using the sharing API with
      | path        | /PARENT       |
      | permissions | <permissions> |
      | password    |               |
    Then the HTTP status code should be "<http-status-code>"
    And the OCS status code should be "400"
    And the OCS status message should be "missing required password"
    Examples:
      | ocs-api-version | permissions | permissions-role | http-status-code |
      | 1               | change      | edit             | 200              |
      | 2               | change      | edit             | 400              |
      | 1               | create      | createOnly       | 200              |
      | 2               | create      | createOnly       | 400              |

  @issue-web-10473
  Scenario: user tries to download public link file using own basic auth
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "FOLDER/textfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When user "Alice" tries to download file "textfile.txt" from the last public link using own basic auth and public WebDAV API
    Then the HTTP status code should be "401"
