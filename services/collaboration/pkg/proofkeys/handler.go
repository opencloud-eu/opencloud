package proofkeys

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/rs/zerolog"
)

type PubKeys struct {
	Key        *rsa.PublicKey
	OldKey     *rsa.PublicKey
	ExpireTime time.Time
}

type Verifier interface {
	Verify(accessToken, url, timestamp, sig64, oldSig64 string, opts ...VerifyOption) error
}

type VerifyHandler struct {
	discoveryURL string
	insecure     bool
	cachedKeys   *PubKeys
	cachedDur    time.Duration
	// httpClient is reused for every key fetch, so its transport keeps one connection
	httpClient *http.Client
}

// NewVerifyHandler will return a new Verifier with the provided parameters
// The discoveryURL must point to the https://office.wopi/hosting/discovery
// address, which contains the xml with the proof keys (and more information)
// The insecure parameter can be used to disable certificate verification when
// conecting to the provided discoveryURL
// CachedDur is the duration the keys will be cached in memory. The cached keys
// will be used for the duration provided, after that new keys will be fetched
// from the discoveryURL.
//
// For WOPI apps whose proof keys rotate after a while, you must ensure that
// the provided duration is shorter than the rotation time. This should
// guarantee that we can't fail to verify a request due to obsolete keys.
func NewVerifyHandler(discoveryURL string, insecure bool, cachedDur time.Duration) Verifier {
	return &VerifyHandler{
		discoveryURL: discoveryURL,
		insecure:     insecure,
		cachedDur:    cachedDur,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: insecure,
				},
			},
		},
	}
}

// Verify the request comes from a trusted source
// All the provided parameters are strings:
// * accessToken: The access token used for this request (targeting this collaboration service)
// * url: The full url for this request, including scheme, host and all query parameters,
// something like "https://wopi.opencloud.test/wopi/file/abcbcbd?access_token=oiuiu" or
// "http://wopiserver:8888/wopi/file/abcdef?access_token=zzxxyy"
// * timestamp: The timestamp provided by the WOPI app in the "X-WOPI-TimeStamp" header, as string
// * sig64: The base64-encoded signature, which should come directly from the "X-WOPI-Proof" header
// * oldSig64: The base64-encoded previous signature, coming from the "X-WOPI-ProofOld" header
//
// The public keys will be obtained from the /hosting/discovery path of the target WOPI app.
// Note that the method will perform the following checks in that order:
// * current signature with the current key
// * old signature with the current key
// * current signature with the old key
// If all of those checks are wrong, the method will fail, and the request should be rejected.
//
// The method will return an error if something fails, or nil if everything is ok
func (vh *VerifyHandler) Verify(accessToken, url, timestamp, sig64, oldSig64 string, opts ...VerifyOption) error {
	verifyOptions := newOptions(opts...)

	// check timestamp
	if err := vh.checkTimestamp(timestamp); err != nil {
		return err
	}

	// need to decode the signatures
	signature, err := base64.StdEncoding.DecodeString(sig64)
	if err != nil {
		return err
	}

	var oldSignature []byte
	if oldSig64 != "" {
		if oldSig, err := base64.StdEncoding.DecodeString(oldSig64); err != nil {
			return err
		} else {
			oldSignature = oldSig
		}
	}

	pubkeys := vh.cachedKeys
	if pubkeys == nil || pubkeys.ExpireTime.Before(time.Now()) {
		// fetch the public keys
		newpubkeys, err := vh.fetchPublicKeys(verifyOptions.Logger)
		if err != nil {
			return err
		}
		pubkeys = newpubkeys
		vh.cachedKeys = newpubkeys
	}

	// build and hash the expected proof
	expectedProof := vh.generateProof(accessToken, url, timestamp)
	hashedProof := sha256.Sum256(expectedProof)

	// verify
	if err := rsa.VerifyPKCS1v15(pubkeys.Key, crypto.SHA256, hashedProof[:], signature); err != nil {
		if err := rsa.VerifyPKCS1v15(pubkeys.Key, crypto.SHA256, hashedProof[:], oldSignature); err != nil {
			if pubkeys.OldKey != nil {
				return rsa.VerifyPKCS1v15(pubkeys.OldKey, crypto.SHA256, hashedProof[:], signature)
			} else {
				return err
			}
		}
	}
	return nil
}

