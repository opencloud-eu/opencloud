package identity

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/CiscoM31/godata"
	"github.com/go-ldap/ldap/v3"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var groupEntry = ldap.NewEntry("cn=group",
	map[string][]string{
		"cn":        {"group"},
		"entryuuid": {"abcd-defg"},
		"member": {
			"uid=user,ou=people,dc=test",
			"uid=invalid,ou=people,dc=test",
		},
	})

var invalidGroupEntry = ldap.NewEntry("cn=invalid",
	map[string][]string{
		"cn": {"invalid"},
	})

var queryParamExpand = url.Values{
	"$expand": []string{"members"},
}

var queryParamSelect = url.Values{
	"$select": []string{"members"},
}

var groupLookupSearchRequest = &ldap.SearchRequest{
	BaseDN:     "ou=groups,dc=test",
	Scope:      2,
	SizeLimit:  1,
	Filter:     "(&(objectClass=groupOfNames)(|(cn=group)(entryUUID=group)))",
	Attributes: []string{"cn", "entryUUID", "member"},
	Controls:   []ldap.Control(nil),
}

var groupListSearchRequest = &ldap.SearchRequest{
	BaseDN:     "ou=groups,dc=test",
	Scope:      2,
	Filter:     "(&(objectClass=groupOfNames))",
	Attributes: []string{"cn", "entryUUID", "member"},
	Controls:   []ldap.Control(nil),
}

func TestGetGroup(t *testing.T) {
	// Mock a Sizelimit Error
	lm := &mocks.Client{}
	lm.On("Search", mock.Anything).Return(nil, ldap.NewError(ldap.LDAPResultSizeLimitExceeded, errors.New("mock")))

	b, _ := getMockedBackend(lm, lconfig, &logger)
	_, err := b.GetGroup(context.Background(), "group", nil)
	assert.ErrorContains(t, err, "itemNotFound:")
	_, err = b.GetGroup(context.Background(), "group", queryParamExpand)
	assert.ErrorContains(t, err, "itemNotFound:")
	_, err = b.GetGroup(context.Background(), "group", queryParamSelect)
	assert.ErrorContains(t, err, "itemNotFound:")

	// Mock an empty Search Result
	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).Return(&ldap.SearchResult{}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	_, err = b.GetGroup(context.Background(), "group", nil)
	assert.ErrorContains(t, err, "itemNotFound:")
	_, err = b.GetGroup(context.Background(), "group", queryParamExpand)
	assert.ErrorContains(t, err, "itemNotFound:")
	_, err = b.GetGroup(context.Background(), "group", queryParamSelect)
	assert.ErrorContains(t, err, "itemNotFound:")

	// Mock an invalid Search Result
	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).Return(&ldap.SearchResult{
		Entries: []*ldap.Entry{invalidGroupEntry},
	}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	_, err = b.GetGroup(context.Background(), "group", nil)
	assert.ErrorContains(t, err, "itemNotFound:")
	_, err = b.GetGroup(context.Background(), "group", queryParamExpand)
	assert.ErrorContains(t, err, "itemNotFound:")
	_, err = b.GetGroup(context.Background(), "group", queryParamSelect)
	assert.ErrorContains(t, err, "itemNotFound:")

	// Mock a valid	Search Result
	lm = &mocks.Client{}
	sr2 := &ldap.SearchRequest{
		BaseDN:     "uid=user,ou=people,dc=test",
		SizeLimit:  1,
		Filter:     "(objectClass=inetOrgPerson)",
		Attributes: ldapUserAttributes,
		Controls:   []ldap.Control(nil),
	}
	sr3 := &ldap.SearchRequest{
		BaseDN:     "uid=invalid,ou=people,dc=test",
		SizeLimit:  1,
		Filter:     "(objectClass=inetOrgPerson)",
		Attributes: ldapUserAttributes,
		Controls:   []ldap.Control(nil),
	}

	lm.On("Search", groupLookupSearchRequest).Return(&ldap.SearchResult{Entries: []*ldap.Entry{groupEntry}}, nil)
	lm.On("Search", sr2).Return(&ldap.SearchResult{Entries: []*ldap.Entry{userEntry}}, nil)
	lm.On("Search", sr3).Return(&ldap.SearchResult{Entries: []*ldap.Entry{invalidUserEntry}}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	g, err := b.GetGroup(context.Background(), "group", nil)
	if err != nil {
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	} else if *g.Id != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id) {
		t.Errorf("Expected GetGroup to return a valid group")
	}
	g, err = b.GetGroup(context.Background(), "group", queryParamExpand)
	switch {
	case err != nil:
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	case g.GetId() != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id):
		t.Errorf("Expected GetGroup to return a valid group")
	case len(g.Members) != 1:
		t.Errorf("Expected GetGroup with expand to return one member")
	case g.Members[0].GetId() != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id):
		t.Errorf("Expected GetGroup with expand to return correct member")
	}
	g, err = b.GetGroup(context.Background(), "group", queryParamSelect)
	switch {
	case err != nil:
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	case g.GetId() != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id):
		t.Errorf("Expected GetGroup to return a valid group")
	case len(g.Members) != 1:
		t.Errorf("Expected GetGroup with expand to return one member")
	case g.Members[0].GetId() != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id):
		t.Errorf("Expected GetGroup with expand to return correct member")
	}
}

