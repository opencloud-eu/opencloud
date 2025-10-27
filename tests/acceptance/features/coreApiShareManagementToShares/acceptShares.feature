@skipOnReva @owncloud-issue-1289 @owncloud-issue-1328
Feature: accept/decline shares coming from internal users
  As a user
  I want to have control of which received shares I accept
  So that I can keep my file system clean

  Background:
    Given using OCS API version "1"
    And using new DAV path
    And these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Carol" has been added to group "grp1"
    And user "Alice" has created folder "PARENT"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Brian" has created folder "PARENT"
    And user "Brian" has created folder "FOLDER"
    And user "Brian" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    And user "Brian" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"

  @smokeTest  @owncloud-issue-2540
  Scenario: share a file & folder with another internal group when auto accept is disabled
    Given user "Brian" has disabled auto-accepting
    And user "Carol" has disabled auto-accepting
    And user "Carol" has created folder "FOLDER"
    And user "Carol" has created folder "PARENT"
    And user "Carol" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    When user "Alice" shares folder "/PARENT" with group "grp1" using the sharing API
    And user "Alice" shares file "/textfile0.txt" with group "grp1" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /FOLDER        |
      | /PARENT        |
      | /textfile0.txt |
    But user "Brian" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Brian" that these shares are in the pending state
      | path           |
      | /PARENT/       |
      | /textfile0.txt |
    And user "Carol" should see the following elements
      | /FOLDER        |
      | /PARENT        |
      | /textfile0.txt |
    But user "Carol" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Carol" that these shares are in the pending state
      | path           |
      | /PARENT/       |
      | /textfile0.txt |

  @owncloud-issue-2540
  Scenario: share a file & folder with another internal user when auto accept is disabled
    Given user "Brian" has disabled auto-accepting
    When user "Alice" shares folder "/PARENT" with user "Brian" using the sharing API
    And user "Alice" shares file "/textfile0.txt" with user "Brian" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /FOLDER        |
      | /PARENT        |
      | /textfile0.txt |
    But user "Brian" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Brian" that these shares are in the pending state
      | path           |
      | /PARENT/       |
      | /textfile0.txt |

  @smokeTest @owncloud-issue-2131
  Scenario: accept a pending share
    Given user "Brian" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | File Editor   |
    When user "Brian" accepts share "/PARENT" offered by user "Alice" using the sharing API
    And user "Brian" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And the fields of the last response to user "Alice" sharing with user "Brian" should include
      | id                     | A_STRING                      |
      | share_type             | user                          |
      | uid_owner              | %username%                    |
      | displayname_owner      | %displayname%                 |
      | permissions            | read,update                   |
      | uid_file_owner         | %username%                    |
      | displayname_file_owner | %displayname%                 |
      | state                  | 0                             |
      | path                   | /Shares/textfile0.txt         |
      | item_type              | file                          |
      | mimetype               | text/plain                    |
      | storage_id             | shared::/Shares/textfile0.txt |
      | storage                | A_STRING                      |
      | item_source            | A_STRING                      |
      | file_source            | A_STRING                      |
      | file_target            | /Shares/textfile0.txt         |
      | share_with             | %username%                    |
      | share_with_displayname | %displayname%                 |
      | mail_send              | 0                             |
    And user "Brian" should see the following elements
      | /FOLDER                   |
      | /PARENT                   |
      | /textfile0.txt            |
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Brian" that these shares are in the accepted state
      | path                  |
      | /Shares/PARENT        |
      | /Shares/textfile0.txt |


  Scenario: accept an accepted share
    Given user "Alice" has created folder "/shared"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "shared" synced
    When user "Brian" accepts the already accepted share "/shared" offered by user "Alice" using the sharing API
    Then the OCS status code should be "100"
    And the HTTP status code should be "200"
    And user "Brian" should see the following elements
      | /Shares/shared |
    And the sharing API should report to user "Brian" that these shares are in the accepted state
      | path            |
      | /Shares/shared/ |

  @smokeTest  @owncloud-issue-2540
  Scenario: declines a pending share
    Given user "Brian" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Brian" declines share "/PARENT" offered by user "Alice" using the sharing API
    And user "Brian" declines share "/textfile0.txt" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /FOLDER        |
      | /PARENT        |
      | /textfile0.txt |
    But user "Brian" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Brian" that these shares are in the declined state
      | path           |
      | /PARENT/       |
      | /textfile0.txt |

  @smokeTest @owncloud-issue-2128 @owncloud-issue-2540
  Scenario: decline an accepted share
    Given user "Brian" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has accepted share "/PARENT" offered by user "Alice"
    And user "Brian" has accepted share "/textfile0.txt" offered by user "Alice"
    When user "Brian" declines share "/Shares/PARENT" offered by user "Alice" using the sharing API
    And user "Brian" declines share "/Shares/textfile0.txt" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Brian" that these shares are in the declined state
      | path           |
      | /PARENT/       |
      | /textfile0.txt |


  Scenario Outline: deleting shares in pending state
    Given using <dav-path-version> DAV path
    And user "Brian" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" deletes folder "/PARENT" using the WebDAV API
    And user "Alice" deletes file "/textfile0.txt" using the WebDAV API
    Then the HTTP status code of responses on all endpoints should be "204"
    And the sharing API should report that no shares are shared with user "Brian"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-2540
  Scenario: only one user in a group accepts a share
    Given user "Brian" has disabled auto-accepting
    And user "Carol" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Brian" accepts share "/PARENT" offered by user "Alice" using the sharing API
    And user "Brian" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Carol" should not see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Carol" that these shares are in the pending state
      | path           |
      | /PARENT/       |
      | /textfile0.txt |
    But user "Brian" should see the following elements
      | /Shares/PARENT            |
      | /Shares/PARENT/parent.txt |
      | /Shares/textfile0.txt     |
    And the sharing API should report to user "Brian" that these shares are in the accepted state
      | path                  |
      | /Shares/PARENT/       |
      | /Shares/textfile0.txt |

  @owncloud-issue-2131
  Scenario: receive two shares with identical names from different users, accept one by one
    Given user "Carol" has disabled auto-accepting
    And user "Alice" has created folder "/shared"
    And user "Alice" has created folder "/shared/Alice"
    And user "Brian" has created folder "/shared"
    And user "Brian" has created folder "/shared/Brian"
    And user "Alice" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Carol    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has sent the following resource share invitation:
      | resource        | shared   |
      | space           | Personal |
      | sharee          | Carol    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    When user "Carol" accepts share "/shared" offered by user "Brian" using the sharing API
    And user "Carol" accepts share "/shared" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Carol" should see the following elements
      | /Shares/shared/Brian     |
      | /Shares/shared (1)/Alice |
    And the sharing API should report to user "Carol" that these shares are in the accepted state
      | path                |
      | /Shares/shared/     |
      | /Shares/shared (1)/ |

  @owncloud-issue-2540
  Scenario: share with a group that you are part of yourself
    Given user "Brian" has disabled auto-accepting
    When user "Alice" shares folder "/PARENT" with group "grp1" using the sharing API
    Then the OCS status code should be "100"
    And the HTTP status code should be "200"
    And the sharing API should report to user "Brian" that these shares are in the pending state
      | path     |
      | /PARENT/ |
    And the sharing API should report that no shares are shared with user "Alice"


  Scenario: user accepts file that was initially accepted from another user and then declined
    Given user "Alice" has uploaded file with content "First file" to "/testfile.txt"
    And user "Brian" has uploaded file with content "Second file" to "/testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testfile.txt |
      | space           | Personal     |
      | sharee          | Carol        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Carol" has a share "testfile.txt" synced
    And user "Carol" has declined share "/Shares/testfile.txt" offered by user "Alice"
    And user "Carol" has disabled auto-accepting
    And user "Brian" has sent the following resource share invitation:
      | resource        | testfile.txt |
      | space           | Personal     |
      | sharee          | Carol        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Carol" accepts share "/testfile.txt" offered by user "Brian" using the sharing API
    Then the HTTP status code should be "200"
    And the OCS status code should be "100"
    And the sharing API should report to user "Carol" that these shares are in the accepted state
      | path                 |
      | /Shares/testfile.txt |
    And the content of file "/Shares/testfile.txt" for user "Carol" should be "Second file"


  Scenario: user accepts shares received from multiple users with the same name when auto-accept share is disabled
    Given user "Alice" has disabled auto-accepting
    And user "David" has been created with default attributes
    And user "David" has created folder "PARENT"
    And user "Brian" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Carol" has created folder "PARENT"
    And user "Carol" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    When user "Alice" accepts share "/PARENT" offered by user "Brian" using the sharing API
    And user "Alice" accepts share "/PARENT" offered by user "Carol" using the sharing API
    And user "Alice" declines share "/Shares/PARENT (1)" offered by user "Carol" using the sharing API
    And user "Alice" declines share "/Shares/PARENT" offered by user "Brian" using the sharing API
    And user "David" shares folder "/PARENT" with user "Alice" using the sharing API
    And user "Alice" accepts share "/PARENT" offered by user "David" using the sharing API
    And user "Alice" accepts share "/PARENT" offered by user "Carol" using the sharing API
    And user "Alice" accepts share "/PARENT" offered by user "Brian" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And the sharing API should report to user "Alice" that these shares are in the accepted state
      | path               | uid_owner |
      | /Shares/PARENT     | David     |
      | /Shares/PARENT (1) | Carol     |
      | /Shares/PARENT (2) | Brian     |


  Scenario: user shares folder with matching folder-name for both user involved in sharing
    Given user "Brian" has disabled auto-accepting
    And user "Alice" has uploaded file with content "uploaded content" to "/PARENT/abc.txt"
    And user "Alice" has uploaded file with content "uploaded content" to "/FOLDER/abc.txt"
    When user "Alice" shares folder "/PARENT" with user "Brian" using the sharing API
    And user "Alice" shares folder "/FOLDER" with user "Brian" using the sharing API
    And user "Brian" accepts share "/PARENT" offered by user "Alice" using the sharing API
    And user "Brian" accepts share "/FOLDER" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /FOLDER                |
      | /PARENT                |
      | /Shares/PARENT         |
      | /Shares/PARENT/abc.txt |
      | /Shares/FOLDER         |
      | /Shares/FOLDER/abc.txt |
    And user "Brian" should not see the following elements
      | /FOLDER/abc.txt |
      | /PARENT/abc.txt |
    And the content of file "/Shares/PARENT/abc.txt" for user "Brian" should be "uploaded content"
    And the content of file "/Shares/FOLDER/abc.txt" for user "Brian" should be "uploaded content"


  Scenario: user shares folder in a group with matching folder-name for every users involved
    Given user "Brian" has disabled auto-accepting
    And user "Carol" has disabled auto-accepting
    And user "Alice" has uploaded file with content "uploaded content" to "/PARENT/abc.txt"
    And user "Alice" has uploaded file with content "uploaded content" to "/FOLDER/abc.txt"
    And user "Carol" has created folder "PARENT"
    And user "Carol" has created folder "FOLDER"
    When user "Alice" shares folder "/PARENT" with group "grp1" using the sharing API
    And user "Alice" shares folder "/FOLDER" with group "grp1" using the sharing API
    And user "Brian" accepts share "/PARENT" offered by user "Alice" using the sharing API
    And user "Brian" accepts share "/FOLDER" offered by user "Alice" using the sharing API
    And user "Carol" accepts share "/PARENT" offered by user "Alice" using the sharing API
    And user "Carol" accepts share "/FOLDER" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /FOLDER                |
      | /PARENT                |
      | /Shares/PARENT         |
      | /Shares/FOLDER         |
      | /Shares/PARENT/abc.txt |
      | /Shares/FOLDER/abc.txt |
    And user "Brian" should not see the following elements
      | /FOLDER/abc.txt |
      | /PARENT/abc.txt |
    And user "Carol" should see the following elements
      | /FOLDER                |
      | /PARENT                |
      | /Shares/PARENT         |
      | /Shares/FOLDER         |
      | /Shares/PARENT/abc.txt |
      | /Shares/FOLDER/abc.txt |
    And user "Carol" should not see the following elements
      | /FOLDER/abc.txt |
      | /PARENT/abc.txt |
    And the content of file "/Shares/PARENT/abc.txt" for user "Brian" should be "uploaded content"
    And the content of file "/Shares/FOLDER/abc.txt" for user "Brian" should be "uploaded content"
    And the content of file "/Shares/PARENT/abc.txt" for user "Carol" should be "uploaded content"
    And the content of file "/Shares/FOLDER/abc.txt" for user "Carol" should be "uploaded content"


  Scenario: user shares files in a group with matching file-names for every users involved in sharing
    Given user "Brian" has disabled auto-accepting
    And user "Carol" has disabled auto-accepting
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile1.txt"
    And user "Brian" has uploaded file "filesForUpload/textfile.txt" to "textfile1.txt"
    And user "Carol" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    And user "Carol" has uploaded file "filesForUpload/textfile.txt" to "textfile1.txt"
    When user "Alice" shares file "/textfile0.txt" with group "grp1" using the sharing API
    And user "Alice" shares file "/textfile1.txt" with group "grp1" using the sharing API
    And user "Brian" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    And user "Brian" accepts share "/textfile1.txt" offered by user "Alice" using the sharing API
    And user "Carol" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    And user "Carol" accepts share "/textfile1.txt" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /textfile0.txt        |
      | /textfile1.txt        |
      | /Shares/textfile0.txt |
      | /Shares/textfile1.txt |
    And user "Carol" should see the following elements
      | /textfile0.txt        |
      | /textfile1.txt        |
      | /Shares/textfile0.txt |
      | /Shares/textfile1.txt |


  Scenario: user shares resource with matching resource-name with another user when auto accept is disabled
    Given user "Brian" has disabled auto-accepting
    When user "Alice" shares folder "/PARENT" with user "Brian" using the sharing API
    And user "Alice" shares file "/textfile0.txt" with user "Brian" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /PARENT        |
      | /textfile0.txt |
    But user "Brian" should not see the following elements
      | /Shares/textfile0.txt |
      | /Shares/PARENT        |
    When user "Brian" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    And user "Brian" accepts share "/PARENT" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /PARENT               |
      | /textfile0.txt        |
      | /Shares/PARENT        |
      | /Shares/textfile0.txt |


  Scenario: user shares file in a group with matching filename when auto accept is disabled
    Given user "Brian" has disabled auto-accepting
    And user "Carol" has disabled auto-accepting
    And user "Carol" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    When user "Alice" shares file "/textfile0.txt" with group "grp1" using the sharing API
    Then the OCS status code should be "100"
    And the HTTP status code should be "200"
    And user "Brian" should see the following elements
      | /textfile0.txt |
    But user "Brian" should not see the following elements
      | /Shares/textfile0.txt |
    And user "Carol" should see the following elements
      | /textfile0.txt |
    But user "Carol" should not see the following elements
      | /Shares/textfile0.txt |
    When user "Brian" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    And user "Carol" accepts share "/textfile0.txt" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /textfile0.txt        |
      | /Shares/textfile0.txt |
    And user "Carol" should see the following elements
      | /textfile0.txt        |
      | /Shares/textfile0.txt |


  Scenario: user shares folder with matching folder name to  a user before that user has logged in
    Given these users have been created without being initialized:
      | username |
      | David    |
    And user "David" has disabled auto-accepting
    And user "Alice" has uploaded file with content "uploaded content" to "/PARENT/abc.txt"
    When user "Alice" shares folder "/PARENT" with user "David" using the sharing API
    And user "David" accepts share "/PARENT" offered by user "Alice" using the sharing API
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "David" should see the following elements
      | /Shares/PARENT         |
      | /Shares/PARENT/abc.txt |
    And user "David" should not see the following elements
      | /PARENT (2) |
    And the content of file "/Shares/PARENT/abc.txt" for user "David" should be "uploaded content"

  @owncloud-issue-1123 @owncloud-issue-2540
  Scenario Outline: deleting a share accepted file and folder
    Given using <dav-path-version> DAV path
    And user "Brian" has disabled auto-accepting
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has accepted share "/PARENT" offered by user "Alice"
    When user "Brian" deletes file "/Shares/PARENT" using the WebDAV API
    Then the HTTP status code should be "204"
    And the sharing API should report to user "Brian" that these shares are in the declined state
      | path    |
      | /PARENT |
    Examples:
      | dav-path-version |
      | old              |
      | new              |

  @owncloud-issue-765 @owncloud-issue-2131
  Scenario Outline: shares exist after restoring already shared file to a previous version
    Given using <dav-path-version> DAV path
    And user "Brian" has disabled auto-accepting
    And user "Alice" has uploaded file with content "Test Content." to "/toShareFile.txt"
    And user "Alice" has uploaded file with content "Content Test Updated." to "/toShareFile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | toShareFile.txt |
      | space           | Personal        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | File Editor     |
    And user "Brian" has accepted share "/toShareFile.txt" offered by user "Alice"
    When user "Alice" restores version index "1" of file "/toShareFile.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/toShareFile.txt" for user "Alice" should be "Test Content."
    And the content of file "/Shares/toShareFile.txt" for user "Brian" should be "Test Content."
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-2131
  Scenario: user receives multiple group shares for matching file and folder name
    Given user "Brian" has disabled auto-accepting
    And group "grp2" has been created
    And user "Alice" has been added to group "grp2"
    And user "Brian" has been added to group "grp2"
    And user "Carol" has created folder "/PARENT"
    And user "Alice" has created folder "/PaRent"
    And user "Alice" has uploaded the following files with content "subfile, from alice to grp2"
      | path               |
      | /PARENT/parent.txt |
      | /PaRent/parent.txt |
    And user "Alice" has uploaded the following files with content "from alice to grp2"
      | path        |
      | /PARENT.txt |
    And user "Carol" has uploaded the following files with content "subfile, from carol to grp1"
      | path               |
      | /PARENT/parent.txt |
    And user "Carol" has uploaded the following files with content "from carol to grp1"
      | path        |
      | /PARENT.txt |
      | /parent.txt |
    When user "Alice" shares the following entries with group "grp2" using the sharing API
      | path        |
      | /PARENT     |
      | /PaRent     |
      | /PARENT.txt |
    And user "Brian" accepts the following shares offered by user "Alice" using the sharing API
      | path        |
      | /PARENT     |
      | /PaRent     |
      | /PARENT.txt |
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /PARENT            |
      | /Shares/PARENT     |
      | /Shares/PaRent     |
      | /Shares/PARENT.txt |
    And the content of file "/Shares/PARENT/parent.txt" for user "Brian" should be "subfile, from alice to grp2"
    And the content of file "/Shares/PaRent/parent.txt" for user "Brian" should be "subfile, from alice to grp2"
    And the content of file "/Shares/PARENT.txt" for user "Brian" should be "from alice to grp2"
    When user "Carol" shares the following entries with group "grp2" using the sharing API
      | path        |
      | /PARENT     |
      | /PARENT.txt |
      | /parent.txt |
    And user "Brian" accepts the following shares offered by user "Carol" using the sharing API
      | path        |
      | /PARENT     |
      | /PARENT.txt |
      | /parent.txt |
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Brian" should see the following elements
      | /PARENT                |
      | /Shares/PARENT         |
      | /Shares/PARENT (1)     |
      | /Shares/PaRent         |
      | /Shares/PARENT.txt     |
      | /Shares/PARENT (1).txt |
      | /Shares/parent.txt     |
    And the content of file "/Shares/PARENT (1)/parent.txt" for user "Brian" should be "subfile, from carol to grp1"
    And the content of file "/Shares/PARENT (1).txt" for user "Brian" should be "from carol to grp1"
    And the content of file "/Shares/parent.txt" for user "Brian" should be "from carol to grp1"

  @owncloud-issue-2131
  Scenario: group receives multiple shares from non-member for matching file and folder name
    Given user "Carol" has disabled auto-accepting
    And user "Brian" has been removed from group "grp1"
    And user "Alice" has created folder "/PaRent"
    And user "Carol" has created folder "/PARENT"
    And user "Alice" has uploaded the following files with content "subfile, from alice to grp1"
      | path               |
      | /PARENT/parent.txt |
      | /PaRent/parent.txt |
    And user "Alice" has uploaded the following files with content "from alice to grp1"
      | path        |
      | /PARENT.txt |
    And user "Brian" has uploaded the following files with content "subfile, from brian to grp1"
      | path               |
      | /PARENT/parent.txt |
    And user "Brian" has uploaded the following files with content "from brian to grp1"
      | path        |
      | /PARENT.txt |
      | /parent.txt |
    When user "Alice" shares the following entries with group "grp1" using the sharing API
      | path        |
      | /PARENT     |
      | /PaRent     |
      | /PARENT.txt |
    And user "Carol" accepts the following shares offered by user "Alice" using the sharing API
      | path        |
      | /PARENT     |
      | /PaRent     |
      | /PARENT.txt |
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Carol" should see the following elements
      | /PARENT            |
      | /Shares/PARENT     |
      | /Shares/PaRent     |
      | /Shares/PARENT.txt |
    And the content of file "/Shares/PARENT/parent.txt" for user "Carol" should be "subfile, from alice to grp1"
    And the content of file "/Shares/PARENT.txt" for user "Carol" should be "from alice to grp1"
    When user "Brian" shares the following entries with group "grp1" using the sharing API
      | path        |
      | /PARENT     |
      | /PARENT.txt |
      | /parent.txt |
    And user "Carol" accepts the following shares offered by user "Brian" using the sharing API
      | path        |
      | /PARENT     |
      | /PARENT.txt |
      | /parent.txt |
    Then the OCS status code of responses on all endpoints should be "100"
    And the HTTP status code of responses on all endpoints should be "200"
    And user "Carol" should see the following elements
      | /PARENT                |
      | /Shares/PARENT         |
      | /Shares/PARENT (1)     |
      | /Shares/PaRent         |
      | /Shares/PARENT.txt     |
      | /Shares/PARENT (1).txt |
      | /Shares/parent.txt     |
    And the content of file "/Shares/PARENT (1)/parent.txt" for user "Carol" should be "subfile, from brian to grp1"
    And the content of file "/Shares/PARENT (1).txt" for user "Carol" should be "from brian to grp1"
