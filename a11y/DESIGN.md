# a11y — design

Accessibility toolkit for the front-end stack, in the `c5n`/`f8n`/`l10n` family. Two halves: **accessible-by-construction primitives** the other libs build on, and a distinctive **provable/auditable-compliance** system. Personal open-source (destined). Base layer — depends on nothing. Skeleton — agenda open. See `../l10n/DESIGN.md`, `../etch/DESIGN.md`, `../scribe/DESIGN.md`.

## What it is
- **Accessible primitives / base-classes** — the accessible foundations the rest of the stack composes (focus management, roles/ARIA by construction, keyboard interaction).
- **Announcer / live-region** — a utility for programmatic announcements (used by `etch` for animation state, and anywhere dynamic content changes).
- **Compliance checking** — an analyser (build-time) + testing helpers that flag a11y problems.
- **Provable, auditable accessibility (the distinctive part)** — ingests **axe** reports and **runtime tooling that runs alongside the Vue app**, recording a11y data so QAs can document coverage **with proof** — auditable, WCAG-style evidence. Echoes the stack's provable/auditable theme (`f8n` diff→PR, `l10n` approval gate).

## North star (to firm up)
1. **Accessible by construction** — the primitives make the correct thing the default.
2. **Provable compliance** — coverage documented with evidence, not asserted.
3. **Composable** — consumed cleanly by `etch`, `scribe`, and the app layer.

## Design agenda (open)
- **Primitive set** — which base-classes/components, and the accessibility contract each guarantees.
- **Announcer / live-region API** — the `etch`-animation consumer is the first driver.
- **Compliance-documentation system** — axe ingest + runtime capture + the "coverage with proof" model (what evidence, how stored/reported, how a QA signs off).
- **The `l10n × a11y` seam** — the audit must **emit the l10n-key ↔ control mapping + a screenshot per key** (feeds `l10n`'s translator UI). Cross-cutting; design both sides together.
- **Analyser + test helpers** — build-time checks (hard-coded a11y violations, missing labels) and test-time assertions.

## Change log
- 2026-07-09: created — scaffolded as a forge project dir (base-layer front-end lib). Seed frame: accessible primitives + announcer + provable/auditable compliance; the l10n×a11y translator-UI seam noted. Agenda open.
