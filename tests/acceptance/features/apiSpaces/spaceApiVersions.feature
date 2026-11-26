Feature: manage a space through each version of the Graph API
  As a space admin
  I want to create, get, change and delete a space through v1.0 and through v1beta1
  So that my client can use the version whose format of permissions it understands

  Background:
    Given user "Alice" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API


  Scenario Outline: space admin manages a project space using the Graph API version <api-version>
    When user "Alice" creates a space "Project Venus" of type "project" with quota "2000" using the Graph API version "<api-version>"
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["name", "driveType"],
        "properties": {
          "name": {
            "const": "Project Venus"
          },
          "driveType": {
            "const": "project"
          }
        }
      }
      """
    When user "Alice" looks up the single space "Project Venus" using the Graph API version "<api-version>" with query "$expand=root($expand=permissions)"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["name", "root"],
        "properties": {
          "name": {
            "const": "Project Venus"
          },
          "root": {
            "type": "object",
            "required": ["permissions"],
            "properties": {
              "permissions": {
                "type": "array",
                "minItems": 1,
                "maxItems": 1,
                "items": {
                  "type": "object",
                  "required": ["roles", "grantedToV2"],
                  "properties": {
                    "roles": {
                      "const": ["<manager-role>"]
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    When user "Alice" changes the name of the "Project Venus" space to "Project Mars" using the Graph API version "<api-version>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name": {
            "const": "Project Mars"
          }
        }
      }
      """
    When user "Alice" disables a space "Project Mars" using the Graph API version "<api-version>"
    Then the HTTP status code should be "204"
    When user "Alice" deletes a space "Project Mars" using the Graph API version "<api-version>"
    Then the HTTP status code should be "204"
    Examples:
      | api-version | manager-role                         |
      | v1.0        | manager                              |
      | v1beta1     | 312c0871-5ef7-4b3a-85b6-0e4074c64049 |
