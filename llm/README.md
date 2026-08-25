# llm — instruction context

The shared LLM instruction corpus, vendored from [i10s](https://github.com/434a52/i10s) at
a pinned ref and wired into the agent by a single `@llm/index.md` import.

## Where to edit — read this before changing a rule

| you want to change | edit |
|---|---|
| a shared rule, applying everywhere | the **i10s repo**, then re-sync here |
| a rule for this repo only | `llm/local/` — yours, never overwritten |
| which docs are pulled, or the version | `llm.conf` at the repo root |

**Never edit `llm/synced/`.** It mirrors the source at the pinned ref, so the next sync
overwrites it and the change disappears without warning. The files carry no banner saying
so precisely because they are byte-identical copies — that fidelity is the guarantee.

**Where a new shared rule belongs:** the doc router in the
[i10s README](https://github.com/434a52/i10s#the-docs) lists what each doc is for. Find the
row before writing; a rule that fits none is a sign the corpus needs a new doc rather than
a stretched existing one.

## Updating

1. Bump `ref:` in `llm.conf` to a newer i10s tag.
2. Follow `llm/sync.md` — it fetches at the pin, mirrors `synced/`, regenerates `index.md`.
3. Review the diff, then commit. The review *is* the safety gate; nothing auto-applies.

## Layout

- `synced/` — pinned docs from i10s. Managed; do not edit.
- `local/` — this repo's own instruction docs. Yours; survives every sync.
- `index.md` — generated; imports `synced/` then `local/`, so local takes precedence.
- `sync.md` — the vendored sync instruction: your reviewed copy, never live remote code.
