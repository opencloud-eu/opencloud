---
title: "6. Audit Logging"
---

* Status: accepted
* Deciders: @micbar, @butonic, @schweigisito
* Date: 2026-10-28

Reference: replaces `services/audit`

## Context and Problem Statement

Our audit service still writes the legacy `admin_audit` format. The reasons for
that format are gone. Nobody consumes it, and OpenCloud runs on Kubernetes next
to OpenTelemetry and SIEM systems.

The current service has real problems:

- most fields of the legacy struct (`RemoteAddr`, `URL`, `Method`, `UserAgent`,
  `CLI`) are never filled, the events don't carry them
- file events write the resource owner into `User`, not the user who did the
  action
- space, user and group events leave `User` empty
- event ID, trace parent, impersonating user and tenant are on the event bus,
  but never reach the audit entry
- entries get lost: reva's `events.Consume` acks a message before the audit
  service writes it, write errors only get logged
- login failures, logout, app tokens, denied access, settings and role changes,
  admin CLI commands and GDPR exports are not emitted at all
- `FileDownloaded` fires when a download starts, not when it is done
- public link tokens and ScienceMesh invite tokens are logged in clear text
- the service is off by default
- with several replicas, the queue group splits the trail across pods, and
  there are no metrics for dropped entries
- an unknown `AUDIT_FORMAT`, including the documented empty value, panics on the
  first event

Security teams want audit data in their SIEM, in a stable format, with actor,
source address and outcome of every security relevant action.

ISO 27001 (A.8.15), BSI IT-Grundschutz OPS.1.1.5 and NIS2 ask for the same.

## Decision Drivers

* who did what to which object, when, from where and with which outcome
* also log failed and denied actions
* no silent loss of audit entries
* cheap SIEM onboarding: single-line JSON, stable event names, published
  mappings to OCSF, ECS and ASIM
* use cloud-native conventions like OpenTelemetry, CloudEvents, the Kubernetes
  audit policy, don't invent our own, if possible
* no secrets
* as little personal data as possible by default
* every event carries its tenant (see ADR 0001)
* no SIEM-specific code in OpenCloud

## Considered Options

### Source

#### Source option 1: Keep converting the domain events from `main-queue`, like today

Domain events are built for activitylog, userlog, search and postprocessing.
They have no request context, they are only published on success and some fire
at the wrong time. Fixing that in the shared events affects all their consumers.

#### Source option 2: Dedicated audit events, emitted where the request is handled

The proxy resolves the request context once (actor, client address, user agent,
session, auth method, tenant, trace) and passes it on in `context.Context` and
gRPC metadata. Services emit audit events through a small `pkg/audit` emitter,
including failures and denials. The reva events middleware keeps producing the
storage and sharing events, but with that context.

### Transport

#### Transport option 1: Every service exports audit logs with the OpenTelemetry SDK

The SDK keeps records in memory. A restart or a Collector outage loses them, and
without a Collector there is no audit trail at all.

#### Transport option 2: A dedicated JetStream stream `audit`

File storage, replicated in clustered NATS, subjects
`audit.<tenant>.<category>`, dedup with `Nats-Msg-Id` = event ID. Retention is a
buffer (7 days by default), not the archive. NATS is already a hard dependency.

### Schema

#### Schema option 1: Keep the legacy format

Half of the fields can't be filled, and there is no outcome, event ID, tenant or
trace ID.

#### Schema option 2: Emit full OCSF records

OCSF is the common format for security data (AWS Security Lake, Splunk, Google
SecOps). But the records are big and the schema moves fast, 1.9.0 just
deprecated two classes we would use. Elastic and Sentinel users normalize to ECS
and ASIM anyway.

#### Schema option 3: Our own versioned schema

It uses OpenTelemetry attribute names where they exist (`client.address`,
`user_agent.original`, `session.id`, `trace_id`) and has `ocsf.class_uid` /
`ocsf.activity_id` on every event. We publish mappings to OCSF, ECS and ASIM,
the Collector applies them. On the bus the record is CloudEvents data in binary
mode (`ce-id`, `ce-type`, `ce-source`, `ce-time`, `traceparent`, `partitionkey`
as NATS headers).

### Delivery

#### Delivery option 1: Never block a request

Like Kubernetes `batch` mode, buffer and drop on overflow.

#### Delivery option 2: Refuse requests that can't be audited

Like Vault and Kubernetes `blocking-strict`.

#### Delivery option 3: Both, the operator decides

`best_effort` is the default: publish async with a bounded buffer, count and log
every drop. `strict` waits for the JetStream ack and answers 503 if the event
can't be stored. That one is for regulated installations.

## Decision Outcome

We go with:

* Source option 2: dedicated audit events, emitted where the request is handled
* Transport option 2: a dedicated JetStream stream `audit`
* Schema option 3: our own versioned schema
* Delivery option 3: both, the operator decides

The audit service becomes a stateless consumer of the `audit` stream. It acks a
message only after all outputs have written it. The old service stays one major
release as "legacy audit", then we remove it.

### Event record

One line of JSON per event:

