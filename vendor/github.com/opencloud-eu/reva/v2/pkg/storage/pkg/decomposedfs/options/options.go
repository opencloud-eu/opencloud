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

package options

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/pkg/errors"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {

	// the gateway address
	GatewayAddr string `mapstructure:"gateway_addr"`

	// the metadata backend to use, currently supports `xattr` or `ini`
	MetadataBackend string `mapstructure:"metadata_backend"`

	// the attribute key prefix of the metadata on disk, process wide, see prefixes.SetOcPrefix
	MetadataPrefix string `mapstructure:"metadata_prefix"`

	// the propagator to use for this fs. currently only `sync` is fully supported, `async` is available as an experimental feature
	Propagator string `mapstructure:"propagator"`
	// Options specific to the async propagator
	AsyncPropagatorOptions AsyncPropagatorOptions `mapstructure:"async_propagator_options"`

	// decomposedfs fs works on top of a dir of uuid nodes
	Root string `mapstructure:"root"`

	// the upload directory where uploads in progress are stored
	UploadDirectory string `mapstructure:"upload_directory"`

	// UserLayout describes the relative path from the storage's root node to the users home node.
	UserLayout string `mapstructure:"user_layout"`

	// ProjectLayout describes the relative path from the storage's root node to the project spaces root directory.
	ProjectLayout string `mapstructure:"project_layout"`

	// propagate mtime changes as tmtime (tree modification time) to the parent directory when user.oc.propagation=1 is set on a node
	TreeTimeAccounting bool `mapstructure:"treetime_accounting"`

	// propagate size changes as treesize
	TreeSizeAccounting bool `mapstructure:"treesize_accounting"`

	// permissions service to use when checking permissions
	PermissionsSVC           string `mapstructure:"permissionssvc"`
	PermissionsClientTLSMode string `mapstructure:"permissionssvc_tls_mode"`
	PermTLSMode              pool.TLSMode

	PersonalSpaceAliasTemplate string `mapstructure:"personalspacealias_template"`
	PersonalSpacePathTemplate  string `mapstructure:"personalspacepath_template"`
	GeneralSpaceAliasTemplate  string `mapstructure:"generalspacealias_template"`
	GeneralSpacePathTemplate   string `mapstructure:"generalspacepath_template"`

	AsyncFileUploads bool `mapstructure:"asyncfileuploads"`

	Events EventOptions `mapstructure:"events"`

	Tokens TokenOptions `mapstructure:"tokens"`

	StatCache         cache.Config `mapstructure:"statcache"`
	FileMetadataCache cache.Config `mapstructure:"filemetadatacache"`
	IDCache           cache.Config `mapstructure:"idcache"`

	MaxAcquireLockCycles    int `mapstructure:"max_acquire_lock_cycles"`
	LockCycleDurationFactor int `mapstructure:"lock_cycle_duration_factor"`
	MaxConcurrency          int `mapstructure:"max_concurrency"`

	MaxQuota uint64 `mapstructure:"max_quota"`

	DisableVersioning bool `mapstructure:"disable_versioning"`

	// RevisionsCleanupStrategy defines which old revisions of a file are deleted after a new
	// revision has been created, e.g. "auto", "14d, auto", "auto, 90d" or "7d, auto, 90d".
	// Leaving it empty or setting it to "disabled" keeps all revisions.
	//
	// The supported forms, where X is a duration with a unit suffix (w, d, h, m, s) or
	// a plain number of revisions, which always refers to the newest revisions:
	//   - "auto": thin out revisions the older they get, keeping e.g. one per 10
	//     seconds in the last minute, one per minute in the last hour, one per hour in
	//     the last day, one per day in the last 30 days and one per week beyond
	//   - "X": delete everything beyond X, e.g. "10" or "30d"; "0" deletes all revisions,
	//     which, unlike disable_versioning, also removes the ones accumulated so far
	//   - "X, auto": keep everything within X, apply "auto" to the older revisions
	//   - "auto, X": delete everything beyond X, apply "auto" to the rest
	//   - "X1, X2": keep everything within X1, delete everything beyond X2; one must be
	//     a number and the other a duration, e.g. "5, 30d" or "7d, 10"
	//   - "X1, auto, X2": keep everything within X1, delete everything beyond X2 and
	//     apply "auto" in between, e.g. "7d, auto, 90d"
	//   - "disabled": never delete any revisions
	//
	// A number of revisions to keep counts the ones within X1 too: "7d, 10" keeps at
	// most 10 revisions, unless more than 10 are younger than 7 days.
	//
	// The age of a revision is the time its content was written, i.e. the modification time
	// the file had before it was changed, not the time of that change, which turned the
	// content into a revision. A revision created today from content that was unchanged for
	// two months is two months old, so e.g. "30d" deletes it right away; combine
	// durations with a number of revisions, e.g. "5, 30d", to always keep the newest ones.
	//
	// A duration in days or weeks with an uppercase unit (D, W) is aligned to midnight in the
	// server's local time zone: "1d" means the last 24 hours, "1D" since yesterday 00:00,
	// "0D" since today 00:00, e.g. "1D, auto, 30d".
	//
	// See revisions.NewRevisionCleanupStrategy in pkg/storage/revisions for the exact
	// grammar and pkg/storage/pkg/decomposedfs/revisioncleanup for when it is applied.
	// It is shared by all decomposedfs based drivers, including posix, and also applies
	// to revisions posix creates from changes on disk (enable_fs_revisions).
	// An invalid value makes the driver fail at startup.
	RevisionsCleanupStrategy string `mapstructure:"revisions_cleanup_strategy"`

	// RevisionsCleanupAsync runs the revision cleanup in the background instead of as part of
	// the upload, restore or change on disk that created the new revision, so that those are not
	// slowed down by it. Defaults to true.
	RevisionsCleanupAsync bool `mapstructure:"revisions_cleanup_async"`
	// RevisionsCleanupAsyncWorkers is the number of background workers for the revision cleanup.
	// 0, the default, determines it automatically: half the available CPUs (which respects
	// container CPU limits), but at least 2.
	RevisionsCleanupAsyncWorkers int `mapstructure:"revisions_cleanup_async_workers"`
	// RevisionsCleanupAsyncQueueSize is the number of nodes that can wait for a background
	// revision cleanup. 0, the default, determines it automatically from the number of workers.
	RevisionsCleanupAsyncQueueSize int `mapstructure:"revisions_cleanup_async_queue_size"`
	// RevisionsCleanupAsyncQueueFull defines what happens when a new revision is created while
	// the queue is full: "drop" (the default) skips the cleanup of that node and logs it, the
	// node is cleaned up again when it gets its next revision; "block" waits until there is
	// room, for at most RevisionsCleanupAsyncQueueFullTimeout, so that no cleanup is skipped
	// unless the queue stays full that long, at the cost of slowing down writes under heavy
	// load.
	RevisionsCleanupAsyncQueueFull string `mapstructure:"revisions_cleanup_async_queue_full"`
	// RevisionsCleanupAsyncQueueFullTimeout is how long "block" waits for room in a full queue,
	// before the cleanup of the node is skipped as with "drop". 0 waits as long as it takes.
	RevisionsCleanupAsyncQueueFullTimeout time.Duration `mapstructure:"revisions_cleanup_async_queue_full_timeout"`

	MountID string `mapstructure:"mount_id"`

	MultiTenantEnabled bool `mapstructure:"multi_tenant_enabled"`
}

