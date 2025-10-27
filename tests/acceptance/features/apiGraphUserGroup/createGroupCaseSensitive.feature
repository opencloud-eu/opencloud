Feature: create groups, group names are case insensitive
  As an admin
  I want to create groups with similar cases
  So that I can check if the group names are case sensitive

  @owncloud-issue-3516
  Scenario Outline: group names are case insensitive, creating groups with different upper and lower case names
    Given using OCS API version "<ocs-api-version>"
    And group "<group>" has been created
    When the administrator creates a group "<group-2>" using the Graph API
    And the administrator creates a group "<group-3>" using the Graph API
    Then the HTTP status code of responses on all endpoints should be "409"
    And these groups should not exist:
      | groupname |
      | <group-2> |
      | <group-3> |
    Examples:
      | ocs-api-version | group                | group-2              | group-3              |
      | 1               | case-sensitive-group | Case-Sensitive-Group | CASE-SENSITIVE-GROUP |
      | 1               | Case-Sensitive-Group | CASE-SENSITIVE-GROUP | case-sensitive-group |
      | 1               | CASE-SENSITIVE-GROUP | case-sensitive-group | Case-Sensitive-Group |
      | 2               | case-sensitive-group | Case-Sensitive-Group | CASE-SENSITIVE-GROUP |
      | 2               | Case-Sensitive-Group | CASE-SENSITIVE-GROUP | case-sensitive-group |
      | 2               | CASE-SENSITIVE-GROUP | case-sensitive-group | Case-Sensitive-Group |
