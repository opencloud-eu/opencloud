@ocm
Feature: ocm well-known URI
  As a user
  I want to verify the response of well-known URI
  So that I can ensure the configuration works correctly


  Scenario: check the ocm well-known endpoint response
    Given using server "LOCAL"
    When a user requests "/.well-known/ocm" with "GET" and no authentication
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "enabled",
          "apiVersion",
          "endPoint",
          "provider",
          "resourceTypes",
          "capabilities"
        ],
        "properties": {
          "enabled": {
            "const": true
          },
          "apiVersion": {
            "const": "1.2.0"
          },
          "endPoint": {
            "pattern": "^%base_url%/ocm"
          },
          "provider": {
            "const": "OpenCloud"
          },
          "resourceTypes": {
            "type": "array",
            "minItems": 1,
            "maxItems": 1,
            "items": {
              "type": "object",
              "required": [
                "name",
                "shareTypes",
                "protocols"
              ],
              "properties": {
                "name": {
                  "const": "file"
                },
                "sharesTypes": {
                  "type": "array",
                  "minItems": 1,
                  "maxItems": 1,
                  "items": {
                    "const": "user"
                  }
                },
                "protocols": {
                  "type": "object",
                  "required": [
                    "webdav"
                  ],
                  "properties": {
                    "webdav": {
                      "const": "/dav/ocm"
                    }
                  }
                }
              }
            }
          },
          "capabilities": {
            "type": "array",
            "minItems": 4,
            "maxItems": 4,
            "uniqueItems": true,
            "items": {
              "oneOf": [
                {
                  "const": "invites"
                },
                {
                  "const": "webdav-uri"
                },
                {
                  "const": "protocol-object"
                },
                {
                  "const": "invite-wayf"
                }
              ]
            }
          }
        }
      }
      """
