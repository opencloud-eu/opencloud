@email
Feature: share types of resources shared with a guest
  As a user
  I want to see which of my resources are shared with a guest
  So that I know who has access to them

  Background:
    Given using SharingNG
    And user "Alice" has been created with default attributes
    And user "Alice" has created folder "parent"
    And user "Alice" has created folder "parent/folderToShare"
    And user "Alice" has uploaded file with content "to share" to "parent/textfile.txt"


  Scenario Outline: check the share-types property of a resource shared with a guest
    Given user "Alice" has sent the following resource share invitation:
      | resource        | <resource>        |
      | space           | Personal          |
      | sharee          | guest@example.org |
      | shareType       | guest             |
      | permissionsRole | Viewer            |
    When user "Alice" gets the following properties of resource "<resource>" inside space "Personal" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain a share-types property with
      | 4 |
    Examples:
      | resource             |
      | parent/textfile.txt  |
      | parent/folderToShare |


  Scenario: check the share-types property of a resource shared with a user, a group and a guest
    Given user "Brian" has been created with default attributes
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent/textfile.txt |
      | space           | Personal            |
      | sharee          | Brian               |
      | shareType       | user                |
      | permissionsRole | Viewer              |
    And group "grp1" has been created
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent/textfile.txt |
      | space           | Personal            |
      | sharee          | grp1                |
      | shareType       | group               |
      | permissionsRole | Viewer              |
    And user "Alice" has sent the following resource share invitation:
      | resource        | parent/textfile.txt |
      | space           | Personal            |
      | sharee          | guest@example.org   |
      | shareType       | guest               |
      | permissionsRole | Viewer              |
    When user "Alice" gets the following properties of resource "parent/textfile.txt" inside space "Personal" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain a share-types property with
      | 0 |
      | 1 |
      | 4 |


  Scenario: check the share-types property of a resource after the guest share is removed
    Given user "Alice" has sent the following resource share invitation:
      | resource        | parent/textfile.txt |
      | space           | Personal            |
      | sharee          | guest@example.org   |
      | shareType       | guest               |
      | permissionsRole | Viewer              |
    And user "Alice" has removed the access of guest "guest@example.org" from resource "parent/textfile.txt" of space "Personal"
    When user "Alice" gets the following properties of resource "parent/textfile.txt" inside space "Personal" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain an empty property "oc:share-types"
