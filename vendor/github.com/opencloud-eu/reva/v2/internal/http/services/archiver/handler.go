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

package archiver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"regexp"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/gdexlab/go-render/render"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/internal/http/services/archiver/manager"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/downloader"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/walker"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/rs/zerolog"
)

type svc struct {
	config          *Config
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	log             *zerolog.Logger
	walker          walker.Walker
	downloader      downloader.Downloader

	allowedFolders []*regexp.Regexp
}

// Config holds the config options that need to be passed down to all ocdav handlers
type Config struct {
	Prefix         string   `mapstructure:"prefix"`
	GatewaySvc     string   `mapstructure:"gatewaysvc"`
	Timeout        int64    `mapstructure:"timeout"`
	Insecure       bool     `mapstructure:"insecure"`
	Name           string   `mapstructure:"name"`
	MaxNumFiles    int64    `mapstructure:"max_num_files"`
	MaxSize        int64    `mapstructure:"max_size"`
	AllowedFolders []string `mapstructure:"allowed_folders"`
}

func init() {
	global.Register("archiver", New)
}

// New creates a new archiver service
func New(conf map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	c := &Config{}
	err := mapstructure.Decode(conf, c)
	if err != nil {
		return nil, err
	}

	c.init()

	gatewaySelector, err := pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return nil, err
	}

	// compile all the regex for filtering folders
	allowedFolderRegex := make([]*regexp.Regexp, 0, len(c.AllowedFolders))
	for _, s := range c.AllowedFolders {
		regex, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		allowedFolderRegex = append(allowedFolderRegex, regex)
	}

	return &svc{
		config:          c,
		gatewaySelector: gatewaySelector,
		downloader:      downloader.NewDownloader(gatewaySelector, rhttp.Insecure(c.Insecure), rhttp.Timeout(time.Duration(c.Timeout*int64(time.Second)))),
		walker:          walker.NewWalker(gatewaySelector),
		log:             log,
		allowedFolders:  allowedFolderRegex,
	}, nil
}

func (c *Config) init() {
	if c.Prefix == "" {
		c.Prefix = "download_archive"
	}

	if c.Name == "" {
		c.Name = "download"
	}

	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)
}

func (s *svc) getResources(ctx context.Context, paths, ids []string) ([]*provider.ResourceId, error) {
	if len(paths) == 0 && len(ids) == 0 {
		return nil, errtypes.BadRequest("path and id lists are both empty")
	}

	resources := make([]*provider.ResourceId, 0, len(paths)+len(ids))

	for _, id := range ids {
		// id is base64 encoded and after decoding has the form <storage_id>:<resource_id>

		decodedID, err := storagespace.ParseID(id)
		if err != nil {
			return nil, errors.New("could not unwrap given file id")
		}

		resources = append(resources, &decodedID)

	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		// id is base64 encoded and after decoding has the form <storage_id>:<resource_id>

		resp, err := gatewayClient.Stat(ctx, &provider.StatRequest{
			Ref: &provider.Reference{
				Path: p,
			},
		})

		switch {
		case err != nil:
			return nil, err
		case resp.Status.Code == rpc.Code_CODE_NOT_FOUND:
			return nil, errtypes.NotFound(p)
		case resp.Status.Code != rpc.Code_CODE_OK:
			return nil, errtypes.InternalError(fmt.Sprintf("error stating %s", p))
		}

		resources = append(resources, resp.Info.Id)

	}

	// check if all the folders are allowed to be archived
	/* FIXME bring back filtering
	err := s.allAllowed(resources)
	if err != nil {
		return nil, err
	}
	*/

	return resources, nil
}

// return true if path match with at least with one allowed folder regex
/*
func (s *svc) isPathAllowed(path string) bool {
	for _, reg := range s.allowedFolders {
		if reg.MatchString(path) {
			return true
		}
	}
	return false
}

// return nil if all the paths in the slide match with at least one allowed folder regex
func (s *svc) allAllowed(paths []string) error {
	if len(s.allowedFolders) == 0 {
		return nil
	}

	for _, f := range paths {
		if !s.isPathAllowed(f) {
			return errtypes.BadRequest(fmt.Sprintf("resource at %s not allowed to be archived", f))
		}
	}
	return nil
}
*/

