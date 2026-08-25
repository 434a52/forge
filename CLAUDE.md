# CLAUDE.md — forge

Monorepo for the codegen-coupled stack (`c5n`, `f8n`, `l10n`, `a11y`, `palette`, `doppel`, `etch`, `scribe`, `press`, `lattice`, `portfolio`, …). **Private during development.** Design docs live in each project dir (`c5n/DESIGN.md`, `f8n/DESIGN.md`, `l10n/DESIGN.md`, …).

## Instruction context

@llm/index.md

The corpus-driven methodology + collaboration/design-rigour payloads, synced from **i10s** (pinned in `llm.conf`). To update: bump the ref in `llm.conf` and follow `llm/sync.md`. Project-specific instructions go under `llm/local/`.

## Attribution

Credit Claude as co-contributor on all git activity:
- End commit messages with: `Co-Authored-By: Claude <noreply@anthropic.com>` (generic — not model-specific).
- Note Claude Code involvement in PR bodies.

## Structure

- One project per top-level dir (`c5n/`, `f8n/`, …), each **self-contained** (own build + package) so it can `git subtree split` out later.
- Each project dir carries **`DESIGN.md`** (the *why* — accretes, keeps a change log) and, once it is being built, **`PLAN.md`** (the *what next* — phases → steps, `✓` marked in place, retired when the work lands). Both are bare, uppercase, and sit beside each other; the directory already says which project they belong to.
- One root `llm/` — shared instruction context synced from i10s (`synced/` = pinned + managed; `local/` = yours), referenced via `@llm/index.md`.

## Architecture (the load-bearing decisions)

- **c5n generates data + a thin typed boundary; behaviour is hand-written per language**, verified identical by golden vectors — "generate the wiring, hand-write the algorithm".
- **Conformance = spec → golden vectors → code.** The prose spec is the oracle; edges hand-verified against an authority.
- **Codegen-native by construction** — every generative library is designed *for* c5n from day one.

## This repo is public-bound — keep it clean

Everything here is written to be read by strangers. **Design docs and change logs describe the engineering, not the circumstances around it.**

Never write into this repo:

- **Personal or colleague names**, or anything identifying an individual.
- **Employer, client or project references** from paid work — including implicit ones (what a client's sector is, when a contract ends, who a manager is).
- **Career, hiring or commercial framing** — why a project helps a job search, who it's meant to impress, IP or contract reasoning.
- **First-person notes about the author** or their background.
- **Strategy, roadmap or circumstance.** This repo records *what* and *why technically* — never *why now*. **Technical sequencing is not covered by this**: an implementation plan ordered by dependency belongs here, because a contributor needs it to pick up the work. What stays out is *why this project, why now* — priority between projects, what a piece of work is for, what it is timed against.

**Change logs are the highest-risk surface.** They're written in the moment, when private-repo assumptions still hold, and never re-read. Apply the same rule to them as to the prose above them.

If a rationale is genuinely about circumstance rather than engineering, the entry here should simply omit it — not gesture at it.

## Conventions

- UK English. Terse, load-bearing docs; change logs newest-first.
- Build and test commands live under **Build & test** below.

## Build & test

The four gates. `.github/workflows/ci.yml` runs exactly these on push and PR.

- **c5n engine** — `cd c5n && go test ./...`
- **drift-guard** — `cd c5n && go build -o "$TMPDIR/c5n" ./cmd/c5n && "$TMPDIR/c5n" check ../f8n`
  (build the CLI to a path outside the module: a bare `go build ./...` drops an in-tree
  binary, which is why `/c5n/c5n` is gitignored. `c5n build ../f8n` regenerates.)
- **C# target** — `cd f8n/dotnet && dotnet build`
- **TS target** — `cd f8n/ts && npm ci && npm run typecheck`
