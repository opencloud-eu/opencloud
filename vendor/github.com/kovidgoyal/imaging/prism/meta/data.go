package meta

import (
	"bytes"
	"fmt"
	"slices"
	"sync"

	"github.com/kovidgoyal/imaging/prism/meta/icc"
	"github.com/kovidgoyal/imaging/types"
	"github.com/rwcarlsen/goexif/exif"
)

var _ = fmt.Println

// Data represents the metadata for an image.
type Data struct {
	Format              types.Format
	PixelWidth          uint32
	PixelHeight         uint32
	BitsPerComponent    uint32
	HasFrames           bool
	NumFrames, NumPlays int
	CICP                CodingIndependentCodePoints

	mutex          sync.Mutex
	exifData       []byte
	exif           *exif.Exif
	exifErr        error
	iccProfileData []byte
	iccProfileErr  error
	iccProfile     *icc.Profile
}

func (s *Data) Clone() *Data {
	return &Data{
		Format: s.Format, PixelWidth: s.PixelWidth, PixelHeight: s.PixelHeight, BitsPerComponent: s.BitsPerComponent,
		HasFrames: s.HasFrames, NumFrames: s.NumFrames, NumPlays: s.NumPlays, CICP: s.CICP,
		exifData: slices.Clone(s.exifData), exifErr: s.exifErr, iccProfileData: slices.Clone(s.iccProfileData),
		iccProfileErr: s.iccProfileErr,
	}
}

func (s *Data) IsSRGB() bool {
	if s.CICP.IsSet {
		return s.CICP.IsSRGB()
	}
	if p, err := s.ICCProfile(); p != nil && err == nil {
		return p.IsSRGB()
	}
	return true
}

// Returns an extracted EXIF metadata object from this metadata.
//
// An error is returned if the EXIF profile could not be correctly parsed.
//
// If no EXIF data was found, nil is returned without an error.
func (md *Data) Exif() (*exif.Exif, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	if md.exifErr != nil {
		return nil, md.exifErr
	}
	if md.exif != nil {
		return md.exif, nil
	}
	if len(md.exifData) == 0 {
		return nil, nil
	}
	md.exif, md.exifErr = exif.Decode(bytes.NewReader(md.exifData))
	return md.exif, md.exifErr
}

func (md *Data) SetExifData(data []byte) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.exifData = data
	md.exifErr = nil
	md.exif = nil
}

func (md *Data) SetExif(e *exif.Exif) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.exifData = nil
	md.exifErr = nil
	md.exif = e
}

func (md *Data) SetExifError(e error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.exifData = nil
	md.exifErr = e
	md.exif = nil
}

func (md *Data) ExifData() []byte {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	return md.exifData
}

// ICCProfile returns an extracted ICC profile from this metadata.
//
// An error is returned if the ICC profile could not be correctly parsed.
//
// If no profile data was found, nil is returned without an error.
func (md *Data) ICCProfile() (*icc.Profile, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	if md.iccProfileErr != nil {
		return nil, md.iccProfileErr
	}
	if len(md.iccProfileData) == 0 {
		return nil, nil
	}
	md.iccProfile, md.iccProfileErr = icc.NewProfileReader(bytes.NewReader(md.iccProfileData)).ReadProfile()
	return md.iccProfile, md.iccProfileErr
}

// ICCProfileData returns the raw ICC profile data from this metadata.
//
// An error is returned if the ICC profile could not be correctly extracted from
// the image.
//
// If no profile data was found, nil is returned without an error.
func (md *Data) ICCProfileData() ([]byte, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	return md.iccProfileData, md.iccProfileErr
}

func (md *Data) SetICCProfileData(data []byte) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.iccProfileData = data
	md.iccProfileErr = nil
	md.iccProfile = nil
}

func (md *Data) SetICCProfileError(err error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.iccProfileData = nil
	md.iccProfile = nil
	md.iccProfileErr = err
}
