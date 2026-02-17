package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons  *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service  Service         `yaml:"-"`
	LogLevel string          `yaml:"loglevel" env:"OC_LOG_LEVEL;AUTH_BASIC_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	SkipUserGroupsInToken bool          `yaml:"skip_user_groups_in_token" env:"AUTH_BASIC_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the encoding of the user's group memberships in the reva access token. This reduces the token size, especially when users are members of a large number of groups." introductionVersion:"1.0.0"`
	AuthProvider          string        `yaml:"auth_provider" env:"AUTH_BASIC_AUTH_MANAGER" desc:"The authentication manager to check if credentials are valid. Supported value is 'ldap'." introductionVersion:"1.0.0"`
	AuthProviders         AuthProviders `yaml:"auth_providers"`

	Context context.Context `yaml:"-"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"AUTH_BASIC_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"AUTH_BASIC_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"AUTH_BASIC_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"AUTH_BASIC_DEBUG_ZPAGES" desc:"Enables zpages, which can  be used for collecting and viewing traces in-memory." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"AUTH_BASIC_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;AUTH_BASIC_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type AuthProviders struct {
	LDAP        LDAPProvider        `yaml:"ldap"`
	OwnCloudSQL OwnCloudSQLProvider `yaml:"owncloudsql"`
	JSON        JSONProvider        `yaml:"json,omitempty"` // not supported by the OpenCloud product, therefore not part of docs
}

type JSONProvider struct {
	File string `yaml:"file,omitempty"`
}

