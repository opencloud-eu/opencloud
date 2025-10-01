package jmap

import (
	"io"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
)

// https://www.iana.org/assignments/jmap/jmap.xml#jmap-data-types
type ObjectType string

// Where `UTCDate` is given as a type, it means a `Date` where the "time-offset"
// component MUST be `"Z"` (i.e., it must be in UTC time).
//
// For example, `"2014-10-30T06:12:00Z"`.
type UTCDate struct {
	time.Time
}

func (t UTCDate) MarshalJSON() ([]byte, error) {
	// TODO imperfect, we're going to need something smarter here as the timezone is not actually
	// fixed to be UTC but, instead, depends on the timezone that is defined in another property
	// of the object where this LocalDate shows up in; alternatively, we might have to use a string
	// here and leave the conversion to a usable timestamp up to the client or caller instead
	return []byte("\"" + t.UTC().Format(time.RFC3339) + "\""), nil
}

func (t *UTCDate) UnmarshalJSON(b []byte) error {
	var tt time.Time
	err := tt.UnmarshalJSON(b)
	if err != nil {
		return err
	}
	t.Time = tt.UTC()
	return nil
}

// Where `LocalDate` is given as a type, it means a string in the same format as `Date`
// (see [RFC8620, Section 1.4]), but with the time-offset omitted from the end.
//
// For example, `2014-10-30T14:12:00`.
//
// The interpretation in absolute time depends upon the time zone for the event, which
// may not be a fixed offset (for example when daylight saving time occurs).
//
// [RFC8620, Section 1.4]: https://www.rfc-editor.org/rfc/rfc8620.html#section-1.4
type LocalDate struct {
	time.Time
}

const RFC3339Local = "2006-01-02T15:04:05"

func (t LocalDate) MarshalJSON() ([]byte, error) {
	// TODO imperfect, we're going to need something smarter here as the timezone is not actually
	// fixed to be UTC but, instead, depends on the timezone that is defined in another property
	// of the object where this LocalDate shows up in; alternatively, we might have to use a string
	// here and leave the conversion to a usable timestamp up to the client or caller instead
	return []byte("\"" + t.UTC().Format(RFC3339Local) + "\""), nil
}

func (t *LocalDate) UnmarshalJSON(b []byte) error {
	var tt time.Time
	err := tt.UnmarshalJSON(b)
	if err != nil {
		return err
	}
	t.Time = tt.UTC()
	return nil
}

// Should the calendar’s events be used as part of availability calculation?
//
// This MUST be one of:
// !- `all“: all events are considered.
// !- `attending“: events the user is a confirmed or tentative participant of are considered.
// !- `none“: all events are ignored (but may be considered if also in another calendar).
//
// This should default to “all” for the calendars in the user’s own account, and “none” for calendars shared with the user.
type IncludeInAvailability string

type TypeOfCalendarAlert string

// `CalendarEventNotification` type.
//
// This MUST be one of
// !- `created`
// !- `updated`
// !- `destroyed`
type CalendarEventNotificationTypeOption string

// `Principal` type.
//
// This MUST be one of the following values:
// !- `individual`: This represents a single person.
// !- `group`: This represents a group of people.
// !- `resource`: This represents some resource, e.g. a projector.
// !- `location`: This represents a location.
// !- `other`: This represents some other undefined principal.
type PrincipalTypeOption string

// Algorithms in this list MUST be present in the ["HTTP Digest Algorithm Values" registry]
// defined by [RFC3230]; however, in JMAP, they must be lowercased, e.g., "md5" rather than
// "MD5".
//
// Clients SHOULD prefer algorithms listed earlier in this list.
//
// ["HTTP Digest Algorithm Values" registry]: https://www.iana.org/assignments/http-dig-alg/http-dig-alg.xhtml
type HttpDigestAlgorithm string

// The ResourceType data type is used to act as a unit of measure for the quota usage.
type ResourceType string

// The Scope data type is used to represent the entities the quota applies to.
type Scope string

type ActionMode string
type SendingMode string
type DispositionTypeOption string

const (
	JmapCore             = "urn:ietf:params:jmap:core"
	JmapMail             = "urn:ietf:params:jmap:mail"
	JmapMDN              = "urn:ietf:params:jmap:mdn" // https://datatracker.ietf.org/doc/rfc9007/
	JmapSubmission       = "urn:ietf:params:jmap:submission"
	JmapVacationResponse = "urn:ietf:params:jmap:vacationresponse"
	JmapCalendars        = "urn:ietf:params:jmap:calendars"
	JmapContacts         = "urn:ietf:params:jmap:contacts"
	JmapSieve            = "urn:ietf:params:jmap:sieve"
	JmapBlob             = "urn:ietf:params:jmap:blob"
	JmapQuota            = "urn:ietf:params:jmap:quota"
	JmapWebsocket        = "urn:ietf:params:jmap:websocket"
	JmapPrincipals       = "urn:ietf:params:jmap:principals"
	JmapPrincipalsOwner  = "urn:ietf:params:jmap:principals:owner"

	CoreType                      = ObjectType("Core")
	PushSubscriptionType          = ObjectType("PushSubscription")
	MailboxType                   = ObjectType("Mailbox")
	ThreadType                    = ObjectType("Thread")
	EmailType                     = ObjectType("Email")
	EmailDeliveryType             = ObjectType("EmailDelivery")
	SearchSnippetType             = ObjectType("SearchSnippet")
	IdentityType                  = ObjectType("Identity")
	EmailSubmissionType           = ObjectType("EmailSubmission")
	VacationResponseType          = ObjectType("VacationResponse")
	MDNType                       = ObjectType("MDN")
	QuotaType                     = ObjectType("Quota")
	SieveScriptType               = ObjectType("SieveScript")
	PrincipalType                 = ObjectType("PrincipalType")
	ShareNotificationType         = ObjectType("ShareNotification")
	AddressBookType               = ObjectType("AddressBook")
	ContactCardType               = ObjectType("ContactCard")
	CalendarType                  = ObjectType("Calendar")
	CalendarEventType             = ObjectType("CalendarEvent")
	CalendarEventNotificationType = ObjectType("CalendarEventNotification")
	ParticipantIdentityType       = ObjectType("ParticipantIdentity")

	JmapKeywordPrefix    = "$"
	JmapKeywordSeen      = "$seen"
	JmapKeywordDraft     = "$draft"
	JmapKeywordFlagged   = "$flagged"
	JmapKeywordAnswered  = "$answered"
	JmapKeywordForwarded = "$forwarded"
	JmapKeywordPhishing  = "$phising"
	JmapKeywordJunk      = "$junk"
	JmapKeywordNotJunk   = "$notjunk"
	JmapKeywordMdnSent   = "$mdnsent"

	// https://www.iana.org/assignments/imap-mailbox-name-attributes/imap-mailbox-name-attributes.xhtml
	//JmapMailboxRoleAll        = "all"
	//JmapMailboxRoleArchive    = "archive"
	JmapMailboxRoleDrafts = "drafts"
	//JmapMailboxRoleFlagged    = "flagged"
	//JmapMailboxRoleImportant  = "important"
	JmapMailboxRoleInbox = "inbox"
	JmapMailboxRoleJunk  = "junk"
	JmapMailboxRoleSent  = "sent"
	//JmapMailboxRoleSubscribed = "subscribed"
	JmapMailboxRoleTrash = "trash"

	CalendarAlertType = TypeOfCalendarAlert("CalendarAlert")

	CalendarEventNotificationTypeOptionCreated   = CalendarEventNotificationTypeOption("created")
	CalendarEventNotificationTypeOptionUpdated   = CalendarEventNotificationTypeOption("updated")
	CalendarEventNotificationTypeOptionDestroyed = CalendarEventNotificationTypeOption("destroyed")

	PrincipalTypeOptionIndividual = PrincipalTypeOption("individual")
	PrincipalTypeOptionGroup      = PrincipalTypeOption("group")
	PrincipalTypeOptionResource   = PrincipalTypeOption("resource")
	PrincipalTypeOptionLocation   = PrincipalTypeOption("location")
	PrincipalTypeOptionOther      = PrincipalTypeOption("other")

	HttpDigestAlgorithmAdler32   = HttpDigestAlgorithm("adler32")
	HttpDigestAlgorithmCrc32c    = HttpDigestAlgorithm("crc32c")
	HttpDigestAlgorithmMd5       = HttpDigestAlgorithm("md5")
	HttpDigestAlgorithmSha       = HttpDigestAlgorithm("sha")
	HttpDigestAlgorithmSha256    = HttpDigestAlgorithm("sha-256")
	HttpDigestAlgorithmSha512    = HttpDigestAlgorithm("sha-512")
	HttpDigestAlgorithmUnixSum   = HttpDigestAlgorithm("unixsum")
	HttpDigestAlgorithmUnixcksum = HttpDigestAlgorithm("unixcksum")

	// The quota is measured in a number of data type objects.
	//
	// For example, a quota can have a limit of 50 `Mail` objects.
	ResourceTypeCount = ResourceType("count")

	// The quota is measured in size (in octets).
	//
	// For example, a quota can have a limit of 25000 octets.
	ResourceTypeOctets = ResourceType("octets")

	// The quota information applies to just the client's account.
	ScopeAccount = Scope("account")
	// The quota information applies to all accounts sharing this domain.
	ScopeDomain = Scope("domain")
	// The quota information applies to all accounts belonging to the server.
	ScopeGlobal = Scope("global")

	ActionModeManualAction    = ActionMode("manual-action")
	ActionModeAutomaticAction = ActionMode("automatic-action")

	SendingModeMdnSentManually      = SendingMode("mdn-sent-manually")
	SendingModeMdnSentAutomatically = SendingMode("mdn-sent-automatically")

	DispositionTypeOptionDeleted    = DispositionTypeOption("deleted")
	DispositionTypeOptionDispatched = DispositionTypeOption("dispatched")
	DispositionTypeOptionDisplayed  = DispositionTypeOption("displayed")
	DispositionTypeOptionProcessed  = DispositionTypeOption("processed")

	IncludeInAvailabilityAll       = IncludeInAvailability("all")
	IncludeInAvailabilityAttending = IncludeInAvailability("attending")
	IncludeInAvailabilityNone      = IncludeInAvailability("none")
)

var (
	ObjectTypes = []ObjectType{
		CoreType,
		PushSubscriptionType,
		MailboxType,
		ThreadType,
		EmailType,
		EmailDeliveryType,
		SearchSnippetType,
		IdentityType,
		EmailSubmissionType,
		VacationResponseType,
		MDNType,
		QuotaType,
		SieveScriptType,
		PrincipalType,
		ShareNotificationType,
		AddressBookType,
		ContactCardType,
		CalendarType,
		CalendarEventType,
		CalendarEventNotificationType,
		ParticipantIdentityType,
	}

	JmapMailboxRoles = []string{
		JmapMailboxRoleInbox,
		JmapMailboxRoleSent,
		JmapMailboxRoleDrafts,
		JmapMailboxRoleJunk,
		JmapMailboxRoleTrash,
	}

	CalendarEventNotificationOptionTypes = []CalendarEventNotificationTypeOption{
		CalendarEventNotificationTypeOptionCreated,
		CalendarEventNotificationTypeOptionUpdated,
		CalendarEventNotificationTypeOptionDestroyed,
	}

	PrincipalTypeOptions = []PrincipalTypeOption{
		PrincipalTypeOptionIndividual,
		PrincipalTypeOptionGroup,
		PrincipalTypeOptionResource,
		PrincipalTypeOptionLocation,
		PrincipalTypeOptionOther,
	}

	HttpDigestAlgorithms = []HttpDigestAlgorithm{
		HttpDigestAlgorithmAdler32,
		HttpDigestAlgorithmCrc32c,
		HttpDigestAlgorithmMd5,
		HttpDigestAlgorithmSha,
		HttpDigestAlgorithmSha256,
		HttpDigestAlgorithmSha512,
		HttpDigestAlgorithmUnixSum,
		HttpDigestAlgorithmUnixcksum,
	}

	ResourceTypes = []ResourceType{
		ResourceTypeCount,
		ResourceTypeOctets,
	}

	Scopes = []Scope{
		ScopeAccount,
		ScopeDomain,
		ScopeGlobal,
	}

	ActionModes = []ActionMode{
		ActionModeManualAction,
		ActionModeAutomaticAction,
	}

	SendingModes = []SendingMode{
		SendingModeMdnSentManually,
		SendingModeMdnSentAutomatically,
	}

	DispositionTypeOptions = []DispositionTypeOption{
		DispositionTypeOptionDeleted,
		DispositionTypeOptionDispatched,
		DispositionTypeOptionDisplayed,
		DispositionTypeOptionProcessed,
	}

	IncludeInAvailabilities = []IncludeInAvailability{
		IncludeInAvailabilityAll,
		IncludeInAvailabilityAttending,
		IncludeInAvailabilityNone,
	}
)

type SessionMailAccountCapabilities struct {
	// The maximum number of Mailboxes that can be can assigned to a single Email object.
	//
	// This MUST be an integer >= 1, or null for no limit (or rather, the limit is always
	// the number of Mailboxes in the account).
	MaxMailboxesPerEmail int `json:"maxMailboxesPerEmail"`

	// The maximum depth of the Mailbox hierarchy (i.e., one more than the maximum
	// number of ancestors a Mailbox may have), or null for no limit.
	MaxMailboxDepth int `json:"maxMailboxDepth"`

	// The maximum length, in (UTF-8) octets, allowed for the name of a Mailbox.
	//
	// This MUST be at least 100, although it is recommended servers allow more.
	MaxSizeMailboxName int `json:"maxSizeMailboxName"`

	// The maximum total size of attachments, in octets, allowed for a single Email object.
	//
	// A server MAY still reject the import or creation of an Email with a lower attachment size
	// total (for example, if the body includes several megabytes of text, causing the size of
	// the encoded MIME structure to be over some server-defined limit).
	//
	// Note that this limit is for the sum of unencoded attachment sizes. Users are generally
	// not knowledgeable about encoding overhead, etc., nor should they need to be, so marketing
	// and help materials normally tell them the “max size attachments”.
	//
	// This is the unencoded size they see on their hard drive, so this capability matches that
	// and allows the client to consistently enforce what the user understands as the limit.
	//
	// The server may separately have a limit for the total size of the message [RFC5322],
	// created by combining the attachments (often base64 encoded) with the message headers and bodies.
	//
	// For example, suppose the server advertises maxSizeAttachmentsPerEmail: 50000000 (50 MB).
	// The enforced server limit may be for a message size of 70000000 octets.
	// Even with base64 encoding and a 2 MB HTML body, 50 MB attachments would fit under this limit.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	MaxSizeAttachmentsPerEmail int `json:"maxSizeAttachmentsPerEmail"`

	// A list of all the values the server supports for the “property” field of the Comparator
	// object in an Email/query sort.
	//
	// This MAY include properties the client does not recognise (for example, custom properties
	// specified in a vendor extension). Clients MUST ignore any unknown properties in the list.
	EmailQuerySortOptions []string `json:"emailQuerySortOptions"`

	// If true, the user may create a Mailbox in this account with a null parentId.
	//
	// (Permission for creating a child of an existing Mailbox is given by the myRights property
	// on that Mailbox.)
	MayCreateTopLevelMailbox bool `json:"mayCreateTopLevelMailbox"`
}

type SessionSubmissionAccountCapabilities struct {
	// The number in seconds of the maximum delay the server supports in sending.
	//
	// This is 0 if the server does not support delayed send.
	MaxDelayedSend int `json:"maxDelayedSend"`

	// The set of SMTP submission extensions supported by the server, which the client may use
	// when creating an EmailSubmission object.
	//
	// Each key in the object is the ehlo-name, and the value is a list of ehlo-args.
	//
	// A JMAP implementation that talks to a submission server [RFC6409] SHOULD have a configuration
	// setting that allows an administrator to modify the set of submission EHLO capabilities it may
	// expose on this property.
	//
	// This allows a JMAP server to easily add access to a new submission extension without code changes.
	//
	// By default, the JMAP server should hide EHLO capabilities that have to do with the transport
	// mechanism and thus are only relevant to the JMAP server (for example, PIPELINING, CHUNKING, or STARTTLS).
	//
	// Examples of Submission extensions to include:
	//   - FUTURERELEASE [RFC4865]
	//   - SIZE [RFC1870]
	//   - DSN [RFC3461]
	//   - DELIVERYBY [RFC2852]
	//   - MT-PRIORITY [RFC6710]
	//
	// A JMAP server MAY advertise an extension and implement the semantics of that extension locally
	// on the JMAP server even if a submission server used by JMAP doesn’t implement it.
	//
	// The full IANA registry of submission extensions can be found at [iana.org].
	//
	// [RFC6409]: https://www.rfc-editor.org/rfc/rfc6409.html
	// [RFC4865]: https://www.rfc-editor.org/rfc/rfc4865.html
	// [RFC1870]: https://www.rfc-editor.org/rfc/rfc1870.html
	// [RFC3461]: https://www.rfc-editor.org/rfc/rfc3461.html
	// [RFC2852]: https://www.rfc-editor.org/rfc/rfc2852.html
	// [RFC6710]: https://www.rfc-editor.org/rfc/rfc6710.html
	// [iana.org]: https://www.iana.org/assignments/mail-parameters
	SubmissionExtensions map[string][]string `json:"submissionExtensions"`
}

// This represents support for the VacationResponse data type and associated API methods.
//
// The value of this property is an empty object in both the JMAP session capabilities
// property and an account’s accountCapabilities property.
type SessionVacationResponseAccountCapabilities struct {
}

type SessionSieveAccountCapabilities struct {
	// The maximum length, in octets, allowed for the name of a SieveScript.
	//
	// For compatibility with ManageSieve, this MUST be at least 512 (up to 128 Unicode characters).
	MaxSizeScriptName int `json:"maxSizeScriptName"`

	// The maximum size (in octets) of a Sieve script the server is willing to store for the user,
	// or null for no limit.
	MaxSizeScript int `json:"maxSizeScript"`

	// The maximum number of Sieve scripts the server is willing to store for the user, or null for no limit.
	MaxNumberScripts int `json:"maxNumberScripts"`

	// The maximum number of Sieve "redirect" actions a script can perform during a single evaluation,
	// or null for no limit.
	//
	// Note that this is different from the total number of "redirect" actions a script can contain.
	MaxNumberRedirects int `json:"maxNumberRedirects"`

	// A list of case-sensitive Sieve capability strings (as listed in the Sieve "require" action;
	// see [RFC5228, Section 3.2]) indicating the extensions supported by the Sieve engine.
	//
	// [RFC5228, Section 3.2]: https://www.rfc-editor.org/rfc/rfc5228.html#section-3.2
	SieveExtensions []string `json:"sieveExtensions"`

	// A list of URI scheme parts [RFC3986] for notification methods supported by the Sieve "enotify"
	// extension [RFC5435], or null if the extension is not supported by the Sieve engine.
	//
	// [RFC3986]: https://www.rfc-editor.org/rfc/rfc3986.html
	// [RFC5435]: https://www.rfc-editor.org/rfc/rfc5435.html
	NotificationMethods []string `json:"notificationMethods"`

	// A list of URI scheme parts [RFC3986] for externally stored list types supported by the
	// Sieve "extlists" extension [RFC6134], or null if the extension is not supported by the Sieve engine.
	//
	// [RFC3986]: https://www.rfc-editor.org/rfc/rfc3986.html
	// [RFC6134]: https://www.rfc-editor.org/rfc/rfc6134.html
	ExternalLists []string `json:"externalLists"`
}

type SessionBlobAccountCapabilities struct {
	// The maximum size of the blob (in octets) that the server will allow to be created
	// (including blobs created by concatenating multiple data sources together).
	//
	// Clients MUST NOT attempt to create blobs larger than this size.
	//
	// If this value is null, then clients are not required to limit the size of the blob
	// they try to create, though servers can always reject creation of blobs regardless of
	// size, e.g., due to lack of disk space or per-user rate limits.
	MaxSizeBlobSet int `json:"maxSizeBlobSet"`

	// The maximum number of DataSourceObjects allowed per creation in a Blob/upload.
	//
	// Servers MUST allow at least 64 DataSourceObjects per creation.
	MaxDataSources int `json:"maxDataSources"`

	// An array of data type names that are supported for Blob/lookup.
	//
	// If the server does not support lookups, then this will be the empty list.
	//
	// Note that the supportedTypeNames list may include private types that are not in the
	// "JMAP Data Types" registry defined by this document.
	//
	// Clients MUST ignore type names they do not recognise.
	SupportedTypeNames []string `json:"supportedTypeNames"`

	// An array of supported digest algorithms that are supported for Blob/get.
	//
	// If the server does not support calculating blob digests, then this will be the empty
	// list.
	//
	// Algorithms in this list MUST be present in the ["HTTP Digest Algorithm Values" registry]
	// defined by [RFC3230]; however, in JMAP, they must be lowercased, e.g., "md5" rather than
	// "MD5".
	//
	// Clients SHOULD prefer algorithms listed earlier in this list.
	//
	// ["HTTP Digest Algorithm Values" registry]: https://www.iana.org/assignments/http-dig-alg/http-dig-alg.xhtml
	SupportedDigestAlgorithms []HttpDigestAlgorithm `json:"supportedDigestAlgorithms"`
}

