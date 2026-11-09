package thumbnail

// Image is an image and the type it is in, as an extractor hands it on and as
// the cache keeps it. webdav does not decode it, the generator does.
type Image struct {
	ContentType string
	Data        []byte
}
