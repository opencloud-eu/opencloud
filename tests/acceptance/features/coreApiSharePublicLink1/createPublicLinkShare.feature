@owncloud-issue-1269 @owncloud-issue-1293

Feature: create a public link share
  As a user
  I want to create public links
  So that I can share resources to people who aren't OpenCloud users

  Background:
    Given user "Alice" has been created with default attributes

  @smokeTest @skipOnReva
  Scenario Outline: creating public link share of a file or a folder using the default permissions without password using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "Random data" to "/randomfile.txt"
    And user "Alice" has created folder "/PARENT"
    When user "Alice" creates a public link share using the sharing API with settings
      | path | randomfile.txt |
    Then the OCS status code should be "400"
    And the HTTP status code should be "<http-status-code>"
    When user "Alice" creates a public link share using the sharing API with settings
      | path | PARENT |
    Then the OCS status code should be "400"
    And the HTTP status code should be "<http-status-code>"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 400              |

  @smokeTest @owncloud-issue-10331 @owncloud-issue-9724
  Scenario Outline: creating a new public link share of a file with password using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "Random data" to "/randomfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path     | randomfile.txt |
      | password | %public%       |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | item_type              | file            |
      | mimetype               | text/plain      |
      | file_target            | /randomfile.txt |
      | path                   | /randomfile.txt |
      | permissions            | read            |
      | share_type             | public_link     |
      | displayname_file_owner | %displayname%   |
      | displayname_owner      | %displayname%   |
      | uid_file_owner         | %username%      |
      | uid_owner              | %username%      |
      | name                   |                 |
    And the public should be able to download the last publicly shared file using the public WebDAV API with password "%public%" and the content should be "Random data"
    When the public tries to download the last public link shared file with password "%regular%" using the public WebDAV API
    Then the HTTP status code should be "401"
    And the value of the item "//s:message" in the response should match "/Username or password was incorrect/"
    When the public tries to download the last public link shared file using the public WebDAV API
    Then the HTTP status code should be "401"
    And the value of the item "//s:message" in the response should match "/No 'Authorization: Basic' header found/"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-10331 @owncloud-issue-9724
  Scenario Outline: create a new public link share of a file with edit permissions
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "Random data" to "/randomfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | randomfile.txt            |
      | permissions | read,update,create,delete |
      | password    | %public%                  |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | item_type              | file            |
      | mimetype               | text/plain      |
      | file_target            | /randomfile.txt |
      | path                   | /randomfile.txt |
      | permissions            | read,update     |
      | share_type             | public_link     |
      | displayname_file_owner | %displayname%   |
      | displayname_owner      | %displayname%   |
      | uid_file_owner         | %username%      |
      | uid_owner              | %username%      |
      | name                   |                 |
    And the public should be able to download the last publicly shared file using the public WebDAV API with password "%public%" and the content should be "Random data"
    And uploading content to a public link shared file with password "%public%" should work using the public WebDAV API
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-10331 @owncloud-issue-9724
  Scenario Outline: creating a new public link share of a folder, with a password and accessing using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has uploaded file with content "Random data" to "/PARENT/randomfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | PARENT   |
      | password    | %public% |
      | permissions | change   |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | item_type              | folder               |
      | mimetype               | httpd/unix-directory |
      | file_target            | /PARENT              |
      | path                   | /PARENT              |
      | permissions            | change               |
      | share_type             | public_link          |
      | displayname_file_owner | %displayname%        |
      | displayname_owner      | %displayname%        |
      | uid_file_owner         | %username%           |
      | uid_owner              | %username%           |
      | name                   |                      |
    And the public should be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API with password "%public%" and the content should be "Random data"
    And the public should be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API with password "%public%" and the content should be "Random data"
    But the public should not be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API without a password
    And the public should not be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API with password "%regular%"
    And the public should not be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API without a password
    And the public should not be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API with password "%regular%"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @smokeTest
  Scenario Outline: getting the share information of public link share from the OCS API does not expose sensitive information
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "Random data" to "/randomfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path     | randomfile.txt |
      | password | %public%       |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | file_target            | /randomfile.txt |
      | path                   | /randomfile.txt |
      | item_type              | file            |
      | share_type             | public_link     |
      | permissions            | read            |
      | uid_owner              | Alice           |
      | share_with             | ***redacted***  |
      | share_with_displayname | ***redacted***  |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @env-config
  Scenario Outline: getting the share information of password less public-links hides credential placeholders
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "Random data" to "/randomfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path | randomfile.txt |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | file_target | /randomfile.txt |
      | path        | /randomfile.txt |
      | item_type   | file            |
      | share_type  | public_link     |
      | permissions | read            |
      | uid_owner   | %username%      |
    And the fields of the last response should not include
      | share_with             | ANY_VALUE |
      | share_with_displayname | ANY_VALUE |
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-10331
  Scenario Outline: creating a link share with no specified permissions defaults to read permissions when public upload is disabled globally and accessing using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "/afolder"
    When user "Alice" creates a public link share using the sharing API with settings
      | path     | /afolder |
      | password | %public% |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | id          | A_STRING    |
      | share_type  | public_link |
      | permissions | read        |
    And the public upload to the last publicly shared folder using the public WebDAV API with password "%public%" should fail with HTTP status code "403"
    And the public upload to the last publicly shared folder using the public WebDAV API with password "%public%" should fail with HTTP status code "403"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-10331 @owncloud-issue-9724
  Scenario Outline: creating a link share with edit permissions keeps it using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "/afolder"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /afolder                  |
      | permissions | read,update,create,delete |
      | password    | %public%                  |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | id          | A_STRING                  |
      | share_type  | public_link               |
      | permissions | read,update,create,delete |
    And uploading a file with password "%public%" should work using the public WebDAV API
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-10331 @owncloud-issue-9724
  Scenario Outline: creating a link share with upload permissions keeps it using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "/afolder"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /afolder    |
      | permissions | read,create |
      | password    | %public%    |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | id          | A_STRING    |
      | share_type  | public_link |
      | permissions | read,create |
    And uploading a file with password "%public%" should work using the public WebDAV API
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: Do not allow public sharing of the root on OpenCloud when the default permission is read and access using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" creates a public link share using the sharing API with settings
      | path     | /        |
      | password | %public% |
    Then the OCS status code should be "400"
    And the HTTP status code should be "<http-status-code>"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 400              |

  @owncloud-issue-10331 @owncloud-issue-9724
  Scenario Outline: user creates a public link share of a file with file name longer than 64 chars using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "long file" to "/aquickbrownfoxjumpsoveraverylazydogaquickbrownfoxjumpsoveralazydog.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path     | /aquickbrownfoxjumpsoveraverylazydogaquickbrownfoxjumpsoveralazydog.txt |
      | password | %public%                                                                |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the public should be able to download the last publicly shared file using the public WebDAV API with password "%public%" and the content should be "long file"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario Outline: user creates a public link share of a folder with folder name longer than 64 chars and access using the public WebDAV API
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has created folder "/aquickbrownfoxjumpsoveraverylazydogaquickbrownfoxjumpsoveralazydog"
    And user "Alice" has uploaded file with content "Random data" to "/aquickbrownfoxjumpsoveraverylazydogaquickbrownfoxjumpsoveralazydog/randomfile.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path     | /aquickbrownfoxjumpsoveraverylazydogaquickbrownfoxjumpsoveralazydog |
      | password | %public%                                                            |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And the public should be able to download file "/randomfile.txt" from inside the last public link shared folder using the public WebDAV API with password "%public%" and the content should be "Random data"
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |

  @owncloud-issue-1293 @skipOnReva @owncloud-issue-10331 @owncloud-issue-9724
  Scenario: delete a folder that has been publicly shared and try to access using the public WebDAV API
    Given user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file with content "Random data" to "/PARENT/parent.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When user "Alice" deletes folder "/PARENT" using the WebDAV API
    And the public tries to download file "/parent.txt" from inside the last public link shared folder with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "404"

  @owncloud-issue-1269 @owncloud-issue-1293 @skipOnReva @owncloud-issue-10331 @owncloud-issue-9724
  Scenario: try to download from a public share that has upload only permissions using the public webdav api
    Given user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file with content "Random data" to "/PARENT/parent.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT     |
      | space           | Personal   |
      | permissionsRole | createOnly |
      | password        | %public%   |
    When the public tries to download file "/parent.txt" from inside the last public link shared folder with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "403"

  @env-config @skipOnReva @owncloud-issue-10331 @owncloud-issue-10071
  Scenario: get the size of a file shared by public link
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And user "Alice" has uploaded file with content "This is a test file" to "test-file.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | test-file.txt |
      | space           | Personal      |
      | permissionsRole | view          |
    When the public gets the size of the last shared public link using the WebDAV API
    Then the HTTP status code should be "207"
    And the size of the file should be "19"

  @env-config @owncloud-issue-10331 @owncloud-issue-10071
  Scenario Outline: get the mtime of a file shared by public link
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | file.txt |
      | permissions | read     |
    Then the HTTP status code should be "200"
    And the mtime of file "file.txt" in the last shared public link using the WebDAV API should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @env-config @owncloud-issue-10331 @owncloud-issue-10071
  Scenario Outline: get the mtime of a file inside a folder shared by public link
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "testFolder"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "testFolder/file.txt" with mtime "Thu, 08 Aug 2019 04:18:13 GMT"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | /testFolder |
      | permissions | read        |
    Then the HTTP status code should be "200"
    And the mtime of file "file.txt" in the last shared public link using the WebDAV API should be "Thu, 08 Aug 2019 04:18:13 GMT"
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @env-config @skipOnReva @owncloud-issue-10331 @owncloud-issue-10071
  Scenario: get the mtime of a file inside a folder shared by public link using new webDAV version
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And user "Alice" has created folder "testFolder"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testFolder |
      | space           | Personal   |
      | permissionsRole | edit       |
    When the public uploads file "file.txt" to the last public link shared folder with password "%public%" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "testFolder/file.txt" should exist
    And as "Alice" the mtime of the file "testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    And the mtime of file "file.txt" in the last shared public link using the WebDAV API should be "Thu, 08 Aug 2019 04:18:13 GMT"

  @env-config @owncloud-issue-10331 @owncloud-issue-10071
  Scenario: overwriting a file changes its mtime (public webDAV API)
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And user "Alice" has created folder "testFolder"
    When user "Alice" uploads file with content "uploaded content for file name ending with a dot" to "testFolder/file.txt" using the WebDAV API
    And user "Alice" creates a public link share using the sharing API with settings
      | path        | /testFolder               |
      | permissions | read,update,create,delete |
    And the public uploads file "file.txt" to the last public link shared folder with password "%public%" with mtime "Thu, 08 Aug 2019 04:18:13 GMT" using the public WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/testFolder/file.txt" should exist
    And as "Alice" the mtime of the file "testFolder/file.txt" should be "Thu, 08 Aug 2019 04:18:13 GMT"
    And the mtime of file "file.txt" in the last shared public link using the WebDAV API should be "Thu, 08 Aug 2019 04:18:13 GMT"

  @env-config @skipOnReva @owncloud-issue-10331 @owncloud-issue-10071
  Scenario: check the href of a public link file
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using new DAV path
    And user "Alice" has uploaded file with content "Random data" to "/file.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | file.txt |
      | space           | Personal |
      | permissionsRole | view     |
    When the public lists the resources in the last created public link with depth "1" using the WebDAV API
    Then the HTTP status code should be "207"
    And the value of the item "//d:response[2]//d:href" in the response should match "/\/dav\/public-files\/%public_token%\/file.txt$/"
    When the public gets the following properties of entry "/file.txt" in the last created public link using the WebDAV API
      | propertyName |
      | d:href       |
    Then the HTTP status code should be "207"
    And the value of the item "//d:href" in the response should match "/\/dav\/public-files\/%public_token%\/file.txt$/"

  @owncloud-issue-6929 @@skipOnReva
  Scenario Outline: create a password-protected public link on a file with the name same to the previously deleted one
    Given using OCS API version "<ocs-api-version>"
    And user "Alice" has uploaded file with content "test data 1" to "/test.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | test.txt |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    And user "Alice" has deleted file "test.txt"
    When user "Alice" updates the last public link share using the sharing API with
      | password | Test:123345 |
    Then the OCS status code should be "998"
    And the HTTP status code should be "<http-status-code>"
    And the OCS status message should be "update public share: resource not found"
    Examples:
      | ocs-api-version | http-status-code |
      | 1               | 200              |
      | 2               | 404              |
