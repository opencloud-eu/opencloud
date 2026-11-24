Feature: Multi-tenancy
  I want to make sure that users from different tenants are isolated from each other,
  so that each tenant's data and users remain private and secure.

  Note:
  All users are managed via LDAP and are assumed to exist.
  Tests will use existing users without creating or deleting them.

  Prepared LDAP users:
    | user  | tenant   | group                |
    |-------|----------|----------------------|
    | alice | tenant-1 | new-features-lovers  |
    | brian | tenant-1 | -                    |
    | carol | tenant-2 | -                    |
    | david | tenant-2 | release-lover        |


  Scenario: users from the same tenant can see each other
    When user "Brian" searches for user "ali" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "displayName",
                "id",
                "onPremisesSamAccountName",
                "userType"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
                },
                "onPremisesSamAccountName": {
                  "const": "alice"
                },
                "userType": {
                  "const": "Member"
                }
              }
            }
          }
        }
      }
      """


  Scenario: users from different tenants cannot see each other
    When user "David" searches for user "brian" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 0,
            "maxItems": 0
          }
        }
      }
      """
