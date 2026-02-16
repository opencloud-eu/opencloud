---
title: "Discover OIDC Client configuration via WebFinger"
---

* Status: pending
* Deciders: [@TheOneRing @kulmann @rhafer @dragotin]
* Date: 2026-02-02

Reference: https://github.com/opencloud-eu/opencloud/pull/2072, https://github.com/opencloud-eu/desktop/issues/217

## Context and Problem Statement

Up to now our client applications used hard-coded OIDC client configurations.
So it is not possible to change the client id that a client should use or the
list of scopes that a client needs to request. This makes it hard to integrate
OpenCloud with various existing identity providers. For example:

- Authentik basically creates a different issuer URL for each client. As OpenCloud
  can only work with a single issuer URL, all OpenCloud clients need to use the
  same client id to work with Authentik.
- Some IDPs (kanidm) are not able to work with user-supplied client ids. They generate
  client ids automatically and do not allow to specify them manually.
- To make features like automatic role assignment work, clients need to request
  specific scopes, depending on which exact IDP is used.

## Decision Drivers

* Support broader set of IDPs
* avoid any manual configuration adjustments on the client side

## Decision

Enhance the WebFinger service in OpenCloud to provide platform-specific OIDC
discovery, enabling clients to query for the correct OIDC `client_id` and
`scopes` based on their application type (e.g., web, desktop, android, ios).

This is achieved by allowing an additional `platform` query parameter to be used
when querying the WebFinger endpoint. The response will include the appropriate
`client_id` and `scopes` in the `properties` section of the response.

This is implemented in a backward-compatible way, so existing clients that do not
specify the `platform` parameter will continue to receive just the issuer information.

## Example

### Client Request

```
GET /.well-known/webfinger?resource=https://cloud.opencloud.test&rel=http://openid.net/specs/connect/1.0/issuer&platform=desktop
```

### Server Response

```json
{
  "subject": "https://cloud.opencloud.test",
  "links": [{
    "rel": "http://openid.net/specs/connect/1.0/issuer",
    "href": "https://idp.example.com"
  }],
  "properties": {
    "http://opencloud.eu/ns/oidc/client_id": "desktop-client-id",
    "http://opencloud.eu/ns/oidc/scopes": ["openid", "profile", "email", "offline_access"]
  }
}
```

### Server configuration (suggestion)

To configure the OpenCloud server a couple of new config settings need to be introduced. This would
be two new settings per client, e.g.:


```
WEBFINGER_ANDROID_OIDC_CLIENT_ID
WEBFINGER_ANDROID_OIDC_CLIENT_SCOPE
WEBFINGER_DESKTOP_OIDC_CLIENT_ID
WEBFINGER_DESKTOP_OIDC_CLIENT_SCOPE
WEBFINGER_IOS_OIDC_CLIENT_ID
WEBFINGER_IOS_OIDC_CLIENT_SCOPE
WEBFINGER_WEB_OIDC_CLIENT_ID
WEBFINGER_WEB_OIDC_CLIENT_SCOPE
```

Additionally for backwards compatibility the existing `WEB_OIDC_CLIENT_ID` and
`WEB_OIDC_CLIENT_SCOPE` settings should be used as fallback for the `web`
platform. Also we should make it easy to configure the same settings for all
platforms at once by using `OC_OIDC_CLIENT_ID` and `OC_OIDC_CLIENT_SCOPE` as
fallback for all platforms if the platform-specific settings are not set.
