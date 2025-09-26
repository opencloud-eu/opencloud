package jscontact

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func jsoneq(t *testing.T, expected string, object any) {
	str, err := json.MarshalIndent(object, "", "")
	require.NoError(t, err)
	require.JSONEq(t, expected, string(str))
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
		Uri:       "https://opencloud.eu/calendar/d05779b6-9638-4694-9869-008a61df6025",
		MediaType: "application/jscontact+json",
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
		Uri:   "https://opa.org/cdrummer",
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
			"mobile": true,
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
	ts, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00")
	require.NoError(t, err)
	jsoneq(t, `{
		"@type": "Timestamp",
		"utc": "2025-09-25T18:26:14.094725532+02:00"
	}`, Timestamp{
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
		Date: PartialDate{
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
		Date: Timestamp{
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

func TestContactCard(t *testing.T) {
	created, err := time.Parse(time.RFC3339, "2025-09-25T18:26:14.094725532+02:00")
	require.NoError(t, err)

	updated, err := time.Parse(time.RFC3339, "2025-09-26T09:58:01+02:00")
	require.NoError(t, err)

	jsoneq(t, `{
		"@type": "Card",
		"kind": "group",
		"id": "20fba820-2f8e-432d-94f1-5abbb59d3ed7",
		"addressBookIds": {
			"79047052-ae0e-4299-8860-5bff1a139f3d": true,
			"44eb6105-08c1-458b-895e-4ad1149dfabd": true
		},
		"version": "1.0",
		"created": "2025-09-25T18:26:14.094725532+02:00",
		"language": "fr-BE",
		"members": {
			"314815dd-81c8-4640-aace-6dc83121616d": true,
			"c528b277-d8cb-45f2-b7df-1aa3df817463": true,
			"81dea240-c0a4-4929-82e7-79e713a8bbe4": true
		},
		"prodId": "OpenCloud Groupware 1.0",
		"relatedTo": {
			"urn:uid:ca9d2a62-e068-43b6-a470-46506976d505": {
				"@type": "Relation",
				"relation": {
					"contact": true
				}
			},
			"urn:uid:72183ec2-b218-4983-9c89-ff117eeb7c5e": {
				"relation": {
					"emergency": true,
					"spouse": true
				}
			}
		},
		"uid": "1091f2bb-6ae6-4074-bb64-df74071d7033",
		"updated": "2025-09-26T09:58:01+02:00",
		"name": {
			"@type": "Name",
			"components": [
				{"@type": "NameComponent", "value": "OpenCloud", "kind": "surname"},
				{"value": " ", "kind": "separator"},
				{"value": "Team", "kind": "surname2"}
			],
			"isOrdered": true,
			"defaultSeparator": ", ",
			"sortAs": {
				"surname": "OpenCloud Team"
			},
			"full": "OpenCloud Team"
		},
		"nicknames": {
			"a": {
				"@type": "Nickname",
				"name": "The Team",
				"contexts": {
					"work": true
				},
				"pref": 1
			}
		},
		"organizations": {
			"o": {
				"@type": "Organization",
				"name": "OpenCloud GmbH",
				"units": [
					{"@type": "OrgUnit", "name": "Marketing", "sortAs": "marketing"},
					{"@type": "OrgUnit", "name": "Sales"},
					{"name": "Operations", "sortAs": "ops"}
				],
				"sortAs": "opencloud",
				"contexts": {
					"work": true
				}
			}
		},
		"speakToAs": {
			"@type": "SpeakToAs",
			"grammaticalGender": "inanimate",
			"pronouns": {
				"p": {
					"@type": "Pronouns",
					"pronouns": "it",
					"contexts": {
						"work": true
					},
					"pref": 1
				}
			}
		},
		"titles": {
			"t": {
				"@type": "Title",
				"name": "The",
				"kind": "title",
				"organizationId": "o"
			}
		},
		"emails": {
			"e": {
				"@type": "EmailAddress",
				"address": "info@opencloud.eu.example.com",
				"contexts": {
					"work": true
				},
				"pref": 1,
				"label": "work"
			}
		},
		"onlineServices": {
			"s": {
				"@type": "OnlineService",
				"service": "The Misinformation Game",
				"uri": "https://misinfogame.com/91886aa0-3586-4ade-b9bb-ec031464a251",
				"user": "opencloudeu",
				"contexts": {
					"work": true
				},
				"pref": 1,
				"label": "imaginary"
			}
		},
		"phones": {
			"p": {
				"@type": "Phone",
				"number": "+1-804-222-1111",
				"features": {
					"voice": true,
					"text": true
				},
				"contexts": {
					"work": true
				},
				"pref": 1,
				"label": "imaginary"
			}
		},
		"preferredLanguages": {
			"wa": {
				"@type": "LanguagePref",
				"language": "wa-BE",
				"contexts": {
					"private": true
				},
				"pref": 1
			},
			"de": {
				"language": "de-DE",
				"contexts": {
					"work": true
				},
				"pref": 2
			}
		},
		"calendars": {
			"c": {
				"@type": "Calendar",
				"kind": "calendar",
				"uri": "https://opencloud.eu/calendars/521b032b-a2b3-4540-81b9-3f6bccacaab2",
				"mediaType": "application/jscontact+json",
				"contexts": {
					"work": true
				},
				"pref": 1,
				"label": "work"
			}
		},
		"schedulingAddresses": {
			"s": {
				"@type": "SchedulingAddress",
				"uri": "mailto:scheduling@opencloud.eu.example.com",
				"contexts": {
					"work": true
				},
				"pref": 1,
				"label": "work"
			}
		},
		"addresses": {
			"k26": {
				"@type": "Address",
				"components": [
					{"@type": "AddressComponent", "kind": "block", "value": "2-7"},
					{"kind": "separator", "value": "-"},
					{"kind": "number", "value": "2"},
					{"kind": "separator", "value": " "},
					{"kind": "district", "value": "Marunouchi"},
					{"kind": "locality", "value": "Chiyoda-ku"},
					{"kind": "region", "value": "Tokyo"},
					{"kind": "separator", "value": " "},
					{"kind": "postcode", "value": "100-8994"}
				],
				"isOrdered": true,
				"defaultSeparator": ", ",
				"full": "2-7-2 Marunouchi, Chiyoda-ku, Tokyo 100-8994",
				"countryCode": "JP",
				"coordinates": "geo:35.6796373,139.7616907",
				"timeZone": "JST",
				"contexts": {
					"delivery": true,
					"work": true
				},
				"pref": 2
			}
		},
		"cryptoKeys": {
			"k1": {
				"@type": "CryptoKey",
				"uri": "https://opencloud.eu.example.com/keys/d550f57c-582c-43cc-8d94-822bded9ab36",
				"mediaType": "application/pgp-keys",
				"contexts": {
					"work": true
				},
				"pref": 1,
				"label": "keys"
			}
		},
		"directories": {
			"d1": {
				"@type": "Directory",
				"kind": "entry",
				"uri": "https://opencloud.eu.example.com/addressbook/8c2f0363-af0a-4d16-a9d5-8a9cd885d722",
				"listAs": 1
			}
		},
		"links": {
			"r1": {
				"@type": "Link",
				"kind": "contact",
				"uri": "mailto:contact@opencloud.eu.example.com",
				"contexts": {
					"work": true
				}
			}
		},
		"media": {
			"m": {
				"@type": "Media",
				"kind": "logo",
				"uri": "https://opencloud.eu.example.com/opencloud.svg",
				"mediaType": "image/svg+xml",
				"contexts": {
					"work": true
				},
				"pref": 123,
				"label": "svg",
				"blobId": "53feefbabeb146fcbe3e59e91462fa5f"
			}
		},
		"anniversaries": {
			"birth": {
				"@type": "Anniversary",
				"kind": "birth",
				"date": {
					"@type": "PartialDate",
					"year": 2025,
					"month": 9,
					"day": 26,
					"calendarScale": "iso8601"
				}
			}
		},
		"keywords": {
			"imaginary": true,
			"test": true
		},
		"notes": {
			"n1": {
				"@type": "Note",
				"note": "This is a note.",
				"created": "2025-09-25T18:26:14.094725532+02:00",
				"author": {
					"@type": "Author",
					"name": "Test Data",
					"uri": "https://isbn.example.com/a461f292-6bf1-470e-b08d-f6b4b0223fe3"
				}
			}
		},
		"personalInfo": {
			"p1": {
				"@type": "PersonalInfo",
				"kind": "expertise",
				"value": "Clouds",
				"level": "high",
				"listAs": 1,
				"label": "experts"
			}
		},
		"localizations": {
			"fr": {
				"personalInfo": {
					"value": "Nuages"
				}
			}
		}
	}`, ContactCard{
		Type: ContactCardType,
		Kind: ContactCardKindGroup,
		Id:   "20fba820-2f8e-432d-94f1-5abbb59d3ed7",
		AddressBookIds: map[string]bool{
			"79047052-ae0e-4299-8860-5bff1a139f3d": true,
			"44eb6105-08c1-458b-895e-4ad1149dfabd": true,
		},
		Version:  JSContactVersion_1_0,
		Created:  created,
		Language: "fr-BE",
		Members: map[string]bool{
			"314815dd-81c8-4640-aace-6dc83121616d": true,
			"c528b277-d8cb-45f2-b7df-1aa3df817463": true,
			"81dea240-c0a4-4929-82e7-79e713a8bbe4": true,
		},
		ProdId: "OpenCloud Groupware 1.0",
		RelatedTo: map[string]Relation{
			"urn:uid:ca9d2a62-e068-43b6-a470-46506976d505": {
				Type: RelationType,
				Relation: map[Relationship]bool{
					RelationContact: true,
				},
			},
			"urn:uid:72183ec2-b218-4983-9c89-ff117eeb7c5e": {
				Relation: map[Relationship]bool{
					RelationEmergency: true,
					RelationSpouse:    true,
				},
			},
		},
		Uid:     "1091f2bb-6ae6-4074-bb64-df74071d7033",
		Updated: updated,
		Name: &Name{
			Type: NameType,
			Components: []NameComponent{
				{Type: NameComponentType, Value: "OpenCloud", Kind: NameComponentKindSurname},
				{Value: " ", Kind: NameComponentKindSeparator},
				{Value: "Team", Kind: NameComponentKindSurname2},
			},
			IsOrdered:        true,
			DefaultSeparator: ", ",
			SortAs: map[string]string{
				string(NameComponentKindSurname): "OpenCloud Team",
			},
			Full: "OpenCloud Team",
		},
		Nicknames: map[string]Nickname{
			"a": {
				Type: NicknameType,
				Name: "The Team",
				Contexts: map[NicknameContext]bool{
					NicknameContextWork: true,
				},
				Pref: 1,
			},
		},
		Organizations: map[string]Organization{
			"o": {
				Type: OrganizationType,
				Name: "OpenCloud GmbH",
				Units: []OrgUnit{
					{Type: OrgUnitType, Name: "Marketing", SortAs: "marketing"},
					{Type: OrgUnitType, Name: "Sales"},
					{Name: "Operations", SortAs: "ops"},
				},
				SortAs: "opencloud",
				Contexts: map[OrganizationContext]bool{
					OrganizationContextWork: true,
				},
			},
		},
		SpeakToAs: &SpeakToAs{
			Type:              SpeakToAsType,
			GrammaticalGender: GrammaticalGenderInanimate,
			Pronouns: map[string]Pronouns{
				"p": {
					Type:     PronounsType,
					Pronouns: "it",
					Contexts: map[PronounsContext]bool{
						PronounsContextWork: true,
					},
					Pref: 1,
				},
			},
		},
		Titles: map[string]Title{
			"t": {
				Type:           TitleType,
				Name:           "The",
				Kind:           TitleKindTitle,
				OrganizationId: "o",
			},
		},
		Emails: map[string]EmailAddress{
			"e": {
				Type:    EmailAddressType,
				Address: "info@opencloud.eu.example.com",
				Contexts: map[EmailAddressContext]bool{
					EmailAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		OnlineServices: map[string]OnlineService{
			"s": {
				Type:    OnlineServiceType,
				Service: "The Misinformation Game",
				Uri:     "https://misinfogame.com/91886aa0-3586-4ade-b9bb-ec031464a251",
				User:    "opencloudeu",
				Contexts: map[OnlineServiceContext]bool{
					OnlineServiceContextWork: true,
				},
				Pref:  1,
				Label: "imaginary",
			},
		},
		Phones: map[string]Phone{
			"p": {
				Type:   PhoneType,
				Number: "+1-804-222-1111",
				Features: map[PhoneFeature]bool{
					PhoneFeatureVoice: true,
					PhoneFeatureText:  true,
				},
				Contexts: map[PhoneContext]bool{
					PhoneContextWork: true,
				},
				Pref:  1,
				Label: "imaginary",
			},
		},
		PreferredLanguages: map[string]LanguagePref{
			"wa": {
				Type:     LanguagePrefType,
				Language: "wa-BE",
				Contexts: map[LanguagePrefContext]bool{
					LanguagePrefContextPrivate: true,
				},
				Pref: 1,
			},
			"de": {
				Language: "de-DE",
				Contexts: map[LanguagePrefContext]bool{
					LanguagePrefContextWork: true,
				},
				Pref: 2,
			},
		},
		Calendars: map[string]Calendar{
			"c": {
				Type:      CalendarType,
				Kind:      CalendarKindCalendar,
				Uri:       "https://opencloud.eu/calendars/521b032b-a2b3-4540-81b9-3f6bccacaab2",
				MediaType: "application/jscontact+json",
				Contexts: map[CalendarContext]bool{
					CalendarContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		SchedulingAddresses: map[string]SchedulingAddress{
			"s": {
				Type: SchedulingAddressType,
				Uri:  "mailto:scheduling@opencloud.eu.example.com",
				Contexts: map[SchedulingAddressContext]bool{
					SchedulingAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		Addresses: map[string]Address{
			"k26": {
				Type: AddressType,
				Components: []AddressComponent{
					{Type: AddressComponentType, Kind: AddressComponentKindBlock, Value: "2-7"},
					{Kind: AddressComponentKindSeparator, Value: "-"},
					{Kind: AddressComponentKindNumber, Value: "2"},
					{Kind: AddressComponentKindSeparator, Value: " "},
					{Kind: AddressComponentKindDistrict, Value: "Marunouchi"},
					{Kind: AddressComponentKindLocality, Value: "Chiyoda-ku"},
					{Kind: AddressComponentKindRegion, Value: "Tokyo"},
					{Kind: AddressComponentKindSeparator, Value: " "},
					{Kind: AddressComponentKindPostcode, Value: "100-8994"},
				},
				IsOrdered:        true,
				DefaultSeparator: ", ",
				Full:             "2-7-2 Marunouchi, Chiyoda-ku, Tokyo 100-8994",
				CountryCode:      "JP",
				Coordinates:      "geo:35.6796373,139.7616907",
				TimeZone:         "JST",
				Contexts: map[AddressContext]bool{
					AddressContextDelivery: true,
					AddressContextWork:     true,
				},
				Pref: 2,
			},
		},
		CryptoKeys: map[string]CryptoKey{
			"k1": {
				Type:      CryptoKeyType,
				Uri:       "https://opencloud.eu.example.com/keys/d550f57c-582c-43cc-8d94-822bded9ab36",
				MediaType: "application/pgp-keys",
				Contexts: map[CryptoKeyContext]bool{
					CryptoKeyContextWork: true,
				},
				Pref:  1,
				Label: "keys",
			},
		},
		Directories: map[string]Directory{
			"d1": {
				Type:   DirectoryType,
				Kind:   DirectoryKindEntry,
				Uri:    "https://opencloud.eu.example.com/addressbook/8c2f0363-af0a-4d16-a9d5-8a9cd885d722",
				ListAs: 1,
			},
		},
		Links: map[string]Link{
			"r1": {
				Type: LinkType,
				Kind: LinkKindContact,
				Contexts: map[LinkContext]bool{
					LinkContextWork: true,
				},
				Uri: "mailto:contact@opencloud.eu.example.com",
			},
		},
		Media: map[string]Media{
			"m": {
				Type:      MediaType,
				Kind:      MediaKindLogo,
				Uri:       "https://opencloud.eu.example.com/opencloud.svg",
				MediaType: "image/svg+xml",
				Contexts: map[MediaContext]bool{
					MediaContextWork: true,
				},
				Pref:   123,
				Label:  "svg",
				BlobId: "53feefbabeb146fcbe3e59e91462fa5f",
			},
		},
		Anniversaries: map[string]Anniversary{
			"birth": {
				Type: AnniversaryType,
				Kind: AnniversaryKindBirth,
				Date: PartialDate{
					Type:          PartialDateType,
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
		Notes: map[string]Note{
			"n1": {
				Type:    NoteType,
				Note:    "This is a note.",
				Created: created,
				Author: &Author{
					Type: AuthorType,
					Name: "Test Data",
					Uri:  "https://isbn.example.com/a461f292-6bf1-470e-b08d-f6b4b0223fe3",
				},
			},
		},
		PersonalInfo: map[string]PersonalInfo{
			"p1": {
				Type:   PersonalInfoType,
				Kind:   PersonalInfoKindExpertise,
				Value:  "Clouds",
				Level:  PersonalInfoLevelHigh,
				ListAs: 1,
				Label:  "experts",
			},
		},
		Localizations: map[string]PatchObject{
			"fr": {
				"personalInfo": PatchObject{
					"value": "Nuages",
				},
			},
		},
	})
}
