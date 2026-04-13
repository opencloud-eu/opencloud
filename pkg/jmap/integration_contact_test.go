package jmap

import (
	golog "log"
	"math/rand"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

const (
	// currently not supported, reported as https://github.com/stalwartlabs/stalwart/issues/2431
	EnableMediaWithBlobId = false
)

type AddressBookBoxes struct {
	sharedReadOnly  bool
	sharedReadWrite bool
	sharedDelete    bool
	sortOrdered     bool
}

func TestAddressBooks(t *testing.T) {
	if skip(t) {
		return
	}

	containerTest(t,
		func(session *Session) string { return session.PrimaryAccounts.Contacts },
		list,
		getid,
		func(s *StalwartTest, accountId string, ids []string, ctx Context) (AddressBookGetResponse, SessionState, State, Language, Error) {
			return s.client.GetAddressbooks(accountId, ids, ctx)
		},
		func(s *StalwartTest, accountId string, id string, change AddressBookChange, ctx Context) (AddressBook, SessionState, State, Language, Error) { //NOSONAR
			return s.client.UpdateAddressBook(accountId, id, change, ctx)
		},
		func(s *StalwartTest, accountId string, ids []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) { //NOSONAR
			return s.client.DeleteAddressBook(accountId, ids, ctx)
		},
		func(s *StalwartTest, t *testing.T, accountId string, count uint, ctx Context, user User, principalIds []string) (AddressBookBoxes, []AddressBook, SessionState, State, error) {
			return s.fillAddressBook(t, accountId, count, ctx, user, principalIds)
		},
		func(orig AddressBook) AddressBookChange {
			return AddressBookChange{
				Description:  strPtr(orig.Description + " (changed)"),
				IsSubscribed: boolPtr(!orig.IsSubscribed),
			}
		},
		func(t *testing.T, orig AddressBook, _ AddressBookChange, changed AddressBook) {
			require.Equal(t, orig.Name, changed.Name)
			require.Equal(t, orig.Description+" (changed)", changed.Description)
			require.Equal(t, !orig.IsSubscribed, changed.IsSubscribed)
		},
	)
}

func TestContacts(t *testing.T) {
	if skip(t) {
		return
	}

	count := uint(20 + rand.Intn(30))

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	user := pickUser()
	session := s.Session(user.name)
	ctx := s.Context(session)

	accountId, addressbookId, expectedContactCardsById, boxes, err := s.fillContacts(t, count, session, user)
	require.NoError(err)
	require.NotEmpty(accountId)
	require.NotEmpty(addressbookId)

	filter := ContactCardFilterCondition{
		InAddressBook: addressbookId,
	}
	sortBy := []ContactCardComparator{
		{Property: ContactCardPropertyCreated, IsAscending: true},
	}

	contactsByAccount, _, _, _, err := s.client.QueryContactCards([]string{accountId}, filter, sortBy, 0, 0, true, ctx)
	require.NoError(err)

	require.Len(contactsByAccount, 1)
	require.Contains(contactsByAccount, accountId)
	results := contactsByAccount[accountId]
	require.Len(results.Results, int(count))
	require.Equal(uint(0), results.Limit)
	require.Equal(uint(0), results.Position)
	require.NotNil(results.Total)
	require.Equal(count, *results.Total)
	require.Equal(true, results.CanCalculateChanges)

	for _, actual := range results.Results {
		expected, ok := expectedContactCardsById[actual.Id]
		require.True(ok, "failed to find created contact by its id")
		matchContact(t, actual, expected)
	}

	// retrieve all objects at once
	{
		ids := structs.Map(results.Results, func(c ContactCard) string { return c.Id })
		fetched, _, _, _, err := s.client.GetContactCards(accountId, ids, ctx)
		require.NoError(err)
		require.Empty(fetched.NotFound)
		require.Len(fetched.List, len(ids))
		byId := structs.Index(fetched.List, func(r ContactCard) string { return r.Id })
		for _, actual := range results.Results {
			expected, ok := byId[actual.Id]
			require.True(ok, "failed to find created contact by its id")
			matchContact(t, actual, expected)
		}
	}

	// retrieve each object one by one
	for _, actual := range results.Results {
		fetched, _, _, _, err := s.client.GetContactCards(accountId, []string{actual.Id}, ctx)
		require.NoError(err)
		require.Len(fetched.List, 1)
		matchContact(t, fetched.List[0], actual)
	}

	exceptions := []string{}
	if !EnableMediaWithBlobId {
		exceptions = append(exceptions, "mediaWithBlobId")
	}
	allBoxesAreTicked(t, boxes, exceptions...)
}

func matchContact(t *testing.T, actual ContactCard, expected ContactCard) {
	// require.Equal(t, expected, actual)
	deepEqual(t, expected, actual)
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

func (s *StalwartTest) fillAddressBook( //NOSONAR
	t *testing.T,
	accountId string,
	count uint,
	ctx Context,
	_ User,
	principalIds []string,
) (AddressBookBoxes, []AddressBook, SessionState, State, error) {
	require := require.New(t)

	boxes := AddressBookBoxes{}
	created := []AddressBook{}
	ss := SessionState("")
	as := EmptyState

	printer := func(s string) { golog.Println(s) }

	for i := range count {
		name := gofakeit.Company()
		description := gofakeit.SentenceSimple()
		subscribed := gofakeit.Bool()
		abook := AddressBookChange{
			Name:         &name,
			Description:  &description,
			IsSubscribed: &subscribed,
		}
		if i%2 == 0 {
			abook.SortOrder = uintPtr(gofakeit.Uint())
			boxes.sortOrdered = true
		}
		var sharing *AddressBookRights = nil
		switch i % 4 {
		default:
			// no sharing
		case 1:
			sharing = &AddressBookRights{MayRead: true, MayWrite: true, MayAdmin: false, MayDelete: false}
			boxes.sharedReadWrite = true
		case 2:
			sharing = &AddressBookRights{MayRead: true, MayWrite: false, MayAdmin: false, MayDelete: false}
			boxes.sharedReadOnly = true
		case 3:
			sharing = &AddressBookRights{MayRead: true, MayWrite: true, MayAdmin: false, MayDelete: true}
			boxes.sharedDelete = true
		}
		if sharing != nil {
			numPrincipals := 1 + rand.Intn(len(principalIds)-1)
			m := make(map[string]AddressBookRights, numPrincipals)
			for _, p := range pickRandomN(numPrincipals, principalIds...) {
				m[p] = *sharing
			}
			abook.ShareWith = m
		}

		a, sessionState, state, _, err := s.client.CreateAddressBook(accountId, abook, ctx)
		if err != nil {
			return boxes, created, ss, as, err
		}
		require.NotEmpty(sessionState)
		require.NotEmpty(state)
		if ss != SessionState("") {
			require.Equal(ss, sessionState)
		}
		if as != EmptyState {
			require.NotEqual(as, state)
		}
		require.NotNil(a)
		created = append(created, *a)
		ss = sessionState
		as = state

		printer(fmt.Sprintf("📔 created %*s/%v id=%v", int(math.Log10(float64(count))+1), strconv.Itoa(int(i+1)), count, a.Id))
	}
	return boxes, created, ss, as, nil
}

func (s *StalwartTest) fillContacts( //NOSONAR
	t *testing.T,
	count uint,
	session *Session,
	user User,
) (string, string, map[string]ContactCard, ContactsBoxes, error) {
	require := require.New(t)
	c, err := NewTestJmapClient(session, user.name, user.password, true, true)
	require.NoError(err)
	defer c.Close()

	boxes := ContactsBoxes{}

	printer := func(s string) { golog.Println(s) }

	accountId := c.session.PrimaryAccounts.Contacts
	require.NotEmpty(accountId, "no primary account for contacts in session")

	addressbookId := ""
	{
		addressBooksById, err := c.objectsById(accountId, AddressBookType)
		require.NoError(err)

		for id, addressbook := range addressBooksById {
			if isDefault, ok := addressbook["isDefault"]; ok {
				if isDefault.(bool) {
					addressbookId = id
					break
				}
			} else {
				printer(fmt.Sprintf("abook without isDefault: %v", addressbook))
			}
		}
		if addressbookId == "" {
			ids := structs.Keys(addressBooksById)
			slices.Sort(ids)
			addressbookId = ids[0]
		}
	}
	require.NotEmpty(addressbookId)

	filled := map[string]ContactCard{}
	for i := range count {
		person := gofakeit.Person()
		nameMap, nameObj := createName(person)
		language := pickLanguage()
		contact := map[string]any{
			"@type":          "Card",
			"version":        "1.0",
			"addressBookIds": toBoolMap([]string{addressbookId}),
			"prodId":         productName,
			"language":       language,
			"kind":           "individual",
			"name":           nameMap,
		}
		card := ContactCard{
			Type:           jscontact.ContactCardType,
			Version:        "1.0",
			AddressBookIds: toBoolMap([]string{addressbookId}),
			ProdId:         productName,
			Language:       language,
			Kind:           jscontact.ContactCardKindIndividual,
			Name:           &nameObj,
		}

		if i%3 == 0 {
			nicknameMap, nicknameObj := createNickName(person)
			id := id()
			contact["nicknames"] = map[string]map[string]any{id: nicknameMap}
			card.Nicknames = map[string]jscontact.Nickname{id: nicknameObj}
			boxes.nicknames = true
		}

		{
			emailMaps := map[string]map[string]any{}
			emailObjs := map[string]jscontact.EmailAddress{}
			emailId := id()
			emailMap, emailObj := createEmail(person, 10)
			emailMaps[emailId] = emailMap
			emailObjs[emailId] = emailObj

			for i := range rand.Intn(3) {
				id := id()
				m, o := createSecondaryEmail(gofakeit.Email(), i*100)
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
				}, jscontact.Phone{
					Type:     jscontact.PhoneType,
					Number:   tel,
					Features: features,
					Contexts: contexts,
				}, nil
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
				components = append(components, jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindName, Value: m[0][2]})
				components = append(components, jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindNumber, Value: m[0][1]})
			} else {
				components = append(components, jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindName, Value: source.Street})
			}
			components = append(components,
				jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindLocality, Value: source.City},
				jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindCountry, Value: source.Country},
				jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindRegion, Value: source.State},
				jscontact.AddressComponent{Type: jscontact.AddressComponentType, Kind: jscontact.AddressComponentKindPostcode, Value: source.Zip},
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
				}, jscontact.Address{
					Type:             jscontact.AddressType,
					Components:       components,
					DefaultSeparator: ", ",
					IsOrdered:        true,
					TimeZone:         tz,
				}, nil
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
					}, jscontact.OnlineService{
						Type:    jscontact.OnlineServiceType,
						Service: "Mastodon",
						User:    "@" + person.Contact.Email,
						Uri:     "https://mastodon.example.com/@" + strings.ToLower(person.FirstName),
					}, nil
			case 1:
				return map[string]any{
						"@type": "OnlineService",
						"uri":   "xmpp:" + person.Contact.Email,
					}, jscontact.OnlineService{
						Type: jscontact.OnlineServiceType,
						Uri:  "xmpp:" + person.Contact.Email,
					}, nil
			default:
				return map[string]any{
						"@type":   "OnlineService",
						"service": "Discord",
						"user":    person.Contact.Email,
						"uri":     "https://discord.example.com/user/" + person.Contact.Email,
					}, jscontact.OnlineService{
						Type:    jscontact.OnlineServiceType,
						Service: "Discord",
						User:    person.Contact.Email,
						Uri:     "https://discord.example.com/user/" + person.Contact.Email,
					}, nil
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
				}, jscontact.LanguagePref{
					Type:     jscontact.LanguagePrefType,
					Language: lang,
					Contexts: toBoolMap(structs.Map(contexts, func(s string) jscontact.LanguagePrefContext { return jscontact.LanguagePrefContext(s) })),
					Pref:     uint(i + 1),
				}, nil
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
				organizationObjs[orgId] = jscontact.Organization{
					Type:     jscontact.OrganizationType,
					Name:     person.Job.Company,
					Contexts: toBoolMapS(jscontact.OrganizationContextWork),
				}
				titleMaps[titleId] = map[string]any{
					"@type":          "Title",
					"kind":           "title",
					"name":           person.Job.Title,
					"organizationId": orgId,
				}
				titleObjs[titleId] = jscontact.Title{
					Type:           jscontact.TitleType,
					Kind:           jscontact.TitleKindTitle,
					Name:           person.Job.Title,
					OrganizationId: orgId,
				}
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
					"@type":     "CryptoKey",
					"uri":       "data:application/pgp-keys;base64," + encoded,
					"mediaType": "application/pgp-keys",
				}, jscontact.CryptoKey{
					Type:      jscontact.CryptoKeyType,
					Uri:       "data:application/pgp-keys;base64," + encoded,
					MediaType: "application/pgp-keys",
				}, nil
		}); err != nil {
			return "", "", nil, boxes, err
		}

		if err := propmap(i%2 == 0, 1, 2, contact, "media", &card.Media, func(i int, id string) (map[string]any, jscontact.Media, error) {
			label := fmt.Sprintf("photo-%d", 1000+rand.Intn(9000))

			r := 0
			if EnableMediaWithBlobId {
				r = rand.Intn(3)
			} else {
				r = rand.Intn(2)
			}

			switch r {
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
					}, jscontact.Media{
						Type:      jscontact.MediaType,
						Kind:      jscontact.MediaKindPhoto,
						Uri:       uri,
						MediaType: mime,
						Contexts:  contexts,
						Label:     label,
					}, nil

			case 1:
				boxes.mediaWithExternalUri = true
				// use external uri
				uri := externalImageUri()
				contexts := toBoolMapS(jscontact.MediaContextWork)
				return map[string]any{
						"@type":    "Media",
						"kind":     string(jscontact.MediaKindPhoto),
						"uri":      uri,
						"contexts": structs.MapKeys(contexts, func(c jscontact.MediaContext) string { return string(c) }),
						"label":    label,
					}, jscontact.Media{
						Type:     jscontact.MediaType,
						Kind:     jscontact.MediaKindPhoto,
						Uri:      uri,
						Contexts: contexts,
						Label:    label,
					}, nil

			default:
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
					}, jscontact.Media{
						Type:      jscontact.MediaType,
						Kind:      jscontact.MediaKindPhoto,
						BlobId:    blob.BlobId,
						MediaType: blob.Type,
						Contexts:  contexts,
						Label:     label,
					}, nil

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
				}, jscontact.Link{
					Type: jscontact.LinkType,
					Kind: jscontact.LinkKindContact,
					Uri:  "mailto:" + person.Contact.Email,
					Pref: uint((i + 1) * 10),
				}, nil
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
	return j.create1(accountId, ContactCardType, contact)
}
