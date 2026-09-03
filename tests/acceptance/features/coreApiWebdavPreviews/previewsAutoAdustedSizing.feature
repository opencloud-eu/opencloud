@skipOnReva
Feature: sizing of previews of files downloaded through the webdav API
  As a user
  I want the aspect-ratio of previews to be preserved even when I ask for an unusual preview size
  So that the previews always have a similar look-and-feel to the original file

  This is optional behavior of an implementation. OpenCloud happens like this,
  but oC10 does not do this auto-fix of the aspect ratio.

  Background:
    Given user "Alice" has been created with default attributes


  Scenario Outline: download different sizes of previews of file (default = fill)
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    When user "Alice" downloads the preview of "/parent.txt" with width <request-width> and height <request-height> using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be <return-width> pixels wide and <return-height> pixels high
    # No processor is given, so the default behavior applies: fill. The source
    # preview of a text file is always 640x480 (landscape). The requested size is
    # snapped onto a configured resolution (the box) and the image is center-cropped
    # to fill that box exactly (it may be upscaled or downscaled to cover it), so
    # the output is always exactly the box dimensions.
    Examples:
      | request-width | request-height | return-width | return-height | dav-path-version |
      | 1             | 1              | 16           | 16            | old              |
      | 32            | 32             | 32           | 32            | old              |
      | 1024          | 1024           | 1024         | 1024          | old              |
      | 1             | 1024           | 1080         | 1920          | old              |
      | 1024          | 1              | 1024         | 1024          | old              |
      | 1             | 1              | 16           | 16            | new              |
      | 32            | 32             | 32           | 32            | new              |
      | 1024          | 1024           | 1024         | 1024          | new              |
      | 1             | 1024           | 1080         | 1920          | new              |
      | 1024          | 1              | 1024         | 1024          | new              |
      | 1             | 1              | 16           | 16            | spaces           |
      | 32            | 32             | 32           | 32            | spaces           |
      | 1024          | 1024           | 1024         | 1024          | spaces           |
      | 1             | 1024           | 1080         | 1920          | spaces           |
      | 1024          | 1              | 1024         | 1024          | spaces           |


  Scenario Outline: download a preview with an explicit processor
    Given using <dav-path-version> DAV path
    And user "Alice" has uploaded file "filesForUpload/lorem.txt" to "/parent.txt"
    When user "Alice" downloads the preview of "/parent.txt" with width <request-width> and height <request-height> and processor "<processor>" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded image should be <return-width> pixels wide and <return-height> pixels high
    # An explicit processor overrides the default. The source preview of a text
    # file is always 640x480 (landscape); the requested size is snapped onto a
    # configured resolution (the box) and the operation is applied within it:
    #   fit  -> aspect-fit within the box, never upscaled (keeps source ratio)
    #   fill -> center-crop to the exact box (may upscale/downscale to cover)
    #   resize -> stretch to the exact box (distorts aspect ratio)
    #
    # The OpenCloud web client always sends a processor: list view requests
    # processor=thumbnail at 64x64, tiles view requests processor=fit at a dynamic
    # square between 320 and 768. It currently also sends the legacy a=1 flag on
    # every request; that is redundant now that an explicit processor takes
    # precedence over a, so web should stop sending it.
    Examples:
      | request-width | request-height | processor | return-width | return-height | dav-path-version |
      | 1             | 1              | fit       | 16           | 12            | spaces           |
      | 32            | 32             | fit       | 32           | 24            | spaces           |
      | 1024          | 1024           | fit       | 640          | 480           | spaces           |
      | 1             | 1024           | fit       | 640          | 480           | spaces           |
      | 1024          | 1              | fit       | 640          | 480           | spaces           |
      | 1             | 1              | fill      | 16           | 16            | spaces           |
      | 32            | 32             | fill      | 32           | 32            | spaces           |
      | 1024          | 1024           | fill      | 1024         | 1024          | spaces           |
      | 1             | 1              | resize    | 16           | 16            | spaces           |
      | 32            | 32             | resize    | 32           | 32            | spaces           |
      | 1024          | 1024           | resize    | 1024         | 1024          | spaces           |
