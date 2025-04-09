---
title: 'Conventions'
sidebar_position: 1
id: conventions
---



This is a collection of tips and conventions to follow when working on the [OpenCloud Web frontend](https://github.com/opencloud-eu/web).
Since it is a living document, please open a PR if you find something missing.

## Contributing to OpenCloud Web

Everyone is invited to contribute. Simply fork the [codebase](https://github.com/opencloud-eu/web/),
check the [issues](https://github.com/opencloud-eu/web/issues?q=is%3Aopen%20is%3Aissue%20label%3AType%3AGood-First-Issue)
for a suitable one and open a pull request!

### Linting and Tests

To make sure your pull request can be efficiently reviewed and won't need a lot of changes down the road, please run the linter and
the unit tests via `pnpm lint --fix` and `pnpm test:unit` locally. Our [CI](https://drone.opencloud.eu/opencloud/web) will run on
pull requests and report back any problems after that. For a further introduction on how we handle testing, please head to
the [testing docs](./../30-testing/index.md).


## Code Conventions

### Early Returns

We're trying to stick with early returns in our code to make it more performant and simpler to reason about it.

### Translations

Use the `v-text` directive in combination with `$gettext` (or a variation of it) inside HTML tags (instead of
a `<translate tag="h1">` or similar) in order to make reasoning about the DOM tree easier.

### TypeScript

We're using TypeScript, which allows us to catch bugs at transpile time. Clean types make sure our IDEs can support us
in reasoning about our (ever growing, complex) codebase.

### Vue 3 and Composition API

We've migrated from Vue 2 to Vue 3 late in 2022 and since then have been investing continuous efforts to move away from the Vue options API
in favor of the Vue composition API. The `web-pkg` helper package provides quite some composables which will help you in
app & extension development, so we encourage you to make use of the Vue composition API as well, even outside of the
OpenCloud Web repository.

### Dependencies

To keep the bundle size small and reduce the risk of introducing security problems for our users, we try to limit
the amount of dependencies in our code base and keep them as up-to-date as possible.
