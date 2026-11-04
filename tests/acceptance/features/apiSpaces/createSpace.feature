Feature: create space
  As an admin and space admin
  I want to create new spaces
  So that I can organize a set of resources in a hierarchical tree

  Background:
    Given user "Alice" has been created with default attributes

  @issue-5938
  Scenario Outline: user with role user and user light can't create space via Graph API
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to create a space "Project Mars" of type "project" with the default quota using the Graph API
    Then the HTTP status code should be "403"
    And the user "Alice" should not have a space called "share space"
    Examples:
      | user-role  |
      | User       |
      | User Light |


  Scenario Outline: admin or space admin user can create a space via the Graph API with a default quota
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" creates a space "Project Mars" of type "project" with the default quota using the Graph API
    Then the HTTP status code should be "201"
    And the JSON response should contain space called "Project Mars" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "driveAlias",
          "name",
          "id",
          "quota",
          "root",
          "webUrl"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Mars"]
          },
          "driveType": {
            "type": "string",
            "enum": ["project"]
          },
          "driveAlias": {
            "type": "string",
            "enum": ["project/project-mars"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          },
          "quota": {
            "type": "object",
            "required": [
              "total"
            ],
            "properties": {
              "total": {
                "type": "number",
                "enum": [1000000000]
              }
            }
          },
          "root": {
            "type": "object",
            "required": [
              "webDavUrl"
            ],
            "properties": {
              "webDavUrl": {
                "type": "string",
                "enum": ["%base_url%/dav/spaces/%space_id%"]
              }
            }
          },
          "webUrl": {
            "type": "string",
            "enum": ["%base_url%/f/%space_id%"]
          }
        }
      }
      """
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |


  Scenario Outline: admin or space admin user can create a space via the Graph API with certain quota
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" creates a space "Project Venus" of type "project" with quota "2000" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON response should contain space called "Project Venus" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "name",
          "id",
          "quota",
          "root",
          "webUrl"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Venus"]
          },
          "driveType": {
            "type": "string",
            "enum": ["project"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          },
          "quota": {
            "type": "object",
            "required": [
              "total"
            ],
            "properties": {
              "total": {
                "type": "number",
                "enum": [2000]
              }
            }
          },
          "root": {
            "type": "object",
            "required": [
              "webDavUrl"
            ],
            "properties": {
              "webDavUrl": {
                "type": "string",
                "enum": ["%base_url%/dav/spaces/%space_id%"]
              }
            }
          },
          "webUrl": {
            "type": "string",
            "enum": ["%base_url%/f/%space_id%"]
          }
        }
      }
      """
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |


  Scenario: space admin creates a project space with a client-supplied id via the Graph API
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    When user "Alice" creates a space "Project Mars" of type "project" with id "4e3b1a52-8f0d-4c6e-9a71-2d5f6b8c9e10" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON response should contain space called "Project Mars" and match
      """
      {
        "type": "object",
        "required": ["id"],
        "properties": {
          "id": {
            "type": "string",
            "pattern": "\\$4e3b1a52-8f0d-4c6e-9a71-2d5f6b8c9e10$"
          }
        }
      }
      """
    When user "Alice" looks up the single space "Project Mars" via the Graph API by using its id
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Mars" and match
      """
      {
        "type": "object",
        "required": ["id", "name"],
        "properties": {
          "id": {
            "type": "string",
            "pattern": "\\$4e3b1a52-8f0d-4c6e-9a71-2d5f6b8c9e10$"
          },
          "name": {
            "type": "string",
            "enum": ["Project Mars"]
          }
        }
      }
      """
    When user "Alice" tries to create a space "Project Venus" of type "project" with id "4e3b1a52-8f0d-4c6e-9a71-2d5f6b8c9e10" using the Graph API
    Then the HTTP status code should be "409"
    And the user "Alice" should not have a space called "Project Venus"