type SessionQuotaAccountCapabilities struct {
}

type SessionContactsAccountCapabilities struct {
	// The maximum number of AddressBooks that can be can assigned to a single ContactCard object.
	//
	// This MUST be an integer >= 1, or null for no limit (or rather, the limit is always the number of AddressBooks
	// in the account).
	MaxAddressBooksPerCard uint `json:"maxAddressBooksPerCard,omitzero"`

	// If true, the user may create an AddressBook in this account.
	MayCreateAddressBook bool `json:"mayCreateAddressBook"`
}

type SessionPrincipalsAccountCapabilities struct {
	// The id of the principal in this account that corresponds to the user fetching this object, if any.
	CurrentUserPrincipalId string `json:"currentUserPrincipalId,omitempty"`
}

type SessionPrincipalsOwnerAccountCapabilities struct {
	// The id of an account with the `urn:ietf:params:jmap:principals` capability that contains the
	// corresponding `Principal` object.
	AccountIdForPrincipal string `json:"accountIdForPrincipal,omitempty"`

	// The id of the `Principal` that owns this account.
	PrincipalId string `json:"principalId,omitempty"`
}

type SessionMDNAccountCapabilities struct {
}

type SessionAccountCapabilities struct {
	Mail             SessionMailAccountCapabilities             `json:"urn:ietf:params:jmap:mail"`
	Submission       SessionSubmissionAccountCapabilities       `json:"urn:ietf:params:jmap:submission"`
	VacationResponse SessionVacationResponseAccountCapabilities `json:"urn:ietf:params:jmap:vacationresponse"`
	Sieve            SessionSieveAccountCapabilities            `json:"urn:ietf:params:jmap:sieve"`
	Blob             SessionBlobAccountCapabilities             `json:"urn:ietf:params:jmap:blob"`
	Quota            SessionQuotaAccountCapabilities            `json:"urn:ietf:params:jmap:quota"`
	Contacts         SessionContactsAccountCapabilities         `json:"urn:ietf:params:jmap:contacts"`
	Principals       *SessionPrincipalsAccountCapabilities      `json:"urn:ietf:params:jmap:principals,omitempty"`
	PrincipalsOwner  *SessionPrincipalsOwnerAccountCapabilities `json:"urn:ietf:params:jmap:principals:owner,omitempty"`
	MDN              *SessionMDNAccountCapabilities             `json:"urn:ietf:params:jmap:mdn,omitempty"`
}

type Account struct {
	// A user-friendly string to show when presenting content from this account, e.g., the email address representing the owner of the account.
	Name string `json:"name,omitempty"`
	// This is true if the account belongs to the authenticated user rather than a group account or a personal account of another user that has been shared with them.
	IsPersonal bool `json:"isPersonal"`
	// This is true if the entire account is read-only.
	IsReadOnly          bool                       `json:"isReadOnly"`
	AccountCapabilities SessionAccountCapabilities `json:"accountCapabilities"`
}

type SessionCoreCapabilities struct {
	// The maximum file size, in octets, that the server will accept for a single file upload (for any purpose)
	MaxSizeUpload int `json:"maxSizeUpload"`
	// The maximum number of concurrent requests the server will accept to the upload endpoint.
	MaxConcurrentUpload int `json:"maxConcurrentUpload"`
	// The maximum size, in octets, that the server will accept for a single request to the API endpoint.
	MaxSizeRequest int `json:"maxSizeRequest"`
	// The maximum number of concurrent requests the server will accept to the API endpoint.
	MaxConcurrentRequests int `json:"maxConcurrentRequests"`
	// The maximum number of method calls the server will accept in a single request to the API endpoint.
	MaxCallsInRequest int `json:"maxCallsInRequest"`
	// The maximum number of objects that the client may request in a single /get type method call.
	MaxObjectsInGet int `json:"maxObjectsInGet"`
	// The maximum number of objects the client may send to create, update, or destroy in a single /set type method call.
	// This is the combined total, e.g., if the maximum is 10, you could not create 7 objects and destroy 6, as this would be 13 actions,
	// which exceeds the limit.
	MaxObjectsInSet int `json:"maxObjectsInSet"`
	// A list of identifiers for algorithms registered in the collation registry, as defined in [@!RFC4790], that the server
	// supports for sorting when querying records.
	CollationAlgorithms []string `json:"collationAlgorithms"`
}

type SessionMailCapabilities struct {
}

type SessionSubmissionCapabilities struct {
}

type SessionVacationResponseCapabilities struct {
}

type SessionSieveCapabilities struct {
}

type SessionBlobCapabilities struct {
}

type SessionQuotaCapabilities struct {
}

type SessionWebsocketCapabilities struct {
	// The wss-URI (see [Section 3 of RFC6455]) to use for initiating a JMAP-over-WebSocket
	// handshake (the "WebSocket URL endpoint" colloquially).
	//
	// [Section 3 of RFC6455]: https://www.rfc-editor.org/rfc/rfc6455.html#section-3
	Url string `json:"url"`

	// This is true if the server supports push notifications over the WebSocket,
	// as described in [Section 4.3.5 of RFC 8887].
	//
	// [Section 4.3.5 of RFC 8887]: https://www.rfc-editor.org/rfc/rfc8887.html#name-jmap-push-notifications
	SupportsPush bool `json:"supportsPush"`
}

type SessionContactsCapabilities struct {
}

type SessionPrincipalCapabilities struct {
}

type SessionMDNCapabilities struct {
}

type SessionCapabilities struct {
	Core             SessionCoreCapabilities             `json:"urn:ietf:params:jmap:core"`
	Mail             SessionMailCapabilities             `json:"urn:ietf:params:jmap:mail"`
	Submission       SessionSubmissionCapabilities       `json:"urn:ietf:params:jmap:submission"`
	VacationResponse SessionVacationResponseCapabilities `json:"urn:ietf:params:jmap:vacationresponse"`
	Sieve            SessionSieveCapabilities            `json:"urn:ietf:params:jmap:sieve"`
	Blob             SessionBlobCapabilities             `json:"urn:ietf:params:jmap:blob"`
	Quota            SessionQuotaCapabilities            `json:"urn:ietf:params:jmap:quota"`
	Websocket        SessionWebsocketCapabilities        `json:"urn:ietf:params:jmap:websocket"`
	Contacts         *SessionContactsCapabilities        `json:"urn:ietf:params:jmap:contacts"`
	Principals       *SessionPrincipalCapabilities       `json:"urn:ietf:params:jmap:principals"`
	MDN              *SessionMDNCapabilities             `json:"urn:ietf:params:jmap:mdn,omitempty"`
}

type SessionPrimaryAccounts struct {
	Core             string `json:"urn:ietf:params:jmap:core"`
	Mail             string `json:"urn:ietf:params:jmap:mail"`
	Submission       string `json:"urn:ietf:params:jmap:submission"`
	VacationResponse string `json:"urn:ietf:params:jmap:vacationresponse"`
	Sieve            string `json:"urn:ietf:params:jmap:sieve"`
	Blob             string `json:"urn:ietf:params:jmap:blob"`
	Quota            string `json:"urn:ietf:params:jmap:quota"`
	Websocket        string `json:"urn:ietf:params:jmap:websocket"`
}

type SessionState string

type State string

type SessionResponse struct {
	Capabilities SessionCapabilities `json:"capabilities"`

	Accounts map[string]Account `json:"accounts,omitempty"`

	// A map of capability URIs (as found in accountCapabilities) to the account id that is considered to be the user’s main or default
	// account for data pertaining to that capability.
	// If no account being returned belongs to the user, or in any other way there is no appropriate way to determine a default account,
	// there MAY be no entry for a particular URI, even though that capability is supported by the server (and in the capabilities object).
	// urn:ietf:params:jmap:core SHOULD NOT be present.
	PrimaryAccounts SessionPrimaryAccounts `json:"primaryAccounts"`

	// The username associated with the given credentials, or the empty string if none.
	Username string `json:"username,omitempty"`

	// The URL to use for JMAP API requests.
	ApiUrl string `json:"apiUrl,omitempty"`

	// The URL endpoint to use when downloading files, in URI Template (level 1) format [@!RFC6570].
	// The URL MUST contain variables called accountId, blobId, type, and name.
	DownloadUrl string `json:"downloadUrl,omitempty"`

	// The URL endpoint to use when uploading files, in URI Template (level 1) format [@!RFC6570].
	// The URL MUST contain a variable called accountId.
	UploadUrl string `json:"uploadUrl,omitempty"`

	// The URL to connect to for push events, as described in Section 7.3, in URI Template (level 1) format [@!RFC6570].
	// The URL MUST contain variables called types, closeafter, and ping.
	EventSourceUrl string `json:"eventSourceUrl,omitempty"`

	// A (preferably short) string representing the state of this object on the server.
	// If the value of any other property on the Session object changes, this string will change.
	// The current value is also returned on the API Response object (see Section 3.4), allowing clients to quickly
	// determine if the session information has changed (e.g., an account has been added or removed),
	// so they need to refetch the object.
	State SessionState `json:"state,omitempty"`
}

// Method level error types.
const (
	// Some internal server resource was temporarily unavailable.
	//
	// Attempting the same operation later (perhaps after a backoff with a random factor) may succeed.
	MethodLevelErrorServerUnavailable = "serverUnavailable"

	// An unexpected or unknown error occurred during the processing of the call.
	//
	// A description property should provide more details about the error. The method call made no changes
	// to the server’s state. Attempting the same operation again is expected to fail again.
	// Contacting the service administrator is likely necessary to resolve this problem if it is persistent.
	MethodLevelErrorServerFail = "serverFail"

	// Some, but not all, expected changes described by the method occurred.
	//
	// The client MUST resynchronise impacted data to determine server state. Use of this error is strongly discouraged.
	MethodLevelErrorServerPartialFail = "serverPartialFail"

	// The server does not recognise this method name.
	MethodLevelErrorUnknownMethod = "unknownMethod"

	// One of the arguments is of the wrong type or is otherwise invalid, or a required argument is missing.
	//
	// A description property MAY be present to help debug with an explanation of what the problem was.
	// This is a non-localised string, and it is not intended to be shown directly to end users.
	MethodLevelErrorInvalidArguments = "invalidArguments"

	// The method used a result reference for one of its arguments, but this failed to resolve.
	MethodLevelErrorInvalidResultReference = "invalidResultReference"

	// The method and arguments are valid, but executing the method would violate an Access Control List
	// (ACL) or other permissions policy.
	MethodLevelErrorForbidden = "forbidden"

	// The accountId does not correspond to a valid account.
	MethodLevelErrorAccountNotFound = "accountNotFound"

	// The accountId given corresponds to a valid account, but the account does not support this method or data type.
	MethodLevelErrorAccountNotSupportedByMethod = "accountNotSupportedByMethod"

	// This method modifies state, but the account is read-only (as returned on the corresponding Account object in
	// the JMAP Session resource).
	MethodLevelErrorAccountReadOnly = "accountReadOnly"
)

// SetError type values.
const (
	// The create/update/destroy would violate an ACL or other permissions policy.
	//
	// (create; update; destroy).
	SetErrorTypeForbidden = "forbidden"

	// The create would exceed a server-defined limit on the number or total size of objects of this type.
	//
	// (create; update).
	SetErrorTypeOverQuota = "overQuota"

	// The create/update would result in an object that exceeds a server-defined limit for the maximum
	// size of a single object of this type.
	//
	// (create; update).
	SetErrorTypeTooLarge = "tooLarge"

	// Too many objects of this type have been created recently, and a server-defined rate limit has been reached.
	// It may work if tried again later.
	//
	// (create).
	SetErrorTypeRateLimit = "rateLimit"

	// The id given to update/destroy cannot be found.
	//
	// (update; destroy).
	SetErrorTypeNotFound = "notFound"

	// The PatchObject given to update the record was not a valid patch (see the patch description).
	//
	// (update).
	SetErrorTypeInvalidPatch = "invalidPatch"

	// The client requested that an object be both updated and destroyed in the same /set request, and the server
	// has decided to therefore ignore the update.
	//
	// (update).
	SetErrorTypeWillDestroy = "willDestroy"

	// The record given is invalid in some way. For example:
	//
	//   - It contains properties that are invalid according to the type specification of this record type.
	//   - It contains a property that may only be set by the server (e.g., “id”) and is different to the current value.
	//     Note, to allow clients to pass whole objects back, it is not an error to include a server-set property in an
	//     update as long as the value is identical to the current value on the server.
	//   - There is a reference to another record (foreign key), and the given id does not correspond to a valid record.
	//
	// The SetError object SHOULD also have a property called properties of type String[] that lists all the properties
	// that were invalid.
	//
	// Individual methods MAY specify more specific errors for certain conditions that would otherwise result in an
	// invalidProperties error. If the condition of one of these is met, it MUST be returned instead of the invalidProperties error.
	//
	// (create; update).
	SetErrorTypeInvalidProperties = "invalidProperties"

	// This is a singleton type, so you cannot create another one or destroy the existing one.
	//
	// (create; destroy).
	SetErrorTypeSingleton = "singleton"

	// The total number of objects to create, update, or destroy exceeds the maximum number the server is
	// willing to process in a single method call.
	SetErrorTypeRequestTooLarge = "requestTooLarge"

	// An ifInState argument was supplied, and it does not match the current state.
	SetErrorTypeStateMismatch = "stateMismatch"

	// The Email to be sent is invalid in some way.
	//
	// The SetError SHOULD contain a property called properties of type String[] that lists all the properties
	// of the Email that were invalid.
	SetErrorInvalidEmail = "invalidEmail"

	// The envelope (supplied or generated) has more recipients than the server allows.
	//
	// A maxRecipients UnsignedInt property MUST also be present on the SetError specifying
	// the maximum number of allowed recipients.
	SetErrorTooManyRecipients = "tooManyRecipients"

	// The envelope (supplied or generated) does not have any rcptTo email addresses.
	SetErrorNoRecipients = "noRecipients"

	// The rcptTo property of the envelope (supplied or generated) contains at least one rcptTo value which
	// is not a valid email address for sending to.
	//
	// An invalidRecipients String[] property MUST also be present on the SetError, which is a list of the invalid addresses.
	SetErrorInvalidRecipients = "invalidRecipients"

	// The server does not permit the user to send a message with this envelope From address [RFC5321].
	//
	// [RFC5321]: https://datatracker.ietf.org/doc/html/rfc5321
	SetErrorForbiddenMailFrom = "forbiddenMailFrom"

	// The server does not permit the user to send a message with the From header field [RFC5322] of the message to be sent.
	//
	// [RFC5322]: https://datatracker.ietf.org/doc/html/rfc5322
	SetErrorForbiddenFrom = "forbiddenFrom"

	// The user does not have permission to send at all right now for some reason.
	//
	// A description String property MAY be present on the SetError object to display to the user why they are not permitted.
	SetErrorForbiddenToSend = "forbiddenToSend"

	// The message has the `$mdnsent` keyword already set.
	SetErrorMdnAlreadySent = "mdnAlreadySent"
)

type SetError struct {
	// The type of error.
	Type string `json:"type"`

	// A description of the error to help with debugging that includes an explanation of what the problem was.
	//
	// This is a non-localised string and is not intended to be shown directly to end users.
	Description string `json:"description,omitempty"`

	// Lists all the properties of the Email that were invalid.
	//
	// Only set for the invalidEmail error after a failed EmailSubmission/set errors.
	Properties []string `json:"properties,omitempty"`

	// Specifies the maximum number of allowed recipients.
	//
	// Only set for the tooManyRecipients error after a failed EmailSubmission/set errors.
	MaxRecipients int `json:"maxRecipients,omitzero"`

	// List of invalid addresses.
	//
	// Only set for the invalidRecipients error after a failed EmailSubmission/set errors.
	InvalidRecipients []string `json:"invalidRecipients,omitempty"`
}

type FilterOperatorTerm string

const (
	And FilterOperatorTerm = "AND"
	Or  FilterOperatorTerm = "OR"
	Not FilterOperatorTerm = "NOT"
)

type MailboxRights struct {
	// If true, the user may use this Mailbox as part of a filter in an Email/query call,
	// and the Mailbox may be included in the mailboxIds property of Email objects.
	//
	// Email objects may be fetched if they are in at least one Mailbox with this permission.
	//
	// If a sub-Mailbox is shared but not the parent Mailbox, this may be false.
	//
	// Corresponds to IMAP ACLs lr (if mapping from IMAP, both are required for this to be true).
	MayReadItems bool `json:"mayReadItems"`

	// The user may add mail to this Mailbox (by either creating a new Email or moving an existing one).
	//
	// Corresponds to IMAP ACL i.
	MayAddItems bool `json:"mayAddItems"`

	// The user may remove mail from this Mailbox (by either changing the Mailboxes of an Email or
	// destroying the Email).
	//
	// Corresponds to IMAP ACLs te (if mapping from IMAP, both are required for this to be true).
	MayRemoveItems bool `json:"mayRemoveItems"`

	// The user may add or remove the $seen keyword to/from an Email.
	//
	// If an Email belongs to multiple Mailboxes, the user may only modify $seen if they have this
	// permission for all of the Mailboxes.
	//
	// Corresponds to IMAP ACL s.
	MaySetSeen bool `json:"maySetSeen"`

	// The user may add or remove any keyword other than $seen to/from an Email.
	//
	// If an Email belongs to multiple Mailboxes, the user may only modify keywords if they have this
	// permission for all of the Mailboxes.
	//
	// Corresponds to IMAP ACL w.
	MaySetKeywords bool `json:"maySetKeywords"`

	// The user may create a Mailbox with this Mailbox as its parent.
	//
	// Corresponds to IMAP ACL k.
	MayCreateChild bool `json:"mayCreateChild"`

	// The user may rename the Mailbox or make it a child of another Mailbox.
	//
	// Corresponds to IMAP ACL x (although this covers both rename and delete permissions).
	MayRename bool `json:"mayRename"`

	// The user may delete the Mailbox itself.
	//
	// Corresponds to IMAP ACL x (although this covers both rename and delete permissions).
	MayDelete bool `json:"mayDelete"`

	// Messages may be submitted directly to this Mailbox.
	//
	// Corresponds to IMAP ACL p.
	MaySubmit bool `json:"maySubmit"`
}

type Mailbox struct {
	// The id of the Mailbox.
	Id string `json:"id,omitempty"`

	// User-visible name for the Mailbox, e.g., “Inbox”.
	//
	// This MUST be a Net-Unicode string [@!RFC5198] of at least 1 character in length, subject to the maximum size
	// given in the capability object.
	//
	// There MUST NOT be two sibling Mailboxes with both the same parent and the same name.
	//
	// Servers MAY reject names that violate server policy (e.g., names containing a slash (/) or control characters).
	Name string `json:"name,omitempty"`

	// The Mailbox id for the parent of this Mailbox, or null if this Mailbox is at the top level.
	//
	// Mailboxes form acyclic graphs (forests) directed by the child-to-parent relationship. There MUST NOT be a loop.
	ParentId string `json:"parentId,omitempty"`

	// Identifies Mailboxes that have a particular common purpose (e.g., the “inbox”), regardless of the name property
	// (which may be localised).
	//
	// This value is shared with IMAP (exposed in IMAP via the SPECIAL-USE extension [RFC6154]).
	// However, unlike in IMAP, a Mailbox MUST only have a single role, and there MUST NOT be two Mailboxes in the same
	// account with the same role.
	//
	// Servers providing IMAP access to the same data are encouraged to enforce these extra restrictions in IMAP as well.
	// Otherwise, modifying the IMAP attributes to ensure compliance when exposing the data over JMAP is implementation dependent.
	//
	// The value MUST be one of the Mailbox attribute names listed in the IANA IMAP Mailbox Name Attributes registry,
	// as established in [RFC8457], converted to lowercase. New roles may be established here in the future.
	//
	// An account is not required to have Mailboxes with any particular roles.
	//
	// [RFC6154]: https://www.rfc-editor.org/rfc/rfc6154.html
	// [RFC8457]: https://www.rfc-editor.org/rfc/rfc8457.html
	Role string `json:"role,omitempty"`

	// Defines the sort order of Mailboxes when presented in the client’s UI, so it is consistent between devices.
	//
	// Default value: 0
	//
	// The number MUST be an integer in the range 0 <= sortOrder < 2^31.
	//
	// A Mailbox with a lower order should be displayed before a Mailbox with a higher order
	// (that has the same parent) in any Mailbox listing in the client’s UI.
	// Mailboxes with equal order SHOULD be sorted in alphabetical order by name.
	// The sorting should take into account locale-specific character order convention.
	SortOrder int `json:"sortOrder,omitzero"`

	// The number of Emails in this Mailbox.
	TotalEmails int `json:"totalEmails"`

	// The number of Emails in this Mailbox that have neither the $seen keyword nor the $draft keyword.
	UnreadEmails int `json:"unreadEmails"`

	// The number of Threads where at least one Email in the Thread is in this Mailbox.
	TotalThreads int `json:"totalThreads"`

	// An indication of the number of “unread” Threads in the Mailbox.
	UnreadThreads int `json:"unreadThreads"`

	// The set of rights (Access Control Lists (ACLs)) the user has in relation to this Mailbox.
	//
	// These are backwards compatible with IMAP ACLs, as defined in [RFC4314].
	//
	// [RFC4314]: https://www.rfc-editor.org/rfc/rfc4314.html
	MyRights MailboxRights `json:"myRights,omitempty"`

	// Has the user indicated they wish to see this Mailbox in their client?
	//
	// This SHOULD default to false for Mailboxes in shared accounts the user has access to and true
	// for any new Mailboxes created by the user themself.
	//
	// This MUST be stored separately per user where multiple users have access to a shared Mailbox.
	//
	// A user may have permission to access a large number of shared accounts, or a shared account with a very
	// large set of Mailboxes, but only be interested in the contents of a few of these.
	//
	// Clients may choose to only display Mailboxes where the isSubscribed property is set to true, and offer
	// a separate UI to allow the user to see and subscribe/unsubscribe from the full set of Mailboxes.
	//
	// However, clients MAY choose to ignore this property, either entirely for ease of implementation or just
	// for an account where isPersonal is true (indicating it is the user’s own rather than a shared account).
	//
	// This property corresponds to IMAP [RFC3501] Mailbox subscriptions.
	//
	// [RFC3501]: https://www.rfc-editor.org/rfc/rfc3501.html
	IsSubscribed bool `json:"isSubscribed"`
}

