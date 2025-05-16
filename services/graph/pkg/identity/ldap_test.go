package identity

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/CiscoM31/godata"
	"github.com/go-ldap/ldap/v3"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func getMockedBackend(l ldap.Client, lc config.LDAP, logger *log.Logger) (*LDAP, error) {
	return NewLDAPBackend(l, lc, logger)
}

const (
	disableUsersGroup = "cn=DisabledUsersGroup,ou=groups,o=testing"
	groupSearchFilter = "(objectClass=groupOfNames)"
)

var lconfig = config.LDAP{
	UserBaseDN:               "ou=people,dc=test",
	UserObjectClass:          "inetOrgPerson",
	UserSearchScope:          "sub",
	UserFilter:               "",
	UserDisplayNameAttribute: "displayname",
	UserIDAttribute:          "entryUUID",
	UserEmailAttribute:       "mail",
	UserNameAttribute:        "uid",
	UserEnabledAttribute:     "userEnabledAttribute",
	UserTypeAttribute:        "userTypeAttribute",
	LdapDisabledUsersGroupDN: disableUsersGroup,
	DisableUserMechanism:     "attribute",

	GroupBaseDN:          "ou=groups,dc=test",
	GroupObjectClass:     "groupOfNames",
	GroupSearchScope:     "sub",
	GroupFilter:          "",
	GroupNameAttribute:   "cn",
	GroupMemberAttribute: "member",
	GroupIDAttribute:     "entryUUID",

	WriteEnabled: true,
}

var userEntry = ldap.NewEntry("uid=user",
	map[string][]string{
		"uid":                  {"user"},
		"displayname":          {"DisplayName"},
		"mail":                 {"user@example"},
		"entryuuid":            {"abcd-defg"},
		"sn":                   {"surname"},
		"givenname":            {"givenName"},
		"userenabledattribute": {"TRUE"},
		"usertypeattribute":    {"Member"},
	})

var invalidUserEntry = ldap.NewEntry("uid=user",
	map[string][]string{
		"uid":         {"invalid"},
		"displayname": {"DisplayName"},
		"mail":        {"user@example"},
	})

var logger = log.NewLogger(log.Level("debug"))

var ldapUserAttributes = []string{"displayname", "entryUUID", "mail", "uid", "sn", "givenname", "userEnabledAttribute", "userTypeAttribute", "openCloudExternalIdentity", "openCloudLastSignInTimestamp"}

func TestNewLDAPBackend(t *testing.T) {
	l := &mocks.Client{}

	tc := lconfig
	tc.UserDisplayNameAttribute = ""
	if _, err := NewLDAPBackend(l, tc, &logger); err == nil {
		t.Error("Should fail with incomplete user attr config")
	}

	tc = lconfig
	tc.GroupIDAttribute = ""
	if _, err := NewLDAPBackend(l, tc, &logger); err == nil {
		t.Errorf("Should fail with incomplete group	config")
	}

	tc = lconfig
	tc.UserSearchScope = ""
	if _, err := NewLDAPBackend(l, tc, &logger); err == nil {
		t.Errorf("Should fail with invalid user search scope")
	}

	tc = lconfig
	tc.GroupSearchScope = ""
	if _, err := NewLDAPBackend(l, tc, &logger); err == nil {
		t.Errorf("Should fail with invalid group search scope")
	}

	if _, err := NewLDAPBackend(l, lconfig, &logger); err != nil {
		t.Errorf("Should fail with invalid group search scope")
	}
}

