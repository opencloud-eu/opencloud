Feature: colon-syntax path lookup on the Graph API
  As a client
  I want to address drive items by path using colon-syntax URLs on the Graph API
  So that I do not have to walk the path with successive lookups before issuing a request

  The colon-syntax shapes recognised by the path-lookup middleware are:

    /graph/{version}/drives/{driveID}/root:/<path>[:/<suffix>][:]
    /graph/{version}/drives/{driveID}/items/{itemID}:/<relativePath>[:/<suffix>][:]

  Both /v1.0 and /v1beta1 are supported. NOT_FOUND and PERMISSION_DENIED
  collapse to 404 so the middleware never discloses the existence of
  resources the caller is not allowed to see.

  Clients must percent-encode each path segment, exactly as MS Graph requires.
  The only OpenCloud-specific point: ':' must be encoded as "%3A" (the split is
  on the literal ":/", so an encoded ':' is never a delimiter). OpenCloud allows
  ':' in names whereas MS Graph/OneDrive forbid it, so "%3A" is the one extra
  character to encode.

  Background:
    Given user "Alice" has been created with default attributes
    And user "Alice" has created folder "folder1"
    And user "Alice" has created folder "folder1/sub"
    And user "Alice" has uploaded file with content "hello" to "folder1/file.txt"
    And user "Alice" has uploaded file with content "deep" to "folder1/sub/deep.txt"


  Scenario: get a drive item by root-anchored colon path
    When user "Alice" gets the drive item with colon path "folder1/file.txt" of space "Personal" using the Graph API version "v1.0"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["id", "name", "parentReference"],
        "properties": {
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "name": {
            "const": "file.txt"
          }
        }
      }
      """


  Scenario: get a drive item by root-anchored colon path with a deep path
    When user "Alice" gets the drive item with colon path "folder1/sub/deep.txt" of space "Personal" using the Graph API version "v1.0"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["id", "name"],
        "properties": {
          "name": {
            "const": "deep.txt"
          }
        }
      }
      """


  Scenario: get a drive item by root-anchored colon path with a trailing colon
    When user "Alice" gets the drive item with colon path "folder1/file.txt" of space "Personal" with trailing colon using the Graph API version "v1.0"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["id", "name"],
        "properties": {
          "name": {
            "const": "file.txt"
          }
        }
      }
      """


  Scenario: get a drive item by item-anchored colon path
    When user "Alice" gets the drive item with colon path "file.txt" relative to folder "folder1" of space "Personal" using the Graph API version "v1.0"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["id", "name"],
        "properties": {
          "name": {
            "const": "file.txt"
          }
        }
      }
      """


  Scenario: list permissions of a drive item via colon path with a sub-route suffix on v1beta1
    # Exercises the "/<path>:/<suffix>" rewrite shape and, at the same time,
    # the v1beta1 mount of the middleware. The /permissions sub-route is only
    # registered at /v1beta1/, and the canonical /v1beta1 GetDriveItem
    # handler is share-jail-only, so this is the cleanest way to assert that
    # the v1beta1 colon-syntax path actually reaches a working handler for
    # regular drive items.
    When user "Alice" lists permissions of the drive item with colon path "folder1" of space "Personal" using the Graph API version "v1beta1"
    Then the HTTP status code should be "200"


  Scenario: non-existent colon path returns 404
    When user "Alice" gets the drive item with colon path "folder1/does-not-exist.txt" of space "Personal" using the Graph API version "v1.0"
    Then the HTTP status code should be "404"


  Scenario: another user cannot disclose existence of a resource via colon path
    Given user "Brian" has been created with default attributes
    When user "Brian" gets the drive item with colon path "folder1/file.txt" of the personal space of "Alice" using the Graph API version "v1.0"
    Then the HTTP status code should be "404"
