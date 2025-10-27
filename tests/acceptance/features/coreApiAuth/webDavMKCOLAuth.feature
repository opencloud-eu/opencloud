Feature: create folder using MKCOL
  As a user
  I want to create folders
  So that I can organise resources in folders

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/parent.txt"

  @smokeTest
  Scenario: send MKCOL requests to webDav endpoints as normal user with wrong password
    When user "Alice" requests these endpoints with "MKCOL" including body "doesnotmatter" using password "invalid" about user "Alice"
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
  Scenario: send MKCOL requests to webDav endpoints as normal user with no password
    When user "Alice" requests these endpoints with "MKCOL" including body "doesnotmatter" using password "" about user "Alice"
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

  @owncloud-issue-5049 @owncloud-issue-1347 @owncloud-issue-1292
  Scenario: send MKCOL requests to another user's webDav endpoints as normal user
    Given user "Brian" has been created with default attributes
    When user "Brian" requests these endpoints with "MKCOL" including body "" about user "Alice"
      | endpoint                                |
      | /dav/files/%username%/textfile0.txt     |
      | /dav/files/%username%/PARENT            |
      | /dav/files/%username%/does-not-exist    |
      | /dav/files/%username%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-5049 @owncloud-issue-1347 @owncloud-issue-1292
  Scenario: send MKCOL requests to non-existent user's webDav endpoints as normal user
    Given user "Brian" has been created with default attributes
    When user "Brian" requests these endpoints with "MKCOL" including body "" about user "non-existent-user"
      | endpoint                                       |
      | /dav/files/non-existent-user/textfile0.txt     |
      | /dav/files/non-existent-user/PARENT            |
      | /dav/files/non-existent-user/does-not-exist    |
      | /dav/files/non-existent-user/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-1347 @owncloud-issue-1292
  Scenario: send MKCOL requests to another user's webDav endpoints as normal user using the spaces WebDAV API
    Given user "Brian" has been created with default attributes
    When user "Brian" requests these endpoints with "MKCOL" including body "" about user "Alice"
      | endpoint                                |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/does-not-exist    |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-5049 @owncloud-issue-1347 @owncloud-issue-1292
  Scenario: send MKCOL requests to non-existent user's webDav endpoints as normal user using the spaces WebDAV API
    Given user "Brian" has been created with default attributes
    When user "Brian" requests these endpoints with "MKCOL" including body "" about user "non-existent-user"
      | endpoint                                |
      | /dav/spaces/%spaceid%/textfile0.txt     |
      | /dav/spaces/%spaceid%/PARENT            |
      | /dav/spaces/%spaceid%/does-not-exist    |
      | /dav/spaces/%spaceid%/PARENT/parent.txt |
    Then the HTTP status code of responses on all endpoints should be "404"


  Scenario: send MKCOL requests to webDav endpoints using invalid username but correct password
    When user "usero" requests these endpoints with "MKCOL" including body "doesnotmatter" using the password of user "Alice"
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


  Scenario: send MKCOL requests to webDav endpoints using valid password and username of different user
    Given user "Brian" has been created with default attributes
    When user "Brian" requests these endpoints with "MKCOL" including body "doesnotmatter" using the password of user "Alice"
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
  Scenario: send MKCOL requests to webDav endpoints without any authentication
    When a user requests these endpoints with "MKCOL" with body "doesnotmatter" and no authentication about user "Alice"
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
