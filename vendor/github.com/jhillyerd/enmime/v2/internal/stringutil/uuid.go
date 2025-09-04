package stringutil

import (
	"fmt"
	"math/rand"
)

// UUID generates a random UUID according to RFC 4122, using optional rand if supplied
func UUID(rs rand.Source) string {
	uuid := make([]byte, 16)
	if rs == nil {
		rs = globalRandSource
	}
	_, _ = rand.New(rs).Read(uuid)
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
