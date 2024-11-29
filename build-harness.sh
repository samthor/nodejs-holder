#!/bin/bash

# Builds the harness JS and puts it inside the Go source.

set -eu

esbuild --bundle --format=esm harness/main.ts --outfile=dist/harness.js --platform=node

HARNESS=$(cat dist/harness.js)

echo "package lib

const (
  jsHarness = \`
${HARNESS}
  \`
)
" > lib/harness_code.go

echo "OK in lib/harness_code.go"
