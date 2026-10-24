@tikaServiceNeeded
Feature: propfind extracted props
  As a user
  I want to get extracted properties of resource
  So that I can make sure that the response contains audio, location, image and photo properties

  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path


  Scenario: check extracted properties of a file from project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file "filesForUpload/testaudio.mp3" to "testaudio.mp3" in space "new-space"
    And user "Alice" has uploaded a file "filesForUpload/testavatar.jpg" to "testavatar.jpg" in space "new-space"
    When user "Alice" gets the following extracted properties of resource "testaudio.mp3" inside space "new-space" using the WebDAV API
      | propertyName |
      | oc:audio     |
    Then the HTTP status code should be "207"
    And as user "Alice" the REPORT response should contain a resource "testaudio.mp3" with these key and value pairs:
      | key                | value                          |
      | oc:audio/oc:album  | ALBUM1234567890123456789012345 |
      | oc:audio/oc:artist | ARTIST123456789012345678901234 |
      | oc:audio/oc:genre  | Pop                            |
      | oc:audio/oc:title  | TITLE1234567890123456789012345 |
      | oc:audio/oc:track  | 1                              |
      | oc:audio/oc:year   | 2001                           |
    When user "Alice" gets the following extracted properties of resource "testavatar.jpg" inside space "new-space" using the WebDAV API
      | propertyName |
      | oc:image     |
      | oc:location  |
      | oc:photo     |
    Then the HTTP status code should be "207"
    And as user "Alice" the REPORT response should contain a resource "testavatar.jpg" with these key and value pairs:
      | key                              | value                |
      | oc:image/oc:width                | 1402                 |
      | oc:image/oc:height               | 500                  |
      | oc:location/oc:latitude          | 52.437804            |
      | oc:location/oc:longitude         | 13.341866            |
      | oc:photo/oc:camera-make          | NIKON                |
      | oc:photo/oc:camera-model         | COOLPIX P6000        |
      | oc:photo/oc:f-number             | 4.5                  |
      | oc:photo/oc:focal-length         | 6                    |


  Scenario: check extracted properties of a file from personal space
    Given user "Alice" has uploaded file "filesForUpload/testaudio.mp3" to "testaudio.mp3"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    When user "Alice" gets the following extracted properties of resource "testaudio.mp3" inside space "Personal" using the WebDAV API
      | propertyName |
      | oc:audio     |
    Then the HTTP status code should be "207"
    And as user "Alice" the REPORT response should contain a resource "testaudio.mp3" with these key and value pairs:
      | key                | value                          |
      | oc:audio/oc:album  | ALBUM1234567890123456789012345 |
      | oc:audio/oc:artist | ARTIST123456789012345678901234 |
      | oc:audio/oc:genre  | Pop                            |
      | oc:audio/oc:title  | TITLE1234567890123456789012345 |
      | oc:audio/oc:track  | 1                              |
      | oc:audio/oc:year   | 2001                           |
    When user "Alice" gets the following extracted properties of resource "testavatar.jpg" inside space "Personal" using the WebDAV API
      | propertyName |
      | oc:image     |
      | oc:location  |
      | oc:photo     |
    Then the HTTP status code should be "207"
    And as user "Alice" the REPORT response should contain a resource "testavatar.jpg" with these key and value pairs:
      | key                              | value                |
      | oc:image/oc:width                | 1402                 |
      | oc:image/oc:height               | 500                  |
      | oc:location/oc:latitude          | 52.437804            |
      | oc:location/oc:longitude         | 13.341866            |
      | oc:photo/oc:camera-make          | NIKON                |
      | oc:photo/oc:camera-model         | COOLPIX P6000        |
      | oc:photo/oc:f-number             | 4.5                  |
      | oc:photo/oc:focal-length         | 6                    |


  Scenario: check extracted properties of a file by sharee from shares space
    Given these users have been created with default attributes:
      | username |
      | Brian    |
    And user "Alice" has uploaded file "filesForUpload/testaudio.mp3" to "testaudio.mp3"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testaudio.mp3 |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "testaudio.mp3" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | testavatar.jpg |
      | space           | Personal       |
      | sharee          | Brian          |
      | shareType       | user           |
      | permissionsRole | Viewer         |
    And user "Brian" has a share "testavatar.jpg" synced
    When user "Brian" gets the following extracted properties of resource "testaudio.mp3" inside space "Shares" using the WebDAV API
      | propertyName |
      | oc:audio     |
    Then the HTTP status code should be "207"
    And as user "Brian" the REPORT response should contain a resource "testaudio.mp3" with these key and value pairs:
      | key                | value                          |
      | oc:audio/oc:album  | ALBUM1234567890123456789012345 |
      | oc:audio/oc:artist | ARTIST123456789012345678901234 |
      | oc:audio/oc:genre  | Pop                            |
      | oc:audio/oc:title  | TITLE1234567890123456789012345 |
      | oc:audio/oc:track  | 1                              |
      | oc:audio/oc:year   | 2001                           |
    When user "Brian" gets the following extracted properties of resource "testavatar.jpg" inside space "Shares" using the WebDAV API
      | propertyName |
      | oc:image     |
      | oc:location  |
      | oc:photo     |
    Then the HTTP status code should be "207"
    And as user "Brian" the REPORT response should contain a resource "testavatar.jpg" with these key and value pairs:
      | key                              | value                |
      | oc:image/oc:width                | 1402                 |
      | oc:image/oc:height               | 500                  |
      | oc:location/oc:latitude          | 52.437804            |
      | oc:location/oc:longitude         | 13.341866            |
      | oc:photo/oc:camera-make          | NIKON                |
      | oc:photo/oc:camera-model         | COOLPIX P6000        |
      | oc:photo/oc:f-number             | 4.5                  |
      | oc:photo/oc:focal-length         | 6                    |


  Scenario: GET extracted properties of an audio file (Personal space)
    Given user "Alice" has uploaded a file "filesForUpload/testaudio.mp3" to "testaudio.mp3" in space "Personal"
    When user "Alice" gets the file "testaudio.mp3" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should eventually match
      """
        {
          "type": "object",
          "required": [
            "audio"
          ],
          "properties": {
            "audio": {
              "type": "object",
              "required": [
                "album",
                "artist",
                "genre",
                "title",
                "track",
                "year"
              ],
              "properties": {
                "album": {
                  "const": "ALBUM1234567890123456789012345"
                },
                "artist": {
                  "const": "ARTIST123456789012345678901234"
                },
                "genre": {
                  "const": "Pop"
                },
                "title": {
                  "const": "TITLE1234567890123456789012345"
                },
                "track": {
                  "const": 1
                },
                "year": {
                  "const": 2001
                }
              }
            }
          }
        }
      """


  Scenario: GET extracted properties of an image file (Personal space)
    Given user "Alice" has uploaded a file "filesForUpload/testavatar.jpg" to "testavatar.jpg" in space "Personal"
    When user "Alice" gets the file "testavatar.jpg" from space "Personal" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should eventually match
      """
        {
          "type": "object",
          "required": [
            "image",
            "location",
            "photo"
          ],
          "properties": {
            "image": {
              "type": "object",
              "required": [ "height", "width" ],
              "properties": {
                "height": {
                  "const": 500
                },
                "width": {
                  "const": 1402
                }
              }
            },
            "location": {
              "type": "object",
              "required": [ "latitude", "longitude" ],
              "properties": {
                "latitude": {
                  "const": 52.437804
                },
                "longitude": {
                  "const": 13.341866
                }
              }
            },
            "photo": {
              "type": "object",
              "required": [
                "cameraMake",
                "cameraModel",
                "fNumber",
                "focalLength",
                "orientation"
              ],
              "properties": {
                "cameraMake": {
                  "const": "NIKON"
                },
                "cameraModel": {
                  "const": "COOLPIX P6000"
                },
                "fNumber": {
                  "const": 4.5
                },
                "focalLength": {
                  "const": 6
                },
                "orientation": {
                  "const": 1
                }
              }
            }
          }
        }
      """


  Scenario: GET extracted properties of an audio file (Project space)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file "filesForUpload/testaudio.mp3" to "testaudio.mp3" in space "new-space"
    When user "Alice" gets the file "testaudio.mp3" from space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should eventually match
      """
        {
          "type": "object",
          "required": [
            "audio"
          ],
          "properties": {
            "audio": {
              "type": "object",
              "required": [
                "album",
                "artist",
                "genre",
                "title",
                "track",
                "year"
              ],
              "properties": {
                "album": {
                  "const": "ALBUM1234567890123456789012345"
                },
                "artist": {
                  "const": "ARTIST123456789012345678901234"
                },
                "genre": {
                  "const": "Pop"
                },
                "title": {
                  "const": "TITLE1234567890123456789012345"
                },
                "track": {
                  "const": 1
                },
                "year": {
                  "const": 2001
                }
              }
            }
          }
        }
      """


  Scenario: GET extracted properties of an image file (Project space)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file "filesForUpload/testavatar.jpg" to "testavatar.jpg" in space "new-space"
    When user "Alice" gets the file "testavatar.jpg" from space "new-space" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should eventually match
      """
        {
          "type": "object",
          "required": [
            "image",
            "location",
            "photo"
          ],
          "properties": {
            "image": {
              "type": "object",
              "required": [ "height", "width" ],
              "properties": {
                "height": {
                  "const": 500
                },
                "width": {
                  "const": 1402
                }
              }
            },
            "location": {
              "type": "object",
              "required": [ "latitude", "longitude" ],
              "properties": {
                "latitude": {
                  "const": 52.437804
                },
                "longitude": {
                  "const": 13.341866
                }
              }
            },
            "photo": {
              "type": "object",
              "required": [
                "cameraMake",
                "cameraModel",
                "fNumber",
                "focalLength",
                "orientation"
              ],
              "properties": {
                "cameraMake": {
                  "const": "NIKON"
                },
                "cameraModel": {
                  "const": "COOLPIX P6000"
                },
                "fNumber": {
                  "const": 4.5
                },
                "focalLength": {
                  "const": 6
                },
                "orientation": {
                  "const": 1
                }
              }
            }
          }
        }
      """


  Scenario: GET extracted properties of an audio file (Shares space)
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded a file "filesForUpload/testaudio.mp3" to "testaudio.mp3" in space "Personal"
    And user "Alice" has sent the following resource share invitation:
      | resource           | testaudio.mp3        |
      | space              | Personal             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | Viewer               |
    And user "Brian" has a share "testaudio.mp3" synced
    When user "Brian" gets the file "testaudio.mp3" from space "Shares" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should eventually match
      """
        {
          "type": "object",
          "required": [
            "audio"
          ],
          "properties": {
            "audio": {
              "type": "object",
              "required": [
                "album",
                "artist",
                "genre",
                "title",
                "track",
                "year"
              ],
              "properties": {
                "album": {
                  "const": "ALBUM1234567890123456789012345"
                },
                "artist": {
                  "const": "ARTIST123456789012345678901234"
                },
                "genre": {
                  "const": "Pop"
                },
                "title": {
                  "const": "TITLE1234567890123456789012345"
                },
                "track": {
                  "const": 1
                },
                "year": {
                  "const": 2001
                }
              }
            }
          }
        }
      """


  Scenario: GET extracted properties of an image file (Shares space)
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded a file "filesForUpload/testavatar.jpg" to "testavatar.jpg" in space "Personal"
    And user "Alice" has sent the following resource share invitation:
      | resource           | testavatar.jpg       |
      | space              | Personal             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | Viewer               |
    And user "Brian" has a share "testavatar.jpg" synced
    When user "Brian" gets the file "testavatar.jpg" from space "Shares" using the Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should eventually match
      """
        {
          "type": "object",
          "required": [
            "image",
            "location",
            "photo"
          ],
          "properties": {
            "image": {
              "type": "object",
              "required": [ "height", "width" ],
              "properties": {
                "height": {
                  "const": 500
                },
                "width": {
                  "const": 1402
                }
              }
            },
            "location": {
              "type": "object",
              "required": [ "latitude", "longitude" ],
              "properties": {
                "latitude": {
                  "const": 52.437804
                },
                "longitude": {
                  "const": 13.341866
                }
              }
            },
            "photo": {
              "type": "object",
              "required": [
                "cameraMake",
                "cameraModel",
                "fNumber",
                "focalLength",
                "orientation"
              ],
              "properties": {
                "cameraMake": {
                  "const": "NIKON"
                },
                "cameraModel": {
                  "const": "COOLPIX P6000"
                },
                "fNumber": {
                  "const": 4.5
                },
                "focalLength": {
                  "const": 6
                },
                "orientation": {
                  "const": 1
                }
              }
            }
          }
        }
      """