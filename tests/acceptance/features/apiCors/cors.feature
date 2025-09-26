# NOTE: set env OC_CORS_ALLOW_ORIGINS=https://aphno.badal while running OpenCloud server
@env-config
Feature: CORS headers
  As a user
  I want to send a cross-origin request
  So that I can check if the correct headers are set

  Background:
    Given user "Alice" has been created with default attributes
    And the config "OC_CORS_ALLOW_ORIGINS" has been set to "https://aphno.badal"

  @issue-5195
  Scenario Outline: CORS headers should be returned when setting CORS domain sending origin header
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" sends HTTP method "GET" to OCS API endpoint "<endpoint>" with headers
      | header | value               |
      | Origin | https://aphno.badal |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "<http-status-code>"
    And the following headers should be set
      | header                           | value               |
      | Access-Control-Expose-Headers    | Location            |
      | Access-Control-Allow-Origin      | https://aphno.badal |
    Examples:
      | ocs-api-version | endpoint                          | ocs-status-code | http-status-code |
      | 1               | /config                           | 100             | 200              |
      | 2               | /config                           | 200             | 200              |
      | 1               | /apps/files_sharing/api/v1/shares | 100             | 200              |
      | 2               | /apps/files_sharing/api/v1/shares | 200             | 200              |


  Scenario Outline: CORS headers should not be returned when CORS domain does not match origin header
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" sends HTTP method "GET" to OCS API endpoint "<endpoint>" with headers
      | header | value              |
      | Origin | https://mero.badal |
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "<http-status-code>"
    And the following headers should not be set
      | header                        |
      | Access-Control-Allow-Headers  |
      | Access-Control-Expose-Headers |
      | Access-Control-Allow-Origin   |
      | Access-Control-Allow-Methods  |
    Examples:
      | ocs-api-version | endpoint                          | ocs-status-code | http-status-code |
      | 1               | /config                           | 100             | 200              |
      | 2               | /config                           | 200             | 200              |
      | 1               | /apps/files_sharing/api/v1/shares | 100             | 200              |
      | 2               | /apps/files_sharing/api/v1/shares | 200             | 200              |

  @issue-5194
  # The Access-Control-Request-Headers need to be in lower-case and alphabetically order to comply with the rs/cors
  # package see: https://github.com/rs/cors/commit/4c32059b2756926619f6bf70281b91be7b5dddb2#diff-bf80d8fbedf172fab9ba2604da7f7be972e48b2f78a8d0cd21619d5f93665895R367
  Scenario Outline: CORS headers should be returned when an preflight request is sent
    Given using OCS API version "<ocs-api-version>"
    When user "Alice" sends HTTP method "OPTIONS" to OCS API endpoint "<endpoint>" with headers
      | header                         | value                                                                                                                                                                                                                                                                                                                                                 |
      | Origin                         | https://aphno.badal                                                                                                                                                                                                                                                                                                                                   |
      | Access-Control-Request-Headers | accept,authorization,cache-control,content-type,depth,destination,if-match,if-none-match,ocs-apirequest,origin,overwrite,tus-checksum-algorithm,tus-resumable,upload-checksum,upload-concat,upload-defer-length,upload-expires,upload-length,upload-metadata,upload-offset,x-http-method-override,x-request-id,x-requested-with |
      | Access-Control-Request-Method  | <request-method>                                                                                                                                                                                                                                                                                                                                      |
    And the HTTP status code should be "204"
    And the following headers should be set
      | header                       | value                                                                                                                                                                                                                                                                                                                                                 |
      | Access-Control-Allow-Headers | accept,authorization,cache-control,content-type,depth,destination,if-match,if-none-match,ocs-apirequest,origin,overwrite,tus-checksum-algorithm,tus-resumable,upload-checksum,upload-concat,upload-defer-length,upload-expires,upload-length,upload-metadata,upload-offset,x-http-method-override,x-request-id,x-requested-with |
      | Access-Control-Allow-Origin  | https://aphno.badal                                                                                                                                                                                                                                                                                                                                   |
      | Access-Control-Allow-Methods | <request-method>                                                                                                                                                                                                                                                                                                                                      |
    Examples:
      | ocs-api-version | endpoint                          | request-method |
      | 1               | /apps/files_sharing/api/v1/shares | GET            |
      | 2               | /apps/files_sharing/api/v1/shares | PUT            |
      | 1               | /apps/files_sharing/api/v1/shares | DELETE         |
      | 2               | /apps/files_sharing/api/v1/shares | POST           |


  Scenario: CORS headers should be returned when setting CORS domain sending origin header in the Graph api
    When user "Alice" lists all available spaces with headers using the Graph API
      | header | value               |
      | Origin | https://aphno.badal |
    Then the HTTP status code should be "200"
    And the following headers should be set
      | header                      | value               |
      | Access-Control-Allow-Origin | https://aphno.badal |

  @issue-8231
  Scenario Outline: CORS headers should be returned when setting CORS domain sending origin header in the Webdav api
    Given using <dav-path-version> DAV path
    When user "Alice" sends PROPFIND request to space "Personal" with headers using the WebDAV API
      | header | value               |
      | Origin | https://aphno.badal |
    Then the HTTP status code should be "207"
    And the following headers should be set
      | header                      | value               |
      | Access-Control-Allow-Origin | https://aphno.badal |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario: CORS headers should be returned when setting CORS domain sending origin header in the settings api
    When user "Alice" lists values-list with headers using the Settings API
      | header | value               |
      | Origin | https://aphno.badal |
    Then the HTTP status code should be "201"
    And the following headers should be set
      | header                      | value               |
      | Access-Control-Allow-Origin | https://aphno.badal |

  @issue-8380
  Scenario: CORS headers should be returned when uploading file using Tus and when CORS domain sending origin header in the Webdav api
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
      | Origin          | https://aphno.badal       |
    When user "Alice" sends a chunk to the last created TUS Location with data "01234" with the following headers:
      | Origin          | https://aphno.badal                  |
      | Upload-Checksum | MD5 4100c4d44da9177247e44a5fc1546778 |
      | Upload-Offset   | 0                                    |
    Then the HTTP status code should be "204"
    And the following headers should be set
      | header                      | value               |
      | Access-Control-Allow-Origin | https://aphno.badal |
    And for user "Alice" the content of the file "/textFile.txt" of the space "Personal" should be "01234"

  @issue-8380
  Scenario: uploading file using Tus using different CORS headers
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
      | Origin          | https://something.else    |
    When user "Alice" sends a chunk to the last created TUS Location with data "01234" with the following headers:
      | Origin          | https://something.else               |
      | Upload-Checksum | MD5 4100c4d44da9177247e44a5fc1546778 |
      | Upload-Offset   | 0                                    |
    Then the HTTP status code should be "403"

  @issue-8380
  # The Access-Control-Request-Headers need to be in lower-case and alphabetically order to comply with the rs/cors
  # package see: https://github.com/rs/cors/commit/4c32059b2756926619f6bf70281b91be7b5dddb2#diff-bf80d8fbedf172fab9ba2604da7f7be972e48b2f78a8d0cd21619d5f93665895R367
  Scenario Outline: CORS headers should be returned when an preflight request is sent to Tus upload
    Given user "Alice" has created a new TUS resource in the space "Personal" with the following headers:
      | Upload-Length   | 5                         |
      #    dGV4dEZpbGUudHh0 is the base64 encode of textFile.txt
      | Upload-Metadata | filename dGV4dEZpbGUudHh0 |
      | Tus-Resumable   | 1.0.0                     |
    When user "Alice" sends HTTP method "OPTIONS" to URL "<endpoint>" with headers
      | header                         | value                                                                                                                                                                                                                                                                                                                                                 |
      | Origin                         | https://aphno.badal                                                                                                                                                                                                                                                                                                                                   |
      | Access-Control-Request-Headers | accept,authorization,cache-control,content-type,depth,destination,if-match,if-none-match,ocs-apirequest,origin,overwrite,tus-checksum-algorithm,tus-resumable,upload-checksum,upload-concat,upload-defer-length,upload-expires,upload-length,upload-metadata,upload-offset,x-http-method-override,x-request-id,x-requested-with |
      | Access-Control-Request-Method  | <request-method>                                                                                                                                                                                                                                                                                                                                      |
    And the HTTP status code should be "204"
    And the following headers should be set
      | header                       | value                                                                                                                                                                                                                                                                                                                                                 |
      | Access-Control-Allow-Headers | accept,authorization,cache-control,content-type,depth,destination,if-match,if-none-match,ocs-apirequest,origin,overwrite,tus-checksum-algorithm,tus-resumable,upload-checksum,upload-concat,upload-defer-length,upload-expires,upload-length,upload-metadata,upload-offset,x-http-method-override,x-request-id,x-requested-with |
      | Access-Control-Allow-Origin  | https://aphno.badal                                                                                                                                                                                                                                                                                                                                   |
      | Access-Control-Allow-Methods | <request-method>                                                                                                                                                                                                                                                                                                                                      |
    Examples:
      | endpoint               | request-method |
      | /%tus_upload_location% | PUT            |
      | /%tus_upload_location% | POST           |
      | /%tus_upload_location% | HEAD           |
      | /%tus_upload_location% | PATCH          |
