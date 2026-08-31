@tikaServiceNeeded
Feature: title search
  As a user
  I want to find a document by the title in its metadata
  So that I find it even when its name says nothing about it


  Background:
    Given user "Alice" has been created with default attributes
    And using spaces DAV path


  Scenario: search a document by the title the extractor read from it
    Given user "Alice" has uploaded file with content "<html><head><title>quarterly report</title></head><body>some data</body></html>" to "q1.html"
    And user "Alice" has uploaded file with content "<html><head><title>notes</title></head><body>some data</body></html>" to "q2.html"
    When user "Alice" searches for 'Title:"quarterly report"' using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | q1.html |
