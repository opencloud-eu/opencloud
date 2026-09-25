/*
 * Copyright 2026 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package samlext

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

// maxDecompressedMessageSize is the maximal amount of data which is
// decompressed from a single compressed SAML message. It matches the limit
// which the upstream crewjam/saml library applies to the messages it
// decompresses itself.
const maxDecompressedMessageSize = 10 * 1024 * 1024 // 10MB

// inflateMessage decompresses the provided raw DEFLATE data as used by the
// SAML HTTP-Redirect binding. Data which decompresses to more than
// maxDecompressedMessageSize bytes is rejected instead of decompressed, so
// compressed messages cannot allocate unbounded memory.
func inflateMessage(compressed []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()

	// Read one byte more than allowed, to tell oversized data apart from data
	// which just reaches the limit.
	decompressed, err := io.ReadAll(io.LimitReader(r, maxDecompressedMessageSize+1))
	if err != nil {
		return nil, err
	}
	if len(decompressed) > maxDecompressedMessageSize {
		return nil, fmt.Errorf("uncompress limit exceeded (%d bytes)", maxDecompressedMessageSize)
	}

	return decompressed, nil
}
