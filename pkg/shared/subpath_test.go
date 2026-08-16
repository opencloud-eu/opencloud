package shared

import "testing"

func TestSubPath(t *testing.T) {
	tests := map[string]string{
		"https://example.com":                   "",
		"https://example.com/":                  "",
		"https://example.com/test/opencloud":    "/test/opencloud",
		"https://example.com/test/opencloud/":   "/test/opencloud",
		"https://example.com/test//opencloud//": "/test/opencloud",
		"not a url \x7f":                        "",
		"":                                      "",
	}

	for rawURL, want := range tests {
		if got := SubPath(rawURL); got != want {
			t.Errorf("SubPath(%q) = %q, want %q", rawURL, got, want)
		}
	}
}
