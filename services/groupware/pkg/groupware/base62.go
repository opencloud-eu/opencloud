package groupware

import (
	"errors"
	"math/big"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var base = big.NewInt(int64(len(alphabet)))

// EncodeBytes converts any byte slice into a Base62 string
func EncodeBytesToBase62(src []byte) string {
	if len(src) == 0 {
		return ""
	}

	// 1. Convert the bytes into a single large big.Int
	n := new(big.Int).SetBytes(src)
	if n.Cmp(big.NewInt(0)) == 0 {
		return string(alphabet[0])
	}

	// 2. Convert the big.Int to Base62
	var result []byte
	mod := new(big.Int)
	for n.Cmp(big.NewInt(0)) > 0 {
		n.DivMod(n, base, mod)
		result = append(result, alphabet[mod.Int64()])
	}

	// 3. Handle leading zeros in the original byte array.
	// Because math/big drops leading zeros, we must explicitly preserve them.
	for _, b := range src {
		if b != 0x00 {
			break
		}
		result = append(result, alphabet[0])
	}

	// Reverse the result slice to get the correct order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// DecodeBytes converts a Base62 string back into its original byte slice
func DecodeBytesFromBase62(src string) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	n := big.NewInt(0)
	idx := big.NewInt(0)

	// 1. Reconstruct the large integer from the Base62 characters
	for i := 0; i < len(src); i++ {
		pos := findAlphabetIndex(src[i])
		if pos == -1 {
			return nil, errors.New("invalid character in base62 string")
		}
		idx.SetInt64(int64(pos))
		n.Mul(n, base)
		n.Add(n, idx)
	}

	// 2. Extract the raw bytes from the big.Int
	decoded := n.Bytes()

	// 3. Re-prepend the leading zeros that were stripped during encoding
	var leadingZeros int
	for i := 0; i < len(src); i++ {
		if src[i] != alphabet[0] {
			break
		}
		leadingZeros++
	}

	if leadingZeros > 0 {
		zeroBytes := make([]byte, leadingZeros)
		decoded = append(zeroBytes, decoded...)
	}

	return decoded, nil
}

// Helper to find the index of a character in our alphabet
func findAlphabetIndex(char byte) int {
	for i := 0; i < len(alphabet); i++ {
		if alphabet[i] == char {
			return i
		}
	}
	return -1
}
