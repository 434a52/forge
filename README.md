# forge

Cross-language codegen — raw material (canonical data, typed schemas) **forged** into typed, conformance-verified code for C#, TypeScript, and beyond.

A monorepo for the codegen-coupled stack: the engine (`c5n`) and the libraries built on it. Private during development; each library publishes as an independent package, and public demo/docs sites build from here with the source staying private.

## Projects

| dir | what |
|---|---|
| `c5n/` | the codegen engine (Go) — schema + data → typed cross-language code |
| `f8n/` | domain primitives — Money, Currency, Country, rates, temporal (C# + TS) |

First milestone: the **c5n + f8n walking skeleton** (one currency → C# + TS constant → one golden vector proving parity). `l10n`, `doppel`, and the rest land as they're built.

Full design docs live with each project — see its `DESIGN`.

## Conventions

See `CLAUDE.md`.
