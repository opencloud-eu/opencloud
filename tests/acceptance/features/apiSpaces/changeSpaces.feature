Feature: Change data of space
  As a user with space admin rights
  I want to be able to change the meta-data of a created space (increase the quota, change name, etc.)
  So that I can manage the spaces

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project Jupiter" of type "project" with quota "20"
    And user "Alice" has sent the following space share invitation:
      | space           | Project Jupiter |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | Space Editor    |
    And user "Alice" has sent the following space share invitation:
      | space           | Project Jupiter |
      | sharee          | Bob             |
      | shareType       | user            |
      | permissionsRole | Space Viewer    |
    And using spaces DAV path


  Scenario: user with space manager role can change the name of a space via the Graph API
    When user "Alice" changes the name of the "Project Jupiter" space to "Project Death Star"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
       {
        "type": "object",
        "required": [
          "name",
          "driveType"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Death Star"]
          },
          "driveType": {
            "type": "string",
            "enum": ["project"]
          }
        }
      }
      """


  Scenario Outline: user other than space manager role can't change the name of a Space via the Graph API
    When user "<user>" changes the name of the "Project Jupiter" space to "Project Jupiter"
    Then the HTTP status code should be "404"
    Examples:
      | user  |
      | Brian |
      | Bob   |


  Scenario: user with space manager role can change the description(subtitle) of a space via the Graph API
    When user "Alice" changes the description of the "Project Jupiter" space to "The Death Star is a fictional mobile space station"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
       {
        "type": "object",
        "required": [
          "name",
          "driveType",
          "description"
        ],
        "properties": {
          "driveType": {
            "type": "string",
            "enum": ["project"]
          },
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "description": {
            "type": "string",
            "enum": ["The Death Star is a fictional mobile space station"]
          }
        }
      }
      """


  Scenario Outline: viewer and editor cannot change the description(subtitle) of a space via the Graph API
    When user "<user>" changes the description of the "Project Jupiter" space to "The Death Star is a fictional mobile space station"
    Then the HTTP status code should be "404"
    Examples:
      | user  |
      | Brian |
      | Bob   |


  Scenario Outline: user with normal space permission can't increase the quota of a Space via the Graph API
    When user "<user>" changes the quota of the "Project Jupiter" space to "100"
    Then the HTTP status code should be "403"
    Examples:
      | user  |
      | Brian |
      | Bob   |


  Scenario Outline: space admin user set no restriction quota of a Space via the Graph API
    When user "Alice" changes the quota of the "Project Jupiter" space to "<quota-value>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
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
            "enum": ["Project Jupiter"]
          },
          "quota": {
            "type": "object",
            "required": [
              "used",
              "total"
            ],
            "properties": {
              "used" : {
                "type": "number",
                "enum": [0]
              },
              "total" : {
                "type": "number",
                "enum": [0]
              }
            }
          }
        }
      }
      """
    Examples:
      | quota-value |
      | 0           |
      | -1          |


  Scenario: user space admin set readme file as description of the space via the Graph API
    Given user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "Alice" has uploaded a file inside space "Project Jupiter" with content "space description" to ".space/readme.md"
    When user "Alice" sets the file ".space/readme.md" as a description in a special section of the "Project Jupiter" space
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Jupiter" owned by "Alice" with description file ".space/readme.md" and match
      """
      {
        "type": "object",
        "required": [
          "name",
          "special"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "special": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "size",
                "name",
                "specialFolder",
                "file",
                "id",
                "eTag"
              ],
              "properties": {
                "size": {
                  "type": "number",
                  "enum": [17]
                },
                "name": {
                  "type": "string",
                  "enum": ["readme.md"]
                },
                "specialFolder": {
                  "type": "object",
                  "required": [
                    "name"
                  ],
                  "properties": {
                    "name": {
                      "type": "string",
                      "enum": ["readme"]
                    }
                  }
                },
                "file": {
                  "type": "object",
                  "required": [
                    "mimeType"
                  ],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["text/markdown"]
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "enum": ["%file_id%"]
                },
                "tag": {
                  "type": "string",
                  "enum": ["%eTag%"]
                }
              }
            }
          }
        }
      }
      """
    And for user "Alice" the content of the file ".space/readme.md" of the space "Project Jupiter" should be "space description"


  Scenario Outline: user member of the space changes readme file
    Given user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "Alice" has uploaded a file inside space "Project Jupiter" with content "space description" to ".space/readme.md"
    And user "Alice" has set the file ".space/readme.md" as a description in a special section of the "Project Jupiter" space
    When user "<user>" uploads a file inside space "Project Jupiter" with content "new description" to ".space/readme.md" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "<user>" the content of the file ".space/readme.md" of the space "Project Jupiter" should be "<content>"
    Examples:
      | user  | http-status-code | content           |
      | Brian | 204              | new description   |
      | Bob   | 403              | space description |


  Scenario Outline: user space admin and editor set image file as space image of the space via the Graph API
    Given user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "<user>" has uploaded a file inside space "Project Jupiter" with content "" to ".space/<file-name>"
    When user "<user>" sets the file ".space/<file-name>" as a space image in a special section of the "Project Jupiter" space
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Jupiter" owned by "Alice" with description file ".space/<file-name>" and match
      """
      {
        "type": "object",
        "required": [
          "name",
          "special"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "special": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "size",
                "name",
                "specialFolder",
                "file",
                "id",
                "eTag"
              ],
              "properties": {
                "size": {
                  "type": "number",
                  "enum": [0]
                },
                "name": {
                  "type": "string",
                  "enum": ["<file-name>"]
                },
                "specialFolder": {
                  "type": "object",
                  "required": [
                    "name"
                  ],
                  "properties": {
                    "name": {
                      "type": "string",
                      "enum": ["image"]
                    }
                  }
                },
                "file": {
                  "type": "object",
                  "required": [
                    "mimeType"
                  ],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["<mime-type>"]
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "enum": ["%file_id%"]
                },
                "tag": {
                  "type": "string",
                  "enum": ["%eTag%"]
                }
              }
            }
          }
        }
      }
      """
    And for user "<user>" folder ".space/" of the space "Project Jupiter" should contain these entries:
      | <file-name> |
    Examples:
      | user  | file-name       | mime-type  |
      | Alice | spaceImage.jpeg | image/jpeg |
      | Brian | spaceImage.png  | image/png  |
      | Alice | spaceImage.gif  | image/gif  |


  Scenario: user viewer cannot set image file as space image of the space via the Graph API
    Given user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "Alice" has uploaded a file inside space "Project Jupiter" with content "" to ".space/someImageFile.jpg"
    When user "Bob" sets the file ".space/someImageFile.jpg" as a space image in a special section of the "Project Jupiter" space
    Then the HTTP status code should be "404"


  Scenario Outline: user set new readme file as description of the space via the graph API
    Given user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "Alice" has uploaded a file inside space "Project Jupiter" with content "space description" to ".space/readme.md"
    And user "Alice" has set the file ".space/readme.md" as a description in a special section of the "Project Jupiter" space
    When user "<user>" uploads a file inside space "Project Jupiter" owned by the user "Alice" with content "new content" to ".space/readme.md" using the WebDAV API
    Then the HTTP status code should be "<http-status-code>"
    And for user "<user>" the content of the file ".space/readme.md" of the space "Project Jupiter" should be "<expected-content>"
    When user "<user>" lists all available spaces via the Graph API
    And the JSON response should contain space called "Project Jupiter" owned by "Alice" with description file ".space/readme.md" and match
      """
      {
        "type": "object",
        "required": [
          "name",
          "special"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "special": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "size",
                "name",
                "specialFolder",
                "file",
                "id",
                "eTag"
              ],
              "properties": {
                "size": {
                  "type": "number",
                  "enum": [<expected-size>]
                },
                "name": {
                  "type": "string",
                  "enum": ["readme.md"]
                },
                "specialFolder": {
                  "type": "object",
                  "required": [
                    "name"
                  ],
                  "properties": {
                    "name": {
                      "type": "string",
                      "enum": ["readme"]
                    }
                  }
                },
                "file": {
                  "type": "object",
                  "required": [
                    "mimeType"
                  ],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["text/markdown"]
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "enum": ["%file_id%"]
                },
                "tag": {
                  "type": "string",
                  "enum": ["%eTag%"]
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | user  | http-status-code | expected-size | expected-content  |
      | Alice | 204              | 11            | new content       |
      | Brian | 204              | 11            | new content       |
      | Bob   | 403              | 17            | space description |


  Scenario Outline: user set new image file as space image of the space via the Graph API
    Given user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "Alice" has uploaded a file inside space "Project Jupiter" with content "" to ".space/spaceImage.jpeg"
    And user "Alice" has set the file ".space/spaceImage.jpeg" as a space image in a special section of the "Project Jupiter" space
    When user "<user>" uploads a file inside space "Project Jupiter" with content "" to ".space/newSpaceImage.png" using the WebDAV API
    And user "<user>" sets the file ".space/newSpaceImage.png" as a space image in a special section of the "Project Jupiter" space
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Jupiter" owned by "Alice" with space image ".space/newSpaceImage.png" and match
      """
      {
        "type": "object",
        "required": [
          "name",
          "special"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "special": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "size",
                "name",
                "specialFolder",
                "file",
                "id",
                "eTag"
              ],
              "properties": {
                "size": {
                  "type": "number",
                  "enum": [0]
                },
                "name": {
                  "type": "string",
                  "enum": ["newSpaceImage.png"]
                },
                "specialFolder": {
                  "type": "object",
                  "required": [
                    "name"
                  ],
                  "properties": {
                    "name": {
                      "type": "string",
                      "enum": ["image"]
                    }
                  }
                },
                "file": {
                  "type": "object",
                  "required": [
                    "mimeType"
                  ],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["image/png"]
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "enum": ["%file_id%"]
                },
                "tag": {
                  "type": "string",
                  "enum": ["%eTag%"]
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | user  |
      | Alice |
      | Brian |


  Scenario Outline: user can't upload resource greater than set quota
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    And user "Admin" has changed the quota of the personal space of user "Alice" space to "15"
    When user "Alice" uploads a file inside space "Alice Hansen" with content "file is more than 15 bytes" to "file.txt" using the WebDAV API
    Then the HTTP status code should be "507"
    And for user "Alice" the space "Personal" should not contain these entries:
      | file.txt |
    Examples:
      | user-role   |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: admin user set own quota of a personal space via the Graph API and upload resource
    When user "Admin" changes the quota of the "Admin" space to "<quota-value>"
    Then the HTTP status code should be "200"
    When user "Admin" uploads a file inside space "Admin" with content "file is more than 15 bytes" to "file.txt" using the WebDAV API
    Then the HTTP status code should be <http-status-code>
    And for user "Admin" the space "Personal" should contain these entries:
      | file.txt |
    Examples:
      | quota-value | http-status-code        |
      | 10000       | between "201" and "204" |
      | 0           | between "201" and "204" |
      | -1          | between "201" and "204" |


  Scenario Outline: admin user set an user personal space quota of via the Graph API and upload resource
    When user "Admin" changes the quota of the "Brian Murphy" space to "<quota-value>"
    Then the HTTP status code should be "200"
    When user "Brian" uploads a file inside space "Brian Murphy" with content "file is more than 15 bytes" to "file.txt" using the WebDAV API
    Then the HTTP status code should be <http-status-code>
    And for user "Brian" the space "Personal" should contain these entries:
      | file.txt |
    Examples:
      | quota-value | http-status-code        |
      | 10000       | between "201" and "204" |
      | 0           | between "201" and "204" |
      | -1          | between "201" and "204" |


  Scenario: user sends invalid space uuid via the graph API
    When user "Admin" tries to change the name of the "non-existing" space to "new name"
    Then the HTTP status code should be "404"
    When user "Admin" tries to change the quota of the "non-existing" space to "10"
    Then the HTTP status code should be "404"
    When user "Alice" tries to change the description of the "non-existing" space to "new description"
    Then the HTTP status code should be "404"


  Scenario: user sends PATCH request to other user's space that they don't have access to
    Given these users have been created with default attributes:
      | username |
      | Carol    |
    When user "Carol" sends PATCH request to the space "Personal" of user "Alice" with data "{}"
    Then the HTTP status code should be "404"
    When user "Carol" sends PATCH request to the space "Project Jupiter" of user "Alice" with data "{}"
    Then the HTTP status code should be "404"

  @env-config
  Scenario Outline: space member with role 'Space Editor Without Versions' and Space Editor edits the space
    Given the administrator has enabled the permissions role "Space Editor Without Versions"
    And these users have been created with default attributes:
      | username |
      | Carol    |
    And user "Alice" has sent the following space share invitation:
      | space           | Project Jupiter |
      | sharee          | Carol           |
      | shareType       | user            |
      | permissionsRole | <role>          |
    When user "Carol" creates a folder ".space" in space "Project Jupiter" using the WebDav Api
    Then the HTTP status code should be "201"
    When user "Carol" uploads a file inside space "Project Jupiter" with content "hello" to ".space/readme.md" using the WebDAV API
    Then the HTTP status code should be "201"
    When user "Carol" sets the file ".space/readme.md" as a description in a special section of the "Project Jupiter" space
    Then the HTTP status code should be "200"
    When user "Carol" removes the folder ".space" from space "Project Jupiter"
    Then the HTTP status code should be "204"
    Examples:
      | role                          |
      | Space Editor Without Versions |
      | Space Editor                  |

  @issue-462
  Scenario: user doesn't lose the space image when admin fetches the space
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Alice" has created a folder ".space" in space "Project Jupiter"
    And user "Alice" has uploaded a file inside space "Project Jupiter" with content "" to ".space/spaceImage.jpeg"
    And user "Alice" has set the file ".space/spaceImage.jpeg" as a space image in a special section of the "Project Jupiter" space
    When user "Brian" lists all spaces via the Graph API
    And user "Alice" lists all available spaces via the Graph API with query "$filter=driveType eq 'project'"
    Then the HTTP status code should be "200"
    And the JSON response should contain space called "Project Jupiter" owned by "Alice" with space image ".space/spaceImage.png" and match
      """
      {
        "type": "object",
        "required": [
          "name",
          "special"
        ],
        "properties": {
          "name": {
            "type": "string",
            "enum": ["Project Jupiter"]
          },
          "special": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "size",
                "name",
                "specialFolder",
                "file"
              ],
              "properties": {
                "size": {
                  "type": "number",
                  "enum": [0]
                },
                "name": {
                  "type": "string",
                  "enum": ["spaceImage.jpeg"]
                },
                "specialFolder": {
                  "type": "object",
                  "required": [
                    "name"
                  ],
                  "properties": {
                    "name": {
                      "type": "string",
                      "enum": ["image"]
                    }
                  }
                },
                "file": {
                  "type": "object",
                  "required": [
                    "mimeType"
                  ],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["image/jpeg"]
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
