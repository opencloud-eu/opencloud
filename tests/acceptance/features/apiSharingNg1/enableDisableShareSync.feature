Feature:  enable or disable sync of incoming shares
  As a user
  I want to have control over the share received
  So that I can filter out the files and folder shared with Me

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario Outline: disable sync of shared resource
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    When user "Brian" disables sync of share "<resource>" using the Graph API
    Then the HTTP status code should be "204"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "value"
        ],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@client.synchronize"
              ],
              "properties": {
                "@client.synchronize": {
                  "const": false
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario Outline: enable sync of shared resource when auto-sync is disabled
    Given user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" enables sync of share "<resource>" offered by "Alice" from "Personal" space using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@client.synchronize"
        ],
        "properties": {
          "@client.synchronize": {
            "const": true
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | folder        |


  Scenario Outline: enable a group share sync by only one user in a group
    Given user "Carol" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has disabled the auto-sync share
    And user "Brian" has disabled the auto-sync share
    And user "Carol" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Carol" has created folder "FolderToShare"
    And user "Carol" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | grp1       |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    When user "Alice" enables sync of share "<resource>" offered by "Carol" from "Personal" space using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@client.synchronize"
        ],
        "properties": {
          "@client.synchronize": {
            "const": true
          }
        }
      }
      """
    And user "Alice" should have sync enabled for share "<resource>"
    And user "Brian" should have sync disabled for share "<resource>"
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario Outline: disable group share sync by only one user in a group
    Given user "Carol" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has created folder "FolderToShare"
    And user "Carol" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | grp1       |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    And user "Alice" has a share "<resource>" synced
    When user "Alice" disables sync of share "<resource>" using the Graph API
    Then the HTTP status code should be "204"
    And user "Alice" should have sync disabled for share "<resource>"
    And user "Brian" should have sync enabled for share "<resource>"
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario Outline: enable sync of shared resource from project space
    Given user "Brian" has disabled the auto-sync share
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" enables sync of share "<resource>" offered by "Alice" from "NewSpace" space using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@client.synchronize"
        ],
        "properties": {
          "@client.synchronize": {
            "const": true
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario Outline: disable sync of shared resource from project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    When user "Brian" disables sync of share "<resource>" using the Graph API
    Then the HTTP status code should be "204"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "value"
        ],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@client.synchronize"
              ],
              "properties": {
                "@client.synchronize": {
                  "const": false
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario Outline: enable a group share sync shared from Project Space by only one user in a group
    Given user "Carol" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Carol" using the Graph API
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has disabled the auto-sync share
    And user "Brian" has disabled the auto-sync share
    And user "Carol" has created a space "NewSpace" with the default quota using the Graph API
    And user "Carol" has created a folder "FolderToShare" in space "NewSpace"
    And user "Carol" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | grp1       |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    When user "Alice" enables sync of share "<resource>" offered by "Carol" from "NewSpace" space using the Graph API
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@client.synchronize"
        ],
        "properties": {
          "@client.synchronize": {
            "const": true
          }
        }
      }
      """
    And user "Alice" should have sync enabled for share "<resource>"
    And user "Brian" should have sync disabled for share "<resource>"
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario Outline: disable group share sync shared from Project space by only one user in a group
    Given user "Carol" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Carol" using the Graph API
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Carol" has created a space "NewSpace" with the default quota using the Graph API
    And user "Carol" has created a folder "FolderToShare" in space "NewSpace"
    And user "Carol" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | grp1       |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    And user "Alice" has a share "<resource>" synced
    When user "Alice" disables sync of share "<resource>" using the Graph API
    Then the HTTP status code should be "204"
    And user "Alice" should have sync disabled for share "<resource>"
    And user "Brian" should have sync enabled for share "<resource>"
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |


  Scenario: try to enable share sync of a non-existent resource
    Given user "Brian" has disabled the auto-sync share
    When user "Brian" tries to enable share sync of a resource "nonexistent" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "no shares found"
              }
            }
          }
        }
      }
      """


  Scenario: try to enable share sync with empty resource id
    Given user "Brian" has disabled the auto-sync share
    When user "Brian" tries to enable share sync of a resource "" using the Graph API
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "invalidRequest"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "invalid id"
              }
            }
          }
        }
      }
      """


  Scenario: try to enable share sync with not shared resource id
    Given user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded file with content "some data" to "/fileNotShared.txt"
    And we save it into "FILEID"
    When user "Brian" tries to enable share sync of a resource "<<FILEID>>" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "no shares found"
              }
            }
          }
        }
      }
      """


  Scenario: try to enable sync of shared resource from Personal Space when sharer is deleted
    Given user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" tries to enable share sync of a resource "<<FILEID>>" using the Graph API
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "invalidRequest"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "converting to drive items failed"
              }
            }
          }
        }
      }
      """


  Scenario: try to disable sync of shared resource from Personal Space when sharer is deleted
    Given user "Brian" has disabled the auto-sync share
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" tries to disable sync of share "textfile0.txt" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "no shares found"
              }
            }
          }
        }
      }
      """


  Scenario: enable sync of shared resource from Project Space when sharer is deleted
    Given user "Brian" has disabled the auto-sync share
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | NewSpace      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" enables share sync of a resource "<<FILEID>>" using the Graph API
    Then the HTTP status code should be "201"
    And user "Brian" should have sync enabled for share "textfile0.txt"


  Scenario: disable sync of shared resource from Project Space when sharer is deleted
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | NewSpace      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" disables sync of share "textfile0.txt" using the Graph API
    Then the HTTP status code should be "204"
    And user "Brian" should have sync disabled for share "textfile0.txt"


  Scenario: try to enable sync of group shared resource when sharer is deleted
    Given user "Brian" has disabled the auto-sync share
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And group "grp1" has been deleted
    When user "Brian" tries to enable share sync of a resource "<<FILEID>>" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "no shares found"
              }
            }
          }
        }
      }
      """


  Scenario: try to disable sync of group shared resource when sharer is deleted
    Given user "Brian" has disabled the auto-sync share
    And group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And we save it into "FILEID"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And group "grp1" has been deleted
    When user "Brian" tries to disable sync of share "textfile0.txt" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "error getting received share"
              }
            }
          }
        }
      }
      """

  @issue-9001
  Scenario: try to disable share sync of a non-existent resource
    When user "Brian" tries to disable share sync of a resource "nonexistent" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "error getting received share"
              }
            }
          }
        }
      }
      """


  Scenario: try to disable share sync with empty resource id
    When user "Brian" tries to disable share sync of a resource "" using the Graph API
    Then the HTTP status code should be "400"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "invalidRequest"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "invalid driveID or itemID"
              }
            }
          }
        }
      }
      """


  Scenario: try to disable share sync with not shared resource id
    Given user "Alice" has uploaded file with content "some data" to "/fileNotShared.txt"
    And we save it into "FILEID"
    When user "Brian" tries to disable share sync of a resource "<<FILEID>>" using the Graph API
    Then the HTTP status code should be "404"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "itemNotFound"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "error getting received share"
              }
            }
          }
        }
      }
      """

  @issue-8876
  Scenario Outline: try to enable sync of already synced share
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    When user "Brian" enables sync of share "<resource>" offered by "Alice" from "Personal" space using the Graph API
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "nameAlreadyExists"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "shares already mounted"
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | folder        |

  @issue-8876
  Scenario Outline: try to disable sync of already unsynced share
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    And user "Brian" has disabled sync of last shared resource
    When user "Brian" disables sync of share "<resource>" using the Graph API
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "nameAlreadyExists"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "shares already unmounted"
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | folder        |

  @issue-8876
  Scenario Outline: try to enable sync of already synced share shared from Project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    When user "Brian" enables sync of share "<resource>" offered by "Alice" from "NewSpace" space using the Graph API
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "nameAlreadyExists"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "shares already mounted"
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |

  @issue-8876
  Scenario Outline: try to disable sync of already unsynced share shared from Project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has a share "<resource>" synced
    And user "Brian" has disabled sync of last shared resource
    And user "Brian" disables sync of share "<resource>" using the Graph API
    Then the HTTP status code should be "409"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["error"],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "innererror",
              "message"
            ],
            "properties": {
              "code" : {
                "const": "nameAlreadyExists"
              },
              "innererror" : {
                "type": "object",
                "required": [
                  "date",
                  "request-id"
                ]
              },
              "message" : {
                "const": "shares already unmounted"
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | textfile0.txt |
      | FolderToShare |
