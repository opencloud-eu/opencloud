@skipOnReva
Feature: accessing a public link share
  As a person who knows a public link
  I want be able to access the preview of a public link file
  So that I can see a small preview of the file before deciding to download it

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |


  Scenario: access to the preview of password protected public link without providing the password is not allowed
    Given user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testavatar.jpg |
      | space           | Personal       |
      | permissionsRole | edit           |
      | password        | %public%       |
    When the public accesses the preview of file "testavatar.jpg" from the last shared public link using the sharing API
    Then the HTTP status code should be "404"

  @env-config @owncloud-issue-10341
  Scenario: access to the preview of public shared file without password
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "testavatar.jpg"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | testavatar.jpg |
      | space           | Personal       |
      | permissionsRole | edit           |
    When the public accesses the preview of file "testavatar.jpg" from the last shared public link using the sharing API
    Then the HTTP status code should be "200"


  Scenario: access to the preview of password protected public shared file inside a folder without providing the password is not allowed
    Given user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "FOLDER/testavatar.jpg"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "FOLDER/textfile0.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | edit     |
      | password        | %public% |
    When the public accesses the preview of the following files from the last shared public link using the sharing API
      | path           |
      | testavatar.jpg |
      | textfile0.txt  |
    Then the HTTP status code of responses on all endpoints should be "404"

  @env-config @owncloud-issue-10341
  Scenario: access to the preview of public shared file inside a folder without password
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And user "Alice" has created folder "FOLDER"
    And user "Alice" has uploaded file "filesForUpload/testavatar.jpg" to "FOLDER/testavatar.jpg"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "FOLDER/textfile0.txt"
    And using SharingNG
    And user "Alice" has created the following resource link share:
      | resource        | FOLDER   |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public accesses the preview of the following files from the last shared public link using the sharing API
      | path           |
      | testavatar.jpg |
      | textfile0.txt  |
    Then the HTTP status code of responses on all endpoints should be "200"
