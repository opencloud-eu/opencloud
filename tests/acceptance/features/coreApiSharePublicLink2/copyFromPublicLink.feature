@env-config
Feature: copying from public link share
  As a user
  I want to make a copy of a resource within a public link
  So that I can have a backup

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/PARENT"
    And the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"

  @owncloud-issue-10331
  Scenario: copy file within a public link folder
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/copy1.txt" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/testfile.txt" should exist
    And as "Alice" file "/PARENT/copy1.txt" should exist
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"
    And the content of file "/PARENT/copy1.txt" for user "Alice" should be "some data"

  @owncloud-issue-10331
  Scenario: copy folder within a public link folder
    Given user "Alice" has created folder "/PARENT/testFolder"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/testFolder/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies folder "/testFolder" to "/testFolder-copy" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "/PARENT/testFolder" should exist
    And as "Alice" folder "/PARENT/testFolder-copy" should exist
    And the content of file "/PARENT/testFolder/testfile.txt" for user "Alice" should be "some data"
    And the content of file "/PARENT/testFolder-copy/testfile.txt" for user "Alice" should be "some data"

  @owncloud-issue-10331
  Scenario: copy file within a public link folder to a new folder
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And user "Alice" has created folder "/PARENT/testFolder"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/testFolder/copy1.txt" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/testfile.txt" should exist
    And as "Alice" file "/PARENT/testFolder/copy1.txt" should exist
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"
    And the content of file "/PARENT/testFolder/copy1.txt" for user "Alice" should be "some data"

  @owncloud-issue-10331
  Scenario: copy file within a public link folder to existing file
    Given user "Alice" has uploaded file with content "some data 0" to "/PARENT/testfile.txt"
    And user "Alice" has uploaded file with content "some data 1" to "/PARENT/copy1.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/copy1.txt" using the public WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/PARENT/testfile.txt" should exist
    And as "Alice" file "/PARENT/copy1.txt" should exist
    And the content of file "/PARENT/copy1.txt" for user "Alice" should be "some data 0"

  @owncloud-issue-1232 @owncloud-issue-10331
  Scenario: copy folder within a public link folder to existing file
    Given user "Alice" has created folder "/PARENT/testFolder"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/testFolder/testfile.txt"
    And user "Alice" has uploaded file with content "some data 1" to "/PARENT/copy1.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies folder "/testFolder" to "/copy1.txt" using the public WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/PARENT/testFolder" should exist
    And the content of file "/PARENT/copy1.txt/testfile.txt" for user "Alice" should be "some data"
    But as "Alice" file "/PARENT/copy1.txt" should not exist
    And as "Alice" file "/copy1.txt" should exist in the trashbin

  @owncloud-issue-10331
  Scenario: copy file within a public link folder and delete file
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/copy1.txt" using the public WebDAV API
    And user "Alice" deletes file "/PARENT/copy1.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/PARENT/copy1.txt" should not exist

  @owncloud-issue-1232 @owncloud-issue-10331
  Scenario: copy file within a public link folder to existing folder
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And user "Alice" has created folder "/PARENT/new-folder"
    And user "Alice" has uploaded file with content "some data 1" to "/PARENT/new-folder/testfile1.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/new-folder" using the public WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"
    And the content of file "/PARENT/new-folder" for user "Alice" should be "some data"
    And as "Alice" folder "/PARENT/new-folder" should not exist
    And as "Alice" folder "new-folder" should exist in the trashbin

  @owncloud-issue-10331
  Scenario Outline: copy file with special characters in it's name within a public link folder
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/<file-name>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/<file-name>" to "/copy1.txt" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/<file-name>" should exist
    And as "Alice" file "/PARENT/copy1.txt" should exist
    And the content of file "/PARENT/<file-name>" for user "Alice" should be "some data"
    And the content of file "/PARENT/copy1.txt" for user "Alice" should be "some data"
    Examples:
      | file-name        |
      | नेपाली.txt       |
      | strängé file.txt |
      | C++ file.cpp     |
      | sample,1.txt     |

  @owncloud-issue-10331
  Scenario Outline: copy file within a public link folder to a file with special characters in it's name
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/<destination-file-name>" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/testfile.txt" should exist
    And as "Alice" file "/PARENT/<destination-file-name>" should exist
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"
    And the content of file "/PARENT/<destination-file-name>" for user "Alice" should be "some data"
    Examples:
      | destination-file-name |
      | नेपाली.txt            |
      | strängé file.txt      |
      | C++ file.cpp          |
      | sample,1.txt          |

  @owncloud-issue-10331
  Scenario Outline: copy file within a public link folder into a folder with special characters
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And user "Alice" has created folder "/PARENT/<destination-folder-name>"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/<destination-folder-name>/copy1.txt" using the public WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/PARENT/testfile.txt" should exist
    And as "Alice" file "/PARENT/<destination-folder-name>/copy1.txt" should exist
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"
    And the content of file "/PARENT/<destination-folder-name>/copy1.txt" for user "Alice" should be "some data"
    Examples:
      | destination-folder-name |
      | नेपाली.txt              |
      | strängé file.txt        |
      | C++ file.cpp            |
      | sample,1.txt            |

  @owncloud-issue-8711 @owncloud-issue-10331
  Scenario: copy file within a public link folder to a same file
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/testfile.txt" using the public WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"

  @owncloud-issue-8711 @owncloud-issue-10331
  Scenario: copy folder within a public link folder to a same folder
    Given user "Alice" has created folder "/PARENT/testFolder"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/testFolder/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies folder "/testFolder" to "/testFolder" using the public WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/PARENT/testFolder" should exist
    And the content of file "/PARENT/testFolder/testfile.txt" for user "Alice" should be "some data"

  @owncloud-issue-1230 @owncloud-issue-10331
  Scenario: copy file within a public link folder to a share item root
    Given user "Alice" has uploaded file with content "some data" to "/PARENT/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies file "/testfile.txt" to "/" using the public WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" file "/PARENT/testfile.txt" should exist
    And the content of file "/PARENT/testfile.txt" for user "Alice" should be "some data"

  @owncloud-issue-1230 @owncloud-issue-10331
  Scenario: copy folder within a public link folder to a share item root
    Given user "Alice" has created folder "/PARENT/testFolder"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/testFolder/testfile.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public copies folder "/testFolder" to "/" using the public WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" folder "/PARENT/testFolder" should exist
    And the content of file "/PARENT/testFolder/testfile.txt" for user "Alice" should be "some data"
