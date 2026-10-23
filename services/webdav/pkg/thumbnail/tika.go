package thumbnail

import (
	"mime"
	"slices"
	"strings"
)

// DefaultTikaTypes are the types sent to a Tika server for their thumbnail,
// the formats a Tika release serves one for. The camera raws have to be named
// because reva types them as images, and an image is handed to the generator
// as it is unless something claims it.
var DefaultTikaTypes = []string{
	"application/pdf",
	"application/epub+zip",
	"application/vnd.ms-xpsdocument",

	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.ms-powerpoint",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/vnd.ms-excel",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.oasis.opendocument.text",
	"application/vnd.oasis.opendocument.spreadsheet",
	"application/vnd.oasis.opendocument.presentation",
	"application/vnd.apple.pages",
	"application/vnd.apple.numbers",
	"application/vnd.apple.keynote",

	"application/vnd.geogebra.file",
	"application/vnd.geogebra.slides",
	"application/vnd.geogebra.pinboard",

	"audio/flac",
	"audio/mpeg",
	"audio/ogg",
	"audio/x-flac",
	"audio/x-m4a",

	"application/x-msdownload",

	"image/x-adobe-dng",
	"image/x-canon-cr2",
	"image/x-canon-crw",
	"image/x-epson-erf",
	"image/x-fuji-raf",
	"image/x-kodak-dcr",
	"image/x-kodak-kdc",
	"image/x-minolta-mrw",
	"image/x-nikon-nef",
	"image/x-nikon-nrw",
	"image/x-olympus-orf",
	"image/x-panasonic-raw",
	"image/x-panasonic-rw2",
	"image/x-pentax-pef",
	"image/x-sigma-x3f",
	"image/x-sony-arw",
	"image/x-sony-sr2",
	"image/x-sony-srf",
}

// Tika is the configured Tika server: its URL and the types it is asked for.
// It only decides which file goes there, the call lives in the preprocessor.
type Tika struct {
	URL string
	// our mime type -> Tika's, "" when equal; "*" means every type
	mimeTypes map[string]string
}

// NewTika parses "mime[:tika-mime]" entries. mimeTypes is the configured list,
// which the default config fills with DefaultTikaTypes, additional adds to it,
// and "*" in either sends every type that nothing else claims. An empty list
// sends nothing, which leaves a shared Tika server to the search service.
func NewTika(url string, mimeTypes, additional []string) Tika {
	t := Tika{URL: url, mimeTypes: map[string]string{}}
	for _, entry := range slices.Concat(mimeTypes, additional) {
		ours, theirs, _ := strings.Cut(strings.ToLower(strings.TrimSpace(entry)), ":")
		if ours = strings.TrimSpace(ours); ours != "" {
			t.mimeTypes[ours] = strings.TrimSpace(theirs)
		}
	}

	return t
}

// Supports reports whether the type is sent to Tika for its thumbnail.
func (t Tika) Supports(mimeType string) bool {
	if t.URL == "" {
		return false
	}
	m, _, err := mime.ParseMediaType(mimeType)
	if err != nil || m == "httpd/unix-directory" {
		return false
	}
	if _, ok := t.mimeTypes[m]; ok {
		return true
	}
	_, all := t.mimeTypes["*"]
	return all
}

// ConfiguredFor reports whether OC_TIKA_THUMBNAIL_MIME_TYPES lists this type,
// as opposed to the catch-all taking it. An image is only taken away from the
// generator when the configuration asks for it, because the generator may well
// read the type itself.
func (t Tika) ConfiguredFor(mimeType string) bool {
	if t.URL == "" {
		return false
	}
	m, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return false
	}
	_, named := t.mimeTypes[m]

	return named
}

// ContentType is the type Tika is told, mapped where configured. Tika takes a
// declared type only when it refines what it detected itself, so a mapping
// cannot overrule it; what it can do is pin the leaf type a detector left at
// the parent, a raw photo it only read as a plain tiff.
//
// That is the point of the mapping, and it is worth keeping even while nothing
// uses it: the type decides which parser reads the file, so a raw read as a
// tiff is parsed as a tiff, its maker notes stay unread, and the embedded
// preview the thumbnail is made of is never found. Every raw format shipped
// here is detected down to the leaf type today, so no entry is needed; the next
// format whose detection stops one level short needs exactly this.
func (t Tika) ContentType(mimeType string) string {
	m, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return ""
	}
	if theirs := t.mimeTypes[m]; theirs != "" {
		return theirs
	}
	return m
}
