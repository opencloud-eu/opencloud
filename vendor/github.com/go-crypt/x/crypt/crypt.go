package crypt

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"strconv"
)

// KeySHACrypt calculates the shacrypt SHA256/SHA512 key given an appropriate hash.Hash, password, salt, and number of rounds.
func KeySHACrypt(hashFunc func() hash.Hash, password, salt []byte, rounds int) []byte {
	// Step 1.
	digest := hashFunc()

	size := digest.Size()

	switch size {
	case sha1.Size:
		return KeySHA1Crypt(password, salt, uint32(rounds))
	case sha256.Size, sha512.Size:
		break
	default:
		return nil
	}

	length := len(password)

	// Step 2.
	digest.Write(password)

	// Step 3.
	digest.Write(salt)

	// Step 4.
	digestB := hashFunc()

	// Step 5.
	digestB.Write(password)

	// Step 6.
	digestB.Write(salt)

	// Step 7.
	digestB.Write(password)

	// Step 8.
	sumB := digestB.Sum(nil)
	digestB.Reset()
	digestB = nil

	// Step 9 and 10:
	digest.Write(repeat(sumB, length))

	// Step 11.
	for i := length; i > 0; i >>= 1 {
		if even(i) {
			digest.Write(password)
		} else {
			digest.Write(sumB)
		}
	}

	clean(sumB)
	sumB = nil

	// Step 12.
	sumA := digest.Sum(nil)
	digest.Reset()

	// Step 13-14.
	for i := 0; i < length; i++ {
		digest.Write(password)
	}

	// Step 15.
	sumDP := digest.Sum(nil)
	digest.Reset()

	// Step 16.
	seqP := repeat(sumDP, length)
	sumDP = nil

	// Step 17-18.
	for i := 0; i < 16+int(sumA[0]); i++ {
		digest.Write(salt)
	}

	// Step 19.
	sumDS := digest.Sum(nil)
	digest.Reset()

	// Step 20.
	seqS := repeat(sumDS, len(salt))

	// Step 21.
	for i := 0; i < rounds; i++ {
		digest.Reset()

		// Step 21 Sub-Step B and C.
		if i&1 != 0 {
			// Step 21 Sub-Step B.
			digest.Write(seqP)
		} else {
			// Step 21 Sub-Step C.
			digest.Write(sumA)
		}

		// Step 21 Sub-Step D.
		if i%3 != 0 {
			digest.Write(seqS)
		}

		// Step 21 Sub-Step E.
		if i%7 != 0 {
			digest.Write(seqP)
		}

		// Step 21 Sub-Step F and G.
		if i&1 != 0 {
			// Step 21 Sub-Step F.
			digest.Write(sumA)
		} else {
			// Step 21 Sub-Step G.
			digest.Write(seqP)
		}

		// Sub-Step H.
		copy(sumA, digest.Sum(nil))
	}

	digest.Reset()
	digest = nil

	seqP, seqS = nil, nil

	switch size {
	case sha256.Size:
		// Step 22 Sub Step E.
		return permute(sumA, permuteTableSHACryptSHA256[:])
	case sha512.Size:
		// Step 22 Sub Step E.
		return permute(sumA, permuteTableSHACryptSHA512[:])
	}

	return nil
}

// KeySHA1Crypt calculates the sha1crypt key given a password, salt, and number of rounds.
func KeySHA1Crypt(password, salt []byte, rounds uint32) []byte {
	digest := hmac.New(sha1.New, password)
	digest.Write(salt)
	digest.Write(prefixSHA1Crypt)
	digest.Write([]byte(strconv.FormatUint(uint64(rounds), 10)))

	sumA := digest.Sum(nil)

	if rounds == 0 {
		return permute(sumA, permuteTableSHA1Crypt[:])
	}

	for rounds--; rounds > 0; rounds-- {
		digest.Reset()

		digest.Write(sumA)

		copy(sumA, digest.Sum(nil))
	}

	return permute(sumA, permuteTableSHA1Crypt[:])
}

// KeyMD5Crypt calculates the md5crypt key given a password and salt.
func KeyMD5Crypt(password, salt []byte) []byte {
	length := len(password)

	digest := md5.New()

	digest.Write(password)
	digest.Write(salt)
	digest.Write(password)

	sumB := digest.Sum(nil)

	digest.Reset()

	digest.Write(password)
	digest.Write(prefixMD5Crypt)
	digest.Write(salt)
	digest.Write(repeat(sumB, length))

	clean(sumB)

	for i := length; i > 0; i >>= 1 {
		if even(i) {
			digest.Write(password[0:1])
		} else {
			digest.Write([]byte{0})
		}
	}

	sumA := digest.Sum(nil)

	for i := 0; i < 1000; i++ {
		digest.Reset()

		if even(i) {
			digest.Write(sumA)
		} else {
			digest.Write(password)
		}

		if i%3 != 0 {
			digest.Write(salt)
		}

		if i%7 != 0 {
			digest.Write(password)
		}

		if i&1 == 0 {
			digest.Write(password)
		} else {
			digest.Write(sumA)
		}

		copy(sumA, digest.Sum(nil))
	}

	return permute(sumA, permuteTableMD5Crypt[:])
}

// KeyMD5CryptSun calculates the md5crypt (Sun Version) key given a password, salt, and number rounds.
func KeyMD5CryptSun(password, salt []byte, rounds uint32) []byte {
	digest := md5.New()

	digest.Write(password)

	if rounds == 0 {
		digest.Write(prefixSunMD5Crypt)
		digest.Write(salt)
		digest.Write(sepCrypt)
	} else {
		digest.Write(prefixSunMD5CryptRounds)
		digest.Write([]byte(strconv.FormatUint(uint64(rounds), 10)))
		digest.Write(sepCrypt)
		digest.Write(salt)
		digest.Write(sepCrypt)
	}

	sumA := digest.Sum(nil)

	iterations := uint32(rounds + 4096)

	bit := func(off uint32) uint32 {
		off %= 128
		if (sumA[off/8] & (0x01 << (off % 8))) != 0 {
			return 1
		}

		return 0
	}

	var ind7 [md5.Size]byte

	for i := uint32(0); i < iterations; i++ {
		digest.Reset()

		digest.Write(sumA)

		for j := 0; j < md5.Size; j++ {
			off := (j + 3) % 16
			ind4 := (sumA[j] >> (sumA[off] % 5)) & 0x0F
			sh7 := (sumA[off] >> (sumA[j] % 8)) & 0x01
			ind7[j] = (sumA[ind4] >> sh7) & 0x7F
		}

		var indA, indB uint32

		for j := uint(0); j < 8; j++ {
			indA |= bit(uint32(ind7[j])) << j
			indB |= bit(uint32(ind7[j+8])) << j
		}

		indA = (indA >> bit(i)) & 0x7F
		indB = (indB >> bit(i+64)) & 0x7F

		if bit(indA)^bit(indB) == 1 {
			digest.Write(magicTableMD5CryptSunHamlet[:])
		}

		digest.Write([]byte(strconv.FormatUint(uint64(i), 10)))

		copy(sumA, digest.Sum(nil))
	}

	return permute(sumA, permuteTableMD5Crypt[:])
}
