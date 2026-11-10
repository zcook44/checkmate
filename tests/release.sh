#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:-0.4.0}"
./scripts/package.sh "$version"
test -x release/checkmate
test -f "release/checkmate_${version}_linux_amd64.tar.gz"
test -f "release/checkmate_${version}_linux_arm64.tar.gz"
(cd release && sha256sum -c SHA256SUMS)
git check-ignore -q release/checkmate
git check-ignore -q "release/checkmate_${version}_linux_amd64.tar.gz"
cmp README.md <(tar -xOzf "release/checkmate_${version}_linux_amd64.tar.gz" "checkmate_${version}_linux_amd64/README.md")
fixture=$(mktemp -d "${TMPDIR:-/home/zrc/.hermes/cache/scratch}/checkmate-release-test.XXXXXX")
trap 'rm -r "$fixture"' EXIT
mkdir -m 700 "$fixture/checklists"
printf '%s\n' '{"title":"fixture.cklb","id":"123e4567-e89b-12d3-a456-426614174000","stigs":[{"stig_name":"Cisco IOS STIG","display_name":"Cisco IOS","stig_id":"Cisco_IOS_STIG","release_info":"V1R2","uuid":"123e4567-e89b-12d3-a456-426614174001","size":1,"rules":[{"group_id":"V-42","rule_id":"SV-42r1_rule","rule_title":"Login policy","severity":"high","status":"open","comments":"","finding_details":""}]}]}' > "$fixture/checklists/sample.cklb"
printf '%s\n' '{"$schema":"https://json-schema.org/draft/2019-09/schema","type":"object","required":["title"],"properties":{"title":{"type":"string"}}}' > "$fixture/schema.json"
release/checkmate --root "$fixture/checklists" --list | grep -q 'Cisco_IOS_STIG  V-42'
if release/checkmate --root "$fixture/checklists" --validate >/dev/null 2>&1; then echo '--validate unexpectedly accepted no schema' >&2; exit 1; fi
release/checkmate --root "$fixture/checklists" --validate --schema "$fixture/schema.json" | grep -q 'Validated 1 checklist(s) against supplied schema'
if tar -tzf "release/checkmate_${version}_linux_amd64.tar.gz" | grep -q 'SV3_CKLB_1_0_JSON_SCHEMA'; then echo 'schema shipped in archive' >&2; exit 1; fi
echo "Release directory verified"
