# Threat model OpenCloud

## What this project does and where untrusted input enters

OpenCloud is a self-hosted file sync and share platform written in Go. It runs as one binary that starts many services. Only the proxy is meant to be reachable from the outside.
- Many users share one instance. Each user's files must stay within their personal space, spaces that they are member of and what was shared with them, at the permissions of the share.
- Storage, sharing and WebDAV come from Reva (`github.com/opencloud-eu/reva/v2`), which is vendored under `vendor/`.

Untrusted input enters through the proxy on port 9200:
- Unauthenticated requests: public links (`/s/<token>`, `/remote.php/dav/public-files/<token>`), the IDP login and OIDC endpoints (`/signin`, `/konnect`, `/.well-known/openid-configuration`), `/status.php`, capabilities and webfinger.
- Guest links: `verify/token`, `renew` and `verify/pin` under `/graph/v1beta1/extensions/org.libregraph/guestLinks/` take requests without a session.
- Authenticated users via WebDAV (`/remote.php/dav/spaces`, `/remote.php/dav/files`, `/remote.php/webdav`), the LibreGraph API (`/graph`), OCS (`/ocs/v[12].php`), TUS uploads, app tokens (`/auth-app/tokens`), the archiver (`/archiver`), settings (`/api/v0/settings`), and the search, thumbnails, sse and userlog endpoints.
- File content uploaded by users: thumbnails, search indexing, and archive downloads.

## Principals and trust
From least to most privileged:
- **Anonymous visitor.** Is not signed in and sees only public endpoints and the login page.
- **Public link holder.** Holds a link token, sometimes with a password, sometimes upload-only (file drop). Must reach only the shared resource, with only the link's permissions, and only after the password check.
- **Guest.** Invited by email to a single share, with no OpenCloud account. Redeems a single-use link token (valid 30 minutes) or, after a renewal, a 6-digit PIN bound to a nonce cookie of the browser that renewed. Either way gets a signed session cookie (`__Host-oc_guest_session`, 24 hours). Must reach only the shared resource, with only the share's permissions, and nothing once the share is removed or expired. Knowing only the share id must not be enough to get a session, to get a PIN, or to lock the guest out.
- **User.** Must stay within their personal space, the project spaces they are a member of, and what was shared with them. Space roles (viewer, editor, manager) and share permissions must hold, and resharing must not exceed what the resharer has.
- **App token.** Must not exceed the scope of the user who issued it. It serves as a basic auth token for 3rd party clients.
- **Space admin.** The "Space Admin" and the "Admin" role can create and manage project spaces, but must not read the content of spaces they are not a member of.
- **Admin.** The admin is able to manage the users and groups on top of the "Space Admin" permissions. Access to content is only given via space membership or sharing.
- **SysAdmin** Not necessarily the application admin, is the operator, the config in `/etc/opencloud`, and the data in `/var/lib/opencloud`.

The services trust each other. Their internal gRPC and HTTP ports, NATS, and the embedded LDAP server (idm) bind to localhost and are not reachable from outside. A finding needs a path through the proxy. The whole service mesh is always deployed behind a reverse proxy.

## Components that matter most / least
- Most important:
  - authentication in the proxy: OIDC, basic auth, app tokens, and the user and role lookup (`services/proxy`, `services/auth-*`)
  - the IDP (`services/idp`, vendored `github.com/libregraph/lico`)
  - guest links: tokens, PINs, nonces, sessions and their rate limits (`services/auth-guest`), the guest session check in the proxy, and the guest invite in `services/graph`. `services/auth-guest/README.md` describes the intended flow.
  - sharing and public links (`services/sharing`, `services/storage-publiclink`, `services/ocs`, and the share managers and public share provider in Reva)
  - space membership and permissions in the LibreGraph API (`services/graph`)
  - WebDAV and the storage drivers, including path handling and the metadata folders inside spaces (Reva `internal/http/services/owncloud/ocdav`, `pkg/storage/fs/decomposed`, `pkg/storage/fs/posix`)
  - outgoing requests: webfinger and app provider URLs
- Asymmetries: a check enforced on one path (WebDAV, Graph, OCS) but not on another path to the same data.
- Oracles: a difference in a response that reveals something the requester cannot see.
- Also in scope: the vendored `github.com/opencloud-eu/*` and `github.com/libregraph/*` packages. Reva's tests are not vendored, so a reproducer for Reva has to go through OpenCloud.
- The rest of `vendor/` is in scope only where OpenCloud reaches a defect in it in an exploitable way.
- Out of scope: `tests/`, `devtools/`, `deployments/`, `docs/`, the generated mocks and `protogen/`, and the compiled web frontend in `services/web/assets` (its source is in opencloud-eu/web).
- Out of scope: OCM and federated sharing (`services/ocm`, the OCM code in Reva, `/ocm`, `/.well-known/ocm`, `/sciencemesh`). Testing it needs a second public OpenCloud server.

