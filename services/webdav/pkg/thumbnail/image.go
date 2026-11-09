package thumbnail

// Image is an image and the type it is in, as an extractor hands it on and as
// the cache keeps it. webdav does not decode it, the generator does. An Image
// without data records that a file has none.
type Image struct {
	ContentType string
	Data        []byte
}
