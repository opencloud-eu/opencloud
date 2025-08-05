package jmap

import (
	"io"
	"time"
)

const (
	JmapCore             = "urn:ietf:params:jmap:core"
	JmapMail             = "urn:ietf:params:jmap:mail"
	JmapMDN              = "urn:ietf:params:jmap:mdn" // https://datatracker.ietf.org/doc/rfc9007/
	JmapSubmission       = "urn:ietf:params:jmap:submission"
	JmapVacationResponse = "urn:ietf:params:jmap:vacationresponse"
	JmapCalendars        = "urn:ietf:params:jmap:calendars"
	JmapSieve            = "urn:ietf:params:jmap:sieve"
	JmapBlob             = "urn:ietf:params:jmap:blob"
	JmapQuota            = "urn:ietf:params:jmap:quota"
	JmapWebsocket        = "urn:ietf:params:jmap:websocket"

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
)

type SessionMailAccountCapabilities struct {
	MaxMailboxesPerEmail       int      `json:"maxMailboxesPerEmail"`
	MaxMailboxDepth            int      `json:"maxMailboxDepth"`
	MaxSizeMailboxName         int      `json:"maxSizeMailboxName"`
	MaxSizeAttachmentsPerEmail int      `json:"maxSizeAttachmentsPerEmail"`
	EmailQuerySortOptions      []string `json:"emailQuerySortOptions"`
	MayCreateTopLevelMailbox   bool     `json:"mayCreateTopLevelMailbox"`
}

type SessionSubmissionAccountCapabilities struct {
	MaxDelayedSend       int                 `json:"maxDelayedSend"`
	SubmissionExtensions map[string][]string `json:"submissionExtensions"`
}

type SessionVacationResponseAccountCapabilities struct {
}

type SessionSieveAccountCapabilities struct {
	MaxSizeScriptName   int      `json:"maxSizeScriptName"`
	MaxSizeScript       int      `json:"maxSizeScript"`
	MaxNumberScripts    int      `json:"maxNumberScripts"`
	MaxNumberRedirects  int      `json:"maxNumberRedirects"`
	SieveExtensions     []string `json:"sieveExtensions"`
	NotificationMethods []string `json:"notificationMethods"`
	ExternalLists       any      `json:"externalLists"` // ?
}

type SessionBlobAccountCapabilities struct {
	MaxSizeBlobSet            int      `json:"maxSizeBlobSet"`
	MaxDataSources            int      `json:"maxDataSources"`
	SupportedTypeNames        []string `json:"supportedTypeNames"`
	SupportedDigestAlgorithms []string `json:"supportedDigestAlgorithms"`
}

type SessionQuotaAccountCapabilities struct {
}

type SessionAccountCapabilities struct {
	Mail             SessionMailAccountCapabilities             `json:"urn:ietf:params:jmap:mail"`
	Submission       SessionSubmissionAccountCapabilities       `json:"urn:ietf:params:jmap:submission"`
	VacationResponse SessionVacationResponseAccountCapabilities `json:"urn:ietf:params:jmap:vacationresponse"`
	Sieve            SessionSieveAccountCapabilities            `json:"urn:ietf:params:jmap:sieve"`
	Blob             SessionBlobAccountCapabilities             `json:"urn:ietf:params:jmap:blob"`
	Quota            SessionQuotaAccountCapabilities            `json:"urn:ietf:params:jmap:quota"`
}