| Field | Content |
|---|---|
| `event.id` | UUIDv7, also CloudEvents `id` and `Nats-Msg-Id` |
| `event.name` | stable name, e.g. `opencloud.file.download` |
| `event.code` | short code per name and outcome |
| `schema_version` | `1.0`, only additive changes |
| `time` | RFC 3339, UTC, milliseconds |
| `outcome`, `outcome.reason` | `success`, `failure` or `denied`, plus a machine readable reason |
| `severity` | `info`, `notice`, `warning`, `alert` |
| `tenant.id` | empty on single-tenant installations |
| `actor.type`, `actor.id`, `actor.name`, `actor.impersonated_by` | type is `user`, `service`, `public_link`, `guest`, `anonymous`, `system` or `cli` |
| `auth.method`, `auth.client_id`, `session.id` | how the actor logged in |
| `client.address`, `user_agent.original` | real client address behind trusted proxies |
| `target.type`, `target.id`, `target.space.id`, `target.name`, `target.path` | `target.path` only at `detail` level |
| `change.before`, `change.after` | changed attributes, never secrets |
| `trace_id`, `span_id` | OpenTelemetry correlation |
| `service.name`, `service.version`, `service.instance.id` | emitting service |
| `ocsf.class_uid`, `ocsf.activity_id` | OCSF 1.9.0 mapping |

No human readable message. The SIEM or a viewer renders text from name and
fields.

### Event catalog

Every event is defined once in Go: name, codes, OCSF mapping, required fields.
The emitter validates against it and the catalog docs are generated from it.

Event names are `opencloud.<object>.<action>`, e.g.
`opencloud.space.member.add`.

| Category       | Object                | Actions                                                                                          | OCSF 1.9.0                        |
|----------------|-----------------------|--------------------------------------------------------------------------------------------------|-----------------------------------|
| Authentication | `auth`                | login, backchannel_logout                                                                        | Authentication 3002               |
| Credentials    | `app_token`           | create, delete                                                                                   | User Management 3007              |
|                | `guest_token`         | create, renew                                                                                    | User Management 3007              |
|                | `user.password`       | change, reset                                                                                    | User Management 3007              |
| Files          | `file`                | upload, download, preview, open_in_app, rename, move, copy, delete, restore, purge, lock, unlock | File Hosting Activity 6006        |
|                | `file.version`        | restore, delete                                                                                  | File Hosting Activity 6006        |
|                | `file.malware`        | detected                                                                                         | Detection Finding 2004            |
|                | `folder`              | create, download, rename, move, copy, delete, restore, purge                                     | File Hosting Activity 6006        |
|                | `trash`               | purge_all                                                                                        | File Hosting Activity 6006        |
| Sharing        | `share`               | create, update, delete, accept, decline                                                          | File Hosting Activity 6006        |
|                | `public_link`         | create, update, delete                                                                           | File Hosting Activity 6006        |
|                | `public_link`         | access                                                                                           | Web Resource Access Activity 6004 |
|                | `ocm.share`           | create, update, delete                                                                           | File Hosting Activity 6006        |
|                | `ocm.invite`          | create, accept                                                                                   | File Hosting Activity 6006        |
| Spaces         | `space`               | create, update, enable, disable, delete                                                          | Entity Management 3004            |
|                | `space.member`        | add, update, remove                                                                              | File Hosting Activity 6006        |
|                | `space.quota`         | change                                                                                           | Entity Management 3004            |
| Identity       | `user`                | create, update, enable, disable, soft_delete, delete                                             | User Management 3007              |
|                | `group`               | create, update, delete                                                                           | Group Management 3006             |
|                | `group.member`        | add, remove                                                                                      | Group Management 3006             |
|                | `app_role`            | assign, unassign                                                                                 | User Management 3007              |
| Administration | `app_role.definition` | change                                                                                           | Role Management 3008              |
|                | `sharing_role`        | enable, disable                                                                                  | Role Management 3008              |
|                | `settings`            | change                                                                                           | API Activity 6003                 |
|                | `config`              | change                                                                                           | API Activity 6003                 |
|                | `cli`                 | command                                                                                          | API Activity 6003                 |
|                | `personal_data`       | export                                                                                           | API Activity 6003                 |
| Security       | `access`              | denied                                                                                           | Web Resource Access Activity 6004 |

Failures and denials are no separate actions. They use the same event name with
`outcome` `failure` or `denied` and their own `event.code`, e.g. a failed login
is `opencloud.auth.login` with `outcome: failure`. A guest token request that
hits the rate limit is `opencloud.guest_token.create` with `outcome: denied` and
`outcome.reason: rate_limited`. `access.denied` is only for authorization
failures that have no more specific event.

`file.download` and `folder.download` (zip from the archiver) are emitted when
the download is done, not when it starts.

`app_role` are the roles in the settings service (Admin, Space Admin, User, User
Light). `sharing_role` are the unified roles used for shares and space members.

Logout and session expiry happen at the IdP. OpenCloud only sees the backchannel
logout, everything else comes from the IdP's own events (see Output and SIEM).

