# Auth-App

The auth-app service provides authentication for 3rd party apps unable to use
OpenID Connect. The service is enabled by default and started automatically. It
is possible to disable the service by setting:

```bash
OC_EXCLUDE_RUN_SERVICES=auth-app # deployment specific. Removes service from the list of automatically started services, use with single-binary deployments
PROXY_ENABLE_APP_AUTH=false      # mandatory, disables app authentication. In case of a distributed environment, this envvar needs to be set in the proxy service.
```

## App Tokens

App Tokens are password specifically generated to be used by 3rd party applications
for authentication when accessing the OpenCloud API endpoints. To
be able to use an app token, one must first create a token. There are different
options of creating a token.

## Important Security Note

When using an external IDP for authentication, App Token are NOT invalidated
when the user is disabled or locked in that external IDP. That means the user
will still be able to use its existing App Tokens for authentication for as
long as the App Tokes are valid.

## Managing App Tokens

### Via API

Please note: This API is preliminary. In the future we will provide endpoints
in the `graph` service for allowing the management of App Tokens.

The `auth-app` service provides an API to create (POST), list (GET) and delete (DELETE) tokens at the `/auth-app/tokens` endpoint.

* **Create a token**\
  The POST request requires:
  * A `expiry` key/value pair in the form of `expiry=<number><h|m|s>`\
    Example: `expiry=72h`
  ```bash
  curl --request POST 'https://<your host:9200>/auth-app/tokens?expiry={value}' \
       --header 'accept: application/json'
  ```
  Example output:
  ```
  {
  "token": "3s2K7816M4vuSpd5",
  "expiration_date": "2024-08-08T13:42:42.796888022+02:00",
  "created_date": "2024-08-07T13:42:42+02:00",
  "label": "Generated via API"
  }
  ```

  Note, that this is the only time the app token will be returned in cleartext. To use the token
  please copy it from the response.

* **List tokens**\
  ```bash
  curl --request GET 'https://<your host:9200>/auth-app/tokens' \
       --header 'accept: application/json'
  ```

  Note that the `token` value in the response to the "List Tokens` request is not the actual
  app token, but the UUID of the token. So this value cannot be used for authenticating
  with the token.

  Example output:
  ```
  [
    {
      "token": "155f402e-1c5c-411c-92d4-92f3b612cd99"
      "expiration_date": "2024-08-08T13:44:31.025199075+02:00",
      "created_date": "2024-08-07T13:44:31+02:00",
      "label": "Generated via Impersonation API"
    },
    {
      "token": "8c606bdb-e22e-4094-9304-732fd4702bc9"
      "expiration_date": "2024-08-08T13:46:41.936052281+02:00",
      "created_date": "2024-08-07T13:46:42+02:00",
      "label": "Generated via Impersonation API"
    }
  ]
  ```

* **Delete a token**\
  The DELETE request requires:
  * A `token` key/value pair in the form of `token=<token_issued>`. The value needs to be the hashed value as returned by the `List Tokens` respone.\
    Example: `token=8c606bdb-e22e-4094-9304-732fd4702bc9`
  ```bash
  curl --request DELETE 'https://<your host:9200>/auth-app/tokens?token={value}' \
       --header 'accept: application/json'
  ```

### Via Impersonation API

When setting the environment variable `AUTH_APP_ENABLE_IMPERSONATION` to
`true`, admins will be able to use the `/auth-app/tokens` endpoint to create
tokens for other users. This can be important for migration scenarios, but
should not be considered for regular tasks on a production system for security
reasons.

To impersonate, the respective requests from the CLI commands above extend with
the following parameters, where you can use one or the other:

* The `userID` in the form of: `userID={value}`\
  Example:\
  `userID=4c510ada- ... -42cdf82c3d51`

* The `userName` in the form of: `userName={value}`\
  Example:\
  `userName=alan`

Example:\
A final create request would then look like:
```bash
curl --request POST 'https://<your host:9200>/auth-app/tokens?expiry={value}&userName={value}' \
     --header 'accept: application/json'
```

### Via CLI (developer only)

As the CLI is using the internal CS3Apis this needs access to the reva gateway
service. This is mainly of developer (and admin) usage.
Replace the `user-name` with an existing user. For the `token-expiration`, you
can use any time abbreviation from the following list: `h, m, s`. Examples:
`72h` or `1h` or `1m` or `1s.` Default is `72h`.

```bash
opencloud auth-app create --user-name={user-name} --expiration={token-expiration}
```

## Authenticating using App Tokens

To autenticate using an App Token simply use the username for which token was generated
and the token value as returned by the "Create Token" request.

```bash
curl -u <username>:<tokenvalue> 'https://<your host>/graph/v1.0/me' \
     --header 'accept: application/json'
```
