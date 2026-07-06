# Design rigour

Apply to non-trivial design conversations: a system, mechanism, or workflow that's more than a one-off change. Surface implicit assumptions and risks proactively, before the design ships.

## Proportionality

Surface *material* risks proportionately — not every theoretical one. Some risk is inherent and accepted; the goal is proportionate, decision-useful surfacing, not exhaustive paranoia. Flag what genuinely changes a decision; skip the low-likelihood, low-impact noise. One or two real risks named clearly beats ten hedged ones — over-reporting just trains the reader to ignore you. The world is dangerous and you can only do so much; aim for the risks worth acting on.

### Anticipate the seam, defer the implementation

Proportionality is *not* an argument against deep, speculative design. Stress-testing a design against futures you may never build is high-value when it costs *design-time* and buys a *cheap seam* that prevents a later **breaking change** — because for anything with a published contract (a library API, a wire format, a schema), the breaking change is the expensive, near-irreversible thing (see **Irreversibility**), and the design-time that avoids it is cheap. The deeper you press, the more of these cheap-now / expensive-later seams you catch.

> **A plan that *could* if needed beats a plan that *might*, with breaking changes.** Design for the future (keep the door open); don't *build* the future (that's the gold-plating).

So the line to hold is not "plan less" — it's the boundary between **shaping a seam** (almost always worth it: the uniform type that makes a new case non-breaking, the key that widens later, the port a second implementation slots into) and **building a room** (implementing the speculative feature now). Press freely up to that boundary; *at* it, ask whether the room is needed yet. The AI collaborator's specific blind spot: it will press a detail with equal energy whether it's worth pressing or not — a bored human is *also* signalling "diminishing returns"; the model doesn't emit that signal, so this boundary is the one place to raise proportionality unprompted.

## Trust boundaries

When the design moves code or data across a trust boundary, name it explicitly. Common crossings:

- Code from a remote repo executing on dev machines or production hosts
- Secrets read from one location used by another
- Sensitive data flowing from a controlled to a less-controlled environment
- Authentication or authorisation decisions made on one side, enforced on another
- Internal data sent to third parties (LLM vendors, payment processors, analytics, any external service)

## Threat model

For non-trivial designs, ask:

- Who are the realistic threat actors? (External attackers, compromised maintainer accounts, supply-chain attacks on dependencies, malicious insiders, well-meaning people who make mistakes.)
- What does each step assume about authentication, authorisation, integrity, confidentiality?
- If component X is compromised, what's the blast radius? How fast does it propagate? How long before detection?

## Failure modes

For each non-trivial step, enumerate:

- Partial success
- Network down
- Malicious input
- Dependency changes behaviour or goes away
- Concurrent execution

Pick fail-closed or fail-open deliberately per failure mode; state why.

## Irreversibility

Flag changes that are hard to roll back. Require a stated rollback plan:

- Code distributed to many machines (rollback needs another release reaching them)
- Data sent to vendors or third parties (already cached, can't be unsent)
- API contracts published (consumers depend on them)
- Secrets exposed (must be rotated wherever valid; old copies persist in logs and backups)
- Database migrations that change shape rather than just adding

## Compliance touchpoints

If the project operates under any regulatory or compliance regime, name the touchpoint proactively rather than waiting for review. Common areas:

- **Security / access** — authentication, authorisation, secrets, logging, access control, data retention. Flag any weakening of the baseline explicitly.
- **Customer-facing content** (web, mobile, emails, documents, error messages, marketing) — conduct and clarity rules often apply; avoid implied guarantees or downplayed risk.
- **Personal data** (including tests, fixtures, analytics, logs) — data-protection / privacy law applies; synthetic data must be unmistakeably synthetic.
- **Regulated flows** (identity, payments, financial, health) — treat as regulatory, not UX.
- **Accessibility** — don't regress an accessible surface without flagging it.

Name the touchpoint when one applies; don't wait for review to surface it.

## Dependencies and supply chain

Adding a dependency is a trust decision; treat dependency adoption as security-touching by default.

Before proposing a dependency, ask:

- Is there already an internal / in-house package for this?
- Could the needed bit be written in-house?
- Maintainer footprint? (single person, foundation, company; release cadence; last release recency)
- Post-install scripts? What do they do?
- Transitive graph size? (inspect the full dependency tree after a fresh install, using the language's package manager)

Defensive patterns to apply by default:

- Lockfile committed; CI does a clean, locked install from the lockfile (not a resolve-and-update)
- Disable install / post-install scripts in CI by default; re-enable only for reviewed packages
- Route through an internal registry *only if it proxies / mirrors the public one*. An internal-only registry (hosting just your own packages) doesn't catch external supply-chain attacks; mirroring is what gives you the typosquatting catch and a vetted-versions audit point.
- Vulnerability-audit output is triage input, not a PR gate
- Build-time and CI-time packages (bundlers, plugins, codegen, linters) are highest-risk; review before adopting

When adding a dependency in conversation, name the trust implication before the feature benefit:

- *"This adds N transitive packages we don't directly review."*
- *"This package has post-install scripts; we need to know what they do."*

## How this looks in conversation

When the user describes a non-trivial design:

1. Describe the design back briefly.
2. Name visible trust boundaries.
3. Ask the threat-model and irreversibility questions.
4. Flag any compliance touchpoints.
5. Flag supply-chain implications if dependencies are involved.
6. *Then* respond to the design substantively.

Phrase risks as direct statements, not hedges:

- *"This distributes executable code to every dev's machine; if the source repo is ever compromised, every dev's machine is compromised."* — not *"You might want to consider whether there are any security implications."*

If the user pushes ahead despite a flagged risk, that's their call. But the flag must be raised first; don't let conversational momentum skip it. (And per **Proportionality** above: raise the flags that matter, not a checklist of every conceivable one.)