type MailboxGetCommand struct {
	AccountId string   `json:"accountId"`
	Ids       []string `json:"ids,omitempty"`
}

type MailboxGetRefCommand struct {
	AccountId string           `json:"accountId"`
	IdsRef    *ResultReference `json:"#ids,omitempty"`
}

type MailboxChangesCommand struct {
	// The id of the account to use.
	AccountId string `json:"accountId"`

	// The current state of the client.
	//
	// This is the string that was returned as the state argument in the Mailbox/get response.
	//
	// The server will return the changes that have occurred since this state.
	SinceState string `json:"sinceState,omitempty"`

	// The maximum number of ids to return in the response.
	//
	// The server MAY choose to return fewer than this value but MUST NOT return more.
	//
	// If not given by the client, the server may choose how many to return.
	//
	// If supplied by the client, the value MUST be a positive integer greater than 0.
	//
	// If a value outside of this range is given, the server MUST reject the call with an invalidArguments error.
	MaxChanges uint `json:"maxChanges,omitzero"`
}

type MailboxFilterElement interface {
	_isAMailboxFilterElement() // marker method
}

type MailboxFilterCondition struct {
	MailboxFilterElement
	ParentId     string `json:"parentId,omitempty"`
	Name         string `json:"name,omitempty"`
	Role         string `json:"role,omitempty"`
	HasAnyRole   *bool  `json:"hasAnyRole,omitempty"`
	IsSubscribed *bool  `json:"isSubscribed,omitempty"`
}

var _ MailboxFilterElement = &MailboxFilterCondition{}

type MailboxFilterOperator struct {
	MailboxFilterElement
	Operator   FilterOperatorTerm     `json:"operator"`
	Conditions []MailboxFilterElement `json:"conditions,omitempty"`
}

var _ MailboxFilterElement = &MailboxFilterOperator{}

type MailboxComparator struct {
	Property       string `json:"property"`
	IsAscending    bool   `json:"isAscending,omitempty"`
	Limit          int    `json:"limit,omitzero"`
	CalculateTotal bool   `json:"calculateTotal,omitempty"`
}

type MailboxQueryCommand struct {
	AccountId    string               `json:"accountId"`
	Filter       MailboxFilterElement `json:"filter,omitempty"`
	Sort         []MailboxComparator  `json:"sort,omitempty"`
	SortAsTree   bool                 `json:"sortAsTree,omitempty"`
	FilterAsTree bool                 `json:"filterAsTree,omitempty"`
}

type EmailFilterElement interface {
	_isAnEmailFilterElement() // marker method
	IsNotEmpty() bool
}

type EmailFilterCondition struct {
	//  A Mailbox id.
	//
	// An Email must be in this Mailbox to match the condition.
	InMailbox string `json:"inMailbox,omitempty"`

	// A list of Mailbox ids.
	//
	// An Email must be in at least one Mailbox not in this list to match the condition.
	//
	// This is to allow messages solely in trash/spam to be easily excluded from a search.
	InMailboxOtherThan []string `json:"inMailboxOtherThan,omitempty"`

	// The receivedAt date-time of the Email must be before this date-time to match
	// the condition.
	Before time.Time `json:"before,omitzero"` // omitzero requires Go 1.24

	// The receivedAt date-time of the Email must be the same or after this date-time
	// to match the condition.
	After time.Time `json:"after,omitzero"`

	// The size property of the Email must be equal to or greater than this number to match
	// the condition.
	MinSize int `json:"minSize,omitempty"`

	// The size property of the Email must be less than this number to match the condition.
	MaxSize int `json:"maxSize,omitempty"`

	// All Emails (including this one) in the same Thread as this Email must have the given
	// keyword to match the condition.
	AllInThreadHaveKeyword string `json:"allInThreadHaveKeyword,omitempty"`

	// At least one Email (possibly this one) in the same Thread as this Email must have the
	// given keyword to match the condition.
	SomeInThreadHaveKeyword string `json:"someInThreadHaveKeyword,omitempty"`

	// All Emails (including this one) in the same Thread as this Email must not have the
	// given keyword to match the condition.
	NoneInThreadHaveKeyword string `json:"noneInThreadHaveKeyword,omitempty"`

	// This Email must have the given keyword to match the condition.
	HasKeyword string `json:"hasKeyword,omitempty"`

	// This Email must not have the given keyword to match the condition.
	NotKeyword string `json:"notKeyword,omitempty"`

	// The hasAttachment property of the Email must be identical to the value given to match
	// the condition.
	HasAttachment bool `json:"hasAttachment,omitempty"`

	// Looks for the text in Emails.
	//
	// The server MUST look up text in the From, To, Cc, Bcc, and Subject header fields of the
	// message and SHOULD look inside any text/* or other body parts that may be converted to
	// text by the server.
	//
	// The server MAY extend the search to any additional textual property.
	Text string `json:"text,omitempty"`

	// Looks for the text in the From header field of the message.
	From string `json:"from,omitempty"`

	// Looks for the text in the To header field of the message.
	To string `json:"to,omitempty"`

	// Looks for the text in the Cc header field of the message.
	Cc string `json:"cc,omitempty"`

	// Looks for the text in the Bcc header field of the message.
	Bcc string `json:"bcc,omitempty"`

	// Looks for the text in the Subject header field of the message.
	Subject string `json:"subject,omitempty"`

	// Looks for the text in one of the body parts of the message.
	//
	// The server MAY exclude MIME body parts with content media types other than text/*
	// and message/* from consideration in search matching.
	//
	// Care should be taken to match based on the text content actually presented to an end user
	// by viewers for that media type or otherwise identified as appropriate for search indexing.
	//
	// Matching document metadata uninteresting to an end user (e.g., markup tag and attribute
	// names) is undesirable.
	Body string `json:"body,omitempty"`

	// The array MUST contain either one or two elements.
	//
	// The first element is the name of the header field to match against.
	//
	// The second (optional) element is the text to look for in the header field value.
	//
	// If not supplied, the message matches simply if it has a header field of the given name.
	Header []string `json:"header,omitempty"`
}

func (f EmailFilterCondition) _isAnEmailFilterElement() {
}

func (f EmailFilterCondition) IsNotEmpty() bool {
	if !f.After.IsZero() {
		return true
	}
	if f.AllInThreadHaveKeyword != "" {
		return true
	}
	if len(f.Bcc) > 0 {
		return true
	}
	if !f.Before.IsZero() {
		return true
	}
	if f.Body != "" {
		return true
	}
	if f.Cc != "" {
		return true
	}
	if f.From != "" {
		return true
	}
	if f.HasAttachment {
		return true
	}
	if f.HasKeyword != "" {
		return true
	}
	if len(f.Header) > 0 {
		return true
	}
	if f.InMailbox != "" {
		return true
	}
	if len(f.InMailboxOtherThan) > 0 {
		return true
	}
	if f.MaxSize != 0 {
		return true
	}
	if f.MinSize != 0 {
		return true
	}
	if f.NoneInThreadHaveKeyword != "" {
		return true
	}
	if f.NotKeyword != "" {
		return true
	}
	if f.SomeInThreadHaveKeyword != "" {
		return true
	}
	if f.Subject != "" {
		return true
	}
	if f.Text != "" {
		return true
	}
	if f.To != "" {
		return true
	}
	return false
}

var _ EmailFilterElement = &EmailFilterCondition{}

type EmailFilterOperator struct {
	Operator   FilterOperatorTerm   `json:"operator"`
	Conditions []EmailFilterElement `json:"conditions,omitempty"`
}

func (o EmailFilterOperator) _isAnEmailFilterElement() {
}

func (o EmailFilterOperator) IsNotEmpty() bool {
	return len(o.Conditions) > 0
}

var _ EmailFilterElement = &EmailFilterOperator{}

type EmailComparator struct {
	// The name of the property on the objects to compare.
	Property string `json:"property,omitempty"`

	// If true, sort in ascending order.
	//
	// Optional; default value: true.
	//
	// If false, reverse the comparator’s results to sort in descending order.
	IsAscending bool `json:"isAscending,omitempty"`

	// The identifier, as registered in the collation registry defined in [RFC4790],
	// for the algorithm to use when comparing the order of strings.
	//
	// Optional; default is server dependent.
	//
	// The algorithms the server supports are advertised in the capabilities object returned
	// with the Session object.
	//
	// [RFC4790]: https://www.rfc-editor.org/rfc/rfc4790.html
	Collation string `json:"collation,omitempty"`

	// Email-specific: keyword that must be included in the Email object.
	Keyword string `json:"keyword,omitempty"`
}

// If an anchor argument is given, the anchor is looked for in the results after filtering
// and sorting.
//
// If found, the anchorOffset is then added to its index. If the resulting index is now negative,
// it is clamped to 0. This index is now used exactly as though it were supplied as the position
// argument. If the anchor is not found, the call is rejected with an anchorNotFound error.
//
// If an anchor is specified, any position argument supplied by the client MUST be ignored.
// If no anchor is supplied, any anchorOffset argument MUST be ignored.
//
// A client can use anchor instead of position to find the index of an id within a large set of results.
type EmailQueryCommand struct {
	// The id of the account to use.
	AccountId string `json:"accountId"`

	// Determines the set of Emails returned in the results.
	//
	// If null, all objects in the account of this type are included in the results.
	Filter EmailFilterElement `json:"filter,omitempty"`

	// Lists the names of properties to compare between two Email records, and how to compare
	// them, to determine which comes first in the sort.
	//
	// If two Email records have an identical value for the first comparator, the next comparator
	// will be considered, and so on. If all comparators are the same (this includes the case
	// where an empty array or null is given as the sort argument), the sort order is server
	// dependent, but it MUST be stable between calls to Email/query.
	Sort []EmailComparator `json:"sort,omitempty"`

	// If true, Emails in the same Thread as a previous Email in the list (given the
	// filter and sort order) will be removed from the list.
	//
	// This means only one Email at most will be included in the list for any given Thread.
	CollapseThreads bool `json:"collapseThreads,omitempty"`

	// The zero-based index of the first id in the full list of results to return.
	//
	// If a negative value is given, it is an offset from the end of the list.
	// Specifically, the negative value MUST be added to the total number of results given
	// the filter, and if still negative, it’s clamped to 0. This is now the zero-based
	// index of the first id to return.
	//
	// If the index is greater than or equal to the total number of objects in the results
	// list, then the ids array in the response will be empty, but this is not an error.
	Position uint `json:"position,omitempty"`

	// An Email id.
	//
	// If supplied, the position argument is ignored.
	// The index of this id in the results will be used in combination with the anchorOffset
	// argument to determine the index of the first result to return.
	Anchor string `json:"anchor,omitempty"`

	// The index of the first result to return relative to the index of the anchor,
	// if an anchor is given.
	//
	// Default: 0.
	//
	// This MAY be negative.
	//
	// For example, -1 means the Email immediately preceding the anchor is the first result in
	// the list returned.
	AnchorOffset int `json:"anchorOffset,omitzero"`

	// The maximum number of results to return.
	//
	// If null, no limit presumed.
	// The server MAY choose to enforce a maximum limit argument.
	// In this case, if a greater value is given (or if it is null), the limit is clamped
	// to the maximum; the new limit is returned with the response so the client is aware.
	//
	// If a negative value is given, the call MUST be rejected with an invalidArguments error.
	Limit uint `json:"limit,omitempty"`

	// Does the client wish to know the total number of results in the query?
	//
	// This may be slow and expensive for servers to calculate, particularly with complex filters,
	// so clients should take care to only request the total when needed.
	CalculateTotal bool `json:"calculateTotal,omitempty"`
}

type EmailGetCommand struct {
	// The ids of the Email objects to return.
	//
	// If null, then all records of the data type are returned, if this is supported for that
	// data type and the number of records does not exceed the maxObjectsInGet limit.
	Ids []string `json:"ids,omitempty"`

	// The id of the account to use.
	AccountId string `json:"accountId"`

	// If supplied, only the properties listed in the array are returned for each Email object.
	//
	// If null, the following properties are returned:
	//
	//    [ "id", "blobId", "threadId", "mailboxIds", "keywords", "size",
	//      "receivedAt", "messageId", "inReplyTo", "references", "sender", "from",
	//      "to", "cc", "bcc", "replyTo", "subject", "sentAt", "hasAttachment",
	//      "preview", "bodyValues", "textBody", "htmlBody", "attachments" ]
	//
	// The id property of the object is always returned, even if not explicitly requested.
	//
	// If an invalid property is requested, the call MUST be rejected with an invalidArguments error.
	Properties []string `json:"properties,omitempty"`

	// A list of properties to fetch for each EmailBodyPart returned.
	//
	// If omitted, this defaults to:
	//
	//    [ "partId", "blobId", "size", "name", "type", "charset", "disposition", "cid", "language", "location" ]
	//
	BodyProperties []string `json:"bodyProperties,omitempty"`

	// (default: false) If true, the bodyValues property includes any text/* part in the textBody property.
	FetchTextBodyValues bool `json:"fetchTextBodyValues,omitzero"`

	// (default: false) If true, the bodyValues property includes any text/* part in the htmlBody property.
	FetchHTMLBodyValues bool `json:"fetchHTMLBodyValues,omitzero"`

	// (default: false) If true, the bodyValues property includes any text/* part in the bodyStructure property.
	FetchAllBodyValues bool `json:"fetchAllBodyValues,omitzero"`

	// If greater than zero, the value property of any EmailBodyValue object returned in bodyValues
	// MUST be truncated if necessary so it does not exceed this number of octets in size.
	//
	// If 0 (the default), no truncation occurs.
	//
	// The server MUST ensure the truncation results in valid UTF-8 and does not occur mid-codepoint.
	//
	// If the part is of type text/html, the server SHOULD NOT truncate inside an HTML tag, e.g., in
	// the middle of <a href="https://example.com">.
	//
	// There is no requirement for the truncated form to be a balanced tree or valid HTML (indeed, the original
	// source may well be neither of these things).
	MaxBodyValueBytes uint `json:"maxBodyValueBytes,omitempty"`
}

// Reference to Previous Method Results
//
// To allow clients to make more efficient use of the network and avoid round trips, an argument to one method
// can be taken from the result of a previous method call in the same request.
//
// To do this, the client prefixes the argument name with # (an [octothorpe]).
//
// When processing a method call, the server MUST first check the arguments object for any names beginning with #.
//
// If found, the result reference should be resolved and the value used as the “real” argument.
//
// The method is then processed as normal.
//
// If any result reference fails to resolve, the whole method MUST be rejected with an invalidResultReference error.
//
// If an arguments object contains the same argument name in normal and referenced form (e.g., foo and #foo),
// the method MUST return an invalidArguments error.
//
// To resolve:
//
//  1. Find the first response with a method call id identical to the resultOf property of the ResultReference
//     in the methodResponses array from previously processed method calls in the same request.
//     If none, evaluation fails.
//  2. If the response name is not identical to the name property of the ResultReference, evaluation fails.
//  3. Apply the path to the arguments object of the response (the second item in the response array)
//     following the JSON Pointer algorithm [RFC6901], except with the following addition in “Evaluation” (see Section 4):
//  4. If the currently referenced value is a JSON array, the reference token may be exactly the single character *,
//     making the new referenced value the result of applying the rest of the JSON Pointer tokens to every item in the
//     array and returning the results in the same order in a new array.
//  5. If the result of applying the rest of the pointer tokens to each item was itself an array, the contents of this
//     array are added to the output rather than the array itself (i.e., the result is flattened from an array of
//     arrays to a single array).
//
// [octothorpe]; https://en.wiktionary.org/wiki/octothorpe
// [RFC6901]: https://datatracker.ietf.org/doc/html/rfc6901
type ResultReference struct {
	// The method call id of a previous method call in the current request.
	ResultOf string `json:"resultOf"`

	// The required name of a response to that method call.
	Name Command `json:"name"`

	// A pointer into the arguments of the response selected via the name and resultOf properties.
	//
	// This is a JSON Pointer [RFC6901], except it also allows the use of * to map through an array.
	//
	// [RFC6901]: https://datatracker.ietf.org/doc/html/rfc6901
	Path string `json:"path,omitempty"`
}

type EmailGetRefCommand struct {
	// The ids of the Email objects to return.
	//
	// If null, then all records of the data type are returned, if this is supported for that
	// data type and the number of records does not exceed the maxObjectsInGet limit.
	IdsRef *ResultReference `json:"#ids,omitempty"`

	// The id of the account to use.
	AccountId string `json:"accountId"`

	// If supplied, only the properties listed in the array are returned for each Email object.
	//
	// If null, the following properties are returned:
	//
	//    [ "id", "blobId", "threadId", "mailboxIds", "keywords", "size",
	//      "receivedAt", "messageId", "inReplyTo", "references", "sender", "from",
	//      "to", "cc", "bcc", "replyTo", "subject", "sentAt", "hasAttachment",
	//      "preview", "bodyValues", "textBody", "htmlBody", "attachments" ]
	//
	// The id property of the object is always returned, even if not explicitly requested.
	//
	// If an invalid property is requested, the call MUST be rejected with an invalidArguments error.
	Properties []string `json:"properties,omitempty"`

	// A list of properties to fetch for each EmailBodyPart returned.
	//
	// If omitted, this defaults to:
	//
	//    [ "partId", "blobId", "size", "name", "type", "charset", "disposition", "cid", "language", "location" ]
	//
	BodyProperties []string `json:"bodyProperties,omitempty"`

	// (default: false) If true, the bodyValues property includes any text/* part in the textBody property.
	FetchTextBodyValues bool `json:"fetchTextBodyValues,omitzero"`

	// (default: false) If true, the bodyValues property includes any text/* part in the htmlBody property.
	FetchHTMLBodyValues bool `json:"fetchHTMLBodyValues,omitzero"`

	// (default: false) If true, the bodyValues property includes any text/* part in the bodyStructure property.
	FetchAllBodyValues bool `json:"fetchAllBodyValues,omitzero"`

	// If greater than zero, the value property of any EmailBodyValue object returned in bodyValues
	// MUST be truncated if necessary so it does not exceed this number of octets in size.
	//
	// If 0 (the default), no truncation occurs.
	//
	// The server MUST ensure the truncation results in valid UTF-8 and does not occur mid-codepoint.
	//
	// If the part is of type text/html, the server SHOULD NOT truncate inside an HTML tag, e.g., in
	// the middle of <a href="https://example.com">.
	//
	// There is no requirement for the truncated form to be a balanced tree or valid HTML (indeed, the original
	// source may well be neither of these things).
	MaxBodyValueBytes uint `json:"maxBodyValueBytes,omitempty"`
}

type EmailChangesCommand struct {
	// The id of the account to use.
	AccountId string `json:"accountId"`

	// The current state of the client.
	//
	// This is the string that was returned as the state argument in the Email/get response.
	// The server will return the changes that have occurred since this state.
	SinceState string `json:"sinceState,omitzero,omitempty"`

	// The maximum number of ids to return in the response.
	//
	// The server MAY choose to return fewer than this value but MUST NOT return more.
	// If not given by the client, the server may choose how many to return.
	// If supplied by the client, the value MUST be a positive integer greater than 0.
	MaxChanges uint `json:"maxChanges,omitzero"`
}