func TestGetGroupReadOnlyBackend(t *testing.T) {
	readOnlyConfig := lconfig
	readOnlyConfig.WriteEnabled = false
	readOnlyConfig.GroupBaseDN = "ou=groups,dc=test"
	readOnlyConfig.GroupCreateBaseDN = "ou=local,ou=group,dc=test"
	localGroupEntry := groupEntry
	localGroupEntry.DN = "cn=local,ou=local,o=base"

	lm := &mocks.Client{}
	lm.On("Search", groupLookupSearchRequest).Return(&ldap.SearchResult{Entries: []*ldap.Entry{groupEntry}}, nil)
	b, _ := getMockedBackend(lm, readOnlyConfig, &logger)
	g, err := b.GetGroup(context.Background(), "group", url.Values{})
	switch {
	case err != nil:
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	case g.GetId() != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id):
		t.Errorf("Expected GetGroup to return a valid group")
	}
	types := g.GetGroupTypes()
	switch {
	case len(types) == 0:
		t.Errorf("No groupTypes attribute on readonly Group")
	case len(types) > 1:
		t.Errorf("Expected a single groupTypes value on readonly Group")
	case types[0] != "ReadOnly":
		t.Errorf("Expected a groupTypes 'ReadOnly' on readonly Group")
	}
}
func TestGetGroupReadOnlySubtree(t *testing.T) {
	readOnlyTreeConfig := lconfig
	readOnlyTreeConfig.GroupCreateBaseDN = "ou=write,ou=groups,dc=test"
	var writeGroupEntry = ldap.NewEntry("cn=group,ou=write,ou=groups,dc=test",
		map[string][]string{
			"cn":        {"group"},
			"entryuuid": {"abcd-defg"},
			"member": {
				"uid=user,ou=people,dc=test",
				"uid=invalid,ou=people,dc=test",
			},
		})

	lm := &mocks.Client{}
	lm.On("Search", groupLookupSearchRequest).Return(&ldap.SearchResult{Entries: []*ldap.Entry{groupEntry}}, nil)
	b, _ := getMockedBackend(lm, readOnlyTreeConfig, &logger)
	g, err := b.GetGroup(context.Background(), "group", url.Values{})
	switch {
	case err != nil:
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	case g.GetId() != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id):
		t.Errorf("Expected GetGroup to return a valid group")
	}
	types := g.GetGroupTypes()
	switch {
	case len(types) == 0:
		t.Errorf("No groupTypes attribute on readonly Group")
	case len(types) > 1:
		t.Errorf("Expected a single groupTypes value on readonly Group")
	case types[0] != "ReadOnly":
		t.Errorf("Expected a groupTypes 'ReadOnly' on readonly Group")
	}

	lm = &mocks.Client{}
	lm.On("Search", groupLookupSearchRequest).Return(&ldap.SearchResult{Entries: []*ldap.Entry{writeGroupEntry}}, nil)
	b, _ = getMockedBackend(lm, readOnlyTreeConfig, &logger)
	g, err = b.GetGroup(context.Background(), "group", url.Values{})
	switch {
	case err != nil:
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	case g.GetId() != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id):
		t.Errorf("Expected GetGroup to return a valid group")
	}
	types = g.GetGroupTypes()
	if len(types) != 0 {
		t.Errorf("No groupTypes attribute expected on writeable Group")
	}
}