// checkTimestamp will check if the provided timestamp is valid.
// The timestamp is valid if it isn't older than 20 minutes (info from
// MS WOPI docs).
//
// Note: the timestamp is based on C# DateTime.UtcNow.Ticks
// "One tick equals 100 nanoseconds. The value of this property represents
// the number of ticks that have elapsed since 12:00:00 midnight, January 1, 0001."
// It is NOT a unix timestamp (current unix timestamp ~1718123417 secs ;
// expected timestamp ~638537195321890000 100-nanosecs)
func (vh *VerifyHandler) checkTimestamp(timestamp string) error {
	// set the stamp
	stamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("Invalid timestamp")
	}

	// 62135596800 seconds from "January 1, 1 AD" to "January 1, 1970 12:00:00 AM"
	// need to convert those secs into 100-nanosecs in order to compare the stamp
	unixBaseStamp := int64(62135596800 * 1000 * 1000 * 10)

	// stamp - unixBaseStamp gives us the unix-based timestamp we can use
	unixStamp := stamp - unixBaseStamp

	// divide between 1000*1000*10 to get the seconds and 100-nanoseconds
	unixStampSec := unixStamp / (1000 * 1000 * 10)
	unixStampNanoSec := (unixStamp % (1000 * 1000 * 10)) * 100

	// both seconds and nanoseconds should be within int64 range
	convertedUnixTimestamp := time.Unix(unixStampSec, unixStampNanoSec)

	if time.Now().After(convertedUnixTimestamp.Add(20 * time.Minute)) {
		return errors.New("Timestamp expired")
	}
	return nil
}

// generateProof will generated a expected proof to be verified later.
// The method will return a slice of bytes with the proof (consider it binary
// data).
// The bytes will need to be hashed later in order to perform the verification
func (vh *VerifyHandler) generateProof(accessToken, url, timestamp string) []byte {
	tokenBytes := []byte(accessToken)
	tokenLen := len(tokenBytes)
	tokenLenBytes := big.NewInt(int64(tokenLen)).FillBytes(make([]byte, 4))

	// url needs to be uppercase
	urlBytes := []byte(strings.ToUpper(url))
	urlLen := len(urlBytes)
	urlLenBytes := big.NewInt(int64(urlLen)).FillBytes(make([]byte, 4))

	stampBigInt, _ := new(big.Int).SetString(timestamp, 10)
	stampBytes := stampBigInt.FillBytes(make([]byte, 8))
	stampLen := len(stampBytes)
	stampLenBytes := big.NewInt(int64(stampLen)).FillBytes(make([]byte, 4))

	proof := new(bytes.Buffer)
	proof.Write(tokenLenBytes)
	proof.Write(tokenBytes)
	proof.Write(urlLenBytes)
	proof.Write(urlBytes)
	proof.Write(stampLenBytes)
	proof.Write(stampBytes)
	return proof.Bytes()
}

// fetchPublicKeys will fetch the public keys from the /hosting/discovery URL
// of the provided WOPI app.
// It will return a PubKeys struct to hold the public keys based on the modulus
// and exponent found.
// The PubKeys returned might be either nil (with the non-nil error), or might
// contain only a PubKeys.Key field (the PubKeys.OldKey might be nil)
func (vh *VerifyHandler) fetchPublicKeys(logger *zerolog.Logger) (*PubKeys, error) {
	logger.Debug().Str("WopiAppUrl", vh.discoveryURL).Msg("WopiDiscovery: requesting new public keys")

	httpResp, err := vh.httpClient.Get(vh.discoveryURL)
	if err != nil {
		logger.Error().
			Err(err).
			Str("WopiAppUrl", vh.discoveryURL).
			Msg("WopiDiscovery: failed to access wopi app url")
		return nil, err
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		logger.Error().
			Str("WopiAppUrl", vh.discoveryURL).
			Int("HttpCode", httpResp.StatusCode).
			Msg("WopiDiscovery: wopi app url failed with unexpected code")
		return nil, errors.New("wopi app url failed with unexpected code")
	}

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(httpResp.Body); err != nil {
		return nil, err
	}

	root := doc.SelectElement("wopi-discovery")
	if root == nil {
		return nil, errors.New("wopi-discovery element not found in the XML body")
	}

	proofKey := root.SelectElement("proof-key")
	if proofKey == nil {
		return nil, errors.New("proof-key element not found in the XML body")
	}

	mod64 := proofKey.SelectAttrValue("modulus", "")
	exp64 := proofKey.SelectAttrValue("exponent", "")
	oldMod64 := proofKey.SelectAttrValue("oldmodulus", "")
	oldExp64 := proofKey.SelectAttrValue("oldexponent", "")

	if mod64 == "" || exp64 == "" {
		return nil, errors.New("modulus or exponent not found in the proof-key element")
	}

	keys := &PubKeys{
		Key:        vh.keyFromBase64(mod64, exp64),
		ExpireTime: time.Now().Add(vh.cachedDur),
	}

	if oldMod64 != "" && oldExp64 != "" {
		keys.OldKey = vh.keyFromBase64(oldMod64, oldExp64)
	}

	return keys, nil
}

// keyFromBase64 will create a rsa public key from the provided modulus and
// exponent, both encoded with base64.
// If any of the provided strings can't be decoded, nil will be returned.
func (vh *VerifyHandler) keyFromBase64(mod64, exp64 string) *rsa.PublicKey {
	dataMod, err := base64.StdEncoding.DecodeString(mod64)
	if err != nil {
		return nil
	}

	dataE, err := base64.StdEncoding.DecodeString(exp64)
	if err != nil {
		return nil
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(dataMod),
		E: int(new(big.Int).SetBytes(dataE).Int64()),
	}
	return pub
}
