# lattice — design

**lattice** — an accessible, localised, token-driven **component library — Vue-first, React live**, in the `c5n`/`f8n`/`l10n` family. UI components built on `palette` (tokens) + `a11y` (accessibility) + `l10n` (localisation). Personal open-source (destined). Mid-layer. Skeleton — agenda open. See `../palette/DESIGN.md`, `../a11y/DESIGN.md`, `../l10n/DESIGN.md`, `../etch/DESIGN.md`.

## What it is
- **Accessible, localised, token-driven components** — real UI components (the assembled things), built on `a11y`'s primitives, `l10n`'s formatting, and `palette`'s tokens; can embed `etch` graphics.
- **Fills the gap** between `a11y` (accessibility *primitives*) and the apps (which need actual *components*).

## North star (to firm up)
1. **The DNA, not the breadth** — codegen-native tokens + accessible-by-construction + localised. The differentiator, not a component count.
2. **Accessible by construction** — components *are* the accessible ones (built on `a11y`).
3. **Themeable** — driven by `palette` tokens.

## Framework strategy — Vue-first, React live
Vue is the **primary target** (Vue's fine-grained reactivity binds the shared core near-free); **React is kept live** because the component-library skill has to travel to non-Vue teams. Each framework stays **fully idiomatic** — its own views, reactivity, slots, event conventions, and component API — over a **thin shared layer**: the pure services from `a11y` (announcer / state-transition fns / ARIA-computation) + `l10n` formatting + `palette` tokens. **No shared adapter/binding layer** — a headless-core/adapter (à la Zag/Ark) reads as non-idiomatic to *both* audiences; the shared/idiomatic line is the prop-getter/binding layer. **Cross-framework parity is proven by conformance** — one behavioural spec + interaction vectors run against *both* the Vue and React implementations (the family's spec→vectors→conformance technique, applied across frameworks instead of languages). **Scope: build Vue fully; prove React on 1–2 flagship components** (where "prove" = passes the same vectors as the Vue build).

## Scope (the deliberate line)
A Vue component library is a **crowded, commodity** space (Vuetify/PrimeVue/Radix/shadcn); *breadth is not the differentiator.* So lattice is **curated** — enough components to demonstrate the DNA and serve `portfolio`/the apps, **not** a comprehensive rival. The `palette` + `a11y` + `l10n` integration is the star; the components are the vehicle that proves it.

## Design agenda (open)
- **Component set** — the curated list (what's needed to demonstrate + serve the apps); the composition model.
- **`a11y` integration** — components built on the accessibility primitives; the accessibility contract each guarantees.
- **`palette` theming** — how tokens flow in (CSS vars vs typed constants); light/dark/brand.
- **`l10n` integration** — localised labels/formatting; RTL.
- **Relationship to `etch`** — graphics/icons as components within the UI (complementary, not overlapping).

## Change log
- 2026-08-21: **framework strategy set — Vue-first, React live.** Thin shared layer of pure services (`a11y`/`l10n`/`palette`); each framework fully idiomatic above the prop-getter/binding line; no shared adapter layer. Cross-framework parity proven by conformance vectors against both implementations. Scope: build Vue fully, prove React on 1–2 flagship components.
- 2026-07-09: created — scaffolded as a forge project dir (mid-layer front-end lib). Seed frame: accessible + localised + token-driven Vue components; **curated, not comprehensive** (the tokens/a11y/l10n DNA is the differentiator, not breadth); fills the a11y-primitives → real-components gap. Agenda open.
