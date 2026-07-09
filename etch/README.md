# etch

A component system for **SVG** — build vector graphics *without* wrestling raw SVG (hide the horrible, expose clean components). Localised (`l10n`) and accessible (`a11y`, including an announcer for animations). **One component source → multiple outputs:** animated UI SVG (Vue binds to parts to show/hide), PDF (rendered as PNG), and email (PNG).

Mid-layer; consumes `l10n` + `a11y`; consumed by the app layer (UI/charts), `scribe` (as PNG), and `doppel` (document rendering).

Design: `./DESIGN.md`. Status: skeleton.
