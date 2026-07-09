# palette — design

**palette** — the **design-tokens** layer, in the `c5n`/`f8n`/`l10n` family. Canonical design values (colour, spacing, type, motion) as source data, generated via `c5n` into typed, cross-target outputs. The shared design foundation for every front-end surface. Personal open-source (destined). Base layer + a `c5n` consumer. Skeleton — agenda open. See `../c5n/DESIGN.md`, `../lattice/DESIGN.md`.

## What it is
- **Tokens as canonical data** — colour, spacing, type scale, radii, motion, etc., authored once as source (YAML/JSON).
- **Generated via `c5n`** — the tokens are *data → typed code*, exactly `c5n`'s job (like `f8n` generates domain data). Emits **CSS custom properties** + **typed TS constants** (and cross-language targets as needed). Demonstrates c5n's generality beyond domain data.
- **The shared foundation** — `lattice`, `etch`, `scribe`, `portfolio` all draw from the same tokens → one design source of truth across UI / SVG / email / PDF.
- **Theming** — swap token sets to re-skin/re-brand; the components don't change.

## North star (to firm up)
1. **Single source of truth** for design values, across every surface.
2. **Codegen-native, cross-target** — one source → CSS + TS (+ more) via `c5n`.
3. **Themeable** — token sets are swappable.

## Design agenda (open)
- **Token taxonomy** — the categories + naming (base vs semantic/alias tokens; e.g. `colour.blue.500` → `colour.action`).
- **c5n emitters** — a **CSS custom-properties emitter** (new c5n target) + typed TS constants; the emitter bundles.
- **Theming / multi-brand** — how token sets compose and switch (light/dark, brands).
- **Format alignment** — align the source with the **W3C Design Tokens** format for interop; contrast with Style Dictionary (palette = the codegen-native alternative).
- **Consumers** — how `lattice`/`etch`/`scribe` bind (CSS vars at runtime vs typed constants at build).

## Change log
- 2026-07-09: created — scaffolded as a forge project dir (base-layer + `c5n` consumer). Seed frame: design tokens as canonical data → `c5n` → CSS custom properties + typed TS; the shared design foundation for the front-end; theming via token-set swap. Notes a new **c5n CSS emitter** as the driver. Agenda open.
