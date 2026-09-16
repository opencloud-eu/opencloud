package event

import "testing"

func TestTouchesOpenExtensions(t *testing.T) {
	const state = "http://opencloud.eu/ns/extensions/com.example.project/state"
	for keys, want := range map[*[]string]bool{
		{state}:                             true,
		{"tags", state}:                     true,
		{"tags"}:                            false,
		{"http://owncloud.org/ns/favorite"}: false,
		{}:                                  false,
	} {
		if got := touchesOpenExtensions(*keys); got != want {
			t.Errorf("touchesOpenExtensions(%v) = %v, want %v", *keys, got, want)
		}
	}
}
