Feature: low level tests for upload of chunks
  As a user
  I want to be able to upload resources in chunks
  So that I can manage my resources

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: upload a chunk twice
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 10                    |
      #    ZmlsZS50eHQ= is the base64 encode of file.txt
      | Upload-Metadata | filename ZmlsZS50eHQ= |
    When user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "123" using the WebDAV API
    And user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "000" using the WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" file "file.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: finalize file upload after uploading a chunk twice
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 10                    |
      #    ZmlsZS50eHQ= is the base64 encode of file.txt
      | Upload-Metadata | filename ZmlsZS50eHQ= |
    When user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "123" using the WebDAV API
    And user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "000" using the WebDAV API
    And user "Alice" sends a chunk to the last created TUS Location with offset "3" and data "4567890" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/file.txt" for user "Alice" should be "1234567890"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: send last chunk twice
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 10                    |
      #    ZmlsZS50eHQ= is the base64 encode of file.txt
      | Upload-Metadata | filename ZmlsZS50eHQ= |
    When user "Alice" sends a chunk to the last created TUS Location with offset "0" and data "123" using the WebDAV API
    And user "Alice" sends a chunk to the last created TUS Location with offset "3" and data "4567890" using the WebDAV API
    And the user waits for "2" seconds for postprocessing to finish
    And user "Alice" sends a chunk to the last created TUS Location with offset "3" and data "0000000" using the WebDAV API
    Then the HTTP status code should be "404"
    And the content of file "/file.txt" for user "Alice" should be "1234567890"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: start with uploading not at the beginning of the file
    Given using <dav-path-version> DAV path
    And user "Alice" has created a new TUS resource on the WebDAV API with these headers:
      | Upload-Length   | 10                    |
      #    ZmlsZS50eHQ= is the base64 encode of file.txt
      | Upload-Metadata | filename ZmlsZS50eHQ= |
    When user "Alice" sends a chunk to the last created TUS Location with offset "1" and data "123" using the WebDAV API
    Then the HTTP status code should be "409"
    And as "Alice" file "file.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
