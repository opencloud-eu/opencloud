# TODO
Currently, clients need to make subsequent calls to:
*   /status.php to check if the instance is in maintenance mode or if the version is supported
*   /config.json to get the available apps for OpenCloud web to determine which routes require authentication
*   /themes/opencloud/theme.json for theming info
*   /.well-known/openid-configuration, auth2 token and oidc userinfo endpoints to authenticate the user
*   /ocs/v1.php/cloud/user to get the username, eg. alan ... again? it contains the oc10 user id (mary, not the uuid)
*   /ocs/v1.php/cloud/capabilities to fetch instance capabilites
*   /ocs/v1.php/cloud/users/alan to fetch the quota which could come from graph and actually is now tied to the spaces, not to users
*   /graph/v1.0/me?%24expand=memberOf to fetch the user id and the groups the user is a member of

We need a way to pass oidc claims from the proxy, which does the authentication to the webfinger service, preferably by minting them into the internal reva token.
*   Currently, we use machine auth so we can autoprovision an account if it does not exist. We should use revas oidc auth and, when autoprovisioning is enabled, retry the authentication after provisioning the account. This would allow us to use a `roles` claim to decide which roles to use and eg. a `school` claim to determine a specific instance. We may use https://github.com/PerimeterX/marshmallow to parse the RegisteredClaims and get the custom claims as a separate map.

For now, webfinger can only match users based on a regex and produce a list of instances based on that.

Here are some Ideas which need to be discussed with all client teams in the future:

## Implement a Backend Lookup

We could use ldap, the graph service or a reva based authentication to look up more properties that can be used to determine which instances to list. The initial implementation works on oidc claims and does not work with basic auth.

## Replace status.php with Properties

The /.well-known/webfinger enpdoint allows us to not only get rid of some of these calls, e.g. by embedding status.php info:

```json
{
    "subject": "https://drive.opencloud.test",
    "properties": {
        "http://webfinger.opencloud/prop/maintenance": "false",
        "http://webfinger.opencloud/prop/version": "10.11.0.6"
    },
    "links": [
        {
            "rel": "http://openid.net/specs/connect/1.0/issuer",
            "href": "https://idp.opencloud.test"
        }
    ]
}
```

## Introduce Dedicated OpenCloud web Endpoint

It also allows us to move some services out of a sharded deployment. We could e.g. introduce a relation for a common OpenCloud web endpoint to not exponse the different instances in the browser bar:
```json
{
    "subject": "acct:alan@drive.opencloud.test",
    "links": [
        {
            "rel": "http://openid.net/specs/connect/1.0/issuer",
            "href": "https://idp.opencloud.test"
        },
        {
            "rel": "http://webfinger.opencloud/rel/web",
            "href": "https://drive.opencloud.test"
        },
        {
            "rel": "http://webfinger.opencloud/rel/server-instance",
            "href": "https://abc.drive.opencloud.test",
    	    "titles": {
    	      "en": "Readable Instance Name"
    	    }
        },
        {
            "rel": "http://webfinger.opencloud/rel/server-instance",
            "href": "https://xyz.drive.opencloud.test",
    	    "titles": {
    	      "en": "Readable Other Instance Name"
    	    }
        }
    ]
}
```

## Dedicated OpenCloud web Endpoint

We could also omit the `http://webfinger.opencloud/rel/server-instance` relation and go straight for a graph service with e.g. `rel=http://libregraph.org/rel/graph`:
```json
{
    "subject": "acct:alan@drive.opencloud.test",
    "links": [
        {
            "rel": "http://openid.net/specs/connect/1.0/issuer",
            "href": "https://idp.opencloud.test"
        },
        {
            "rel": "http://webfinger.opencloud/rel/web",
            "href": "https://drive.opencloud.test"
        },
        {
            "rel": "http://libregraph.org/rel/graph",
            "href": "https://abc.drive.opencloud.test/graph/v1.0",
    	    "titles": {
    	      "en": "Readable Instance Name"
    	    }
        }
    ]
}
```

In theory the graph endpoint would allow discovering drives on any domain. But there is a lot more work to be done here.

## Subject Properties

We could also embed subject metadata, however since apps like OpenCloud web also need the groups a user is member of a dedicated call to the libregraph api is probably better. In any case, we could return properties for the subject:
```json
{
    "subject": "acct:alan@drive.opencloud.test",
    "properties": {
        "http://libregraph.org/prop/user/id": "b1f74ec4-dd7e-11ef-a543-03775734d0f7",
        "http://libregraph.org/prop/user/onPremisesSamAccountName": "alan",
        "http://libregraph.org/prop/user/mail": "alan@example.org",
        "http://libregraph.org/prop/user/displayName": "Alan Turing",
    },
    "links": [
        {
            "rel": "http://openid.net/specs/connect/1.0/issuer",
            "href": "https://idp.opencloud.test"
        },
        {
            "rel": "http://webfinger.opencloud/rel/server-instance",
            "href": "https://abc.drive.opencloud.test",
    	    "titles": {
    	      "en": "Readable Instance Name"
    	    }
        },
        {
            "rel": "http://webfinger.opencloud/rel/server-instance",
            "href": "https://xyz.drive.opencloud.test",
    	    "titles": {
    	      "en": "Readable Other Instance Name"
    	    }
        },
    ]
}
```
