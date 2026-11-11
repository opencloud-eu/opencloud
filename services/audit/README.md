# Audit

The audit service writes the audit log of OpenCloud: who did what to which object, when, from where and with which outcome. It records successful actions as well as failed and denied ones, for example logins with a wrong password or a user trying to give themselves the admin role.

The services emit audit events into the JetStream stream `audit`. The audit service reads that stream and writes every event as one line of JSON. Shipping the records to a SIEM or a log store is done by a log agent or the OpenTelemetry Collector, not by OpenCloud.

The design is described in [ADR 0006](../../docs/adr/0006-audit-logging.md).

## Enabling the Audit Log

Two things are needed:

- The services must emit audit events. Set `OC_AUDIT_ENABLED=true`. It is read by the proxy, graph, frontend, sharing and storage-users services, each of them also has its own variable, e.g. `PROXY_AUDIT_ENABLED`.
- The audit service must run. It is not started automatically when running as single binary via `opencloud server` or as docker container. Add it with `OC_ADD_RUN_SERVICES=audit`.

Events emitted while the audit service is not running are kept in the stream and written when it starts, see `AUDIT_STREAM_MAX_AGE`.

## Audit Records

Every record is one line of JSON. Field names follow the OpenTelemetry semantic conventions where they exist.

```json
{"event.id":"019a3c1e-7b2f-7c4e-9a10-5d2f0e6b8a41","event.name":"opencloud.auth.login","event.code":"A001F","schema_version":"1.0","time":"2026-10-28T09:14:03.512Z","outcome":"failure","outcome.reason":"invalid_credentials","severity":"warning","tenant.id":"","actor.type":"user","actor.name":"alice","auth.method":"basic","client.address":"203.0.113.7","user_agent.original":"Mozilla/5.0 ...","target.type":"user","target.id":"alice","target.name":"alice","service.name":"proxy","service.version":"5.1.0","ocsf.class_uid":3002,"ocsf.activity_id":1}
```

