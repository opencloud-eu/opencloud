Feature: check activities
  As a user
  I want to check who made which changes to files
  So that I can track modifications

  Background:
    Given user "Alice" has been created with default attributes

  @issue-9712
  Scenario: check activities after uploading a file and a folder
    Given user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile.txt"
    And user "Alice" has created folder "/FOLDER"
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
            "minItems": 1,
            "maxItems": 1,
            "items": {
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
            "minItems": 1,
            "maxItems": 1,
            "items": {
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
                          "required": ["name"],
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
                            "id": {
                              "type": "string",
                              "pattern": "%file_id_pattern%"
                            },
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
          }
        }
      }
      """

  @issue-10001
  Scenario: check edit activity of a file
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "/textfile.txt"
    And user "Alice" has uploaded file with content "edited OpenCloud test text file" to "/textfile.txt"
    When user "Alice" lists the activities of file "/textfile.txt" from space "Personal" using the Graph API
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

  @issue-9712
  Scenario: check activities after deleting a file and a folder
    Given user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has deleted file "textfile.txt"
    And user "Alice" has deleted folder "FOLDER"
    When user "Alice" lists the activities of personal space using the Graph API
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
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource", "folder", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "textfile.txt"
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
                    "template": {
                      "type": "object",
                      "required": ["message","variables"],
                      "properties": {
                        "message": {
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource", "folder", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
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
                          "required": ["resource","folder","user"],
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
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "Alice Hansen"
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
                          "required": ["resource","folder","user"],
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
                                  "const": "FOLDER"
                                }
                              }
                            },
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "Alice Hansen"
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

  @issue-9712
  Scenario: check move activity for a folder
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has created folder "/New Folder"
    And user "Alice" has moved folder "FOLDER" to "New Folder/FOLDER"
    When user "Alice" lists the activities of folder "New Folder/FOLDER" from space "Personal" using the Graph API
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


  Scenario: check activities of destination file and folder after moving a file into it
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file "textfile.txt" to "FOLDER/textfile.txt"
    When user "Alice" lists the activities of file "FOLDER/textfile.txt" from space "Personal" using the Graph API
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
                              "required": ["id","name"],
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


  Scenario: check activities of destination file and folder after moving a 0 byte file into it
    Given user "Alice" has uploaded file "filesForUpload/zerobyte.txt" to "/zerobyte.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file "zerobyte.txt" to "FOLDER/zerobyte.txt"
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


  Scenario: check activities of destination file and folder after moving a file by renaming destination file
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file "textfile.txt" to "FOLDER/renamed.txt"
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
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
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
                            "oldResource": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "textfile.txt"
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
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
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
                            "oldResource": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "textfile.txt"
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


  Scenario: check activities of destination file and folder after moving a 0 byte file by renaming destination file
    Given user "Alice" has uploaded file "filesForUpload/zerobyte.txt" to "/zerobyte.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file "zerobyte.txt" to "FOLDER/textfile.txt"
    When user "Alice" lists the activities of file "FOLDER/textfile.txt" from space "Personal" using the Graph API
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
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
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
                            "oldResource": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "zerobyte.txt"
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
                          "const": "{user} renamed {oldResource} to {resource}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "oldResource", "resource", "user"],
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
                            "oldResource": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "zerobyte.txt"
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

  @issue-9712
  Scenario: check rename activity for a file and a folder
    Given user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/textfile.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has moved file "textfile.txt" to "renamed.txt"
    And user "Alice" has moved folder "/FOLDER" to "RENAMED FOLDER"
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
    When user "Alice" lists the activities of folder "RENAMED FOLDER" from space "Personal" using the Graph API
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
                                  "const": "FOLDER"
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
                                  "const": "RENAMED FOLDER"
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

  @issue-9712
  Scenario: check activities of a folder
    Given user "Alice" has created folder "/New Folder"
    And user "Alice" has created folder "/New Folder/Folder"
    And user "Alice" has created folder "/New Folder/Sub Folder"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/New Folder/textfile.txt"
    And user "Alice" has moved file "/New Folder/textfile.txt" to "/New Folder/Sub Folder/textfile.txt"
    And user "Alice" has moved folder "/New Folder/Folder" to "/New Folder/Sub Folder/Folder"
    And user "Alice" has moved file "/New Folder/Sub Folder/textfile.txt" to "/New Folder/Sub Folder/renamed.txt"
    And user "Alice" has moved folder "/New Folder/Sub Folder/Folder" to "/New Folder/Sub Folder/Renamed Folder"
    And user "Alice" has deleted file "/New Folder/Sub Folder/renamed.txt"
    And user "Alice" has deleted folder "/New Folder/Sub Folder/Renamed Folder"
    When user "Alice" lists the activities of folder "/New Folder" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems": 10,
            "maxItems": 10,
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
                                "id": {
                                  "type": "string"
                                },
                                "name": {
                                  "const": "New Folder"
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
                          "required": ["resource", "folder", "user"],
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
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "Folder"
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
                                  "const": "New Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Sub Folder"
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
                                  "const": "New Folder"
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
                                  "const": "Sub Folder"
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
                                  "const": "Sub Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Folder"
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
                                "name": {
                                  "const": "textfile.txt"
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
                                "name": {
                                  "const": "Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Renamed Folder"
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
                          "const": "{user} deleted {resource} from {folder}"
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
                                  "const": "Sub Folder"
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
                          "const": "{user} deleted {resource} from {folder}"
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
                                  "const": "Sub Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Renamed Folder"
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
                }
              ]
            }
          }
        }
      }
      """

  @issue-10210
  Scenario: check activities of a folder after renaming
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "FOLDER/textfile.txt"
    And user "Alice" has moved folder "FOLDER" to "RENAMED FOLDER"
    And user "Alice" has uploaded file with content "updated OpenCloud test text file" to "RENAMED FOLDER/textfile.txt"
    When user "Alice" lists the activities of file "RENAMED FOLDER" from space "Personal" using the Graph API
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
                          "const": "{user} added {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "FOLDER"
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
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
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
                              "required": ["id", "name"],
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
                                  "const": "Alice Hansen"
                                }
                              }
                            },
                            "oldResource": {
                              "type": "object",
                              "required": ["id", "name"],
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
                                  "const": "RENAMED FOLDER"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
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
                          "const": "{user} updated {resource} in {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["folder", "resource", "user"],
                          "properties": {
                            "folder": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "RENAMED FOLDER"
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
                              "required": ["id", "displayName"],
                              "properties": {
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

  @issue-9856 @issue-10127 @issue-369
  Scenario: check activity message with different language
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has switched the system language to "de" using the Graph API
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
            "minItems": 1,
            "maxItems": 1,
            "items": {
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
                      "const": "{user} hat {resource} zu {folder} hinzugefügt"
                    },
                    "variables": {
                      "type": "object",
                      "required": ["resource","folder","user"],
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
                        "space": {
                          "type": "object",
                          "required": ["id","name"],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%!%user_id_pattern%$"
                            },
                            "name": {
                              "const": "Alice Hansen"
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
          }
        }
      }
      """

  @issue-9850
  Scenario: check activity with -1 depth filter
    Given user "Alice" has created folder "/New Folder"
    And user "Alice" has created folder "/New Folder/Sub Folder"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/New Folder/Sub Folder/textfile.txt"
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with depth "-1" using the Graph API
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
                                  "const": "New Folder"
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
                                  "const": "New Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Sub Folder"
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
                                  "const": "Sub Folder"
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

  @issue-9850
  Scenario: check activity with depth filter
    Given user "Alice" has created folder "/New Folder"
    And user "Alice" has created folder "/New Folder/Sub Folder"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/New Folder/Sub Folder/textfile.txt"
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with depth "1" using the Graph API
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
                                  "const": "New Folder"
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
                                  "const": "New Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Sub Folder"
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

  @issue-9880
  Scenario: check activity with limit filter
    Given user "Alice" has created folder "/New Folder"
    And user "Alice" has created folder "/New Folder/Sub Folder"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/New Folder/Sub Folder/textfile.txt"
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with limit "2" using the Graph API
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
                                  "const": "New Folder"
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
                                  "const": "New Folder"
                                }
                              }
                            },
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "Sub Folder"
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


  Scenario: check activity with sort filter
    Given user "Alice" has created folder "/New Folder"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/New Folder/textfile.txt"
    And user "Alice" has uploaded file with content "updated OpenCloud test text file" to "/New Folder/textfile.txt"
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with sort "asc" using the Graph API
    Then the HTTP status code should be "200"
    And the activities should be in the following order:
      | resource     | message                               |
      | New Folder   | {user} added {resource} to {folder}   |
      | textfile.txt | {user} added {resource} to {folder}   |
      | textfile.txt | {user} updated {resource} in {folder} |
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with sort "desc" using the Graph API
    Then the HTTP status code should be "200"
    And the activities should be in the following order:
      | resource     | message                               |
      | textfile.txt | {user} updated {resource} in {folder} |
      | textfile.txt | {user} added {resource} to {folder}   |
      | New Folder   | {user} added {resource} to {folder}   |

  @issue-10150
  Scenario: check activities with limit and sort filters (asc/desc)
    Given user "Alice" has created folder "/New Folder"
    And user "Alice" has created folder "/New Folder/Sub Folder"
    And user "Alice" has uploaded file with content "OpenCloud test text file 0" to "/New Folder/Sub Folder/textfile.txt"
    And user "Alice" has uploaded file with content "updated OpenCloud test text file 0" to "/New Folder/Sub Folder/textfile.txt"
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with limit "2" and sort "asc" using the Graph API
    Then the HTTP status code should be "200"
    And the activities should be in the following order:
      | resource   | message                             |
      | New Folder | {user} added {resource} to {folder} |
      | Sub Folder | {user} added {resource} to {folder} |
    When user "Alice" lists the activities of folder "New Folder" from space "Personal" with limit "2" and sort "desc" using the Graph API
    Then the HTTP status code should be "200"
    And the activities should be in the following order:
      | resource     | message                               |
      | textfile.txt | {user} updated {resource} in {folder} |
      | textfile.txt | {user} added {resource} to {folder}   |

  @issue-9860
  Scenario: user tries to check activities of another user's file
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has uploaded file with content "updated OpenCloud test text file" to "textfile.txt"
    When user "Brian" tries to list the activities of file "textfile.txt" from space "Personal" owned by user "Alice" using the Graph API
    Then the HTTP status code should be "403"


  Scenario: check activities of destination resources after copying a file
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has copied file "textfile.txt" to "FOLDER/textfile.txt"
    When user "Alice" lists the activities of file "FOLDER/textfile.txt" from space "Personal" using the Graph API
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
                                  "const": "FOLDER"
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


  Scenario: check activities of destination resources after copying a file by renaming the destination file
    Given user "Alice" has uploaded file with content "OpenCloud test text file" to "textfile.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has copied file "textfile.txt" to "FOLDER/renamed.txt"
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
                                  "const": "FOLDER"
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


  Scenario: check activities of destination resources after copying a file into same folder by renaming the destination file
    Given user "Alice" has created folder "/FOLDER"
    And user "Alice" has uploaded file with content "OpenCloud test text file" to "FOLDER/textfile.txt"
    And user "Alice" has copied file "FOLDER/textfile.txt" to "FOLDER/renamed.txt"
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

  @issue-9536
  Scenario: check activities after moving a file inside a subfolder in a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "parent" in space "new-space"
    And user "Alice" has created a folder "parent/subfolder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "parent/test.md"
    And user "Alice" has moved file "parent/test.md" to "parent/subfolder/test.md" in space "new-space"
    When user "Alice" lists the activities of folder "parent" from space "new-space" using the Graph API
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
                          "required": ["resource", "folder", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "parent"
                                }
                              }
                            },
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "new-space"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
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
                          "required": ["resource", "folder", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "subfolder"
                                }
                              }
                            },
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "parent"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
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
                          "required": ["resource", "folder", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "test.md"
                                }
                              }
                            },
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "parent"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
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
                          "const": "{user} moved {resource} to {folder}"
                        },
                        "variables": {
                          "type": "object",
                          "required": ["resource", "folder", "user"],
                          "properties": {
                            "resource": {
                              "type": "object",
                              "required": ["id", "name"],
                              "properties": {
                                "name": {
                                  "const": "test.md"
                                }
                              }
                            },
                            "folder": {
                              "type": "object",
                              "required": ["name"],
                              "properties": {
                                "name": {
                                  "const": "subfolder"
                                }
                              }
                            },
                            "user": {
                              "type": "object",
                              "required": ["id", "displayName"],
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
                }
              ]
            }
          }
        }
      }
      """
