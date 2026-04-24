//go:build groupware_examples

package jmap

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
	c "github.com/opencloud-eu/opencloud/pkg/jscontact"
)

func SerializeExamples(e any) { //NOSONAR
	type example struct {
		Type    string `json:"type"`
		Key     string `json:"key,omitempty"`
		Title   string `json:"title,omitempty"`
		Scope   string `json:"scope,omitempty"`
		Origin  string `json:"origin,omitempty"`
		Example any    `json:"example"`
	}

	filename := os.Getenv("EXAMPLE_OUTPUT_FILE")
	if filename == "" {
		filename = "apidoc-examples.json"
	}

	funcs := map[string]func() (example, error){}
	reflected := reflect.ValueOf(e)
	r := reflect.TypeOf(e)
	conflicts := map[string]example{}
	for i := 0; i < r.NumMethod(); i++ {
		name := r.Method(i).Name
		m := reflected.MethodByName(name)
		funcs[name] = func() (example, error) {
			results := m.Call(nil)
			title := ""
			key := "default"
			typ := ""
			uniqueId := ""
			var result reflect.Value
			switch len(results) {
			case 1:
				result = results[0]
			case 2:
				result = results[0]
				title = results[1].String()
			case 3:
				result = results[0]
				title = results[1].String()
				key = results[2].String()
			case 4:
				result = results[0]
				title = results[1].String()
				key = results[2].String()
				typ = results[3].String()
			default:
				return example{}, fmt.Errorf("method result does not have 1 or 2 or 3 or 4 results but %d", len(results))
			}
			t := result.Type()
			scope := "" // same as "any"
			if strings.HasSuffix(name, "_req") {
				scope = "request"
			}
			origin := fmt.Sprintf("%s:%s", r.String(), name)

			if typ == "" {
				typ = t.String()
			}

			if uniqueId == "" {
				uniqueId = typ + "." + key
			}
			conflictKey := uniqueId + "/" + scope
			if conflict, ok := conflicts[conflictKey]; ok {
				panic(fmt.Errorf("conflicting examples with the same unique identifier '%s', consider adding a key to either of '%s' or '%s'", conflictKey, conflict.Origin, origin))
			}

			ex := example{
				Type:    typ,
				Key:     uniqueId,
				Title:   title,
				Scope:   scope,
				Origin:  origin,
				Example: result.Interface(),
			}
			conflicts[conflictKey] = ex
			return ex, nil
		}
	}

	examples := []example{}
	for name, f := range funcs {
		if ex, err := f(); err != nil {
			panic(fmt.Errorf("the example producing method '%s' produced an error: %w", name, err))
		} else {
			examples = append(examples, ex)
		}
	}
	if b, err := json.MarshalIndent(examples, "", "  "); err != nil {
		panic(fmt.Errorf("failed to serialize to JSON: %w", err))
	} else {
		if err := os.WriteFile(filename, b, 0644); err != nil {
			panic(fmt.Errorf("failed to write the serialized JSON output to the file '%s': %w", filename, err))
		}
	}
}

type Exemplar struct {
	AccountId                  string
	SharedAccountId            string
	IdentityId                 string
	IdentityName               string
	EmailAddress               string
	BccName                    string
	BccAddress                 string
	OtherIdentityId            string
	OtherIdentityName          string
	OtherIdentityEmailAddress  string
	SharedIdentityId           string
	SharedIdentityName         string
	SharedIdentityEmailAddress string
	ThreadId                   string
	EmailId                    string
	EmailIds                   []string
	QuotaId                    string
	SharedQuotaId              string
	Username                   string
	SharedAccountName          string
	TextSignature              string
	MailboxInboxId             string
	MailboxDraftsId            string
	MailboxSentId              string
	MailboxJunkId              string
	MailboxDeletedId           string
	MailboxProjectId           string
	SenderEmailAddress         string
	SenderName                 string
}

var ExemplarInstance = Exemplar{
	AccountId:                  "b",
	Username:                   "cdrummer",
	IdentityId:                 "aemua9ai",
	IdentityName:               "Camina Drummer", //NOSONAR
	EmailAddress:               "cdrummer@opa.example.com",
	BccName:                    "OPA Secretary",
	BccAddress:                 "secretary@opa.example.com",
	OtherIdentityId:            "reogh7ia",
	OtherIdentityName:          "Transport Union President",
	OtherIdentityEmailAddress:  "pres@tu.example.com",
	SharedAccountId:            "s",
	SharedAccountName:          "OPA Leadership",
	SharedIdentityId:           "eeyie4qu",
	SharedIdentityName:         "OPA",
	SharedIdentityEmailAddress: "bosmangs@opa.example.com",
	ThreadId:                   "soh0aixi",
	EmailId:                    "oot8eev2",
	EmailIds:                   []string{"oot8eev2", "phah6ang", "fo7raidi", "kahsha6p"},
	QuotaId:                    "iezuu7ah",
	SharedQuotaId:              "voos4oht",
	TextSignature:              strings.Join([]string{"Camina Drummer", "President of the Transport Union"}, "\n"),
	MailboxInboxId:             "a",
	MailboxDraftsId:            "d",
	MailboxSentId:              "e",
	MailboxJunkId:              "c",
	MailboxDeletedId:           "b",
	MailboxProjectId:           "i",
	SenderEmailAddress:         "klaes@opa.example.com",
	SenderName:                 "Klaes Ashford",
}

func (e Exemplar) SessionMailAccountCapabilities() SessionMailAccountCapabilities {
	return SessionMailAccountCapabilities{
		MaxMailboxesPerEmail:       0,
		MaxMailboxDepth:            10,
		MaxSizeMailboxName:         255,
		MaxSizeAttachmentsPerEmail: 50000000,
		EmailQuerySortOptions: []string{
			EmailPropertyReceivedAt,
			EmailPropertySize,
			EmailPropertyFrom,
			EmailPropertyTo,
			EmailPropertySubject,
			EmailPropertySentAt,
			EmailPropertyHasAttachment,
			EmailSortPropertyAllInThreadHaveKeyword,
			EmailSortPropertySomeInThreadHaveKeyword,
		},
		MayCreateTopLevelMailbox: true,
	}
}

func (e Exemplar) SessionSubmissionAccountCapabilities() SessionSubmissionAccountCapabilities {
	return SessionSubmissionAccountCapabilities{
		MaxDelayedSend: 2592000,
		SubmissionExtensions: map[string][]string{
			"DELIVERYBY":    {},
			"DSN":           {},
			"FUTURERELEASE": {},
			"MT-PRIORITY":   {"MIXER"},
			"REQUIRETLS":    {},
			"SIZE":          {},
		},
	}
}

func (e Exemplar) SessionVacationResponseAccountCapabilities() SessionVacationResponseAccountCapabilities {
	return SessionVacationResponseAccountCapabilities{}
}

func (e Exemplar) SessionSieveAccountCapabilities() SessionSieveAccountCapabilities {
	return SessionSieveAccountCapabilities{
		MaxSizeScriptName:  512,
		MaxSizeScript:      1048576,
		MaxNumberScripts:   100,
		MaxNumberRedirects: 1,
		SieveExtensions: []string{
			"body",
			"comparator-elbonia",
			"comparator-i;ascii-casemap",
			"comparator-i;ascii-numeric",
			"comparator-i;octet",
			"convert",
			"copy",
			"date",
			"duplicate",
			"editheader",
			"enclose",
			"encoded-character",
			"enotify",
			"envelope",
			"envelope-deliverby",
			"envelope-dsn",
			"environment",
			"ereject",
			"extlists",
			"extracttext",
			"fcc",
			"fileinto",
			"foreverypart",
			"ihave",
			"imap4flags",
			"imapsieve",
			"include",
			"index",
			"mailbox",
			"mailboxid",
			"mboxmetadata",
			"mime",
			"redirect-deliverby",
			"redirect-dsn",
			"regex",
			"reject",
			"relational",
			"replace",
			"servermetadata",
			"spamtest",
			"spamtestplus",
			"special-use",
			"subaddress",
			"vacation",
			"vacation-seconds",
			"variables",
			"virustest",
		},
		NotificationMethods: []string{"mailto"},
		ExternalLists:       nil,
	}
}

func (e Exemplar) SessionBlobAccountCapabilities() SessionBlobAccountCapabilities {
	return SessionBlobAccountCapabilities{
		MaxSizeBlobSet:     7499488,
		MaxDataSources:     16,
		SupportedTypeNames: []string{"Email", "Thread", "SieveScript"},
		SupportedDigestAlgorithms: []HttpDigestAlgorithm{
			HttpDigestAlgorithmSha,
			HttpDigestAlgorithmSha256,
			HttpDigestAlgorithmSha512,
		},
	}
}

func (e Exemplar) SessionQuotaAccountCapabilities() SessionQuotaAccountCapabilities {
	return SessionQuotaAccountCapabilities{}
}

func (e Exemplar) SessionContactsAccountCapabilities() SessionContactsAccountCapabilities {
	return SessionContactsAccountCapabilities{
		MaxAddressBooksPerCard: 128,
		MayCreateAddressBook:   true,
	}
}

func (e Exemplar) SessionCalendarsAccountCapabilities() SessionCalendarsAccountCapabilities {
	var maxCals uint = 128
	var maxParticipants uint = 20
	minDate := UTCDate("0001-01-01T00:00:00Z")
	maxDate := UTCDate("65534-12-31T23:59:59Z")
	maxExp := Duration("P52W1D")
	create := true
	return SessionCalendarsAccountCapabilities{
		MaxCalendarsPerEvent:     &maxCals,
		MinDateTime:              &minDate,
		MaxDateTime:              &maxDate,
		MaxExpandedQueryDuration: maxExp,
		MaxParticipantsPerEvent:  &maxParticipants,
		MayCreateCalendar:        &create,
	}
}