## How to exercise it
- Run `opencloud-start`. It runs `opencloud server` in the background, logs to `/var/log/opencloud.log`, and returns once WebDAV answers, usually after 10 to 20 seconds. `opencloud-stop` stops it.
- It runs as unprivileged user `www-data`
- The first start creates the demo users. Each user's personal space is created on their first request.
- TLS uses a self-signed certificate, so use `curl -k`.
- Basic auth is enabled (`PROXY_ENABLE_BASIC_AUTH=true`) so you can call the APIs without OIDC. Production instances usually have it off.
- Accounts, all with the password `demo` except admin:
  - `admin`/`admin`: role admin
  - `dennis`: role admin
  - `margaret`: role space admin
  - `alan`, `mary`, `lynn`: role user
- Use the Graph API to create project spaces and invite members, and OCS or Graph to create shares and links.
- Guest links are enabled (`OC_ENABLE_GUEST_LINKS=true`, off by default). Invite a guest with `POST /graph/v1beta1/drives/<space id>/items/<item id>/invite` and `{"recipients":[{"email":"guest@example.org"}],"roles":["<role id>"]}`. The viewer role is `b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5`.
- Mail goes to Mailpit, which `opencloud-start` starts. Read it with `curl http://127.0.0.1:8025/api/v1/messages` and `/api/v1/message/<ID>`. The link token is the `v1.<hash>.<secret>` part after `/g/` in the mail, the PIN follows `Your PIN:`.
- Redeem a token with `curl -k -c jar -H 'Content-Type: application/json' -d '{"token":"<token>"}' https://localhost:9200/graph/v1beta1/extensions/org.libregraph/guestLinks/verify/token`, then send the cookie with `-b jar`.
- Try `curl -k -u alan:demo -X PROPFIND https://localhost:9200/remote.php/dav/files/alan/` or `curl -k -u alan:demo https://localhost:9200/graph/v1.0/me/drives`.
- Storage driver: decomposed, with data under `/var/lib/opencloud`. Config is in `/src/opencloud`.
- Notifications runs (`OC_ADD_RUN_SERVICES=notifications`) to send the guest mails. Not running: antivirus, audit, collaboration (WOPI), policies and invitations, which are off by default. There is no ClamAV, Tika, office server, Redis or external LDAP.
- The binary is `/src/opencloud/bin/opencloud`. After changing code, rebuild it with `go build -tags disable_crypt -o /src/opencloud/bin/opencloud ./opencloud/cmd/opencloud`, then `opencloud-stop && opencloud-start`.
- Go tests: `go test ./services/<service>/...` or `go test ./pkg/...`. The build cache is already warm. The tests in `services/search/pkg/opensearch` need Docker and fail here.
- Run `opencloud-reset` if the instance stops working. It deletes all data and restores the config from the build. The next `opencloud-start` creates the demo users again.

## How you rate severity
Score every finding with CVSS 3.1 and report the vector.

Examples from published advisories:
- GHSA-9j2f-3rj3-wgpg (Reva) and GHSA-vf5j-r2hw-2hrw (OpenCloud), bypass of the public link scope, reaching resources outside the shared one: `CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N` (8.2, high).
- GHSA-3v8c-738m-4gcj (Reva) and GHSA-gf4p-7p27-26w7 (OpenCloud), authenticated users can reach internal metadata folders in their spaces: `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L` (6.3, medium).

## Anything to leave alone
- Self-XSS, missing hardening headers, and clickjacking on pages without sensitive actions.
- Anything that needs access to the internal service ports, NATS, or the embedded LDAP server.
- Weak settings that the admin can change, like the password policy or link expiry.
- `OC_INSECURE`, basic auth, the demo users and their passwords, Mailpit, and other choices made only for this image.
- Anyone who can read the guest's mailbox can use the guest link. That is by design.
- What admins can do by design.
- User enumeration through the sharing autocomplete, within what the instance is configured to allow.
- SSRF: a blind request to a public host is not a finding. Reaching local or private addresses is.

## Reports and patches
- One report per root cause, naming the principal, the boundary crossed, and every affected route (WebDAV, Graph, OCS).
- The build fills `services/idp/assets` and `services/web/assets` in `/src`. Leave them out of patches.
- Include a reproducer: HTTP requests against the instance above, or a Go test that fails before the fix.
- For a bug in Reva, patch the file under `vendor/github.com/opencloud-eu/reva/v2` and say so. The fix then goes upstream to opencloud-eu/reva.
- Patches should be minimal and fix the check where it is enforced, not in one caller. Say whether sibling routes have the same gap. Run `gofmt`, and match the surrounding code.
