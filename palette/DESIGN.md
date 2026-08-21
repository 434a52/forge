# palette — design

**palette** — the **design-tokens** layer, in the `c5n`/`f8n`/`l10n` family. Canonical design values (colour, spacing, type, motion) as source data, generated via `c5n` into typed, cross-target outputs. The shared design foundation for every front-end surface. Personal open-source (destined). Base layer + a `c5n` consumer. Skeleton — agenda open. See `../c5n/DESIGN.md`, `../lattice/DESIGN.md`.

## What it is
- **Tokens as canonical data** — colour, spacing, type scale, radii, motion, etc., authored once as source (YAML/JSON).
- **Generated via `c5n`** — the tokens are *data → typed code*, exactly `c5n`'s job (like `f8n` generates domain data). Emits **CSS custom properties** + **typed TS constants** (and cross-language targets as needed). Demonstrates c5n's generality beyond domain data.
- **The shared foundation** — `lattice`, `etch`, `scribe`, `portfolio` all draw from the same tokens → one design source of truth across UI / SVG / email / PDF.
- **Theming** — swap token sets to re-skin/re-brand; the components don't change.

## Taste bar (visual craft)
palette is not just "tokens exist" — it's the **taste artifact** of the stack: the layer where visual craft is made systematic and legible. Because every front-end surface draws from these tokens, taste invested here **compounds** into `lattice`, `etch`, `scribe`, and `portfolio`. The bar to hit, calibrated against reference systems (Linear, Vercel/Geist, Radix Themes, Stripe):
- **Colour** — a perceptually-even ramp (OKLCH), semantic aliases over base scales, **contrast pairs accessible by construction** (WCAG AA+ *provable*, not asserted — the a11y ∩ visual overlap).
- **Type** — a considered modular scale; deliberate line-height and measure.
- **Spacing** — a consistent rhythm (a base unit + scale, not ad-hoc values).
- **Motion** — a small, disciplined set of durations/easings; restraint over flourish.
- **Dark mode** — a first-class token set, not an afterthought inversion.

"Visual quality, tested" applies here: contrast is provable in the token *data*; downstream visual-regression guards the rendered result.

## North star (to firm up)
1. **Single source of truth** for design values, across every surface.
2. **Codegen-native, cross-target** — one source → CSS + TS (+ more) via `c5n`.
3. **Themeable** — token sets are swappable.

## CSS emission (mechanism)
Tokens → CSS is a **new in-tree `c5n` emitter**, and its complexity (media queries + cascade **precedence**) lives in **Go, not the template**:
- A **Go pre-render pass** lowers `tokens × themes × breakpoints` into a **flat, correctly-ordered `[]Rule`** (`{ media, selector, declarations }`) — precedence baked into the order.
- A simple **`text/template`** (Go stdlib — no new dep) walks that ordered list; no cascade logic in the template.

So it's a **logic-bearing in-tree emitter** (like c5n's shared value-emitter) — fine for blessed in-tree targets; the "pure template bundle" bar is only for third-party bundles (see `../c5n/DESIGN.md` → *Emitters are template bundles*). A more powerful template engine isn't needed, and would only tempt logic into the wrong layer. Much more declarative than Sass (tokens as data, not imperative mixins). Output: **CSS custom properties** (runtime-themeable) + **typed TS constants** (build-time). **The emitter itself lives in `c5n`** (in-tree, `go:embed`'d) and stays **CSS-general, not palette-aware** — palette supplies the schema + token data (incl. the breakpoint/theme structure); c5n knows *CSS*, never "design tokens" (domain-blindness held). Anything genuinely palette-specific is palette's lowering/front-end (à la `l10n`).

## Design agenda (open)
- **Token taxonomy** — the categories + naming (base vs semantic/alias tokens; e.g. `colour.blue.500` → `colour.action`).
- **c5n CSS emitter** — mechanism decided (*CSS emission* above): Go pre-render → ordered rule-list → `text/template`. Remaining: the exact CSS-var naming + the TS-constant shape.
- **Theming / multi-brand** — how token sets compose and switch (light/dark, brands).
- **Format alignment** — align the source with the **W3C Design Tokens** format for interop; contrast with Style Dictionary (palette = the codegen-native alternative).
- **Consumers** — how `lattice`/`etch`/`scribe` bind (CSS vars at runtime vs typed constants at build).

## Change log
- 2026-08-21: **taste bar added — palette is the stack's *visual-craft* artifact, not just "tokens exist."** Named the reference bar (OKLCH ramp with provable contrast pairs, modular type scale, spacing rhythm, disciplined motion, first-class dark mode; calibrated against Linear/Geist/Radix Themes). Taste here compounds downstream; "visual quality, tested" (contrast provable in the token data).
- 2026-07-10: **CSS-emission mechanism decided** — a logic-bearing in-tree `c5n` emitter: a Go pre-render pass resolves media-query/cascade precedence into an ordered flat rule-list, and a simple `text/template` (stdlib, no dep) renders it. No fancier template engine; logic stays in Go, template stays declarative. (Mirrored in `c5n`'s emitter section — in-tree emitters may carry logic; portable-template-language deferred to third-party bundles.)
- 2026-07-09: created — scaffolded as a forge project dir (base-layer + `c5n` consumer). Seed frame: design tokens as canonical data → `c5n` → CSS custom properties + typed TS; the shared design foundation for the front-end; theming via token-set swap. Notes a new **c5n CSS emitter** as the driver. Agenda open.
