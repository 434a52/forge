# doppel — design

**doppel** — realistic, locale-aware **synthetic data** for development and testing. Built on `f8n` (real Country/Currency/Money identity) and `l10n` (localised money/date/address formatting), generated codegen-native via `c5n`. Personal open-source (destined). Design-stage — this doc is the starting frame; the agenda below is open.

> **Codegen-coupled — lives in the monorepo.** doppel depends on `c5n` (codegen), `f8n` (primitives), and `l10n` (formatting), so it sits *downstream* of them and is built after they exist. See `../f8n/DESIGN.md`, `../l10n/DESIGN.md`, `../c5n/DESIGN.md`.

## What it is
- **Coherent synthetic personas** — the differentiator is **internal consistency**, not random field values. Ask for a person and the parts *agree*: an address as `UK › county › town › street` with a **postcode plausibly right for the area**; accounts, transaction histories, salary, and employer that line up into one believable *set of facts*.
- **Locale-aware** — built on `f8n` + `l10n`, so a German persona gets a German address format, EUR formatted German-style, plausible German names/postcodes. A locale switch re-coheres the whole persona.
- **Fake-but-realistic documents** — statements, and optionally IDs (driving licence, passport) — rendered as images: **unmistakeably synthetic, never passable as real** (per `design-rigour`), yet information-realistic as a coherent fact-set for testing.

## North star (to firm up)
1. **Coherence over randomness** — whole consistent personas whose parts agree. The differentiator.
2. **Locale-correct by construction** — via `f8n`/`l10n`; a locale switch re-coheres everything.
3. **Codegen-native** — designed *for* `c5n` from day one (data generated, behaviour hand-written), like `f8n`/`l10n`.
4. **Unmistakeably synthetic** — realism of *facts*, never of *authenticity*; nothing output should be passable as a real document.

## Showcase role
doppel is the **visual `f8n` + `l10n` demo** — coherent personas are money-heavy (accounts, transactions, statements, salaries), so it shows the primitives + localisation *visually* (a locale-correct fake bank statement reads instantly).

**Demo site** — an interactive **persona explorer**: click *"generate a person"* → a fresh coherent persona (address whose parts agree, accounts, locale-formatted transactions, employer); drill into each feature; a **locale switcher** (e.g. UK/DE/FR/CY) makes the combined `f8n` + `l10n` + doppel coherence tangible in one click. Client-side (browser TS → static site); **seed-based**, so "random each click" = a new seed, and any persona is shareable by URL.

## Dual-use flag (`design-rigour`) — decide before any public surface
The fake **ID / statement images are misusable** (fraud) even when "unmistakeably synthetic", and a *public* generator raises the exposure. Irreversible once public. Mitigations to decide **before** publishing a demo:
- **Lead with the coherent-data explorer** (the novel bit), not photo-realistic documents.
- **Heavily watermark** any document *images*, or **omit** passports/licences from the public demo entirely.
- The coherence is the star; photo-realism is not the point. Keep realism at the level of *facts*, not *authenticity*.

## Design agenda (open)
- **Coherence engine** — how personas stay internally consistent (hierarchical generation: region → area → address → postcode; income → accounts → transactions). The heart of it.
- **Locale model** — how a locale switch re-coheres names/addresses/postcodes/currency (drawing on `f8n`/`l10n` + locale-specific generators).
- **Seed / determinism** — seed-based reproducibility; shareable-by-URL personas; stable across runs.
- **Document rendering** — statements/IDs as images (SVG → PNG?), the watermark/omit policy, the synthetic-by-construction guarantee.
- **Codegen boundary** — what `c5n` generates (reference/lookup tables) vs hand-written generators/behaviour.
- **Demo delivery** — client-side static site building from the private monorepo (the shopfront model), public output, and the dual-use gate above.

## Change log
- 2026-07-08: created — scaffolded as a forge project dir (codegen-coupled: built on c5n/f8n/l10n). Seeded from prior roadmap notes: coherent locale-aware synthetic personas, the visual f8n+l10n demo (persona explorer + locale switcher, client-side/seed-shareable), and the dual-use flag on fake-document images. Design agenda opened.
