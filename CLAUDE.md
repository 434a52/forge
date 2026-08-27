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
- **Conformance = one language-neutral vector dataset → a thin runner per language.** Parity is transitive through the shared dataset, so a new target is one more spoke. Each non-obvious vector carries its rationale and authority citation *beside the numbers*. Green proves every language matches the dataset — never that the dataset is right; that stays one human pass per rule.
- **Codegen-native by construction** — every generative library is designed *for* c5n from day one.
- **Anything crossing a wire gets both round trips, in both languages.** `fromJson(toJSON(x)) == x` says nothing is lost on the way out; `toJSON(fromJson(w)) == w` says nothing *else* is accepted on the way in — the second holds only for a canonical wire form, so it doubles as a test of canonicality. The two halves of the check do different jobs and both are needed: **vectors** feed *the same input to both languages and pin the exact bytes*, which is what makes C#'s output TS-readable (transitively, through the shared dataset); **properties** carry no expected value, so they catch classes of error a fixed dataset cannot and depend on no implementation. A property alone proves one language is self-consistent and says nothing about the other. *Proportionality: consumer apps inherit the properties, run in their own test suite — they do not need a language-neutral dataset and a driver, which exist so a third party can audit a published contract.*

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
- Language-agnostic code style (naming, braces, no clever one-liners) is in `llm/synced/pair-coding.md`. Below is only what is specific to a language, or specific to this repo.

### C#

- **`var` when the right-hand side names the type** — constructors, literals, casts, and calls whose name carries it (`var reduced = Canonical(num, den);`). Explicit only where the type is not on the line, or where a declaration is separated from its assignment, which `var` cannot express.
- **A bare integer literal is an `int`.** `var x = 10;` narrows where a `BigInteger` or `decimal` was intended, and the code still compiles. In numeric work, suffix the literal (`10m`) or name the type. This matters here more than in most codebases: the whole substrate is scaled integers and exact fractions, and a silent narrowing is exactly the class of bug that survives review and fails a vector.

### TypeScript

- **`const` by default**; `let` only where the value is reassigned.
- **`bigint` literals carry `n`** — `100n`, never `100`. A bare literal is a `number`, and mixing the two throws at runtime. The same narrowing trap as C#'s bare `int`, one language over.
- **Never `Number(...)` on a value that must stay exact.** It is a float64 and silently loses anything past 2^53 — the defect that has already appeared twice here, once decoding data through `any` and once as the obvious way to parse a decimal. Parse digit strings into exact integer parts instead.
- **Import specifiers carry `.js`** — `from "./percentage.js"`. TypeScript resolves it back to the `.ts` source, and Node's ESM loader requires it; without it the compiled output only resolves inside a bundler. The c5n TypeScript writer emits this, and hand-written files must match.

## Build & test

The gates. `.github/workflows/ci.yml` runs exactly these on push and PR. Deliberately
uncounted — the list grows, and a number in the prose is a thing that goes stale silently.

- **c5n engine** — `cd c5n && go test ./...`
- **drift-guard** — `cd c5n && go build -o "$TMPDIR/c5n" ./cmd/c5n && "$TMPDIR/c5n" check ../f8n`
  (build the CLI to a path outside the module: a bare `go build ./...` drops an in-tree
  binary, which is why `/c5n/c5n` is gitignored. `c5n build ../f8n` regenerates.)
- **C# target** — `cd f8n/dotnet && dotnet build`
- **TS target** — `cd f8n/ts && npm ci && npm run typecheck`
- **conformance** — both languages against one vector dataset, and again under
  `LC_ALL=tr_TR.UTF-8` (see `.github/workflows/ci.yml`; `Country.Find` is the one path a
  locale can move). Needs the TS build first
  (`cd f8n/ts && npm run build`), since the runner executes the compiled `dist/`:

  ```
  go build -o "$TMPDIR/conform" ./c5n/cmd/conform
  for v in f8n/vectors/*.json; do
    "$TMPDIR/conform" -vectors "$v" \
      -runner "csharp=dotnet run --project f8n/dotnet/RunVector --" \
      -runner "ts=node f8n/ts/dist/run-vector.js"
  done
  ```
