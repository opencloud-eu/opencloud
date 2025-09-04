package base64

import (
	"encoding/base64"
)

var AdaptedEncoding = base64.NewEncoding(encodeAdapted)

// BcryptEncoding is the Bcrypt Base64 Alternative encoding.
var BcryptEncoding = base64.NewEncoding(bcryptB64Alphabet)

// EncodeCrypt implements the linux crypt lib's B64 encoding.
func EncodeCrypt(src []byte) (dst []byte) {
	if len(src) == 0 {
		return nil
	}

	dst = make([]byte, (len(src)*8+5)/6)

	idst, isrc := 0, 0

	for isrc < len(src)/3*3 {
		v := uint(src[isrc+2])<<16 | uint(src[isrc+1])<<8 | uint(src[isrc])
		dst[idst+0] = cryptB64Alphabet[v&0x3f]
		dst[idst+1] = cryptB64Alphabet[v>>6&0x3f]
		dst[idst+2] = cryptB64Alphabet[v>>12&0x3f]
		dst[idst+3] = cryptB64Alphabet[v>>18]
		idst += 4
		isrc += 3
	}

	remainder := len(src) - isrc

	if remainder == 0 {
		return dst
	}

	v := uint(src[isrc+0])
	if remainder == 2 {
		v |= uint(src[isrc+1]) << 8
	}

	dst[idst+0] = cryptB64Alphabet[v&0x3f]
	dst[idst+1] = cryptB64Alphabet[v>>6&0x3f]

	if remainder == 2 {
		dst[idst+2] = cryptB64Alphabet[v>>12]
	}

	return dst
}
