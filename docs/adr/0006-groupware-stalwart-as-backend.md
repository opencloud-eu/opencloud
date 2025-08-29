---
status: proposed
date: 2025-06-25
author: Pascal Bleser <p.bleser@opencloud.eu>
decision-makers:
consulted:
informed:
title: "Stalwart as Groupware Backend"
template: https://raw.githubusercontent.com/adr/madr/refs/tags/4.0.0/template/adr-template.md
---

* Status: draft

## Context

Which Groupware backend should be used?

## Considered Options

* [Stalwart](https://stalw.art/), contains not only mail but also collaborative features in an integrated package
* traditional IMAP/POP/SMTP stacks (e.g. [Dovecot](https://www.dovecot.org/) + [Postfix](https://www.postfix.org/))

## Decision Outcome

The decision was made to go with Stalwart, as it reduces the implementation effort on our end, allowing us for a much faster time-to-market with a significantly smaller team of developers.

### Consequences

We will most probably not need to develop much of a calendar or contact stack ourselves, as Stalwart is planning to implement those as part of the upcoming [JMAP](https://jmap.io/spec-core.html) specifications for [contacts](https://jmap.io/spec-contacts.html) and [calendars](https://jmap.io/spec-calendars.html).

The Groupware API will largely consist of a translation of high-level operations for the UI into JMAP operations sent to Stalwart.

#### Risks

On the flip side, there are a number of risks associated with that decision.

* Stalwart underdelivers on its promises
  * calendaring provides insufficient features for our implementation (e.g. event series handling being too basic)
  * not scaling for large deployments
  * necessary adaptations (e.g. for authentication integration) are rejected upstream
  * etc...

### Confirmation

TODO

## Pros and Cons of the Options

### Stalwart

* good: integrated package that contains IMAP/POP, SMTP, anti-spam, AI, encryption at rest and many other features in one
* good: modern stack
* good: capable of fault tolerance in large deployments through its use of [FoundationDB](https://www.foundationdb.org/)
* bad: relatively new project with few to no large scale productive deployments (yet)
* bad: significant human [SPoF](https://en.wikipedia.org/wiki/Single_point_of_failure)/[bus factor](https://en.wikipedia.org/wiki/Bus_factor) issue as the development team currently consists of one
* good: supports and drives the JMAP protocol ([JMAP Core](https://jmap.io/spec-core.html), [JMAP Mail](https://jmap.io/spec-mail.html), [JMAP Contacts](https://jmap.io/spec-contacts.html), [JMAP Calendars](https://jmap.io/spec-calendars.html), [JMAP Tasks](https://jmap.io/spec-tasks.html), ...),
  * which provides more high-level operations that we don't need to implement ourselves,
  * as well as a much cleaner specification that reduces efforts too,
  * and additionally can be implemented with an efficient stateless HTTP I/O stack
* bad: no viable broad JMAP implementation alternatives in case Stalwart does not deliver ([Apache James](https://james.apache.org/) only seems to support a basic subset of JMAP)
* good: implements a lot of Groupware "business logic" on its own, reducing the implementation effort on our end,
  * most notably by not having to deal with IMAP extensions and quirks,
  * or the complexity of calendar events

### IMAP/SMTP

* good: there are a number of alternatives in case a specific implementation does not deliver
* good: the best implementation candidates are well-established, used in large amounts of productive deployments, supported by teams of developers
* bad: more complex stack composed of numerous components as opposed to an all-in-one implementation
* bad: the effort and complexity of having to deal with IMAP,
  * its complexity due to its extensions and its many quirks,
  * as well as a significantly less efficient I/O stack that requires stateful session handling
* bad: requires the complete implementation of contacts, calendars and tasks in our own stack, as none of those services are provided by IMAP/SMTP backends