func (e Exemplar) SessionCalendarsParseAccountCapabilities() SessionCalendarsParseAccountCapabilities {
	return SessionCalendarsParseAccountCapabilities{}
}

func (e Exemplar) sessionPrincipalsAccountCapabilities(accountId string) SessionPrincipalsAccountCapabilities {
	return SessionPrincipalsAccountCapabilities{
		CurrentUserPrincipalId: accountId,
	}
}

func (e Exemplar) SessionPrincipalsAccountCapabilities() SessionPrincipalsAccountCapabilities {
	return e.sessionPrincipalsAccountCapabilities(e.AccountId)
}

func (e Exemplar) SessionPrincipalAvailabilityAccountCapabilities() SessionPrincipalAvailabilityAccountCapabilities {
	return SessionPrincipalAvailabilityAccountCapabilities{
		MaxAvailabilityDuration: Duration("P52W1D"),
	}
}

func (e Exemplar) SessionTasksAccountCapabilities() SessionTasksAccountCapabilities {
	return SessionTasksAccountCapabilities{
		MinDateTime:       LocalDate("0001-01-01T00:00:00Z"),
		MaxDateTime:       LocalDate("65534-12-31T23:59:59Z"),
		MayCreateTaskList: true,
	}
}

func (e Exemplar) SessionTasksAlertsAccountCapabilities() SessionTasksAlertsAccountCapabilities {
	return SessionTasksAlertsAccountCapabilities{}
}

func (e Exemplar) SessionTasksAssigneesAccountCapabilities() SessionTasksAssigneesAccountCapabilities {
	return SessionTasksAssigneesAccountCapabilities{}
}

func (e Exemplar) SessionTasksRecurrencesAccountCapabilities() SessionTasksRecurrencesAccountCapabilities {
	return SessionTasksRecurrencesAccountCapabilities{
		MaxExpandedQueryDuration: Duration("P260W1D"),
	}
}

func (e Exemplar) SessionTasksMultilingualAccountCapabilities() SessionTasksMultilingualAccountCapabilities {
	return SessionTasksMultilingualAccountCapabilities{}
}

func (e Exemplar) SessionTasksCustomTimezonesAccountCapabilities() SessionTasksCustomTimezonesAccountCapabilities {
	return SessionTasksCustomTimezonesAccountCapabilities{}
}

func (e Exemplar) SessionPrincipalsOwnerAccountCapabilities() SessionPrincipalsOwnerAccountCapabilities {
	return SessionPrincipalsOwnerAccountCapabilities{
		AccountIdForPrincipal: e.AccountId,
		PrincipalId:           e.AccountId,
	}
}

func (e Exemplar) SessionMDNAccountCapabilities() SessionMDNAccountCapabilities {
	return SessionMDNAccountCapabilities{}
}

func (e Exemplar) SessionAccountCapabilities() SessionAccountCapabilities {
	return e.sessionAccountCapabilities(e.AccountId)
}

func (e Exemplar) sessionAccountCapabilities(accountId string) SessionAccountCapabilities {
	mail := e.SessionMailAccountCapabilities()
	submission := e.SessionSubmissionAccountCapabilities()
	vacationResponse := e.SessionVacationResponseAccountCapabilities()
	sieve := e.SessionSieveAccountCapabilities()
	blob := e.SessionBlobAccountCapabilities()
	quota := e.SessionQuotaAccountCapabilities()
	contacts := e.SessionContactsAccountCapabilities()
	calendars := e.SessionCalendarsAccountCapabilities()
	calendarsParse := e.SessionCalendarsParseAccountCapabilities()
	principals := e.sessionPrincipalsAccountCapabilities(accountId)
	principalsAvailability := e.SessionPrincipalAvailabilityAccountCapabilities()
	tasks := e.SessionTasksAccountCapabilities()
	tasksAlerts := e.SessionTasksAlertsAccountCapabilities()
	tasksAssignees := e.SessionTasksAssigneesAccountCapabilities()
	tasksRecurrences := e.SessionTasksRecurrencesAccountCapabilities()
	tasksMultilingual := e.SessionTasksMultilingualAccountCapabilities()
	tasksCustomTimezones := e.SessionTasksCustomTimezonesAccountCapabilities()
	principalsOwner := e.SessionPrincipalsOwnerAccountCapabilities()
	mdn := e.SessionMDNAccountCapabilities()
	return SessionAccountCapabilities{
		Mail:                   &mail,
		Submission:             &submission,
		VacationResponse:       &vacationResponse,
		Sieve:                  &sieve,
		Blob:                   &blob,
		Quota:                  &quota,
		Contacts:               &contacts,
		Calendars:              &calendars,
		CalendarsParse:         &calendarsParse,
		Principals:             &principals,
		PrincipalsAvailability: &principalsAvailability,
		Tasks:                  &tasks,
		TasksAlerts:            &tasksAlerts,
		TasksAssignees:         &tasksAssignees,
		TasksRecurrences:       &tasksRecurrences,
		TasksMultilingual:      &tasksMultilingual,
		TasksCustomTimezones:   &tasksCustomTimezones,
		PrincipalsOwner:        &principalsOwner,
		MDN:                    &mdn,
	}
}

func (e Exemplar) Account() (Account, string) {
	return Account{
		Name:                e.Username,
		IsPersonal:          true,
		IsReadOnly:          false,
		AccountCapabilities: e.SessionAccountCapabilities(),
	}, "A personal account"
}

func (e Exemplar) SharedAccount() (Account, string, string) {
	return Account{
		Name:                e.SharedAccountId,
		IsPersonal:          false,
		IsReadOnly:          true,
		AccountCapabilities: e.sessionAccountCapabilities(e.SharedAccountId),
	}, "A read-only shared account", "shared"
}

func (e Exemplar) Accounts() []Account {
	a, _ := e.Account()
	s, _, _ := e.SharedAccount()
	return []Account{a, s}
}

func (e Exemplar) Quota() Quota {
	return Quota{
		Id:           e.QuotaId,
		ResourceType: "octets",
		Scope:        "account",
		Used:         11696865,
		HardLimit:    20000000000,
		Name:         e.Username,
		Types: []ObjectTypeName{
			EmailName,
			SieveScriptName,
			FileNodeName,
			CalendarEventName,
			ContactCardName,
		},
		Description: e.IdentityName,
		SoftLimit:   19000000000,
		WarnLimit:   10000000000,
	}
}

func (e Exemplar) Quotas() []Quota {
	return []Quota{
		e.Quota(),
		{
			Id:           e.SharedQuotaId,
			ResourceType: "octets",
			Scope:        "account",
			Used:         29102918,
			HardLimit:    50000000000,
			Name:         e.SharedAccountId,
			Types: []ObjectTypeName{
				EmailName,
				SieveScriptName,
				FileNodeName,
				CalendarEventName,
				ContactCardName,
			},
			Description: e.SharedAccountName,
			SoftLimit:   90000000000,
			WarnLimit:   100000000000,
		},
	}
}

func (e Exemplar) QuotaGetResponse() QuotaGetResponse {
	return QuotaGetResponse{
		AccountId: e.AccountId,
		State:     "oroomoh1",
		NotFound:  []string{"aab2n", "aab8f"},
		List: []Quota{
			e.Quota(),
		},
	}
}

func (e Exemplar) Identity() Identity {
	return Identity{
		Id:    e.IdentityId,
		Name:  e.IdentityName,
		Email: e.EmailAddress,
		Bcc: &[]EmailAddress{
			{Name: e.BccName, Email: e.BccAddress},
		},
		MayDelete:     true,
		TextSignature: &e.TextSignature,
	}
}

func (e Exemplar) OtherIdentity() (Identity, string, string) {
	return Identity{
		Id:        e.OtherIdentityId,
		Name:      e.OtherIdentityName,
		Email:     e.OtherIdentityEmailAddress,
		MayDelete: false,
	}, "Another Identity", "other"
}

func (e Exemplar) Identities() []Identity {
	a := e.Identity()
	b, _, _ := e.OtherIdentity()
	return []Identity{a, b}
}

func (e Exemplar) Identity_req() Identity { //NOSONAR
	return Identity{
		Name:  e.IdentityName,
		Email: e.EmailAddress,
		Bcc: &[]EmailAddress{
			{Name: e.BccName, Email: e.BccAddress},
		},
		TextSignature: &e.TextSignature,
	}
}

func (e Exemplar) IdentityGetResponse() IdentityGetResponse {
	return IdentityGetResponse{
		AccountId: e.AccountId,
		State:     "geechae0",
		NotFound:  []string{"eea2"},
		List:      e.Identities(),
	}
}

func (e Exemplar) Thread() Thread {
	return Thread{
		Id:       e.ThreadId,
		EmailIds: e.EmailIds,
	}
}

func (e Exemplar) MailboxInbox() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxInboxId,
		Name:          "Inbox",
		Role:          JmapMailboxRoleInbox,
		SortOrder:     ptr(0),
		TotalEmails:   1291,
		UnreadEmails:  82,
		TotalThreads:  891,
		UnreadThreads: 55,
		MyRights: &MailboxRights{
			MayReadItems:   true,
			MayAddItems:    true,
			MayRemoveItems: true,
			MaySetSeen:     true,
			MaySetKeywords: true,
			MayCreateChild: true,
			MayRename:      true,
			MayDelete:      true,
			MaySubmit:      true,
		},
		IsSubscribed: truep,
	}, "An Inbox Mailbox", "inbox"
}

