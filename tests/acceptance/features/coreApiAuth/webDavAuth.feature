Feature: auth
  As a user
  I want to check the authentication of the application
  So that I can make sure it's secure

  Background:
    Given user "Alice" has been created with default attributes

  @smokeTest @owncloud-issue-10334
  Scenario Outline: using WebDAV anonymously
    When a user requests "<dav-path>" with "PROPFIND" and no authentication
    Then the HTTP status code should be "401"
    Examples:
      | dav-path              |
      | /webdav               |
      | /dav/files/%username% |
      | /dav/spaces/%spaceid% |

  @smokeTest @owncloud-issue-10334
  Scenario Outline: using WebDAV with basic auth
    When user "Alice" requests "<dav-path>" with "PROPFIND" using basic auth
    Then the HTTP status code should be "207"
    Examples:
      | dav-path              |
      | /webdav               |
      | /dav/files/%username% |
      | /dav/spaces/%spaceid% |
