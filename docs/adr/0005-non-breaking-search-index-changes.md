---
title: "Non-Breaking Search Index Changes"
---

* Status: proposed
* Deciders: [@fschade, @rhafer, @aduffeck, @butonic]
* Date: 2026-07-29

Reference: https://github.com/opencloud-eu/opencloud/pull/2659#issuecomment-5103043714

## Context and Problem Statement

The search service runs on one of two engines, bleve or OpenSearch. Both store a mapping
that describes the shape of an indexed document. Whenever we change that shape in an
incompatible way, for example a changed field type or a different analyzer, the index
that is already on disk or in the cluster no longer matches what the code expects.

Today this stops the instance. The service detects the mismatch on startup, returns an
error and refuses to come up until the operator drops the index and reindexes. Two
consequences follow from that:

* an upgrade can take an instance down, and it stays down until the operator reacts,
* the change is breaking, so we can only ship it with a major release.

We already hit this once and solved it by waiting for the next major release. That is
not a process, it just moves the problem.

## Decision Drivers

* an upgrade must never keep an instance from starting
* index changes should be shippable in a minor release
* reindexing is expensive, it costs CPU and can run for hours on large data sets, so the
  operator has to decide when it happens
* users should not silently lose search results and favorites without knowing why

## Considered Options

1. **Keep failing on startup.** Ship index changes only in major releases. Safe in the
   sense that nothing is silently empty, but the instance is down and we are blocked by
   the release cadence.
2. **Migrate automatically on startup.** The service reindexes itself into the new
   mapping before it serves requests. Fully automated, but it blocks the start for an
   unbounded time and it takes the timing away from the operator.
3. **Create a new index, keep the old one.** The service creates a fresh index for the
   new mapping and starts using it immediately. Reindexing is a separate step, triggered
   by the operator.

## Decision Outcome

We go with option 3.

On a breaking index change the search service creates a new index instead of touching
the existing one. For OpenSearch that is a new index name, for bleve a new directory.
The old index is left alone, it is neither read nor deleted, so a rollback to the
previous release still finds its data. The instance starts, everything else keeps
working.

The price is that the new index starts out empty. Search returns no results and
favorites look empty. From there it fills up again on its own, every upload, move or
metadata change is indexed as it happens, but that only covers what is touched after
the upgrade. Everything older stays invisible until the operator reindexes. Nothing is
lost, the storage is the source of truth, but it is visible to every user. So the manual
step needs to be communicated in two directions:

* **to the operator**, through the release notes. Every release that changes the index
  says so and points to the reindex command.
* **to the users**, through the web UI. The operator can publish a sticky message that
  announces the reindex and explains why search and favorites are incomplete during
  that time. The operator also decides when to run it, for example outside of working
  hours.

Release notes alone are not enough here. Plenty of instances are upgraded by pulling a
new image tag, and nobody reads a changelog for that. So the service has to say it
itself: when it creates a new index it logs that a reindex is pending and names the
command. If we skip this, the realistic path is that the operator learns about it from
users who miss their favorites.

This is only about the moment the index is created. Whether a reindex has finished is
not something we track, and the reindex is the operator's job anyway.

We deliberately do not ship this batteries included. A migration that runs by itself is
a black box for us, we cannot look into the instance it runs on and we cannot promise it
goes through cleanly. Data volume, cluster sizing and what load is acceptable are things
the operator knows and we do not. So the operator triggers the reindex and watches it.
We would rather have a manual step that is understood than an automatic one that fails
halfway.

Old indexes stay around and cost storage. Removing them is the operator's call as well,
we only document how.

To be clear about what this decision does and does not buy us: the upgrade is no longer
breaking, the instance comes up on its own. But it is still a manual step, and without
the release note and the sticky message it looks like data loss to the user.

### Implementation Steps

1. Derive the index name (OpenSearch) and the index path (bleve) from the mapping
   version, so a breaking mapping change automatically means a new, empty index.
   `SEARCH_ENGINE_OPEN_SEARCH_RESOURCE_INDEX_NAME` becomes the prefix of that name
   instead of the full name. A configured index can then never be the wrong version,
   there is only "this version doesn't exist yet", which is the normal path.
2. Keep the mismatch detection as it is and only change what an operator gets out of it:
   a new index instead of a service that refuses to start. The error path stays, as the
   fallback when the new index cannot be created and as a strict mode for development,
   where failing loudly on an unversioned mapping change is what we want.
3. Log that a reindex is pending when a new index is created, and name the command.
4. Make the reindex fit for a long run. It has to report progress, survive longer than a
   fixed timeout, and a single failing space must not kill the whole run.
5. Add the sticky message to the web UI, publishable by the operator. In progress in
   https://github.com/opencloud-eu/opencloud/pull/3189.
6. Document the reindex, `opencloud search index --all-spaces`, and the cleanup of old
   indexes in the admin docs.
7. Add the manual step to the release notes of every release that bumps the index
   version.
