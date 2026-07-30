// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

// DefaultUTF16WithBOMByteOrder is the byte order used when the "UTF16 with BOM" encoding
// is specified without a corresponding BOM in the data.
var DefaultUTF16WithBOMByteOrder binary.ByteOrder = binary.LittleEndian

// ID3v2.2.0 frames (see http://id3.org/id3v2-00, sec 4).
var id3v22Frames = map[string]string{
	"BUF": "Recommended buffer size",

	"CNT": "Play counter",
	"COM": "Comments",
	"CRA": "Audio encryption",
	"CRM": "Encrypted meta frame",

	"ETC": "Event timing codes",
	"EQU": "Equalization",

	"GEO": "General encapsulated object",

	"IPL": "Involved people list",

	"LNK": "Linked information",

	"MCI": "Music CD Identifier",
	"MLL": "MPEG location lookup table",

	"PIC": "Attached picture",
	"POP": "Popularimeter",

	"REV": "Reverb",
	"RVA": "Relative volume adjustment",

	"SLT": "Synchronized lyric/text",
	"STC": "Synced tempo codes",

	"TAL": "Album/Movie/Show title",
	"TBP": "BPM (Beats Per Minute)",
	"TCM": "Composer",
	"TCO": "Content type",
	"TCR": "Copyright message",
	"TDA": "Date",
	"TDY": "Playlist delay",
	"TEN": "Encoded by",
	"TFT": "File type",
	"TIM": "Time",
	"TKE": "Initial key",
	"TLA": "Language(s)",
	"TLE": "Length",
	"TMT": "Media type",
	"TOA": "Original artist(s)/performer(s)",
	"TOF": "Original filename",
	"TOL": "Original Lyricist(s)/text writer(s)",
	"TOR": "Original release year",
	"TOT": "Original album/Movie/Show title",
	"TP1": "Lead artist(s)/Lead performer(s)/Soloist(s)/Performing group",
	"TP2": "Band/Orchestra/Accompaniment",
	"TP3": "Conductor/Performer refinement",
	"TP4": "Interpreted, remixed, or otherwise modified by",
	"TPA": "Part of a set",
	"TPB": "Publisher",
	"TRC": "ISRC (International Standard Recording Code)",
	"TRD": "Recording dates",
	"TRK": "Track number/Position in set",
	"TSI": "Size",
	"TSS": "Software/hardware and settings used for encoding",
	"TT1": "Content group description",
	"TT2": "Title/Songname/Content description",
	"TT3": "Subtitle/Description refinement",
	"TXT": "Lyricist/text writer",
	"TXX": "User defined text information frame",
	"TYE": "Year",

	"UFI": "Unique file identifier",
	"ULT": "Unsychronized lyric/text transcription",

	"WAF": "Official audio file webpage",
	"WAR": "Official artist/performer webpage",
	"WAS": "Official audio source webpage",
	"WCM": "Commercial information",
	"WCP": "Copyright/Legal information",
	"WPB": "Publishers official webpage",
	"WXX": "User defined URL link frame",
}

