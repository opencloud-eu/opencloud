package thumbnail

import (
	"testing"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

func resourceInfo(mime string, meta map[string]string) *provider.ResourceInfo {
	ri := &provider.ResourceInfo{MimeType: mime}
	if meta != nil {
		ri.ArbitraryMetadata = &provider.ArbitraryMetadata{Metadata: meta}
	}
	return ri
}

func TestHasPreview(t *testing.T) {
	cases := []struct {
		name string
		md   *provider.ResourceInfo
		want bool
	}{
		{"nil", nil, false},
		{"unconditional image", resourceInfo("image/png", nil), true},
		{"unconditional text", resourceInfo("text/plain", nil), true},
		{"unsupported type", resourceInfo("application/pdf", nil), false},
		{"audio without preview dims", resourceInfo("audio/mpeg", nil), false},
		{"audio with empty dims", resourceInfo("audio/mpeg", map[string]string{
			PreviewWidthKey: "0", PreviewHeightKey: "0",
		}), false},
		{"audio with preview dims", resourceInfo("audio/mpeg", map[string]string{
			PreviewWidthKey: "500", PreviewHeightKey: "500",
		}), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasPreview(tc.md); got != tc.want {
				t.Errorf("HasPreview(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestSupportedMimeTypesIsUnion(t *testing.T) {
	for k := range UnconditionalPreviewMimeTypes {
		if _, ok := SupportedMimeTypes[k]; !ok {
			t.Errorf("SupportedMimeTypes missing unconditional type %q", k)
		}
	}
	for k := range EmbeddedPreviewMimeTypes {
		if _, ok := SupportedMimeTypes[k]; !ok {
			t.Errorf("SupportedMimeTypes missing embedded type %q", k)
		}
	}
}
