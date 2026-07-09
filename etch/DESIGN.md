# etch — design

**etch** — a component system for **SVG**, in the `c5n`/`f8n`/`l10n` family. Lets a developer create SVGs *without really knowing SVG* — the "hide the horrible, expose clean components" pattern. Localised and accessible by construction; **one component source, many outputs**. Personal open-source (destined). Mid-layer. Skeleton — agenda open. See `../l10n/DESIGN.md`, `../a11y/DESIGN.md`.

## What it is
- **Clean SVG components** — a component model over raw SVG, so authors compose shapes/charts/diagrams without hand-writing SVG.
- **Localised + accessible** — consumes `l10n` (localised labels/values in the graphics) and `a11y` (accessible SVG, plus the **announcer** for animation state).
- **Multi-output — one source, three targets:**
  - **UI** — animated SVG where **Vue can bind to parts** (show/hide, drive transitions).
  - **PDF** — rendered as **PNG**.
  - **Email** — as **PNG**.
  - *Server-side rasterisation (PDF/email PNG) runs in the **`press`** render service (Node + Playwright), not in-library.*

## North star (to firm up)
1. **Hide the horrible** — clean components; the author never touches raw SVG.
2. **One source → many outputs** — UI SVG / PDF-PNG / email-PNG from a single component.
3. **Localised + accessible by construction** — never bolt-ons.

## Design agenda (open)
- **Component model** — how components compose; the authoring surface (Vue components? a DSL?).
- **The multi-output renderer** — one source → animated UI SVG vs rasterised PNG (PDF/email); where the split lives; the PNG rasterisation path.
- **Animation + Vue binding** — how parts expose bind points; the `a11y` announcer for animation state.
- **`l10n` integration** — localised text/values inside graphics; layout reflow for locale.
- **Consumers** — app-layer charts/UI, `scribe` (PNG), and **`doppel`'s document rendering** (SVG templates → PNG) — a known cross-project edge to design against.

## Change log
- 2026-07-09: created — scaffolded as a forge project dir (mid-layer front-end lib). Seed frame: clean SVG components, localised + accessible, one-source-multi-output (UI/PDF-PNG/email-PNG); the `doppel` document-rendering consumer noted. Agenda open.
