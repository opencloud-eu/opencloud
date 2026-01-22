//go:build groupware_examples

package jmap

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Exampler struct {
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

var ExamplerInstance = Exampler{
	AccountId:                  "b",
	Username:                   "cdrummer",
	IdentityId:                 "aemua9ai",
	IdentityName:               "Camina Drummer",
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

func (e Exampler) SessionMailAccountCapabilities() SessionMailAccountCapabilities {
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

func (e Exampler) SessionSubmissionAccountCapabilities() SessionSubmissionAccountCapabilities {
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

func (e Exampler) SessionVacationResponseAccountCapabilities() SessionVacationResponseAccountCapabilities {
	return SessionVacationResponseAccountCapabilities{}
}

func (e Exampler) SessionSieveAccountCapabilities() SessionSieveAccountCapabilities {
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

func (e Exampler) SessionBlobAccountCapabilities() SessionBlobAccountCapabilities {
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

func (e Exampler) SessionQuotaAccountCapabilities() SessionQuotaAccountCapabilities {
	return SessionQuotaAccountCapabilities{}
}

func (e Exampler) SessionContactsAccountCapabilities() SessionContactsAccountCapabilities {
	return SessionContactsAccountCapabilities{
		MaxAddressBooksPerCard: 128,
		MayCreateAddressBook:   true,
	}
}

func (e Exampler) SessionCalendarsAccountCapabilities() SessionCalendarsAccountCapabilities {
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

func (e Exampler) SessionCalendarsParseAccountCapabilities() SessionCalendarsParseAccountCapabilities {
	return SessionCalendarsParseAccountCapabilities{}
}

func (e Exampler) sessionPrincipalsAccountCapabilities(accountId string) SessionPrincipalsAccountCapabilities {
	return SessionPrincipalsAccountCapabilities{
		CurrentUserPrincipalId: accountId,
	}
}

func (e Exampler) SessionPrincipalsAccountCapabilities() SessionPrincipalsAccountCapabilities {
	return e.sessionPrincipalsAccountCapabilities(e.AccountId)
}

func (e Exampler) SessionPrincipalAvailabilityAccountCapabilities() SessionPrincipalAvailabilityAccountCapabilities {
	return SessionPrincipalAvailabilityAccountCapabilities{
		MaxAvailabilityDuration: Duration("P52W1D"),
	}
}

func (e Exampler) SessionTasksAccountCapabilities() SessionTasksAccountCapabilities {
	return SessionTasksAccountCapabilities{
		MinDateTime:       LocalDate("0001-01-01T00:00:00Z"),
		MaxDateTime:       LocalDate("65534-12-31T23:59:59Z"),
		MayCreateTaskList: true,
	}
}

func (e Exampler) SessionTasksAlertsAccountCapabilities() SessionTasksAlertsAccountCapabilities {
	return SessionTasksAlertsAccountCapabilities{}
}

func (e Exampler) SessionTasksAssigneesAccountCapabilities() SessionTasksAssigneesAccountCapabilities {
	return SessionTasksAssigneesAccountCapabilities{}
}

func (e Exampler) SessionTasksRecurrencesAccountCapabilities() SessionTasksRecurrencesAccountCapabilities {
	return SessionTasksRecurrencesAccountCapabilities{
		MaxExpandedQueryDuration: Duration("P260W1D"),
	}
}

func (e Exampler) SessionTasksMultilingualAccountCapabilities() SessionTasksMultilingualAccountCapabilities {
	return SessionTasksMultilingualAccountCapabilities{}
}

func (e Exampler) SessionTasksCustomTimezonesAccountCapabilities() SessionTasksCustomTimezonesAccountCapabilities {
	return SessionTasksCustomTimezonesAccountCapabilities{}
}

func (e Exampler) SessionPrincipalsOwnerAccountCapabilities() SessionPrincipalsOwnerAccountCapabilities {
	return SessionPrincipalsOwnerAccountCapabilities{
		AccountIdForPrincipal: e.AccountId,
		PrincipalId:           e.AccountId,
	}
}

func (e Exampler) SessionMDNAccountCapabilities() SessionMDNAccountCapabilities {
	return SessionMDNAccountCapabilities{}
}

func (e Exampler) SessionAccountCapabilities() SessionAccountCapabilities {
	return e.sessionAccountCapabilities(e.AccountId)
}

func (e Exampler) sessionAccountCapabilities(accountId string) SessionAccountCapabilities {
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

func (e Exampler) Account() (Account, string) {
	return Account{
		Name:                e.Username,
		IsPersonal:          true,
		IsReadOnly:          false,
		AccountCapabilities: e.SessionAccountCapabilities(),
	}, "A personal account"
}

func (e Exampler) SharedAccount() (Account, string, string) {
	return Account{
		Name:                e.SharedAccountId,
		IsPersonal:          false,
		IsReadOnly:          true,
		AccountCapabilities: e.sessionAccountCapabilities(e.SharedAccountId),
	}, "A read-only shared account", "shared"
}

func (e Exampler) Accounts() []Account {
	a, _ := e.Account()
	s, _, _ := e.SharedAccount()
	return []Account{a, s}
}

func (e Exampler) Quota() Quota {
	return Quota{
		Id:           e.QuotaId,
		ResourceType: "octets",
		Scope:        "account",
		Used:         11696865,
		HardLimit:    20000000000,
		Name:         e.Username,
		Types: []ObjectType{
			EmailType,
			SieveScriptType,
			FileNodeType,
			CalendarEventType,
			ContactCardType,
		},
		Description: e.IdentityName,
		SoftLimit:   19000000000,
		WarnLimit:   10000000000,
	}
}

func (e Exampler) Quotas() []Quota {
	return []Quota{
		e.Quota(),
		{
			Id:           e.SharedQuotaId,
			ResourceType: "octets",
			Scope:        "account",
			Used:         29102918,
			HardLimit:    50000000000,
			Name:         e.SharedAccountId,
			Types: []ObjectType{
				EmailType,
				SieveScriptType,
				FileNodeType,
				CalendarEventType,
				ContactCardType,
			},
			Description: e.SharedAccountName,
			SoftLimit:   90000000000,
			WarnLimit:   100000000000,
		},
	}
}

func (e Exampler) Identity() Identity {
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

func (e Exampler) OtherIdentity() (Identity, string, string) {
	return Identity{
		Id:        e.OtherIdentityId,
		Name:      e.OtherIdentityName,
		Email:     e.OtherIdentityEmailAddress,
		MayDelete: false,
	}, "Another Identity", "other"
}

func (e Exampler) Identities() []Identity {
	a := e.Identity()
	b, _, _ := e.OtherIdentity()
	return []Identity{a, b}
}

func (e Exampler) Identity_req() Identity {
	return Identity{
		Name:  e.IdentityName,
		Email: e.EmailAddress,
		Bcc: &[]EmailAddress{
			{Name: e.BccName, Email: e.BccAddress},
		},
		TextSignature: &e.TextSignature,
	}
}

func (e Exampler) Thread() Thread {
	return Thread{
		Id:       e.ThreadId,
		EmailIds: e.EmailIds,
	}
}

func (e Exampler) MailboxInbox() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxInboxId,
		Name:          "Inbox",
		Role:          JmapMailboxRoleInbox,
		SortOrder:     intPtr(0),
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
		IsSubscribed: boolPtr(true),
	}, "An Inbox Mailbox", "inbox"
}

func (e Exampler) MailboxInboxProjects() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxProjectId,
		ParentId:      e.MailboxInboxId,
		Name:          "Projects",
		SortOrder:     intPtr(0),
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
		IsSubscribed: boolPtr(true),
	}, "A Projects Mailbox under the Inbox", "projects"
}

func (e Exampler) MailboxDrafts() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxDraftsId,
		Name:          "Drafts",
		Role:          JmapMailboxRoleDrafts,
		SortOrder:     intPtr(0),
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
		IsSubscribed: boolPtr(true),
	}, "A Drafts Mailbox", "drafts"
}

func (e Exampler) MailboxSent() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxSentId,
		Name:          "Sent Items",
		Role:          JmapMailboxRoleSent,
		SortOrder:     intPtr(0),
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
		IsSubscribed: boolPtr(true),
	}, "A Sent Mailbox", "sent"
}

func (e Exampler) MailboxJunk() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxJunkId,
		Name:          "Junk Mail",
		Role:          JmapMailboxRoleJunk,
		SortOrder:     intPtr(0),
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
		IsSubscribed: boolPtr(true),
	}, "A Junk Mailbox", "junk"
}

func (e Exampler) MailboxDeleted() (Mailbox, string, string) {
	return Mailbox{
		Id:            e.MailboxDeletedId,
		Name:          "Deleted Items",
		Role:          JmapMailboxRoleTrash,
		SortOrder:     intPtr(0),
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
		IsSubscribed: boolPtr(true),
	}, "A Trash Mailbox", "deleted"
}

func (e Exampler) Mailboxes() []Mailbox {
	a, _, _ := e.MailboxInbox()
	b, _, _ := e.MailboxDrafts()
	c, _, _ := e.MailboxSent()
	d, _, _ := e.MailboxJunk()
	f, _, _ := e.MailboxDeleted()
	g, _, _ := e.MailboxInboxProjects()
	return []Mailbox{a, b, c, d, f, g}
}

func SerializeExamples(e any) {
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
