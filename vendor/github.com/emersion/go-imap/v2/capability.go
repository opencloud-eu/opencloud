package imap

import (
	"strconv"
	"strings"
)

// Cap represents an IMAP capability.
type Cap string

// Registered capabilities.
//
// See: https://www.iana.org/assignments/imap-capabilities/
const (
	CapIMAP4rev1 Cap = "IMAP4rev1" // RFC 3501
	CapIMAP4rev2 Cap = "IMAP4rev2" // RFC 9051

	CapAuthPlain Cap = "AUTH=PLAIN"

	CapStartTLS      Cap = "STARTTLS"
	CapLoginDisabled Cap = "LOGINDISABLED"

	// Folded in IMAP4rev2
	CapNamespace    Cap = "NAMESPACE"     // RFC 2342
	CapUnselect     Cap = "UNSELECT"      // RFC 3691
	CapUIDPlus      Cap = "UIDPLUS"       // RFC 4315
	CapESearch      Cap = "ESEARCH"       // RFC 4731
	CapSearchRes    Cap = "SEARCHRES"     // RFC 5182
	CapEnable       Cap = "ENABLE"        // RFC 5161
	CapIdle         Cap = "IDLE"          // RFC 2177
	CapSASLIR       Cap = "SASL-IR"       // RFC 4959
	CapListExtended Cap = "LIST-EXTENDED" // RFC 5258
	CapListStatus   Cap = "LIST-STATUS"   // RFC 5819
	CapMove         Cap = "MOVE"          // RFC 6851
	CapLiteralMinus Cap = "LITERAL-"      // RFC 7888
	CapStatusSize   Cap = "STATUS=SIZE"   // RFC 8438

	CapACL              Cap = "ACL"                // RFC 4314
	CapAppendLimit      Cap = "APPENDLIMIT"        // RFC 7889
	CapBinary           Cap = "BINARY"             // RFC 3516
	CapCatenate         Cap = "CATENATE"           // RFC 4469
	CapChildren         Cap = "CHILDREN"           // RFC 3348
	CapCondStore        Cap = "CONDSTORE"          // RFC 7162
	CapConvert          Cap = "CONVERT"            // RFC 5259
	CapCreateSpecialUse Cap = "CREATE-SPECIAL-USE" // RFC 6154
	CapESort            Cap = "ESORT"              // RFC 5267
	CapFilters          Cap = "FILTERS"            // RFC 5466
	CapID               Cap = "ID"                 // RFC 2971
	CapLanguage         Cap = "LANGUAGE"           // RFC 5255
	CapListMyRights     Cap = "LIST-MYRIGHTS"      // RFC 8440
	CapLiteralPlus      Cap = "LITERAL+"           // RFC 7888
	CapLoginReferrals   Cap = "LOGIN-REFERRALS"    // RFC 2221
	CapMailboxReferrals Cap = "MAILBOX-REFERRALS"  // RFC 2193
	CapMetadata         Cap = "METADATA"           // RFC 5464
	CapMetadataServer   Cap = "METADATA-SERVER"    // RFC 5464
	CapMultiAppend      Cap = "MULTIAPPEND"        // RFC 3502
	CapMultiSearch      Cap = "MULTISEARCH"        // RFC 7377
	CapNotify           Cap = "NOTIFY"             // RFC 5465
	CapObjectID         Cap = "OBJECTID"           // RFC 8474
	CapPreview          Cap = "PREVIEW"            // RFC 8970
	CapQResync          Cap = "QRESYNC"            // RFC 7162
	CapQuota            Cap = "QUOTA"              // RFC 9208
	CapQuotaSet         Cap = "QUOTASET"           // RFC 9208
	CapReplace          Cap = "REPLACE"            // RFC 8508
	CapSaveDate         Cap = "SAVEDATE"           // RFC 8514
	CapSearchFuzzy      Cap = "SEARCH=FUZZY"       // RFC 6203
	CapSort             Cap = "SORT"               // RFC 5256
	CapSortDisplay      Cap = "SORT=DISPLAY"       // RFC 5957
	CapSpecialUse       Cap = "SPECIAL-USE"        // RFC 6154
	CapUnauthenticate   Cap = "UNAUTHENTICATE"     // RFC 8437
	CapURLPartial       Cap = "URL-PARTIAL"        // RFC 5550
	CapURLAuth          Cap = "URLAUTH"            // RFC 4467
	CapUTF8Accept       Cap = "UTF8=ACCEPT"        // RFC 6855
	CapUTF8Only         Cap = "UTF8=ONLY"          // RFC 6855
	CapWithin           Cap = "WITHIN"             // RFC 5032
	CapUIDOnly          Cap = "UIDONLY"            // RFC 9586
	CapListMetadata     Cap = "LIST-METADATA"      // RFC 9590
	CapInProgress       Cap = "INPROGRESS"         // RFC 9585
)

