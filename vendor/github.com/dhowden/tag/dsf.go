// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
)

// ReadDSFTags reads DSF metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
// samples: http://www.2l.no/hires/index.html
func ReadDSFTags(r io.ReadSeeker) (Metadata, error) {
	dsd, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if dsd != "DSD " {
		return nil, errors.New("expected 'DSD '")
	}

	_, err = r.Seek(int64(16), io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	id3Pointer, err := readUint64LittleEndian(r)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(int64(id3Pointer), io.SeekStart)
	if err != nil {
		return nil, err
	}

	id3, err := ReadID3v2Tags(r)
	if err != nil {
		return nil, err
	}

	return metadataDSF{id3}, nil
}

type metadataDSF struct {
	id3 Metadata
}

func (m metadataDSF) Format() Format {
	return m.id3.Format()
}

func (m metadataDSF) FileType() FileType {
	return DSF
}

func (m metadataDSF) Title() string {
	return m.id3.Title()
}

func (m metadataDSF) Album() string {
	return m.id3.Album()
}

func (m metadataDSF) Artist() string {
	return m.id3.Artist()
}

func (m metadataDSF) AlbumArtist() string {
	return m.id3.AlbumArtist()
}

func (m metadataDSF) Composer() string {
	return m.id3.Composer()
}

func (m metadataDSF) Year() int {
	return m.id3.Year()
}

func (m metadataDSF) Genre() string {
	return m.id3.Genre()
}

func (m metadataDSF) Track() (int, int) {
	return m.id3.Track()
}

func (m metadataDSF) Disc() (int, int) {
	return m.id3.Disc()
}

func (m metadataDSF) Picture() *Picture {
	return m.id3.Picture()
}

func (m metadataDSF) Pictures() []Picture {
	return m.id3.Pictures()
}

func (m metadataDSF) Lyrics() string {
	return m.id3.Lyrics()
}

func (m metadataDSF) Comment() string {
	return m.id3.Comment()
}

func (m metadataDSF) Raw() map[string]interface{} {
	return m.id3.Raw()
}
