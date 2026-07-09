# scribe — design

**scribe** — a **TS/Vue** component engine for **email + PDF**, in the `c5n`/`f8n`/`l10n` family. Same "hide the horrible, expose clean components" DNA as `etch`: write clean, localised, accessible components; render one source to email HTML *and* PDF. Personal open-source (destined). Consumer of the stack. Skeleton — agenda open (the specific templating engine is undecided). See `../l10n/DESIGN.md`, `../a11y/DESIGN.md`, `../etch/DESIGN.md`.

> **TS/Vue, not Razor.** The stack is Vue/TS-centric, and `a11y`/`etch` are Vue — a C#/Razor server engine can't consume them cleanly, and the PDF path (Playwright) is Node-native. So scribe is TS/Vue, sharing one component model with `etch` (one source → UI *and* email/PDF). The concrete templating engine is still **open** (below).

## What it is
- **Clean components** — composed from base-classes/components that use `l10n` (localised) + `a11y` (accessible) by construction, and can embed `etch` graphics.
- **Email** — components compile to bulletproof email HTML, hiding the awful email-HTML quirks (legacy-client nonsense). Emails reach the server **ready-formed** (HTML body + narrative).
- **PDF** — the same HTML renders in the **`press`** render service (Node + Playwright). Given C#-business-logic backends, scribe is **service-first**: BEs call `press` over HTTP → ready-formed email/PDF; the direct TS-library path is secondary.
- **One source → two outputs** — email HTML and PDF from one component.

## North star (to firm up)
1. **Hide the horrible** — clean components; the author never fights email-HTML.
2. **One source → email + PDF** — a single component, two rendered outputs.
3. **Localised + accessible by construction** — via `l10n` + `a11y`.
4. **One component model with `etch`** — UI/email/PDF from the same Vue foundation.

## Design agenda (open)
- **Templating engine — OPEN; real alternatives, nothing decided:**
  - **MJML** — mature, widely-used markup → bulletproof email HTML. Not Vue components (its own markup); would need a bridge.
  - **vue-email** — Vue components → email HTML; stack-native, but newer/smaller.
  - **Vue SSR + own email-safe components** — full control, most work.
  - Trade-off axis: *maturity vs Vue-nativeness vs control.* Ecosystem moves fast — verify current maturity when deciding.
- **Email-HTML abstraction** — which quirks the components hide (tables/inline-styles/client hacks) and how.
- **PDF via Playwright** — the **service boundary** (a headless browser is a heavy runtime dep + an ops/trust surface — `design-rigour`); where it runs; HTML → PDF fidelity.
- **`etch` embedding** — graphics as PNG (or inline SVG) in email/PDF.
- **Compliance seam** — email is customer-facing content (`l10n`'s translation-approval + compliance gate applies); accessible email/PDF via `a11y`.

## Change log
- 2026-07-09: **dropped Razor → TS/Vue** for stack coherence (`a11y`/`etch` are Vue; Playwright is Node-native; unifies with `etch` under one component model). The specific templating engine left **open** with real alternatives (MJML / vue-email / Vue-SSR), nothing decided.
- 2026-07-09: created — scaffolded as a forge project dir (consumer lib). Seed frame + open agenda.
