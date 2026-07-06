# i10s sync

An instruction for your coding agent. It updates `llm/synced/` from the i10s source at the **pinned ref**, **verbatim via git** — never retype or reword the docs; fidelity comes from git, not from you. Everything lands as a reviewable diff; nothing auto-applies.

## Steps

1. Read `llm.conf` (repo root): `source`, the pinned `ref`, and the ordered `docs` globs.
2. Fetch the source at exactly that `ref` (shallow is fine):
   `git clone --depth 1 --branch <ref> https://<source> <tmp>`
   Use the pinned tag/commit — **never** `main`/`HEAD`.
3. Copy the files matching the `docs` globs from `<tmp>` into `llm/synced/`, **byte-for-byte** (overwrite; delete any synced file no longer matched — `synced/` mirrors the source). Do not edit contents.
4. Regenerate `llm/index.md`:
   - a header noting `source`, `ref`, and the date (provenance);
   - `@`-imports of `synced/` files in `llm.conf` order (stable-sorted within a glob), then `local/` files (sorted). `local/` comes after `synced/` so local overrides.
5. Leave everything as a **git diff for review** — do not commit unless asked; if opening a PR, summarise what changed.

If the pinned ref's docs look wrong, or the diff is surprising, **stop and flag it** — do not proceed. Fetch nothing from the source except the doc files at the pinned ref.
