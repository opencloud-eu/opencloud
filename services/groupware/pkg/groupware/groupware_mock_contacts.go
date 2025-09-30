package groupware

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
)

func MustParse(text string) time.Time {
	t, err := time.Parse(time.RFC3339, text)
	if err != nil {
		panic(err)
	}
	return t
}

var A1 = jmap.AddressBook{
	Id:           "a1",
	Name:         "Contacts",
	Description:  "Your good old personal address book",
	SortOrder:    1,
	IsDefault:    true,
	IsSubscribed: true,
	MyRights: jmap.AddressBookRights{
		MayRead:   true,
		MayWrite:  true,
		MayAdmin:  true,
		MayDelete: true,
	},
}

var A2 = jmap.AddressBook{
	Id:           "a2",
	Name:         "Collected Contacts",
	Description:  "This address book contains the contacts that were collected when sending and receiving emails",
	SortOrder:    10,
	IsDefault:    false,
	IsSubscribed: true,
	MyRights: jmap.AddressBookRights{
		MayRead:   true,
		MayWrite:  false,
		MayAdmin:  false,
		MayDelete: false,
	},
}

var CaminaDrummerContact = jscontact.ContactCard{
	Type: jscontact.ContactCardType,
	Id:   "cc1",
	AddressBookIds: map[string]bool{
		A1.Id: true,
		A2.Id: true,
	},
	Version:  jscontact.JSContactVersion_1_0,
	Created:  MustParse("2025-09-30T11:00:12Z").UTC(),
	Updated:  MustParse("2025-09-30T11:00:12Z").UTC(),
	Kind:     jscontact.ContactCardKindIndividual,
	Language: "en-GB",
	ProdId:   "Mock 0.0",
	Uid:      "e8317f89-2a09-481d-8ce5-de3ab968dc63",
	Name: &jscontact.Name{
		Type: jscontact.NameType,
		Components: []jscontact.NameComponent{
			{
				Type:  jscontact.NameComponentType,
				Kind:  jscontact.NameComponentKindGiven,
				Value: "Camina",
			},
			{
				Type:  jscontact.NameComponentType,
				Kind:  jscontact.NameComponentKindSurname,
				Value: "Drummer",
			},
		},
		IsOrdered:        true,
		DefaultSeparator: " ",
		Full:             "Camina Drummer",
	},
	Nicknames: map[string]jscontact.Nickname{
		"n1": {
			Type: jscontact.NicknameType,
			Name: "Bosmang",
			Contexts: map[jscontact.NicknameContext]bool{
				jscontact.NicknameContextWork: true,
			},
			Pref: 1,
		},
	},
	Organizations: map[string]jscontact.Organization{
		"o1": {
			Type:   jscontact.OrganizationType,
			Name:   "Outer Planets Alliance",
			SortAs: "OPA",
			Contexts: map[jscontact.OrganizationContext]bool{
				jscontact.OrganizationContextWork: true,
			},
		},
	},
	SpeakToAs: &jscontact.SpeakToAs{
		Type:              jscontact.SpeakToAsType,
		GrammaticalGender: jscontact.GrammaticalGenderFeminine,
		Pronouns: map[string]jscontact.Pronouns{
			"p1": {
				Type:     jscontact.PronounsType,
				Pronouns: "she/her",
				Contexts: map[jscontact.PronounsContext]bool{
					jscontact.PronounsContextPrivate: true,
				},
				Pref: 1,
			},
		},
	},
	Titles: map[string]jscontact.Title{
		"t1": {
			Type:           jscontact.TitleType,
			Name:           "Bosmang",
			Kind:           jscontact.TitleKindTitle,
			OrganizationId: "o1",
		},
	},
	Emails: map[string]jscontact.EmailAddress{
		"e1": {
			Type:    jscontact.EmailAddressType,
			Address: "cdrummer@opa.org",
			Contexts: map[jscontact.EmailAddressContext]bool{
				jscontact.EmailAddressContextWork:    true,
				jscontact.EmailAddressContextPrivate: true,
			},
			Pref:  10,
			Label: "opa",
		},
		"e2": {
			Type:    jscontact.EmailAddressType,
			Address: "camina.drummer@ceres.net",
			Contexts: map[jscontact.EmailAddressContext]bool{
				jscontact.EmailAddressContextPrivate: true,
			},
			Pref: 20,
		},
	},
	OnlineServices: map[string]jscontact.OnlineService{
		"s1": {
			Type:    jscontact.OnlineServiceType,
			Service: "Ring Network",
			Uri:     "https://ring.example.com/contact/@cdrummer",
			User:    "@cdrummer18219",
			Contexts: map[jscontact.OnlineServiceContext]bool{
				jscontact.OnlineServiceContextPrivate: true,
				jscontact.OnlineServiceContextWork:    true,
			},
			Label: "ring",
		},
	},
	Phones: map[string]jscontact.Phone{
		"p1": {
			Type:   jscontact.PhoneType,
			Number: "+1-999-555-1234",
			Features: map[jscontact.PhoneFeature]bool{
				jscontact.PhoneFeatureMainNumber: true,
				jscontact.PhoneFeatureMobile:     true,
				jscontact.PhoneFeatureVoice:      true,
				jscontact.PhoneFeatureText:       true,
				jscontact.PhoneFeatureVideo:      true,
			},
			Contexts: map[jscontact.PhoneContext]bool{
				jscontact.PhoneContextPrivate: true,
				jscontact.PhoneContextWork:    true,
			},
			Pref:  1,
			Label: "main",
		},
	},
	PreferredLanguages: map[string]jscontact.LanguagePref{
		"en": {
			Type:     jscontact.LanguagePrefType,
			Language: "en-GB",
			Contexts: map[jscontact.LanguagePrefContext]bool{
				jscontact.LanguagePrefContextPrivate: true,
				jscontact.LanguagePrefContextWork:    true,
			},
		},
	},
	Calendars: map[string]jscontact.Calendar{
		"c1": {
			Type:      jscontact.CalendarType,
			Kind:      jscontact.CalendarKindCalendar,
			Uri:       "https://ceres.org/calendars/@cdrummer/c1",
			MediaType: "application/jscontact+json",
			Contexts: map[jscontact.CalendarContext]bool{
				jscontact.CalendarContextPrivate: true,
				jscontact.CalendarContextWork:    true,
			},
			Pref:  1,
			Label: "main",
		},
	},
	SchedulingAddresses: map[string]jscontact.SchedulingAddress{
		"sa1": {
			Type: jscontact.SchedulingAddressType,
			Uri:  "https://scheduling.example.com/@cdrummer/c1",
			Contexts: map[jscontact.SchedulingAddressContext]bool{
				jscontact.SchedulingAddressContextPrivate: true,
				jscontact.SchedulingAddressContextWork:    true,
			},
			Pref:  1,
			Label: "main",
		},
	},
	Addresses: map[string]jscontact.Address{
		"ad1": {
			Type: jscontact.AddressType,
			Components: []jscontact.AddressComponent{
				{
					Kind:  jscontact.AddressComponentKindNumber,
					Value: "12",
				},
				{
					Kind:  jscontact.AddressComponentKindSeparator,
					Value: " ",
				},
				{
					Kind:  jscontact.AddressComponentKindName,
					Value: "Gravity Street",
				},
				{
					Kind:  jscontact.AddressComponentKindLocality,
					Value: "Medina Station",
				},
				{
					Kind:  jscontact.AddressComponentKindRegion,
					Value: "Outer Belt",
				},
				{
					Kind:  jscontact.AddressComponentKindSeparator,
					Value: " ",
				},
				{
					Kind:  jscontact.AddressComponentKindPostcode,
					Value: "618291",
				},
				{
					Kind:  jscontact.AddressComponentKindCountry,
					Value: "Sol",
				},
			},
			IsOrdered:        true,
			DefaultSeparator: ", ",
			CountryCode:      "SOL",
			Coordinates:      "geo:43.6466107,-79.3889872",
			TimeZone:         "EDT",
			Contexts: map[jscontact.AddressContext]bool{
				jscontact.AddressContextDelivery: true,
				jscontact.AddressContextWork:     true,
			},
			Full: "12 Gravity Street, Medina Station, Outer Belt 618291, Sol",
			Pref: 1,
		},
	},
	CryptoKeys: map[string]jscontact.CryptoKey{
		"k1": {
			Type:      jscontact.CryptoKeyType,
			Uri:       "https://opa.org/keys/@cdrummer.gpg",
			MediaType: "application/pgp-keys",
			Contexts: map[jscontact.CryptoKeyContext]bool{
				jscontact.CryptoKeyContextPrivate: true,
				jscontact.CryptoKeyContextWork:    true,
			},
			Pref:  10,
			Label: "opa",
		},
	},
	Directories: map[string]jscontact.Directory{
		"d1": {
			Type:      jscontact.DirectoryType,
			Kind:      jscontact.DirectoryKindEntry,
			Uri:       "https://directory.opa.org/addrbook/cdrummer/Camina%20Drummer.vcf",
			MediaType: "text/vcard",
		},
		"d2": {
			Type: jscontact.DirectoryType,
			Kind: jscontact.DirectoryKindDirectory,
			Uri:  "ldap://ldap.opa.org/o=OPA,ou=Bosmangs",
			Pref: 1,
		},
	},
	Links: map[string]jscontact.Link{
		"l1": {
			Type: jscontact.LinkType,
			Kind: jscontact.LinkKindContact,
			Uri:  "mailto:contact@opa.org",
			Pref: 1,
		},
	},
	Media: map[string]jscontact.Media{
		"m1": {
			Type:      jscontact.MediaType,
			Kind:      jscontact.MediaKindPhoto,
			Uri:       "https://static.wikia.nocookie.net/expanse/images/c/c7/Tycho-stn-14.png/revision/latest/scale-to-width-down/1000?cb=20170225140521",
			MediaType: "image/png",
		},
	},
	Anniversaries: map[string]jscontact.Anniversary{
		"an1": {
			Type: jscontact.AnniversaryType,
			Kind: jscontact.AnniversaryKindBirth,
			Date: jscontact.PartialDate{
				Type:          jscontact.PartialDateType,
				Year:          1983,
				Month:         7,
				Day:           18,
				CalendarScale: jscontact.RscaleIso8601,
			},
		},
	},
	Keywords: map[string]bool{
		"bosmang": true,
		"opa":     true,
		"tycho":   true,
		"rebel":   true,
	},
	PersonalInfo: map[string]jscontact.PersonalInfo{
		"p1": {
			Type:  jscontact.PersonalInfoType,
			Kind:  jscontact.PersonalInfoKindExpertise,
			Value: "loyalty",
			Level: jscontact.PersonalInfoLevelHigh,
		},
	},
	Notes: map[string]jscontact.Note{
		"n1": {
			Type:    jscontact.NoteType,
			Created: MustParse("2025-09-30T11:00:12Z").UTC(),
			Author: &jscontact.Author{
				Type: jscontact.AuthorType,
				Name: "expanse.fandom.com",
				Uri:  "https://expanse.fandom.com/wiki/Camina_Drummer_(TV)",
			},
			Note: "Cammina Drummer is a strong-willed, pragmatic, and no-nonsense Belter captain. Having a strong connection to her roots and her cultural identity, Drummer is a Belter through and through: She is resilient and adaptable, treats her crew with respect and equality, and is committed to the Belter way of life, which involves hard work, communal life shared with others, and not taking anything for granted.",
		},
	},
}

