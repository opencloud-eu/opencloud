//  Copyright (c) 2026 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.  See the License for the specific language governing
// permissions and limitations under the License.

package zap

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	index "github.com/blevesearch/bleve_index_api"
)

// This file provides a mechanism for users of zap to provide callbacks
// that can process data before it is written to disk, and after it is read
// from disk.  This can be used for things like encryption, compression, etc.

// The user is responsible for ensuring that the writer and reader callbacks
// are compatible with each other, and that any state needed by the callbacks
// is managed appropriately.  For example, if the writer callback uses a
// unique key or nonce per write, the reader callback must be able to
// determine the correct key or nonce to use for each read.

// The callbacks are identified by an id string, which is returned by the
// WriterCallbackGetter. The same id string is passed to the ReaderCallbackGetter
// when creating a reader.  This allows the reader to determine which
// callback to use for a given file.

// An example implementation using AES-GCM encryption is provided in
// file_callbacks_test.go within initFileCallbacks().

// the default id to use for file callbacks that indicates a no op
const DefaultFileCallbackId = ""

// isLittleEndian reports whether the running architecture stores multi-byte
// integers least-significant-byte first. WriteUint64Array always writes
// little-endian on disk, so on a little-endian host ReadUint64Array can
// return a zero-copy view of the mmap'd file rather than decoding a copy.
// On any other host we default to decoding a copy.
var isLittleEndian = func() bool {
	var x uint16 = 1
	return *(*byte)(unsafe.Pointer(&x)) == 1
}()

// alignPad is zero-filled padding written before an aligned array's payload.
var alignPad [8]byte

// FileWriter wraps a CountHashWriter and applies a user provided
// writer callback to the data being written.
type FileWriter struct {
	id         string
	c          *CountHashWriter
	varintBuf  []byte // Reusable buffer for varint length prefixes
	payloadBuf []byte // Reusable buffer for encoding array payloads
	processor  func(data []byte) []byte
}

// creates an empty FileWriter with no callback. Used
// when we are writing data that is not going to be persisted
func NewFileWriterEmpty(c *CountHashWriter) *FileWriter {
	rv := &FileWriter{
		c: c,
	}

	return rv
}

// NewFileWriter creates a FileWriter with the provided CountHashWriter and applies
// the writer callback identified by the context.
func NewFileWriter(c *CountHashWriter, context []byte) (*FileWriter, error) {
	rv := &FileWriter{
		c: c,
	}

	if index.WriterHook != nil {
		var err error
		rv.id, rv.processor, err = index.WriterHook(context)
		if err != nil {
			return nil, err
		}
	}

	return rv, nil
}

func (w *FileWriter) Write(data []byte) (int, error) {
	return w.c.Write(data)
}

// process applies the writer callback to the data, if one is set
func (w *FileWriter) process(data []byte) []byte {
	if w.processor != nil {
		return w.processor(data)
	}
	return data
}

func (w *FileWriter) Count() int {
	return w.c.Count()
}

func (w *FileWriter) Sum32() uint32 {
	return w.c.Sum32()
}

func (w *FileWriter) grabVarintBuf(size int) []byte {
	if cap(w.varintBuf) < size {
		w.varintBuf = make([]byte, size)
	}
	return w.varintBuf[:size]
}

func (w *FileWriter) grabPayloadBuf(size int) []byte {
	if cap(w.payloadBuf) < size {
		w.payloadBuf = make([]byte, size)
	}
	return w.payloadBuf[:size]
}

// WriteUint64Array writes arr as a length-prefixed array of little-endian
// uint64 values, padded so the payload begins on an 8-byte boundary in the
// file. Padding is required for zero-copy reads because of go's
// unsafe.Pointer conversion rules.
func (w *FileWriter) WriteUint64Array(arr []uint64) (int, error) {
	// encode the array as a contiguous slice of bytes, little-endian.
	buf := w.grabPayloadBuf(len(arr) * 8)
	for i, v := range arr {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], v)
	}
	buf = w.process(buf)

	// write the length of the array as a varint
	numBuf := w.grabVarintBuf(binary.MaxVarintLen64)
	n := binary.PutUvarint(numBuf, uint64(len(buf)))
	total, err := w.Write(numBuf[:n])
	if err != nil {
		return total, err
	}

	// pad so buf starts on an 8-byte boundary in the file, and write the
	// padding length first so the reader can skip the padding. The +1
	// accounts for the pad-length byte itself, which is written before
	// the padding.
	pad := byte((8 - (uint64(w.Count())+1)%8) % 8)
	written, err := w.Write([]byte{pad})
	total += written
	if err != nil {
		return total, err
	}
	if pad > 0 {
		// write the actual padding bytes, which are always zero
		written, err = w.Write(alignPad[:pad])
		total += written
		if err != nil {
			return total, err
		}
	}

	// write the actual array bytes
	written, err = w.Write(buf)
	total += written
	return total, err
}

