# Corpus-driven design

A working method: build a living, cross-referenced corpus of decision and design docs, captured *conversationally* with an LLM, and let that corpus become the engine that drives your design — plans, decisions, and onboarding all draw from it. The doc is a byproduct of the thinking you were doing anyway.

## The shift

Most people's first instinct is to use an LLM as a *content generator*: "write me a design doc for X." The output is generic, you learn nothing producing it, and the doc sits unread.

The high-value use is *conversation-driven capture*: you work through a real decision *with* the model, and the doc falls out as a byproduct. Get that distinction right and the rest follows; get it wrong and you get plausible-sounding docs you don't trust.

## The loop

1. Pick a real decision you're actively facing — not a hypothetical.
2. Describe the constraints: what you've considered, what you've ruled out and why.
3. **Talk through it.** Push back. Correct mistakes as you spot them.
4. Once a direction is clear, have the model write it into the doc.
5. **Read the edit carefully.** Correct anything wrong before moving on.
6. Come back tomorrow. Edit more. The doc is never "done."

Steps 3 and 5 are where the value lives. Skip them and you get confident-sounding, subtly-wrong content.

## Why it works

- **Parallel writing.** The model writes while you think about the next question. Documentation normally lags decisions by weeks and often never happens; this drops its cost to near zero.
- **The editing hand.** The model will produce plausible prose that's wrong in detail. *Your corrections are what make the corpus trustworthy.* A confidently-wrong doc is worse than no doc — you'll act on it later and find the error the hard way. You are the reviewer, not a passive consumer.
- **Compounding.** Cross-link docs where they touch. New decisions lean on captured prior ones; future-you reconstructs *why* a choice was made without trawling chat history. One doc is nice; a cross-referenced corpus is transformative — and the links let the model load related context on demand, not just humans clicking through.

## From discussion to plan

The loop produces *discussion docs* — decisions and tradeoffs captured as you worked them out. When the thinking has crystallised enough that "what exactly do we do?" is the useful question, synthesise a plan from the accumulated corpus.

A plan written cold is the model's best guess: generic, under-specified, subtly wrong in ways you can't spot. A plan synthesised from weeks of captured decisions inherits the corrections, the pushback, the ruled-out alternatives. **It's right *because* the prior docs were wrong and got corrected.** If the answer is still "it depends / we haven't decided X," keep iterating in the discussion docs — premature plans lock in the wrong shape and then get defended.

## Where the leverage is — and isn't

Genuinely good at: synthesising tradeoffs you'd lose track of mid-thought; surveying an option space when you don't know what's possible; cross-referencing across a corpus; keeping a consistent voice as it grows; and writing at speed while you think.

*Not* the leverage: writing code faster than you can. If you're fast and know the codebase, code-gen is a wash. The leverage is in architecture, design, tradeoffs, and synthesis.

Catch its weaknesses: current facts (pricing, versions, regulatory limits — training cutoffs are real, and the more recently something changed the more confidently wrong it tends to be); niche specifics it can't know; and *honest uncertainty* — it often sounds confident about things it shouldn't. That last one is the single biggest risk.

## The supporting setup

Light, but it matters:

- **A notes repo** — git-tracked markdown, one file per topic, organised by topic not chronology. Commit often.
- **A `CLAUDE.md`** (or equivalent standing-instructions file) at the root, holding your persistent preferences and conventions. Let the model maintain it: when a preference surfaces ("no emojis", "UK English"), say "add that" — over weeks it becomes a precise spec of how you work, authored almost entirely by accretion.
- **Cross-links** between docs, so the corpus is a navigable graph for both humans and the model.
- **A change log** at the bottom of long-lived docs, newest first.
- **Backup and privacy decided up front.** Reflective decision logs suit neither a shared/team remote (wrong audience) nor an arbitrary public one; decide where they live *before* reflexively pushing. Local git plus a private bundle backup threads the needle.

## Caveat

This suits some people and some work, not all. It needs you to: care about capturing decisions; push back on confident-sounding answers rather than accept them; have work with *decisions* in it (design, strategy, tradeoffs) rather than pure repetition; and accept that the model makes mistakes and catching them is your job. If those don't fit, use the tool differently — this is one workflow, not *the* workflow.