// AsyncPropagatorOptions holds the configuration for the async propagator
type AsyncPropagatorOptions struct {
	PropagationDelay time.Duration `mapstructure:"propagation_delay"`
}

// EventOptions are the configurable options for events
type EventOptions struct {
	NumConsumers int `mapstructure:"numconsumers"`
}

// TokenOptions are the configurable option for tokens
type TokenOptions struct {
	DownloadEndpoint     string `mapstructure:"download_endpoint"`
	DataGatewayEndpoint  string `mapstructure:"datagateway_endpoint"`
	TransferSharedSecret string `mapstructure:"transfer_shared_secret"`
	TransferExpires      int64  `mapstructure:"transfer_expires"`
}

// New returns a new Options instance for the given configuration
func New(m map[string]interface{}) (*Options, error) {
	o := &Options{
		// defaults for options that are not false/0/"" by default; mapstructure only
		// overwrites the fields that are present in m
		RevisionsCleanupAsync:                 true,
		RevisionsCleanupAsyncQueueFullTimeout: 5 * time.Second,
	}
	if err := mapstructure.Decode(m, o); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}

	o.GatewayAddr = sharedconf.GetGatewaySVC(o.GatewayAddr)

	if o.MetadataBackend == "" {
		o.MetadataBackend = "messagepack"
	}

	// ensure user layout has no starting or trailing /
	o.UserLayout = strings.Trim(o.UserLayout, "/")

	// c.DataDirectory should never end in / unless it is the root
	o.Root = filepath.Clean(o.Root)

	if o.PersonalSpaceAliasTemplate == "" {
		o.PersonalSpaceAliasTemplate = "{{.SpaceType}}/{{.User.Username | lower}}"
	}

	if o.GeneralSpaceAliasTemplate == "" {
		o.GeneralSpaceAliasTemplate = "{{.SpaceType}}/{{.SpaceName | replace \" \" \"-\" | lower}}"
	}

	if o.PermissionsClientTLSMode != "" {
		var err error
		o.PermTLSMode, err = pool.StringToTLSMode(o.PermissionsClientTLSMode)
		if err != nil {
			return nil, err
		}
	} else {
		sharedOpt := sharedconf.GRPCClientOptions()
		var err error

		if o.PermTLSMode, err = pool.StringToTLSMode(sharedOpt.TLSMode); err != nil {
			return nil, err
		}
	}

	if o.MaxConcurrency <= 0 {
		o.MaxConcurrency = 5
	}

	if o.Propagator == "" {
		o.Propagator = "sync"
	}
	if o.AsyncPropagatorOptions.PropagationDelay == 0 {
		o.AsyncPropagatorOptions.PropagationDelay = 5 * time.Second
	}

	if o.UploadDirectory == "" {
		o.UploadDirectory = filepath.Join(o.Root, "uploads")
	}

	return o, nil
}