// WriteUint32Array writes arr as a length-prefixed array of little-endian
// uint32 values, padded so the payload begins on a 4-byte boundary in the
// file. Padding is required for the zero-copy reads in ReadUint32Array because
// of go's unsafe.Pointer conversion rules.
func (w *FileWriter) WriteUint32Array(arr []uint32) (int, error) {
	// encode the array as a contiguous slice of bytes, little-endian.
	buf := w.grabPayloadBuf(len(arr) * 4)
	for i, v := range arr {
		binary.LittleEndian.PutUint32(buf[i*4:(i+1)*4], v)
	}
	buf = w.process(buf)

	// write the length of the array as a varint
	numBuf := w.grabVarintBuf(binary.MaxVarintLen64)
	n := binary.PutUvarint(numBuf, uint64(len(buf)))
	total, err := w.Write(numBuf[:n])
	if err != nil {
		return total, err
	}

	// pad so buf starts on a 4-byte boundary in the file, and write the
	// padding length first so the reader can skip the padding. The +1
	// accounts for the pad-length byte itself, which is written before
	// the padding.
	pad := byte((4 - (uint64(w.Count())+1)%4) % 4)
	written, err := w.Write([]byte{pad})
	total += written
	if err != nil {
		return total, err
	}
	if pad > 0 {
		// write the actual padding bytes, which are always zero
		written, err = w.Write(alignPad[:pad])
		total += written
		if err != nil {
			return total, err
		}
	}

	// write the actual array bytes
	written, err = w.Write(buf)
	total += written
	return total, err
}

// WriteArrayWithOffsets writes a slice of byte slices as a length-prefixed
// array of offsets, followed by the concatenated payloads. Each offset is
// the end position of the corresponding payload in the concatenated buffer.
func (w *FileWriter) WriteArrayWithOffsets(arr [][]byte) (int, error) {
	offsets := make([]uint64, len(arr))

	// preallocate the payload buffer from the summed input lengths so it is
	// not repeatedly reallocated as payloads are appended.
	var payloadLen int
	for _, a := range arr {
		payloadLen += len(a)
	}
	buf := make([]byte, 0, payloadLen)

	for i, a := range arr {
		a = w.process(a)
		buf = append(buf, a...)
		offsets[i] = uint64(len(buf))
	}

	// write the offsets as a length-prefixed array of uint64 values
	total, err := w.WriteUint64Array(offsets)
	if err != nil {
		return total, err
	}

	// write the concatenated payloads as a length-prefixed byte slice
	numBuf := w.grabVarintBuf(binary.MaxVarintLen64)
	n := binary.PutUvarint(numBuf, uint64(len(buf)))
	written, err := w.Write(numBuf[:n])
	total += written
	if err != nil {
		return total, err
	}

	// write the actual concatenated payloads
	written, err = w.Write(buf)
	total += written
	return total, err
}

// FileReader wraps a reader callback to be applied to data read from a file.
type FileReader struct {
	id        string
	processor func(data []byte) ([]byte, error)
}

// NewFileReader creates a FileReader with the reader callback identified by the context.
// The id is used to identify which callback to use when reading data.
func NewFileReader(id string, context []byte) (*FileReader, error) {
	rv := &FileReader{
		id: id,
	}

	if index.ReaderHook != nil {
		var err error
		rv.processor, err = index.ReaderHook(id, context)
		if err != nil {
			return nil, err
		}
	} else if id != DefaultFileCallbackId {
		return nil, fmt.Errorf("reader callback id %s provided but no ReaderHook is set", id)
	}

	return rv, nil
}

// process applies the reader callback to the data, if one is set
func (r *FileReader) process(data []byte) ([]byte, error) {
	if r.processor != nil {
		return r.processor(data)
	}
	return data, nil
}

