#!/bin/bash
set -e

if ! command -v grype &>/dev/null; then
    export PATH="$(go env GOPATH)/bin:$PATH"
    if ! command -v grype &>/dev/null; then
        go install github.com/anchore/grype/cmd/grype@latest
    fi
fi
if [[ $(find "$(go env GOPATH)/bin/grype" -mtime +7 -print) ]]; then
    echo "updating grype to @latest..."
    go install github.com/anchore/grype/cmd/grype@latest
fi

grype dir:$(pwd)
