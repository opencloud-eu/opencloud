Feature: an user gets the resources shared to them
  As a user
  I want to get resources shared with me
  So that I can know about what resources I have access to

  https://docs.opencloud.eu/libre-graph-api/#/me.drive/ListSharedWithMe

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario Outline: sharee lists the file share (Personal space)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
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
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden": {
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize": {
                  "type": "boolean",
                  "enum": [true]
                },
                "createdBy": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id"],
                      "properties": {
                        "displayName": {
                          "type": "string",
                          "enum": ["Alice Hansen"]
                        },
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        }
                      }
                    }
                  }
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "file": {
                  "type": "object",
                  "required": ["mimeType"],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["text/plain"]
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": [
                    "textfile0.txt"
                  ]
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType": {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions",
                    "size"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": [
                        "user"
                      ],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": [
                            "id",
                            "displayName"
                          ],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "type": "string",
                              "enum": [
                                "Alice Hansen"
                              ]
                            }
                          }
                        }
                      }
                    },
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "file": {
                      "type": "object",
                      "required": [
                        "mimeType"
                      ],
                      "properties": {
                        "mimeType": {
                          "type": "string",
                          "pattern": "text/plain"
                        }
                      }
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": [
                        "textfile0.txt"
                      ]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%file_id_pattern%$"
                        },
                        "driveType": {
                          "type": "string",
                          "enum": ["personal"]
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
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
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
                                  "displayName": {
                                    "type": "string",
                                    "enum": ["Brian Murphy"]
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "invitation": {
                            "type": "object",
                            "properties": {
                              "invitedBy": {
                                "type": "object",
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Alice Hansen"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    },
                                    "required": [
                                      "displayName",
                                      "id"
                                    ]
                                  }
                                },
                                "required": [
                                  "user"
                                ]
                              }
                            },
                            "required": [
                              "invitedBy"
                            ]
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
                    }
                  }
                },
                "size": {
                  "type": "number",
                  "enum": [11]
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | File Editor      |
      | Viewer           |


  Scenario: sharee gets thumbnails when listing shared image file (Personal space)
    Given user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testavatar.jpg     |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | Viewer             |
    When user "Brian" lists the shares shared with him using the Graph API
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
            "required": ["thumbnails"],
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
        }
      }
    }
      """


  Scenario Outline: sharee lists the folder share (Personal space)
    Given user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder             |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
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
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden": {
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize": {
                  "type": "boolean",
                  "enum": [true]
                },
                "createdBy": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id"],
                      "properties": {
                        "displayName": {
                          "type": "string",
                          "enum": ["Alice Hansen"]
                        },
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        }
                      }
                    }
                  }
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "folder": {
                  "type": "object",
                  "required": [],
                  "properties": {}
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": [
                    "folder"
                  ]
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType": {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": [
                        "user"
                      ],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": [
                            "id",
                            "displayName"
                          ],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "type": "string",
                              "enum": [
                                "Alice Hansen"
                              ]
                            }
                          }
                        }
                      }
                    },
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "file": {
                      "type": "object",
                      "required": [
                        "mimeType"
                      ],
                      "properties": {
                        "mimeType": {
                          "type": "string",
                          "enum": ["text/plain"]
                        }
                      }
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": [
                        "folder"
                      ]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%file_id_pattern%$"
                        },
                        "driveType": {
                          "type": "string",
                          "enum": ["personal"]
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
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "grantedToV2": {
                            "type": "object",
                            "required": [
                              "user"
                            ],
                            "properties": {
                              "user": {
                                "type": "object",
                                "properties": {
                                  "displayName": {
                                    "type": "string",
                                    "enum": ["Brian Murphy"]
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                },
                                "required": [
                                  "displayName",
                                  "id"
                                ]
                              }
                            }
                          },
                          "invitation": {
                            "type": "object",
                            "properties": {
                              "invitedBy": {
                                "type": "object",
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Alice Hansen"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    },
                                    "required": [
                                      "displayName",
                                      "id"
                                    ]
                                  }
                                },
                                "required": [
                                  "user"
                                ]
                              }
                            },
                            "required": [
                              "invitedBy"
                            ]
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
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario: sharee lists the file share received via group invitation (Personal space)
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has uploaded file with content "hello" to "textfile0.txt"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
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
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden":{
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize":{
                  "type": "boolean",
                  "enum": [true]
                },
                "createdBy": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id"],
                      "properties": {
                        "displayName": {
                          "type": "string",
                          "enum": ["Alice Hansen"]
                        },
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        }
                      }
                    }
                  }
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "file": {
                  "type": "object",
                  "required": ["mimeType"],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "enum": ["text/plain"]
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": [
                    "textfile0.txt"
                  ]
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions",
                    "size"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": [
                        "user"
                      ],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": [
                            "id",
                            "displayName"
                          ],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "type": "string",
                              "enum": [
                                "Alice Hansen"
                              ]
                            }
                          }
                        }
                      }
                    },
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "file": {
                      "type": "object",
                      "required": ["mimeType"],
                      "properties": {
                        "mimeType": {
                          "type": "string",
                          "enum": ["text/plain"]
                        }
                      }
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": [
                        "textfile0.txt"
                      ]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "type": "string",
                          "enum": ["personal"]
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["group"],
                            "properties":{
                              "group": {
                                "type": "object",
                                "required": [
                                  "displayName",
                                  "id"
                                ],
                                "properties": {
                                  "displayName": {
                                    "const": "grp1"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "user":{
                                "type": "object",
                                "required": [
                                  "displayName",
                                  "id"
                                ],
                                "properties": {
                                  "displayName": {
                                    "const": "Alice Hansen"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
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
                    "size": {
                      "enum": [
                        5
                      ]
                    }
                  }
                },
                "size": {
                  "enum": [
                    5
                  ]
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee lists the folder share received via group invitation (Personal space)
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And  user "Alice" has created folder "folder"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    When user "Brian" lists the shares shared with him using the Graph API
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
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden":{
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize":{
                  "type": "boolean",
                  "enum": [true]
                },
                "createdBy": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id"],
                      "properties": {
                        "displayName": {
                          "const": "Alice Hansen"
                        },
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        }
                      }
                    }
                  }
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "folder"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
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
                    },
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "folder"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "const": "personal"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["group"],
                            "properties":{
                              "group": {
                                "type": "object",
                                "required": [
                                  "displayName",
                                  "id"
                                ],
                                "properties": {
                                  "displayName": {
                                    "const": "grp1"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "user":{
                                "type": "object",
                                "required": [
                                  "displayName",
                                  "id"
                                ],
                                "properties": {
                                  "displayName": {
                                    "const": "Alice Hansen"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
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
                    }
                  }
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee list the same file shares received via user and group invitation (Personal space)
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has uploaded file with content "hello" to "textfile0.txt"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
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
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden":{
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize":{
                  "type": "boolean",
                  "enum": [true]
                },
                "createdBy": {
                  "type": "object",
                  "required": [
                    "user"
                  ],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id"],
                      "properties": {
                        "displayName": {
                          "type": "string",
                          "enum": ["Alice Hansen"]
                        },
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        }
                      }
                    }
                  }
                },
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
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "textfile0.txt"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions",
                    "size"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": [
                        "user"
                      ],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": [
                            "id",
                            "displayName"
                          ],
                          "properties": {
                            "id": {
                              "type": "string",
                              "pattern": "^%user_id_pattern%$"
                            },
                            "displayName": {
                              "type": "string",
                              "enum": [
                                "Alice Hansen"
                              ]
                            }
                          }
                        }
                      }
                    },
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
                    "name": {
                      "const": "textfile0.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "type": "string",
                          "enum": ["personal"]
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 2,
                      "maxItems": 2,
                      "uniqueItems": true,
                      "items": {
                        "oneOf": [
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": [
                                  "group"
                                ],
                                "properties":{
                                  "group": {
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["grp1"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": [
                                  "invitedBy"
                                ],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Alice Hansen"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
                          },
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": [
                                  "user"
                                ],
                                "properties":{
                                  "user": {
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": [
                                  "invitedBy"
                                ],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Alice Hansen"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
                        ]
                      }
                    },
                    "size": {
                      "const": 5
                    }
                  }
                },
                "size": {
                  "const": 5
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee list the same folder shares received via user and group invitation (Personal space)
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And  user "Alice" has created folder "folder"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | grp1     |
      | shareType       | group    |
      | permissionsRole | Viewer   |
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
                "@UI.Hidden",
                "@client.synchronize",
                "createdBy",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden":{
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize":{
                  "type": "boolean",
                  "enum": [true]
                },
                "createdBy": {
                  "type": "object",
                  "required": ["user"],
                  "properties": {
                    "user": {
                      "type": "object",
                      "required": ["displayName", "id"],
                      "properties": {
                        "displayName": {
                          "const": "Alice Hansen"
                        },
                        "id": {
                          "type": "string",
                          "pattern": "^%user_id_pattern%$"
                        }
                      }
                    }
                  }
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "folder": {
                  "const": {}
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": [
                    "folder"
                  ]
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
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
                    },
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "file": {},
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": [
                        "folder"
                      ]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "const": "personal"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 2,
                      "maxItems": 2,
                      "uniqueItems": true,
                      "items": {
                        "oneOf": [
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": [
                                  "group"
                                ],
                                "properties":{
                                  "group": {
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["grp1"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": [
                                  "invitedBy"
                                ],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Alice Hansen"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
                          },
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": [
                                  "user"
                                ],
                                "properties":{
                                  "user": {
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": [
                                  "invitedBy"
                                ],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Alice Hansen"]
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
      """


  Scenario: sharee lists the same name file shares received from different users (Personal space)
    Given user "Carol" has been created with default attributes
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Carol" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Carol" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    When user "Brian" lists the shares shared with him using the Graph API
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
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "@UI.Hidden": {
                      "const": false
                    },
                    "@client.synchronize": {
                      "const": true
                    },
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName", "id"],
                          "properties": {
                            "displayName": {
                              "const": "Carol King"
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "const": "textfile (1).txt"
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": [
                        "createdBy",
                        "eTag",
                        "file",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "createdBy": {
                          "type": "object",
                          "required": ["user"],
                          "properties": {
                            "user": {
                              "type": "object",
                              "required": ["displayName","id"],
                              "properties": {
                                "displayName": {
                                  "const": "Carol King"
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "const": "textfile.txt"
                        },
                        "permissions": {
                          "type": "array",
                          "maxItems": 1,
                          "minItems": 1,
                          "items": {
                            "type": "object",
                            "required": ["grantedToV2", "id", "invitation", "roles"],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["user"],
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Brian Murphy"
                                      }
                                    }
                                  }
                                }
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "invitedBy": {
                                    "type": "object",
                                    "required": ["user"],
                                    "properties": {
                                      "user": {
                                        "type": "object",
                                        "required": ["displayName", "id"],
                                        "properties": {
                                          "displayName": {
                                            "const": "Carol King"
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
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "@UI.Hidden": {
                      "const": false
                    },
                    "@client.synchronize": {
                      "const": true
                    },
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName", "id"],
                          "properties": {
                            "displayName": {
                              "const": "Alice Hansen"
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "const": "textfile.txt"
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": [
                        "createdBy",
                        "eTag",
                        "file",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "createdBy": {
                          "type": "object",
                          "required": ["user"],
                          "properties": {
                            "user": {
                              "type": "object",
                              "required": ["displayName", "id"],
                              "properties": {
                                "displayName": {
                                  "type": "string",
                                  "enum": ["Alice Hansen"]
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "const": "textfile.txt"
                        },
                        "permissions": {
                          "type": "array",
                          "maxItems": 1,
                          "minItems": 1,
                          "items": {
                            "type": "object",
                            "required": ["grantedToV2", "id", "invitation", "roles"],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["user"],
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Brian Murphy"
                                      }
                                    }
                                  }
                                }
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "invitedBy": {
                                    "type": "object",
                                    "required": ["user"],
                                    "properties": {
                                      "user": {
                                        "type": "object",
                                        "required": ["displayName", "id"],
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


  Scenario: sharee lists the same name folder shares received from different users (Personal space)
    Given user "Carol" has been created with default attributes
    And user "Alice" has created folder "folderToShare"
    And user "Carol" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Carol" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    When user "Brian" lists the shares shared with him using the Graph API
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
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "@UI.Hidden": {
                      "const": false
                    },
                    "@client.synchronize": {
                      "const": true
                    },
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName", "id"],
                          "properties": {
                            "displayName": {
                              "const": "Carol King"
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "const": "folderToShare (1)"
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": [
                        "createdBy",
                        "eTag",
                        "folder",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "createdBy": {
                          "type": "object",
                          "required": ["user"],
                          "properties": {
                            "user": {
                              "type": "object",
                              "required": ["displayName","id"],
                              "properties": {
                                "displayName": {
                                  "const": "Carol King"
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "const": "folderToShare"
                        },
                        "permissions": {
                          "type": "array",
                          "maxItems": 1,
                          "minItems": 1,
                          "items": {
                            "type": "object",
                            "required": ["grantedToV2", "id", "invitation", "roles"],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["user"],
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Brian Murphy"
                                      }
                                    }
                                  }
                                }
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "invitedBy": {
                                    "type": "object",
                                    "required": ["user"],
                                    "properties": {
                                      "user": {
                                        "type": "object",
                                        "required": ["displayName", "id"],
                                        "properties": {
                                          "displayName": {
                                            "const": "Carol King"
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
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "@UI.Hidden",
                    "@client.synchronize",
                    "createdBy",
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "remoteItem"
                  ],
                  "properties": {
                    "@UI.Hidden": {
                      "const": false
                    },
                    "@client.synchronize": {
                      "const": true
                    },
                    "createdBy": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName", "id"],
                          "properties": {
                            "displayName": {
                              "const": "Alice Hansen"
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "const": "folderToShare"
                    },
                    "remoteItem": {
                      "type": "object",
                      "required": [
                        "createdBy",
                        "eTag",
                        "folder",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "createdBy": {
                          "type": "object",
                          "required": ["user"],
                          "properties": {
                            "user": {
                              "type": "object",
                              "required": ["displayName", "id"],
                              "properties": {
                                "displayName": {
                                  "const": "Alice Hansen"
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "const": "folderToShare"
                        },
                        "permissions": {
                          "type": "array",
                          "maxItems": 1,
                          "minItems": 1,
                          "items": {
                            "type": "object",
                            "required": ["grantedToV2", "id", "invitation", "roles"],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["user"],
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Brian Murphy"
                                      }
                                    }
                                  }
                                }
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "invitedBy": {
                                    "type": "object",
                                    "required": ["user"],
                                    "properties": {
                                      "user": {
                                        "type": "object",
                                        "required": ["displayName", "id"],
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

  @issue-8471
  Scenario: sharee lists the same name file and folder shares received from different users (Personal space)
    Given using spaces DAV path
    And user "Carol" has been created with default attributes
    And user "Brian" has created folder "folder"
    And user "Brian" has uploaded file with content "hello world" to "/textfile.txt"
    And user "Carol" has created folder "folder"
    And user "Carol" has uploaded file with content "hello world" to "/textfile.txt"
    And user "Brian" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Carol" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Alice        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Carol" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Alice    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    When user "Alice" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 4,
            "minItems": 4,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "folder"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "folder (1)"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "textfile.txt"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "textfile (1).txt"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """

  @issue-8027 @issue-8314
  Scenario Outline: sharee lists the file share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testfile.txt       |
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" lists the shares shared with him using the Graph API
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
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden":{
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize":{
                  "type": "boolean",
                  "enum": [true]
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "file": {
                  "type": "object",
                  "required": ["mimeType"],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "pattern": "^text/plain"
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "testfile.txt"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "type": "string",
                      "enum": ["virtual"]
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions",
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
                          "type": "string",
                          "pattern": "^text/plain"
                        }
                      }
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "testfile.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%file_id_pattern%$"
                        },
                        "driveType" : {
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
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "const": "Brian Murphy"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern":"^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "invitedBy": {
                                "type": "object",
                                "required": ["user"],
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Alice Hansen"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              }
                            }
                          },
                          "roles": {
                            "type": "array",
                            "maxItems": 1,
                            "minItems": 1,
                            "items": {
                              "type": "string",
                              "pattern": "^%role_id_pattern%$"
                            }
                          }
                        }
                      }
                    },
                    "size": {
                      "const": 12
                    }
                  }
                },
                "size": {
                  "const": 12
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | permissions-role |
      | File Editor      |
      | Viewer           |

  @issue-8027 @issue-8314
  Scenario Outline: sharee lists the folder share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder             |
      | space           | new-space          |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "folder": {
                  "const": {}
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "folder"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "folder": {
                      "const": {}
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "folder"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%file_id_pattern%$"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "properties": {
                                  "displayName": {
                                    "const": "Brian Murphy"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern":"^%user_id_pattern%$"
                                  }
                                },
                                "required": [
                                  "displayName",
                                  "id"
                                ]
                              }
                            }
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "invitedBy": {
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
                                        "const": "Alice Hansen"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              }
                            }
                          },
                          "roles": {
                            "type": "array",
                            "maxItems": 1,
                            "minItems": 1,
                            "items": {
                              "type": "string",
                              "pattern": "^%role_id_pattern%$"
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
      }
      """
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |


  Scenario: sharee lists the file share received via group invitation (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile0.txt"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | new-space     |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Brian" lists the shares shared with him using the Graph API
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
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
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
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "textfile0.txt"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
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
                    "name": {
                      "const": "textfile0.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["group"],
                            "properties":{
                              "group": {
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "const": "grp1"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "user":{
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "const": "Alice Hansen"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
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
                    "size": {
                      "const": 11
                    }
                  }
                },
                "size": {
                  "const": 11
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee lists the folder share received via group invitation (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | grp1      |
      | shareType       | group     |
      | permissionsRole | Viewer    |
    When user "Brian" lists the shares shared with him using the Graph API
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
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "folder"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "folder"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "grantedToV2": {
                            "type": "object",
                            "required": ["group"],
                            "properties":{
                              "group": {
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "const": "grp1"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "user":{
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "const": "Alice Hansen"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%user_id_pattern%$"
                                  }
                                }
                              }
                            }
                          },
                          "roles": {
                            "type": "array",
                            "maxItems": 1,
                            "minItems": 1,
                            "items": {
                              "type": "string",
                              "pattern": "^%role_id_pattern%$"
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
      }
      """


  Scenario: sharee list the same file shares received via user and group invitation (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "textfile0.txt"
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | new-space     |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | new-space     |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    When user "Brian" lists the shares shared with him using the Graph API
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
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
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
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "textfile0.txt"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions",
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
                    "name": {
                      "const": "textfile0.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 2,
                      "maxItems": 2,
                      "uniqueItems": true,
                      "items": {
                        "oneOf": [
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["group"],
                                "properties":{
                                  "group": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "grp1"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Alice Hansen"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
                          },
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["user"],
                                "properties":{
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Brian Murphy"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Alice Hansen"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
                        ]
                      }
                    },
                    "size": {
                      "const": 11
                    }
                  }
                },
                  "size": {
                  "const": 11
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee list the same folder shares received via user and group invitation (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a group "grp1" using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Brian" has been added to group "grp1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | grp1      |
      | shareType       | group     |
      | permissionsRole | Viewer    |
    When user "Brian" lists the shares shared with him using the Graph API
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
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "folder": {
                  "const": {}
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "folder"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "%space_id_pattern%"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "folder": {
                      "const": {}
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "folder"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "%space_id_pattern%"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "minItems": 2,
                      "maxItems": 2,
                      "uniqueItems": true,
                      "items": {
                        "oneOf": [
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["group"],
                                "properties":{
                                  "group": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "grp1"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Alice Hansen"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
                          },
                          {
                            "type": "object",
                            "required": [
                              "grantedToV2",
                              "id",
                              "invitation",
                              "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["user"],
                                "properties":{
                                  "user": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Brian Murphy"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              },
                              "id": {
                                "type": "string",
                                "pattern": "^%permissions_id_pattern%$"
                              },
                              "invitation": {
                                "type": "object",
                                "required": ["invitedBy"],
                                "properties": {
                                  "user":{
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "const": "Alice Hansen"
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
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
      """

  @issue-8471
  Scenario: sharee lists the same name file and folder shares received from different projects (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "space-moon" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "space-moon"
    And user "Alice" has uploaded a file inside space "space-moon" with content "hello world" to "textfile.txt"
    And user "Alice" has created a space "space-mars" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "space-mars"
    And user "Alice" has uploaded a file inside space "space-mars" with content "hello world" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | space-moon   |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder     |
      | space           | space-moon |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | space-mars   |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder     |
      | space           | space-mars |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 4,
            "minItems": 4,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "folder"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "folder (1)"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "textfile.txt"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": ["name"],
                  "properties": {
                    "name": {
                      "const": "textfile (1).txt"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """


  Scenario: sharee lists the file share after sharer is deleted (Personal space)
    Given user "Alice" has uploaded file with content "hello" to "textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "minItems":0,
            "maxItems":0
          }
        }
      }
      """


  Scenario: sharee lists the folder share after sharer is deleted (Personal space)
    Given user "Alice" has created folder "folder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And the user "Admin" has deleted a user "Alice"
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
            "minItems": 0,
            "maxItems": 0
          }
        }
      }
      """


  Scenario: sharee lists the file share after the sharer is disabled (Personal space)
    Given user "Alice" has uploaded file with content "hello" to "textfile0.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And the user "Admin" has disabled user "Alice"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "textfile0.txt" with the following data:
      """
      {
        "type": "object",
        "required": [
          "@UI.Hidden",
          "@client.synchronize",
          "createdBy",
          "eTag",
          "file",
          "id",
          "name",
          "parentReference",
          "remoteItem",
          "size"
        ],
        "properties": {
          "@UI.Hidden": {
            "const": false
          },
          "@client.synchronize": {
            "const": true
          },
          "createdBy": {
            "type": "object",
            "required": ["user"],
            "properties": {
              "user": {
                "type": "object",
                "required": ["displayName", "id"],
                "properties": {
                  "displayName": {
                    "const": "Alice Hansen"
                  }
                }
              }
            }
          },
          "name": {
            "const": "textfile0.txt"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id"
            ],
            "properties": {
              "driveType": {
                "const": "virtual"
              }
            }
          },
          "remoteItem": {
            "type": "object",
            "required": [
              "createdBy",
              "eTag",
              "file",
              "id",
              "name",
              "parentReference",
              "permissions",
              "size"
            ],
            "properties": {
              "createdBy": {
                "type": "object",
                "required": ["user"],
                "properties": {
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
              },
              "name": {
                "const": "textfile0.txt"
              },
              "parentReference": {
                "type": "object",
                "required": ["driveId", "driveType"],
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
                    "invitation",
                    "roles"
                  ],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%permissions_id_pattern%$"
                    },
                    "grantedToV2": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName", "id"],
                          "properties": {
                            "displayName": {
                              "const": "Brian Murphy"
                            }
                          }
                        }
                      }
                    },
                    "invitation": {
                      "type": "object",
                      "required": ["invitedBy"],
                      "properties": {
                        "invitedBy": {
                          "type": "object",
                          "required": ["user"],
                          "properties": {
                            "user": {
                              "type": "object",
                              "required": ["displayName", "id"],
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
              }
            }
          }
        }
      }
      """

  @env-config @issue-8314
  Scenario: sharee lists the folder share after sharer is deleted (Project space)
    Given the config "GRAPH_SPACES_USERS_CACHE_TTL" has been set to "1"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" lists the shares shared with him after clearing user cache using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "folder",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "folder": {
                  "const": {}
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "folder"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "folder",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions"
                  ],
                  "properties": {
                    "eTag": {
                      "type": "string",
                      "pattern": "%etag_pattern%"
                    },
                    "folder": {
                      "const": {}
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "folder"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%file_id_pattern%$"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation",
                          "roles"
                        ],
                        "properties": {
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "properties": {
                                  "displayName": {
                                    "const": "Brian Murphy"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern":"^%user_id_pattern%$"
                                  }
                                },
                                "required": [
                                  "displayName",
                                  "id"
                                ]
                              }
                            }
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "invitedBy": {
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
                                        "const": ""
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    }
                                  }
                                }
                              }
                            }
                          },
                          "roles": {
                            "type": "array",
                            "maxItems": 1,
                            "minItems": 1,
                            "items": {
                              "type": "string",
                              "pattern": "^%role_id_pattern%$"
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
      }
      """

  @env-config @issue-8314
  Scenario: sharee lists the file share with after sharer is deleted (Project space)
    Given the config "GRAPH_SPACES_USERS_CACHE_TTL" has been set to "1"
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testfile.txt |
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And the user "Admin" has deleted a user "Alice"
    When user "Brian" lists the shares shared with him after clearing user cache using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden",
                "@client.synchronize",
                "eTag",
                "file",
                "id",
                "lastModifiedDateTime",
                "name",
                "parentReference",
                "remoteItem",
                "size"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                },
                "@client.synchronize":{
                  "const": true
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "file": {
                  "type": "object",
                  "required": ["mimeType"],
                  "properties": {
                    "mimeType": {
                      "type": "string",
                      "pattern": "^text/plain"
                    }
                  }
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "const": "testfile.txt"
                },
                "parentReference": {
                  "type": "object",
                  "required": [
                    "driveId",
                    "driveType",
                    "id"
                  ],
                  "properties": {
                    "driveId": {
                      "type": "string",
                      "pattern": "^%space_id_pattern%$"
                    },
                    "driveType" : {
                      "const": "virtual"
                    },
                    "id" : {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    }
                  }
                },
                "remoteItem": {
                  "type": "object",
                  "required": [
                    "eTag",
                    "file",
                    "id",
                    "lastModifiedDateTime",
                    "name",
                    "parentReference",
                    "permissions",
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
                          "type": "string",
                          "pattern": "^text/plain"
                        }
                      }
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "const": "testfile.txt"
                    },
                    "parentReference": {
                      "type": "object",
                      "required": [
                        "driveId",
                        "driveType"
                      ],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%file_id_pattern%$"
                        },
                        "driveType" : {
                          "const": "project"
                        }
                      }
                    },
                    "permissions": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "grantedToV2",
                          "id",
                          "invitation"
                        ],
                        "properties": {
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "grantedToV2": {
                            "type": "object",
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "properties": {
                                  "displayName": {
                                    "const": "Brian Murphy"
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern":"^%user_id_pattern%$"
                                  }
                                },
                                "required": [
                                  "displayName",
                                  "id"
                                ]
                              }
                            }
                          },
                          "invitation": {
                            "type": "object",
                            "properties": {
                              "invitedBy": {
                                "type": "object",
                                "properties": {
                                  "user": {
                                    "type": "object",
                                    "properties": {
                                      "displayName": {
                                        "const": ""
                                      },
                                      "id": {
                                        "type": "string",
                                        "pattern": "^%user_id_pattern%$"
                                      }
                                    },
                                    "required": [
                                      "displayName",
                                      "id"
                                    ]
                                  }
                                },
                                "required": [
                                  "user"
                                ]
                              }
                            },
                            "required": [
                              "invitedBy"
                            ]
                          },
                          "roles": {
                            "type": "array",
                            "maxItems": 1,
                            "minItems": 1,
                            "items": {
                              "type": "string",
                              "pattern": "^%role_id_pattern%$"
                            }
                          }
                        }
                      }
                    },
                    "size": {
                      "const": 12
                    }
                  }
                },
                "size": {
                  "const": 12
                }
              }
            }
          }
        }
      }
      """


  Scenario: sharee lists the folder share after the sharer is disabled (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "new-space"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folder    |
      | space           | new-space |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And the user "Admin" has disabled user "Alice"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should contain resource "folder" with the following data:
      """
      {
        "type": "object",
        "required": [
          "@UI.Hidden",
          "@client.synchronize",
          "eTag",
          "folder",
          "id",
          "lastModifiedDateTime",
          "name",
          "parentReference",
          "remoteItem"
        ],
        "properties": {
          "@UI.Hidden":{
            "const": false
          },
          "@client.synchronize":{
            "const": true
          },
          "folder": {
            "const": {}
          },
          "id": {
            "type": "string",
            "pattern": "^%share_id_pattern%$"
          },
          "name": {
            "const": "folder"
          },
          "parentReference": {
            "type": "object",
            "required": [
              "driveId",
              "driveType",
              "id"
            ],
            "properties": {
              "driveType" : {
                "const": "virtual"
              }
            }
          },
          "remoteItem": {
            "type": "object",
            "required": [
              "eTag",
              "folder",
              "id",
              "lastModifiedDateTime",
              "name",
              "parentReference",
              "permissions"
            ],
            "properties": {
              "folder": {
                "const": {}
              },
              "id": {
                "type": "string",
                "pattern": "^%file_id_pattern%$"
              },
              "name": {
                "const": "folder"
              },
              "parentReference": {
                "type": "object",
                "required": ["driveId", "driveType"],
                "properties": {
                  "driveType" : {
                    "const": "project"
                  }
                }
              },
              "permissions": {
                "type": "array",
                "maxItems": 1,
                "minItems": 1,
                "items": {
                  "type": "object",
                  "required": [
                    "grantedToV2",
                    "id",
                    "invitation",
                    "roles"
                  ],
                  "properties": {
                    "id": {
                      "type": "string",
                      "pattern": "^%permissions_id_pattern%$"
                    },
                    "grantedToV2": {
                      "type": "object",
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "properties": {
                            "displayName": {
                              "const": "Brian Murphy"
                            }
                          },
                          "required": ["displayName", "id"]
                        }
                      }
                    },
                    "invitation": {
                      "type": "object",
                      "required": ["invitedBy"],
                      "properties": {
                        "invitedBy": {
                          "type": "object",
                          "required": ["user"],
                          "properties": {
                            "user": {
                              "type": "object",
                              "required": ["displayName", "id"],
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
                    "roles": {
                      "type": "array",
                      "maxItems": 1,
                      "minItems": 1,
                      "items": {
                        "type": "string",
                        "pattern": "^%role_id_pattern%$"
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


  Scenario Outline: sharee hides the shared resource (Personal space)
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" hides the shared resource "<resource>" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["@UI.Hidden"],
        "properties": {
          "@UI.Hidden": {
            "const": true
          }
        }
      }
      """
    Examples:
      | resource     |
      | testfile.txt |
      | folder       |


  Scenario Outline: sharee hides the shared resource (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "testfile.txt"
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Brian" hides the shared resource "<resource>" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["@UI.Hidden"],
        "properties": {
          "@UI.Hidden": {
            "const": true
          }
        }
      }
      """
    Examples:
      | resource      |
      | testfile.txt  |
      | FolderToShare |


  Scenario Outline: sharee lists the shares after hiding (Personal space)
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has hidden the share "<resource>"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": true
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | resource     |
      | testfile.txt |
      | folder       |


  Scenario Outline: sharee lists the shares after hiding (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "testfile.txt"
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has hidden the share "<resource>"
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": true
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | resource      |
      | testfile.txt  |
      | FolderToShare |


  Scenario Outline: sharee lists the shares after unhiding share (Personal space)
    Given user "Alice" has created folder "folder"
    And user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has hidden the share "<resource>"
    When user "Brian" unhides the shared resource "<resource>" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["@UI.Hidden"],
        "properties": {
          "@UI.Hidden": {
            "const": false
          }
        }
      }
      """
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden"
              ],
              "properties": {
                "@UI.Hidden":{
                  "const": false
                }
              }
            }
          }
        }
      }
      """
    Examples:
      | resource     |
      | testfile.txt |
      | folder       |


  Scenario Outline: sharee lists the shares after unhiding share (Project space)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "testfile.txt"
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | NewSpace   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Brian" has hidden the share "<resource>"
    When user "Brian" unhides the shared resource "<resource>" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["@UI.Hidden"],
        "properties": {
          "@UI.Hidden": {
            "const": false
          }
        }
      }
      """
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": ["value"],
        "properties": {
          "value": {
            "type": "array",
            "maxItems": 1,
            "minItems": 1,
            "items": {
              "type": "object",
              "required": [
                "@UI.Hidden"
              ],
              "properties": {
                "@UI.Hidden":{
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
      | testfile.txt  |
      | FolderToShare |

  @issue-9642
  Scenario: list multiple shares (> 10 shares)
    Given using spaces DAV path
    And user "Alice" has created the following folders
      | path     |
      | folder1  |
      | folder2  |
      | folder3  |
      | folder4  |
      | folder5  |
      | folder6  |
      | folder7  |
      | folder8  |
      | folder9  |
      | folder10 |
    And user "Alice" has uploaded the following files with content "lorem epsum"
      | path          |
      | textfile1.txt |
      | textfile2.txt |
      | textfile3.txt |
      | textfile4.txt |
      | textfile5.txt |
    And user "Alice" has shared the following files from space "Personal" with user "Brian" and role "Viewer":
      | textfile1.txt |
      | textfile2.txt |
      | textfile3.txt |
      | textfile4.txt |
      | textfile5.txt |
    And user "Alice" has shared the following folders from space "Personal" with user "Brian" and role "Editor":
      | folder1  |
      | folder2  |
      | folder3  |
      | folder4  |
      | folder5  |
      | folder6  |
      | folder7  |
      | folder8  |
      | folder9  |
      | folder10 |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And the json response should contain the following shares:
      | folder1       |
      | folder2       |
      | folder3       |
      | folder4       |
      | folder5       |
      | folder6       |
      | folder7       |
      | folder8       |
      | folder9       |
      | folder10      |
      | textfile1.txt |
      | textfile2.txt |
      | textfile3.txt |
      | textfile4.txt |
      | textfile5.txt |


  Scenario: sharee (also a group member) lists shares shared to group (Personal space)
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
    When user "Alice" lists the shares shared with her using the Graph API
    Then the HTTP status code should be "200"
    And the json response should contain the following shares:
      | textfile.txt  |
      | FolderToShare |


  Scenario: sharee (also a group member) lists shares shared to group (Project space)
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
    When user "Alice" lists the shares shared with her using the Graph API
    Then the HTTP status code should be "200"
    And the json response should contain the following shares:
      | textfile.txt  |
      | FolderToShare |

  @env-config
  Scenario: share a folder with denied permission role (Personal Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "FolderToShare/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Denied        |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And user "Brian" should not have a share "FolderToShare" shared by user "Alice" from space "Personal"
    And user "Brian" should not be able to download file "FolderToShare/lorem.txt" from space "Shares"

  @env-config
  Scenario: share a folder with denied permission role (Project Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "FolderToShare/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Denied        |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And user "Brian" should not have a share "FolderToShare" shared by user "Alice" from space "NewSpace"
    And user "Brian" should not be able to download file "FolderToShare/lorem.txt" from space "Shares"

  @env-config
  Scenario Outline: check share access after updating the permission role of a shared folder from other roles to Denied (Personal Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "FolderToShare/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole | Denied        |
      | space           | Personal      |
      | resource        | FolderToShare |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And user "Brian" should not have a share "FolderToShare" shared by user "Alice" from space "Personal"
    And user "Brian" should not be able to download file "FolderToShare/lorem.txt" from space "Shares"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |
      | Uploader         |

  @env-config
  Scenario Outline: check share access after updating the permission role of a shared folder from Denied to other roles (Personal Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has uploaded file with content "hello world" to "FolderToShare/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Denied        |
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole | <permissions-role> |
      | space           | Personal           |
      | resource        | FolderToShare      |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And user "Brian" should have a share "FolderToShare" shared by user "Alice" from space "Personal"
    And for user "Brian" the content of the file "FolderToShare/lorem.txt" of the space "Shares" should be "hello world"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |
      | Uploader         |

  @env-config
  Scenario Outline: check share access after updating the permission role of a shared folder from other roles to Denied (Project Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "FolderToShare/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare      |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole | Denied        |
      | space           | NewSpace      |
      | resource        | FolderToShare |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And user "Brian" should not have a share "FolderToShare" shared by user "Alice" from space "NewSpace"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |
      | Uploader         |

  @env-config
  Scenario Outline: check share access after updating the permission role of a shared folder from Denied to other roles (Project Space)
    Given using spaces DAV path
    And the administrator has enabled the permissions role "Denied"
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has uploaded a file inside space "NewSpace" with content "hello world" to "FolderToShare/lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FolderToShare |
      | space           | NewSpace      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Denied        |
    And user "Alice" has updated the last resource share with the following properties:
      | permissionsRole | <permissions-role> |
      | space           | NewSpace           |
      | resource        | FolderToShare      |
    When user "Brian" lists the shares shared with him using the Graph API
    Then the HTTP status code should be "200"
    And user "Brian" should have a share "FolderToShare" shared by user "Alice" from space "NewSpace"
    And for user "Brian" the content of the file "FolderToShare/lorem.txt" of the space "Shares" should be "hello world"
    Examples:
      | permissions-role |
      | Editor           |
      | Viewer           |
      | Uploader         |