func TestCreateUser(t *testing.T) {
	displayName := "DisplayName"
	mail := "user@example"
	userName := "user"
	surname := "surname"
	givenName := "givenName"
	userType := "Member"

	ar := ldap.NewAddRequest(fmt.Sprintf("uid=user,%s", lconfig.UserBaseDN), nil)
	ar.Attribute(lconfig.UserDisplayNameAttribute, []string{displayName})
	ar.Attribute(lconfig.UserEmailAttribute, []string{mail})
	ar.Attribute(lconfig.UserNameAttribute, []string{userName})
	ar.Attribute("sn", []string{surname})
	ar.Attribute("givenname", []string{givenName})
	ar.Attribute(lconfig.UserEnabledAttribute, []string{"TRUE"})
	ar.Attribute(lconfig.UserTypeAttribute, []string{"Member"})
	ar.Attribute("cn", []string{userName})
	ar.Attribute("objectClass", []string{"inetOrgPerson", "organizationalPerson", "person", "top", "openCloudUser"})

	l := &mocks.Client{}
	l.On("Search", mock.Anything).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{userEntry},
			},
			nil)
	l.On("Add", ar).Return(nil)
	logger := log.NewLogger(log.Level("debug"))

	user := libregraph.NewUser(displayName, userName)
	user.SetMail(mail)
	user.SetSurname(surname)
	user.SetGivenName(givenName)
	user.SetAccountEnabled(true)
	user.SetUserType(userType)

	c := lconfig
	c.UseServerUUID = true
	b, _ := NewLDAPBackend(l, c, &logger)

	newUser, err := b.CreateUser(context.Background(), *user)
	assert.Nil(t, err)
	assert.Equal(t, displayName, newUser.GetDisplayName())
	assert.Equal(t, mail, newUser.GetMail())
	assert.Equal(t, userName, newUser.GetOnPremisesSamAccountName())
	assert.Equal(t, givenName, newUser.GetGivenName())
	assert.Equal(t, surname, newUser.GetSurname())
	assert.True(t, newUser.GetAccountEnabled())
	assert.Equal(t, userType, newUser.GetUserType())
}

func TestCreateUserModelFromLDAP(t *testing.T) {
	l := &mocks.Client{}
	logger := log.NewLogger(log.Level("debug"))

	b, _ := NewLDAPBackend(l, lconfig, &logger)
	if user := b.createUserModelFromLDAP(nil); user != nil {
		t.Errorf("createUserModelFromLDAP should return on nil Entry")
	}
	user := b.createUserModelFromLDAP(userEntry)
	if user == nil {
		t.Error("Converting a valid LDAP Entry should succeed")
	} else {
		if user.OnPremisesSamAccountName != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.userName) {
			t.Errorf("Error creating msGraph User from LDAP Entry: %v != %v", user.OnPremisesSamAccountName, pointerOrNil(userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.userName)))
		}
		if *user.Mail != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.mail) {
			t.Errorf("Error creating msGraph User from LDAP Entry: %s != %s", *user.Mail, userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.mail))
		}
		if user.DisplayName != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.displayName) {
			t.Errorf("Error creating msGraph User from LDAP Entry: %s != %s", user.DisplayName, userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.displayName))
		}
		if *user.Id != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id) {
			t.Errorf("Error creating msGraph User from LDAP Entry: %s != %s", *user.Id, userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id))
		}
	}
}

