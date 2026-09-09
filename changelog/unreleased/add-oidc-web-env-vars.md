Enhancement: Add OIDC env vars to expand external IDP compatibility

Added two new environment variables, OC_OIDC_CLIENT_SECRET  / 
WEB_OIDC_CLIENT_SECRET and WEB_OIDC_CLIENT_AUTHENTICATION to 
control additional functionality of the frontend's OAuth code exchanges.
Both are passed to the frontend to control oidc-client-ts.

https://github.com/opencloud-eu/opencloud/issues/2345
