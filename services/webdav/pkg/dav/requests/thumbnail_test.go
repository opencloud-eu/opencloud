package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/constants"
)

func newThumbnailRequest(t *testing.T, rawQuery string) *http.Request {
	t.Helper()
	req := httptest.NewRequest("GET", "/files/test.png?"+rawQuery, nil)
	ctx := context.WithValue(req.Context(), constants.ContextKeyPath, "/files/test.png")
	return req.WithContext(ctx)
}

func TestParseThumbnailRequestAspect(t *testing.T) {
	tests := []struct {
		name  string
		query string
		wantA bool
	}{
		{"a absent defaults to preserve aspect", "x=1024&y=1024", true},
		{"a=1 preserves aspect", "x=1024&y=1024&a=1", true},
		{"a=0 fills the box", "x=1024&y=1024&a=0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newThumbnailRequest(t, tt.query)
			tr, err := ParseThumbnailRequest(r)
			if err != nil {
				t.Fatalf("ParseThumbnailRequest() error = %v", err)
			}
			if tr.Aspect != tt.wantA {
				t.Errorf("Aspect = %v, want %v", tr.Aspect, tt.wantA)
			}
		})
	}
}

func TestParseThumbnailRequestProcessor(t *testing.T) {
	r := newThumbnailRequest(t, "x=320&y=320&processor=fit")
	tr, err := ParseThumbnailRequest(r)
	if err != nil {
		t.Fatalf("ParseThumbnailRequest() error = %v", err)
	}
	if tr.Processor != "fit" {
		t.Errorf("Processor = %q, want %q", tr.Processor, "fit")
	}
	if tr.Width != 320 || tr.Height != 320 {
		t.Errorf("dimensions = %dx%d, want 320x320", tr.Width, tr.Height)
	}
}
