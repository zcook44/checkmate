# Checkmate

A terminal editor for STIG Viewer `.cklb` checklists. It reads CKLB JSON, so Cisco IOS, RHEL, and other benchmarks all get the same treatment. Apparently the paperwork is the one platform everyone agreed to standardize.

Checkmate edits checklist answers. It does **not** run scans, assess a host, approve a control, or make an incorrect answer correct by displaying it in a handsome terminal.

## Run locally

You need Go to build it. The resulting binary, sensibly, does not need Go to supervise it. The build lives in the ignored `release/` directory:

```sh
./scripts/package.sh 0.1.0
./release/checkmate --root /private/checklists --list   # read-only inventory
./release/checkmate --root /private/checklists          # interactive editor
```

Point `--root` at a directory. Checkmate walks it and selects the most recently modified `.cklb` **in each directory**. A tie for newest, invalid CKLB data, or a symlink in the tree stops the run. Guessing which assessment you meant would be a bold feature for a compliance tool. Other files are ignored.

Controls are grouped by benchmark ID, release text, group ID, and rule ID. Matching numbers from unrelated products or revisions do not earn a group photo together.

## Drive the terminal

- `j/k` or arrows: move between controls. `Tab`: focus a host. `Space`: include or exclude that host from the edit.
- `v`: read the focused host's full finding details and comment. `f`: cycle status filters. `/`: search IDs and titles.
- `e`: edit. Choose `1` Open, `2` Not a Finding, `3` Not Applicable, or `4` Not Reviewed; type a required comment; press Enter to inspect the affected paths; then `y` to write or Esc to cancel. `q` quits.

All four existing states are editable. Every matching host starts selected. **Deselect any host that does not share the decision**; a shared STIG ID is not a shared configuration. Search and comments are single-line in this first cut. The terminal has limits, despite its obvious charisma.

## What an edit actually does

Checkmate replaces the status and comment for each selected control in its existing CKLB. Other JSON fields are preserved, although the JSON formatting is rewritten. It stages each file beside the original, syncs it, retains its permissions, and renames it over the original. If the file's bytes changed since loading, Checkmate refuses the edit. A small mercy for anyone else with the same bright idea.

A multi-file edit is **not transactionally atomic**. If a later file fails, earlier files remain changed; the TUI reports how many succeeded. Do not edit the same checklists concurrently or run the tool on a tree writable by an untrusted process.

**Important:** Existing `.cklb.provenance.json` sidecars are not changed. Their recorded CKLB hashes become stale. Checkmate does not sign output, validate against an external CKLB schema, or establish that a bulk decision is correct for every host. Keep a separate backup before editing irreplaceable assessments.

## Disconnected installation

From the source tree on a build host with the vendored dependencies:

```sh
./scripts/package.sh 0.1.0
(cd release && sha256sum -c SHA256SUMS)
```

The build produces `release/checkmate` for Linux amd64, Linux amd64/arm64 archives, and `release/SHA256SUMS`. The entire `release/` directory is ignored by Git; none of those binaries is a source file. Transfer the appropriate `release/checkmate_0.1.0_linux_<arch>.tar.gz` and `release/SHA256SUMS` to the disconnected host. Verify its archive hash there, extract it, and install the binary in a user-owned location. No package repository, installation wizard, or dramatic appeal to the internet is required.

```sh
sha256sum -c SHA256SUMS --ignore-missing
mkdir -p "$HOME/.local/bin"
tar -xzf checkmate_0.1.0_linux_amd64.tar.gz
install -m 755 checkmate_0.1.0_linux_amd64/checkmate "$HOME/.local/bin/checkmate"
```

The archive contains the binary and this README. Build and test use vendored modules with `GOPROXY=off GOSUMDB=off` and `CGO_ENABLED=0`. Packages target Linux amd64 and arm64. Building an arm64 archive is not the same thing as running it on an arm64 host, however persuasive the filename looks.

## Development

```sh
GOPROXY=off GOSUMDB=off go test -mod=vendor ./...
```

Keep source scans and real CKLB files outside this repository. Do not commit generated checklists, scan output, or host findings. The repository has enough material to review without donating your infrastructure data to it.
