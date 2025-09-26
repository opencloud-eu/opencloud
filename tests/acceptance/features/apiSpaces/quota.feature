Feature: State of the quota
  As a user
  I want to be able to see the state of the quota
  So that I will not let the quota overrun


  quota state indication:
  | 0 - 75%  | normal   |
  | 76 - 90% | nearing  |
  | 91 - 99% | critical |
  | 100 %    | exceeded |

  Background:
    Given user "Alice" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path


  Scenario Outline: quota information is returned in the list of spaces returned via the Graph API
    Given user "Alice" has created a space "<space-name>" of type "project" with quota "<quota>"
    When user "Alice" uploads a file inside space "<space-name>" with content "<file-content>" to "test.txt" using the WebDAV API
    And user "Alice" lists all available spaces via the Graph API
    Then the JSON response should contain space called "<space-name>" and match
      """
       {
        "type": "object",
        "required": [
          "name",
          "quota"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["<space-name>"]
          },
          "quota": {
            "type": "object",
            "required": [
              "state",
              "total",
              "remaining",
              "used"
            ],
            "properties": {
              "state" : {
                "type": "string",
                "enum": ["<state>"]
              },
              "total" : {
                "type": "number",
                "enum": [<quota>]
              },
              "remaining" : {
                "type": "number",
                "enum": [<remaining>]
              },
              "used": {
                "type": "number",
                "enum": [<used>]
              }
            }
          }
        }
      }
      """
    Examples:
      | space-name | file-content                                                                                         | state    | remaining     | used | quota         |
      | Quota1%    | 1                                                                                                    | normal   | 99            | 1    | 100           |
      | Quota75%   | 123456789 123456789 123456789 123456789 123456789 123456789 123456789 12345                          | normal   | 25            | 75   | 100           |
      | Quota76%   | 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456                         | nearing  | 24            | 76   | 100           |
      | Quota90%   | 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 1234567890           | nearing  | 10            | 90   | 100           |
      | Quota91%   | 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 1          | critical | 9             | 91   | 100           |
      | Quota99%   | 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789  | critical | 1             | 99   | 100           |
      | Quota100%  | 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789 1234567890 | exceeded | 0             | 100  | 100           |
      | Quota1     | 0                                                                                                    | normal   | 2499999999999 |    1 | 2500000000000 |


  Scenario: file cannot be uploaded if there is insufficient quota
    Given user "Alice" has created a space "Project Alfa" of type "project" with quota "10"
    When user "Alice" uploads a file inside space "Project Alfa" with content "More than 10 bytes" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "507"


  Scenario: folder can be created even if there is insufficient quota for file content
    Given user "Alice" has created a space "Project Beta" of type "project" with quota "7"
    And user "Alice" has uploaded a file inside space "Project Beta" with content "7 bytes" to "test.txt"
    When user "Alice" creates a folder "NewFolder" in space "Project Beta" using the WebDav Api
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project Beta" should contain these entries:
      | NewFolder |


  Scenario: file can be overwritten if there is enough quota
    Given user "Alice" has created a space "Project Gamma" of type "project" with quota "10"
    And user "Alice" has uploaded a file inside space "Project Gamma" with content "7 bytes" to "test.txt"
    When user "Alice" uploads a file inside space "Project Gamma" with content "0010 bytes" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "204"


  Scenario: file cannot be overwritten if there is insufficient quota
    Given user "Alice" has created a space "Project Delta" of type "project" with quota "10"
    And user "Alice" has uploaded a file inside space "Project Delta" with content "7 bytes" to "test.txt"
    When user "Alice" uploads a file inside space "Project Delta" with content "00011 bytes" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "507"


  Scenario Outline: check the relative amount of quota of personal space
    Given user "Admin" has changed the quota of the personal space of user "Alice" space to "10000"
    And user "Alice" has uploaded file "<file-upload>" to "/demo.txt"
    When the user "Alice" requests these endpoints with "GET" with basic auth
      | endpoint    |
      | <end-point> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "<ocs-status-code>"
    And the relative quota amount should be "<quota-relative>"
    Examples:
      | file-upload                   | end-point                          | ocs-status-code | quota-relative |
      | /filesForUpload/lorem.txt     | /ocs/v1.php/cloud/users/%username% | 100             | 6.99           |
      | /filesForUpload/lorem-big.txt | /ocs/v1.php/cloud/users/%username% | 100             | 91.17          |
      | /filesForUpload/lorem.txt     | /ocs/v2.php/cloud/users/%username% | 200             | 6.99           |
      | /filesForUpload/lorem-big.txt | /ocs/v2.php/cloud/users/%username% | 200             | 91.17          |

  @env-config
  Scenario: upload a file by setting OpenCloud spaces max quota
    Given the config "OC_SPACES_MAX_QUOTA" has been set to "10"
    And user "Brian" has been created with default attributes
    When user "Brian" uploads file with content "more than 10 bytes content" to "lorem.txt" using the WebDAV API
    Then the HTTP status code should be "507"

  @env-config
  Scenario: try to create a space with quota greater than OpenCloud spaces max quota
    Given the config "OC_SPACES_MAX_QUOTA" has been set to "50"
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    When user "Brian" tries to create a space "new space" of type "project" with quota "51" using the Graph API
    Then the HTTP status code should be "400"
    And the user "Brian" should not have a space called "new space"


  Scenario: user can restore a file version even if there is not enough quota to do so
    Given user "Admin" has changed the quota of the personal space of user "Alice" space to "30"
    And user "Alice" has uploaded file with content "file is less than 30 bytes" to "/file.txt"
    And user "Alice" has uploaded file with content "reduceContent" to "/file.txt"
    And user "Alice" has uploaded file with content "some content" to "newFile.txt"
    When user "Alice" restores version index "1" of file "/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/file.txt" for user "Alice" should be "file is less than 30 bytes"
