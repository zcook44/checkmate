#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:?usage: scripts/package.sh VERSION}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'version must be X.Y.Z' >&2; exit 2; }
export GOPROXY=off GOSUMDB=off CGO_ENABLED=0
mkdir -p dist
go test -mod=vendor ./...
: > dist/SHA256SUMS
for arch in amd64 arm64; do
  name="checkmate_${version}_linux_${arch}"
  mkdir -p "dist/$name"
  GOOS=linux GOARCH="$arch" go build -mod=vendor -trimpath -ldflags='-s -w' -o "dist/$name/checkmate" .
  cp README.md "dist/$name/README.md"
  tar -C dist -czf "dist/$name.tar.gz" "$name"
  (cd dist && sha256sum "$name.tar.gz") >> dist/SHA256SUMS
  rm -r "dist/$name"
done
(cd dist && sha256sum -c SHA256SUMS)
