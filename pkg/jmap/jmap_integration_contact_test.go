package jmap

import (
	"math/rand"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func TestContacts(t *testing.T) {
	if skip(t) {
		return
	}

	count := uint(20 + rand.Intn(30))

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	accountId, addressbookId, expectedContactCardsById, boxes, err := s.fillContacts(t, count)
	require.NoError(err)
	require.NotEmpty(accountId)
	require.NotEmpty(addressbookId)

	allTrue(t, boxes, "mediaWithBlobId")

	filter := ContactCardFilterCondition{
		InAddressBook: addressbookId,
	}
	sortBy := []ContactCardComparator{
		{Property: jscontact.ContactCardPropertyCreated, IsAscending: true},
	}

	contactsByAccount, _, _, _, err := s.client.QueryContactCards([]string{accountId}, s.session, t.Context(), s.logger, "", filter, sortBy, 0, 0)
	require.NoError(err)

	require.Len(contactsByAccount, 1)
	require.Contains(contactsByAccount, accountId)
	contacts := contactsByAccount[accountId]
	require.Len(contacts, int(count))

	for _, actual := range contacts {
		expected, ok := expectedContactCardsById[actual.Id]
		require.True(ok, "failed to find created contact by its id")
		matchContact(t, actual, expected)
	}
}

func matchContact(t *testing.T, actual jscontact.ContactCard, expected jscontact.ContactCard) {
	require.Equal(t, expected, actual)
}

type ContactsBoxes struct {
	nicknames            bool
	secondaryEmails      bool
	secondaryAddress     bool
	phones               bool
	onlineService        bool
	preferredLanguage    bool
	mediaWithBlobId      bool
	mediaWithDataUri     bool
	mediaWithExternalUri bool
	organization         bool
	cryptoKey            bool
	link                 bool
}

var streetNumberRegex = regexp.MustCompile(`^(\d+)\s+(.+)$`)

func (s *StalwartTest) fillContacts(
	t *testing.T,
	count uint,
) (string, string, map[string]jscontact.ContactCard, ContactsBoxes, error) {
	require := require.New(t)
	c, err := NewTestJmapClient(s.session, s.username, s.password, true, true)
	require.NoError(err)
	defer c.Close()

	boxes := ContactsBoxes{}

	printer := func(s string) { log.Println(s) }

	accountId := c.session.PrimaryAccounts.Contacts
	require.NotEmpty(accountId, "no primary account for contacts in session")

	addressbookId := ""
	{
		addressBooksById, err := c.objectsById(accountId, AddressBookType, JmapContacts)
		require.NoError(err)

		for id, addressbook := range addressBooksById {
			if isDefault, ok := addressbook["isDefault"]; ok {
				if isDefault.(bool) {
					addressbookId = id
					break
				}
			}
		}
	}
	require.NotEmpty(addressbookId)

	u := true

	filled := map[string]jscontact.ContactCard{}
	for i := range count {
		person := gofakeit.Person()
		nameMap, nameObj := createName(person, u)
		contact := map[string]any{
			"@type":          "Card",
			"version":        "1.0",
			"addressBookIds": toBoolMap([]string{addressbookId}),
			"prodId":         productName,
			"language":       pickLanguage(),
			"kind":           "individual",
			"name":           nameMap,
		}
		card := jscontact.ContactCard{
			//Type:           jscontact.ContactCardType,
			Version:        "1.0",
			AddressBookIds: toBoolMap([]string{addressbookId}),
			ProdId:         productName,
			Language:       contact["language"].(string),
			Kind:           jscontact.ContactCardKindIndividual,
			Name:           &nameObj,
		}

		if i%3 == 0 {
			nicknameMap, nicknameObj := createNickName(person, u)
			id := id()
			contact["nicknames"] = map[string]map[string]any{id: nicknameMap}
			card.Nicknames = map[string]jscontact.Nickname{id: nicknameObj}
			boxes.nicknames = true
		}

		{
			emailMaps := map[string]map[string]any{}
			emailObjs := map[string]jscontact.EmailAddress{}
			emailId := id()
			emailMap, emailObj := createEmail(person, 10, u)
			emailMaps[emailId] = emailMap
			emailObjs[emailId] = emailObj

			for i := range rand.Intn(3) {
				id := id()
				m, o := createSecondaryEmail(gofakeit.Email(), i*100, u)
				emailMaps[id] = m
				emailObjs[id] = o
				boxes.secondaryEmails = true
			}
			if len(emailMaps) > 0 {
				contact["emails"] = emailMaps
				card.Emails = emailObjs
			}
		}
		if err := propmap(i%2 == 0, 1, 2, contact, "phones", &card.Phones, func(i int, id string) (map[string]any, jscontact.Phone, error) {
			boxes.phones = true
			num := person.Contact.Phone
			if i > 0 {
				num = gofakeit.Phone()
			}
			var features map[jscontact.PhoneFeature]bool = nil
			if rand.Intn(3) < 2 {
				features = toBoolMapS(jscontact.PhoneFeatureMobile, jscontact.PhoneFeatureVoice, jscontact.PhoneFeatureVideo, jscontact.PhoneFeatureText)
			} else {
				features = toBoolMapS(jscontact.PhoneFeatureVoice, jscontact.PhoneFeatureMainNumber)
			}

			contexts := map[jscontact.PhoneContext]bool{jscontact.PhoneContextWork: true}
			if rand.Intn(2) < 1 {
				contexts[jscontact.PhoneContextPrivate] = true
			}
			tel := "tel:" + "+1" + num
			return map[string]any{
					"@type":    "Phone",
					"number":   tel,
					"features": structs.MapKeys(features, func(f jscontact.PhoneFeature) string { return string(f) }),
					"contexts": structs.MapKeys(contexts, func(c jscontact.PhoneContext) string { return string(c) }),
				}, untype(jscontact.Phone{
					Type:     jscontact.PhoneType,
					Number:   tel,
					Features: features,
					Contexts: contexts,
				}, u), nil
		}); err != nil {
			return "", "", nil, boxes, err
		}
		if err := propmap(i%5 < 4, 1, 2, contact, "addresses", &card.Addresses, func(i int, id string) (map[string]any, jscontact.Address, error) {
			var source *gofakeit.AddressInfo
			if i == 0 {
				source = person.Address
			} else {
				source = gofakeit.Address()
				boxes.secondaryAddress = true
			}
			components := []jscontact.AddressComponent{}
			m := streetNumberRegex.FindAllStringSubmatch(source.Street, -1)
			if m != nil {
				components = append(components, jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindName, Value: m[0][2]})
				components = append(components, jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindNumber, Value: m[0][1]})
			} else {
				components = append(components, jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindName, Value: source.Street})
			}
			components = append(components,
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindLocality, Value: source.City},
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindCountry, Value: source.Country},
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindRegion, Value: source.State},
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindPostcode, Value: source.Zip},
			)
			tz := pickRandom(timezones...)
			return map[string]any{
					"@type": "Address",
					"components": structs.Map(components, func(c jscontact.AddressComponent) map[string]string {
						return map[string]string{"kind": string(c.Kind), "value": c.Value}
					}),
					"defaultSeparator": ", ",
					"isOrdered":        true,
					"timeZone":         tz,
				}, untype(jscontact.Address{
					Type:             jscontact.AddressType,
					Components:       components,
					DefaultSeparator: ", ",
					IsOrdered:        true,
					TimeZone:         tz,
				}, u), nil
		}); err != nil {
			return "", "", nil, boxes, err
		}
		if err := propmap(i%2 == 0, 1, 2, contact, "onlineServices", &card.OnlineServices, func(i int, id string) (map[string]any, jscontact.OnlineService, error) {
			boxes.onlineService = true
			switch rand.Intn(3) {
			case 0:
				return map[string]any{
						"@type":   "OnlineService",
						"service": "Mastodon",
						"user":    "@" + person.Contact.Email,
						"uri":     "https://mastodon.example.com/@" + strings.ToLower(person.FirstName),
					}, untype(jscontact.OnlineService{
						Type:    jscontact.OnlineServiceType,
						Service: "Mastodon",
						User:    "@" + person.Contact.Email,
						Uri:     "https://mastodon.example.com/@" + strings.ToLower(person.FirstName),
					}, u), nil
			case 1:
				return map[string]any{
						"@type": "OnlineService",
						"uri":   "xmpp:" + person.Contact.Email,
					}, untype(jscontact.OnlineService{
						Type: jscontact.OnlineServiceType,
						Uri:  "xmpp:" + person.Contact.Email,
					}, u), nil
			default:
				return map[string]any{
						"@type":   "OnlineService",
						"service": "Discord",
						"user":    person.Contact.Email,
						"uri":     "https://discord.example.com/user/" + person.Contact.Email,
					}, untype(jscontact.OnlineService{
						Type:    jscontact.OnlineServiceType,
						Service: "Discord",
						User:    person.Contact.Email,
						Uri:     "https://discord.example.com/user/" + person.Contact.Email,
					}, u), nil
			}
		}); err != nil {
			return "", "", nil, boxes, err
		}

		if err := propmap(i%3 == 0, 1, 2, contact, "preferredLanguages", &card.PreferredLanguages, func(i int, id string) (map[string]any, jscontact.LanguagePref, error) {
			boxes.preferredLanguage = true
			lang := pickRandom("en", "fr", "de", "es", "it")
			contexts := pickRandoms1("work", "private")
			return map[string]any{
					"@type":    "LanguagePref",
					"language": lang,
					"contexts": toBoolMap(contexts),
					"pref":     i + 1,
				}, untype(jscontact.LanguagePref{
					Type:     jscontact.LanguagePrefType,
					Language: lang,
					Contexts: toBoolMap(structs.Map(contexts, func(s string) jscontact.LanguagePrefContext { return jscontact.LanguagePrefContext(s) })),
					Pref:     uint(i + 1),
				}, u), nil
		}); err != nil {
			return "", "", nil, boxes, err
		}

		if i%2 == 0 {
			organizationMaps := map[string]map[string]any{}
			organizationObjs := map[string]jscontact.Organization{}
			titleMaps := map[string]map[string]any{}
			titleObjs := map[string]jscontact.Title{}
			for range 1 + rand.Intn(2) {
				boxes.organization = true
				orgId := id()
				titleId := id()
				organizationMaps[orgId] = map[string]any{
					"@type":    "Organization",
					"name":     person.Job.Company,
					"contexts": toBoolMapS("work"),
				}
				organizationObjs[orgId] = untype(jscontact.Organization{
					Type:     jscontact.OrganizationType,
					Name:     person.Job.Company,
					Contexts: toBoolMapS(jscontact.OrganizationContextWork),
				}, u)
				titleMaps[titleId] = map[string]any{
					"@type":          "Title",
					"kind":           "title",
					"name":           person.Job.Title,
					"organizationId": orgId,
				}
				titleObjs[titleId] = untype(jscontact.Title{
					Type:           jscontact.TitleType,
					Kind:           jscontact.TitleKindTitle,
					Name:           person.Job.Title,
					OrganizationId: orgId,
				}, u)
			}
			contact["organizations"] = organizationMaps
			contact["titles"] = titleMaps
			card.Organizations = organizationObjs
			card.Titles = titleObjs
		}

		if err := propmap(i%2 == 0, 1, 1, contact, "cryptoKeys", &card.CryptoKeys, func(i int, id string) (map[string]any, jscontact.CryptoKey, error) {
			boxes.cryptoKey = true
			entity, err := openpgp.NewEntity(person.FirstName+" "+person.LastName, "test", person.Contact.Email, nil)
			if err != nil {
				return nil, jscontact.CryptoKey{}, err
			}
			var b bytes.Buffer
			err = entity.PrimaryKey.Serialize(&b)
			if err != nil {
				return nil, jscontact.CryptoKey{}, err
			}
			encoded := base64.RawStdEncoding.EncodeToString(b.Bytes())
			return map[string]any{
					"@type": "CryptoKey",
					"uri":   "data:application/pgp-keys;base64," + encoded,
				}, untype(jscontact.CryptoKey{
					Type: jscontact.CryptoKeyType,
					Uri:  "data:application/pgp-keys;base64," + encoded,
				}, u), nil
		}); err != nil {
			return "", "", nil, boxes, err
		}

		if err := propmap(i%2 == 0, 1, 2, contact, "media", &card.Media, func(i int, id string) (map[string]any, jscontact.Media, error) {
			label := fmt.Sprintf("photo-%d", 1000+rand.Intn(9000))
			switch rand.Intn(3) {
			case 0:
				boxes.mediaWithDataUri = true
				// use data uri
				//size := 16 + rand.Intn(512-16+1) // <- let's not do that right now, makes debugging errors very difficult due to the ASCII wall noise
				size := pickRandom(16, 24, 32, 48, 64)
				img := gofakeit.ImagePng(size, size)
				mime := "image/png"
				uri := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(img)
				contexts := toBoolMapS(jscontact.MediaContextPrivate)
				return map[string]any{
						"@type":     "Media",
						"kind":      string(jscontact.MediaKindPhoto),
						"uri":       uri,
						"mediaType": mime,
						"contexts":  structs.MapKeys(contexts, func(c jscontact.MediaContext) string { return string(c) }),
						"label":     label,
					}, untype(jscontact.Media{
						Type:      jscontact.MediaType,
						Kind:      jscontact.MediaKindPhoto,
						Uri:       uri,
						MediaType: mime,
						Contexts:  contexts,
						Label:     label,
					}, u), nil
			// currently not supported, reported as https://github.com/stalwartlabs/stalwart/issues/2431
			case -1: // change this to 1 to enable it again
				boxes.mediaWithBlobId = true
				size := pickRandom(16, 24, 32, 48, 64)
				img := gofakeit.ImageJpeg(size, size)
				blob, err := c.uploadBlob(accountId, img, "image/jpeg")
				if err != nil {
					return nil, jscontact.Media{}, err
				}
				contexts := toBoolMapS(jscontact.MediaContextPrivate)
				return map[string]any{
						"@type":    "Media",
						"kind":     string(jscontact.MediaKindPhoto),
						"blobId":   blob.BlobId,
						"contexts": structs.MapKeys(contexts, func(c jscontact.MediaContext) string { return string(c) }),
						"label":    label,
					}, untype(jscontact.Media{
						Type:      jscontact.MediaType,
						Kind:      jscontact.MediaKindPhoto,
						BlobId:    blob.BlobId,
						MediaType: blob.Type,
						Contexts:  contexts,
						Label:     label,
					}, u), nil

			default:
				boxes.mediaWithExternalUri = true
				// use external uri
				uri := picsum(128, 128)
				contexts := toBoolMapS(jscontact.MediaContextWork)
				return map[string]any{
						"@type":    "Media",
						"kind":     string(jscontact.MediaKindPhoto),
						"uri":      uri,
						"contexts": structs.MapKeys(contexts, func(c jscontact.MediaContext) string { return string(c) }),
						"label":    label,
					}, untype(jscontact.Media{
						Type:     jscontact.MediaType,
						Kind:     jscontact.MediaKindPhoto,
						Uri:      uri,
						Contexts: contexts,
						Label:    label,
					}, u), nil
			}
		}); err != nil {
			return "", "", nil, boxes, err
		}
		if err := propmap(i%2 == 0, 1, 1, contact, "links", &card.Links, func(i int, id string) (map[string]any, jscontact.Link, error) {
			boxes.link = true
			return map[string]any{
					"@type": "Link",
					"kind":  "contact",
					"uri":   "mailto:" + person.Contact.Email,
					"pref":  (i + 1) * 10,
				}, untype(jscontact.Link{
					Type: jscontact.LinkType,
					Kind: jscontact.LinkKindContact,
					Uri:  "mailto:" + person.Contact.Email,
					Pref: uint((i + 1) * 10),
				}, u), nil
		}); err != nil {
			return "", "", nil, boxes, err
		}

		id, err := s.CreateContact(c, accountId, contact)
		if err != nil {
			return "", "", nil, boxes, err
		}
		card.Id = id
		filled[id] = card
		printer(fmt.Sprintf("🧑🏻 created %*s/%v id=%v", int(math.Log10(float64(count))+1), strconv.Itoa(int(i+1)), count, id))
	}
	return accountId, addressbookId, filled, boxes, nil
}

func (s *StalwartTest) CreateContact(j *TestJmapClient, accountId string, contact map[string]any) (string, error) {
	return j.create1(accountId, ContactCardType, JmapContacts, contact)
}
