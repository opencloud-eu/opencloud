package shacrypt

import (
	"fmt"
	"strconv"

	"github.com/go-crypt/crypt/algorithm"
	"github.com/go-crypt/crypt/internal/encoding"
)

// RegisterDecoder the decoder with the algorithm.DecoderRegister.
func RegisterDecoder(r algorithm.DecoderRegister) (err error) {
	if err = RegisterDecoderSHA256(r); err != nil {
		return err
	}

	if err = RegisterDecoderSHA512(r); err != nil {
		return err
	}

	return nil
}

// RegisterDecoderSHA256 registers specifically the sha256 decoder variant with the algorithm.DecoderRegister.
func RegisterDecoderSHA256(r algorithm.DecoderRegister) (err error) {
	if err = r.RegisterDecodeFunc(VariantSHA256.Prefix(), DecodeVariant(VariantSHA256)); err != nil {
		return err
	}

	return nil
}

// RegisterDecoderSHA512 registers specifically the sha512 decoder variant with the algorithm.DecoderRegister.
func RegisterDecoderSHA512(r algorithm.DecoderRegister) (err error) {
	if err = r.RegisterDecodeFunc(VariantSHA512.Prefix(), DecodeVariant(VariantSHA512)); err != nil {
		return err
	}

	return nil
}

// Decode the encoded digest into a algorithm.Digest.
func Decode(encodedDigest string) (digest algorithm.Digest, err error) {
	return DecodeVariant(VariantNone)(encodedDigest)
}

// DecodeVariant the encoded digest into a algorithm.Digest provided it matches the provided Variant. If VariantNone is
// used all variants can be decoded.
func DecodeVariant(v Variant) func(encodedDigest string) (digest algorithm.Digest, err error) {
	return func(encodedDigest string) (digest algorithm.Digest, err error) {
		var (
			parts   []string
			variant Variant
		)

		if variant, parts, err = decoderParts(encodedDigest); err != nil {
			return nil, fmt.Errorf(algorithm.ErrFmtDigestDecode, AlgName, err)
		}

		if v != VariantNone && v != variant {
			return nil, fmt.Errorf(algorithm.ErrFmtDigestDecode, AlgName, fmt.Errorf("the '%s' variant cannot be decoded only the '%s' variant can be", variant.String(), v.String()))
		}

		if digest, err = decode(variant, parts); err != nil {
			return nil, fmt.Errorf(algorithm.ErrFmtDigestDecode, AlgName, err)
		}

		return digest, nil
	}
}

func decoderParts(encodedDigest string) (variant Variant, parts []string, err error) {
	parts = encoding.Split(encodedDigest, -1)

	if n := len(parts); n != 4 && n != 5 {
		return VariantNone, nil, algorithm.ErrEncodedHashInvalidFormat
	}

	variant = NewVariant(parts[1])

	if variant == VariantNone {
		return variant, nil, fmt.Errorf("%w: identifier '%s' is not an encoded %s digest", algorithm.ErrEncodedHashInvalidIdentifier, parts[1], AlgName)
	}

	return variant, parts[2:], nil
}

func decode(variant Variant, parts []string) (digest algorithm.Digest, err error) {
	decoded := &Digest{
		variant: variant,
	}

	var (
		ip, is, ik int
	)

	switch len(parts) {
	case 2:
		ip, is, ik = -1, 0, 1
	case 3:
		ip, is, ik = 0, 1, 2
	}

	if len(parts[ik]) == 0 {
		return nil, fmt.Errorf("%w: key has 0 bytes", algorithm.ErrEncodedHashKeyEncoding)
	}

	decoded.iterations = IterationsDefaultOmitted

	var params []encoding.Parameter

	if ip >= 0 {
		if params, err = encoding.DecodeParameterStr(parts[ip]); err != nil {
			return nil, err
		}
	}

	for _, param := range params {
		switch param.Key {
		case "rounds":
			var rounds uint64

			if rounds, err = strconv.ParseUint(param.Value, 10, 32); err != nil {
				return nil, fmt.Errorf("%w: option '%s' has invalid value '%s': %v", algorithm.ErrEncodedHashInvalidOptionValue, param.Key, param.Value, err)
			}

			decoded.iterations = int(rounds)
		default:
			return nil, fmt.Errorf("%w: option '%s' with value '%s' is unknown", algorithm.ErrEncodedHashInvalidOptionKey, param.Key, param.Value)
		}
	}

	decoded.salt, decoded.key = []byte(parts[is]), []byte(parts[ik])

	return decoded, nil
}
