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

package storageprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
)

// name is the Tracer name used to identify this instrumentation library.
const tracerName = "storageprovider"

func init() {
	rgrpc.Register("storageprovider", New)
}

type config struct {
	Driver              string                            `mapstructure:"driver" docs:"localhome;The storage driver to be used."`
	Drivers             map[string]map[string]interface{} `mapstructure:"drivers" docs:"url:pkg/storage/fs/localhome/localhome.go"`
	DataServerURL       string                            `mapstructure:"data_server_url" docs:"http://localhost/data;The URL for the data server."`
	ExposeDataServer    bool                              `mapstructure:"expose_data_server" docs:"false;Whether to expose data server."` // if true the client will be able to upload/download directly to it
	AvailableXS         map[string]uint32                 `mapstructure:"available_checksums" docs:"nil;List of available checksums."`
	CustomMimeTypesJSON string                            `mapstructure:"custom_mimetypes_json" docs:"nil;An optional mapping file with the list of supported custom file extensions and corresponding mime types."`
	MountID             string                            `mapstructure:"mount_id"`
	UploadExpiration    int64                             `mapstructure:"upload_expiration" docs:"0;Duration for how long uploads will be valid."`
	Events              eventconfig                       `mapstructure:"events" docs:"0;Event stream configuration"`
}

type eventconfig struct {
	Endpoint             string `mapstructure:"nats_address" docs:"address of the nats server"`
	Cluster              string `mapstructure:"nats_clusterid" docs:"clusterid of the nats server"`
	TLSInsecure          bool   `mapstructure:"tls_insecure"  docs:"Whether to verify the server TLS certificates."`
	TLSRootCACertificate string `mapstructure:"tls_root_ca_cert"  docs:"The root CA certificate used to validate the server's TLS certificate."`
	EnableTLS            bool   `mapstructure:"nats_enable_tls" docs:"events tls switch"`
	AuthUsername         string `mapstructure:"nats_username" docs:"event stream username"`
	AuthPassword         string `mapstructure:"nats_password" docs:"event stream password"`
}

func (c *config) init() {
	if c.Driver == "" {
		c.Driver = "localhome"
	}

	if c.DataServerURL == "" {
		host, err := os.Hostname()
		if err != nil || host == "" {
			c.DataServerURL = "http://0.0.0.0:19001/data"
		} else {
			c.DataServerURL = fmt.Sprintf("http://%s:19001/data", host)
		}
	}

	// set sane defaults
	if len(c.AvailableXS) == 0 {
		c.AvailableXS = map[string]uint32{"md5": 100, "unset": 1000}
	}
}

type Service struct {
	conf          *config
	Storage       storage.FS
	dataServerURL *url.URL
	availableXS   []*provider.ResourceChecksumPriority
}

func (s *Service) Close() error {
	return s.Storage.Shutdown(context.Background())
}

func (s *Service) UnprotectedEndpoints() []string { return []string{} }

func (s *Service) Register(ss *grpc.Server) {
	provider.RegisterProviderAPIServer(ss, s)
	provider.RegisterSpacesAPIServer(ss, s)
}

