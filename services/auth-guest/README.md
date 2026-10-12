# auth-guest

The `auth-guest` service gives guest users access to a share without a full
OpenCloud account. When a share is created for a user of type
`USER_TYPE_GUEST`, the service issues a one-time guest link token; redeeming
that token exchanges it for a signed session cookie that authenticates the
guest.

When the session or the link token is no longer valid, the guest can *renew*:
the service issues a new link token together with a one-time PIN. The guest
then either redeems the new link or exchanges the PIN for a fresh session.

The unauthenticated endpoints are protected against brute-force and abuse: the
PIN is bound to a browser-held nonce, failed PIN attempts back off exponentially
and invalidate the PIN after a maximum number of attempts, and `renew` is
limited per share (cooldown and daily cap). The magic link is never affected by
a PIN lockout.

It is disabled by default. Set `OC_ENABLE_GUEST_LINKS=true` to enable the guest
links feature and start the service.

## Overview

- **Consumes** the share lifecycle events `ShareCreated`, `ShareRemoved` and
  `ShareExpired`.
- **Publishes** the `GuestTokenCreated` event (initial token) and the
  `GuestTokenRenewed` event (refresh, carrying the new token and the PIN). The
  `notifications` service consumes both and mails the link (and the PIN) to
  the guest.
- Exposes unauthenticated endpoints to redeem a token, renew it and verify a
  PIN; redeeming a token or verifying a PIN sets a session cookie. `renew` and
  `verify/pin` are rate-limited.
- Stores only hashes of the token secret, of the PIN and of the nonce, and
  deletes the stored record when the share is removed or expires.

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
    Redeem->>-Web: Set Cookie, return shareid and session expiry
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
Verifying a token or a PIN returns the session expiry as `expirationDateTime`,
so the client can warn the guest before the session runs out, since it cannot
read the HttpOnly cookie. Renewal starts from a *failed* attempt:

- an authenticated request made with an expired session is rejected with `401`;
  the browser still holds the (expired) session cookie;
- `redeem` on a link that was already used answers `409`, and on an expired
  link `401`; the client still holds the link token.

`renew` requires the share id (`permissionId`) and is authorized by either the
(possibly expired) session cookie or the previous link token, whichever the
client has; the credential must belong to the given share. It does not create a
session. It requires an existing token record for the share, validates the
share, generates a new link token and a 6-digit PIN (both valid for 30 minutes),
overwrites the stored token, PIN and nonce hashes (invalidating the previous
link and PIN), resets the redeemed flag so the new link can be redeemed, and
publishes `GuestTokenRenewed`. It answers `200` and sets the nonce cookie that
binds the new PIN to the requesting browser. This works whether the record is
unused, expired or already redeemed (the guest already redeemed an earlier
link); if publishing fails, the previous record is restored.

`renew` is limited per share: a cooldown between two renewals
(`AUTH_GUEST_QUOTA_COOLDOWN`, default 1 minute) and a daily cap
(`AUTH_GUEST_QUOTA_DAILY_CAP`, default 20 per `AUTH_GUEST_QUOTA_WINDOW`, default
24 hours). A request within the cooldown or over the cap is rejected with `429`
and a `Retry-After` header.

