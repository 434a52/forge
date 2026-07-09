# forge

Cross-language codegen — raw material (canonical data, typed schemas) **forged** into typed, conformance-verified code for C#, TypeScript, and beyond.

A monorepo for the codegen-coupled stack: the engine (`c5n`) and the libraries built on it. Private during development; each library publishes as an independent package, and public demo/docs sites build from here with the source staying private.

## Projects

| dir | what |
|---|---|
| `c5n/` | the codegen engine (Go) — schema + data → typed cross-language code |
| `f8n/` | domain primitives — Money, Currency, Country, rates, temporal (C# + TS) |
| `l10n/` | localisation library — cross-language, on-device, tree-shakable (C# + TS); design only for now |
| `doppel/` | realistic locale-aware synthetic data (coherent personas) on f8n + l10n — the visual demo; design only for now |
| `a11y/` | accessibility toolkit — accessible primitives, announcer, provable/auditable compliance; skeleton |
| `etch/` | SVG component system — clean vector components, localised + accessible, one source → UI/PDF/email; skeleton |
| `scribe/` | Razor engine for email + PDF — clean components → email HTML + PDF (Playwright); skeleton |
| `portfolio/` | bespoke Vue/Vite showcase site that dogfoods the stack (l10n/a11y/etch); skeleton |

First milestone: the **c5n + f8n walking skeleton** (one currency → C# + TS constant → one golden vector proving parity). `l10n`, `doppel`, and the rest land as they're built.

Each project's design docs live in its own dir (`<project>/DESIGN.md`).

## Conventions

See `CLAUDE.md`.