func (e Exemplar) MailboxInboxProjects() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxProjectId,
		ParentId:      e.MailboxInboxId,
		Name:          "Projects",
		SortOrder:     ptr(0),
		TotalEmails:   112,
		UnreadEmails:  3,
		TotalThreads:  85,
		UnreadThreads: 2,
		MyRights: &MailboxRights{
			MayReadItems:   true,
			MayAddItems:    true,
			MayRemoveItems: true,
			MaySetSeen:     true,
			MaySetKeywords: true,
			MayCreateChild: true,
			MayRename:      true,
			MayDelete:      true,
			MaySubmit:      true,
		},
		IsSubscribed: truep,
	}, "A Projects Mailbox under the Inbox", "projects"
}

func (e Exemplar) MailboxDrafts() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxDraftsId,
		Name:          "Drafts",
		Role:          JmapMailboxRoleDrafts,
		SortOrder:     ptr(0),
		TotalEmails:   12,
		UnreadEmails:  1,
		TotalThreads:  12,
		UnreadThreads: 1,
		MyRights: &MailboxRights{
			MayReadItems:   true,
			MayAddItems:    true,
			MayRemoveItems: true,
			MaySetSeen:     true,
			MaySetKeywords: true,
			MayCreateChild: true,
			MayRename:      true,
			MayDelete:      true,
			MaySubmit:      true,
		},
		IsSubscribed: truep,
	}, "A Drafts Mailbox", "drafts"
}

func (e Exemplar) MailboxSent() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxSentId,
		Name:          "Sent Items",
		Role:          JmapMailboxRoleSent,
		SortOrder:     ptr(0),
		TotalEmails:   1621,
		UnreadEmails:  0,
		TotalThreads:  1621,
		UnreadThreads: 0,
		MyRights: &MailboxRights{
			MayReadItems:   true,
			MayAddItems:    true,
			MayRemoveItems: true,
			MaySetSeen:     true,
			MaySetKeywords: true,
			MayCreateChild: true,
			MayRename:      true,
			MayDelete:      true,
			MaySubmit:      true,
		},
		IsSubscribed: truep,
	}, "A Sent Mailbox", "sent"
}

func (e Exemplar) MailboxJunk() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxJunkId,
		Name:          "Junk Mail",
		Role:          JmapMailboxRoleJunk,
		SortOrder:     ptr(0),
		TotalEmails:   251,
		UnreadEmails:  0,
		TotalThreads:  251,
		UnreadThreads: 0,
		MyRights: &MailboxRights{
			MayReadItems:   true,
			MayAddItems:    true,
			MayRemoveItems: true,
			MaySetSeen:     true,
			MaySetKeywords: true,
			MayCreateChild: true,
			MayRename:      true,
			MayDelete:      true,
			MaySubmit:      true,
		},
		IsSubscribed: truep,
	}, "A Junk Mailbox", "junk"
}

func (e Exemplar) MailboxDeleted() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxDeletedId,
		Name:          "Deleted Items",
		Role:          JmapMailboxRoleTrash,
		SortOrder:     ptr(0),
		TotalEmails:   99,
		UnreadEmails:  0,
		TotalThreads:  91,
		UnreadThreads: 0,
		MyRights: &MailboxRights{
			MayReadItems:   true,
			MayAddItems:    true,
			MayRemoveItems: true,
			MaySetSeen:     true,
			MaySetKeywords: true,
			MayCreateChild: true,
			MayRename:      true,
			MayDelete:      true,
			MaySubmit:      true,
		},
		IsSubscribed: truep,
	}, "A Trash Mailbox", "deleted"
}

func (e Exemplar) Mailboxes() []Mailbox {
	a, _, _ := e.MailboxInbox()
	b, _, _ := e.MailboxDrafts()
	c, _, _ := e.MailboxSent()
	d, _, _ := e.MailboxJunk()
	f, _, _ := e.MailboxDeleted()
	g, _, _ := e.MailboxInboxProjects()
	return []Mailbox{a, b, c, d, f, g}
}

func (e Exemplar) MailboxGetResponse() MailboxGetResponse {
	return MailboxGetResponse{
		AccountId: e.AccountId,
		State:     "aesh2ahj",
		List:      e.Mailboxes(),
		NotFound:  []string{"ah"},
	}
}

func (e Exemplar) MailboxChange() MailboxChange {
	a, _, _ := e.MailboxInbox()
	return copyTo[MailboxChange](a)
}

func (e Exemplar) MailboxChanges() MailboxChanges {
	a, _, _ := e.MailboxInbox()
	return MailboxChanges{
		NewState:  "aesh2ahj",
		Created:   []Mailbox{a},
		Destroyed: []string{"baingow4"},
	}
}

func (e Exemplar) UploadedBlob() UploadedBlob {
	return UploadedBlob{
		BlobId: "eisoochohl9iekohf5ramaiqu4oucaegheith7otae0xeeg7zuexia4ohjut",
		Size:   12762,
		Type:   "image/png", //NOSONAR
	}
}

func (e Exemplar) Blob() Blob {
	return Blob{
		Id:                "eisoochohl9iekohf5ramaiqu4oucaegheith7otae0xeeg7zuexia4ohjut",
		IsTruncated:       false,
		IsEncodingProblem: false,
		DigestSha512:      "4c33d477254ad056e6919ddecf1df8fffa76ef1e6556e99af9ad43c1cd301fd2c06a35d7d101ec1abbe1c0479276e9cd8cb9046822c6345abf65e453e8ce012d",
		DataAsBase64:      "iVBORw0KGgoAAAANSUhEUgAAAqgAAAC+CAYAAAD5qP3zAAACKWlUWHRYTUw6Y29tLmFkb2JlLnhtcAAAAAAAPD94cGFja2V0IGJlZ2luPSLvu78iIGlkPSJXNU0wTXBDZWhpSHpyZVN6TlRjemtjOWQiPz4KPHg6eG1wbWV0YSB4bWxuczp4PSJhZG9iZTpuczptZXRhLyIgeDp4bXB0az0iWE1QIENvcmUgNi4wLjAiPgogPHJkZjpSREYgeG1sbnM6cmRmPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5LzAyLzIyLXJkZi1zeW50YXgtbnMjIj4KICA8",
		Size:              12762,
	}
}

func (e Exemplar) Email() Email {
	sent, _ := time.Parse(time.RFC3339, "2026-01-12T21:46:01Z")
	received, _ := time.Parse(time.RFC3339, "2026-01-12T21:47:21Z")
	return Email{
		Id:         "ov7ienge",
		BlobId:     "ccyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasiba",
		ThreadId:   "is",
		MailboxIds: map[string]bool{e.MailboxInboxId: true},
		Keywords:   map[string]bool{JmapKeywordAnswered: true},
		Size:       1084,
		ReceivedAt: received,
		MessageId:  []string{"1768845021.1753110@example.com"},
		Sender: []EmailAddress{
			{Name: e.SenderName, Email: e.SenderEmailAddress},
		},
		From: []EmailAddress{
			{Name: e.SenderName, Email: e.SenderEmailAddress},
		},
		To: []EmailAddress{
			{Name: e.IdentityName, Email: e.EmailAddress},
		},
		Subject: "Remember the Cant",
		SentAt:  sent,
		BodyValues: map[string]EmailBodyValue{
			"1": {
				Value: "The demise of the Scopuli and Canterbury was an event where Protogen using its Stealth ship Anubis to capture the crew of the Scopuli to lure the ice-hauler Canterbury into a trap. This event increased tensions amongst Belters, and the two major powers within the Sol system, the Martian Congressional Republic and United Nations.",
			},
			"2": {
				Value: "<p>The demise of the <i>Scopuli</i> and <i>Canterbury</i> was an event where Protogen using its Stealth ship <i>Anubis</i> to capture the crew of the <i>Scopuli</i> to lure the ice-hauler <i>Canterbury</i> into a trap.</p><p>This event increased tensions amongst Belters, and the two major powers within the Sol system, the Martian Congressional Republic and United Nations.</p>",
			},
		},
		TextBody: []EmailBodyPart{
			{PartId: "1", BlobId: "ckyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasibnebdw", Size: 115, Type: "text/plain", Charset: "utf-8"}, //NOSONAR
		},
		HtmlBody: []EmailBodyPart{
			{PartId: "2", BlobId: "ckyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasibnsbvjae", Size: 163, Type: "text/html", Charset: "utf-8"}, //NOSONAR
		},
		Preview: "The Canterbury was destroyed while investigating a false distress call from the Scopuli.",
	}

}

func (e Exemplar) EmailBodyPart() EmailBodyPart {
	return EmailBodyPart{
		PartId:  "1",
		BlobId:  "ckyxndo0fxob1jnm3z2lroex131oj7eo2ezo1djhlfgtsu7jgucfeaiasibnebdw",
		Size:    115,
		Type:    "text/plain",
		Charset: "utf-8",
	}
}

func (e Exemplar) Emails() EmailSearchResults {
	return EmailSearchResults{
		Results:             []Email{e.Email()},
		Total:               uintPtr(132),
		Limit:               uintPtr(1),
		Position:            5,
		CanCalculateChanges: true,
	}
}

func (e Exemplar) EmailGetResponse() EmailGetResponse {
	return EmailGetResponse{
		AccountId: e.AccountId,
		State:     "aesh2ahj",
		NotFound:  []string{"ahx"},
		List:      e.Emails().Results,
	}
}

