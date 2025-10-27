Feature: OPTIONS request
  As a user
  I want to check OPTIONS request
  So that I can get information about communication options for target resource

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: send OPTIONS request to webDav endpoints using the TUS protocol with valid password and username
    When user "Alice" requests these endpoints with "OPTIONS" including body "doesnotmatter" using the password of user "Alice"
      | endpoint               |
      | /webdav/               |
      | /dav/files/%username%/ |
      | /dav/spaces/%spaceid%/ |
    Then the HTTP status code should be "204"
    And the following headers should be set
      | header                 | value                                             |
      | Tus-Resumable          | 1.0.0                                             |
      | Tus-Version            | 1.0.0                                             |
      | Tus-Extension          | creation,creation-with-upload,checksum,expiration |
      | Tus-Checksum-Algorithm | md5,sha1,crc32                                    |


  Scenario: send OPTIONS request to webDav endpoints using the TUS protocol without any authentication
    When a user requests these endpoints with "OPTIONS" with body "doesnotmatter" and no authentication about user "Alice"
      | endpoint               |
      | /webdav/               |
      | /dav/files/%username%/ |
      | /dav/spaces/%spaceid%/ |
    Then the HTTP status code should be "204"
    And the following headers should be set
      | header                 | value                                             |
      | Tus-Resumable          | 1.0.0                                             |
      | Tus-Version            | 1.0.0                                             |
      | Tus-Extension          | creation,creation-with-upload,checksum,expiration |
      | Tus-Checksum-Algorithm | md5,sha1,crc32                                    |

  @owncloud-issue-1012
  Scenario: send OPTIONS request to webDav endpoints using the TUS protocol with valid username and wrong password
    When user "Alice" requests these endpoints with "OPTIONS" including body "doesnotmatter" using password "invalid" about user "Alice"
      | endpoint               |
      | /webdav/               |
      | /dav/files/%username%/ |
      | /dav/spaces/%spaceid%/ |
    Then the HTTP status code should be "204"
    And the following headers should be set
      | header                 | value                                             |
      | Tus-Resumable          | 1.0.0                                             |
      | Tus-Version            | 1.0.0                                             |
      | Tus-Extension          | creation,creation-with-upload,checksum,expiration |
      | Tus-Checksum-Algorithm | md5,sha1,crc32                                    |

  @owncloud-issue-1012
  Scenario: send OPTIONS requests to webDav endpoints using valid password and username of different user
    Given user "Brian" has been created with default attributes
    When user "Brian" requests these endpoints with "OPTIONS" including body "doesnotmatter" using the password of user "Alice"
      | endpoint               |
      | /webdav/               |
      | /dav/files/%username%/ |
      | /dav/spaces/%spaceid%/ |
    Then the HTTP status code should be "204"
    And the following headers should be set
      | header                 | value                                             |
      | Tus-Resumable          | 1.0.0                                             |
      | Tus-Version            | 1.0.0                                             |
      | Tus-Extension          | creation,creation-with-upload,checksum,expiration |
      | Tus-Checksum-Algorithm | md5,sha1,crc32                                    |
