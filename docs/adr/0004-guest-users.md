---
title: "Inviting Guest users as fully integrated user in OpenCloud"
---

* Status: postponed
* Deciders: []
* Date: 2026-01-20

Reference: https://github.com/opencloud-eu/opencloud/issues/2111

## Important Disclaimer

The approach discussed here has been postponed (as of March, 2026) in favor
of a different solution that does not require full-blown user-accounts in
OpenCloud. That approach is currently tracked here:
https://github.com/opencloud-eu/opencloud/issues/2513

## Context and Problem statement

To allow collaboration with external Users (Users that don't yet have an
account in the IDP, and might be external to the organization), it should
be possible to invite "Guest Users" into an OpenCloud instance.

## Requirements

- the audit trail of the external user accessing the resource needs to
  be maintained, that means sharing via a password protected public link
  is not sufficient as access to that one is tracked as if the creator
  of the link accessed the resource
- external users need to be authenticated just like "normal" users, when
  accessing the shared resource (including the possibility to use 2FA)
- the ability to invite external users is tied to a separate permission
  (e.g. "can invite guest users")
- make it work with all (most) of the user-management configurations we support.
  The built-in IDP (lico) does not need to be supported.
- avoid creating "Shadow IT" Infrastructure, e.g. we don't want to
  create/maintain a separate IDP instance just for Guest User that would
  allow bypassing corporate rules for Identity Management
- the process of inviting a guest must can be asynchrounous, i.e. the user
  account of the guest user might not be created at the moment of
  creating the share/sending the invitaion as the whole process crosses
  multiple systems (OpenCloud, Identity Management System, Email) and might
  even require manual steps.
- It should be possible to "convert" a guest user into a "normal" user without
  the user loosing their shares.
- Guest user invitations should have an expiration date, after which they can
  no longer be accepted.


### Privileges of guest users

- guest users can not share or invite other users to a space or create public
  links. (primary focus of the feature is to provide a simple way to grant
  external, authorized access. anything else like resharing would undermine
  regular user accounts).
- guestusers can use the desktop and mobile client to access their shares or spaces


- all "normal" users are able to share with guest users, just as if they where "normal" users.


## Questions still to be answered

- what's the life cycle of a guest user?
  - Who's responsible for deprovisioning?
  - Do guest users expire after a certain time?
  - Do we need to keep track of who invited whom and when? (not just in
    the audit log?)
- What if the user already exists but used a different mail address in
  his account (e.g. sub-addressing?).

## Obstacles

### UserIDs

- Every user in OpenCloud needs to have a userid assigned
- Sharing, as many other features, needs that userid for storing the
  share (share service) and for assigning the grants on the shared
  resources (storage provider)
- When an external IDP is used the generation of that userid is usually
  not in control of OpenCloud (exception User-Autoprovisioning, or when
  the Provisioning/Education API is used). In that case, the userid is
  taken from a LDAP Attribute maintained in the external system

### Lots of identity management options

- OpenCloud provides many different ways to consume user-accounts. Guest
  users are supposed to be working with all/most of them:
  - External IDP, with external LDAP service
  - External IDP, with manual provisioning via the
    Education/Provisioning APIs (to a local OpenCloud specific LDAP
    service) - e.g. in multi-tenant setups
  - External IDP, with User-Autoprovisioning (also to a local OpenCloud
    specific LDAP service)
  - everything in-between and outside of the above
- Each of these options have different ways for user-provisioning and in
  the way userids are generated and managed

### How do we keep track of invitations?

- Completely rely on external system?
- Track creation and acceptance of invitations somehow?
- Do invitation expire at some point?

## Possible solutions

### Re-vitalize the PoC implementation of the invitations service and finalize it (<https://github.com/opencloud-eu/opencloud/blob/main/services/invitations/README.md>)

- Implements parts of the MSGraph Invitation Specification
  (<https://learn.microsoft.com/en-us/graph/api/resources/invitation?view=graph-rest-1.0>)
- Currently there's just a single backend that allows creating users,
  using the Keycloak Admin API
- As part of the user creation keycloak triggers an email to be sent to
  the invited user to get him to verify his email address and set a
  password. This is not really and invitation email.

#### Pros

    - A partial implementation already exists
    - no shadow IT

#### Cons

    - while the emails sent by keycloak can be themed. There is no way
      to add custom content, like: "you've being invited by user X to
      access resource Y"
    - the keycloak admin API does not return the password reset link in
      the response, so we can't use that to send a custom email
    - the keycloak implementation is not a real "user invitation"
      workflow, the user experience for the invited user is not ideal
    - The workflow likely only works with a limited set of setups.
      (Specifically: a keycloak that is able to write into a connected
      LDAP database, that OpenCloud can consume)
    - As the invitations are not really tracked, e.g. we don't really
      "know" if an invitation was accepted
    - Requires direct access to the Identity Management System

### Invitation Service + support for pending shares in the share manager

- Create some form in invitation manager and provide tools/documentation
  for customers to hook that up with their Identity Management System
- User's with the "right" privileges are able to create invitations,
  invitations get a unique identifier. Other data maintained on the
  invitation:
  - Invited user's email address
  - Invited user's userid (once the user account was provisioned)
  - Inviting user's userid
  - Creation timestamp
  - Invitation State (Pending, Accepted, …)
  - (more probably)
- our sharing API
  ('graph/v1beta1/drives/{drive-id}/items/{item-id}/invite') is enhanced
  to allow creating shares that target an invitation as the share
  recipient. (That share would only be persisted in the 'shares' service
  and would not yet crate any grants on the filesystem, or send out
  sharing notifications). (Requires changes to the CS3 sharing APIs)
- A middleware (specific to the Identity Management System) is
  "informed" (e.g. via web hooks or a message queue) when a new
  invitation is created. That middleware is responsible for provisioning
  the user account of the guest user. Whatever this process looks like
  it completely up to the middleware (maybe it triggers some invitation
  workflow or it could just even open a support ticket with the IDP
  admin)
- once the user is provisioned the middleware calls back into our
  invitations service,marks the invitation as "accepted" and provides
  the "userid" of the guest user. The invitations service then triggers
  the "pending" shares to be processes, which causes the filesystem
  grants to be written and notifications to be send out to the guest
  user.
- We'd provide a reference implementation of that middleware, that works
  with keycloak

#### Pros

    - Agnostic to whatever Identity Management System is used
    - We have an audittrail about who was invited by whom at what point
      in time
    - no shadow IT

#### Cons

    - somewhat complex
    - likely requires changes to the CS3 APIs

#### Implementation Obstacles

    - Permissions on spaces are currently not tracked in the share
      manager, the are purely managed via grants. So currently the share
      manager service currently does not know anything about (invited)
      users being assigned to spaces

## Additional thoughts

If OpenCloud were responsible for allocating the UserIDs of all users
the solution sketch above would likely loose some of its complexity. We
would "roll" the userid for the invited user already when creating the invite.
That would allow to skip the step of creating a "pending" Share with an
invitation assigned. As we have an ID already, we could just create a "normal"
share and even populate the grants on the filesystem for that share (or space)

We've been pondering on the idea of making OpenCloud manage all UserIDs
for quite a while as it would have some additional benefits for the
whole user management story.

- We wouldn't rely anymore on the external Identity Management system to
  provide a unique id with certain properties. Ideally the only unique
  thing we'd need from the external system is the `iss` and `sub`
  claims of the IDP and those are required by the OIDC standards.

It could be worth to spend some time on figuring out a migration path
towards such a solution, before spending resources on a complex guest
features implementation.
