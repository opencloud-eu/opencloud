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

package propagator

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/decomposedfs/tree/propagator")
}

type Propagator interface {
	Propagate(ctx context.Context, node *node.Node, sizediff int64) error
}

func New(lookup node.PathLookup, o *options.Options, log *zerolog.Logger) Propagator {
	switch o.Propagator {
	case "async":
		return NewAsyncPropagator(o.TreeSizeAccounting, o.TreeTimeAccounting, o.AsyncPropagatorOptions, lookup, log)
	default:
		return NewSyncPropagator(o.TreeSizeAccounting, o.TreeTimeAccounting, lookup)
	}
}

func calculateTreeSize(ctx context.Context, lookup node.PathLookup, n *node.Node) (uint64, error) {
	childrenPath := n.InternalPath()

	ctx, span := tracer.Start(ctx, "calculateTreeSize")
	defer span.End()
	var size uint64

	f, err := os.Open(childrenPath)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("childrenPath", childrenPath).Msg("could not open dir")
		return 0, err
	}
	defer f.Close()

	names, err := f.Readdirnames(0)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("childrenPath", childrenPath).Msg("could not read dirnames")
		return 0, err
	}
	for i := range names {
		cPath := filepath.Join(childrenPath, names[i])

		// raw read of the attributes for performance reasons
		nodeID, err := node.ReadChildNodeFromLink(ctx, cPath)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("childpath", cPath).Msg("could not read child node")
			continue // continue after an error
		}
		n := node.NewBaseNode(n.SpaceID, nodeID, lookup)
		attribs, err := lookup.MetadataBackend().All(ctx, n)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("childpath", cPath).Msg("could not read attributes of child entry")
			continue // continue after an error
		}
		sizeAttr := ""
		if string(attribs[prefixes.TypeAttr]) == strconv.FormatUint(uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE), 10) {
			sizeAttr = string(attribs[prefixes.BlobsizeAttr])
		} else {
			sizeAttr = string(attribs[prefixes.TreesizeAttr])
		}
		csize, err := strconv.ParseInt(sizeAttr, 10, 64)
		if err != nil {
			return 0, errors.Wrapf(err, "invalid blobsize xattr format")
		}
		size += uint64(csize)
	}
	return size, err
}