var AndersonDawesContact = jscontact.ContactCard{
	Type: jscontact.ContactCardType,
	Id:   "cc2",
	AddressBookIds: map[string]bool{
		A1.Id: true,
	},
	Version:  jscontact.JSContactVersion_1_0,
	Created:  MustParse("2025-09-30T11:00:12Z").UTC(),
	Updated:  MustParse("2025-09-30T11:00:12Z").UTC(),
	Kind:     jscontact.ContactCardKindIndividual,
	Language: "en-GB",
	ProdId:   "Mock 0.0",
	Uid:      "3c1c478e-ac6c-4c2f-a01d-5e528015958d",
	Name: &jscontact.Name{
		Type: jscontact.NameType,
		Components: []jscontact.NameComponent{
			{
				Type:  jscontact.NameComponentType,
				Kind:  jscontact.NameComponentKindGiven,
				Value: "Anderson",
			},
			{
				Type:  jscontact.NameComponentType,
				Kind:  jscontact.NameComponentKindSurname,
				Value: "Dawes",
			},
		},
		IsOrdered:        true,
		DefaultSeparator: " ",
		Full:             "Anderson Dawes",
	},
	Organizations: map[string]jscontact.Organization{
		"o1": {
			Type:   jscontact.OrganizationType,
			Name:   "Outer Planets Alliance",
			SortAs: "OPA",
			Contexts: map[jscontact.OrganizationContext]bool{
				jscontact.OrganizationContextWork: true,
			},
		},
	},
	SpeakToAs: &jscontact.SpeakToAs{
		Type:              jscontact.SpeakToAsType,
		GrammaticalGender: jscontact.GrammaticalGenderMasculine,
		Pronouns: map[string]jscontact.Pronouns{
			"p1": {
				Type:     jscontact.PronounsType,
				Pronouns: "he/him",
				Contexts: map[jscontact.PronounsContext]bool{
					jscontact.PronounsContextPrivate: true,
				},
				Pref: 1,
			},
		},
	},
	Titles: map[string]jscontact.Title{
		"t1": {
			Type:           jscontact.TitleType,
			Name:           "President",
			Kind:           jscontact.TitleKindRole,
			OrganizationId: "o1",
		},
	},
	Emails: map[string]jscontact.EmailAddress{
		"e1": {
			Type:    jscontact.EmailAddressType,
			Address: "adawes@opa.org",
			Contexts: map[jscontact.EmailAddressContext]bool{
				jscontact.EmailAddressContextWork:    true,
				jscontact.EmailAddressContextPrivate: true,
			},
			Pref:  10,
			Label: "opa",
		},
	},
	OnlineServices: map[string]jscontact.OnlineService{
		"s1": {
			Type:    jscontact.OnlineServiceType,
			Service: "Ring Network",
			Uri:     "https://ring.example.com/contact/@adawes",
			User:    "@anderson.1882",
			Contexts: map[jscontact.OnlineServiceContext]bool{
				jscontact.OnlineServiceContextPrivate: true,
				jscontact.OnlineServiceContextWork:    true,
			},
			Label: "ring",
		},
	},
	Phones: map[string]jscontact.Phone{
		"p1": {
			Type:   jscontact.PhoneType,
			Number: "+1-999-555-5678",
			Features: map[jscontact.PhoneFeature]bool{
				jscontact.PhoneFeatureMainNumber: true,
				jscontact.PhoneFeatureMobile:     true,
				jscontact.PhoneFeatureVoice:      true,
			},
			Contexts: map[jscontact.PhoneContext]bool{
				jscontact.PhoneContextPrivate: true,
				jscontact.PhoneContextWork:    true,
			},
		},
	},
	PreferredLanguages: map[string]jscontact.LanguagePref{
		"en": {
			Type:     jscontact.LanguagePrefType,
			Language: "en-GB",
			Contexts: map[jscontact.LanguagePrefContext]bool{
				jscontact.LanguagePrefContextPrivate: true,
				jscontact.LanguagePrefContextWork:    true,
			},
		},
	},
	Calendars: map[string]jscontact.Calendar{
		"c5": {
			Type:      jscontact.CalendarType,
			Kind:      jscontact.CalendarKindCalendar,
			Uri:       "https://ceres.org/calendars/@adawes/c5",
			MediaType: "application/jscontact+json",
			Contexts: map[jscontact.CalendarContext]bool{
				jscontact.CalendarContextPrivate: true,
				jscontact.CalendarContextWork:    true,
			},
		},
	},
	SchedulingAddresses: map[string]jscontact.SchedulingAddress{
		"sa1": {
			Type: jscontact.SchedulingAddressType,
			Uri:  "mailto:adawes@opa.org",
			Contexts: map[jscontact.SchedulingAddressContext]bool{
				jscontact.SchedulingAddressContextPrivate: true,
				jscontact.SchedulingAddressContextWork:    true,
			},
		},
	},
	Addresses: map[string]jscontact.Address{
		"ad1": {
			Type: jscontact.AddressType,
			Components: []jscontact.AddressComponent{
				{
					Kind:  jscontact.AddressComponentKindNumber,
					Value: "9218",
				},
				{
					Kind:  jscontact.AddressComponentKindSeparator,
					Value: " ",
				},
				{
					Kind:  jscontact.AddressComponentKindName,
					Value: "Main Street",
				},
				{
					Kind:  jscontact.AddressComponentKindLocality,
					Value: "Ceres Station",
				},
				{
					Kind:  jscontact.AddressComponentKindSeparator,
					Value: " ",
				},
				{
					Kind:  jscontact.AddressComponentKindPostcode,
					Value: "87A",
				},
				{
					Kind:  jscontact.AddressComponentKindCountry,
					Value: "Ceres",
				},
			},
			IsOrdered:        true,
			DefaultSeparator: ", ",
			CountryCode:      "CRS",
			Coordinates:      "geo:43.6466107,-79.3889872",
			TimeZone:         "EDT",
			Contexts: map[jscontact.AddressContext]bool{
				jscontact.AddressContextWork: true,
			},
		},
	},
	CryptoKeys: map[string]jscontact.CryptoKey{
		"k1": {
			Type:      jscontact.CryptoKeyType,
			Uri:       "https://opa.org/keys/@adawes.gpg",
			MediaType: "application/pgp-keys",
			Contexts: map[jscontact.CryptoKeyContext]bool{
				jscontact.CryptoKeyContextPrivate: true,
				jscontact.CryptoKeyContextWork:    true,
			},
		},
	},
	Media: map[string]jscontact.Media{
		"m1": {
			Type:      jscontact.MediaType,
			Kind:      jscontact.MediaKindPhoto,
			Uri:       "https://static.wikia.nocookie.net/expanse/images/0/0b/S02E07-JaredHarris_as_AndersonDawes_01c.jpg/revision/latest?cb=20170621040250",
			MediaType: "image/png",
		},
	},
	Anniversaries: map[string]jscontact.Anniversary{
		"an1": {
			Type: jscontact.AnniversaryType,
			Kind: jscontact.AnniversaryKindBirth,
			Date: jscontact.Timestamp{
				Type: jscontact.TimestampType,
				Utc:  MustParse("1961-08-24T00:00:00Z"),
			},
		},
	},
	Keywords: map[string]bool{
		"opa":   true,
		"ceres": true,
		"rebel": true,
	},
}

var AllAddressBooks = []jmap.AddressBook{A1, A2}

var ContactsMapByAddressBookId = map[string][]jscontact.ContactCard{
	A1.Id: {
		CaminaDrummerContact,
		AndersonDawesContact,
	},
	A2.Id: {
		CaminaDrummerContact,
	},
}
