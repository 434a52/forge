# ampersand — design

**ampersand** — a consumer **home-electrification planner**, in the `c5n`/`f8n`/`l10n` family. Answers the homeowner's question: *can I run solar + battery + heat pump + EV charger(s) on my existing supply — without blowing the fuse or the budget — and how?* Top-of-stack consumer product; dogfoods the whole stack. Skeleton — agenda open. See `../f8n/DESIGN.md`, `../l10n/DESIGN.md`, `../palette/DESIGN.md`, `../lattice/DESIGN.md`, `../etch/DESIGN.md`, `../a11y/DESIGN.md`.

## The name
**amp** (the supply constraint — a UK home is ~60–100A single-phase) **+ & / "and"** (running solar *and* battery *and* heat pump *and* chargers, all together). The product is a pile of "ands" that has to fit inside the amps. The `&` glyph is the mark.

## What it is
- **The problem** — as homes electrify, the main supply (100A ≈ 24kW; many homes 60–80A) becomes the binding constraint. Heat pump (~3–5kW) + EV charger (7kW each) + battery inverter (~3.6–5kW) + shower/oven (8–10kW) can't all run at once without tripping the **main fuse** — a DNO fuse, expensive/slow to upgrade and sometimes unavailable. The alternative to a supply upgrade is **intelligent load orchestration**.
- **The product** — a consumer **planner + load-orchestration simulator** (a decision tool, *not* a live hardware controller): describe your home (supply, devices, car(s), tariff, solar/battery) → *does it fit? what's the smart schedule that stays under the limit while minimising cost and maximising solar self-use? what if I add a second EV / a heat pump?* → shown as a simulated day/year with a live-style dashboard + scenario comparison.
- **The engine (`ballast`)** — the constraint-solver core: a **hard constraint** (total draw ≤ supply amps) + **multi-objective optimisation** (min cost vs a time-of-use tariff, max solar self-consumption, comfort, battery health) + **interchangeable priority/shed strategies**. A pipes-and-filters allocation engine: *decorator* = eligibility/constraints, *strategy* = the optimisation algorithm. (`ballast` = an electrical ballast limits current — the mechanism *is* the name.)

## North star (to firm up)
1. **Consumer-legible** — a homeowner gets it in seconds; the depth is underneath (turtles-all-the-way-down when they dig).
2. **Real engine, simulated I/O** — the brain that *would* drive a real HEMS, proven by simulation on synthetic + real open data (tariff APIs, solar profiles). No live hardware.
3. **Dogfoods the stack** — `f8n` (units/cost/money), `l10n`, `palette`/`lattice`/`etch` (dashboards + charts), `a11y`.

## Design agenda (open)
- **User & job-to-be-done** — homeowner-first (decided — relatable to a non-specialist audience); the installer/advisor view is a *deferred* second surface over the same core (shape the seam, defer the room).
- **The `ballast` engine** — the constraint + objective model; the strategies; conserving/priority allocation for power (echoes `f8n`'s `allocate` discipline — a scarce resource split to sum exactly to a budget).
- **Data** — tariff (Octopus Agile/Go), solar irradiance/generation profiles, device load models; synthetic vs real open data. *(Verify current tariff / DNO / G98–G99 specifics before relying on them — fast-moving.)*
- **The simulation** — day/year modelling; what to visualise (amp-budget spend, schedule timeline, solar/tariff overlays, battery state, scenario what-ifs).
- **Scope guard** — build the brain + simulation + UI; **no live device integration** (that's the sinkhole).

## Change log
- 2026-08-22: created — scaffolded as a forge project dir (top-of-stack consumer product). **Energy** domain, **consumer-focused**, **visual** (dashboards and timelines). Constraint-solver engine named `ballast` (current-limiting — the mechanism is the name). Agenda open; scope held to brain + simulation + UI (no live hardware).
