package jscontact

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func jsoneq[X any](t *testing.T, expected string, object X) {
	data, err := json.MarshalIndent(object, "", "")
	require.NoError(t, err)
	require.JSONEq(t, expected, string(data))

	var rec X
	err = json.Unmarshal(data, &rec)
	require.NoError(t, err)
	require.Equal(t, object, rec)
}

func TestCalendar(t *testing.T) {
	jsoneq(t, `{
		"@type": "Calendar",
		"kind": "calendar",
		"uri": "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025",
		"mediaType": "application/jscontact+json",
		"contexts": {
			"work": true
		},
		"label": "test"
	}`, Calendar{
		Type:      CalendarType,
		Kind:      CalendarKindCalendar,
		Uri:       "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025", //NOSONAR
		MediaType: "application/jscontact+json",                                         //NOSONAR
		Contexts: map[CalendarContext]bool{
			CalendarContextWork: true,
		},
		Pref:  0,
		Label: "test",
	})
}

func TestLink(t *testing.T) {
	jsoneq(t, `{
		"@type": "Link",
		"kind": "contact",
		"uri": "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025",
		"mediaType": "application/jscontact+json",
		"contexts": {
			"work": true
		},
		"label": "test"
	}`, Link{
		Type:      LinkType,
		Kind:      LinkKindContact,
		Uri:       "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025",
		MediaType: "application/jscontact+json",
		Contexts: map[LinkContext]bool{
			LinkContextWork: true,
		},
		Pref:  0,
		Label: "test",
	})
}

func TestCryptoKey(t *testing.T) {
	jsoneq(t, `{
		"@type": "CryptoKey",
		"uri": "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025.pgp",
		"mediaType": "application/pgp-keys",
		"contexts": {
			"work": true
		},
		"label": "test"
	}`, CryptoKey{
		Type:      CryptoKeyType,
		Uri:       "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025.pgp",
		MediaType: "application/pgp-keys",
		Contexts: map[CryptoKeyContext]bool{
			CryptoKeyContextWork: true,
		},
		Pref:  0,
		Label: "test",
	})
}

func TestDirectory(t *testing.T) {
	jsoneq(t, `{
		"@type": "Directory",
		"kind": "entry",
		"uri": "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025",
		"mediaType": "application/jscontact+json",
		"contexts": {
			"work": true
		},
		"label": "test",
		"listAs": 3
	}`, Directory{
		Type:      DirectoryType,
		Kind:      DirectoryKindEntry,
		Uri:       "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025",
		MediaType: "application/jscontact+json",
		Contexts: map[DirectoryContext]bool{
			DirectoryContextWork: true,
		},
		Pref:   0,
		Label:  "test",
		ListAs: 3,
	})
}

func TestMedia(t *testing.T) {
	jsoneq(t, `{
		"@type": "Media",
		"kind": "logo",
		"uri": "https://opencloud.eu/opencloud.svg",
		"mediaType": "image/svg+xml",
		"contexts": {
			"work": true
		},
		"label": "test",
		"blobId": "1d92cf97e32b42ceb5538f0804a41891"
	}`, Media{
		Type:      MediaType,
		Kind:      MediaKindLogo,
		Uri:       "https://opencloud.eu/opencloud.svg",
		MediaType: "image/svg+xml",
		Contexts: map[MediaContext]bool{
			MediaContextWork: true,
		},
		Pref:   0,
		Label:  "test",
		BlobId: "1d92cf97e32b42ceb5538f0804a41891",
	})
}

func TestRelation(t *testing.T) {
	jsoneq(t, `{
		"@type": "Relation",
		"relation": {
			"co-worker": true,
			"friend": true
		}
	}`, Relation{
		Type: RelationType,
		Relation: map[Relationship]bool{
			RelationCoWorker: true,
			RelationFriend:   true,
		},
	})
}

func TestNameComponent(t *testing.T) {
	jsoneq(t, `{
		"@type": "NameComponent",
		"value": "Robert",
		"kind": "given",
		"phonetic": "Bob"
	}`, NameComponent{
		Type:     NameComponentType,
		Value:    "Robert",
		Kind:     NameComponentKindGiven,
		Phonetic: "Bob",
	})
}

