# scribe — design

**scribe** — a **Razor** component engine for **email + PDF**, in the `c5n`/`f8n`/`l10n` family. Same "hide the horrible, expose clean components" DNA as `etch`: write clean, localised, accessible components; render one source to email HTML *and* PDF. Personal open-source (destined). Consumer of the stack. Skeleton — agenda open. See `../l10n/DESIGN.md`, `../a11y/DESIGN.md`, `../etch/DESIGN.md`.

## What it is
- **Component base-classes (Razor)** — templates composed from base-classes that use `l10n` (localised) + `a11y` (accessible) by construction.
- **Email** — the base-classes hide the awful email-HTML quirks (legacy client nonsense); the author writes clean components → HTML output. Emails reach the server **ready-formed** (HTML body + narrative).
- **PDF** — the same HTML is handed to a **PDF service using Playwright** (headless-browser conversion).
- **One source → two outputs** — email HTML and PDF from one component.

## North star (to firm up)
1. **Hide the horrible** — clean components; the author never fights email-HTML.
2. **One source → email + PDF** — a single component, two rendered outputs.
3. **Localised + accessible by construction** — via `l10n` + `a11y`.

## Design agenda (open)
- **Component model** — the Razor base-class surface; how `l10n`/`a11y` compose in.
- **Email-HTML abstraction** — which quirks the base-classes hide, and how (tables/inline-styles/client hacks).
- **PDF via Playwright** — the **PDF service boundary** (a headless browser is a heavy runtime dep + an ops/trust surface — `design-rigour`); where it runs; HTML → PDF fidelity.
- **`etch` embedding** — graphics as PNG in email/PDF.
- **Compliance seam** — email is customer-facing content (`l10n`'s translation-approval + compliance gate applies); accessible email/PDF via `a11y`.

## Change log
- 2026-07-09: created — scaffolded as a forge project dir (consumer lib). Seed frame: Razor component base-classes over `l10n`+`a11y`, one-source → email HTML + PDF (Playwright); flagged the Playwright/PDF-service dependency boundary. Agenda open.
