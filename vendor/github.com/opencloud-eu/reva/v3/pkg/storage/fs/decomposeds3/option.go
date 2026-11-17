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

package decomposeds3

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {

	// Endpoint of the s3 blobstore
	S3Endpoint string `mapstructure:"s3.endpoint"`

	// Region of the s3 blobstore
	S3Region string `mapstructure:"s3.region"`

	// Bucket of the s3 blobstore
	S3Bucket string `mapstructure:"s3.bucket"`

	// Access key for the s3 blobstore
	S3AccessKey string `mapstructure:"s3.access_key"`

	// Secret key for the s3 blobstore
	S3SecretKey string `mapstructure:"s3.secret_key"`

	// disable sending content sha256
	DisableContentSha256 bool `mapstructure:"s3.disable_content_sha254"`

	// disable multipart uploads
	DisableMultipart bool `mapstructure:"s3.disable_multipart"`

	// enable sending content md5, defaults to true if unset
	SendContentMd5 bool `mapstructure:"s3.send_content_md5"`

	// use concurrent stream parts
	ConcurrentStreamParts bool `mapstructure:"s3.concurrent_stream_parts"`

	// number of concurrent uploads
	NumThreads uint `mapstructure:"s3.num_threads"`

	// part size for concurrent uploads
	PartSize uint64 `mapstructure:"s3.part_size"`
}

// S3ConfigComplete return true if all required s3 fields are set
func (o *Options) S3ConfigComplete() bool {
	return o.S3Endpoint != "" &&
		o.S3Region != "" &&
		o.S3Bucket != "" &&
		o.S3AccessKey != "" &&
		o.S3SecretKey != ""
}

func parseConfig(m map[string]interface{}) (*Options, error) {
	o := &Options{}
	if err := mapstructure.Decode(m, o); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}

	// if unset we set these defaults
	if m["s3.send_content_md5"] == nil {
		o.SendContentMd5 = true
	}
	if m["s3.concurrent_stream_parts"] == nil {
		o.ConcurrentStreamParts = true
	}
	if m["s3.num_threads"] == nil {
		o.NumThreads = 4
	}
	return o, nil
}
