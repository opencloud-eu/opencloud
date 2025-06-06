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

type WellKnownAccount struct {
	Name                string         `json:"name,omitempty"`
	IsPersonal          bool           `json:"isPersonal"`
	IsReadOnly          bool           `json:"isReadOnly"`
	AccountCapabilities map[string]any `json:"accountCapabilities,omitempty"`
}

type WellKnownResponse struct {
	Capabilities    map[string]any              `json:"capabilities,omitempty"`
	Accounts        map[string]WellKnownAccount `json:"accounts,omitempty"`
	PrimaryAccounts map[string]string           `json:"primaryAccounts,omitempty"`
	Username        string                      `json:"username,omitempty"`
	ApiUrl          string                      `json:"apiUrl,omitempty"`
	DownloadUrl     string                      `json:"downloadUrl,omitempty"`
	UploadUrl       string                      `json:"uploadUrl,omitempty"`
	EventSourceUrl  string                      `json:"eventSourceUrl,omitempty"`
	State           string                      `json:"state,omitempty"`
}

type Mailbox struct {
	Id            string
	Name          string
	ParentId      string
	Role          string
	SortOrder     int
	IsSubscribed  bool
	TotalEmails   int
	UnreadEmails  int
	TotalThreads  int
	UnreadThreads int
	MyRights      map[string]bool
}

type MailboxGetCommand struct {
	AccountId string `json:"accountId"`
}

type Filter struct {
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
}

type EmailQueryCommand struct {
	AccountId       string  `json:"accountId"`
	Filter          *Filter `json:"filter,omitempty"`
	Sort            []Sort  `json:"sort,omitempty"`
	CollapseThreads bool    `json:"collapseThreads,omitempty"`
	Position        int     `json:"position,omitempty"`
	Limit           int     `json:"limit,omitempty"`
	CalculateTotal  bool    `json:"calculateTotal,omitempty"`
}

type Ref struct {
	Name     Command `json:"name"`
	Path     string  `json:"path,omitempty"`
	ResultOf string  `json:"resultOf,omitempty"`
}

type EmailGetCommand struct {
	AccountId          string `json:"accountId"`
	FetchAllBodyValues bool   `json:"fetchAllBodyValues,omitempty"`
	MaxBodyValueBytes  int    `json:"maxBodyValueBytes,omitempty"`
	IdRef              *Ref   `json:"#ids,omitempty"`
}

type EmailAddress struct {
	Name  string
	Email string
}

type EmailBodyRef struct {
	PartId      string
	BlobId      string
	Size        int
	Name        string
	Type        string
	Charset     string
	Disposition string
	Cid         string
	Language    string
	Location    string
}

type EmailBody struct {
	IsEncodingProblem bool
	IsTruncated       bool
	Value             string
}
type Email struct {
	Id             string
	MessageId      []string
	BlobId         string
	ThreadId       string
	Size           int
	From           []EmailAddress
	To             []EmailAddress
	Cc             []EmailAddress
	Bcc            []EmailAddress
	ReplyTo        []EmailAddress
	Subject        string
	HasAttachments bool
	ReceivedAt     time.Time
	SentAt         time.Time
	Preview        string
	BodyValues     map[string]EmailBody
	TextBody       []EmailBodyRef
	HtmlBody       []EmailBodyRef
	Keywords       map[string]bool
	MailboxIds     map[string]bool
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

func NewInvocation(command Command, parameters any, tag string) Invocation {
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

func NewRequest(methodCalls ...Invocation) (Request, error) {
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
	//BodyStructure map[string]any  `json:"bodyStructure,omitempty"`
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

type VacationResponse struct {
	Id        string    `json:"id"`
	IsEnabled bool      `json:"isEnabled"`
	FromDate  time.Time `json:"fromDate,omitzero"`
	ToDate    time.Time `json:"toDate,omitzero"`
	Subject   string    `json:"subject,omitempty"`
	TextBody  string    `json:"textBody,omitempty"`
	HtmlBody  string    `json:"htmlBody,omitempty"`
}

type VacationResponseGetResponse struct {
	AccountId string             `json:"accountId"`
	State     string             `json:"state,omitempty"`
	List      []VacationResponse `json:"list,omitempty"`
	NotFound  []any              `json:"notFound,omitempty"`
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