func (e Exemplar) EmailSubmission() EmailSubmission {
	sendAt, err := time.Parse(time.RFC3339, "2026-04-08T14:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return EmailSubmission{
		Id:         "cea1ae",
		IdentityId: e.IdentityId,
		EmailId:    e.EmailId,
		ThreadId:   e.ThreadId,
		Envelope: &Envelope{
			MailFrom: Address{
				Email: "camina@opa.org.example.com",
			},
			RcptTo: []Address{
				{Email: "crissy@earth.gov.example.com"},
			},
		},
		SendAt:     sendAt,
		UndoStatus: UndoStatusPending,
	}
}

func (e Exemplar) EmailSubmissionGetResponse() EmailSubmissionGetResponse {
	return EmailSubmissionGetResponse{
		AccountId: e.AccountId,
		State:     "eiph2pha",
		NotFound:  []string{"zfa92bn"},
		List: []EmailSubmission{
			e.EmailSubmission(),
		},
	}
}

func (e Exemplar) VacationResponse() VacationResponse {
	from, _ := time.Parse(time.RFC3339, "20260101T00:00:00.000Z")
	to, _ := time.Parse(time.RFC3339, "20260114T23:59:59.999Z")
	return VacationResponse{
		Id:        "aefee7ae",
		IsEnabled: true,
		FromDate:  from,
		ToDate:    to,
		Subject:   "On PTO",
		TextBody:  "I am currently on PTO, please contact info@example.com for any urgent matters.",
	}
}

func (e Exemplar) VacationResponseGetResponse() VacationResponseGetResponse {
	return VacationResponseGetResponse{
		AccountId: e.AccountId,
		State:     "quain7ku",
		List: []VacationResponse{
			e.VacationResponse(),
		},
	}
}

func (e Exemplar) AddressBook() AddressBook {
	return AddressBook{
		Id:           "bar5kike",
		Name:         "My Friends",
		Description:  "An address book of the people I trust",
		IsDefault:    true,
		IsSubscribed: true,
		MyRights: AddressBookRights{
			MayRead:   true,
			MayWrite:  true,
			MayAdmin:  true,
			MayDelete: true,
		},
	}
}

func (e Exemplar) AddressBookChange() AddressBookChange {
	return AddressBookChange{
		Description: ptr("A different name"),
	}
}

func (e Exemplar) OtherAddressBook() (AddressBook, string, string) {
	return AddressBook{
		Id:           "aemaeWun6Fei",
		Name:         "Shared",
		Description:  "People I share Files with",
		IsDefault:    false,
		IsSubscribed: true,
		MyRights: AddressBookRights{
			MayRead:   true,
			MayWrite:  true,
			MayAdmin:  false,
			MayDelete: false,
		},
	}, "Another AddressBook", "other"
}

func (e Exemplar) AddressBookGetResponse() AddressBookGetResponse {
	a := e.AddressBook()
	return AddressBookGetResponse{
		AccountId: e.AccountId,
		State:     "liew7dah",
		NotFound:  []string{},
		List:      []AddressBook{a},
	}
}

func (e Exemplar) JSContactEmailAddress() c.EmailAddress {
	return c.EmailAddress{
		Type:    c.EmailAddressType,
		Address: "camina@opa.org",
		Contexts: map[c.EmailAddressContext]bool{
			c.EmailAddressContextWork:    true,
			c.EmailAddressContextPrivate: true,
		},
		Pref:  1,
		Label: "bosmang",
	}
}

func (e Exemplar) NameComponent() c.NameComponent {
	return c.NameComponent{Type: c.NameComponentType, Value: "Camina", Kind: c.NameComponentKindGiven}
}

func (e Exemplar) OtherNameComponent() (c.NameComponent, string, string) {
	return c.NameComponent{Type: c.NameComponentType, Value: "Drummer", Kind: c.NameComponentKindSurname}, "A surname NameComponent", "surname"
}

func (e Exemplar) NameComponents() []c.NameComponent {
	a := e.NameComponent()
	b, _, _ := e.OtherNameComponent()
	return []c.NameComponent{a, b}
}

func (e Exemplar) Name() c.Name {
	return c.Name{
		Type: c.NameType,
		Components: []c.NameComponent{
			{Type: c.NameComponentType, Value: "Drummer", Kind: c.NameComponentKindSurname},
			{Type: c.NameComponentType, Value: "Camina", Kind: c.NameComponentKindGiven},
		},
		IsOrdered:        true,
		DefaultSeparator: ", ",
	}
}

func (e Exemplar) OtherName() (c.Name, string, string) {
	return c.Name{
		Type: c.NameType,
		Components: []c.NameComponent{
			{Type: c.NameComponentType, Value: "Klaes", Kind: c.NameComponentKindGiven},
			{Type: c.NameComponentType, Value: "Ashford", Kind: c.NameComponentKindSurname},
		},
		IsOrdered:        false,
		DefaultSeparator: "  ",
		Full:             "Klaes Ashford",
	}, "Another Name", "other"
}

func (e Exemplar) ComplexName() (c.Name, string, string) {
	return c.Name{
		Type: c.NameType,
		Components: []c.NameComponent{
			{
				Type:     c.NameComponentType,
				Value:    "Diego",
				Kind:     c.NameComponentKindGiven,
				Phonetic: "/diˈeɪɡəʊ/",
			},
			{
				Value: "Rivera",
				Kind:  c.NameComponentKindSurname,
			},
			{
				Value: "Barrientos",
				Kind:  c.NameComponentKindSurname2,
			},
		},
		IsOrdered:        true,
		DefaultSeparator: " ",
		Full:             "Diego Rivera Barrientos",
		SortAs: map[string]string{
			string(c.NameComponentKindSurname): "Rivera Barrientos",
			string(c.NameComponentKindGiven):   "Diego",
		},
	}, "A complex name", "complex"
}

func (e Exemplar) Names() []c.Name {
	a := e.Name()
	b, _, _ := e.OtherName()
	d, _, _ := e.ComplexName()
	return []c.Name{a, b, d}
}

func (e Exemplar) JSCalendar() c.Calendar {
	return c.Calendar{
		Type:      c.CalendarType,
		Kind:      c.CalendarKindCalendar,
		Uri:       "https://opa.example.com/cal/3051dd3f-065a-4087-ab86-a790510aebe1.ics",
		MediaType: "text/calendar",
		Contexts: map[c.CalendarContext]bool{
			c.CalendarContextPrivate: true,
		},
	}
}

func (e Exemplar) OtherJSCalendar() (c.Calendar, string, string) {
	return c.Calendar{
		Type:      c.CalendarType,
		Kind:      c.CalendarKindCalendar,
		Uri:       "https://opencloud.example.com/calendar/d05779b6-9638-4694-9869-008a61df6025",
		MediaType: "application/jscontact+json", //NOSONAR
		Contexts: map[c.CalendarContext]bool{
			c.CalendarContextWork: true,
		},
		Pref:  0,
		Label: "business-contacts",
	}, "A JSContact Calendar", "other"
}

func (e Exemplar) JSCalendars() []c.Calendar {
	a := e.JSCalendar()
	b, _, _ := e.OtherJSCalendar()
	return []c.Calendar{a, b}
}

func (e Exemplar) Calendar() Calendar {
	return Calendar{
		Id:                    "uh3uchohteeC",
		Name:                  "Default Calendar",
		Description:           "The default calendar for this user",
		Color:                 "blue",
		IsSubscribed:          true,
		IsVisible:             true,
		IsDefault:             true,
		IncludeInAvailability: IncludeInAvailabilityAll,
		SortOrder:             0,
		TimeZone:              "Europe/Berlin",
		MyRights: &CalendarRights{
			MayReadFreeBusy:  true,
			MayReadItems:     true,
			MayWriteAll:      true,
			MayWriteOwn:      true,
			MayUpdatePrivate: true,
			MayRSVP:          true,
			MayAdmin:         true,
			MayDelete:        true,
		},
	}
}

func (e Exemplar) CalendarGetResponse() CalendarGetResponse {
	a := e.Calendar()
	return CalendarGetResponse{
		AccountId: e.AccountId,
		State:     "aesh2ahj",
		List:      []Calendar{a},
		NotFound:  []string{"eehn", "eehz"},
	}
}

func (e Exemplar) Link() c.Link {
	return c.Link{
		Type:      c.LinkType,
		Kind:      c.LinkKindContact,
		Uri:       "https://opencloud.example.com/calendar/d05779b6-9638-4694-9869-008a61df6025",
		MediaType: "application/jscontact+json",
		Contexts: map[c.LinkContext]bool{
			c.LinkContextWork: true,
		},
		Pref:  0,
		Label: "sample",
	}
}

func (e Exemplar) CryptoKey() c.CryptoKey {
	return c.CryptoKey{
		Type:      c.CryptoKeyType,
		Uri:       "https://opencloud.example.com/keys/53c25ae8-2800-4905-86a9-92d765b9efb9.pgp",
		MediaType: "application/pgp-keys",
		Contexts: map[c.CryptoKeyContext]bool{
			c.CryptoKeyContextWork: true,
		},
		Pref:  0,
		Label: "work",
	}
}

func (e Exemplar) Directory() c.Directory {
	return c.Directory{
		Type:      c.DirectoryType,
		Kind:      c.DirectoryKindEntry,
		Uri:       "https://opencloud.example.com/dir/38b850ea-f6ac-419d-8a26-1d19615cfa9b.vcf",
		MediaType: "text/vcard",
		Contexts: map[c.DirectoryContext]bool{
			c.DirectoryContextWork: true,
		},
		Pref:   0,
		Label:  "work",
		ListAs: 3,
	}
}

