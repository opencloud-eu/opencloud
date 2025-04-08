Feature: Status
  As a admin
  I want to check status of the server
  So that I can ensure that the server is working

  @smokeTest
  Scenario: Status.php is correct
    When the administrator requests status.php
    Then the status.php response should include
      """
      {"installed":true,"maintenance":false,"needsDbUpgrade":false,"version":"$CURRENT_VERSION","versionstring":"$CURRENT_VERSION_STRING","edition":"$EDITION","productname":"$PRODUCTNAME","product":"$PRODUCT"}
      """