func TestNickname(t *testing.T) {
	jsoneq(t, `{
		"@type": "Nickname",
		"name": "Bob",
		"contexts": {
			"private": true
		},
		"pref": 3
	}`, Nickname{
		Type: NicknameType,
		Name: "Bob",
		Contexts: map[NicknameContext]bool{
			NicknameContextPrivate: true,
		},
		Pref: 3,
	})
}

func TestOrgUnit(t *testing.T) {
	jsoneq(t, `{
		"@type": "OrgUnit",
		"name": "Skynet",
		"sortAs": "SKY"
	}`, OrgUnit{
		Type:   OrgUnitType,
		Name:   "Skynet",
		SortAs: "SKY",
	})
}

func TestOrganization(t *testing.T) {
	jsoneq(t, `{
		"@type": "Organization",
		"name": "Cyberdyne",
		"sortAs": "CYBER",
		"units": [{
			"@type": "OrgUnit",
			"name": "Skynet",
			"sortAs": "SKY"
			}, {
			"@type": "OrgUnit",
			"name": "Cybernics"
			}
		],
		"contexts": {
			"work": true
		}
	}`, Organization{
		Type:   OrganizationType,
		Name:   "Cyberdyne",
		SortAs: "CYBER",
		Units: []OrgUnit{
			{
				Type:   OrgUnitType,
				Name:   "Skynet",
				SortAs: "SKY",
			},
			{
				Type: OrgUnitType,
				Name: "Cybernics",
			},
		},
		Contexts: map[OrganizationContext]bool{
			OrganizationContextWork: true,
		},
	})
}

func TestPronouns(t *testing.T) {
	jsoneq(t, `{
		"@type": "Pronouns",
		"pronouns": "they/them",
		"contexts": {
			"work": true,
			"private": true
		},
		"pref": 1
	}`, Pronouns{
		Type:     PronounsType,
		Pronouns: "they/them",
		Contexts: map[PronounsContext]bool{
			PronounsContextWork:    true,
			PronounsContextPrivate: true,
		},
		Pref: 1,
	})
}

func TestTitle(t *testing.T) {
	jsoneq(t, `{
		"@type": "Title",
		"name": "Doctor",
		"kind": "title",
		"organizationId": "407e1992-9a2b-4e4f-a11b-85a509a4b5ae"
	}`, Title{
		Type:           TitleType,
		Name:           "Doctor",
		Kind:           TitleKindTitle,
		OrganizationId: "407e1992-9a2b-4e4f-a11b-85a509a4b5ae",
	})
}

func TestSpeakToAs(t *testing.T) {
	jsoneq(t, `{
		"@type": "SpeakToAs",
		"grammaticalGender": "neuter",
		"pronouns": {
			"a": {
				"@type": "Pronouns",
				"pronouns": "they/them",
				"contexts": {
					"private": true
				},
				"pref": 1
			},
			"b": {
				"@type": "Pronouns",
				"pronouns": "he/him",
				"contexts": {
					"work": true
				},
				"pref": 99
			}
		}
	}`, SpeakToAs{
		Type:              SpeakToAsType,
		GrammaticalGender: GrammaticalGenderNeuter,
		Pronouns: map[string]Pronouns{
			"a": {
				Type:     PronounsType,
				Pronouns: "they/them",
				Contexts: map[PronounsContext]bool{
					PronounsContextPrivate: true,
				},
				Pref: 1,
			},
			"b": {
				Type:     PronounsType,
				Pronouns: "he/him",
				Contexts: map[PronounsContext]bool{
					PronounsContextWork: true,
				},
				Pref: 99,
			},
		},
	})
}

func TestName(t *testing.T) {
	jsoneq(t, `{
		"@type": "Name",
		"components": [
  			{ "@type": "NameComponent", "kind": "given", "value": "Diego", "phonetic": "/di\u02C8e\u026A\u0261əʊ/" },
    		{ "kind": "surname", "value": "Rivera" },
    		{ "kind": "surname2", "value": "Barrientos" }
		],
		"isOrdered": true,
		"defaultSeparator": " ",
		"full": "Diego Rivera Barrientos",
		"sortAs": {
			"surname": "Rivera Barrientos",
			"given": "Diego"
		}
	}`, Name{
		Type: NameType,
		Components: []NameComponent{
			{
				Type:     NameComponentType,
				Value:    "Diego",
				Kind:     NameComponentKindGiven,
				Phonetic: "/diˈeɪɡəʊ/",
			},
			{
				Value: "Rivera",
				Kind:  NameComponentKindSurname,
			},
			{
				Value: "Barrientos",
				Kind:  NameComponentKindSurname2,
			},
		},
		IsOrdered:        true,
		DefaultSeparator: " ",
		Full:             "Diego Rivera Barrientos",
		SortAs: map[string]string{
			string(NameComponentKindSurname): "Rivera Barrientos",
			string(NameComponentKindGiven):   "Diego",
		},
	})
}

