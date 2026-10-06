# auth-guest

The `auth-guest` service gives guest users access to a share without a full
OpenCloud account. When a share is created for a user of type
`USER_TYPE_GUEST`, the service issues a one-time guest link token; redeeming
that token exchanges it for a signed session cookie that authenticates the
guest.

When the session or the link token is no longer valid, the guest can *renew*:
the service issues a new link token together with a one-time PIN. The guest
then either redeems the new link or exchanges the PIN for a fresh session.

It is disabled by default. Set `OC_ENABLE_GUEST_LINKS=true` to enable the guest
links feature and start the service.

## Overview

- **Consumes** the share lifecycle events `ShareCreated`, `ShareRemoved` and
  `ShareExpired`.
- **Publishes** the `GuestTokenCreated` event (initial token) and the
  `GuestTokenRenewed` event (refresh, carrying the new token and the PIN).
- Exposes unauthenticated endpoints to redeem a token, renew it and verify a
  PIN; redeeming a token or verifying a PIN sets a session cookie.
- Stores only hashes of the token secret and of the PIN and deletes the stored
  record when the share is removed or expires.

## Guest links flow

The following sequence diagram describes the guest links flow:

```mermaid
sequenceDiagram
    autonumber
    actor User as Guest user
    participant Web as Web client
    participant Redeem as Redeem endpoint
    participant Proxy as OpenCloud proxy
    participant Graph as Graph / sharedWithMe
    participant DAV as WebDAV
    participant Reva as Reva

    User->>Web: Open guest link with valid token
    Web->>+Redeem: Redeem Token
    Note right of Redeem: Validate Token
    Redeem->>+Reva: Get Share
    Reva->>-Redeem: Share
    Note right of Redeem: Validate Share, Mark Token used
    Redeem->>-Web: Set Cookie, return shareid
    Note right of Web: HTTP only cookie with signed JWT (24h by default)
    Web->>+Proxy: "/graph/me/drives/sharedWithMe"
    Proxy->>+Reva: validate token extracted from JWT
    Note right of Reva: Sign Reva Token for Guest User
    Reva->>-Proxy: Authenticated
    Proxy->>+Graph: "/graph/me/drives/sharedWithMe"
    Note right of Proxy: Using Reva Token
    Graph->>+Reva: Requests to ShareProvider
    Reva->>-Graph: Shares
    Graph->>-Proxy: driveItems (all shares for the Guest User)
    Proxy->>-Web: driveItems
    Note right of Web: Extracts driveItem for the specific share
    Web->>+Proxy: PROPFIND (resource id extracted from driveItem)
    Note right of Web: Using Cookie
    Proxy->>+Reva: validate token extracted from JWT
    Note right of Reva: Sign Reva Token for Guest User
    Reva->>-Proxy: Authenticated
    Proxy->>+DAV: PROPFIND
    Note right of Proxy: Using Reva Token
    DAV->>+Reva: Requests to StorageProvider
    Reva->>-DAV: StorageProvider Responses
    DAV->>-Proxy: PROPFIND Response
    Proxy->>-Web: PROPFIND Response
```

## Renewal and PIN flow

A guest session (signed JWT) is valid for 24 hours by default
(`AUTH_GUEST_JWT_TTL`); a link token is single-use and valid for 30 minutes.
Renewal starts from a *failed* attempt`:

- an authenticated request made with an expired session is rejected with `401`;
- `redeem` on a link that was already used answers `409`, and on an expired
  link `401`.

Both errors carry the share id, which the client sends to `renew` as
`permissionId`.

`renew` itself does not create a
session. It requires an existing token record for the share
, validates the share, generates a new link token and a 6-digit PIN (both valid
for 30 minutes), overwrites the stored token and PIN hashes (invalidating the
previous link and PIN), resets the redeemed flag so the new link can be
redeemed, and publishes `GuestTokenRenewed`. It answers `200`. This works
whether the record is unused, expired or already redeemed (the guest already
redeemed an earlier link); if publishing fails, the previous record is
restored.

To turn the new credentials into a session the guest either redeems the link
(`redeem`) or enters the PIN (`verify/pin`); both return a session cookie and
the `permissionId`. `verify/pin` does not consume the link token.

```mermaid
sequenceDiagram
    autonumber
    actor User as Guest user
    participant Web as Web client
    participant Renew as Renew endpoint
    participant Redeem as Redeem endpoint
    participant Pin as Verify PIN endpoint
    participant Bus as Event bus

    Note over Web: Session expired / previous link used or expired
    Web->>+Renew: POST renew { permissionId }
    Note right of Renew: Validate share, generate token + PIN, store hashes
    Renew->>Bus: publish GuestTokenRenewed
    Note right of Bus: External consumer delivers token + PIN (out of scope)
    Renew->>-Web: 200 OK

    alt Guest opens the new link
        User->>Web: Open link
        Web->>+Redeem: POST redeem { token }
        Note right of Redeem: Validate token + share, mark used
        Redeem->>-Web: Set Cookie, return permissionId
    else Guest enters the PIN
        User->>Web: Enter PIN
        Web->>+Pin: POST verify/pin { pin, permissionId }
        Note right of Pin: Validate share, consume PIN (single-use)
        Pin->>-Web: Set Cookie, return permissionId
    end
    Note over Web: HTTP only cookie, signed JWT (24h default)
