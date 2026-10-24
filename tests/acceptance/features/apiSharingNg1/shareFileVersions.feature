Feature: file versions of a shared resource
  As a user
  I want to access versions of a resource shared with me
  So that I can view and manage its history according to my role

  # These scenarios address the shared file the same way a web client does: by the
  # Shares-mount id resolved from the sharee's shared-with-me list (not by the owner's
  # storage resource id), so that regressions of the sharee code path are caught.

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "hello world version 1" to "text.txt"
    And user "Alice" has uploaded file with content "hello world version 1.1" to "text.txt"

  @env-config
  Scenario Outline: sharee can view file versions of a shared file while upload and restore depend on the role
    Given the administrator has enabled the permissions role "<role>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | text.txt |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | <role>   |
    And user "Brian" has a share "text.txt" synced
    When user "Brian" gets the number of versions of shared resource "text.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" uploads file with content "shared file new version" to "/Shares/text.txt" using the WebDAV API
    Then the HTTP status code should be "<upload-code>"
    Examples:
      | role                      | upload-code |
      | Viewer With Versions      | 403         |
      | File Editor With Versions | 204         |

  @env-config @issue-3168
  Scenario Outline: sharee tries to restore file version
    Given the administrator has enabled the permissions role "<role>"
    And user "Alice" has sent the following resource share invitation:
      | resource        | text.txt |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | <role>   |
    And user "Brian" has a share "text.txt" synced
    When user "Brian" gets the number of versions of shared resource "text.txt"
    Then the HTTP status code should be "207"
    And the number of versions should be "1"
    When user "Brian" restores version index "1" of shared resource "text.txt"
    Then the HTTP status code should be "<restore-code>"
    Examples:
      | role                      | restore-code |
      | Viewer With Versions      | 403          |
      | File Editor With Versions | 204          |
