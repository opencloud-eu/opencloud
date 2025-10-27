Feature: check activities
  As a user
  I want to check who made which changes to files using file-id
  So that I can track modifications

  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path


  Scenario: check copy activity of destination resources
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has created folder "newFolder"
    And user "Alice" has copied file with id "<<FILEID>>" as "textfile.txt" into folder "newFolder" inside space "Personal"
    When user "Alice" lists the activities of file "newFolder/textfile.txt" from space "Personal" using the Graph API
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
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "newFolder"
                                }
                              }
                            },
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
    When user "Alice" lists the activities of folder "newFolder" from space "Personal" using the Graph API
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
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
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
                                  "const": "newFolder"
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
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "newFolder"
                                }
                              }
                            },
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


  Scenario: check edit activity
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has updated a file with content "updated content" using file-id "<<FILEID>>"
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} updated {resource} in {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
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
                              "required": ["id", "name"],
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

  @owncloud-issue-9744
  Scenario: check rename activity
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has renamed file with id "<<FILEID>>" to "renamed.txt" inside space "Personal"
    When user "Alice" lists the activities of file "renamed.txt" from space "Personal" using the Graph API
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["oldResource", "resource", "user"],
                          "properties": {
                            "oldResource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "id": {
                                  "const": ""
                                },
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
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


  Scenario: check activities of destination file and folder after moving a file using file-id
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has created folder "New Folder"
    And user "Alice" has moved file with id "<<FILEID>>" as "textfile.txt" into folder "New Folder" inside space "Personal"
    When user "Alice" lists the activities of file "New Folder/textfile.txt" from space "Personal" using the Graph API
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
                          "const": "{user} moved {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "New Folder"
                                }
                              }
                            },
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
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" using the Graph API
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
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "New Folder"
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
                          "const": "{user} moved {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "New Folder"
                                }
                              }
                            },
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


  Scenario: check activities of destination file and folder after moving a 0 byte file using file-id
    Given user "Alice" has uploaded file "filesForUpload/zerobyte.txt" to "zerobyte.txt"
    And we save it into "FILEID"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has moved file with id "<<FILEID>>" as "zerobyte.txt" into folder "FOLDER" inside space "Personal"
    When user "Alice" lists the activities of file "FOLDER/zerobyte.txt" from space "Personal" using the Graph API
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
                          "const": "{user} moved {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
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
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "zerobyte.txt"
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
    When user "Alice" lists the activities of folder "FOLDER" from space "Personal" using the Graph API
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
                          "const": "{user} moved {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
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
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "zerobyte.txt"
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


  Scenario: check activities of destination file and folder after moving a file by renaming destination file using file-id
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file with id "<<FILEID>>" as "renamed.txt" into folder "FOLDER" inside space "Personal"
    When user "Alice" lists the activities of file "FOLDER/renamed.txt" from space "Personal" using the Graph API
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "oldResource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
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
    When user "Alice" lists the activities of folder "FOLDER" from space "Personal" using the Graph API
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "oldResource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "textfile.txt"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
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


  Scenario: check activities of destination file and folder after moving a 0 byte file by renaming destination file using file-id
    Given user "Alice" has uploaded file "filesForUpload/zerobyte.txt" to "/zerobyte.txt"
    And we save it into "FILEID"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file with id "<<FILEID>>" as "renamed.txt" into folder "FOLDER" inside space "Personal"
    When user "Alice" lists the activities of file "FOLDER/renamed.txt" from space "Personal" using the Graph API
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "oldResource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "zerobyte.txt"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
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
    When user "Alice" lists the activities of folder "FOLDER" from space "Personal" using the Graph API
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "oldResource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "zerobyte.txt"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%user_id_pattern%$"
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


  Scenario: check activities of destination resources after copying a file by renaming the destination file
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has created folder "newFolder"
    And user "Alice" has copied file with id "<<FILEID>>" as "renamed.txt" into folder "newFolder" inside space "Personal"
    When user "Alice" lists the activities of file "newFolder/renamed.txt" from space "Personal" using the Graph API
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
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "newFolder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
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
    When user "Alice" lists the activities of folder "newFolder" from space "Personal" using the Graph API
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
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
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
                                  "const": "newFolder"
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
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "newFolder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id","name"],
                              "properties": {
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
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


  Scenario: check activities of destination resources after copying file into same folder by renaming the destination file
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "FOLDER/textfile.txt"
    And we save it into "FILEID"
    And user "Alice" has copied file with id "<<FILEID>>" as "renamed.txt" into folder "FOLDER" inside space "Personal"
    When user "Alice" lists the activities of file "FOLDER/renamed.txt" from space "Personal" using the Graph API
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
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
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
                                "id": {
                                  "type": "string",
                                  "pattern": "^%file_id_pattern%$"
                                },
                                "name": {
                                  "const": "renamed.txt"
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
            "minItems": 3,
            "maxItems": 3,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "Alice Hansen"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
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
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
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
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["id", "template", "times"],
                  "properties": {
                    "template": {
                      "type": "object",
                      "required": ["message", "variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "renamed.txt"
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
