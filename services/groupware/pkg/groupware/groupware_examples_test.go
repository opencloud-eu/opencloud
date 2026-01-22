//go:build groupware_examples

package groupware

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var (
	exampleQuotaState = "veiv8iez"
)

type Exampler struct{}

var j = jmap.ExamplerInstance

func Example() {
	jmap.SerializeExamples(Exampler{})
	//Output:
}

func (e Exampler) AccountQuota() AccountQuota {
	return AccountQuota{
		Quotas: []jmap.Quota{j.Quota()},
		State:  jmap.State(exampleQuotaState),
	}
}

func (e Exampler) AccountQuotaMap() map[string]AccountQuota {
	return map[string]AccountQuota{
		j.AccountId: e.AccountQuota(),
	}
}

func (e Exampler) AccountWithId() AccountWithId {
	a, _ := j.Account()
	return AccountWithId{
		AccountId: j.AccountId,
		Account:   a,
	}
}

func (e Exampler) AccountWithIdAndIdentities() AccountWithIdAndIdentities {
	a, _ := j.Account()
	return AccountWithIdAndIdentities{
		AccountId:  j.AccountId,
		Account:    a,
		Identities: j.Identities(),
	}
}

func (e Exampler) IndexAccountMailCapabilities() IndexAccountMailCapabilities {
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

func (e Exampler) IndexAccountSieveCapabilities() IndexAccountSieveCapabilities {
	s := j.SessionSieveAccountCapabilities()
	return IndexAccountSieveCapabilities{
		MaxSizeScriptName:  s.MaxSizeScriptName,
		MaxSizeScript:      s.MaxSizeScript,
		MaxNumberScripts:   s.MaxNumberScripts,
		MaxNumberRedirects: s.MaxNumberRedirects,
	}
}

func (e Exampler) IndexAccountCapabilities() IndexAccountCapabilities {
	return IndexAccountCapabilities{
		Mail:  e.IndexAccountMailCapabilities(),
		Sieve: e.IndexAccountSieveCapabilities(),
	}
}

func (e Exampler) IndexAccount() IndexAccount {
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

func (e Exampler) IndexAccounts() []IndexAccount {
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

func (e Exampler) IndexPrimaryAccounts() IndexPrimaryAccounts {
	return IndexPrimaryAccounts{
		Mail:             j.AccountId,
		Submission:       j.AccountId,
		Blob:             j.AccountId,
		VacationResponse: j.AccountId,
		Sieve:            j.AccountId,
	}
}

func (e Exampler) IndexResponse() IndexResponse {
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

func (e Exampler) ErrorResponse() ErrorResponse {
	err := apiError("6d9c65d1-0368-4833-b09f-885aa0171b95", ErrorNoMailboxWithDraftRole)
	return ErrorResponse{
		Errors: []Error{
			*err,
		},
	}
}

func (e Exampler) MailboxesByAccountId() (map[string][]jmap.Mailbox, string) {
	j := jmap.ExamplerInstance
	return map[string][]jmap.Mailbox{
		j.AccountId: j.Mailboxes(),
	}, "All mailboxes for all accounts, without a role filter"
}

func (e Exampler) MailboxesByAccountIdFilteredOnInboxRole() (map[string][]jmap.Mailbox, string, string) {
	j := jmap.ExamplerInstance
	return map[string][]jmap.Mailbox{
		j.AccountId: structs.Filter(j.Mailboxes(), func(m jmap.Mailbox) bool { return m.Role == jmap.JmapMailboxRoleInbox }),
	}, "All mailboxes for all accounts, filtered on the 'inbox' role", "inboxrole"
}

func (e Exampler) EmailSearchResults() EmailSearchResults {
	sent, _ := time.Parse(time.RFC3339, "2026-01-12T21:46:01Z")
	received, _ := time.Parse(time.RFC3339, "2026-01-12T21:47:21Z")
	j := jmap.ExamplerInstance
	return EmailSearchResults{
		Results: []jmap.Email{
			{
				Id:         "ov7ienge",
				BlobId:     "ccyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasiba",
				ThreadId:   "is",
				MailboxIds: map[string]bool{j.MailboxInboxId: true},
				Keywords:   map[string]bool{jmap.JmapKeywordAnswered: true},
				Size:       1084,
				ReceivedAt: received,
				MessageId:  []string{"1768845021.1753110@example.com"},
				Sender: []jmap.EmailAddress{
					{Name: j.SenderName, Email: j.SenderEmailAddress},
				},
				From: []jmap.EmailAddress{
					{Name: j.SenderName, Email: j.SenderEmailAddress},
				},
				To: []jmap.EmailAddress{
					{Name: j.IdentityName, Email: j.EmailAddress},
				},
				Subject: "Remember the Cant",
				SentAt:  sent,
				TextBody: []jmap.EmailBodyPart{
					{PartId: "1", BlobId: "ckyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasibnebdw", Size: 115, Type: "text/plain", Charset: "utf-8"},
				},
				HtmlBody: []jmap.EmailBodyPart{
					{PartId: "2", BlobId: "ckyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasibnsbvjae", Size: 163, Type: "text/html", Charset: "utf-8"},
				},
				Preview: "The Canterbury was destroyed while investigating a false distress call from the Scopuli.",
			},
		},
		Total:      132,
		Limit:      1,
		QueryState: "seehug3p",
	}
}

func (e Exampler) MailboxRolesByAccounts() (map[string][]string, string, string, string) {
	j := jmap.ExamplerInstance
	return map[string][]string{
		j.AccountId:       jmap.JmapMailboxRoles,
		j.SharedAccountId: jmap.JmapMailboxRoles,
	}, "Roles of the Mailboxes of each Account", "", "mailboxrolesbyaccount"
}

func (e Exampler) DeletedMailboxes() ([]string, string, string, string) {
	j := jmap.ExamplerInstance
	return []string{j.MailboxProjectId, j.MailboxJunkId}, "Identifiers of the Mailboxes that have successfully been deleted", "", "deletedmailboxes"
}