type SessionAccount struct {
	// A user-friendly string to show when presenting content from this account, e.g., the email address representing the owner of the account.
	Name string `json:"name,omitempty"`
	// This is true if the account belongs to the authenticated user rather than a group account or a personal account of another user that has been shared with them.
	IsPersonal bool `json:"isPersonal"`
	// This is true if the entire account is read-only.
	IsReadOnly          bool                       `json:"isReadOnly"`
	AccountCapabilities SessionAccountCapabilities `json:"accountCapabilities,omitempty"`
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
	Url          string `json:"url"`
	SupportsPush bool   `json:"supportsPush"`
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

type SessionResponse struct {
	Capabilities SessionCapabilities       `json:"capabilities"`
	Accounts     map[string]SessionAccount `json:"accounts,omitempty"`
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
	State string `json:"state,omitempty"`
}

// SetError type values.
const (
	// (create; update; destroy). The create/update/destroy would violate an ACL or other permissions policy.
	SetErrorTypeForbidden = "forbidden"

	// (create; update). The create would exceed a server-defined limit on the number or total size of objects of this type.
	SetErrorTypeOverQuota = "overQuota"

	// (create; update). The create/update would result in an object that exceeds a server-defined limit for the maximum
	// size of a single object of this type.
	SetErrorTypeTooLarge = "tooLarge"

	// (create). Too many objects of this type have been created recently, and a server-defined rate limit has been reached.
	// It may work if tried again later.
	SetErrorTypeRateLimit = "rateLimit"

	// (update; destroy). The id given to update/destroy cannot be found.
	SetErrorTypeNotFound = "notFound"

	// (update) The PatchObject given to update the record was not a valid patch (see the patch description).
	SetErrorTypeInvalidPatch = "invalidPatch"

	// (update). The client requested that an object be both updated and destroyed in the same /set request, and the server
	// has decided to therefore ignore the update.
	SetErrorTypeWillDestroy = "willDestroy"

	// (create; update). The record given is invalid in some way. For example:
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
	SetErrorTypeInvalidProperties = "invalidProperties"

	// (create; destroy). This is a singleton type, so you cannot create another one or destroy the existing one.
	SetErrorTypeSingleton = "singleton"

	// The total number of objects to create, update, or destroy exceeds the maximum number the server is
	// willing to process in a single method call.
	SetErrorTypeRequestTooLarge = "requestTooLarge"

	// An ifInState argument was supplied, and it does not match the current state.
	SetErrorTypeStateMismatch = "stateMismatch"
)

type SetError struct {
	// The type of error.
	Type string `json:"type"`

	// A description of the error to help with debugging that includes an explanation of what the problem was.
	//
	// This is a non-localised string and is not intended to be shown directly to end users.
	Description string `json:"description,omitempty"`
}

type FilterOperatorTerm string

const (
	And FilterOperatorTerm = "AND"
	Or  FilterOperatorTerm = "OR"
	Not FilterOperatorTerm = "NOT"
)

type Mailbox struct {
	Id            string          `json:"id,omitempty"`
	Name          string          `json:"name,omitempty"`
	ParentId      string          `json:"parentId,omitempty"`
	Role          string          `json:"role,omitempty"`
	SortOrder     int             `json:"sortOrder"`
	IsSubscribed  bool            `json:"isSubscribed"`
	TotalEmails   int             `json:"totalEmails"`
	UnreadEmails  int             `json:"unreadEmails"`
	TotalThreads  int             `json:"totalThreads"`
	UnreadThreads int             `json:"unreadThreads"`
	MyRights      map[string]bool `json:"myRights,omitempty"`
}

type MailboxGetCommand struct {
	AccountId string   `json:"accountId"`
	Ids       []string `json:"ids,omitempty"`
}

type MailboxGetRefCommand struct {
	AccountId string           `json:"accountId"`
	IdRef     *ResultReference `json:"#ids,omitempty"`
}

type MailboxChangesCommand struct {
	AccountId  string `json:"accountId"`
	SinceState string `json:"sinceState,omitempty"`
	MaxChanges int    `json:"maxChanges,omitzero"`
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
}

type EmailFilterCondition struct {
	EmailFilterElement
	InMailbox               string    `json:"inMailbox,omitempty"`
	InMailboxOtherThan      []string  `json:"inMailboxOtherThan,omitempty"`
	Before                  time.Time `json:"before,omitzero"` // omitzero requires Go 1.24
	After                   time.Time `json:"after,omitzero"`
	MinSize                 int       `json:"minSize,omitempty"`
	MaxSize                 int       `json:"maxSize,omitempty"`
	AllInThreadHaveKeyword  string    `json:"allInThreadHaveKeyword,omitempty"`
	SomeInThreadHaveKeyword string    `json:"someInThreadHaveKeyword,omitempty"`
	NoneInThreadHaveKeyword string    `json:"noneInThreadHaveKeyword,omitempty"`
	HasKeyword              string    `json:"hasKeyword,omitempty"`
	NotKeyword              string    `json:"notKeyword,omitempty"`
	HasAttachment           bool      `json:"hasAttachment,omitempty"`
	Text                    string    `json:"text,omitempty"`
	From                    string    `json:"from,omitempty"`
	To                      string    `json:"to,omitempty"`
	Cc                      string    `json:"cc,omitempty"`
	Bcc                     string    `json:"bcc,omitempty"`
	Subject                 string    `json:"subject,omitempty"`
	Body                    string    `json:"body,omitempty"`
	Header                  []string  `json:"header,omitempty"`
}

var _ EmailFilterElement = &EmailFilterCondition{}

type EmailFilterOperator struct {
	EmailFilterElement
	Operator   FilterOperatorTerm   `json:"operator"`
	Conditions []EmailFilterElement `json:"conditions,omitempty"`
}

var _ EmailFilterElement = &EmailFilterOperator{}

type Sort struct {
	Property    string `json:"property,omitempty"`
	IsAscending bool   `json:"isAscending,omitempty"`
	Keyword     string `json:"keyword,omitempty"`
	Collation   string `json:"collation,omitempty"`
}

type EmailQueryCommand struct {
	AccountId       string             `json:"accountId"`
	Filter          EmailFilterElement `json:"filter,omitempty"`
	Sort            []Sort             `json:"sort,omitempty"`
	CollapseThreads bool               `json:"collapseThreads,omitempty"`
	Position        int                `json:"position,omitempty"`
	Limit           int                `json:"limit,omitempty"`
	CalculateTotal  bool               `json:"calculateTotal,omitempty"`
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

	// (default: 0) If greater than zero, the value property of any EmailBodyValue object returned in bodyValues
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
	MaxBodyValueBytes int `json:"maxBodyValueBytes,omitempty"`
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
	IdRef *ResultReference `json:"#ids,omitempty"`

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

	// (default: 0) If greater than zero, the value property of any EmailBodyValue object returned in bodyValues
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
	MaxBodyValueBytes int `json:"maxBodyValueBytes,omitempty"`
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
	MaxChanges int `json:"maxChanges,omitzero"`
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

type EmailBodyPart struct {
	// Identifies this part uniquely within the Email.
	//
	// This is scoped to the emailId and has no meaning outside of the JMAP Email object representation.
	// This is null if, and only if, the part is of type multipart/*.
	PartId string `json:"partId,omitempty"`

	// The id representing the raw octets of the contents of the part, after decoding any known
	// Content-Transfer-Encoding (as defined in [RFC2045]), or null if, and only if, the part is of type multipart/*.
	//
	// Note that two parts may be transfer-encoded differently but have the same blob id if their decoded octets are identical
	// and the server is using a secure hash of the data for the blob id.
	// If the transfer encoding is unknown, it is treated as though it had no transfer encoding.
	//
	// [RFC2045]: https://www.rfc-editor.org/rfc/rfc2045.html
	BlobId string `json:"blobId,omitempty"`

	// The size, in octets, of the raw data after content transfer decoding (as referenced by the blobId, i.e.,
	// the number of octets in the file the user would download).
	Size int `json:"size,omitempty"`

	// This is a list of all header fields in the part, in the order they appear in the message.
	//
	// The values are in Raw form.
	Headers []EmailHeader `json:"headers,omitempty"`

	// This is the decoded filename parameter of the Content-Disposition header field per [RFC2231], or
	// (for compatibility with existing systems).
	//
	// If not present, then it’s the decoded name parameter of the Content-Type header field per [RFC2047].
	//
	// [RFC2231]: https://www.rfc-editor.org/rfc/rfc2231.html
	// [RFC2047]: https://www.rfc-editor.org/rfc/rfc2047.html
	Name string `json:"name,omitempty"`

	// The value of the Content-Type header field of the part, if present; otherwise, the implicit type as per
	// the MIME standard (text/plain or message/rfc822 if inside a multipart/digest).
	//
	// CFWS is removed and any parameters are stripped.
	Type string `json:"type,omitempty"`

	// The value of the charset parameter of the Content-Type header field, if present, or null if the header
	// field is present but not of type text/*.
	//
	// If there is no Content-Type header field, or it exists and is of type text/* but has no charset parameter,
	// this is the implicit charset as per the MIME standard: us-ascii.
	Charset string `json:"charset,omitempty"`

	// The value of the Content-Disposition header field of the part, if present;
	// otherwise, it’s null.
	//
	// CFWS is removed and any parameters are stripped.
	Disposition string `json:"disposition,omitempty"`

	// The value of the Content-Id header field of the part, if present; otherwise it’s null.
	//
	// CFWS and surrounding angle brackets (<>) are removed.
	// This may be used to reference the content from within a text/html body part HTML using the cid: protocol, as defined in [RFC2392].
	//
	// [RFC2392]: https://www.rfc-editor.org/rfc/rfc2392.html
	Cid string `json:"cid,omitempty"`

	// The list of language tags, as defined in [RFC3282], in the Content-Language header field of the part, if present.
	//
	// [RFC3282]: https://www.rfc-editor.org/rfc/rfc3282.html
	Language string `json:"language,omitempty"`

	// The URI, as defined in [RFC2557], in the Content-Location header field of the part, if present.
	//
	// [RFC2557]: https://www.rfc-editor.org/rfc/rfc2557.html
	Location string `json:"location,omitempty"`

	// If the type is multipart/*, this contains the body parts of each child.
	SubParts []EmailBodyPart `json:"subParts,omitempty"`
}

type EmailBodyValue struct {
	// The value of the body part after decoding Content-Transfer-Encoding and the Content-Type charset,
	// if both known to the server, and with any CRLF replaced with a single LF.
	//
	// The server MAY use heuristics to determine the charset to use for decoding if the charset is unknown,
	// no charset is given, or it believes the charset given is incorrect.
	//
	// Decoding is best effort; the server SHOULD insert the unicode replacement character (U+FFFD) and continue
	// when a malformed section is encountered.
	//
	// Note that due to the charset decoding and line ending normalisation, the length of this string will
	// probably not be exactly the same as the size property on the corresponding EmailBodyPart.
	Value string `json:"value,omitempty"`

	// (default: false) This is true if malformed sections were found while decoding the charset,
	// or the charset was unknown, or the content-transfer-encoding was unknown.
	IsEncodingProblem bool `json:"isEncodingProblem,omitzero"`

	// (default: false) This is true if the value has been truncated.
	IsTruncated bool `json:"isTruncated,omitzero"`
}

type Email struct {
	// The id of the Email object.
	//
	// Note that this is the JMAP object id, NOT the Message-ID header field value of the message [RFC5322].
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	Id string `json:"id,omitempty"`

	// The id representing the raw octets of the message [RFC5322] for this Email.
	//
	// This may be used to download the raw original message or to attach it directly to another Email, etc.
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	BlobId string `json:"blobId,omitempty"`

	// The id of the Thread to which this Email belongs.
	ThreadId string `json:"threadId,omitempty"`

	// The set of Mailbox ids this Email belongs to.
	//
	// An Email in the mail store MUST belong to one or more Mailboxes at all times (until it is destroyed).
	// The set is represented as an object, with each key being a Mailbox id.
	//
	// The value for each key in the object MUST be true.
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
	// The client MAY include capability identifiers even if the method calls it makes do not utilise those capabilities.
	// The server advertises the set of specifications it supports in the Session object (see [Section 2]), as keys on the capabilities property.
	//
	// [Section 2]: https://jmap.io/spec-core.html#the-jmap-session-resource
	Using []string `json:"using"`
	// An array of method calls to process on the server.
	// The method calls MUST be processed sequentially, in order.
	MethodCalls []Invocation `json:"methodCalls"`
	// (optional) A map of a (client-specified) creation id to the id the server assigned when a record was successfully created.
	CreatedIds map[string]string `json:"createdIds,omitempty"`
}

func request(methodCalls ...Invocation) (Request, error) {
	return Request{
		Using:       []string{JmapCore, JmapMail},
		MethodCalls: methodCalls,
		CreatedIds:  nil,
	}, nil
}

type Response struct {
	// An array of responses, in the same format as the methodCalls on the Request object.
	// The output of the methods MUST be added to the methodResponses array in the same order that the methods are processed.
	MethodResponses []Invocation `json:"methodResponses"`
	// (optional; only returned if given in the request) A map of a (client-specified) creation id to the id the server
	// assigned when a record was successfully created.
	// This MUST include all creation ids passed in the original createdIds parameter of the Request object, as well as any
	// additional ones added for newly created records.
	CreatedIds map[string]string `json:"createdIds,omitempty"`
	// The current value of the “state” string on the Session object, as described in [Section 2].
	// Clients may use this to detect if this object has changed and needs to be refetched.
	//
	// [Section 2]: https://jmap.io/spec-core.html#the-jmap-session-resource
	SessionState string `json:"sessionState"`
}

type EmailQueryResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`
	// A string encoding the current state of the query on the server.
	// This string MUST change if the results of the query (i.e., the matching ids and their sort order) have changed.
	// The queryState string MAY change if something has changed on the server, which means the results may have changed
	// but the server doesn’t know for sure.
	// The queryState string only represents the ordered list of ids that match the particular query (including its sort/filter).
	// There is no requirement for it to change if a property on an object matching the query changes but the query results are unaffected
	// (indeed, it is more efficient if the queryState string does not change in this case).
	// The queryState string only has meaning when compared to future responses to a query with the same type/sort/filter or when used with
	// /queryChanges to fetch changes.
	// Should a client receive back a response with a different queryState string to a previous call, it MUST either throw away the currently
	// cached query and fetch it again (note, this does not require fetching the records again, just the list of ids) or call
	// Email/queryChanges to get the difference.
	QueryState string `json:"queryState"`
	// This is true if the server supports calling Email/queryChanges with these filter/sort parameters.
	// Note, this does not guarantee that the Email/queryChanges call will succeed, as it may only be possible for a limited time
	// afterwards due to server internal implementation details.
	CanCalculateChanges bool `json:"canCalculateChanges"`
	// The zero-based index of the first result in the ids array within the complete list of query results.
	Position int `json:"position"`
	// The list of ids for each Email in the query results, starting at the index given by the position argument of this
	// response and continuing until it hits the end of the results or reaches the limit number of ids.
	// If position is >= total, this MUST be the empty list.
	Ids []string `json:"ids"`
	// (only if requested) The total number of Emails in the results (given the filter).
	// This argument MUST be omitted if the calculateTotal request argument is not true.
	Total int `json:"total,omitempty,omitzero"`
	// (if set by the server) The limit enforced by the server on the maximum number of results to return.
	// This is only returned if the server set a limit or used a different limit than that given in the request.
	Limit int `json:"limit,omitempty,omitzero"`
}

type EmailGetResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`
	// A (preferably short) string representing the state on the server for all the data of this type
	// in the account (not just the objects returned in this call).
	// If the data changes, this string MUST change.
	// If the Email data is unchanged, servers SHOULD return the same state string on subsequent requests for this data type.
	State string `json:"state"`
	// An array of the Email objects requested.
	// This is the empty array if no objects were found or if the ids argument passed in was also an empty array.
	// The results MAY be in a different order to the ids in the request arguments.
	// If an identical id is included more than once in the request, the server MUST only include it once in either
	// the list or the notFound argument of the response.
	List []Email `json:"list"`
	// This array contains the ids passed to the method for records that do not exist.
	// The array is empty if all requested ids were found or if the ids argument passed in was either null or an empty array.
	NotFound []any `json:"notFound"`
}

type EmailChangesResponse struct {
	// The id of the account used for the call.
	AccountId string `json:"accountId"`
	// This is the sinceState argument echoed back; it’s the state from which the server is returning changes.
	OldState string `json:"oldState"`
	// This is the state the client will be in after applying the set of changes to the old state.
	NewState string `json:"newState"`
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
	State string `json:"state"`
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
	OldState string `json:"oldState"`

	// This is the state the client will be in after applying the set of changes to the old state.
	NewState string `json:"newState"`

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
	QueryState string `json:"queryState"`

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

	// (only if requested) The total number of Mailbox in the results (given the filter).
	//
	// This argument MUST be omitted if the calculateTotal request argument is not true.
	Total int `json:"total,omitzero"`

	// (if set by the server) The limit enforced by the server on the maximum number of results to return.
	//
	// This is only returned if the server set a limit or used a different limit than that given in the request.
	Limit int `json:"limit,omitzero"`
}

type EmailBodyStructure struct {
	Type   string         //`json:"type"`
	PartId string         //`json:"partId"`
	Other  map[string]any `mapstructure:",remain"`
}

type EmailCreate struct {
	MailboxIds    map[string]bool    `json:"mailboxIds,omitempty"`
	Keywords      map[string]bool    `json:"keywords,omitempty"`
	From          []EmailAddress     `json:"from,omitempty"`
	Subject       string             `json:"subject,omitempty"`
	ReceivedAt    time.Time          `json:"receivedAt,omitzero"`
	SentAt        time.Time          `json:"sentAt,omitzero"`
	BodyStructure EmailBodyStructure `json:"bodyStructure"`
}

type EmailSetCommand struct {
	AccountId string                 `json:"accountId"`
	Create    map[string]EmailCreate `json:"create,omitempty"`
}

type EmailSetResponse struct {
}

const (
	EmailMimeType = "message/rfc822"
)

type EmailToImport struct {
	BlobId     string          `json:"blobId"`
	MailboxIds map[string]bool `json:"mailboxIds"`
	Keywords   map[string]bool `json:"keywords"`
	ReceivedAt time.Time       `json:"receivedAt"`
}

type EmailImportCommand struct {
	AccountId string                   `json:"accountId"`
	IfInState string                   `json:"ifInState,omitempty"`
	Emails    map[string]EmailToImport `json:"emails"`
}

type ImportedEmail struct {
	Id       string `json:"id"`
	BlobId   string `json:"blobId"`
	ThreadId string `json:"threadId"`
	Size     int    `json:"size"`
}

type EmailImportResponse struct {
	AccountId  string                   `json:"accountId"`
	OldState   string                   `json:"oldState"`
	NewState   string                   `json:"newState"`
	Created    map[string]ImportedEmail `json:"created"`
	NotCreated map[string]SetError      `json:"notCreated"`
}

type Thread struct {
	Id       string
	EmailIds []string
}

type ThreadGetResponse struct {
	AccountId string
	State     string
	List      []Thread
	NotFound  []any
}

type IdentityGetCommand struct {
	AccountId string   `json:"accountId"`
	Ids       []string `json:"ids,omitempty"`
}

type Identity struct {
	Id            string         `json:"id"`
	Name          string         `json:"name,omitempty"`
	Email         string         `json:"email,omitempty"`
	ReplyTo       string         `json:"replyTo:omitempty"`
	Bcc           []EmailAddress `json:"bcc,omitempty"`
	TextSignature string         `json:"textSignature,omitempty"`
	HtmlSignature string         `json:"htmlSignature,omitempty"`
	MayDelete     bool           `json:"mayDelete"`
}

type IdentityGetResponse struct {
	AccountId string     `json:"accountId"`
	State     string     `json:"state"`
	List      []Identity `json:"list,omitempty"`
	NotFound  []any      `json:"notFound,omitempty"`
}

type VacationResponseGetCommand struct {
	AccountId string `json:"accountId"`
}

// https://datatracker.ietf.org/doc/html/rfc8621#section-8
type VacationResponse struct {
	// The id of the object.
	// There is only ever one VacationResponse object, and its id is "singleton"
	Id string `json:"id"`
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
	// If the data changes, this string MUST change. If the data is unchanged, servers SHOULD return the same state string
	// on subsequent requests for this data type.
	State string `json:"state,omitempty"`
	// An array of VacationResponse objects.
	List []VacationResponse `json:"list,omitempty"`
	// Contains identifiers of requested objects that were not found.
	NotFound []any `json:"notFound,omitempty"`
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
	BlobPropertyData         = "data"
	BlobPropertySize         = "size"
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
	Id                string `json:"id"`
	DataAsText        string `json:"data:asText,omitempty"`
	DataAsBase64      string `json:"data:asBase64,omitempty"`
	DigestSha256      string `json:"digest:sha256,omitempty"`
	DigestSha512      string `json:"digest:sha512,omitempty"`
	IsEncodingProblem bool   `json:"isEncodingProblem,omitzero"`
	IsTruncated       bool   `json:"isTruncated,omitzero"`
	Size              int    `json:"size"`
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
	AccountId string `json:"accountId"`
	State     string `json:"state,omitempty"`
	List      []Blob `json:"list,omitempty"`
	NotFound  []any  `json:"notFound,omitempty"`
}

type BlobDownload struct {
	Body               io.ReadCloser
	Size               int
	Type               string
	ContentDisposition string
	CacheControl       string
}

type SearchSnippet struct {
	EmailId string `json:"emailId"`
	Subject string `json:"subject,omitempty"`
	Preview string `json:"preview,omitempty"`
}

type SearchSnippetRefCommand struct {
	AccountId  string             `json:"accountId"`
	Filter     EmailFilterElement `json:"filter,omitempty"`
	EmailIdRef *ResultReference   `json:"#emailIds,omitempty"`
}

type SearchSnippetGetResponse struct {
	AccountId string          `json:"accountId"`
	List      []SearchSnippet `json:"list,omitempty"`
	NotFound  []string        `json:"notFound,omitempty"`
}

const (
	BlobGet             Command = "Blob/get"
	BlobUpload          Command = "Blob/upload"
	EmailGet            Command = "Email/get"
	EmailQuery          Command = "Email/query"
	EmailChanges        Command = "Email/changes"
	EmailSet            Command = "Email/set"
	EmailImport         Command = "Email/import"
	ThreadGet           Command = "Thread/get"
	MailboxGet          Command = "Mailbox/get"
	MailboxQuery        Command = "Mailbox/query"
	MailboxChanges      Command = "Mailbox/changes"
	IdentityGet         Command = "Identity/get"
	VacationResponseGet Command = "VacationResponse/get"
	SearchSnippetGet    Command = "SearchSnippet/get"
)

var CommandResponseTypeMap = map[Command]func() any{
	BlobGet:             func() any { return BlobGetResponse{} },
	BlobUpload:          func() any { return BlobUploadResponse{} },
	MailboxQuery:        func() any { return MailboxQueryResponse{} },
	MailboxGet:          func() any { return MailboxGetResponse{} },
	MailboxChanges:      func() any { return MailboxChangesResponse{} },
	EmailQuery:          func() any { return EmailQueryResponse{} },
	EmailChanges:        func() any { return EmailChangesResponse{} },
	EmailGet:            func() any { return EmailGetResponse{} },
	ThreadGet:           func() any { return ThreadGetResponse{} },
	IdentityGet:         func() any { return IdentityGetResponse{} },
	VacationResponseGet: func() any { return VacationResponseGetResponse{} },
	SearchSnippetGet:    func() any { return SearchSnippetGetResponse{} },
}
