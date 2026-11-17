// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package chunking

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	chunkingPathRE = regexp.MustCompile(`-chunking-\w+-[0-9]+-[0-9]+$`)
)

// IsChunked checks if a given path refers to a chunk or not
func IsChunked(fn string) bool {
	// FIXME: also need to check whether the OC-Chunked header is set
	return chunkingPathRE.MatchString(fn)
}

// ChunkBLOBInfo stores info about a particular chunk
type ChunkBLOBInfo struct {
	Path         string
	TransferID   string
	TotalChunks  int
	CurrentChunk int
}

// Not using the resource path in the chunk folder name allows uploading to
// the same folder after a move without having to restart the chunk upload
func (c *ChunkBLOBInfo) uploadID() string {
	return fmt.Sprintf("chunking-%s-%d", c.TransferID, c.TotalChunks)
}

// GetChunkBLOBInfo decodes a chunk name to retrieve info about it.
func GetChunkBLOBInfo(path string) (*ChunkBLOBInfo, error) {
	parts := strings.Split(path, "-chunking-")
	tail := strings.Split(parts[1], "-")

	totalChunks, err := strconv.Atoi(tail[1])
	if err != nil {
		return nil, err
	}

	currentChunk, err := strconv.Atoi(tail[2])
	if err != nil {
		return nil, err
	}
	if currentChunk >= totalChunks {
		return nil, fmt.Errorf("current chunk:%d exceeds total number of chunks:%d", currentChunk, totalChunks)
	}

	return &ChunkBLOBInfo{
		Path:         parts[0],
		TransferID:   tail[0],
		TotalChunks:  totalChunks,
		CurrentChunk: currentChunk,
	}, nil
}

// ChunkHandler manages chunked uploads, storing the chunks in a temporary directory
// until it gets the final chunk which is then returned.
type ChunkHandler struct {
	ChunkFolder string `mapstructure:"chunk_folder"`
}

// NewChunkHandler creates a handler for chunked uploads.
func NewChunkHandler(chunkFolder string) *ChunkHandler {
	return &ChunkHandler{chunkFolder}
}

func (c *ChunkHandler) createChunkTempFile() (string, *os.File, error) {
	file, err := os.CreateTemp(fmt.Sprintf("/%s", c.ChunkFolder), "")
	if err != nil {
		return "", nil, err
	}

	return file.Name(), file, nil
}

