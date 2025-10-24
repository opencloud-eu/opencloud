package groupware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type IndexLimits struct {
	// The maximum file size, in octets, that the server will accept for a single file upload (for any purpose).
	MaxSizeUpload int `json:"maxSizeUpload"`

	// The maximum number of concurrent requests the server will accept to the upload endpoint.
	MaxConcurrentUpload int `json:"maxConcurrentUpload"`

	// The maximum size, in octets, that the server will accept for a single request to the API endpoint.
	MaxSizeRequest int `json:"maxSizeRequest"`

	// The maximum number of concurrent requests the server will accept to the API endpoint.
	MaxConcurrentRequests int `json:"maxConcurrentRequests"`
}

type IndexAccountMailCapabilities struct {
	// The maximum depth of the Mailbox hierarchy (i.e., one more than the maximum number of ancestors
	// a Mailbox may have), or null for no limit.
	MaxMailboxDepth int `json:"maxMailboxDepth"`

	// The maximum length, in (UTF-8) octets, allowed for the name of a Mailbox.
	//
	// This MUST be at least 100, although it is recommended servers allow more.
	MaxSizeMailboxName int `json:"maxSizeMailboxName"`

	// The maximum number of Mailboxes that can be can assigned to a single Email object.
	//
	// This MUST be an integer >= 1, or null for no limit (or rather, the limit is always the number of
	// Mailboxes in the account).
	MaxMailboxesPerEmail int `json:"maxMailboxesPerEmail"`

	// The maximum total size of attachments, in octets, allowed for a single Email object.
	//
	// A server MAY still reject the import or creation of an Email with a lower attachment size total
	// (for example, if the body includes several megabytes of text, causing the size of the encoded
	// MIME structure to be over some server-defined limit).
	//
	// Note that this limit is for the sum of unencoded attachment sizes. Users are generally not
	// knowledgeable about encoding overhead, etc., nor should they need to be, so marketing and help
	// materials normally tell them the “max size attachments”. This is the unencoded size they see
	// on their hard drive, so this capability matches that and allows the client to consistently
	// enforce what the user understands as the limit.
	MaxSizeAttachmentsPerEmail int `json:"maxSizeAttachmentsPerEmail"`

	// If true, the user may create a Mailbox in this account with a null parentId.
	MayCreateTopLevelMailbox bool `json:"mayCreateTopLevelMailbox"`

	// The number in seconds of the maximum delay the server supports in sending.
	//
	// This is 0 if the server does not support delayed send.
	MaxDelayedSend int `json:"maxDelayedSend"`
}

type IndexAccountSieveCapabilities struct {
	// The maximum length, in octets, allowed for the name of a SieveScript.
	//
	// For compatibility with ManageSieve, this MUST be at least 512 (up
	// to 128 Unicode characters).
	MaxSizeScriptName int `json:"maxSizeScriptName"`

	// The maximum size (in octets) of a Sieve script the server is willing
	// to store for the user, or null for no limit.
	MaxSizeScript int `json:"maxSizeScript"`

	// The maximum number of Sieve scripts the server is willing to store
	// for the user, or null for no limit.
	MaxNumberScripts int `json:"maxNumberScripts"`

	// The maximum number of Sieve "redirect" actions a script can perform
	// during a single evaluation, or null for no limit.
	//
	// Note that this is different from the total number of "redirect"
	// actions a script can contain.
	MaxNumberRedirects int `json:"maxNumberRedirects"`
}

// Capabilities of the Account.
type IndexAccountCapabilities struct {
	Mail  IndexAccountMailCapabilities  `json:"mail"`
	Sieve IndexAccountSieveCapabilities `json:"sieve"`
}

type IndexAccount struct {
	AccountId string `json:"accountId"`

	// A user-friendly string to show when presenting content from this Account,
	// e.g., the email address representing the owner of the account.
	Name string `json:"name"`

	// This is true if the Account belongs to the authenticated user rather than
	// a group Account or a personal Account of another user that has been shared
	// with them.
	IsPersonal bool `json:"isPersonal"`

	// This is true if the entire Account is read-only.
	IsReadOnly bool `json:"isReadOnly"`

	// Capabilities of the Account.
	Capabilities IndexAccountCapabilities `json:"capabilities"`

	// The Identities associated with this Account.
	Identities []jmap.Identity `json:"identities,omitempty"`

	// The quotas for this Account.
	Quotas []jmap.Quota `json:"quotas,omitempty"`
}

// Primary account identifiers per API usage type.
type IndexPrimaryAccounts struct {
	Mail             string `json:"mail"`
	Submission       string `json:"submission"`
	Blob             string `json:"blob"`
	VacationResponse string `json:"vacationResponse"`
	Sieve            string `json:"sieve"`
}

