@skipOnReva
Feature: sizing of previews of files downloaded through the webdav API
  As a user
  I want the aspect-ratio of previews to be preserved even when I ask for an unusual preview size
  So that the previews always have a similar look-and-feel to the original file

  This is optional behavior of an implementation. OpenCloud happens like this,
  but oC10 does not do this auto-fix of the aspect ratio.

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: download different sizes of previews of file
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    When user "Alice" downloads the preview of "/parent.txt" with width <request-width> and height <request-height> using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be <return-width> pixels wide and <return-height> pixels high
    Examples:
      | request-width | request-height | return-width | return-height | dav-path-version |
      | 1             | 1              | 16           | 16            | old              |
      | 32            | 32             | 32           | 32            | old              |
      | 1024          | 1024           | 640          | 640           | old              |
      | 1             | 1024           | 16           | 16            | old              |
      | 1024          | 1              | 640          | 640           | old              |
      | 1             | 1              | 16           | 16            | new              |
      | 32            | 32             | 32           | 32            | new              |
      | 1024          | 1024           | 640          | 640           | new              |
      | 1             | 1024           | 16           | 16            | new              |
      | 1024          | 1              | 640          | 640           | new              |
      | 1             | 1              | 16           | 16            | spaces           |
      | 32            | 32             | 32           | 32            | spaces           |
      | 1024          | 1024           | 640          | 640           | spaces           |
      | 1             | 1024           | 16           | 16            | spaces           |
      | 1024          | 1              | 640          | 640           | spaces           |