func TestGetUser(t *testing.T) {
	// Mock a Sizelimit Error
	lm := &mocks.Client{}
	lm.On("Search", mock.Anything).
		Return(
			nil, ldap.NewError(ldap.LDAPResultSizeLimitExceeded, errors.New("mock")))
	b, _ := getMockedBackend(lm, lconfig, &logger)

	odataReqDefault, err := godata.ParseRequest(context.Background(), "",
		url.Values{})
	if err != nil {
		t.Errorf("Expected success got '%s'", err.Error())
	}

	odataReqExpand, err := godata.ParseRequest(context.Background(), "",
		url.Values{"$expand": []string{"memberOf"}})
	if err != nil {
		t.Errorf("Expected success got '%s'", err.Error())
	}

	_, err = b.GetUser(context.Background(), "fred", odataReqDefault)
	assert.ErrorContains(t, err, "itemNotFound:")

	_, err = b.GetUser(context.Background(), "fred", odataReqExpand)
	assert.ErrorContains(t, err, "itemNotFound:")

	// Mock an empty Search Result
	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).
		Return(
			&ldap.SearchResult{}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	_, err = b.GetUser(context.Background(), "fred", odataReqDefault)
	assert.ErrorContains(t, err, "itemNotFound:")

	_, err = b.GetUser(context.Background(), "fred", odataReqExpand)
	assert.ErrorContains(t, err, "itemNotFound:")

	// Mock a valid Search Result
	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{userEntry},
			},
			nil)

	b, _ = getMockedBackend(lm, lconfig, &logger)
	u, err := b.GetUser(context.Background(), "user", odataReqDefault)
	if err != nil {
		t.Errorf("Expected GetUser to succeed. Got %s", err.Error())
	} else if *u.Id != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id) {
		t.Errorf("Expected GetUser to return a valid user")
	}

	u, err = b.GetUser(context.Background(), "user", odataReqExpand)
	if err != nil {
		t.Errorf("Expected GetUser to succeed. Got %s", err.Error())
	} else if *u.Id != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id) {
		t.Errorf("Expected GetUser to return a valid user")
	}

	// Mock invalid Search Result
	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).
		Return(
			&ldap.SearchResult{
				Entries: []*ldap.Entry{invalidUserEntry},
			},
			nil)

	b, _ = getMockedBackend(lm, lconfig, &logger)
	_, err = b.GetUser(context.Background(), "invalid", nil)
	assert.ErrorContains(t, err, "itemNotFound:")
}

func TestGetUsers(t *testing.T) {
	// Mock a Sizelimit Error
	lm := &mocks.Client{}
	lm.On("Search", mock.Anything).Return(nil, ldap.NewError(ldap.LDAPResultOperationsError, errors.New("mock")))

	odataReqDefault, err := godata.ParseRequest(context.Background(), "",
		url.Values{})
	if err != nil {
		t.Errorf("Expected success got '%s'", err.Error())
	}

	b, _ := getMockedBackend(lm, lconfig, &logger)
	_, err = b.GetUsers(context.Background(), odataReqDefault)
	assert.ErrorContains(t, err, "generalException:")

	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).Return(&ldap.SearchResult{}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	g, err := b.GetUsers(context.Background(), odataReqDefault)
	if err != nil {
		t.Errorf("Expected success, got '%s'", err.Error())
	} else if g == nil || len(g) != 0 {
		t.Errorf("Expected zero length user slice")
	}
}

func TestGetUsersSearch(t *testing.T) {
	lm := &mocks.Client{}
	odataReqDefault, err := godata.ParseRequest(context.Background(), "",
		url.Values{
			"$search": []string{"\"term\""},
		},
	)
	if err != nil {
		t.Errorf("Expected success got '%s'", err.Error())
	}

	// only match if the filter contains the search term unquoted
	lm.On("Search", mock.MatchedBy(
		func(req *ldap.SearchRequest) bool {
			return req.Filter == "(&(objectClass=inetOrgPerson)(|(uid=*term*)(mail=*term*)(displayname=*term*)))"
		})).
		Return(&ldap.SearchResult{}, nil)
	b, _ := getMockedBackend(lm, lconfig, &logger)
	g, err := b.GetUsers(context.Background(), odataReqDefault)
	if err != nil {
		t.Errorf("Expected success, got '%s'", err.Error())
	} else if g == nil || len(g) != 0 {
		t.Errorf("Expected zero length user slice")
	}
}

