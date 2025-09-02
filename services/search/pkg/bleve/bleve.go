package bleve

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2/search"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"google.golang.org/protobuf/types/known/timestamppb"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var queryEscape = regexp.MustCompile(`([` + regexp.QuoteMeta(`+=&|><!(){}[]^\"~*?:\/`) + `\-\s])`)

func getFieldValue[T any](m map[string]interface{}, key string) (out T) {
	val, ok := m[key]
	if !ok {
		return
	}

	out, _ = val.(T)

	return
}

func resourceIDtoSearchID(id storageProvider.ResourceId) *searchMessage.ResourceID {
	return &searchMessage.ResourceID{
		StorageId: id.GetStorageId(),
		SpaceId:   id.GetSpaceId(),
		OpaqueId:  id.GetOpaqueId()}
}

func getFieldSliceValue[T any](m map[string]interface{}, key string) (out []T) {
	iv := getFieldValue[interface{}](m, key)
	add := func(v interface{}) {
		cv, ok := v.(T)
		if !ok {
			return
		}

		out = append(out, cv)
	}

	// bleve tend to convert []string{"foo"} to type string if slice contains only one value
	// bleve: []string{"foo"} -> "foo"
	// bleve: []string{"foo", "bar"} -> []string{"foo", "bar"}
	switch v := iv.(type) {
	case T:
		add(v)
	case []interface{}:
		for _, rv := range v {
			add(rv)
		}
	}

	return
}

func getFragmentValue(m bleveSearch.FieldFragmentMap, key string, idx int) string {
	val, ok := m[key]
	if !ok {
		return ""
	}

	if len(val) <= idx {
		return ""
	}

	return val[idx]
}

func getAudioValue[T any](fields map[string]interface{}) *T {
	if !strings.HasPrefix(getFieldValue[string](fields, "MimeType"), "audio/") {
		return nil
	}

	var audio = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(audio, fields, "audio."); ok {
		return audio
	}

	return nil
}

func getImageValue[T any](fields map[string]interface{}) *T {
	var image = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(image, fields, "image."); ok {
		return image
	}

	return nil
}

func getLocationValue[T any](fields map[string]interface{}) *T {
	var location = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(location, fields, "location."); ok {
		return location
	}

	return nil
}

func getPhotoValue[T any](fields map[string]interface{}) *T {
	var photo = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(photo, fields, "photo."); ok {
		return photo
	}

	return nil
}

func newPointerOfType[T any]() *T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	ptr := reflect.New(t).Interface()
	return ptr.(*T)
}

func unmarshalInterfaceMap(out any, flatMap map[string]interface{}, prefix string) bool {
	nonEmpty := false
	obj := reflect.ValueOf(out).Elem()
	for i := 0; i < obj.NumField(); i++ {
		field := obj.Field(i)
		structField := obj.Type().Field(i)
		mapKey := prefix + getFieldName(structField)

		if value, ok := flatMap[mapKey]; ok {
			if field.Kind() == reflect.Ptr {
				alloc := reflect.New(field.Type().Elem())
				elemType := field.Type().Elem()

				// convert time strings from index for search requests
				if elemType == reflect.TypeOf(timestamppb.Timestamp{}) {
					if strValue, ok := value.(string); ok {
						if parsedTime, err := time.Parse(time.RFC3339, strValue); err == nil {
							alloc.Elem().Set(reflect.ValueOf(*timestamppb.New(parsedTime)))
							field.Set(alloc)
							nonEmpty = true
						}
					}
					continue
				}

				// convert time strings from index for libregraph structs when updating resources
				if elemType == reflect.TypeOf(time.Time{}) {
					if strValue, ok := value.(string); ok {
						if parsedTime, err := time.Parse(time.RFC3339, strValue); err == nil {
							alloc.Elem().Set(reflect.ValueOf(parsedTime))
							field.Set(alloc)
							nonEmpty = true
						}
					}
					continue
				}

				alloc.Elem().Set(reflect.ValueOf(value).Convert(elemType))
				field.Set(alloc)
				nonEmpty = true
			}
		}
	}

	return nonEmpty
}

func getFieldName(structField reflect.StructField) string {
	tag := structField.Tag.Get("json")
	if tag == "" {
		return structField.Name
	}

	return strings.Split(tag, ",")[0]
}

func matchToResource(match *bleveSearch.DocumentMatch) *search.Resource {
	return &search.Resource{
		ID:       getFieldValue[string](match.Fields, "ID"),
		RootID:   getFieldValue[string](match.Fields, "RootID"),
		Path:     getFieldValue[string](match.Fields, "Path"),
		ParentID: getFieldValue[string](match.Fields, "ParentID"),
		Type:     uint64(getFieldValue[float64](match.Fields, "Type")),
		Deleted:  getFieldValue[bool](match.Fields, "Deleted"),
		Document: content.Document{
			Name:     getFieldValue[string](match.Fields, "Name"),
			Title:    getFieldValue[string](match.Fields, "Title"),
			Size:     uint64(getFieldValue[float64](match.Fields, "Size")),
			Mtime:    getFieldValue[string](match.Fields, "Mtime"),
			MimeType: getFieldValue[string](match.Fields, "MimeType"),
			Content:  getFieldValue[string](match.Fields, "Content"),
			Tags:     getFieldSliceValue[string](match.Fields, "Tags"),
			Audio:    getAudioValue[libregraph.Audio](match.Fields),
			Image:    getImageValue[libregraph.Image](match.Fields),
			Location: getLocationValue[libregraph.GeoCoordinates](match.Fields),
			Photo:    getPhotoValue[libregraph.Photo](match.Fields),
		},
	}
}

func escapeQuery(s string) string {
	return queryEscape.ReplaceAllString(s, "\\$1")
}
