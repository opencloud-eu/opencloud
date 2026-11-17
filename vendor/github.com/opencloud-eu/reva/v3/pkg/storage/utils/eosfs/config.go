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

package eosfs

// Config holds the configuration details for the EOS fs.
type Config struct {
	// Namespace for metadata operations
	Namespace string `mapstructure:"namespace"`

	// The root at which the file system is mounted
	MountPath string `mapstructure:"mount_path"`

	// QuotaNode for storing quota information
	QuotaNode string `mapstructure:"quota_node"`

	// DefaultQuotaBytes sets the default maximum bytes available for a user
	DefaultQuotaBytes uint64 `mapstructure:"default_quota_bytes"`

	// DefaultQuotaFiles sets the default maximum files available for a user
	DefaultQuotaFiles uint64 `mapstructure:"default_quota_files"`

	// ShadowNamespace for storing shadow data
	ShadowNamespace string `mapstructure:"shadow_namespace"`

	// UploadsNamespace for storing upload data
	UploadsNamespace string `mapstructure:"uploads_namespace"`

	// ShareFolder defines the name of the folder in the
	// shadowed namespace. Ex: /eos/user/.shadow/h/hugo/MyShares
	ShareFolder string `mapstructure:"share_folder"`

	// Location of the eos binary.
	// Default is /usr/bin/eos.
	EosBinary string `mapstructure:"eos_binary"`

	// Location of the xrdcopy binary.
	// Default is /opt/eos/xrootd/bin/xrdcopy.
	XrdcopyBinary string `mapstructure:"xrdcopy_binary"`

	// URL of the Master EOS MGM.
	// Default is root://eos-example.org
	MasterURL string `mapstructure:"master_url"`

	// URL of the Slave EOS MGM.
	// Default is root://eos-example.org
	SlaveURL string `mapstructure:"slave_url"`

	// Location on the local fs where to store reads.
	// Defaults to os.TempDir()
	CacheDirectory string `mapstructure:"cache_directory"`

	// SecProtocol specifies the xrootd security protocol to use between the server and EOS.
	SecProtocol string `mapstructure:"sec_protocol"`

	// Keytab specifies the location of the keytab to use to authenticate to EOS.
	Keytab string `mapstructure:"keytab"`

	// SingleUsername is the username to use when SingleUserMode is enabled
	SingleUsername string `mapstructure:"single_username"`

	// UserLayout wraps the internal path with user information.
	// Example: if conf.Namespace is /eos/user and received path is /docs
	// and the UserLayout is {{.Username}} the internal path will be:
	// /eos/user/<username>/docs
	UserLayout string `mapstructure:"user_layout"`

	// Enables logging of the commands executed
	// Defaults to false
	EnableLogging bool `mapstructure:"enable_logging"`

	// ShowHiddenSysFiles shows internal EOS files like
	// .sys.v# and .sys.a# files.
	ShowHiddenSysFiles bool `mapstructure:"show_hidden_sys_files"`

	// ForceSingleUserMode will force connections to EOS to use SingleUsername
	ForceSingleUserMode bool `mapstructure:"force_single_user_mode"`

	// UseKeyTabAuth changes will authenticate requests by using an EOS keytab.
	UseKeytab bool `mapstructure:"use_keytab"`

	// EnableHome enables the creation of home directories.
	EnableHome bool `mapstructure:"enable_home"`

	// Whether to maintain the same inode across various versions of a file.
	// Requires extra metadata operations if set to true
	VersionInvariant bool `mapstructure:"version_invariant"`

	// UseGRPC controls whether we spawn eosclient processes or use GRPC to connect to EOS.
	UseGRPC bool `mapstructure:"use_grpc"`

	// GatewaySvc stores the endpoint at which the GRPC gateway is exposed.
	GatewaySvc string `mapstructure:"gatewaysvc"`

	// GRPCAuthkey is the key that authorizes this client to connect to the GRPC service
	// It's unclear whether this will be the final solution
	GRPCAuthkey string `mapstructure:"grpc_auth_key"`

	// URI of the EOS MGM grpc server
	// Default is empty
	GrpcURI string `mapstructure:"master_grpc_uri"`

	// Size of the cache used to store user ID and UID resolution.
	// Default value is 1000000.
	UserIDCacheSize int `mapstructure:"user_id_cache_size"`

	// The depth, starting from root, that we'll parse directories to lookup the
	// owner and warm up the cache. For example, for a layout of {{substr 0 1 .Username}}/{{.Username}}
	// and a depth of 2, we'll lookup each user's home directory.
	// Default value is 2.
	UserIDCacheWarmupDepth int `mapstructure:"user_id_cache_warmup_depth"`

	// Normally the eosgrpc plugin streams data on the fly.
	// Setting this to true will make reva use the temp cachedirectory
	// as intermediate step for read operations
	ReadUsesLocalTemp bool `mapstructure:"read_uses_local_temp"`

	// Normally the eosgrpc plugin streams data on the fly.
	// Setting this to true will make reva use the temp cachedirectory
	// as intermediate step for write operations
	// Beware: in pure streaming mode the FST must support
	// the HTTP chunked encoding
	WriteUsesLocalTemp bool `mapstructure:"write_uses_local_temp"`

	// Whether to allow recycle operations on base paths.
	// If set to true, we'll look up the owner of the passed path and perform
	// operations on that user's recycle bin.
	// Only considered when EnableHome is false.
	AllowPathRecycleOperations bool `mapstructure:"allow_path_recycle_operations"`

	// Whether we should impersonate the owner of a resource when trying to perform
	// revisions-related operations.
	ImpersonateOwnerforRevisions bool `mapstructure:"impersonate_owner_for_revisions"`

	// HTTP connections to EOS: max number of idle conns
	MaxIdleConns int `mapstructure:"max_idle_conns"`

	// HTTP connections to EOS: max number of conns per host
	MaxConnsPerHost int `mapstructure:"max_conns_per_host"`

	// HTTP connections to EOS: max number of idle conns per host
	MaxIdleConnsPerHost int `mapstructure:"max_idle_conns_per_host"`

	// HTTP connections to EOS: idle conections TTL
	IdleConnTimeout int `mapstructure:"idle_conn_timeout"`

	// HTTP connections to EOS: client certificate (usually a X509 host certificate)
	ClientCertFile string `mapstructure:"http_client_certfile"`
	// HTTP connections to EOS: client certificate key (usually a X509 host certificate)
	ClientKeyFile string `mapstructure:"http_client_keyfile"`
	// HTTP connections to EOS: CA directories
	ClientCADirs string `mapstructure:"http_client_cadirs"`
	// HTTP connections to EOS: CA files
	ClientCAFiles string `mapstructure:"http_client_cafiles"`

	// TokenExpiry stores in seconds the time after which generated tokens will expire
	// Default is 3600
	TokenExpiry int `mapstructure:"token_expiry"`

	SpacesConfig SpacesConfig `mapstructure:"spaces_config"`
}
