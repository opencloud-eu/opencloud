# Simple Bare Metal Install

This install script downloads and installs the OpenCloud binary and
configures it in a sandbox directory in the folder where you called
the install script. It also adds a start script called `runopencloud.sh`
to start OpenCloud later.

The installation only consists of the bare minimum functionality 
without web office and other optional components. Also, it is bound
to localhost and has no valid certificates. ** It is only
useful for simple test- and demo cases and not for production.**

To use OpenCloud, start it with the start script and head your
browser to https://localhost:9200. The invalid certificate  must
be acknowledged in the browser.

The demo users (eg. alan / demo) are enabled, the admin password
is surprisingly `admin`.

This script should **NOT** be run as user root.

# Options

## Version

Set the environment variable `OC_VERSION` to the version you want
to download. If not set, there is a reasonable default. 

## Data Location

Set the environment variable `OC_BASE_DIR` to a directory where the
`data` and `config` subdirectories shall be located. Per default,
both configuration and storage data are within a sandbox subdirectory
in the current working directory.

## Server Address

Set the environment variable `OC_HOST` to the fully qualified hostname
of this server to allow remote accesse. Default: `localhost`.

# Example

Call

```
OC_VERSION="2.0.0" ./install.sh
``` 
to install the OpenCloud version 2.0.0

There is also a hosted version of this script that makes it even
easier:

```
curl -L https://opencloud.eu/install.sh | bash -x 
```