type LDAPProvider struct {
	URI                      string          `yaml:"uri" env:"OC_LDAP_URI;AUTH_BASIC_LDAP_URI" desc:"URI of the LDAP Server to connect to. Supported URI schemes are 'ldaps://' and 'ldap://'" introductionVersion:"1.0.0"`
	CACert                   string          `yaml:"ca_cert" env:"OC_LDAP_CACERT;AUTH_BASIC_LDAP_CACERT" desc:"Path/File name for the root CA certificate (in PEM format) used to validate TLS server certificates of the LDAP service. If not defined, the root directory derives from $OC_BASE_DATA_PATH/idm." introductionVersion:"1.0.0"`
	Insecure                 bool            `yaml:"insecure" env:"OC_LDAP_INSECURE;AUTH_BASIC_LDAP_INSECURE" desc:"Disable TLS certificate validation for the LDAP connections. Do not set this in production environments." introductionVersion:"1.0.0"`
	BindDN                   string          `yaml:"bind_dn" env:"OC_LDAP_BIND_DN;AUTH_BASIC_LDAP_BIND_DN" desc:"LDAP DN to use for simple bind authentication with the target LDAP server." introductionVersion:"1.0.0"`
	BindPassword             string          `yaml:"bind_password" env:"OC_LDAP_BIND_PASSWORD;AUTH_BASIC_LDAP_BIND_PASSWORD" desc:"Password to use for authenticating the 'bind_dn'." introductionVersion:"1.0.0"`
	UserBaseDN               string          `yaml:"user_base_dn" env:"OC_LDAP_USER_BASE_DN;AUTH_BASIC_LDAP_USER_BASE_DN" desc:"Search base DN for looking up LDAP users." introductionVersion:"1.0.0"`
	GroupBaseDN              string          `yaml:"group_base_dn" env:"OC_LDAP_GROUP_BASE_DN;AUTH_BASIC_LDAP_GROUP_BASE_DN" desc:"Search base DN for looking up LDAP groups." introductionVersion:"1.0.0"`
	UserScope                string          `yaml:"user_scope" env:"OC_LDAP_USER_SCOPE;AUTH_BASIC_LDAP_USER_SCOPE" desc:"LDAP search scope to use when looking up users. Supported values are 'base', 'one' and 'sub'." introductionVersion:"1.0.0"`
	GroupScope               string          `yaml:"group_scope" env:"OC_LDAP_GROUP_SCOPE;AUTH_BASIC_LDAP_GROUP_SCOPE" desc:"LDAP search scope to use when looking up groups. Supported values are 'base', 'one' and 'sub'." introductionVersion:"1.0.0"`
	UserFilter               string          `yaml:"user_filter" env:"OC_LDAP_USER_FILTER;AUTH_BASIC_LDAP_USER_FILTER" desc:"LDAP filter to add to the default filters for user search like '(objectclass=openCloudUser)'." introductionVersion:"1.0.0"`
	GroupFilter              string          `yaml:"group_filter" env:"OC_LDAP_GROUP_FILTER;AUTH_BASIC_LDAP_GROUP_FILTER" desc:"LDAP filter to add to the default filters for group searches." introductionVersion:"1.0.0"`
	UserObjectClass          string          `yaml:"user_object_class" env:"OC_LDAP_USER_OBJECTCLASS;AUTH_BASIC_LDAP_USER_OBJECTCLASS" desc:"The object class to use for users in the default user search filter ('inetOrgPerson')." introductionVersion:"1.0.0"`
	GroupObjectClass         string          `yaml:"group_object_class" env:"OC_LDAP_GROUP_OBJECTCLASS;AUTH_BASIC_LDAP_GROUP_OBJECTCLASS" desc:"The object class to use for groups in the default group search filter ('groupOfNames')." introductionVersion:"1.0.0"`
	LoginAttributes          []string        `yaml:"login_attributes" env:"LDAP_LOGIN_ATTRIBUTES;AUTH_BASIC_LDAP_LOGIN_ATTRIBUTES" desc:"A list of user object attributes that can be used for login. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	IDP                      string          `yaml:"idp" env:"OC_URL;OC_OIDC_ISSUER;AUTH_BASIC_IDP_URL" desc:"The identity provider value to set in the userids of the CS3 user objects for users returned by this user provider." introductionVersion:"1.0.0"`
	DisableUserMechanism     string          `yaml:"disable_user_mechanism" env:"OC_LDAP_DISABLE_USER_MECHANISM;AUTH_BASIC_DISABLE_USER_MECHANISM" desc:"An option to control the behavior for disabling users. Valid options are 'none', 'attribute' and 'group'. If set to 'group', disabling a user via API will add the user to the configured group for disabled users, if set to 'attribute' this will be done in the ldap user entry, if set to 'none' the disable request is not processed." introductionVersion:"1.0.0"`
	LdapDisabledUsersGroupDN string          `yaml:"ldap_disabled_users_group_dn" env:"OC_LDAP_DISABLED_USERS_GROUP_DN;AUTH_BASIC_DISABLED_USERS_GROUP_DN" desc:"The distinguished name of the group to which added users will be classified as disabled when 'disable_user_mechanism' is set to 'group'." introductionVersion:"1.0.0"`
	UserSchema               LDAPUserSchema  `yaml:"user_schema"`
	GroupSchema              LDAPGroupSchema `yaml:"group_schema"`
}

type LDAPUserSchema struct {
	ID              string `yaml:"id" env:"OC_LDAP_USER_SCHEMA_ID;AUTH_BASIC_LDAP_USER_SCHEMA_ID" desc:"LDAP Attribute to use as the unique ID for users. This should be a stable globally unique ID like a UUID." introductionVersion:"1.0.0"`
	TenantID        string `yaml:"tenant_id" env:"OC_LDAP_USER_SCHEMA_TENANT_ID;AUTH_BASIC_LDAP_USER_SCHEMA_TENANT_ID" desc:"LDAP Attribute to use for the tenant ID of users. This is used to identify the tenant of a user in a multi-tenant environment." introductionVersion:"4.0.0"`
	IDIsOctetString bool   `yaml:"id_is_octet_string" env:"OC_LDAP_USER_SCHEMA_ID_IS_OCTETSTRING;AUTH_BASIC_LDAP_USER_SCHEMA_ID_IS_OCTETSTRING" desc:"Set this to true if the defined 'ID' attribute for users is of the 'OCTETSTRING' syntax. This is e.g. required when using the 'objectGUID' attribute of Active Directory for the user IDs." introductionVersion:"1.0.0"`
	Mail            string `yaml:"mail" env:"OC_LDAP_USER_SCHEMA_MAIL;AUTH_BASIC_LDAP_USER_SCHEMA_MAIL" desc:"LDAP Attribute to use for the email address of users." introductionVersion:"1.0.0"`
	DisplayName     string `yaml:"display_name" env:"OC_LDAP_USER_SCHEMA_DISPLAYNAME;AUTH_BASIC_LDAP_USER_SCHEMA_DISPLAYNAME" desc:"LDAP Attribute to use for the displayname of users." introductionVersion:"1.0.0"`
	Username        string `yaml:"user_name" env:"OC_LDAP_USER_SCHEMA_USERNAME;AUTH_BASIC_LDAP_USER_SCHEMA_USERNAME" desc:"LDAP Attribute to use for username of users." introductionVersion:"1.0.0"`
	Enabled         string `yaml:"user_enabled" env:"OC_LDAP_USER_ENABLED_ATTRIBUTE;AUTH_BASIC_LDAP_USER_ENABLED_ATTRIBUTE" desc:"LDAP attribute to use as a flag telling if the user is enabled or disabled." introductionVersion:"1.0.0"`
}

type LDAPGroupSchema struct {
	ID              string `yaml:"id" env:"OC_LDAP_GROUP_SCHEMA_ID;AUTH_BASIC_LDAP_GROUP_SCHEMA_ID" desc:"LDAP Attribute to use as the unique id for groups. This should be a stable globally unique id (e.g. a UUID)." introductionVersion:"1.0.0"`
	IDIsOctetString bool   `yaml:"id_is_octet_string" env:"OC_LDAP_GROUP_SCHEMA_ID_IS_OCTETSTRING;AUTH_BASIC_LDAP_GROUP_SCHEMA_ID_IS_OCTETSTRING" desc:"Set this to true if the defined 'id' attribute for groups is of the 'OCTETSTRING' syntax. This is e.g. required when using the 'objectGUID' attribute of Active Directory for the group IDs." introductionVersion:"1.0.0"`
	Mail            string `yaml:"mail" env:"OC_LDAP_GROUP_SCHEMA_MAIL;AUTH_BASIC_LDAP_GROUP_SCHEMA_MAIL" desc:"LDAP Attribute to use for the email address of groups (can be empty)." introductionVersion:"1.0.0"`
	DisplayName     string `yaml:"display_name" env:"OC_LDAP_GROUP_SCHEMA_DISPLAYNAME;AUTH_BASIC_LDAP_GROUP_SCHEMA_DISPLAYNAME" desc:"LDAP Attribute to use for the displayname of groups (often the same as groupname attribute)." introductionVersion:"1.0.0"`
	Groupname       string `yaml:"group_name" env:"OC_LDAP_GROUP_SCHEMA_GROUPNAME;AUTH_BASIC_LDAP_GROUP_SCHEMA_GROUPNAME" desc:"LDAP Attribute to use for the name of groups." introductionVersion:"1.0.0"`
	Member          string `yaml:"member" env:"OC_LDAP_GROUP_SCHEMA_MEMBER;AUTH_BASIC_LDAP_GROUP_SCHEMA_MEMBER" desc:"LDAP Attribute that is used for group members." introductionVersion:"1.0.0"`
}

type OwnCloudSQLProvider struct {
	DBUsername       string `yaml:"db_username" env:"AUTH_BASIC_OWNCLOUDSQL_DB_USERNAME" desc:"Database user to use for authenticating with the owncloud database." introductionVersion:"1.0.0"`
	DBPassword       string `yaml:"db_password" env:"AUTH_BASIC_OWNCLOUDSQL_DB_PASSWORD" desc:"Password for the database user." introductionVersion:"1.0.0"`
	DBHost           string `yaml:"db_host" env:"AUTH_BASIC_OWNCLOUDSQL_DB_HOST" desc:"Hostname of the database server." introductionVersion:"1.0.0"`
	DBPort           int    `yaml:"db_port" env:"AUTH_BASIC_OWNCLOUDSQL_DB_PORT" desc:"Network port to use for the database connection." introductionVersion:"1.0.0"`
	DBName           string `yaml:"db_name" env:"AUTH_BASIC_OWNCLOUDSQL_DB_NAME" desc:"Name of the owncloud database." introductionVersion:"1.0.0"`
	IDP              string `yaml:"idp" env:"AUTH_BASIC_OWNCLOUDSQL_IDP" desc:"The identity provider value to set in the userids of the CS3 user objects for users returned by this user provider." introductionVersion:"1.0.0"`
	Nobody           int64  `yaml:"nobody" env:"AUTH_BASIC_OWNCLOUDSQL_NOBODY" desc:"Fallback number if no numeric UID and GID properties are provided." introductionVersion:"1.0.0"`
	JoinUsername     bool   `yaml:"join_username" env:"AUTH_BASIC_OWNCLOUDSQL_JOIN_USERNAME" desc:"Join the user properties table to read usernames" introductionVersion:"1.0.0"`
	JoinOwnCloudUUID bool   `yaml:"join_owncloud_uuid" env:"AUTH_BASIC_OWNCLOUDSQL_JOIN_OWNCLOUD_UUID" desc:"Join the user properties table to read user ID's." introductionVersion:"1.0.0"`
}
