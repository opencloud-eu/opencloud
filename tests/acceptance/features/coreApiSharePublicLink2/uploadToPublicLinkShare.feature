@issue-1276 @skipOnReva

Feature: upload to a public link share
  As a user
  I want to create a public link with upload permission
  So that the recipient can upload resources

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "FOLDER"

  @issue-10331
  Scenario Outline: uploading file to a public upload-only share using public API that was deleted does not work
    Given using <dav-path-version> DAV path
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    And user "Alice" has deleted folder "/FOLDER"
    When the public uploads file "test.txt" with password "%public%" and content "test-file" using the public WebDAV API
    And the HTTP status code should be "404"

    @issue-1268
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1269 @issue-10331
  Scenario: uploading file to a public read-only share folder with public API does not work
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When the public uploads file "test.txt" with password "%public%" and content "test-file" using the public WebDAV API
    And the HTTP status code should be "403"

  @issue-10331
  Scenario: uploading to a public upload-only share with public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public uploads file "test.txt" with password "%public%" and content "test-file" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/test.txt" for user "Alice" should be "test-file"
    And the following headers should match these regular expressions
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |

  @issue-10331
  Scenario: uploading to a public upload-only share with password with public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public uploads file "test.txt" with password "%public%" and content "test-file" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/test.txt" for user "Alice" should be "test-file"

  @issue-10331
  Scenario: uploading to a public read/write share with password with public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public uploads file "test.txt" with password "%public%" and content "test-file" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/test.txt" for user "Alice" should be "test-file"

  @skipOnReva @issue-10331
  Scenario: uploading file to a public shared folder with read/write permission when the sharer has insufficient quota does not work with public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "1"
    When the public uploads file "test.txt" with password "%public%" and content "test2" using the public WebDAV API
    Then the HTTP status code should be "507"

  @skipOnReva @issue-10331
  Scenario: uploading file to a public shared folder with upload-only permission when the sharer has insufficient quota does not work with public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "1"
    When the public uploads file "test.txt" with password "%public%" and content "test2" using the public WebDAV API
    Then the HTTP status code should be "507"

  @smokeTest @issue-10331
  Scenario: uploading to a public upload-write and no edit and no overwrite share with public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public uploads file "test.txt" with password "%public%" and content "test2" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/test.txt" for user "Alice" should be "test2"

  @smokeTest @issue-1267 @issue-10331
  Scenario: uploading same file to a public upload-write and no edit and no overwrite share multiple times with new public API
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public uploads file "test.txt" with password "%public%" and content "test" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the following headers should match these regular expressions
      | ETag | /^"[a-f0-9:\.]{1,32}"$/ |
    When the public uploads file "test.txt" with password "%public%" and content "test2" using the public WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/test.txt" for user "Alice" should be "test"
    And the content of file "/FOLDER/test (2).txt" for user "Alice" should be "test2"
