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

## Flagship: `MoneyInput` — the React-parity proof (2026-08-22)
The **one component built fully in both Vue and React**, passing the same interaction vectors. Chosen deliberately over the usual candidates (combobox, date picker, dialog):

- **It is the only candidate that exercises *both* agnosticism axes.** Cross-language — `Money` identical in C# and TS, proven by `f8n`'s golden vectors; and cross-framework — Vue ≡ React, proven by interaction vectors. Every other component tests the framework axis alone.
- **It is the argument for this library, executable.** A bought component library cannot ship a money input: currency-aware decimal places (0/2/3 by currency), locale grouping and decimal separators, symbol placement, correct rounding, and above all *not a float*. General-purpose libraries are domain-blind **by construction** — that is what makes them general. The gap between *a number input* and *a `Money` input* is precisely the integration layer lattice exists to own.
- **Its behaviour is genuinely hard**, so the vectors mean something. Parity proven on a trivial component proves nothing.
- **The apps need it anyway**, so it is product-pulled rather than built for the demo — and it generalises directly to a `QuantityInput` (kW/kWh) over the same machinery. The real flagship is *a domain-typed numeric input*; `Money` is the first instance.

### The seam, for this component
- **Shared floor (pure TS, called not spread):** `l10n`'s money **formatter** and **parser** over `f8n`'s `Money`/`Currency`, plus the **editing state machine** as pure `(state, event) => state` transitions (`a11y`'s interaction-state territory, not a component concern).
- **Idiomatic ceiling (never shared):** how that binds to an `<input>` — Vue `ref`/`computed`/`v-model`, React `useState` + controlled input. The line stops exactly at the binding layer, as everywhere else.

### Editing state is the value's source of truth; `Money` is a projection
The hard part is **not** formatting a `Money` — it is the intermediate states that *are not a `Money` yet*: `""`, `"-"`, `"1"`, `"1."`, `"1.2"`, a value past the currency's decimal places, a separator typed mid-number. Modelling this as `string ↔ Money` means fighting the caret forever.

So: **the editing state is the truth, and `Money` is derived from it — legitimately absent while the input is mid-edit.** Everything else follows from that one decision, and it is what makes the behaviour testable rather than felt.

### Blur is the commit boundary — which is *why* the value lifecycle works
Blur is the moment the editing state resolves into a value or fails to. That makes it the natural point at which three things happen together:
1. the `Money` projection is **committed**;
2. the property advances through its **lifecycle** — *unset → set*, and distinctly *set-then-cleared*, which is not the same as never-touched and is the difference between showing a required-field error and not;
3. the **delta is sent** (small and precise; the response returns complete state).

Validating mid-edit would mean validating a string that is not yet a value — which is why *"required"*, range and cross-field rules are meaningful only at the commit point. Character-level rejection still happens during typing, but it **prevents or holds**; it does not raise an error.

**One refinement to hold:** validate on blur, then **re-validate on input once a field is already in error**, so a user sees the error clear as they fix it rather than only on the next blur. First error on blur; subsequent feedback live.

### Live re-validation covers the client subset only — so errors need provenance
Live re-validation can only re-run the rules the client actually holds: the **declarative, generated** ones. Server-only rules — imperative, asynchronous, or spanning entities the client cannot see — are not re-evaluable locally, and their errors legitimately persist until the next round-trip. *We can only do what we can do*, and the design should say so rather than pretend otherwise.

That makes **error provenance a first-class field**: every problem is tagged `client` or `server`, and the two are reconciled by rule, never merged blindly:

- **Server-origin problems are replaced wholesale on every response.** The response carries the complete application-level set (see the delta/state asymmetry), so the client swaps its entire server-origin set each time — it never clears one individually, and never decides a server problem has been resolved.
- **Client-origin problems are the client's to own** — re-evaluated on input for the field being edited, cleared the instant the value satisfies the rule.
- **The displayed set is the union, deduplicated by (property key, rule identity), with server winning any conflict.** The server is authoritative: if the client evaluates a rule as satisfied and the server disagrees, the server's verdict is what the user sees.

**Why deduplication is possible at all — and it is a direct payoff of codegen.** The server re-validates everything, so it *will* re-report rules the client already evaluated. Suppressing the duplicate requires that the same rule carry the **same stable identity on both sides** — which it does precisely because both are generated from one declarative source. Without generation, reconciliation degrades to matching on message strings, which is exactly as fragile as it sounds. Deduplicating errors is therefore not an incidental UI nicety; it is one of the concrete things cross-language generation buys.

**The practical consequence of the declarative/imperative line:** it does not only decide *what may be generated* — it decides **which errors the client is permitted to clear on its own**. Anything imperative belongs to the server, both to evaluate and to retract.

### What the vectors assert
A sequence of input events → the expected **editing state**, **caret position**, and **projected `Money`** (or its absence), run identically against the Vue and the React implementation. Same spec→vectors→conformance technique as the rest of the family, one axis over. Together with `f8n`'s C# ≡ TS golden vectors, one component carries a **three-layer conformance story** meeting at a single primitive — a strong candidate for the `portfolio` site's worked example, since an interactive trace beats a static one.

## Design agenda (open)
- **Component set** — the curated list (what's needed to demonstrate + serve the apps); the composition model.
- **`a11y` integration** — components built on the accessibility primitives; the accessibility contract each guarantees.
- **`palette` theming** — how tokens flow in (CSS vars vs typed constants); light/dark/brand.
- **`l10n` integration** — localised labels/formatting; RTL.
- **Relationship to `etch`** — graphics/icons as components within the UI (complementary, not overlapping).

## Change log
- 2026-08-22: **`MoneyInput` chosen as the React-parity flagship**, and its design settled. Picked because it is the only candidate exercising **both** agnosticism axes (C# ≡ TS on `Money` via golden vectors; Vue ≡ React via interaction vectors), because it is the library's own argument made executable (a bought library cannot ship a money input — general-purpose components are domain-blind by construction), and because the apps need it anyway (generalises to `QuantityInput` over the same machinery — the real flagship is *a domain-typed numeric input*). **Core design decision: the editing state is the source of truth and `Money` is a projection**, legitimately absent mid-edit — modelling it as `string ↔ Money` means fighting the caret forever. **Blur is the commit boundary**: the projection commits, the property advances *unset → set* (distinct from *set-then-cleared*, which is why required-errors behave correctly), and the delta is sent — mid-edit validation would be validating a string that is not yet a value. Refinement held: first error on blur, live re-validation once already in error. Vectors assert (editing state · caret position · projected `Money`) over an event sequence, run against both implementations. Also recorded: **live re-validation covers the client subset only** (declarative/generated rules; server-only rules persist until the next round-trip), so **problems carry provenance** — server-origin replaced wholesale per response, client-origin owned and cleared by the client, displayed as the union deduplicated by (property key, rule identity) with **server winning conflicts**. Deduplication is possible only because generated rules share **stable identity across both languages** — a concrete payoff of codegen, not a UI nicety; without it reconciliation degrades to matching message strings. Naming: **`MoneyInput`**, not `CurrencyInput` (the bound type is `Money`; `Currency` is a separate primitive and that name reads as a currency picker) — `../l10n/DESIGN.md` updated to match.
- 2026-08-21: **framework strategy set — Vue-first, React live.** Thin shared layer of pure services (`a11y`/`l10n`/`palette`); each framework fully idiomatic above the prop-getter/binding line; no shared adapter layer. Cross-framework parity proven by conformance vectors against both implementations. Scope: build Vue fully, prove React on 1–2 flagship components.
- 2026-07-09: created — scaffolded as a forge project dir (mid-layer front-end lib). Seed frame: accessible + localised + token-driven Vue components; **curated, not comprehensive** (the tokens/a11y/l10n DNA is the differentiator, not breadth); fills the a11y-primitives → real-components gap. Agenda open.
