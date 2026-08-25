# Collaborative Claude

Running the collaborative-agents model (`collaborative-agents.md`) in **Claude Code** specifically. That doc is the vendor-neutral principle — what to keep private, what to share, why. This is the harness mechanics: how context actually loads, how a collaborator reaches parity on clone, and — optionally — how to span more than one repo.

*Claude Code behaviour changes between versions. The load rules below are current at time of writing; verify the version-sensitive details (especially the added-directory env var) against the Claude Code docs before relying on them.*

## How Claude Code loads context

At launch Claude Code discovers and loads:

- **`~/.claude/CLAUDE.md`** — user-level, per-machine. **Private to you, never shared.**
- **Project `CLAUDE.md`** (or `.claude/CLAUDE.md`) — walked *upward* from the working dir and loaded root-to-cwd. This is the **committed, shared** layer everyone who clones gets.
- **Nested `CLAUDE.md`** in subdirectories — loaded **on demand** when Claude reads files in that subtree. Ideal for per-project conventions in a monorepo (each project dir carries its own).
- **`@import`** — a `CLAUDE.md` can inline other files with `@path` (relative to the *importing* file, or absolute), up to ~4 hops deep. This is how a repo pulls its aggregated instruction context in with one line: `@llm/index.md`.
- **`.claude/skills/`** — committed skills, auto-discovered. A cloned repo's skills appear in the collaborator's `/` menu with no setup.

Mapped onto the two layers: `~/.claude/CLAUDE.md` is **yours** (personal prefs); the project `CLAUDE.md` + `@llm/index.md` + the committed design docs + `.claude/skills/` are the **shared** layer. Everything a collaborator needs is in the second set, and all of it is just committed files.

## The default: root in the repo

The simple, friction-free posture — and the one everyone uses day to day, you and collaborators alike:

1. Clone the repo.
2. Open Claude Code **in the repo root.**

That auto-loads the project `CLAUDE.md`, the `@llm/index.md`-imported synced payloads (the shared working method), the per-project nested `CLAUDE.md` on demand, and the committed `.claude/skills/`. No env vars, no flags, no access to your private corpus. **They are at project-parity from the first session** — the model working exactly as designed.

## Optional — spanning a private corpus and repos

Only if you keep private planning outside the repo (see `collaborative-agents.md` → *Optional — keeping a private planning space*) *and* steer several repos at once. A normal single-repo user never meets this; it is the one place a cross-repo complication lives. Two ways to arrange it:

- **(a) Root in the repo you're working; `/add-dir` the corpus.** The repo's `CLAUDE.md` loads natively; the corpus is reachable for reference. Best for focused work on one repo. *Recommended default.*
- **(b) Root in the corpus; `/add-dir` the repos.** One session spans everything — but **added directories do not load their `CLAUDE.md` by default.** Set `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` at session start to load them. Note the asymmetry: `.claude/skills/` from an added dir load automatically; `CLAUDE.md` does not (without the flag). File read/write access is granted either way.

The trap in (b): forget the env var and your session silently runs **without** the added repos' conventions — the invisible-context-gap failure from `collaborative-agents.md`, now biting the orchestrator at the harness layer. If you span repos this way, set the var (or prefer posture (a)).

## Wiring the shared working method

The method half of parity — `collaboration.md`, `design-rigour.md`, `pair-coding.md`, the corpus-driven docs — lands via the i10s sync into the repo's `llm/` dir and is pulled into context by a single `@llm/index.md` line in the project `CLAUDE.md`. So "root in the repo" gets the shared conventions automatically, from one pinned source. See `getting-started.md` and `sync.md`.

## The sync as a committed skill

Vendor the sync instruction as a committed skill (`.claude/skills/<name>/SKILL.md`; the legacy `.claude/commands/<name>.md` also works). Because committed skills auto-discover on clone, a collaborator gets the `/sync` command with zero setup — the distribution mechanism dogfooding Claude Code's own skill discovery. Updating the shared payloads is then just: run the skill, review the git diff, commit.

## Private vs shared, by file

| Location | Layer | Shared? |
|---|---|---|
| `~/.claude/CLAUDE.md` | your personal prefs | **No** — per-machine, yours |
| private corpus (separate dir/repo) | orchestration — roadmap, strategy | **No** — yours |
| repo `CLAUDE.md` / `.claude/CLAUDE.md` | shared project conventions | **Yes** — on clone |
| committed design docs (in-repo) | the project *why* | **Yes** — on clone |
| repo `llm/` (synced) + `@llm/index.md` | shared working method | **Yes** — on clone |
| repo `.claude/skills/` | shared commands (e.g. sync) | **Yes** — on clone |

## The Claude-specific gotcha

**A project convention in your *user* `CLAUDE.md` is invisible to collaborators.** It loads for *your* agent and no one else's — the exact asymmetry `collaborative-agents.md` warns against, recurring at the config layer. If a rule matters to the project, it belongs in the **committed project `CLAUDE.md`**, never `~/.claude/CLAUDE.md`. Keep the user file for genuinely personal, cross-project preferences (your editor habits, your machine) — nothing a contributor's agent would need to do the work.
