// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestS3CacheSSECValidation(t *testing.T) {
	validKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{42}, 32))
	for _, tc := range []struct {
		name, key, endpoint string
		valid               bool
	}{
		{"unset", "", "http://localhost:9000", true},
		{"valid", validKey, "https://localhost:9000", true},
		{"bare host", validKey, "localhost:9000", true},
		{"HTTP", validKey, "http://localhost:9000", false},
		{"base64", "secret!", "https://localhost:9000", false},
		{"length", base64.StdEncoding.EncodeToString([]byte("short")), "https://localhost:9000", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := BuildS3CacheConfig("test-bucket", "us-east-1", tc.endpoint, "ak", "sk", tc.key)
			_, err := NewThumbnailCache("s3", "", cfg)
			if (err == nil) != tc.valid {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), tc.key) {
				t.Fatal("error exposes key")
			}
			if !tc.valid {
				c := NewS3Cache(*cfg)
				c.newFunc = func(S3CacheConfig) (s3Client, error) {
					t.Fatal("invalid configuration initialized a client")
					return nil, nil
				}
				if err := c.Put("key", []byte("data")); err == nil {
					t.Fatal("Put accepted invalid configuration")
				}
				if _, err := c.Get("key"); err == nil {
					t.Fatal("Get accepted invalid configuration")
				}
			}
		})
	}
	cfg := BuildS3CacheConfig("test-bucket", "us-east-1", "localhost:9000", "ak", "sk", validKey)
	cfg.Secure = false
	if _, err := NewThumbnailCache("s3", "", cfg); err == nil {
		t.Fatal("accepted insecure bare endpoint")
	}
	// An unused S3 configuration must not prevent a different backend starting.
	cfg.SSECKey = "invalid!"
	if _, err := NewThumbnailCache("memory", "", cfg); err != nil {
		t.Fatal(err)
	}
}

func TestS3CacheSSECRoundTrip(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		t.Run(fmt.Sprintf("encrypted=%t", encrypted), func(t *testing.T) {
			var key string
			if encrypted {
				key = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{42}, 32))
			}
			var mu sync.Mutex
			var data []byte
			requests := map[string]int{}
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests[r.Method]++
				gotKey := r.Header.Get("X-Amz-Server-Side-Encryption-Customer-Key")
				if gotKey != key {
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, "<Error><Code>AccessDenied</Code><Message>Invalid customer key</Message></Error>")
					return
				}
				if encrypted {
					decoded, _ := base64.StdEncoding.DecodeString(key)
					sum := md5.Sum(decoded)
					if r.Header.Get("X-Amz-Server-Side-Encryption-Customer-Algorithm") != "AES256" || r.Header.Get("X-Amz-Server-Side-Encryption-Customer-Key-MD5") != base64.StdEncoding.EncodeToString(sum[:]) {
						t.Error("invalid SSE-C headers")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				} else if r.Header.Get("X-Amz-Server-Side-Encryption-Customer-Algorithm") != "" {
					t.Error("unset key sends encryption headers")
				}
				w.Header().Set("ETag", `"etag"`)
				w.Header().Set("Last-Modified", "Mon, 26 Oct 2026 12:00:00 GMT")
				switch r.Method {
				case http.MethodPut:
					data, _ = io.ReadAll(r.Body)
				case http.MethodGet:
					w.Header().Set("Content-Length", strconv.Itoa(len(data)))
					w.Write(data)
				case http.MethodHead:
					w.Header().Set("Content-Length", strconv.Itoa(len(data)))
				default:
					t.Errorf("unexpected S3 method %s", r.Method)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer srv.Close()
			cert := filepath.Join(t.TempDir(), "ca.pem")
			if err := os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("SSL_CERT_FILE", cert)
			cfg := BuildS3CacheConfig("test-bucket", "us-east-1", srv.URL, "ak", "sk", key)
			c, err := NewThumbnailCache("s3", "", cfg)
			if err != nil {
				t.Fatal(err)
			}
			want := []byte("thumbnail-data")
			if err := c.Put("preview.jpg", want); err != nil {
				t.Fatal(err)
			}
			got, err := c.Get("preview.jpg")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("round trip returned %q", got)
			}
			if encrypted {
				for _, badKey := range []string{"", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))} {
					cfg.SSECKey = badKey
					bad, err := NewThumbnailCache("s3", "", cfg)
					if err != nil {
						t.Fatal(err)
					}
					mu.Lock()
					before := requests[http.MethodGet]
					mu.Unlock()
					if _, err := bad.Get("preview.jpg"); err == nil || err == ErrCacheMiss {
						t.Fatalf("wrong/missing key did not surface as error: %v", err)
					}
					mu.Lock()
					after := requests[http.MethodGet]
					mu.Unlock()
					if after != before+1 {
						t.Fatal("failed SSE-C read was retried")
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if requests[http.MethodPut] != 1 || requests[http.MethodGet] == 0 {
				t.Fatalf("unexpected requests: %v", requests)
			}
		})
	}
}