// resourceName resolves the name of a single resource so the archive can be named after it instead
// of the generic "download". It returns an empty string on any failure, so the caller keeps the
// default name. The name is sanitized via sanitizeArchiveName.
func (s *svc) resourceName(ctx context.Context, id *provider.ResourceId) (string, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		s.log.Debug().Err(err).Msg("archiver: could not select gateway to resolve the archive name, using the default")
		return "", err
	}

	res, err := gatewayClient.Stat(ctx, &provider.StatRequest{
		Ref: &provider.Reference{ResourceId: id},
	})
	if err != nil {
		s.log.Debug().Err(err).Msg("archiver: stat failed while resolving the archive name, using the default")
		return "", err
	}
	if code := res.GetStatus().GetCode(); code != rpc.Code_CODE_OK {
		s.log.Debug().Str("code", code.String()).Msg("archiver: stat returned non-OK while resolving the archive name, using the default")
		return "", fmt.Errorf("stat returned non-OK code %s", code.String())
	}

	name := res.GetInfo().GetName()
	if name == "" {
		name = path.Base(res.GetInfo().GetPath())
	}
	return sanitizeArchiveName(name), nil
}

// sanitizeArchiveName removes characters that would break the Content-Disposition header (CR, LF,
// double quote) or let the name act as a path (slash, backslash), plus all control characters
// (C0, DEL and C1). It returns an empty string if nothing usable is left.
func sanitizeArchiveName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r == '"', r == '\\', r == '/':
			return -1
		case r < 0x20 || (r >= 0x7f && r <= 0x9f):
			return -1
		default:
			return r
		}
	}, name)
	name = strings.TrimSpace(name)
	if name == "." || name == ".." {
		return ""
	}
	return name
}

func (s *svc) writeHTTPError(rw http.ResponseWriter, err error) {
	s.log.Error().Msg(err.Error())

	switch err.(type) {
	case errtypes.NotFound, errtypes.PermissionDenied:
		rw.WriteHeader(http.StatusNotFound)
	case manager.ErrMaxSize, manager.ErrMaxFileCount:
		rw.WriteHeader(http.StatusRequestEntityTooLarge)
	case errtypes.BadRequest:
		rw.WriteHeader(http.StatusBadRequest)
	default:
		rw.WriteHeader(http.StatusInternalServerError)
	}

	_, _ = rw.Write([]byte(err.Error()))
}

func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// get the paths and/or the resources id from the query
		ctx := r.Context()
		v := r.URL.Query()

		paths, ok := v["path"]
		if !ok {
			paths = []string{}
		}
		ids, ok := v["id"]
		if !ok {
			ids = []string{}
		}
		format := v.Get("output-format")
		if format == "" {
			format = "zip"
		}

		resources, err := s.getResources(ctx, paths, ids)
		if err != nil {
			s.writeHTTPError(rw, err)
			return
		}

		arch, err := manager.NewArchiver(resources, s.walker, s.downloader, manager.Config{
			MaxNumFiles: s.config.MaxNumFiles,
			MaxSize:     s.config.MaxSize,
		})
		if err != nil {
			s.writeHTTPError(rw, err)
			return
		}

		// Name the archive after the resource when a single one was requested, instead of the
		// generic "download". The name must be resolved here, before the body is streamed: the
		// Content-Disposition header below is written before CreateZip/CreateTar run, so the name
		// the walker resolves while building the archive would come too late.
		// See https://github.com/opencloud-eu/reva/issues/308
		archName := s.config.Name
		if len(resources) == 1 {
			if name, err := s.resourceName(ctx, resources[0]); name != "" && err == nil {
				archName = name
			} else {
				s.log.Debug().Err(err).Msg("could not resolve the archive name, using the default")
				archName = "download"
			}
		}
		if format == "tar" {
			archName += ".tar"
		} else {
			archName += ".zip"
		}

		s.log.Debug().Msg("Requested the following resources to archive: " + render.Render(resources))

		rw.Header().Set(net.HeaderContentDisposistion, net.ContentDispositionAttachment(archName))
		rw.Header().Set("Content-Transfer-Encoding", "binary")

		// create the archive
		var closeArchive func()
		if format == "tar" {
			closeArchive, err = arch.CreateTar(ctx, rw)
		} else {
			closeArchive, err = arch.CreateZip(ctx, rw)
		}
		defer closeArchive()

		if err != nil {
			s.writeHTTPError(rw, err)
			return
		}

	})
}

func (s *svc) Prefix() string {
	return s.config.Prefix
}

func (s *svc) Close() error {
	return nil
}

func (s *svc) Unprotected() []string {
	return nil
}
