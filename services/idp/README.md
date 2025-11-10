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



