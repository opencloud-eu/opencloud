package identity

import (
	"context"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var eduUserAttrs = []string{
	"displayname",
	"entryUUID",
	"mail",
	"uid",
	"sn",
	"givenname",
	"userEnabledAttribute",
	"userTypeAttribute",
	"openCloudExternalIdentity",
	"userClass",
	"openCloudMemberOfSchool",
}

var eduUserEntry = ldap.NewEntry("uid=user,ou=people,dc=test",
	map[string][]string{
		"uid":         {"testuser"},
		"displayname": {"Test User"},
		"mail":        {"user@example"},
		"entryuuid":   {"abcd-defg"},
		"userClass":   {"student"},
		"openCloudExternalIdentity": {
			"$ http://idp $ testuser",
			"xxx $ http://idpnew $ xxxxx-xxxxx-xxxxx",
		},
		"userTypeAttribute":    {"Member"},
		"userEnabledAttribute": {"FALSE"},
	})
var renamedEduUserEntry = ldap.NewEntry("uid=newtestuser,ou=people,dc=test",
	map[string][]string{
		"uid":         {"newtestuser"},
		"displayname": {"Test User"},
		"mail":        {"user@example"},
		"entryuuid":   {"abcd-defg"},
		"userClass":   {"student"},
		"openCloudExternalIdentity": {
			"$ http://idp $ testuser",
			"xxx $ http://idpnew $ xxxxx-xxxxx-xxxxx",
		},
		"userTypeAttribute":    {"Guest"},
		"userEnabledAttribute": {"TRUE"},
	})
var eduUserEntryWithSchool = ldap.NewEntry("uid=user,ou=people,dc=test",
	map[string][]string{
		"uid":                     {"testuser"},
		"displayname":             {"Test User"},
		"mail":                    {"user@example"},
		"entryuuid":               {"abcd-defg"},
		"userClass":               {"student"},
		"openCloudMemberOfSchool": {"abcd-defg"},
		"openCloudExternalIdentity": {
			"$ http://idp $ testuser",
			"xxx $ http://idpnew $ xxxxx-xxxxx-xxxxx",
		},
	})

var sr1 *ldap.SearchRequest = &ldap.SearchRequest{
	BaseDN:     "ou=people,dc=test",
	Scope:      2,
	SizeLimit:  1,
	Filter:     "(&(objectClass=openCloudEducationUser)(|(uid=abcd-defg)(entryUUID=abcd-defg)))",
	Attributes: eduUserAttrs,
	Controls:   []ldap.Control(nil),
}
var sr2 *ldap.SearchRequest = &ldap.SearchRequest{
	BaseDN:     "ou=people,dc=test",
	Scope:      2,
	SizeLimit:  1,
	Filter:     "(&(objectClass=openCloudEducationUser)(|(uid=xxxx-xxxx)(entryUUID=xxxx-xxxx)))",
	Attributes: eduUserAttrs,
	Controls:   []ldap.Control(nil),
}

func TestCreateEducationUser(t *testing.T) {
	lm := &mocks.Client{}
	b, err := getMockedBackend(lm, eduConfig, &logger)
	assert.Nil(t, err)
	//assert.NotEqual(t, "", b.educationConfig.schoolObjectClass)
	lm.On("Add", mock.Anything).Return(nil)

	lm.On("Search", mock.Anything).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{
					eduUserEntry,
				},
			},
			nil)
	user := libregraph.NewEducationUser()
	user.SetDisplayName("Test User")
	user.SetOnPremisesSamAccountName("testuser")
	user.SetMail("testuser@example.org")
	user.SetPrimaryRole("student")
	user.SetUserType(("Member"))
	user.SetAccountEnabled(false)
	eduUser, err := b.CreateEducationUser(context.Background(), *user)
	lm.AssertNumberOfCalls(t, "Add", 1)
	lm.AssertNumberOfCalls(t, "Search", 1)
	assert.NotNil(t, eduUser)
	assert.Nil(t, err)
	assert.Equal(t, eduUser.GetDisplayName(), user.GetDisplayName())
	assert.Equal(t, eduUser.GetOnPremisesSamAccountName(), user.GetOnPremisesSamAccountName())
	assert.Equal(t, "abcd-defg", eduUser.GetId())
	assert.Equal(t, eduUser.GetPrimaryRole(), user.GetPrimaryRole())
	assert.Equal(t, eduUser.GetUserType(), user.GetUserType())
	assert.Equal(t, eduUser.GetAccountEnabled(), false)
}

func TestDeleteEducationUser(t *testing.T) {
	lm := &mocks.Client{}

	lm.On("Search", sr1).Return(&ldap.SearchResult{Entries: []*ldap.Entry{eduUserEntry}}, nil)
	lm.On("Search", sr2).Return(&ldap.SearchResult{Entries: []*ldap.Entry{}}, nil)
	dr1 := &ldap.DelRequest{
		DN: "uid=user,ou=people,dc=test",
	}
	lm.On("Del", dr1).Return(nil)
	b, err := getMockedBackend(lm, eduConfig, &logger)
	assert.Nil(t, err)
	err = b.DeleteEducationUser(context.Background(), "abcd-defg")
	lm.AssertNumberOfCalls(t, "Search", 1)
	lm.AssertNumberOfCalls(t, "Del", 1)
	assert.Nil(t, err)

	err = b.DeleteEducationUser(context.Background(), "xxxx-xxxx")
	lm.AssertNumberOfCalls(t, "Search", 2)
	lm.AssertNumberOfCalls(t, "Del", 1)
	assert.NotNil(t, err)
	assert.Equal(t, "itemNotFound: not found", err.Error())
}

