package time

var decodeAsLocal = true

// SetDecodedAsLocal sets the decoded time to local time.
func SetDecodedAsLocal(b bool) {
	decodeAsLocal = b
}