type EmailAddress struct {
	// The display-name of the mailbox [RFC5322].
	//
	// If this is a quoted-string:
	//   1. The surrounding DQUOTE characters are removed.
	//   2. Any quoted-pair is decoded.
	//   3. White space is unfolded, and then any leading and trailing white space is removed.
	// If there is no display-name but there is a comment immediately following the addr-spec, the value of this
	// SHOULD be used instead. Otherwise, this property is null.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	//
	// example: $emailAddressName
	Name string `json:"name,omitempty"`

	// The addr-spec of the mailbox [RFC5322].
	//
	// Any syntactically correct encoded sections [RFC2047] with a known encoding MUST be decoded,
	// following the same rules as for the Text form.
	//
	// Parsing SHOULD be best effort in the face of invalid structure to accommodate invalid messages and
	// semi-complete drafts. EmailAddress objects MAY have an email property that does not conform to the
	// addr-spec form (for example, may not contain an @ symbol).
	//
	// [RFC2047]: https://www.rfc-editor.org/rfc/rfc2047.html
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	//
	// example: $emailAddressEmail
	Email string `json:"email,omitempty"`
}

type EmailAddressGroup struct {
	// The display-name of the group [RFC5322], or null if the addresses are not part of a group.
	//
	// If this is a quoted-string, it is processed the same as the name in the EmailAddress type.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html

	Name string `json:"name,omitempty"`

	// The mailbox values that belong to this group, represented as EmailAddress objects.
	Addresses []EmailAddress `json:"addresses,omitempty"`
}

type EmailHeader struct {
	// The header field name as defined in [RFC5322], with the same capitalization that it has in the message.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	Name string `json:"name"`

	// The header field value as defined in [RFC5322], in Raw form.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	Value string `json:"value"`
}

// Email body part.
//
// The client may specify a `partId` OR a `blobId`, but not both.
// If a `partId` is given, this `partId` MUST be present in the `bodyValues` property.
//
// The `charset` property MUST be omitted if a `partId` is given (the part’s content is included
// in `bodyValues`, and the server may choose any appropriate encoding).
//
// The `size` property MUST be omitted if a `partId` is given. If a `blobId` is given, it may be
// included but is ignored by the server (the size is actually calculated from the blob content
// itself).
//
// A `Content-Transfer-Encoding` header field MUST NOT be given.
type EmailBodyPart struct {
	// Identifies this part uniquely within the Email.
	//
	// This is scoped to the `emailId` and has no meaning outside of the JMAP Email object representation.
	// This is null if, and only if, the part is of type `multipart/*`.
	//
	// example: $attachmentPartId
	PartId string `json:"partId,omitempty"`

	// The id representing the raw octets of the contents of the part, after decoding any known
	// `Content-Transfer-Encoding` (as defined in [RFC2045]), or null if, and only if, the part is of type `multipart/*`.
	//
	// Note that two parts may be transfer-encoded differently but have the same blob id if their decoded octets are identical
	// and the server is using a secure hash of the data for the blob id.
	// If the transfer encoding is unknown, it is treated as though it had no transfer encoding.
	//
	// [RFC2045]: https://www.rfc-editor.org/rfc/rfc2045.html
	//
	// example: $blobId
	BlobId string `json:"blobId,omitempty"`

	// The size, in octets, of the raw data after content transfer decoding (as referenced by the `blobId`, i.e.,
	// the number of octets in the file the user would download).
	//
	// example: 31219
	Size int `json:"size,omitempty"`

	// This is a list of all header fields in the part, in the order they appear in the message.
	//
	// The values are in Raw form.
	Headers []EmailHeader `json:"headers,omitempty"`

	// This is the decoded filename parameter of the `Content-Disposition` header field per [RFC2231], or
	// (for compatibility with existing systems) if not present, then it’s the decoded name parameter of
	// the `Content-Type` header field per [RFC2047].
	//
	// [RFC2231]: https://www.rfc-editor.org/rfc/rfc2231.html
	// [RFC2047]: https://www.rfc-editor.org/rfc/rfc2047.html
	//
	// name: $attachmentName
	Name string `json:"name,omitempty"`

	// The value of the `Content-Type` header field of the part, if present; otherwise, the implicit type as per
	// the MIME standard (`text/plain` or `message/rfc822` if inside a `multipart/digest`).
	//
	// [CFWS] is removed and any parameters are stripped.
	//
	// [CFWS]: https://www.rfc-editor.org/rfc/rfc5322#section-3.2.2
	//
	// example: $attachmentType
	Type string `json:"type,omitempty"`

	// The value of the `charset` parameter of the `Content-Type` header field, if present, or null if the header
	// field is present but not of type `text/*`.
	//
	// If there is no `Content-Type` header field, or it exists and is of type `text/*` but has no `charset` parameter,
	// this is the implicit charset as per the MIME standard: `us-ascii`.
	//
	// example: $attachmentCharset
	Charset string `json:"charset,omitempty"`

	// The value of the `Content-Disposition` header field of the part, if present;
	// otherwise, it’s null.
	//
	// [CFWS] is removed and any parameters are stripped.
	//
	// [CFWS]: https://www.rfc-editor.org/rfc/rfc5322#section-3.2.2
	//
	// example: $attachmentDisposition
	Disposition string `json:"disposition,omitempty"`

	// The value of the `Content-Id` header field of the part, if present; otherwise it’s null.
	//
	// [CFWS] and surrounding angle brackets (`<>`) are removed.
	//
	// This may be used to reference the content from within a `text/html` body part HTML using the `cid:` protocol,
	// as defined in [RFC2392].
	//
	// [RFC2392]: https://www.rfc-editor.org/rfc/rfc2392.html
	// [CFWS]: https://www.rfc-editor.org/rfc/rfc5322#section-3.2.2
	//
	// example: $attachmentCid
	Cid string `json:"cid,omitempty"`

	// The list of language tags, as defined in [RFC3282], in the `Content-Language` header field of the part,
	// if present.
	//
	// [RFC3282]: https://www.rfc-editor.org/rfc/rfc3282.html
	Language string `json:"language,omitempty"`

	// The URI, as defined in [RFC2557], in the `Content-Location` header field of the part, if present.
	//
	// [RFC2557]: https://www.rfc-editor.org/rfc/rfc2557.html
	Location string `json:"location,omitempty"`

	// If the type is `multipart/*`, this contains the body parts of each child.
	SubParts []EmailBodyPart `json:"subParts,omitempty"`
}

type EmailBodyValue struct {
	// The value of the body part after decoding `Content-Transfer-Encoding` and the `Content-Type` charset,
	// if both known to the server, and with any CRLF replaced with a single LF.
	//
	// The server MAY use heuristics to determine the charset to use for decoding if the charset is unknown,
	// no charset is given, or it believes the charset given is incorrect.
	//
	// Decoding is best effort; the server SHOULD insert the unicode replacement character (`U+FFFD`) and continue
	// when a malformed section is encountered.
	//
	// Note that due to the charset decoding and line ending normalisation, the length of this string will
	// probably not be exactly the same as the size property on the corresponding EmailBodyPart.
	Value string `json:"value,omitempty"`

	// This is true if malformed sections were found while decoding the charset,
	// or the charset was unknown, or the content-transfer-encoding was unknown.
	//
	// Default value is false.
	IsEncodingProblem bool `json:"isEncodingProblem,omitzero"`

	// This is true if the value has been truncated.
	//
	// Default value is false.
	IsTruncated bool `json:"isTruncated,omitzero"`
}

// An Email.
//
// swagger:model
type Email struct {
	// The id of the Email object.
	//
	// Note that this is the JMAP object id, NOT the `Message-ID` header field value of the message [RFC5322].
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	//
	// required: true
	// example: $emailId
	Id string `json:"id,omitempty"`

	// The id representing the raw octets of the message [RFC5322] for this Email.
	//
	// This may be used to download the raw original message or to attach it directly to another Email, etc.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	//
	// example: $blobId
	BlobId string `json:"blobId,omitempty"`

	// The id of the Thread to which this Email belongs.
	//
	// example: $threadId
	ThreadId string `json:"threadId,omitempty"`

	// The set of Mailbox ids this Email belongs to.
	//
	// An Email in the mail store MUST belong to one or more Mailboxes at all times (until it is destroyed).
	// The set is represented as an object, with each key being a Mailbox id.
	//
	// The value for each key in the object MUST be true.
	//
	// example: {"a": true}
	MailboxIds map[string]bool `json:"mailboxIds,omitempty"`

	// A set of keywords that apply to the Email.
	//
	// The set is represented as an object, with the keys being the keywords.
	//
	// The value for each key in the object MUST be true.
	//
	// Keywords are shared with IMAP.
	//
	// The six system keywords from IMAP get special treatment.
	//
	// The following four keywords have their first character changed from \ in IMAP to $ in JMAP and have particular semantic meaning:
	//
	//   - $draft: The Email is a draft the user is composing.
	//   - $seen: The Email has been read.
	//   - $flagged: The Email has been flagged for urgent/special attention.
	//   - $answered: The Email has been replied to.
	//
	// The IMAP \Recent keyword is not exposed via JMAP. The IMAP \Deleted keyword is also not present: IMAP uses a delete+expunge model,
	// which JMAP does not. Any message with the \Deleted keyword MUST NOT be visible via JMAP (and so are not counted in the
	// “totalEmails”, “unreadEmails”, “totalThreads”, and “unreadThreads” Mailbox properties).
	//
	// Users may add arbitrary keywords to an Email.
	// For compatibility with IMAP, a keyword is a case-insensitive string of 1–255 characters in the ASCII subset
	// %x21–%x7e (excludes control chars and space), and it MUST NOT include any of these characters:
	//
	//    ( ) { ] % * " \
	//
	// Because JSON is case sensitive, servers MUST return keywords in lowercase.
	//
	// The [IMAP and JMAP Keywords] registry as established in [RFC5788] assigns semantic meaning to some other
	// keywords in common use.
	//
	// New keywords may be established here in the future. In particular, note:
	//
	//   - $forwarded: The Email has been forwarded.
	//   - $phishing: The Email is highly likely to be phishing.
	//     Clients SHOULD warn users to take care when viewing this Email and disable links and attachments.
	//   - $junk: The Email is definitely spam.
	//     Clients SHOULD set this flag when users report spam to help train automated spam-detection systems.
	//   - $notjunk: The Email is definitely not spam.
	//     Clients SHOULD set this flag when users indicate an Email is legitimate, to help train automated spam-detection systems.
	//
	// [IMAP and JMAP Keywords]: https://www.iana.org/assignments/imap-jmap-keywords/
	// [RFC5788]: https://www.rfc-editor.org/rfc/rfc5788.html
	Keywords map[string]bool `json:"keywords,omitempty"`

	// The size, in octets, of the raw data for the message [RFC5322]
	// (as referenced by the blobId, i.e., the number of octets in the file the user would download).
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	Size int `json:"size"`

	// The date the Email was received by the message store.
	//
	// This is the internal date in IMAP [RFC3501].
	//
	// [RFC3501]: https://www.rfc-editor.org/rfc/rfc3501.html
	ReceivedAt time.Time `json:"receivedAt,omitempty"`

	// This is a list of all header fields [RFC5322], in the same order they appear in the message.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	Headers []EmailHeader `json:"headers,omitempty"`

	// The value is identical to the value of header:Message-ID:asMessageIds.
	//
	// For messages conforming to [RFC5322] this will be an array with a single entry.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	MessageId []string `json:"messageId,omitempty"`

	// The value is identical to the value of header:In-Reply-To:asMessageIds.
	InReplyTo []string `json:"inReplyTo,omitempty"`

	// The value is identical to the value of header:References:asMessageIds.
	References []string `json:"references,omitempty"`

	// The value is identical to the value of header:Sender:asAddresses.
	Sender []EmailAddress `json:"sender,omitempty"`

	// The value is identical to the value of header:From:asAddresses.
	From []EmailAddress `json:"from,omitempty"`

	// The value is identical to the value of header:To:asAddresses.
	To []EmailAddress `json:"to,omitempty"`

	// The value is identical to the value of header:Cc:asAddresses.
	Cc []EmailAddress `json:"cc,omitempty"`

	// The value is identical to the value of header:Bcc:asAddresses.
	Bcc []EmailAddress `json:"bcc,omitempty"`

	// The value is identical to the value of header:Reply-To:asAddresses.
	ReplyTo []EmailAddress `json:"replyTo,omitempty"`

	// The value is identical to the value of header:Subject:asText.
	Subject string `json:"subject,omitempty"`

	// The value is identical to the value of header:Date:asDate.
	SentAt time.Time `json:"sentAt,omitempty"`

	// This is the full MIME structure of the message body, without recursing into message/rfc822 or message/global parts.
	//
	// Note that EmailBodyParts may have subParts if they are of type multipart/*.
	BodyStructure EmailBodyPart `json:"bodyStructure,omitzero"`

	// This is a map of partId to an EmailBodyValue object for none, some, or all text/* parts.
	//
	// Which parts are included and whether the value is truncated is determined by various arguments to Email/get and Email/parse.
	BodyValues map[string]EmailBodyValue `json:"bodyValues,omitempty"`

	// A list of text/plain, text/html, image/*, audio/*, and/or video/* parts to display (sequentially) as the
	// message body, with a preference for text/plain when alternative versions are available.
	TextBody []EmailBodyPart `json:"textBody,omitempty"`

	// A list of text/plain, text/html, image/*, audio/*, and/or video/* parts to display (sequentially) as the
	// message body, with a preference for text/html when alternative versions are available.
	HtmlBody []EmailBodyPart `json:"htmlBody,omitempty"`

	// A list, traversing depth-first, of all parts in bodyStructure.
	//
	// They must satisfy either of the following conditions:
	//
	//   - not of type multipart/* and not included in textBody or htmlBody
	//   - of type image/*, audio/*, or video/* and not in both textBody and htmlBody
	//
	// None of these parts include subParts, including message/* types.
	//
	// Attached messages may be fetched using the Email/parse method and the blobId.
	//
	// Note that a text/html body part HTML may reference image parts in attachments by using cid:
	// links to reference the Content-Id, as defined in [RFC2392], or by referencing the Content-Location.
	//
	// [RFC2392]: https://www.rfc-editor.org/rfc/rfc2392.html
	Attachments []EmailBodyPart `json:"attachments,omitempty"`

	// This is true if there are one or more parts in the message that a client UI should offer as downloadable.
	//
	// A server SHOULD set hasAttachment to true if the attachments list contains at least one item that
	// does not have Content-Disposition: inline.
	//
	// The server MAY ignore parts in this list that are processed automatically in some way or are referenced
	// as embedded images in one of the text/html parts of the message.
	//
	// The server MAY set hasAttachment based on implementation-defined or site-configurable heuristics.
	HasAttachment bool `json:"hasAttachment,omitempty"`

	// A plaintext fragment of the message body.
	//
	// This is intended to be shown as a preview line when listing messages in the mail store and may be truncated
	// when shown.
	//
	// The server may choose which part of the message to include in the preview; skipping quoted sections and
	// salutations and collapsing white space can result in a more useful preview.
	//
	// This MUST NOT be more than 256 characters in length.
	//
	// As this is derived from the message content by the server, and the algorithm for doing so could change over
	// time, fetching this for an Email a second time MAY return a different result.
	// However, the previous value is not considered incorrect, and the change SHOULD NOT cause the Email object
	// to be considered as changed by the server.
	Preview string `json:"preview,omitempty"`
}

type Address struct {
	// The email address being represented by the object.
	//
	// This is a “Mailbox” as used in the Reverse-path or Forward-path of the MAIL FROM or RCPT TO command in [RFC5321].
	//
	// [RFC5321]: https://datatracker.ietf.org/doc/html/rfc5321
	Email string `json:"email,omitempty"`

	// Any parameters to send with the email address (either mail-parameter or rcpt-parameter as appropriate,
	// as specified in [RFC5321]).
	//
	// If supplied, each key in the object is a parameter name, and the value is either the parameter value (type String)
	// or null if the parameter does not take a value.
	//
	// [RFC5321]: https://datatracker.ietf.org/doc/html/rfc5321
	Parameters map[string]any `json:"parameters,omitempty"` // TODO RFC5321
}

// Information for use when sending via SMTP.
type Envelope struct {
	// The email address to use as the return address in the SMTP submission,
	// plus any parameters to pass with the MAIL FROM address.
	MailFrom Address `json:"mailFrom"`

	// The email addresses to send the message to, and any RCPT TO parameters to pass with the recipient.
	RcptTo []Address `json:"rcptTo"`
}

type EmailSubmissionUndoStatus string

const (
	// It may be possible to cancel this submission.
	UndoStatusPending EmailSubmissionUndoStatus = "pending"

	// The message has been relayed to at least one recipient in a manner that cannot be recalled.
	// It is no longer possible to cancel this submission.
	UndoStatusFinal EmailSubmissionUndoStatus = "final"

	// The submission was canceled and will not be delivered to any recipient.
	UndoStatusCanceled EmailSubmissionUndoStatus = "canceled"
)

type DeliveryStatusDelivered string

const (
	// The message is in a local mail queue and status will change once it exits the local mail
	// queues.
	// The smtpReply property may still change.
	DeliveredQueued DeliveryStatusDelivered = "queued"

	// The message was successfully delivered to the mail store of the recipient.
	// The smtpReply property is final.
	DeliveredYes DeliveryStatusDelivered = "yes"

	// Delivery to the recipient permanently failed.
	// The smtpReply property is final.
	DeliveredNo DeliveryStatusDelivered = "no"

	// The final delivery status is unknown, (e.g., it was relayed to an external machine
	// and no further information is available).
	//
	// The smtpReply property may still change if a DSN arrives.
	DeliveredUnknown DeliveryStatusDelivered = "unknown"
)

type DeliveryStatusDisplayed string

const (
	// The display status is unknown.
	//
	// This is the initial value.
	DisplayedUnknown DeliveryStatusDisplayed = "unknown"

	// The recipient’s system claims the message content has been displayed to the recipient.
	//
	// Note that there is no guarantee that the recipient has noticed, read, or understood the content.
	DisplayedYes DeliveryStatusDisplayed = "yes"
)

type DeliveryStatus struct {
	// The SMTP reply string returned for this recipient when the server last tried to
	// relay the message, or in a later Delivery Status Notification (DSN, as defined in
	// [RFC3464]) response for the message.
	//
	// This SHOULD be the response to the RCPT TO stage, unless this was accepted and the
	// message as a whole was rejected at the end of the DATA stage, in which case the
	// DATA stage reply SHOULD be used instead.
	//
	// [RFC3464]: https://datatracker.ietf.org/doc/html/rfc3464
	SmtpReply string `json:"smtpReply"`

	// Represents whether the message has been successfully delivered to the recipient.
	//
	// This MUST be one of the following values:
	//   - queued: The message is in a local mail queue and status will change once it exits
	//     the local mail queues. The smtpReply property may still change.
	//   - yes: The message was successfully delivered to the mail store of the recipient.
	//     The smtpReply property is final.
	//   - no: Delivery to the recipient permanently failed. The smtpReply property is final.
	//   - unknown: The final delivery status is unknown, (e.g., it was relayed to an external
	//     machine and no further information is available).
	//     The smtpReply property may still change if a DSN arrives.
	Delivered DeliveryStatusDelivered `json:"delivered"`

	// Represents whether the message has been displayed to the recipient.
	//
	// This MUST be one of the following values:
	//   - unknown: The display status is unknown. This is the initial value.
	//   - yes: The recipient’s system claims the message content has been displayed to the recipient.
	//     Note that there is no guarantee that the recipient has noticed, read, or understood the content.
	Displayed DeliveryStatusDisplayed `json:"displayed"`
}

