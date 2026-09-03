package search

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// SchemaVersion is the shared schema version for both search backends. Bump it
// on a breaking mapping change: each version gets its own index (OpenSearch name
// suffix, bleve path suffix), so the service builds a fresh index instead of
// colliding with the old one. No migration; reindex to populate.
const SchemaVersion = 4

var scopeRegex = regexp.MustCompile(`scope:\s*([^" "\n\r]*)`)

// Engine is the interface to the search engine
type Engine interface {
	Search(ctx context.Context, req *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error)
	DocCount() (uint64, error)

	Upsert(id string, r Resource) error
	Move(id string, parentID string, targetPath string) error
	Delete(id string) error
	Restore(id string) error
	Purge(id string, onlyDeleted bool) error
	PurgeSpace(rootID string) error

	NewBatch(batchSize int) (BatchOperator, error)
}

type BatchOperator interface {
	Upsert(id string, r Resource) error
	Move(id string, parentID string, targetPath string) error
	Delete(id string) error
	Restore(id string) error
	Purge(id string, onlyDeleted bool) error

	Push() error
}

// Resource is the entity that is stored in the index.
type Resource struct {
	content.Document

	ID       string `json:"ID"`
	RootID   string `json:"RootID"`
	Path     string `json:"Path"`
	ParentID string `json:"ParentID"`
	Type     uint64 `json:"Type"`
	Deleted  bool   `json:"Deleted"`
	Hidden   bool   `json:"Hidden"`
}

// resourceFieldOverrides is built once (it never changes) and reused on hot
// paths instead of reallocating per call.
var resourceFieldOverrides = sync.OnceValue(func() map[string]mapping.FieldOpts {
	True, False := true, false
	return map[string]mapping.FieldOpts{
		// every keyword field searches case-insensitively and by word (name,
		// title, the facets) unless opted out: ids are opaque, paths are POSIX,
		// the mime type is normalized already, a tag is one label
		"ID":                  {CaseInsensitive: &False, NoWordBreaker: &True},
		"RootID":              {CaseInsensitive: &False, NoWordBreaker: &True},
		"ParentID":            {CaseInsensitive: &False, NoWordBreaker: &True},
		"Path":                {Type: mapping.TypePath, CaseInsensitive: &False},
		"MimeType":            {CaseInsensitive: &False, NoWordBreaker: &True},
		"Content":             {Type: mapping.TypeFulltext},
		"Tags":                {NoWordBreaker: &True, IncludeInAll: &False},
		"Favorites":           {NoWordBreaker: &True, IncludeInAll: &False, CaseInsensitive: &False}, // opaque user ids
		"livePhoto.contentId": {NoWordBreaker: &True, CaseInsensitive: &False},                       // opaque pairing uuid
		"location":            {Type: mapping.TypeGeopoint},
	}
})

// SearchFieldOverrides returns the field options the mapping package needs to
// build per-backend index mappings for a Resource (keys are json-tag names).
// The map is shared and read-only; clone it before mutating.
func (Resource) SearchFieldOverrides() map[string]mapping.FieldOpts {
	return resourceFieldOverrides()
}

// ResolveReference makes sure the path is relative to the space root
func ResolveReference(ctx context.Context, ref *provider.Reference, ri *provider.ResourceInfo, gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) (*provider.Reference, error) {
	if ref.GetResourceId().GetOpaqueId() == ref.GetResourceId().GetSpaceId() {
		return ref, nil
	}

	gatewayClient, err := gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	gpRes, err := gatewayClient.GetPath(ctx, &provider.GetPathRequest{
		ResourceId: ri.GetId(),
	})
	if err != nil || gpRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, err
	}
	return &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: ref.GetResourceId().GetStorageId(),
			SpaceId:   ref.GetResourceId().GetSpaceId(),
			OpaqueId:  ref.GetResourceId().GetSpaceId(),
		},
		Path: utils.MakeRelativePath(gpRes.GetPath()),
	}, nil
}

type matchArray []*searchmsg.Match

func (ma matchArray) Len() int {
	return len(ma)
}
func (ma matchArray) Swap(i, j int) {
	ma[i], ma[j] = ma[j], ma[i]
}
func (ma matchArray) Less(i, j int) bool {
	return ma[i].GetScore() > ma[j].GetScore()
}

func logDocCount(engine Engine, logger log.Logger) {
	c, err := engine.DocCount()
	if err != nil {
		logger.Error().Err(err).Msg("error getting document count from the index")
	}
	logger.Debug().Interface("count", c).Msg("new document count")
}

func getAuthContext(serviceAccountID string, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], secret string, logger log.Logger) (context.Context, error) {
	gatewayClient, err := gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not get reva gatewayClient")
		return nil, err
	}

	return utils.GetServiceUserContext(serviceAccountID, gatewayClient, secret)
}

