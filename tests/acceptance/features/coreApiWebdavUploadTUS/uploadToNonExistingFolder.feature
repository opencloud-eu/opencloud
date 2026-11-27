Feature: upload file
  As a user
  I want to try uploading files to a nonexistent folder
  So that I can check if the uploading works in such case

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes


  Scenario Outline: attempt to upload a file into a nonexistent shares
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "/Shares/FOLDER/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Alice" folder "/Shares/FOLDER/" should not exist
    And as "Alice" file "/Shares/FOLDER/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |

    @skipOnReva
    Examples:
      | dav-path-version |
      | spaces           |


  Scenario Outline: attempt to upload a file into a nonexistent folder
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file with content "uploaded content" to "/nonExistentFolder/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Alice" folder "/nonExistentFolder" should not exist
    And as "Alice" file "/nonExistentFolder/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |

    @skipOnReva
    Examples:
      | dav-path-version |
      | spaces           |

  @skipOnReva
  Scenario Outline: attempt to upload a file into a nonexistent folder within correctly received share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    When user "Brian" uploads file with content "uploaded content" to "/Shares/FOLDER/nonExistentFolder/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Brian" folder "/Shares/FOLDER/nonExistentFolder" should not exist
    And as "Brian" file "/Shares/FOLDER/nonExistentFolder/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: attempt to upload a file into a nonexistent folder within correctly received read only share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "/FOLDER"
    And user "Alice" has sent the following resource share invitation:
      | resource        | FOLDER   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Viewer   |
    When user "Brian" uploads file with content "uploaded content" to "/Shares/FOLDER/nonExistentFolder/textfile.txt" using the TUS protocol on the WebDAV API
    Then as "Brian" folder "/Shares/FOLDER/nonExistentFolder" should not exist
    And as "Brian" file "/Shares/FOLDER/nonExistentFolder/textfile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
