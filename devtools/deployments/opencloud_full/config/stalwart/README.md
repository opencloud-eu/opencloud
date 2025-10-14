# Stalwart Configuration

The mechanics are currently to mount a different configuration file depending on the environment, as we support two scenarios that are described in [`services/groupware/DEVELOPER.md`](../../../../../services/groupware/DEVELOPER.md):

 * &laquo;production&raquo; setup, with OpenLDAP and Keycloak containers
 * &laquo;homelab&raquo; setup, with the built-in IDM (LDAP) and IDP that run as part of the `opencloud` container

The Docker Compose setup (in [`stalwart.yml`](../../stalwart.yml)) mounts either [`idmldap.toml`](./idmldap.toml) or [`ldap.toml`](./ldap.toml) depending on how the variable `STALWART_AUTH_DIRECTORY` is set, which is either `idmldap` for the homelab setup, or `ldap` for the production setup.

This is thus all done automatically, but whenever changes are performed to Stalwart configuration files, they must be reflected across those two files, to keep them in sync, as the only entry that should differ is this one:

```ruby
storage.directory = "ldap"
```

or this:

```ruby
storage.directory = "idmldap"
```

