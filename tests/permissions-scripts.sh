#!/usr/bin/env bash
set -euo pipefail
repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
tmp=$(mktemp -d "${TMPDIR:-$HOME}/checkmate-script-test.XXXXXXXX")
trap 'rm -rf -- "$tmp"' EXIT
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
mode() { stat -c %a -- "$1"; }
assert_mode() { [[ $(mode "$1") == "$2" ]] || fail "mode $1 expected $2, got $(mode "$1")"; }
source_dir="$tmp/source archives"
mkdir -p "$source_dir"
# Synthetic DISA-style ZIP: extract the one manual XML, not the other files.
mkdir -p "$tmp/input/U_RHEL_9_V2R10_Manual_STIG"
printf '<Benchmark/>\n' > "$tmp/input/U_RHEL_9_V2R10_Manual_STIG/U_RHEL_9_STIG_V2R10_Manual-xccdf.xml"
printf 'not a benchmark\n' > "$tmp/input/readme.txt"
(cd "$tmp/input" && zip -q -r "$source_dir/U_RHEL_9_V2R10_STIG.zip" .)
mkdir -p "$tmp/lib parent"
bash "$repo/scripts/unpack-stig-library.sh" "$source_dir" "$tmp/lib parent/stig_library"
xml="$tmp/lib parent/stig_library/U_RHEL_9_STIG_V2R10_Manual-xccdf.xml"
[[ -f "$xml" ]] || fail 'missing extracted benchmark'
cmp "$xml" "$tmp/input/U_RHEL_9_V2R10_Manual_STIG/U_RHEL_9_STIG_V2R10_Manual-xccdf.xml" || fail 'content mismatch'
assert_mode "$tmp/lib parent/stig_library" 700
assert_mode "$xml" 600
[[ ! -e "$tmp/lib parent/stig_library/readme.txt" ]] || fail 'extracted unrelated file'
if bash "$repo/scripts/unpack-stig-library.sh" "$source_dir" "$tmp/lib parent/stig_library"; then fail 'overwrote destination'; fi
# Detect duplicate official filenames before promoting any files.
cp "$source_dir/U_RHEL_9_V2R10_STIG.zip" "$source_dir/U_RHEL_9_V2R10_STIG-copy.zip"
if bash "$repo/scripts/unpack-stig-library.sh" "$source_dir" "$tmp/duplicate"; then fail 'accepted duplicate benchmark'; fi
[[ ! -e "$tmp/duplicate" ]] || fail 'created destination on failed preflight'
rm "$source_dir/U_RHEL_9_V2R10_STIG-copy.zip"
# A ZIP with no matching manual XCCDF must not create output.
(cd "$tmp/input" && zip -q "$source_dir/irrelevant.zip" readme.txt)
if bash "$repo/scripts/unpack-stig-library.sh" "$source_dir/irrelevant.zip" "$tmp/irrelevant"; then fail 'accepted unrelated ZIP'; fi
[[ ! -e "$tmp/irrelevant" ]] || fail 'created destination for irrelevant ZIP'
ln -s "$source_dir/U_RHEL_9_V2R10_STIG.zip" "$tmp/link.zip"
if bash "$repo/scripts/unpack-stig-library.sh" "$tmp/link.zip" "$tmp/linked"; then fail 'accepted symlink archive'; fi
mkdir -m 777 "$tmp/writable parent"
chmod 777 "$tmp/writable parent"
if bash "$repo/scripts/unpack-stig-library.sh" "$source_dir/U_RHEL_9_V2R10_STIG.zip" "$tmp/writable parent/lib"; then fail 'accepted writable destination parent'; fi
[[ ! -e "$tmp/writable parent/lib" ]] || fail 'created library in writable parent'
# Permissions repair: inspect first, then apply; preserve data and ownership.
root="$tmp/scan results"
mkdir -p "$root/host one/subdir"
printf 'xccdf\n' > "$root/host one/scan.results.xml"
printf 'arf\n' > "$root/host one/scan.results.arf.xml"
printf 'evidence\n' > "$root/host one/subdir/report.txt"
chmod 755 "$root" "$root/host one" "$root/host one/subdir"
chmod 644 "$root/host one/scan.results.xml" "$root/host one/scan.results.arf.xml" "$root/host one/subdir/report.txt"
bash "$repo/scripts/fix-oscap-permissions.sh" "$root"
assert_mode "$root" 755
assert_mode "$root/host one/scan.results.xml" 644
bash "$repo/scripts/fix-oscap-permissions.sh" --apply "$root"
assert_mode "$root" 700
assert_mode "$root/host one" 700
assert_mode "$root/host one/subdir" 700
assert_mode "$root/host one/scan.results.xml" 600
assert_mode "$root/host one/scan.results.arf.xml" 600
assert_mode "$root/host one/subdir/report.txt" 600
[[ $(<"$root/host one/scan.results.xml") == xccdf ]] || fail 'modified contents'
# Refuse symlinks before changing anything in the requested tree.
chmod 755 "$root"
ln -s /etc/passwd "$root/host one/surprise"
if bash "$repo/scripts/fix-oscap-permissions.sh" --apply "$root"; then fail 'accepted symlink in results tree'; fi
assert_mode "$root" 755
rm "$root/host one/surprise"
ln "$root/host one/scan.results.xml" "$root/host one/external-hardlink"
if bash "$repo/scripts/fix-oscap-permissions.sh" --apply "$root"; then fail 'accepted multiply-linked result'; fi
assert_mode "$root" 755
rm "$root/host one/external-hardlink"
if bash "$repo/scripts/fix-oscap-permissions.sh" --apply "$tmp/lib parent"; then fail 'accepted tree without OSCAP results'; fi
printf 'PASS: library extraction and results permissions\n'
