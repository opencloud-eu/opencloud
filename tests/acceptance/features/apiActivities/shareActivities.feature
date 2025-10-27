Feature: check share activity
  As a user
  I want to check who shared which file to whom
  So that I can track activities of a file

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"


  Scenario: check activities after adding share to a file
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Alice" lists the activities of file "textfile.txt" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} with {sharee}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource","sharee","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
                                },
                                "displayName": {
                                  "const": "Brian"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: check activities after removing share from a file
    Given user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has removed the access of user "Brian" from resource "textfile.txt" of space "Personal"
    When user "Alice" lists the activities of file "textfile.txt" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} with {sharee}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} removed {sharee} from {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource","sharee","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: check link creation activity for a file
    Given user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    When user "Alice" lists the activities of file "textfile.txt" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} via link"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: check link deletion activity for a file
    Given user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has removed the last link share of file "textfile.txt" from space "Personal"
    When user "Alice" lists the activities of file "textfile.txt" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} via link"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} removed link to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @owncloud-issue-9712
  Scenario: sharer checks sharee's activities
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "FOLDER" synced
    And user "Brian" has uploaded file with content "some data" to "Shares/FOLDER/newfile.txt"
    And user "Brian" has uploaded file with content "edited data" to "Shares/FOLDER/newfile.txt"
    And user "Brian" has deleted file "Shares/FOLDER/newfile.txt"
    When user "Alice" lists the activities of file "FOLDER" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 5,
            "maxItems": 5,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "Alice Hansen"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} with {sharee}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "newfile.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian Murphy"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {resource} in {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "newfile.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian Murphy"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} deleted {resource} from {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "newfile.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian Murphy"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: check add member to space activity
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    When user "Alice" lists the activities of space "new-space" using the Graph API
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
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {sharee} as member of {space}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["sharee","space","user"],
                          "properties": {
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian"
                                }
                              }
                            },
                            "space": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: check remove member from space activity
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    And user "Alice" has removed the access of user "Brian" from space "new-space"
    When user "Alice" lists the activities of space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {sharee} as member of {space}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["sharee","space","user"],
                          "properties": {
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian"
                                }
                              }
                            },
                            "space": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} removed {sharee} from {space}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["sharee","space","user"],
                          "properties": {
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Brian"
                                }
                              }
                            },
                            "space": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @owncloud-issue-10012
  Scenario: check link share activities of a project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | permissionsRole | view      |
      | password        | %public%  |
    And user "Alice" has updated the last resource link share with
      | space              | new-space                |
      | permissionsRole    | edit                     |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    And user "Alice" has set the following password for the last link share:
      | resource |               |
      | space    | new-space     |
      | password | 6a0Q;A3 +i^m[ |
    When user "Alice" lists the activities of space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 4,
            "maxItems": 4,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} via link"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for a link {token} on {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","resource","token","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "expiration date"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "token": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "type": "string"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for a link {token} on {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","resource","token","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "permission"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "token": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "type": "string"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for a link {token} on {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","resource","token","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "password"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "token": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "type": "string"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @owncloud-issue-10011 @owncloud-issue-10228
  Scenario: check share update activities of a folder from a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | new-space     |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole    | Editor                   |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
      | space              | new-space                |
      | resource           | folderToShare            |
    When user "Alice" lists the activities of folder "folderToShare" from space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} with {sharee}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","sharee","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "folderToShare"
                                }
                              }
                            },
                            "sharee": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "displayName": {
                                  "const": "Brian"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for the {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","folder","resource","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "expiration date, permission"
                                }
                              }
                            },
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "folderToShare"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"]
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @owncloud-issue-10012
  Scenario: check link share activities of a file from a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | new-space    |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has updated the last resource link share with
      | resource           | textfile.txt             |
      | space              | new-space                |
      | permissionsRole    | edit                     |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    And user "Alice" has set the following password for the last link share:
      | resource | textfile.txt  |
      | space    | new-space     |
      | password | 6a0Q;A3 +i^m[ |
    When user "Alice" lists the activities of file "textfile.txt" from space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 5,
            "maxItems": 5,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} via link"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for a link {token} on {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","resource","token","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "expiration date"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "token": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "type": "string"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for a link {token} on {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","resource","token","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "permission"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "token": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "type": "string"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {field} for a link {token} on {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["field","resource","token","user"],
                          "properties": {
                            "field": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "password"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "token": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "type": "string"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @owncloud-issue-10012
  Scenario: public tries to check link share activities of a project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | new-space |
      | permissionsRole | view      |
      | password        | %public%  |
    And user "Alice" has updated the last resource link share with
      | space              | new-space                |
      | permissionsRole    | edit                     |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    And user "Alice" has set the following password for the last link share:
      | resource |               |
      | space    | new-space     |
      | password | 6a0Q;A3 +i^m[ |
    When the public tries to check the activities of space "new-space" owned by user "Alice" with password "%public%" using the Graph API
    Then the HTTP status code should be "401"

  @owncloud-issue-10012
  Scenario: public tries to check link share activities of a project space file
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | new-space    |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has updated the last resource link share with
      | resource           | textfile.txt             |
      | space              | new-space                |
      | permissionsRole    | edit                     |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    And user "Alice" has set the following password for the last link share:
      | resource | textfile.txt  |
      | space    | new-space     |
      | password | 6a0Q;A3 +i^m[ |
    When the public tries to check the activities of file "textfile.txt" from space "new-space" owned by user "Alice" with password "%public%" using the Graph API
    Then the HTTP status code should be "401"

  @owncloud-issue-10012
  Scenario: public tries to check link share activities of a project space folder
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "project-folder" in space "new-space"
    And user "Alice" has created the following resource link share:
      | resource        | project-folder |
      | space           | new-space      |
      | permissionsRole | view           |
      | password        | %public%       |
    And user "Alice" has updated the last resource link share with
      | resource           | project-folder           |
      | space              | new-space                |
      | permissionsRole    | edit                     |
      | expirationDateTime | 2200-07-15T14:00:00.000Z |
    And user "Alice" has set the following password for the last link share:
      | resource | project-folder |
      | space    | new-space      |
      | password | 6a0Q;A3 +i^m[  |
    When the public tries to check the activities of folder "project-folder" from space "new-space" owned by user "Alice" with password "%public%" using the Graph API
    Then the HTTP status code should be "401"


  Scenario: sharee tries to check the activities of a shared folder using share mount-point id
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /FOLDER  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "/FOLDER" synced
    And user "Brian" has uploaded file with content "some data" to "Shares/FOLDER/newfile.txt"
    And user "Brian" has uploaded file with content "edited data" to "Shares/FOLDER/newfile.txt"
    And user "Brian" has deleted file "Shares/FOLDER/newfile.txt"
    When user "Brian" tries to list the activities of folder "FOLDER" with share mount-point id using the Graph API
    Then the HTTP status code should be "403"

  @owncloud-issue-9849
  Scenario: sharee tries to check the activities of a shared folder using file-id
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded file with content "some data" to "FOLDER/newfile.txt"
    And user "ALice" has uploaded file with content "edited data" to "FOLDER/newfile.txt"
    And user "ALice" has deleted file "FOLDER/newfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | /FOLDER  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Brian" has a share "/FOLDER" synced
    When user "Brian" lists the activities of folder "FOLDER" from space "Shares" using the Graph API
    Then the HTTP status code should be "403"

  @owncloud-issue-9860
  Scenario: sharee tries to check the activities of unshared file
    Given user "Alice" has uploaded file with content "another OpenCloud test text file" to "anotherTextfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Brian" tries to list the activities of file "anotherTextfile.txt" from space "Personal" owned by user "Alice" using the Graph API
    Then the HTTP status code should be "403"

  @owcnloud-issue-9676 @owncloud-issue-10331
  Scenario: user checks public activities of a link shared file
    Given using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | edit         |
      | password        | %public%     |
    And the public has uploaded file "textfile.txt" with content "public test" and password "%public%" to the last link share using the public WebDAV API
    When user "Alice" lists the activities of file "textfile.txt" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} via link"
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {resource} in {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "name": {
                                  "const": "Alice Hansen"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%file_id_pattern%"
                                },
                                "displayName": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "[a-zA-z]{15}"
                                },
                                "displayName": {
                                  "const": "Public"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @owncloud-issue-9676 @owncloud-issue-10331
  Scenario: user checks public activities of a link shared folder
    Given using SharingNG
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has created the following resource link share:
      | resource        | /FOLDER  |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    And the public has uploaded file "text.txt" with content "added by public user" and password "%public%" to the last link share using the public WebDAV API
    And the public has uploaded file "text.txt" with content "updated by public user" and password "%public%" to the last link share using the public WebDAV API
    And the public has deleted file "text.txt" from the last link share with password "%public%" using the public WebDAV API
    When user "Alice" lists the activities of file "/FOLDER" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 5,
            "maxItems": 5,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "Alice Hansen"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} shared {resource} via link"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource","user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "%user_id_pattern%"
                                },
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "text.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "[a-zA-z]{15}"
                                },
                                "displayName": {
                                  "const": "Public"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {resource} in {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "text.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "[a-zA-z]{15}"
                                },
                                "displayName": {
                                  "const": "Public"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id","template","times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} deleted {resource} from {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder","resource","user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "name": {
                                  "const": "text.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id","displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "[a-zA-z]{15}"
                                },
                                "displayName": {
                                  "const": "Public"
                                }
                              }
                            }
                          }
                        }
                      }
                    },
                    "times": {
                      "type": "object",
                      "required": ["recordedTime"],
                      "properties": {
                        "recordedTime": {
                          "type": "string",
                          "format": "date-time"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """
