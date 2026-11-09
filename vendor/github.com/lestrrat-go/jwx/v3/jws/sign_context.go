package jws

import (
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/internal/pool"
	"github.com/lestrrat-go/jwx/v3/jwa"
)

type signContext struct {
	format        int
	detached      bool
	validateKey   bool
	strictECDSA   bool
	payload       []byte
	payloadReader io.Reader
	encoder       Base64Encoder
	none          *signatureBuilder // special signature builder
	sigbuilders   []*signatureBuilder
}

var signContextPool = pool.New[*signContext](allocSignContext, freeSignContext)

func allocSignContext() *signContext {
	return &signContext{
		format:      fmtCompact,
		sigbuilders: make([]*signatureBuilder, 0, 1),
		encoder:     base64.DefaultEncoder(),
	}
}

func freeSignContext(ctx *signContext) *signContext {
	ctx.format = fmtCompact
	for _, sb := range ctx.sigbuilders {
		signatureBuilderPool.Put(sb)
	}
	ctx.sigbuilders = ctx.sigbuilders[:0]
	ctx.detached = false
	ctx.validateKey = false
	ctx.strictECDSA = false
	ctx.encoder = base64.DefaultEncoder()
	ctx.none = nil
	ctx.payload = nil
	ctx.payloadReader = nil

	return ctx
}

func (sc *signContext) ProcessOptions(options []SignOption) error {
	for _, option := range options {
		switch option.Ident() {
		case identSerialization{}:
			if err := option.Value(&sc.format); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve serialization option value: %w`, err)
			}
		case identInsecureNoSignature{}:
			var data withInsecureNoSignature
			if err := option.Value(&data); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve insecure-no-signature option value: %w`, err)
			}
			sb := signatureBuilderPool.Get()
			sb.alg = jwa.NoSignature()
			sb.protected = data.protected
			sb.signer = noneSigner{}
			sc.none = sb
			sc.sigbuilders = append(sc.sigbuilders, sb)
		case identKey{}:
			var data *withKey
			if err := option.Value(&data); err != nil {
				return makeSignError(prefixJwsSign, `invalid value for WithKey option: %w`, err)
			}

			alg, ok := data.alg.(jwa.SignatureAlgorithm)
			if !ok {
				return makeSignError(prefixJwsSign, `expected algorithm to be of type jwa.SignatureAlgorithm but got (%[1]q, %[1]T)`, data.alg)
			}

			// No, we don't accept "none" here.
			if alg == jwa.NoSignature() {
				return makeSignError(prefixJwsSign, `"none" (jwa.NoSignature) cannot be used with jws.WithKey`)
			}

			if err := validateAlgorithmForKey(alg, data.key); err != nil {
				return makeSignError(prefixJwsSign, `%w`, err)
			}

			sb := signatureBuilderPool.Get()
			sb.alg = alg
			sb.protected = data.protected
			sb.key = data.key
			sb.public = data.public

			s2, err := SignerFor(alg)
			if err == nil {
				sb.signer2 = s2
			} else {
				s1, err := legacySignerFor(alg)
				if err != nil {
					sb.signer2 = defaultSigner{alg: alg}
				} else {
					sb.signer = s1
				}
			}

			sc.sigbuilders = append(sc.sigbuilders, sb)
		case identDetachedPayload{}:
			if sc.payloadReader != nil {
				return makeSignError(prefixJwsSign, `jws.WithDetachedPayload() and jws.WithDetachedPayloadReader() are mutually exclusive`)
			}
			if sc.payload != nil {
				return makeSignError(prefixJwsSign, `the first argument to jws.Sign() must be nil when jws.WithDetachedPayload() is used`)
			}
			if err := option.Value(&sc.payload); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve detached payload option value: %w`, err)
			}
			sc.detached = true
		case identDetachedPayloadReader{}:
			if sc.payloadReader != nil {
				return makeSignError(prefixJwsSign, `jws.WithDetachedPayloadReader() specified more than once`)
			}
			if sc.detached {
				return makeSignError(prefixJwsSign, `jws.WithDetachedPayload() and jws.WithDetachedPayloadReader() are mutually exclusive`)
			}
			if sc.payload != nil {
				return makeSignError(prefixJwsSign, `the first argument to jws.Sign() must be nil when jws.WithDetachedPayloadReader() is used`)
			}
			if err := option.Value(&sc.payloadReader); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve detached payload reader option value: %w`, err)
			}
			sc.detached = true
		case identValidateKey{}:
			if err := option.Value(&sc.validateKey); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve validate-key option value: %w`, err)
			}
		case identStrictECDSA{}:
			if err := option.Value(&sc.strictECDSA); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve strict-ECDSA option value: %w`, err)
			}
		case identBase64Encoder{}:
			if err := option.Value(&sc.encoder); err != nil {
				return makeSignError(prefixJwsSign, `failed to retrieve base64-encoder option value: %w`, err)
			}
		default:
			return makeSignError(prefixJwsSign, `invalid jws.SignOption %q passed`, `With`+strings.TrimPrefix(fmt.Sprintf(`%T`, option.Ident()), `jws.ident`))
		}
	}
	return nil
}

func (sc *signContext) PopulateMessage(m *Message) error {
	m.payload = sc.payload
	m.detached = sc.detached
	m.signatures = make([]*Signature, 0, len(sc.sigbuilders))

	for i, sb := range sc.sigbuilders {
		// Create signature for each builders
		if sc.validateKey {
			if err := validateKeyBeforeUse(sb.key); err != nil {
				return fmt.Errorf(`failed to validate key for signature %d: %w`, i, err)
			}
		}

		sig, err := sb.Build(sc, m.payload)
		if err != nil {
			return fmt.Errorf(`failed to build signature %d: %w`, i, err)
		}

		m.signatures = append(m.signatures, sig)
	}

	return nil
}
