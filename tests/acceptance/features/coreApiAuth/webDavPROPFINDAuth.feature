Feature: PROPFIND file/folder
  As a user
  I want to send PROPFIND request to various endpoints
  So that I can ensure the endpoints are well authenticated

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
  Scenario: send PROPFIND requests to webDav endpoints as normal user with wrong password
    When user "Alice" requests these endpoints with "PROPFIND" including body "doesnotmatter" using password "invalid" about user "Alice"
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
  Scenario: send PROPFIND requests to webDav endpoints as normal user with no password
    When user "Alice" requests these endpoints with "PROPFIND" including body "doesnotmatter" using password "" about user "Alice"
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

  @owncloud-issue-1347
  Scenario: send PROPFIND requests to another user's webDav endpoints as normal user
    When user "Brian" requests these endpoints with "PROPFIND" to get property "d:getetag" about user "Alice"
      | endpoint                                |
      | /dav/files/%username%/textfile0.txt     |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-1347
  Scenario: send PROPFIND requests to another user's webDav endpoints as normal user using the spaces WebDAV API
    When user "Brian" requests these endpoints with "PROPFIND" to get property "d:getetag" about user "Alice"
      | endpoint                                |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"


  Scenario: send PROPFIND requests to webDav endpoints using invalid username but correct password
    When user "usero" requests these endpoints with "PROPFIND" including body "doesnotmatter" using the password of user "Alice"
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


  Scenario: send PROPFIND requests to webDav endpoints using valid password and username of different user
    When user "Brian" requests these endpoints with "PROPFIND" including body "doesnotmatter" using the password of user "Alice"
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
  Scenario: send PROPFIND requests to webDav endpoints without any authentication
    When a user requests these endpoints with "PROPFIND" with body "doesnotmatter" and no authentication about user "Alice"
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
