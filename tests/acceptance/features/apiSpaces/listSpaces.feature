Feature: List and create spaces
  As a user
  I want to be able to list project spaces
  So that I can retrieve the information about them

  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path


  Scenario: ordinary user can request information about their Space via the Graph API
    When user "Alice" lists all available spaces via the Graph API
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Alice Hansen" and match
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
            "enum": ["Alice Hansen"]
          },
          "driveType": {
             "type": "string",
            "enum": ["personal"]
          },
          "driveAlias": {
             "type": "string",
            "enum": ["personal/alice"]
          },
          "id": {
             "type": "string",
            "enum": ["%space_id%"]
          },
          "quota": {
             "type": "object",
             "required": [
              "state"
             ],
             "properties": {
                "state": {
                  "type": "string",
                  "enum": ["normal"]
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


  Scenario: ordinary user can request information about their Space via the Graph API using a filter
    Given user "Brian" has been created with default attributes
    And user "Brian" has created folder "folder"
    And user "Brian" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Alice" has a share "folder" synced
    When user "Alice" lists all available spaces via the Graph API with query "$filter=driveType eq 'personal'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Alice Hansen" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "driveAlias",
          "name",
          "id",
          "quota",
          "root"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Alice Hansen"]
          },
          "driveType": {
             "type": "string",
            "enum": ["personal"]
          },
          "driveAlias": {
             "type": "string",
            "enum": ["personal/alice"]
          },
          "id": {
             "type": "string",
            "enum": ["%space_id%"]
          },
          "quota": {
             "type": "object",
             "required": [
              "state"
             ],
             "properties": {
                "state": {
                  "type": "string",
                  "enum": ["normal"]
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
    And the json response should not contain a space with name "Shares"
    And the json response should only contain spaces of type "personal"


  Scenario: ordinary user will not see any space when using a filter for project
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "my project" of type "project" with quota "20"
    When user "Alice" lists all available spaces via the Graph API with query "$filter=driveType eq 'project'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "my project" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "name",
          "id"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["my project"]
          },
          "driveType": {
            "type": "string",
            "enum": ["project"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          }
        }
      }
      """
    And the json response should not contain a space with name "Alice Hansen"


  Scenario: ordinary user can access their space via the webDav API
    When user "Alice" lists all available spaces via the Graph API
    And user "Alice" lists the content of the space with the name "Alice Hansen" using the WebDav Api
    Then the HTTP status code should be "207"


  Scenario: user can list his personal space via multiple endpoints
    When user "Alice" lists all available spaces via the Graph API with query "$filter=driveType eq 'personal'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Alice Hansen" owned by "Alice" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "name",
          "root",
          "owner",
          "webUrl"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Alice Hansen"]
          },
          "driveType": {
             "type": "string",
            "enum": ["personal"]
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
          "owner": {
            "type": "object",
            "required": [
              "user"
            ],
            "properties": {
              "user": {
                "type": "object",
                "required": [
                  "id"
                ],
                "properties": {
                  "id": {
                    "type": "string",
                    "enum": ["%user_id%"]
                  }
                }
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
    When user "Alice" looks up the single space "Alice Hansen" via the Graph API by using its id
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Alice Hansen" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "name",
          "root",
          "webUrl"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Alice Hansen"]
          },
          "driveType": {
             "type": "string",
            "enum": ["personal"]
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


  Scenario Outline: user can list his created spaces via multiple endpoints
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" creates a space "Project Venus" of type "project" with quota "2000" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON response should contain space called "Project Venus" and match
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
            "enum": ["Project Venus"]
          },
          "driveType": {
             "type": "string",
            "enum": ["project"]
          },
          "driveAlias": {
            "type": "string",
            "enum": ["project/project-venus"]
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
    When user "Alice" looks up the single space "Project Venus" via the Graph API by using its id
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Venus" and match
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
            "enum": ["Project Venus"]
          },
          "driveType": {
             "type": "string",
            "enum": ["project"]
          },
          "driveAlias": {
            "type": "string",
            "enum": ["project/project-venus"]
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


  Scenario Outline: user cannot list space by id if he is not member of the space
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    And user "Admin" has created a space "Project Venus" with the default quota using the Graph API
    When user "Alice" tries to look up the single space "Project Venus" owned by the user "Admin" by using its id
    Then the HTTP status code should be "404"
    And the json response should not contain a space with name "Project Venus"
    Examples:
      | user-role  |
      | User       |
      | User Light |

  @owncloud-issue-7160
  Scenario Outline: get share jail space information of the user when user has a pending share
    Given user "Brian" has been created with default attributes
    And user "Alice" has disabled auto-accepting
    And user "Brian" has uploaded file with content "this is a test file." to "test.txt"
    And the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    And user "Brian" has sent the following resource share invitation:
      | resource        | test.txt |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    When user "Alice" lists all available spaces via the Graph API
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Shares" owned by "Alice" and match
      """
      {
        "type": "object",
        "required": [
          "driveType",
          "driveAlias",
          "name",
          "id",
          "root",
          "webUrl"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Shares"]
          },
          "driveType": {
            "type": "string",
            "enum": ["virtual"]
          },
          "driveAlias": {
            "type": "string",
            "enum": ["virtual/shares"]
          },
          "id": {
            "type": "string",
            "enum": ["%space_id%"]
          },
          "quota": {
            "type": "object",
            "required": [
              "state"
            ],
            "properties": {
              "state": {
                "type": "string",
                "enum": ["normal"]
              }
            }
          },
          "root": {
            "type": "object",
            "required": [
              "eTag",
              "webDavUrl"
            ],
            "properties": {
              "eTag": {
                "type": "string",
                "enum": ["%space_etag%"]
              },
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
      | User        |
      | User Light  |