func (c *ChunkHandler) getChunkFolderName(i *ChunkBLOBInfo) (string, error) {
	path := filepath.Join("/", c.ChunkFolder, filepath.Join("/", i.uploadID()))
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func (c *ChunkHandler) saveChunk(path string, r io.ReadCloser) (bool, string, error) {
	chunkInfo, err := GetChunkBLOBInfo(path)
	if err != nil {
		err := fmt.Errorf("error getting chunk info from path: %s", path)
		return false, "", err
	}

	chunkTempFilename, chunkTempFile, err := c.createChunkTempFile()
	if err != nil {
		return false, "", err
	}
	defer chunkTempFile.Close()

	if _, err := io.Copy(chunkTempFile, r); err != nil {
		return false, "", err
	}

	// force close of the file here because if it is the last chunk to
	// assemble the big file we must have all the chunks already closed.
	if err = chunkTempFile.Close(); err != nil {
		return false, "", err
	}

	chunksFolderName, err := c.getChunkFolderName(chunkInfo)
	if err != nil {
		return false, "", err
	}
	// c.logger.Info().Log("chunkfolder", chunksFolderName)

	chunkTarget := filepath.Join(chunksFolderName, strconv.Itoa(chunkInfo.CurrentChunk))
	if err = os.Rename(chunkTempFilename, chunkTarget); err != nil {
		return false, "", err
	}

	// Check that all chunks are uploaded.
	// This is very inefficient, the server has to check that it has all the
	// chunks after each uploaded chunk.
	// A two-phase upload like DropBox is better, because the server will
	// assembly the chunks when the client asks for it.
	chunksFolder, err := os.Open(chunksFolderName)
	if err != nil {
		return false, "", err
	}
	defer chunksFolder.Close()

	// read all the chunks inside the chunk folder; -1 == all
	chunks, err := chunksFolder.Readdir(-1)
	if err != nil {
		return false, "", err
	}

	// there are still some chunks to be uploaded.
	// we return CodeUploadIsPartial to notify upper layers that the upload is still
	// not complete and requires more actions.
	// This code is needed to notify the OpenCloud webservice that the upload has not yet been
	// completed and needs to continue uploading chunks.
	if len(chunks) < chunkInfo.TotalChunks {
		return false, "", nil
	}

	assembledFileName, assembledFile, err := c.createChunkTempFile()
	if err != nil {
		return false, "", err
	}
	defer assembledFile.Close()

	// walk all chunks and append to assembled file
	for i := range chunks {
		target := filepath.Join(chunksFolderName, strconv.Itoa(i))

		chunk, err := os.Open(target)
		if err != nil {
			return false, "", err
		}
		defer chunk.Close()

		if _, err = io.Copy(assembledFile, chunk); err != nil {
			return false, "", err
		}

		// we close the chunk here because if the assembled file contains hundreds of chunks
		// we will end up with hundreds of open file descriptors
		if err = chunk.Close(); err != nil {
			return false, "", err

		}
	}

	// at this point the assembled file is complete
	// so we free space removing the chunks folder
	defer os.RemoveAll(chunksFolderName)

	return true, assembledFileName, nil
}

// Assemble saves an intermediate chunk and, once all chunks have arrived,
// assembles them into a single temp file. Returns (reader, size, true, nil)
// when the transfer is complete. Returns (nil, 0, false, nil) when more
// chunks are still expected (partial transfer)
// the returned reader removes the temp file on Close
func (c *ChunkHandler) Assemble(chunkName string, body io.ReadCloser) (io.ReadCloser, int64, bool, error) {
	origPath, assembledPath, err := c.WriteChunk(chunkName, body)
	if err != nil {
		return nil, 0, false, err
	}
	if origPath == "" {
		return nil, 0, false, nil // partial
	}
	f, err := os.Open(assembledPath)
	if err != nil {
		_ = os.RemoveAll(assembledPath)
		return nil, 0, false, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		_ = os.RemoveAll(assembledPath)
		return nil, 0, false, err
	}
	return &assembledFile{File: f, path: assembledPath}, fi.Size(), true, nil
}

// assembledFile wraps an *os.File and removes the temp file on Close.
type assembledFile struct {
	*os.File
	path string
}

func (a *assembledFile) Close() error {
	err := a.File.Close()
	_ = os.RemoveAll(a.path)
	return err
}

// WriteChunk saves an intermediate chunk temporarily and assembles all chunks
// once the final one is received.
func (c *ChunkHandler) WriteChunk(fn string, r io.ReadCloser) (string, string, error) {
	finish, chunk, err := c.saveChunk(fn, r)
	if err != nil {
		return "", "", err
	}

	if !finish {
		return "", "", nil
	}

	chunkInfo, err := GetChunkBLOBInfo(fn)
	if err != nil {
		return "", "", err
	}

	return chunkInfo.Path, chunk, nil

	// TODO(labkode): implement old chunking

	/*
		req2 := &provider.StartWriteSessionRequest{}
		res2, err := client.StartWriteSession(ctx, req2)
		if err != nil {
			logger.Error(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if res2.Status.Code != rpc.Code_CODE_OK {
			logger.Println(ctx, res2.Status)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		sessID := res2.SessionId
		logger.Build().Str("sessID", sessID).Msg(ctx, "got write session id")

		stream, err := client.Write(ctx)
		if err != nil {
			logger.Error(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		buffer := make([]byte, 1024*1024*3)
		var offset uint64
		var numChunks uint64

		for {
			n, err := fd.Read(buffer)
			if n > 0 {
				req := &provider.WriteRequest{Data: buffer, Length: uint64(n), SessionId: sessID, Offset: offset}
				err = stream.Send(req)
				if err != nil {
					logger.Error(ctx, err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				numChunks++
				offset += uint64(n)
			}

			if err == io.EOF {
				break
			}

			if err != nil {
				logger.Error(ctx, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		res3, err := stream.CloseAndRecv()
		if err != nil {
			logger.Error(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if res3.Status.Code != rpc.Code_CODE_OK {
			logger.Println(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req4 := &provider.FinishWriteSessionRequest{Filename: chunkInfo.path, SessionId: sessID}
		res4, err := client.FinishWriteSession(ctx, req4)
		if err != nil {
			logger.Error(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if res4.Status.Code != rpc.Code_CODE_OK {
			logger.Println(ctx, res4.Status)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req.Filename = chunkInfo.path
		res, err = client.Stat(ctx, req)
		if err != nil {
			logger.Error(ctx, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if res.Status.Code != rpc.Code_CODE_OK {
			logger.Println(ctx, res.Status)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		md2 := res.Metadata

		w.Header().Add("Content-Type", md2.Mime)
		w.Header().Set("ETag", md2.Etag)
		w.Header().Set("OC-FileId", md2.Id)
		w.Header().Set("OC-ETag", md2.Etag)
		t := time.Unix(int64(md2.Mtime), 0)
		lastModifiedString := t.Format(time.RFC1123Z)
		w.Header().Set("Last-Modified", lastModifiedString)
		w.Header().Set("X-OC-MTime", "accepted")

		if md == nil {
			w.WriteHeader(http.StatusCreated)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	*/
}
