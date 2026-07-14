---
status: proposed
date: 2025-06-25
author: Pascal Bleser <p.bleser@opencloud.eu>
decision-makers:
consulted:
informed:
title: "Groupware Software Stack"
template: https://raw.githubusercontent.com/adr/madr/refs/tags/4.0.0/template/adr-template.md
---

* Status: draft

## Context

Which software stack to choose for the implementation of the OpenCloud Groupware service?

## Considered Options

* [Go](https://go.dev/) with the OpenCloud framework, as it is used in OpenCloud
* Rust, as it is a similarly modern language, with know-how in Opentalk
* Java with an opinionated microservice framework (e.g. [Micronaut](https://micronaut.io/))

## Decision Outcome

The decision was taken to go with the existing Go technology stack used in OpenCloud, since it allows for

* everyone in the Groupware backend team to contribute
* having a single technology stack across all OpenCloud backend features
* having the option of a single binary deployment

### Consequences

TODO

### Confirmation

TODO

## Pros and Cons of the Options

### Go

* good: established in the OpenCloud team, with expertise, potentially broadening the team that can contribute to Groupware development
* good: make use of the existing infrastructure and framework, including the single binary deployment option
* bad: less mature and capable technology stack, potentially problematic with regards to lack of asynchronous I/O and streamed HTTP processing

### Rust

* good: shared knowledge with the team of developers at OpenTalk
* bad: little to no experience in the current OpenCloud team

### Java

* bad: little to no experience in the current OpenCloud team, with exception of the Groupware members
* good: extensive experience with Micronaut with one OpenCloud developer
* good: opinionated and well documented
* good: cloud native
* good: mature technology stack
* good: asynchronous I/O and virtual threads make for efficient resource usage
* potentially bad: likely to not fit well into low resource environments (although native compilation using GraalVM is possible)
* potentially bad: prevents the single binary deployment option from including Groupware
