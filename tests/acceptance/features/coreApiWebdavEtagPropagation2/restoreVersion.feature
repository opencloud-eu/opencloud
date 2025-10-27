@owncloud-issue-4251
Feature: propagation of etags when restoring a version of a file
  As a client app
  I want metadata (etags) of parent folders to change when a file version is restored
  So that the client app can know to re-scan and sync the content of the folder(s)

  Background:
    Given using OCS API version "2"
    And user "Alice" has been created with default attributes

  @skipOnStorage:ceph @skipOnStorage:scality
  Scenario: restoring a file changes the etags of all parents
    Given user "Alice" has created folder "/upload"
    And user "Alice" has created folder "/upload/sub"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/file.txt"
    And user "Alice" has uploaded file with content "changed content" to "/upload/sub/file.txt"
    And user "Alice" has stored etag of element "/"
    And user "Alice" has stored etag of element "/upload"
    And user "Alice" has stored etag of element "/upload/sub"
    When user "Alice" restores version index "1" of file "/upload/sub/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed:
      | user  | path        |
      | Alice | /           |
      | Alice | /upload     |
      | Alice | /upload/sub |
