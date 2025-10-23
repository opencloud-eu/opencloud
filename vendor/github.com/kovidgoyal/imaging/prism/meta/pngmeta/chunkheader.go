package pngmeta

import (
	"encoding/binary"
	"fmt"
	"io"
)

type chunkHeader struct {
	Length    uint32
	ChunkType [4]byte
}

func (ch chunkHeader) String() string {
	return fmt.Sprintf("%c%c%c%c(%d)", ch.ChunkType[0], ch.ChunkType[1], ch.ChunkType[2], ch.ChunkType[3], ch.Length)
}

func readChunkHeader(r io.Reader) (ch chunkHeader, err error) {
	err = binary.Read(r, binary.BigEndian, &ch)
	return
}
