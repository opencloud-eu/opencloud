Feature: audit log
  As an administrator
  I want every relevant action to be recorded in the audit log
  So that I can find out who did what and when

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And using SharingNG


  Scenario: uploading and downloading a file is recorded
    When user "Alice" uploads file with content "audit" to "/audit.txt" using the WebDAV API
    And user "Alice" downloads file "/audit.txt" using the WebDAV API
    Then there should be an audit record "file.upload" with these fields:
      | field       | value       |
      | outcome     | success     |
      | actor.id    | Alice       |
      | target.type | file        |
      | target.path | ./audit.txt |
    And there should be an audit record "file.download" with these fields:
      | field       | value       |
      | outcome     | success     |
      | actor.id    | Alice       |
      | target.path | ./audit.txt |


  Scenario: a login with a wrong password is recorded as failure
    When user "Alice" requests these endpoints with "PROPFIND" using password "invalid" about user "Alice"
      | endpoint                  |
      | /dav/files/%username%/    |
    Then there should be an audit record "auth.login" with these fields:
      | field          | value               |
      | outcome        | failure             |
      | outcome.reason | invalid_credentials |
      | actor.name     | Alice               |


  Scenario: creating, declining and accepting a share is recorded, hiding it is not
    Given user "Alice" has created folder "folderToShare"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    Then there should be an audit record "share.create" with outcome "success" by user "Alice"
    When user "Brian" hides the shared resource "folderToShare" using the Graph API
    Then there should be no audit record "share.update"
    And there should be no audit record "share.decline"
    When user "Brian" disables sync of share "folderToShare" using the Graph API
    Then there should be "1" audit records "share.decline" with these fields:
      | field    | value   |
      | outcome  | success |
      | actor.id | Brian   |
    When user "Brian" enables sync of share "folderToShare" offered by "Alice" from "Personal" space using the Graph API
    Then there should be "1" audit records "share.accept" with these fields:
      | field    | value   |
      | outcome  | success |
      | actor.id | Brian   |


  Scenario: every wrong public link password is recorded, a successful access once per client
    Given user "Alice" has created folder "linkFolder"
    And user "Alice" has uploaded file with content "audit" to "/linkFolder/test.txt"
    And user "Alice" has created the following resource link share:
      | resource        | linkFolder |
      | space           | Personal   |
      | permissionsRole | view       |
      | password        | %public%   |
    When the public downloads file "/test.txt" from inside the last public link shared folder with password "wrong" using the public WebDAV API
    And the public downloads file "/test.txt" from inside the last public link shared folder with password "wrong" using the public WebDAV API
    And the public downloads file "/test.txt" from inside the last public link shared folder with password "%public%" using the public WebDAV API
    And the public downloads file "/test.txt" from inside the last public link shared folder with password "%public%" using the public WebDAV API
    Then there should be "2" audit records "public_link.access" with these fields:
      | field   | value  |
      | outcome | denied |
    And there should be "1" audit records "public_link.access" with these fields:
      | field   | value   |
      | outcome | success |


  Scenario: adding members with a PATCH and renaming a group is recorded
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a group "grp1" using the Graph API
    When the administrator "Alice" adds the following users to a group "grp1" at once using the Graph API
      | username |
      | Brian    |
    Then there should be an audit record "group.member.add" with these fields:
      | field    | value   |
      | outcome  | success |
      | actor.id | Alice   |
    When user "Alice" renames group "grp1" to "grp2" using the Graph API
    Then there should be an audit record "group.update" with these fields:
      | field                    | value   |
      | outcome                  | success |
      | actor.id                 | Alice   |
      | target.name              | grp1    |
      | change.after.displayname | grp2    |
