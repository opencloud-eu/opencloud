Feature: check file info with different wopi apps
  As a user
  I want to request file information on wopi server
  So that I can get all the information of a file

  Background:
    Given user "Alice" has been created with default attributes


  Scenario: check file info with fakeOffice
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" checks the information of file "textfile0.txt" of space "Personal" using office "FakeOffice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "OwnerId",
          "Size",
          "UserId",
          "Version",
          "SupportsCobalt",
          "SupportsContainers",
          "SupportsDeleteFile",
          "SupportsEcosystem",
          "SupportsExtendedLockLength",
          "SupportsFolders",
          "SupportsGetLock",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "SupportsUserInfo",
          "UserFriendlyName",
          "ReadOnly",
          "RestrictedWebViewOnly",
          "UserCanAttend",
          "UserCanNotWriteRelative",
          "UserCanPresent",
          "UserCanRename",
          "UserCanWrite",
          "AllowAdditionalMicrosoftServices",
          "AllowExternalMarketplace",
          "DisablePrint",
          "DisableTranslation",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsCobalt": {
            "const": false
          },
          "SupportsContainers": {
            "const": false
          },
          "SupportsDeleteFile": {
            "const": true
          },
          "SupportsEcosystem": {
            "const": false
          },
          "SupportsExtendedLockLength": {
            "const": true
          },
          "SupportsFolders": {
            "const": false
          },
          "SupportsGetLock": {
            "const": true
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "SupportsUserInfo": {
            "const": false
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "RestrictedWebViewOnly": {
            "const": false
          },
          "UserCanAttend": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanPresent": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "AllowAdditionalMicrosoftServices": {
            "const": false
          },
          "AllowExternalMarketplace": {
            "const": false
          },
          "DisablePrint": {
            "const": false
          },
          "DisableTranslation": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          }
        }
      }
      """


  Scenario: check file info with collabora
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" checks the information of file "textfile0.txt" of space "Personal" using office "Collabora"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "DisablePrint",
          "OwnerId",
          "PostMessageOrigin",
          "Size",
          "UserCanWrite",
          "UserCanNotWriteRelative",
          "UserId",
          "UserFriendlyName",
          "EnableOwnerTermination",
          "SupportsLocks",
          "SupportsRename",
          "UserCanRename",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "PostMessageOrigin": {
            "const": "https://localhost:9200"
          },
          "DisablePrint": {
            "const": false
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserCanWrite": {
            "const": true
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "EnableOwnerTermination": {
            "const": true
          },
          "UserId": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "UserCanRename": {
            "const": true
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          }
        }
      }
      """


  Scenario: check file info with onlyOffice
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" checks the information of file "textfile0.txt" of space "Personal" using office "OnlyOffice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "Version",
          "BreadcrumbDocName",
          "BreadcrumbFolderName",
          "BreadcrumbFolderUrl",
          "PostMessageOrigin",
          "DisablePrint",
          "UserFriendlyName",
          "UserId",
          "ReadOnly",
          "UserCanNotWriteRelative",
          "UserCanRename",
          "UserCanWrite",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "HostEditUrl",
          "FileSharingUrl",
          "FileVersionUrl"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "DisablePrint": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          },
          "BreadcrumbFolderName": {
            "const": "Alice Hansen"
          },
          "BreadcrumbFolderUrl": {
            "type": "string"
          },
          "PostMessageOrigin": {
            "type": "string"
          },
          "FileSharingUrl": {
            "type": "string"
          },
          "HostEditUrl": {
            "type": "string"
          },
          "FileVersionUrl": {
            "type": "string"
          }
        }
      }
      """


  Scenario Outline: check file info with different mode (onlyOffice)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" checks the information of file "textfile0.txt" of space "Personal" using office "OnlyOffice" with view mode "<mode>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "Version",
          "BreadcrumbDocName",
          "BreadcrumbFolderName",
          "BreadcrumbFolderUrl",
          "PostMessageOrigin",
          "DisablePrint",
          "UserFriendlyName",
          "UserId",
          "ReadOnly",
          "UserCanNotWriteRelative",
          "UserCanRename",
          "UserCanWrite",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "HostEditUrl",
          "FileSharingUrl",
          "FileVersionUrl"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanRename": {
            "const": <user-can-rename>
          },
          "UserCanWrite": {
            "const": <user-can-write>
          },
          "DisablePrint": {
            "const": <disable-print>
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          },
          "BreadcrumbFolderName": {
            "const": "Alice Hansen"
          },
          "BreadcrumbFolderUrl": {
            "type": "string"
          },
          "PostMessageOrigin": {
            "type": "string"
          },
          "FileSharingUrl": {
            "type": "string"
          },
          "HostEditUrl": {
            "type": "string"
          },
          "FileVersionUrl": {
            "type": "string"
          }
        }
      }
      """
    Examples:
      | mode  | disable-print | user-can-write | user-can-rename |
      | view  | true          | false          | false           |
      | read  | false         | false          | false           |
      | write | false         | true           | true            |


  Scenario Outline: check file info with different mode (fakeOffice)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" checks the information of file "textfile0.txt" of space "Personal" using office "FakeOffice" with view mode "<mode>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "OwnerId",
          "Size",
          "UserId",
          "Version",
          "SupportsCobalt",
          "SupportsContainers",
          "SupportsDeleteFile",
          "SupportsEcosystem",
          "SupportsExtendedLockLength",
          "SupportsFolders",
          "SupportsGetLock",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "SupportsUserInfo",
          "UserFriendlyName",
          "ReadOnly",
          "RestrictedWebViewOnly",
          "UserCanAttend",
          "UserCanNotWriteRelative",
          "UserCanPresent",
          "UserCanRename",
          "UserCanWrite",
          "AllowAdditionalMicrosoftServices",
          "AllowExternalMarketplace",
          "DisablePrint",
          "DisableTranslation",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsCobalt": {
            "const": false
          },
          "SupportsContainers": {
            "const": false
          },
          "SupportsDeleteFile": {
            "const": true
          },
          "SupportsEcosystem": {
            "const": false
          },
          "SupportsExtendedLockLength": {
            "const": true
          },
          "SupportsFolders": {
            "const": false
          },
          "SupportsGetLock": {
            "const": true
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "SupportsUserInfo": {
            "const": false
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "RestrictedWebViewOnly": {
            "const": false
          },
          "UserCanAttend": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanPresent": {
            "const": false
          },
          "UserCanRename": {
            "const": <user-can-rename>
          },
          "UserCanWrite": {
            "const": <user-can-write>
          },
          "AllowAdditionalMicrosoftServices": {
            "const": false
          },
          "AllowExternalMarketplace": {
            "const": false
          },
          "DisablePrint": {
            "const": <disable-print>
          },
          "DisableTranslation": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          }
        }
      }
      """
    Examples:
      | mode  | disable-print | user-can-write | user-can-rename |
      | view  | true          | false          | false           |
      | read  | false         | false          | false           |
      | write | false         | true           | true            |


  Scenario Outline: check file info with different view-mode (collabora)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" checks the information of file "textfile0.txt" of space "Personal" using office "Collabora" with view mode "<mode>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "DisablePrint",
          "OwnerId",
          "PostMessageOrigin",
          "Size",
          "UserCanWrite",
          "UserCanNotWriteRelative",
          "UserId",
          "UserFriendlyName",
          "EnableOwnerTermination",
          "SupportsLocks",
          "SupportsRename",
          "UserCanRename",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "PostMessageOrigin": {
            "const": "https://localhost:9200"
          },
          "DisablePrint": {
            "const": <disable-print>
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserCanWrite": {
            "const": <user-can-write>
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "EnableOwnerTermination": {
            "const": true
          },
          "UserId": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "UserCanRename": {
            "const": <user-can-rename>
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          }
        }
      }
      """
    Examples:
      | mode  | disable-print | user-can-write | user-can-rename |
      | view  | true          | false          | false           |
      | read  | false         | false          | false           |
      | write | false         | true           | true            |


  Scenario Outline: try to get file info using invalid file id with office suites
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    When user "Alice" tries to check the information of file "textfile0.txt" of space "Personal" using office "<office-suites>" with invalid file-id
    Then the HTTP status code should be "401"
    Examples:
      | office-suites |
      | Collabora     |
      | FakeOffice    |
      | OnlyOffice    |

  @issue-10097
  Scenario Outline: try to get file info of deleted file with office suites
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt   |
      | space    | Personal        |
      | app      | <office-suites> |
    And user "Alice" has deleted file "/textfile0.txt"
    When user "Alice" tries to get the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "404"
    Examples:
      | office-suites |
      | Collabora     |
      | FakeOffice    |
      | OnlyOffice    |


  Scenario: get file info of restored file from trashbin (collabora)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt |
      | space    | Personal      |
      | app      | Collabora     |
    And user "Alice" has deleted file "/textfile0.txt"
    And user "Alice" has restored the file with original path "/textfile0.txt"
    When user "Alice" gets the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "DisablePrint",
          "OwnerId",
          "PostMessageOrigin",
          "Size",
          "UserCanWrite",
          "UserCanNotWriteRelative",
          "UserId",
          "UserFriendlyName",
          "EnableOwnerTermination",
          "SupportsLocks",
          "SupportsRename",
          "UserCanRename",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "PostMessageOrigin": {
            "const": "https://localhost:9200"
          },
          "DisablePrint": {
            "const": false
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserCanWrite": {
            "const": true
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "EnableOwnerTermination": {
            "const": true
          },
          "UserId": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "UserCanRename": {
            "const": true
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          }
        }
      }
      """


  Scenario: get file info of restored file from trashbin (fakeOffice)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt |
      | space    | Personal      |
      | app      | FakeOffice    |
    And user "Alice" has deleted file "/textfile0.txt"
    And user "Alice" has restored the file with original path "/textfile0.txt"
    When user "Alice" gets the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "OwnerId",
          "Size",
          "UserId",
          "Version",
          "SupportsCobalt",
          "SupportsContainers",
          "SupportsDeleteFile",
          "SupportsEcosystem",
          "SupportsExtendedLockLength",
          "SupportsFolders",
          "SupportsGetLock",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "SupportsUserInfo",
          "UserFriendlyName",
          "ReadOnly",
          "RestrictedWebViewOnly",
          "UserCanAttend",
          "UserCanNotWriteRelative",
          "UserCanPresent",
          "UserCanRename",
          "UserCanWrite",
          "AllowAdditionalMicrosoftServices",
          "AllowExternalMarketplace",
          "DisablePrint",
          "DisableTranslation",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsCobalt": {
            "const": false
          },
          "SupportsContainers": {
            "const": false
          },
          "SupportsDeleteFile": {
            "const": true
          },
          "SupportsEcosystem": {
            "const": false
          },
          "SupportsExtendedLockLength": {
            "const": true
          },
          "SupportsFolders": {
            "const": false
          },
          "SupportsGetLock": {
            "const": true
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "SupportsUserInfo": {
            "const": false
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "RestrictedWebViewOnly": {
            "const": false
          },
          "UserCanAttend": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanPresent": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "AllowAdditionalMicrosoftServices": {
            "const": false
          },
          "AllowExternalMarketplace": {
            "const": false
          },
          "DisablePrint": {
            "const": false
          },
          "DisableTranslation": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          }
        }
      }
      """


  Scenario: get file info of restored file from trashbin (onlyOffice)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt |
      | space    | Personal      |
      | app      | OnlyOffice    |
    And user "Alice" has deleted file "/textfile0.txt"
    And user "Alice" has restored the file with original path "/textfile0.txt"
    When user "Alice" gets the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "Version",
          "BreadcrumbDocName",
          "BreadcrumbFolderName",
          "BreadcrumbFolderUrl",
          "PostMessageOrigin",
          "DisablePrint",
          "UserFriendlyName",
          "UserId",
          "ReadOnly",
          "UserCanNotWriteRelative",
          "UserCanRename",
          "UserCanWrite",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate"
        ],
        "properties": {
          "BaseFileName": {
            "const": "textfile0.txt"
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "DisablePrint": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "textfile0.txt"
          },
          "BreadcrumbFolderName": {
            "const": "Alice Hansen"
          },
          "BreadcrumbFolderUrl": {
            "type": "string"
          },
          "PostMessageOrigin": {
            "type": "string"
          }
        }
      }
      """


  Scenario: get file info after renaming file (onlyOffice)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt |
      | space    | Personal      |
      | app      | OnlyOffice    |
    And user "Alice" has moved file "textfile0.txt" to "renamedfile.txt"
    When user "Alice" gets the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "Version",
          "BreadcrumbDocName",
          "BreadcrumbFolderName",
          "BreadcrumbFolderUrl",
          "PostMessageOrigin",
          "DisablePrint",
          "UserFriendlyName",
          "UserId",
          "ReadOnly",
          "UserCanNotWriteRelative",
          "UserCanRename",
          "UserCanWrite",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate"
        ],
        "properties": {
          "BaseFileName": {
            "const": "renamedfile.txt"
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "DisablePrint": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "renamedfile.txt"
          },
          "BreadcrumbFolderName": {
            "const": "Alice Hansen"
          },
          "BreadcrumbFolderUrl": {
            "type": "string"
          },
          "PostMessageOrigin": {
            "type": "string"
          }
        }
      }
      """


  Scenario: get file info after renaming file (collabora)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt |
      | space    | Personal      |
      | app      | Collabora     |
    And user "Alice" has moved file "textfile0.txt" to "renamedfile.txt"
    When user "Alice" gets the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "DisablePrint",
          "OwnerId",
          "PostMessageOrigin",
          "Size",
          "UserCanWrite",
          "UserCanNotWriteRelative",
          "UserId",
          "UserFriendlyName",
          "EnableOwnerTermination",
          "SupportsLocks",
          "SupportsRename",
          "UserCanRename",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "renamedfile.txt"
          },
          "PostMessageOrigin": {
            "const": "https://localhost:9200"
          },
          "DisablePrint": {
            "const": false
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserCanWrite": {
            "const": true
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "EnableOwnerTermination": {
            "const": true
          },
          "UserId": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "UserCanRename": {
            "const": true
          },
          "BreadcrumbDocName": {
            "const": "renamedfile.txt"
          }
        }
      }
      """


  Scenario: get file info after renaming file with (fakeOffice)
    Given user "Alice" has uploaded file with content "hello world" to "/textfile0.txt"
    And user "Alice" has sent the following app-open request:
      | resource | textfile0.txt |
      | space    | Personal      |
      | app      | FakeOffice    |
    And user "Alice" has moved file "textfile0.txt" to "renamedfile.txt"
    When user "Alice" gets the information of the last opened file using wopi endpoint
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "OwnerId",
          "Size",
          "UserId",
          "Version",
          "SupportsCobalt",
          "SupportsContainers",
          "SupportsDeleteFile",
          "SupportsEcosystem",
          "SupportsExtendedLockLength",
          "SupportsFolders",
          "SupportsGetLock",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "SupportsUserInfo",
          "UserFriendlyName",
          "ReadOnly",
          "RestrictedWebViewOnly",
          "UserCanAttend",
          "UserCanNotWriteRelative",
          "UserCanPresent",
          "UserCanRename",
          "UserCanWrite",
          "AllowAdditionalMicrosoftServices",
          "AllowExternalMarketplace",
          "DisablePrint",
          "DisableTranslation",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "renamedfile.txt"
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsCobalt": {
            "const": false
          },
          "SupportsContainers": {
            "const": false
          },
          "SupportsDeleteFile": {
            "const": true
          },
          "SupportsEcosystem": {
            "const": false
          },
          "SupportsExtendedLockLength": {
            "const": true
          },
          "SupportsFolders": {
            "const": false
          },
          "SupportsGetLock": {
            "const": true
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "SupportsUserInfo": {
            "const": false
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "RestrictedWebViewOnly": {
            "const": false
          },
          "UserCanAttend": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanPresent": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "AllowAdditionalMicrosoftServices": {
            "const": false
          },
          "AllowExternalMarketplace": {
            "const": false
          },
          "DisablePrint": {
            "const": false
          },
          "DisableTranslation": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "renamedfile.txt"
          }
        }
      }
      """


  Scenario: check the information of a project space file with fakeOffice
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" checks the information of file "text.txt" of space "new-space" using office "FakeOffice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "OwnerId",
          "Size",
          "UserId",
          "Version",
          "SupportsCobalt",
          "SupportsContainers",
          "SupportsDeleteFile",
          "SupportsEcosystem",
          "SupportsExtendedLockLength",
          "SupportsFolders",
          "SupportsGetLock",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "SupportsUserInfo",
          "UserFriendlyName",
          "ReadOnly",
          "RestrictedWebViewOnly",
          "UserCanAttend",
          "UserCanNotWriteRelative",
          "UserCanPresent",
          "UserCanRename",
          "UserCanWrite",
          "AllowAdditionalMicrosoftServices",
          "AllowExternalMarketplace",
          "DisablePrint",
          "DisableTranslation",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "text.txt"
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsCobalt": {
            "const": false
          },
          "SupportsContainers": {
            "const": false
          },
          "SupportsDeleteFile": {
            "const": true
          },
          "SupportsEcosystem": {
            "const": false
          },
          "SupportsExtendedLockLength": {
            "const": true
          },
          "SupportsFolders": {
            "const": false
          },
          "SupportsGetLock": {
            "const": true
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "SupportsUserInfo": {
            "const": false
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "RestrictedWebViewOnly": {
            "const": false
          },
          "UserCanAttend": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanPresent": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "AllowAdditionalMicrosoftServices": {
            "const": false
          },
          "AllowExternalMarketplace": {
            "const": false
          },
          "DisablePrint": {
            "const": false
          },
          "DisableTranslation": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "text.txt"
          }
        }
      }
      """


  Scenario: check the information of a project space file with collabora
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" checks the information of file "text.txt" of space "new-space" using office "Collabora"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "DisablePrint",
          "OwnerId",
          "PostMessageOrigin",
          "Size",
          "UserCanWrite",
          "UserCanNotWriteRelative",
          "UserId",
          "UserFriendlyName",
          "EnableOwnerTermination",
          "SupportsLocks",
          "SupportsRename",
          "UserCanRename",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "text.txt"
          },
          "PostMessageOrigin": {
            "const": "https://localhost:9200"
          },
          "DisablePrint": {
            "const": false
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserCanWrite": {
            "const": true
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "EnableOwnerTermination": {
            "const": true
          },
          "UserId": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "UserCanRename": {
            "const": true
          },
          "BreadcrumbDocName": {
            "const": "text.txt"
          }
        }
      }
      """


  Scenario: check the information of a project space file with onlyOffice
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" checks the information of file "text.txt" of space "new-space" using office "OnlyOffice"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "Version",
          "BreadcrumbDocName",
          "BreadcrumbFolderName",
          "BreadcrumbFolderUrl",
          "PostMessageOrigin",
          "DisablePrint",
          "UserFriendlyName",
          "UserId",
          "ReadOnly",
          "UserCanNotWriteRelative",
          "UserCanRename",
          "UserCanWrite",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate"
        ],
        "properties": {
          "BaseFileName": {
            "const": "text.txt"
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanRename": {
            "const": true
          },
          "UserCanWrite": {
            "const": true
          },
          "DisablePrint": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "text.txt"
          },
          "BreadcrumbFolderName": {
            "const": "new-space"
          },
          "BreadcrumbFolderUrl": {
            "type": "string"
          },
          "PostMessageOrigin": {
            "type": "string"
          }
        }
      }
      """


  Scenario Outline: check the information of a project space file with different modes (onlyOffice)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" checks the information of file "text.txt" of space "new-space" using office "OnlyOffice" with view mode "<mode>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "Version",
          "BreadcrumbDocName",
          "BreadcrumbFolderName",
          "BreadcrumbFolderUrl",
          "PostMessageOrigin",
          "DisablePrint",
          "UserFriendlyName",
          "UserId",
          "ReadOnly",
          "UserCanNotWriteRelative",
          "UserCanRename",
          "UserCanWrite",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate"
        ],
        "properties": {
          "BaseFileName": {
            "const": "text.txt"
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanRename": {
            "const": <user-can-rename>
          },
          "UserCanWrite": {
            "const": <user-can-write>
          },
          "DisablePrint": {
            "const": <disable-print>
          },
          "BreadcrumbDocName": {
            "const": "text.txt"
          },
          "BreadcrumbFolderName": {
            "const": "new-space"
          },
          "BreadcrumbFolderUrl": {
            "type": "string"
          },
          "PostMessageOrigin": {
            "type": "string"
          }
        }
      }
      """
    Examples:
      | mode  | disable-print | user-can-write | user-can-rename |
      | view  | true          | false          | false           |
      | read  | false         | false          | false           |
      | write | false         | true           | true            |


  Scenario Outline: check the information of a project space file with different modes (fakeOffice)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" checks the information of file "text.txt" of space "new-space" using office "FakeOffice" with view mode "<mode>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "OwnerId",
          "Size",
          "UserId",
          "Version",
          "SupportsCobalt",
          "SupportsContainers",
          "SupportsDeleteFile",
          "SupportsEcosystem",
          "SupportsExtendedLockLength",
          "SupportsFolders",
          "SupportsGetLock",
          "SupportsLocks",
          "SupportsRename",
          "SupportsUpdate",
          "SupportsUserInfo",
          "UserFriendlyName",
          "ReadOnly",
          "RestrictedWebViewOnly",
          "UserCanAttend",
          "UserCanNotWriteRelative",
          "UserCanPresent",
          "UserCanRename",
          "UserCanWrite",
          "AllowAdditionalMicrosoftServices",
          "AllowExternalMarketplace",
          "DisablePrint",
          "DisableTranslation",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "text.txt"
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserId": {
            "type": "string"
          },
          "Version": {
            "type": "string"
          },
          "SupportsCobalt": {
            "const": false
          },
          "SupportsContainers": {
            "const": false
          },
          "SupportsDeleteFile": {
            "const": true
          },
          "SupportsEcosystem": {
            "const": false
          },
          "SupportsExtendedLockLength": {
            "const": true
          },
          "SupportsFolders": {
            "const": false
          },
          "SupportsGetLock": {
            "const": true
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "SupportsUpdate": {
            "const": true
          },
          "SupportsUserInfo": {
            "const": false
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "ReadOnly": {
            "const": false
          },
          "RestrictedWebViewOnly": {
            "const": false
          },
          "UserCanAttend": {
            "const": false
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "UserCanPresent": {
            "const": false
          },
          "UserCanRename": {
            "const": <user-can-rename>
          },
          "UserCanWrite": {
            "const": <user-can-write>
          },
          "AllowAdditionalMicrosoftServices": {
            "const": false
          },
          "AllowExternalMarketplace": {
            "const": false
          },
          "DisablePrint": {
            "const": <disable-print>
          },
          "DisableTranslation": {
            "const": false
          },
          "BreadcrumbDocName": {
            "const": "text.txt"
          }
        }
      }
      """
    Examples:
      | mode  | disable-print | user-can-write | user-can-rename |
      | view  | true          | false          | false           |
      | read  | false         | false          | false           |
      | write | false         | true           | true            |


  Scenario Outline: check the information of a project space file with different modes (collabora)
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" checks the information of file "text.txt" of space "new-space" using office "Collabora" with view mode "<mode>"
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
      """
      {
        "type": "object",
        "required": [
          "BaseFileName",
          "DisablePrint",
          "OwnerId",
          "PostMessageOrigin",
          "Size",
          "UserCanWrite",
          "UserCanNotWriteRelative",
          "UserId",
          "UserFriendlyName",
          "EnableOwnerTermination",
          "SupportsLocks",
          "SupportsRename",
          "UserCanRename",
          "BreadcrumbDocName"
        ],
        "properties": {
          "BaseFileName": {
            "const": "text.txt"
          },
          "PostMessageOrigin": {
            "const": "https://localhost:9200"
          },
          "DisablePrint": {
            "const": <disable-print>
          },
          "OwnerId": {
            "type": "string"
          },
          "Size": {
            "const": 11
          },
          "UserCanWrite": {
            "const": <user-can-write>
          },
          "UserCanNotWriteRelative": {
            "const": false
          },
          "EnableOwnerTermination": {
            "const": true
          },
          "UserId": {
            "type": "string"
          },
          "SupportsLocks": {
            "const": true
          },
          "SupportsRename": {
            "const": true
          },
          "UserFriendlyName": {
            "const": "Alice Hansen"
          },
          "UserCanRename": {
            "const": <user-can-rename>
          },
          "BreadcrumbDocName": {
            "const": "text.txt"
          }
        }
      }
      """
    Examples:
      | mode  | disable-print | user-can-write | user-can-rename |
      | view  | true          | false          | false           |
      | read  | false         | false          | false           |
      | write | false         | true           | true            |


  Scenario Outline: try to get the information of a project space file using invalid file-id
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And using spaces DAV path
    And user "Alice" has created a space "new-space" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "new-space" with content "hello world" to "text.txt"
    When user "Alice" tries to check the information of file "text.txt" of space "new-space" using office "<office-suites>" with invalid file-id
    Then the HTTP status code should be "401"
    Examples:
      | office-suites |
      | Collabora     |
      | FakeOffice    |
      | OnlyOffice    |

  @issue-10086
  Scenario: check that '/app/list' doesn't contain 'application/octet-stream' MIME type
    When user "Alice" sends HTTP method "GET" to URL "/app/list"
    Then the HTTP status code should be "200"
    And the response should not contain the following MIME types:
      | application/octet-stream |


  Scenario: check target extensions for template mime types
    When user "Alice" sends HTTP method "GET" to URL "/app/list"
    Then the HTTP status code should be "200"
    And the app list response should contain the following template information for office "OnlyOffice":
      | mime-type                                                               | target-extension |
      | application/vnd.oasis.opendocument.spreadsheet-template                 | xlsx             |
      | application/vnd.oasis.opendocument.text-template                        | docx             |
      | application/vnd.oasis.opendocument.presentation-template                | pptx             |
      | application/vnd.openxmlformats-officedocument.spreadsheetml.template    | xlsx             |
      | application/vnd.openxmlformats-officedocument.presentationml.template   | pptx             |
      | application/vnd.openxmlformats-officedocument.wordprocessingml.template | docx             |
      | application/vnd.ms-word.template.macroenabled.12                        | docx             |
      | application/vnd.ms-powerpoint.template.macroenabled.12                  | pptx             |
      | application/vnd.ms-excel.template.macroenabled.12                       | xlsx             |
    And the app list response should contain the following template information for office "Collabora":
      | mime-type                                                | target-extension |
      | application/vnd.oasis.opendocument.spreadsheet-template  | ods              |
      | application/vnd.oasis.opendocument.text-template         | odt              |
      | application/vnd.oasis.opendocument.presentation-template | odp              |
