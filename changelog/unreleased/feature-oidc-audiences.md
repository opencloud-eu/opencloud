Enhancement: Optional OIDC access token audience validation

The proxy can now restrict OIDC access tokens to configured audiences using
PROXY_OIDC_AUDIENCES or oidc.audiences in proxy.yaml. For example, setting
PROXY_OIDC_AUDIENCES=opencloud,opencloud-api requires at least one of these values
in the access token's aud claim. Matching is exact and case-sensitive, and both
string and array claims are supported.

The list defaults to empty to preserve existing deployments. When OIDC is active
and audience validation is disabled, the proxy emits one startup warning.
Enabling validation is recommended for production, especially when an identity
provider serves multiple applications. Administrators must configure the selected
audience in their identity provider's access tokens before enabling the check.

Configured audiences require JWT verification. Missing, empty, malformed or
nonmatching token audiences are rejected, including when Userinfo is already
cached. Changing the configuration requires restarting the proxy.

https://github.com/opencloud-eu/opencloud/issues/3456
