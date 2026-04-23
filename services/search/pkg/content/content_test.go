package content_test

import (
	"testing"

	. "github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
)

func TestCleanContent(t *testing.T) {
	tests := []struct {
		given  string
		expect string
	}{
		{
			given:  "find can keeper should keeper will",
			expect: "keeper keeper",
		},
		{
			given:  "user1 shares the file to Mary",
			expect: "user1 shares file mary",
		},
		{
			given:  "content contains https://localhost/remote.php/dav/files/admin/Photos/San%20Francisco.jpg and stop word",
			expect: "content contains https://localhost/remote.php/dav/files/admin/photos/san%20francisco.jpg stop word",
		},
	}

	for _, tc := range tests {
		t.Run(tc.given, func(t *testing.T) {
			Equal(t, tc.expect, content.CleanString(tc.given, "en"))
		})
	}
}