func parseXSTypes(xsTypes map[string]uint32) ([]*provider.ResourceChecksumPriority, error) {
	var types = make([]*provider.ResourceChecksumPriority, 0, len(xsTypes))
	for xs, prio := range xsTypes {
		t := PKG2GRPCXS(xs)
		if t == provider.ResourceChecksumType_RESOURCE_CHECKSUM_TYPE_INVALID {
			return nil, errtypes.BadRequest("checksum type is invalid: " + xs)
		}
		xsPrio := &provider.ResourceChecksumPriority{
			Priority: prio,
			Type:     t,
		}
		types = append(types, xsPrio)
	}
	return types, nil
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

func registerMimeTypes(mappingFile string) error {
	if mappingFile != "" {
		f, err := os.ReadFile(mappingFile)
		if err != nil {
			return fmt.Errorf("storageprovider: error reading the custom mime types file: +%v", err)
		}
		mimeTypes := map[string]string{}
		err = json.Unmarshal(f, &mimeTypes)
		if err != nil {
			return fmt.Errorf("storageprovider: error unmarshalling the custom mime types file: +%v", err)
		}
		// register all mime types that were read
		for e, m := range mimeTypes {
			mime.RegisterMime(e, m)
		}
	}
	return nil
}

// New creates a new storage provider svc
func New(m map[string]interface{}, ss *grpc.Server, log *zerolog.Logger) (rgrpc.Service, error) {

	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	c.init()

	fs, err := getFS(c, log)
	if err != nil {
		return nil, err
	}

	// parse data server url
	u, err := url.Parse(c.DataServerURL)
	if err != nil {
		return nil, err
	}

	// validate available checksums
	xsTypes, err := parseXSTypes(c.AvailableXS)
	if err != nil {
		return nil, err
	}

	if len(xsTypes) == 0 {
		return nil, errtypes.NotFound("no available checksum, please set in config")
	}

	// read and register custom mime types if configured
	err = registerMimeTypes(c.CustomMimeTypesJSON)
	if err != nil {
		return nil, err
	}

	service := &Service{
		conf:          c,
		Storage:       fs,
		dataServerURL: u,
		availableXS:   xsTypes,
	}

	return service, nil
}

func (s *Service) SetArbitraryMetadata(ctx context.Context, req *provider.SetArbitraryMetadataRequest) (*provider.SetArbitraryMetadataResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	err := s.Storage.SetArbitraryMetadata(ctx, req.Ref, req.ArbitraryMetadata)

	return &provider.SetArbitraryMetadataResponse{
		Status: status.NewStatusFromErrType(ctx, "set arbitrary metadata", err),
	}, nil
}

func (s *Service) UnsetArbitraryMetadata(ctx context.Context, req *provider.UnsetArbitraryMetadataRequest) (*provider.UnsetArbitraryMetadataResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	err := s.Storage.UnsetArbitraryMetadata(ctx, req.Ref, req.ArbitraryMetadataKeys)

	return &provider.UnsetArbitraryMetadataResponse{
		Status: status.NewStatusFromErrType(ctx, "unset arbitrary metadata", err),
	}, nil
}

// SetLock puts a lock on the given reference
func (s *Service) SetLock(ctx context.Context, req *provider.SetLockRequest) (*provider.SetLockResponse, error) {
	if !canLockPublicShare(ctx) {
		return &provider.SetLockResponse{
			Status: status.NewPermissionDenied(ctx, nil, "no permission to lock the share"),
		}, nil
	}
	err := s.Storage.SetLock(ctx, req.Ref, req.Lock)

	return &provider.SetLockResponse{
		Status: status.NewStatusFromErrType(ctx, "set lock", err),
	}, nil
}

// GetLock returns an existing lock on the given reference
func (s *Service) GetLock(ctx context.Context, req *provider.GetLockRequest) (*provider.GetLockResponse, error) {
	lock, err := s.Storage.GetLock(ctx, req.Ref)

	return &provider.GetLockResponse{
		Status: status.NewStatusFromErrType(ctx, "get lock", err),
		Lock:   lock,
	}, nil
}

// RefreshLock refreshes an existing lock on the given reference
func (s *Service) RefreshLock(ctx context.Context, req *provider.RefreshLockRequest) (*provider.RefreshLockResponse, error) {
	if !canLockPublicShare(ctx) {
		return &provider.RefreshLockResponse{
			Status: status.NewPermissionDenied(ctx, nil, "no permission to refresh the share lock"),
		}, nil
	}

	err := s.Storage.RefreshLock(ctx, req.Ref, req.Lock, req.ExistingLockId)

	return &provider.RefreshLockResponse{
		Status: status.NewStatusFromErrType(ctx, "refresh lock", err),
	}, nil
}

// Unlock removes an existing lock from the given reference
func (s *Service) Unlock(ctx context.Context, req *provider.UnlockRequest) (*provider.UnlockResponse, error) {
	if !canLockPublicShare(ctx) {
		return &provider.UnlockResponse{
			Status: status.NewPermissionDenied(ctx, nil, "no permission to unlock the share"),
		}, nil
	}

	err := s.Storage.Unlock(ctx, req.Ref, req.Lock)

	return &provider.UnlockResponse{
		Status: status.NewStatusFromErrType(ctx, "unlock", err),
	}, nil
}

func (s *Service) InitiateFileDownload(ctx context.Context, req *provider.InitiateFileDownloadRequest) (*provider.InitiateFileDownloadResponse, error) {
	// TODO(labkode): maybe add some checks before download starts? eg. check permissions?
	// TODO(labkode): maybe add short-lived token?
	// We now simply point the client to the data server.
	// For example, https://data-server.example.org/home/docs/myfile.txt
	log := appctx.GetLogger(ctx)
	u := *s.dataServerURL
	log.Info().Str("data-server", u.String()).Interface("ref", req.Ref).Msg("file download")

	protocol := &provider.FileDownloadProtocol{Expose: s.conf.ExposeDataServer}

	if utils.IsRelativeReference(req.Ref) {
		s.addMissingStorageProviderID(req.GetRef().GetResourceId(), nil)
		protocol.Protocol = "spaces"
		u.Path = path.Join(u.Path, "spaces", storagespace.FormatResourceID(req.Ref.ResourceId), req.Ref.Path)
	} else {
		// Currently, we only support the simple protocol for GET requests
		// Once we have multiple protocols, this would be moved to the fs layer
		protocol.Protocol = "simple"
		u.Path = path.Join(u.Path, "simple", req.Ref.GetPath())
	}

	protocol.DownloadEndpoint = u.String()

	return &provider.InitiateFileDownloadResponse{
		Protocols: []*provider.FileDownloadProtocol{protocol},
		Status:    status.NewOK(ctx),
	}, nil
}

func validateIfMatch(ifMatch string, info *provider.ResourceInfo) bool {
	return ifMatch == info.GetEtag()
}

func validateIfUnmodifiedSince(ifUnmodifiedSince *typesv1beta1.Timestamp, info *provider.ResourceInfo) bool {
	switch {
	case ifUnmodifiedSince == nil || info.GetMtime() == nil:
		return true
	case utils.LaterTS(info.GetMtime(), ifUnmodifiedSince) == info.GetMtime():
		return false
	default:
		return true
	}
}

func (s *Service) InitiateFileUpload(ctx context.Context, req *provider.InitiateFileUploadRequest) (*provider.InitiateFileUploadResponse, error) {
	// TODO(labkode): same considerations as download
	log := appctx.GetLogger(ctx)
	if req.Ref.GetPath() == "/" {
		return &provider.InitiateFileUploadResponse{
			Status: status.NewInternal(ctx, "can't upload to mount path"),
		}, nil
	}

	// FIXME move the etag check into the InitiateUpload call instead of making a Stat call here
	sRes, err := s.Stat(ctx, &provider.StatRequest{Ref: req.Ref})
	if err != nil {
		return nil, err
	}
	switch sRes.Status.Code {
	case rpc.Code_CODE_OK:
		if req.GetIfNotExist() {
			return &provider.InitiateFileUploadResponse{
				Status: status.NewAlreadyExists(ctx, errors.New("already exists"), "already exists"),
			}, nil
		}
	case rpc.Code_CODE_NOT_FOUND:
		// Just continue with a normal upload
	default:
		return &provider.InitiateFileUploadResponse{
			Status: sRes.Status,
		}, nil
	}

	metadata := map[string]string{}
	ifMatch := req.GetIfMatch()
	if ifMatch != "" {
		if !validateIfMatch(ifMatch, sRes.GetInfo()) {
			return &provider.InitiateFileUploadResponse{
				Status: status.NewFailedPrecondition(ctx, errors.New("etag mismatch"), "etag mismatch"),
			}, nil
		}
		metadata["if-match"] = ifMatch
	}
	ifUnmodifiedSince := req.GetIfUnmodifiedSince()
	if ifUnmodifiedSince != nil {
		metadata["if-unmodified-since"] = utils.TSToTime(ifUnmodifiedSince).Format(time.RFC3339Nano)
		if !validateIfUnmodifiedSince(ifUnmodifiedSince, sRes.GetInfo()) {
			return &provider.InitiateFileUploadResponse{
				Status: status.NewFailedPrecondition(ctx, errors.New("resource has been modified"), "resource has been modified"),
			}, nil
		}
	}

	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	var uploadLength int64
	if req.Opaque != nil && req.Opaque.Map != nil {
		if req.Opaque.Map["Upload-Length"] != nil {
			var err error
			uploadLength, err = strconv.ParseInt(string(req.Opaque.Map["Upload-Length"].Value), 10, 64)
			if err != nil {
				log.Error().Err(err).Msg("error parsing upload length")
				return &provider.InitiateFileUploadResponse{
					Status: status.NewInternal(ctx, "error parsing upload length"),
				}, nil
			}
		}
		// TUS forward Upload-Checksum header as checksum, uses '[type] [hash]' format
		if req.Opaque.Map["Upload-Checksum"] != nil {
			metadata["checksum"] = string(req.Opaque.Map["Upload-Checksum"].Value)
		}
		// OpenCloud mtime to set for the uploaded file
		if req.Opaque.Map["X-OC-Mtime"] != nil {
			metadata["mtime"] = string(req.Opaque.Map["X-OC-Mtime"].Value)
		}
	}

	// pass on the provider it to be persisted with the upload info. that is required to correlate the upload with the proper provider later on
	metadata["providerID"] = s.conf.MountID
	var expirationTimestamp *typesv1beta1.Timestamp
	if s.conf.UploadExpiration > 0 {
		expirationTimestamp = &typesv1beta1.Timestamp{
			Seconds: uint64(time.Now().UTC().Add(time.Duration(s.conf.UploadExpiration) * time.Second).Unix()),
		}
		metadata["expires"] = strconv.Itoa(int(expirationTimestamp.Seconds))
	}

	uploadIDs, err := s.Storage.InitiateUpload(ctx, req.Ref, uploadLength, metadata)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "path not found when initiating upload")
		case errtypes.IsBadRequest, errtypes.IsChecksumMismatch:
			st = status.NewInvalid(ctx, err.Error())
			// TODO TUS uses a custom ChecksumMismatch 460 http status which is in an unassigned range in
			// https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
			// maybe 409 conflict is good enough
			// someone is proposing `419 Checksum Error`, see https://stackoverflow.com/a/35665694
			// - it is also unassigned
			// - ends in 9 as the 409 conflict
			// - is near the 4xx errors about conditions: 415 Unsupported Media Type, 416 Range Not Satisfiable or 417 Expectation Failed
			// OpenCloud only expects a 400 Bad request so InvalidArg is good enough for now
			// seealso errtypes.StatusChecksumMismatch
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		case errtypes.InsufficientStorage:
			st = status.NewInsufficientStorage(ctx, err, "insufficient storage")
		case errtypes.PreconditionFailed:
			st = status.NewFailedPrecondition(ctx, err, "failed precondition")
		case errtypes.Locked:
			st = status.NewLocked(ctx, "locked")
		default:
			st = status.NewInternal(ctx, "error getting upload id: "+err.Error())
		}
		log.Error().
			Err(err).
			Interface("status", st).
			Msg("failed to initiate upload")
		return &provider.InitiateFileUploadResponse{
			Status: st,
		}, nil
	}

	protocols := make([]*provider.FileUploadProtocol, len(uploadIDs))
	var i int
	for protocol, ID := range uploadIDs {
		u := *s.dataServerURL
		u.Path = path.Join(u.Path, protocol, ID)
		protocols[i] = &provider.FileUploadProtocol{
			Protocol:           protocol,
			UploadEndpoint:     u.String(),
			AvailableChecksums: s.availableXS,
			Expose:             s.conf.ExposeDataServer,
			Expiration:         expirationTimestamp,
		}
		i++
		log.Info().Str("data-server", u.String()).
			Str("fn", req.Ref.GetPath()).
			Str("xs", fmt.Sprintf("%+v", s.conf.AvailableXS)).
			Msg("file upload")
	}

	res := &provider.InitiateFileUploadResponse{
		Protocols: protocols,
		Status:    status.NewOK(ctx),
	}
	// FIXME make created flag a property on the InitiateFileUploadResponse
	if sRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
		res.Opaque = utils.AppendPlainToOpaque(res.Opaque, "created", "true")
	}
	return res, nil
}

