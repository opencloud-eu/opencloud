Feature: check etag propagation after different file alterations
  As a user
  I want to check the etag
  So that I can make sure that they are correct after different file alterations

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And user "Alice" has created folder "/upload"


  Scenario: copying a file inside a folder as a share receiver changes its etag for all collaborators
    Given user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/file.txt" inside space "Shares"
    And user "Brian" has stored etag of element "upload/file.txt" on path "/upload/renamed.txt" inside space "Shares"
    When user "Brian" copies file "/upload/file.txt" to "/upload/renamed.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path                | space    |
      | Alice | /                   | Personal |
      | Alice | /upload             | Personal |
      | Alice | /upload/renamed.txt | Personal |
      | Brian | /                   | Shares   |
      | Brian | /upload             | Shares   |
      | Brian | /upload/renamed.txt | Shares   |
    And these etags should not have changed
      | user  | path             | space    |
      | Alice | /upload/file.txt | Personal |
      | Brian | /upload/file.txt | Shares   |


  Scenario: copying a file inside a folder as a sharer changes its etag for all collaborators
    Given user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/file.txt" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Shares"
    When user "Alice" copies file "/upload/file.txt" to "/upload/renamed.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path                | space    |
      | Alice | /                   | Personal |
      | Alice | /upload             | Personal |
      | Alice | /upload/renamed.txt | Personal |
      | Brian | /                   | Shares   |
      | Brian | /upload             | Shares   |
      | Brian | /upload/renamed.txt | Shares   |
    And these etags should not have changed
      | user  | path             | space    |
      | Alice | /upload/file.txt | Personal |
      | Brian | /upload/file.txt | Shares   |


  Scenario: share receiver renaming a file inside a folder changes its etag for all collaborators
    Given user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Shares"
    When user "Brian" moves file "/upload/file.txt" to "/upload/renamed.txt" in space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |
    And these etags should not have changed
      | user  | path                | space    |
      | Alice | /upload/renamed.txt | Personal |
      | Brian | /upload/renamed.txt | Shares   |


  Scenario: sharer renaming a file inside a folder changes its etag for all collaborators
    Given user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/file.txt" on path "/upload/renamed.txt" inside space "Shares"
    When user "Alice" moves file "/upload/file.txt" to "/upload/renamed.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |
    And these etags should not have changed
      | user  | path                | space    |
      | Alice | /upload/renamed.txt | Personal |
      | Brian | /upload/renamed.txt | Shares   |


  Scenario: sharer moving a file from one folder to an other changes the etags of both folders for all collaborators
    Given user "Alice" has created folder "/dst"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | dst      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/dst" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/file.txt" on path "/dst/file.txt" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/dst" inside space "Shares"
    When user "Alice" moves file "/upload/file.txt" to "/dst/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Alice | /dst    | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |
      | Brian | /dst    | Shares   |


  Scenario: sharer moving a folder from one folder to an other changes the etags of both folders for all collaborators
    Given user "Alice" has created folder "/dst"
    And user "Alice" has created folder "/upload/toMove"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has sent the following resource share invitation:
      | resource        | dst      |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/dst" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/dst" inside space "Shares"
    When user "Alice" moves file "/upload/toMove" to "/dst/toMove" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Alice | /dst    | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |
      | Brian | /dst    | Shares   |


  Scenario: share receiver creating a folder inside a folder received as a share changes its etag for all collaborators
    Given user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    When user "Brian" creates a subfolder "/upload/new" in space "Shares" using the WebDav Api
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |


  Scenario: sharer creating a folder inside a shared folder changes etag for all collaborators
    Given user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    When user "Alice" creates folder "/upload/new" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |


  Scenario: share receiver uploading a file inside a folder received as a share changes its etag for all collaborators
    Given user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    When user "Brian" uploads a file inside space "Shares" with content "uploaded content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |


  Scenario: sharer uploading a file inside a shared folder should update etags for all collaborators
    Given user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    When user "Alice" uploads file with content "uploaded content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "201"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |


  Scenario: share receiver overwriting a file inside a received shared folder should update etags for all collaborators
    Given user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    When user "Brian" uploads a file inside space "Shares" with content "new content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |


  Scenario: sharer overwriting a file inside a shared folder should update etags for all collaborators
    Given user "Alice" has uploaded file with content "uploaded content" to "/upload/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    When user "Alice" uploads file with content "new content" to "/upload/file.txt" using the WebDAV API
    Then the HTTP status code should be "204"
    And these etags should have changed
      | user  | path    | space    |
      | Alice | /       | Personal |
      | Alice | /upload | Personal |
      | Brian | /       | Shares   |
      | Brian | /upload | Shares   |


  Scenario: share receiver deleting (removing) a file changes the etags of all parents for all collaborators
    Given user "Alice" has created folder "/upload/sub"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/sub" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/sub" inside space "Shares"
    When user "Brian" removes the file "upload/sub/file.txt" from space "Shares"
    Then the HTTP status code should be "204"
    And these etags should have changed
      | user  | path        | space    |
      | Alice | /           | Personal |
      | Alice | /upload     | Personal |
      | Alice | /upload/sub | Personal |
      | Brian | /           | Shares   |
      | Brian | /upload     | Shares   |
      | Brian | /upload/sub | Shares   |
    And these etags should not have changed
      | user  | path | space    |
      | Brian | /    | Personal |


  Scenario: sharer deleting (removing) a file changes the etags of all parents for all collaborators
    Given user "Alice" has created folder "/upload/sub"
    And user "Alice" has uploaded file with content "uploaded content" to "/upload/sub/file.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/sub" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/sub" inside space "Shares"
    When user "Alice" removes the file "upload/sub/file.txt" from space "Personal"
    Then the HTTP status code should be "204"
    And these etags should have changed
      | user  | path        | space    |
      | Alice | /           | Personal |
      | Alice | /upload     | Personal |
      | Alice | /upload/sub | Personal |
      | Brian | /           | Shares   |
      | Brian | /upload     | Shares   |
      | Brian | /upload/sub | Shares   |
    And these etags should not have changed
      | user  | path | space    |
      | Brian | /    | Personal |


  Scenario: share receiver deleting (removing) a folder changes the etags of all parents for all collaborators
    Given user "Alice" has created folder "/upload/sub"
    And user "Alice" has created folder "/upload/sub/toDelete"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/sub" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/sub" inside space "Shares"
    When user "Brian" removes the file "upload/sub/toDelete" from space "Shares"
    Then the HTTP status code should be "204"
    And these etags should have changed
      | user  | path        | space    |
      | Alice | /           | Personal |
      | Alice | /upload     | Personal |
      | Alice | /upload/sub | Personal |
      | Brian | /           | Shares   |
      | Brian | /upload     | Shares   |
      | Brian | /upload/sub | Shares   |
    And these etags should not have changed
      | user  | path | space    |
      | Brian | /    | Personal |


  Scenario: sharer deleting (removing) a folder changes the etags of all parents for all collaborators
    Given user "Alice" has created folder "/upload/sub"
    And user "Alice" has created folder "/upload/sub/toDelete"
    And user "Alice" has sent the following resource share invitation:
      | resource        | upload   |
      | space           | Personal |
      | sharee          | Brian    |
      | shareType       | user     |
      | permissionsRole | Editor   |
    And user "Alice" has stored etag of element "/" inside space "Personal"
    And user "Alice" has stored etag of element "/upload" inside space "Personal"
    And user "Alice" has stored etag of element "/upload/sub" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Personal"
    And user "Brian" has stored etag of element "/" inside space "Shares"
    And user "Brian" has stored etag of element "/upload" inside space "Shares"
    And user "Brian" has stored etag of element "/upload/sub" inside space "Shares"
    When user "Alice" removes the file "upload/sub/toDelete" from space "Personal"
    Then the HTTP status code should be "204"
    And these etags should have changed
      | user  | path        | space    |
      | Alice | /           | Personal |
      | Alice | /upload     | Personal |
      | Alice | /upload/sub | Personal |
      | Brian | /           | Shares   |
      | Brian | /upload     | Shares   |
      | Brian | /upload/sub | Shares   |
    And these etags should not have changed
      | user  | path | space    |
      | Brian | /    | Personal |
