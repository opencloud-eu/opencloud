package meta

import (
	"bytes"
	"fmt"

	"github.com/kovidgoyal/imaging/prism/meta/icc"
)

var _ = fmt.Println

// Data represents the metadata for an image.
type Data struct {
	Format           ImageFormat
	PixelWidth       uint32
	PixelHeight      uint32
	BitsPerComponent uint32
	ExifData         []byte
	iccProfileData   []byte
	iccProfileErr    error
}

// ICCProfile returns an extracted ICC profile from this metadata.
//
// An error is returned if the ICC profile could not be correctly parsed.
//
// If no profile data was found, nil is returned without an error.
func (md *Data) ICCProfile() (*icc.Profile, error) {
	if md.iccProfileData == nil {
		return nil, md.iccProfileErr
	}

	return icc.NewProfileReader(bytes.NewReader(md.iccProfileData)).ReadProfile()
}

// ICCProfile returns the raw ICC profile data from this metadata.
//
// An error is returned if the ICC profile could not be correctly extracted from
// the image.
//
// If no profile data was found, nil is returned without an error.
func (md *Data) ICCProfileData() ([]byte, error) {
	return md.iccProfileData, md.iccProfileErr
}

func (md *Data) SetICCProfileData(data []byte) {
	md.iccProfileData = data
	md.iccProfileErr = nil
}

func (md *Data) SetICCProfileError(err error) {
	md.iccProfileData = nil
	md.iccProfileErr = err
}
