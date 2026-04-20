package groupware

import (
	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type ObjectType[T jmap.Foo, CH jmap.Change, CHS jmap.Changes[T]] struct {
	name                  string
	responseType          ResponseObjectType
	uriParamName          string
	containerUriParamName string
	accountFunc           func(r *Request) (bool, string, Response)
	failedToDeleteError   GroupwareError
}

var (
	Blob = ObjectType[jmap.Blob, jmap.BlobChange, jmap.BlobChanges]{
		name:                  "blob",
		responseType:          BlobResponseObjectType,
		uriParamName:          UriParamBlobId,
		containerUriParamName: "",
		accountFunc:           (*Request).needBloblWithAccount,
		failedToDeleteError:   ErrorServerUnavailable,
	}

	AddressBook = ObjectType[jmap.AddressBook, jmap.AddressBookChange, jmap.AddressBookChanges]{
		name:                  "address book",
		responseType:          AddressBookResponseObjectType,
		uriParamName:          UriParamAddressBookId,
		containerUriParamName: "",
		accountFunc:           (*Request).needContactWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteAddressBook,
	}

	Calendar = ObjectType[jmap.Calendar, jmap.CalendarChange, jmap.CalendarChanges]{
		name:                  "calendar",
		responseType:          CalendarResponseObjectType,
		uriParamName:          UriParamCalendarId,
		containerUriParamName: "",
		accountFunc:           (*Request).needCalendarWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteCalendar,
	}

	Contact = ObjectType[jmap.ContactCard, jmap.ContactCardChange, jmap.ContactCardChanges]{
		name:                  "contact",
		responseType:          ContactResponseObjectType,
		uriParamName:          UriParamContactId,
		containerUriParamName: UriParamCalendarId,
		accountFunc:           (*Request).needCalendarWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteContact,
	}

	Email = ObjectType[jmap.Email, jmap.EmailChange, jmap.EmailChanges]{
		name:                  "email",
		responseType:          EmailResponseObjectType,
		uriParamName:          UriParamEmailId,
		containerUriParamName: UriParamMailboxId,
		accountFunc:           (*Request).needMailWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteEmail,
	}

	Event = ObjectType[jmap.CalendarEvent, jmap.CalendarEventChange, jmap.CalendarEventChanges]{
		name:                  "event",
		responseType:          EventResponseObjectType,
		uriParamName:          UriParamEventId,
		containerUriParamName: UriParamCalendarId,
		accountFunc:           (*Request).needCalendarWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteEvent,
	}

	Identity = ObjectType[jmap.Identity, jmap.IdentityChange, jmap.IdentityChanges]{
		name:                  "identity",
		responseType:          IdentityResponseObjectType,
		uriParamName:          UriParamIdentityId,
		containerUriParamName: "",
		accountFunc:           (*Request).needMailWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteIdentity,
	}

	Mailbox = ObjectType[jmap.Mailbox, jmap.MailboxChange, jmap.MailboxChanges]{
		name:                  "mailbox",
		responseType:          MailboxResponseObjectType,
		uriParamName:          UriParamMailboxId,
		containerUriParamName: "",
		accountFunc:           (*Request).needMailWithAccount,
		failedToDeleteError:   ErrorFailedToDeleteMailbox,
	}

	Quota = ObjectType[jmap.Quota, jmap.QuotaChange, jmap.QuotaChanges]{
		name:                  "quota",
		responseType:          QuotaResponseObjectType,
		uriParamName:          "",
		containerUriParamName: "",
		accountFunc:           (*Request).needQuotaWithAccount,
		failedToDeleteError:   ErrorServerUnavailable,
	}

	VacationResponse = ObjectType[jmap.VacationResponse, jmap.VacationResponseChange, jmap.VacationResponseChanges]{
		name:                  "vacation response",
		responseType:          VacationResponseResponseObjectType,
		uriParamName:          "",
		containerUriParamName: "",
		accountFunc:           (*Request).needMailWithAccount,
		failedToDeleteError:   ErrorServerUnavailable,
	}
)
