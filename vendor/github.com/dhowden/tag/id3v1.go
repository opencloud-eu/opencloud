// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
	"strconv"
	"strings"
)

// id3v1Genres is a list of genres as given in the ID3v1 specification.
var id3v1Genres = [...]string{
	"Blues", "Classic Rock", "Country", "Dance", "Disco", "Funk", "Grunge",
	"Hip-Hop", "Jazz", "Metal", "New Age", "Oldies", "Other", "Pop", "R&B",
	"Rap", "Reggae", "Rock", "Techno", "Industrial", "Alternative", "Ska",
	"Death Metal", "Pranks", "Soundtrack", "Euro-Techno", "Ambient",
	"Trip-Hop", "Vocal", "Jazz+Funk", "Fusion", "Trance", "Classical",
	"Instrumental", "Acid", "House", "Game", "Sound Clip", "Gospel",
	"Noise", "AlternRock", "Bass", "Soul", "Punk", "Space", "Meditative",
	"Instrumental Pop", "Instrumental Rock", "Ethnic", "Gothic",
	"Darkwave", "Techno-Industrial", "Electronic", "Pop-Folk",
	"Eurodance", "Dream", "Southern Rock", "Comedy", "Cult", "Gangsta",
	"Top 40", "Christian Rap", "Pop/Funk", "Jungle", "Native American",
	"Cabaret", "New Wave", "Psychadelic", "Rave", "Showtunes", "Trailer",
	"Lo-Fi", "Tribal", "Acid Punk", "Acid Jazz", "Polka", "Retro",
	"Musical", "Rock & Roll", "Hard Rock", "Folk", "Folk-Rock",
	"National Folk", "Swing", "Fast Fusion", "Bebob", "Latin", "Revival",
	"Celtic", "Bluegrass", "Avantgarde", "Gothic Rock", "Progressive Rock",
	"Psychedelic Rock", "Symphonic Rock", "Slow Rock", "Big Band",
	"Chorus", "Easy Listening", "Acoustic", "Humour", "Speech", "Chanson",
	"Opera", "Chamber Music", "Sonata", "Symphony", "Booty Bass", "Primus",
	"Porn Groove", "Satire", "Slow Jam", "Club", "Tango", "Samba",
	"Folklore", "Ballad", "Power Ballad", "Rhythmic Soul", "Freestyle",
	"Duet", "Punk Rock", "Drum Solo", "Acapella", "Euro-House", "Dance Hall",
}

// ErrNotID3v1 is an error which is returned when no ID3v1 header is found.
var ErrNotID3v1 = errors.New("invalid ID3v1 header")

// ReadID3v1Tags reads ID3v1 tags from the io.ReadSeeker.  Returns ErrNotID3v1
// if there are no ID3v1 tags, otherwise non-nil error if there was a problem.
func ReadID3v1Tags(r io.ReadSeeker) (Metadata, error) {
	_, err := r.Seek(-128, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	if tag, err := readString(r, 3); err != nil {
		return nil, err
	} else if tag != "TAG" {
		return nil, ErrNotID3v1
	}

	title, err := readString(r, 30)
	if err != nil {
		return nil, err
	}

	artist, err := readString(r, 30)
	if err != nil {
		return nil, err
	}

	album, err := readString(r, 30)
	if err != nil {
		return nil, err
	}

	year, err := readString(r, 4)
	if err != nil {
		return nil, err
	}

	commentBytes, err := readBytes(r, 30)
	if err != nil {
		return nil, err
	}

	var comment string
	var track int
	if commentBytes[28] == 0 {
		comment = trimString(string(commentBytes[:28]))
		track = int(commentBytes[29])
	} else {
		comment = trimString(string(commentBytes))
	}

	var genre string
	genreID, err := readBytes(r, 1)
	if err != nil {
		return nil, err
	}
	if int(genreID[0]) < len(id3v1Genres) {
		genre = id3v1Genres[int(genreID[0])]
	}

	m := make(map[string]interface{})
	m["title"] = trimString(title)
	m["artist"] = trimString(artist)
	m["album"] = trimString(album)
	m["year"] = trimString(year)
	m["comment"] = trimString(comment)
	m["track"] = track
	m["genre"] = genre

	return metadataID3v1(m), nil
}

func trimString(x string) string {
	return strings.TrimSpace(strings.Trim(x, "\x00"))
}

// metadataID3v1 is the implementation of Metadata used for ID3v1 tags.
type metadataID3v1 map[string]interface{}

func (metadataID3v1) Format() Format                { return ID3v1 }
func (metadataID3v1) FileType() FileType            { return MP3 }
func (m metadataID3v1) Raw() map[string]interface{} { return m }

func (m metadataID3v1) Title() string  { return m["title"].(string) }
func (m metadataID3v1) Album() string  { return m["album"].(string) }
func (m metadataID3v1) Artist() string { return m["artist"].(string) }
func (m metadataID3v1) Genre() string  { return m["genre"].(string) }

func (m metadataID3v1) Year() int {
	y := m["year"].(string)
	n, err := strconv.Atoi(y)
	if err != nil {
		return 0
	}
	return n
}

func (m metadataID3v1) Track() (int, int) { return m["track"].(int), 0 }

func (m metadataID3v1) AlbumArtist() string { return "" }
func (m metadataID3v1) Composer() string    { return "" }
func (metadataID3v1) Disc() (int, int)      { return 0, 0 }
func (m metadataID3v1) Picture() *Picture   { return nil }
func (m metadataID3v1) Pictures() []Picture { return nil }
func (m metadataID3v1) Lyrics() string      { return "" }
func (m metadataID3v1) Comment() string     { return m["comment"].(string) }
