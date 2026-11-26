package helpers

import (
	"crypto/sha256"
	"encoding/hex"

	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
)

// HashResourceId builds a urlsafe and stable file reference that can be used for proxy routing,
// so that all sessions on one file end on the same office server
func HashResourceId(resourceId *providerv1beta1.ResourceId) string {
	c := sha256.New()
	c.Write([]byte(storagespace.FormatResourceID(resourceId)))
	return hex.EncodeToString(c.Sum(nil))
}
