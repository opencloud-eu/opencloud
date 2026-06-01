---
title: "Embed Authelia as the default Identity Provider"
---

* Status: proposed
* Deciders: [@micbar @rhafer @butonic @fschade]
* Date: 2026-05-31

Reference: https://github.com/authelia/authelia/issues/5803, https://github.com/authelia/authelia/pull/8841

## Context and Problem Statement

OpenCloud ships a fully self-contained, out-of-the-box authentication stack by
embedding upstream Go libraries directly into its service binary:

- The **idm** service embeds `github.com/libregraph/idm` and runs an in-process
  LDAP server (`server.NewServer` in `services/idm/pkg/command/server.go`).
- The **idp** service embeds `github.com/libregraph/lico` ("LibreConnect") as the
  OIDC provider. It boots lico via `bootstrap.Boot()`, extracts an
  `http.Handler` from the returned managers, and mounts it into OpenCloud's own
  chi router under a go-micro HTTP service
  (`services/idp/pkg/service/v0/service.go:118-127`).

Both dependencies are used **unforked, without `replace` directives**. This gives
operators a working IdP with zero external dependencies — a key part of the
OpenCloud value proposition.

lico is functional but minimal. It provides OIDC and a basic login form, but no
second-factor authentication (TOTP / WebAuthn), no access-control policy engine,
and no self-service SSO portal. The OpenCloud team has long been interested in
**Authelia** as a richer embedded default IdP — it provides OIDC 1.0, 2FA
(TOTP/WebAuthn/Duo), per-resource access control, a regulation/brute-force layer,
and a complete login portal.

