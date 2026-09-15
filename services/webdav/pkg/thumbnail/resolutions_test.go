package thumbnail

import (
	"image"
	"testing"
)

func TestInitWithEmptyArray(t *testing.T) {
	rs, err := ParseResolutions([]string{})
	if err != nil {
		t.Errorf("Init with an empty array should not fail. Error: %s.\n", err.Error())
	}
	if rs == nil || len(rs.landscape)+len(rs.portrait) != 0 {
		t.Error("Init with an empty array should return an empty Resolutions instance.\n")
	}
}

func TestInitWithNil(t *testing.T) {
	rs, err := ParseResolutions(nil)
	if err != nil {
		t.Errorf("Init with nil parameter should not fail. Error: %s.\n", err.Error())
	}
	if rs == nil || len(rs.landscape)+len(rs.portrait) != 0 {
		t.Error("Init with nil parameter should return an empty Resolutions instance.\n")
	}
}

func TestInitWithInvalidValuesInArray(t *testing.T) {
	_, err := ParseResolutions([]string{"invalid"})
	if err == nil {
		t.Error("Init with invalid parameter should fail.\n")
	}
}

func TestInit(t *testing.T) {
	rs, err := ParseResolutions([]string{"16x16"})
	if err != nil {
		t.Errorf("Init with valid parameter should not fail. Error: %s.\n", err.Error())
	}
	if len(rs.landscape) != 1 {
		t.Errorf("resolutions has %d landscape entries, expected 1.\n", len(rs.landscape))
	}
}

func TestInitPartitionsByOrientation(t *testing.T) {
	rs, err := ParseResolutions([]string{"16x16", "280x500", "500x280"})
	if err != nil {
		t.Fatalf("Init with valid parameter should not fail. Error: %s.\n", err.Error())
	}
	if len(rs.landscape) != 2 { // 16x16 (square) + 500x280
		t.Errorf("expected 2 landscape entries, got %d\n", len(rs.landscape))
	}
	if len(rs.portrait) != 1 { // 280x500
		t.Errorf("expected 1 portrait entry, got %d\n", len(rs.portrait))
	}
}

func TestMatchWithEmptyResolutions(t *testing.T) {
	rs, _ := ParseResolutions(nil)
	want := image.Rect(0, 0, 24, 24)

	r := rs.Match(want)
	if r != want {
		t.Errorf("Match from empty resolutions should return the given resolution, got %dx%d", r.Dx(), r.Dy())
	}
}

func TestMatch(t *testing.T) {
	rs, _ := ParseResolutions([]string{"16x16", "32x32", "64x64", "128x128", "500x280", "280x500"})

	testData := []struct {
		requested image.Rectangle
		expected  image.Rectangle
	}{
		// landscape requests (width >= height) match the landscape group
		{image.Rect(0, 0, 17, 17), image.Rect(0, 0, 32, 32)},    // snap up from 16
		{image.Rect(0, 0, 24, 24), image.Rect(0, 0, 32, 32)},    // exact-ish -> next higher
		{image.Rect(0, 0, 80, 20), image.Rect(0, 0, 128, 128)},  // dominant 80 -> 128x128
		{image.Rect(0, 0, 1024, 1), image.Rect(0, 0, 500, 280)}, // exceeds all -> largest landscape
		// portrait requests (height > width) match the portrait group
		{image.Rect(0, 0, 1, 1024), image.Rect(0, 0, 280, 500)}, // dominant 1024 -> 280x500
		{image.Rect(0, 0, 20, 80), image.Rect(0, 0, 280, 500)},  // dominant 80 -> 280x500
	}

	for _, row := range testData {
		match := rs.Match(row.requested)
		if match != row.expected {
			t.Errorf("Match(%dx%d) = %dx%d, expected %dx%d",
				row.requested.Dx(), row.requested.Dy(), match.Dx(), match.Dy(), row.expected.Dx(), row.expected.Dy())
		}
	}
}

// TestMatchSquareResolutions verifies that square requests map onto the square
// resolutions (the boxes the web UI actually requests), and that the closest-fit
// rule picks the smallest resolution still at least as large as requested.
func TestMatchSquareResolutions(t *testing.T) {
	rs, _ := ParseResolutions([]string{
		"16x16", "32x32", "64x64", "128x128",
		"320x320",
		"500x280", "280x500", "1000x560", "560x1000",
		"1024x1024",
		"512x2048", "1080x1920", "1920x1080",
		"2160x3840", "3840x2160", "4320x7680", "7680x4320",
	})

	testData := []struct {
		requested image.Rectangle
		expected  image.Rectangle
	}{
		// square requests map onto square resolutions
		{image.Rect(0, 0, 16, 16), image.Rect(0, 0, 16, 16)},
		{image.Rect(0, 0, 32, 32), image.Rect(0, 0, 32, 32)},
		{image.Rect(0, 0, 320, 320), image.Rect(0, 0, 320, 320)},
		{image.Rect(0, 0, 1024, 1024), image.Rect(0, 0, 1024, 1024)},
		// closest fit: smallest resolution still >= requested dominant dim
		{image.Rect(0, 0, 1920, 1080), image.Rect(0, 0, 1920, 1080)},
		{image.Rect(0, 0, 1080, 1920), image.Rect(0, 0, 1080, 1920)},
	}

	for _, row := range testData {
		match := rs.Match(row.requested)
		if match != row.expected {
			t.Errorf("Match(%dx%d) = %dx%d, expected %dx%d",
				row.requested.Dx(), row.requested.Dy(), match.Dx(), match.Dy(), row.expected.Dx(), row.expected.Dy())
		}
	}
}

func TestParseWithEmptyString(t *testing.T) {
	if _, err := ParseResolution(""); err == nil {
		t.Error("Parse with empty string should return an error.")
	}
}

func TestParseWithInvalidWidth(t *testing.T) {
	_, err := ParseResolution("invalidx42")
	if err == nil {
		t.Error("Parse with invalid width should return an error.")
	}
}

func TestParseWithInvalidHeight(t *testing.T) {
	_, err := ParseResolution("42xinvalid")
	if err == nil {
		t.Error("Parse with invalid height should return an error.")
	}
}

func TestParseResolution(t *testing.T) {
	rStr := "42x23"
	r, _ := ParseResolution(rStr)
	if r.Dx() != 42 || r.Dy() != 23 {
		t.Errorf("Expected resolution %s got %s", rStr, r.String())
	}
}
