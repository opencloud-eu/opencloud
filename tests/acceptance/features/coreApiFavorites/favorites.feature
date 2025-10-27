Feature: favorite
  As a user
  I want to favorite resources
  So that I can access them quickly

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes
    And user "Alice" has uploaded file with content "some data" to "/textfile0.txt"
    And user "Alice" has uploaded file with content "some data" to "/textfile1.txt"
    And user "Alice" has uploaded file with content "some data" to "/textfile2.txt"
    And user "Alice" has uploaded file with content "some data" to "/textfile3.txt"
    And user "Alice" has uploaded file with content "some data" to "/textfile4.txt"
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has created folder "/PARENT"
    And user "Alice" has uploaded file with content "some data" to "/PARENT/parent.txt"

  @owncloud-issue-1263
  Scenario Outline: favorite a folder
    Given using <dav-path-version> DAV path
    When user "Alice" favorites element "/FOLDER" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Alice" folder "/FOLDER" should be favorited
    When user "Alice" gets the following properties of folder "/FOLDER" using the WebDAV API
      | propertyName |
      | oc:favorite  |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:favorite" with value "1"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1263
  Scenario Outline: unfavorite a folder
    Given using <dav-path-version> DAV path
    And user "Alice" has favorited element "/FOLDER"
    When user "Alice" unfavorites element "/FOLDER" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Alice" folder "/FOLDER" should not be favorited
    When user "Alice" gets the following properties of folder "/FOLDER" using the WebDAV API
      | propertyName |
      | oc:favorite  |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:favorite" with value "0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest @owncloud-issue-1263
  Scenario Outline: favorite a file
    Given using <dav-path-version> DAV path
    When user "Alice" favorites element "/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Alice" file "/textfile0.txt" should be favorited
    When user "Alice" gets the following properties of file "/textfile0.txt" using the WebDAV API
      | propertyName |
      | oc:favorite  |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:favorite" with value "1"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest @owncloud-issue-1263
  Scenario Outline: unfavorite a file
    Given using <dav-path-version> DAV path
    And user "Alice" has favorited element "/textfile0.txt"
    When user "Alice" unfavorites element "/textfile0.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And as user "Alice" file "/textfile0.txt" should not be favorited
    When user "Alice" gets the following properties of file "/textfile0.txt" using the WebDAV API
      | propertyName |
      | oc:favorite  |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:favorite" with value "0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @smokeTest @owncloud-issue-1228
  Scenario Outline: get favorited elements of a folder
    Given using <dav-path-version> DAV path
    When user "Alice" favorites element "/FOLDER" using the WebDAV API
    And user "Alice" favorites element "/textfile0.txt" using the WebDAV API
    And user "Alice" favorites element "/textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And user "Alice" should have the following favorited items
      | /FOLDER        |
      | /textfile0.txt |
      | /textfile1.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1228
  Scenario Outline: get favorited elements of a subfolder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/subfolder"
    And user "Alice" has uploaded file with content "some data" to "/subfolder/textfile0.txt"
    And user "Alice" has uploaded file with content "some data" to "/subfolder/textfile1.txt"
    And user "Alice" has uploaded file with content "some data" to "/subfolder/textfile2.txt"
    When user "Alice" favorites element "/subfolder/textfile0.txt" using the WebDAV API
    And user "Alice" favorites element "/subfolder/textfile1.txt" using the WebDAV API
    And user "Alice" favorites element "/subfolder/textfile2.txt" using the WebDAV API
    And user "Alice" unfavorites element "/subfolder/textfile1.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And user "Alice" should have the following favorited items
      | /subfolder/textfile0.txt |
      | /subfolder/textfile2.txt |
    And user "Alice" should not have the following favorited items
      | /subfolder/textfile1.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1228
  Scenario Outline: get favorited elements and limit count of entries
    Given using <dav-path-version> DAV path
    And user "Alice" has favorited element "/textfile0.txt"
    And user "Alice" has favorited element "/textfile1.txt"
    And user "Alice" has favorited element "/textfile2.txt"
    And user "Alice" has favorited element "/textfile3.txt"
    And user "Alice" has favorited element "/textfile4.txt"
    When user "Alice" lists the favorites and limits the result to 3 elements using the WebDAV API
    Then the search result should contain any "3" of these entries:
      | /textfile0.txt |
      | /textfile1.txt |
      | /textfile2.txt |
      | /textfile3.txt |
      | /textfile4.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1228
  Scenario Outline: get favorited elements paginated in subfolder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/subfolder"
    And user "Alice" has copied file "/textfile0.txt" to "/subfolder/textfile0.txt"
    And user "Alice" has copied file "/textfile0.txt" to "/subfolder/textfile1.txt"
    And user "Alice" has copied file "/textfile0.txt" to "/subfolder/textfile2.txt"
    And user "Alice" has copied file "/textfile0.txt" to "/subfolder/textfile3.txt"
    And user "Alice" has copied file "/textfile0.txt" to "/subfolder/textfile4.txt"
    And user "Alice" has copied file "/textfile0.txt" to "/subfolder/textfile5.txt"
    And user "Alice" has favorited element "/subfolder/textfile0.txt"
    And user "Alice" has favorited element "/subfolder/textfile1.txt"
    And user "Alice" has favorited element "/subfolder/textfile2.txt"
    And user "Alice" has favorited element "/subfolder/textfile3.txt"
    And user "Alice" has favorited element "/subfolder/textfile4.txt"
    And user "Alice" has favorited element "/subfolder/textfile5.txt"
    When user "Alice" lists the favorites and limits the result to 3 elements using the WebDAV API
    Then the search result should contain any "3" of these entries:
      | /subfolder/textfile0.txt |
      | /subfolder/textfile1.txt |
      | /subfolder/textfile2.txt |
      | /subfolder/textfile3.txt |
      | /subfolder/textfile4.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1228
  Scenario Outline: favoriting a folder does not change the favorite state of elements inside the folder
    Given using <dav-path-version> DAV path
    When user "Alice" favorites element "/PARENT/parent.txt" using the WebDAV API
    And user "Alice" favorites element "/PARENT" using the WebDAV API
    Then the HTTP status code should be "207"
    And user "Alice" should have the following favorited items
      | /PARENT            |
      | /PARENT/parent.txt |
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
