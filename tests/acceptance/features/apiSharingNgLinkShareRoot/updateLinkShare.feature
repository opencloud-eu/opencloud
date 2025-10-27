Feature: Update a link share for a resource
  https://docs.opencloud.eu/libre-graph-api/#/drives.permissions/CreateLink

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |

  @env-config @owncloud-issue-9724 @owncloud-issue-10331
  Scenario: set password on a existing link share of a project-space drive using root endpoint
    Given the following configs have been set:
      | config                                       | value |
      | OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD | false |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Alice" has created the following space link share:
      | space           | projectSpace |
      | permissionsRole | view         |
    When user "Alice" sets the following password for the last space link share using root endpoint of the Graph API:
      | space    | projectSpace |
      | password | %public%     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          }
        }
      }
      """
    And the public should be able to download file "textfile.txt" from the last link share with password "%public%" and the content should be "to share"

  @owncloud-issue-9724 @owncloud-issue-10331
  Scenario: update password on a existing link share of a project-space drive using root endpoint
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Alice" has created the following space link share:
      | space           | projectSpace |
      | permissionsRole | view         |
      | password        | $heLlo*1234* |
    When user "Alice" sets the following password for the last space link share using root endpoint of the Graph API:
      | space    | projectSpace |
      | password | %public%     |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "hasPassword"
        ],
        "properties": {
          "hasPassword": {
            "const": true
          }
        }
      }
      """
    And the public should be able to download file "textfile.txt" from the last link share with password "%public%" and the content should be "to share"
    And the public download of file "textfile.txt" from the last link share with password "$heLlo*1234*" should fail with HTTP status code "401" using shareNg
