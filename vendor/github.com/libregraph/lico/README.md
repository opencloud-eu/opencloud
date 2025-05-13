# LibreGraph Connect

LibreGraph Connect implements an [OpenID provider](http://openid.net/specs/openid-connect-core-1_0.html)
(OP) with integrated web login and consent forms.

[![Go Report Card](https://goreportcard.com/badge/github.com/libregraph/lico)](https://goreportcard.com/report/github.com/libregraph/lico)

LibreGraph Connect has it origin in Kopano Konnect and is meant as its vendor
agnostic successor.

## Technologies

- Go
- React

## Standards supported by Lico

Lico provides services based on open standards. To get you an idea what
Lico can do and how you could use it, this section lists the
[OpenID Connect](https://openid.net/connect/) standards which are implemented.

- https://openid.net/specs/openid-connect-core-1_0.html
- https://openid.net/specs/openid-connect-discovery-1_0.html
- https://openid.net/specs/openid-connect-frontchannel-1_0.html
- https://openid.net/specs/openid-connect-session-1_0.html
- https://openid.net/specs/openid-connect-registration-1_0.html

Furthermore the following extensions/base specifications extend, define and
combine the implementation details.

- https://tools.ietf.org/html/rfc6749
- https://tools.ietf.org/html/rfc7517
- https://tools.ietf.org/html/rfc7519
- https://tools.ietf.org/html/rfc7636
- https://tools.ietf.org/html/rfc7693
- https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html
- https://openid.net/specs/oauth-v2-form-post-response-mode-1_0.html
- https://www.iana.org/assignments/jose/jose.xhtml
- https://nacl.cr.yp.to/secretbox.html

## Build dependencies

Make sure you have Go 1.18 or later installed. This project uses Go Modules.

Lico also includes a modern web app which requires a couple of additional
build dependencies which are furthermore also assumed to be in your $PATH.

  - yarn - [Yarn](https://yarnpkg.com)
  - convert, identify - [Imagemagick](https://www.imagemagick.org)
  - scour - [Scour](https://github.com/scour-project/scour)

To build Lico, a `Makefile` is provided, which requires [make](https://www.gnu.org/software/make/manual/make.html).

When building, third party dependencies will tried to be fetched from the Internet
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
  - go2xunit - [go2xunit](https://github.com/tebeka/go2xunit)
  - gocov - [gocov](https://github.com/axw/gocov)
  - gocov-xml - [gocov-xml](https://github.com/AlekSi/gocov-xml)
  - gocovmerge - [gocovmerge](https://github.com/wadey/gocovmerge)

### Build with Docker

```
docker build -t licod-builder -f Dockerfile.build .
docker run -it --rm -u $(id -u):$(id -g) -v $(pwd):/build licod-builder
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
Issuer Identifier with the `--iss` parametter when starting Lico.

Furthermore to allow clients to utilize the Lico services, clients need to
be known/registered. For now Lico uses a static configuration file which
allows clients and their allowed urls to be registered. See the the example at
`identifier-registration.yaml.in`. Copy and modify that file to include all
the clients which should be able to use OpenID Connect and/or OAuth2 and start
Lico with the `--identifier-registration-conf` parameter pointing to that
file. Without any explicitly registered clients, Lico will only accept clients
which redirect to an URI which starts with the value provided with the `--iss`
parameter.

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
