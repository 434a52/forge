# press — design

**press** — the **render service** for the front-end stack, in the `c5n`/`f8n`/`l10n` family. A deployable **Node** process that hosts **Playwright** and produces final output: `scribe`'s email HTML + PDF, and `etch`'s SVG → PNG. Consumers (the C# business-logic backends) call it over **HTTP** with data → ready-formed artifacts. Personal open-source (destined). The one *running-service* member of the stack. Skeleton — agenda open. See `../scribe/DESIGN.md`, `../etch/DESIGN.md`, `../c5n/DESIGN.md`.

## Why a service (not a static binary)
`c5n`'s "zero runtime added to the consumer" property (its Go static binary) is the right instinct — but **HTML → PDF fundamentally needs a browser** (Chromium via Playwright, for real layout fidelity), which is irreducibly heavy; there's no static-binary equivalent. So the c5n *spirit* is delivered a different way: **isolate the heavy runtime (Node + Vue + Playwright/Chromium) in one service**, and let consumers of *any* language call it over HTTP. A **C# backend never spins up Node or Playwright** — it makes an HTTP call. Service boundary instead of binary.

## What it is
- **Deployable Node service** — hosts Playwright/Chromium; single responsibility: render components → final output.
- **Consumes `scribe` + `etch`** — renders scribe's email/PDF and rasterises etch's SVG → PNG, from **one browser pool**.
- **HTTP boundary** — `data + template ref → ready-formed email HTML / PDF / PNG`. Called by the C# BEs.
- **Contract via `c5n`** — the request/response DTOs are `c5n`-generated (C# ↔ TS, cross-language-conformant, with `fromJson`), so the service boundary can't drift.

## North star (to firm up)
1. **Right tool per job** — Node for rendering (Playwright-native, Vue-native), C# for business logic; neither forced into the other.
2. **Zero imposition on consumers** — they call HTTP; the heavy runtime lives here.
3. **One pool, both jobs** — `scribe` + `etch` share the browser host.

## Design agenda (open)
- **HTTP API** — endpoints/shape; sync vs async (a slow PDF → job/queue?); the `c5n` contract DTOs.
- **Browser pool** — warm Playwright contexts for latency; concurrency/limits.
- **Security / sandbox** (`design-rigour`) — it runs a browser over content: containerised, restricted network egress, renders *own* components + data, never untrusted arbitrary HTML (XSS/SSRF surface).
- **Ops / deploy** — containerised on the **official Playwright Docker image**; target **Azure Container Apps** (**scale-to-zero** suits the call-when-needed pattern → near-free at low traffic); **internal-only** (private ingress / auth — not publicly exposed); **GitHub Actions** builds + ships the image (registry → deploy). Size for Chromium (2GB+). The one running-service dependency.
- **Fidelity** — HTML/CSS → PDF correctness; fonts; page sizing.

## Change log
- 2026-07-10: created — scaffolded as a forge project dir (the first running-service member). Seed frame: deployable Node render service hosting Playwright; consumes `scribe` + `etch` from one browser pool; HTTP boundary for C# backends (service-not-binary — the browser is the irreducible floor); contract DTOs via `c5n`; sandbox + ops flagged. Agenda open.