func (s *Service) GetPath(ctx context.Context, req *provider.GetPathRequest) (*provider.GetPathResponse, error) {
	// TODO(labkode): check that the storage ID is the same as the storage provider id.
	fn, err := s.Storage.GetPathByID(ctx, req.ResourceId)
	if err != nil {
		return &provider.GetPathResponse{
			Status: status.NewStatusFromErrType(ctx, "get path", err),
		}, nil
	}
	res := &provider.GetPathResponse{
		Path:   fn,
		Status: status.NewOK(ctx),
	}
	return res, nil
}

func (s *Service) GetHome(ctx context.Context, req *provider.GetHomeRequest) (*provider.GetHomeResponse, error) {
	return nil, errtypes.NotSupported("unused, use the gateway to look up the user home")
}

func (s *Service) CreateHome(ctx context.Context, req *provider.CreateHomeRequest) (*provider.CreateHomeResponse, error) {
	return nil, errtypes.NotSupported("use CreateStorageSpace with type personal")
}

// CreateStorageSpace creates a storage space
func (s *Service) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	resp, err := s.Storage.CreateStorageSpace(ctx, req)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "not found when creating space")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		case errtypes.NotSupported:
			// if trying to create a user home fall back to CreateHome
			if u, ok := ctxpkg.ContextGetUser(ctx); ok && req.Type == "personal" && utils.UserEqual(req.GetOwner().Id, u.Id) {
				if err := s.Storage.CreateHome(ctx); err != nil {
					st = status.NewInternal(ctx, "error creating home")
				} else {
					st = status.NewOK(ctx)
					// TODO we cannot return a space, but the gateway currently does not expect one...
				}
			} else {
				st = status.NewUnimplemented(ctx, err, "not implemented")
			}
		case errtypes.AlreadyExists:
			st = status.NewAlreadyExists(ctx, err, "already exists")
		case errtypes.BadRequest:
			st = status.NewInvalid(ctx, err.Error())
		default:
			st = status.NewInternal(ctx, "error creating space")
			appctx.GetLogger(ctx).
				Error().
				Err(err).
				Interface("status", st).
				Interface("request", req).
				Msg("failed to create storage space")
		}
		return &provider.CreateStorageSpaceResponse{
			Status: st,
		}, nil
	}

	s.addMissingStorageProviderID(resp.GetStorageSpace().GetRoot(), resp.GetStorageSpace().GetId())
	return resp, nil
}

