#!/bin/bash

set -euo pipefail

#
# Quick and dirty quickstart script to fire up a local OpenCloud instance.
# Klaas Freitag <k.freitag@opencloud.eu>
#

# This script supports the following environment variables:
#   OC_VERSION: Version to download, e.g. OC_VERSION="1.2.0"

# Call this script directly from opencloud:
# curl -L https://opencloud.eu/quickinstall.sh | /bin/bash

# This function is borrowed from openSUSEs /usr/bin/old, thanks.
function backup_file () {
    local DATESTRING=`date +"%Y%m%d"`

    i=${1%%/}
    if [ -e "$i" ] ; then
        local NEWNAME=$i-$DATESTRING
        local NUMBER=0
        while [ -e "$NEWNAME" ] ; do
            NEWNAME=$i-$DATESTRING-$NUMBER
            let NUMBER=$NUMBER+1
        done
        echo moving "$i" to "$NEWNAME"
        if [ "${i:0:1}" = "-" ] ; then
            i="./$i"
            NEWNAME="./$NEWNAME"
        fi
        mv "$i" "$NEWNAME"
    fi
}

function get_latest_version() {
     latest_version=$(curl -s https://api.github.com/repos/opencloud-eu/opencloud/releases/latest \
	     | grep '"tag_name":' \
	     | awk -F: '{print $2}' \
         | tr -d ' ",v')
}

# URL pattern of the download file
# https://github.com/opencloud-eu/opencloud/releases/download/v1.0.0/opencloud-1.0.0-linux-amd64

get_latest_version

dlversion="${OC_VERSION:-$latest_version}"
dlurl="https://github.com/opencloud-eu/opencloud/releases/download/v${dlversion}/"

sandbox="opencloud-sandbox-${dlversion}"

# Create a sandbox
[ -d "./${sandbox}" ] && backup_file ${sandbox}
mkdir ${sandbox} && cd ${sandbox}

# The operating system
os="linux"

if [[ $OSTYPE == 'darwin'* ]]; then
  os="darwin"
fi

# The platform
dlarch="amd64"

if [[ $(uname -s) == "Darwin" && $(uname -m) == "arm64" ]]; then
    dlarch="arm64"
fi

# ...results in the download file
dlfile="opencloud-${dlversion}-${os}-${dlarch}"

# download
echo "Downloading ${dlurl}/${dlfile}"

curl -L -o "${dlfile}" --progress-bar "${dlurl}/${dlfile}"
chmod 755 ${dlfile}

basedir="${OC_BASE_DIR:-$(pwd)}"
export OC_CONFIG_DIR="$basedir/config"
export OC_BASE_DATA_PATH="$basedir/data"
mkdir -p "$OC_CONFIG_DIR" "$OC_BASE_DATA_PATH"

# It is bound to localhost for now to deal with non existing routes
# to certain host names for example in WSL
host="${OC_HOST:-localhost}"

./${dlfile} init --insecure yes --ap admin

echo '#!/bin/bash
SCRIPT_DIR="$(dirname "$(readlink -f "${0}")")"
cd "${SCRIPT_DIR}"' > runopencloud.sh

echo "export OC_CONFIG_DIR=${OC_CONFIG_DIR}
export OC_BASE_DATA_PATH=${OC_BASE_DATA_PATH}

export OC_INSECURE=true
export OC_URL=https://${host}:9200
export IDM_CREATE_DEMO_USERS=true
export PROXY_ENABLE_BASIC_AUTH=true
export OC_LOG_LEVEL=warning

./"${dlfile}" server
" >> runopencloud.sh

chmod 755 runopencloud.sh

echo "Connect to OpenCloud via https://${host}:9200"
echo ""
echo "*** This is a fragile test setup, not suitable for production! ***"
echo "    If you stop this script now, you can run your test OpenCloud again"
echo "    using the script ${sandbox}/runopencloud.sh"
echo ""

./runopencloud.sh

