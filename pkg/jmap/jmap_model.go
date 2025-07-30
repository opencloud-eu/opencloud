package jmap

import (
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
	Name                string                     `json:"name,omitempty"`
	IsPersonal          bool                       `json:"isPersonal"`
	IsReadOnly          bool                       `json:"isReadOnly"`
	AccountCapabilities SessionAccountCapabilities `json:"accountCapabilities,omitempty"`
}

type SessionCoreCapabilities struct {
	MaxSizeUpload         int      `json:"maxSizeUpload"`
	MaxConcurrentUpload   int      `json:"maxConcurrentUpload"`
	MaxSizeRequest        int      `json:"maxSizeRequest"`
	MaxConcurrentRequests int      `json:"maxConcurrentRequests"`
	MaxCallsInRequest     int      `json:"maxCallsInRequest"`
	MaxObjectsInGet       int      `json:"maxObjectsInGet"`
	MaxObjectsInSet       int      `json:"maxObjectsInSet"`
	CollationAlgorithms   []string `json:"collationAlgorithms"`
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
	Capabilities    SessionCapabilities       `json:"capabilities,omitempty"`
	Accounts        map[string]SessionAccount `json:"accounts,omitempty"`
	PrimaryAccounts SessionPrimaryAccounts    `json:"primaryAccounts,omitempty"`
	Username        string                    `json:"username,omitempty"`
	ApiUrl          string                    `json:"apiUrl,omitempty"`
	DownloadUrl     string                    `json:"downloadUrl,omitempty"`
	UploadUrl       string                    `json:"uploadUrl,omitempty"`
	EventSourceUrl  string                    `json:"eventSourceUrl,omitempty"`
	State           string                    `json:"state,omitempty"`
}

type Mailbox struct {
	Id            string          `json:"id,omitempty"`
	Name          string          `json:"name,omitempty"`
	ParentId      string          `json:"parentId,omitempty"`
	Role          string          `json:"role,omitempty"`
	SortOrder     int             `json:"sortOrder,omitempty"`
	IsSubscribed  bool            `json:"isSubscribed,omitempty"`
	TotalEmails   int             `json:"totalEmails,omitempty"`
	UnreadEmails  int             `json:"unreadEmails,omitempty"`
	TotalThreads  int             `json:"totalThreads,omitempty"`
	UnreadThreads int             `json:"unreadThreads,omitempty"`
	MyRights      map[string]bool `json:"myRights,omitempty"`
}

type MailboxGetCommand struct {
	AccountId string   `json:"accountId"`
	Ids       []string `json:"ids,omitempty"`
}

type MailboxGetRefCommand struct {
	AccountId string `json:"accountId"`
	IdRef     *Ref   `json:"#ids,omitempty"`
}

type MailboxFilterCondition struct {
	ParentId     string `json:"parentId,omitempty"`
	Name         string `json:"name,omitempty"`
	Role         string `json:"role,omitempty"`
	HasAnyRole   *bool  `json:"hasAnyRole,omitempty"`
	IsSubscribed *bool  `json:"isSubscribed,omitempty"`
}

type MailboxFilterOperator struct {
	Operator   string                   `json:"operator"`
	Conditions []MailboxFilterCondition `json:"conditions"`
}

type MailboxComparator struct {
	Property       string `json:"property"`
	IsAscending    bool   `json:"isAscending,omitempty"`
	Limit          int    `json:"limit,omitzero"`
	CalculateTotal bool   `json:"calculateTotal,omitempty"`
}

type SimpleMailboxQueryCommand struct {
	AccountId    string                 `json:"accountId"`
	Filter       MailboxFilterCondition `json:"filter,omitempty"`
	Sort         []MailboxComparator    `json:"sort,omitempty"`
	SortAsTree   bool                   `json:"sortAsTree,omitempty"`
	FilterAsTree bool                   `json:"filterAsTree,omitempty"`
}

type MessageFilter struct {
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
}

type Sort struct {
	Property    string `json:"property,omitempty"`
	IsAscending bool   `json:"isAscending,omitempty"`
	Keyword     string `json:"keyword,omitempty"`
	Collation   string `json:"collation,omitempty"`
}

type EmailQueryCommand struct {
	AccountId       string         `json:"accountId"`
	Filter          *MessageFilter `json:"filter,omitempty"`
	Sort            []Sort         `json:"sort,omitempty"`
	CollapseThreads bool           `json:"collapseThreads,omitempty"`
	Position        int            `json:"position,omitempty"`
	Limit           int            `json:"limit,omitempty"`
	CalculateTotal  bool           `json:"calculateTotal,omitempty"`
}

type Ref struct {
	Name     Command `json:"name"`
	Path     string  `json:"path,omitempty"`
	ResultOf string  `json:"resultOf,omitempty"`
}

type EmailGetRefCommand struct {
	AccountId          string `json:"accountId"`
	FetchAllBodyValues bool   `json:"fetchAllBodyValues,omitempty"`
	MaxBodyValueBytes  int    `json:"maxBodyValueBytes,omitempty"`
	IdRef              *Ref   `json:"#ids,omitempty"`
}

type EmailAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type EmailBodyRef struct {
	PartId      string `json:"partId,omitempty"`
	BlobId      string `json:"blobId,omitempty"`
	Size        int    `json:"size,omitempty"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Charset     string `json:"charset,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Cid         string `json:"cid,omitempty"`
	Language    string `json:"language,omitempty"`
	Location    string `json:"location,omitempty"`
}

type EmailBody struct {
	IsEncodingProblem bool   `json:"isEncodingProblem,omitempty"`
	IsTruncated       bool   `json:"isTruncated,omitempty"`
	Value             string `json:"value,omitempty"`
}
type Email struct {
	Id             string               `json:"id,omitempty"`
	MessageId      []string             `json:"messageId,omitempty"`
	BlobId         string               `json:"blobId,omitempty"`
	ThreadId       string               `json:"threadId,omitempty"`
	Size           int                  `json:"size,omitempty"`
	From           []EmailAddress       `json:"from,omitempty"`
	To             []EmailAddress       `json:"to,omitempty"`
	Cc             []EmailAddress       `json:"cc,omitempty"`
	Bcc            []EmailAddress       `json:"bcc,omitempty"`
	ReplyTo        []EmailAddress       `json:"replyTo,omitempty"`
	Subject        string               `json:"subject,omitempty"`
	HasAttachments bool                 `json:"hasAttachments,omitempty"`
	ReceivedAt     time.Time            `json:"receivedAt,omitempty"`
	SentAt         time.Time            `json:"sentAt,omitempty"`
	Preview        string               `json:"preview,omitempty"`
	BodyValues     map[string]EmailBody `json:"bodyValues,omitempty"`
	TextBody       []EmailBodyRef       `json:"textBody,omitempty"`
	HtmlBody       []EmailBodyRef       `json:"htmlBody,omitempty"`
	Keywords       map[string]bool      `json:"keywords,omitempty"`
	MailboxIds     map[string]bool      `json:"mailboxIds,omitempty"`
}

type Command string

const (
	EmailGet            Command = "Email/get"
	EmailQuery          Command = "Email/query"
	EmailSet            Command = "Email/set"
	ThreadGet           Command = "Thread/get"
	MailboxGet          Command = "Mailbox/get"
	MailboxQuery        Command = "Mailbox/query"
	IdentityGet         Command = "Identity/get"
	VacationResponseGet Command = "VacationResponse/get"
)

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
	Using       []string          `json:"using"`
	MethodCalls []Invocation      `json:"methodCalls"`
	CreatedIds  map[string]string `json:"createdIds,omitempty"`
}

func request(methodCalls ...Invocation) (Request, error) {
	return Request{
		Using:       []string{JmapCore, JmapMail},
		MethodCalls: methodCalls,
		CreatedIds:  nil,
	}, nil
}

type Response struct {
	MethodResponses []Invocation      `json:"methodResponses"`
	CreatedIds      map[string]string `json:"createdIds,omitempty"`
	SessionState    string            `json:"sessionState"`
}

type EmailQueryResponse struct {
	AccountId           string   `json:"accountId"`
	QueryState          string   `json:"queryState"`
	CanCalculateChanges bool     `json:"canCalculateChanges"`
	Position            int      `json:"position"`
	Ids                 []string `json:"ids"`
	Total               int      `json:"total"`
}
type EmailGetResponse struct {
	AccountId string  `json:"accountId"`
	State     string  `json:"state"`
	List      []Email `json:"list"`
	NotFound  []any   `json:"notFound"`
}

type MailboxGetResponse struct {
	AccountId string    `json:"accountId"`
	State     string    `json:"state"`
	List      []Mailbox `json:"list"`
	NotFound  []any     `json:"notFound"`
}

type MailboxQueryResponse struct {
	AccountId           string   `json:"accountId"`
	QueryState          string   `json:"queryState"`
	CanCalculateChanges bool     `json:"canCalculateChanges"`
	Position            int      `json:"position"`
	Ids                 []string `json:"ids"`
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
	BodyStructure EmailBodyStructure `json:"bodyStructure,omitempty"`
}

type EmailSetCommand struct {
	AccountId string                 `json:"accountId"`
	Create    map[string]EmailCreate `json:"create,omitempty"`
}

type EmailSetResponse struct {
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

var CommandResponseTypeMap = map[Command]func() any{
	MailboxQuery:        func() any { return MailboxQueryResponse{} },
	MailboxGet:          func() any { return MailboxGetResponse{} },
	EmailQuery:          func() any { return EmailQueryResponse{} },
	EmailGet:            func() any { return EmailGetResponse{} },
	ThreadGet:           func() any { return ThreadGetResponse{} },
	IdentityGet:         func() any { return IdentityGetResponse{} },
	VacationResponseGet: func() any { return VacationResponseGetResponse{} },
}
