package jmap

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/stretchr/testify/require"
)

func TestObjectNames(t *testing.T) { //NOSONAR
	require := require.New(t)
	objectTypeNames, err := parseConsts("github.com/opencloud-eu/opencloud/pkg/jmap", "Name", "ObjectTypeName")
	require.NoError(err)
	for n, v := range objectTypeNames {
		require.True(strings.HasSuffix(n, "Name"))
		prefix := n[0 : len(n)-len("Name")]
		require.Equal(prefix, v)
	}
}

func jsoneq[X any](t *testing.T, expected string, object X) {
	data, err := json.MarshalIndent(object, "", "")
	require.NoError(t, err)
	require.JSONEq(t, expected, string(data))

	var rec X
	err = json.Unmarshal(data, &rec)
	require.NoError(t, err)
	require.Equal(t, object, rec)
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
		Type: jscontact.ContactCardType,
		Kind: jscontact.ContactCardKindGroup,
		Id:   "20fba820-2f8e-432d-94f1-5abbb59d3ed7",
		AddressBookIds: map[string]bool{
			"79047052-ae0e-4299-8860-5bff1a139f3d": true,
			"44eb6105-08c1-458b-895e-4ad1149dfabd": true,
		},
		Version:  jscontact.JSContactVersion_1_0,
		Created:  created,
		Language: "fr-BE",
		Members: map[string]bool{
			"314815dd-81c8-4640-aace-6dc83121616d": true,
			"c528b277-d8cb-45f2-b7df-1aa3df817463": true,
			"81dea240-c0a4-4929-82e7-79e713a8bbe4": true,
		},
		ProdId: "OpenCloud Groupware 1.0",
		RelatedTo: map[string]jscontact.Relation{
			"urn:uid:ca9d2a62-e068-43b6-a470-46506976d505": {
				Type: jscontact.RelationType,
				Relation: map[jscontact.Relationship]bool{
					jscontact.RelationContact: true,
				},
			},
			"urn:uid:72183ec2-b218-4983-9c89-ff117eeb7c5e": {
				Relation: map[jscontact.Relationship]bool{
					jscontact.RelationEmergency: true,
					jscontact.RelationSpouse:    true,
				},
			},
		},
		Uid:     "1091f2bb-6ae6-4074-bb64-df74071d7033",
		Updated: updated,
		Name: &jscontact.Name{
			Type: jscontact.NameType,
			Components: []jscontact.NameComponent{
				{Type: jscontact.NameComponentType, Value: "OpenCloud", Kind: jscontact.NameComponentKindSurname},
				{Value: " ", Kind: jscontact.NameComponentKindSeparator},
				{Value: "Team", Kind: jscontact.NameComponentKindSurname2},
			},
			IsOrdered:        true,
			DefaultSeparator: ", ",
			SortAs: map[string]string{
				string(jscontact.NameComponentKindSurname): "OpenCloud Team",
			},
			Full: "OpenCloud Team",
		},
		Nicknames: map[string]jscontact.Nickname{
			"a": {
				Type: jscontact.NicknameType,
				Name: "The Team",
				Contexts: map[jscontact.NicknameContext]bool{
					jscontact.NicknameContextWork: true,
				},
				Pref: 1,
			},
		},
		Organizations: map[string]jscontact.Organization{
			"o": {
				Type: jscontact.OrganizationType,
				Name: "OpenCloud GmbH",
				Units: []jscontact.OrgUnit{
					{Type: jscontact.OrgUnitType, Name: "Marketing", SortAs: "marketing"},
					{Type: jscontact.OrgUnitType, Name: "Sales"},
					{Name: "Operations", SortAs: "ops"},
				},
				SortAs: "opencloud",
				Contexts: map[jscontact.OrganizationContext]bool{
					jscontact.OrganizationContextWork: true,
				},
			},
		},
		SpeakToAs: &jscontact.SpeakToAs{
			Type:              jscontact.SpeakToAsType,
			GrammaticalGender: jscontact.GrammaticalGenderInanimate,
			Pronouns: map[string]jscontact.Pronouns{
				"p": {
					Type:     jscontact.PronounsType,
					Pronouns: "it",
					Contexts: map[jscontact.PronounsContext]bool{
						jscontact.PronounsContextWork: true,
					},
					Pref: 1,
				},
			},
		},
		Titles: map[string]jscontact.Title{
			"t": {
				Type:           jscontact.TitleType,
				Name:           "The",
				Kind:           jscontact.TitleKindTitle,
				OrganizationId: "o",
			},
		},
		Emails: map[string]jscontact.EmailAddress{
			"e": {
				Type:    jscontact.EmailAddressType,
				Address: "info@opencloud.eu.example.com",
				Contexts: map[jscontact.EmailAddressContext]bool{
					jscontact.EmailAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		OnlineServices: map[string]jscontact.OnlineService{
			"s": {
				Type:    jscontact.OnlineServiceType,
				Service: "The Misinformation Game",
				Uri:     "https://misinfogame.com/91886aa0-3586-4ade-b9bb-ec031464a251",
				User:    "opencloudeu",
				Contexts: map[jscontact.OnlineServiceContext]bool{
					jscontact.OnlineServiceContextWork: true,
				},
				Pref:  1,
				Label: "imaginary",
			},
		},
		Phones: map[string]jscontact.Phone{
			"p": {
				Type:   jscontact.PhoneType,
				Number: "+1-804-222-1111",
				Features: map[jscontact.PhoneFeature]bool{
					jscontact.PhoneFeatureVoice: true,
					jscontact.PhoneFeatureText:  true,
				},
				Contexts: map[jscontact.PhoneContext]bool{
					jscontact.PhoneContextWork: true,
				},
				Pref:  1,
				Label: "imaginary",
			},
		},
		PreferredLanguages: map[string]jscontact.LanguagePref{
			"wa": {
				Type:     jscontact.LanguagePrefType,
				Language: "wa-BE",
				Contexts: map[jscontact.LanguagePrefContext]bool{
					jscontact.LanguagePrefContextPrivate: true,
				},
				Pref: 1,
			},
			"de": {
				Language: "de-DE",
				Contexts: map[jscontact.LanguagePrefContext]bool{
					jscontact.LanguagePrefContextWork: true,
				},
				Pref: 2,
			},
		},
		Calendars: map[string]jscontact.Calendar{
			"c": {
				Type:      jscontact.CalendarType,
				Kind:      jscontact.CalendarKindCalendar,
				Uri:       "https://opencloud.eu/calendars/521b032b-a2b3-4540-81b9-3f6bccacaab2",
				MediaType: "application/jscontact+json",
				Contexts: map[jscontact.CalendarContext]bool{
					jscontact.CalendarContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		SchedulingAddresses: map[string]jscontact.SchedulingAddress{
			"s": {
				Type: jscontact.SchedulingAddressType,
				Uri:  "mailto:scheduling@opencloud.eu.example.com",
				Contexts: map[jscontact.SchedulingAddressContext]bool{
					jscontact.SchedulingAddressContextWork: true,
				},
				Pref:  1,
				Label: "work",
			},
		},
		Addresses: map[string]jscontact.Address{
			"k26": {
				Type: jscontact.AddressType,
				Components: []jscontact.AddressComponent{
					{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindBlock, Value: "2-7"},
					{Kind: jscontact.AddressComponentKindSeparator, Value: "-"},
					{Kind: jscontact.AddressComponentKindNumber, Value: "2"},
					{Kind: jscontact.AddressComponentKindSeparator, Value: " "},
					{Kind: jscontact.AddressComponentKindDistrict, Value: "Marunouchi"},
					{Kind: jscontact.AddressComponentKindLocality, Value: "Chiyoda-ku"},
					{Kind: jscontact.AddressComponentKindRegion, Value: "Tokyo"},
					{Kind: jscontact.AddressComponentKindSeparator, Value: " "},
					{Kind: jscontact.AddressComponentKindPostcode, Value: "100-8994"},
				},
				IsOrdered:        true,
				DefaultSeparator: ", ",
				Full:             "2-7-2 Marunouchi, Chiyoda-ku, Tokyo 100-8994",
				CountryCode:      "JP",
				Coordinates:      "geo:35.6796373,139.7616907",
				TimeZone:         "JST",
				Contexts: map[jscontact.AddressContext]bool{
					jscontact.AddressContextDelivery: true,
					jscontact.AddressContextWork:     true,
				},
				Pref: 2,
			},
		},
		CryptoKeys: map[string]jscontact.CryptoKey{
			"k1": {
				Type:      jscontact.CryptoKeyType,
				Uri:       "https://opencloud.eu.example.com/keys/d550f57c-582c-43cc-8d94-822bded9ab36",
				MediaType: "application/pgp-keys",
				Contexts: map[jscontact.CryptoKeyContext]bool{
					jscontact.CryptoKeyContextWork: true,
				},
				Pref:  1,
				Label: "keys",
			},
		},
		Directories: map[string]jscontact.Directory{
			"d1": {
				Type:   jscontact.DirectoryType,
				Kind:   jscontact.DirectoryKindEntry,
				Uri:    "https://opencloud.eu.example.com/addressbook/8c2f0363-af0a-4d16-a9d5-8a9cd885d722",
				ListAs: 1,
			},
		},
		Links: map[string]jscontact.Link{
			"r1": {
				Type: jscontact.LinkType,
				Kind: jscontact.LinkKindContact,
				Contexts: map[jscontact.LinkContext]bool{
					jscontact.LinkContextWork: true,
				},
				Uri: "mailto:contact@opencloud.eu.example.com",
			},
		},
		Media: map[string]jscontact.Media{
			"m": {
				Type:      jscontact.MediaType,
				Kind:      jscontact.MediaKindLogo,
				Uri:       "https://opencloud.eu.example.com/opencloud.svg",
				MediaType: "image/svg+xml",
				Contexts: map[jscontact.MediaContext]bool{
					jscontact.MediaContextWork: true,
				},
				Pref:   123,
				Label:  "svg",
				BlobId: "53feefbabeb146fcbe3e59e91462fa5f",
			},
		},
		Anniversaries: map[string]jscontact.Anniversary{
			"birth": {
				Type: jscontact.AnniversaryType,
				Kind: jscontact.AnniversaryKindBirth,
				Date: &jscontact.PartialDate{
					Type:          jscontact.PartialDateType,
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
		Notes: map[string]jscontact.Note{
			"n1": {
				Type:    jscontact.NoteType,
				Note:    "This is a note.",
				Created: created,
				Author: &jscontact.Author{
					Type: jscontact.AuthorType,
					Name: "Test Data",
					Uri:  "https://isbn.example.com/a461f292-6bf1-470e-b08d-f6b4b0223fe3",
				},
			},
		},
		PersonalInfo: map[string]jscontact.PersonalInfo{
			"p1": {
				Type:   jscontact.PersonalInfoType,
				Kind:   jscontact.PersonalInfoKindExpertise,
				Value:  "Clouds",
				Level:  jscontact.PersonalInfoLevelHigh,
				ListAs: 1,
				Label:  "experts",
			},
		},
		Localizations: map[string]jscontact.PatchObject{
			"fr": {
				"personalInfo": map[string]any{
					"value": "Nuages",
				},
			},
		},
	})
}