var imap4rev2Caps = CapSet{
	CapNamespace:    {},
	CapUnselect:     {},
	CapUIDPlus:      {},
	CapESearch:      {},
	CapSearchRes:    {},
	CapEnable:       {},
	CapIdle:         {},
	CapSASLIR:       {},
	CapListExtended: {},
	CapListStatus:   {},
	CapMove:         {},
	CapLiteralMinus: {},
	CapStatusSize:   {},
}

// AuthCap returns the capability name for an SASL authentication mechanism.
func AuthCap(mechanism string) Cap {
	return Cap("AUTH=" + mechanism)
}

// CapSet is a set of capabilities.
type CapSet map[Cap]struct{}

func (set CapSet) has(c Cap) bool {
	_, ok := set[c]
	return ok
}

// Has checks whether a capability is supported.
//
// Some capabilities are implied by others, as such Has may return true even if
// the capability is not in the map.
func (set CapSet) Has(c Cap) bool {
	if set.has(c) {
		return true
	}

	if set.has(CapIMAP4rev2) && imap4rev2Caps.has(c) {
		return true
	}

	if c == CapLiteralMinus && set.has(CapLiteralPlus) {
		return true
	}
	if c == CapCondStore && set.has(CapQResync) {
		return true
	}
	if c == CapUTF8Accept && set.has(CapUTF8Only) {
		return true
	}
	if c == CapAppendLimit {
		_, ok := set.AppendLimit()
		return ok
	}

	return false
}

// AuthMechanisms returns the list of supported SASL mechanisms for
// authentication.
func (set CapSet) AuthMechanisms() []string {
	var l []string
	for c := range set {
		if !strings.HasPrefix(string(c), "AUTH=") {
			continue
		}
		mech := strings.TrimPrefix(string(c), "AUTH=")
		l = append(l, mech)
	}
	return l
}

// AppendLimit checks the APPENDLIMIT capability.
//
// If the server supports APPENDLIMIT, ok is true. If the server doesn't have
// the same upload limit for all mailboxes, limit is nil and per-mailbox
// limits must be queried via STATUS.
func (set CapSet) AppendLimit() (limit *uint32, ok bool) {
	if set.has(CapAppendLimit) {
		return nil, true
	}

	for c := range set {
		if !strings.HasPrefix(string(c), "APPENDLIMIT=") {
			continue
		}

		limitStr := strings.TrimPrefix(string(c), "APPENDLIMIT=")
		limit64, err := strconv.ParseUint(limitStr, 10, 32)
		if err == nil && limit64 > 0 {
			limit32 := uint32(limit64)
			return &limit32, true
		}
	}

	limit32 := ^uint32(0)
	return &limit32, false
}

// QuotaResourceTypes returns the list of supported QUOTA resource types.
func (set CapSet) QuotaResourceTypes() []QuotaResourceType {
	var l []QuotaResourceType
	for c := range set {
		if !strings.HasPrefix(string(c), "QUOTA=RES-") {
			continue
		}
		t := strings.TrimPrefix(string(c), "QUOTA=RES-")
		l = append(l, QuotaResourceType(t))
	}
	return l
}

// ThreadAlgorithms returns the list of supported threading algorithms.
func (set CapSet) ThreadAlgorithms() []ThreadAlgorithm {
	var l []ThreadAlgorithm
	for c := range set {
		if !strings.HasPrefix(string(c), "THREAD=") {
			continue
		}
		alg := strings.TrimPrefix(string(c), "THREAD=")
		l = append(l, ThreadAlgorithm(alg))
	}
	return l
}