A policy decides what gets written. Ordered rules, first match wins, like the
Kubernetes audit policy. Rules match on event name (wildcards), actor type and
tenant and set the level `none`, `metadata` (default) or `detail`.

### Output and SIEM

The audit service writes to stdout (default, one JSON record per line and
nothing else), to an OpenTelemetry Collector via OTLP, or to syslog RFC 5424
over TLS. Any combination works.

Routing to Splunk, Elastic, OpenSearch, Sentinel, Google SecOps, AWS Security
Lake or Loki happens in the Collector. We ship tested Collector configs, the
generated catalog, the mappings and a few Sigma rules. Keycloak events can go
through the same Collector, we only document that.

### Privacy and integrity

- passwords, tokens and file contents are never written, public links are
  identified by their share ID
- paths and before/after values only at `detail` level
- optional pseudonymization of `actor.id`, `actor.name` and `client.address`
  with HMAC-SHA256, an admin CLI command hashes a known value for searching
- optional truncation of client addresses (/24 for IPv4, /48 for IPv6)
- retention is configured in the sink, not in OpenCloud
- integrity comes from shipping off the host quickly, to a SIEM or WORM storage
  like S3 Object Lock; a hash chain with signed checkpoints can come later

### Performance

The request context is cheap. A lot of it already travels with every gRPC call:
`x-access-token` carries the whole CS3 user (ID, tenant, name, mail), and
`x-user-agent`, `initiator-id` and `traceparent` are already there. New are
client address, session ID, auth method, OIDC client ID and
`actor.impersonated_by`, about 100 to 150 bytes per call. The access token alone
has 1 to 2 KB. We put the new values into one header, `x-audit-context`, not
five. HTTP/2 header compression makes repeated values within one request almost
free. In the process it is one `context.WithValue` per request.

The events cost more than the context:

- every audited action is one publish of about 1 KB to NATS, replicated three
  times in clustered NATS
- `best_effort` publishes async and doesn't add request latency
- `strict` waits for the JetStream ack, that adds one round trip (a few
  milliseconds in the cluster) to every audited request
- reads make the volume: `file.preview` is opening a file in the preview app,
  thumbnails are not audited, otherwise a folder with 200 images would write
  200 events
- `public_link.access` is emitted once per access, not on every token lookup

A rough number: 10,000 active users with 50 audited actions per day are 500,000
events or about 0.5 GB of JSON per day, 1.5 GB in the stream with three
replicas. NATS handles that easily. We measure it with the metrics from step 1
before we change any defaults.

### Consequences

Good: correct actor, client address, outcome, tenant and trace ID on every
record. Nothing gets lost silently. SIEM teams get a documented format instead
of writing parsers. Audit no longer depends on log levels or the operational
logs.

Bad: breaking change for everyone who parses the legacy format, that is why we
keep it one major release. Reva needs changes in the events middleware. One more
stream to size and monitor. `strict` mode makes requests depend on NATS.

### Implementation Steps

1. Foundation: event definitions, `pkg/audit` emitter, request context at the
   proxy and through reva, the `audit` stream, the new audit service with ack
   after write, stdout and OTLP output, metrics
   (`opencloud_audit_events_emitted_total`, `_dropped_total`,
   `_sink_errors_total`, consumer lag). Map everything the legacy service writes
   today, with the correct actor. In the Helm chart the audit service gets its
   own deployment and is enabled by default.
2. Coverage: authentication, app and guest tokens, password reset, access
   denied, file and folder downloads when done, settings, app and sharing roles,
   admin CLI, GDPR export, locks, soft delete, malware, OCM invites. Emit public
   link access once per access. Policy file.
3. SIEM: syslog output, generated catalog, mappings, tested Collector configs,
   Sigma rules, pseudonymization and address truncation.
4. Later: `strict` mode, hash chain with signed checkpoints, audit view for
   tenant admins.
5. Deprecate the legacy service with the release that contains step 1, remove it
   one major release later.

## References

* [Kubernetes auditing](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)
* [OpenTelemetry Logs Data Model](https://opentelemetry.io/docs/specs/otel/logs/data-model/)
* [OpenTelemetry Audit Logging SIG proposal](https://github.com/open-telemetry/community/pull/2409)
* [CloudEvents NATS binding](https://github.com/cloudevents/spec/blob/main/cloudevents/bindings/nats-protocol-binding.md)
* [CloudEvents Auth Context extension](https://github.com/cloudevents/spec/blob/main/cloudevents/extensions/authcontext.md)
* [OCSF 1.9.0 File Hosting Activity](https://schema.ocsf.io/1.9.0/classes/file_hosting)
* [Elastic Common Schema: event fields](https://www.elastic.co/docs/reference/ecs/ecs-event)
* [Microsoft Sentinel ASIM Audit Event schema](https://learn.microsoft.com/en-us/azure/sentinel/normalization-schema-audit)
* [HashiCorp Vault audit devices](https://developer.hashicorp.com/vault/docs/audit)
* [Harbor audit log](https://goharbor.io/docs/edge/administration/audit-log/)