// ID3v2.3.0 frames (see http://id3.org/id3v2.3.0#Declared_ID3v2_frames).
var id3v23Frames = map[string]string{
	"AENC": "Audio encryption]",
	"APIC": "Attached picture",
	"COMM": "Comments",
	"COMR": "Commercial frame",
	"ENCR": "Encryption method registration",
	"EQUA": "Equalization",
	"ETCO": "Event timing codes",
	"GEOB": "General encapsulated object",
	"GRID": "Group identification registration",
	"IPLS": "Involved people list",
	"LINK": "Linked information",
	"MCDI": "Music CD identifier",
	"MLLT": "MPEG location lookup table",
	"OWNE": "Ownership frame",
	"PRIV": "Private frame",
	"PCNT": "Play counter",
	"POPM": "Popularimeter",
	"POSS": "Position synchronisation frame",
	"RBUF": "Recommended buffer size",
	"RVAD": "Relative volume adjustment",
	"RVRB": "Reverb",
	"SYLT": "Synchronized lyric/text",
	"SYTC": "Synchronized tempo codes",
	"TALB": "Album/Movie/Show title",
	"TBPM": "BPM (beats per minute)",
	"TCMP": "iTunes Compilation Flag",
	"TCOM": "Composer",
	"TCON": "Content type",
	"TCOP": "Copyright message",
	"TDAT": "Date",
	"TDLY": "Playlist delay",
	"TENC": "Encoded by",
	"TEXT": "Lyricist/Text writer",
	"TFLT": "File type",
	"TIME": "Time",
	"TIT1": "Content group description",
	"TIT2": "Title/songname/content description",
	"TIT3": "Subtitle/Description refinement",
	"TKEY": "Initial key",
	"TLAN": "Language(s)",
	"TLEN": "Length",
	"TMED": "Media type",
	"TOAL": "Original album/movie/show title",
	"TOFN": "Original filename",
	"TOLY": "Original lyricist(s)/text writer(s)",
	"TOPE": "Original artist(s)/performer(s)",
	"TORY": "Original release year",
	"TOWN": "File owner/licensee",
	"TPE1": "Lead performer(s)/Soloist(s)",
	"TPE2": "Band/orchestra/accompaniment",
	"TPE3": "Conductor/performer refinement",
	"TPE4": "Interpreted, remixed, or otherwise modified by",
	"TPOS": "Part of a set",
	"TPUB": "Publisher",
	"TRCK": "Track number/Position in set",
	"TRDA": "Recording dates",
	"TRSN": "Internet radio station name",
	"TRSO": "Internet radio station owner",
	"TSIZ": "Size",
	"TSO2": "iTunes uses this for Album Artist sort order",
	"TSOC": "iTunes uses this for Composer sort order",
	"TSRC": "ISRC (international standard recording code)",
	"TSSE": "Software/Hardware and settings used for encoding",
	"TYER": "Year",
	"TXXX": "User defined text information frame",
	"UFID": "Unique file identifier",
	"USER": "Terms of use",
	"USLT": "Unsychronized lyric/text transcription",
	"WCOM": "Commercial information",
	"WCOP": "Copyright/Legal information",
	"WOAF": "Official audio file webpage",
	"WOAR": "Official artist/performer webpage",
	"WOAS": "Official audio source webpage",
	"WORS": "Official internet radio station homepage",
	"WPAY": "Payment",
	"WPUB": "Publishers official webpage",
	"WXXX": "User defined URL link frame",
}