func (s *Service) ListStorageSpaces(ctx context.Context, req *provider.ListStorageSpacesRequest) (*provider.ListStorageSpacesResponse, error) {
	log := appctx.GetLogger(ctx)

	// TODO this is just temporary. Update the API to include this flag.
	unrestricted := false
	if req.Opaque != nil {
		if entry, ok := req.Opaque.Map["unrestricted"]; ok {
			unrestricted, _ = strconv.ParseBool(string(entry.Value))
		}
	}

	spaces, err := s.Storage.ListStorageSpaces(ctx, req.Filters, unrestricted)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "not found when listing spaces")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		case errtypes.NotSupported:
			st = status.NewUnimplemented(ctx, err, "not implemented")
		default:
			st = status.NewInternal(ctx, "error listing spaces")
		}
		log.Error().
			Err(err).
			Interface("status", st).
			Interface("filters", req.Filters).
			Msg("failed to list storage spaces")
		return &provider.ListStorageSpacesResponse{
			Status: st,
		}, nil
	}

	for _, sp := range spaces {
		if sp.Id == nil || sp.Id.OpaqueId == "" {
			log.Error().Str("service", "storageprovider").Str("driver", s.conf.Driver).Interface("space", sp).Msg("space is missing space id and root id")
			continue
		}

		s.addMissingStorageProviderID(sp.GetRoot(), sp.GetId())
	}

	return &provider.ListStorageSpacesResponse{
		Status:        status.NewOK(ctx),
		StorageSpaces: spaces,
	}, nil
}

func (s *Service) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	res, err := s.Storage.UpdateStorageSpace(ctx, req)
	if err != nil {
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("req", req).
			Msg("failed to update storage space")
		return nil, err
	}
	s.addMissingStorageProviderID(res.GetStorageSpace().GetRoot(), res.GetStorageSpace().GetId())
	return res, nil
}

func (s *Service) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*provider.DeleteStorageSpaceResponse, error) {
	// we need to get the space before so we can return critical information
	// FIXME: why is this string parsing necessary?
	idraw, _ := storagespace.ParseID(req.Id.GetOpaqueId())
	idraw.OpaqueId = idraw.GetSpaceId()
	id := &provider.StorageSpaceId{OpaqueId: storagespace.FormatResourceID(&idraw)}

	spaces, err := s.Storage.ListStorageSpaces(ctx, []*provider.ListStorageSpacesRequest_Filter{{Type: provider.ListStorageSpacesRequest_Filter_TYPE_ID, Term: &provider.ListStorageSpacesRequest_Filter_Id{Id: id}}}, true)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "space not found")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		case errtypes.BadRequest:
			st = status.NewInvalid(ctx, err.Error())
		default:
			st = status.NewInternal(ctx, "error deleting space: "+req.Id.String())
		}
		return &provider.DeleteStorageSpaceResponse{
			Status: st,
		}, nil
	} else if len(spaces) != 1 {
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewNotFound(ctx, "space not found"),
		}, nil
	}

	if err := s.Storage.DeleteStorageSpace(ctx, req); err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "space not found")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		case errtypes.BadRequest:
			st = status.NewInvalid(ctx, err.Error())
		default:
			st = status.NewInternal(ctx, "error deleting space: "+req.Id.String())
		}
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("status", st).
			Interface("storage_space_id", req.Id).
			Msg("failed to delete storage space")
		return &provider.DeleteStorageSpaceResponse{
			Status: st,
		}, nil
	}

	// TODO: update cs3api
	o := utils.AppendPlainToOpaque(nil, "spacename", spaces[0].GetName())
	o.Map["grants"] = spaces[0].GetOpaque().GetMap()["grants"]

	res := &provider.DeleteStorageSpaceResponse{
		Opaque: o,
		Status: status.NewOK(ctx),
	}
	return res, nil
}

