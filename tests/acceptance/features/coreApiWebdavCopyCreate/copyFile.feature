Feature: copy file
  As a user
  I want to be able to copy files
  So that I can manage my files

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "OpenCloud test text file 1" to "/textfile1.txt"
    And user "Alice" has created folder "/FOLDER"

  @smokeTest
  Scenario Outline: copying a file
    Given using <dav-path-version> DAV path
    When user "Alice" copies file "/textfile0.txt" to "/FOLDER/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/FOLDER/textfile0.txt" for user "Alice" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest
  Scenario Outline: copying and overwriting a file
    Given using <dav-path-version> DAV path
    When user "Alice" copies file "/textfile0.txt" to "/textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/textfile1.txt" for user "Alice" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copying a file when 2 files exist with different case
    Given using <dav-path-version> DAV path
    # "/textfile1.txt" already exists in the skeleton, make another with only case differences in the file name
    When user "Alice" copies file "/textfile0.txt" to "/TextFile1.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/textfile1.txt" for user "Alice" should be "OpenCloud test text file 1"
    And the content of file "/TextFile1.txt" for user "Alice" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @env-config
  Scenario Outline: copying a file to a folder with no permissions
    Given using <dav-path-version> DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/textfile0.txt" to "/Shares/testshare/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And user "Alice" should not be able to download file "/Shares/testshare/textfile0.txt"
    Examples:
      | dav-path-version | permissions-role |
      | old              | Viewer           |
      | new              | Viewer           |
      | spaces           | Viewer           |
      | old              | Secure Viewer    |
      | new              | Secure Viewer    |
      | spaces           | Secure Viewer    |

  @skipOnReva
  Scenario Outline: copying a file to overwrite a file into a folder with no permissions
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "OpenCloud test text file 1" to "textfile1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    And user "Brian" has copied file "textfile1.txt" to "/testshare/overwritethis.txt"
    When user "Alice" copies file "/textfile0.txt" to "/Shares/testshare/overwritethis.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And the content of file "/Shares/testshare/overwritethis.txt" for user "Alice" should be "OpenCloud test text file 1"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1345 @issue-2177
  Scenario Outline: copying file to a path with extension .part should not be possible
    Given using <dav-path-version> DAV path
    When user "Alice" copies file "/textfile1.txt" to "/textfile1.part" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should see the following elements
      | /textfile1.part |
      | /textfile1.txt  |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239
  Scenario Outline: copy a file over the top of an existing folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER/sample-folder"
    When user "Alice" copies file "/textfile1.txt" to "/FOLDER" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/FOLDER" for user "Alice" should be "OpenCloud test text file 1"
    And as "Alice" folder "/FOLDER/sample-folder" should not exist
    And as "Alice" file "/textfile1.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239
  Scenario Outline: copy a folder over the top of an existing file
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER/sample-folder"
    When user "Alice" copies folder "/FOLDER" to "/textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/FOLDER/sample-folder" should exist
    And as "Alice" folder "/textfile1.txt/sample-folder" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239
  Scenario Outline: copy a folder into another folder at different level
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER/second-level-folder"
    And user "Alice" has created folder "FOLDER/second-level-folder/third-level-folder"
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b/sample-folder-c"
    When user "Alice" copies folder "Sample-Folder-A/sample-folder-b" to "FOLDER/second-level-folder/third-level-folder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/Sample-Folder-A/sample-folder-b/sample-folder-c" should exist
    And as "Alice" folder "/FOLDER/second-level-folder/third-level-folder/sample-folder-c" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239
  Scenario Outline: copy a file into a folder at different level
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER/second-level-folder"
    And user "Alice" has created folder "FOLDER/second-level-folder/third-level-folder"
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "Sample-Folder-A/sample-folder-b/textfile-c.txt" to "FOLDER/second-level-folder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "FOLDER/second-level-folder/third-level-folder" should not exist
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" file "FOLDER/second-level-folder" should exist
    And the content of file "FOLDER/second-level-folder" for user "Alice" should be "sample file-c"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239
  Scenario Outline: copy a file into a file at different level
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "file at second level" to "FOLDER/second-level-file.txt"
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "Sample-Folder-A/sample-folder-b/textfile-c.txt" to "FOLDER/second-level-file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" file "FOLDER/second-level-file.txt" should exist
    And as "Alice" file "FOLDER/textfile-c.txt" should not exist
    And the content of file "FOLDER/second-level-file.txt" for user "Alice" should be "sample file-c"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239
  Scenario Outline: copy a folder into a file at different level
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER/second-level-folder"
    And user "Alice" has created folder "FOLDER/second-level-folder/third-level-folder"
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies folder "FOLDER/second-level-folder" to "Sample-Folder-A/sample-folder-b/textfile-c.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" folder "FOLDER/second-level-folder/third-level-folder" should exist
    And as "Alice" folder "Sample-Folder-A/sample-folder-b/textfile-c.txt/third-level-folder" should exist
    And as "Alice" folder "Sample-Folder-A/sample-folder-b/second-level-folder" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @issue-3874 @issue-9753 @skipOnReva
  Scenario Outline: copy a file over the top of an existing folder received as a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/BRIAN-Folder"
    And user "Brian" has created folder "BRIAN-Folder/sample-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-Folder |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-Folder" synced
    When user "Alice" copies file "/textfile1.txt" to "/Shares/BRIAN-Folder" using the WebDAV API
    Then the HTTP status code should be "400"
    And as "Alice" folder "/Shares/BRIAN-Folder/sample-folder" should exist
    And as "Alice" file "/Shares/BRIAN-Folder" should not exist
    And user "Alice" should have a share "BRIAN-Folder" shared by user "Brian" from space "Personal"
    And as "Brian" folder "BRIAN-Folder" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @issue-9753
  Scenario Outline: copy a file over the top of an existing file received as a share
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "file to copy" to "copy.txt"
    And user "Brian" has been created with default attributes
    And user "Brian" has uploaded file with content "file to share" to "lorem.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | lorem.txt   |
      | space           | Personal    |
      | sharee          | Alice       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Alice" has a share "lorem.txt" synced
    When user "Alice" copies file "copy.txt" to "Shares/lorem.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "Shares/lorem.txt" for user "Alice" should be "file to copy"
    And user "Alice" should have a share "lorem.txt" shared by user "Brian" from space "Personal"
    And the content of file "lorem.txt" for user "Brian" should be "file to copy"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @issue-6999 @issue-9753 @skipOnReva
  Scenario Outline: copy a folder over the top of an existing file received as a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has uploaded file with content "file to share" to "/sharedfile1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | sharedfile1.txt |
      | space           | Personal        |
      | sharee          | Alice           |
      | shareType       | user            |
      | permissionsRole | File Editor     |
    And user "Alice" has a share "sharedfile1.txt" synced
    And user "Alice" has created folder "FOLDER/sample-folder"
    When user "Alice" copies folder "/FOLDER" to "/Shares/sharedfile1.txt" using the WebDAV API
    Then the HTTP status code should be "400"
    And the content of file "Shares/sharedfile1.txt" for user "Alice" should be "file to share"
    And as "Alice" folder "/Shares/sharedfile1.txt" should not exist
    And user "Alice" should have a share "sharedfile1.txt" shared by user "Brian" from space "Personal"
    And the content of file "sharedfile1.txt" for user "Brian" should be "file to share"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-6999 @issue-9753 @skipOnReva
  Scenario Outline: copy a folder over the top of an existing folder received as a share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "BRIAN-Folder"
    And user "Brian" has created folder "BRIAN-Folder/brian-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-Folder |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-Folder" synced
    And user "Alice" has created folder "FOLDER/alice-folder"
    When user "Alice" copies folder "FOLDER" to "Shares/BRIAN-Folder" using the WebDAV API
    Then the HTTP status code should be "400"
    And as "Alice" folder "Shares/BRIAN-Folder/brian-folder" should exist
    And as "Alice" folder "Shares/BRIAN-Folder/alice-folder" should not exist
    And user "Alice" should have a share "BRIAN-Folder" shared by user "Brian" from space "Personal"
    And as "Brian" folder "BRIAN-Folder" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a folder into another folder at different level which is received as a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder/third-level-folder"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b/sample-folder-c"
    When user "Alice" copies folder "Sample-Folder-A/sample-folder-b" to "Shares/BRIAN-FOLDER/second-level-folder/third-level-folder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/Sample-Folder-A/sample-folder-b/sample-folder-c" should exist
    And as "Alice" folder "/Shares/BRIAN-FOLDER/second-level-folder/third-level-folder/sample-folder-c" should exist
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a file into a folder at different level received as a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder/third-level-folder"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "Sample-Folder-A/sample-folder-b/textfile-c.txt" to "Shares/BRIAN-FOLDER/second-level-folder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/third-level-folder" should not exist
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" file "Shares/BRIAN-FOLDER/second-level-folder" should exist
    And the content of file "Shares/BRIAN-FOLDER/second-level-folder" for user "Alice" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a file into a file at different level received as a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has uploaded file with content "file at second level" to "BRIAN-FOLDER/second-level-file.txt"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "Sample-Folder-A/sample-folder-b/textfile-c.txt" to "Shares/BRIAN-FOLDER/second-level-file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" file "Shares/BRIAN-FOLDER/second-level-file.txt" should exist
    And as "Alice" file "Shares/BRIAN-FOLDER/textfile-c.txt" should not exist
    And the content of file "Shares/BRIAN-FOLDER/second-level-file.txt" for user "Alice" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a folder into a file at different level received as a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder"
    And user "Brian" has uploaded file with content "file at third level" to "BRIAN-FOLDER/second-level-folder/third-level-file.txt"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "FOLDER/second-level-folder"
    And user "Alice" has created folder "FOLDER/second-level-folder/third-level-folder"
    When user "Alice" copies folder "FOLDER/second-level-folder" to "/Shares/BRIAN-FOLDER/second-level-folder/third-level-file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/third-level-file.txt" should exist
    And as "Alice" folder "FOLDER/second-level-folder/third-level-folder" should exist
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/third-level-file.txt/third-level-folder" should exist
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/second-level-folder" should not exist
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @issue-9753 @skipOnReva
  Scenario Outline: copy a file over the top of an existing folder received as a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "/BRIAN-Folder"
    And user "Brian" has created folder "BRIAN-Folder/sample-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-Folder |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-Folder" synced
    When user "Alice" copies file "/textfile1.txt" to "/Shares/BRIAN-Folder" using the WebDAV API
    Then the HTTP status code should be "400"
    And as "Alice" folder "/Shares/BRIAN-Folder/sample-folder" should exist
    And as "Alice" file "/Shares/BRIAN-Folder" should not exist
    And user "Alice" should have a share "BRIAN-Folder" shared by user "Brian" from space "Personal"
    And as "Brian" folder "BRIAN-Folder/sample-folder" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @issue-6999 @issue-9753 @skipOnReva
  Scenario Outline: copy a folder over the top of an existing file received as a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has uploaded file with content "file to share" to "/sharedfile1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | sharedfile1.txt |
      | space           | Personal        |
      | sharee          | grp1            |
      | shareType       | group           |
      | permissionsRole | File Editor     |
    And user "Alice" has a share "sharedfile1.txt" synced
    And user "Alice" has created folder "FOLDER/sample-folder"
    When user "Alice" copies folder "/FOLDER" to "/Shares/sharedfile1.txt" using the WebDAV API
    Then the HTTP status code should be "400"
    And as "Alice" file "/Shares/sharedfile1.txt" should exist
    And as "Alice" folder "/Shares/sharedfile1.txt" should not exist
    And user "Alice" should have a share "sharedfile1.txt" shared by user "Brian" from space "Personal"
    And as "Brian" file "sharedfile1.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a folder into another folder at different level which is received as a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder/third-level-folder"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b/sample-folder-c"
    When user "Alice" copies folder "Sample-Folder-A/sample-folder-b" to "Shares/BRIAN-FOLDER/second-level-folder/third-level-folder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/Sample-Folder-A/sample-folder-b/sample-folder-c" should exist
    And as "Alice" folder "/Shares/BRIAN-FOLDER/second-level-folder/third-level-folder/sample-folder-c" should exist
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a file into a folder at different level received as a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder/third-level-folder"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "Sample-Folder-A/sample-folder-b/textfile-c.txt" to "Shares/BRIAN-FOLDER/second-level-folder" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/third-level-folder" should not exist
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" file "Shares/BRIAN-FOLDER/second-level-folder" should exist
    And the content of file "Shares/BRIAN-FOLDER/second-level-folder" for user "Alice" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a file into a file at different level received as a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has uploaded file with content "file at second level" to "BRIAN-FOLDER/second-level-file.txt"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "Sample-Folder-A"
    And user "Alice" has created folder "Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "Sample-Folder-A/sample-folder-b/textfile-c.txt" to "Shares/BRIAN-FOLDER/second-level-file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And as "Alice" file "Shares/BRIAN-FOLDER/second-level-file.txt" should exist
    And as "Alice" file "Shares/BRIAN-FOLDER/textfile-c.txt" should not exist
    And the content of file "Shares/BRIAN-FOLDER/second-level-file.txt" for user "Alice" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-1239 @skipOnReva
  Scenario Outline: copy a folder into a file at different level received as a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has created folder "BRIAN-FOLDER/second-level-folder"
    And user "Brian" has uploaded file with content "file at third level" to "BRIAN-FOLDER/second-level-folder/third-level-file.txt"
    And using SharingNG
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "FOLDER/second-level-folder"
    And user "Alice" has created folder "FOLDER/second-level-folder/third-level-folder"
    When user "Alice" copies folder "FOLDER/second-level-folder" to "Shares/BRIAN-FOLDER/second-level-folder/third-level-file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/third-level-file.txt" should exist
    And as "Alice" folder "FOLDER/second-level-folder/third-level-folder" should exist
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/third-level-file.txt/third-level-folder" should exist
    And as "Alice" folder "Shares/BRIAN-FOLDER/second-level-folder/second-level-folder" should not exist
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copy a file of size zero byte
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/zerobyte.txt" to "/zerobyte.txt"
    And user "Alice" has created folder "/testZeroByte"
    When user "Alice" copies file "/zerobyte.txt" to "/testZeroByte/zerobyte.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/testZeroByte/zerobyte.txt" should exist
    And as "Alice" file "/zerobyte.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copy file into a nonexistent folder
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "fileToCopy.txt"
    When user "Alice" copies file "/fileToCopy.txt" to "/not-existing-folder/fileToCopy.txt" using the WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" file "/fileToCopy.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copy a nonexistent file into a folder
    Given using <dav-path-version> DAV path
    When user "Alice" copies file "/doesNotExist.txt" to "/FOLDER/doesNotExist.txt" using the WebDAV API
    Then the HTTP status code should be "404"
    And as "Alice" file "/FOLDER/doesNotExist.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copy a folder into a nonexistent one
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    When user "Alice" copies folder "/testshare" to "/not-existing/testshare" using the WebDAV API
    Then the HTTP status code should be "409"
    And user "Alice" should see the following elements
      | /testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: copying a file into a shared folder as the sharee
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/textfile0.txt" to "/Shares/testshare/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/Shares/testshare/textfile0.txt" for user "Alice" should be "OpenCloud test text file 0"
    And the content of file "/testshare/textfile0.txt" for user "Brian" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: copying a file into a shared folder as the sharer
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Alice" has a share "testshare" synced
    And user "Brian" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    When user "Brian" copies file "/textfile0.txt" to "/testshare/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/Shares/testshare/textfile0.txt" for user "Alice" should be "OpenCloud test text file 0"
    And the content of file "/testshare/textfile0.txt" for user "Brian" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: copying a file out of a shared folder as the sharee
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Alice" has a share "testshare" synced
    And user "Alice" has uploaded file with content "OpenCloud test text file inside share" to "/Shares/testshare/fileInsideShare.txt"
    When user "Alice" copies file "/Shares/testshare/fileInsideShare.txt" to "/fileInsideShare.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/fileInsideShare.txt" should exist
    And the content of file "/fileInsideShare.txt" for user "Alice" should be "OpenCloud test text file inside share"
    And the content of file "/testshare/fileInsideShare.txt" for user "Brian" should be "OpenCloud test text file inside share"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: sharee copies a file from a shared folder, shared with  viewer permission
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "hello world" to "testshare/fileInsideShare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/Shares/testshare/fileInsideShare.txt" to "/fileInsideShare.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/fileInsideShare.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @env-config
  Scenario Outline: sharee copies a file from a shared folder, shared with secure viewer permission
    Given using <dav-path-version> DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "hello world" to "testshare/fileInsideShare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare     |
      | space           | Personal      |
      | sharee          | Alice         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/Shares/testshare/fileInsideShare.txt" to "/fileInsideShare.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/fileInsideShare.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: copying a file out of a shared folder as the sharer
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "OpenCloud test text file inside share" to "/testshare/fileInsideShare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Alice" has a share "testshare" synced
    When user "Brian" copies file "testshare/fileInsideShare.txt" to "/fileInsideShare.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/fileInsideShare.txt" should exist
    And the content of file "/Shares/testshare/fileInsideShare.txt" for user "Alice" should be "OpenCloud test text file inside share"
    And the content of file "/fileInsideShare.txt" for user "Brian" should be "OpenCloud test text file inside share"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copying a hidden file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded the following files with content "hidden file"
      | path                    |
      | .hidden_file101         |
      | /FOLDER/.hidden_file102 |
    When user "Alice" copies file ".hidden_file101" to "/FOLDER/.hidden_file101" using the WebDAV API
    And user "Alice" copies file "/FOLDER/.hidden_file102" to ".hidden_file102" using the WebDAV API
    And as "Alice" the following files should exist
      | path                    |
      | .hidden_file102         |
      | /FOLDER/.hidden_file101 |
    And the content of the following files for user "Alice" should be "hidden file"
      | path                    |
      | .hidden_file102         |
      | /FOLDER/.hidden_file101 |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: copying a file between shares received from different users
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Carol" has been created with default attributes
    And user "Brian" has created folder "/testshare0"
    And user "Brian" has uploaded file with content "content inside testshare0" to "/testshare0/testshare0.txt"
    And user "Carol" has created folder "/testshare1"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare0 |
      | space           | Personal   |
      | sharee          | Alice      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Alice" has a share "testshare0" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | testshare1 |
      | space           | Personal   |
      | sharee          | Alice      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Alice" has a share "testshare1" synced
    When user "Alice" copies file "/Shares/testshare0/testshare0.txt" to "/Shares/testshare1/testshare0.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Carol" file "testshare1/testshare0.txt" should exist
    And as "Alice" file "Shares/testshare1/testshare0.txt" should exist
    And the content of file "/testshare1/testshare0.txt" for user "Carol" should be "content inside testshare0"
    And the content of file "/Shares/testshare1/testshare0.txt" for user "Alice" should be "content inside testshare0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @env-config
  Scenario Outline: copying a file between shares received from different users when one share is shared via Viewer and Secure viewer permission
    Given using <dav-path-version> DAV path
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Brian" has been created with default attributes
    And user "Carol" has been created with default attributes
    And user "Brian" has created folder "/testshare0"
    And user "Brian" has uploaded file with content "content inside testshare0" to "/testshare0/testshare0.txt"
    And user "Brian" has created folder "/testshare0/folder_to_copy/"
    And user "Carol" has created folder "/testshare1"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare0           |
      | space           | Personal             |
      | sharee          | Alice                |
      | shareType       | user                 |
      | permissionsRole | <permissions-role-1> |
    And user "Alice" has a share "testshare0" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | testshare1           |
      | space           | Personal             |
      | sharee          | Alice                |
      | shareType       | user                 |
      | permissionsRole | <permissions-role-2> |
    And user "Alice" has a share "testshare1" synced
    When user "Alice" copies folder "/Shares/testshare0/folder_to_copy/" to "/Shares/testshare1/folder_to_copy/" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" folder "/Shares/testshare1/folder_to_copy/" should not exist
    When user "Alice" copies file "/Shares/testshare0/testshare0.txt" to "/Shares/testshare1/testshare0.txt" using the WebDAV API
    Then the HTTP status code should be "403"
    And as "Alice" file "/Shares/testshare1/testshare0.txt" should not exist

    Examples:
      | dav-path-version | permissions-role-1 | permissions-role-2 |
      | old              | Secure Viewer      | Secure Viewer      |
      | new              | Secure Viewer      | Secure Viewer      |
      | spaces           | Secure Viewer      | Secure Viewer      |
      | old              | Secure Viewer      | Viewer             |
      | new              | Secure Viewer      | Viewer             |
      | spaces           | Secure Viewer      | Viewer             |
      | old              | Editor             | Secure Viewer      |
      | new              | Editor             | Secure Viewer      |
      | spaces           | Editor             | Secure Viewer      |
      | old              | Viewer             | Secure Viewer      |
      | new              | Viewer             | Secure Viewer      |
      | spaces           | Viewer             | Secure Viewer      |

  @skipOnReva
  Scenario Outline: copying a folder between shares received from different users
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Carol" has been created with default attributes
    And user "Brian" has created folder "/testshare0"
    And user "Brian" has created folder "/testshare0/folder_to_copy/"
    And user "Brian" has uploaded file with content "content inside testshare0" to "/testshare0/folder_to_copy/testshare0.txt"
    And user "Carol" has created folder "/testshare1"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare0 |
      | space           | Personal   |
      | sharee          | Alice      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Alice" has a share "testshare0" synced
    And user "Carol" has sent the following resource share invitation:
      | resource        | testshare1 |
      | space           | Personal   |
      | sharee          | Alice      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Alice" has a share "testshare1" synced
    When user "Alice" copies file "/Shares/testshare0/folder_to_copy/" to "/Shares/testshare1/folder_to_copy/" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Carol" file "testshare1/folder_to_copy/testshare0.txt" should exist
    And as "Alice" file "/Shares/testshare1/folder_to_copy/testshare0.txt" should exist
    And the content of file "testshare1/folder_to_copy/testshare0.txt" for user "Carol" should be "content inside testshare0"
    And the content of file "/Shares/testshare1/folder_to_copy/testshare0.txt" for user "Alice" should be "content inside testshare0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: copying a file to a folder that is shared with multiple users
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Carol" has been created with default attributes
    And user "Alice" has created folder "/testshare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Brian" has a share "testshare" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Carol     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Carol" has a share "testshare" synced
    When user "Alice" copies file "/textfile0.txt" to "/testshare/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Brian" file "/Shares/testshare/textfile0.txt" should exist
    And as "Carol" file "/Shares/testshare/textfile0.txt" should exist
    And the content of file "/Shares/testshare/textfile0.txt" for user "Brian" should be "OpenCloud test text file 0"
    And the content of file "/Shares/testshare/textfile0.txt" for user "Carol" should be "OpenCloud test text file 0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copy a folder into another one
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/testshare"
    And user "Alice" has created folder "/an-other-folder"
    When user "Alice" copies folder "/testshare" to "/an-other-folder/testshare" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should see the following elements
      | /testshare |
    And user "Alice" should see the following elements
      | /an-other-folder/testshare |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-3023
  Scenario Outline: copying a folder into a sub-folder of itself
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has created folder "/PARENT/CHILD"
    And user "Alice" has uploaded file with content "parent text" to "/PARENT/parent.txt"
    And user "Alice" has uploaded file with content "child text" to "/PARENT/CHILD/child.txt"
    When user "Alice" copies folder "/PARENT" to "/PARENT/CHILD/PARENT" using the WebDAV API
    Then the HTTP status code should be "409"
    And the content of file "/PARENT/parent.txt" for user "Alice" should be "parent text"
    And the content of file "/PARENT/CHILD/child.txt" for user "Alice" should be "child text"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copying a folder with a file into another folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/FOLDER1"
    And user "Alice" has created folder "/FOLDER2"
    And user "Alice" has uploaded file with content "Folder 1 text" to "/FOLDER1/textfile.txt"
    When user "Alice" copies folder "/FOLDER1" to "/FOLDER2/FOLDER1" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" folder "/FOLDER2/FOLDER1" should exist
    And as "Alice" file "/FOLDER2/FOLDER1/textfile.txt" should exist
    And as "Alice" folder "/FOLDER1" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: copy a file into a folder with special characters
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder <folder-name>
    And user "Alice" has uploaded file with content "test file" to <file-name>
    When user "Alice" copies file <file-name> to <destination> using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file <file-name> should exist
    And as "Alice" file <destination> should exist
    And as "Alice" folder <folder-name> should exist
    Examples:
      | dav-path-version | file-name   | folder-name        | destination                  |
      | old              | "'single'"  | "folder-'single'"  | "folder-'single'/'single'"   |
      | old              | "question?" | "folder-question?" | "folder-question?/question?" |
      | old              | "&and#hash" | "folder-&and#hash" | "folder-&and#hash/&and#hash" |
      | new              | "'single'"  | "folder-'single'"  | "folder-'single'/'single'"   |
      | new              | "question?" | "folder-question?" | "folder-question?/question?" |
      | new              | "&and#hash" | "folder-&and#hash" | "folder-&and#hash/&and#hash" |
      | spaces           | "'single'"  | "folder-'single'"  | "folder-'single'/'single'"   |
      | spaces           | "question?" | "folder-question?" | "folder-question?/question?" |
      | spaces           | "&and#hash" | "folder-&and#hash" | "folder-&and#hash/&and#hash" |

  @issue-8711
  Scenario Outline: copying a file to itself
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "lorem epsum" to "textfile.txt"
    When user "Alice" copies file "textfile.txt" to "textfile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "textfile.txt" for user "Alice" should be "lorem epsum"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-8711
  Scenario Outline: copying a folder to itself
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "FOLDER1"
    And user "Alice" has uploaded file with content "Folder 1 text" to "FOLDER1/textfile.txt"
    When user "Alice" copies folder "FOLDER1" to "FOLDER1" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "FOLDER1" should exist
    And the content of file "FOLDER1/textfile.txt" for user "Alice" should be "lorem epsum"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
