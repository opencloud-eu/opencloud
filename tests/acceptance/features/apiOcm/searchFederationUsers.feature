@ocm
Feature: search federation users
  As a user
  I can find federation users after accepting an invitation to share resources


  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Carol    |
    And using server "REMOTE"
    And user "Brian" has been created with default attributes

  @issue-9813
  Scenario: users search for federation users by display name
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    When user "Brian" searches for federated user "ali" using Graph API
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
                "displayName",
                "id",
                "userType",
                "identities"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                },
                "userType": {
                  "type": "string",
                  "const": "Federated"
                },
                "identities": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": [
                      "issuer",
                      "issuerAssignedId"
                    ],
                    "properties": {
                      "issuer": {
                        "const": "%local_host_port%"
                      },
                      "issuerAssignedId": {
                        "type": "string",
                        "pattern": "^%federated_user_id_pattern%$"
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
    And using server "LOCAL"
    When user "Alice" searches for federated user "bri" using Graph API
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
                "displayName",
                "id",
                "userType",
                "identities"
              ],
              "properties": {
                "displayName": {
                  "const": "Brian Murphy"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                },
                "userType": {
                  "type": "string",
                  "const": "Federated"
                },
                "identities": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": [
                      "issuer",
                      "issuerAssignedId"
                    ],
                    "properties": {
                      "issuer": {
                        "const": "%remote_host_port%"
                      },
                      "issuerAssignedId": {
                        "type": "string",
                        "pattern": "^%federated_user_id_pattern%$"
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

  @issue-9813
  Scenario: user search for federation users by email
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    When user "Brian" searches for federated user "%22alice@example.org%22" using Graph API
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
                "displayName",
                "id",
                "userType",
                "identities"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                },
                "userType": {
                  "type": "string",
                  "const": "Federated"
                },
                "identities": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": [
                      "issuer",
                      "issuerAssignedId"
                    ],
                    "properties": {
                      "issuer": {
                        "const": "%local_host_port%"
                      },
                      "issuerAssignedId": {
                        "type": "string",
                        "pattern": "^%federated_user_id_pattern%$"
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
    And using server "LOCAL"
    When user "Alice" searches for federated user "%22brian@example.org%22" using Graph API
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
                "displayName",
                "id",
                "userType",
                "identities"
              ],
              "properties": {
                "displayName": {
                  "const": "Brian Murphy"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                },
                "userType": {
                  "type": "string",
                  "const": "Federated"
                },
                "identities": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": [
                      "issuer",
                      "issuerAssignedId"
                    ],
                    "properties": {
                      "issuer": {
                        "const": "%remote_host_port%"
                      },
                      "issuerAssignedId": {
                        "type": "string",
                        "pattern": "^%federated_user_id_pattern%$"
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


  Scenario: users search for federation users without federated connection
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    When user "Brian" searches for federated user "%22carol@example.org%22" using Graph API
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
    And using server "LOCAL"
    When user "Carol" searches for federated user "bria" using Graph API
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


  Scenario: users search all federation users
    Given using server "REMOTE"
    And "Brian" has created the federation share invitation
    And using server "LOCAL"
    And "Alice" has accepted invitation
    And "Carol" has accepted invitation
    When "Alice" searches for accepted users
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "array",
        "minItems": 1,
        "maxItems": 1,
        "items": {
          "type": "object",
          "required": [
            "display_name",
            "idp",
            "user_id",
            "mail"
          ],
          "properties": {
            "display_name": {
              "type": "string",
              "const": "Brian Murphy"
            },
            "idp": {
              "type": "string",
              "const": "%remote_host_port%"
            },
            "mail": {
              "type": "string",
              "pattern": "brian@example.org"
            },
            "user_id": {
              "type": "string",
              "pattern": "^%federated_user_id_pattern%$"
            }
          }
        }
      }
      """
    When using server "REMOTE"
    And "Brian" searches for accepted users
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "array",
        "minItems": 2,
        "maxItems": 2,
        "uniqueItems": true,
        "items": {
          "oneOf": [
            {
              "type": "object",
              "required": [
                "display_name",
                "idp",
                "user_id",
                "mail"
              ],
              "properties": {
                "display_name": {
                  "const": "Alice Hansen"
                },
                "idp": {
                  "const": "%local_host_port%"
                },
                "mail": {
                  "pattern": "alice@example.org"
                },
                "user_id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                }
              }
            },
            {
              "type": "object",
              "required": [
                "display_name",
                "idp",
                "user_id",
                "mail"
              ],
              "properties": {
                "display_name": {
                  "const": "Carol King"
                },
                "idp": {
                  "const": "%local_host_port%"
                },
                "mail": {
                  "pattern": "carol@example.org"
                },
                "user_id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                }
              }
            }
          ]
        }
      }
      """

  @issue-9829
  Scenario: admin gets federated users
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    When the administrator gets federated users using the Graph API
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
                "displayName",
                "id",
                "mail",
                "userType",
                "identities"
              ],
              "properties": {
                "displayName": {
                  "const": "Alice Hansen"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                },
                "mail": {
                  "const": "alice@example.org"
                },
                "userType": {
                  "const": "Federated"
                },
                "identities": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": [
                      "issuer",
                      "issuerAssignedId"
                    ],
                    "properties": {
                      "issuer": {
                        "const": "%local_host_port%"
                      },
                      "issuerAssignedId": {
                        "type": "string",
                        "pattern": "^%uuidv4_pattern%$"
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
    And using server "LOCAL"
    When the administrator gets federated users using the Graph API
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
                "displayName",
                "id",
                "mail",
                "userType",
                "identities"
              ],
              "properties": {
                "displayName": {
                  "const": "Brian Murphy"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%federated_user_id_pattern%$"
                },
                "mail": {
                  "const": "brian@example.org"
                },
                "userType": {
                  "const": "Federated"
                },
                "identities": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "type": "object",
                    "required": [
                      "issuer",
                      "issuerAssignedId"
                    ],
                    "properties": {
                      "issuer": {
                        "const": "%remote_host_port%"
                      },
                      "issuerAssignedId": {
                        "type": "string",
                        "pattern": "^%uuidv4_pattern%$"
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


  Scenario: user without admin permissions cannot get federated users
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    When user "Carol" tries to get federated users using the Graph API
    Then the HTTP status code should be "403"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "error"
        ],
        "properties": {
          "error": {
            "type": "object",
            "required": [
              "code",
              "message"
            ],
            "properties": {
              "code": {
                "const": "accessDenied"
              },
              "message": {
                "const": "search term too short"
              }
            }
          }
        }
      }
      """

  @issue-9829
  Scenario: admin gets federated and local users
    Given using server "LOCAL"
    And "Alice" has created the federation share invitation
    And using server "REMOTE"
    And "Brian" has accepted invitation
    And using server "LOCAL"
    When the administrator gets federated and local users using the Graph API
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
            "maxItems": 2,
            "minItems": 2,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "type": "object",
                  "required": [
                    "accountEnabled",
                    "displayName",
                    "id",
                    "mail",
                    "onPremisesSamAccountName",
                    "surname",
                    "userType"
                  ],
                  "properties": {
                    "accountEnabled": {
                      "const": true
                    },
                    "displayName": {
                      "const": "Carol King"
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%user_id_pattern%$"
                    },
                    "mail": {
                      "const": "carol@example.org"
                    },
                    "onPremisesSamAccountName": {
                      "const": "Carol"
                    },
                    "surname": {
                      "const": "Carol"
                    },
                    "userType": {
                      "const": "Member"
                    }
                  }
                },
                {
                  "type": "object",
                  "required": [
                    "displayName",
                    "id",
                    "mail",
                    "userType",
                    "identities"
                  ],
                  "properties": {
                    "displayName": {
                      "const": "Brian Murphy"
                    },
                    "id": {
                      "type": "string",
                      "pattern": "^%federated_user_id_pattern%$"
                    },
                    "mail": {
                      "const": "brian@example.org"
                    },
                    "userType": {
                      "const": "Federated"
                    },
                    "identities": {
                      "type": "array",
                      "minItems": 1,
                      "maxItems": 1,
                      "items": {
                        "type": "object",
                        "required": [
                          "issuer",
                          "issuerAssignedId"
                        ],
                        "properties": {
                          "issuer": {
                            "const": "%remote_host_port%"
                          },
                          "issuerAssignedId": {
                            "type": "string",
                            "pattern": "^%uuidv4_pattern%$"
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

# TODO try to find federation users after deleting federated conection