func (s *Service) CreateContainer(ctx context.Context, req *provider.CreateContainerRequest) (*provider.CreateContainerResponse, error) {
	// FIXME these should be part of the CreateContainerRequest object
	if req.Opaque != nil {
		if e, ok := req.Opaque.Map["lockid"]; ok && e.Decoder == "plain" {
			ctx = ctxpkg.ContextSetLockID(ctx, string(e.Value))
		}
	}

	err := s.Storage.CreateDir(ctx, req.Ref)

	return &provider.CreateContainerResponse{
		Status: status.NewStatusFromErrType(ctx, "create container", err),
	}, nil
}

func (s *Service) TouchFile(ctx context.Context, req *provider.TouchFileRequest) (*provider.TouchFileResponse, error) {
	// FIXME these should be part of the TouchFileRequest object
	var mtime string
	if req.Opaque != nil {
		if e, ok := req.Opaque.Map["lockid"]; ok && e.Decoder == "plain" {
			ctx = ctxpkg.ContextSetLockID(ctx, string(e.Value))
		}
		mtime = utils.ReadPlainFromOpaque(req.Opaque, "X-OC-Mtime")
	}

	err := s.Storage.TouchFile(ctx, req.Ref, utils.ExistsInOpaque(req.Opaque, "markprocessing"), mtime)

	return &provider.TouchFileResponse{
		Status: status.NewStatusFromErrType(ctx, "touch file", err),
	}, nil
}

func (s *Service) Delete(ctx context.Context, req *provider.DeleteRequest) (*provider.DeleteResponse, error) {
	if req.Ref.GetPath() == "/" {
		return &provider.DeleteResponse{
			Status: status.NewInternal(ctx, "can't delete mount path"),
		}, nil
	}

	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	// check DeleteRequest for any known opaque properties.
	// FIXME these should be part of the DeleteRequest object
	if req.Opaque != nil {
		if _, ok := req.Opaque.Map["deleting_shared_resource"]; ok {
			// it is a binary key; its existence signals true. Although, do not assume.
			ctx = appctx.WithDeletingSharedResource(ctx)
		}
	}

	md, err := s.Storage.GetMD(ctx, req.Ref, []string{}, []string{"id", "status"})
	if err != nil {
		return &provider.DeleteResponse{
			Status: status.NewStatusFromErrType(ctx, "can't stat resource to delete", err),
		}, nil
	}

	if utils.ReadPlainFromOpaque(md.GetOpaque(), "status") == "processing" {
		return &provider.DeleteResponse{
			Status: &rpc.Status{
				Code:    rpc.Code_CODE_TOO_EARLY,
				Message: "file is processing",
			},
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"status": {Decoder: "plain", Value: []byte("processing")},
				},
			},
		}, nil
	}

	err = s.Storage.Delete(ctx, req.Ref)

	return &provider.DeleteResponse{
		Status: status.NewStatusFromErrType(ctx, "delete", err),
		Opaque: &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"opaque_id": {Decoder: "plain", Value: []byte(md.Id.OpaqueId)},
			},
		},
	}, nil
}

func (s *Service) Move(ctx context.Context, req *provider.MoveRequest) (*provider.MoveResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	err := s.Storage.Move(ctx, req.Source, req.Destination)

	return &provider.MoveResponse{
		Status: status.NewStatusFromErrType(ctx, "move", err),
	}, nil
}

func (s *Service) Stat(ctx context.Context, req *provider.StatRequest) (*provider.StatResponse, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "stat")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "reference",
		Value: attribute.StringValue(req.GetRef().String()),
	})

	md, err := s.Storage.GetMD(ctx, req.GetRef(), req.GetArbitraryMetadataKeys(), req.GetFieldMask().GetPaths())
	if err != nil {
		return &provider.StatResponse{
			Status: status.NewStatusFromErrType(ctx, "stat", err),
		}, nil
	}

	s.addMissingStorageProviderID(md.GetId(), nil)
	s.addMissingStorageProviderID(md.GetParentId(), nil)
	s.addMissingStorageProviderID(md.GetSpace().GetRoot(), nil)

	return &provider.StatResponse{
		Status: status.NewOK(ctx),
		Info:   md,
	}, nil
}

