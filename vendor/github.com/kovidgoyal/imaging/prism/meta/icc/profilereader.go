package icc

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/kovidgoyal/go-parallel"
)

var _ = fmt.Println
var _ = os.Stderr

type ProfileReader struct {
	reader io.Reader
}

func (pr *ProfileReader) ReadProfile() (p *Profile, err error) {
	defer func() {
		if r := recover(); r != nil {
			p = nil
			err = parallel.Format_stacktrace_on_panic(r, 1)
		}
	}()

	profile := newProfile()

	err = pr.readHeader(&profile.Header)
	if err != nil {
		return nil, fmt.Errorf("failed to reader header from ICC profile: %w", err)
	}
	profile.PCSIlluminant = profile.Header.ParsedPCSIlluminant()

	err = pr.readTagTable(&profile.TagTable)
	if err != nil {
		return nil, fmt.Errorf("failed to read tag table from ICC profile: %w", err)
	}

	return profile, nil
}

func (pr *ProfileReader) readHeader(header *Header) (err error) {
	var data [128]byte
	if _, err = io.ReadFull(pr.reader, data[:]); err == nil {
		var n int
		n, err = binary.Decode(data[:], binary.BigEndian, header)
		if err == nil {
			if header.FileSignature != ProfileFileSignature {
				return fmt.Errorf("ICC header has invalid signature: %s", header.FileSignature)
			}
			if n != len(data) {
				return fmt.Errorf("decoding header consumed %d instead of %d bytes", n, len(data))
			}
			if header.ProfileConnectionSpace != ColorSpaceXYZ && header.ProfileConnectionSpace != ColorSpaceLab {
				return fmt.Errorf("unsupported profile connection space colorspace: %s", header.ProfileConnectionSpace)
			}
			if header.DataColorSpace != ColorSpaceRGB && header.DataColorSpace != ColorSpaceCMYK {
				return fmt.Errorf("unsupported device colorspace: %s", header.DataColorSpace)
			}
		}
	}
	return
}

func (pr *ProfileReader) readTagTable(tagTable *TagTable) (err error) {
	var tagCount uint32
	if err = binary.Read(pr.reader, binary.BigEndian, &tagCount); err != nil {
		return
	}
	type tagIndexEntry struct {
		Sig    Signature
		Offset uint32
		Size   uint32
	}
	endOfTagData := uint32(0)
	tag_indices := make([]tagIndexEntry, tagCount)
	if err = binary.Read(pr.reader, binary.BigEndian, tag_indices); err != nil {
		return fmt.Errorf("failed to read tag indices from ICC profile: %w", err)
	}
	for _, t := range tag_indices {
		endOfTagData = max(endOfTagData, t.Offset+t.Size)
	}
	tagDataOffset := 132 + tagCount*12
	if endOfTagData > tagDataOffset {
		tagData := make([]byte, endOfTagData-tagDataOffset)
		if _, err = io.ReadFull(pr.reader, tagData); err != nil {
			return fmt.Errorf("failed to read tag data from ICC profile: %w", err)
		}
		for _, t := range tag_indices {
			startOffset := t.Offset - tagDataOffset
			endOffset := startOffset + t.Size
			tagTable.add(t.Sig, int(startOffset), tagData[startOffset:endOffset])
		}
	}

	return nil
}

func NewProfileReader(r io.Reader) *ProfileReader {
	return &ProfileReader{
		reader: r,
	}
}