The blocker, raised by OpenCloud in [authelia/authelia#5803][issue] (2023), was
that all of Authelia's logic lived in `internal/` and was therefore not
importable from another Go module. [authelia/authelia#8841][pr] (merged
2025-03-08) resolved this by adding a public, curated
`github.com/authelia/authelia/v4/experimental/embed` package. As of `v4.39.20`
(2026-05) this package is still present and actively maintained.

This ADR evaluates whether and how to embed Authelia, and what role it should
play relative to the existing idp (lico) service.

[issue]: https://github.com/authelia/authelia/issues/5803
[pr]: https://github.com/authelia/authelia/pull/8841

## Decision Drivers

* **Out-of-the-box experience** — keep the "single binary, no external auth
  dependency" property that lico provides today.
* **Feature parity and beyond** — gain 2FA, access policies and an SSO portal
  without operators bolting on a separate product.
* **Integration cost** — reuse of OpenCloud's runtime supervisor, config system,
  routing and middleware; minimal custom glue.
* **Upstream stability** — the embed API is explicitly experimental; we must be
  able to absorb breakage on Authelia version bumps.
* **Dependency weight** — added impact on the monolith binary's module graph,
  build time and binary size.

## Considered Options

### Option A — Supervised standalone service (run the whole Authelia daemon)

This is the integration model the `experimental/embed` package is designed for.
The public surface is intentionally small and config-file driven:

```go
ctx, val, err := embed.New(paths []string, filterNames []string) // load + validate config
err = embed.ProvidersStartupCheck(ctx, log)
err = embed.ServiceRunAll(ctx)                                   // run all Authelia services
```

`embed.ServiceRunAll` → `service.RunAll` provisions the full set of Authelia
services: the main `fasthttp` server **with its own listener**, the metrics
server, the users file watcher, and the logging signal handler
(`internal/service/provider.go` `GetProvisioners()`).

We would add an OpenCloud service (e.g. `services/auth-authelia` or a new mode of
the existing idp service) whose `Server()` command:

1. Renders an Authelia YAML config from OpenCloud's config system to a file **on
   every start** (with secrets persisted separately; see "Config generation"
   below) — mirroring how the idp (lico) service generates its clients
   registration config in `NewService` (`createTemporaryClientsConfig`). It does
   not depend on `opencloud init`.
2. Calls `embed.New(paths, filters)` and surfaces validation errors/warnings.
3. Wraps `embed.ServiceRunAll(ctx)` in a `runner.Runner` registered with the
   suture supervisor in `opencloud/pkg/runtime/service/service.go`, exactly like
   other services.

Authelia owns its own port, router, lifecycle and config hot-reload.

#### Pros:

* Matches the embed package's intended usage → least friction with upstream, most
  likely to keep working across version bumps.
* Reuses Authelia's own server, TLS, metrics and hot-reload — little glue code.
* Full feature set (portal, 2FA, policies) available immediately.

#### Cons:

* Authelia binds its **own** listener; it is not mounted into OpenCloud's chi
  router. OpenCloud cannot inject its logging/tracing/static-asset middleware the
  way it does for lico. Tracing and access-log correlation need separate wiring.
* Config is file/env driven, not struct driven (see "Config" note below) — we
  must generate and template Authelia config.
* Two HTTP servers in the binary (Authelia's fasthttp + go-micro), each with its
  own port/proxy entry.

### Option B — Handler-mounted, lico-style

Mirror the current idp integration: obtain an `http.Handler` from Authelia and
mount it into OpenCloud's existing chi router / go-micro HTTP service, reusing
OpenCloud middleware.

The `experimental/embed/provider` subpackage exposes individual constructors
(`provider.New(config, caCertPool)` for the full `middlewares.Providers`,
`provider.NewOpenIDConnect`, `NewSession`, `NewAuthorizer`, …). In principle one
could assemble providers and build the server handler manually.

#### Pros:

* Identical integration shape to lico → uniform routing, middleware, tracing,
  single HTTP server.

#### Cons:

* The embed package does **not** export the assembled `*fasthttp.RequestHandler`
  / router. `server.New(...)` lives in `internal/` and returns a `*fasthttp.Server`,
  not a `net/http.Handler` — Authelia is fasthttp-based, lico is `net/http`-based,
  so there is no drop-in handler to mount into chi.
* We would be re-implementing Authelia's `internal/server` wiring against an API
  the maintainers explicitly reserve the right to break ("methods may panic if
  not properly utilized", `doc.go`). High maintenance risk, fights upstream
  intent.
* fasthttp ↔ net/http adaptation adds overhead and edge cases (hijacking,
  streaming, header semantics).

### Option C — Keep lico, do nothing

Stay on the current lico-based idp.

#### Pros:

* Zero work and zero new dependencies.

#### Cons:

* No 2FA, no policy engine, no portal — the gap that motivated this ADR remains.

## Decision Outcome

**Option A (supervised standalone service). Authelia is now the default IdP in
supervised (`opencloud server`) mode; the lico `idp` service is kept as an opt-in
alternative.**

Rationale:

* Option A is the only path that aligns with how `experimental/embed` is
  designed and tested upstream, which is the single biggest factor in keeping a
  declared-experimental dependency maintainable.
* Option B fights the fasthttp/`internal` boundary and would couple us to
  unexported wiring the maintainers refuse to stabilize.

The integration first shipped opt-in (lico default, Authelia behind a flag) to
de-risk it. It is now promoted to the **default**, with lico demoted to an opt-in
fallback so an exit path is preserved. See "Making Authelia the default" below for
the concrete wiring.

Because Authelia maps to the **idp / OIDC-provider** role — not the LDAP **idm**
role — it supplements/replaces lico. The embedded libregraph **idm** LDAP server
stays as Authelia's `authentication_backend` (`provider.NewAuthenticationLDAP`),
so the existing user directory is reused unchanged.

### Making Authelia the default

Three changes flip the default from lico to Authelia (all in supervised
`opencloud server` mode):

* **Service run-set** (`opencloud/pkg/runtime/service/service.go`): `auth-authelia`
  moves to the core `reg(3, ...)` group (runs by default); `idp` moves to the
  optional `areg(...)` group (registered but off). Falling back to lico is then
  `OC_ADD_RUN_SERVICES=idp` + `OC_EXCLUDE_RUN_SERVICES=auth-authelia` (plus the
  issuer/route notes below).

* **OIDC issuer = `<OC_URL>/authelia`.** Nine services derive the issuer from
  `OC_URL;OC_OIDC_ISSUER;…` (last value wins, so `OC_OIDC_ISSUER` overrides
  `OC_URL`). Authelia serves OIDC under the `/authelia` base path, so the issuer
  is the subpath. The `server` command sets `OC_OIDC_ISSUER=<OC_URL>/authelia`
  before config parsing when it is unset, which propagates to all nine without
  per-service wiring. An explicit `OC_OIDC_ISSUER` (external IdP, or lico fallback
  which needs the root `<OC_URL>`) is left untouched. Standalone single-service
  deployments set `OC_OIDC_ISSUER` themselves.

  *Why not serve Authelia at the root so `issuer = OC_URL` needs no change?*
  Authelia serves its login portal and its OIDC endpoints under one shared base
  path and derives the issuer from it (`IssuerURL()` → `BasePath()`); the portal
  index is `GET /`. At the root the portal would collide with the OpenCloud web UI
  (also `/`), breaking the redirect-based login. The subpath is required, hence
  the issuer subpath.

* **Proxy routes** (`services/proxy/.../defaultconfig.go`): the `/authelia` route
  (→ Authelia's listener) stays; the lico-specific `/konnect/` and `/signin/`
  routes and the root `/.well-known/openid-configuration` route are removed —
  discovery is now at `<OC_URL>/authelia/.well-known/openid-configuration`, served
  by the `/authelia` route. The default `proxy.oidc.issuer` literal becomes
  `https://localhost:9200/authelia` for the no-env localhost case.

* **Access-token verification = `none`** (`proxy.oidc.access_token_verify_method`):
  Authelia issues **opaque** access tokens, whereas lico issued JWTs. The proxy's
  `jwt` method verifies the token locally against the JWKS and cannot validate an
  opaque token, so the default is `none`: local JWT verification is skipped and the
  token is validated against Authelia's `/userinfo` endpoint instead (`SkipUserInfo`
  stays `false`; the result is cached). This is validation by the IdP, not a bypass.
  The proxy already anticipates this pairing (see the Authelia branch in
  `services/proxy/pkg/middleware/oidc_auth.go`). The cost is a cached `/userinfo`
  round-trip per new token; to avoid it, configure Authelia clients with
  `access_token_signed_response_alg: 'RS256'` (JWT access tokens) and switch the
  method back to `jwt`.

### Service startup ordering

`auth-authelia` runs in priority group 4 (with `frontend`), not group 3. Its
provider startup check dials the idm LDAP server (`ldaps://…:9235`), which the idm
service brings up in group 3; the runtime's post-group-3 wait ensures idm is
listening before group 4 starts. Registering it in group 3 races idm and fails the
startup check with "connection refused". (As an `areg`/optional service it was
implicitly scheduled after all groups, which masked the ordering requirement.)

### Risks and mitigations

* **Experimental API instability.** `doc.go` warns of breaking changes at any
  minor bump and methods that may panic. *Mitigation:* pin exact Authelia
  versions, gate upgrades behind an integration test that boots
  `embed.New` + `ServiceRunAll` and exercises an OIDC flow, budget for glue
  churn per bump.
* **Dependency graph / binary bloat.** Authelia pulls in fasthttp, its own SQL
  storage (postgres/mysql/sqlite), webauthn, totp, session and regulation.
  *Mitigation:* run the spike below first and measure `go mod graph` deltas,
  build time and binary size before committing.
* **Config model mismatch.** `embed.Configuration` is a type **alias** to the
  `internal` `schema.Configuration`; external code can hold the value but cannot
  import the internal package to construct nested fields. *Mitigation:* drive
  configuration through a generated YAML file + `embed.New(paths, filters)` (the
  approach prototyped by @rhafer in 2023), not by building structs in Go. The
  service renders that file itself on every start (see "Config generation" below),
  so there is no struct-construction path to maintain.
* **Two HTTP servers / proxy wiring.** *Mitigation:* register Authelia's port in
  the OpenCloud proxy/route table; document the topology for operators.

### Implementation Steps

1. **Dependency spike** (timeboxed): in a scratch module, `go get
   github.com/authelia/authelia/v4@v4.39.20`, call `embed.New` with a minimal
   generated config and `embed.ServiceRunAll`, boot it standalone against the
   libregraph LDAP backend. Record module-graph, build-time and binary-size
   deltas. Go / no-go gate.
2. **Re-engage upstream** ([#5803][issue] thread): confirm with @james-d-elliott
   that the supervised-daemon embed model covers our use case and clarify the
   support expectations for `experimental/embed`.
3. **Config generation layer**: template an Authelia YAML config from OpenCloud's
   config system (issuer, OIDC clients, LDAP backend, session/storage, TOTP),
   rendered by the service itself on every start (secrets persisted separately)
   rather than by `opencloud init` (see "Config generation" below). This is the
   bulk of the work.
4. **Service skeleton**: new opt-in service wrapping `embed.ServiceRunAll` in a
   `runner.Runner`, registered with the suture supervisor in
   `opencloud/pkg/runtime/service/service.go`; selectable vs. lico via config.
5. **Routing / proxy**: register Authelia's listener in the proxy/route table;
   wire tracing and access-log correlation.
6. **Tests & docs**: boot + OIDC-flow integration test pinned to the chosen
   Authelia version; admin docs in the docs repo (`/docs/admin/configuration/`)
   covering the lico ↔ Authelia choice and the new settings.
7. **Review default**: once validated in production, revisit whether Authelia
   becomes the default IdP (a follow-up ADR referencing this one).

### Config generation

Authelia is configured entirely through a YAML file (`embed.New(paths)`), so the
integration needs that file to exist with valid secrets, the OIDC clients, and
the LDAP backend pointing at the embedded libregraph-idm directory. Two patterns
were possible:

* **Render it from `opencloud init`** (the first prototype): init writes
  `authelia.yaml` next to `opencloud.yaml`. Rejected — it makes init aware of a
  second, foreign config schema, and no other service works this way.
* **Render it from the service** (chosen): the `auth-authelia` service renders the
  Authelia config in its `Server` command, then calls `embed.New`. This mirrors
  the idp (lico) service, which generates its clients registration config in
  `NewService` (`createTemporaryClientsConfig`), and keeps init free of Authelia
  specifics. (See the Decision below for *when* it renders — on every start, with
  secrets persisted separately.)

Decision: **the service renders its own config on every start.** Concretely
(`services/auth-authelia/pkg/render`).

A first prototype rendered `authelia.yaml` **once** (only if missing) and reused
it thereafter, so the embedded secrets would persist. That turned out to be a
trap: the file also froze every value *derived* from OpenCloud config (OIDC
issuer / `OC_URL`, SMTP settings, the LDAP bind password). After an admin changed
those in OpenCloud, the stale `authelia.yaml` silently kept the old values, with
no signal that it no longer reflected the configuration.

The fix is to **separate the regenerated config from the persisted secrets**, the
same split OpenCloud already uses elsewhere (e.g. lico's `createTemporaryClientsConfig`
is rewritten on every start, while its signing keys are generate-once files).
Authelia deep-merges multiple config files, so the service passes a small set:

* **`authelia.yaml` — regenerated on every start** from the current OpenCloud
  configuration, so it can never go stale. It holds only non-secret, derived
  values (issuer/domain, OIDC clients, LDAP backend incl. bind password, storage
  path, notifier, log level) and carries a `GENERATED FILE - DO NOT EDIT` header
  that points admins at OpenCloud config / env vars.
* **`authelia.secrets.yaml` — generated once**, then left untouched. It holds the
  random secrets (session, storage-encryption, OIDC HMAC and RSA signing key,
  password-reset JWT). Persisting these keeps active sessions and issued OIDC
  tokens valid across restarts and config regenerations; deleting the file forces
  fresh secrets.
* **`authelia.override.yaml` — optional, admin-managed**, never written by
  OpenCloud, merged **last** so it takes precedence. This restores the ability to
  customise Authelia (settings OpenCloud does not manage) without reintroducing
  the staleness problem, since the generated file is no longer the edit surface.

The public URL (OIDC issuer, session cookie domain, client redirect URIs) comes
from the shared `OC_URL` (`cfg.Commons.OpenCloudURL`).

**Secrets vs. init's role.** The Authelia-internal secrets are generated and
persisted by the service (in `authelia.secrets.yaml`), not by init. The only value
shared with the rest of OpenCloud is the **LDAP bind password**: rather than
seeding a dedicated `authelia` LDAP account, Authelia **reuses the existing `idp`
service user** (`uid=idp,ou=sysusers,o=libregraph-idm`) — the same account the
lico idp binds as. As with every LDAP service user, `opencloud init` generates
that password and persists it in `opencloud.yaml`; for Authelia it is written to
`auth_authelia.ldap.bind_password` (mirroring `idp.ldap.bind_password`, and equal
to it) and the service reads it via its config. So init still owns *passwords*
(consistent with all services) and seeds no new account, while the service owns
the *Authelia config file*.

This keeps the "single binary, no external dependency" property: a fresh
deployment that has run `opencloud init` and starts the opt-in `auth-authelia`
service gets a working IdP with no extra config steps, and one whose generated
config always reflects the live OpenCloud configuration.

### Serving the Authelia frontend

Authelia serves its **own** login portal — OpenCloud does not host the assets.
The flow, verified end-to-end against a running `auth-authelia` service:

* The compiled React portal is **embedded into the binary** at compile time via
  `//go:embed public_html` in Authelia's `internal/server/asset.go`. There is no
  separate static-asset directory to deploy.
* Authelia serves the portal, the OIDC endpoints and the assets from **its own
  HTTP listener** (`127.0.0.1:9091`, base path `/authelia`, set via
  `server.address` in the rendered `authelia.yaml`).
* The OpenCloud proxy forwards `/authelia/*` to that listener (the
  `/authelia → http://127.0.0.1:9091` route in
  `services/proxy/pkg/config/defaults/defaultconfig.go`).
* **Base path is automatic.** Authelia strips its configured router path
  (`middlewares.StripPath`) and templates `<base href=".../authelia/">` into
  `index.html`, so the SPA's relative `./static/...` asset URLs resolve under
  `/authelia` without any rewriting on the OpenCloud side.
* **OIDC issuer needs forwarded headers.** Authelia derives the effective issuer
  from `X-Forwarded-Proto` / `X-Forwarded-Host`. The OpenCloud proxy sets these
  (`req.SetXForwarded()` in `services/proxy/pkg/router/router.go`), so behind the
  TLS-terminating proxy the issuer resolves to `https://<OC_URL>/authelia`.
  Hitting the `:9091` listener directly over plain HTTP returns `400` on the OIDC
  endpoints — this is expected, not a bug.

#### Building the embedded frontend

`go:embed` only captures what is on disk **when the OpenCloud binary is
compiled**. While we build against a local `replace` checkout of Authelia (rather
than a published module), the frontend must be built into that checkout first.
Two sharp edges, both hit during the spike:

1. **`vite build` wipes the committed `api/` placeholders.** Authelia commits
   `internal/server/public_html/api/{index.html,openapi.yml}` as **0-byte
   placeholders** (the real Swagger docs are generated by a separate build step).
   Authelia's `templates.LoadTemplatedAssets` **requires both files to exist** at
   startup. `vite build` empties the output dir and deletes them, so the service
   then fails to boot with
   `loading template 'assets/public_html/api/index.html': file does not exist`.
   The empty placeholders are enough (they only blank the `/api` docs page; the
   portal and OIDC are unaffected).

2. **`go:embed` is compile-time.** Changing the assets on disk does nothing to an
   already-built binary; OpenCloud must be rebuilt afterwards.

The reproducible sequence is therefore:

```bash
# 1. build the portal into the Authelia checkout (the replace target)
cd <authelia-checkout>/web && pnpm install && pnpm build   # vite → internal/server/public_html

# 2. restore the committed api/ placeholders that vite deleted
git -C <authelia-checkout> checkout -- internal/server/public_html/api

# 3. rebuild the OpenCloud binary so go:embed captures the real assets
cd <opencloud> && make build
```

This friction is a direct consequence of the local `replace` directive and
**disappears once we depend on a published `authelia/v4` release**: the module zip
on the proxy ships `public_html` already built, including the generated `api/`
docs. Until then, steps 1–3 (or an equivalent build target) must run before any
build that needs a working portal.