Feature: copy file
  As a user
  I want to be able to copy files
  So that I can manage my files

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario Outline: copying a file within a same project space with role manager and editor
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/newfolder" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "some content" to "/insideSpace.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Brian" copies file "/insideSpace.txt" to "/newfolder/insideSpace.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Brian" folder "newfolder" of the space "Project" should contain these files:
      | insideSpace.txt |
    And for user "Alice" the content of the file "/newfolder/insideSpace.txt" of the space "Project" should be "some content"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |


  Scenario: copying a file within a same project space with role viewer
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/newfolder" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "some content" to "insideSpace.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" copies file "/insideSpace.txt" to "/newfolder/insideSpace.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Brian" folder "newfolder" of the space "Project" should not contain these files:
      | insideSpace.txt |


  Scenario Outline: user copies a file from a project space with a different role to a project space with the manager role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project1" with the default quota using the Graph API
    And user "Brian" has created a space "Project2" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project1" with content "Project1 content" to "/project1.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project2        |
      | sharee          | Alice           |
      | shareType       | user            |
      | permissionsRole | <to-space-role> |
    And user "Brian" has sent the following space share invitation:
      | space           | Project1          |
      | sharee          | Alice             |
      | shareType       | user              |
      | permissionsRole | <from-space-role> |
    When user "Alice" copies file "/project1.txt" from space "Project1" to "/project1.txt" inside space "Project2" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project2" should contain these entries:
      | /project1.txt |
    And for user "Alice" the content of the file "/project1.txt" of the space "Project2" should be "Project1 content"
    Examples:
      | from-space-role | to-space-role |
      | Manager         | Manager       |
      | Manager         | Space Editor  |
      | Space Editor    | Manager       |
      | Space Editor    | Space Editor  |


  Scenario Outline: user copies a file from a project space with a different role to a project space with a viewer role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project1" with the default quota using the Graph API
    And user "Brian" has created a space "Project2" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project1" with content "Project1 content" to "/project1.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project2     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Brian" has sent the following space share invitation:
      | space           | Project1     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" copies file "/project1.txt" from space "Project1" to "/project1.txt" inside space "Project2" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Project2" should not contain these entries:
      | project1.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |


  Scenario Outline: user copies a file from project space with different role to personal space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "/project.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" copies file "/project.txt" from space "Project" to "/project.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | project.txt |
    And for user "Alice" the content of the file "/project.txt" of the space "Personal" should be "Project content"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: user copies a file from project space with different role to share space with editor role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "/project.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/project.txt" from space "Project" to "/testshare/project.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "testshare" of the space "Shares" should contain these files:
      | /project.txt |
    And for user "Alice" the content of the file "/testshare/project.txt" of the space "Shares" should be "Project content"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: user copies a file from project space with different role to Shares with viewer role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "/project.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/project.txt" from space "Project" to "/testshare/project.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these files:
      | project.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: user copies a file from personal space to project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Alice" has uploaded file with content "personal space content" to "/personal.txt"
    When user "Alice" copies file "/personal.txt" from space "Personal" to "/personal.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project" should contain these entries:
      | /personal.txt |
    And for user "Alice" the content of the file "/personal.txt" of the space "Project" should be "personal space content"
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |


  Scenario: user copies a file from personal space to project space with role viewer
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Alice" has uploaded file with content "personal space content" to "/personal.txt"
    When user "Alice" copies file "/personal.txt" from space "Personal" to "/personal.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Project" should not contain these entries:
      | /personal.txt |


  Scenario: user copies a file from personal space to share space with role editor
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Alice" has a share "testshare" synced
    And user "Alice" has uploaded file with content "personal content" to "personal.txt"
    When user "Alice" copies file "/personal.txt" from space "Personal" to "/testshare/personal.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "testshare" of the space "Shares" should contain these files:
      | personal.txt |
    And for user "Alice" the content of the file "/testshare/personal.txt" of the space "Shares" should be "personal content"


  Scenario: user copies a file from personal space to share space with role viewer
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    And user "Alice" has uploaded file with content "personal content" to "/personal.txt"
    When user "Alice" copies file "/personal.txt" from space "Personal" to "/testshare/personal.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these files:
      | personal.txt |


  Scenario Outline: user copies a file from share space with different role to personal space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/testshare/testshare.txt" from space "Shares" to "/testshare.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | /testshare.txt |
    And for user "Alice" the content of the file "/testshare.txt" of the space "Personal" should be "testshare content"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |

  @owncloud-issue-9482 @env-config
  Scenario: user copies a file from share space with secure viewer role to personal space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare     |
      | space           | Personal      |
      | sharee          | Alice         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/testshare/testshare.txt" from space "Shares" to "/testshare.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Personal" should not contain these entries:
      | /testshare.txt |


  Scenario Outline: user copies a file from share space with different role to project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/testshare/testshare.txt" from space "Shares" to "/testshare.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project" should contain these entries:
      | /testshare.txt |
    And for user "Alice" the content of the file "/testshare.txt" of the space "Project" should be "testshare content"
    Examples:
      | space-role   | permissions-role |
      | Manager      | Editor           |
      | Manager      | Viewer           |
      | Space Editor | Editor           |
      | Space Editor | Viewer           |

  @owncloud-issue-9482 @env-config
  Scenario Outline: user copies a file from share space with secure viewer role to project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare     |
      | space           | Personal      |
      | sharee          | Alice         |
      | shareType       | user          |
      | permissionsRole | Secure Viewer |
    When user "Alice" copies file "/testshare/testshare.txt" from space "Shares" to "/testshare.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Project" should not contain these entries:
      | /testshare.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |


  Scenario Outline: user copies a file from share space with different role to project space with role viewer
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/testshare/testshare.txt" from space "Shares" to "/testshare.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Project" should not contain these entries:
      | /testshare.txt |
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario Outline: user copies a file from share space with different role to share space with role editor
    Given user "Brian" has created folder "/testshare1"
    And user "Brian" has created folder "/testshare2"
    And user "Brian" has uploaded file with content "testshare1 content" to "/testshare1/testshare1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare1         |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare1" synced
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare2 |
      | space           | Personal   |
      | sharee          | Alice      |
      | shareType       | user       |
      | permissionsRole | Editor     |
    And user "Alice" has a share "testshare2" synced
    When user "Alice" copies file "/testshare1/testshare1.txt" from space "Shares" to "/testshare2/testshare1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "testshare2" of the space "Shares" should contain these files:
      | /testshare1.txt |
    And for user "Brian" folder "testshare2" of the space "Personal" should contain these files:
      | /testshare1.txt |
    And for user "Alice" the content of the file "/testshare2/testshare1.txt" of the space "Shares" should be "testshare1 content"
    And for user "Brian" the content of the file "/testshare1/testshare1.txt" of the space "Personal" should be "testshare1 content"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |

  @owncloud-issue-9482 @env-config
  Scenario Outline: user copies a file from share space with different role to share space with role viewer or Secure Viewer
    Given user "Brian" has created folder "/testshare1"
    And user "Brian" has created folder "/testshare2"
    And the administrator has enabled the permissions role "Secure Viewer"
    And user "Brian" has uploaded file with content "testshare1 content" to "/testshare1/testshare1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare1           |
      | space           | Personal             |
      | sharee          | Alice                |
      | shareType       | user                 |
      | permissionsRole | <permissions-role-1> |
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare2           |
      | space           | Personal             |
      | sharee          | Alice                |
      | shareType       | user                 |
      | permissionsRole | <permissions-role-2> |
    When user "Alice" copies file "/testshare1/testshare1.txt" from space "Shares" to "/testshare2/testshare1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" folder "testshare2" of the space "Shares" should not contain these files:
      | testshare1.txt |
    And for user "Brian" folder "testshare2" of the space "Personal" should not contain these files:
      | testshare1.txt |
    Examples:
      | permissions-role-1 | permissions-role-2 |
      | Editor             | Viewer             |
      | Editor             | Secure Viewer      |
      | Viewer             | Viewer             |
      | Viewer             | Secure Viewer      |
      | Secure Viewer      | Viewer             |
      | Secure Viewer      | Secure Viewer      |


  Scenario Outline: copying a folder within the same project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/folder1" in space "Project"
    And user "Alice" has created a folder "/folder2" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "some content" to "/folder2/demo.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Brian" copies folder "/folder2" to "/folder1/folder2" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "Brian" folder "<parent-folder>" of the space "Project" <should-or-not> contain these files:
      | <resource> |
    Examples:
      | space-role   | should-or-not | http-status-code | parent-folder   | resource |
      | Manager      | should        | 201              | folder1/folder2 | demo.txt |
      | Space Editor | should        | 201              | folder1/folder2 | demo.txt |
      | Space Viewer | should not    | 403              | folder1         | folder2  |


  Scenario Outline: user copies a folder from a project space with different role to a project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project1" with the default quota using the Graph API
    And user "Brian" has created a space "Project2" with the default quota using the Graph API
    And user "Brian" has created a folder "/folder1" in space "Project1"
    And user "Brian" has uploaded a file inside space "Project1" with content "some content" to "/folder1/demo.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project2        |
      | sharee          | Alice           |
      | shareType       | user            |
      | permissionsRole | <to-space-role> |
    And user "Brian" has sent the following space share invitation:
      | space           | Project1          |
      | sharee          | Alice             |
      | shareType       | user              |
      | permissionsRole | <from-space-role> |
    When user "Alice" copies folder "/folder1" from space "Project1" to "/folder1" inside space "Project2" using the WebDAV API
    Then the HTTP status code should be "<status-code>"
    And for user "Alice" folder "<parent-folder>" of the space "Project2" <should-or-not> contain these files:
      | <entry> |
    Examples:
      | from-space-role | to-space-role | status-code | should-or-not | parent-folder | entry    |
      | Manager         | Manager       | 201         | should        | folder1       | demo.txt |
      | Manager         | Space Editor  | 201         | should        | folder1       | demo.txt |
      | Space Editor    | Manager       | 201         | should        | folder1       | demo.txt |
      | Space Editor    | Space Editor  | 201         | should        | folder1       | demo.txt |
      | Manager         | Space Viewer  | 403         | should not    | /             | folder1  |
      | Space Editor    | Space Viewer  | 403         | should not    | /             | folder1  |
      | Space Viewer    | Space Viewer  | 403         | should not    | /             | folder1  |


  Scenario Outline: user copies a folder from project space with different role to personal space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created a folder "/folder1" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "/folder1/demo.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    When user "Alice" copies file "/folder1" from space "Project" to "/folder1" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder1" of the space "Personal" should contain these files:
      | demo.txt |
    Examples:
      | space-role   |
      | Manager      |
      | Space Editor |
      | Space Viewer |


  Scenario Outline: user copies a folder from project space with different role to share space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created folder "/testshare"
    And user "Brian" has created a folder "/folder1" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "/folder1/demo.txt"
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies folder "/folder1" from space "Project" to "/testshare/folder1" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "<status-code>"
    And for user "Alice" folder "<parent-folder>" of the space "Shares" <should-or-not> contain these files:
      | <entry> |
    Examples:
      | space-role   | should-or-not | permissions-role | status-code | parent-folder     | entry    |
      | Manager      | should        | Editor           | 201         | testshare/folder1 | demo.txt |
      | Space Editor | should        | Editor           | 201         | testshare/folder1 | demo.txt |
      | Space Viewer | should        | Editor           | 201         | testshare/folder1 | demo.txt |
      | Manager      | should not    | Viewer           | 403         | testshare         | folder1  |
      | Space Editor | should not    | Viewer           | 403         | testshare         | folder1  |
      | Space Viewer | should not    | Viewer           | 403         | testshare         | folder1  |


  Scenario Outline: user copies a folder from personal space to project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Alice" has created folder "/folder1"
    And user "Alice" has uploaded file with content "some content" to "folder1/demo.txt"
    When user "Alice" copies folder "/folder1" from space "Personal" to "/folder1" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "<status-code>"
    And for user "Alice" folder "<parent-folder>" of the space "Project" <should-or-not> contain these files:
      | <entry> |
    Examples:
      | space-role   | should-or-not | status-code | parent-folder | entry    |
      | Manager      | should        | 201         | folder1       | demo.txt |
      | Space Editor | should        | 201         | folder1       | demo.txt |
      | Space Viewer | should not    | 403         | /             | folder1  |


  Scenario Outline: user copies a folder from personal space to share space with different permissions
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    And user "Alice" has created folder "folder1"
    And user "Alice" has uploaded file with content "some content" to "folder1/demo.txt"
    When user "Alice" copies folder "/folder1" from space "Personal" to "/testshare/folder1" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "<status-code>"
    And for user "Alice" folder "<parent-folder>" of the space "Shares" <should-or-not> contain these files:
      | <resource> |
    Examples:
      | permissions-role | should-or-not | status-code | parent-folder     | resource |
      | Editor           | should        | 201         | testshare/folder1 | demo.txt |
      | Viewer           | should not    | 403         | testshare         | folder1  |


  Scenario Outline: user copies a folder from share space with different role to personal space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/testshare/testshare.txt" from space "Shares" to "/testshare.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Personal" should contain these entries:
      | /testshare.txt |
    And for user "Alice" the content of the file "/testshare.txt" of the space "Personal" should be "testshare content"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario Outline: user copies a folder from share space with different role to project space with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | <space-role> |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has created folder "/testshare/folder1"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/folder1/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies folder "/testshare/folder1" from space "Shares" to "folder1" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "folder1" of the space "Project" should contain these files:
      | testshare.txt |
    Examples:
      | space-role   | permissions-role |
      | Manager      | Editor           |
      | Manager      | Viewer           |
      | Space Editor | Editor           |
      | Space Editor | Viewer           |


  Scenario Outline: user copies a folder from share space with different role to project space with role viewer
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has sent the following space share invitation:
      | space           | Project      |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has created folder "/testshare/folder1"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/folder1/testshare.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare          |
      | space           | Personal           |
      | sharee          | Alice              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies folder "/testshare/folder1" from space "Shares" to "folder1" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" folder "/" of the space "Project" should not contain these files:
      | folder1 |
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario: copying a file to a folder with no permissions
    Given using spaces DAV path
    And user "Brian" has created folder "/testshare"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    When user "Alice" copies file "/textfile0.txt" from space "Personal" to "/testshare/textfile0.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And user "Alice" should not be able to download file "/testshare/textfile0.txt" from space "Shares"


  Scenario: copying a file to overwrite a file into a folder with no permissions
    Given using spaces DAV path
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "OpenCloud test text file 1" to "/testshare/overwritethis.txt"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile0.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | testshare |
      | space           | Personal  |
      | sharee          | Alice     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has a share "testshare" synced
    When user "Alice" copies file "/textfile0.txt" from space "Personal" to "/testshare/overwritethis.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the content of the file "/testshare/overwritethis.txt" of the space "Shares" should be "OpenCloud test text file 1"

  @owncloud-issue-7208
  Scenario: copy a file over the top of an existing folder received as a user share
    Given using spaces DAV path
    And user "Alice" has uploaded file with content "OpenCloud test text file 1" to "/textfile1.txt"
    And user "Brian" has created folder "/BRIAN-Folder"
    And user "Brian" has created folder "BRIAN-Folder/sample-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-Folder |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-Folder" synced
    When user "Alice" copies file "/textfile1.txt" from space "Personal" to "/BRIAN-Folder" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "400"
    And as "Alice" folder "Shares/BRIAN-Folder/sample-folder" should exist
    And as "Brian" folder "BRIAN-Folder/sample-folder" should exist
    But as "Alice" file "Shares/BRIAN-Folder" should not exist
    And as "Alice" file "Shares/textfile1.txt" should not exist
    And user "Alice" should have a share "BRIAN-Folder" shared by user "Brian" from space "Personal"

  @owncloud-issue-7208
  Scenario: copy a folder over the top of an existing file received as a user share
    Given using spaces DAV path
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has created folder "/FOLDER/sample-folder"
    And user "Brian" has uploaded file with content "file to share" to "/sharedfile1.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | sharedfile1.txt |
      | space           | Personal        |
      | sharee          | Alice           |
      | shareType       | user            |
      | permissionsRole | File Editor     |
    And user "Alice" has a share "sharedfile1.txt" synced
    When user "Alice" copies folder "/FOLDER" from space "Personal" to "/sharedfile1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "400"
    And for user "Alice" the content of the file "sharedfile1.txt" of the space "Shares" should be "file to share"
    And for user "Brian" the content of the file "sharedfile1.txt" of the space "Personal" should be "file to share"
    But as "Alice" folder "Shares/FOLDER/sample-folder" should not exist
    And user "Alice" should have a share "sharedfile1.txt" shared by user "Brian" from space "Personal"


  Scenario: copy a folder into another folder at different level which is received as a user share
    Given using spaces DAV path
    And using SharingNG
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder/third-level-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/Sample-Folder-A"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b/sample-folder-c"
    When user "Alice" copies folder "/Sample-Folder-A/sample-folder-b" from space "Personal" to "/BRIAN-FOLDER/second-level-folder/third-level-folder" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/Sample-Folder-A/sample-folder-b/sample-folder-c" should exist
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder/third-level-folder" of the space "Shares" should contain these entries:
      | sample-folder-c |
    And for user "Brian" folder "BRIAN-FOLDER/second-level-folder/third-level-folder" of the space "Personal" should contain these entries:
      | sample-folder-c |
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copy a file into a folder at different level received as a user share
    Given using spaces DAV path
    And using SharingNG
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder/third-level-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/Sample-Folder-A"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "/Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "/Sample-Folder-A/sample-folder-b/textfile-c.txt" from space "Personal" to "/BRIAN-FOLDER/second-level-folder" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder" of the space "Shares" should not contain these entries:
      | third-level-folder |
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And for user "Alice" folder "BRIAN-FOLDER" of the space "Shares" should contain these files:
      | /second-level-folder |
    And for user "Alice" the content of the file "/BRIAN-FOLDER/second-level-folder" of the space "Shares" should be "sample file-c"
    And for user "Brian" the content of the file "/BRIAN-FOLDER/second-level-folder" of the space "Personal" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copy a file into a file at different level received as a user share
    Given using spaces DAV path
    And using SharingNG
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has uploaded file with content "file at second level" to "/BRIAN-FOLDER/second-level-file.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/Sample-Folder-A"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "/Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "/Sample-Folder-A/sample-folder-b/textfile-c.txt" from space "Personal" to "/BRIAN-FOLDER/second-level-file.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And for user "Alice" folder "BRIAN-FOLDER" of the space "Shares" should contain these files:
      | /second-level-file.txt |
    And for user "Alice" folder "BRIAN-FOLDER" of the space "Shares" should not contain these files:
      | /textfile-c.txt |
    And for user "Alice" the content of the file "/BRIAN-FOLDER/second-level-file.txt" of the space "Shares" should be "sample file-c"
    And for user "Brian" the content of the file "/BRIAN-FOLDER/second-level-file.txt" of the space "Personal" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copy a folder into a file at different level received as a user share
    Given using spaces DAV path
    And using SharingNG
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has created folder "/FOLDER/second-level-folder"
    And user "Alice" has created folder "/FOLDER/second-level-folder/third-level-folder"
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder"
    And user "Brian" has uploaded file with content "file at third level" to "BRIAN-FOLDER/second-level-folder/third-level-file.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    When user "Alice" copies folder "/FOLDER/second-level-folder" from space "Personal" to "/BRIAN-FOLDER/second-level-folder/third-level-file.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder/third-level-file.txt" of the space "Shares" should contain these entries:
      | third-level-folder |
    But for user "Alice" folder "BRIAN-FOLDER/second-level-folder" of the space "Shares" should not contain these entries:
      | second-level-folder |
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copy a folder into another folder at different level which is received as a group share
    Given using spaces DAV path
    And using SharingNG
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder/third-level-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/Sample-Folder-A"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b/sample-folder-c"
    When user "Alice" copies folder "/Sample-Folder-A/sample-folder-b" from space "Personal" to "/BRIAN-FOLDER/second-level-folder/third-level-folder" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" folder "/Sample-Folder-A/sample-folder-b/sample-folder-c" should exist
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder/third-level-folder" of the space "Shares" should contain these entries:
      | sample-folder-c |
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copy a file into a folder at different level received as a group share
    Given using spaces DAV path
    And using SharingNG
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder/third-level-folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/Sample-Folder-A"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "/Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "/Sample-Folder-A/sample-folder-b/textfile-c.txt" from space "Personal" to "/BRIAN-FOLDER/second-level-folder" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder" of the space "Shares" should not contain these entries:
      | third-level-folder |
    And for user "Alice" the content of the file "/BRIAN-FOLDER/second-level-folder" of the space "Shares" should be "sample file-c"
    And for user "Brian" the content of the file "/BRIAN-FOLDER/second-level-folder" of the space "Personal" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: overwrite a file received as a group share with a file from different level
    Given using spaces DAV path
    And using SharingNG
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "BRIAN-FOLDER"
    And user "Brian" has uploaded file with content "file at second level" to "/BRIAN-FOLDER/second-level-file.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/Sample-Folder-A"
    And user "Alice" has created folder "/Sample-Folder-A/sample-folder-b"
    And user "Alice" has uploaded file with content "sample file-c" to "/Sample-Folder-A/sample-folder-b/textfile-c.txt"
    When user "Alice" copies file "/Sample-Folder-A/sample-folder-b/textfile-c.txt" from space "Personal" to "/BRIAN-FOLDER/second-level-file.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And as "Alice" file "/Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And for user "Alice" folder "/BRIAN-FOLDER" of the space "Shares" should not contain these files:
      | /textfile-c.txt |
    And as "Alice" file "/Sample-Folder-A/sample-folder-b/textfile-c.txt" should exist
    And for user "Alice" the content of the file "/BRIAN-FOLDER/second-level-file.txt" of the space "Shares" should be "sample file-c"
    And for user "Brian" the content of the file "/BRIAN-FOLDER/second-level-file.txt" of the space "Personal" should be "sample file-c"
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copy a folder into a file at different level received as a group share
    Given using spaces DAV path
    And using SharingNG
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Brian" has created folder "/BRIAN-FOLDER"
    And user "Brian" has created folder "/BRIAN-FOLDER/second-level-folder"
    And user "Brian" has uploaded file with content "file at third level" to "/BRIAN-FOLDER/second-level-folder/third-level-file.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | BRIAN-FOLDER |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Alice" has a share "BRIAN-FOLDER" synced
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has created folder "/FOLDER/second-level-folder"
    And user "Alice" has created folder "/FOLDER/second-level-folder/third-level-folder"
    When user "Alice" copies folder "/FOLDER/second-level-folder" from space "Personal" to "/BRIAN-FOLDER/second-level-folder/third-level-file.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder" of the space "Shares" should contain these files:
      | third-level-file.txt |
    And for user "Alice" folder "BRIAN-FOLDER/second-level-folder/third-level-file.txt" of the space "Shares" should contain these files:
      | third-level-folder |
    And as "Alice" folder "FOLDER/second-level-folder/third-level-folder" should exist
    And for user "Alice" folder "BRIAN-FOLDER" of the space "Shares" should not contain these files:
      | /second-level-folder/second-level-folder |
    And as user "Alice" the last share should include the following properties:
      | file_target | /Shares/BRIAN-FOLDER |


  Scenario: copying a file with an option "keep both" inside of the project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/newfolder" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "some content" to "/newfolder/insideSpace.txt"
    And user "Alice" has uploaded a file inside space "Project" with content "new content" to "/insideSpace.txt"
    When user "Alice" copies file "/insideSpace.txt" to "/newfolder/insideSpace (1).txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "newfolder" of the space "Project" should contain these entries:
      | insideSpace.txt     |
      | insideSpace (1).txt |
    And for user "Alice" the content of the file "/newfolder/insideSpace (1).txt" of the space "Project" should be "new content"

  @owncloud-issue-4797
  Scenario: copying a file with an option "replace" inside of the project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/newfolder" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "old content version 1" to "/newfolder/insideSpace.txt"
    And user "Alice" has uploaded a file inside space "Project" with content "old content version 2" to "/newfolder/insideSpace.txt"
    And user "Alice" has uploaded a file inside space "Project" with content "new content" to "/insideSpace.txt"
    When user "Alice" overwrites file "/insideSpace.txt" from space "Project" to "/newfolder/insideSpace.txt" inside space "Project" while copying using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" the content of the file "/newfolder/insideSpace.txt" of the space "Project" should be "new content"
    When user "Alice" downloads version of the file "/newfolder/insideSpace.txt" with the index "2" of the space "Project" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "old content version 1"
    When user "Alice" downloads version of the file "/newfolder/insideSpace.txt" with the index "1" of the space "Project" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "old content version 2"
    And as "Alice" file "insideSpace.txt" should not exist in the trashbin of the space "Project"


  Scenario: copying a file from Personal to Shares with an option "keep both"
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/newfolder" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "some content" to "/newfolder/personal.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | newfolder |
      | space           | Project   |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Brian" has a share "newfolder" synced
    And user "Brian" has uploaded file with content "new content" to "/personal.txt"
    When user "Brian" copies file "/personal.txt" from space "Personal" to "/newfolder/personal (1).txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "newfolder" of the space "Project" should contain these entries:
      | personal.txt     |
      | personal (1).txt |
    And for user "Alice" the content of the file "/newfolder/personal (1).txt" of the space "Project" should be "new content"
    And for user "Brian" folder "newfolder" of the space "Shares" should contain these entries:
      | personal.txt     |
      | personal (1).txt |


  Scenario: copying a file from Personal to Shares with an option "replace"
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project" with the default quota using the Graph API
    And user "Alice" has created a folder "/newfolder" in space "Project"
    And user "Alice" has uploaded a file inside space "Project" with content "old content version 1" to "/newfolder/personal.txt"
    And user "Alice" has uploaded a file inside space "Project" with content "old content version 2" to "/newfolder/personal.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | newfolder |
      | space           | Project   |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Editor    |
    And user "Brian" has a share "newfolder" synced
    And user "Brian" has uploaded file with content "new content" to "/personal.txt"
    When user "Brian" overwrites file "/personal.txt" from space "Personal" to "/newfolder/personal.txt" inside space "Shares" while copying using the WebDAV API
    Then the HTTP status code should be "204"
    And for user "Alice" folder "newfolder" of the space "Project" should contain these entries:
      | personal.txt |
    And for user "Alice" the content of the file "/newfolder/personal.txt" of the space "Project" should be "new content"
    When user "Alice" downloads version of the file "/newfolder/personal.txt" with the index "1" of the space "Project" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "old content version 2"
    And for user "Brian" the content of the file "/newfolder/personal.txt" of the space "Shares" should be "new content"
    And as "Brian" file "insideSpace.txt" should not exist in the trashbin of the space "Personal"
