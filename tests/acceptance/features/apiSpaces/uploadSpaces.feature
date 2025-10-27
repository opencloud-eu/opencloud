Feature: Upload files into a space
  As a user
  I want to be able to create folders and files in the space
  So that I can store various information in them

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project Ceres" of type "project" with quota "2000"
    And using spaces DAV path


  Scenario Outline: user creates a folder in the space via the Graph API
    Given user "Alice" has sent the following space share invitation:
      | space           | Project Ceres |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" creates a folder "mainFolder" in space "Project Ceres" using the WebDav Api
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" the space "Project Ceres" <should-or-not> contain these entries:
      | mainFolder |
    Examples:
      | space-role   | http-status-code | should-or-not |
      | Manager      | 201              | should        |
      | Space Editor | 201              | should        |
      | Space Viewer | 403              | should not    |


  Scenario Outline: user uploads a file in shared space via the Graph API
    Given user "Alice" has sent the following space share invitation:
      | space           | Project Ceres |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | <space-role>  |
    When user "Brian" uploads a file inside space "Project Ceres" with content "Test" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" the space "Project Ceres" <should-or-not> contain these entries:
      | test.txt |
    Examples:
      | space-role   | http-status-code | should-or-not |
      | Manager      | 201              | should        |
      | Space Editor | 201              | should        |
      | Space Viewer | 403              | should not    |


  Scenario: user can create subfolders in a space via the Graph API
    When user "Alice" creates a subfolder "mainFolder/subFolder1/subFolder2" in space "Project Ceres" using the WebDav Api
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project Ceres" should contain these entries:
      | mainFolder |
    And for user "Alice" folder "mainFolder/subFolder1/" of the space "Project Ceres" should contain these entries:
      | subFolder2 |


  Scenario: user can create a folder and upload a file to a space
    When user "Alice" creates a folder "NewFolder" in space "Project Ceres" using the WebDav Api
    Then the HTTP status code should be "201"
    And user "Alice" uploads a file inside space "Project Ceres" with content "Test" to "test.txt" using the WebDAV API
    And the HTTP status code should be "201"
    And for user "Alice" the space "Project Ceres" should contain these entries:
      | NewFolder |
      | test.txt  |


  Scenario: user cannot create a folder or a file in a space if they do not have permission
    When user "Bob" creates a folder "forAlice" in space "Project Ceres" owned by the user "Alice" using the WebDav Api
    Then the HTTP status code should be "404"
    When user "Bob" uploads a file inside space "Project Ceres" owned by the user "Alice" with content "Test" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "404"
    And for user "Alice" the space "Project Ceres" should not contain these entries:
      | forAlice |
      | test.txt |


  Scenario: user cannot create folder with an existing name
    Given user "Alice" has created a folder "NewFolder" in space "Project Ceres"
    When user "Alice" creates a folder "NewFolder" in space "Project Ceres" using the WebDav Api
    Then the HTTP status code should be "405"


  Scenario Outline: user cannot create subfolder in a nonexistent folder
    When user "Alice" tries to create subfolder "<folder-name>" in a nonexistent folder of the space "Project Ceres" using the WebDav Api
    Then the HTTP status code should be "409"
    Examples:
      | folder-name |
      | foo/bar     |
      | foo/bar/baz |

  @owncloud-issue-10331 @owncloud-issue-10469
  Scenario: public uploads a zero byte file to a public share folder
    Given using SharingNG
    And user "Alice" has created folder "/uploadFolder"
    And user "Alice" has created the following resource link share:
      | resource        | uploadFolder |
      | space           | Personal     |
      | permissionsRole | createOnly   |
      | password        | %public%     |
    When the public uploads file "filesForUpload/zerobyte.txt" to "textfile.txt" inside last link shared folder with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "uploadFolder" of the space "Personal" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "uploadFolder" of the space "Personal" should not contain these files:
      | textfile (1).txt |
      | textfile (2).txt |

  @owncloud-issue-10331 @owncloud-issue-10469
  Scenario: public uploads a zero byte file to a public share folder inside project space
    Given using SharingNG
    And user "Alice" has created a folder "/uploadFolder" in space "Project Ceres"
    And user "Alice" has created the following resource link share:
      | resource        | uploadFolder  |
      | space           | Project Ceres |
      | permissionsRole | createOnly    |
      | password        | %public%      |
    When the public uploads file "filesForUpload/zerobyte.txt" to "textfile.txt" inside last link shared folder with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "uploadFolder" of the space "Project Ceres" should contain these files:
      | textfile.txt |
    And for user "Alice" folder "uploadFolder" of the space "Project Ceres" should not contain these files:
      | textfile (1).txt |
      | textfile (2).txt |

  @owncloud-issue-10331 @owncloud-issue-10469
  Scenario: public uploads a zero byte file to a public share project space
    Given using SharingNG
    And user "Alice" has created the following space link share:
      | space           | Project Ceres |
      | permissionsRole | createOnly    |
      | password        | %public%      |
    When the public uploads file "filesForUpload/zerobyte.txt" to "textfile.txt" inside last link shared folder with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project Ceres" should contain these files:
      | textfile.txt |
    And for user "Alice" the space "Project Ceres" should not contain these files:
      | textfile (1).txt |
      | textfile (2).txt |
