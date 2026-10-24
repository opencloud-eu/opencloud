Feature: download multiple resources bundled into an archive
  As a user
  I want to be able to download multiple items at once
  So that I don't have to execute repetitive tasks

  As a developer
  I want to be able to use the resource ID to download multiple items at once
  So that I don't have to know the full path of the resource

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: download a single file
    Given user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    When user "Alice" downloads the <archive-type> archive of "/textfile0.txt" using the resource id and setting these headers:
      | header     | value        |
      | User-Agent | <user-agent> |
    Then the HTTP status code should be "200"
    And the downloaded <archive-type> archive should contain these files:
      | name          | content   |
      | textfile0.txt | some data |
    Examples:
      | user-agent | archive-type |
      | Linux      | tar          |
      | Windows NT | zip          |


  Scenario Outline: download a single folder
    Given user "Alice" has created folder "my_data"
    And user "Alice" has uploaded file with content "some data" to "/my_data/textfile0.txt"
    And user "Alice" has uploaded file with content "more data" to "/my_data/an_other_file.txt"
    When user "Alice" downloads the <archive-type> archive of "/my_data" using the resource id and setting these headers:
      | header     | value        |
      | User-Agent | <user-agent> |
    Then the HTTP status code should be "200"
    And the downloaded <archive-type> archive should contain these files:
      | name                      | content   |
      | my_data/textfile0.txt     | some data |
      | my_data/an_other_file.txt | more data |
    Examples:
      | user-agent | archive-type |
      | Linux      | zip          |
      | Windows NT | tar          |


  Scenario: download multiple files and folders
    Given user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "other data" to "/textfile1.txt"
    And user "Alice" has created folder "my_data"
    And user "Alice" has uploaded file with content "some data" to "/my_data/textfile2.txt"
    And user "Alice" has created folder "more_data"
    And user "Alice" has uploaded file with content "more data" to "/more_data/an_other_file.txt"
    When user "Alice" downloads the archive of these items using the resource ids
      | textfile0.txt |
      | textfile1.txt |
      | my_data       |
      | more_data     |
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name                        | content    |
      | textfile0.txt               | some data  |
      | textfile1.txt               | other data |
      | my_data/textfile2.txt       | some data  |
      | more_data/an_other_file.txt | more data  |


  Scenario: download a single file as different user
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    When user "Brian" downloads the archive of "/textfile0.txt" of user "Alice" using the resource id
    Then the HTTP status code should be "404"


  Scenario: download multiple shared items as share receiver
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "other data" to "/textfile1.txt"
    And user "Alice" has created folder "my_data"
    And user "Alice" has uploaded file with content "some data" to "/my_data/textfile2.txt"
    And user "Alice" has created folder "more_data"
    And user "Alice" has uploaded file with content "more data" to "/more_data/an_other_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile1.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | my_data  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "my_data" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | more_data |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Brian" has a share "more_data" synced
    When user "Brian" downloads the archive of these items using the resource ids
      | /Shares/textfile0.txt |
      | /Shares/textfile1.txt |
      | /Shares/my_data       |
      | /Shares/more_data     |
    Then the HTTP status code should be "200"
    And the downloaded zip archive should contain these files:
      | name                        | content    |
      | textfile0.txt               | some data  |
      | textfile1.txt               | other data |
      | my_data/textfile2.txt       | some data  |
      | more_data/an_other_file.txt | more data  |

  @issue-4636
  Scenario Outline: download the Shares folder as share receiver
    Given user "Brian" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "other data" to "/textfile1.txt"
    And user "Alice" has created folder "my_data"
    And user "Alice" has uploaded file with content "some data" to "/my_data/textfile2.txt"
    And user "Alice" has created folder "more_data"
    And user "Alice" has uploaded file with content "more data" to "/more_data/an_other_file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile0.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile0.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | textfile1.txt |
      | space           | Personal      |
      | sharee          | Brian         |
      | shareType       | user          |
      | permissionsRole | Viewer        |
    And user "Brian" has a share "textfile1.txt" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | my_data  |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    And user "Brian" has a share "my_data" synced
    And user "Alice" has sent the following resource share invitation:
      | resource        | more_data |
      | space           | Personal  |
      | sharee          | Brian     |
      | shareType       | user      |
      | permissionsRole | Viewer    |
    And user "Brian" has a share "more_data" synced
    When user "Brian" downloads the <archive-type> archive of "/Shares" using the resource id and setting these headers:
      | header     | value        |
      | User-Agent | <user-agent> |
    Then the HTTP status code should be "200"
    And the downloaded <archive-type> archive should contain these files:
      | name                        | content    |
      | textfile0.txt               | some data  |
      | textfile1.txt               | other data |
      | my_data/textfile2.txt       | some data  |
      | more_data/an_other_file.txt | more data  |
    Examples:
      | user-agent | archive-type |
      | Linux      | tar          |
      | Windows NT | zip          |


  Scenario Outline: download a single folder into an archive named after it
    Given user "Alice" has created folder "my_data"
    And user "Alice" has uploaded file with content "some data" to "my_data/textfile0.txt"
    When user "Alice" downloads the <archive-type> archive of "my_data" using the resource id and setting these headers:
      | header     | value |
      | User-Agent | Linux |
    Then the HTTP status code should be "200"
    And the following headers should match these regular expressions
      | Content-Disposition | /filename="my_data\.<archive-type>"/ |
    Examples:
      | archive-type |
      | zip          |
      | tar          |
