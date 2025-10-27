Feature: PROPFIND
  As a user
  I want to retrieve all properties of a resource
  So that I can get the information about a resource

  @owncloud-issue-751
  Scenario Outline: send PROPFIND request to "/dav/(files|spaces)"
    Given user "Alice" has been created with default attributes
    When user "Alice" requests "<dav-path>" with "PROPFIND" using basic auth
    Then the HTTP status code should be "405"
    Examples:
      | dav-path    |
      | /dav/files  |
      | /dav/spaces |

  @owncloud-issue-10334
  Scenario Outline: send PROPFIND request to "/dav/(files|spaces)" with depth header
    Given user "Alice" has been created with default attributes
    When user "Alice" requests "<dav-path>" with "PROPFIND" using basic auth and with headers
      | header | value   |
      | depth  | <depth> |
    Then the HTTP status code should be "<http-status-code>"
    Examples:
      | dav-path              | depth    | http-status-code |
      | /webdav               | 0        | 207              |
      | /webdav               | 1        | 207              |
      | /dav/files/alice      | 0        | 207              |
      | /dav/files/alice      | 1        | 207              |
      | /dav/spaces/%spaceid% | 0        | 207              |
      | /dav/spaces/%spaceid% | 1        | 207              |
      | /dav/spaces/%spaceid% | infinity | 400              |

    @skipOnReva
    Examples:
      | dav-path         | depth    | http-status-code |
      | /webdav          | infinity | 400              |
      | /dav/files/alice | infinity | 400              |

  @skipOnReva @owncloud-issue-10071 @owncloud-issue-10331
  Scenario: send PROPFIND request to a public link shared with password
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/PARENT"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When the public sends "PROPFIND" request to the last public link share using the public WebDAV API with password "%public%"
    Then the HTTP status code should be "207"
    And the value of the item "//d:href" in the response should match "/\/dav\/public-files\/%public_token%\/$/"
    And the value of the item "//oc:public-link-share-owner" in the response should be "Alice"

  @skipOnReva @owncloud-issue-10071 @owncloud-issue-10331
  Scenario: send PROPFIND request to a public link shared with password (request without password)
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/PARENT"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When the public sends "PROPFIND" request to the last public link share using the public WebDAV API
    Then the HTTP status code should be "401"
    And the value of the item "/d:error/s:exception" in the response should be "Sabre\DAV\Exception\NotAuthenticated"

  @skipOnReva @owncloud-issue-10071 @owncloud-issue-10331
  Scenario: send PROPFIND request to a public link shared with password (request with incorrect password)
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "/PARENT"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | PARENT   |
      | space           | Personal |
      | permissionsRole | view     |
      | password        | %public% |
    When the public sends "PROPFIND" request to the last public link share using the public WebDAV API with password "1234"
    Then the HTTP status code should be "401"
    And the value of the item "/d:error/s:exception" in the response should be "Sabre\DAV\Exception\NotAuthenticated"
