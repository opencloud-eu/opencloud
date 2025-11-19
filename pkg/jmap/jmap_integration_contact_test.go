package jmap

import (
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
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

func matchContact(t *testing.T, actual jscontact.ContactCard, expected jscontact.ContactCard) {
	require.Equal(t, expected, actual)
}