func TestEmailAddress(t *testing.T) {
	jsoneq(t, `{
		"@type": "EmailAddress",
		"address": "camina@opa.org",
		"contexts": {
			"work": true,
			"private": true
		},
		"pref": 1,
		"label": "bosmang"
	}`, EmailAddress{
		Type:    EmailAddressType,
		Address: "camina@opa.org",
		Contexts: map[EmailAddressContext]bool{
			EmailAddressContextWork:    true,
			EmailAddressContextPrivate: true,
		},
		Pref:  1,
		Label: "bosmang",
	})
}

func TestOnlineService(t *testing.T) {
	jsoneq(t, `{
		"@type": "OnlineService",
		"service": "OPA Network",
		"contexts": {
			"work": true
		},
		"uri": "https://opa.org/cdrummer",
		"user": "cdrummer@opa.org",
		"pref": 12,
		"label": "opa"
	}`, OnlineService{
		Type:    OnlineServiceType,
		Service: "OPA Network",
		Contexts: map[OnlineServiceContext]bool{
			OnlineServiceContextWork: true,
		},
		Uri:   "https://opa.org/cdrummer", //NOSONAR
		User:  "cdrummer@opa.org",
		Pref:  12,
		Label: "opa",
	})
}

func TestPhone(t *testing.T) {
	jsoneq(t, `{
		"@type": "Phone",
		"number": "+15551234567",
		"features": {
			"text": true,
			"main-number": true,
			"cell": true,
			"video": true,
			"voice": true
		},
		"contexts": {
			"work": true,
			"private": true
		},
		"pref": 42,
		"label": "opa"
	}`, Phone{
		Type:   PhoneType,
		Number: "+15551234567",
		Features: map[PhoneFeature]bool{
			PhoneFeatureText:       true,
			PhoneFeatureMainNumber: true,
			PhoneFeatureMobile:     true,
			PhoneFeatureVideo:      true,
			PhoneFeatureVoice:      true,
		},
		Contexts: map[PhoneContext]bool{
			PhoneContextWork:    true,
			PhoneContextPrivate: true,
		},
		Pref:  42,
		Label: "opa",
	})
}

func TestLanguagePref(t *testing.T) {
	jsoneq(t, `{
		"@type": "LanguagePref",
		"language": "fr-BE",
		"contexts": {
			"private": true
		},
		"pref": 2
	}`, LanguagePref{
		Type:     LanguagePrefType,
		Language: "fr-BE",
		Contexts: map[LanguagePrefContext]bool{
			LanguagePrefContextPrivate: true,
		},
		Pref: 2,
	})
}

func TestSchedulingAddress(t *testing.T) {
	jsoneq(t, `{
		"@type": "SchedulingAddress",
		"uri": "mailto:camina@opa.org",
		"contexts": {
			"work": true
		},
		"pref": 3,
		"label": "opa"
	}`, SchedulingAddress{
		Type:  SchedulingAddressType,
		Uri:   "mailto:camina@opa.org",
		Label: "opa",
		Contexts: map[SchedulingAddressContext]bool{
			SchedulingAddressContextWork: true,
		},
		Pref: 3,
	})
}

func TestAddressComponent(t *testing.T) {
	jsoneq(t, `{
		"@type": "AddressComponent",
		"kind": "postcode",
		"value": "12345",
		"phonetic": "un-deux-trois-quatre-cinq"
	}`, AddressComponent{
		Type:     AddressComponentType,
		Kind:     AddressComponentKindPostcode,
		Value:    "12345",
		Phonetic: "un-deux-trois-quatre-cinq",
	})
}

