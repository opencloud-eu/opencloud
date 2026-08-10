@notification
Feature: Disable notifications for individual events
  As a user
  I want to disable notifications for specific events
  So that I do not get notified of unimportant events

  | Event                    | Event description                                                  | In-App | Mail |
  | Share Received           | Notify when I have received a share                                |   x    |  x   |
  | Share Removed            | Notify when a received share has been removed                      |   x    |      |
  | Share Expired            | Notify when a received share has expired                           |   x    |  x   |
  | Added as space member    | Notify when I have been added as a member to a space               |   x    |  x   |
  | Removed as space member  | Notify when I have been removed as member from a space             |   x    |  x   |
  | Space membership expired | Notify when a space membership has expired                         |   x    |  x   |
  | Space disabled           | Notify when a space I am member of has been disabled               |   x    |      |
  | Space deleted            | Notify when a space I am member of has been deleted                |   x    |      |
  | File rejected            | Notify when a file I uploaded was rejected (virus/policy violation)|   x    |      |
  # "File rejected" (subject "Virus found") notification is covered in apiAntivirus/antivirus.feature

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "some data" to "lorem.txt"

  @email
  Scenario: disable mail and in-app notification for "Share Received" event
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Share Received |
      | notificationTypes | mail,in-app    |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{ "const": "event-share-created-options" }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Brian" should have "0" emails
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And the notifications should be empty

  @email
  Scenario: disable mail and in-app notification for "Share Removed" event
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Share Removed |
      | notificationTypes | mail,in-app  |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{ "const": "event-share-removed-options" }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has removed the access of user "Brian" from resource "lorem.txt" of space "Personal"
    And user "Brian" should have "1" emails
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                |
      | Alice Hansen shared lorem.txt with you |
    But user "Brian" should not have a notification related to resource "lorem.txt" with subject "Resource unshared"

  @email
  Scenario: disable mail and in-app notification for "Share Removed" event (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "newSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "newSpace" with content "some content" to "insideSpace.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | insideSpace.txt |
      | space           | newSpace        |
      | sharee          | Brian           |
      | shareType       | user            |
      | permissionsRole | Viewer          |
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Share Removed |
      | notificationTypes | mail,in-app  |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{ "const": "event-share-removed-options" }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%user_id_pattern%" },
                  "bundleId":{ "pattern":"%user_id_pattern%" },
                  "settingId":{ "pattern":"%user_id_pattern%" },
                  "accountUuid":{ "pattern":"%user_id_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has removed the access of user "Brian" from resource "insideSpace.txt" of space "newSpace"
    And user "Brian" should have "1" emails
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                      |
      | Alice Hansen shared insideSpace.txt with you |
    But user "Brian" should not have a notification related to resource "insideSpace.txt" with subject "Resource unshared"

  @email
  Scenario: disable mail and in-app notification for "Share Expired" event
    Given using SharingNG
    And user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Share Expired |
      | notificationTypes | mail,in-app   |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{
                    "const": "event-share-expired-options"
                  }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """

  @email
  Scenario Outline: no in-app and mail notification should appear when "Share Expired" event is disabled (Personal space)
    Given using SharingNG
    And user "Alice" has created folder "my_data"
    And user "Alice" has uploaded file with content "hello world" to "lorem.txt"
    And user "Brian" has disabled notification for the following event using the settings API:
      | event             | Share Expired |
      | notificationTypes | mail,in-app   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Alice" expires the last share of resource "<resource>" inside of the space "Personal"
    Then the HTTP status code should be "200"
    And user "Brian" should have "1" emails
    And user "Brian" should have received the following email from user "Alice"
      """
      Hello Brian Murphy

      %displayname% has shared "<resource>" with you.

      Click here to view it: %base_url%/files/shares/with-me
      """
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                 |
      | Alice Hansen shared <resource> with you |
    But user "Brian" should not have a notification related to space "Alice Hansen" with subject "Share expired"
    Examples:
      | resource  |
      | lorem.txt |
      | my_data   |

  @email
  Scenario Outline: no in-app and mail notification should appear when "Share Expired" event is disabled (Project space)
    Given using spaces DAV path
    And using SharingNG
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "uploadFolder" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has disabled notification for the following event using the settings API:
      | event             | Share Expired |
      | notificationTypes | mail,in-app   |
    When user "Alice" expires the last share of resource "<resource>" inside of the space "NewSpace"
    Then the HTTP status code should be "200"
    And user "Brian" should have "1" emails
    And user "Brian" should have received the following email from user "Alice" about the share of project space "NewSpace"
      """
      Hello Brian Murphy

      %displayname% has shared "<resource>" with you.

      Click here to view it: %base_url%/files/shares/with-me
      """
    And user "Brian" should get a notification with subject "Resource shared" and message:
      | message                                 |
      | Alice Hansen shared <resource> with you |
    But user "Brian" should not have a notification related to space "NewSpace" with subject "Share expired"
    Examples:
      | resource     |
      | lorem.txt    |
      | uploadFolder |

  @email
  Scenario: disable mail and in-app notification for "Added as space member" event
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "newSpace" with the default quota using the Graph API
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Added As Space Member |
      | notificationTypes | mail,in-app           |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{ "const": "event-space-shared-options" }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has sent the following space share invitation:
      | space           | newSpace     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And the notifications should be empty
    And user "Brian" should have "0" emails

  @email
  Scenario: disable mail and in-app notification for "Removed as space member" event
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "newSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | newSpace     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Removed As Space Member |
      | notificationTypes | mail,in-app             |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{ "const": "event-space-unshared-options" }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has removed the access of user "Brian" from space "newSpace"
    And user "Brian" should get a notification with subject "Space shared" and message:
      | message                                  |
      | Alice Hansen added you to Space newSpace |
    But user "Brian" should not have a notification related to space "newSpace" with subject "Removed from Space"
    And user "Brian" should have "1" emails
    And user "Brian" should have received the following email from user "Alice" about the share of project space "newSpace"
      """
      Hello Brian Murphy,

      %displayname% has invited you to join "newSpace".

      Click here to view it: %base_url%/f/%space_id%
      """

  @email
  Scenario: disable mail and in-app notification for "Space Membership Expired" event
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "newSpace" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | newSpace     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Space Membership Expired |
      | notificationTypes | mail,in-app              |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{ "const": "event-space-membership-expired-options" }
                }
              },
              "value":{
                "type": "object",
                "required": ["id","bundleId","settingId","accountUuid","resource","collectionValue"],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 2,
                        "minItems": 2,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "mail" },
                                "boolValue":{ "const": false }
                              }
                            },
                            {
                              "type": "object",
                              "required": ["key","boolValue"],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    When user "Alice" expires the user share of space "newSpace" for user "Brian"
    Then the HTTP status code should be "200"
    And user "Brian" should get a notification with subject "Space shared" and message:
      | message                                  |
      | Alice Hansen added you to Space newSpace |
    But user "Brian" should not have a notification related to space "newSpace" with subject "Membership expired"
    And user "Brian" should have "1" emails
    And user "Brian" should have received the following email from user "Alice" about the share of project space "newSpace"
      """
      Hello Brian Murphy,

      %displayname% has invited you to join "newSpace".

      Click here to view it: %base_url%/f/%space_id%
      """


  Scenario: disable in-app notification for "Space disabled" event (note: no mail notification)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Space Disabled |
      | notificationTypes | in-app         |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier","value"],
            "properties": {
              "identifier":{
                "type": "object",
                "required": ["extension","bundle","setting"],
                "properties": {
                  "extension":{ "const": "opencloud-accounts" },
                  "bundle":{ "const": "profile" },
                  "setting":{
                    "const": "event-space-disabled-options"
                  }
                }
              },
              "value":{
                "type": "object",
                "required": [
                  "id",
                  "bundleId",
                  "settingId",
                  "accountUuid",
                  "resource",
                  "collectionValue"
                ],
                "properties":{
                  "id":{ "pattern":"%uuidv4_pattern%" },
                  "bundleId":{ "pattern":"%uuidv4_pattern%" },
                  "settingId":{ "pattern":"%uuidv4_pattern%" },
                  "accountUuid":{ "pattern":"%uuidv4_pattern%" },
                  "resource":{
                    "type": "object",
                    "required":["type"],
                    "properties": {
                      "type":{ "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue":{
                    "type": "object",
                    "required":["values"],
                    "properties": {
                      "values":{
                        "type": "array",
                        "maxItems": 1,
                        "minItems": 1,
                        "uniqueItems": true,
                        "items": {
                          "oneOf": [
                            {
                              "type": "object",
                              "required": [
                                "key",
                                "boolValue"
                              ],
                              "properties": {
                                "key":{ "const": "in-app" },
                                "boolValue":{ "const": false }
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has disabled a space "new-space"
    And user "Brian" should get a notification with subject "Space shared" and message:
      | message                                   |
      | Alice Hansen added you to Space new-space |
    But user "Brian" should not have a notification related to space "new-space" with subject "Space disabled"


  Scenario: disable in-app notification for "Space Deleted" event
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Brian" disables notification for the following event using the settings API:
      | event             | Space Deleted |
      | notificationTypes | in-app        |
    Then the HTTP status code should be "201"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "object",
            "required": ["identifier", "value"],
            "properties": {
              "identifier": {
                "type": "object",
                "required": ["extension", "bundle", "setting"],
                "properties": {
                  "extension": { "const": "opencloud-accounts" },
                  "bundle": { "const": "profile" },
                  "setting": { "const": "event-space-deleted-options" }
                }
              },
              "value": {
                "type": "object",
                "required": ["id", "bundleId", "settingId", "accountUuid", "resource", "collectionValue"],
                "properties": {
                  "id": { "pattern": "%user_id_pattern%" },
                  "bundleId": { "pattern": "%user_id_pattern%" },
                  "settingId": { "pattern": "%user_id_pattern%" },
                  "accountUuid": { "pattern": "%user_id_pattern%" },
                  "resource": {
                    "type": "object",
                    "required": ["type"],
                    "properties": {
                      "type": { "const": "TYPE_USER" }
                    }
                  },
                  "collectionValue": {
                    "type": "object",
                    "required": ["values"],
                    "properties": {
                      "values": {
                        "type": "array",
                        "minItems": 1,
                        "maxItems": 1,
                        "items": {
                          "type": "object",
                          "required": ["key", "boolValue"],
                          "properties": {
                            "key": { "const": "in-app" },
                            "boolValue": { "const": false }
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    And user "Alice" has disabled a space "new-space"
    And user "Alice" has deleted a space "new-space"
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And user "Brian" should not have any notification
