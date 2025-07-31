package groupware

import (
	"net/http"
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
	g.respond(w, r, func(req Request) (any, string, *Error) {
		accounts := make(map[string]IndexAccount, len(req.session.Accounts))
		for i, a := range req.session.Accounts {
			accounts[i] = IndexAccount{
				Name:       a.Name,
				IsPersonal: a.IsPersonal,
				IsReadOnly: a.IsReadOnly,
				Capabilities: IndexAccountCapabilities{
					Mail: IndexAccountMailCapabilities{
						MaxMailboxDepth:            a.AccountCapabilities.Mail.MaxMailboxDepth,
						MaxSizeMailboxName:         a.AccountCapabilities.Mail.MaxSizeMailboxName,
						MaxSizeAttachmentsPerEmail: a.AccountCapabilities.Mail.MaxSizeAttachmentsPerEmail,
						MayCreateTopLevelMailbox:   a.AccountCapabilities.Mail.MayCreateTopLevelMailbox,
						MaxDelayedSend:             a.AccountCapabilities.Submission.MaxDelayedSend,
					},
					Sieve: IndexAccountSieveCapabilities{
						MaxSizeScriptName:  a.AccountCapabilities.Sieve.MaxSizeScript,
						MaxSizeScript:      a.AccountCapabilities.Sieve.MaxSizeScript,
						MaxNumberScripts:   a.AccountCapabilities.Sieve.MaxNumberScripts,
						MaxNumberRedirects: a.AccountCapabilities.Sieve.MaxNumberRedirects,
					},
				},
			}
		}

		return IndexResponse{
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
		}, req.session.State, nil
	})
}