type EmailSubmission struct {
	// The id of the EmailSubmission (server-set).
	Id string `json:"id"`

	// The id of the Identity to associate with this submission.
	IdentityId string `json:"identityId"`

	// The id of the Email to send.
	//
	// The Email being sent does not have to be a draft, for example, when “redirecting” an existing Email
	// to a different address.
	EmailId string `json:"emailId"`

	// The Thread id of the Email to send (server-set).
	//
	// This is set by the server to the threadId property of the Email referenced by the emailId.
	ThreadId string `json:"threadId"`

	// Information for use when sending via SMTP.
	//
	// If the envelope property is null or omitted on creation, the server MUST generate this from the
	// referenced Email as follows:
	//
	//   - mailFrom: The email address in the Sender header field, if present; otherwise,
	//     it’s the email address in the From header field, if present.
	//     In either case, no parameters are added.
	//   - rcptTo: The deduplicated set of email addresses from the To, Cc, and Bcc header fields,
	//     if present, with no parameters for any of them.
	Envelope *Envelope `json:"envelope,omitempty"`

	// The date the submission was/will be released for delivery (server-set).
	SendAt time.Time `json:"sendAt,omitzero"`

	// This represents whether the submission may be canceled (server-set).
	//
	// This is server set on create and MUST be one of the following values:
	//
	//   - pending: It may be possible to cancel this submission.
	//   - final: The message has been relayed to at least one recipient in a manner that cannot be
	//     recalled. It is no longer possible to cancel this submission.
	//   - canceled: The submission was canceled and will not be delivered to any recipient.
	UndoStatus EmailSubmissionUndoStatus `json:"undoStatus"`

	// This represents the delivery status for each of the submission’s recipients, if known (server-set).
	//
	// This property MAY not be supported by all servers, in which case it will remain null.
	//
	// Servers that support it SHOULD update the EmailSubmission object each time the status of any of
	// the recipients changes, even if some recipients are still being retried.
	//
	// This value is a map from the email address of each recipient to a DeliveryStatus object.
	DeliveryStatus map[string]DeliveryStatus `json:"deliveryStatus"`

	// A list of blob ids for DSNs [RFC3464] received for this submission,
	// in order of receipt, oldest first (server-set) .
	//
	// The blob is the whole MIME message (with a top-level content-type of multipart/report), as received.
	//
	// [RFC3464]: https://datatracker.ietf.org/doc/html/rfc3464
	DsnBlobIds []string `json:"dsnBlobIds,omitempty"`

	// A list of blob ids for MDNs [RFC8098] received for this submission,
	// in order of receipt, oldest first (server-set).
	//
	// The blob is the whole MIME message (with a top-level content-type of multipart/report), as received.
	//
	// [RFC8098]: https://datatracker.ietf.org/doc/html/rfc8098
	MdnBlobIds []string `json:"mdnBlobIds,omitempty"`
}

type EmailSubmissionGetRefCommand struct {
	// The id of the account to use.
	AccountId string `json:"accountId"`

	// The ids of the EmailSubmission objects to return.
	//
	// If null, then all records of the data type are returned, if this is supported for that data
	// type and the number of records does not exceed the maxObjectsInGet limit.
	IdRef *ResultReference `json:"#ids,omitempty"`

	// If supplied, only the properties listed in the array are returned for each EmailSubmission object.
	//
	// If null, all properties of the object are returned. The id property of the object is always returned,
	// even if not explicitly requested. If an invalid property is requested, the call MUST be rejected
	// with an invalidArguments error.
	Properties []string `json:"properties,omitempty"`
}

type EmailSubmissionGetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// A (preferably short) string representing the state on the server for all the data
	// of this type in the account (not just the objects returned in this call).
	//
	// If the data changes, this string MUST change. If the EmailSubmission data is unchanged,
	// servers SHOULD return the same state string on subsequent requests for this data type.
	//
	// When a client receives a response with a different state string to a previous call,
	// it MUST either throw away all currently cached objects for the type or call
	// EmailSubmission/changes to get the exact changes.
	State State `json:"state"`

	// An array of the EmailSubmission objects requested.
	//
	// This is the empty array if no objects were found or if the ids argument passed in
	// was also an empty array.
	//
	// The results MAY be in a different order to the ids in the request arguments.
	// If an identical id is included more than once in the request, the server MUST only
	// include it once in either the list or the notFound argument of the response.
	List []EmailSubmission `json:"list,omitempty"`

	// This array contains the ids passed to the method for records that do not exist.
	//
	// The array is empty if all requested ids were found or if the ids argument passed in was
	// either null or an empty array.
	NotFound []string `json:"notFound,omitempty"`
}

// Patch Object.
//
// Example:
//
//   - moves it from the drafts folder (which has Mailbox id “7cb4e8ee-df87-4757-b9c4-2ea1ca41b38e”)
//     to the sent folder (which we presume has Mailbox id “73dbcb4b-bffc-48bd-8c2a-a2e91ca672f6”)
//
//   - removes the $draft flag and
//
//     {
//     "mailboxIds/7cb4e8ee-df87-4757-b9c4-2ea1ca41b38e": null,
//     "mailboxIds/73dbcb4b-bffc-48bd-8c2a-a2e91ca672f6": true,
//     "keywords/$draft": null
//     }
type PatchObject map[string]any

// same as EmailSubmission but without the server-set attributes
type EmailSubmissionCreate struct {
	// The id of the Identity to associate with this submission.
	IdentityId string `json:"identityId"`

	// The id of the Email to send.
	//
	// The Email being sent does not have to be a draft, for example, when “redirecting” an existing
	// Email to a different address.
	EmailId string `json:"emailId"`

	// Information for use when sending via SMTP.
	Envelope *Envelope `json:"envelope,omitempty"`
}

type EmailSubmissionSetCommand struct {
	AccountId string                           `json:"accountId"`
	Create    map[string]EmailSubmissionCreate `json:"create,omitempty"`
	OldState  State                            `json:"oldState,omitempty"`
	NewState  State                            `json:"newState,omitempty"`

	// A map of EmailSubmission id to an object containing properties to update on the Email object
	// referenced by the EmailSubmission if the create/update/destroy succeeds.
	//
	// (For references to EmailSubmissions created in the same “/set” invocation, this is equivalent
	// to a creation-reference, so the id will be the creation id prefixed with a #.)
	OnSuccessUpdateEmail map[string]PatchObject `json:"onSuccessUpdateEmail,omitempty"`

	// A list of EmailSubmission ids for which the Email with the corresponding emailId should be destroyed
	// if the create/update/destroy succeeds.
	//
	// (For references to EmailSubmission creations, this is equivalent to a creation-reference so the
	// id will be the creation id prefixed with a #.)
	OnSuccessDestroyEmail []string `json:"onSuccessDestroyEmail,omitempty"`
}

type CreatedEmailSubmission struct {
	Id string `json:"id"`
}

type EmailSubmissionSetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// This is the sinceState argument echoed back; it’s the state from which the server is returning changes.
	OldState State `json:"oldState"`

	// This is the state the client will be in after applying the set of changes to the old state.
	NewState State `json:"newState"`

	// If true, the client may call EmailSubmission/changes again with the newState returned to get further
	// updates.
	//
	// If false, newState is the current server state.
	HasMoreChanges bool `json:"hasMoreChanges"`

	// An array of ids for records that have been created since the old state.
	Created map[string]CreatedEmailSubmission `json:"created,omitempty"`

	// A map of the creation id to a SetError object for each record that failed to be created, or
	// null if all successful.
	NotCreated map[string]SetError `json:"notCreated,omitempty"`

	// TODO(pbleser-oc) add updated and destroyed when they are needed
}

type Command string

type Invocation struct {
	Command    Command
	Parameters any
	Tag        string
}

func invocation(command Command, parameters any, tag string) Invocation {
	return Invocation{
		Command:    command,
		Parameters: parameters,
		Tag:        tag,
	}
}

type Request struct {
	// The set of capabilities the client wishes to use.
	//
	// The client MAY include capability identifiers even if the method calls it makes do not utilise those capabilities.
	// The server advertises the set of specifications it supports in the Session object (see [Section 2]), as keys on
	// the capabilities property.
	//
	// [Section 2]: https://jmap.io/spec-core.html#the-jmap-session-resource
	Using []string `json:"using"`

	// An array of method calls to process on the server.
	//
	// The method calls MUST be processed sequentially, in order.
	MethodCalls []Invocation `json:"methodCalls"`

	// A map of a (client-specified) creation id to the id the server assigned when a record was successfully created (optional).
	CreatedIds map[string]string `json:"createdIds,omitempty"`
}

type Response struct {
	// An array of responses, in the same format as the methodCalls on the Request object.
	// The output of the methods MUST be added to the methodResponses array in the same order that the methods are processed.
	MethodResponses []Invocation `json:"methodResponses"`

	// A map of a (client-specified) creation id to the id the server assigned when a record was successfully created.
	//
	// Optional; only returned if given in the request.
	//
	// This MUST include all creation ids passed in the original createdIds parameter of the Request object, as well as any
	// additional ones added for newly created records.
	CreatedIds map[string]string `json:"createdIds,omitempty"`

	// The current value of the “state” string on the Session object, as described in [Section 2].
	// Clients may use this to detect if this object has changed and needs to be refetched.
	//
	// [Section 2]: https://jmap.io/spec-core.html#the-jmap-session-resource
	SessionState SessionState `json:"sessionState"`
}

type EmailQueryResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// A string encoding the current state of the query on the server.
	//
	// This string MUST change if the results of the query (i.e., the matching ids and their sort order) have changed.
	// The queryState string MAY change if something has changed on the server, which means the results may have changed
	// but the server doesn’t know for sure.
	//
	// The queryState string only represents the ordered list of ids that match the particular query (including its sort/filter).
	// There is no requirement for it to change if a property on an object matching the query changes but the query results are unaffected
	// (indeed, it is more efficient if the queryState string does not change in this case).
	//
	// The queryState string only has meaning when compared to future responses to a query with the same type/sort/filter or when used with
	// /queryChanges to fetch changes.
	//
	// Should a client receive back a response with a different queryState string to a previous call, it MUST either throw away the currently
	// cached query and fetch it again (note, this does not require fetching the records again, just the list of ids) or call
	// Email/queryChanges to get the difference.
	QueryState State `json:"queryState"`

	// This is true if the server supports calling Email/queryChanges with these filter/sort parameters.
	//
	// Note, this does not guarantee that the Email/queryChanges call will succeed, as it may only be possible for a limited time
	// afterwards due to server internal implementation details.
	CanCalculateChanges bool `json:"canCalculateChanges"`

	// The zero-based index of the first result in the ids array within the complete list of query results.
	Position uint `json:"position"`

	// The list of ids for each Email in the query results, starting at the index given by the position argument of this
	// response and continuing until it hits the end of the results or reaches the limit number of ids.
	//
	// If position is >= total, this MUST be the empty list.
	Ids []string `json:"ids"`

	// The total number of Emails in the results (given the filter).
	//
	// Only if requested.
	//
	// This argument MUST be omitted if the calculateTotal request argument is not true.
	Total uint `json:"total,omitempty,omitzero"`

	// The limit enforced by the server on the maximum number of results to return (if set by the server).
	//
	// This is only returned if the server set a limit or used a different limit than that given in the request.
	Limit uint `json:"limit,omitempty,omitzero"`
}

type EmailGetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// A (preferably short) string representing the state on the server for all the data of this type
	// in the account (not just the objects returned in this call).
	//
	// If the data changes, this string MUST change.
	// If the Email data is unchanged, servers SHOULD return the same state string on subsequent requests for this data type.
	State State `json:"state"`

	// An array of the Email objects requested.
	//
	// This is the empty array if no objects were found or if the ids argument passed in was also an empty array.
	//
	// The results MAY be in a different order to the ids in the request arguments.
	//
	// If an identical id is included more than once in the request, the server MUST only include it once in either
	// the list or the notFound argument of the response.
	List []Email `json:"list"`

	// This array contains the ids passed to the method for records that do not exist.
	//
	// The array is empty if all requested ids were found or if the ids argument passed in was either null or an empty array.
	NotFound []any `json:"notFound"`
}

type EmailChangesResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// This is the sinceState argument echoed back; it’s the state from which the server is returning changes.
	OldState State `json:"oldState"`

	// This is the state the client will be in after applying the set of changes to the old state.
	NewState State `json:"newState"`

	// If true, the client may call Email/changes again with the newState returned to get further updates.
	// If false, newState is the current server state.
	HasMoreChanges bool `json:"hasMoreChanges"`

	// An array of ids for records that have been created since the old state.
	Created []string `json:"created,omitempty"`

	// An array of ids for records that have been updated since the old state.
	Updated []string `json:"updated,omitempty"`

	// An array of ids for records that have been destroyed since the old state.
	Destroyed []string `json:"destroyed,omitempty"`
}

type MailboxGetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// A (preferably short) string representing the state on the server for all the data of this type in the account
	// (not just the objects returned in this call).
	// If the data changes, this string MUST change.
	// If the Mailbox data is unchanged, servers SHOULD return the same state string on subsequent requests for this data type.
	// When a client receives a response with a different state string to a previous call, it MUST either throw away all currently
	// cached objects for the type or call Foo/changes to get the exact changes.
	State State `json:"state"`

	// An array of the Mailbox objects requested.
	// This is the empty array if no objects were found or if the ids argument passed in was also an empty array.
	// The results MAY be in a different order to the ids in the request arguments.
	// If an identical id is included more than once in the request, the server MUST only include it once in either
	// the list or the notFound argument of the response.
	List []Mailbox `json:"list"`

	// This array contains the ids passed to the method for records that do not exist.
	// The array is empty if all requested ids were found or if the ids argument passed in was either null or an empty array.
	NotFound []any `json:"notFound"`
}

type MailboxChangesResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// This is the sinceState argument echoed back; it’s the state from which the server is returning changes.
	OldState State `json:"oldState"`

	// This is the state the client will be in after applying the set of changes to the old state.
	NewState State `json:"newState"`

	// If true, the client may call Mailbox/changes again with the newState returned to get further updates.
	//
	// If false, newState is the current server state.
	HasMoreChanges bool `json:"hasMoreChanges"`

	// An array of ids for records that have been created since the old state.
	Created []string `json:"created,omitempty"`

	// An array of ids for records that have been updated since the old state.
	Updated []string `json:"updated,omitempty"`

	// An array of ids for records that have been destroyed since the old state.
	Destroyed []string `json:"destroyed,omitempty"`

	// If only the “totalEmails”, “unreadEmails”, “totalThreads”, and/or “unreadThreads” Mailbox properties have
	// changed since the old state, this will be the list of properties that may have changed.
	//
	// If the server is unable to tell if only counts have changed, it MUST just be null.
	UpdatedProperties []string `json:"updatedProperties,omitempty"`
}

type MailboxQueryResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// A string encoding the current state of the query on the server.
	//
	// This string MUST change if the results of the query (i.e., the matching ids and their sort order) have changed.
	// The queryState string MAY change if something has changed on the server, which means the results may have
	// changed but the server doesn’t know for sure.
	//
	// The queryState string only represents the ordered list of ids that match the particular query (including its
	// sort/filter). There is no requirement for it to change if a property on an object matching the query changes
	// but the query results are unaffected (indeed, it is more efficient if the queryState string does not change
	// in this case). The queryState string only has meaning when compared to future responses to a query with the
	// same type/sort/filter or when used with /queryChanges to fetch changes.
	//
	// Should a client receive back a response with a different queryState string to a previous call, it MUST either
	// throw away the currently cached query and fetch it again (note, this does not require fetching the records
	// again, just the list of ids) or call Mailbox/queryChanges to get the difference.
	QueryState State `json:"queryState"`

	// This is true if the server supports calling Mailbox/queryChanges with these filter/sort parameters.
	//
	// Note, this does not guarantee that the Mailbox/queryChanges call will succeed, as it may only be possible for
	// a limited time afterwards due to server internal implementation details.
	CanCalculateChanges bool `json:"canCalculateChanges"`

	// The zero-based index of the first result in the ids array within the complete list of query results.
	Position int `json:"position"`

	// The list of ids for each Mailbox in the query results, starting at the index given by the position argument
	// of this response and continuing until it hits the end of the results or reaches the limit number of ids.
	//
	// If position is >= total, this MUST be the empty list.
	Ids []string `json:"ids"`

	// The total number of Mailbox in the results (given the filter) (only if requested).
	//
	// This argument MUST be omitted if the calculateTotal request argument is not true.
	Total int `json:"total,omitzero"`

	// The limit enforced by the server on the maximum number of results to return (if set by the server).
	//
	// This is only returned if the server set a limit or used a different limit than that given in the request.
	Limit int `json:"limit,omitzero"`
}

type EmailBodyStructure struct {
	Type   string         `json:"type"`
	PartId string         `json:"partId"`
	Other  map[string]any `mapstructure:",remain"`
}

type EmailCreate struct {
	// The set of Mailbox ids this Email belongs to.
	//
	// An Email in the mail store MUST belong to one or more Mailboxes at all times
	// (until it is destroyed).
	//
	// The set is represented as an object, with each key being a Mailbox id.
	// The value for each key in the object MUST be true.
	MailboxIds map[string]bool `json:"mailboxIds,omitempty"`

	// A set of keywords that apply to the Email.
	//
	// The set is represented as an object, with the keys being the keywords.
	// The value for each key in the object MUST be true.
	Keywords map[string]bool `json:"keywords,omitempty"`

	// The ["From:" field] specifies the author(s) of the message, that is, the mailbox(es)
	// of the person(s) or system(s) responsible for the writing of the message
	//
	// ["From:" field]: https://www.rfc-editor.org/rfc/rfc5322.html#section-3.6.2
	From []EmailAddress `json:"from,omitempty"`

	// The "Subject:" field contains a short string identifying the topic of the message.
	Subject string `json:"subject,omitempty"`

	// The date the Email was received by the message store.
	//
	// (default: time of most recent Received header, or time of import on server if none).
	ReceivedAt time.Time `json:"receivedAt,omitzero"`

	// The origination date specifies the date and time at which the creator of the message indicated that
	// the message was complete and ready to enter the mail delivery system.
	//
	// For instance, this might be the time that a user pushes the "send" or "submit" button in an
	// application program.
	//
	// In any case, it is specifically not intended to convey the time that the message is actually transported,
	// but rather the time at which the human or other creator of the message has put the message into its final
	// form, ready for transport.
	//
	// (For example, a portable computer user who is not connected to a network might queue a message for delivery.
	// The origination date is intended to contain the date and time that the user queued the message, not the time
	// when the user connected to the network to send the message.)
	SentAt time.Time `json:"sentAt,omitzero"`

	// This is the full MIME structure of the message body, without recursing into message/rfc822 or message/global parts.
	//
	// Note that EmailBodyParts may have subParts if they are of type multipart/*.
	BodyStructure EmailBodyStructure `json:"bodyStructure"`

	// This is a map of partId to an EmailBodyValue object for none, some, or all text/* parts.
	BodyValues map[string]EmailBodyValue `json:"bodyValues,omitempty"`
}

type EmailUpdate map[string]any

type EmailSetCommand struct {
	AccountId string                 `json:"accountId"`
	Create    map[string]EmailCreate `json:"create,omitempty"`
	Update    map[string]EmailUpdate `json:"update,omitempty"`
	Destroy   []string               `json:"destroy,omitempty"`
}

type EmailSetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// The state string that would have been returned by Email/get before making the
	// requested changes, or null if the server doesn’t know what the previous state
	// string was.
	OldState State `json:"oldState,omitempty"`

	// The state string that will now be returned by Email/get.
	NewState State `json:"newState"`

	// A map of the creation id to an object containing any properties of the created Email object
	// that were not sent by the client.
	//
	// This includes all server-set properties (such as the id in most object types) and any properties
	// that were omitted by the client and thus set to a default by the server.
	//
	// This argument is null if no Email objects were successfully created.
	Created map[string]Email `json:"created,omitempty"`

	// The keys in this map are the ids of all Emails that were successfully updated.
	//
	// The value for each id is an Email object containing any property that changed in a way not
	// explicitly requested by the PatchObject sent to the server, or null if none.
	//
	// This lets the client know of any changes to server-set or computed properties.
	//
	// This argument is null if no Email objects were successfully updated.
	Updated map[string]Email `json:"updated,omitempty"`

	// A list of Email ids for records that were successfully destroyed, or null if none.
	Destroyed []string `json:"destroyed,omitempty"`

	// A map of the creation id to a SetError object for each record that failed to be created,
	// or null if all successful.
	NotCreated map[string]SetError `json:"notCreated,omitempty"`

	// A map of the Email id to a SetError object for each record that failed to be updated,
	// or null if all successful.
	NotUpdated map[string]SetError `json:"notUpdated,omitempty"`

	// A map of the Email id to a SetError object for each record that failed to be destroyed,
	// or null if all successful.
	NotDestroyed map[string]SetError `json:"notDestroyed,omitempty"`
}

const (
	EmailMimeType = "message/rfc822"
)

type EmailImport struct {
	// The id of the blob containing the raw message [RFC5322].
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	BlobId string `json:"blobId"`

	// The ids of the Mailboxes to assign this Email to.
	//
	// At least one Mailbox MUST be given.
	MailboxIds map[string]bool `json:"mailboxIds"`

	// The keywords to apply to the Email.
	Keywords map[string]bool `json:"keywords"`

	// (default: time of most recent Received header, or time of import
	// on server if none) The receivedAt date to set on the Email.
	ReceivedAt time.Time `json:"receivedAt"`
}

type EmailImportCommand struct {
	AccountId string `json:"accountId"`

	// This is a state string as returned by the Email/get method.
	//
	// If supplied, the string must match the current state of the account referenced
	// by the accountId; otherwise, the method will be aborted and a stateMismatch
	// error returned.
	//
	// If null, any changes will be applied to the current state.
	IfInState string `json:"ifInState,omitempty"`

	// A map of creation id (client specified) to EmailImport objects.
	Emails map[string]EmailImport `json:"emails"`
}

