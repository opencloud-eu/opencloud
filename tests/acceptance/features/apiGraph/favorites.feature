Feature: favorites
  As a user
  I want to check that I can mark and unmark files and folders as favorites using the Graph API

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: add a file to favorites in the personal space
    Given user "Alice" has created folder "parent"
    And user "Alice" has uploaded file "filesForUpload/<file>" to "/parent/<file>"
    When user "Alice" marks file "parent/<file>" as favorite from space "Personal" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "eTag",
          "file",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "size"
        ],
        "properties": {
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "file": {
            "type": "object",
            "required": ["mimeType"],
            "properties": {
              "mimeType": {
                "const": "<mimeType>"
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "lastModifiedDateTime": {
            "type": "string",
            "format": "date-time"
          },
          "name": {
            "const": "<file>"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id",
              "name",
              "path"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "parent"
              },
              "path": {
                "const": "/parent"
              }
            }
          },
          "size": {
            "type": "integer",
            "const": <size>
          }
        }
      }
      """
    And as user "Alice" file "parent/<file>" should be favorited
    Examples:
      | file           | size  | mimeType                                |
      | simple.odt     | 10119 | application/vnd.oasis.opendocument.text |
      | testavatar.jpg | 45343 | image/jpeg                              |
      | simple.pdf     | 17684 | application/pdf                         |


  Scenario: add a folder to favorites in the personal space
    Given user "Alice" has created folder "parent"
    And user "Alice" has uploaded file with content "first" to "/parent/first.txt"
    And user "Alice" has uploaded file with content "second" to "/parent/second.txt"
    When user "Alice" marks folder "parent" as favorite from space "Personal" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "eTag",
          "folder",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "size"
        ],
        "properties": {
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "folder": {
            "type": "object"
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "lastModifiedDateTime": {
            "type": "string",
            "format": "date-time"
          },
          "name": {
            "const": "parent"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id",
              "name",
              "path"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "/"
              },
              "path": {
                "const": "/"
              }
            }
          },
          "size": {
            "type": "integer",
            "const": 11
          }
        }
      }
      """
    And as user "Alice" folder "parent" should be favorited


  Scenario: add a shared file to favorites
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has a share "textfile.txt" synced
    When user "Brian" marks file "textfile.txt" as favorite from space "Shares" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "eTag",
          "file",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "size"
        ],
        "properties": {
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "file": {
            "type": "object",
            "required": ["mimeType"],
            "properties": {
              "mimeType": {
                "const": "text/plain"
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "lastModifiedDateTime": {
            "type": "string",
            "format": "date-time"
          },
          "name": {
            "const": "textfile.txt"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id",
              "name",
              "path"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "/"
              },
              "path": {
                "const": "/"
              }
            }
          },
          "size": {
            "type": "integer",
            "const": 24
          }
        }
      }
      """
    And as user "Brian" file "Shares/textfile.txt" should be favorited
    But as user "Alice" file "textfile.txt" should not be favorited


  Scenario: add a shared folder to favorites
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/sub"
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "parent/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "parent" synced
    When user "Brian" marks folder "parent/sub" as favorite from space "Shares" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "eTag",
          "folder",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "size"
        ],
        "properties": {
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "folder": {
            "type": "object"
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "lastModifiedDateTime": {
            "type": "string",
            "format": "date-time"
          },
          "name": {
            "const": "sub"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id",
              "name",
              "path"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "parent"
              },
              "path": {
                "const": "/parent"
              }
            }
          },
          "size": {
            "type": "integer",
            "const": 0
          }
        }
      }
      """
    And as user "Brian" folder "Shares/parent/sub" should be favorited
    But as user "Alice" folder "parent/sub" should not be favorited


  Scenario: add a file of the project space to favorites
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" marks file "text.txt" as favorite from space "new-space" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "eTag",
          "file",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "size"
        ],
        "properties": {
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "file": {
            "type": "object",
            "required": ["mimeType"],
            "properties": {
              "mimeType": {
                "const": "text/plain"
              }
            }
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "lastModifiedDateTime": {
            "type": "string",
            "format": "date-time"
          },
          "name": {
            "const": "text.txt"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id",
              "name",
              "path"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "/"
              },
              "path": {
                "const": "/"
              }
            }
          },
          "size": {
            "type": "integer",
            "const": 11
          }
        }
      }
      """


  Scenario: add a folder of the project space to favorites
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "space-folder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" marks folder "space-folder" as favorite from space "new-space" using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "eTag",
          "folder",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "size"
        ],
        "properties": {
          "eTag": {
            "type": "string",
            "pattern": "%etag_pattern%"
          },
          "folder": {
            "type": "object"
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "lastModifiedDateTime": {
            "type": "string",
            "format": "date-time"
          },
          "name": {
            "const": "space-folder"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id",
              "name",
              "path"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "/"
              },
              "path": {
                "const": "/"
              }
            }
          },
          "size": {
            "type": "integer",
            "const": 0
          }
        }
      }
      """


  Scenario: remove file from favorites from the personal space
    Given user "Alice" has created folder "parent"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/parent/textfile.txt"
    And user "Alice" has marked file "parent/textfile.txt" as favorite from space "Personal"
    When user "Alice" unmarks file "parent/textfile.txt" as favorite from space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    And as user "Alice" file "parent/textfile.txt" should not be favorited


  Scenario: remove folder from favorites from the personal space
    Given user "Alice" has created folder "parent"
    And user "Alice" has marked folder "parent" as favorite from space "Personal"
    When user "Alice" unmarks folder "parent" as favorite from space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    And as user "Alice" folder "parent" should not be favorited


  Scenario: remove file from favorites from the shares
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "parent"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/parent/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent/textfile.txt |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "textfile.txt" synced
    And user "Brian" has marked file "textfile.txt" as favorite from space "Shares"
    When user "Brian" unmarks file "textfile.txt" as favorite from space "Shares" using the Graph API
    Then the HTTP status code should be "204"
    And as user "Brian" file "Shares/textfile.txt" should not be favorited


  Scenario: remove folder from favorites from the shares
    Given user "Brian" has been created with default attributes
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/sub"
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "parent" synced
    And user "Brian" has marked folder "parent/sub" as favorite from space "Shares"
    When user "Brian" unmarks folder "parent/sub" as favorite from space "Shares" using the Graph API
    Then the HTTP status code should be "204"
    And as user "Brian" folder "Shares/parent/sub" should not be favorited


  Scenario: remove file from favorites from the project space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Brian" has marked file "text.txt" as favorite from space "new-space"
    When user "Brian" unmarks file "text.txt" as favorite from space "new-space" using the Graph API
    Then the HTTP status code should be "204"


  Scenario: remove folder from favorites from the project space
    Given user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "space-folder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    And user "Brian" has marked folder "space-folder" as favorite from space "new-space"
    When user "Brian" unmarks folder "space-folder" as favorite from space "new-space" using the Graph API
    Then the HTTP status code should be "204"


  Scenario: add a file to favorites after unmarking it as favorite in the personal space
    Given user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/testavatar.jpg"
    And user "Alice" has marked file "testavatar.jpg" as favorite from space "Personal"
    And user "Alice" has unmarked file "testavatar.jpg" as favorite from space "Personal"
    When user "Alice" marks file "/testavatar.jpg" as favorite from space "Personal" using the Graph API
    Then the HTTP status code should be "201"
    And as user "Alice" file "testavatar.jpg" should be favorited


  Scenario: add a file to favorites twice
    Given user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "/testavatar.jpg"
    And user "Alice" has marked file "testavatar.jpg" as favorite from space "Personal"
    When user "Alice" marks file "/testavatar.jpg" as favorite from space "Personal" using the Graph API
    Then the HTTP status code should be "201"
    And as user "Alice" file "testavatar.jpg" should be favorited
