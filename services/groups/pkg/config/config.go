package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons  *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service  Service         `yaml:"-"`
	LogLevel string          `yaml:"loglevel" env:"OC_LOG_LEVEL;GROUPS_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"GROUPS_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the loading of user's group memberships from the reva access token." introductionVersion:"1.0.0"`

	Driver  string  `yaml:"driver" env:"GROUPS_DRIVER" desc:"The driver which should be used by the groups service. Supported values are 'ldap' and 'owncloudsql'." introductionVersion:"1.0.0"`
	Drivers Drivers `yaml:"drivers"`

	Context context.Context `yaml:"-"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"GROUPS_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"GROUPS_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"GROUPS_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"GROUPS_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"GROUPS_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;GROUPS_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type Drivers struct {
	LDAP        LDAPDriver        `yaml:"ldap"`
	OwnCloudSQL OwnCloudSQLDriver `yaml:"owncloudsql"`

	JSON JSONDriver   `yaml:"json,omitempty"` // not supported by OpenCloud, therefore not part of docs
	REST RESTProvider `yaml:"rest,omitempty"` // not supported by OpenCloud, therefore not part of docs
}

type LDAPDriver struct {
	URI                      string          `yaml:"uri" env:"OC_LDAP_URI;GROUPS_LDAP_URI" desc:"URI of the LDAP Server to connect to. Supported URI schemes are 'ldaps://' and 'ldap://'" introductionVersion:"1.0.0"`
	CACert                   string          `yaml:"ca_cert" env:"OC_LDAP_CACERT;GROUPS_LDAP_CACERT" desc:"Path/File name for the root CA certificate (in PEM format) used to validate TLS server certificates of the LDAP service. If not defined, the root directory derives from $OC_BASE_DATA_PATH/idm." introductionVersion:"1.0.0"`
	Insecure                 bool            `yaml:"insecure" env:"OC_LDAP_INSECURE;GROUPS_LDAP_INSECURE" desc:"Disable TLS certificate validation for the LDAP connections. Do not set this in production environments." introductionVersion:"1.0.0"`
	BindDN                   string          `yaml:"bind_dn" env:"OC_LDAP_BIND_DN;GROUPS_LDAP_BIND_DN" desc:"LDAP DN to use for simple bind authentication with the target LDAP server." introductionVersion:"1.0.0"`
	BindPassword             string          `yaml:"bind_password" env:"OC_LDAP_BIND_PASSWORD;GROUPS_LDAP_BIND_PASSWORD" desc:"Password to use for authenticating the 'bind_dn'." introductionVersion:"1.0.0"`
	UserBaseDN               string          `yaml:"user_base_dn" env:"OC_LDAP_USER_BASE_DN;GROUPS_LDAP_USER_BASE_DN" desc:"Search base DN for looking up LDAP users." introductionVersion:"1.0.0"`
	GroupBaseDN              string          `yaml:"group_base_dn" env:"OC_LDAP_GROUP_BASE_DN;GROUPS_LDAP_GROUP_BASE_DN" desc:"Search base DN for looking up LDAP groups." introductionVersion:"1.0.0"`
	UserScope                string          `yaml:"user_scope" env:"OC_LDAP_USER_SCOPE;GROUPS_LDAP_USER_SCOPE" desc:"LDAP search scope to use when looking up users. Supported scopes are 'base', 'one' and 'sub'." introductionVersion:"1.0.0"`
	GroupScope               string          `yaml:"group_scope" env:"OC_LDAP_GROUP_SCOPE;GROUPS_LDAP_GROUP_SCOPE" desc:"LDAP search scope to use when looking up groups. Supported scopes are 'base', 'one' and 'sub'." introductionVersion:"1.0.0"`
	GroupSubstringFilterType string          `yaml:"group_substring_filter_type" env:"LDAP_GROUP_SUBSTRING_FILTER_TYPE;GROUPS_LDAP_GROUP_SUBSTRING_FILTER_TYPE" desc:"Type of substring search filter to use for substring searches for groups. Supported values are 'initial', 'final' and 'any'. The value 'initial' is used for doing prefix only searches, 'final' for doing suffix only searches or 'any' for doing full substring searches" introductionVersion:"1.0.0"`
	UserFilter               string          `yaml:"user_filter" env:"OC_LDAP_USER_FILTER;GROUPS_LDAP_USER_FILTER" desc:"LDAP filter to add to the default filters for user search like '(objectclass=openCloudUser)'." introductionVersion:"1.0.0"`
	GroupFilter              string          `yaml:"group_filter" env:"OC_LDAP_GROUP_FILTER;GROUPS_LDAP_GROUP_FILTER" desc:"LDAP filter to add to the default filters for group searches." introductionVersion:"1.0.0"`
	UserObjectClass          string          `yaml:"user_object_class" env:"OC_LDAP_USER_OBJECTCLASS;GROUPS_LDAP_USER_OBJECTCLASS" desc:"The object class to use for users in the default user search filter ('inetOrgPerson')." introductionVersion:"1.0.0"`
	GroupObjectClass         string          `yaml:"group_object_class" env:"OC_LDAP_GROUP_OBJECTCLASS;GROUPS_LDAP_GROUP_OBJECTCLASS" desc:"The object class to use for groups in the default group search filter ('groupOfNames')." introductionVersion:"1.0.0"`
	IDP                      string          `yaml:"idp" env:"OC_URL;OC_OIDC_ISSUER;GROUPS_IDP_URL" desc:"The identity provider value to set in the group IDs of the CS3 group objects for groups returned by this group provider." introductionVersion:"1.0.0"`
	UserSchema               LDAPUserSchema  `yaml:"user_schema"`
	GroupSchema              LDAPGroupSchema `yaml:"group_schema"`
}

type LDAPUserSchema struct {
	ID              string `yaml:"id" env:"OC_LDAP_USER_SCHEMA_ID;GROUPS_LDAP_USER_SCHEMA_ID" desc:"LDAP Attribute to use as the unique id for users. This should be a stable globally unique id like a UUID." introductionVersion:"1.0.0"`
	IDIsOctetString bool   `yaml:"id_is_octet_string" env:"OC_LDAP_USER_SCHEMA_ID_IS_OCTETSTRING;GROUPS_LDAP_USER_SCHEMA_ID_IS_OCTETSTRING" desc:"Set this to true if the defined 'ID' attribute for users is of the 'OCTETSTRING' syntax. This is e.g. required when using the 'objectGUID' attribute of Active Directory for the user ID's." introductionVersion:"1.0.0"`
	Mail            string `yaml:"mail" env:"OC_LDAP_USER_SCHEMA_MAIL;GROUPS_LDAP_USER_SCHEMA_MAIL" desc:"LDAP Attribute to use for the email address of users." introductionVersion:"1.0.0"`
	DisplayName     string `yaml:"display_name" env:"OC_LDAP_USER_SCHEMA_DISPLAYNAME;GROUPS_LDAP_USER_SCHEMA_DISPLAYNAME" desc:"LDAP Attribute to use for the displayname of users." introductionVersion:"1.0.0"`
	Username        string `yaml:"user_name" env:"OC_LDAP_USER_SCHEMA_USERNAME;GROUPS_LDAP_USER_SCHEMA_USERNAME" desc:"LDAP Attribute to use for username of users." introductionVersion:"1.0.0"`
}

type LDAPGroupSchema struct {
	ID              string `yaml:"id" env:"OC_LDAP_GROUP_SCHEMA_ID;GROUPS_LDAP_GROUP_SCHEMA_ID" desc:"LDAP Attribute to use as the unique id for groups. This should be a stable globally unique ID like a UUID." introductionVersion:"1.0.0"`
	IDIsOctetString bool   `yaml:"id_is_octet_string" env:"OC_LDAP_GROUP_SCHEMA_ID_IS_OCTETSTRING;GROUPS_LDAP_GROUP_SCHEMA_ID_IS_OCTETSTRING" desc:"Set this to true if the defined 'id' attribute for groups is of the 'OCTETSTRING' syntax. This is e.g. required when using the 'objectGUID' attribute of Active Directory for the group ID's." introductionVersion:"1.0.0"`
	Mail            string `yaml:"mail" env:"OC_LDAP_GROUP_SCHEMA_MAIL;GROUPS_LDAP_GROUP_SCHEMA_MAIL" desc:"LDAP Attribute to use for the email address of groups (can be empty)." introductionVersion:"1.0.0"`
	DisplayName     string `yaml:"display_name" env:"OC_LDAP_GROUP_SCHEMA_DISPLAYNAME;GROUPS_LDAP_GROUP_SCHEMA_DISPLAYNAME" desc:"LDAP Attribute to use for the displayname of groups (often the same as groupname attribute)." introductionVersion:"1.0.0"`
	Groupname       string `yaml:"group_name" env:"OC_LDAP_GROUP_SCHEMA_GROUPNAME;GROUPS_LDAP_GROUP_SCHEMA_GROUPNAME" desc:"LDAP Attribute to use for the name of groups." introductionVersion:"1.0.0"`
	Member          string `yaml:"member" env:"OC_LDAP_GROUP_SCHEMA_MEMBER;GROUPS_LDAP_GROUP_SCHEMA_MEMBER" desc:"LDAP Attribute that is used for group members." introductionVersion:"1.0.0"`
}

type OwnCloudSQLDriver struct {
	DBUsername         string `yaml:"db_username" env:"GROUPS_OWNCLOUDSQL_DB_USERNAME" desc:"Database user to use for authenticating with the owncloud database." introductionVersion:"1.0.0"`
	DBPassword         string `yaml:"db_password" env:"GROUPS_OWNCLOUDSQL_DB_PASSWORD" desc:"Password for the database user." introductionVersion:"1.0.0"`
	DBHost             string `yaml:"db_host" env:"GROUPS_OWNCLOUDSQL_DB_HOST" desc:"Hostname of the database server." introductionVersion:"1.0.0"`
	DBPort             int    `yaml:"db_port" env:"GROUPS_OWNCLOUDSQL_DB_PORT" desc:"Network port to use for the database connection." introductionVersion:"1.0.0"`
	DBName             string `yaml:"db_name" env:"GROUPS_OWNCLOUDSQL_DB_NAME" desc:"Name of the owncloud database." introductionVersion:"1.0.0"`
	IDP                string `yaml:"idp" env:"GROUPS_OWNCLOUDSQL_IDP" desc:"The identity provider value to set in the userids of the CS3 user objects for users returned by this user provider." introductionVersion:"1.0.0"`
	Nobody             int64  `yaml:"nobody" env:"GROUPS_OWNCLOUDSQL_NOBODY" desc:"Fallback number if no numeric UID and GID properties are provided." introductionVersion:"1.0.0"`
	JoinUsername       bool   `yaml:"join_username" env:"GROUPS_OWNCLOUDSQL_JOIN_USERNAME" desc:"Join the user properties table to read usernames." introductionVersion:"1.0.0"`
	JoinOwnCloudUUID   bool   `yaml:"join_owncloud_uuid" env:"GROUPS_OWNCLOUDSQL_JOIN_OWNCLOUD_UUID" desc:"Join the user properties table to read user IDs." introductionVersion:"1.0.0"`
	EnableMedialSearch bool   `yaml:"enable_medial_search" env:"GROUPS_OWNCLOUDSQL_ENABLE_MEDIAL_SEARCH" desc:"Allow 'medial search' when searching for users instead of just doing a prefix search. This allows finding 'Alice' when searching for 'lic'." introductionVersion:"1.0.0"`
}

type JSONDriver struct {
	File string
}

type RESTProvider struct {
	ClientID          string
	ClientSecret      string
	RedisAddr         string
	RedisUsername     string
	RedisPassword     string
	IDProvider        string
	APIBaseURL        string
	OIDCTokenEndpoint string
	TargetAPI         string
}