// Successfully imported Email.
type ImportedEmail struct {
	// Id of the successfully imported Email.
	Id string `json:"id"`

	// Blob id of the successfully imported Email.
	BlobId string `json:"blobId"`

	// Thread id of the successfully imported Email.
	ThreadId string `json:"threadId"`

	// Size of the successfully imported Email.
	Size int `json:"size"`
}

type EmailImportResponse struct {
	// The id of the account used for this call.
	AccountId string `json:"accountId"`

	// The state string that would have been returned by Email/get on this account
	// before making the requested changes, or null if the server doesn’t know
	// what the previous state string was.
	OldState State `json:"oldState"`

	// The state string that will now be returned by Email/get on this account.
	NewState State `json:"newState"`

	// A map of the creation id to an object containing the id, blobId, threadId,
	// and size properties for each successfully imported Email, or null if none.
	Created map[string]ImportedEmail `json:"created"`

	// A map of the creation id to a SetError object for each Email that failed to
	// be created, or null if all successful.
	NotCreated map[string]SetError `json:"notCreated"`
}

// Replies are grouped together with the original message to form a Thread.
//
// In JMAP, a Thread is simply a flat list of Emails, ordered by date.
//
// Every Email MUST belong to a Thread, even if it is the only Email in the Thread.
type Thread struct {
	// The id of the Thread.
	Id string

	// The ids of the Emails in the Thread, sorted by the receivedAt date of the Email,
	// oldest first.
	//
	// If two Emails have an identical date, the sort is server dependent but MUST be
	// stable (sorting by id is recommended).
	EmailIds []string
}

type ThreadGetCommand struct {
	AccountId string   `json:"accountId"`
	Ids       []string `json:"ids,omitempty"`
}

type ThreadGetResponse struct {
	AccountId string
	State     State
	List      []Thread
	NotFound  []any
}

type IdentityGetCommand struct {
	AccountId string   `json:"accountId"`
	Ids       []string `json:"ids,omitempty"`
}

type Identity struct {
	// The id of the Identity.
	Id string `json:"id"`

	// The “From” name the client SHOULD use when creating a new Email from this Identity.
	Name string `json:"name,omitempty"`

	// The “From” email address the client MUST use when creating a new Email from this Identity.
	//
	// If the mailbox part of the address (the section before the “@”) is the single character
	// * (e.g., *@example.com) then the client may use any valid address ending in that domain
	// (e.g., foo@example.com).
	Email string `json:"email,omitempty"`

	// The Reply-To value the client SHOULD set when creating a new Email from this Identity.
	ReplyTo string `json:"replyTo,omitempty"`

	// The Bcc value the client SHOULD set when creating a new Email from this Identity.
	Bcc []EmailAddress `json:"bcc,omitempty"`

	// A signature the client SHOULD insert into new plaintext messages that will be sent from
	// this Identity.
	//
	// Clients MAY ignore this and/or combine this with a client-specific signature preference.
	TextSignature string `json:"textSignature,omitempty"`

	// A signature the client SHOULD insert into new HTML messages that will be sent from this
	// Identity.
	//
	// This text MUST be an HTML snippet to be inserted into the <body></body> section of the HTML.
	//
	// Clients MAY ignore this and/or combine this with a client-specific signature preference.
	HtmlSignature string `json:"htmlSignature,omitempty"`

	// Is the user allowed to delete this Identity?
	//
	// Servers may wish to set this to false for the user’s username or other default address.
	//
	// Attempts to destroy an Identity with mayDelete: false will be rejected with a standard
	// forbidden SetError.
	MayDelete bool `json:"mayDelete"`
}

type IdentityGetResponse struct {
	AccountId string     `json:"accountId"`
	State     State      `json:"state"`
	List      []Identity `json:"list,omitempty"`
	NotFound  []string   `json:"notFound,omitempty"`
}

type VacationResponseGetCommand struct {
	AccountId string `json:"accountId"`
}

// Vacation Response
//
// A vacation response sends an automatic reply when a message is delivered to the mail store,
// informing the original sender that their message may not be read for some time.
//
// Automated message sending can produce undesirable behaviour.
// To avoid this, implementors MUST follow the recommendations set forth in [RFC3834].
//
// The VacationResponse object represents the state of vacation-response-related settings for an account.
//
// [RFC3834]: https://www.rfc-editor.org/rfc/rfc3834.html
type VacationResponse struct {
	// The id of the object.
	// There is only ever one VacationResponse object, and its id is "singleton"
	Id string `json:"id,omitempty"`

	// Should a vacation response be sent if a message arrives between the "fromDate" and "toDate"?
	IsEnabled bool `json:"isEnabled"`

	// If "isEnabled" is true, messages that arrive on or after this date-time (but before the "toDate" if defined) should receive the
	// user's vacation response. If null, the vacation response is effective immediately.
	FromDate time.Time `json:"fromDate,omitzero"`

	// If "isEnabled" is true, messages that arrive before this date-time but on or after the "fromDate" if defined) should receive the
	// user's vacation response.  If null, the vacation response is effective indefinitely.
	ToDate time.Time `json:"toDate,omitzero"`

	// The subject that will be used by the message sent in response to messages when the vacation response is enabled.
	// If null, an appropriate subject SHOULD be set by the server.
	Subject string `json:"subject,omitempty"`

	// The plaintext body to send in response to messages when the vacation response is enabled.
	// If this is null, the server SHOULD generate a plaintext body part from the "htmlBody" when sending vacation responses
	// but MAY choose to send the response as HTML only.  If both "textBody" and "htmlBody" are null, an appropriate default
	// body SHOULD be generated for responses by the server.
	TextBody string `json:"textBody,omitempty"`

	// The HTML body to send in response to messages when the vacation response is enabled.
	// If this is null, the server MAY choose to generate an HTML body part from the "textBody" when sending vacation responses
	// or MAY choose to send the response as plaintext only.
	HtmlBody string `json:"htmlBody,omitempty"`
}

type VacationResponseGetResponse struct {
	// The identifier of the account this response pertains to.
	AccountId string `json:"accountId"`

	// A string representing the state on the server for all the data of this type in the account
	// (not just the objects returned in this call).
	//
	// If the data changes, this string MUST change. If the data is unchanged, servers SHOULD return the same state string
	// on subsequent requests for this data type.
	State State `json:"state,omitempty"`

	// An array of VacationResponse objects.
	List []VacationResponse `json:"list,omitempty"`

	// Contains identifiers of requested objects that were not found.
	NotFound []any `json:"notFound,omitempty"`
}

type VacationResponseSetCommand struct {
	AccountId string                      `json:"accountId"`
	IfInState string                      `json:"ifInState,omitempty"`
	Create    map[string]VacationResponse `json:"create,omitempty"`
	Update    map[string]PatchObject      `json:"update,omitempty"`
	Destroy   []string                    `json:"destroy,omitempty"`
}

type VacationResponseSetResponse struct {
	AccountId    string                      `json:"accountId"`
	OldState     State                       `json:"oldState,omitempty"`
	NewState     State                       `json:"newState,omitempty"`
	Created      map[string]VacationResponse `json:"created,omitempty"`
	Updated      map[string]VacationResponse `json:"updated,omitempty"`
	Destroyed    []string                    `json:"destroyed,omitempty"`
	NotCreated   map[string]SetError         `json:"notCreated,omitempty"`
	NotUpdated   map[string]SetError         `json:"notUpdated,omitempty"`
	NotDestroyed map[string]SetError         `json:"notDestroyed,omitempty"`
}

// One of these attributes must be set, but not both.
type DataSourceObject struct {
	DataAsText   string `json:"data:asText,omitempty"`
	DataAsBase64 string `json:"data:asBase64,omitempty"`
}

type UploadObject struct {
	Data []DataSourceObject `json:"data"`
	Type string             `json:"type,omitempty"`
}

type BlobUploadCommand struct {
	AccountId string                  `json:"accountId"`
	Create    map[string]UploadObject `json:"create"`
}

type BlobUploadCreateResult struct {
	Id   string `json:"id"`
	Type string `json:"type,omitempty"`
	Size int    `json:"size"`
}

type BlobUploadResponse struct {
	AccountId string                            `json:"accountId"`
	Created   map[string]BlobUploadCreateResult `json:"created"`
}

const (
	BlobPropertyDataAsText   = "data:asText"
	BlobPropertyDataAsBase64 = "data:asBase64"
	// Returns data:asText if the selected octets are valid UTF-8 or data:asBase64.
	BlobPropertyData = "data"
	BlobPropertySize = "size"
	// https://www.iana.org/assignments/http-digest-hash-alg/http-digest-hash-alg.xhtml
	BlobPropertyDigestSha256 = "digest:sha256"
	// https://www.iana.org/assignments/http-digest-hash-alg/http-digest-hash-alg.xhtml
	BlobPropertyDigestSha512 = "digest:sha512"
)

type BlobGetCommand struct {
	AccountId  string   `json:"accountId"`
	Ids        []string `json:"ids,omitempty"`
	Properties []string `json:"properties,omitempty"`
	Offset     int      `json:"offset,omitzero"`
	Length     int      `json:"length,omitzero"`
}

type BlobGetRefCommand struct {
	AccountId  string           `json:"accountId"`
	IdRef      *ResultReference `json:"#ids,omitempty"`
	Properties []string         `json:"properties,omitempty"`
	Offset     int              `json:"offset,omitzero"`
	Length     int              `json:"length,omitzero"`
}

type Blob struct {
	// The unique identifier of the blob.
	Id string `json:"id"`

	// (raw octets, must be UTF-8)
	DataAsText string `json:"data:asText,omitempty"`

	// (base64 representation of octets)
	DataAsBase64 string `json:"data:asBase64,omitempty"`

	// The base64 encoding of the digest of the octets in the selected range,
	// calculated using the SHA-256 algorithm.
	DigestSha256 string `json:"digest:sha256,omitempty"`

	// The base64 encoding of the digest of the octets in the selected range,
	// calculated using the SHA-512 algorithm.
	DigestSha512 string `json:"digest:sha512,omitempty"`

	// If an encoding problem occured.
	//
	// The data fields contain a representation of the octets within the selected range
	// that are present in the blob.
	//
	// If the octets selected are not valid UTF-8 (including truncating in the middle of a
	// multi-octet sequence) and data or data:asText was requested, then the key isEncodingProblem
	// MUST be set to true, and the data:asText response value MUST be null.
	//
	// In the case where data was requested and the data is not valid UTF-8, then data:asBase64
	// MUST be returned.
	IsEncodingProblem bool `json:"isEncodingProblem,omitzero"`

	// When requesting a range: the isTruncated property in the result MUST be
	// set to true to tell the client that the requested range could not be fully satisfied.
	IsTruncated bool `json:"isTruncated,omitzero"`

	// The number of octets in the entire blob.
	Size int `json:"size"`
}

// Picks the best digest if available, or ""
func (b *Blob) Digest() string {
	if b.DigestSha512 != "" {
		return b.DigestSha512
	} else if b.DigestSha256 != "" {
		return b.DigestSha256
	} else {
		return ""
	}
}

type BlobGetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`

	// A string representing the state on the server for all the data of this type in the
	// account (not just the objects returned in this call).
	//
	// If the data changes, this string MUST change. If the Blob data is unchanged, servers
	// SHOULD return the same state string on subsequent requests for this data type.
	//
	// When a client receives a response with a different state string to a previous call,
	// it MUST either throw away all currently cached objects for the type or call
	// Blob/changes to get the exact changes.
	State State `json:"state,omitempty"`

	// An array of the Blob objects requested.
	//
	// This is the empty array if no objects were found or if the ids argument passed in
	// was also an empty array. The results MAY be in a different order to the ids in the
	// request arguments. If an identical id is included more than once in the request,
	// the server MUST only include it once in either the list or the notFound argument of the response.
	List []Blob `json:"list,omitempty"`

	// This array contains the ids passed to the method for records that do not exist.
	//
	// The array is empty if all requested ids were found or if the ids argument passed
	// in was either null or an empty array.
	NotFound []any `json:"notFound,omitempty"`
}

type BlobDownload struct {
	Body               io.ReadCloser
	Size               int
	Type               string
	ContentDisposition string
	CacheControl       string
}

// When doing a search on a String property, the client may wish to show the relevant
// section of the body that matches the search as a preview and to highlight any
// matching terms in both this and the subject of the Email.
//
// Search snippets represent this data.
//
// What is a relevant section of the body for preview is server defined. If the server is
// unable to determine search snippets, it MUST return null for both the subject and preview
// properties.
//
// Note that unlike most data types, a SearchSnippet DOES NOT have a property called id.
type SearchSnippet struct {
	// The Email id the snippet applies to.
	EmailId string `json:"emailId"`

	// If text from the filter matches the subject, this is the subject of the Email
	// with the following transformations:
	//
	//   1. Any instance of the following three characters MUST be replaced by an
	//      appropriate HTML entity: & (ampersand), < (less-than sign), and > (greater-than sign)
	//      HTML. Other characters MAY also be replaced with an HTML entity form.
	//   2. The matching words/phrases from the filter are wrapped in HTML <mark></mark> tags.
	//
	// If the subject does not match text from the filter, this property is null.
	Subject string `json:"subject,omitempty"`

	// If text from the filter matches the plaintext or HTML body, this is the
	// relevant section of the body (converted to plaintext if originally HTML),
	// with the same transformations as the subject property.
	//
	// It MUST NOT be bigger than 255 octets in size.
	//
	// If the body does not contain a match for the text from the filter, this property is null.
	Preview string `json:"preview,omitempty"`
}

type SearchSnippetGetRefCommand struct {
	// The id of the account to use.
	AccountId string `json:"accountId"`

	// The same filter as passed to Email/query.
	Filter EmailFilterElement `json:"filter,omitempty"`

	// The ids of the Emails to fetch snippets for.
	EmailIdRef *ResultReference `json:"#emailIds,omitempty"`
}

type SearchSnippetGetResponse struct {
	AccountId string          `json:"accountId"`
	List      []SearchSnippet `json:"list,omitempty"`
	NotFound  []string        `json:"notFound,omitempty"`
}

type StateChange struct {
	// This MUST be the string "StateChange".
	Type string `json:"@type"`

	// A map of an "account id" to an object encoding the state of data types that have
	// changed for that account since the last StateChange object was pushed, for each
	// of the accounts to which the user has access and for which something has changed.
	//
	// The value is a map.  The keys are the type name "Foo" e.g., "Mailbox" or "Email"),
	// and the value is the "state" property that would currently be returned by a call to
	// "Foo/get".
	//
	// The client can compare the new state strings with its current values to see whether
	// it has the current data for these types. If not, the changes can then be efficiently
	// fetched in a single standard API request (using the /changes type methods).
	Changed map[string]map[ObjectType]string `json:"changed"`

	// A (preferably short) string that encodes the entire server state visible to the user
	// (not just the objects returned in this call).
	//
	// The purpose of the "pushState" token is to allow a client to immediately get any changes
	// that occurred while it was disconnected. If the server does not support "pushState" tokens,
	// the client will have to issue a series of "/changes" requests upon reconnection to update
	// its state to match that of the server.
	PushState string `json:"pushState"`
}

type AddressBookRights struct {
	// The user may fetch the ContactCards in this AddressBook.
	MayRead bool `json:"mayRead"`

	// The user may create, modify or destroy all ContactCards in this AddressBook, or move them to or from this AddressBook.
	MayWrite bool `json:"mayWrite"`

	// The user may modify the “shareWith” property for this AddressBook.
	MayAdmin bool `json:"mayAdmin"`

	// The user may delete the AddressBook itself.
	MayDelete bool `json:"mayDelete"`
}

// An AddressBook is a named collection of ContactCards.
//
// All ContactCards are associated with one or more AddressBook.
type AddressBook struct {
	// The id of the AddressBook (immutable; server-set).
	Id string `json:"id"`

	// The user-visible name of the AddressBook.
	//
	// This may be any UTF-8 string of at least 1 character in length and maximum 255 octets in size.
	Name string `json:"name"`

	// An optional longer-form description of the AddressBook, to provide context in shared environments
	// where users need more than just the name.
	Description string `json:"description,omitempty"`

	// Defines the sort order of AddressBooks when presented in the client’s UI, so it is consistent between devices.
	//
	// The number MUST be an integer in the range 0 <= sortOrder < 2^31.
	//
	// An AddressBook with a lower order should be displayed before a AddressBook with a higher order in any list
	// of AddressBooks in the client’s UI.
	//
	// AddressBooks with equal order SHOULD be sorted in alphabetical order by name.
	//
	// The sorting should take into account locale-specific character order convention.
	//
	// Default: 0
	SortOrder uint `json:"sortOrder,omitzero"`

	// This SHOULD be true for exactly one AddressBook in any account, and MUST NOT be true for more than one
	// AddressBook within an account.
	//
	// The default AddressBook should be used by clients whenever they need to choose an AddressBook for the user
	// within this account, and they do not have any other information on which to make a choice.
	//
	// For example, if the user creates a new contact card, the client may automatically set the card as belonging
	// to the default AddressBook from the user’s primary account.
	IsDefault bool `json:"isDefault,omitzero"`

	// True if the user has indicated they wish to see this AddressBook in their client.
	//
	// This SHOULD default to false for AddressBooks in shared accounts the user has access to and true for any
	// new AddressBooks created by the user themself.
	//
	// If false, the AddressBook and its contents SHOULD only be displayed when the user explicitly requests it
	// or to offer it for the user to subscribe to.
	IsSubscribed bool `json:"isSubscribed"`

	// A map of Principal id to rights for principals this AddressBook is shared with.
	//
	// The principal to which this AddressBook belongs MUST NOT be in this set.
	//
	// This is null if the AddressBook is not shared with anyone.
	//
	// May be modified only if the user has the mayAdmin right.
	//
	// The account id for the principals may be found in the urn:ietf:params:jmap:principals:owner capability
	// of the Account to which the AddressBook belongs.
	ShareWith map[string]AddressBookRights `json:"shareWith,omitempty"`

	// The set of access rights the user has in relation to this AddressBook (server-set).
	MyRights AddressBookRights `json:"myRights"`
}

type CalendarRights struct {
	// The user may read the free-busy information for this calendar.
	MayReadFreeBusy bool `json:"mayReadFreeBusy"`

	// The user may fetch the events in this calendar.
	MayReadItems bool `json:"mayReadItems"`

	// The user may create, modify or destroy all events in this calendar, or move events
	// to or from this calendar.
	//
	// If this is `true`, the `mayWriteOwn`, `mayUpdatePrivate` and `mayRSVP`
	// properties MUST all also be `true`.
	MayWriteAll bool `json:"mayWriteAll"`

	// The user may create, modify or destroy an event on this calendar if either they are
	// the owner of the event or the event has no owner.
	//
	// This means the user may also transfer ownership by updating an event so they are no longer an owner.
	MayWriteOwn bool `json:"mayWriteOwn"`

	// The user may modify per-user properties on all events in the calendar, even if they would
	// not otherwise have permission to modify that event.
	//
	// These properties MUST all be stored per-user, and changes do not affect any other user of the calendar.
	//
	// The user may also modify these properties on a per-occurrence basis for recurring events
	// (updating the `recurrenceOverrides` property of the event to do so).
	MayUpdatePrivate bool `json:"mayUpdatePrivate"`

	// The user may modify the following properties of any `Participant` object that corresponds
	// to one of the user's `ParticipantIdentity` objects in the account, even if they would not
	// otherwise have permission to modify that event.
	//
	// !- `participationStatus`
	// !- `participationComment`
	// !- `expectReply`
	// !- `scheduleAgent`
	// !- `scheduleSequence`
	// !- `scheduleUpdated`
	//
	// If the event has its `mayInviteSelf` property set to `true`, then the user may also add a
	// new `Participant` to the event with `scheduleId`/`sendTo` properties that are the same as
	// the `scheduleId`/`sendTo` properties of one of the user's `ParticipantIdentity` objects in
	// the account.
	//
	// The `roles` property of the participant MUST only contain `attendee`.
	//
	// If the event has its `mayInviteOthers` property set to `true` and there is an existing
	// `Participant` in the event corresponding to one of the user's `ParticipantIdentity` objects
	// in the account, then the user may also add new participants.
	//
	// The `roles` property of any new participant MUST only contain `attendee`.
	//
	// The user may also do all of the above on a per-occurrence basis for recurring events
	// (updating the recurrenceOverrides property of the event to do so).
	MayRSVP bool `json:"mayRSVP"`

	// The user may modify the `shareWith` property for this calendar.
	MayAdmin bool `json:"mayAdmin"`

	// The user may delete the calendar itself.
	MayDelete bool `json:"mayDelete"`
}

// A Calendar is a named collection of events.
//
// All events are associated with at least one calendar.
//
// The user is an owner for an event if the `CalendarEvent` object has a `participants`
// property, and one of the `Participant` objects both:
// 1. Has the `owner` role.
// 2. Corresponds to one of the user's `ParticipantIdentity` objects in the account.
//
// An event has no owner if its `participants` property is null or omitted, or if none
// of the `Participant` objects have the `owner` role.
type Calendar struct {
	// The id of the calendar (immutable; server-set).
	Id string `json:"id"`

	// The user-visible name of the calendar.
	//
	// This may be any UTF-8 string of at least 1 character in length and maximum 255 octets in size.
	Name string `json:"name"`

	// An optional longer-form description of the calendar, to provide context in shared environments
	// where users need more than just the name.
	Description string `json:"description,omitempty"`

	// A color to be used when displaying events associated with the calendar.
	//
	// If not null, the value MUST be a case-insensitive color name taken from the set of names
	// defined in Section 4.3 of CSS Color Module Level 3 COLORS, or an RGB value in hexadecimal
	// notation, as defined in Section 4.2.1 of CSS Color Module Level 3.
	//
	// The color SHOULD have sufficient contrast to be used as text on a white background.
	Color string `json:"color,omitempty"`

	// Defines the sort order of calendars when presented in the client’s UI, so it is consistent
	// between devices.
	//
	// The number MUST be an integer in the range 0 <= sortOrder < 2^31.
	//
	// A calendar with a lower order should be displayed before a calendar with a higher order in any
	// list of calendars in the client’s UI.
	//
	// Calendars with equal order SHOULD be sorted in alphabetical order by name.
	//
	// The sorting should take into account locale-specific character order convention.
	SortOrder uint `json:"sortOrder,omitzero"`

	// True if the user has indicated they wish to see this Calendar in their client.
	//
	// This SHOULD default to `false` for Calendars in shared accounts the user has access to and `true`
	// for any new Calendars created by the user themself.
	//
	// If false, the calendar SHOULD only be displayed when the user explicitly requests it or to offer
	// it for the user to subscribe to.
	//
	// For example, a company may have a large number of shared calendars which all employees have
	// permission to access, but you would only subscribe to the ones you care about and want to be able
	// to have normally accessible.
	IsSubscribed bool `json:"isSubscribed"`

	// Should the calendar’s events be displayed to the user at the moment?
	//
	// Clients MUST ignore this property if `isSubscribed` is false.
	//
	// If an event is in multiple calendars, it should be displayed if `isVisible` is `true`
	// for any of those calendars.
	//
	// default: true
	IsVisible bool `json:"isVisible"`

	// This SHOULD be true for exactly one calendar in any account, and MUST NOT be true for more
	// than one calendar within an account (server-set).
	//
	// The default calendar should be used by clients whenever they need to choose a calendar
	// for the user within this account, and they do not have any other information on which to make
	// a choice.
	//
	// For example, if the user creates a new event, the client may automatically set the event as
	// belonging to the default calendar from the user’s primary account.
	IsDefault bool `json:"isDefault,omitzero"`

	// Should the calendar’s events be used as part of availability calculation?
	//
	// This MUST be one of:
	// !- `all``: all events are considered.
	// !- `attending``: events the user is a confirmed or tentative participant of are considered.
	// !- `none``: all events are ignored (but may be considered if also in another calendar).
	//
	// This should default to “all” for the calendars in the user’s own account, and “none” for calendars shared with the user.
	IncludeInAvailability IncludeInAvailability `json:"includeInAvailability,omitempty"`

	// A map of alert ids to Alert objects (see [@!RFC8984], Section 4.5.2) to apply for events
	// where `showWithoutTime` is `false` and `useDefaultAlerts` is `true`.
	//
	// Ids MUST be unique across all default alerts in the account, including those in other
	// calendars; a UUID is recommended.

	// If omitted on creation, the default is server dependent.
	//
	// For example, servers may choose to always default to null, or may copy the alerts from the default calendar.
	DefaultAlertsWithTime map[string]jscalendar.Alert `json:"defaultAlertsWithTime,omitempty"`

	// A map of alert ids to Alert objects (see [@!RFC8984], Section 4.5.2) to apply for events where
	// `showWithoutTime` is `true` and `useDefaultAlerts` is `true`.
	//
	// Ids MUST be unique across all default alerts in the account, including those in other
	// calendars; a UUID is recommended.
	//
	// If omitted on creation, the default is server dependent.
	//
	// For example, servers may choose to always default to null, or may copy the alerts from the default calendar.
	DefaultAlertsWithoutTime map[string]jscalendar.Alert `json:"defaultAlertsWithoutTime,omitempty"`

	// The time zone to use for events without a time zone when the server needs to resolve them into
	// absolute time, e.g., for alerts or availability calculation.
	//
	// The value MUST be a time zone id from the IANA Time Zone Database TZDB.
	//
	// If null, the `timeZone` of the account’s associated `Principal` will be used.
	//
	// Clients SHOULD use this as the default for new events in this calendar if set.
	TimeZone string `json:"timeZone,omitempty"`

	// A map of `Principal` id to rights for principals this calendar is shared with.
	//
	// The principal to which this calendar belongs MUST NOT be in this set.
	//
	// This is null if the calendar is not shared with anyone.
	//
	// May be modified only if the user has the `mayAdmin` right.
	//
	// The account id for the principals may be found in the `urn:ietf:params:jmap:principals:owner`
	// capability of the `Account` to which the calendar belongs.
	ShareWith map[string]CalendarRights `json:"shareWith,omitempty"`

	// The set of access rights the user has in relation to this Calendar.
	//
	// If any event is in multiple calendars, the user has the following rights:
	// !- The user may fetch the event if they have the mayReadItems right on any calendar the event is in.
	// !- The user may remove an event from a calendar (by modifying the event’s “calendarIds” property) if the user
	// has the appropriate permission for that calendar.
	// !- The user may make other changes to the event if they have the right to do so in all calendars to which the
	// event belongs.
	MyRights *CalendarRights `json:"myRights,omitempty"`
}