To turn the new credentials into a session the guest either redeems the link
(`verify/token`) or enters the PIN (`verify/pin`); both return a session cookie and
the `permissionId`. `verify/pin` does not consume the link token, but it requires
the nonce cookie set by `renew`, so the PIN can only be verified from the browser
that requested the renewal. Each wrong PIN increments a counter and the wait
between attempts grows exponentially; after
`AUTH_GUEST_BACKOFF_MAX_ATTEMPTS` (default 5) failures the PIN is invalidated.
While the backoff is active `verify/pin` answers `429` with a `Retry-After`
header. The magic link (`verify/token`) is not affected by any of this: it
remains usable even when the PIN is locked out.

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
    Web->>+Renew: POST renew { permissionId } (+ session cookie or link token)
    Note right of Renew: Validate share, generate token + PIN + nonce, store hashes
    Renew->>Bus: publish GuestTokenRenewed
    Note right of Bus: notifications service mails the new link + PIN to the guest
    Renew->>-Web: 200 OK

    alt Guest opens the new link
        User->>Web: Open link
        Web->>+Redeem: POST redeem { token }
        Note right of Redeem: Validate token + share, mark used
        Redeem->>-Web: Set Cookie, return permissionId
    else Guest enters the PIN
        User->>Web: Enter PIN
        Web->>+Pin: POST verify/pin { pin, permissionId } (+ nonce cookie)
        Note right of Pin: Validate nonce + share, consume PIN (single-use)
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
   `POST /graph/v1beta1/extensions/org.libregraph/guestLinks/verify/token`.
   The service validates the token and the share, marks the token as used and
   returns a signed JWT session token in a cookie plus the share's
   `permissionId` in the response body. Tokens are single-use and valid for
   30 minutes; the session cookie lives for 24 hours by default
   (`AUTH_GUEST_JWT_TTL`).
3. **Renew** — the guest posts the `permissionId` to
   `POST /graph/v1beta1/extensions/org.libregraph/guestLinks/renew`, authorized
   by either the previous link token (`{ "token": "<token>" }`) or the (possibly
   expired) session cookie, which must both belong to that share. An existing
   record is required; otherwise the request fails with `tokenNotFound`. The
   service validates the share, generates a new link token and a 6-digit PIN
   (both valid for 30 minutes), overwrites the stored token, PIN and nonce
   hashes (invalidating the previous link and PIN), resets the redeemed flag so
   the new link can be redeemed, publishes the `GuestTokenRenewed` event and
   sets the nonce cookie. Renewals are limited per share (cooldown plus daily
   cap); an exceeded limit fails with `429`. This works whether the record is
   unused, expired or already redeemed. If publishing fails, the previous record
   is restored.
4. **Verify PIN** — the guest posts the PIN, the `permissionId` and the nonce
   cookie to
   `POST /graph/v1beta1/extensions/org.libregraph/guestLinks/verify/pin`.
   The service validates the nonce, the record and the share, compares the PIN
   against the stored argon2id hash and, on success, clears the PIN (single-use)
   and returns a signed JWT session token in a cookie plus the share's
   `permissionId`. Each wrong PIN increments a counter and increases the wait
   before the next attempt; after the maximum number of failed attempts the PIN
   is invalidated and while the backoff is active the endpoint answers `429`.
   Verifying the PIN does not consume the magic link.

### Endpoints

All error responses have the body
`{ "errorType": "<type>", "message": "<msg>", "permissionId": "<share-id>" }`.
Rate-limited responses (`429`) also carry a `retryAfter` field (seconds) and a
`Retry-After` header.

#### `POST .../guestLinks/verify/token`

- **Request body:** `{ "token": "<token>" }`
- **Success:** `200`, session cookie,
  `{ "permissionId": "<share-id>" }`
- **Errors:** `400 invalidRequest`, `401 tokenInvalid`,
  `401 tokenExpired`, `404 tokenNotFound`, `409 tokenAlreadyRedeemed`,
  `404 shareNotFound`, `410 shareExpired`, `500 internalError`

#### `POST .../guestLinks/renew`

- **Request body:**
  `{ "permissionId": "<share-id>", "token": "<previous link token>" }`
  (`token` optional when the session cookie is sent)
- **Authorization:** the share id is required; the body link token or the
  session cookie must belong to it, the link token takes precedence
- **Success:** `200` (no session cookie; the new link/PIN are delivered out
  of band, the nonce cookie is set)
- **Errors:** `400 invalidRequest`, `401 tokenInvalid`, `401 sessionInvalid`,
  `404 tokenNotFound`, `404 shareNotFound`, `410 shareExpired`,
  `429 tooManyRequests` (with `Retry-After`), `503 serviceUnavailable`,
  `500 internalError`