type IndexResponse struct {
	// The API version.
	Version string `json:"version"`

	// A list of capabilities of this API version.
	Capabilities []string `json:"capabilities"`

	// API limits.
	Limits IndexLimits `json:"limits"`

	// Accounts that are available to the user.
	//
	// The key of the map is the Account identifier.
	Accounts []IndexAccount `json:"accounts"`

	// Primary account identifiers per API usage type.
	PrimaryAccounts IndexPrimaryAccounts `json:"primaryAccounts"`
}

// When the request suceeds.
// swagger:response IndexResponse
type SwaggerIndexResponse struct {
	// in: body
	Body struct {
		*IndexResponse
	}
}

// swagger:route GET /groupware bootstrap index
// Get initial bootstrapping information for a user.
//
// responses:
//
//	200: IndexResponse
func (g *Groupware) Index(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountIds := structs.Keys(req.session.Accounts)

		boot, sessionState, lang, err := g.jmap.GetBootstrap(accountIds, req.session, req.ctx, req.logger, req.language())
		if err != nil {
			return req.errorResponseFromJmap(err)
		}

		return response(IndexResponse{
			Version:         Version,
			Capabilities:    Capabilities,
			Limits:          buildIndexLimits(req.session),
			Accounts:        buildIndexAccounts(req.session, boot),
			PrimaryAccounts: buildIndexPrimaryAccounts(req.session),
		}, sessionState, lang)
	})
}

func buildIndexLimits(session *jmap.Session) IndexLimits {
	result := IndexLimits{}
	if core := session.Capabilities.Core; core != nil {
		result.MaxSizeUpload = core.MaxSizeUpload
		result.MaxConcurrentUpload = core.MaxConcurrentUpload
		result.MaxSizeRequest = core.MaxSizeRequest
		result.MaxConcurrentRequests = core.MaxConcurrentRequests
	}
	return result
}

func buildIndexPrimaryAccounts(session *jmap.Session) IndexPrimaryAccounts {
	return IndexPrimaryAccounts{
		Mail:             session.PrimaryAccounts.Mail,
		Submission:       session.PrimaryAccounts.Submission,
		Blob:             session.PrimaryAccounts.Blob,
		VacationResponse: session.PrimaryAccounts.VacationResponse,
		Sieve:            session.PrimaryAccounts.Sieve,
	}
}

func buildIndexAccounts(session *jmap.Session, boot map[string]jmap.AccountBootstrapResult) []IndexAccount {
	accounts := make([]IndexAccount, len(session.Accounts))
	i := 0
	for accountId, account := range session.Accounts {
		indexAccount := IndexAccount{
			AccountId:  accountId,
			Name:       account.Name,
			IsPersonal: account.IsPersonal,
			IsReadOnly: account.IsReadOnly,
			Capabilities: IndexAccountCapabilities{
				Mail:  buildIndexAccountMailCapabilities(account),
				Sieve: buildIndexAccountSieveCapabilities(account),
			},
		}
		if b, ok := boot[accountId]; ok {
			indexAccount.Identities = b.Identities
			indexAccount.Quotas = b.Quotas
		}
		accounts[i] = indexAccount
		i++
	}
	slices.SortFunc(accounts, func(a, b IndexAccount) int { return strings.Compare(a.AccountId, b.AccountId) })
	return accounts
}

func buildIndexAccountMailCapabilities(account jmap.Account) IndexAccountMailCapabilities {
	result := IndexAccountMailCapabilities{}
	if mail := account.AccountCapabilities.Mail; mail != nil {
		result.MaxMailboxDepth = mail.MaxMailboxDepth
		result.MaxSizeMailboxName = mail.MaxSizeMailboxName
		result.MaxMailboxesPerEmail = mail.MaxMailboxesPerEmail
		result.MaxSizeAttachmentsPerEmail = mail.MaxSizeAttachmentsPerEmail
		result.MayCreateTopLevelMailbox = mail.MayCreateTopLevelMailbox
	}
	if subm := account.AccountCapabilities.Submission; subm != nil {
		result.MaxDelayedSend = subm.MaxDelayedSend
	}
	return result
}

func buildIndexAccountSieveCapabilities(account jmap.Account) IndexAccountSieveCapabilities {
	result := IndexAccountSieveCapabilities{}
	if sieve := account.AccountCapabilities.Sieve; sieve != nil {
		result.MaxSizeScriptName = sieve.MaxSizeScriptName
		result.MaxSizeScript = sieve.MaxSizeScript
		result.MaxNumberScripts = sieve.MaxNumberScripts
		result.MaxNumberRedirects = sieve.MaxNumberRedirects
	}
	return result
}
