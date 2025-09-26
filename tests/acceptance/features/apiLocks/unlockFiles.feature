Feature: unlock locked items
  As a user
  I want to unlock the resources previously locked by myself
  So that other users can make changes to the resources

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: unlock file locked by the user
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    And user "Alice" has locked file "textfile0.txt" setting the following properties
      | lockscope | exclusive |
    When user "Alice" unlocks the last created lock of file "textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And 0 locks should be reported for file "textfile0.txt" of user "Alice" by the WebDAV API
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-7761 @issue-10331
  Scenario Outline: public tries to unlock a file in a share that was locked by the file owner
    Given using <dav-path-version> DAV path
    And using SharingNG
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    And user "Alice" has locked file "PARENT/parent.txt" setting the following properties
      | lockscope | <lock-scope> |
    When the public unlocks file "/parent.txt" with the last created lock of file "PARENT/parent.txt" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    And 1 locks should be reported for file "PARENT/parent.txt" of user "Alice" by the WebDAV API
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | exclusive  |
      | new              | shared     |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7599
  Scenario Outline: unlock one of multiple locks set by the user itself
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    And user "Alice" has locked file "textfile0.txt" setting the following properties
      | lockscope | shared |
    And user "Alice" has locked file "textfile0.txt" setting the following properties
      | lockscope | shared |
    When user "Alice" unlocks the last created lock of file "textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And 1 locks should be reported for file "textfile0.txt" of user "Alice" by the WebDAV API
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-7638
  Scenario Outline: unlocking a file with the same name as another file in another part of the file system
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile0.txt"
    And user "Alice" has created folder "locked"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/locked/textfile0.txt"
    And user "Alice" has created folder "notlocked"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/notlocked/textfile0.txt"
    And user "Alice" has locked file "locked/textfile0.txt" setting the following properties
      | lockscope | <lock-scope> |
    And user "Alice" has locked file "notlocked/textfile0.txt" setting the following properties
      | lockscope | <lock-scope> |
    And user "Alice" has locked file "textfile0.txt" setting the following properties
      | lockscope | <lock-scope> |
    When user "Alice" unlocks the last created lock of file "notlocked/textfile0.txt" using the WebDAV API
    And user "Alice" unlocks the last created lock of file "textfile0.txt" using the WebDAV API
    Then user "Alice" should be able to upload file "filesForUpload/lorem.txt" to "/notlocked/textfile0.txt"
    And user "Alice" should be able to upload file "filesForUpload/lorem.txt" to "/textfile0.txt"
    But user "Alice" should not be able to upload file "filesForUpload/lorem.txt" to "/locked/textfile0.txt"
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7767
  Scenario Outline: trying to unlock a shared file that has been locked by the file owner
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT/parent.txt |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | File Editor       |
    And user "Brian" has a share "parent.txt" synced
    And user "Alice" has locked file "PARENT/parent.txt" setting the following properties
      | lockscope | <lock-scope> |
    When user "Brian" unlocks file "Shares/parent.txt" with the last created lock of file "PARENT/parent.txt" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    And 1 locks should be reported for file "PARENT/parent.txt" of user "Alice" by the WebDAV API
    And 1 locks should be reported for file "Shares/parent.txt" of user "Brian" by the WebDAV API
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7767
  Scenario Outline: trying to unlock a file inside the shared folder that has been locked by the file owner
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Alice" has locked file "PARENT/parent.txt" setting the following properties
      | lockscope | <lock-scope> |
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    When user "Brian" unlocks file "Shares/PARENT/parent.txt" with the last created lock of file "PARENT/parent.txt" of user "Alice" using the WebDAV API
    Then the HTTP status code should be "403"
    And 1 locks should be reported for file "PARENT/parent.txt" of user "Alice" by the WebDAV API
    And 1 locks should be reported for file "Shares/PARENT/parent.txt" of user "Brian" by the WebDAV API
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7599
  Scenario Outline: sharee unlocks a shared file
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT/parent.txt |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | File Editor       |
    And user "Brian" has a share "parent.txt" synced
    And user "Brian" has locked file "Shares/parent.txt" setting the following properties
      | lockscope | <lock-scope> |
    When user "Brian" unlocks the last created lock of file "Shares/parent.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And 0 locks should be reported for file "PARENT/parent.txt" of user "Alice" by the WebDAV API
    And 0 locks should be reported for file "Shares/parent.txt" of user "Brian" by the WebDAV API
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7599
  Scenario Outline: try to unlock a shared file locked by the receiver
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT/parent.txt |
      | space           | Personal          |
      | sharee          | Brian             |
      | shareType       | user              |
      | permissionsRole | File Editor       |
    And user "Brian" has a share "parent.txt" synced
    And user "Brian" has locked file "Shares/parent.txt" setting the following properties
      | lockscope | <lock-scope> |
    When user "Alice" unlocks file "PARENT/parent.txt" with the last created lock of file "Shares/parent.txt" of user "Brian" using the WebDAV API
    Then the HTTP status code should be "403"
    And 1 locks should be reported for file "PARENT/parent.txt" of user "Alice" by the WebDAV API
    And 1 locks should be reported for file "Shares/parent.txt" of user "Brian" by the WebDAV API
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7599
  Scenario Outline: try to unlock a file in a shared folder, which was locked by the sharee as the owner
    Given user "Brian" has been created with default attributes
    And using <dav-path-version> DAV path
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/parent.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | PARENT   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "PARENT" synced
    And user "Brian" has locked file "Shares/PARENT/parent.txt" setting the following properties
      | lockscope | <lock-scope> |
    When user "Alice" unlocks file "PARENT/parent.txt" with the last created lock of file "Shares/PARENT/parent.txt" of user "Brian" using the WebDAV API
    Then the HTTP status code should be "403"
    And 1 locks should be reported for file "PARENT/parent.txt" of user "Alice" by the WebDAV API
    And 1 locks should be reported for file "Shares/PARENT/parent.txt" of user "Brian" by the WebDAV API
    Examples:
      | dav-path-version | lock-scope |
      | old              | shared     |
      | old              | exclusive  |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |

  @issue-7696
  Scenario Outline: unlock a locked file in project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "project-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "project-space" with content "some data" to "textfile.txt"
    And user "Alice" has locked file "textfile.txt" inside space "project-space" setting the following properties
      | lockscope | <lock-scope> |
    When user "Alice" unlocks the last created lock of file "textfile.txt" inside space "project-space" using the WebDAV API
    Then the HTTP status code should be "204"
    Examples:
      | lock-scope |
      | shared     |
      | exclusive  |


  Scenario: unlock a file using file-id
    Given using spaces DAV path
    And user "Alice" has uploaded a file inside space "Personal" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has locked file "textfile.txt" using file-id "<<FILEID>>" setting the following properties
      | lockscope | exclusive   |
      | timeout   | Second-3600 |
    When user "Alice" unlocks the last created lock of file "textfile.txt" using file-id "<<FILEID>>" using the WebDAV API
    Then the HTTP status code should be "204"
    And 0 locks should be reported for file "textfile.txt" of user "Alice" by the WebDAV API
    And user "Alice" should be able to upload file "filesForUpload/lorem.txt" to "textfile.txt"


  Scenario: unlock a file in project space using file-id
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "Project" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has locked file "textfile.txt" inside the space "Project" setting the following properties
      | lockscope | exclusive   |
      | timeout   | Second-3600 |
    When user "Alice" unlocks the last created lock of file "textfile.txt" using file-id "<<FILEID>>" using the WebDAV API
    Then the HTTP status code should be "204"
    And 0 locks should be reported for file "textfile.txt" inside the space "Project" of user "Alice"
    And user "Alice" should be able to upload file "filesForUpload/lorem.txt" to "textfile.txt"


  Scenario: unlock a file in the shares using file-id
    Given user "Brian" has been created with default attributes
    And using spaces DAV path
    And user "Alice" has uploaded a file inside space "Personal" with content "some content" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | File Editor  |
    And user "Brian" has a share "textfile.txt" synced
    And user "Brian" has locked file "textfile.txt" using file-id "<<FILEID>>" setting the following properties
      | lockscope | exclusive   |
      | timeout   | Second-3600 |
    When user "Brian" unlocks the last created lock of file "textfile.txt" using file-id "<<FILEID>>" using the WebDAV API
    Then the HTTP status code should be "204"
    And 0 locks should be reported for file "textfile.txt" inside the space "Personal" of user "Alice"
    And 0 locks should be reported for file "textfile.txt" inside the space "Shares" of user "Brian"
    And user "Alice" should be able to upload file "filesForUpload/lorem.txt" to "textfile.txt"
    And using new DAV path
    And user "Brian" should be able to upload file "filesForUpload/lorem.txt" to "Shares/textfile.txt"

  @issue-10331
  Scenario Outline: unlock a file as an anonymous user
    Given using <dav-path-version> DAV path
    And using SharingNG
    And user "Alice" has created folder "PARENT"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "PARENT/textfile0.txt"
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    And the public has locked "textfile0.txt" in the last public link shared folder setting the following properties
      | lockscope | <lock-scope> |
    When the public unlocks file "textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And 0 locks should be reported for file "PARENT/textfile0.txt" of user "Alice" by the WebDAV API
    And user "Alice" should be able to upload file "filesForUpload/lorem.txt" to "PARENT/textfile0.txt"
    Examples:
      | dav-path-version | lock-scope |
      | new              | shared     |
      | new              | exclusive  |
      | spaces           | shared     |
      | spaces           | exclusive  |
