package mapping

import (
	"sort"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/openextension"
)

// An open extension property is indexed under the sibling of its value's
// kind, ext.<extensionName>.<property>.@<sibling>, so it can change its kind
// between writes without a mapping change. The sibling segment starts with
// "@", which neither a name nor a property may, so a leaf never collides with
// an object of another extension in the dotted tree OpenSearch builds.
const (
	OpenExtensionsRoot = "ext"
	// OpenExtensionsStoredField holds the stored values by metadata key, so an
	// engine can rebuild the document from a hit.
	OpenExtensionsStoredField = "OpenExtensions"
	// OpenExtensionsQueryPrefix: KQL addresses extensions.<extensionName>.<property>
	OpenExtensionsQueryPrefix = "extensions."

	SiblingKeyword = "@keyword"
	SiblingLower   = "@lower"
	SiblingNumber  = "@number"
	SiblingBool    = "@bool"
	SiblingDate    = "@date"
	SiblingGeo     = "@geo"
)

// OpenExtensionField is the indexed field name of one typed sibling.
func OpenExtensionField(name, property, sibling string) string {
	return OpenExtensionsRoot + "." + name + "." + property + "." + sibling
}

// OpenExtensionFieldFromQuery turns a KQL key into the sibling's field name. The
// prefix is matched case-insensitively, name and property are taken as written.
func OpenExtensionFieldFromQuery(key, sibling string) (string, bool) {
	rest, ok := openExtensionPath(key)
	if !ok {
		return "", false
	}
	return OpenExtensionsRoot + "." + rest + "." + sibling, true
}

// IsOpenExtensionQueryField reports whether a KQL key addresses an extension property.
func IsOpenExtensionQueryField(key string) bool {
	_, ok := openExtensionPath(key)
	return ok
}

func openExtensionPath(key string) (string, bool) {
	if len(key) <= len(OpenExtensionsQueryPrefix) || !strings.EqualFold(key[:len(OpenExtensionsQueryPrefix)], OpenExtensionsQueryPrefix) {
		return "", false
	}
	rest := key[len(OpenExtensionsQueryPrefix):]
	// the property is the last segment, the name needs at least two more
	if strings.Count(rest, ".") < 2 || strings.HasPrefix(rest, ".") || strings.HasSuffix(rest, ".") {
		return "", false
	}
	return rest, true
}

// OpenExtensionLeaf is one typed sibling value, the unit both engines index.
type OpenExtensionLeaf struct {
	Field   string
	Sibling string

	Strings []string
	Numbers []float64
	Bools   []bool
	Times   []time.Time
	Geo     *openextension.GeoPoint
}

// OpenExtensionLeaves flattens stored extension properties (metadata key to
// stored value) into the leaves to index, sorted by field. Unreadable values
// are skipped, the index must never fail on user data.
func OpenExtensionLeaves(values map[string]string) []OpenExtensionLeaf {
	var leaves []OpenExtensionLeaf
	for key, raw := range values {
		name, property, ok := openextension.SplitKey(key)
		if !ok {
			continue
		}
		v, err := openextension.DecodeValue(raw)
		if err != nil {
			continue
		}
		leaves = append(leaves, propertyLeaves(name, property, v)...)
	}
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].Field < leaves[j].Field })
	return leaves
}

func propertyLeaves(name, property string, v openextension.Value) []OpenExtensionLeaf {
	leaf := func(sibling string) OpenExtensionLeaf {
		return OpenExtensionLeaf{Field: OpenExtensionField(name, property, sibling), Sibling: sibling}
	}
	switch v.Kind {
	case openextension.KindString:
		values := v.Strings()
		if len(values) == 0 {
			return nil
		}
		lower := make([]string, len(values))
		for i, s := range values {
			lower[i] = strings.ToLower(s)
		}
		keyword := leaf(SiblingKeyword)
		keyword.Strings = values
		lowered := leaf(SiblingLower)
		lowered.Strings = lower
		return []OpenExtensionLeaf{keyword, lowered}
	case openextension.KindNumber:
		l := leaf(SiblingNumber)
		l.Numbers = v.Numbers()
		if len(l.Numbers) == 0 {
			return nil
		}
		return []OpenExtensionLeaf{l}
	case openextension.KindBool:
		l := leaf(SiblingBool)
		l.Bools = v.Bools()
		if len(l.Bools) == 0 {
			return nil
		}
		return []OpenExtensionLeaf{l}
	case openextension.KindDate:
		l := leaf(SiblingDate)
		l.Times = v.Times()
		if len(l.Times) == 0 {
			return nil
		}
		return []OpenExtensionLeaf{l}
	case openextension.KindGeo:
		p, ok := v.Geo()
		if !ok {
			return nil
		}
		l := leaf(SiblingGeo)
		l.Geo = &p
		return []OpenExtensionLeaf{l}
	}
	return nil
}