func (s *Service) ListContainerStream(req *provider.ListContainerStreamRequest, ss provider.ProviderAPI_ListContainerStreamServer) error {
	ctx := ss.Context()
	log := appctx.GetLogger(ctx)

	mds, err := s.Storage.ListFolder(ctx, req.GetRef(), req.GetArbitraryMetadataKeys(), req.GetFieldMask().GetPaths())
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "path not found when listing container")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error listing container: "+req.Ref.String())
		}
		log.Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Msg("failed to list folder (stream)")
		res := &provider.ListContainerStreamResponse{
			Status: st,
		}
		if err := ss.Send(res); err != nil {
			log.Error().Err(err).Msg("ListContainerStream: error sending response")
			return err
		}
		return nil
	}

	for _, md := range mds {
		s.addMissingStorageProviderID(md.GetId(), nil)
		s.addMissingStorageProviderID(md.GetParentId(), nil)
		s.addMissingStorageProviderID(md.GetSpace().GetRoot(), nil)
		res := &provider.ListContainerStreamResponse{
			Info:   md,
			Status: status.NewOK(ctx),
		}

		if err := ss.Send(res); err != nil {
			log.Error().Err(err).Msg("ListContainerStream: error sending response")
			return err
		}
	}
	return nil
}

func (s *Service) ListContainer(ctx context.Context, req *provider.ListContainerRequest) (*provider.ListContainerResponse, error) {
	mds, err := s.Storage.ListFolder(ctx, req.GetRef(), req.GetArbitraryMetadataKeys(), req.GetFieldMask().GetPaths())
	res := &provider.ListContainerResponse{
		Status: status.NewStatusFromErrType(ctx, "list container", err),
		Infos:  mds,
	}
	if err != nil {
		return res, nil
	}

	for _, i := range res.Infos {
		s.addMissingStorageProviderID(i.GetId(), nil)
		s.addMissingStorageProviderID(i.GetParentId(), nil)
		s.addMissingStorageProviderID(i.GetSpace().GetRoot(), nil)
	}
	return res, nil
}

func (s *Service) ListFileVersions(ctx context.Context, req *provider.ListFileVersionsRequest) (*provider.ListFileVersionsResponse, error) {
	revs, err := s.Storage.ListRevisions(ctx, req.Ref)

	sort.Sort(descendingMtime(revs))

	return &provider.ListFileVersionsResponse{
		Status:   status.NewStatusFromErrType(ctx, "list file versions", err),
		Versions: revs,
	}, nil
}

func (s *Service) RestoreFileVersion(ctx context.Context, req *provider.RestoreFileVersionRequest) (*provider.RestoreFileVersionResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	err := s.Storage.RestoreRevision(ctx, req.Ref, req.Key)

	return &provider.RestoreFileVersionResponse{
		Status: status.NewStatusFromErrType(ctx, "restore file version", err),
	}, nil
}

func (s *Service) ListRecycleStream(req *provider.ListRecycleStreamRequest, ss provider.ProviderAPI_ListRecycleStreamServer) error {
	ctx := ss.Context()
	log := appctx.GetLogger(ctx)

	// if no slash is present in the key, do not pass a relative path to the storage
	// when a path is passed to the storage, it will list the contents of the directory
	key, relativePath := splitKeyAndPath(req.GetKey())
	items, err := s.Storage.ListRecycle(ctx, req.Ref, key, relativePath)

	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "resource not found when listing recycle stream")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error listing recycle stream")
		}
		log.Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Str("key", req.Key).
			Msg("failed to list recycle (stream)")
		res := &provider.ListRecycleStreamResponse{
			Status: st,
		}
		if err := ss.Send(res); err != nil {
			log.Error().Err(err).Msg("ListRecycleStream: error sending response")
			return err
		}
		return nil
	}

	// TODO(labkode): CRITICAL: fill recycle info with storage provider.
	for _, item := range items {
		s.addMissingStorageProviderID(item.GetRef().GetResourceId(), nil)
		res := &provider.ListRecycleStreamResponse{
			RecycleItem: item,
			Status:      status.NewOK(ctx),
		}
		if err := ss.Send(res); err != nil {
			log.Error().Err(err).Msg("ListRecycleStream: error sending response")
			return err
		}
	}
	return nil
}

func (s *Service) ListRecycle(ctx context.Context, req *provider.ListRecycleRequest) (*provider.ListRecycleResponse, error) {
	// if no slash is present in the key, do not pass a relative path to the storage
	// when a path is passed to the storage, it will list the contents of the directory
	key, relativePath := splitKeyAndPath(req.GetKey())
	items, err := s.Storage.ListRecycle(ctx, req.Ref, key, relativePath)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "resource not found when listing recycle")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error listing recycle")
		}
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Str("key", req.Key).
			Msg("failed to list recycle")
		return &provider.ListRecycleResponse{
			Status: st,
		}, nil
	}

	for _, i := range items {
		s.addMissingStorageProviderID(i.GetRef().GetResourceId(), nil)
	}
	res := &provider.ListRecycleResponse{
		Status:       status.NewOK(ctx),
		RecycleItems: items,
	}
	return res, nil
}

func (s *Service) RestoreRecycleItem(ctx context.Context, req *provider.RestoreRecycleItemRequest) (*provider.RestoreRecycleItemResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	// TODO(labkode): CRITICAL: fill recycle info with storage provider.
	key, relativePath := splitKeyAndPath(req.GetKey())
	err := s.Storage.RestoreRecycleItem(ctx, req.Ref, key, relativePath, req.RestoreRef)

	res := &provider.RestoreRecycleItemResponse{
		Status: status.NewStatusFromErrType(ctx, "restore recycle item", err),
	}
	return res, nil
}

