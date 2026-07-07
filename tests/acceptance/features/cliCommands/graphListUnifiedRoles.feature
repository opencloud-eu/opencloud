Feature: list unified roles via CLI command
  As an admin
  I want to list the available unified roles
  So that I know their IDs when configuring GRAPH_AVAILABLE_ROLES or writing documentation


  Scenario: list unified roles
    When the administrator lists the unified roles using the CLI
    Then the command should be successful
    And the command output should contain "Viewer"
    And the command output should contain "Editor"
    And the command output should contain "SpaceManager"
    And the command output should contain "SecureViewer"
