#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:-0.3.0}"
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
printf '%s\n' '{"title":"fixture.cklb","id":"123e4567-e89b-12d3-a456-426614174000","stigs":[{"stig_name":"Cisco IOS STIG","display_name":"Cisco IOS","stig_id":"Cisco_IOS_STIG","release_info":"V1R2","uuid":"123e4567-e89b-12d3-a456-426614174001","size":1,"rules":[{"group_id":"V-42","rule_id":"SV-42r1_rule","rule_title":"Login policy","severity":"high","status":"open","comments":"","finding_details":""}]}]}' > "$fixture/sample.cklb"
release/checkmate --root "$fixture" --list | grep -q 'Cisco_IOS_STIG  V-42'
release/checkmate --root "$fixture" --validate | grep -q 'Validated 1 checklist(s)'
echo "Release directory verified"
