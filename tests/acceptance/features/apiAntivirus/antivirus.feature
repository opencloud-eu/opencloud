@antivirus @skipOnReva
Feature: antivirus
  As a system administrator and user
  I want to protect myself and others from known viruses
  So that I can prevent files with viruses from being uploaded

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: upload a normal file without virus
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/textfile.txt" to "/normalfile.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/normalfile.txt" should exist
    And the content of file "/normalfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: upload a file with virus
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/filesWithVirus/<file-name>" to "<new-file-name>" using the WebDAV API
    # antivirus service can scan files during post-processing. on demand scanning is currently not available
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    And as "Alice" file "<new-file-name>" should not exist
    Examples:
      | dav-path-version | file-name     | new-file-name  | message                                                    |
      | old              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | old              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | new              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | new              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | spaces           | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | spaces           | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |


  Scenario Outline: upload a file with virus and a file without virus
    Given using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/filesWithVirus/<file-name>" to "<new-file-name>" using the WebDAV API
    # antivirus service can scan files during post-processing. on demand scanning is currently not available
    Then the HTTP status code should be "201"
    And user "Alice" uploads file "filesForUpload/textfile.txt" to "/normalfile.txt" using the WebDAV API
    And the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    And as "Alice" file "<new-file-name>" should not exist
    But as "Alice" file "/normalfile.txt" should exist
    And the content of file "/normalfile.txt" for user "Alice" should be:
      """
      This is a testfile.

      Cheers.
      """
    Examples:
      | dav-path-version | file-name     | new-file-name  | message                                                    |
      | old              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | old              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | new              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | new              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | spaces           | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | spaces           | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |


  Scenario Outline: upload a file with virus in chunks
    Given using <dav-path-version> DAV path
    When user "Alice" uploads the following chunks to "/myChunkedFile.txt" with old chunking and using the WebDAV API
      | number | content                 |
      | 1      | X5O!P%@AP[4\PZX54(P^)7C |
      | 2      | C)7}$EICAR-STANDARD-ANT |
      | 3      | IVIRUS-TEST-FILE!$H+H*  |
    # antivirus service can scan files during post-processing. on demand scanning is currently not available
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message                                                       |
      | Virus found in myChunkedFile.txt. Upload not possible. Virus: |
    And as "Alice" file "/myChunkedFile.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-10331 @env-config
  Scenario Outline: public uploads a file with the virus to a public share
    Given using <dav-path-version> DAV path
    And the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using SharingNG
    And user "Alice" has created folder "/uploadFolder"
    And user "Alice" has created the following resource link share:
      | resource           | uploadFolder             |
      | space              | Personal                 |
      | permissionsRole    | edit                     |
      | expirationDateTime | 2040-01-01T23:59:59.000Z |
    When the public uploads file "filesForUpload/filesWithVirus/<file-name>" to "<new-file-name>" inside last link shared folder using the public WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    And as "Alice" file "/uploadFolder/<new-file-name>" should not exist
    Examples:
      | dav-path-version | file-name     | new-file-name  | message                                                    |
      | old              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | old              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | new              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | new              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | spaces           | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | spaces           | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |

  @issue-10331
  Scenario Outline: public uploads a file with the virus to a password-protected public share
    Given using <dav-path-version> DAV path
    And using SharingNG
    And user "Alice" has created folder "/uploadFolder"
    And user "Alice" has created the following resource link share:
      | resource           | uploadFolder             |
      | space              | Personal                 |
      | permissionsRole    | edit                     |
      | password           | %public%                 |
      | expirationDateTime | 2040-01-01T23:59:59.000Z |
    When the public uploads file "filesForUpload/filesWithVirus/<file-name>" to "<new-file-name>" inside last link shared folder with password "%public%" using the public WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    And as "Alice" file "/uploadFolder/<new-file-name>" should not exist
    Examples:
      | dav-path-version | file-name     | new-file-name  | message                                                    |
      | old              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | old              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | new              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | new              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | spaces           | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | spaces           | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |


  Scenario Outline: upload a file with virus to a user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Brian" has a share "uploadFolder" synced
    When user "Brian" uploads file "filesForUpload/filesWithVirus/<file-name>" to "/Shares/uploadFolder/<new-file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Brian" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    And as "Brian" file "/Shares/uploadFolder/<new-file-name>" should not exist
    And as "Alice" file "/uploadFolder/<new-file-name>" should not exist
    Examples:
      | dav-path-version | file-name     | new-file-name  | message                                                    |
      | old              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | old              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | new              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | new              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | spaces           | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | spaces           | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |


  Scenario Outline: upload a file with virus to a group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | group1       |
      | shareType       | group        |
      | permissionsRole | Editor       |
    And user "Brian" has a share "uploadFolder" synced
    When user "Brian" uploads file "filesForUpload/filesWithVirus/<file-name>" to "/Shares/uploadFolder/<new-file-name>" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Brian" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    When the user waits for "10" seconds for postprocessing to finish
    And as "Brian" file "/Shares/uploadFolder/<new-file-name>" should not exist
    And as "Alice" file "/uploadFolder/<new-file-name>" should not exist
    Examples:
      | dav-path-version | file-name     | new-file-name  | message                                                    |
      | old              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | old              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | new              | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | new              | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |
      | spaces           | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | spaces           | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |


  Scenario Outline: upload a file with virus to a project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder "uploadFolder" in space "new-space"
    When user "Alice" uploads a file "filesForUpload/filesWithVirus/<file-name>" to "/uploadFolder/<new-file-name>" in space "new-space" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification for resource "<new-file-name>" with subject "Virus found" and message:
      | message   |
      | <message> |
    And for user "Alice" folder "uploadFolder" of the space "new-space" should not contain these entries:
      | <new-file-name> |
    When user "Alice" uploads a file "filesForUpload/filesWithVirus/<file-name>" to "/<new-file-name>" in space "new-space" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification for resource "<new-file-name>" with subject "Virus found" and message:
      | message   |
      | <message> |
    And for user "Alice" the space "new-space" should not contain these entries:
      | /<new-file-name> |
    Examples:
      | file-name     | new-file-name  | message                                                    |
      | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |


  Scenario Outline: upload a file with virus to a shared project space
    Given using spaces DAV path
    And user "Brian" has been created with default attributes
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    When user "Brian" uploads a file "/filesForUpload/filesWithVirus/<file-name>" to "/<new-file-name>" in space "new-space" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Brian" should get a notification with subject "Virus found" and message:
      | message   |
      | <message> |
    And for user "Brian" the space "new-space" should not contain these entries:
      | /<new-file-name> |
    And for user "Alice" the space "new-space" should not contain these entries:
      | /<new-file-name> |
    Examples:
      | file-name     | new-file-name  | message                                                    |
      | eicar.com     | virusFile1.txt | Virus found in virusFile1.txt. Upload not possible. Virus: |
      | eicar_com.zip | virusFile2.zip | Virus found in virusFile2.zip. Upload not possible. Virus: |

  @env-config  @issue-6494
  Scenario Outline: upload a file with virus by setting antivirus infected file handling config to continue
    Given the config "ANTIVIRUS_INFECTED_FILE_HANDLING" has been set to "continue"
    And using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/filesWithVirus/eicar.com" to "/aFileWithVirus.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/aFileWithVirus.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @env-config
  Scenario Outline: upload a file with virus smaller than the upload threshold
    Given the config "ANTIVIRUS_MAX_SCAN_SIZE" has been set to "100"
    And using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/filesWithVirus/eicar.com" to "/aFileWithVirus.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message                                                        |
      | Virus found in aFileWithVirus.txt. Upload not possible. Virus: |
    And as "Alice" file "/aFileWithVirus.txt" should not exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @env-config
  Scenario Outline: upload a file with virus larger than the upload threshold
    Given the config "ANTIVIRUS_MAX_SCAN_SIZE" has been set to "100"
    And using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/filesWithVirus/eicar_com.zip" to "/aFileWithVirus.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And as "Alice" file "/aFileWithVirus.txt" should exist
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @issue-enterprise-5706 @issue-183 @issue-369
  Scenario Outline: upload a file with virus and get notification in different languages
    Given user "Alice" has switched the system language to "<language>" using the Graph API
    And using <dav-path-version> DAV path
    When user "Alice" uploads file "filesForUpload/filesWithVirus/eicar.com" to "/aFileWithVirus.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And user "Alice" should get a notification with subject "<subject>" and message:
      | message   |
      | <message> |
    And as "Alice" file "/aFileWithVirus.txt" should not exist
    Examples:
      | dav-path-version | language | subject          | message                                                                                                                        |
      | old              | es       | Virus encontrado | Virus encontrado en aFileWithVirus.txt. La subida no ha sido posible. Virus:                              |
      | new              | es       | Virus encontrado | Virus encontrado en aFileWithVirus.txt. La subida no ha sido posible. Virus:                              |
      | spaces           | es       | Virus encontrado | Virus encontrado en aFileWithVirus.txt. La subida no ha sido posible. Virus:                              |
      | old              | de       | Virus gefunden   | In aFileWithVirus.txt wurde potenziell schädlicher Code gefunden. Das Hochladen wurde abgebrochen. Grund: |
      | new              | de       | Virus gefunden   | In aFileWithVirus.txt wurde potenziell schädlicher Code gefunden. Das Hochladen wurde abgebrochen. Grund: |
      | spaces           | de       | Virus gefunden   | In aFileWithVirus.txt wurde potenziell schädlicher Code gefunden. Das Hochladen wurde abgebrochen. Grund: |

  @issue-enterprise-5709
  Scenario Outline: try to create a version of file by uploading virus content
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "hello world" to "test.txt"
    And user "Alice" has uploaded file with content "hello nepal" to "test.txt"
    When user "Alice" uploads file with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" to "test.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message                                              |
      | Virus found in test.txt. Upload not possible. Virus: |
    And as "Alice" file "/test.txt" should exist
    And the version folder of file "/test.txt" for user "Alice" should contain "1" element
    And the content of file "/test.txt" for user "Alice" should be "hello nepal"
    When user "Alice" restores version index "1" of file "/test.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/test.txt" for user "Alice" should be "hello world"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @env-config @issue-10331
  Scenario Outline: try to overwrite a file with the virus content in a public link share
    Given the config "OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" has been set to "false"
    And using <dav-path-version> DAV path
    And using SharingNG
    And user "Alice" has uploaded file with content "hello" to "test.txt"
    And user "Alice" has created the following resource link share:
      | resource        | test.txt |
      | space           | Personal |
      | permissionsRole | edit     |
    When the public overwrites file "test.txt" with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" using the public WebDAV API
    Then the HTTP status code should be "204"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message                                              |
      | Virus found in test.txt. Upload not possible. Virus: |
    And the content of file "/test.txt" for user "Alice" should be "hello"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: try to overwrite a file with the virus content in group share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "group1" has been created
    And user "Brian" has been added to group "group1"
    And user "Alice" has been added to group "group1"
    And user "Alice" has uploaded file with content "hello" to "/test.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | test.txt    |
      | space           | Personal    |
      | sharee          | group1      |
      | shareType       | group       |
      | permissionsRole | File Editor |
    And user "Brian" has a share "test.txt" synced
    When user "Brian" uploads file with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" to "Shares/test.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Brian" should get a notification with subject "Virus found" and message:
      | message                                              |
      | Virus found in test.txt. Upload not possible. Virus: |
    And the content of file "/test.txt" for user "Alice" should be "hello"
    And the content of file "Shares/test.txt" for user "Brian" should be "hello"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: try to overwrite a file with the virus content in user share
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "uploadFolder"
    And user "Alice" has uploaded file with content "this is a test file." to "uploadFolder/test.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | uploadFolder |
      | space           | Personal     |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Editor       |
    And user "Brian" has a share "uploadFolder" synced
    And user "Alice" has uploaded file with content "this is a test file." to "/test.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | test.txt    |
      | space           | Personal    |
      | sharee          | Brian       |
      | shareType       | user        |
      | permissionsRole | File Editor |
    And user "Brian" has a share "test.txt" synced
    When user "Brian" uploads file with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" to "Shares/uploadFolder/test.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Brian" should get a notification for resource "test.txt" with subject "Virus found" and message:
      | message                                              |
      | Virus found in test.txt. Upload not possible. Virus: |
    And the content of file "Shares/uploadFolder/test.txt" for user "Brian" should be "this is a test file."
    And the content of file "uploadFolder/test.txt" for user "Alice" should be "this is a test file."
    When user "Brian" uploads file with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" to "Shares/test.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Brian" should get a notification for resource "test.txt" with subject "Virus found" and message:
      | message                                              |
      | Virus found in test.txt. Upload not possible. Virus: |
    And the content of file "Shares/test.txt" for user "Brian" should be "this is a test file."
    And the content of file "/test.txt" for user "Alice" should be "this is a test file."
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario: try to overwrite the .space/readme.md file
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has created a folder ".space" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "Here you can add a description for this Space." to ".space/readme.md"
    When user "Alice" uploads a file inside space "new-space" with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" to ".space/readme.md" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Alice" should get a notification with subject "Virus found" and message:
      | message                                               |
      | Virus found in readme.md. Upload not possible. Virus: |
    And for user "Alice" the content of the file ".space/readme.md" of the space "new-space" should be "Here you can add a description for this Space."


  Scenario: try to overwrite the .space/readme.md file in space share
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    And user "Alice" has created a folder ".space" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "Here you can add a description for this Space." to ".space/readme.md"
    And user "Alice" has set the file ".space/readme.md" as a description in a special section of the "new-space" space
    When user "Brian" uploads a file inside space "new-space" with content "X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" to ".space/readme.md" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Brian" should get a notification with subject "Virus found" and message:
      | message                                               |
      | Virus found in readme.md. Upload not possible. Virus: |
    And for user "Brian" the content of the file ".space/readme.md" of the space "new-space" should be "Here you can add a description for this Space."
    And for user "Alice" the content of the file ".space/readme.md" of the space "new-space" should be "Here you can add a description for this Space."


  Scenario: member of a project space tries to overwrite a file with virus content
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has sent the following space share invitation:
      | space           | new-space    |
      | sharee          | Brian        |
      | shareType       | user         |
      | permissionsRole | Space Editor |
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Brian" uploads a file "filesForUpload/filesWithVirus/eicar.com" to "text.txt" in space "new-space" using the WebDAV API
    Then the HTTP status code should be "204"
    And user "Brian" should get a notification with subject "Virus found" and message:
      | message                                              |
      | Virus found in text.txt. Upload not possible. Virus: |
    And for user "Brian" the content of the file "/text.txt" of the space "new-space" should be "hello world"
    And for user "Alice" the content of the file "/text.txt" of the space "new-space" should be "hello world"
