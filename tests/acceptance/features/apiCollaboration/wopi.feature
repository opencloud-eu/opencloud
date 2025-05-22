Feature: collaboration (wopi)
  As a user
  I want to access files with collaboration service apps
  So that I can collaborate with other users

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: open file with .odt extension
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "app_url",
          "method",
          "form_parameters"
        ],
        "properties": {
          "app_url": {
            "type": "string",
            "pattern": "^.*\\?WOPISrc=.*wopi%2Ffiles%2F[a-fA-F0-9]{64}$"
          },
          "method": {
            "const": "POST"
          },
          "form_parameters": {
            "type": "object",
            "required": [
              "access_token",
              "access_token_ttl"
            ],
            "properties": {
              "access_token": {
                "type": "string"
              },
              "access_token_ttl": {
                "type": "string"
              }
            }
          }
        }
      }
      """
    Examples:
      | app-endpoint                                     |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open?file_id=<<FILEID>>                     |


  Scenario: open text file with app name in url query (MIME type not registered in app-registry)
    Given user "Alice" has uploaded file "filesForUpload/lorem.txt" to "lorem.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "/app/open?file_id=<<FILEID>>&app_name=FakeOffice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "app_url",
          "method",
          "form_parameters"
        ],
        "properties": {
          "app_url": {
            "type": "string",
            "pattern": "^.*\\?WOPISrc=.*wopi%2Ffiles%2F[a-fA-F0-9]{64}$"
          },
          "method": {
            "const": "POST"
          },
          "form_parameters": {
            "type": "object",
            "required": [
              "access_token",
              "access_token_ttl"
            ],
            "properties": {
              "access_token": {
                "type": "string"
              },
              "access_token_ttl": {
                "type": "string"
              }
            }
          }
        }
      }
      """

  @issue-9928
  Scenario: user tries to open text file without app name in url query (MIME type not registered in app-registry)
    Given user "Alice" has uploaded file "filesForUpload/lorem.txt" to "lorem.txt"
    And we save it into "FILEID"
    When user "Alice" tries to send HTTP method "POST" to URL "/app/open?file_id=<<FILEID>>"
    Then the HTTP status code should be "500"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "SERVER_ERROR"
          },
          "message": {
            "const": "Error contacting the requested application, please use a different one or try again later"
          }
        }
      }
      """


  Scenario Outline: sharee open file with .odt extension
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | simple.odt |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "app_url",
          "method",
          "form_parameters"
        ],
        "properties": {
          "app_url": {
            "type": "string",
            "pattern": "^.*\\?WOPISrc=.*wopi%2Ffiles%2F[a-fA-F0-9]{64}$"
          },
          "method": {
            "const": "POST"
          },
          "form_parameters": {
            "type": "object",
            "required": [
              "access_token",
              "access_token_ttl"
            ],
            "properties": {
              "access_token": {
                "type": "string"
              },
              "access_token_ttl": {
                "type": "string"
              }
            }
          }
        }
      }
      """
    Examples:
      | app-endpoint                                     |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open?file_id=<<FILEID>>                     |


  Scenario Outline: public user opens file with .odt extension
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    And user "Alice" has created the following resource link share:
      | resource        | simple.odt |
      | space           | Personal   |
      | permissionsRole | view       |
      | password        | %public%   |
    When the public sends HTTP method "POST" to URL "<app-endpoint>" with password "%public%"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "app_url",
          "method",
          "form_parameters"
        ],
        "properties": {
          "app_url": {
            "type": "string",
            "pattern": "^.*\\?WOPISrc=.*wopi%2Ffiles%2F[a-fA-F0-9]{64}$"
          },
          "method": {
            "const": "POST"
          },
          "form_parameters": {
            "type": "object",
            "required": [
              "access_token",
              "access_token_ttl"
            ],
            "properties": {
              "access_token": {
                "type": "string"
              },
              "access_token_ttl": {
                "type": "string"
              }
            }
          }
        }
      }
      """
    Examples:
      | app-endpoint                                     |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open?file_id=<<FILEID>>                     |

  @issue-9928
  Scenario Outline: user tries to open unsupported file format
    Given user "Alice" has uploaded file "filesForUpload/simple.pdf" to "simple.pdf"
    And we save it into "FILEID"
    When user "Alice" tries to send HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "500"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "SERVER_ERROR"
          },
          "message": {
            "const": "Error contacting the requested application, please use a different one or try again later"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                     |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open?file_id=<<FILEID>>                     |


  Scenario Outline: user tries to open deleted file
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    And user "Alice" has deleted file "/simple.odt"
    When user "Alice" tries to send HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "RESOURCE_NOT_FOUND"
          },
          "message": {
            "const": "file does not exist"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                     |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open?file_id=<<FILEID>>                     |


  Scenario Outline: open file with .odt extension with different view mode
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "app_url",
          "method",
          "form_parameters"
        ],
        "properties": {
          "app_url": {
            "type": "string",
            "pattern": "^.*\\?WOPISrc=.*wopi%2Ffiles%2F[a-fA-F0-9]{64}$"
          },
          "method": {
            "const": "POST"
          },
          "form_parameters": {
            "type": "object",
            "required": [
              "access_token",
              "access_token_ttl"
            ],
            "properties": {
              "access_token": {
                "type": "string"
              },
              "access_token_ttl": {
                "type": "string"
              }
            }
          }
        }
      }
      """
    Examples:
      | app-endpoint                                                     |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice&view_mode=view  |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice&view_mode=read  |
      | /app/open?file_id=<<FILEID>>&app_name=FakeOffice&view_mode=write |

  @issue-9495
  Scenario Outline: open file with .odt extension (open-with-web)
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "uri"
        ],
        "properties": {
          "uri": {
            "type": "string",
             "pattern": "%base_url%/external\\?<url-query>contextRouteName=files-spaces-personal&fileId=%uuidv4_pattern%%24%uuidv4_pattern%%21%uuidv4_pattern%$"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                              | url-query       |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice | app=FakeOffice& |
      | /app/open-with-web?file_id=<<FILEID>>                     |                 |


  Scenario: open text file using open-with-web with app name in url query (MIME type not registered in app-registry)
    Given user "Alice" has uploaded file "filesForUpload/lorem.txt" to "lorem.txt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "/app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "uri"
        ],
        "properties": {
          "uri": {
            "type": "string",
             "pattern": "%base_url%/external\\?app=FakeOffice&contextRouteName=files-spaces-personal&fileId=%uuidv4_pattern%%24%uuidv4_pattern%%21%uuidv4_pattern%$"
          }
        }
      }
      """

  @issue-9928
  Scenario: user tries to open text file using open-with-web without app name in url query (MIME type not registered in app-registry)
    Given user "Alice" has uploaded file "filesForUpload/lorem.txt" to "lorem.txt"
    And we save it into "FILEID"
    When user "Alice" tries to send HTTP method "POST" to URL "/app/open-with-web?file_id=<<FILEID>>"
    Then the HTTP status code should be "500"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "SERVER_ERROR"
          },
          "message": {
            "const": "Error contacting the requested application, please use a different one or try again later"
          }
        }
      }
      """


  Scenario Outline: public user opens file with .odt extension (open-with-web)
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    And user "Alice" has created the following resource link share:
      | resource        | simple.odt |
      | space           | Personal   |
      | permissionsRole | view       |
      | password        | %public%   |
    When the public sends HTTP method "POST" to URL "<app-endpoint>" with password "%public%"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "uri"
        ],
        "properties": {
          "uri": {
            "type": "string",
             "pattern": "%base_url%/external\\?<url-query>contextRouteName=files-spaces-personal&fileId=%uuidv4_pattern%%24%uuidv4_pattern%%21%uuidv4_pattern%$"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                              | url-query       |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice | app=FakeOffice& |
      | /app/open-with-web?file_id=<<FILEID>>                     |                 |


  Scenario Outline: sharee open file with .odt extension (open-with-web)
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | simple.odt |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "uri"
        ],
        "properties": {
          "uri": {
            "type": "string",
             "pattern": "%base_url%/external\\?<url-query>contextRouteName=files-spaces-personal&fileId=%uuidv4_pattern%%24%uuidv4_pattern%%21%uuidv4_pattern%$"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                              | url-query       |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice | app=FakeOffice& |
      | /app/open-with-web?file_id=<<FILEID>>                     |                 |


  Scenario Outline: open file with .odt extension with different view mode (open-with-web)
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "uri"
        ],
        "properties": {
          "uri": {
            "type": "string",
             "pattern": "%base_url%/external\\?app=FakeOffice&contextRouteName=files-spaces-personal&fileId=%uuidv4_pattern%%24%uuidv4_pattern%%21%uuidv4_pattern%$"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                                              |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice&view_mode=view  |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice&view_mode=read  |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice&view_mode=write |

  @issue-9928
  Scenario Outline: user tries to open unsupported file format (open-with-web)
    Given user "Alice" has uploaded file "filesForUpload/simple.pdf" to "simple.pdf"
    And we save it into "FILEID"
    When user "Alice" tries to send HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "500"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "SERVER_ERROR"
          },
          "message": {
            "const": "Error contacting the requested application, please use a different one or try again later"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                              |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open-with-web?file_id=<<FILEID>>                     |


  Scenario Outline: user tries to open deleted file (open-with-web)
    Given user "Alice" has uploaded file "filesForUpload/simple.odt" to "simple.odt"
    And we save it into "FILEID"
    And user "Alice" has deleted file "/simple.odt"
    When user "Alice" tries to send HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "RESOURCE_NOT_FOUND"
          },
          "message": {
            "const": "file does not exist"
          }
        }
      }
      """
    Examples:
      | app-endpoint                                              |
      | /app/open-with-web?file_id=<<FILEID>>&app_name=FakeOffice |
      | /app/open-with-web?file_id=<<FILEID>>                     |


  Scenario: create a text file using wopi endpoint in Personal space
    Given user "Alice" has created folder "testFolder"
    When user "Alice" creates a file "testfile.txt" inside folder "testFolder" in space "Personal" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And as "Alice" file "testFolder/testfile.txt" should exist


  Scenario Outline: sharee with permission Editor/Uploader creates text file inside shared folder
    Given user "Alice" has created folder "testFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder         |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates a file "testFile.txt" inside folder "testFolder" in space "Shares" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And as "Alice" file "testFolder/testFile.txt" should exist
    And as "Brian" file "Shares/testFolder/testFile.txt" should exist
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |

  @issue-10126
  Scenario: sharee with permission Viewer tries to create text file inside shared folder
    Given user "Alice" has created folder "testFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" tries to create a file "testFile.txt" inside folder "testFolder" in space "Shares" using wopi endpoint
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "PERMISSION_DENIED"
          },
          "message": {
            "const": "permission denied to create the file"
          }
        }
      }
      """
    And as "Alice" file "testFolder/testFile.txt" should not exist
    And as "Brian" file "Shares/testFolder/testFile.txt" should not exist


  Scenario: space admin creates a text file in project space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    When user "Alice" creates a file "testFile.txt" inside folder "testFolder" in space "new-space" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should contain these files:
      | testFile.txt |


  Scenario Outline: user with Space Editor/Manager role creates a text file inside shared project space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates a file "testFile.txt" inside folder "testFolder" in space "new-space" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should contain these files:
      | testFile.txt |
    And for user "Brian" folder "testFolder" of the space "new-space" should contain these files:
      | testFile.txt |
    Examples:
      | permissions-role |
      | Space Editor     |
      | Manager          |

  @issue-10126
  Scenario: user with Viewer role tries to create a text file inside shared project space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" tries to create a file "testFile.txt" inside folder "testFolder" in space "new-space" using wopi endpoint
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "PERMISSION_DENIED"
          },
          "message": {
            "const": "permission denied to create the file"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should not contain these files:
      | testFile.txt |
    And for user "Brian" folder "testFolder" of the space "new-space" should not contain these files:
      | testFile.txt |


  Scenario: create a odt file using app endpoint in Personal space
    Given user "Alice" has created folder "testFolder"
    When user "Alice" creates a file "simple.odt" inside folder "testFolder" in space "Personal" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And as "Alice" file "testFolder/simple.odt" should exist


  Scenario: space admin creates a odt file in project space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    When user "Alice" creates a file "simple.odt" inside folder "testFolder" in space "new-space" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should contain these files:
      | simple.odt |


  Scenario Outline: user with Space Editor/Manager role creates a odt file inside shared project space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates a file "simple.odt" inside folder "testFolder" in space "new-space" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should contain these files:
      | simple.odt |
    And for user "Brian" folder "testFolder" of the space "new-space" should contain these files:
      | simple.odt |
    Examples:
      | permissions-role |
      | Space Editor     |
      | Manager          |

  @issue-10126
  Scenario: user with Viewer role tries to create a odt file inside shared project space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" tries to create a file "simple.odt" inside folder "testFolder" in space "new-space" using wopi endpoint
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "PERMISSION_DENIED"
          },
          "message": {
            "const": "permission denied to create the file"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should not contain these files:
      | simple.odt |
    And for user "Brian" folder "testFolder" of the space "new-space" should not contain these files:
      | simple.odt |


  Scenario Outline: sharee with permission Editor/Uploader creates odt file inside shared folder using wopi endpoint
    Given user "Alice" has created folder "testFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder         |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" creates a file "simple.odt" inside folder "testFolder" in space "Shares" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And as "Alice" file "testFolder/simple.odt" should exist
    And as "Brian" file "Shares/testFolder/simple.odt" should exist
    Examples:
      | permissions-role |
      | Editor           |
      | Uploader         |

  @issue-10126
  Scenario: sharee with permission Viewer tries to create odt file inside shared folder using wopi endpoint
    Given user "Alice" has created folder "testFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testFolder |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" tries to create a file "simple.odt" inside folder "testFolder" in space "Shares" using wopi endpoint
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "PERMISSION_DENIED"
          },
          "message": {
            "const": "permission denied to create the file"
          }
        }
      }
      """
    And as "Alice" file "testFolder/simple.odt" should not exist
    And as "Brian" file "Shares/testFolder/simple.odt" should not exist

  @issue-10331
  Scenario Outline: public user with permission edit/upload/createOnly creates odt file inside public folder using wopi endpoint
    Given user "Alice" has created folder "publicFolder"
    And user "Alice" has created the following resource link share:
      | resource        | publicFolder       |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When the public creates a file "simple.odt" inside the last shared public link folder with password "%public%" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And as "Alice" file "publicFolder/simple.odt" should exist
    Examples:
      | permissions-role |
      | edit             |
      | upload           |
      | createOnly       |

  @issue-10126 @issue-10331
  Scenario: public user with permission view tries to creates odt file inside public folder using wopi endpoint
    Given user "Alice" has created folder "publicFolder"
    And user "Alice" has created the following resource link share:
      | resource        | publicFolder |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    When the public tries to create a file "simple.odt" inside the last shared public link folder with password "%public%" using wopi endpoint
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "PERMISSION_DENIED"
          },
          "message": {
            "const": "permission denied to create the file"
          }
        }
      }
      """
    And as "Alice" file "publicFolder/simple.odt" should not exist


  Scenario: user tries to create odt file inside deleted parent folder using wopi endpoint
    Given user "Alice" has created folder "testFolder"
    And user "Alice" has stored id of folder "testFolder"
    And user "Alice" has deleted folder "testFolder"
    When user "Alice" tries to create a file "simple.odt" inside deleted folder using wopi endpoint
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "RESOURCE_NOT_FOUND"
          },
          "message": {
            "const": "the parent container is not accessible or does not exist"
          }
        }
      }
      """

  @issue-8691 @issue-10331
  Scenario Outline: public user with permission edit/upload/createOnly creates odt file inside folder of public space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    And user "Alice" has created the following space link share:
      | space           | new-space          |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When the public creates a file "simple.odt" inside folder "testFolder" in the last shared public link space with password "%public%" using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "file_id"
        ],
        "properties": {
          "file_id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should contain these files:
      | simple.odt |
    Examples:
      | permissions-role |
      | edit             |
      | upload           |
      | createOnly       |

  @issue-8691 @issue-10126 @issue-10331
  Scenario: public user with permission view tries to create odt file inside folder of public space using wopi endpoint
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "testFolder" in space "new-space"
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | permissionsRole | view      |
      | password        | %public%  |
    When the public tries to create a file "simple.odt" inside folder "testFolder" in the last shared public link space with password "%public%" using wopi endpoint
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "code",
          "message"
        ],
        "properties": {
          "code": {
            "const": "PERMISSION_DENIED"
          },
          "message": {
            "const": "permission denied to create the file"
          }
        }
      }
      """
    And for user "Alice" folder "testFolder" of the space "new-space" should not contain these files:
      | simple.odt |


  Scenario Outline: create a file using a template
    Given using spaces DAV path
    And user "Alice" has uploaded file "filesForUpload/<template>" to "<template>"
    And we save it into "TEMPLATEID"
    And user "Alice" has created a file "<target>" using wopi endpoint
    And we save it into "FILEID"
    When user "Alice" sends HTTP method "POST" to URL "<app-endpoint>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "app_url",
          "method",
          "form_parameters"
        ],
        "properties": {
          "app_url": {
            "type": "string",
            "pattern": "^.*\\?WOPISrc=.*wopi%2Ffiles%2F[a-fA-F0-9]{64}(&.*)?$"
          },
          "method": {
            "const": "POST"
          },
          "form_parameters": {
            "type": "object",
            "required": [
              "access_token",
              "access_token_ttl"
            ],
            "properties": {
              "access_token": {
                "type": "string"
              },
              "access_token_ttl": {
                "type": "string"
              }
            }
          }
        }
      }
      """
    Examples:
      | app-endpoint                                                                                | template      | target        |
      | /app/open?file_id=<<FILEID>>&app_name=Collabora&view_mode=write&template_id=<<TEMPLATEID>>  | template.ott  | template.odt  |
      | /app/open?file_id=<<FILEID>>&app_name=OnlyOffice&view_mode=write&template_id=<<TEMPLATEID>> | template.dotx | template.docx |