// A CalendarEvent object contains information about an event, or recurring series of events,
// that takes place at a particular time.
//
// It is a JSCalendar Event object, as defined in [@!RFC8984], with additional properties.
type CalendarEvent struct {

	// The id of the CalendarEvent (immutable; server-set).
	//
	// The id uniquely identifies a JSCalendar Event with a particular `uid` and
	// `recurrenceId` within a particular account.
	Id string `json:"id"`

	// This is only defined if the `id` property is a synthetic id, generated by the
	// server to represent a particular instance of a recurring event (immutable; server-set).
	//
	// This property gives the id of the "real" `CalendarEvent` this was generated from.
	BaseEventId string `json:"baseEventId,omitempty"`

	// The set of Calendar ids this event belongs to.
	//
	// An event MUST belong to one or more Calendars at all times (until it is destroyed).
	//
	// The set is represented as an object, with each key being a Calendar id.
	//
	// The value for each key in the object MUST be `true`.
	CalendarIds map[string]bool `json:"calendarIds,omitempty"`

	// If true, this event is to be considered a draft.
	//
	// The server will not send any scheduling messages to participants or send push notifications
	// for alerts.
	//
	// This may only be set to `true` upon creation.
	//
	// Once set to `false`, the value cannot be updated to `true`.
	//
	// This property MUST NOT appear in `recurrenceOverrides`.
	IsDraft bool `json:"isDraft,omitzero"`

	// Is this the authoritative source for this event (i.e., does it control scheduling for
	// this event; the event has not been added as a result of an invitation from another calendar system)?
	//
	// This is true if, and only if:
	// !- the event’s `replyTo` property is null; or
	// !- the account will receive messages sent to at least one of the methods specified in the `replyTo` property of the event.
	IsOrigin bool `json:"isOrigin,omitzero"`

	// For simple clients that do not implement time zone support.
	//
	// Clients should only use this if also asking the server to expand recurrences, as you cannot accurately
	// expand a recurrence without the original time zone.
	//
	// This property is calculated at fetch time by the server.
	//
	// Time zones are political and they can and do change at any time.
	//
	// Fetching exactly the same property again may return a different results if the time zone data has been updated on the server.
	//
	// Time zone data changes are not considered `updates` to the event.
	//
	// If set, the server will convert the UTC date to the event's current time zone and store the local time.
	//
	// This property is not included in `CalendarEvent/get` responses by default and must be requested explicitly.
	//
	// Floating events (events without a time zone) will be interpreted as per the time zone given as a `CalendarEvent/get` argument.
	//
	// Note that it is not possible to accurately calculate the expansion of recurrence rules or recurrence overrides with the
	// `utcStart` property rather than the local start time. Even simple recurrences such as "repeat weekly" may cross a
	// daylight-savings boundary and end up at a different UTC time. Clients that wish to use "utcStart" are RECOMMENDED to
	// request the server expand recurrences.
	UtcStart UTCDate `json:"utcStart,omitzero"`

	// The server calculates the end time in UTC from the start/timeZone/duration properties of the event.
	//
	// This property is not included by default and must be requested explicitly.
	//
	// Like `utcStart`, it is calculated at fetch time if requested and may change due to time zone data changes.
	//
	// Floating events will be interpreted as per the time zone given as a `CalendarEvent/get` argument.
	UtcEnd UTCDate `json:"utcEnd,omitzero"`

	jscalendar.Event
}

// A ParticipantIdentity stores information about a URI that represents the user within that account in an event’s participants.
type ParticipantIdentity struct {
	// The id of the ParticipantIdentity (immutable; server-set).
	Id string `json:"id"`

	// The display name of the participant to use when adding this participant to an event, e.g. "Joe Bloggs".
	//
	// default:
	Name string `json:"name,omitempty"`

	// The URI that represents this participant for scheduling.
	//
	// This URI MAY also be the URI for one of the sendTo methods.
	ScheduleId string `json:"scheduleId"`

	// Represents methods by which the participant may receive invitations and updates to an event.
	//
	// The keys in the property value are the available methods and MUST only contain ASCII alphanumeric
	// characters (`A-Za-z0-9`).
	//
	// The value is a URI for the method specified in the key.
	SendTo map[string]string `json:"sendTo,omitempty"`

	// This SHOULD be true for exactly one participant identity in any account, and MUST NOT be true for more
	// than one participant identity within an account (server-set).
	//
	// The default identity should be used by clients whenever they need to choose an identity for the user
	// within this account, and they do not have any other information on which to make a choice.
	//
	// For example, if creating a scheduled event in this account, the default identity may be automatically
	// added as an owner. (But the client may ignore this if, for example, it has its own feature to allow
	// users to choose which identity to use based on the invitees.)
	IsDefault bool `json:"isDefault,omitzero"`
}

type CalendarAlert struct {
	//  This MUST be the string `CalendarAlert`.
	Type TypeOfCalendarAlert `json:"@type,omitempty"`

	// The account id for the calendar in which the alert triggered.
	AccountId string `json:"accountId"`

	// The CalendarEvent id for the alert that triggered.
	//
	// Note, for a recurring event this is the id of the base event, never a synthetic id for a particular instance.
	CalendarEventId string `json:"calendarEventId"`

	// The uid property of the CalendarEvent for the alert that triggered.
	Uid string `json:"uid"`

	// The `recurrenceId` for the instance of the event for which this alert is being
	// triggered, or null if the event is not recurring.
	RecurrenceId LocalDate `json:"recurrenceId,omitzero"`

	// The id for the alert that triggered.
	AlertId string `json:"alertId"`
}

type Person struct {
	// The name of the person who made the change.
	Name string `json:"name"`

	// The email of the person who made the change, or null if no email is available.
	Email string `json:"email,omitempty"`

	// The id of the `Principal` corresponding to the person who made the change, if any.
	//
	// This will be null if the change was due to receving an iTIP message.
	PrincipalId string `json:"principalId,omitempty"`

	// The `scheduleId` URI of the person who made the change, if any.
	//
	// This will normally be set if the change was made due to receving an iTIP message.
	ScheduleId string `json:"scheduleId,omitempty"`
}

type CalendarEventNotification struct {
	// The id of the `CalendarEventNotification`.
	Id string `json:"id"`

	// The time this notification was created.
	Created UTCDate `json:"created,omitzero"`

	// Who made the change.
	ChangedBy *Person `json:"person,omitempty"`

	// Comment sent along with the change by the user that made it.
	//
	// (e.g. `COMMENT` property in an iTIP message), if any.
	Comment string `json:"comment,omitempty"`

	// `CalendarEventNotification` type.
	//
	// This MUST be one of
	// !- `created`
	// !- `updated`
	// !- `destroyed`
	Type CalendarEventNotificationTypeOption `json:"type"`

	// The id of the CalendarEvent that this notification is about.
	//
	// If the change only affects a single instance of a recurring event, the server MAY set the
	// `event` and `event`atch properties for just that instance; the `calendarEventId` MUST
	// still be for the base event.
	CalendarEventId string `json:"calendarEventId"`

	// Is this event a draft? (created/updated only)
	IsDraft bool `json:"isDraft,omitzero"`

	// The data before the change (if updated or destroyed),
	// or the data after creation (if created).
	Event *jscalendar.Event `json:"event,omitempty"`

	// A patch encoding the change between the data in the event property,
	// and the data after the update (updated only).
	EventPatch PatchObject `json:"eventPatch,omitempty"`
}

// Denotes the task list has a special purpose.
//
// This MUST be one of the following:
// !- `inbox`: This is the principal’s default task list;
// !- `trash`: This task list holds messages the user has discarded;
type TaskListRole string

const (
	// This is the principal’s default task list.
	TaskListRoleInbox = TaskListRole("inbox")
	// This task list holds messages the user has discarded.
	TaskListRoleTrash = TaskListRole("trash")
)

var (
	DefaultWorkflowStatuses = []string{
		"completed",
		"failed",
		"in-process",
		"needs-action",
		"cancelled",
		"pending",
	}
)

type TaskRights struct {
	// The user may fetch the tasks in this task list.
	MayReadItems bool `json:"mayReadItems"`

	// The user may create, modify or destroy all tasks in this task list,
	// or move tasks to or from this task list.
	//
	// If this is `true`, the `mayWriteOwn`, `mayUpdatePrivate` and `mayRSVP` properties
	// MUST all also be `true`.
	MayWriteAll bool `json:"mayWriteAll"`

	// The user may create, modify or destroy a task on this task list if either they are
	// the owner of the task (see below) or the task has no owner.
	//
	// This means the user may also transfer ownership by updating a task so they are no longer
	// an owner.
	MayWriteOwn bool `json:"mayWriteOwn"`

	// The user may modify the following properties on all tasks in the task list, even
	// if they would not otherwise have permission to modify that task.
	//
	// These properties MUST all be stored per-user, and changes do not affect any other user of the task list.
	//
	// The user may also modify the above on a per-occurrence basis for recurring tasks
	// (updating the `recurrenceOverrides` property of the task to do so).
	MayUpdatePrivate bool `json:"mayUpdatePrivate"`

	// The user may modify the following properties of any `Participant` object that corresponds
	// to one of the user’s `ParticipantIdentity` objects in the account, even if they would not
	// otherwise have permission to modify that task
	// !- `participationStatus`
	// !- `participationComment`
	// !- `expectReply`
	//
	// If the task has its `mayInviteSelf` property set to true, then the user may also add a new
	// `Participant` to the task with a `sendTo` property that is the same as the `sendTo` property
	// of one of the user’s `ParticipantIdentity` objects in the account.
	// The `roles` property of the participant MUST only contain `attendee`.
	//
	// If the task has its `mayInviteOthers` property set to `true` and there is an existing
	// `Participant` in the task corresponding to one of the user’s `ParticipantIdentity` objects
	// in the account, then the user may also add new participants.
	// The `roles` property of any new participant MUST only contain `attendee`.
	//
	// The user may also do all of the above on a per-occurrence basis for recurring tasks
	// (updating the `recurrenceOverrides` property of the task to do so).
	MayRSVP bool `json:"mayRSVP"`

	//  The user may modify sharing for this task list.
	MayAdmin bool `json:"mayAdmin"`

	// The user may delete the task list itself (server-set).
	//
	// This property MUST be false if the account to which this task list belongs has the `isReadOnly`
	// property set to true.
	MayDelete bool `json:"mayDelete"`
}

type TaskList struct {
	// The id of the task list (immutable; server-set).
	Id string `json:"id,omitempty"`

	// Denotes the task list has a special purpose.
	//
	// This MUST be one of the following:
	// !- `inbox`: This is the principal’s default task list;
	// !- `trash`: This task list holds messages the user has discarded;
	Role TaskListRole `json:"role,omitempty"`

	// The user-visible name of the task list.
	//
	// This may be any UTF-8 string of at least 1 character in length and maximum 255 octets in size.
	Name string `json:"name,omitempty"`

	// An optional longer-form description of the task list, to provide context in shared environments
	// where users need more than just the name.
	Description string `json:"description,omitempty"`

	// A color to be used when displaying tasks associated with the task list.
	//
	// If not null, the value MUST be a case-insensitive color name taken from the set of names defined
	// in Section 4.3 of CSS Color Module Level 3 COLORS, or an RGB value in hexadecimal notation,
	// as defined in Section 4.2.1 of CSS Color Module Level 3.
	//
	// The color SHOULD have sufficient contrast to be used as text on a white background.
	Color string `json:"color,omitempty"`

	// A map of keywords to the colors used when displaying the keywords associated to a task.
	//
	// The same considerations, as for `color` above, apply.
	KeywordColors map[string]string `json:"keywordColors,omitempty"`

	// A map of categories to the colors used when displaying the categories associated to a task.
	//
	// The same considerations, as for `color` above, apply.
	CategoryColors map[string]string `json:"categoryColors,omitempty"`

	// Defines the sort order of task lists when presented in the client’s UI, so it is consistent
	// between devices.
	//
	// The number MUST be an integer in the range 0 ≤ sortOrder < 2^31.
	//
	// A task list with a lower order should be displayed before a list with a higher order in any list
	// of task lists in the client’s UI.
	//
	// Task lists with equal order SHOULD be sorted in alphabetical order by name.
	//
	// The sorting should take into account locale-specific character order convention.
	SortOrder uint `json:"sortOrder,omitzero"`

	// Has the user indicated they wish to see this task list in their client?
	//
	// This SHOULD default to false for task lists in shared accounts the user has access to,
	// and true for any new task list created by the user themselves.
	//
	// If false, the task list should only be displayed when the user explicitly
	// requests it or to offer it for the user to subscribe to.
	IsSubscribed bool `json:"isSubscribed,omitzero"`

	// The time zone to use for tasks without a time zone when the server needs to resolve them
	// into absolute time, e.g., for alerts or availability calculation.
	//
	// The value MUST be a time zone id from the IANA Time Zone Database TZDB.
	//
	// If null, the timeZone of the account’s associated Principal will be used.
	//
	// Clients SHOULD use this as the default for new tasks in this task list, if set.
	TimeZone string `json:"timeZone,omitempty"`

	// Defines the allowed values for `workflowStatus`.
	//
	// The default values are based on the values defined within [@!RFC8984], Section 5.2.5 and `pending`.
	//
	// `pending` indicates the task has been created and accepted, but it currently is on-hold.
	//
	// As naming and workflows differ between systems, mapping the status correctly to the present values
	// of the `Task` can be challenging. In the most simple case, a task system may support merely two states - `done`
	// and `not-done`.
	//
	// On the other hand, statuses and their semantic meaning can differ between systems or task lists (e.g. projects).
	//
	// In case of uncertainty, here are some recommendations for mapping commonly observed values that can help
	// during implementation:
	// !- `completed`: `done` (most simple case), `closed`, `verified`, …
	// !- `in-process`: `in-progress`, `active`, `assigned`, …
	// !- `needs-action`: `not-done` (most simple case), `not-started`, `new`, …
	// !- `pending`: `waiting`, `deferred`, `on-hold`, `paused`, …
	WorkflowStatuses []string `json:"workflowStatuses,omitempty"`

	// A map of `Principal` id to rights for principals this task list is shared with.
	//
	// The principal to which this task list belongs MUST NOT be in this set.
	//
	// This is null if the task list is not shared with anyone.
	//
	// May be modified only if the user has the `mayAdmin` right.
	//
	// The account id for the principals may be found in the `urn:ietf:params:jmap:principals:owner` capability
	// of the `Account` to which the task list belongs.
	ShareWith map[string]TaskRights `json:"shareWith,omitempty"`

	// The set of access rights the user has in relation to this `TaskList`.
	//
	// The user may fetch the task if they have the `mayReadItems` right on any task list the task is in.
	//
	// The user may remove a task from a task list (by modifying the task’s `taskListId` property) if the user has the
	// appropriate permission for that task list.
	//
	// The user may make other changes to the task if they have the right to do so in all task list to which the task belongs.
	MyRights *TaskRights `json:"myRights,omitempty"`

	// A map of alert ids to `Alert` objects (see [@!RFC8984], Section 4.5.2) to apply for tasks
	// where `showWithoutTime` is `false` and `useDefaultAlerts` is `true`.
	//
	// Ids MUST be unique across all default alerts in the account, including those in other task
	// lists; a UUID is recommended.
	//
	// If omitted on creation, the default is server dependent.
	//
	// For example, servers may choose to always default to null, or may copy the alerts from the default task list.
	DefaultAlertsWithTime map[string]jscalendar.Alert `json:"defaultAlertsWithTime,omitempty"`

	// A map of alert ids to `Alert` objects (see [@!RFC8984], Section 4.5.2) to apply for tasks
	// where `showWithoutTime` is `true` and `useDefaultAlerts` is `true`.
	//
	// Ids MUST be unique across all default alerts in the account, including those in other task
	// lists; a UUID is recommended.
	//
	// If omitted on creation, the default is server dependent. For example, servers may choose to always
	// default to `null`, or may copy the alerts from the default task list.
	DefaultAlertsWithoutTime map[string]jscalendar.Alert `json:"defaultAlertsWithoutTime,omitempty"`
}

