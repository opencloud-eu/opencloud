---
status: proposed
date: 2025-07-07
author: Pascal Bleser <p.bleser@opencloud.eu>
decision-makers:
consulted:
informed:
title: "Groupware Configuration Settings"
template: https://raw.githubusercontent.com/adr/madr/refs/tags/4.0.0/template/adr-template.md
---

* Status: draft

## Context

User Preferences need to be configurable through the UI and persisted in a backend service in order to be reliably available and backed up.

Such configuration options have default values that need to be set on multiple levels:

* globally
* by tenant
* by sub-tenant
* by group of users
* by user

Some options might even be client-specific, e.g. differ between the OpenCloud Web UI on desktop and the OpenCloud Web UI on mobile.

Furthermore, some options might be enforced and may not be overridden on every level (e.g. only globally or by tenant, by not modifiable by users.)

Ideally, the configuration settings have an architecture that permits pluggable sources.

This level of necessary complexity has a few drawbacks, the primary one being that it can become difficult to find out why a user sees this or that behavior in their UI, and thus to trace down where a given configuration setting is made (globally, on tenant level, etc...). It is thus critical to include tooling that allows to debug them.

## Considered Options

TODO

## Decision Outcome

TODO

### Consequences

TODO

### Confirmation

TODO

## Pros and Cons of the Options

TODO
