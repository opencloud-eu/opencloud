package jmap

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
)

func TestContacts(t *testing.T) {
	if skip(t) {
		return
	}

	count := uint(15 + rand.Intn(20))

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	accountId, addressbookId, cardsById, err := s.fillContacts(t, count)
	require.NoError(err)
	require.NotEmpty(accountId)
	require.NotEmpty(addressbookId)

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
		expected.Id = actual.Id
		matchContact(t, actual, expected)
	}
}

func matchContact(t *testing.T, actual jscontact.ContactCard, expected jscontact.ContactCard) {
	require := require.New(t)

	//a := actual
	//unType(t, &a)

	require.Equal(expected, actual)
}

func unType(t *testing.T, s any) {
	ty := reflect.TypeOf(s)

	switch ty.Kind() {
	case reflect.Map, reflect.Array, reflect.Pointer, reflect.Slice:
		ty = ty.Elem()
	}

	switch ty.Kind() {
	case reflect.Struct:
		for i := range ty.NumField() {
			f := ty.Field(i)
			n := f.Name
			if n == "Type" {
				v := reflect.ValueOf(s).Field(i)
				require.True(t, v.Elem().CanSet(), "cannot set the Type field")
				v.SetString("")
			} else {
				//unType(t, f)
			}
		}
	}
}