| Field | Content |
|---|---|
| `event.id` | UUIDv7, unique per event |
| `event.name` | `opencloud.<object>.<action>`, see the event catalog |
| `event.code` | stable code per event and outcome: a letter for the category, a number and `S`, `F` or `D` for success, failure or denied |
| `schema_version` | `1.0`, changes are additive only |
| `time` | RFC 3339, UTC, milliseconds |
| `outcome`, `outcome.reason` | `success`, `failure` or `denied`, with a machine readable reason on failures |
| `severity` | `info`, `notice`, `warning` or `alert`; failures and denials are at least `warning` |
| `tenant.id` | empty on single-tenant installations |
| `actor.type`, `actor.id`, `actor.name`, `actor.impersonated_by` | type is `user`, `service`, `public_link`, `guest`, `anonymous`, `system` or `cli` |
| `auth.method`, `auth.client_id`, `session.id` | `oidc`, `basic`, `app_token`, `public_link`, `guest_link` or `signed_url`; the OIDC client and session |
| `client.address`, `user_agent.original` | address of the client, see [Client Address](#client-address) |
| `target.type`, `target.id`, `target.space.id`, `target.name`, `target.path` | the object acted on |
| `change.before`, `change.after` | changed attributes of an update, never secrets |
| `trace_id`, `span_id` | OpenTelemetry trace correlation |
| `service.name`, `service.version`, `service.instance.id` | the service that emitted the event |
| `ocsf.class_uid`, `ocsf.activity_id` | mapping to OCSF 1.9.0, for SIEMs that normalize to OCSF |

Records contain no human readable message. Passwords, tokens and file contents are never written. Public links are identified by their share ID; when a link lookup fails there is no share ID, then a hash of the token is used.

## Event Catalog

| Category | Object | Actions |
|---|---|---|
| Authentication | `auth` | login, backchannel_logout |
| Credentials | `app_token` | create, delete |
| | `guest_token` | create, renew |
| | `user.password` | change, reset |
| Files | `file` | upload, download, preview, open_in_app, rename, move, copy, delete, restore, purge, lock, unlock |
| | `file.version` | restore, delete |
| | `file.malware` | detected |
| | `folder` | create, download, rename, move, copy, delete, restore, purge |
| | `trash` | purge_all |
| Sharing | `share` | create, update, delete, accept, decline |
| | `public_link` | create, update, delete, access |
| | `ocm.share` | create, update, delete |
| | `ocm.invite` | create, accept |
| Spaces | `space` | create, update, enable, disable, delete |
| | `space.member` | add, update, remove |
| | `space.quota` | change |
| Identity | `user` | create, update, enable, disable, soft_delete, delete |
| | `group` | create, update, delete |
| | `group.member` | add, remove |
| | `app_role` | assign, unassign |
| Administration | `app_role.definition` | change |
| | `sharing_role` | enable, disable |
| | `settings` | change |
| | `config` | change |
| | `cli` | command |
| | `personal_data` | export |
| Security | `access` | denied |

The catalog is defined in reva's `pkg/audit`. Some of its events are not emitted yet: app and guest tokens, previews, open in app, copies, deleted versions, the malware finding, OCM, settings, configuration and CLI commands, app role and sharing role definitions, and `access.denied`. `group.update` is only recorded when it fails.

Some actions are left out on purpose:

- Thumbnails are not audited. `file.preview` is opening a file in the preview app.
- Uploads and downloads are recorded when the transfer is done. Uploads and downloads by OpenCloud itself, e.g. for a copy, thumbnails, search or an archive, are not recorded.
- A download that is resumed or seeked makes many range requests, only the one starting at the first byte counts.
- `auth.login` is recorded for a new OIDC session. Basic auth and app tokens authenticate every single request, for them only failed logins are recorded. Expired OIDC tokens are part of the normal token renewal and are not recorded as failed logins.
- `public_link.access` is recorded once per link and client within 30 minutes, every failed attempt is recorded. The web client asking whether a link needs a password is not an attempt.

## Outputs

The audit service writes to stdout, one record per line and nothing else, see `AUDIT_OUTPUT_STDOUT`. Any log agent can pick it up. When the audit service runs with other services in one process or container, their logs share the same stdout; run it in its own container to keep the audit log separate.

With `AUDIT_OUTPUT_OTLP_ENDPOINT` the audit service also sends the records to an OpenTelemetry Collector as OTLP logs over gRPC; use `AUDIT_OUTPUT_OTLP_INSECURE` for a Collector without TLS. Every record is an OpenTelemetry event: the event name is `event.name`, the fields are attributes and the body is the record as JSON. The Collector routes the records to the SIEM, e.g. with its Splunk HEC, Elasticsearch, OpenSearch or syslog exporters. Every record is sent on its own and acknowledged only after the Collector accepted it, so a Collector outage delays the records but loses none.

At least one output must be enabled.

## Delivery

The services publish the events to the JetStream stream `audit`. The stream is a buffer, not the archive: it keeps the events for `AUDIT_STREAM_MAX_AGE` (default 7 days). Use `AUDIT_STREAM_REPLICAS=3` in a clustered NATS.

The audit service acknowledges an event only after it was written. If it stops or an output fails, the event is delivered again. A record can therefore appear twice after a restart, with the same `event.id`.

`OC_AUDIT_MODE` decides what a service does when it can't store an event:

- `best_effort` (default) publishes in the background and never slows down a request. Events that can't be stored are dropped and counted in `opencloud_audit_events_dropped_total`.
- `strict` waits until the event is stored. In this version the events are emitted after the action, so a failed event is logged and counted, but the request still succeeds.

## Client Address

The proxy determines the client address. It only uses the `X-Forwarded-For` header, or the header set in `PROXY_CLIENT_IP_HEADER`, when the request comes from a trusted reverse proxy, see `PROXY_TRUSTED_PROXIES`. The default trusts the loopback and the private network ranges.

Check the setting if your reverse proxy is not in a private network:

- Your reverse proxy reaches OpenCloud from a public address, e.g. Cloudflare, a load balancer of a cloud provider or a proxy on another server. Its headers are ignored with the default, every request then has the address of the proxy and a SIEM sees all logins coming from one address. Add the addresses of your proxy to `PROXY_TRUSTED_PROXIES`, for Cloudflare its published IP ranges.
- Clients can reach OpenCloud from a private network without passing your reverse proxy. With the default they can set their address themselves. Set `PROXY_TRUSTED_PROXIES` to the addresses of your reverse proxies only.

`PROXY_TRUSTED_PROXIES` replaces the default, list the loopback and private ranges again if you still need them. See [Trusted Reverse Proxies](../proxy/README.md#trusted-reverse-proxies) in the proxy service.

The proxy passes the address and the other request details to the services in the `X-Audit-Context` header and drops any value a client sent. The services trust this header like the other internal headers, so the services must not be reachable directly from outside.

## Metrics

| Metric | Content |
|---|---|
| `opencloud_audit_events_emitted_total` | events handed to the emitter, by service, event and outcome |
| `opencloud_audit_events_dropped_total` | events not stored, by service and reason |
| `opencloud_audit_publish_duration_seconds` | time to store an event in the stream |
| `opencloud_audit_records_total` | records handled by the audit service, by result |
| `opencloud_audit_sink_errors_total` | failed writes to an output |
| `opencloud_audit_consumer_lag` | events in the stream the audit service has not read yet |

Alert on dropped events and on a growing consumer lag.

## Legacy Audit Log

The audit log of earlier versions is still available with `AUDIT_LEGACY_ENABLED=true` and will be removed in a later version. It reads the domain events instead of the audit stream and uses the `AUDIT_LOG_TO_CONSOLE`, `AUDIT_LOG_TO_FILE`, `AUDIT_FILEPATH` and `AUDIT_FORMAT` settings. Its records have a different format:

```json
{"RemoteAddr":"","User":"user_id","URL":"","Method":"","UserAgent":"","Time":"","App":"admin_audit","Message":"user 'user_id' trashed file 'item_id'","Action":"file_delete","CLI":false,"Level":1,"Path":"path","Owner":"user_id","FileID":"item_id"}
```
