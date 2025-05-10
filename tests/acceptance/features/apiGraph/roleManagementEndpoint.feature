Feature: permissions role definitions
  As a user
  I want to get the role management endpoints
  So that I can find out if those endpoints are working correctly or not

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: get a list of permissions role definitions
    When user "Alice" gets a list of permissions role definitions using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "array",
        "maxItems": 7,
        "minItems": 7,
        "uniqueItems": true,
        "items": {
          "oneOf": [
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 10
                },
                "description": {
                  "const": "View and download."
                },
                "displayName": {
                  "const": "Can view"
                },
                "id": {
                  "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 4,
                  "minItems": 4,
                  "uniqueItems": true,
                  "items": {
                    "oneOf": [
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.File"
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.Folder"
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.File \u0026\u0026 @Subject.UserType==\"Federated\""
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.Folder \u0026\u0026 @Subject.UserType==\"Federated\""
                          }
                        }
                      }
                    ]
                  }
                }
              }
            },
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 40
                },
                "description": {
                  "const": "View and download."
                },
                "displayName": {
                  "const": "Can view"
                },
                "id": {
                  "const": "a8d5fe5e-96e3-418d-825b-534dbdf22b99"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "uniqueItems": true,
                  "items": {
                    "type": "object",
                    "required": [
                      "allowedResourceActions",
                      "condition"
                    ],
                    "properties": {
                      "allowedResourceActions": {
                        "const": [
                          "libre.graph/driveItem/path/read",
                          "libre.graph/driveItem/quota/read",
                          "libre.graph/driveItem/content/read",
                          "libre.graph/driveItem/permissions/read",
                          "libre.graph/driveItem/children/read",
                          "libre.graph/driveItem/deleted/read",
                          "libre.graph/driveItem/basic/read"
                        ]
                      },
                      "condition": {
                        "const": "exists @Resource.Root"
                      }
                    }
                  }
                }
              }
            },
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 60
                },
                "description": {
                  "const": "View, download, upload, edit, add and delete."
                },
                "displayName": {
                  "const": "Can edit"
                },
                "id": {
                  "const": "fb6c3e19-e378-47e5-b277-9732f9de6e21"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 2,
                  "minItems": 2,
                  "uniqueItems": true,
                  "items": {
                    "oneOf": [
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/children/create",
                              "libre.graph/driveItem/standard/delete",
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/upload/create",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/path/update",
                              "libre.graph/driveItem/deleted/update",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.Folder"
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/children/create",
                              "libre.graph/driveItem/standard/delete",
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/upload/create",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/path/update",
                              "libre.graph/driveItem/deleted/update",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.Folder \u0026\u0026 @Subject.UserType==\"Federated\""
                          }
                        }
                      }
                    ]
                  }
                }
              }
            },
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 90
                },
                "description": {
                  "const": "View, download, upload, edit, add, delete including the history."
                },
                "displayName": {
                  "const": "Can edit"
                },
                "id": {
                  "const": "58c63c02-1d89-4572-916a-870abc5a1b7d"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "uniqueItems": true,
                  "items": {
                    "type": "object",
                    "required": [
                      "allowedResourceActions",
                      "condition"
                    ],
                    "properties": {
                      "allowedResourceActions": {
                        "const": [
                          "libre.graph/driveItem/children/create",
                          "libre.graph/driveItem/standard/delete",
                          "libre.graph/driveItem/path/read",
                          "libre.graph/driveItem/quota/read",
                          "libre.graph/driveItem/content/read",
                          "libre.graph/driveItem/upload/create",
                          "libre.graph/driveItem/permissions/read",
                          "libre.graph/driveItem/children/read",
                          "libre.graph/driveItem/versions/read",
                          "libre.graph/driveItem/deleted/read",
                          "libre.graph/driveItem/path/update",
                          "libre.graph/driveItem/versions/update",
                          "libre.graph/driveItem/deleted/update",
                          "libre.graph/driveItem/basic/read"
                        ]
                      },
                      "condition": {
                        "const": "exists @Resource.Root"
                      }
                    }
                  }
                }
              }
            },
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 100
                },
                "description": {
                  "const": "View, download and edit."
                },
                "displayName": {
                  "const": "Can edit"
                },
                "id": {
                  "const": "2d00ce52-1fc2-4dbc-8b95-a73b73395f5a"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 2,
                  "minItems": 2,
                  "uniqueItems": true,
                  "items": {
                    "oneOf": [
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/upload/create",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/deleted/update",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const":"exists @Resource.File"
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/upload/create",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/deleted/update",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const":"exists @Resource.File \u0026\u0026 @Subject.UserType==\"Federated\""
                          }
                        }
                      }
                    ]
                  }
                }
              }
            },
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 50
                },
                "description": {
                  "const": "View, download and upload."
                },
                "displayName": {
                  "const": "Can upload"
                },
                "id": {
                  "const": "1c996275-f1c9-4e71-abdf-a42f6495e960"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "uniqueItems": true,
                  "items": {
                    "type": "object",
                    "required": [
                      "allowedResourceActions",
                      "condition"
                    ],
                    "properties": {
                      "allowedResourceActions": {
                        "const": [
                          "libre.graph/driveItem/children/create",
                          "libre.graph/driveItem/path/read",
                          "libre.graph/driveItem/content/read",
                          "libre.graph/driveItem/upload/create",
                          "libre.graph/driveItem/children/read",
                          "libre.graph/driveItem/path/update",
                          "libre.graph/driveItem/basic/read"
                        ]
                      },
                      "condition": {
                        "const": "exists @Resource.Folder"
                      }
                    }
                  }
                }
              }
            },
            {
              "type": "object",
              "required": [
                "@libre.graph.weight",
                "description",
                "displayName",
                "id",
                "rolePermissions"
              ],
              "properties": {
                "@libre.graph.weight": {
                  "const": 120
                },
                "description": {
                  "const": "View, download, upload, edit, add, delete and manage members."
                },
                "displayName": {
                  "const": "Can manage"
                },
                "id": {
                  "const": "312c0871-5ef7-4b3a-85b6-0e4074c64049"
                },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 1,
                  "minItems": 1,
                  "uniqueItems": true,
                  "items": {
                    "type": "object",
                    "required": [
                      "allowedResourceActions",
                      "condition"
                    ],
                    "properties": {
                      "allowedResourceActions": {
                        "const": [
                          "libre.graph/driveItem/permissions/create",
                          "libre.graph/driveItem/children/create",
                          "libre.graph/driveItem/standard/delete",
                          "libre.graph/driveItem/path/read",
                          "libre.graph/driveItem/quota/read",
                          "libre.graph/driveItem/content/read",
                          "libre.graph/driveItem/upload/create",
                          "libre.graph/driveItem/permissions/read",
                          "libre.graph/driveItem/children/read",
                          "libre.graph/driveItem/versions/read",
                          "libre.graph/driveItem/deleted/read",
                          "libre.graph/driveItem/path/update",
                          "libre.graph/driveItem/permissions/delete",
                          "libre.graph/driveItem/deleted/delete",
                          "libre.graph/driveItem/versions/update",
                          "libre.graph/driveItem/deleted/update",
                          "libre.graph/driveItem/basic/read",
                          "libre.graph/driveItem/permissions/update",
                          "libre.graph/driveItem/permissions/deny"
                        ]
                      },
                      "condition": {
                        "const": "exists @Resource.Root"
                      }
                    }
                  }
                }
              }
            }
          ]
        }
      }
      """


  Scenario: get details of a specific permission role definition
    When user "Alice" gets the "Viewer" role definition using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
          "required": [
            "@libre.graph.weight",
            "description",
            "displayName",
            "id",
            "rolePermissions"
          ],
          "properties": {
            "@libre.graph.weight":{
              "const": 10
            },
            "description": {
              "const": "View and download."
            },
            "displayName": {
              "const": "Can view"
            },
            "id": {
              "const": "b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5"
            },
                "rolePermissions": {
                  "type": "array",
                  "maxItems": 4,
                  "minItems": 4,
                  "uniqueItems": true,
                  "items": {
                    "oneOf": [
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.File"
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.Folder"
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.File \u0026\u0026 @Subject.UserType==\"Federated\""
                          }
                        }
                      },
                      {
                        "type": "object",
                        "required": [
                          "allowedResourceActions",
                          "condition"
                        ],
                        "properties": {
                          "allowedResourceActions": {
                            "const": [
                              "libre.graph/driveItem/path/read",
                              "libre.graph/driveItem/quota/read",
                              "libre.graph/driveItem/content/read",
                              "libre.graph/driveItem/children/read",
                              "libre.graph/driveItem/deleted/read",
                              "libre.graph/driveItem/basic/read"
                            ]
                          },
                          "condition": {
                            "const": "exists @Resource.Folder \u0026\u0026 @Subject.UserType==\"Federated\""
                          }
                        }
                      }
                    ]
                  }
                }
          }
      }
      """

  @env-config
  Scenario: get details of a secure viewer role definition
    Given the administrator has enabled the permissions role "Secure Viewer"
    When user "Alice" gets the "Secure Viewer" role definition using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "@libre.graph.weight",
          "description",
          "displayName",
          "id",
          "rolePermissions"
        ],
        "properties": {
          "@libre.graph.weight": {
            "const": 20
          },
          "description": {
            "const": "View only documents, images and PDFs. Watermarks will be applied."
          },
          "displayName": {
            "const": "Can view (secure)"
          },
          "id": {
            "const": "aa97fe03-7980-45ac-9e50-b325749fd7e6"
          },
          "rolePermissions": {
            "type": "array",
            "maxItems": 2,
            "minItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": [
                    "allowedResourceActions",
                    "condition"
                  ],
                  "properties": {
                    "allowedResourceActions": {
                      "const": [
                        "libre.graph/driveItem/path/read",
                        "libre.graph/driveItem/children/read",
                        "libre.graph/driveItem/basic/read"
                      ]
                    },
                    "condition": {
                      "const": "exists @Resource.File"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "allowedResourceActions",
                    "condition"
                  ],
                  "properties": {
                    "allowedResourceActions": {
                      "const": [
                        "libre.graph/driveItem/path/read",
                        "libre.graph/driveItem/children/read",
                        "libre.graph/driveItem/basic/read"
                      ]
                    },
                    "condition": {
                      "const": "exists @Resource.Folder"
                    }
                  }
                }
              ]
            }
          }
        }
      }
      """
