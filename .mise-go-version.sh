#!/bin/bash
set -euo pipefail

# Purpose of this script is to parse the go.mod file to retrieve the version of
# Go that we are using, either from a "toolchain" directive (preferred), or from
# a "go" directive as a fallback.
#
# The script then outputs that version, where it will be picked up as an environment
# variable in mise.

F="./go.mod"

AWK=awk
# prefer 'gawk' over 'awk' to make sure we get GNU awk
command -v gawk &>/dev/null && AWK=gawk

[[ -e $F ]] || { echo "ERROR: failed to find $F" >&2; exit 1;}

GO_VERSION=$("$AWK" '/^toolchain/ {print $2; exit} /^go / {print $2}' "$F")
GO_VERSION=${GO_VERSION#go} # strip the potential 'go' prefix:
echo "${GO_VERSION}"
