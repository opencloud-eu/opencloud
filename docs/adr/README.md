# Architecture Decision Records (ADRs)

## Purpose

This folder contains Architecture Decision Records (ADRs) for the OpenCloud related topics.
ADRs capture important architectural decisions, their context, alternatives, and rationale.

They help us:

- Document the reasoning behind significant technical choices.
- Share knowledge and context with current and future team members.
- Ensure transparency and continuity in our architectural evolution.

## Why Use ADRs?

ADRs provide a structured way to record, discuss, and find architectural decisions over time.
They make it easier to:

- Understand why certain approaches were chosen.
- Avoid revisiting previous discussions without context.
- Onboard new contributors efficiently.

## When to Create an ADR

Not every technical or architectural decision needs a dedicated ADR.
Use an ADR to document decisions which are significant, such as:

* It substantially affects the architecture, design, or direction of OpenCloud.
* It involves trade-offs between multiple options.
* It needs Team consensus or input from multiple stakeholders.

## Writing ADRs

- **Location**: Store all ADRs as Markdown files in this folder.
- **Format**: Use [Markdown](https://commonmark.org/).
- **Naming**: Use a consistent naming convention, e.g., `0001-descriptive-title.md`.

### ADR Template

```markdown
---
title: "Some Descriptive Title"
---

* Status: proposed / accepted / deprecated / superseded
* Deciders: [@user1, @user2]
* Date: YYYY-MM-DD

Reference: (link to relevant epic, story, issue)

## Context and Problem Statement

Describe the background and why this decision is needed.

## Decision Drivers

Describe the criteria that explains why this decision has to be made.

## Considered Options

Describe single or multiple options that were considered or could be considered.

## Decision Outcome

Describe the chosen option and why it was selected.

### Implementation Steps

Describe the steps needed to implement the decision.
```

## Process

### New ADRs

1. Write a new ADR as a Markdown file.
2. Submit it via pull request for review.
3. Decision is made collaboratively, details will be discussed in the PR, which can lead to further changes.
4. Update the ADR status once a decision is reached.
5. Reference ADRs in code, documentation, or issues where relevant.

### Updating ADRs

1. If an ADR needs to be updated, create a new ADR that references the original.
2. Follow the same process as for new ADRs.
3. Once accepted, update the status of the original ADR and reference that new ADR.

## References

- [ADR GitHub Template](https://github.com/joelparkerhenderson/architecture_decision_record)
- [Wikipedia on ADRs](https://en.wikipedia.org/wiki/Architectural_decision)
