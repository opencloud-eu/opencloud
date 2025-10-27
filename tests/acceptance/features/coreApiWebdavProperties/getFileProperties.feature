Feature: get file properties
  As a user
  I want to be able to get meta-information about files
  So that I can know file meta-information (detailed requirement TBD)

  Background:
    Given using OCS API version "1"
    And user "Alice" has been created with default attributes

  @smokeTest
  Scenario Outline: user sends a PROPFIND request on various file names
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "<file-name>"
    When user "Alice" gets the properties of file "<file-name>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the properties response should contain an etag
    Examples:
      | dav-path-version | file-name         |
      | old              | /upload.txt       |
      | old              | /strängé file.txt |
      | old              | /नेपाली.txt       |
      | old              | s,a,m,p,l,e.txt   |
      | new              | /upload.txt       |
      | new              | /strängé file.txt |
      | new              | /नेपाली.txt       |
      | new              | s,a,m,p,l,e.txt   |
      | spaces           | /upload.txt       |
      | spaces           | /strängé file.txt |
      | spaces           | /नेपाली.txt       |
      | spaces           | s,a,m,p,l,e.txt   |

  @owncloud-issue-1296
  Scenario Outline: user sends a PROPFIND request on various file with '?, #, &, +' in its name
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "<file-name>"
    When user "Alice" gets the properties of file "<file-name>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the properties response should contain an etag
    And there should be an entry with href containing "<expected-href>" in the response to user "Alice"
    Examples:
      | dav-path-version | file-name     | expected-href                     |
      | old              | /C++ file.cpp | webdav/C++ file.cpp               |
      | old              | /file #2.txt  | webdav/file #2.txt                |
      | old              | /file ?2.txt  | webdav/file ?2.txt                |
      | old              | /file &2.txt  | webdav/file &2.txt                |
      | new              | /C++ file.cpp | dav/files/%username%/C++ file.cpp |
      | new              | /file #2.txt  | dav/files/%username%/file #2.txt  |
      | new              | /file ?2.txt  | dav/files/%username%/file ?2.txt  |
      | new              | /file &2.txt  | dav/files/%username%/file &2.txt  |
      | spaces           | /C++ file.cpp | dav/spaces/%spaceid%/C++ file.cpp |
      | spaces           | /file #2.txt  | dav/spaces/%spaceid%/file #2.txt  |
      | spaces           | /file ?2.txt  | dav/spaces/%spaceid%/file ?2.txt  |
      | spaces           | /file &2.txt  | dav/spaces/%spaceid%/file &2.txt  |

  @owncloud-issue-1296
  Scenario Outline: user sends a PROPFIND request on various folder names
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    And user "Alice" has uploaded file with content "uploaded content" to "<folder-name>/file1.txt"
    And user "Alice" has uploaded file with content "uploaded content" to "<folder-name>/file2.txt"
    When user "Alice" gets the properties of folder "<folder-name>" with depth 1 using the WebDAV API
    Then the HTTP status code should be "207"
    And there should be an entry with href containing "<expected-href>/" in the response to user "Alice"
    And there should be an entry with href containing "<expected-href>/file1.txt" in the response to user "Alice"
    And there should be an entry with href containing "<expected-href>/file2.txt" in the response to user "Alice"
    Examples:
      | dav-path-version | folder-name     | expected-href                       |
      | old              | /upload         | webdav/upload                       |
      | old              | /strängé folder | webdav/strängé folder               |
      | old              | /C++ folder     | webdav/C++ folder                   |
      | old              | /नेपाली           | webdav/नेपाली                         |
      | old              | /folder #2.txt  | webdav/folder #2.txt                |
      | old              | /folder ?2.txt  | webdav/folder ?2.txt                |
      | old              | /folder &2.txt  | webdav/folder &2.txt                |
      | new              | /upload         | dav/files/%username%/upload         |
      | new              | /strängé folder | dav/files/%username%/strängé folder |
      | new              | /C++ folder     | dav/files/%username%/C++ folder     |
      | new              | /नेपाली           | dav/files/%username%/नेपाली           |
      | new              | /folder #2.txt  | dav/files/%username%/folder #2.txt  |
      | new              | /folder ?2.txt  | dav/files/%username%/folder ?2.txt  |
      | new              | /folder &2.txt  | dav/files/%username%/folder &2.txt  |
      | spaces           | /upload         | dav/spaces/%spaceid%/upload         |
      | spaces           | /strängé folder | dav/spaces/%spaceid%/strängé folder |
      | spaces           | /C++ folder     | dav/spaces/%spaceid%/C++ folder     |
      | spaces           | /नेपाली           | dav/spaces/%spaceid%/नेपाली           |
      | spaces           | /folder #2.txt  | dav/spaces/%spaceid%/folder #2.txt  |
      | spaces           | /folder ?2.txt  | dav/spaces/%spaceid%/folder ?2.txt  |
      | spaces           | /folder &2.txt  | dav/spaces/%spaceid%/folder &2.txt  |


  Scenario Outline: user sends a PROPFIND request on various files inside various folders
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "<folder-name>"
    And user "Alice" has uploaded file with content "uploaded content" to "<folder-name>/<file-name>"
    When user "Alice" gets the properties of file "<folder-name>/<file-name>" using the WebDAV API
    Then the HTTP status code should be "207"
    And the properties response should contain an etag
    Examples:
      | dav-path-version | folder-name                      | file-name                     |
      | old              | /upload                          | abc.txt                       |
      | old              | /strängé folder                  | strängé file.txt              |
      | old              | /C++ folder                      | C++ file.cpp                  |
      | old              | /नेपाली                          | नेपाली                        |
      | old              | /folder #2.txt                   | file #2.txt                   |
      | new              | /upload                          | abc.txt                       |
      | new              | /strängé folder (duplicate #2 &) | strängé file (duplicate #2 &) |
      | new              | /C++ folder                      | C++ file.cpp                  |
      | new              | /नेपाली                          | नेपाली                        |
      | new              | /folder #2.txt                   | file #2.txt                   |
      | spaces           | /upload                          | abc.txt                       |
      | spaces           | /strängé folder                  | strängé file.txt              |
      | spaces           | /C++ folder                      | C++ file.cpp                  |
      | spaces           | /नेपाली                          | नेपाली                        |
      | spaces           | /folder #2.txt                   | file #2.txt                   |

  @owncloud-issue-1259
  #after fixing all issues delete this Scenario and merge with the one above
  Scenario Outline: user sends a PROPFIND request on various files inside various folders with '?' character in its name
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/folder ?2.txt"
    And user "Alice" has uploaded file with content "uploaded content" to "/folder ?2.txt/file ?2.txt"
    When user "Alice" gets the properties of file "/folder ?2.txt/file ?2.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the properties response should contain an etag
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: file that is not shared does not have a share-types property
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain an empty property "oc:share-types"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |

  @skipOnReva
  Scenario Outline: resource that is shared to a user has a share-types property
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And user "Alice" has created folder "/test"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    When user "Alice" gets the following properties of folder "<resource>" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain a share-types property with
      | 0 |
    Examples:
      | dav-path-version | resource     |
      | old              | test         |
      | new              | test         |
      | spaces           | test         |
      | old              | textfile.txt |
      | new              | textfile.txt |
      | spaces           | textfile.txt |

  @skipOnReva
  Scenario Outline: resource that is shared to a group has a share-types property
    Given using <dav-path-version> DAV path
    And group "grp1" has been created
    And user "Alice" has created folder "/test"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | grp1       |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    When user "Alice" gets the following properties of folder "<resource>" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain a share-types property with
      | 1 |
    Examples:
      | dav-path-version | resource     |
      | old              | test         |
      | new              | test         |
      | spaces           | test         |
      | old              | textfile.txt |
      | new              | textfile.txt |
      | spaces           | textfile.txt |

  @skipOnReva
  Scenario Outline: resource that is shared by link has a share-types property
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | <resource> |
      | space           | Personal   |
      | permissionsRole | edit       |
      | password        | %public%   |
    When user "Alice" gets the following properties of folder "<resource>" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain a share-types property with
      | 3 |
    Examples:
      | dav-path-version | resource     |
      | old              | test         |
      | new              | test         |
      | spaces           | test         |
      | old              | textfile.txt |
      | new              | textfile.txt |
      | spaces           | textfile.txt |

  @skipOnReva
  Scenario Outline: resource that is shared by user,group and link has a share-types property
    Given using <dav-path-version> DAV path
    And user "Brian" has been created with default attributes
    And group "grp1" has been created
    And user "Alice" has created folder "/test"
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/textfile.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | Brian      |
      | shareType       | user       |
      | permissionsRole | Viewer     |
    And user "Alice" has sent the following resource share invitation:
      | resource        | <resource> |
      | space           | Personal   |
      | sharee          | grp1       |
      | shareType       | group      |
      | permissionsRole | Viewer     |
    And user "Alice" has created the following resource link share:
      | resource        | <resource> |
      | space           | Personal   |
      | permissionsRole | edit       |
      | password        | %public%   |
    When user "Alice" gets the following properties of folder "<resource>" using the WebDAV API
      | propertyName   |
      | oc:share-types |
    Then the HTTP status code should be "207"
    And the response should contain a share-types property with
      | 0 |
      | 1 |
      | 3 |
    Examples:
      | dav-path-version | resource     |
      | old              | test         |
      | new              | test         |
      | spaces           | test         |
      | old              | textfile.txt |
      | new              | textfile.txt |
      | spaces           | textfile.txt |

  @smokeTest @owncloud-issue-2809
  Scenario Outline: retrieving a private link
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/textfile.txt" to "/somefile.txt"
    When user "Alice" gets the following properties of file "/somefile.txt" using the WebDAV API
      | propertyName   |
      | oc:privatelink |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:privatelink" with value like "%(/(index.php/)?f/[0-9]*)%"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user sends a PROPFIND request on a nonexistent URL
    When user "Alice" requests "<url>" with "PROPFIND" using basic auth
    Then the HTTP status code should be "404"
    And the value of the item "/d:error/s:message" in the response about user "Alice" should be "<message1>" or "<message2>"
    And the value of the item "/d:error/s:exception" in the response about user "Alice" should be "Sabre\DAV\Exception\NotFound"

    Examples:
      | url                                  | message1               | message2           |
      | /dav/files/does-not-exist            | Resource not found     | Resource not found |
      | /dav/does-not-exist                  | File not found in root |                    |
      | /dav/spaces/%spaceid%/does-not-exist | Resource not found     |                    |
      | /dav/spaces/%spaceid%/file1.txt      | Resource not found     |                    |

  @owncloud-issue-1297
  Scenario Outline: add, receive multiple custom meta properties to a file
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/TestFolder"
    And user "Alice" has uploaded file with content "test data one" to "/TestFolder/test1.txt"
    And user "Alice" has set the following properties to file "/TestFolder/test1.txt" using the WebDav API
      | propertyName | propertyValue |
      | testprop1    | AAAAA         |
      | testprop2    | BBBBB         |
    When user "Alice" gets the following properties of file "/TestFolder/test1.txt" using the WebDAV API
      | propertyName |
      | testprop1    |
      | testprop2    |
    Then the HTTP status code should be "207"
    And as user "Alice" the last response should have the following properties
      | resource              | propertyName | propertyValue   |
      | /TestFolder/test1.txt | testprop1    | AAAAA           |
      | /TestFolder/test1.txt | testprop2    | BBBBB           |
      | /TestFolder/test1.txt | d:status     | HTTP/1.1 200 OK |
    Examples:
      | dav-path-version |
      | new              |
      | old              |
      | spaces           |

  @owncloud-issue-1297
  Scenario Outline: add multiple properties to files inside a folder and do a propfind of the parent folder
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/TestFolder"
    And user "Alice" has uploaded file with content "test data one" to "/TestFolder/test1.txt"
    And user "Alice" has uploaded file with content "test data two" to "/TestFolder/test2.txt"
    And user "Alice" has set the following properties to file "/TestFolder/test1.txt" using the WebDav API
      | propertyName | propertyValue |
      | testprop1    | AAAAA         |
      | testprop2    | BBBBB         |
    And user "Alice" has set the following properties to file "/TestFolder/test2.txt" using the WebDav API
      | propertyName | propertyValue |
      | testprop1    | CCCCC         |
      | testprop2    | DDDDD         |
    When user "Alice" gets the following properties of folder "/TestFolder" using the WebDAV API
      | propertyName |
      | testprop1    |
      | testprop2    |
    Then the HTTP status code should be "207"
    And as user "Alice" the last response should have the following properties
      | resource              | propertyName | propertyValue          |
      | /TestFolder/test1.txt | testprop1    | AAAAA                  |
      | /TestFolder/test1.txt | testprop2    | BBBBB                  |
      | /TestFolder/test2.txt | testprop1    | CCCCC                  |
      | /TestFolder/test2.txt | testprop2    | DDDDD                  |
      | /TestFolder/          | d:status     | HTTP/1.1 404 Not Found |
    Examples:
      | dav-path-version |
      | new              |
      | old              |
      | spaces           |


  Scenario Outline: user tries to get the last modified date of a folder using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName      |
      | d:getlastmodified |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:getlastmodified" with value like "/^[MTWFS][uedhfriatno]{2},\s(\d){2}\s[JFMAJSOND][anebrpyulgctov]{2}\s\d{4}\s\d{2}:\d{2}:\d{2} GMT$/"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the content type of a folder using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName     |
      | d:getcontenttype |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:getcontenttype" with value ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the content type of a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of folder "file.txt" using the WebDAV API
      | propertyName     |
      | d:getcontenttype |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:getcontenttype" with value "text/plain.*"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the etag of a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of folder "file.txt" using the WebDAV API
      | propertyName |
      | d:getetag    |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:getetag" with value like '%\"[a-z0-9:]{1,32}\"%'
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the resource type of a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of folder "file.txt" using the WebDAV API
      | propertyName   |
      | d:resourcetype |
    Then the HTTP status code should be "207"
    And the single response should contain a property "d:resourcetype" with value ""
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the size of a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of folder "file.txt" using the WebDAV API
      | propertyName |
      | oc:size      |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:size" with value "16"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the size of a folder using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName |
      | oc:size      |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:size" with value "0"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the file id of a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of folder "file.txt" using the WebDAV API
      | propertyName |
      | oc:fileid    |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:fileid" with value like '/[a-zA-Z0-9]+/'
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the file id of a folder using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName |
      | oc:fileid    |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:fileid" with value like '/[a-zA-Z0-9]+/'
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the owner display name of a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of file "file.txt" using the WebDAV API
      | propertyName          |
      | oc:owner-display-name |
    Then the HTTP status code should be "207"
    And the single response about the file owned by "Alice" should contain a property "oc:owner-display-name" with value "%displayname%"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the owner display name of a folder using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName          |
      | oc:owner-display-name |
    Then the HTTP status code should be "207"
    And the single response about the file owned by "Alice" should contain a property "oc:owner-display-name" with value "%displayname%"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the permissions on a file using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file with content "uploaded content" to "file.txt"
    When user "Alice" gets the following properties of folder "file.txt" using the WebDAV API
      | propertyName   |
      | oc:permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:permissions" with value like '/RM{0,1}DNVW/'
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the permissions on a folder using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName   |
      | oc:permissions |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:permissions" with value like '/RM{0,1}DNVCK/'
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |


  Scenario Outline: user tries to get the size of a folder with files using webdav api
    Given using <dav-path-version> DAV path
    And user "Alice" has created folder "/test"
    And user "Alice" has uploaded file with content "a" to "/test/file.txt"
    And user "Alice" has uploaded file with content "b" to "/test/file1.txt"
    And user "Alice" has uploaded file with content "c" to "/test/file2.txt"
    And user "Alice" has uploaded file with content "d" to "/test/file3.txt"
    And user "Alice" has uploaded file with content "e" to "/test/file4.txt"
    When user "Alice" gets the following properties of folder "/test" using the WebDAV API
      | propertyName |
      | oc:size      |
    Then the HTTP status code should be "207"
    And the single response should contain a property "oc:size" with value "5"
    Examples:
      | dav-path-version |
      | old              |
      | new              |
      | spaces           |