// ID3v2.4.0 frames (see http://id3.org/id3v2.4.0-frames, sec 4).
var id3v24Frames = map[string]string{
	"AENC": "Audio encryption",
	"APIC": "Attached picture",
	"ASPI": "Audio seek point index",

	"COMM": "Comments",
	"COMR": "Commercial frame",

	"ENCR": "Encryption method registration",
	"EQU2": "Equalisation (2)",
	"ETCO": "Event timing codes",

	"GEOB": "General encapsulated object",
	"GRID": "Group identification registration",

	"LINK": "Linked information",

	"MCDI": "Music CD identifier",
	"MLLT": "MPEG location lookup table",

	"OWNE": "Ownership frame",

	"PRIV": "Private frame",
	"PCNT": "Play counter",
	"POPM": "Popularimeter",
	"POSS": "Position synchronisation frame",

	"RBUF": "Recommended buffer size",
	"RVA2": "Relative volume adjustment (2)",
	"RVRB": "Reverb",

	"SEEK": "Seek frame",
	"SIGN": "Signature frame",
	"SYLT": "Synchronised lyric/text",
	"SYTC": "Synchronised tempo codes",

	"TALB": "Album/Movie/Show title",
	"TBPM": "BPM (beats per minute)",
	"TCMP": "iTunes Compilation Flag",
	"TCOM": "Composer",
	"TCON": "Content type",
	"TCOP": "Copyright message",
	"TDEN": "Encoding time",
	"TDLY": "Playlist delay",
	"TDOR": "Original release time",
	"TDRC": "Recording time",
	"TDRL": "Release time",
	"TDTG": "Tagging time",
	"TENC": "Encoded by",
	"TEXT": "Lyricist/Text writer",
	"TFLT": "File type",
	"TIPL": "Involved people list",
	"TIT1": "Content group description",
	"TIT2": "Title/songname/content description",
	"TIT3": "Subtitle/Description refinement",
	"TKEY": "Initial key",
	"TLAN": "Language(s)",
	"TLEN": "Length",
	"TMCL": "Musician credits list",
	"TMED": "Media type",
	"TMOO": "Mood",
	"TOAL": "Original album/movie/show title",
	"TOFN": "Original filename",
	"TOLY": "Original lyricist(s)/text writer(s)",
	"TOPE": "Original artist(s)/performer(s)",
	"TOWN": "File owner/licensee",
	"TPE1": "Lead performer(s)/Soloist(s)",
	"TPE2": "Band/orchestra/accompaniment",
	"TPE3": "Conductor/performer refinement",
	"TPE4": "Interpreted, remixed, or otherwise modified by",
	"TPOS": "Part of a set",
	"TPRO": "Produced notice",
	"TPUB": "Publisher",
	"TRCK": "Track number/Position in set",
	"TRSN": "Internet radio station name",
	"TRSO": "Internet radio station owner",
	"TSO2": "iTunes uses this for Album Artist sort order",
	"TSOA": "Album sort order",
	"TSOC": "iTunes uses this for Composer sort order",
	"TSOP": "Performer sort order",
	"TSOT": "Title sort order",
	"TSRC": "ISRC (international standard recording code)",
	"TSSE": "Software/Hardware and settings used for encoding",
	"TSST": "Set subtitle",
	"TXXX": "User defined text information frame",

	"UFID": "Unique file identifier",
	"USER": "Terms of use",
	"USLT": "Unsynchronised lyric/text transcription",

	"WCOM": "Commercial information",
	"WCOP": "Copyright/Legal information",
	"WOAF": "Official audio file webpage",
	"WOAR": "Official artist/performer webpage",
	"WOAS": "Official audio source webpage",
	"WORS": "Official Internet radio station homepage",
	"WPAY": "Payment",
	"WPUB": "Publishers official webpage",
	"WXXX": "User defined URL link frame",
}

// ID3 frames that are defined in the specs.
var id3Frames = map[Format]map[string]string{
	ID3v2_2: id3v22Frames,
	ID3v2_3: id3v23Frames,
	ID3v2_4: id3v24Frames,
}

func validID3Frame(version Format, name string) bool {
	names, ok := id3Frames[version]
	if !ok {
		return false
	}
	_, ok = names[name]
	return ok
}

func readWFrame(b []byte) (string, error) {
	// Frame text is always encoded in ISO-8859-1
	b = append([]byte{0}, b...)
	return readTFrame(b)
}

func readTFrame(b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}

	txt, err := decodeText(b[0], b[1:])
	if err != nil {
		return "", err
	}
	return strings.Join(strings.Split(txt, string(singleZero)), ""), nil
}

const (
	encodingISO8859      byte = 0
	encodingUTF16WithBOM byte = 1
	encodingUTF16        byte = 2
	encodingUTF8         byte = 3
)

func decodeText(enc byte, b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}

	switch enc {
	case encodingISO8859: // ISO-8859-1
		return decodeISO8859(b), nil

	case encodingUTF16WithBOM: // UTF-16 with byte order marker
		if len(b) == 1 {
			return "", nil
		}
		return decodeUTF16WithBOM(b)

	case encodingUTF16: // UTF-16 without byte order (assuming BigEndian)
		if len(b) == 1 {
			return "", nil
		}
		return decodeUTF16(b, binary.BigEndian)

	case encodingUTF8: // UTF-8
		return string(b), nil

	default: // Fallback to ISO-8859-1
		return decodeISO8859(b), nil
	}
}

var (
	singleZero = []byte{0}
	doubleZero = []byte{0, 0}
)

func dataSplit(b []byte, enc byte) [][]byte {
	delim := singleZero
	if enc == encodingUTF16 || enc == encodingUTF16WithBOM {
		delim = doubleZero
	}

	result := bytes.SplitN(b, delim, 2)
	if len(result) != 2 {
		return result
	}

	if len(result[1]) == 0 {
		return result
	}

	if result[1][0] == 0 {
		// there was a double (or triple) 0 and we cut too early
		result[0] = append(result[0], result[1][0])
		result[1] = result[1][1:]
	}
	return result
}

