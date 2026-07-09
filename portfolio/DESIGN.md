# portfolio — design

**portfolio** — a bespoke personal **showcase site**: the curated front door to the projects (links, diagrams, narrative). Built in the project's own stack (Vue/Vite), and — as the libraries land — **dogfoods them**, which is the point: a portfolio built *with* the tools it showcases is meta-proof by construction. Personal open-source (destined). Top-of-stack consumer. Skeleton — agenda open. See `../l10n/DESIGN.md`, `../a11y/DESIGN.md`, `../etch/DESIGN.md`.

## What it is
- **Curated front door** — hand-authored links to the projects of consequence, diagrams, and narrative. The opposite of `d11n`'s auto-aggregated docs: this is the deliberate, designed surface.
- **Dogfoods the stack** — localised via `l10n`, accessible via `a11y`, animated diagrams via `etch`; sits on the whole front-end stack. The medium becomes the message.
- **Own stack** — Vue/Vite, hand-built (not a template).

## North star (to firm up)
1. **Meta-proof** — built *with* the libraries it showcases; the site itself is evidence.
2. **Curated, not generated** — a designed narrative, distinct from `d11n`'s aggregation.
3. **Accessible + localised by construction** — dogfoods `a11y` + `l10n`.

## Two phases
1. **Simple** — links + diagrams, for early presence; minimal deps.
2. **Dogfooded** — evolve to consume `l10n`/`a11y`/`etch` as they exist. Same content, richer medium.

## Design agenda (open)
- **Site structure & content** — the sections, the narrative spine, what "projects of consequence" surface.
- **Hosting** — static (e.g. GitHub Pages / a static host); the build/publish path.
- **Phase-1 vs phase-2 boundary** — what ships simple, what waits for the stack.
- **Dogfooding integration** — how `l10n`/`a11y`/`etch` plug into a Vue/Vite site cleanly.
- **Relationship to `d11n`** — the curated front door vs the aggregated docs site; how they link/embed (portfolio → deep-links into the docs?).

## Change log
- 2026-07-09: created — scaffolded as a forge project dir (top-of-stack consumer). Seed frame: curated bespoke Vue/Vite showcase site that dogfoods `l10n`/`a11y`/`etch` (meta-proof); two-phase (simple → dogfooded); distinct from `d11n`'s auto-aggregation. Agenda open.
