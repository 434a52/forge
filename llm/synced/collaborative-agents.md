# Collaborative agents

Working with a coding agent across a **shared repo** and a **team**. The single-corpus workflow (`corpus-driven-design.md`, `corpus-driven-development.md`) assumes one person over one body of context; this covers what changes when other people — and their agents — work the same code. It is the vendor-neutral **model**; for the Claude Code mechanics, see `collaborative-claude.md`.

## The failure it prevents

**Agents fill missing context with plausible guesses — silently.** That is what makes a context gap between collaborators dangerous in a way it was not before.

When your agent can see design rationale a collaborator's agent cannot, the collaborator's agent does not stop and ask. It invents a plausible answer and proceeds. They ship work that is confidently wrong about decisions they never saw, and **nobody can see why** — the gap left no error, just divergence. Before agents, a person missing context hit a wall and asked; an agent papers over the wall. So "docs are nice to have" becomes load-bearing.

## The rule: shared context lives in the repo

**Commit the design and its rationale in the repo it describes.** The "why" travels with the code. Anyone who clones — and their agent — gets the decisions and trade-offs as first-class committed content, not folklore in your head. That is the whole fix: the friction was never that one agent was "better," but that shared work depended on unshared context. Put the context in the shared artifact and the dependency is gone.

For most teams this is the entire model: **one repo, the shared why committed inside it, done.**

## Repo docs stand on their own

A committed doc must carry **its own why** — it cannot silently assume context that is not in the repo. If it leans on something only you know, a collaborator's agent is back to guessing. So when a decision needs background to make sense, put enough of that background in the repo; and if a reference would point somewhere a collaborator cannot follow, it does not belong in a repo doc.

## Context flows both ways

A collaborator is not only a *consumer* of shared context — they **produce** decisions and code. The same rule governs their output: their "why" goes in the repo too, under normal review (PRs, reviewable diffs — shared authorship, not unreviewed authorship). Whoever leads the roadmap, **design rationale can come from anyone and lands in the same committed docs** — do not let a contributor's reasoning stay outside the repo.

## Two halves of parity

A collaborator's agent reaches parity through two committed channels:

1. **Committed design docs** — the project's *why* (this doc's subject).
2. **Synced instruction payloads** — the shared *working method* (`collaboration.md`, `design-rigour.md`, `pair-coding.md`, the corpus-driven docs), distributed into the repo so everyone runs the same conventions from one pinned source (`getting-started.md`, `sync.md`).

Why plus how, both committed — whatever agent each person runs.

## Onboarding

Clone the repo, open the agent in it. The committed conventions, design docs, and synced context load; the collaborator is at parity from the first session. Onboarding is a clone, not a hand-over of tribal knowledge.

## When context spans repos: the shared store

When work spans several repos, some context belongs to *no single one* — how the repos fit, the contracts between them, the sequencing of a change that touches several. It still has to be shared (a collaborator needs it), so it needs a home that is neither any one repo nor your private corpus. Two facts make this simpler than it first looks.

**It usually has a steward.** Cross-repo context tends to have one natural author — whoever works across the repos — while others mostly read. That makes it **publish/subscribe, not co-editing**: one writer, many readers. So the concurrency problem you might reach for a lock or a merge engine to solve *mostly isn't there* — no write–write contention with one hand on the pen. Don't build symmetric-concurrency machinery for an asymmetric pattern.

**A low-ceremony shared git repo is the right store.** Given a single writer, a small repo the steward pushes and readers pull does everything the job needs:
- **Atomic publish, across files** — a commit lands as one unit and a reader pulls whole commits, never a half-applied change. The **commit is the coherent-publish boundary**: the writer edits freely and commits a coherent state; readers and their agents only ever see coherent states. No in-file locks, no status flags, no working-copy dance — the primitive you'd otherwise hand-roll *is* `commit`.
- **Pull-on-request reads** — a reader pulls when they want the latest; the files are ordinary markdown, so an agent consumes them as context directly.
- **History** — you see how the shared understanding evolved.

This is why a **database is the wrong reach** — its appeal is atomicity, which git has natively, with agent-readable files and no infrastructure — and why a **file share is only a fallback** (below).

**The catch is governance, not technology.** In a governed environment git is often locked down — mandatory review, protected main — for real reasons: production-code integrity, audit, supply-chain safety. That ceremony is *right for production code and wrong for high-velocity shared notes.* The fix is a **risk-proportionate repo tier, never circumvention** — a low-ceremony repo under stated conditions:
- **private, with explicit access**;
- **isolated from all CI/CD and dependency resolution** — a *sink*, never a build source (an unreviewed repo is only safe if nothing downstream pulls from it);
- **still audit-logged**;
- **data-classification-bounded**.

The framing that earns a yes: *relax review — it guards production code — while keeping access-control, audit, and supply-chain isolation, which still apply.* A governance **tier** matched to a lower-risk repo class, not an exemption.

**Sanctioned channels only.** Never route around a governance control by moving data to storage you personally control (an external or personal account). The control exists for a reason, and data sent out cannot be recalled. Get a sanctioned low-ceremony tier, or use a sanctioned file share — but stay inside the boundary.

**Fallback — a shared file share.** If a low-ceremony repo is not attainable, a synced file share works, with two compromises: **per-file coherence only** (no cross-file atomicity — each file syncs independently), so the writer writes each shared file in **one coherent pass**, and any genuinely atomic multi-file change routes through the ceremonied path instead. Fine for single-writer notes — it just lacks the `commit` primitive that makes the repo version clean.

## Optional — keeping a private planning space

Some people keep private planning *outside* the shared repo: a personal corpus, a roadmap, half-formed strategy not ready (or not appropriate) to share. That is fine, with one rule:

**The repo is the single source of truth — prefer it over your local notes.** Your private space may point *into* the repo; the repo never depends on it. Do not keep an editable duplicate of a design doc in your private notes — the two drift and the committed one loses. Author design *in* the repo; keep only genuine orchestration (sequence, priorities, private reasoning) outside it. The split is about *roadmap*, not *design voice*.

## Optional — if the repo is, or will be, public

If the repo is open (or headed there), mind what enters it: **leaks are irreversible** — git history persists, clones and forks exist. Clean-room anything work-derived before committing — strip employer names, internal system names, confidential specifics; keep the reusable lesson. Treating the repo as already-public from the first commit is cheaper than scrubbing later. (See `design-rigour.md`.)

## Where it fits

**Human-facing methodology** — read when setting up a repo and onboarding people, not a payload reloaded every task. The companion to the corpus-driven docs: those capture one person's thinking into a corpus and build from it; this is what to share, what to keep private, and where each lives once other people join. Cheap to adopt early, expensive to retrofit — shape the seam before you need the room. One workflow, not *the* workflow.