func decodeISO8859(b []byte) string {
	r := make([]rune, len(b))
	for i, x := range b {
		r[i] = rune(x)
	}
	return string(r)
}

func decodeUTF16WithBOM(b []byte) (string, error) {
	if len(b) < 2 {
		return "", errors.New("invalid encoding: expected at least 2 bytes for UTF-16 byte order mark")
	}

	var bo binary.ByteOrder
	switch {
	case b[0] == 0xFE && b[1] == 0xFF:
		bo = binary.BigEndian
		b = b[2:]

	case b[0] == 0xFF && b[1] == 0xFE:
		bo = binary.LittleEndian
		b = b[2:]

	default:
		bo = DefaultUTF16WithBOMByteOrder
	}
	return decodeUTF16(b, bo)
}

func decodeUTF16(b []byte, bo binary.ByteOrder) (string, error) {
	if len(b)%2 != 0 {
		return "", errors.New("invalid encoding: expected even number of bytes for UTF-16 encoded text")
	}
	s := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		s = append(s, bo.Uint16(b[i:i+2]))
	}
	return string(utf16.Decode(s)), nil
}

// Comm is a type used in COMM, UFID, TXXX, WXXX and USLT tag.
// It's a text with a description and a specified language
// For WXXX, TXXX and UFID, we don't set a Language
type Comm struct {
	Language    string
	Description string
	Text        string
}

// String returns a string representation of the underlying Comm instance.
func (t Comm) String() string {
	if t.Language != "" {
		return fmt.Sprintf("Text{Lang: '%v', Description: '%v', %v lines}",
			t.Language, t.Description, strings.Count(t.Text, "\n"))
	}
	return fmt.Sprintf("Text{Description: '%v', %v}", t.Description, t.Text)
}

