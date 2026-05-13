# IDM

The IDM service provides a minimal LDAP Service, based on [Libregraph idm](https://github.com/owncloud/ocis/tree/master/services/idm), for OpenCloud. It is started as part of the default configuration and serves as a central place for storing user and group information.

It is mainly targeted at small OpenCloud installations. For larger setups it is recommended to replace IDM with a “real” LDAP server or to switch to an external identity management solution.

IDM listens on port 9235 by default. In the default configuration it only accepts TLS-protected connections (LDAPS). The BaseDN of the LDAP tree is `o=libregraph-idm`. IDM gives LDAP write permissions to a single user (DN: `uid=libregraph,ou=sysusers,o=libregraph-idm`). Any other authenticated user has read-only access. IDM stores its data in a boltdb file `idm/idm.boltdb` inside the OpenCloud base data directory.

The internal LDAP certificate and key are stored as `ldap.crt` and `ldap.key` in the IDM data directory. By default, these certificates expire after 12 months. When the certificate has expired, IDM can no longer establish valid TLS connections and requests that depend on LDAP may fail with `500 Internal Server Error`.

To renew the internal LDAP certificate, stop or restart the OpenCloud container after deleting the expired certificate and key:

```bash
cd .opencloud/idm
rm ldap.crt ldap.key
docker compose restart
```

The certificate and key are automatically regenerated when the container starts again. For more details, see [Internal LibreIDM cert expires](https://docs.opencloud.eu/docs/admin/resources/common-issues/#internal-libreidm-cert-expires).

Note: IDM is limited in its functionality. It only supports a subset of the LDAP operations (namely `BIND`, `SEARCH`, `ADD`, `MODIFY`, `DELETE`). Also, IDM currently does not do any schema verification (like. structural vs. auxiliary object classes, require and option attributes, syntax checks, …). Therefore it is not meant as a general purpose LDAP server.
