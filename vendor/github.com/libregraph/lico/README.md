# LibreGraph Connect

LibreGraph Connect implements an [OpenID provider](http://openid.net/specs/openid-connect-core-1_0.html)
(OP) with integrated web login and consent forms.

[![Build](https://github.com/libregraph/lico/actions/workflows/build.yml/badge.svg)](https://github.com/libregraph/lico/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/libregraph/lico.svg)](https://pkg.go.dev/github.com/libregraph/lico)

LibreGraph Connect has it origin in Kopano Konnect and is meant as its vendor
agnostic successor.

## Technologies

- Go
- React

## Standards supported by Lico

Lico provides services based on open standards. To get you an idea what Lico
can do and how you could use it, this section lists what is implemented, with
the limits worth knowing before you build on it.

### OpenID Connect

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)
- [OpenID Connect Front-Channel Logout 1.0](https://openid.net/specs/openid-connect-frontchannel-1_0.html)
- [OpenID Connect Session Management 1.0](https://openid.net/specs/openid-connect-session-1_0.html)
- [OpenID Connect Dynamic Client Registration 1.0](https://openid.net/specs/openid-connect-registration-1_0.html) -
  stateless, the registration is signed into the `client_id` itself

### OAuth 2.0

- [RFC 6749 - The OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749) -
  the authorization code and refresh token grants
- [RFC 6750 - Bearer Token Usage](https://datatracker.ietf.org/doc/html/rfc6750) -
  access tokens are bearer tokens, carried in the `Authorization` header
- [RFC 7591 - Dynamic Client Registration](https://datatracker.ietf.org/doc/html/rfc7591) -
  behind `--allow-dynamic-client-registration`, without the registration
  management of RFC 7592
- [RFC 7636 - PKCE](https://datatracker.ietf.org/doc/html/rfc7636) -
  `S256` only, `plain` is refused
- [RFC 8252 - OAuth 2.0 for Native Apps](https://datatracker.ietf.org/doc/html/rfc8252) -
  loopback redirect URIs on any port, and private-use URI schemes
- [RFC 8414 - Authorization Server Metadata](https://datatracker.ietf.org/doc/html/rfc8414) -
  the same document the OpenID Connect discovery endpoint serves
- [RFC 8707 - Resource Indicators](https://www.rfc-editor.org/rfc/rfc8707.html) -
  the `resource` parameter binds an access token to the resource it is for
- [RFC 9207 - Authorization Server Issuer Identification](https://datatracker.ietf.org/doc/html/rfc9207) -
  every authorization response redirected to the client, errors included,
  carries `iss`
- [OAuth Client ID Metadata Document](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-02) -
  draft `-02`, for the hosts named with `--client-id-metadata-document-host`,
  public clients only
- [OAuth 2.0 Multiple Response Type Encoding Practices](https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html)

### JOSE and cryptography

- [RFC 7517 - JSON Web Key](https://datatracker.ietf.org/doc/html/rfc7517)
- [RFC 7519 - JSON Web Token](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 7693 - The BLAKE2 Cryptographic Hash](https://datatracker.ietf.org/doc/html/rfc7693)
- [JOSE algorithm registry](https://www.iana.org/assignments/jose/jose.xhtml)
- [NaCl secretbox](https://nacl.cr.yp.to/secretbox.html)

## Build dependencies

Make sure you have Go 1.25 or later installed, as declared in `go.mod`. This
project uses Go Modules.

Lico also includes a modern web app, which needs one additional build
dependency, assumed to be in your $PATH.

  - bun - [Bun](https://bun.com) 1.4.0 or later

Regenerating the favicon from `identifier/src/images/lico-icon.svg` with
`make -C identifier/src/images icons` additionally needs the two below. No
other target uses them, so an ordinary build does not require either.

  - magick, or convert on ImageMagick 6 - [ImageMagick](https://www.imagemagick.org)
  - rsvg-convert - [librsvg](https://gitlab.gnome.org/GNOME/librsvg)

To build Lico, a `Makefile` is provided, which requires [make](https://www.gnu.org/software/make/manual/make.html).

When building, third party dependencies will be fetched from the Internet
if not there already.

## Building from source

```
git clone <THIS-PROJECT> lico
cd lico
make
```

### Optional build dependencies

Some optional build dependencies are required for linting and continuous
integration. Those tools are mostly used by make to perform various tasks and
are expected to be found in your $PATH.

  - golangci-lint - [golangci-lint](https://github.com/golangci/golangci-lint)
  - gotestsum - [gotestsum](https://github.com/gotestyourself/gotestsum)
  - gocover-cobertura - [gocover-cobertura](https://github.com/boumenot/gocover-cobertura)

### Build with Docker

The image carries Go, Bun and the optional tools above, so nothing beyond the
container engine is needed.

```
docker build -t licod-builder -f Dockerfile.build .
docker run -it --rm -u $(id -u):$(id -g) -v $(pwd):/build licod-builder
```

With rootless podman, use `--userns=keep-id` instead of `-u`. Podman maps your
user to root inside the container, so passing your own uid lands on an
unprivileged one that cannot write the mounted tree.

```
podman build -t licod-builder -f Dockerfile.build .
podman run -it --rm --userns=keep-id -v $(pwd):/build licod-builder
```

## Running Lico

Lico can provide user login based on available backends.

All backends require certain general parameters to be present. Create a RSA
key-pair file with `openssl genpkey -algorithm RSA -out private-key.pem -pkeyopt rsa_keygen_bits:4096`
and provide the key file with the `--signing-private-key` parameter. Lico can
load PEM encoded PKCS#1 and PKCS#8 key files and JSON Web Keys from `.json` files
If you skip this, Lico will create a random non-persistent RSA key on startup.

To encrypt certain values, Lico needs a secure encryption key. Create a
suitable key of 32 bytes with `openssl rand -out encryption.key 32` and provide
the full path to that file via the `--encryption-secret` parameter. If you skip
this, Lico will generate a random key on startup.

To run a functional OpenID Connect provider, an issuer identifier is required.
The `iss` is a full qualified https:// URI pointing to the web server which
serves the requests to Lico (example: https://example.com). Provide the
Issuer Identifier with the `--iss` parameter when starting Lico.

Furthermore to allow clients to utilize the Lico services, clients need to
be known/registered. Lico reads them from a static configuration file which
allows clients and their allowed urls to be registered. See the the example at
`identifier-registration.yaml.in`. Copy and modify that file to include all
the clients which should be able to use OpenID Connect and/or OAuth2 and start
Lico with the `--identifier-registration-conf` parameter pointing to that
file. Without any explicitly registered clients, Lico will only accept clients
which redirect to an URI which starts with the value provided with the `--iss`
parameter.

A client which the operator has never heard of can also register itself, when
Lico runs with `--allow-dynamic-client-registration`. Such a registration is
stateless: it is signed into the `client_id` it returns, nothing is stored, and
every one of them has to use PKCE. A client which asks for
`token_endpoint_auth_method: none` is a public client and is issued no secret,
whether it registered itself or was written into the file. Use `--require-pkce`
to ask the same of every client, and `--dynamic-client-secret-expiration` to
expire the credentials of a dynamic registration, which by default do not
expire.

A client can also be identified by the `https` URL of a JSON document which
describes it, its Client ID Metadata Document. Lico fetches the document
instead of keeping a registration, for the hosts named with
`--client-id-metadata-document-host`, which can be given several times:
`app.example.com` allows that host as listed, `*.example.com` the subdomains of
`example.com`, and `*` any public host. A wildcard never matches an address, a
name without a dot or a name which only means something inside a local network,
such as `localhost`, `*.local`, `*.internal` or `*.home.arpa`, and no document is
ever fetched from a loopback, private or other special-use address. Such a
client is public and has to use PKCE, its non-loopback redirect URIs have to be
on the host of its `client_id`, and the consent screen names it by that host,
with the name the document claims beside it. A document is cached as its
`Cache-Control` says, between one minute and a day, and is used for up to a day
longer while its host cannot be reached.

### Protected resources

Lico can issue access tokens for an API which is not Lico itself, a
[Model Context Protocol](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization)
server for example. Such a resource is registered in the same file as the
clients:

```yaml
resources:
  - id: https://mcp.example.com/mcp
    name: Example MCP server
    scopes:
      - Example.Mcp.Tools
    implicit_scopes:
      - LibgreGraph.UUID
```

The `id` is the canonical URI of that resource, which is what a client sends as
the `resource` parameter of RFC 8707 and what ends up in the `aud` claim of the
access token, so it has to be exactly the value the resource publishes for
itself. A client is identified in `azp` instead, and a request which names no
resource is unaffected: its tokens stay bound to the client, as before.

The `scopes` are the ones that resource accepts. A request naming it is
narrowed to them, so a scope missing there never reaches the token, not even
when a client registration adds it implicitly; `openid` and `offline_access`
are exempt, being about the client rather than a permission of the resource.
The `implicit_scopes` are granted whether a client asked for them or not, for
what has to reach the resource without every client knowing about it. Every
scope named in either list must also be defined in the file given with
`--identifier-scopes-conf`, or Lico refuses to start.

The resource publishes its own
[RFC 9728](https://datatracker.ietf.org/doc/html/rfc9728) metadata, naming Lico
in `authorization_servers`, which is how a client finds its way here. Lico
neither serves nor fetches that document: it is the resource's own statement
about itself.

### Lico cryptography and validation

A tool can be used to create keys for Lico and also to validate tokens to
ensure correct operation is [Step CLI](https://github.com/smallstep/cli). This
helps since OpenSSL is not able to create or validate all of the different key
formats, ciphers and curves which are supported by Lico.

Here are some examples relevant for Lico.

```
step crypto keypair 1-rsa.pub 1-rsa.pem \
  --kty RSA --size 4096 --no-password --insecure
```

```
step crypto keypair 1-ecdsa-p-256.pub 1-ecdsa-p-256.pem \
  --kty EC --curve P-256 --no-password --insecure
```

```
step crypto jwk create 1-eddsa-ed25519.pub.json 1-eddsa-ed25519.key.json \
  -kty OKP --crv Ed25519 --no-password --insecure
```

```
echo $TOKEN_VALUE | step crypto jwt verify --iss $ISS \
  --aud playground-trusted.js --jwks $ISS/konnect/v1/jwks.json
```

### URL endpoints

Take a look at `Caddyfile.example` on the URL endpoints provided by Lico and
how to expose them through a TLS proxy.

The base URL of the frontend proxy is what will become the value of the `--iss`
parameter when starting up Lico. OIDC requires the Issuer Identifier to be
secure (https:// required).

The provider meta data is served at `/.well-known/openid-configuration` and,
identically, at `/.well-known/oauth-authorization-server`, where an OAuth 2.0
client which does not speak OpenID Connect discovery looks for it first. A
proxy which forwards by explicit path has to forward both.

### LibreGraph backend

Generic backend support is available through the LibreGraph API. Any service can
provide the required endpoints and Lico connects to them.

```
export LIBREGRAPH_URI=http://your-backend.local:5050
bin/licod serve --listen=127.0.0.1:8777 \
  --iss=https://mylico.local \
  libregraph
```

### LDAP backend

This assumes that Lico can directly connect to an LDAP server via TCP.

```
export LDAP_URI=ldap://myldap.local:389
export LDAP_BINDDN="cn=admin,dc=example,dc=local"
export LDAP_BINDPW="its-a-secret"
export LDAP_BASEDN="dc=example,dc=local"
export LDAP_SCOPE=sub
export LDAP_LOGIN_ATTRIBUTE=uid
export LDAP_EMAIL_ATTRIBUTE=mail
export LDAP_NAME_ATTRIBUTE=cn
export LDAP_UUID_ATTRIBUTE=uidNumber
export LDAP_UUID_ATTRIBUTE_TYPE=text
export LDAP_FILTER="(objectClass=organizationalPerson)"

bin/licod serve --listen=127.0.0.1:8777 \
  --iss=https://mylico.local \
  ldap
```

### Build Lico Docker image

This project includes a `Dockerfile` which can be used to build a Docker
container from the locally build version. Similarly the `Dockerfile.release`
builds the Docker image locally from the latest release download.

```
docker build -t licod .
```

```
docker build -f Dockerfile.release -t licod .
```

## API Documentation

The complete API specification is available in OpenAPI 3.0 format at [`docs/libregraph-connect-api-v1.yaml`](docs/libregraph-connect-api-v1.yaml), covering all OIDC, OAuth2, authentication, and SAML2 endpoints with detailed schemas and examples.

## Run unit tests

```
make test
```

## Development

As Lico includes a web application (identifier), a `Caddyfile.dev` file is
provided which exposes the identifier's web application directly via a
webpack dev server.

### Debugging

Lico is built stripped and without debug symbols by default. To build for
debugging, compile with additional environment variables which override/reset
build optimization like this

```
LDFLAGS="" GCFLAGS="all=-N -l" ASMFLAGS="" make cmd/licod
```

The resulting binary is not stripped and sutiable to be debugged with [Delve](https://github.com/go-delve/delve).

To connect Delve to a running Lico binary you can use the `make dlv` command.
Control its behavior via `DLV_*` environment variables. See the `Makefile` source
for details.

```
DLV_ARGS= make dlv
```

#### Remote debugging

To use remote debugging, pass additional args like this.

```
DLV_ARGS=--listen=:2345 make dlv
```

## License

See `LICENSE.txt` for licensing information of this project.