// IDv2.{3,4}
// -- Header
// <Header for 'Unsynchronised lyrics/text transcription', ID: "USLT">
// <Header for 'Comment', ID: "COMM">
// -- readTextWithDescrFrame(data, true, true)
// Text encoding       $xx
// Language            $xx xx xx
// Content descriptor  <text string according to encoding> $00 (00)
// Lyrics/text         <full text string according to encoding>
// -- Header
// <Header for         'User defined text information frame', ID: "TXXX">
// <Header for         'User defined URL link frame', ID: "WXXX">
// -- readTextWithDescrFrame(data, false, <isDataEncoded>)
// Text encoding       $xx
// Description         <text string according to encoding> $00 (00)
// Value               <text string according to encoding>
func readTextWithDescrFrame(b []byte, hasLang bool, encoded bool) (*Comm, error) {
	if len(b) == 0 {
		return nil, errors.New("error decoding tag description text: invalid encoding")
	}
	enc := b[0]
	b = b[1:]

	c := &Comm{}
	if hasLang {
		if len(b) < 3 {
			return nil, errors.New("hasLang set but not enough data for language information")
		}
		c.Language = string(b[:3])
		b = b[3:]
	}

	descTextSplit := dataSplit(b, enc)
	if len(descTextSplit) == 0 {
		return nil, errors.New("error decoding tag description text: invalid encoding")
	}

	desc, err := decodeText(enc, descTextSplit[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding tag description text: %v", err)
	}
	c.Description = desc

	if len(descTextSplit) == 1 {
		return c, nil
	}

	if !encoded {
		enc = byte(0)
	}
	text, err := decodeText(enc, descTextSplit[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding tag text: %v", err)
	}
	c.Text = text

	return c, nil
}

// UFID is composed of a provider (frequently a URL and a binary identifier)
// The identifier can be a text (Musicbrainz use texts, but not necessary)
type UFID struct {
	Provider   string
	Identifier []byte
}

func (u UFID) String() string {
	return fmt.Sprintf("%v (%v)", u.Provider, string(u.Identifier))
}

func readUFID(b []byte) (*UFID, error) {
	result := bytes.SplitN(b, singleZero, 2)
	if len(result) != 2 {
		return nil, errors.New("expected to split UFID data into 2 pieces")
	}

	return &UFID{
		Provider:   string(result[0]),
		Identifier: result[1],
	}, nil
}

var pictureTypes = map[byte]string{
	0x00: "Other",
	0x01: "32x32 pixels 'file icon' (PNG only)",
	0x02: "Other file icon",
	0x03: "Cover (front)",
	0x04: "Cover (back)",
	0x05: "Leaflet page",
	0x06: "Media (e.g. lable side of CD)",
	0x07: "Lead artist/lead performer/soloist",
	0x08: "Artist/performer",
	0x09: "Conductor",
	0x0A: "Band/Orchestra",
	0x0B: "Composer",
	0x0C: "Lyricist/text writer",
	0x0D: "Recording Location",
	0x0E: "During recording",
	0x0F: "During performance",
	0x10: "Movie/video screen capture",
	0x11: "A bright coloured fish",
	0x12: "Illustration",
	0x13: "Band/artist logotype",
	0x14: "Publisher/Studio logotype",
}

// Picture is a type which represents an attached picture extracted from metadata.
type Picture struct {
	Ext         string // Extension of the picture file.
	MIMEType    string // MIMEType of the picture.
	Type        string // Type of the picture (see pictureTypes).
	RawType     byte   // Raw picture type byte (see pictureTypes); 0x03 is the front cover.
	Description string // Description.
	Data        []byte // Raw picture data.
}

// String returns a string representation of the underlying Picture instance.
func (p Picture) String() string {
	return fmt.Sprintf("Picture{Ext: %v, MIMEType: %v, Type: %v, Description: %v, Data.Size: %v}",
		p.Ext, p.MIMEType, p.Type, p.Description, len(p.Data))
}

// IDv2.2
// -- Header
// Attached picture   "PIC"
// Frame size         $xx xx xx
// -- readPICFrame
// Text encoding      $xx
// Image format       $xx xx xx
// Picture type       $xx
// Description        <textstring> $00 (00)
// Picture data       <binary data>
func readPICFrame(b []byte) (*Picture, error) {
	if len(b) < 5 {
		return nil, errors.New("invalid PIC frame")
	}

	enc := b[0]
	ext := string(b[1:4])
	picType := b[4]

	descDataSplit := dataSplit(b[5:], enc)
	if len(descDataSplit) != 2 {
		return nil, errors.New("error decoding PIC description text: invalid encoding")
	}
	desc, err := decodeText(enc, descDataSplit[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding PIC description text: %v", err)
	}

	var mimeType string
	switch ext {
	case "jpeg", "jpg":
		mimeType = "image/jpeg"
	case "png":
		mimeType = "image/png"
	}

	return &Picture{
		Ext:         ext,
		MIMEType:    mimeType,
		Type:        pictureTypes[picType],
		RawType:     picType,
		Description: desc,
		Data:        descDataSplit[1],
	}, nil
}

// IDv2.{3,4}
// -- Header
// <Header for 'Attached picture', ID: "APIC">
// -- readAPICFrame
// Text encoding   $xx
// MIME type       <text string> $00
// Picture type    $xx
// Description     <text string according to encoding> $00 (00)
// Picture data    <binary data>
func readAPICFrame(b []byte) (*Picture, error) {
	if len(b) == 0 {
		return nil, errors.New("error decoding APIC: invalid encoding")
	}
	enc := b[0]
	mimeDataSplit := bytes.SplitN(b[1:], singleZero, 2)
	if len(mimeDataSplit) != 2 {
		return nil, errors.New("error decoding APIC: invalid encoding")
	}

	mimeType := string(mimeDataSplit[0])

	b = mimeDataSplit[1]
	if len(b) < 1 {
		return nil, fmt.Errorf("error decoding APIC mimetype")
	}
	picType := b[0]

	descDataSplit := dataSplit(b[1:], enc)
	if len(descDataSplit) != 2 {
		return nil, errors.New("error decoding APIC description text: invalid encoding")
	}
	desc, err := decodeText(enc, descDataSplit[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding APIC description text: %v", err)
	}

	var ext string
	switch mimeType {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	}

	return &Picture{
		Ext:         ext,
		MIMEType:    mimeType,
		Type:        pictureTypes[picType],
		RawType:     picType,
		Description: desc,
		Data:        descDataSplit[1],
	}, nil
}