func TestGetGroups(t *testing.T) {
	lm := &mocks.Client{}
	oDataReq, err := godata.ParseRequest(context.Background(), "", url.Values{})
	if err != nil {
		t.Errorf("Expected success, got '%s'", err.Error())
	}
	lm.On("Search", mock.Anything).Return(nil, ldap.NewError(ldap.LDAPResultOperationsError, errors.New("mock")))
	b, _ := getMockedBackend(lm, lconfig, &logger)
	_, err = b.GetGroups(context.Background(), oDataReq)
	assert.ErrorContains(t, err, "itemNotFound:")

	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).Return(&ldap.SearchResult{}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	g, err := b.GetGroups(context.Background(), oDataReq)
	if err != nil {
		t.Errorf("Expected success, got '%s'", err.Error())
	} else if g == nil || len(g) != 0 {
		t.Errorf("Expected zero length user slice")
	}

	lm = &mocks.Client{}
	lm.On("Search", mock.Anything).Return(&ldap.SearchResult{
		Entries: []*ldap.Entry{groupEntry},
	}, nil)
	b, _ = getMockedBackend(lm, lconfig, &logger)
	g, err = b.GetGroups(context.Background(), oDataReq)
	if err != nil {
		t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
	} else if *g[0].Id != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id) {
		t.Errorf("Expected GetGroup to return a valid group")
	}

	// Mock a valid	Search Result with expanded group members
	lm = &mocks.Client{}
	sr2 := &ldap.SearchRequest{
		BaseDN:     "uid=user,ou=people,dc=test",
		SizeLimit:  1,
		Filter:     "(objectClass=inetOrgPerson)",
		Attributes: ldapUserAttributes,
		Controls:   []ldap.Control(nil),
	}
	sr3 := &ldap.SearchRequest{
		BaseDN:     "uid=invalid,ou=people,dc=test",
		SizeLimit:  1,
		Filter:     "(objectClass=inetOrgPerson)",
		Attributes: ldapUserAttributes,
		Controls:   []ldap.Control(nil),
	}

	for _, param := range []url.Values{queryParamSelect, queryParamExpand} {
		oDataReq, err := godata.ParseRequest(context.Background(), "", param)
		if err != nil {
			t.Errorf("Expected success, got '%s'", err.Error())
		}
		lm.On("Search", groupListSearchRequest).Return(&ldap.SearchResult{Entries: []*ldap.Entry{groupEntry}}, nil)
		lm.On("Search", sr2).Return(&ldap.SearchResult{Entries: []*ldap.Entry{userEntry}}, nil)
		lm.On("Search", sr3).Return(&ldap.SearchResult{Entries: []*ldap.Entry{invalidUserEntry}}, nil)
		b, _ = getMockedBackend(lm, lconfig, &logger)
		g, err = b.GetGroups(context.Background(), oDataReq)
		switch {
		case err != nil:
			t.Errorf("Expected GetGroup to succeed. Got %s", err.Error())
		case g[0].GetId() != groupEntry.GetEqualFoldAttributeValue(b.groupAttributeMap.id):
			t.Errorf("Expected GetGroup to return a valid group")
		case len(g[0].Members) != 1:
			t.Errorf("Expected GetGroup to return group with one member")
		case g[0].Members[0].GetId() != userEntry.GetEqualFoldAttributeValue(b.userAttributeMap.id):
			t.Errorf("Expected GetGroup to return group with correct member")
		}
	}
}

func TestGetGroupsSearch(t *testing.T) {
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
			return req.Filter == "(&(objectClass=groupOfNames)(cn=*term*))"
		})).
		Return(&ldap.SearchResult{}, nil)
	b, _ := getMockedBackend(lm, lconfig, &logger)
	g, err := b.GetGroups(context.Background(), odataReqDefault)
	if err != nil {
		t.Errorf("Expected success, got '%s'", err.Error())
	} else if g == nil || len(g) != 0 {
		t.Errorf("Expected zero length user slice")
	}
}
func TestUpdateGroupName(t *testing.T) {
	groupDn := "cn=TheGroup,ou=groups,dc=example,dc=org"

	type args struct {
		groupId   string
		groupName string
		newName   string
	}

	type mockInputs struct {
		funcName string
		args     []interface{}
		returns  []interface{}
	}

	tests := []struct {
		name      string
		args      args
		assertion assert.ErrorAssertionFunc
		ldapMocks []mockInputs
	}{
		{
			name: "Test with no name change",
			args: args{
				groupId: "some-uuid-string",
				newName: "TheGroup",
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=groups,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=groupOfNames)(entryUUID=some-uuid-string))",
							[]string{"cn", "entryUUID", "member"},
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: groupDn,
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "cn",
											Values: []string{"TheGroup"},
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
			name: "Test with name change",
			args: args{
				groupId: "some-uuid-string",
				newName: "TheGroupWithShinyNewName",
			},
			assertion: func(t assert.TestingT, err error, args ...interface{}) bool {
				return assert.Nil(t, err, args...)
			},
			ldapMocks: []mockInputs{
				{
					funcName: "Search",
					args: []interface{}{
						ldap.NewSearchRequest(
							"ou=groups,dc=test",
							ldap.ScopeWholeSubtree,
							ldap.NeverDerefAliases, 1, 0, false,
							"(&(objectClass=groupOfNames)(entryUUID=some-uuid-string))",
							[]string{"cn", "entryUUID", "member"},
							nil,
						),
					},
					returns: []interface{}{
						&ldap.SearchResult{
							Entries: []*ldap.Entry{
								{
									DN: "cn=TheGroup,ou=groups,dc=example,dc=org",
									Attributes: []*ldap.EntryAttribute{
										{
											Name:   "cn",
											Values: []string{"TheGroup"},
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
							DN:           groupDn,
							NewRDN:       "cn=TheGroupWithShinyNewName",
							DeleteOldRDN: true,
							NewSuperior:  "",
							Controls:     []ldap.Control(nil),
						},
					},
					returns: []interface{}{
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
			i, _ := getMockedBackend(lm, ldapConfig, &logger)

			err := i.UpdateGroupName(context.Background(), tt.args.groupId, tt.args.newName)
			tt.assertion(t, err)
		})
	}
}