type TypeOfChecklist string
type TypeOfCheckItem string
type TypeOfTaskPerson string
type TypeOfComment string

type TaskNotificationTypeOption string

const ChecklistType = TypeOfChecklist("Checklist")
const CheckItemType = TypeOfCheckItem("CheckItem")
const TaskPersonType = TypeOfTaskPerson("Person")
const CommentType = TypeOfComment("Comment")
const TaskNotificationTypeOptionCreated = TaskNotificationTypeOption("created")
const TaskNotificationTypeOptionUpdated = TaskNotificationTypeOption("updated")
const TaskNotificationTypeOptionDestroyed = TaskNotificationTypeOption("destroyed")

// The Person object has the following properties of which either principalId or uri MUST be defined.
type TaskPerson struct {
	// Specifies the type of this object, this MUST be `Person`.
	Type TypeOfTaskPerson `json:"@type,omitempty"`

	// The name of the person.
	Name string `json:"name,omitempty"`

	// A URI value that identifies the person.
	//
	// This SHOULD be the `scheduleId` of the participant that this item was assigned to.
	Uri string `json:"uri,omitempty"`

	// The id of the Principal corresponding to the person, if any.
	PrincipalId string `json:"principalId,omitempty"`
}

type Comment struct {
	// Specifies the type of this object, this MUST be `Comment`.
	Type TypeOfComment `json:"@type,omitempty"`

	// The free text value of this comment.
	Message string `json:"message"`

	// The date and time when this note was created.
	Created UTCDate `json:"created,omitzero"`

	// The date and time when this note was updated.
	Updated UTCDate `json:"updated,omitzero"`

	// The author of this comment.
	Author *TaskPerson `json:"author,omitempty"`
}

type CheckItem struct {
	// Specifies the type of this object, this MUST be `CheckItem`.
	Type TypeOfCheckItem `json:"@type,omitempty"`

	// Title of the item.
	Title string `json:"title,omitempty"`

	// Defines the sort order of `CheckItem` when presented in the client’s UI.
	//
	// The number MUST be an integer in the range 0 <= sortOrder < 2^31.
	//
	// An item with a lower order should be displayed before an item with a higher order.
	//
	// Items with equal order SHOULD be sorted in alphabetical order by name.
	//
	// The sorting should take into account locale-specific character order convention.
	SortOrder uint `json:"sortOrder,omitzero"`

	// The date and time when this item was updated.
	Updated UTCDate `json:"updated,omitzero"`

	IsComplete bool `json:"isComplete,omitzero"`

	// The person that this item is assigned to.
	//
	// The `Person` object has the following properties of which either `principalId` or `uri`
	// MUST be defined.
	Assignee *TaskPerson `json:"assignee,omitempty"`

	// Free-text comments associated with this task.
	Comments map[string]Comment `json:"comments,omitempty"`
}

type Checklist struct {
	// Specifies the type of this object, this MUST be `Checklist`.
	Type TypeOfChecklist `json:"@type,omitempty"`

	// Title of the list.
	Title string `json:"title,omitempty"`

	// The items of the check list.
	CheckItems []CheckItem `json:"checkItems,omitempty"`
}

// A `Task` object contains information about a task.
//
// It is a JSTask object, as defined in [@!RFC8984]. However, as use-cases of task systems vary, this
// Section defines relevant parts of the JSTask object to implement the core task capability as well
// as several extensions to it.
//
// Only the core capability MUST be implemented by any task system.
//
// Implementers can choose the extensions that fit their own use case.
//
// For example, the recurrence extension allows having a `Task` object represent a series of recurring `Task`s.
type Task struct {

	// The id of the Task.
	//
	// This property is immutable.
	//
	// The id uniquely identifies a JSTask with a particular `uid` and `recurrenceId` within a particular account.
	Id string `json:"id"`

	// The `TaskList` id this task belongs to.
	//
	// A task MUST belong to exactly one `TaskList` at all times (until it is destroyed).
	TaskListId string `json:"taskListId"`

	// If `true`, this task is to be considered a draft.
	//
	// The server will not send any push notifications for alerts.
	//
	// This may only be set to true upon creation.
	//
	// Once set to `false`, the value cannot be updated to `true`.
	//
	// This property MUST NOT appear in `recurrenceOverrides`.
	IsDraft bool `json:"isDraft,omitzero"`

	UtcStart UTCDate `json:"utcStart,omitzero"`

	UtcDue UTCDate `json:"utcDue,omitzero"`

	SortOrder uint `json:"sortOrder,omitzero"`

	WorkflowStatus string `json:"workflowStatus,omitempty"`

	jscalendar.Task

	// This specifies the estimated amount of work the task takes to complete.
	//
	// In Agile software development or Scrum, it is known as complexity or story points.
	//
	// The number has no actual unit, but a larger number means more work.
	EstimatedWork uint `json:"estimatedWork,omitzero"`

	// This specifies the impact or severity of the task, but does not say anything
	// about the actual prioritization.
	//
	// Some examples are: `minor`, `trivial`, `major` or `block`.
	//
	// Usually, the priority of a task is based upon its impact and urgency.
	Impact string `json:"impact,omitempty"`

	// A map of Checklist IDs to Checklist objects, containing checklist items.
	Checklists map[string]Checklist `json:"checklists,omitempty"`

	// This is only defined if the id property is a synthetic id, generated by the server
	// to represent a particular instance of a recurring Task (immutable; server-set).
	//
	// This property gives the id of the “real” Task this was generated from.
	BaseTaskId string `json:"baseTaskId,omitempty"`

	// Is this the authoritative source for this task (i.e., does it control scheduling
	// for this task; the task has not been added as a result of an invitation from another
	// task management system)?
	//
	// This is `true` if, and only if:
	// !- the task’s “replyTo” property is null; or
	// !- the account will receive messages sent to at least one of the methods specified in
	// the `replyTo` property of the task.
	IsOrigin bool `json:"isOrigin,omitzero"`

	// If true, any user that has access to the task may add themselves to it as a participant
	// with the `attendee` role.
	//
	// This property MUST NOT be altered in the `recurrenceOverrides`; it may only be set on the master object.
	//
	// This indicates the task will accept “party crasher” RSVPs via iTIP, subject to any other domain-specific
	// restrictions, and users may add themselves to the task via JMAP as long as they have the `mayRSVP`
	// permission for the task list.
	//
	// default: false
	MayInviteSelf bool `json:"mayInviteSelf,omitzero"`

	// If true, any current participant with the `attendee` role may add new participants with
	// the `attendee` role to the task.
	//
	// This property MUST NOT be altered in the `recurrenceOverrides`; it may only be set on the master object.
	//
	// default: false
	MayInviteOthers bool `json:"mayInviteOthers,omitzero"`

	// If true, only the owners of the task may see the full set of participants.
	//
	// Other sharees of the task may only see the owners and themselves.
	//
	// This property MUST NOT be altered in the `recurrenceOverrides`; it may only be set on the master object.
	HideAttendees bool `json:"hideAttendees,omitzero"`
}

// The `TaskNotification` data type records changes made by external entities to tasks in task lists
// the user is subscribed to.
//
// Notifications are stored in the same `Account` as the `Task` that was changed.
type TaskNotification struct {
	// The id of the `TaskNotification`.
	Id string `json:"id"`

	// The time this notification was created.
	Created UTCDate `json:"created,omitzero"`

	// Who made the change.
	ChangedBy *TaskPerson `json:"changedBy,omitempty"`

	// Comment sent along with the change by the user that made it.
	//
	// (e.g. `COMMENT` property in an iTIP message), if any.
	Comment string `json:"comment,omitempty"`

	// This MUST be one of
	// !- `created`
	// !- `updated`
	// !- `destroyed`
	Type TaskNotificationTypeOption `json:"type"`

	// The id of the Task that this notification is about.
	TaskId string `json:"taskId"`

	// Is this task a draft? (created/updated only)
	IsDraft bool `json:"isDraft,omitzero"`

	// The data before the change (if updated or destroyed), or the data after creation (if created).
	Task *jscalendar.Task `json:"task,omitempty"`

	// A patch encoding the change between the data in the task property, and the data after the update updated only).
	TaskPatch PatchObject `json:"taskPatch,omitempty"`
}

// A Principal represents an individual, group, location (e.g. a room), resource (e.g. a projector) or other entity
// in a collaborative environment.
//
// Sharing in JMAP is generally configured by assigning rights to certain data within an account to other principals,
// for example a user may assign permission to read their calendar to a principal representing another user, or their team.
//
// In a shared environment such as a workplace, a user may have access to a large number of principals.
//
// In most systems the user will have access to a single `Account` containing `Principal` objects, but they may
// have access to multiple if, for example, aggregating data from different places.
type Principal struct {
	// The id of the principal.
	Id string `json:"id"`

	// `Principal` type.
	//
	// This MUST be one of the following values:
	// !- `individual`: This represents a single person.
	// !- `group`: This represents a group of people.
	// !- `resource`: This represents some resource, e.g. a projector.
	// !- `location`: This represents a location.
	// !- `other`: This represents some other undefined principal.
	Type PrincipalTypeOption `json:"type"`

	// The name of the principal, e.g. `"Jane Doe"`, or `"Room 4B"`.
	Name string `json:"name"`

	// A longer description of the principal, for example details about the
	// facilities of a resource, or null if no description available.
	Description string `json:"description,omitempty"`

	// An email address for the principal, or null if no email is available.
	Email string `json:"email,omitempty"`

	// The time zone for this principal, if known.
	//
	// If not null, the value MUST be a time zone id from the IANA Time Zone Database TZDB.
	TimeZone string `json:"timeZone,omitempty"`

	// A map of JMAP capability URIs to domain specific information about the principal in relation
	// to that capability, as defined in the document that registered the capability.
	Capabilities map[string]any `json:"capabilities,omitempty"`

	// A map of account id to `Account` object for each JMAP Account containing data for
	// this principal that the user has access to, or null if none.
	Accounts map[string]Account `json:"accounts,omitempty"`
}

// TODO https://jmap.io/spec-sharing.html#object-properties
type ShareNotification struct {
}

type Shareable struct {
	// Has the user indicated they wish to see this data?
	//
	// The initial value for this when data is shared by another user is implementation dependent,
	// although data types may give advice on appropriate defaults.
	IsSubscribed bool `json:"isSubscribed,omitzero"`

	// The set of permissions the user currently has.
	//
	// Appropriate permissions are domain specific and must be defined per data type.
	MyRights map[string]bool `json:"myRights,omitempty"`

	// A map of principal id to rights to give that principal, or null if not shared with anyone.
	//
	// The account id for the principal id can be found in the capabilities of the `Account` this object is in.
	//
	// Users with appropriate permission may set this property to modify who the data is shared with.
	//
	// The principal that owns the account this data is in MUST NOT be in the set of sharees; their rights are implicit.
	ShareWith map[string]map[string]bool `json:"shareWith,omitempty"`
}

// The Quota is an object that displays the limit set to an account usage.
//
// It then shows as well the current usage in regard to that limit.
type Quota struct {
	// The unique identifier for this object.
	Id string `json:"id"`

	// The resource type of the quota.
	ResourceType ResourceType `json:"resourceType"`

	// The current usage of the defined quota, using the `resourceType` defined as unit of measure.
	//
	// Computation of this value is handled by the server.
	Used uint `json:"used"`

	// The hard limit set by this quota, using the `resourceType` defined as unit of measure.
	//
	// Objects in scope may not be created or updated if this limit is reached.
	HardLimit uint `json:"hardLimit"`

	// The Scope data type is used to represent the entities the quota applies to.
	//
	// It is defined as a "String" with values from the following set:
	// !- `account`: The quota information applies to just the client's account.
	// !- `domain`: The quota information applies to all accounts sharing this domain.
	// !- `global`: The quota information applies to all accounts belonging to the server.
	Scope Scope `json:"scope"`

	// The name of the quota.
	//
	// Useful for managing quotas and using queries for searching.
	Name string `json:"name"`

	// A list of all the type names as defined in the "JMAP Types Names" registry
	// (e.g., `Email`, `Calendar`, etc.) to which this quota applies.
	//
	// This allows the quotas to be assigned to distinct or shared data types.
	//
	// The server MUST filter out any types for which the client did not request the associated capability
	// in the `using` section of the request.
	//
	// Further, the server MUST NOT return Quota objects for which there are no types recognized by the client.
	Types []ObjectType `json:"types,omitempty"`

	// The warn limit set by this quota, using the `resourceType` defined as unit of measure.
	//
	// It can be used to send a warning to an entity about to reach the hard limit soon, but with no
	// action taken yet.
	//
	// If set, it SHOULD be lower than the `softLimit` (if present and different from null) and the `hardLimit`.
	WarnLimit uint `json:"warnLimit,omitzero"`

	// The soft limit set by this quota, using the `resourceType` defined as unit of measure.
	//
	// It can be used to still allow some operations but refuse some others.
	//
	// What is allowed or not is up to the server.
	//
	// For example, it could be used for blocking outgoing events of an entity (sending emails, creating
	// calendar events, etc.) while still receiving incoming events (receiving emails, receiving calendars
	// events, etc.).
	//
	// If set, it SHOULD be higher than the `warnLimit` (if present and different from null) but lower
	// than the `hardLimit`.
	SoftLimit uint `json:"softLimit,omitzero"`

	// Arbitrary, free, human-readable description of this quota.
	//
	// It might be used to explain where the different limits come from and explain the entities and data
	// types this quota applies to.
	//
	// The description MUST be encoded in UTF-8 [RFC3629] as described in [RFC8620], Section 1.5, and
	// selected based on an `Accept-Language` header in the request (as defined in [RFC9110], Section 12.5.4)
	// or out-of-band information about the user's language or locale.
	Description string `json:"description,omitempty"`
}

// See [RFC8098] for the exact meaning of these different fields.
//
// These fields are defined as case insensitive in [RFC8098] but are case sensitive in this RFC
// and MUST be converted to lowercase by "MDN/parse".
type Disposition struct {
	ActionMode  ActionMode            `json:"actionMode,omitempty"`
	SendingMode SendingMode           `json:"sendingMode,omitempty"`
	Type        DispositionTypeOption `json:"type,omitempty"`
}

// Message Disposition Notifications (MDNs) are defined in [RFC8098] and are used as "read receipts",
// "acknowledgements", or "receipt notifications".
//
// A client can come across MDNs in different ways:
// 1. When receiving an email message, an MDN can be sent to the sender. This specification defines an `MDN/send` method to cover this case.
// 2. When sending an email message, an MDN can be requested. This must be done with the help of a header field, as already specified by [RFC8098];
// the header field can already be handled by guidance in [RFC8621].
// 3. When receiving an MDN, the MDN could be related to an existing sent message. This is already covered by [RFC8621] in the
// `EmailSubmission` object. A client might want to display detailed information about a received MDN.
// This specification defines an `MDN/parse` method to cover this case.
type MDN struct {
	// The `Email` id of the received message to which this MDN is related.
	//
	// This property MUST NOT be null for `MDN/send` but MAY be null in the response from the `MDN/parse` method.
	ForEmailId string `json:"forEmailId,omitempty"`

	// The subject used as `Subject` header field for this MDN.
	Subject string `json:"subject,omitempty"`

	// The human-readable part of the MDN, as plain text.
	TextBody string `json:"textBody,omitempty"`

	// If true, the content of the original message will appear in the third component of the `multipart/report` generated
	// for the MDN.
	//
	// See [RFC8098] for details and security considerations.
	IncludeOriginalMessage bool `json:"includeOriginalMessage,omitzero"`

	// The name of the Mail User Agent (MUA) creating this MDN.
	//
	// It is used to build the MDN report part of the MDN.
	//
	// Note that a null value may have better privacy properties.
	ReportingUA string `json:"reportingUA,omitempty"`

	// The object containing the diverse MDN disposition options.
	Disposition Disposition `json:"disposition"`

	// The name of the gateway or Message Transfer Agent (MTA) that translated a foreign (non-Internet)
	// message disposition notification into this MDN (server-set).
	MdnGateway string `json:"mdnGateway,omitempty"`

	// The original recipient address as specified by the sender of the message for which the MDN is being issued (server-set).
	OriginalRecipient string `json:"originalRecipient,omitempty"`

	// The recipient for which the MDN is being issued.
	//
	// If set, it overrides the value that would be calculated by the server from the `Identity` defined
	// in the `MDN/send` method, unless explicitly set by the client.
	FinalRecipient string `json:"finalRecipient,omitempty"`

	// The `Message-ID` header field [RFC5322] (not the JMAP id) of the message for which the MDN is being issued.
	OriginalMessageId string `json:"originalMessageId,omitempty"`

	// Additional information in the form of text messages when the `error` disposition modifier appears.
	Error []string `json:"error,omitempty"`

	// The object where keys are extension-field names, and values are extension-field values (see [RFC8098], Section 3.3).
	ExtensionFields map[string]string `json:"extensionFields,omitempty"`
}

type SendMDN struct {
	// The id of the account to use.
	AccountId string `json:"accountId"`

	// The id of the `Identity` to associate with these MDNs.
	//
	// The server will use this identity to define the sender of the MDNs and to set the `finalRecipient` field.
	IdentityId string `json:"identityId"`

	// A map of the creation id (client specified) to MDN objects.
	Send map[string]MDN `json:"send,omitempty"`

	// A map of the id to an object containing properties to update on the `Email` object referenced by the `MDN/send`
	// if the sending succeeds.
	//
	// This will always be a backward reference to the creation id.
	OnSuccessUpdateEmail map[string]PatchObject `json:"onSuccessUpdateEmail,omitempty"`
}

type ErrorResponse struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

const (
	ErrorCommand               Command = "error" // only occurs in responses
	CommandBlobGet             Command = "Blob/get"
	CommandBlobUpload          Command = "Blob/upload"
	CommandEmailGet            Command = "Email/get"
	CommandEmailQuery          Command = "Email/query"
	CommandEmailChanges        Command = "Email/changes"
	CommandEmailSet            Command = "Email/set"
	CommandEmailImport         Command = "Email/import"
	CommandEmailSubmissionGet  Command = "EmailSubmission/get"
	CommandEmailSubmissionSet  Command = "EmailSubmission/set"
	CommandThreadGet           Command = "Thread/get"
	CommandMailboxGet          Command = "Mailbox/get"
	CommandMailboxQuery        Command = "Mailbox/query"
	CommandMailboxChanges      Command = "Mailbox/changes"
	CommandIdentityGet         Command = "Identity/get"
	CommandVacationResponseGet Command = "VacationResponse/get"
	CommandVacationResponseSet Command = "VacationResponse/set"
	CommandSearchSnippetGet    Command = "SearchSnippet/get"
)

var CommandResponseTypeMap = map[Command]func() any{
	ErrorCommand:               func() any { return ErrorResponse{} },
	CommandBlobGet:             func() any { return BlobGetResponse{} },
	CommandBlobUpload:          func() any { return BlobUploadResponse{} },
	CommandMailboxQuery:        func() any { return MailboxQueryResponse{} },
	CommandMailboxGet:          func() any { return MailboxGetResponse{} },
	CommandMailboxChanges:      func() any { return MailboxChangesResponse{} },
	CommandEmailQuery:          func() any { return EmailQueryResponse{} },
	CommandEmailChanges:        func() any { return EmailChangesResponse{} },
	CommandEmailGet:            func() any { return EmailGetResponse{} },
	CommandEmailSubmissionGet:  func() any { return EmailSubmissionGetResponse{} },
	CommandEmailSubmissionSet:  func() any { return EmailSubmissionSetResponse{} },
	CommandThreadGet:           func() any { return ThreadGetResponse{} },
	CommandIdentityGet:         func() any { return IdentityGetResponse{} },
	CommandVacationResponseGet: func() any { return VacationResponseGetResponse{} },
	CommandVacationResponseSet: func() any { return VacationResponseSetResponse{} },
	CommandSearchSnippetGet:    func() any { return SearchSnippetGetResponse{} },
}
