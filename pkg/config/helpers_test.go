package config

import (
	"testing"
	"testing/fstest"

	"gotest.tools/v3/assert"
)

type TestConfig struct {
	A string `yaml:"a"`
	B string `yaml:"b"`
	C string `yaml:"c"`
}

func TestBindSourcesToStructs(t *testing.T) {
	// setup test env
	yaml := `
a: "${FOO_VAR|no-foo}"
b: "${BAR_VAR|no-bar}"
c: "${CODE_VAR|code}"
`
	filePath := "etc/opencloud/foo.yaml"
	fs := fstest.MapFS{
		filePath: {Data: []byte(yaml)},
	}
	// perform test
	c := TestConfig{}
	err := bindSourcesToStructs(fs, filePath, "foo", &c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, c.A, "no-foo")
	assert.Equal(t, c.B, "no-bar")
	assert.Equal(t, c.C, "code")
}

func TestBindSourcesToStructs_UnknownFile(t *testing.T) {
	// setup test env
	filePath := "etc/opencloud/foo.yaml"
	fs := fstest.MapFS{}
	// perform test
	c := TestConfig{}
	err := bindSourcesToStructs(fs, filePath, "foo", &c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, c.A, "")
	assert.Equal(t, c.B, "")
	assert.Equal(t, c.C, "")
}

func TestBindSourcesToStructs_NoEnvVar(t *testing.T) {
	// setup test env
	yaml := `
token_manager:
  jwt_secret: f%LovwC6xnKkHhc.!.Lp4ZYpQDIO7=d@
machine_auth_api_key: jG&%ZCmCSYqT#Yi$9y28o5u84ZMo2UBf
system_user_api_key: wqxH7FZHv5gifuLIzxqdyaZOCo2s^yl1
transfer_secret: $1^2xspR1WHussV16knaJ$x@X*XLPL%y
system_user_id: 4d0bf32c-83ee-4703-bd43-5e0d6b78215b
admin_user_id: e2fca2b3-992b-47d5-8ecd-3312418ed3d7
graph:
  application:
    id: 4fdff90c-d13c-47ab-8227-bbd3e6dbee3c
  events:
    tls_insecure: true
  spaces:
    insecure: true
  identity:
    ldap:
      bind_password: $ZZ8fSJR&YA02jBBPx6IRCzW0kVZ#cBO
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
idp:
  ldap:
    bind_password: kWJGC6WRY1wQ+e8Bmt--=-3r6gp0CNVS
idm:
  service_user_passwords:
    admin_password: admin
    idm_password: $ZZ8fSJR&YA02jBBPx6IRCzW0kVZ#cBO
    reva_password: c68JL=V$c@0GHs!%eSb8r&Ps3rgzKnXJ
    idp_password: kWJGC6WRY1wQ+e8Bmt--=-3r6gp0CNVS
proxy:
  oidc:
    insecure: true
  insecure_backends: true
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
frontend:
  app_handler:
    insecure: true
  archiver:
    insecure: true
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
  ocdav:
    insecure: true
auth_basic:
  auth_providers:
    ldap:
      bind_password: c68JL=V$c@0GHs!%eSb8r&Ps3rgzKnXJ
auth_bearer:
  auth_providers:
    oidc:
      insecure: true
users:
  drivers:
    ldap:
      bind_password: c68JL=V$c@0GHs!%eSb8r&Ps3rgzKnXJ
groups:
  drivers:
    ldap:
      bind_password: c68JL=V$c@0GHs!%eSb8r&Ps3rgzKnXJ
ocm:
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
thumbnails:
  thumbnail:
    transfer_secret: 0N05@YXB.h3e@lsVfksL4YxwQC9aE5A.
    webdav_allow_insecure: true
    cs3_allow_insecure: true
search:
  events:
    tls_insecure: true
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
audit:
  events:
    tls_insecure: true
settings:
  service_account_ids:
  - c05389b2-d94c-4d01-a9b5-a2f97952cc14
sharing:
  events:
    tls_insecure: true
storage_users:
  events:
    tls_insecure: true
  mount_id: 64fdfb03-22ff-4788-be4d-d7731a475683
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
notifications:
  notifications:
    events:
      tls_insecure: true
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
nats:
  nats:
    tls_skip_verify_client_cert: true
gateway:
  storage_registry:
    storage_users_mount_id: 64fdfb03-22ff-4788-be4d-d7731a475683
userlog:
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
auth_service:
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
clientlog:
  service_account:
    service_account_id: c05389b2-d94c-4d01-a9b5-a2f97952cc14
    service_account_secret: GW5.x1vDM&+NPRi++eV@.P7Tms4vj!=s
`
	filePath := "etc/opencloud/foo.yaml"
	fs := fstest.MapFS{
		filePath: {Data: []byte(yaml)},
	}
	// perform test
	c := Config{}
	err := bindSourcesToStructs(fs, filePath, "foo", &c)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, c.Graph.Identity.LDAP.BindPassword, "$ZZ8fSJR&YA02jBBPx6IRCzW0kVZ#cBO")
}