func TestUpdateUser(t *testing.T) {
	falseBool := false
	trueBool := true
	memberType := "Member"

	type userProps struct {
		id                       string
		mail                     string
		displayName              string
		onPremisesSamAccountName string
		accountEnabled           *bool
		givenName                *string
		userType                 *string
	}
	type args struct {
		nameOrID             string
		userProps            userProps
		disableUserMechanism string
	}
	type mockInputs struct {
		funcName string
		args     []interface{}
		returns  []interface{}
	}
	tests := []struct {
		name      string
		args      args
		want      *userProps
		assertion assert.ErrorAssertionFunc
		ldapMocks []mockInputs
	}{
		{
			name: "Test changing ID",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					id: "testUser",
				},
			},
			want: nil,
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.NotNil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "ua=testUser",
								},
							},
						},
						nil,
					},
				},
			},
		},
		{
			name: "Test changing mail",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					mail: "testuser@example.org",
				},
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "testUser",
				onPremisesSamAccountName: "testUser",
				accountEnabled:           nil,
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=oldName",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "displayname",
											Values: []string{"oldmail@example.org"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"oldmail@example.org"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN:       "uid=oldName",
							Scope:        0,
							DerefAliases: 0,
							SizeLimit:    1,
							TimeLimit:    0,
							TypesOnly:    false,
							Filter:       "(objectClass=inetOrgPerson)",
							Attributes:   ldapUserAttributes,
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=oldName",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   lconfig.UserIDAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEmailAttribute,
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserDisplayNameAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserNameAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "uid=oldName",
							Changes: []ldap.Change{
								{
									Operation: 0x2,
									Modification: ldap.PartialAttribute{
										Type: "mail",
										Vals: []string{"testuser@example.org"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
			},
		},
		{
			name: "Test changing displayName",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					displayName: "newName",
				},
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "newName",
				onPremisesSamAccountName: "testUser",
				accountEnabled:           nil,
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=oldName",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "displayname",
											Values: []string{"testUser"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN:       "uid=oldName",
							Scope:        0,
							DerefAliases: 0,
							SizeLimit:    1,
							TimeLimit:    0,
							TypesOnly:    false,
							Filter:       "(objectClass=inetOrgPerson)",
							Attributes:   ldapUserAttributes,
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=oldName",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   lconfig.UserIDAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEmailAttribute,
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserDisplayNameAttribute,
											Values: []string{"newName"},
										},
										{
											Name:   lconfig.UserNameAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "uid=oldName",
							Changes: []ldap.Change{
								{
									Operation: 0x2,
									Modification: ldap.PartialAttribute{
										Type: lconfig.UserDisplayNameAttribute,
										Vals: []string{"newName"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
			},
		},
		{
			name: "Test changing userName",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					onPremisesSamAccountName: "newName",
				},
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "newName",
				onPremisesSamAccountName: "newName",
				accountEnabled:           nil,
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=oldName,ou=people,dc=test,dc=net",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   lconfig.UserDisplayNameAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserIDAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEmailAttribute,
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserNameAttribute,
											Values: []string{"oldName"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN: "ou=groups,dc=test",
							Scope:  2, DerefAliases: 0, SizeLimit: 0, TimeLimit: 0,
							TypesOnly:  false,
							Filter:     "(&(objectClass=groupOfNames)(member=uid=oldName,ou=people,dc=test,dc=net))",
							Attributes: []string{"cn", "entryUUID"},
							Controls:   []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "cn=group1",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   lconfig.GroupNameAttribute,
											Values: []string{"group1"},
										},
										{
											Name:   lconfig.GroupIDAttribute,
											Values: []string{"group1-id"},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "ModifyDN",
					args: []interface{}{
						&ldap.ModifyDNRequest{
							DN:           "uid=oldName,ou=people,dc=test,dc=net",
							NewRDN:       "uid=newName",
							DeleteOldRDN: true,
							NewSuperior:  "",
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN:       "uid=newName,ou=people,dc=test,dc=net",
							Scope:        0,
							DerefAliases: 0,
							SizeLimit:    1,
							TimeLimit:    0,
							TypesOnly:    false,
							Filter:       "(objectClass=inetOrgPerson)",
							Attributes:   ldapUserAttributes,
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=newName,ou=people,dc=test,dc=net",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   lconfig.UserIDAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEmailAttribute,
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserDisplayNameAttribute,
											Values: []string{"newName"},
										},
										{
											Name:   lconfig.UserNameAttribute,
											Values: []string{"newName"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "cn=group1",
							Changes: []ldap.Change{
								{
									Operation: 0x1,
									Modification: ldap.PartialAttribute{
										Type: "member",
										Vals: []string{"uid=oldName,ou=people,dc=test,dc=net"},
									},
								},
								{
									Operation: 0x0,
									Modification: ldap.PartialAttribute{
										Type: "member",
										Vals: []string{"uid=newName,ou=people,dc=test,dc=net"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
			},
		},
		{
			name: "Test changing accountEnabled",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					accountEnabled: &falseBool,
				},
				disableUserMechanism: "attribute",
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "testUser",
				onPremisesSamAccountName: "testUser",
				accountEnabled:           &falseBool,
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "displayname",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{"TRUE"},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN:       "uid=name",
							Scope:        0,
							DerefAliases: 0,
							SizeLimit:    1,
							TimeLimit:    0,
							TypesOnly:    false,
							Filter:       "(objectClass=inetOrgPerson)",
							Attributes:   ldapUserAttributes,
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   lconfig.UserIDAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEmailAttribute,
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserDisplayNameAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserNameAttribute,
											Values: []string{"testUser"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{"FALSE"},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "uid=name",
							Changes: []ldap.Change{
								{
									Operation: 0x2,
									Modification: ldap.PartialAttribute{
										Type: lconfig.UserEnabledAttribute,
										Vals: []string{"FALSE"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
			},
		},
		{
			name: "Test disabling user as local admin",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					accountEnabled: &falseBool,
				},
				disableUserMechanism: "group",
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "testUser",
				onPremisesSamAccountName: "testUser",
				accountEnabled:           &falseBool,
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "displayname",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{"TRUE"},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN:       disableUsersGroup,
							Scope:        0,
							DerefAliases: 0,
							SizeLimit:    1,
							TimeLimit:    0,
							TypesOnly:    false,
							Filter:       groupSearchFilter,
							Attributes:   []string{"member"},
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "member",
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "uid=name",
							Changes: []ldap.Change{
								{
									Operation: ldap.AddAttribute,
									Modification: ldap.PartialAttribute{
										Type: "member",
										Vals: []string{"uid=name"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN:       "uid=name",
							Changes:  []ldap.Change(nil),
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"uid=name",
							ldap.ScopeBaseObject,
							ldap.NeverDerefAliases, 1, 0, false,
							"(objectClass=inetOrgPerson)",
							ldapUserAttributes,
							[]ldap.Control(nil),
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "uid",
											Values: []string{"testUser"},
										},
										{
											Name:   "displayname",
											Values: []string{"testUser"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
		},
		{
			name: "Test enabling user as local admin",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					accountEnabled: &trueBool,
				},
				disableUserMechanism: "group",
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "testUser",
				onPremisesSamAccountName: "testUser",
				accountEnabled:           &trueBool,
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "displayname",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserEnabledAttribute,
											Values: []string{"TRUE"},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Search",
					args: []interface{}{
						&ldap.SearchRequest{
							BaseDN:       disableUsersGroup,
							Scope:        0,
							DerefAliases: 0,
							SizeLimit:    1,
							TimeLimit:    0,
							TypesOnly:    false,
							Filter:       groupSearchFilter,
							Attributes:   []string{"member"},
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "member",
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "uid=name",
							Changes: []ldap.Change{
								{
									Operation: ldap.DeleteAttribute,
									Modification: ldap.PartialAttribute{
										Type: "member",
										Vals: []string{"uid=name"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN:       "uid=name",
							Changes:  []ldap.Change(nil),
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"uid=name",
							ldap.ScopeBaseObject,
							ldap.NeverDerefAliases, 1, 0, false,
							"(objectClass=inetOrgPerson)",
							ldapUserAttributes,
							[]ldap.Control(nil),
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "uid",
											Values: []string{"testUser"},
										},
										{
											Name:   "displayname",
											Values: []string{"testUser"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
		},
		{
			name: "Test changing userType",
			args: args{
				nameOrID: "testUser",
				userProps: userProps{
					userType: &memberType,
				},
				disableUserMechanism: "group",
			},
			want: &userProps{
				id:                       "testUser",
				mail:                     "testuser@example.org",
				displayName:              "testUser",
				onPremisesSamAccountName: "testUser",
				userType:                 &memberType,
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=people,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=inetOrgPerson)(|(uid=testUser)(entryUUID=testUser)))",
							ldapUserAttributes,
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "displayname",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserTypeAttribute,
											Values: []string{"Guest"},
										},
									},
								},
							},
						},
						nil,
					},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN: "uid=name",
							Changes: []ldap.Change{
								{
									Operation: ldap.ReplaceAttribute,
									Modification: ldap.PartialAttribute{
										Type: "userTypeAttribute",
										Vals: []string{"Member"},
									},
								},
							},
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
				{
					funcName: "Modify",
					args: []interface{}{
						&ldap.ModifyRequest{
							DN:       "uid=name",
							Changes:  []ldap.Change(nil),
							Controls: []ldap.Control(nil),
						},
					},
					returns: []interface{}{nil},
				},
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"uid=name",
							ldap.ScopeBaseObject,
							ldap.NeverDerefAliases, 1, 0, false,
							"(objectClass=inetOrgPerson)",
							ldapUserAttributes,
							[]ldap.Control(nil),
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "uid=name",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "uid",
											Values: []string{"testUser"},
										},
										{
											Name:   "displayname",
											Values: []string{"testUser"},
										},
										{
											Name:   "entryUUID",
											Values: []string{"testUser"},
										},
										{
											Name:   "mail",
											Values: []string{"testuser@example.org"},
										},
										{
											Name:   lconfig.UserTypeAttribute,
											Values: []string{"Member"},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := &mocks.Client{}
			for _, ldapMock := range tt.ldapMocks {
				lm.On(ldapMock.funcName, ldapMock.args...).Return(ldapMock.returns...)
			}

			ldapConfig := lconfig
			ldapConfig.DisableUserMechanism = tt.args.disableUserMechanism
			i, _ := getMockedBackend(lm, ldapConfig, &logger)

			user := libregraph.UserUpdate{
				Id:                       &tt.args.userProps.id,
				Mail:                     &tt.args.userProps.mail,
				DisplayName:              &tt.args.userProps.displayName,
				OnPremisesSamAccountName: &tt.args.userProps.onPremisesSamAccountName,
				AccountEnabled:           tt.args.userProps.accountEnabled,
				UserType:                 tt.args.userProps.userType,
			}

			emptyString := ""
			var want *libregraph.User = nil
			if tt.want != nil {
				want = &libregraph.User{
					Id:                       &tt.want.id,
					Mail:                     &tt.want.mail,
					DisplayName:              tt.want.displayName,
					OnPremisesSamAccountName: tt.want.onPremisesSamAccountName,
					Surname:                  &emptyString,
					GivenName:                tt.want.givenName,
					UserType:                 tt.want.userType,
				}

				if tt.want.accountEnabled != nil {
					want.AccountEnabled = tt.want.accountEnabled
				}
			}

			got, err := i.UpdateUser(context.Background(), tt.args.nameOrID, user)
			tt.assertion(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestUsersEnabledState(t *testing.T) {
	aliceEnabled := ldap.Entry{
		DN: "alice",
		Attributes: []*ldap.EntryAttribute{
			{
				Name:   lconfig.UserEnabledAttribute,
				Values: []string{"TRUE"},
			},
		},
	}

	bobDisabled := ldap.Entry{
		DN: "bob",
		Attributes: []*ldap.EntryAttribute{
			{
				Name:   lconfig.UserEnabledAttribute,
				Values: []string{"FALSE"},
			},
		},
	}

	carolImplicitlyEnabled := ldap.Entry{
		DN: "carol",
		Attributes: []*ldap.EntryAttribute{
			{
				Name: lconfig.UserEnabledAttribute,
			},
		},
	}

	type args struct {
		usersToCheck         []*ldap.Entry
		expectedUsers        map[string]bool
		disableUserMechanism string
	}
	type mockInputs struct {
		funcName string
		args     []interface{}
		returns  []interface{}
	}
	tests := []struct {
		name      string
		args      args
		want      map[string]bool
		assertion assert.ErrorAssertionFunc
		ldapMocks []mockInputs
	}{
		{
			name: "Test no users",
			args: args{
				usersToCheck:         []*ldap.Entry{},
				expectedUsers:        map[string]bool{},
				disableUserMechanism: "attribute",
			},
			want: map[string]bool{},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{},
		},
		{
			name: "Test attribute enabled users",
			args: args{
				usersToCheck:         []*ldap.Entry{&aliceEnabled, &bobDisabled, &carolImplicitlyEnabled},
				expectedUsers:        map[string]bool{"alice": true, "bob": false, "carol": true},
				disableUserMechanism: "attribute",
			},
			want: map[string]bool{"alice": true, "bob": false, "carol": true},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{},
		},
		{
			name: "Test attribute enabled users not in disabled group",
			args: args{
				usersToCheck:         []*ldap.Entry{&aliceEnabled, &bobDisabled, &carolImplicitlyEnabled},
				expectedUsers:        map[string]bool{"alice": true, "bob": false, "carol": true},
				disableUserMechanism: "group",
			},
			want: map[string]bool{"alice": true, "bob": true, "carol": true},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							disableUsersGroup,
							ldap.ScopeBaseObject,
							ldap.NeverDerefAliases, 1, 0, false,
							groupSearchFilter,
							[]string{"member"},
							[]ldap.Control(nil),
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "cn=DisabledGroup",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "member",
											Values: []string{""},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
		},
		{
			name: "Test attribute enabled users in disabled group",
			args: args{
				usersToCheck:         []*ldap.Entry{&aliceEnabled, &bobDisabled, &carolImplicitlyEnabled},
				expectedUsers:        map[string]bool{"alice": true, "bob": false, "carol": true},
				disableUserMechanism: "group",
			},
			want: map[string]bool{"alice": false, "bob": true, "carol": false},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							disableUsersGroup,
							ldap.ScopeBaseObject,
							ldap.NeverDerefAliases, 1, 0, false,
							groupSearchFilter,
							[]string{"member"},
							[]ldap.Control(nil),
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "cn=DisabledGroup",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "member",
											Values: []string{"alice", "carol"},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
		},
		{
			name: "Test local group disable when ldap is throwing error",
			args: args{
				usersToCheck:         []*ldap.Entry{&aliceEnabled, &bobDisabled, &carolImplicitlyEnabled},
				expectedUsers:        map[string]bool{"alice": true, "bob": false, "carol": true},
				disableUserMechanism: "group",
			},
			want: nil,
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.NotNil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							disableUsersGroup,
							ldap.ScopeBaseObject,
							ldap.NeverDerefAliases, 1, 0, false,
							groupSearchFilter,
							[]string{"member"},
							[]ldap.Control(nil),
						),
					},
					returns: []interface{}{
						nil,
						&ldap.Error{
							Err: fmt.Errorf("very problematic problems"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := &mocks.Client{}
			for _, ldapMock := range tt.ldapMocks {
				lm.On(ldapMock.funcName, ldapMock.args...).Return(ldapMock.returns...)
			}

			ldapConfig := lconfig
			ldapConfig.DisableUserMechanism = tt.args.disableUserMechanism
			i, _ := getMockedBackend(lm, ldapConfig, &logger)

			got, err := i.usersEnabledState(tt.args.usersToCheck)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