func (e Exemplar) Media() c.Media {
	return c.Media{
		Type:      c.MediaType,
		Kind:      c.MediaKindLogo,
		Uri:       "https://opencloud.eu/opencloud.svg",
		MediaType: "image/svg+xml",
		Contexts: map[c.MediaContext]bool{
			c.MediaContextWork: true,
		},
		Pref:   0,
		Label:  "logo",
		BlobId: "1d92cf97e32b42ceb5538f0804a41891",
	}
}

func (e Exemplar) Relation() c.Relation {
	return c.Relation{
		Type: c.RelationType,
		Relation: map[c.Relationship]bool{
			c.RelationCoWorker: true,
			c.RelationFriend:   true,
		},
	}
}

func (e Exemplar) Nickname() c.Nickname {
	return c.Nickname{
		Type: c.NicknameType,
		Name: "Bob",
		Contexts: map[c.NicknameContext]bool{
			c.NicknameContextPrivate: true,
		},
		Pref: 3,
	}
}

func (e Exemplar) OrgUnit() c.OrgUnit {
	return c.OrgUnit{
		Type:   c.OrgUnitType,
		Name:   "Skynet",
		SortAs: "SKY",
	}
}

func (e Exemplar) Organization() c.Organization {
	return c.Organization{
		Type:   c.OrganizationType,
		Name:   "Cyberdyne",
		SortAs: "CYBER",
		Units: []c.OrgUnit{
			{
				Type:   c.OrgUnitType,
				Name:   "Skynet",
				SortAs: "SKY",
			},
			{
				Type: c.OrgUnitType,
				Name: "Cybernics",
			},
		},
		Contexts: map[c.OrganizationContext]bool{
			c.OrganizationContextWork: true,
		},
	}
}

func (e Exemplar) Pronouns() c.Pronouns {
	return c.Pronouns{
		Type:     c.PronounsType,
		Pronouns: "they/them",
		Contexts: map[c.PronounsContext]bool{
			c.PronounsContextWork:    true,
			c.PronounsContextPrivate: true,
		},
		Pref: 1,
	}
}

func (e Exemplar) Title() c.Title {
	return c.Title{
		Type:           c.TitleType,
		Name:           "Doctor",
		Kind:           c.TitleKindTitle,
		OrganizationId: "407e1992-9a2b-4e4f-a11b-85a509a4b5ae",
	}
}

func (e Exemplar) SpeakToAs() c.SpeakToAs {
	return c.SpeakToAs{
		Type:              c.SpeakToAsType,
		GrammaticalGender: c.GrammaticalGenderNeuter,
		Pronouns: map[string]c.Pronouns{
			"a": {
				Type:     c.PronounsType,
				Pronouns: "they/them",
				Contexts: map[c.PronounsContext]bool{
					c.PronounsContextPrivate: true,
				},
				Pref: 1,
			},
			"b": {
				Type:     c.PronounsType,
				Pronouns: "he/him",
				Contexts: map[c.PronounsContext]bool{
					c.PronounsContextWork: true,
				},
				Pref: 99,
			},
		},
	}
}

func (e Exemplar) OnlineService() c.OnlineService {
	return c.OnlineService{
		Type:    c.OnlineServiceType,
		Service: "OPA Network",
		Contexts: map[c.OnlineServiceContext]bool{
			c.OnlineServiceContextWork: true,
		},
		Uri:   "https://opa.org/cdrummer",
		User:  "cdrummer@opa.org",
		Pref:  12,
		Label: "opa",
	}
}

func (e Exemplar) Phone() c.Phone {
	return c.Phone{
		Type:   c.PhoneType,
		Number: "+15551234567",
		Features: map[c.PhoneFeature]bool{
			c.PhoneFeatureText:       true,
			c.PhoneFeatureMainNumber: true,
			c.PhoneFeatureMobile:     true,
			c.PhoneFeatureVideo:      true,
			c.PhoneFeatureVoice:      true,
		},
		Contexts: map[c.PhoneContext]bool{
			c.PhoneContextWork:    true,
			c.PhoneContextPrivate: true,
		},
		Pref:  42,
		Label: "opa",
	}
}

func (e Exemplar) LanguagePref() c.LanguagePref {
	return c.LanguagePref{
		Type:     c.LanguagePrefType,
		Language: "fr-BE",
		Contexts: map[c.LanguagePrefContext]bool{
			c.LanguagePrefContextPrivate: true,
		},
		Pref: 2,
	}
}

func (e Exemplar) SchedulingAddress() c.SchedulingAddress {
	return c.SchedulingAddress{
		Type:  c.SchedulingAddressType,
		Uri:   "mailto:camina@opa.org",
		Label: "opa",
		Contexts: map[c.SchedulingAddressContext]bool{
			c.SchedulingAddressContextWork: true,
		},
		Pref: 3,
	}
}

func (e Exemplar) AddressComponent() c.AddressComponent {
	return c.AddressComponent{
		Type:  c.AddressComponentType,
		Kind:  c.AddressComponentKindPostcode,
		Value: "20190",
	}
}
func (e Exemplar) Address() c.Address {
	return c.Address{
		Type: c.AddressType,
		Contexts: map[c.AddressContext]bool{
			c.AddressContextDelivery: true,
			c.AddressContextWork:     true,
		},
		Components: []c.AddressComponent{
			{Type: c.AddressComponentType, Kind: c.AddressComponentKindNumber, Value: "54321"},
			{Kind: c.AddressComponentKindSeparator, Value: " "},
			{Kind: c.AddressComponentKindName, Value: "Oak St"},
			{Kind: c.AddressComponentKindLocality, Value: "Reston"},
			{Kind: c.AddressComponentKindRegion, Value: "VA"},
			{Kind: c.AddressComponentKindSeparator, Value: " "},
			{Kind: c.AddressComponentKindPostcode, Value: "20190"},
			{Kind: c.AddressComponentKindCountry, Value: "USA"},
		},
		CountryCode:      "US",
		DefaultSeparator: ", ",
		IsOrdered:        true,
	}
}

func (e Exemplar) PartialDate() c.PartialDate {
	return c.PartialDate{
		Type:          c.PartialDateType,
		Year:          2025,
		Month:         9,
		Day:           25,
		CalendarScale: "iso8601",
	}
}

func (e Exemplar) Anniversary() (c.Anniversary, string, string) {
	return c.Anniversary{
		Type: c.AnniversaryType,
		Kind: c.AnniversaryKindBirth,
		Date: &c.PartialDate{
			Type:  c.PartialDateType,
			Year:  2026,
			Month: 3,
			Day:   14,
		},
	}, "An anniversary with a PartialDate", "partialdate"
}

func (e Exemplar) OtherAnniversary() (c.Anniversary, string, string) {
	ts, _ := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00") //NOSONAR
	return c.Anniversary{
		Type: c.AnniversaryType,
		Kind: c.AnniversaryKindBirth,
		Date: &c.Timestamp{
			Type: c.TimestampType,
			Utc:  ts,
		},
	}, "An anniversary with a Timestamp", "timestamp"
}

func (e Exemplar) Author() c.Author {
	return c.Author{
		Type: c.AuthorType,
		Name: "Camina Drummer",
		Uri:  "https://opa.org/cdrummer",
	}
}

func (e Exemplar) Note() c.Note {
	ts, _ := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00")
	a := e.Author()
	return c.Note{
		Type:    c.NoteType,
		Note:    "this is a note",
		Created: ts,
		Author:  &a,
	}
}

func (e Exemplar) PersonalInfo() c.PersonalInfo {
	return c.PersonalInfo{
		Type:   c.PersonalInfoType,
		Kind:   c.PersonalInfoKindExpertise,
		Value:  "motivation",
		Level:  c.PersonalInfoLevelHigh,
		ListAs: 1,
		Label:  "opa",
	}
}

