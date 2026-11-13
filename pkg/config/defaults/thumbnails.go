package defaults

// ThumbnailsMaxInputImageFileSize is the default of OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE, the
// maximum size of an image a thumbnail is made of. The thumbnails, webdav and graph services read
// that setting each, and they must agree on the default too: webdav would otherwise send images the
// thumbnails service refuses, and graph would accept space images no thumbnail is made of.
const ThumbnailsMaxInputImageFileSize = "50MB"
