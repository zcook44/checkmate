# Offline example inputs (private, local only)

The Checkmate checkout on this workstation has two owner-private, Git-ignored directories:

- `examples/rhel{7,8,9,10}/<host>/` — one **copied real** NODE1 OpenSCAP XCCDF result, matching HTML report and ARF, plus its existing CKLB per OS. No provenance sidecars are copied; editing a CKLB does not regenerate them.
- `benchmarks/` — the current local RHEL 7 V3R15, 8 V2R9, 9 V2R10, and 10 V1R3 DISA manual XCCDF XML, plus each OS's current converter tailoring YAML. These are converter inputs, **not** Checkmate inputs.

These contain host assessment evidence and official benchmark content. Neither directory is committed or included in Checkmate's binary release. Do **not** force-add them to Git. The CKLB schema is not included; `--schema` remains an independent, operator-supplied option. The source `stig_library` and saved assessment tree remain unchanged.

## Recreate from the owner-private source trees

Run this from a checkout on the authorized workstation, with both destinations absent. Python 3's standard library suffices. The input layout must have exactly one `RHELN-NODE1/scan_*/<host>` under each `rhelN` directory; for multiple runs, first prepare a private source selection with one desired run per OS. The script refuses symlinks, non-private source files and existing destinations. It stages private copies, writes per-directory `SHA256SUMS`, and does not change originals.

```sh
python3 scripts/stage-offline-examples.py \
  --scan-root /scratch/src/admin/stig/completed_ckl/Q4_2026 \
  --library-root /scratch/src/admin/stig/stig_library
(cd examples && sha256sum -c SHA256SUMS)
(cd benchmarks && sha256sum -c SHA256SUMS)
```

The `--output` argument can instead name another existing, owned, canonical directory without group/world write access. `examples/` and `benchmarks/` must not already exist there. The selected benchmarks are the highest V/R XML available for each OS; the script copies the corresponding per-OS policy YAML. It does not authenticate DISA source bytes or verify the source CKLB against the separate schema.

## Exercise after an offline build

```sh
GOPROXY=off GOSUMDB=off ./scripts/package.sh 0.4.0
./release/checkmate --root "$PWD/examples" --list
# Interactive editor (in-place edits to the PRIVATE COPIES, not saved source):
./release/checkmate --root "$PWD/examples"
# Optional, if you separately obtained the local DISA CKLB schema:
./release/checkmate --root "$PWD/examples" --validate --schema /private/schema/SV3_CKLB_1_0_JSON_SCHEMA.json
```

To test conversion read-only, run the **separate** `burden-of-proof` Go converter from its source checkout, passing absolute paths to this repository's `examples/` and `benchmarks/` with `batch-cklb --dry-run`. The supplied scan bundles include HTML+ARF; a successful dry run reports four bundles and writes nothing. Permission or naming validity alone does not assert assessment correctness. If you edit a staged CKLB, its `examples/SHA256SUMS` will fail until you deliberately re-stage or explicitly account for your changes; that failure is useful.

For disconnected transfer, a **separate private archive** is generated locally as `release/private-offline-inputs.tar.gz` with `release/private-offline-inputs.sha256`. These are Git-ignored, mode `0600`, and are **not** part of the generic Checkmate binary release or its `SHA256SUMS`. Regenerate them from the repository root after changing staged inputs:

```sh
umask 077
tar -czf release/private-offline-inputs.tar.gz examples benchmarks
(cd release && sha256sum private-offline-inputs.tar.gz > private-offline-inputs.sha256)
(cd release && sha256sum -c private-offline-inputs.sha256)
```

Transfer both only to authorized recipients; verify the archive using `(cd release && sha256sum -c private-offline-inputs.sha256)` before extracting into an owner-private directory (`0700`). Once extracted, verify `(cd examples && sha256sum -c SHA256SUMS)` and `(cd benchmarks && sha256sum -c SHA256SUMS)`. The generic binary release intentionally omits private fixtures and official benchmarks. An archive hash checks transport integrity, not the authenticity of its source evidence.
