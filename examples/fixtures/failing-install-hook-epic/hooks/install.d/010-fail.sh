#!/bin/sh
set -eu

mkdir -p runtime
printf '%s\n' '{"status":"about-to-fail","trigger":"install"}' > runtime/failure-sentinel.json
echo "intentional install hook failure" >&2
exit 17
