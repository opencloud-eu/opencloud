Feature: MOVE file/folder
  As a user
  I want to move resources
  So that I can organise resources according to my preference

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/parent.txt"

  @smokeTest
  Scenario: send MOVE requests to webDav endpoints as normal user with wrong password
    When user "Alice" requests these endpoints with "MOVE" using password "invalid" about user "Alice"
      | endpoint                                |
      | /webdav/textfile0.txt                   |
      | /dav/files/%username%/textfile0.txt     |
      | /webdav/PARENT                          |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "401"

  @smokeTest
  Scenario: send MOVE requests to webDav endpoints as normal user with no password
    When user "Alice" requests these endpoints with "MOVE" using password "" about user "Alice"
      | endpoint                                |
      | /webdav/textfile0.txt                   |
      | /dav/files/%username%/textfile0.txt     |
      | /webdav/PARENT                          |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "401"

  @owncloud-issue-3882
  Scenario: send MOVE requests to another user's webDav endpoints as normal user
    When user "Brian" requests these endpoints with "MOVE" about user "Alice"
      | endpoint                                |
      | /dav/files/%username%/textfile0.txt     |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-3882
  Scenario: send MOVE requests to another user's webDav endpoints as normal user using the spaces WebDAV API
    Given using spaces DAV path
    When user "Brian" requests these endpoints with "MOVE" about user "Alice"
      | endpoint                                |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"


  Scenario: send MOVE requests to webDav endpoints using invalid username but correct password
    When user "usero" requests these endpoints with "MOVE" using the password of user "Alice"
      | endpoint                                |
      | /webdav/textfile0.txt                   |
      | /dav/files/%username%/textfile0.txt     |
      | /webdav/PARENT                          |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "401"


  Scenario: send MOVE requests to webDav endpoints using valid password and username of different user
    When user "Brian" requests these endpoints with "MOVE" using the password of user "Alice"
      | endpoint                                |
      | /webdav/textfile0.txt                   |
      | /dav/files/%username%/textfile0.txt     |
      | /webdav/PARENT                          |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "401"

  @smokeTest
  Scenario: send MOVE requests to webDav endpoints without any authentication
    When a user requests these endpoints with "MOVE" with no authentication about user "Alice"
      | endpoint                                |
      | /webdav/textfile0.txt                   |
      | /dav/files/%username%/textfile0.txt     |
      | /webdav/PARENT                          |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "401"

  @owncloud-issue-4332 @owncloud-issue-3882
  Scenario: send MOVE requests to webDav endpoints with body as normal user
    When user "Alice" requests these endpoints with "MOVE" including body "doesnotmatter" about user "Alice"
      | endpoint                                |
      | /webdav/textfile0.txt                   |
      | /dav/files/%username%/textfile0.txt     |
      | /webdav/PARENT                          |
      | /dav/files/%username%/PARENT            |
      | /webdav/PARENT/parent.txt               |
      | /dav/files/%username%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "415"

  @owncloud-issue-4332 @owncloud-issue-3882
  Scenario: send MOVE requests to webDav endpoints with body as normal user using the spaces WebDAV API
    When user "Alice" requests these endpoints with "MOVE" including body "doesnotmatter" about user "Alice"
      | endpoint                                |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "415"
