#!/bin/sh
set -eu

mkdir -p runtime
payload_file="runtime/install-hook-output.json"
cat > "$payload_file"