func (e Exemplar) DesignContactCard() (ContactCard, string, string) {
	created, _ := time.Parse(time.RFC3339, "2025-07-09T07:12:28+02:00")
	updated, _ := time.Parse(time.RFC3339, "2025-07-10T09:58:01+02:00")
	return ContactCard{
		Type: c.ContactCardType,
		Kind: c.ContactCardKindIndividual,
		Id:   "loTh8ahmubei",
		Uid:  "ffcb3f80-8334-46ff-882b-fcb7dff49204",
		AddressBookIds: map[string]bool{
			"79047052-ae0e-4299-8860-5bff1a139f3d": true,
		},
		Version:  c.JSContactVersion_1_0,
		Created:  created,
		Updated:  updated,
		Language: "en-GB",
		ProdId:   "OpenCloud Groupware 1.0", //NOSONAR
		Name: &c.Name{
			Type: c.NameType,
			Components: []c.NameComponent{
				{Value: "Bessie", Kind: c.NameComponentKindGiven},
				{Value: "Cooper", Kind: c.NameComponentKindSurname},
			},
			IsOrdered:        true,
			DefaultSeparator: " ",
		},
		Nicknames: map[string]c.Nickname{
			"soacaivie5Po": {
				Name: "Bess",
				Contexts: map[c.NicknameContext]bool{
					c.NicknameContextPrivate: true,
					c.NicknameContextWork:    true,
				},
			},
		},
		Anniversaries: map[string]c.Anniversary{
			"at4Jaig2TeeT": {
				Kind: c.AnniversaryKindBirth,
				Date: c.PartialDate{
					Type:  c.PartialDateType,
					Year:  1980,
					Month: 1,
					Day:   22,
				},
			},
		},
		SpeakToAs: &c.SpeakToAs{
			GrammaticalGender: c.GrammaticalGenderFeminine,
		},
		Notes: map[string]c.Note{
			"beecaeViShu6": {
				Note:    "Bess likes chocolate",
				Created: created,
				Author: &c.Author{
					Name: "Stina",
				},
			},
		},
		Emails: map[string]c.EmailAddress{
			"foo5Isheel2t": {
				Address: "bessiecooper@mail.com",
				Contexts: map[c.EmailAddressContext]bool{
					c.EmailAddressContextWork:    true,
					c.EmailAddressContextPrivate: true,
				},
			},
			"bueVua3Eith8": {
				Address: "bessiecooper@mail.com",
				Contexts: map[c.EmailAddressContext]bool{
					"home": true,
				},
			},
		},
		Phones: map[string]c.Phone{
			"aish5bahQu4t": {
				Number:   "+1-555-123-4567",
				Contexts: map[c.PhoneContext]bool{c.PhoneContextWork: true},
			},
			"oopeej8ev9Oi": {
				Number:   "+1-555-123-1234",
				Contexts: map[c.PhoneContext]bool{c.PhoneContextPrivate: true, "home": true},
			},
		},
		OnlineServices: map[string]c.OnlineService{
			"soo6fohsh9Ae": {
				Service:  "LinkedIn",
				User:     "@bessiecooper",
				Contexts: map[c.OnlineServiceContext]bool{c.OnlineServiceContextWork: true},
			},
		},
		Addresses: map[string]c.Address{
			"pieF3Uc2eg9M": {
				Components: []c.AddressComponent{
					{Kind: c.AddressComponentKindNumber, Value: "2972"},
					{Kind: c.AddressComponentKindName, Value: "Westheimer Rd."},
					{Kind: c.AddressComponentKindLocality, Value: "Santa Ana"},
					{Kind: c.AddressComponentKindRegion, Value: "Illinois"},
					{Kind: c.AddressComponentKindPostcode, Value: "85486"},
				},
				CountryCode: "USA",
				Contexts:    map[c.AddressContext]bool{c.AddressContextWork: true, c.AddressContextPrivate: true},
			},
		},
		Organizations: map[string]c.Organization{
			"oocooXeiya9L": {
				Name: "Organization's name",
				Units: []c.OrgUnit{
					{Name: "Department"},
				},
			},
		},
		Titles: map[string]c.Title{
			"yohshohTh4so": {
				Name:           "Job title",
				Kind:           c.TitleKindTitle,
				OrganizationId: "oocooXeiya9L",
			},
			"Pahd2Azeedai": {
				Name:           "Role in project/company",
				Kind:           c.TitleKindRole,
				OrganizationId: "oocooXeiya9L",
			},
		},
		RelatedTo: map[string]c.Relation{
			"096fc269-ffc4-4f2f-978f-5bd1bb0962d5": {
				Relation: map[c.Relationship]bool{
					"manager": true,
				},
			},
		},
		Keywords: map[string]bool{
			"sales":     true,
			"important": true,
		},
		PreferredLanguages: map[string]c.LanguagePref{
			"Aelujee9moi1": {
				Language: "en-US",
				Contexts: map[c.LanguagePrefContext]bool{
					c.LanguagePrefContextPrivate: true,
					c.LanguagePrefContextWork:    true,
				},
				Pref: 1,
			},
			"SheFai0Aishi": {
				Language: "de-DE",
				Contexts: map[c.LanguagePrefContext]bool{
					c.LanguagePrefContextPrivate: true,
				},
				Pref: 2,
			},
		},
		Media: map[string]c.Media{
			"iegh7veeJ9ta": {
				Kind:      c.MediaKindPhoto,
				BlobId:    "ieP4huzohv8gexahzeizei0che2keedu",
				MediaType: "image/png",
				Contexts: map[c.MediaContext]bool{
					c.MediaContextWork:    true,
					c.MediaContextPrivate: true,
				},
			},
			"Zae0ahPho6ae": {
				Kind:      c.MediaKindLogo,
				Uri:       "https://acme.com/logo.jpg",
				MediaType: "image/jpeg",
				Contexts: map[c.MediaContext]bool{
					c.MediaContextWork: true,
				},
			},
		},
	}, "Another Contact Card", "other"
}

