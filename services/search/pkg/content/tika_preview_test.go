package content

import "testing"

func TestGetPreview(t *testing.T) {
	audio := map[string][]string{"Content-Type": {"audio/mpeg"}}
	cover := map[string][]string{
		"Content-Type":     {"image/jpeg"},
		"tiff:ImageWidth":  {"500"},
		"tiff:ImageLength": {"400"},
	}
	coverNoDims := map[string][]string{"Content-Type": {"image/jpeg"}}

	t.Run("audio with embedded cover returns dims", func(t *testing.T) {
		p := getPreview("audio/mpeg", []map[string][]string{audio, cover})
		if p == nil || p.Width != 500 || p.Height != 400 {
			t.Fatalf("expected 500x400, got %+v", p)
		}
	})

	t.Run("audio without cover returns nil", func(t *testing.T) {
		if p := getPreview("audio/mpeg", []map[string][]string{audio}); p != nil {
			t.Fatalf("expected nil, got %+v", p)
		}
	})

	t.Run("audio with cover lacking dims returns nil", func(t *testing.T) {
		if p := getPreview("audio/mpeg", []map[string][]string{audio, coverNoDims}); p != nil {
			t.Fatalf("expected nil, got %+v", p)
		}
	})

	t.Run("non-embedded type is gated out", func(t *testing.T) {
		// an image file is unconditional; its preview is not driven by oc.preview,
		// so getPreview must return nil even though an image meta is present.
		if p := getPreview("image/png", []map[string][]string{cover}); p != nil {
			t.Fatalf("expected nil for non-embedded type, got %+v", p)
		}
	})
}
