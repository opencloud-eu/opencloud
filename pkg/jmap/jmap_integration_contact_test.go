package jmap

import (
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

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

	accountId, addressbookId, cardsById, sentById, boxes, err := s.fillContacts(t, count)
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
		expected, ok := cardsById[actual.Id]
		require.True(ok, "failed to find created contact by its id")
		sent := sentById[actual.Id]
		matchContact(t, actual, expected, sent, func() (jscontact.ContactCard, error) {
			cards, _, _, _, err := s.client.GetContactCardsById(accountId, s.session, t.Context(), s.logger, "", []string{actual.Id})
			if err != nil {
				return jscontact.ContactCard{}, err
			}
			require.Contains(cards, actual.Id)
			return cards[actual.Id], nil
		})
	}
}

func allTrue[S any](t *testing.T, s S, exceptions ...string) {
	v := reflect.ValueOf(s)
	typ := v.Type()
	for i := range v.NumField() {
		name := typ.Field(i).Name
		if slices.Contains(exceptions, name) {
			continue
		}
		value := v.Field(i).Bool()
		require.True(t, value, "should be true: %v", name)
	}
}

func matchContact(t *testing.T, actual jscontact.ContactCard, expected jscontact.ContactCard, sent map[string]any, fetcher func() (jscontact.ContactCard, error)) {
	require := require.New(t)
	if structs.AnyValue(expected.Media, func(media jscontact.Media) bool { return media.BlobId != "" }) {
		fmt.Printf("\x1b[33;1m----------------------------------------------------------\x1b[0m\n")
		fmt.Printf("\x1b[45;1m expected media: \x1b[0m\n%v\n\n", expected.Media)
		fmt.Printf("\x1b[46;1m actual media: \x1b[0m\n%v\n\n", actual.Media)
		fmt.Printf("\x1b[43;1m sent: \x1b[0m\n%v\n\n", sent)
		fmt.Printf("\x1b[44;1m pulling: \x1b[0m\n")
		_, err := fetcher()
		require.NoError(err)
		fmt.Printf("\x1b[44;1m pulled. \x1b[0m\n")
	}

	require.Equal(expected.Name, actual.Name)
	require.Equal(expected.Emails, actual.Emails)
	require.Equal(expected.Organizations, actual.Organizations)
	require.Equal(expected.Media, actual.Media)

	require.Equal(expected, actual)
}