#### `POST .../guestLinks/verify/pin`

- **Request body:** `{ "pin": "<pin>", "permissionId": "<share-id>" }`
- **Required cookie:** the nonce cookie set by `renew`
- **Success:** `200`, session cookie, clears the nonce cookie,
  `{ "permissionId": "<share-id>" }`
- **Errors:** `400 invalidRequest` (missing nonce cookie), `401 nonceInvalid`,
  `401 nonceExpired`, `401 pinInvalid`, `401 pinExpired`, `404 shareNotFound`,
  `410 shareExpired`, `429 tooManyRequests` (with `Retry-After`),
  `500 internalError`

## Frontend integration

The three endpoints are unauthenticated and rely on cookies, so the client must
send credentials with every request (for example `credentials: 'include'` in
`fetch`). Cross-origin deployments also need the service CORS to allow the
origin and credentials; the default `AUTH_GUEST_CORS_ALLOW_ORIGINS` of `*` is not
sufficient together with credentials, so set a concrete origin.

The PIN is bound to the browser that renewed, so the order matters:

1. `renew` — sets the nonce cookie on the response.
2. `verify/pin` — the browser sends the nonce cookie back automatically; a
   successful response clears it and sets the session cookie.

The nonce cookie is `HttpOnly`, `Secure`, `Path=/`, `SameSite=Lax` and uses the
`__Host-` prefix, so it is only sent over HTTPS (browsers treat `localhost` as
secure) and cannot be read from JavaScript.

Client-side handling of the relevant responses:

- `400 invalidRequest` "The nonce cookie is missing from the request." — the
  browser has no nonce (it never renewed, or the cookie expired): call `renew`
  again before `verify/pin`.
- `401 nonceInvalid` / `401 nonceExpired` — the nonce does not match or has
  expired: call `renew` again to get a fresh PIN and nonce.
- `401 pinInvalid` — wrong PIN; the wait between attempts grows, so retry with a
  delay.
- `401 pinExpired` — the PIN expired; call `renew` again.
- `429 tooManyRequests` — too many failed attempts, or the renewal cooldown or
  cap is active; wait for the `Retry-After` header (or the `retryAfter` field)
  in seconds, then retry.

## Configuration

The service is configured via `AUTH_GUEST_*` environment variables or a
`auth-guest.yaml` file.

Both transports are enabled by default: the service consumes and publishes
events and serves the HTTP endpoints. Either transport can be disabled:

- `AUTH_GUEST_EVENTS_DISABLED=true` — the service does not consume events
  (no tokens are created or cleaned up) and `renew` cannot publish its event,
  so it responds with `503 serviceUnavailable`.
- `AUTH_GUEST_HTTP_DISABLED=true` — the service does not serve the HTTP
  endpoints (`verify/token`, `renew`, `verify/pin`).

The link token and PIN lifetimes are fixed (30 minutes each) and are not
configurable. The rate-limiting and nonce options are:

- `AUTH_GUEST_NONCE_COOKIE_NAME` (default `__Host-oc_guest_nonce`),
  `AUTH_GUEST_NONCE_TTL` (default `15m`) — name and lifetime of the nonce cookie
  set by `renew` and required by `verify/pin`.
- `AUTH_GUEST_BACKOFF_MAX_ATTEMPTS` (default `5`),
  `AUTH_GUEST_BACKOFF_BASE` (default `1s`),
  `AUTH_GUEST_BACKOFF_CAP` (default `30s`) — failed PIN attempts before the PIN
  is invalidated, and the exponential backoff between attempts.
- `AUTH_GUEST_QUOTA_DAILY_CAP` (default `20`),
  `AUTH_GUEST_QUOTA_COOLDOWN` (default `1m`),
  `AUTH_GUEST_QUOTA_WINDOW` (default `24h`) — maximum renewals per share per
  window and the minimum time between two renewals.

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
