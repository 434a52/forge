# Writing your own LLM instruction files

The short version: **don't write them by hand — get the model to write them, from how you actually work.** An instruction file is a spec of your preferences and conventions; the reliable way to author that spec is by *accretion, in conversation*, not by drafting it cold.

## Two kinds

Keep them separate — different readers, different registers:
- **llm-context payloads** — read by the *model* every session (how to collaborate, conventions, what to flag). Terse and imperative.
- **human-facing docs** — read by *people* (methodology, rationale, the *why*). Explanatory.

## Get the model to write them

- **Work normally.** Don't stop to author instructions up front.
- **When a preference surfaces, capture it.** "UK English." "No emojis." "Always flag uncertainty." "Prefer composition over inheritance." Say *"add that to `CLAUDE.md`"* (or the relevant doc). Over weeks the file becomes a precise spec of how you work — authored almost entirely by accretion, out of decisions you were making anyway.
- **Let the model maintain it.** Ask it to keep the file organised and non-duplicative — merge new rules into existing categories rather than appending forever. A bloated instruction file gets skimmed (by the model *and* you).

## What makes a good one

- **Terse, load-bearing.** No filler, no marketing prose; every line earns its place.
- **Imperative for the model, explanatory for humans.** Payloads say *do X / flag Y*; methodology says *why*.
- **Concrete beats abstract.** "Flag time-sensitive facts — pricing, versions, limits — and say to verify" beats "be accurate."
- **State the *why* for anything non-obvious**, so the rule survives being questioned later instead of being silently dropped.

## The corrections are the value

Same spine as corpus-driven design (`corpus-driven-design.md`): the model will draft plausible instructions that are subtly not-what-you-meant. **Read what it writes and correct it.** Your corrections are what turn a generic template into *your* spec. An instruction file you didn't check is worse than none — the model will follow it, confidently, into the wrong behaviour.

## Then distribute them

Once a file is good, others benefit. That's the rest of this project — a pinned, reviewable sync into any repo's `llm/` dir (see `DESIGN.md`). Author here; distribute there.
