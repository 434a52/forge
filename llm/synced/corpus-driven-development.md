# Corpus-driven development

The companion to **corpus-driven design** (`corpus-driven-design.md`). Design captures the thinking into the corpus; *development* executes it productively. The bridge is a phased plan synthesised from the corpus — which is *why* the build is fast: the model codes against captured decisions, not guesses. This is the "code it in a day" half of "design heavily, then build fast."

## The plan is the spine

The design corpus is synthesised into a `.PLAN` — the implementation plan:

- **Phases → steps.** Phases are manageable chunks; each phase breaks into ordered steps.
- **Sequence matters.** Ordered by dependency (prerequisites first) and by value (MVP first).
- **MVP-first.** Build the minimal working thing first, then layer outward.
- **Work staged across phases** where possible — just enough of a component for the current increment, the rest deferred. Each phase stays a coherent, compilable, testable checkpoint. Vertical slices, not component-by-component.
- **`✓` as it lands.** Mark steps and phases done in place. The plan doubles as progress tracker and session handover — the next session picks up at the first unchecked step.

## Why it makes the coding productive

The structure turns the build into a stream of **small, ordered, individually-verifiable units**:

- Each step's prerequisites are already in place — no forward-references the model has to guess around.
- Each phase compiles and tests — a feedback checkpoint.
- The model always has a clear, bounded, context-rich next thing.
- **The model isn't deciding *what* to build** — the plan (from the design corpus) decided that. It executes well-specified steps in sequence. Remove the freedom to wander and it stops wandering.

## The feedback loop

- **Strong types and tests are the guardrail.** The model writes a step, runs the build and tests, observes, and self-corrects — before you look. A well-typed, well-tested codebase is what makes LLM coding *reliable*, not just fast.
- **The editing hand still applies at the code level.** The model will produce plausible-but-wrong code; tests catch much, you catch the rest. You're the reviewer.

## Every phase ships

Staging work across phases keeps *every* phase a working MVP — you could stop at any checkpoint and have something real. No big-bang integration at the end; risk is spread across small, verified increments.

## The setup that supports it

- **`CLAUDE.md`** — build and test commands, conventions, gotchas — so each session starts productive with no re-explaining.
- **The corpus and the plan loaded as context** — so the model builds the *right* thing, with the *why*.

## Where it fits

This is the *execution* half; it only works on top of a real design corpus. Without the captured thinking, the plan is the model's best guess and the coding is blind — garbage in. Like its companion: one workflow, not *the* workflow.

