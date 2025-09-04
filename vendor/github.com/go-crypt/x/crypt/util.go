package crypt

import (
	b64 "github.com/go-crypt/x/base64"
)

func permute(sum, table []byte) []byte {
	size := len(table)

	key := make([]byte, size)

	for i := 0; i < size; i++ {
		key[i] = sum[table[i]]
	}

	return b64.EncodeCrypt(key)
}

func even(i int) bool {
	return i%2 == 0
}

var (
	cleanBytes = make([]byte, 64)
)

func clean(b []byte) {
	l := len(b)

	for ; l > 64; l -= 64 {
		copy(b[l-64:l], cleanBytes)
	}

	if l > 0 {
		copy(b[0:l], cleanBytes[0:l])
	}
}

func repeat(input []byte, length int) []byte {
	var (
		seq  = make([]byte, length)
		unit = len(input)
	)

	j := length / unit * unit
	for i := 0; i < j; i += unit {
		copy(seq[i:length], input)
	}
	if j < length {
		copy(seq[j:length], input[0:length-j])
	}

	return seq
}
