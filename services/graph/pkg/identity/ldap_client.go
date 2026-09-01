package identity

//go:generate $GOWRAP gen -g -i LdapClient -t ./ldap_client_prometheus.tmpl -o ldap_client_prometheus.go
//go:generate $GOWRAP gen -g -i LdapClient -t ./ldap_client_goldap.tmpl -o ldap_client_goldap.go

import (
	"github.com/go-ldap/ldap/v3"
)

const (
	LdapOpAdd            = "add"
	LdapOpDel            = "del"
	LdapOpModify         = "modify"
	LdapOpModifyDN       = "modify-dn"
	LdapOpPasswordModify = "modify-password"
	LdapOpSearch         = "search"
)

type LdapClient interface {
	Add(*ldap.AddRequest) error
	Del(*ldap.DelRequest) error
	Modify(*ldap.ModifyRequest) error
	ModifyDN(*ldap.ModifyDNRequest) error
	PasswordModify(*ldap.PasswordModifyRequest) (*ldap.PasswordModifyResult, error)
	Search(*ldap.SearchRequest) (*ldap.SearchResult, error)
}
