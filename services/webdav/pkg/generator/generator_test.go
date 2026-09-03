package generator

import (
	"testing"
)

func TestParseMaxInputFileSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"plain bytes", "1234", 1234, false},
		{"KB", "50KB", 50 * 1024, false},
		{"kB case insensitive", "50kB", 50 * 1024, false},
		{"KiB", "50KiB", 50 * 1024, false},
		{"MB", "5MB", 5 * 1024 * 1024, false},
		{"MiB", "5MiB", 5 * 1024 * 1024, false},
		{"GB", "2GB", 2 * 1024 * 1024 * 1024, false},
		{"GiB", "2GiB", 2 * 1024 * 1024 * 1024, false},
		{"whitespace around value", "  5MB  ", 5 * 1024 * 1024, false},
		{"space before suffix", "5 MB", 5 * 1024 * 1024, false},
		{"invalid input", "abc", 0, true},
		{"negative number", "-5MB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMaxInputFileSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMaxInputFileSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseMaxInputFileSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExtensionInfo(t *testing.T) {
	tests := []struct {
		ext    string
		format string
		ct     string
	}{
		{"gif", "gif", "image/gif"},
		{".gif", "gif", "image/gif"},
		{"png", "png", "image/png"},
		{"jpeg", "jpeg", "image/jpeg"},
		{"jpg", "jpeg", "image/jpeg"},
		{"webp", "jpeg", "image/jpeg"},
		{"", "jpeg", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			info := GetExtensionInfo(tt.ext)
			if info.OutputFormat != tt.format {
				t.Errorf("OutputFormat = %q, want %q", info.OutputFormat, tt.format)
			}
			if info.ContentType != tt.ct {
				t.Errorf("ContentType = %q, want %q", info.ContentType, tt.ct)
			}
		})
	}
}

func TestOutputFormat(t *testing.T) {
	if got := OutputFormat("png"); got != "png" {
		t.Errorf("OutputFormat(\"png\") = %q, want \"png\"", got)
	}
	if got := OutputFormat("jpg"); got != "jpeg" {
		t.Errorf("OutputFormat(\"jpg\") = %q, want \"jpeg\"", got)
	}
}

func TestContentType(t *testing.T) {
	if got := ContentType("gif"); got != "image/gif" {
		t.Errorf("ContentType(\"gif\") = %q, want \"image/gif\"", got)
	}
	if got := ContentType("unknown"); got != "image/jpeg" {
		t.Errorf("ContentType(\"unknown\") = %q, want \"image/jpeg\"", got)
	}
}

func TestGuessExtension(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", "jpg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"application/octet-stream", "jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			if got := GuessExtension(tt.mime); got != tt.want {
				t.Errorf("GuessExtension(%q) = %q, want %q", tt.mime, got, tt.want)
			}
		})
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		width     int32
		height    int32
		operation string
		ext       string
		want      string
	}{
		{
			name:      "fill operation center-crops to the box",
			base:      "http://generator",
			width:     128,
			height:    128,
			operation: OpFill,
			ext:       "jpeg",
			want:      "http://generator/unsafe/128x128/filters:format(jpeg)/",
		},
		{
			name:      "fit-in preserves aspect ratio within the box without upscaling",
			base:      "http://generator",
			width:     1024,
			height:    1024,
			operation: OpFitIn,
			ext:       "jpeg",
			want:      "http://generator/unsafe/fit-in/1024x1024/filters:format(jpeg)/",
		},
		{
			name:      "stretch resizes to the exact box",
			base:      "http://generator/",
			width:     64,
			height:    32,
			operation: OpStretch,
			ext:       "png",
			want:      "http://generator/unsafe/stretch/64x32/filters:format(png)/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildURL(tt.base, tt.width, tt.height, tt.operation, tt.ext)
			if got != tt.want {
				t.Errorf("BuildURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
