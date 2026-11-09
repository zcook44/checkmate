# Checkmate

A standalone terminal editor for STIG Viewer `.cklb` checklists. It reads CKLB JSON, so the same workflow applies to Cisco IOS, RHEL, and other benchmarks. It does **not** run scans, assess a host, or approve a control.

## Run locally

Requires Go to build, but not to run:

```sh
GOPROXY=off GOSUMDB=off go build -mod=vendor -o checkmate .
./checkmate --root /private/checklists --list   # read-only inventory
./checkmate --root /private/checklists          # interactive editor
```

The root must be a directory. Checkmate recursively selects the most recently modified `.cklb` **in each directory**. It refuses a tie for newest, invalid CKLB data, or symlinks in the scanned tree. Other files are ignored. It groups controls by benchmark ID, release text, group ID, and rule ID; it does not merge unrelated products or revisions.

In the TUI: `j/k` or arrows navigate controls; `Tab` chooses a host; `Space` includes/excludes that host. `v` displays the focused host's full finding details and comment; `f` cycles status filters; `/` searches IDs and titles; `e` edits. Choose a new state (`1` Open, `2` Not a Finding, `3` Not Applicable, `4` Not Reviewed), type a required comment, press Enter to inspect affected paths, then `y` to write or Esc to cancel. `q` quits. All four existing states are editable, and all matching hosts start selected—**deselect hosts that do not share the decision**. Search and comments are single-line in this first cut.

Edits replace the status and comment of each selected control in the existing CKLB. Other JSON fields are preserved, though JSON formatting is rewritten. Files are staged in the same directory, synced, and renamed over the originals; permissions are retained. The tool refuses a file whose bytes changed since loading. A multi-file edit is **not transactionally atomic**: if a later file fails, earlier files remain changed and the screen reports how many succeeded. Do not edit the same checklists concurrently or run the tool on a tree writable by an untrusted process.

**Important:** Existing `.cklb.provenance.json` sidecars are not changed; their recorded CKLB hashes become stale. The tool does not sign, schema-validate against an external CKLB schema, or establish that a bulk decision is correct for every host. Keep a separate backup before editing irreplaceable assessments.

## Disconnected installation

From the source tree on a build host with the vendored dependencies:

```sh
./scripts/package.sh 0.1.0
(cd dist && sha256sum -c SHA256SUMS)
```

Transfer the appropriate `dist/checkmate_0.1.0_linux_<arch>.tar.gz` and `dist/SHA256SUMS` to the disconnected host. Verify its archive hash there, extract it, then install the binary in a user-owned location:

```sh
sha256sum -c SHA256SUMS --ignore-missing
mkdir -p "$HOME/.local/bin"
tar -xzf checkmate_0.1.0_linux_amd64.tar.gz
install -m 755 checkmate_0.1.0_linux_amd64/checkmate "$HOME/.local/bin/checkmate"
```

The package includes the binary and this README. Build/test is offline (`GOPROXY=off GOSUMDB=off`, vendored modules, `CGO_ENABLED=0`). Release targets are Linux amd64 and arm64; packaging a target does not by itself prove it has been run on that target.

## Development

```sh
GOPROXY=off GOSUMDB=off go test -mod=vendor ./...
```

Source scans and real CKLB files belong outside this repository. Do not commit generated checklists, scan output, or host findings.