```

## Token lifecycle

1. **Issue** — on the consumed `ShareCreated` event, where the grantee is a
   guest, the service generates a random secret and stores a record keyed by
   the hash of the share id. It then publishes the `GuestTokenCreated` event
   with the token.
2. **Redeem** — the guest posts the token to
   `POST /graph/v1beta1/extensions/org.libregraph/guestLinks/redeem` (alias
   `.../guestLinks/verify/token`).
   The service validates the token and the share, marks the token as used and
   returns a signed JWT session token in a cookie plus the share's
   `permissionId` in the response body. Tokens are single-use and valid for
   30 minutes; the session cookie lives for 24 hours by default
   (`AUTH_GUEST_JWT_TTL`).
3. **Renew** — the guest posts the `permissionId` to
   `POST /graph/v1beta1/extensions/org.libregraph/guestLinks/renew`.
   An existing record is required; otherwise the request fails with
   `tokenNotFound`. The service validates the share, generates a new link token
   and a 6-digit PIN (both valid for 30 minutes), overwrites the stored token
   and PIN hashes (invalidating the previous link and PIN) and resets the
   redeemed flag so the new link can be redeemed, then publishes the
   `GuestTokenRenewed` event. This works whether the record is unused, expired
   or already redeemed. If publishing fails, the previous record is restored.
4. **Verify PIN** — the guest posts the PIN and the `permissionId` to
   `POST /graph/v1beta1/extensions/org.libregraph/guestLinks/verify/pin`.
   The service validates the record and the share, compares the PIN against the
   stored argon2id hash and, on success, clears the PIN (single-use) and
   returns a signed JWT session token in a cookie plus the share's
   `permissionId`. Verifying the PIN does not consume the magic link.

### Endpoints

All error responses have the body
`{ "errorType": "<type>", "message": "<msg>", "permissionId": "<share-id>" }`.

#### `POST .../guestLinks/redeem`

Alias: `.../guestLinks/verify/token`

- **Request body:** `{ "token": "<token>" }`
- **Success:** `200`, session cookie,
  `{ "permissionId": "<share-id>" }`
- **Errors:** `400 invalidRequest`, `401 tokenInvalid`,
  `401 tokenExpired`, `404 tokenNotFound`, `409 tokenAlreadyRedeemed`,
  `404 shareNotFound`, `410 shareExpired`, `500 internalError`

#### `POST .../guestLinks/renew`

- **Request body:** `{ "permissionId": "<share-id>" }`
- **Success:** `200` (no session cookie; the new link/PIN are delivered out
  of band)
- **Errors:** `400 invalidRequest`, `404 tokenNotFound`,
  `404 shareNotFound`, `410 shareExpired`, `503 serviceUnavailable`,
  `500 internalError`

#### `POST .../guestLinks/verify/pin`

- **Request body:** `{ "pin": "<pin>", "permissionId": "<share-id>" }`
- **Success:** `200`, session cookie,
  `{ "permissionId": "<share-id>" }`
- **Errors:** `400 invalidRequest`, `401 pinInvalid`, `401 pinExpired`,
  `404 shareNotFound`, `410 shareExpired`, `500 internalError`

## Configuration

The service is configured via `AUTH_GUEST_*` environment variables or a
`auth-guest.yaml` file.

Both transports are enabled by default: the service consumes and publishes
events and serves the HTTP endpoints. Either transport can be disabled:

- `AUTH_GUEST_EVENTS_DISABLED=true` — the service does not consume events
  (no tokens are created or cleaned up) and `renew` cannot publish its event,
  so it responds with `503 serviceUnavailable`.
- `AUTH_GUEST_HTTP_DISABLED=true` — the service does not serve the HTTP
  endpoints (`redeem`, `renew`, `verify/pin`).

The token and PIN lifetimes are fixed (30 minutes each) and are not
configurable.

Relevant options:

- `AUTH_GUEST_JWT_COOKIE_NAME`, `AUTH_GUEST_JWT_TTL` — session cookie name and
  lifetime.
- Guest session tokens are signed with a key derived from `OC_JWT_SECRET`. There
  is no separate secret to configure; rotating `OC_JWT_SECRET` invalidates all
  guest sessions.
- `AUTH_GUEST_TOKENS_STORAGE_ROOT` — where guest link token records are stored.
- `AUTH_GUEST_SERVICE_ACCOUNT_ID`, `AUTH_GUEST_SERVICE_ACCOUNT_SECRET` — service
  account used to query the gateway for share metadata.
- `AUTH_GUEST_NUM_CONSUMERS` — number of concurrent event consumers.
- `OC_REVA_GATEWAY` — CS3 gateway used to look up shares.