func TestAddress(t *testing.T) {
	jsoneq(t, `{
		"@type": "Address",
		"contexts": {
			"delivery": true,
			"work": true
		},
		"components": [
			{"@type": "AddressComponent", "kind": "number", "value": "54321"},
			{"kind": "separator", "value": " "},
			{"kind": "name", "value": "Oak St"},
			{"kind": "locality", "value": "Reston"},
			{"kind": "region", "value": "VA"},
			{"kind": "separator", "value": " "},
			{"kind": "postcode", "value": "20190"},
			{"kind": "country", "value": "USA"}
		],
		"countryCode": "US",
		"defaultSeparator": ", ",
		"isOrdered": true
	}`, Address{
		Type: AddressType,
		Contexts: map[AddressContext]bool{
			AddressContextDelivery: true,
			AddressContextWork:     true,
		},
		Components: []AddressComponent{
			{Type: AddressComponentType, Kind: AddressComponentKindNumber, Value: "54321"},
			{Kind: AddressComponentKindSeparator, Value: " "},
			{Kind: AddressComponentKindName, Value: "Oak St"},
			{Kind: AddressComponentKindLocality, Value: "Reston"},
			{Kind: AddressComponentKindRegion, Value: "VA"},
			{Kind: AddressComponentKindSeparator, Value: " "},
			{Kind: AddressComponentKindPostcode, Value: "20190"},
			{Kind: AddressComponentKindCountry, Value: "USA"},
		},
		CountryCode:      "US",
		DefaultSeparator: ", ",
		IsOrdered:        true,
	})
}

func TestPartialDate(t *testing.T) {
	jsoneq(t, `{
		"@type": "PartialDate",
		"year": 2025,
		"month": 9,
		"day": 25,
		"calendarScale": "iso8601"
	}`, PartialDate{
		Type:          PartialDateType,
		Year:          2025,
		Month:         9,
		Day:           25,
		CalendarScale: "iso8601",
	})
}

func TestTimestamp(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00") //NOSONAR
	require.NoError(t, err)
	jsoneq(t, `{
		"@type": "Timestamp",
		"utc": "2025-09-25T18:26:14.094725532+02:00"
	}`, &Timestamp{
		Type: TimestampType,
		Utc:  ts,
	})
}

func TestAnniversaryWithPartialDate(t *testing.T) {
	jsoneq(t, `{
		"@type": "Anniversary",
		"kind": "birth",
		"date": {
			"@type": "PartialDate",
			"year": 2025,
			"month": 9,
			"day": 25
		}
	}`, Anniversary{
		Type: AnniversaryType,
		Kind: AnniversaryKindBirth,
		Date: &PartialDate{
			Type:  PartialDateType,
			Year:  2025,
			Month: 9,
			Day:   25,
		},
	})
}

func TestAnniversaryWithTimestamp(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00")
	require.NoError(t, err)

	jsoneq(t, `{
		"@type": "Anniversary",
		"kind": "birth",
		"date": {
			"@type": "Timestamp",
			"utc": "2025-09-25T18:26:14.094725532+02:00"
		}
	}`, Anniversary{
		Type: AnniversaryType,
		Kind: AnniversaryKindBirth,
		Date: &Timestamp{
			Type: TimestampType,
			Utc:  ts,
		},
	})
}

func TestAuthor(t *testing.T) {
	jsoneq(t, `{
		"@type": "Author",
		"name": "Camina Drummer",
		"uri": "https://opa.org/cdrummer"
	}`, Author{
		Type: AuthorType,
		Name: "Camina Drummer",
		Uri:  "https://opa.org/cdrummer",
	})
}

func TestNote(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00")
	require.NoError(t, err)

	jsoneq(t, `{
		"@type": "Note",
		"note": "this is a note",
		"created": "2025-09-25T18:26:14.094725532+02:00",
		"author": {
			"@type": "Author",
			"name": "Camina Drummer",
			"uri": "https://opa.org/cdrummer"
		}
	}`, Note{
		Type:    NoteType,
		Note:    "this is a note",
		Created: ts,
		Author: &Author{
			Type: AuthorType,
			Name: "Camina Drummer",
			Uri:  "https://opa.org/cdrummer",
		},
	})
}

func TestPersonalInfo(t *testing.T) {
	jsoneq(t, `{
		"@type": "PersonalInfo",
		"kind": "expertise",
		"value": "motivation",
		"level": "high",
		"listAs": 1,
		"label": "opa"
	}`, PersonalInfo{
		Type:   PersonalInfoType,
		Kind:   PersonalInfoKindExpertise,
		Value:  "motivation",
		Level:  PersonalInfoLevelHigh,
		ListAs: 1,
		Label:  "opa",
	})
}
