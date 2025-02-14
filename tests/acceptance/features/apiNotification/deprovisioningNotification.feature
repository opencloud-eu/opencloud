Feature: Deprovisioning notification
  As a user admin
  I want to inform users about shutting down and deprovisioning the instance
  So they can download and save their data in time

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |


  Scenario: administrator creates a deprovisioning notification about shutting down the instance
    When the administrator creates a deprovisioning notification
    And user "Alice" lists all notifications
    Then the HTTP status code of responses on each endpoint should be "200, 200" respectively
    And the JSON response should contain a notification message with the subject "Instance will be shut down and deprovisioned" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "app",
          "datetime",
          "message",
          "messageRich",
          "messageRichParameters",
          "notification_id",
          "object_id",
          "object_type",
          "subject",
          "subjectRich",
          "user"
        ],
        "properties": {
          "app": {
            "type": "string",
            "enum": [
              "userlog"
            ]
          },
          "message": {
            "type": "string",
            "enum": [
              "Attention! The instance will be shut down and deprovisioned on 2043-07-04T11:23:12Z. Download all your data before that date as no access past that date is possible."
            ]
          },
          "messageRich": {
            "type": "string",
            "enum": [
              "Attention! The instance will be shut down and deprovisioned on {date}. Download all your data before that date as no access past that date is possible."
            ]
          },
          "messageRichParameters": {
            "type": "object"
          },
          "notification_id": {
            "type": "string",
            "enum": [
              "deprovision"
            ]
          },
          "object_id": {
            "type": "string"
          },
          "object_type": {
            "type": "string",
            "enum": [
              "resource"
            ]
          },
          "subject": {
            "type": "string",
            "enum": [
              "Instance will be shut down and deprovisioned"
            ]
          },
          "subjectRich": {
            "type": "string",
            "enum": [
              "Instance will be shut down and deprovisioned"
            ]
          },
          "user": {
            "type": "string"
          }
        }
      }
      """


  Scenario Outline: non-admin user tries to create a deprovisioning notification
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to create a deprovisioning notification
    Then the HTTP status code should be "404"
    And user "Alice" should not have any notification
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |

  @issue-183
  Scenario Outline: get a deprovisioning notification in various languages
    Given the administrator has created a deprovisioning notification
    And user "Alice" has switched the system language to "<language>" using the Graph API
    When user "Alice" lists all notifications
    Then the HTTP status code should be "200"
    And the JSON response should contain a notification message with the subject "<subject>" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "message"
        ],
        "properties": {
          "message": {
            "type": "string",
            "enum": [
              "<message>"
            ]
          }
        }
      }
      """
    Examples:
      | language | subject                                                          | message                                                                                                                                                                                                 |
      | de       | Instanz wird heruntergefahren und außer Betrieb genommen werden. | Achtung! Diese Instanz wird am 2043-07-04T11:23:12Z heruntergefahren und außer Betrieb genommen werden. Laden Sie Ihre Daten vor diesem Tag herunter, da Sie danach nicht mehr darauf zugreifen können. |
      | es       | La instancia se cerrará y se desaprovisionará                    | ¡Atención! La instancia se cerrará y se desaprovisionará el 2043-07-04T11:23:12Z. Descarga todos tus datos antes de esa fecha, puesto que el acceso pasada la fecha no será posible.                    |


  Scenario: deprovisioning notification reappears again even after being marked as read
    Given the administrator has created a deprovisioning notification
    And user "Alice" has deleted all notifications
    When user "Alice" lists all notifications
    Then the HTTP status code should be "200"
    And the JSON response should contain a notification message with the subject "Instance will be shut down and deprovisioned" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "message"
        ],
        "properties": {
          "message": {
            "type": "string",
            "enum": [
              "Attention! The instance will be shut down and deprovisioned on 2043-07-04T11:23:12Z. Download all your data before that date as no access past that date is possible."
            ]
          }
        }
      }
      """


  Scenario: administrator deletes the deprovisioning notification
    Given the administrator has created a deprovisioning notification
    When the administrator deletes the deprovisioning notification
    Then the HTTP status code should be "200"
    And user "Alice" should not have any notification


  Scenario Outline: non-admin user tries to delete the deprovisioning notification
    Given the administrator has assigned the role "<user-role>" to user "Alice" using the Graph API
    When user "Alice" tries to delete the deprovisioning notification
    Then the HTTP status code should be "404"
    And user "Alice" should not have any notification
    Examples:
      | user-role   |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: administrator creates a deprovisioning notification with different date formats
    When the administrator creates a deprovisioning notification for date "<deprovision-date>" of format "<deprovision-date-format>"
    And user "Alice" lists all notifications
    Then the HTTP status code of responses on each endpoint should be "200, 200" respectively
    And the JSON response should contain a notification message with the subject "Instance will be shut down and deprovisioned" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "message"
        ],
        "properties": {
          "message": {
            "type": "string",
            "enum": [
              "Attention! The instance will be shut down and deprovisioned on <deprovision-date>. Download all your data before that date as no access past that date is possible."
            ]
          }
        }
      }
      """
    Examples:
      | deprovision-date                    | deprovision-date-format             |
      | 2030-04-09T15:04:05.999999999+07:00 | 2006-01-02T15:04:05.999999999Z07:00 |
      | 5:15PM                              | 3:04PM                              |
      # with date format like `Jan _2 15:04:05`, `_` gets replaced with a space in the response.
      | Jan  8 23:04:05                     | Jan _2 15:04:05                     |
      | Jan 12 15:04:05.000000000           | Jan _2 15:04:05.000000000           |
      | 2023-01-02 15:04:05                 | 2006-01-02 15:04:05                 |
      | 2023-01-02                          | 2006-01-02                          |
      | 18:24:55                            | 15:04:05                            |


  Scenario Outline: administrator change a deprovisioning notification with different date formats
    Given the administrator has created a deprovisioning notification
    When the administrator creates a deprovisioning notification for date "<deprovision-date>" of format "<deprovision-date-format>"
    And user "Alice" lists all notifications
    Then the HTTP status code of responses on each endpoint should be "200, 200" respectively
    And the JSON response should contain a notification message with the subject "Instance will be shut down and deprovisioned" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "message"
        ],
        "properties": {
          "message": {
            "type": "string",
            "enum": [
              "Attention! The instance will be shut down and deprovisioned on <deprovision-date>. Download all your data before that date as no access past that date is possible."
            ]
          }
        }
      }
      """
    Examples:
      | deprovision-date                | deprovision-date-format         |
      | 01/02 03:04:05PM '23 -0700      | 01/02 03:04:05PM '06 -0700      |
      | Mon Jan  2 15:04:05 UTC 2023    | Mon Jan _2 15:04:05 UTC 2006    |
      | Mon Jan 02 15:04:05 -0700 2023  | Mon Jan 02 15:04:05 -0700 2006  |
      | 02 Jan 23 15:04 -0700           | 02 Jan 06 15:04 -0700           |
      | Monday, 02-Jan-23 15:04:05 UTC  | Monday, 02-Jan-06 15:04:05 UTC  |
      | Mon, 02 Jan 2023 15:04:05 -0700 | Mon, 02 Jan 2006 15:04:05 -0700 |
      | 2023-01-02T15:04:05+07:00       | 2006-01-02T15:04:05Z07:00       |