func (s *Service) PurgeRecycle(ctx context.Context, req *provider.PurgeRecycleRequest) (*provider.PurgeRecycleResponse, error) {
	// FIXME these should be part of the PurgeRecycleRequest object
	if req.Opaque != nil {
		if e, ok := req.Opaque.Map["lockid"]; ok && e.Decoder == "plain" {
			ctx = ctxpkg.ContextSetLockID(ctx, string(e.Value))
		}
	}

	// if a key was sent as opaque id purge only that item
	key, relativePath := splitKeyAndPath(req.GetKey())
	if key != "" {
		if err := s.Storage.PurgeRecycleItem(ctx, req.Ref, key, relativePath); err != nil {
			st := status.NewStatusFromErrType(ctx, "error purging recycle item", err)
			appctx.GetLogger(ctx).
				Error().
				Err(err).
				Interface("status", st).
				Interface("reference", req.Ref).
				Str("key", req.Key).
				Msg("failed to purge recycle item")
			return &provider.PurgeRecycleResponse{
				Status: st,
			}, nil
		}
	} else if err := s.Storage.EmptyRecycle(ctx, req.Ref); err != nil {
		// otherwise try emptying the whole recycle bin
		st := status.NewStatusFromErrType(ctx, "error emptying recycle", err)
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Str("key", req.Key).
			Msg("failed to empty recycle")
		return &provider.PurgeRecycleResponse{
			Status: st,
		}, nil
	}

	res := &provider.PurgeRecycleResponse{
		Status: status.NewOK(ctx),
	}
	return res, nil
}

func (s *Service) ListGrants(ctx context.Context, req *provider.ListGrantsRequest) (*provider.ListGrantsResponse, error) {
	grants, err := s.Storage.ListGrants(ctx, req.Ref)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "path not found when listing grants")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error listing grants")
		}
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Msg("failed to list grants")
		return &provider.ListGrantsResponse{
			Status: st,
		}, nil
	}

	res := &provider.ListGrantsResponse{
		Status: status.NewOK(ctx),
		Grants: grants,
	}
	return res, nil
}

func (s *Service) DenyGrant(ctx context.Context, req *provider.DenyGrantRequest) (*provider.DenyGrantResponse, error) {
	// check grantee type is valid
	if req.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_INVALID {
		return &provider.DenyGrantResponse{
			Status: status.NewInvalid(ctx, "grantee type is invalid"),
		}, nil
	}

	err := s.Storage.DenyGrant(ctx, req.Ref, req.Grantee)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.NotSupported:
			// ignore - setting storage grants is optional
			return &provider.DenyGrantResponse{
				Status: status.NewOK(ctx),
			}, nil
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "path not found when setting grants")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error setting grants")
		}
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Msg("failed to deny grant")
		return &provider.DenyGrantResponse{
			Status: st,
		}, nil
	}

	res := &provider.DenyGrantResponse{
		Status: status.NewOK(ctx),
	}
	return res, nil
}

type spaceTypeKey struct{}

func WithSpaceType(ctx context.Context, spaceType string) context.Context {
	return context.WithValue(ctx, spaceTypeKey{}, spaceType)
}
func SpaceTypeFromContext(ctx context.Context) (string, bool) {
	spaceType, ok := ctx.Value(spaceTypeKey{}).(string)
	return spaceType, ok
}

func (s *Service) AddGrant(ctx context.Context, req *provider.AddGrantRequest) (*provider.AddGrantResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	// TODO: update CS3 APIs
	// FIXME these should be part of the AddGrantRequest object
	// https://github.com/owncloud/ocis/issues/4312
	if utils.ExistsInOpaque(req.Opaque, "spacegrant") {
		ctx = WithSpaceType(ctx, utils.ReadPlainFromOpaque(req.Opaque, "spacetype"))
	}

	// error out if no permissions are set
	if req.GetGrant().GetPermissions() == nil {
		return &provider.AddGrantResponse{
			Status: status.NewInvalid(ctx, "permissions are invalid"),
		}, nil
	}

	// check grantee type is valid
	if req.Grant.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_INVALID {
		return &provider.AddGrantResponse{
			Status: status.NewInvalid(ctx, "grantee type is invalid"),
		}, nil
	}

	// check if grantee has an id
	if req.GetGrant().GetGrantee().GetId() == nil {
		return &provider.AddGrantResponse{
			Status: status.NewInvalid(ctx, "grantee id is invalid"),
		}, nil
	}

	err := s.Storage.AddGrant(ctx, req.Ref, req.Grant)

	return &provider.AddGrantResponse{
		Status: status.NewStatusFromErrType(ctx, "add grant", err),
	}, nil
}

func (s *Service) UpdateGrant(ctx context.Context, req *provider.UpdateGrantRequest) (*provider.UpdateGrantResponse, error) {
	// FIXME these should be part of the UpdateGrantRequest object
	if req.Opaque != nil {
		if e, ok := req.Opaque.Map["lockid"]; ok && e.Decoder == "plain" {
			ctx = ctxpkg.ContextSetLockID(ctx, string(e.Value))
		}
	}

	// TODO: update CS3 APIs
	// FIXME these should be part of the AddGrantRequest object
	// https://github.com/owncloud/ocis/issues/4312
	if utils.ExistsInOpaque(req.Opaque, "spacegrant") {
		ctx = WithSpaceType(ctx, utils.ReadPlainFromOpaque(req.Opaque, "spacetype"))
	}

	// check grantee type is valid
	if req.Grant.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_INVALID {
		return &provider.UpdateGrantResponse{
			Status: status.NewInvalid(ctx, "grantee type is invalid"),
		}, nil
	}

	err := s.Storage.UpdateGrant(ctx, req.Ref, req.Grant)

	return &provider.UpdateGrantResponse{
		Status: status.NewStatusFromErrType(ctx, "update grant", err),
	}, nil
}