func TestGetEducationUser(t *testing.T) {
	lm := &mocks.Client{}
	lm.On("Search", sr1).Return(&ldap.SearchResult{Entries: []*ldap.Entry{eduUserEntry}}, nil)
	lm.On("Search", sr2).Return(&ldap.SearchResult{Entries: []*ldap.Entry{}}, nil)
	b, err := getMockedBackend(lm, eduConfig, &logger)
	assert.Nil(t, err)
	user, err := b.GetEducationUser(context.Background(), "abcd-defg")
	lm.AssertNumberOfCalls(t, "Search", 1)
	assert.Nil(t, err)
	assert.Equal(t, "Test User", user.GetDisplayName())
	assert.Equal(t, "abcd-defg", user.GetId())

	_, err = b.GetEducationUser(context.Background(), "xxxx-xxxx")
	lm.AssertNumberOfCalls(t, "Search", 2)
	assert.NotNil(t, err)
	assert.Equal(t, "itemNotFound: not found", err.Error())
}

func TestGetEducationUsers(t *testing.T) {
	lm := &mocks.Client{}
	sr := &ldap.SearchRequest{
		BaseDN:     "ou=people,dc=test",
		Scope:      2,
		SizeLimit:  0,
		Filter:     "(objectClass=openCloudEducationUser)",
		Attributes: eduUserAttrs,
		Controls:   []ldap.Control(nil),
	}
	lm.On("Search", sr).Return(&ldap.SearchResult{Entries: []*ldap.Entry{eduUserEntry}}, nil)
	b, err := getMockedBackend(lm, eduConfig, &logger)
	assert.Nil(t, err)
	_, err = b.GetEducationUsers(context.Background())
	lm.AssertNumberOfCalls(t, "Search", 1)
	assert.Nil(t, err)
}

func TestUpdateEducationUser(t *testing.T) {
	lm := &mocks.Client{}
	b, err := getMockedBackend(lm, eduConfig, &logger)
	assert.Nil(t, err)
	userSearchReq := &ldap.SearchRequest{
		BaseDN:     "ou=people,dc=test",
		Scope:      2,
		SizeLimit:  1,
		Filter:     "(&(objectClass=openCloudEducationUser)(|(uid=testuser)(entryUUID=testuser)))",
		Attributes: eduUserAttrs,
	}
	userLookupReq := &ldap.SearchRequest{
		BaseDN:     "uid=newtestuser,ou=people,dc=test",
		Scope:      0,
		SizeLimit:  1,
		Filter:     "(objectClass=inetOrgPerson)",
		Attributes: ldapUserAttributes,
	}
	eduUserLookupReq := &ldap.SearchRequest{
		BaseDN:     "uid=newtestuser,ou=people,dc=test",
		Scope:      0,
		SizeLimit:  1,
		Filter:     "(objectClass=openCloudEducationUser)",
		Attributes: eduUserAttrs,
	}
	groupSearchReq := &ldap.SearchRequest{
		BaseDN: "ou=groups,dc=test",
		Scope:  2,
		Filter: "(&(objectClass=groupOfNames)(member=uid=user,ou=people,dc=test))",
		Attributes: []string{
			"cn",
			"entryUUID",
		},
	}
	lm.On("Search", userLookupReq).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{
					renamedEduUserEntry,
				},
			},
			nil)
	lm.On("Search", eduUserLookupReq).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{
					renamedEduUserEntry,
				},
			},
			nil)
	lm.On("Search", userSearchReq).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{
					eduUserEntry,
				},
			},
			nil)
	lm.On("Search", groupSearchReq).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{},
			},
			nil)
	modReq := ldap.ModifyRequest{
		DN: "uid=newtestuser,ou=people,dc=test",
		Changes: []ldap.Change{
			{
				Operation: ldap.ReplaceAttribute,
				Modification: ldap.PartialAttribute{
					Type: "mail",
					Vals: []string{"new@mail.org"},
				},
			},
			{
				Operation: ldap.ReplaceAttribute,
				Modification: ldap.PartialAttribute{
					Type: "userEnabledAttribute",
					Vals: []string{"TRUE"},
				},
			},
		},
	}
	modDNReq := ldap.ModifyDNRequest{
		DN:           "uid=user,ou=people,dc=test",
		NewRDN:       "uid=newtestuser",
		DeleteOldRDN: true,
	}
	lm.On("ModifyDN", &modDNReq).Return(nil)
	lm.On("Modify", &modReq).Return(nil)
	user := libregraph.NewEducationUser()
	user.SetOnPremisesSamAccountName("newtestuser")
	user.SetMail("new@mail.org")
	user.SetAccountEnabled(true)
	eduUser, err := b.UpdateEducationUser(context.Background(), "testuser", *user)
	assert.NotNil(t, eduUser)
	assert.Nil(t, err)
	assert.Equal(t, eduUser.GetOnPremisesSamAccountName(), "newtestuser")
	assert.Equal(t, "abcd-defg", eduUser.GetId())
	assert.Equal(t, "Guest", eduUser.GetUserType())
	assert.Equal(t, eduUser.GetAccountEnabled(), true)
}
