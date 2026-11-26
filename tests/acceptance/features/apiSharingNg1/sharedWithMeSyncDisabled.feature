Feature: listing sharedWithMe when auto-sync is disabled
  As a user
  I want to get resources shared with me when auto-sync is disabled
  So that I can know about what resources I have access to

  https://docs.opencloud.eu/libre-graph-api/#/me.drive/ListSharedWithMe

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And user "Brian" has disabled the auto-sync share


  Scenario: user lists the file shared with them
    Given user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
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
        "required": [
          "value"
        ],
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
                  "enum": [false]
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
                  "enum": ["textfile.txt"]
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
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
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": ["textfile.txt"]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
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
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id"],
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
                      "type": "number",
                      "enum": [8]
                    }
                  }
                },
                "size": {
                  "type": "number",
                  "enum": [8]
                }
              }
            }
          }
        }
      }
      """


  Scenario: user lists the folder shared with them
    Given user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
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
        "required": [
          "value"
        ],
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
                  "enum": [false]
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
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": ["folderToShare"]
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
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
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": ["folderToShare"]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
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
                            "required": ["user"],
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id"],
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


  Scenario: group member lists the file shared with them
    Given group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
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
            "maxItems": 1,
            "minItems": 1,
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
                  "enum": [false]
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
                  "enum": ["textfile.txt"]
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
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
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": ["textfile.txt"]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
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
                            "properties": {
                              "group": {
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "type": "string",
                                    "enum": ["grp1"]
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%group_id_pattern%$"
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
                              "invitedBy": {
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
                      "type": "number",
                      "enum": [8]
                    }
                  }
                },
                "size": {
                  "type": "number",
                  "enum": [8]
                }
              }
            }
          }
        }
      }
      """


  Scenario: group member lists the folder shared with them
    Given group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
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
            "maxItems": 1,
            "minItems": 1,
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
                  "enum": [false]
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
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": ["folderToShare"]
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
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
                      "required": ["user"],
                      "properties": {
                        "user": {
                          "type": "object",
                          "required": ["displayName","id"],
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
                    "id": {
                      "type": "string",
                      "pattern": "^%file_id_pattern%$"
                    },
                    "name": {
                      "type": "string",
                      "enum": ["folderToShare"]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
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
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "type": "string",
                                    "enum": ["grp1"]
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%group_id_pattern%$"
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
                              "invitedBy": {
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


  Scenario: user lists file shares shared with them from project-space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
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
                "@UI.Hidden": {
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize": {
                  "type": "boolean",
                  "enum": [false]
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
                  "enum": ["textfile.txt"]
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
                      "enum": ["textfile.txt"]
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
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
                          "type": "string",
                          "enum": ["project"]
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
                      "type": "number",
                      "enum": [8]
                    }
                  }
                },
                "size": {
                  "type": "number",
                  "enum": [8]
                }
              }
            }
          }
        }
      }
      """


  Scenario: user lists folder shares shared with them from project-space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | projectSpace  |
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
                "@UI.Hidden": {
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize": {
                  "type": "boolean",
                  "enum": [false]
                },
                "eTag": {
                  "type": "string",
                  "pattern": "%etag_pattern%"
                },
                "folder": {
                  "type": "object"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%share_id_pattern%$"
                },
                "name": {
                  "type": "string",
                  "enum": ["folderToShare"]
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
                      "type": "string",
                      "enum": ["folderToShare"]
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
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
                          "type": "string",
                          "enum": ["project"]
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
                          "id": {
                            "type": "string",
                            "pattern": "^%permissions_id_pattern%$"
                          },
                          "invitation": {
                            "type": "object",
                            "required": ["invitedBy"],
                            "properties": {
                              "invitedBy": {
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
                                        "enum": ["Alice Hansen"]
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


  Scenario: group member lists the file shared with them from project-space
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
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
                "@UI.Hidden": {
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize": {
                  "type": "boolean",
                  "enum": [false]
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
                  "enum": ["textfile.txt"]
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
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
                      "enum": ["textfile.txt"]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
                          "type": "string",
                          "enum": ["project"]
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
                            "properties": {
                              "group": {
                                "type": "object",
                                "required": ["displayName", "id"],
                                "properties": {
                                  "displayName": {
                                    "type": "string",
                                    "enum": ["grp1"]
                                  },
                                  "id": {
                                    "type": "string",
                                    "pattern": "^%group_id_pattern%$"
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
                              "invitedBy": {
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
                      "type": "number",
                      "enum": [8]
                    }
                  }
                },
                "size": {
                  "type": "number",
                  "enum": [8]
                }
              }
            }
          }
        }
      }
      """


  Scenario: group member lists the space folder shared with them from project-space
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | projectSpace  |
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
                "@UI.Hidden": {
                  "type": "boolean",
                  "enum": [false]
                },
                "@client.synchronize": {
                  "type": "boolean",
                  "enum": [false]
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
                  "type": "string",
                  "enum": ["folderToShare"]
                },
                "parentReference": {
                  "type": "object",
                  "required": ["driveId", "driveType", "id"],
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
                      "type": "string",
                      "enum": ["folderToShare"]
                    },
                    "parentReference": {
                      "type": "object",
                      "required": ["driveId", "driveType"],
                      "properties": {
                        "driveId": {
                          "type": "string",
                          "pattern": "^%space_id_pattern%$"
                        },
                        "driveType": {
                          "type": "string",
                          "enum": ["project"]
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
                            "properties": {
                              "user": {
                                "type": "object",
                                "required": ["displayName", "id"],
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


  Scenario: user who is also a member of group lists the file shared with them
    Given group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And user "Alice" has uploaded file with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | Viewer       |
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
            "maxItems": 1,
            "minItems": 1,
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
                  "enum": [false]
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
                                "required": ["user"],
                                "properties": {
                                  "group": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
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
                            "grantedToV2",
                            "id",
                            "invitation",
                            "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["group"],
                                "properties": {
                                  "group": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["grp1"]
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
              }
            }
          }
        }
      }
      """


  Scenario: user who is also a member of group lists the folder shared with them
    Given group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And user "Alice" has created folder "folderToShare"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
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
            "maxItems": 1,
            "minItems": 1,
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
                  "enum": [false]
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
                                "required": ["user"],
                                "properties": {
                                  "group": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
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
                            "grantedToV2",
                            "id",
                            "invitation",
                            "roles"
                            ],
                            "properties": {
                              "grantedToV2": {
                                "type": "object",
                                "required": ["group"],
                                "properties": {
                                  "group": {
                                    "type": "object",
                                    "required": ["displayName", "id"],
                                    "properties": {
                                      "displayName": {
                                        "type": "string",
                                        "enum": ["grp1"]
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
              }
            }
          }
        }
      }
      """


  Scenario: user lists the file with same name shared by two users with him/her
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
                      "type": "boolean",
                      "enum": [false]
                    },
                    "@client.synchronize": {
                      "type": "boolean",
                      "enum": [false]
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
                              "type": "string",
                              "enum": ["Carol King"]
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "type": "string",
                      "enum": ["textfile.txt"]
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
                                  "type": "string",
                                  "enum": ["Carol King"]
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "type": "string",
                          "enum": ["textfile.txt"]
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
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
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
                                            "type": "string",
                                            "enum": ["Carol King"]
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
                      "type": "boolean",
                      "enum": [false]
                    },
                    "@client.synchronize": {
                      "type": "boolean",
                      "enum": [false]
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
                              "type": "string",
                              "enum": ["Alice Hansen"]
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "type": "string",
                      "enum": ["textfile.txt"]
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
                          "type": "string",
                          "enum": ["textfile.txt"]
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
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
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
                                            "type": "string",
                                            "enum": ["Alice Hansen"]
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


  Scenario: user lists the folder with same name shared by two users with him/her
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
                      "type": "boolean",
                      "enum": [false]
                    },
                    "@client.synchronize": {
                      "type": "boolean",
                      "enum": [false]
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
                              "type": "string",
                              "enum": ["Carol King"]
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "type": "string",
                      "enum": ["folderToShare"]
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
                                  "type": "string",
                                  "enum": ["Carol King"]
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "type": "string",
                          "enum": ["folderToShare"]
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
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
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
                                            "type": "string",
                                            "enum": ["Carol King"]
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
                      "type": "boolean",
                      "enum": [false]
                    },
                    "@client.synchronize": {
                      "type": "boolean",
                      "enum": [false]
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
                              "type": "string",
                              "enum": ["Alice Hansen"]
                            }
                          }
                        }
                      }
                    },
                    "name": {
                      "type": "string",
                      "enum": ["folderToShare"]
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
                                  "type": "string",
                                  "enum": ["Alice Hansen"]
                                }
                              }
                            }
                          }
                        },
                        "name": {
                          "type": "string",
                          "enum": ["folderToShare"]
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
                                        "type": "string",
                                        "enum": ["Brian Murphy"]
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
                                            "type": "string",
                                            "enum": ["Alice Hansen"]
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


  Scenario: user who is also a member of group lists file shared with them from project-space
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace" with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | projectSpace |
      | sharee          | grp1         |
      | shareType       | group        |
      | permissionsRole | File Editor  |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | projectSpace |
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
                "@UI.Hidden": {
                  "const": false
                },
                "@client.synchronize": {
                  "const": false
                },
                "name": {
                  "const": "textfile.txt"
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
                                "required": ["group"],
                                "properties": {
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
                                        "pattern": "^%group_id_pattern%$"
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


  Scenario: user who is also a member of group lists folder shared with them from project-space
    Given using spaces DAV path
    And group "grp1" has been created
    And the following users have been added to the following groups
      | username | groupname |
      | Brian    | grp1      |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | projectSpace  |
      | sharee          | grp1          |
      | shareType       | group         |
      | permissionsRole | Viewer        |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | projectSpace  |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Editor        |
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
                "@UI.Hidden": {
                  "const": false
                },
                "@client.synchronize": {
                  "const": false
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
                                "required": ["user"],
                                "properties": {
                                  "group": {
                                    "type": "object",
                                    "required": [
                                      "displayName",
                                      "id"
                                    ],
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
                                "required": ["group"],
                                "properties": {
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
                                        "pattern": "^%group_id_pattern%$"
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


  Scenario: sharee lists the files with same name shared from different project-spaces
    Given user "Carol" has been created with default attributes
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "Space Admin" to user "Carol" using the Graph API
    And user "Alice" has created a space "projectSpace1" with the default quota using the Graph API
    And user "Carol" has created a space "projectSpace2" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "projectSpace1" with content "to share" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt  |
      | space           | projectSpace1 |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Carol" has uploaded a file inside space "projectSpace2" with content "to share" to "textfile.txt"
    And user "Carol" has sent the following resource share invitation:
      | resource        | textfile.txt  |
      | space           | projectSpace2 |
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
                      "const": false
                    },
                    "@client.synchronize": {
                      "const": false
                    },
                    "name": {
                      "const": "textfile.txt"
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
                          "const": "virtual"
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
                        "eTag",
                        "file",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "name": {
                          "const": "textfile.txt"
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
                              "pattern": "^%space_id_pattern%$"
                            },
                            "driveType": {
                              "const": "project"
                            }
                          }
                        },
                        "permissions": {
                          "minItems": 1,
                          "maxItems": 1,
                          "type": "array",
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
                      "const": false
                    },
                    "name": {
                      "const": "textfile.txt"
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
                          "const": "virtual"
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
                        "eTag",
                        "file",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "name": {
                          "const": "textfile.txt"
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
                              "pattern": "^%space_id_pattern%$"
                            },
                            "driveType": {
                              "const": "project"
                            }
                          }
                        },
                        "permissions": {
                          "minItems": 1,
                          "maxItems": 1,
                          "type": "array",
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


  Scenario: sharee lists the folders with same name shared from different project-spaces
    Given user "Carol" has been created with default attributes
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "Space Admin" to user "Carol" using the Graph API
    And user "Alice" has created a space "projectSpace1" with the default quota using the Graph API
    And user "Carol" has created a space "projectSpace2" with the default quota using the Graph API
    And user "Alice" has created a folder "folderToShare" in space "projectSpace1"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | projectSpace1 |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Carol" has created a folder "folderToShare" in space "projectSpace2"
    And user "Carol" has sent the following resource share invitation:
      | resource        | folderToShare |
      | space           | projectSpace2 |
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
            "maxItems": 2,
            "minItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
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
                    "@UI.Hidden": {
                      "const": false
                    },
                    "@client.synchronize": {
                      "const": false
                    },
                    "name": {
                      "const": "folderToShare"
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
                          "const": "virtual"
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
                        "eTag",
                        "folder",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "name": {
                          "const": "folderToShare"
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
                              "pattern": "^%space_id_pattern%$"
                            },
                            "driveType": {
                              "const": "project"
                            }
                          }
                        },
                        "permissions": {
                          "minItems": 1,
                          "maxItems": 1,
                          "type": "array",
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
                      "const": false
                    },
                    "name": {
                      "const": "folderToShare"
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
                          "const": "virtual"
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
                        "eTag",
                        "folder",
                        "id",
                        "lastModifiedDateTime",
                        "name",
                        "parentReference",
                        "permissions"
                      ],
                      "properties": {
                        "name": {
                          "const": "folderToShare"
                        },
                        "permissions": {
                          "minItems": 1,
                          "maxItems": 1,
                          "type": "array",
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
                }
              ]
            }
          }
        }
      }
      """