func statResource(ctx context.Context, ref *provider.Reference, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], logger log.Logger) (*provider.StatResponse, error) {
	gatewayClient, err := gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	res, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		logger.Error().Err(err).Msg("failed to stat the moved resource")
		return nil, err
	}
	switch res.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		return res, nil
	case rpc.Code_CODE_NOT_FOUND:
		// Resource was moved or deleted in the meantime. ignore.
		return nil, err
	default:
		err := errors.New("failed to stat the moved resource")
		logger.Error().Interface("res", res).Msg(err.Error())
		return nil, err
	}
}

// NOTE: this converts CS3 to WebDAV permissions
// since conversions pkg is reva internal we have no other choice than to duplicate the logic
func convertToWebDAVPermissions(isShared, isMountpoint, isDir bool, p *provider.ResourcePermissions) string {
	if p == nil {
		return ""
	}
	var b strings.Builder
	if isShared {
		fmt.Fprintf(&b, "S")
	}
	if p.GetListContainer() &&
		p.GetListFileVersions() &&
		p.GetListRecycle() &&
		p.GetStat() &&
		p.GetGetPath() &&
		p.GetGetQuota() &&
		p.GetInitiateFileDownload() {
		fmt.Fprintf(&b, "R")
	}
	if isMountpoint {
		fmt.Fprintf(&b, "M")
	}
	if p.GetDelete() {
		fmt.Fprintf(&b, "D")
	}
	if p.GetInitiateFileUpload() &&
		p.GetRestoreFileVersion() &&
		p.GetRestoreRecycleItem() {
		fmt.Fprintf(&b, "NV")
		if !isDir {
			fmt.Fprintf(&b, "W")
		}
	}
	if isDir &&
		p.GetListContainer() &&
		p.GetStat() &&
		p.GetCreateContainer() &&
		p.GetInitiateFileUpload() {
		fmt.Fprintf(&b, "CK")
	}
	if grants.PermissionsEqual(p, conversions.NewSecureViewerRole().CS3ResourcePermissions()) {
		fmt.Fprintf(&b, "X")
	}
	return b.String()
}

// CompleteRootID completes a bare driveId ("storage$space") to the root
// resource id stored in the index. Assumes a space root's opaque id equals
// its space id (true for decomposedfs); elsewhere a bare driveId matches
// nothing, the pinning stays safe either way.
func CompleteRootID(v string) string {
	// leave wildcards alone, "1$*!*" would match nothing
	if strings.Contains(v, "!") || strings.ContainsAny(v, "*?") {
		return v
	}
	if i := strings.LastIndex(v, "$"); i >= 0 && i+1 < len(v) {
		return v + "!" + v[i+1:]
	}
	return v
}

// PinnedRootID returns the single space root the query is pinned to via
// top-level AND driveId/RootID conjuncts, or "". OR, NOT and group-nested
// restrictions never pin: skipping a space would be wrong.
func PinnedRootID(query string) string {
	a, err := kql.Builder{}.Build(query)
	if err != nil {
		return ""
	}
	pinned := ""
	negated := false
	for _, n := range a.Nodes {
		switch node := n.(type) {
		case *ast.OperatorNode:
			if strings.EqualFold(node.Value, "OR") {
				return ""
			}
			negated = strings.EqualFold(node.Value, "NOT")
			continue
		case ast.OperatorNode:
			if strings.EqualFold(node.Value, "OR") {
				return ""
			}
			negated = strings.EqualFold(node.Value, "NOT")
			continue
		}
		var key, value string
		switch node := n.(type) {
		case *ast.StringNode:
			key, value = node.Key, node.Value
		case ast.StringNode:
			key, value = node.Key, node.Value
		}
		if strings.EqualFold(key, "driveid") || strings.EqualFold(key, "rootid") {
			// a negated or wildcard restriction cannot pin a single space
			if negated || strings.ContainsAny(value, "*?") {
				return ""
			}
			v := CompleteRootID(value)
			if pinned != "" && pinned != v {
				return ""
			}
			pinned = v
		}
		negated = false
	}
	return pinned
}

// ParseScope extract a scope value from the query string and returns search, scope strings
func ParseScope(query string) (string, string) {
	match := scopeRegex.FindStringSubmatch(query)
	if len(match) >= 2 {
		cut := match[0]
		return strings.TrimSpace(strings.ReplaceAll(query, cut, "")), strings.TrimSpace(match[1])
	}
	return query, ""
}

// ParseFlags extracts supported flags from the query string and returns the cleaned query and a map of flags
func ParseFlags(query string) (string, []string) {
	supportedFlags := []string{"is:favorite"}
	flags := []string{}
	for _, flag := range supportedFlags {
		if strings.Contains(query, flag) {
			flags = append(flags, flag)
			query = strings.ReplaceAll(query, flag, "")
		}
	}

	return strings.TrimSpace(query), flags
}
