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

func TestParseThumbnailRequestDimensions(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantW, wantH int32
		wantErr      bool
	}{
		{name: "both given", query: "x=448&y=448", wantW: 448, wantH: 448},
		{name: "neither given keeps the legacy default", query: "preview=1", wantW: DefaultWidth, wantH: DefaultHeight},
		{name: "an omitted axis is unconstrained", query: "y=512&processor=fit", wantW: 0, wantH: 512},
		{name: "an explicit zero is unconstrained", query: "x=0&y=512&processor=fit", wantW: 0, wantH: 512},
		{name: "negative is rejected", query: "x=-1&y=512", wantErr: true},
		{name: "an unconstrained axis without a processor is fitted, not rejected", query: "y=512", wantW: 0, wantH: 512},
		{name: "both axes unconstrained is rejected", query: "x=0&y=0&processor=fit", wantErr: true},
		{name: "cropping to an unconstrained axis is rejected explicitly", query: "y=512&processor=thumbnail", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := ParseThumbnailRequest(newThumbnailRequest(t, tt.query))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseThumbnailRequest() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseThumbnailRequest() error = %v", err)
			}
			if tr.Width != tt.wantW || tr.Height != tt.wantH {
				t.Errorf("dimensions = %dx%d, want %dx%d", tr.Width, tr.Height, tt.wantW, tt.wantH)
			}
		})
	}
}
