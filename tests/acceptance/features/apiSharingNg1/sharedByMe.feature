Feature: resources shared by user
  As a user
  I want to get resources shared by me
  So that I can know about what resources are shared with others

  https://docs.opencloud.eu/libre-graph-api/#/me.drive/ListSharedByMe

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: sharer lists the file share (Personal space)
    Given user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt       |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["personal"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["textfile.txt"]
          },
          "size": {
            "type": "number",
            "enum": [
              11
            ]
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | File Editor      |
      | Viewer           |


  Scenario: sharer gets thumbnails when listing shared image file (Personal space)
    Given user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testavatar.jpg     |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | Viewer             |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "testavatar.jpg" with the following data:
      """
        {
          "type": "object",
          "required": [ "thumbnails" ],
          "properties": {
            "thumbnails": {
              "type": "array",
              "minItems": 1,
              "maxItems": 1,
              "items": {
                "type": "object",
                "required": ["small", "medium", "large"],
                "properties": {
                  "small": {
                    "type": "object",
                    "required": ["url"],
                    "properties": {
                      "url": {
                        "type": "string",
                        "pattern": "^https://.*scalingup=0.*preview=1.*processor=thumbnail.*x=36.*y=36"
                      }
                    }
                  },
                  "medium": {
                    "type": "object",
                    "required": ["url"],
                    "properties": {
                      "url": {
                        "type": "string",
                        "pattern": "^https://.*scalingup=0.*preview=1.*processor=thumbnail.*x=48.*y=48"
                      }
                    }
                  },
                  "large": {
                    "type": "object",
                    "required": ["url"],
                    "properties": {
                      "url": {
                        "type": "string",
                        "pattern": "^https://.*scalingup=0.*preview=1.*processor=thumbnail.*x=96.*y=96"
                      }
                    }
                  }
                }
              }
            }
          }
        }
      """


  Scenario: sharer lists the file share shared from inside a folder (Personal space)
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "FolderToShare/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare/textfile.txt |
      | space           | Personal                   |
      | sharee          | Brian                      |
      | shareType       | user                       |
      | permissionsRole | Viewer                     |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["personal"]
              },
              "path": {
                "type": "string",
                "enum": ["/FolderToShare"]
              },
              "name": {
                "type": "string",
                "enum": ["FolderToShare"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["textfile.txt"]
          },
          "size": {
            "type": "number",
            "enum": [
              11
            ]
          }
        }
      }
      """


  Scenario Outline: sharer lists the folder share (Personal space)
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["personal"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["FolderToShare"]
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario: sharer lists the file and folder shares (Personal space)
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["personal"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["textfile.txt"]
          },
          "size": {
            "type": "number",
            "enum": [
              11
            ]
          }
        }
      }
      """
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["personal"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["FolderToShare"]
          }
        }
      }
      """


  Scenario: sharer lists the file and folder shares shared to group (Personal space)
    Given group "grp1" has been created
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%group_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "textfile.txt"
          }
        }
      }
      """
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "FolderToShare"
          }
        }
      }
      """


  Scenario Outline: sharer lists the file share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt       |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["project"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["textfile.txt"]
          },
          "size": {
            "type": "number",
            "enum": [
              11
            ]
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | File Editor      |
      | Viewer           |


  Scenario: sharer lists the file share shared from inside a folder (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "FolderToShare/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare/textfile.txt |
      | space           | NewSpace                   |
      | sharee          | Brian                      |
      | shareType       | user                       |
      | permissionsRole | Viewer                     |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["project"]
              },
              "path": {
                "type": "string",
                "enum": ["/FolderToShare"]
              },
              "name": {
                "type": "string",
                "enum": ["FolderToShare"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["textfile.txt"]
          },
          "size": {
            "type": "number",
            "enum": [
              11
            ]
          }
        }
      }
      """


  Scenario Outline: sharer lists the folder share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["project"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
          "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["FolderToShare"]
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario: sharer lists the file and folder shares (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | NewSpace     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["project"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["textfile.txt"]
          },
          "size": {
            "type": "number",
            "enum": [
              11
            ]
          }
        }
      }
      """
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "type": "string",
                "enum": ["project"]
              },
              "path": {
                "type": "string",
                "enum": ["/"]
              },
              "name": {
                "type": "string",
                "enum": ["/"]
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "type": "string",
                          "enum": [
                            "Brian Murphy"
                          ]
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "enum": ["FolderToShare"]
          }
        }
      }
      """


  Scenario: sharer lists the file and folder shares shared to group (Project space)
    Given group "grp1" has been created
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has created a folder "FolderToShare" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | new-space     |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%group_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "textfile.txt"
          }
        }
      }
      """
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "FolderToShare"
          }
        }
      }
      """

  @env-config
  Scenario: sharer lists the file share after sharee (user) is deleted (Personal space)
    Given the config "GRAPH_SPACES_USERS_CACHE_TTL" has been set to "1"
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has been deleted
    When user "Alice" lists the shares shared by her after clearing user cache using the Graph API
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
            "minItems":0,
            "maxItems":0
          }
        }
      }
      """

  @env-config
  Scenario: sharer lists the file share after sharee (group) is deleted (Personal space)
    Given the config "GRAPH_SPACES_GROUPS_CACHE_TTL" has been set to "1"
    And group "grp1" has been created
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    And group "grp1" has been deleted
    When user "Alice" lists the shares shared by her after clearing group cache using the Graph API
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
            "minItems":0,
            "maxItems":0
          }
        }
      }
      """

  @env-config
  Scenario: sharer lists the file share after sharee is disabled (Personal space)
    Given the config "GRAPH_SPACES_USERS_CACHE_TTL" has been set to "1"
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And the user "Admin" has disabled user "Brian"
    When user "Alice" lists the shares shared by her after clearing user cache using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveType": {
                "const": "personal"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "displayName": {
                          "const": "Brian Murphy"
                        }
                      }
                    }
                  }
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "type": "string",
            "const": "textfile.txt"
          }
        }
      }
      """

  @env-config
  Scenario: sharer lists the file share after sharee (user) is deleted (Project space)
    Given the config "GRAPH_SPACES_USERS_CACHE_TTL" has been set to "1"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has been deleted
    When user "Alice" lists the shares shared by her after clearing user cache using the Graph API
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
            "minItems":0,
            "maxItems":0
          }
        }
      }
      """

  @env-config
  Scenario: sharer lists the file share after sharee (group) is deleted (Project space)
    Given the config "GRAPH_SPACES_GROUPS_CACHE_TTL" has been set to "1"
    And using spaces DAV path
    And group "grp1" has been created
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    And group "grp1" has been deleted
    When user "Alice" lists the shares shared by her after clearing group cache using the Graph API
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
            "minItems":0,
            "maxItems":0
          }
        }
      }
      """

  @env-config
  Scenario: sharer lists the file share after sharee is disabled (Project space)
    Given the config "GRAPH_SPACES_USERS_CACHE_TTL" has been set to "1"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And the user "Admin" has disabled user "Brian"
    When user "Alice" lists the shares shared by her after clearing user cache using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveType": {
                "const": "project"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "displayName": {
                          "const": "Brian Murphy"
                        }
                      }
                    }
                  }
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "textfile.txt"
          }
        }
      }
      """


  Scenario: sharer lists the file link share (Personal space)
    Given user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "id",
          "eTag",
          "file",
          "lastModifiedDateTime",
          "size"
        ],
        "properties": {
          "name": {
            "const": "textfile.txt"
          },
          "file": {
            "type": "object",
            "required": ["mimeType"]
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "parentReference": {
            "type": "object",
            "required": ["driveId", "driveType", "id", "name", "path"],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": ["hasPassword", "id", "link", "createdDateTime"],
              "properties": {
                "link": {
                  "type": "object",
                  "required": [
                    "@libre.graph.displayName",
                    "@libre.graph.quickLink",
                    "preventsDownload",
                    "type",
                    "webUrl"
                  ],
                  "properties": {
                    "@libre.graph.displayName": {
                      "const": ""
                    },
                    "@libre.graph.quickLink": {
                      "const": false
                    },
                    "preventsDownload": {
                      "const": false
                    },
                    "type": {
                      "const": "view"
                    },
                    "webUrl": {
                      "type": "string",
                      "pattern": "^%base_url%/s/[a-zA-Z]+$"
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "[a-zA-Z]+"
                },
                "hasPassword": {
                  "const": true
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharer lists the folder link share (Personal space)
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | Personal      |
      | permissionsRole | edit          |
      | password        | %public%      |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "id",
          "eTag",
          "folder",
          "lastModifiedDateTime",
          "size"
        ],
        "properties": {
          "name": {
            "const": "FolderToShare"
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "folder": {
            "const": {}
          },
          "parentReference": {
            "type": "object",
            "required": ["driveId", "driveType", "id", "name", "path"],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": ["hasPassword", "id", "link", "createdDateTime"],
              "properties": {
                "link": {
                  "type": "object",
                  "required": [
                    "@libre.graph.displayName",
                    "@libre.graph.quickLink",
                    "preventsDownload",
                    "type",
                    "webUrl"
                  ],
                  "properties": {
                    "@libre.graph.displayName": {
                      "const": ""
                    },
                    "@libre.graph.quickLink": {
                      "const": false
                    },
                    "preventsDownload": {
                      "const": false
                    },
                    "type": {
                      "const": "edit"
                    },
                    "webUrl": {
                      "type": "string",
                      "pattern": "^%base_url%/s/[a-zA-Z]+$"
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "[a-zA-Z]+"
                },
                "hasPassword": {
                  "const": true
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharer lists the link shares of same name files (Personal space)
    Given user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "FolderToShare/textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | edit         |
      | password        | %public%     |
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare/textfile.txt |
      | space           | Personal                   |
      | permissionsRole | view                       |
      | password        | %public%                   |
    When user "Alice" lists the shares shared by her using the Graph API
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "file"
                  ],
                  "properties": {
                    "name": {
                      "const": "textfile.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "personal"
                        },
                        "path": {
                          "const": "/"
                        },
                        "name": {
                          "const": "/"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "edit"
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "file"
                  ],
                  "properties": {
                    "name": {
                      "const": "textfile.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "personal"
                        },
                        "path": {
                          "const": "/FolderToShare"
                        },
                        "name": {
                          "const": "FolderToShare"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "view"
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


  Scenario: sharer lists the link shares of same name folders (Personal space)
    Given user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/FolderToShare"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | Personal      |
      | permissionsRole | edit          |
      | password        | %public%      |
    And user "Alice" has created the following resource link share:
      | resource        | parent/FolderToShare |
      | space           | Personal             |
      | permissionsRole | view                 |
      | password        | %public%             |
    When user "Alice" lists the shares shared by her using the Graph API
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "folder"
                  ],
                  "properties": {
                    "name": {
                      "const": "FolderToShare"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "personal"
                        },
                        "path": {
                          "const": "/"
                        },
                        "name": {
                          "const": "/"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "edit"
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "folder"
                  ],
                  "properties": {
                    "name": {
                      "const": "FolderToShare"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "personal"
                        },
                        "path": {
                          "const": "/parent"
                        },
                        "name": {
                          "const": "parent"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "view"
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


  Scenario: sharer lists the link shares of a file after deleting one link share (Personal space)
    Given user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | edit         |
      | password        | %public%     |
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | Personal     |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has removed the last link share of file "textfile.txt" from space "Personal"
    When user "Alice" lists the shares shared by her using the Graph API
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
              "required": [
                "parentReference",
                "permissions",
                "name",
                "id",
                "eTag",
                "file"
              ],
              "properties": {
                "name": {
                  "const": "textfile.txt"
                },
                "permissions": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": ["hasPassword", "id", "link", "createdDateTime"],
                    "properties": {
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ],
                        "properties": {
                          "type": {
                            "const": "edit"
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


  Scenario: sharer lists the link shares of a folder after deleting one link share (Personal space)
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | Personal      |
      | permissionsRole | edit          |
      | password        | %public%      |
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | Personal      |
      | permissionsRole | view          |
      | password        | %public%      |
    And user "Alice" has removed the last link share of folder "FolderToShare" from space "Personal"
    When user "Alice" lists the shares shared by her using the Graph API
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
              "required": [
                "parentReference",
                "permissions",
                "name",
                "id",
                "eTag",
                "folder"
              ],
              "properties": {
                "name": {
                  "const": "FolderToShare"
                },
                "permissions": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": ["hasPassword", "id", "link", "createdDateTime"],
                    "properties": {
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ],
                        "properties": {
                          "type": {
                            "const": "edit"
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


  Scenario: sharer lists the file link share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | NewSpace     |
      | permissionsRole | view         |
      | password        | %public%     |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "id",
          "eTag",
          "file",
          "lastModifiedDateTime",
          "size"
        ],
        "properties": {
          "name": {
            "const": "textfile.txt"
          },
          "file": {
            "type": "object",
            "required": ["mimeType"]
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "parentReference": {
            "type": "object",
            "required": ["driveId", "driveType", "id", "name", "path"],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": ["hasPassword", "id", "link", "createdDateTime"],
              "properties": {
                "link": {
                  "type": "object",
                  "required": [
                    "@libre.graph.displayName",
                    "@libre.graph.quickLink",
                    "preventsDownload",
                    "type",
                    "webUrl"
                  ],
                  "properties": {
                    "@libre.graph.displayName": {
                      "const": ""
                    },
                    "@libre.graph.quickLink": {
                      "const": false
                    },
                    "preventsDownload": {
                      "const": false
                    },
                    "type": {
                      "const": "view"
                    },
                    "webUrl": {
                      "type": "string",
                      "pattern": "^%base_url%/s/[a-zA-Z]+$"
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "[a-zA-Z]+"
                },
                "hasPassword": {
                  "const": true
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharer lists the folder link share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | permissionsRole | edit          |
      | password        | %public%      |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "id",
          "eTag",
          "folder",
          "lastModifiedDateTime",
          "size"
        ],
        "properties": {
          "name": {
            "const": "FolderToShare"
          },
          "id": {
            "type": "string",
            "pattern": "^%file_id_pattern%$"
          },
          "folder": {
            "const": {}
          },
          "parentReference": {
            "type": "object",
            "required": ["driveId", "driveType", "id", "name", "path"],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": ["hasPassword", "id", "link", "createdDateTime"],
              "properties": {
                "link": {
                  "type": "object",
                  "required": [
                    "@libre.graph.displayName",
                    "@libre.graph.quickLink",
                    "preventsDownload",
                    "type",
                    "webUrl"
                  ],
                  "properties": {
                    "@libre.graph.displayName": {
                      "const": ""
                    },
                    "@libre.graph.quickLink": {
                      "const": false
                    },
                    "preventsDownload": {
                      "const": false
                    },
                    "type": {
                      "const": "edit"
                    },
                    "webUrl": {
                      "type": "string",
                      "pattern": "^%base_url%/s/[a-zA-Z]+$"
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "[a-zA-Z]+"
                },
                "hasPassword": {
                  "const": true
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharer lists the link shares of same name files (project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "textfile.txt"
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "FolderToShare/textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | NewSpace     |
      | permissionsRole | edit         |
      | password        | %public%     |
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare/textfile.txt |
      | space           | NewSpace                   |
      | permissionsRole | view                       |
      | password        | %public%                   |
    When user "Alice" lists the shares shared by her using the Graph API
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "file"
                  ],
                  "properties": {
                    "name": {
                      "const": "textfile.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "project"
                        },
                        "path": {
                          "const": "/"
                        },
                        "name": {
                          "const": "/"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "edit"
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "file"
                  ],
                  "properties": {
                    "name": {
                      "const": "textfile.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "project"
                        },
                        "path": {
                          "const": "/FolderToShare"
                        },
                        "name": {
                          "const": "FolderToShare"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "view"
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


  Scenario: sharer lists the link shares of same name folders (project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has created a folder "parent" in space "NewSpace"
    And user "Alice" has created a folder "parent/FolderToShare" in space "NewSpace"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | permissionsRole | edit          |
      | password        | %public%      |
    And user "Alice" has created the following resource link share:
      | resource        | parent/FolderToShare |
      | space           | NewSpace             |
      | permissionsRole | view                 |
      | password        | %public%             |
    When user "Alice" lists the shares shared by her using the Graph API
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "folder"
                  ],
                  "properties": {
                    "name": {
                      "const": "FolderToShare"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "project"
                        },
                        "path": {
                          "const": "/"
                        },
                        "name": {
                          "const": "/"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "edit"
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
                  "required": [
                    "parentReference",
                    "permissions",
                    "name",
                    "id",
                    "eTag",
                    "folder"
                  ],
                  "properties": {
                    "name": {
                      "const": "FolderToShare"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType", "id", "name", "path"],
                      "properties": {
                        "driveType": {
                          "const": "project"
                        },
                        "path": {
                          "const": "/parent"
                        },
                        "name": {
                          "const": "parent"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": ["hasPassword", "id", "link", "createdDateTime"],
                        "properties": {
                          "link": {
                            "type": "object",
                            "required": [
                              "@libre.graph.displayName",
                              "@libre.graph.quickLink",
                              "preventsDownload",
                              "type",
                              "webUrl"
                            ],
                            "properties": {
                              "type": {
                                "const": "view"
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


  Scenario: sharer lists the link shares of a file after deleting one link share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | NewSpace     |
      | permissionsRole | edit         |
      | password        | %public%     |
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt |
      | space           | NewSpace     |
      | permissionsRole | view         |
      | password        | %public%     |
    And user "Alice" has removed the last link share of file "textfile.txt" from space "NewSpace"
    When user "Alice" lists the shares shared by her using the Graph API
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
              "required": [
                "parentReference",
                "permissions",
                "name",
                "id",
                "eTag",
                "file"
              ],
              "properties": {
                "name": {
                  "const": "textfile.txt"
                },
                "permissions": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": ["hasPassword", "id", "link", "createdDateTime"],
                    "properties": {
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ],
                        "properties": {
                          "type": {
                            "const": "edit"
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


  Scenario: sharer lists the link shares of a folder after deleting one link share (project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | permissionsRole | edit          |
      | password        | %public%      |
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | permissionsRole | view          |
      | password        | %public%      |
    And user "Alice" has removed the last link share of folder "FolderToShare" from space "NewSpace"
    When user "Alice" lists the shares shared by her using the Graph API
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
              "required": [
                "parentReference",
                "permissions",
                "name",
                "id",
                "eTag",
                "folder"
              ],
              "properties": {
                "name": {
                  "const": "FolderToShare"
                },
                "permissions": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": ["hasPassword", "id", "link", "createdDateTime"],
                    "properties": {
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ],
                        "properties": {
                          "type": {
                            "const": "edit"
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

  @issue-8355
  Scenario: sharer lists the link share of a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created the following space link share:
      | space           | NewSpace      |
      | permissionsRole | view          |
      | password        | %public%      |
    When user "Alice" lists the shares shared by her using the Graph API
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
              "required": [
                "parentReference",
                "permissions",
                "name",
                "id",
                "eTag",
                "folder",
                "lastModifiedDateTime",
                "size"
              ],
              "properties": {
                "name": {
                  "const": "."
                },
                "folder": {
                  "const": {}
                },
                "id": {
                  "type": "string",
                  "pattern": "^%file_id_pattern%$"
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id", "name", "path"],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType": {
                      "const": "project"
                    },
                    "path": {
                      "const": "."
                    },
                    "name": {
                      "const": "."
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    }
                  }
                },
                "permissions": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": ["hasPassword", "id", "link", "createdDateTime"],
                    "properties": {
                      "link": {
                        "type": "object",
                        "required": [
                          "@libre.graph.displayName",
                          "@libre.graph.quickLink",
                          "preventsDownload",
                          "type",
                          "webUrl"
                        ],
                        "properties": {
                          "@libre.graph.displayName": {
                            "const": ""
                          },
                          "@libre.graph.quickLink": {
                            "const": false
                          },
                          "preventsDownload": {
                            "const": false
                          },
                          "type": {
                            "const": "view"
                          },
                          "webUrl": {
                            "type": "string",
                            "pattern": "^%base_url%/s/[a-zA-Z]+$"
                          }
                        }
                      },
                      "id": {
                        "type": "string",
                        "pattern": "[a-zA-Z]+"
                      },
                      "hasPassword": {
                        "const": true
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


  Scenario: sharer (also a group member) lists shares shared to group (Personal space)
    Given group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%group_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "textfile.txt"
          }
        }
      }
      """
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "personal"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "FolderToShare"
          }
        }
      }
      """


  Scenario: sharer (also a group member) lists shares shared to group (Project space)
    Given group "grp1" has been created
    And user "Alice" has been added to group "grp1"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "textfile.txt"
    And user "Alice" has created a folder "FolderToShare" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | new-space    |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | new-space     |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Alice" lists the shares shared by her using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name",
          "size"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%group_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }
                }
              }
            }
          },
          "name": {
            "const": "textfile.txt"
          }
        }
      }
      """
    And the JSON data of the response should contain resource "FolderToShare" with the following data:
      """
      {
        "type": "object",
        "required": [
          "parentReference",
          "permissions",
          "name"
        ],
        "properties": {
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "path",
              "name",
              "id"
            ],
            "properties": {
              "driveId": {
                "type": "string",
                "pattern": "^%space_id_pattern%$"
              },
              "driveType": {
                "const": "project"
              },
              "path": {
                "const": "/"
              },
              "name": {
                "const": "/"
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              }
            }
          },
          "permissions": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "grantedToV2",
                "id",
                "roles"
              ],
              "properties": {
                "grantedToV2": {
                  "type": "object",
                  "required": ["group"],
                  "properties": {
                    "group": {
                      "type": "object",
                      "required": [
                        "displayName",
                        "id"
                      ],
                      "properties": {
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        },
                        "displayName": {
                          "const": "grp1"
                        }
                      }
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%permissions_id_pattern%$"
                },
                "roles": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "string",
                    "pattern": "^%role_id_pattern%$"
                  }

                }
              }
            }
          },
          "name": {
            "const": "FolderToShare"
          }
        }
      }
      """
