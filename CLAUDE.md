# CLAUDE.md — forge

Monorepo for the codegen-coupled stack (`c5n`, `f8n`, `l10n`, `doppel`, …). **Private during development.** Design docs live in each project dir (`c5n/DESIGN.md`, `f8n/DESIGN.md`, `l10n/DESIGN.md`, …).

## Instruction context

@llm/index.md

The corpus-driven methodology + collaboration/design-rigour payloads, synced from **i10s** (pinned in `llm.conf`). To update: bump the ref in `llm.conf` and follow `llm/sync.md`. Project-specific instructions go under `llm/local/`.

## Attribution

Credit Claude as co-contributor on all git activity:
- End commit messages with: `Co-Authored-By: Claude <noreply@anthropic.com>` (generic — not model-specific).
- Note Claude Code involvement in PR bodies.

## Structure

- One project per top-level dir (`c5n/`, `f8n/`, …), each **self-contained** (own build + package) so it can `git subtree split` out later.
- One root `llm/` — shared instruction context synced from i10s (`synced/` = pinned + managed; `local/` = yours), referenced via `@llm/index.md`.

## Architecture (the load-bearing decisions)

- **c5n generates data + a thin typed boundary; behaviour is hand-written per language**, verified identical by golden vectors — "generate the wiring, hand-write the algorithm".
- **Conformance = spec → golden vectors → code.** The prose spec is the oracle; edges hand-verified against an authority.
- **Codegen-native by construction** — every generative library is designed *for* c5n from day one.

## Conventions

- UK English. Terse, load-bearing docs; change logs newest-first.
- Build/test commands per project — added here as each lands.
