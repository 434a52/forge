# doppel

Realistic, **locale-aware synthetic data** for development and testing — built on `f8n` (real Country/Currency/Money) and `l10n` (localised formatting). The realism is **internal consistency**: coherent personas whose parts agree (an address whose county/town/postcode line up; accounts and transactions that make sense together), not random field values. Also produces **obviously-fake-but-information-realistic documents** (statements, IDs) as images.

Doubles as the **visual `f8n` + `l10n` demo** — a persona explorer with a locale switcher. Codegen-coupled: lives in this monorepo, built on `c5n`/`f8n`/`l10n`.

Design: `./DESIGN.md`. Status: design.
