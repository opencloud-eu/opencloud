Feature: get robots.txt
  As a user
  I want to get file robots.txt
  So that I can check its content

  @owncloud-issue-1314
  Scenario: robots.txt file should be accessible
    When a user requests "/robots.txt" with "GET" and no authentication
    Then the HTTP status code should be "200"
    And the content in the response should match the following content:
      """
      User-agent: *
      Disallow: /

      """
