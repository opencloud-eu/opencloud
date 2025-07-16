---
title: "1. Simple Multi-Tenancy using a single OpenCloud Instance"
---

* Status: proposed
* Deciders: [@micbar @butonic @dragotin @rhafer]
* Date: 2025-05-20

Technical Story: https://github.com/opencloud-eu/opencloud/issues/877

## Context and Problem Statement

To reduce resource usage and cost service providers want a single OpenCloud
instance to host multiple tenants. Members of the same tenant should be able to
only see each other when trying to share resources. A user can only be a member
of a single tenant. Moving a user from one tenant to another is not supported.

OpenCloud does currently not have any concept of multi-tenancy. All users able to
login to an OpenCloud instance are able to see each other and share resources with
everybody. This ADR is supposed to layout a concept for a minimal multi-tenancy
solution that implements the characteristics mentioned above.

To further limit the scope there are a couple of constraints:

- Tenants are rather small (sometimes just a single user, often less than 10)
- There is just a single IDP with a single "realm".
- The user-management is external to OpenCloud
- The membership of a user to a tenant is represented by a tenant id
  that is provided via a claim in the users' Access Token/UserInfo or by a per User
  Attribute in the LDAP directory.
- There is no need to support per tenant groups
- There is no need to isolate the storage of the tenants from each other
- Role Assignment happens at first login via OIDC claims

## Decision Drivers

* Low Resource Overhead: The solution should not require much additional
  resources (CPU, Memory, Storage) per tenant on the OpenCloud instance.
* Implementation effort: The solution should be easy to implement and maintain.
* Security: The solution should prevent users from seeing or accessing anything
  from other tenants.

## Considered Options

### Option 1: Tenant ID as a new property of the CS3 UserId

The CS3 UserId (https://buf.build/cs3org-buf/cs3apis/docs/main:cs3.identity.user.v1beta1#cs3.identity.user.v1beta1.UserId)
is extended by a new property "tenantId".

#### Pros:

* Everywhere the UserID is used the tenant id is also available.
  * This might allow implementing more sophisticated checks e.g. on permission
    grants and to a certain extend during share creation.
  * the tenant id is immediately available e.g. in Events/Auditlog without an additional
    user lookup

#### Cons:

* Requires changes to the CS3 API
* Adds even more semantics to the UserId. Ideally the UserID would just be an
  opaque identifier without carrying and specific semantics other than being
  globally unique.
* on the GraphAPI the ID of a User is just a opaque string without any additional
  meaning. (Currently it is just using the `OpaqueId` property of the CS3 UserId,
  without considering the `idp` property.)

### Option 2: Tenant ID is stored as the `idp` value of the CS3 UserId

Instead of introducing a new property the on the CS3 UserId we'll just override
the `idp` value with the tenant id.

#### Pros

* No changes to the CS3 API required
* The pros of Option 1 apply here as well

#### Cons:

* It's a crutch, we're already "abusing" the `idp` property of the CS3 UserId
  to have a different meaning in the context of federated sharing. Adding an
  additiona meaning could make the code even more complicated.
* Apart from the API change the Cons of Option 1 apply here as well.

### Option 3: Tenant ID is a property of the CS3 User Object

A new (optional) property "tenantId" is added to the CS3 User Object.

#### Pros:

* Avoid overloading CS3 UserId with additional semantics.
* The tenant id is available everywhere the User Object is used.

#### Cons:

* Requires changes to the CS3 API
* Might require more user lookups in places where we need to find out the
  tenant id of a specific user

### Option 4: Tenant ID is invisible to the CS3 API

Reva Tokens would get a new property `tenantId`. To have the tenant id available
of the signed in user available with every request.
While not being part of the CS3 API objects the `users` service will be made "tenant aware"
so that is only returns users of the same tenant as the requesting user e.g. by using
proper LDAP filters or subtree searches.

#### Pros:

* No changes to the CS3 API required
* Code changes could be limited to the `users` and`share-provider` services and the reva token manager
* The tenant id of the current user is available everywhere the reva token is used.

#### Cons:

* Can this work with the App Token feature if the Tenant Id is not part of the User Object in
  any way, how can the the token manager know with Id to add to the token?
* What about places where the system user is doing stuff on behalf of a user

### Option 5: Tenant membership via LDAP group membership

All members of a tenant are assigned to the same group. The `users` service gets a config
switch to only allow users to search for users that are part of the same group.

#### Pros:

* ?

#### Cons:

* The group needs to be hidden somehow from the `groups` service as this is not supposed to be a
  "sharing group".

## Decision Outcome

### Chosen option: Option 1 - Tenant ID as a new property of the CS3 UserId

As part of the OIDC Connect Authentication OpenCloud will receive a tenant id via the
Access Token or Userinfo endpoint. For the initial implementation it is assume that OpenCloud
has read access to a shared LDAP server, that contains all users of all tenants. Users in
LDAP will have a the tenant id as an dedicated attribute.

### Implementation Steps

* Extend the CS3 UserId with a new property "tenantId"
* Adapt the `users` service` to only return users that are part of the same tenant
* To cleanup technical debt and reduce code duplication the `cs3` users backend in the
  graph service is being improved so that is can fully replace the `LDAP` users backend for
  read operations.
* The reva `share-provider` service only allow shares between users of the same tenant

