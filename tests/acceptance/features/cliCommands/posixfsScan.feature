@env-config @skipOnOpencloud-decomposed-Storage @skipOnOpencloud-decomposeds3-Storage
Feature: posixfs scan CLI command
  As an administrator
  I want to scan the posix filesystem for files that were added outside of OpenCloud
  So that their metadata (ID cache and file metadata) gets assimilated and they become usable

  # WATCH_FS=false so only the scan command assimilates the files; proof read via getfattr from disk
  Background:
    Given the config "STORAGE_USERS_POSIX_WATCH_FS" has been set to "false"
    And user "Alice" has been created with default attributes


  Scenario: scan the whole storage assimilates files added directly to the personal and project spaces
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace1" with the default quota using the Graph API
    And user "Alice" has created a space "projectSpace2" with the default quota using the Graph API
    And the administrator has created the file "scanned.txt" with content "personal" for user "Alice" on the POSIX filesystem
    And the administrator creates the file "inProject1.txt" with content "project one" in the space "projectSpace1" on the POSIX filesystem
    And the administrator creates the file "inProject2.txt" with content "project two" in the space "projectSpace2" on the POSIX filesystem
    When the administrator gets the extended attributes of file "scanned.txt" of user "Alice" on the POSIX filesystem
    Then the command output should not contain "user.oc.id"
    And the administrator gets the extended attributes of file "inProject1.txt" in the space "projectSpace1" on the POSIX filesystem
    And the command output should not contain "user.oc.id"
    And the administrator gets the extended attributes of file "inProject2.txt" in the space "projectSpace2" on the POSIX filesystem
    And the command output should not contain "user.oc.id"
    And the administrator scans the whole storage using the CLI
    And the command should be successful
    And the command output should contain "Scan completed successfully."
    And the administrator gets the extended attributes of file "scanned.txt" of user "Alice" on the POSIX filesystem
    And the command output should contain "user.oc.id"
    And the administrator gets the extended attributes of file "inProject1.txt" in the space "projectSpace1" on the POSIX filesystem
    And the command output should contain "user.oc.id"
    And the administrator gets the extended attributes of file "inProject2.txt" in the space "projectSpace2" on the POSIX filesystem
    And the command output should contain "user.oc.id"
    And as "Alice" the final content of file "scanned.txt" should be "personal"
    And using spaces DAV path
    And for user "Alice" the space "projectSpace1" should contain these entries:
      | inProject1.txt |
    And for user "Alice" the space "projectSpace2" should contain these entries:
      | inProject2.txt |


  Scenario: scan a single folder assimilates the files inside it
    Given the administrator creates the folder "scanFolder" for user "Alice" on the POSIX filesystem
    And the administrator has created the file "scanFolder/inside.txt" with content "inside content" for user "Alice" on the POSIX filesystem
    When the administrator gets the extended attributes of file "scanFolder/inside.txt" of user "Alice" on the POSIX filesystem
    Then the command output should not contain "user.oc.id"
    And the administrator scans the folder "scanFolder" of user "Alice" using the CLI
    And the command should be successful
    And the command output should contain "Scan completed successfully."
    And the administrator gets the extended attributes of file "scanFolder/inside.txt" of user "Alice" on the POSIX filesystem
    And the command output should contain "user.oc.id"
    And as "Alice" the final content of file "scanFolder/inside.txt" should be "inside content"


  Scenario: scan a single project space assimilates only that space
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "projectSpace1" with the default quota using the Graph API
    And user "Alice" has created a space "projectSpace2" with the default quota using the Graph API
    And the administrator creates the file "inProject1.txt" with content "project one" in the space "projectSpace1" on the POSIX filesystem
    And the administrator creates the file "inProject2.txt" with content "project two" in the space "projectSpace2" on the POSIX filesystem
    And the administrator scans the space "projectSpace1" using the CLI
    And the command should be successful
    And the command output should contain "Scan completed successfully."
    And the administrator gets the extended attributes of file "inProject1.txt" in the space "projectSpace1" on the POSIX filesystem
    And the command output should contain "user.oc.id"
    And the administrator gets the extended attributes of file "inProject2.txt" in the space "projectSpace2" on the POSIX filesystem
    And the command output should not contain "user.oc.id"
    And using spaces DAV path
    And for user "Alice" the space "projectSpace1" should contain these entries:
      | inProject1.txt |


  Scenario: scan a single regular file assimilates only that file
    Given the administrator has created the file "singleFile.txt" with content "single" for user "Alice" on the POSIX filesystem
    And the administrator has created the file "otherFile.txt" with content "other" for user "Alice" on the POSIX filesystem
    When the administrator scans the file "singleFile.txt" of user "Alice" using the CLI
    Then the command should be successful
    And the command output should contain "Scan completed successfully."
    And the administrator gets the extended attributes of file "singleFile.txt" of user "Alice" on the POSIX filesystem
    And the command output should contain "user.oc.id"
    And the administrator gets the extended attributes of file "otherFile.txt" of user "Alice" on the POSIX filesystem
    And the command output should not contain "user.oc.id"
    And as "Alice" the final content of file "singleFile.txt" should be "single"


  Scenario: scanning a path outside the storage root fails
    When the administrator scans path "/tmp" using the CLI
    Then the command should not be successful
    And the command output should contain "does not appear to be inside a posixfs storage"


  Scenario Outline: the halt-on-error flag controls whether the scan continues after an error
    Given the administrator creates the folder "validFolder" for user "Alice" on the POSIX filesystem
    And the administrator has created the file "validFolder/inside.txt" with content "keep going" for user "Alice" on the POSIX filesystem
    And the administrator scans a non-existing path and the folder "validFolder" of user "Alice" using the CLI with flag "<flag>"
    And the command should not be successful
    And the command output should contain "<message>"
    And the administrator gets the extended attributes of file "validFolder/inside.txt" of user "Alice" on the POSIX filesystem
    And the command output <shouldOrNot> contain "user.oc.id"
    Examples:
      | flag | message                     | shouldOrNot |
      |      | scan completed with 1 error | should      |
      | -E   | scan aborted with 1 error   | should not  |