func (s *Service) RemoveGrant(ctx context.Context, req *provider.RemoveGrantRequest) (*provider.RemoveGrantResponse, error) {
	ctx = ctxpkg.ContextSetLockID(ctx, req.LockId)

	// check targetType is valid
	if req.Grant.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_INVALID {
		return &provider.RemoveGrantResponse{
			Status: status.NewInvalid(ctx, "grantee type is invalid"),
		}, nil
	}

	// TODO: update CS3 APIs
	// FIXME these should be part of the RemoveGrantRequest object
	// https://github.com/owncloud/ocis/issues/4312
	if utils.ExistsInOpaque(req.Opaque, "spacegrant") {
		ctx = WithSpaceType(ctx, "")
	}

	err := s.Storage.RemoveGrant(ctx, req.Ref, req.Grant)

	return &provider.RemoveGrantResponse{
		Status: status.NewStatusFromErrType(ctx, "remove grant", err),
	}, nil
}

func (s *Service) CreateReference(ctx context.Context, req *provider.CreateReferenceRequest) (*provider.CreateReferenceResponse, error) {
	log := appctx.GetLogger(ctx)

	// parse uri is valid
	u, err := url.Parse(req.TargetUri)
	if err != nil {
		log.Error().Err(err).Msg("invalid target uri")
		return &provider.CreateReferenceResponse{
			Status: status.NewInvalid(ctx, "target uri is invalid: "+err.Error()),
		}, nil
	}

	if err := s.Storage.CreateReference(ctx, req.Ref.GetPath(), u); err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "path not found when creating reference")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error creating reference")
		}
		log.Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Msg("failed to create reference")
		return &provider.CreateReferenceResponse{
			Status: st,
		}, nil
	}

	return &provider.CreateReferenceResponse{
		Status: status.NewOK(ctx),
	}, nil
}

func (s *Service) CreateSymlink(ctx context.Context, req *provider.CreateSymlinkRequest) (*provider.CreateSymlinkResponse, error) {
	return &provider.CreateSymlinkResponse{
		Status: status.NewUnimplemented(ctx, errtypes.NotSupported("CreateSymlink not implemented"), "CreateSymlink not implemented"),
	}, nil
}

func (s *Service) GetQuota(ctx context.Context, req *provider.GetQuotaRequest) (*provider.GetQuotaResponse, error) {
	total, used, remaining, err := s.Storage.GetQuota(ctx, req.Ref)
	if err != nil {
		var st *rpc.Status
		switch err.(type) {
		case errtypes.IsNotFound:
			st = status.NewNotFound(ctx, "path not found when getting quota")
		case errtypes.PermissionDenied:
			st = status.NewPermissionDenied(ctx, err, "permission denied")
		default:
			st = status.NewInternal(ctx, "error getting quota")
		}
		appctx.GetLogger(ctx).
			Error().
			Err(err).
			Interface("status", st).
			Interface("reference", req.Ref).
			Msg("failed to get quota")
		return &provider.GetQuotaResponse{
			Status: st,
		}, nil
	}

	res := &provider.GetQuotaResponse{
		Opaque: &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"remaining": {
					Decoder: "plain",
					Value:   []byte(strconv.FormatUint(remaining, 10)),
				},
			},
		},
		Status:     status.NewOK(ctx),
		TotalBytes: total,
		UsedBytes:  used,
	}
	return res, nil
}

func (s *Service) addMissingStorageProviderID(resourceID *provider.ResourceId, spaceID *provider.StorageSpaceId) {
	// The storage driver might set the mount ID by itself, in which case skip this step
	if resourceID != nil && resourceID.GetStorageId() == "" {
		resourceID.StorageId = s.conf.MountID
		if spaceID != nil {
			rid, _ := storagespace.ParseID(spaceID.GetOpaqueId())
			rid.StorageId = s.conf.MountID
			spaceID.OpaqueId, _ = storagespace.FormatReference(&provider.Reference{ResourceId: &rid})
		}
	}
}

func getFS(c *config, log *zerolog.Logger) (storage.FS, error) {
	evstream, err := estreamFromConfig(c.Events)
	if err != nil {
		return nil, err
	}

	if f, ok := registry.NewFuncs[c.Driver]; ok {
		driverConf := c.Drivers[c.Driver]
		driverConf["mount_id"] = c.MountID // pass the mount id to the driver

		return f(driverConf, evstream, log)
	}

	return nil, errtypes.NotFound("driver not found: " + c.Driver)
}

type descendingMtime []*provider.FileVersion

func (v descendingMtime) Len() int {
	return len(v)
}

func (v descendingMtime) Less(i, j int) bool {
	return v[i].Mtime >= v[j].Mtime
}

func (v descendingMtime) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func estreamFromConfig(c eventconfig) (events.Stream, error) {
	if c.Endpoint == "" {
		return nil, nil
	}

	return stream.NatsFromConfig("storageprovider", false, stream.NatsConfig(c))
}

func canLockPublicShare(ctx context.Context) bool {
	u := ctxpkg.ContextMustGetUser(ctx)
	psr := utils.ReadPlainFromOpaque(u.Opaque, "public-share-role")
	return psr == "" || psr == conversions.RoleEditor
}

// splitKeyAndPath splits a key into a root and a relative path
func splitKeyAndPath(key string) (string, string) {
	root, relativePath := router.ShiftPath(key)
	if relativePath == "/" && !strings.HasSuffix(key, "/") {
		relativePath = ""
	}
	return root, relativePath
}
