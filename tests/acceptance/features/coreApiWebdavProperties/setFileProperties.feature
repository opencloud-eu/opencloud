Feature: set file properties
  As a user
  I want to be able to set meta-information about files
  So that I can record file meta-information (detailed requirement TBD)

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes

  @smokeTest @owncloud-issue-1263
  Scenario Outline: setting custom DAV property and reading it
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "testcustomprop.txt"
    When user "Alice" sets property "very-custom-prop" of file "testcustomprop.txt" to "veryCustomPropValue"
    Then the HTTP status code should be "207"
    And the xml response should contain a property "very-custom-prop"
    And the content in the response should include the following content:
      """
      <d:prop><very-custom-prop></very-custom-prop></d:prop>
      """
    When user "Alice" gets a custom property "very-custom-prop" of file "testcustomprop.txt"
    Then the HTTP status code should be "207"
    And the response should contain a custom "very-custom-prop" property with value "veryCustomPropValue"
    And the content in the response should include the following content:
      """
      <d:prop><very-custom-prop>veryCustomPropValue</very-custom-prop></d:prop>
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1297
  Scenario Outline: setting custom complex DAV property and reading it
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/testcustomprop.txt"
    When user "Alice" sets property "very-custom-prop" of file "testcustomprop.txt" to "<foo xmlns='http://bar'/>"
    Then the HTTP status code should be "207"
    And the xml response should contain a property "very-custom-prop"
    When user "Alice" gets a custom property "very-custom-prop" of file "testcustomprop.txt"
    Then the HTTP status code should be "207"
    And the response should contain a custom "very-custom-prop" property with value "<foo xmlns='http://bar'/>"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1263
  Scenario Outline: setting custom DAV property and reading it after the file is renamed
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/testcustompropwithmove.txt"
    And user "Alice" has set property "very-custom-prop" of file "testcustompropwithmove.txt" to "valueForMovetest"
    And user "Alice" has moved file "/testcustompropwithmove.txt" to "/catchmeifyoucan.txt"
    When user "Alice" gets a custom property "very-custom-prop" of file "catchmeifyoucan.txt"
    Then the response should contain a custom "very-custom-prop" property with value "valueForMovetest"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva @owncloud-issue-1297
  Scenario Outline: setting custom DAV property on a shared file as an owner and reading as a recipient
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/testcustompropshared.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | testcustompropshared.txt |
      | space           | Personal                 |
      | sharee          | Brian                    |
      | shareType       | user                     |
      | permissionsRole | File Editor              |
    And user "Brian" has a share "testcustompropshared.txt" synced
    When user "Alice" sets property "very-custom-prop" of file "testcustompropshared.txt" to "valueForSharetest"
    Then the HTTP status code should be "207"
    And the xml response should contain a property "very-custom-prop"
    When user "Brian" gets a custom property "very-custom-prop" of file "Shares/testcustompropshared.txt"
    Then the HTTP status code should be "207"
    And the response should contain a custom "very-custom-prop" property with value "valueForSharetest"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @owncloud-issue-1263
  Scenario Outline: setting custom DAV property using one endpoint and reading it with other endpoint
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/testnewold.txt"
    When user "Alice" sets property "very-custom-prop" of file "testnewold.txt" to "lucky"
    Then the HTTP status code should be "207"
    And the xml response should contain a property "very-custom-prop"
    And using <dav-path-version-2> DAV path
    When user "Alice" gets a custom property "very-custom-prop" of file "testnewold.txt"
    Then the HTTP status code should be "207"
    And the response should contain a custom "very-custom-prop" property with value "lucky"
    Examples:
      | dav-path-version | dav-path-version-2 |
      | old              | new                |
      | new              | old                |
      | spaces           | new                |
      | spaces           | old                |
      | new              | spaces             |
      | old              | spaces             |

  @owncloud-issue-2140
  Scenario Outline: setting custom DAV property with custom namespace and reading it
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "testcustomprop.txt"
    When user "Alice" sets property "very-custom-prop" with namespace "x1='http://whatever.org/ns'" of file "testcustomprop.txt" to "veryCustomPropValue" using the WebDAV API
    Then the HTTP status code should be "207"
    And the xml response should contain a property "x1:very-custom-prop" with namespace "x1='http://whatever.org/ns'"
    And the content in the response should include the following content:
      """
      <d:prop><x1:very-custom-prop xmlns:x1="http://whatever.org/ns"></x1:very-custom-prop></d:prop>
      """
    When user "Alice" gets a custom property "x1:very-custom-prop" with namespace "x1='http://whatever.org/ns'" of file "testcustomprop.txt"
    Then the HTTP status code should be "207"
    And the response should contain a custom "x1:very-custom-prop" property with namespace "x1='http://whatever.org/ns'" and value "veryCustomPropValue"
    And the content in the response should include the following content:
      """
      <d:prop><x1:very-custom-prop xmlns:x1="http://whatever.org/ns">veryCustomPropValue</x1:very-custom-prop></d:prop>
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
