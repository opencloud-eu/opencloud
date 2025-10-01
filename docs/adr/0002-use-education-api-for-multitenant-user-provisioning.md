---
title: "Use the graph education API for multi-tenant user provisioning"
---

* Status: approved
* Deciders: [@micbar, @butonic, @rhafer]
* Date: 2025-09-23

Reference: https://github.com/opencloud-eu/opencloud/issues/877

## Context and Problem Statement

With the current multi-tenancy implementation, the user-management is mostly external
to the OpenCloud instance. Up to [now](../0001-simple-multi-tenancy-using-a-single-opencloud-instance.md)
we relied on some external LDAP server providing the users including their tenant assignment.
We'd like multi-tenancy to also work in environments where no such LDAP server is available.

## Decision Drivers

* Multi-tenancy must work without some existing external (as in not managed by us) LDAP server
* keep the implementation effort low
* allow integration with existing (de)provisioning systems

## Considered Options

### Use the auto-provisioning feature of OpenCloud

We already have basic auto-provsioning features implemented in OpenCloud.
Currently this is not tenant-aware, but it could be extended to support that.
This would require some changes in the way that the users are managed by the
auto-proviosioning code.

The auto-provisioning code does currently use the "normal" graph API to create
users. That API is not tenant-aware and would need to be significantly changed
to support multi-tenancy. However currently there is no real need to put
tenant-awareness into that API (and it would drive us even further a away from
compatibility with the MS Graph API). We could also switch away from the Graph API
for auto-provisioning and use some direct calls to the underlying LDAP server.

Also, using the auto-provisioning feature means that users are only created
when they first login. This means it is not possible to share files with users that
have not yet logged in. This is a significant limitation.

Also we don't currently have any de-provisioning features implemented.

### Use the existing Eudcation API of the Graph Service

We already implemented the Graph Education API in OpenCloud (based on the MS Graph Education API).
This, apart from the somewhat different naming, does already bring most of what is needed
for provisioning users in a multi-tenant environment.

The customer would just need to hookup their existing (de)provisioning system to call the
Education API to create/delete users and assign them to tenants (schools/classes).

The main drawback of this approach is that the customer needs to create some code to
hookup their existing system to the Education API.

The main advantage is that it would give the customer much more control over the users' lifecycle.

## Decision Outcome

Use the existing Education API of the Graph Service.

* Allows integration with existing (de)provisioning systems
* hopefully keeps the implementation effort low

Note: For now this means that the auto-provisioning feature will not be available for
multi-tenant setups. We might want to revisit this in the future.

### Implementation Steps

* re-vive the existing Education API implementation and run it as a separate service
* (maybe) allow to create tenants with a customer specified ID. The tenant id might also be
  part of the user's claims (provided by the customer's identity provider). It would be better
  if the tenant ids in our system match the tenant ids in the customer's identity provider.
* For de-provisioning to work we need to implement a way to lookup users by an external ID as
  that is only unique identfier the customer's system knows for a user. While the MS Graph API
  already provides an `externalId` Attribute we don't currently support that on our APIs.

