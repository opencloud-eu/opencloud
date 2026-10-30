---
status: proposed
date: 2025-06-24
author: Pascal Bleser <p.bleser@opencloud.eu>
decision-makers:
consulted:
informed:
title: "Implementing Groupware as a separate Microservice vs integrated in the OpenCloud Stack"
template: https://raw.githubusercontent.com/adr/madr/refs/tags/4.0.0/template/adr-template.md
---

* Status: draft

## Context

Should the Groupware backend be an independent microservice or be part of the OpenCloud single binary framework?

The OpenCloud backend is built on a framework that

* implements token based authentication between services
* allows for a "single binary" deployment mode that runs all services within that one binary
* integrates services such as a NATS event bus

This decision is about whether the Groupware backend service should be implemented within that framework or, instead, be implemented as a standalone backend service.

## Decision Drivers

* single binary deployment strategy is potentially important (TODO how important is it really? stakeholders:?)

## Considered Options

* have the Groupware Middleware as an independent microservice
* have the Groupware Middleware implemented within the existing OpenCloud framework

## Decision Outcome

TODO

### Consequences

TODO

### Confirmation

TODO

## Pros and Cons of the Options

### Independent Microservice

* (potentially) good: be free from technical decisions made for the existing OpenCloud stack, to avoid carrying potential technical baggage
* (potentially) good: make use of a framework that is more fitting for the tasks the Groupware backend needs to accomplish
* bad: re-implement framework components that already exist, with the need to maintain those in two separate codebases, or the added complexity of a shared library repository
* bad: not have the ability to include the Groupware backend in the single binary deployment
* neutral: a separate code repository and delivery for the Groupware backend, which might or might not be of advantage
* neutral: may be implemented on a completely different technology stack, including the programming language

### Part of the framework

* good: fit into the opinionated choices that were made for the OpenCloud framework so far
* good: many aspects are already implemented in the current framework and can be made use of, potentially enhanced for the needs of the Groupware backend
* good: the ability to include the Groupware backend in the single binary deployment
* neutral: be in the same code repository and part of the same delivery as other services in OpenCloud
* neutral: must be implemented in Go on top of the same technology stack
