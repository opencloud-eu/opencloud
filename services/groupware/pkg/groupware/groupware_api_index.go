package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type IndexLimits struct {
	// The maximum file size, in octets, that the server will accept for a single file upload (for any purpose).
	MaxSizeUpload         int `json:"maxSizeUpload"`
	MaxConcurrentUpload   int `json:"maxConcurrentUpload"`
	MaxSizeRequest        int `json:"maxSizeRequest"`
	MaxConcurrentRequests int `json:"maxConcurrentRequests"`
}

type IndexAccountMailCapabilities struct {
	MaxMailboxDepth            int  `json:"maxMailboxDepth"`
	MaxSizeMailboxName         int  `json:"maxSizeMailboxName"`
	MaxSizeAttachmentsPerEmail int  `json:"maxSizeAttachmentsPerEmail"`
	MayCreateTopLevelMailbox   bool `json:"mayCreateTopLevelMailbox"`
	MaxDelayedSend             int  `json:"maxDelayedSend"`
}

type IndexAccountSieveCapabilities struct {
	MaxSizeScriptName  int `json:"maxSizeScriptName"`
	MaxSizeScript      int `json:"maxSizeScript"`
	MaxNumberScripts   int `json:"maxNumberScripts"`
	MaxNumberRedirects int `json:"maxNumberRedirects"`
}

type IndexAccountCapabilities struct {
	Mail  IndexAccountMailCapabilities  `json:"mail"`
	Sieve IndexAccountSieveCapabilities `json:"sieve"`
}

type IndexAccount struct {
	Name         string                   `json:"name"`
	IsPersonal   bool                     `json:"isPersonal"`
	IsReadOnly   bool                     `json:"isReadOnly"`
	Capabilities IndexAccountCapabilities `json:"capabilities"`
	Identities   []jmap.Identity          `json:"identities,omitempty"`
}

type IndexPrimaryAccounts struct {
	Mail       string `json:"mail"`
	Submission string `json:"submission"`
}

type IndexResponse struct {
	Version         string                  `json:"version"`
	Capabilities    []string                `json:"capabilities"`
	Limits          IndexLimits             `json:"limits"`
	Accounts        map[string]IndexAccount `json:"accounts"`
	PrimaryAccounts IndexPrimaryAccounts    `json:"primaryAccounts"`
}

// When the request suceeds.
// swagger:response IndexResponse
type SwaggerIndexResponse struct {
	// in: body
	Body struct {
		*IndexResponse
	}
}

// swagger:route GET / index
// Get initial bootup information
//
// responses:
//
//	200: IndexResponse
func (g Groupware) Index(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {

		accountIds := make([]string, len(req.session.Accounts))
		i := 0
		for k := range req.session.Accounts {
			accountIds[i] = k
			i++
		}
		accountIds = structs.Uniq(accountIds)

		identitiesResponse, err := g.jmap.GetIdentities(accountIds, req.session, req.ctx, req.logger)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}

		accounts := make(map[string]IndexAccount, len(req.session.Accounts))
		for accountId, account := range req.session.Accounts {
			indexAccount := IndexAccount{
				Name:       account.Name,
				IsPersonal: account.IsPersonal,
				IsReadOnly: account.IsReadOnly,
				Capabilities: IndexAccountCapabilities{
					Mail: IndexAccountMailCapabilities{
						MaxMailboxDepth:            account.AccountCapabilities.Mail.MaxMailboxDepth,
						MaxSizeMailboxName:         account.AccountCapabilities.Mail.MaxSizeMailboxName,
						MaxSizeAttachmentsPerEmail: account.AccountCapabilities.Mail.MaxSizeAttachmentsPerEmail,
						MayCreateTopLevelMailbox:   account.AccountCapabilities.Mail.MayCreateTopLevelMailbox,
						MaxDelayedSend:             account.AccountCapabilities.Submission.MaxDelayedSend,
					},
					Sieve: IndexAccountSieveCapabilities{
						MaxSizeScriptName:  account.AccountCapabilities.Sieve.MaxSizeScript,
						MaxSizeScript:      account.AccountCapabilities.Sieve.MaxSizeScript,
						MaxNumberScripts:   account.AccountCapabilities.Sieve.MaxNumberScripts,
						MaxNumberRedirects: account.AccountCapabilities.Sieve.MaxNumberRedirects,
					},
				},
			}
			if identity, ok := identitiesResponse.Identities[accountId]; ok {
				indexAccount.Identities = identity
			}
			accounts[accountId] = indexAccount
		}

		return response(IndexResponse{
			Version:      Version,
			Capabilities: Capabilities,
			Limits: IndexLimits{
				MaxSizeUpload:         req.session.Capabilities.Core.MaxSizeUpload,
				MaxConcurrentUpload:   req.session.Capabilities.Core.MaxConcurrentUpload,
				MaxSizeRequest:        req.session.Capabilities.Core.MaxSizeRequest,
				MaxConcurrentRequests: req.session.Capabilities.Core.MaxConcurrentRequests,
			},
			Accounts: accounts,
			PrimaryAccounts: IndexPrimaryAccounts{
				Mail:       req.session.PrimaryAccounts.Mail,
				Submission: req.session.PrimaryAccounts.Submission,
			},
		}, req.session.State)
	})
}
