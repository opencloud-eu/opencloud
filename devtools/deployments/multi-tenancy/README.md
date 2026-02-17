# Development/Test Deployment for a multi-tenacy setup

The docker compose files in this directory are derived from the
opencloud-compose project and can be used to deploy a Development or Testing
environment for a multi-tenancy setup of OpenCloud. It consists of the
following services:

* `provisioning`: The OpenCloud graph service deployed in a standalone mode. It
  is configured to provide the libregraph education API for managing tenants
  and users. The `ldap-server`service (see below) is used to store the tenants
  and users.
* `ldap-server`: An OpenLDAP server that is used by the provisioning service to
  store tenants and users. Used by the OpenCloud services as the user directory
  (for looking up users and searching for sharees).
* `keycloak`: The OpenID Connect Provider used for authenticating users. The
  pre-loaded realm is configured to add `tenantid` claim into the identity and
  access tokens. It's also currently consuming user from the `ldap-server`
  (this federation will likely go away in the future and is optional for future
  configurations).
* `opencloud`: The OpenCloud configured so that is hides users from different
  tenants from each other.

To deploy the setup, run:

```bash
docker compose -f docker-compose.yml -f keycloak.yml -f ldap-server.yml -f traefik.yml up
```

Once deployed you can use the `initialize_users.go` to create a couple of example
tenants and some users in each tenant:

* Tenant `Famous Coders` with users `dennis` and `grace`
* Tenant `Scientists` with users `einstein` and `marie`

The passwords for the users is set to `demo` in keycloak

```
> go run initialize_users.go
Created tenant: Famous Coders with id fc58e19a-3a2a-4afc-90ec-8f94986db340
Created user: Dennis Ritchie with id ee1e14e7-b00b-4eec-8b03-a6bf0e29c77c
Created user: Grace Hopper with id a29f3afd-e4a3-4552-91e8-cc99e26bffce
Created tenant: Scientists with id 18406c53-e2d6-4e83-98b6-a55880eef195
Created user: Albert Einstein with id 12023d37-d6ce-4f19-a318-b70866f265ba
Created user: Marie Curie with id 30c3c825-c37d-4e85-8195-0142e4884872
Setting password for user: grace
Setting password for user: marie
Setting password for user: dennis
Setting password for user: einstein
```
