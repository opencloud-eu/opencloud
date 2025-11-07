#!/bin/bash
# tests/acceptance/scripts/generate-virus-files.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/../filesForUpload/filesWithVirus"

echo "Generating EICAR test files..."

mkdir -p "$TARGET_DIR"

cd "$TARGET_DIR"

echo "Downloading eicar.com..."
curl -s -o eicar.com https://secure.eicar.org/eicar.com

echo "Downloading eicar_com.zip..."
curl -s -o eicar_com.zip https://secure.eicar.org/eicar_com.zip