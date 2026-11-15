# IDP

This service provides a builtin minimal OpenID Connect provider based on [LibreGraph Connect (lico)](https://github.com/libregraph/lico) for OpenCloud.

It is mainly targeted at smaller installations. For larger setups it is recommended to replace IDP with an external OpenID Connect Provider.

By default, it is configured to use the OpenCloud IDM service as its LDAP backend for looking up and authenticating users. Other backends like an external LDAP server can be configured via a set of [enviroment variables](https://docs.opencloud.eu/docs/dev/server/services/idp/environment-variables).

Note that translations provided by the IDP service are not maintained via OpenCloud but part of the embedded  [LibreGraph Connect Identifier](https://github.com/libregraph/lico/tree/master/identifier) package.

## Configuration

### Custom Clients

By default the `idp` service generates a OIDC client configuration suitable for
using OpenCloud with the standard client applications (Web, Desktop, iOS and
Android). If you need to configure additional client it is possible to inject a
custom configuration via `yaml`. This can be done by adding a section `clients`
to the `idp` section of the main configuration file (`opencloud.yaml`). This section
needs to contain configuration for all clients (including the standard clients).

For example if you want to add a (public) client for use with the oidc-agent you would
need to add this snippet to the `idp` section in `opencloud.yaml`.

```yaml
clients:
- id: web
  name: OpenCloud Web App
  trusted: true
  secret: ""
  redirect_uris:
  - https://opencloud.k8s:9200/
  - https://opencloud.k8s:9200/oidc-callback.html
  - https://opencloud.k8s:9200/oidc-silent-redirect.html
  post_logout_redirect_uris: []
  origins:
  - https://opencloud.k8s:9200
  application_type: ""
- id: OpenCloudDesktop
  name: OpenCloud Desktop Client
  trusted: false
  secret: ""
  redirect_uris:
  - http://127.0.0.1
  - http://localhost
  post_logout_redirect_uris: []
  origins: []
  application_type: native
- id: OpenCloudAndroid
  name: OpenCloud Android App
  trusted: false
  secret: ""
  redirect_uris:
  - oc://android.opencloud.eu
  post_logout_redirect_uris:
  - oc://android.opencloud.eu
  origins: []
  application_type: native
- id: OpenCloudIOS
  name: OpenCloud iOS App
  trusted: false
  secret: ""
  redirect_uris:
  - oc://ios.opencloud.eu
  post_logout_redirect_uris:
  - oc://ios.opencloud.eu
  origins: []
  application_type: native
- id: oidc-agent
  name: OIDC Agent
  trusted: false
  secret: ""
  redirect_uris:
  - http://127.0.0.1
  - http://localhost
  post_logout_redirect_uris: []
  origins: []
  application_type: native
```




### Two-Factor Authentication

Users can protect their account with a second factor: a time-based one-time password (TOTP, RFC 6238) from an authenticator app like Google Authenticator, Microsoft Authenticator or FreeOTP.

The feature is built in and available by default, nothing needs to be configured. It is optional per user: only users who turn it on in their account settings are asked for a code. It is offered with the identity managers `ldap` (the default) and `cs3` and when `IDP_ENCRYPTION_SECRET_FILE` is set, which it is by default. To switch it off for the whole instance set:

```
IDP_MFA_ENABLED=false
```

#### How it Works for Users

Users find "Two-factor authentication" in their account settings in the web UI. It opens the page `/signin/v1/mfa` of the login service, for example `https://cloud.example.com/signin/v1/mfa`:

1. The user confirms the password and scans a QR code with the authenticator app.
2. The user enters a code from the app to confirm the setup.
3. The user gets ten recovery codes. Each one can be used once instead of a code from the app, for example when the phone is lost.

From then on the login page asks for a code after the password. This applies to all clients as web, desktop and mobile apps use the same login page. Sessions that exist when two-factor authentication is turned on stay valid.

To turn two-factor authentication off or to get new recovery codes, a code from the app or a recovery code is required.

#### Lost Authenticator App

When users lose their authenticator app and their recovery codes, an administrator can turn two-factor authentication off for them. They can log in with their password again and set it up anew.

```bash
# list the users with two-factor authentication
opencloud idp mfa list

# turn it off for a user, identified by username or user id
opencloud idp mfa disable alan
```

#### Things to Know

* The secrets are kept in the store configured with `IDP_MFA_STORE` (default: `nats-js-kv`). They are encrypted with a key that is derived from the content of `IDP_ENCRYPTION_SECRET_FILE`. When this file is lost or replaced, the secrets cannot be read anymore. Logins of users with two-factor authentication fail until an administrator turns it off for them with the command above. In deployments with more than one `idp` instance all instances need the same file.
* Recovery codes are only stored as hashes.
* The `idp` service needs to reach the store. When the store cannot be reached, nobody can log in, because it is not known who needs a second factor. In deployments where the `idp` service runs separately, set `IDP_MFA_STORE_NODES` (or `OC_PERSISTENT_STORE_NODES`) or switch the feature off.
* After five wrong codes in a row no codes are accepted for five minutes. A one-time code is accepted once.
* Setting `IDP_MFA_ENABLED=false` turns the check off for all users. Their secrets stay in the store and apply again when it is enabled again.
* The second factor is checked by the login page of the builtin IDP only. Other ways to authenticate are not affected: app tokens of the `auth-app` service and basic auth (`PROXY_ENABLE_BASIC_AUTH`, disabled by default) work with the password or token only. With an external IDP like Keycloak, configure two-factor authentication there.