func (e Exemplar) IndividualContactCard() (ContactCard, string, string) {
	return ContactCard{
		Kind: c.ContactCardKindIndividual,
		AddressBookIds: map[string]bool{
			"c34c2bb4-4e8e-4579-b35d-6f6739a11146": true,
		},
		Language: "de-DE",
		ProdId:   "OpenCloud Groupware 1.0",
		RelatedTo: map[string]c.Relation{
			"urn:uid:ca9d2a62-e068-43b6-a470-46506976d505": {
				Type: c.RelationType,
				Relation: map[c.Relationship]bool{
					c.RelationContact: true,
				},
			},
		},
		Name: &c.Name{
			Type: c.NameType,
			Components: []c.NameComponent{
				{Value: "Roberta", Kind: c.NameComponentKindGiven},
				{Value: " ", Kind: c.NameComponentKindSeparator},
				{Value: "Draper", Kind: c.NameComponentKindSurname},
			},
			IsOrdered: true,
		},
		Nicknames: map[string]c.Nickname{
			"a": {
				Type: c.NicknameType,
				Name: "Bobby",
				Contexts: map[c.NicknameContext]bool{
					c.NicknameContextWork:    true,
					c.NicknameContextPrivate: true,
				},
				Pref: 1,
			},
		},
		Organizations: map[string]c.Organization{
			"o": {
				Type: c.OrganizationType,
				Name: "Martian Marine Corps",
				Units: []c.OrgUnit{
					{Name: "Special Forces"},
				},
				Contexts: map[c.OrganizationContext]bool{
					c.OrganizationContextWork: true,
				},
			},
		},
		SpeakToAs: &c.SpeakToAs{
			Type:              c.SpeakToAsType,
			GrammaticalGender: c.GrammaticalGenderFeminine,
			Pronouns: map[string]c.Pronouns{
				"p": {
					Type:     c.PronounsType,
					Pronouns: "she/her",
					Contexts: map[c.PronounsContext]bool{
						c.PronounsContextWork: true,
					},
					Pref: 1,
				},
			},
		},
		Emails: map[string]c.EmailAddress{
			"e": {
				Type:    c.EmailAddressType,
				Address: "gunny@mmc.mars.gov.example.com",
				Contexts: map[c.EmailAddressContext]bool{
					c.EmailAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		OnlineServices: map[string]c.OnlineService{
			"s": {
				Type:    c.OnlineServiceType,
				Service: "MarsNet",
				Uri:     "https://mars.example.com/@gunny",
				User:    "gunny",
				Contexts: map[c.OnlineServiceContext]bool{
					c.OnlineServiceContextWork: true,
				},
				Pref: 1,
			},
		},
		Phones: map[string]c.Phone{
			"p": {
				Type:   c.PhoneType,
				Number: "+1-555-123-4567",
				Features: map[c.PhoneFeature]bool{
					c.PhoneFeatureVoice: true,
					c.PhoneFeatureText:  true,
				},
				Contexts: map[c.PhoneContext]bool{
					c.PhoneContextWork: true,
				},
				Pref: 1,
			},
		},
		Media: map[string]c.Media{
			"m": {
				Type:      c.MediaType,
				Kind:      c.MediaKindLogo,
				Uri:       "https://static.wikia.nocookie.net/expanse/images/3/3a/Bobbie_S4_closeup.png/revision/latest?cb=20191206015449",
				MediaType: "image/png",
				Contexts: map[c.MediaContext]bool{
					c.MediaContextWork: true,
				},
			},
		},
		Keywords: map[string]bool{
			"imaginary": true,
			"test":      true,
		},
	}, "A ContactCard for an individual", "individual"
}

func (e Exemplar) ContactCard() ContactCard {
	created, _ := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00")
	updated, _ := time.Parse(time.RFC3339, "2025-09-26T09:58:01+02:00")
	return ContactCard{
		Type: c.ContactCardType,
		Kind: c.ContactCardKindGroup,
		Id:   "20fba820-2f8e-432d-94f1-5abbb59d3ed7",
		AddressBookIds: map[string]bool{
			"79047052-ae0e-4299-8860-5bff1a139f3d": true,
			"44eb6105-08c1-458b-895e-4ad1149dfabd": true,
		},
		Version:  c.JSContactVersion_1_0,
		Created:  created,
		Language: "fr-BE",
		Members: map[string]bool{
			"314815dd-81c8-4640-aace-6dc83121616d": true,
			"c528b277-d8cb-45f2-b7df-1aa3df817463": true,
			"81dea240-c0a4-4929-82e7-79e713a8bbe4": true,
		},
		ProdId: "OpenCloud Groupware 1.0",
		RelatedTo: map[string]c.Relation{
			"urn:uid:ca9d2a62-e068-43b6-a470-46506976d505": {
				Type: c.RelationType,
				Relation: map[c.Relationship]bool{
					c.RelationContact: true,
				},
			},
			"urn:uid:72183ec2-b218-4983-9c89-ff117eeb7c5e": {
				Relation: map[c.Relationship]bool{
					c.RelationEmergency: true,
					c.RelationSpouse:    true,
				},
			},
		},
		Uid:     "1091f2bb-6ae6-4074-bb64-df74071d7033",
		Updated: updated,
		Name: &c.Name{
			Type: c.NameType,
			Components: []c.NameComponent{
				{Type: c.NameComponentType, Value: "OpenCloud", Kind: c.NameComponentKindSurname},
				{Value: " ", Kind: c.NameComponentKindSeparator},
				{Value: "Team", Kind: c.NameComponentKindSurname2},
			},
			IsOrdered:        true,
			DefaultSeparator: ", ",
			SortAs: map[string]string{
				string(c.NameComponentKindSurname): "OpenCloud Team",
			},
			Full: "OpenCloud Team",
		},
		Nicknames: map[string]c.Nickname{
			"a": {
				Type: c.NicknameType,
				Name: "The Team",
				Contexts: map[c.NicknameContext]bool{
					c.NicknameContextWork: true,
				},
				Pref: 1,
			},
		},
		Organizations: map[string]c.Organization{
			"o": {
				Type: c.OrganizationType,
				Name: "OpenCloud GmbH",
				Units: []c.OrgUnit{
					{Type: c.OrgUnitType, Name: "Marketing", SortAs: "marketing"},
					{Type: c.OrgUnitType, Name: "Sales"},
					{Name: "Operations", SortAs: "ops"},
				},
				SortAs: "opencloud",
				Contexts: map[c.OrganizationContext]bool{
					c.OrganizationContextWork: true,
				},
			},
		},
		SpeakToAs: &c.SpeakToAs{
			Type:              c.SpeakToAsType,
			GrammaticalGender: c.GrammaticalGenderInanimate,
			Pronouns: map[string]c.Pronouns{
				"p": {
					Type:     c.PronounsType,
					Pronouns: "it",
					Contexts: map[c.PronounsContext]bool{
						c.PronounsContextWork: true,
					},
					Pref: 1,
				},
			},
		},
		Titles: map[string]c.Title{
			"t": {
				Type:           c.TitleType,
				Name:           "The",
				Kind:           c.TitleKindTitle,
				OrganizationId: "o",
			},
		},
		Emails: map[string]c.EmailAddress{
			"e": {
				Type:    c.EmailAddressType,
				Address: "info@opencloud.eu.example.com",
				Contexts: map[c.EmailAddressContext]bool{
					c.EmailAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		OnlineServices: map[string]c.OnlineService{
			"s": {
				Type:    c.OnlineServiceType,
				Service: "The Misinformation Game",
				Uri:     "https://misinfogame.com/91886aa0-3586-4ade-b9bb-ec031464a251",
				User:    "opencloudeu",
				Contexts: map[c.OnlineServiceContext]bool{
					c.OnlineServiceContextWork: true,
				},
				Pref:  1,
				Label: "imaginary",
			},
		},
		Phones: map[string]c.Phone{
			"p": {
				Type:   c.PhoneType,
				Number: "+1-804-222-1111",
				Features: map[c.PhoneFeature]bool{
					c.PhoneFeatureVoice: true,
					c.PhoneFeatureText:  true,
				},
				Contexts: map[c.PhoneContext]bool{
					c.PhoneContextWork: true,
				},
				Pref:  1,
				Label: "imaginary",
			},
		},
		PreferredLanguages: map[string]c.LanguagePref{
			"wa": {
				Type:     c.LanguagePrefType,
				Language: "wa-BE",
				Contexts: map[c.LanguagePrefContext]bool{
					c.LanguagePrefContextPrivate: true,
				},
				Pref: 1,
			},
			"de": {
				Language: "de-DE",
				Contexts: map[c.LanguagePrefContext]bool{
					c.LanguagePrefContextWork: true,
				},
				Pref: 2,
			},
		},
		Calendars: map[string]c.Calendar{
			"c": {
				Type:      c.CalendarType,
				Kind:      c.CalendarKindCalendar,
				Uri:       "https://opencloud.eu/calendars/521b032b-a2b3-4540-81b9-3f6bccacaab2",
				MediaType: "application/jscontact+json",
				Contexts: map[c.CalendarContext]bool{
					c.CalendarContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		SchedulingAddresses: map[string]c.SchedulingAddress{
			"s": {
				Type: c.SchedulingAddressType,
				Uri:  "mailto:scheduling@opencloud.eu.example.com",
				Contexts: map[c.SchedulingAddressContext]bool{
					c.SchedulingAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		Addresses: map[string]c.Address{
			"k26": {
				Type: c.AddressType,
				Components: []c.AddressComponent{
					{Type: c.AddressComponentType, Kind: c.AddressComponentKindBlock, Value: "2-7"},
					{Kind: c.AddressComponentKindSeparator, Value: "-"},
					{Kind: c.AddressComponentKindNumber, Value: "2"},
					{Kind: c.AddressComponentKindSeparator, Value: " "},
					{Kind: c.AddressComponentKindDistrict, Value: "Marunouchi"},
					{Kind: c.AddressComponentKindLocality, Value: "Chiyoda-ku"},
					{Kind: c.AddressComponentKindRegion, Value: "Tokyo"},
					{Kind: c.AddressComponentKindSeparator, Value: " "},
					{Kind: c.AddressComponentKindPostcode, Value: "100-8994"},
				},
				IsOrdered:        true,
				DefaultSeparator: ", ",
				Full:             "2-7-2 Marunouchi, Chiyoda-ku, Tokyo 100-8994",
				CountryCode:      "JP",
				Coordinates:      "geo:35.6796373,139.7616907",
				TimeZone:         "JST",
				Contexts: map[c.AddressContext]bool{
					c.AddressContextDelivery: true,
					c.AddressContextWork:     true,
				},
				Pref: 2,
			},
		},
		CryptoKeys: map[string]c.CryptoKey{
			"k1": {
				Type:      c.CryptoKeyType,
				Uri:       "https://opencloud.eu.example.com/keys/d550f57c-582c-43cc-8d94-822bded9ab36",
				MediaType: "application/pgp-keys",
				Contexts: map[c.CryptoKeyContext]bool{
					c.CryptoKeyContextWork: true,
				},
				Pref:  1,
				Label: "keys",
			},
		},
		Directories: map[string]c.Directory{
			"d1": {
				Type:   c.DirectoryType,
				Kind:   c.DirectoryKindEntry,
				Uri:    "https://opencloud.eu.example.com/addressbook/8c2f0363-af0a-4d16-a9d5-8a9cd885d722",
				ListAs: 1,
			},
		},
		Links: map[string]c.Link{
			"r1": {
				Type: c.LinkType,
				Kind: c.LinkKindContact,
				Contexts: map[c.LinkContext]bool{
					c.LinkContextWork: true,
				},
				Uri: "mailto:contact@opencloud.eu.example.com",
			},
		},
		Media: map[string]c.Media{
			"m": {
				Type:      c.MediaType,
				Kind:      c.MediaKindLogo,
				Uri:       "https://opencloud.eu.example.com/opencloud.svg",
				MediaType: "image/svg+xml",
				Contexts: map[c.MediaContext]bool{
					c.MediaContextWork: true,
				},
				Pref:   123,
				Label:  "svg",
				BlobId: "53feefbabeb146fcbe3e59e91462fa5f",
			},
		},
		Anniversaries: map[string]c.Anniversary{
			"birth": {
				Type: c.AnniversaryType,
				Kind: c.AnniversaryKindBirth,
				Date: &c.PartialDate{
					Type:          c.PartialDateType,
					Year:          2025,
					Month:         9,
					Day:           26,
					CalendarScale: "iso8601",
				},
			},
		},
		Keywords: map[string]bool{
			"imaginary": true,
			"test":      true,
		},
		Notes: map[string]c.Note{
			"n1": {
				Type:    c.NoteType,
				Note:    "This is a note.",
				Created: created,
				Author: &c.Author{
					Type: c.AuthorType,
					Name: "Test Data",
					Uri:  "https://isbn.example.com/a461f292-6bf1-470e-b08d-f6b4b0223fe3",
				},
			},
		},
		PersonalInfo: map[string]c.PersonalInfo{
			"p1": {
				Type:   c.PersonalInfoType,
				Kind:   c.PersonalInfoKindExpertise,
				Value:  "Clouds",
				Level:  c.PersonalInfoLevelHigh,
				ListAs: 1,
				Label:  "experts",
			},
		},
		Localizations: map[string]c.PatchObject{
			"fr": {
				"personalInfo": map[string]any{
					"value": "Nuages",
				},
			},
		},
	}
}

func (e Exemplar) ContactCardChangeForCreate() (ContactCardChange, string, string) {
	a, _, _ := e.IndividualContactCard()
	return copyTo[ContactCardChange](a), "A ContactCard to create", "create"
}

func (e Exemplar) ContactCardChangeForUpdate() (ContactCardChange, string, string) {
	return ContactCardChange{
		AddressBookIds: map[string]*bool{
			"c34c2bb4-4e8e-4579-b35d-6f6739a11146": nil,
			"02b6977f-bb60-4511-949e-37f47a930382": truep,
		},
		Nicknames: map[string]c.Nickname{
			"a": {
				Name: "Bobbie",
			},
		},
	}, "Updates to a ContactCard", "update"
}

func (e Exemplar) ContactCardGetResponse() ContactCardGetResponse {
	a := e.ContactCard()
	b, _, _ := e.DesignContactCard()
	return ContactCardGetResponse{
		AccountId: e.AccountId,
		State:     "ewohl8ie",
		NotFound:  []string{"eeaa2"},
		List:      []ContactCard{a, b},
	}
}

func (e Exemplar) CalendarEvent() CalendarEvent {
	cal := e.Calendar()
	return CalendarEvent{
		Id:          "aeZaik2faash",
		CalendarIds: map[string]bool{cal.Id: true},
		IsDraft:     false,
		IsOrigin:    true,
		Event: jscalendar.Event{
			Type: jscalendar.EventType,
			Object: jscalendar.Object{
				CommonObject: jscalendar.CommonObject{
					Uid:                    "dda22c7e-7674-4811-ae2e-2cc1ac605f5c",
					ProdId:                 "Groupware//1.0",
					Created:                "2026-04-01T15:29:12.912Z",
					Updated:                "2026-04-01T15:35:44.091Z",
					Title:                  "James Holden's Intronisation Ceremony",
					Description:            "James Holden will be confirmed as the President of the Transport Union, in room 2201 on station TSL-5.",
					DescriptionContentType: "text/plain",
					Links: map[string]jscalendar.Link{
						"aig1oh": {
							Type:        jscalendar.LinkType,
							Href:        "https://expanse.fandom.com/wiki/TSL-5",
							ContentType: "text/html",
							Display:     "TSL-5",
							Title:       "TSL-5",
						},
					},
					Locale:   "en-US",
					Keywords: map[string]bool{"union": true},
					Categories: map[string]bool{
						"meeting": true,
					},
					Color: "#ff0000",
				},
				ShowWithoutTime: false,
				Locations: map[string]jscalendar.Location{
					"eigha6": {
						Type: jscalendar.LocationType,
						Name: "Room 2201",
						LocationTypes: map[jscalendar.LocationTypeOption]bool{
							jscalendar.LocationTypeOptionOffice: true,
						},
						Coordinates: "geo:40.7495,-73.9681",
						Links: map[string]jscalendar.Link{
							"ohb6qu": {
								Type:        jscalendar.LinkType,
								Href:        "https://nss.org/what-is-l5/",
								ContentType: "text/html",
								Display:     "Lagrange Point 5",
								Title:       "Lagrange Point 5",
							},
						},
					},
				},
				Sequence:       0,
				MainLocationId: "eigha6",
				VirtualLocations: map[string]jscalendar.VirtualLocation{
					"eec4ei": {
						Type: jscalendar.VirtualLocationType,
						Name: "OpenTalk",
						Uri:  "https://earth.gov.example.com/opentalk/l5/2022",
						Features: map[jscalendar.VirtualLocationFeature]bool{
							jscalendar.VirtualLocationFeatureVideo:  true,
							jscalendar.VirtualLocationFeatureScreen: true,
							jscalendar.VirtualLocationFeatureAudio:  true,
						},
					},
				},
				Priority:       1,
				FreeBusyStatus: jscalendar.FreeBusyStatusBusy,
				Privacy:        jscalendar.PrivacyPublic,
				SentBy:         "avasarala@earth.gov.example.com",
				Participants: map[string]jscalendar.Participant{
					"xaku3f": {
						Type:  jscalendar.ParticipantType,
						Name:  "Christjen Avasarala",
						Email: "crissy@earth.gov.example.com",
						Kind:  jscalendar.ParticipantKindIndividual,
						Roles: map[jscalendar.Role]bool{
							jscalendar.RoleRequired: true,
							jscalendar.RoleChair:    true,
							jscalendar.RoleOwner:    true,
						},
						ParticipationStatus: jscalendar.ParticipationStatusAccepted,
					},
					"chao1a": {
						Type:  jscalendar.ParticipantType,
						Name:  "Camina Drummer",
						Email: "camina@opa.org.example.com",
						Kind:  jscalendar.ParticipantKindIndividual,
						Roles: map[jscalendar.Role]bool{
							jscalendar.RoleRequired: true,
						},
						ParticipationStatus:  jscalendar.ParticipationStatusAccepted,
						ParticipationComment: "I'll definitely be there",
						ExpectReply:          true,
						InvitedBy:            "xaku3f",
					},
					"ees0oo": {
						Type:  jscalendar.ParticipantType,
						Name:  "James Holden",
						Email: "james.holden@rocinante.space",
						Kind:  jscalendar.ParticipantKindIndividual,
						Roles: map[jscalendar.Role]bool{
							jscalendar.RoleRequired: true,
						},
						ParticipationStatus: jscalendar.ParticipationStatusAccepted,
						ExpectReply:         true,
						InvitedBy:           "xaku3f",
					},
				},
				Alerts: map[string]jscalendar.Alert{
					"kus9fa": {
						Type:   jscalendar.AlertType,
						Action: jscalendar.AlertActionDisplay,
						Trigger: jscalendar.OffsetTrigger{
							Type:       jscalendar.OffsetTriggerType,
							Offset:     "-PT1H",
							RelativeTo: jscalendar.RelativeToStart,
						},
					},
					"lohve9": {
						Type:   jscalendar.AlertType,
						Action: jscalendar.AlertActionDisplay,
						Trigger: jscalendar.OffsetTrigger{
							Type:       jscalendar.OffsetTriggerType,
							Offset:     "-PT10M",
							RelativeTo: jscalendar.RelativeToStart,
						},
					},
				},
				MayInviteOthers: true,
				HideAttendees:   false,
			},
		},
	}
}

func (e Exemplar) CalendarEventGetResponse() CalendarEventGetResponse {
	ev := e.CalendarEvent()
	return CalendarEventGetResponse{
		AccountId: e.AccountId,
		State:     "zah1ooj0",
		NotFound:  []string{"eea9"},
		List: []CalendarEvent{
			ev,
		},
	}
}

func (e Exemplar) AddressBookChanges() AddressBookChanges {
	a := e.AddressBook()
	return AddressBookChanges{
		OldState:       "eebees6o",
		NewState:       "gae1iey0",
		HasMoreChanges: true,
		Created:        []AddressBook{a},
		Destroyed:      []string{"l9fn"},
	}
}

func (e Exemplar) ContactCardChanges() (ContactCardChanges, string, string) {
	c := e.ContactCard()
	return ContactCardChanges{
		OldState:       "xai3iiraipoo",
		NewState:       "ni7thah7eeY4",
		HasMoreChanges: true,
		Created:        []ContactCard{c},
		Destroyed:      []string{"eaae", "bcba"},
	}, "A created ContactCard and two deleted ones", "created"
}

func (e Exemplar) OtherContactCardChanges() (ContactCardChanges, string, string) {
	c := e.ContactCard()
	return ContactCardChanges{
		OldState:       "xai3iiraipoo",
		NewState:       "ni7thah7eeY4",
		HasMoreChanges: false,
		Updated:        []ContactCard{c},
	}, "An updated ContactCard", "updated"
}

func (e Exemplar) EmailChanges() EmailChanges {
	emails := []Email{e.Email()}
	return EmailChanges{
		OldState:       "xai3iiraipoo",
		NewState:       "ni7thah7eeY4",
		HasMoreChanges: true,
		Created:        emails,
		Destroyed:      []string{"mmnan", "moxzz"},
	}
}

func (e Exemplar) Changes() (ObjectChanges, string, string) {
	return ObjectChanges{
		MaxChanges: 3,
		Mailboxes: &MailboxChangesResponse{
			AccountId:      e.AccountId,
			OldState:       "n",
			NewState:       "rafrag",
			HasMoreChanges: true,
			Created:        []string{"d", "e", "a"},
		},
		Emails: &EmailChangesResponse{
			AccountId:      e.AccountId,
			OldState:       "n",
			NewState:       "rafrag",
			HasMoreChanges: true,
			Created:        []string{"bmaaaaal", "hqaaaab2", "hqaaaab0"},
		},
		Calendars: &CalendarChangesResponse{
			AccountId:      e.AccountId,
			OldState:       "n",
			NewState:       "sci",
			HasMoreChanges: false,
			Created:        []string{"b"},
		},
		Events: &CalendarEventChangesResponse{
			AccountId:      e.AccountId,
			OldState:       "n",
			NewState:       "sci",
			HasMoreChanges: false,
		},
		Addressbooks: &AddressBookChangesResponse{
			AccountId:      e.AccountId,
			OldState:       "n",
			NewState:       "sb2",
			HasMoreChanges: false,
			Created:        []string{"b", "c"},
		},
		Contacts: &ContactCardChangesResponse{
			AccountId:      e.AccountId,
			OldState:       "n",
			NewState:       "rbsxqeay",
			HasMoreChanges: true,
			Created:        []string{"fq", "fr", "fs"},
		},
	}, "A set of changes to objects", "changes"
}

func (e Exemplar) Objects() Objects {
	mailboxes := e.MailboxGetResponse()
	emails := e.EmailGetResponse()
	calendars := e.CalendarGetResponse()
	events := e.CalendarEventGetResponse()
	addressbooks := e.AddressBookGetResponse()
	contacts := e.ContactCardGetResponse()
	quotas := e.QuotaGetResponse()
	identities := e.IdentityGetResponse()
	emailSubmissions := e.EmailSubmissionGetResponse()
	return Objects{
		Mailboxes:        &mailboxes,
		Emails:           &emails,
		Calendars:        &calendars,
		Events:           &events,
		Addressbooks:     &addressbooks,
		Contacts:         &contacts,
		Quotas:           &quotas,
		Identities:       &identities,
		EmailSubmissions: &emailSubmissions,
	}
}

func copyTo[B any, A any](a A) B {
	if b, err := json.Marshal(a); err != nil {
		panic(err)
	} else {
		var t B
		if err := json.Unmarshal(b, &t); err != nil {
			panic(err)
		} else {
			return t
		}
	}
}
