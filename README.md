# Checkmate

A terminal editor for STIG Viewer `.cklb` checklists. It reads CKLB JSON, so Cisco IOS, RHEL, and other benchmarks all get the same treatment. Apparently the paperwork is the one platform everyone agreed to standardize.

Checkmate edits checklist answers. It does **not** run scans, assess a host, approve a control, or make an incorrect answer correct by displaying it in a handsome terminal.

## Run locally

You need Go to build it. The resulting binary, sensibly, does not need Go to supervise it. The build lives in the ignored `release/` directory:

```sh
./scripts/package.sh 0.3.0
./release/checkmate --root /private/checklists --list   # read-only inventory
./release/checkmate --root /private/checklists --validate # official CKLB 1.0 schema check
./release/checkmate --root /private/checklists          # interactive editor
```

Point `--root` at a directory. Checkmate walks it and selects the most recently modified `.cklb` **in each directory**. A tie for newest, invalid CKLB data, or a symlink in the tree stops the run. Guessing which assessment you meant would be a bold feature for a compliance tool. Other files are ignored.

Controls are grouped by benchmark ID, release text, group ID, and rule ID. Matching numbers from unrelated products or revisions do not earn a group photo together.

## Drive the terminal

- `j/k` or arrows: move between controls. `Tab`: focus a host. `Space`: include or exclude that host from the edit. Nothing starts selected; Checkmate has resisted the temptation to decide for your entire fleet.
- `PgUp/PgDn`: move through affected hosts a page at a time. `Home/End`: jump to the first or last host; `Shift+Tab`: focus the previous host. The confirmation screen pages its selected-host list too. Fifty hosts no longer require a fifty-line terminal and a prayer.
- `f`: cycle status filters; `F`: cycle severity filters; `/`: search IDs and titles; `h`: search checklist paths/hosts. Filtering does not select or silently expand the edit set.
- `m`: select only hosts matching the active status filter (within the host search); `A`: select all **visible** hosts; `c`: clear all selection for this control. Check the per-state counts in the confirmation screen before trusting an `A`. Quite the demanding letter.
- `v`: read the focused host's paged full finding details and comment. In the confirmation screen, `v` opens a paged, full old/new status and comment preview for the focused selected host; `Tab` moves to the next selected host.
- `e`: edit selected hosts. Choose `1` Open, `2` Not a Finding, `3` Not Applicable, or `4` Not Reviewed. Type a required comment (`Ctrl+J` inserts a newline), then Enter to review each selected host's old → new answer. `y` applies; Esc cancels; `q` quits. Bulk Open → Not a Finding also requires typing `OVERRIDE` in a second confirmation. That is not evidence; it is a speed bump with manners.

All four existing states are editable. **Select only hosts that actually share the decision**; a shared STIG ID is not a shared configuration. Search and comment editing are independent of control navigation. Full comments and finding details can be paged for review.

Status badges follow the [STIG Viewer 3.x guide](https://dl.dod.cyber.mil/wp-content/uploads/stigs/pdf/U_STIG_Viewer_3-x_User_Guide_V1R7.pdf): Open is red, Not a Finding green, Not Applicable black on a pale badge, and Not Reviewed gray. The symbols and labels remain visible in a monochrome terminal; even the colors know better than to be your sole source of evidence.

## What an edit actually does

Checkmate replaces the status and comment for each selected control in its existing CKLB. Other JSON fields are preserved, although the JSON formatting is rewritten. It stages each file beside the original, syncs it, retains its permissions, and renames it over the original. Before the batch, it checks **all selected files** for unchanged bytes, official schema validity, and ability to stage replacements. Each edit checks its file again before writing. A small mercy for anyone else with the same bright idea.

A multi-file edit is **not transactionally atomic**. If a later file fails, earlier files remain changed; the TUI reports how many succeeded. Do not edit the same checklists concurrently or run the tool on a tree writable by an untrusted process.

**Important:** Existing `.cklb.provenance.json` sidecars are not changed. Their recorded CKLB hashes become stale. Checkmate does not sign output or establish that a bulk decision is correct for every host. Keep a separate backup before editing irreplaceable assessments.

`--validate` checks the newest checklist in each directory against DISA's [STIG Viewer 3 CKLB 1.0 schema](https://dl.dod.cyber.mil/wp-content/uploads/stigs/zip/SV3_CKLB_1_0_JSON_SCHEMA.zip), published 2024-01-10. The original JSON is embedded in the binary for offline operation (SHA-256 `8b88205de07662006e1660b1decad677d344828d6687b1dd26bca15105b7bb37`). Schema validity says the JSON has the expected shape, not that any security finding is true. Schema-invalid selected files are blocked from interactive edits; `--list` is still an inventory, not a validation report.

## Disconnected installation

From the source tree on a build host with the vendored dependencies:

```sh
./scripts/package.sh 0.3.0
(cd release && sha256sum -c SHA256SUMS)
```

The build produces `release/checkmate` for Linux amd64, Linux amd64/arm64 archives, and `release/SHA256SUMS`. The entire `release/` directory is ignored by Git; none of those binaries is a source file. Transfer the appropriate `release/checkmate_0.3.0_linux_<arch>.tar.gz` and `release/SHA256SUMS` to the disconnected host. Verify its archive hash there, extract it, and install the binary in a user-owned location. No package repository, installation wizard, or dramatic appeal to the internet is required.

```sh
sha256sum -c SHA256SUMS --ignore-missing
mkdir -p "$HOME/.local/bin"
tar -xzf checkmate_0.3.0_linux_amd64.tar.gz
install -m 755 checkmate_0.3.0_linux_amd64/checkmate "$HOME/.local/bin/checkmate"
```

The archive contains the binary and this README. Build and test use vendored modules with `GOPROXY=off GOSUMDB=off` and `CGO_ENABLED=0`. Packages target Linux amd64 and arm64. Building an arm64 archive is not the same thing as running it on an arm64 host, however persuasive the filename looks.

## Development

```sh
GOPROXY=off GOSUMDB=off go test -mod=vendor ./...
```

Keep source scans and real CKLB files outside this repository. Do not commit generated checklists, scan output, or host findings. The repository has enough material to review without donating your infrastructure data to it.
