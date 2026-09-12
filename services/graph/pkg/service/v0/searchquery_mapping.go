package svc

import (
	"path"
	"slices"
	"strings"
	"time"

	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

var bucketSortBy = map[string]searchsvc.BucketSortBy{
	"count":       searchsvc.BucketSortBy_BUCKET_SORT_BY_COUNT,
	"keyAsString": searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING,
	"keyAsNumber": searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER,
}

var metricKinds = map[string]searchsvc.MetricKind{
	"sum": searchsvc.MetricKind_METRIC_KIND_SUM,
	"min": searchsvc.MetricKind_METRIC_KIND_MIN,
	"max": searchsvc.MetricKind_METRIC_KIND_MAX,
	"avg": searchsvc.MetricKind_METRIC_KIND_AVG,
}

// driveItemFields are the scalar driveItem properties a hit carries and the
// index fields behind them; a facet property (audio.artist) is its index
// field already.
var driveItemFields = map[string]string{
	"name":                 "Name",
	"size":                 "Size",
	"lastModifiedDateTime": "Mtime",
	"mimeType":             "MimeType",
	"@libre.graph.tags":    "Tags",
}

// indexField resolves a field spelled like the driveItem property, and only
// so: a scalar property by name, a facet property as it is. Any other
// spelling resolves to no field and fails validation.
func indexField(field string) string {
	if index, ok := driveItemFields[field]; ok {
		return index
	}
	if strings.Contains(field, ".") {
		return field
	}
	return ""
}

func resolveFields(opts []*searchsvc.AggregationOption) {
	for _, opt := range opts {
		opt.Field = indexField(opt.Field)
		resolveFields(opt.SubAggregations)
	}
}

func libregraphAggregationsToSearch(in []libregraph.AggregationOption) []*searchsvc.AggregationOption {
	if len(in) == 0 {
		return nil
	}
	out := make([]*searchsvc.AggregationOption, 0, len(in))
	for _, a := range in {
		agg := &searchsvc.AggregationOption{Field: a.Field}
		if a.Size != nil {
			agg.Size = *a.Size
		}
		if a.BucketDefinition != nil {
			agg.BucketDefinition = libregraphBucketDefinitionToSearch(*a.BucketDefinition)
		}
		if len(a.LibreGraphSubAggregations) > 0 {
			agg.SubAggregations = libregraphAggregationsToSearch(a.LibreGraphSubAggregations)
		}
		if md := a.LibreGraphMetricDefinition; md != nil {
			agg.MetricDefinition = &searchsvc.MetricDefinition{Kind: metricKinds[md.Kind]}
		}
		out = append(out, agg)
	}
	return out
}

func libregraphBucketDefinitionToSearch(in libregraph.BucketDefinition) *searchsvc.BucketDefinition {
	bd := &searchsvc.BucketDefinition{
		SortBy: bucketSortBy[in.SortBy],
		Ranges: libregraphRangesToSearch(in.Ranges),
	}
	if in.IsDescending != nil {
		bd.IsDescending = *in.IsDescending
	}
	if in.MinimumCount != nil {
		bd.MinimumCount = *in.MinimumCount
	}
	return bd
}

func libregraphRangesToSearch(in []libregraph.BucketAggregationRange) []*searchsvc.BucketRange {
	if len(in) == 0 {
		return nil
	}
	out := make([]*searchsvc.BucketRange, 0, len(in))
	for _, r := range in {
		out = append(out, &searchsvc.BucketRange{From: r.GetFrom(), To: r.GetTo()})
	}
	return out
}

// searchAggregationsToLibregraph maps the results to their definitions by
// position, at every level: the search service answers one result per
// aggregation, in request order.
func searchAggregationsToLibregraph(in []*searchsvc.AggregationResult, defs []libregraph.AggregationOption) []libregraph.SearchAggregation {
	if len(in) == 0 {
		return nil
	}
	out := make([]libregraph.SearchAggregation, 0, len(defs))
	for i, def := range defs {
		if i >= len(in) {
			break
		}
		agg := libregraph.SearchAggregation{Field: libregraph.PtrString(def.Field)}
		if md := def.LibreGraphMetricDefinition; md != nil {
			agg.LibreGraphMetric = &libregraph.SearchMetric{Kind: libregraph.PtrString(md.Kind)}
			// a metric without a single value has no value
			if m := in[i].GetMetric(); m != nil {
				agg.LibreGraphMetric.Value = m.Value
			}
			out = append(out, agg)
			continue
		}
		agg.Buckets = make([]libregraph.SearchBucket, 0, len(in[i].GetBuckets()))
		for _, b := range in[i].GetBuckets() {
			agg.Buckets = append(agg.Buckets, libregraph.SearchBucket{
				Key:                       libregraph.PtrString(b.GetKey()),
				Count:                     libregraph.PtrInt64(b.GetCount()),
				LibreGraphSubAggregations: searchAggregationsToLibregraph(b.GetSubAggregations(), def.LibreGraphSubAggregations),
			})
		}
		out = append(out, agg)
	}
	return out
}

func searchResourceID(id *searchmsg.ResourceID) string {
	return storagespace.FormatResourceID(&storageprovider.ResourceId{
		StorageId: id.GetStorageId(),
		SpaceId:   id.GetSpaceId(),
		OpaqueId:  id.GetOpaqueId(),
	})
}

func searchEntityToDriveItem(e *searchmsg.Entity, uid string) *libregraph.DriveItem {
	size := int64(e.GetSize())
	di := &libregraph.DriveItem{
		Id:   libregraph.PtrString(searchResourceID(e.GetId())),
		Name: libregraph.PtrString(e.GetName()),
		Size: &size,
	}
	if etag := e.GetEtag(); etag != "" {
		di.ETag = &etag
	}
	if mt := e.GetLastModifiedTime(); mt != nil {
		lm := time.Unix(mt.GetSeconds(), int64(mt.GetNanos())).UTC()
		di.LastModifiedDateTime = &lm
	}
	if e.GetType() == uint64(storageprovider.ResourceType_RESOURCE_TYPE_FILE) && e.GetMimeType() != "" {
		mt := e.GetMimeType()
		di.File = &libregraph.OpenGraphFile{MimeType: &mt}
	}
	if e.GetType() == uint64(storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		di.Folder = &libregraph.Folder{}
	}
	if p := e.GetParentId(); p != nil {
		ref := libregraph.NewItemReference()
		ref.SetDriveId(storagespace.FormatStorageID(p.GetStorageId(), p.GetSpaceId()))
		ref.SetId(searchResourceID(p))
		if refPath := e.GetRef().GetPath(); refPath != "" {
			// the index keeps paths relative to the space root (./dir/file)
			parent := path.Dir(path.Join("/", refPath))
			ref.SetPath(parent)
			if parent != "/" {
				ref.SetName(path.Base(parent))
			}
		}
		di.ParentReference = ref
	}
	di.RemoteItem = searchEntityToRemoteItem(e)
	di.Audio = mapping.FromProto[libregraph.Audio](e.GetAudio())
	di.Image = mapping.FromProto[libregraph.Image](e.GetImage())
	di.Photo = mapping.FromProto[libregraph.Photo](e.GetPhoto())
	di.Location = mapping.FromProto[libregraph.GeoCoordinates](e.GetLocation())
	di.Video = mapping.FromProto[libregraph.Video](e.GetVideo())
	di.LibreGraphMotionPhoto = mapping.FromProto[libregraph.MotionPhoto](e.GetMotionPhoto())
	di.LibreGraphLivePhoto = mapping.FromProto[libregraph.LivePhoto](e.GetLivePhoto())
	if tags := e.GetTags(); len(tags) > 0 {
		di.LibreGraphTags = tags
	}
	if av := e.GetPermissionsActionsAllowedValues(); len(av) > 0 {
		di.LibreGraphPermissionsActionsAllowedValues = av
	}
	// the WebDAV report emits oc:favorite only when the caller favorited the item; mirror that
	if uid != "" && slices.Contains(e.GetFavorites(), uid) {
		di.LibreGraphMeFollowing = libregraph.PtrBool(true)
	}
	return di
}

// remoteItem carries the id in the owner's drive and the mountpoint it is
// reached through; absent for hits from the caller's own spaces
func searchEntityToRemoteItem(e *searchmsg.Entity) *libregraph.RemoteItem {
	id := e.GetRemoteItemId()
	if id == nil {
		return nil
	}

	item := libregraph.NewRemoteItem()
	item.SetId(searchResourceID(id))

	if root := e.GetShareRootName(); root != "" {
		item.SetPath(root)
		item.SetName(path.Base(root))
	}

	return item
}
