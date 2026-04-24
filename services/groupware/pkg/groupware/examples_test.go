//go:build groupware_examples

package groupware

import (
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var (
	exampleQuotaState = "veiv8iez"
)

type Exemplar struct{}

var j = jmap.ExemplarInstance

func Example() {
	jmap.SerializeExamples(Exemplar{})
	//Output:
}

func (e Exemplar) AccountQuota() AccountQuota {
	return AccountQuota{
		Quotas: []jmap.Quota{j.Quota()},
		State:  jmap.State(exampleQuotaState),
	}
}

func (e Exemplar) AccountQuotaMap() map[string]AccountQuota {
	return map[string]AccountQuota{
		j.AccountId: e.AccountQuota(),
	}
}

func (e Exemplar) AccountWithId() AccountWithId {
	a, _ := j.Account()
	return AccountWithId{
		AccountId: j.AccountId,
		Account:   a,
	}
}

func (e Exemplar) AccountWithIdAndIdentities() AccountWithIdAndIdentities {
	a, _ := j.Account()
	return AccountWithIdAndIdentities{
		AccountId:  j.AccountId,
		Account:    a,
		Identities: j.Identities(),
	}
}

func (e Exemplar) IndexAccountMailCapabilities() IndexAccountMailCapabilities {
	m := j.SessionMailAccountCapabilities()
	s := j.SessionSubmissionAccountCapabilities()
	return IndexAccountMailCapabilities{
		MaxMailboxDepth:            m.MaxMailboxDepth,
		MaxSizeMailboxName:         m.MaxSizeMailboxName,
		MaxMailboxesPerEmail:       m.MaxMailboxesPerEmail,
		MaxSizeAttachmentsPerEmail: m.MaxSizeAttachmentsPerEmail,
		MayCreateTopLevelMailbox:   m.MayCreateTopLevelMailbox,
		MaxDelayedSend:             s.MaxDelayedSend,
	}
}

func (e Exemplar) IndexAccountSieveCapabilities() IndexAccountSieveCapabilities {
	s := j.SessionSieveAccountCapabilities()
	return IndexAccountSieveCapabilities{
		MaxSizeScriptName:  s.MaxSizeScriptName,
		MaxSizeScript:      s.MaxSizeScript,
		MaxNumberScripts:   s.MaxNumberScripts,
		MaxNumberRedirects: s.MaxNumberRedirects,
	}
}

func (e Exemplar) IndexAccountCapabilities() IndexAccountCapabilities {
	return IndexAccountCapabilities{
		Mail:  e.IndexAccountMailCapabilities(),
		Sieve: e.IndexAccountSieveCapabilities(),
	}
}

func (e Exemplar) IndexAccount() IndexAccount {
	a, _ := j.Account()
	return IndexAccount{
		AccountId:    j.AccountId,
		Name:         a.Name,
		IsPersonal:   a.IsPersonal,
		IsReadOnly:   a.IsReadOnly,
		Capabilities: e.IndexAccountCapabilities(),
		Identities:   j.Identities(),
		Quotas:       j.Quotas(),
	}
}

func (e Exemplar) IndexAccounts() []IndexAccount {
	return []IndexAccount{
		e.IndexAccount(),
		{
			AccountId:    j.SharedAccountId,
			Name:         j.SharedAccountName,
			IsPersonal:   false,
			IsReadOnly:   true,
			Capabilities: e.IndexAccountCapabilities(),
			Identities: []jmap.Identity{
				{
					Id:    j.SharedIdentityId,
					Name:  j.SharedIdentityName,
					Email: j.SharedIdentityEmailAddress,
				},
			},
			Quotas: j.Quotas(),
		},
	}
}

func (e Exemplar) IndexPrimaryAccounts() IndexPrimaryAccounts {
	return IndexPrimaryAccounts{
		Mail:             j.AccountId,
		Submission:       j.AccountId,
		Blob:             j.AccountId,
		VacationResponse: j.AccountId,
		Sieve:            j.AccountId,
	}
}

func (e Exemplar) IndexResponse() IndexResponse {
	return IndexResponse{
		Version:      "4.0.0",
		Capabilities: []string{"mail:1"},
		Limits: IndexLimits{
			MaxSizeUpload:         50000000,
			MaxConcurrentUpload:   4,
			MaxSizeRequest:        10000000,
			MaxConcurrentRequests: 4,
		},
		PrimaryAccounts: e.IndexPrimaryAccounts(),
		Accounts:        []IndexAccount{e.IndexAccount()},
	}
}

func (e Exemplar) ErrorResponse() ErrorResponse {
	err := apiError("6d9c65d1-0368-4833-b09f-885aa0171b95", ErrorNoMailboxWithDraftRole)
	return ErrorResponse{
		Errors: []Error{
			*err,
		},
	}
}

func (e Exemplar) MailboxesByAccountId() (map[string][]jmap.Mailbox, string) {
	j := jmap.ExemplarInstance
	return map[string][]jmap.Mailbox{
		j.AccountId: j.Mailboxes(),
	}, "All mailboxes for all accounts, without a role filter"
}

func (e Exemplar) MailboxesByAccountIdFilteredOnInboxRole() (map[string][]jmap.Mailbox, string, string) {
	j := jmap.ExemplarInstance
	return map[string][]jmap.Mailbox{
		j.AccountId: structs.Filter(j.Mailboxes(), func(m jmap.Mailbox) bool { return m.Role == jmap.JmapMailboxRoleInbox }),
	}, "All mailboxes for all accounts, filtered on the 'inbox' role", "inboxrole"
}

func (e Exemplar) MailboxRolesByAccounts() (map[string][]string, string, string, string) {
	j := jmap.ExemplarInstance
	return map[string][]string{
		j.AccountId:       jmap.JmapMailboxRoles,
		j.SharedAccountId: jmap.JmapMailboxRoles,
	}, "Roles of the Mailboxes of each Account", "", "mailboxrolesbyaccount"
}

func (e Exemplar) DeletedMailboxes() ([]string, string, string, string) {
	j := jmap.ExemplarInstance
	return []string{j.MailboxProjectId, j.MailboxJunkId}, "Identifiers of the Mailboxes that have successfully been deleted", "", "deletedmailboxes"
}

func (e Exemplar) ObjectsRequest() ObjectsRequest {
	return ObjectsRequest{
		Mailboxes:        []string{"ahh9ye", "ahbei8"},
		Emails:           []string{"koo6ka", "fa1ees", "zaish0", "iek2fo"},
		Addressbooks:     []string{"ungu0a"},
		Contacts:         []string{"oo8ahv", "lexue6", "mohth3"},
		Calendars:        []string{"aa8aqu", "detho5"},
		Events:           []string{"oo8thu", "mu9sha", "aim1sh", "sair6a"},
		Quotas:           []string{"vei4ai"},
		Identities:       []string{"iuj4ae", "mahv9y"},
		EmailSubmissions: []string{"eidoo6", "aakie7", "uh7ous"},
	}
}
