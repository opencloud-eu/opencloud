@email
Feature: Email notification
  As a user
  I want to get email notification of events related to me
  So that I can stay updated about the events

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |


  Scenario: user gets an email notification when someone shares a project space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    When user "Alice" sends the following space share invitation using root endpoint of the Graph API:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    Then the HTTP status code should be "200"
    And user "Brian" should have received the following email from user "Alice" about the share of project space "new-space"
      """
      Hello Brian Murphy,

      %displayname% has invited you to join "new-space".

      Click here to view it: %base_url%/f/%space_id%
      """


  Scenario: user gets an email notification when someone shares a file
    Given user "Alice" has uploaded file with content "sample text" to "lorem.txt"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | lorem.txt  |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    Then the HTTP status code should be "200"
    And user "Brian" should have received the following email from user "Alice"
      """
      Hello Brian Murphy

      %displayname% has shared "lorem.txt" with you.

      Click here to view it: %base_url%/files/shares/with-me
      """


  Scenario: group members get an email notification when someone shares a project space with the group
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Carol" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Carol" has been added to group "group1"
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    Then the HTTP status code should be "200"
    When user "Alice" sends the following space share invitation using root endpoint of the Graph API:
      | space           | new-space    |
      | sharee          | group1       |
      | shareType       | group        |
      | permissionsRole | Space Viewer |
    Then the HTTP status code should be "200"
    And user "Brian" should have received the following email from user "Alice" about the share of project space "new-space"
      """
      Hello Brian Murphy,

      %displayname% has invited you to join "new-space".

      Click here to view it: %base_url%/f/%space_id%
      """
    And user "Carol" should have received the following email from user "Alice" about the share of project space "new-space"
      """
      Hello Carol King,

      %displayname% has invited you to join "new-space".

      Click here to view it: %base_url%/f/%space_id%
      """

  @issue-183
  Scenario: group members get an email notification in their respective languages when someone shares a folder with the group
    Given user "Carol" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Carol" has been added to group "group1"
    # And user "Brian" has switched the system language to "es" using the Graph API
    And user "Carol" has switched the system language to "de" using the Graph API
    And user "Alice" has created folder "/HelloWorld"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | HelloWorld |
      | space           | Personal   |
      | sharee          | group1     |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    Then the HTTP status code should be "200"
    # And user "Brian" should have received the following email from user "Alice"
    #   """
    #   Hola Brian Murphy

    #   %displayname% ha compartido "HelloWorld" contigo.

    #   Click aquí para verlo: %base_url%/files/shares/with-me
    #   """
    And user "Carol" should have received the following email from user "Alice"
      """
      Hallo Carol King

      %displayname% hat "HelloWorld" mit Ihnen geteilt.

      Zum Ansehen hier klicken: %base_url%/files/shares/with-me
      """

  @issue-183
  Scenario: group members get an email notification in their respective languages when someone shares a file with the group
    Given user "Carol" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Carol" has been added to group "group1"
    # And user "Brian" has switched the system language to "es" using the Graph API
    And user "Carol" has switched the system language to "de" using the Graph API
    And user "Alice" has uploaded file with content "hello world" to "text.txt"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | text.txt |
      | space           | Personal |
      | sharee          | group1   |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    Then the HTTP status code should be "200"
    # And user "Brian" should have received the following email from user "Alice"
    #   """
    #   Hola Brian Murphy

    #   %displayname% ha compartido "text.txt" contigo.

    #   Click aquí para verlo: %base_url%/files/shares/with-me
    #   """
    And user "Carol" should have received the following email from user "Alice"
      """
      Hallo Carol King

      %displayname% hat "text.txt" mit Ihnen geteilt.

      Zum Ansehen hier klicken: %base_url%/files/shares/with-me
      """

  @issue-183
  Scenario: group members get an email notification in their respective languages when someone shares a space with the group
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Carol" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Carol" has been added to group "group1"
    # And user "Brian" has switched the system language to "es" using the Graph API
    And user "Carol" has switched the system language to "de" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" sends the following space share invitation using root endpoint of the Graph API:
      | space           | new-space    |
      | sharee          | group1       |
      | shareType       | group        |
      | permissionsRole | Space Viewer |
    Then the HTTP status code should be "200"
    # And user "Brian" should have received the following email from user "Alice" about the share of project space "new-space"
    #   """
    #   Hola Brian Murphy,

    #   Alice Hansen te ha invitado a unirte a "new-space".

    #   Click aquí para verlo: %base_url%/f/%space_id%
    #   """
    And user "Carol" should have received the following email from user "Alice" about the share of project space "new-space"
      """
      Hallo Carol King,

      Alice Hansen hat Sie eingeladen, dem Space "new-space" beizutreten.

      Zum Ansehen hier klicken: %base_url%/f/%space_id%
      """


  Scenario: user gets an email notification when space admin unshares a space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" unshares a space "new-space" to user "Brian"
    Then the HTTP status code should be "200"
    And user "Brian" should have received the following email from user "Alice" about the share of project space "new-space"
      """
      Hello Brian Murphy,

      %displayname% has removed you from "new-space".

      You might still have access through your other groups or direct membership.

      Click here to check it: %base_url%/f/%space_id%
      """

  @env-config
  Scenario: group members get an email notification in default language when someone shares a file with the group
    Given the config "OC_DEFAULT_LANGUAGE" has been set to "de"
    And user "Carol" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Carol" has been added to group "group1"
    And user "Alice" has uploaded file with content "hello world" to "text.txt"
    When user "Alice" sends the following resource share invitation using the Graph API:
      | resource        | text.txt |
      | space           | Personal |
      | sharee          | group1   |
      | shareType       | group    |
      | permissionsRole | Viewer   |
    Then the HTTP status code should be "200"
    And user "Brian" should have received the following email from user "Alice"
      """
      Hallo Brian Murphy

      %displayname% hat "text.txt" mit Ihnen geteilt.

      Zum Ansehen hier klicken: %base_url%/files/shares/with-me
      """
    And user "Carol" should have received the following email from user "Alice"
      """
      Hallo Carol King

      %displayname% hat "text.txt" mit Ihnen geteilt.

      Zum Ansehen hier klicken: %base_url%/files/shares/with-me
      """

  @issue-9530
  Scenario: user gets an email notification when someone with comma in display name shares a file
    Given the administrator has assigned the role "Admin" to user "Brian" using the Graph API
    And the user "Brian" has created a new user with the following attributes:
      | userName    | Carol             |
      | displayName | Carol, King       |
      | email       | carol@example.com |
      | password    | 1234              |
    And user "Carol" has uploaded file with content "sample text" to "lorem.txt"
    When user "Carol" sends the following resource share invitation using the Graph API:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    Then the HTTP status code should be "200"
    And user "Brian" should have received the following email from user "Carol"
      """
      Hello Brian Murphy

      Carol, King has shared "lorem.txt" with you.

      Click here to view it: %base_url%/files/shares/with-me
      """


  Scenario: user gets an email notification when a received share expires
    Given using SharingNG
    And user "Alice" has uploaded file with content "hello world" to "lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    When user "Alice" expires the last share of resource "lorem.txt" inside of the space "Personal"
    Then the HTTP status code should be "200"
    When user "Brian" lists the shares shared with him using the Graph API
    Then user "Brian" should have received the following email from user "Alice"
      """
      Your share to lorem.txt has expired at
      """


  Scenario: user gets an email notification when a space membership expires
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Viewer |
    When user "Alice" expires the user share of space "new-space" for user "Brian"
    Then the HTTP status code should be "200"
    # trigger the email notification
    When user "Brian" lists the shares shared with him using the Graph API
    Then user "Brian" should have received the following email from user "Alice"
      """
      Your membership of space new-space has expired at
      """


  Scenario: user gets an email notification when a received share is removed
    Given user "Alice" has uploaded file with content "sample text" to "lorem.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    When user "Alice" has removed the access of user "Brian" from resource "lorem.txt" of space "Personal"
    Then user "Brian" should have received the following email from user "Alice"
      """
      Hello Brian Murphy,

      %displayname% has unshared 'lorem.txt' with you.

      Even though this share has been revoked you still might have access through other shares and/or space memberships.
      """


  Scenario: no email is sent when the email sending interval is set to never
    Given user "Brian" has set the email sending interval to "never" using the settings API
    And user "Alice" has uploaded file with content "sample text" to "lorem.txt"
    And user "Alice" has uploaded file with content "more sample text" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    Then user "Brian" should have "0" emails


  Scenario Outline: queued notifications are delivered as one grouped email when the interval is <interval>
    Given user "Brian" has set the email sending interval to "<interval>" using the settings API
    And user "Alice" has uploaded file with content "sample text" to "lorem.txt"
    And user "Alice" has uploaded file with content "more sample text" to "textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | lorem.txt |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile.txt |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Viewer       |
    And user "Brian" should have "0" emails
    When the administrator sends the grouped "<interval>" email notifications using the CLI
    Then user "Brian" should have received the following email from user "Alice"
      """
      %displayname% has shared "lorem.txt" with you.
      """
    And user "Brian" should have received the following email from user "Alice"
      """
      %displayname% has shared "textfile.txt" with you.
      """
    And user "Brian" should have "1" emails
    Examples:
      | interval |
      | daily    |
      | weekly   |
