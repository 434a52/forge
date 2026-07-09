# a11y

Accessibility toolkit for the front-end stack — accessible component **primitives/base-classes**, an **announcer / live-region** utility (used by `etch` animations), and **compliance checking** (analyser + test helpers). The distinctive part: **provable, auditable accessibility** — it ingests axe reports + runtime data so QAs can document WCAG coverage *with proof*.

Base layer (depends on nothing); consumed by `etch`, `scribe`, and the app layer. Cross-cutting seam with `l10n` (the audit emits the l10n-key ↔ control mapping + per-key screenshots for l10n's translator UI).

Design: `./DESIGN.md`. Status: skeleton.