// ReadUint64Array reads an array written by WriteUint64Array and returns its
// values, along with the raw byte buffer they were decoded from (mem) when
// that buffer is a zero-copy view worth retaining - nil otherwise.
// Callers must treat both vals and mem as read-only.
func (r *FileReader) ReadUint64Array(data []byte) (vals []uint64, mem []byte, shift uint64, err error) {
	var pos uint64
	// read the length of the array as a varint
	bufLen, n := binary.Uvarint(data[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)

	// read the padding length and skip over the padding bytes
	pad := data[pos]
	pos += 1 + uint64(pad)

	if bufLen == 0 {
		return nil, nil, pos, nil
	}

	// read the actual array bytes and apply the reader callback
	src := data[pos : pos+bufLen]
	buf, err := r.process(src)
	if err != nil {
		return nil, nil, 0, err
	}
	pos += bufLen

	// if the host is little-endian and the processed buffer is the same length as
	// the original buffer and has the same backing array, we can return a zero-copy
	// view of the buffer as a []uint64 slice.
	if isLittleEndian && len(buf) == len(src) && unsafe.SliceData(buf) == unsafe.SliceData(src) &&
		uintptr(unsafe.Pointer(&buf[0]))%8 == 0 {
		return unsafe.Slice((*uint64)(unsafe.Pointer(&buf[0])), len(buf)/8), buf, pos, nil
	}

	// decode the buffer into a new []uint64 slice
	vals = make([]uint64, len(buf)/8)
	for i := range vals {
		vals[i] = binary.LittleEndian.Uint64(buf[i*8 : (i+1)*8])
	}

	return vals, nil, pos, nil
}

// ReadUint32Array reads an array written by WriteUint32Array and returns its
// values, along with the raw byte buffer they were decoded from (mem) when
// that buffer is a zero-copy view worth retaining - nil otherwise.
// Callers must treat both vals and mem as read-only.
func (r *FileReader) ReadUint32Array(data []byte) (vals []uint32, mem []byte, shift uint64, err error) {
	var pos uint64

	// read the length of the array as a varint
	bufLen, n := binary.Uvarint(data[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)

	// read the padding length and skip over the padding bytes
	pad := data[pos]
	pos += 1 + uint64(pad)

	if bufLen == 0 {
		return nil, nil, pos, nil
	}

	// read the actual array bytes and apply the reader callback
	src := data[pos : pos+bufLen]
	buf, err := r.process(src)
	if err != nil {
		return nil, nil, 0, err
	}
	pos += bufLen

	// if the host is little-endian and the processed buffer is the same length
	// as the original buffer and has the same backing array, we can return a
	// zero-copy view of the buffer as a []uint32 slice.
	if isLittleEndian && len(buf) == len(src) && unsafe.SliceData(buf) == unsafe.SliceData(src) &&
		uintptr(unsafe.Pointer(&buf[0]))%4 == 0 {
		return unsafe.Slice((*uint32)(unsafe.Pointer(&buf[0])), len(buf)/4), buf, pos, nil
	}

	// decode the buffer into a new []uint32 slice
	vals = make([]uint32, len(buf)/4)
	for i := range vals {
		vals[i] = binary.LittleEndian.Uint32(buf[i*4 : (i+1)*4])
	}

	return vals, nil, pos, nil
}

// ReadArrayWithOffsets reads an array written by WriteArrayWithOffsets and
// returns the individual payloads, each processed through the reader
// callback, along with the number of bytes consumed from data.
// Callers must treat the returned payloads as read-only.
func (r *FileReader) ReadArrayWithOffsets(data []byte) ([][]byte, uint64, error) {
	var pos uint64
	// read the offsets as a length-prefixed array of uint64 values
	offsets, _, shift, err := r.ReadUint64Array(data[pos:])
	if err != nil {
		return nil, 0, err
	}
	pos += shift

	// read the concatenated payloads as a length-prefixed byte slice
	dataLen, n := binary.Uvarint(data[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)
	if dataLen == 0 {
		return nil, 0, fmt.Errorf("read array length is 0")
	}
	rawData := data[pos : pos+dataLen]
	pos += dataLen

	// process the concatenated payloads with the reader callback
	arr := make([][]byte, len(offsets))
	for i := range offsets {
		var start uint64
		if i > 0 {
			start = offsets[i-1]
		}
		end := offsets[i]
		arr[i], err = r.process(rawData[start:end])
		if err != nil {
			return nil, 0, err
		}
	}

	return arr, pos, nil
}
