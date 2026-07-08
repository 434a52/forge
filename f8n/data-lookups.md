# f8n — data & lookups

How `f8n` handles reference/rate **data**: what it ships, how consumers bring their own, and the effective-dated lookup mechanism. Split out of `DESIGN.md` (the core money/rate model) because it grew into its own topic. See `DESIGN.md` for `Money`/`Percentage`/`InterestRate`/`ExchangeRate`/`TaxRate` and the fixed-point representation model.

## Who owns what
| | **`f8n` provides** | **Consumer owns (their repo)** |
|---|---|---|
| **Types + behaviour** | Library dep (NuGet/npm): `Money`, `Percentage`, `TaxRate`, `ExchangeRate`, `EffectiveDated<>`, resolver, validation | — |
| **Tooling** | The `c5n` codegen (Go binary; MSBuild target for C#, Vite plugin/CLI for TS) + the row **schema/instructions** | — |
| **Raw data** | Only a static, non-authoritative *sample* (e.g. GBR VAT bands) | **The real data** — their YAML; their source of truth, review, update cadence |
| **Generated code** | — | **The named accessors** (`GbrVat.Standard`, …) — thin, typed constants **of `f8n` types**, over their data (checked-in or generated-on-build) |

**The payoff:** `f8n` ships ~zero data → no staleness, no maintenance, no compliance liability; the data lives with whoever owns it; yet the consumer still gets compile-safe, doc-commented, tree-shakable, cross-language-conformant access. The generated container/members are theirs; the *element types* they're made of are `f8n`'s (via the lib dep) — which is exactly why they drop into `f8n`'s ops for free.

There's also a pure-**runtime** path (load rows into the in-memory `EffectiveDated` impl without codegen) for data that can't be baked (live FX). Codegen = the blessed compile-time path; runtime = the escape hatch; **same interface either way.**

## Building blocks, not an engine
`f8n` provides the **building blocks**, never the tax/finance *engine* (income/corp/CGT/payroll are stateful calculators — bands, allowances, reliefs — and live in an application, not here). On the data side that means: **ship what's canonical, make it trivial to bring what isn't.**

- **ISO identity — shipped.** Country/currency reference data (canonical, stable) is `f8n`'s to provide.
- **Rate values — supplied, via an interface `f8n` understands.** One shared **effective-dated lookup** — `EffectiveDated<Key, Value>` with `resolve(key, date) → value` — serves *every* rate: tax key `(jurisdiction, type, category)`, FX key `(from, to)`, interest, etc. **Dependency inversion at the data boundary:** `f8n` depends on the interface; the consumer provides the data. This is where the dating lives — so the rate types stay **timeless values** (no `Dated<T>` bolted on each).
- **`f8n` ships (both C# + TS):** the interface, an easy **in-memory implementation** you populate from rows `{key, value, validFrom, validTo?}`, and **validation**. Consumer brings rows; `f8n` resolves + applies to `Money`.

**Direct value is first-class; the lookup is optional sugar.** `Money.convert(rate)` / `Money.taxOf(rate)` take a bare rate value (spot rate, test value, a number off an API) — **no time-series required**. `EffectiveDated` is *one way* to obtain a rate (resolve-by-date over a preloaded series), not a gate. Bypass it and the rate's timing/correctness is the caller's concern; `f8n` just does the math. (Some FX is stable enough to preload; we support that but never mandate it.)

**Resolver contract** (a shared-semantics surface, like the money math — golden-vectored, conformant C#/TS):
- **Half-open intervals `[validFrom, validTo)`** — no boundary overlap; open `validTo` = "current".
- **Gaps → fail-closed** — a date no row covers → throw / `None`, **never** nearest-match. (Silently returning a wrong rate is the cardinal money bug.)
- **Overlaps → rejected at load**, not at query — ambiguity caught when the series is built.
- **`f8n` guarantees resolution *integrity*, not value *correctness*** — it proves the series is dated, non-overlapping, resolvable; it can't vouch the `20%` is right. **Liability stays with the data supplier** — which keeps `f8n` out of the compliance firing line.
- **Sync + on-device by default** (loaded rows — matches the no-API north star); a consumer *may* back the same interface with an async/service provider (live FX) without `f8n`'s own path leaving the device.

## Consumer data → typed accessors (how the "ref sticks")
The consumer authors their data (YAML) and `f8n`'s **build tool** turns it into a **typed, doc-commented object** in *their* project. The key move: **the tool generates named constants that are instances of `f8n`'s own types — not a new type.** The container is the consumer's; the elements are `f8n.TaxRate` / `f8n.ExchangeRate` values.

- **Dependency points one way** — the generated code references `f8n`; `f8n` never references it. So passing `GbrVat.Standard` into `money.taxOf(...)` "just works" — it *is* an `f8n` type. Nothing to register, no ambient binding; the value-based API means `f8n` never holds the ref. (Cf. a generated `Colors.Primary : Color` dropping into any `f(Color)`.)
- **What the consumer gets:** compile-safety (typo'd band → compile error, `.` lists the bands), **doc comments** (from a YAML `doc:` field, and the codegen *auto-appends* the rate + effective date — machine-generated docs, on-brand), and it's an `f8n` type so every money op takes it.
- **Build-graph mechanism (via `c5n`, one binary both sides):** C# = an **MSBuild target** invoking `c5n` (emits accessors into the build — nothing checked in if generated-on-build); TS = a **Vite plugin / CLI** emitting a `.ts` module — with a **checked-in (reviewable, can drift) vs generated-on-build (fresh, extra step)** trade-off. Both land on "a typed symbol you import". *(Same invocation model both sides — `c5n`'s thin-wrapper design; **not** a Roslyn source generator, which is C#-only and was rejected — see `../c5n/DESIGN.md`.)* `f8n` supplies the row schema; `c5n` does the emit; the *output* is the ref, living in consumer code.
- **`f8n` is thus a build-time dependency in the consumer's build** (highest-risk category per `design-rigour`) — mitigated because it's the consumer's own data and the generator must be **deterministic + output reviewable**.

## Timeless value vs named binding
Sharpening "rates are dated facts": a **rate *value*** (`0%`, `20%`) is **timeless**; a ***named* rate** (`Standard` / `FullRate`) is a **name→value binding that moves over time** (`FullRate`: 17.5% → 20%). **The series attaches to the *name*, not the value** — the named rate *is* the lookup entry.

- **Uniform type, not heterogeneous.** Every named rate generates as `EffectiveDated<TaxRate>`, *including* constants. Do **not** type constants as bare `TaxRate` and temporals as `EffectiveDated<TaxRate>`: the day a "constant" gains history — **a government decision you don't control** — its generated type would flip and **break every call site**. Uniform series ⇒ **adding history is only ever adding rows**, never a breaking change.
- **The exception lives only in authoring.** A constant is authored without dates (`ZeroRated: 0%`); the codegen **normalises it to a degenerate one-entry, open-ended series**. Ergonomic to write, uniform to consume. (`ZeroRated` is the one *definitionally* constant band — a non-zero "zero band" is a contradiction — but it still generates as a trivial series to keep the call site uniform.)

## As-of API — "today" made nice *and* correct
- **The trap:** do **not** implement "current"/"latest" as *the newest row*. Tax changes are routinely **announced ahead and future-dated** (pre-loaded), so the newest entry is often *not yet in effect*. "Current" must mean **resolve as of an actual date**, never "last entry".
- **The sugar:** the op takes the series + date and resolves internally — `money.taxOf(GbrVat.Standard, taxPoint)` — one call, explicit date, deterministic; the value-first-class overload `taxOf(rate)` stays. Multi-line invoice → **seed the date once**: `money.asOf(taxPoint)` returns a basis you apply rates through (explicit seed, *not* ambient/hidden-clock magic).
- **Default "now" if omitted** — sourced from an **injected clock** (`TimeProvider` / a `now()` fn), *never* the static `DateTime.UtcNow` (testability; pinned "as-of" scenarios). A `.current`, if offered, is defined as `.on(clock.now())` — never newest-row.

## Temporal model — civil dates vs instants
There are **two temporal kinds**, and the stored ISO-8601 shape encodes which:
- **Tax / policy boundaries = civil dates.** A "VAT effective 2011-07-01" is a *jurisdiction-local civil date*, not a UTC instant. Store ISO-8601 **date** form (`2011-07-01`, no `Z`) → parse **`DateOnly`** → **civil-to-civil compare, no tz**. Modelling it as a UTC instant introduces a **boundary-offset bug**: a sale at `2011-06-30T23:30Z` is UK-civil `2011-07-01 00:30 BST` → new rate, but raw-UTC-date says 30 June → old rate. (UK VAT has historically changed in GMT winter so the UK dodges it; **1-July fiscal-year changes are common elsewhere** — the bug is real.)
- **FX / intraday = instants.** Store ISO-8601 **`…Z`** → parse **`DateTimeOffset`**/UTC → instant compare. UTC is correct here.
- **The data declares the semantic** (time+`Z` ⇒ instant; bare date ⇒ civil); the resolver is generic over the time coordinate. Never parse the *same* value both ways — that inconsistency is where boundary bugs breed.
- **Timezone is derived, never a call parameter.** The jurisdiction is in the rate's key (`GbrVat.Standard` = GBR); the resolver derives `GBR → Europe/London` from a tiny **country→civil-tz-id mapping** (CLDR/IANA — fits the canonical pipeline), and delegates the actual instant↔civil conversion to the **platform** (`TimeZoneInfo` / `Intl`/`Temporal`). `f8n` ships a mapping, not a tz database. The tz only engages when the caller passes an *instant* or omits the date (project "now" → jurisdiction civil day); a `DateOnly` tax point needs no tz at all.
- **Multi-tz countries** (US, Russia, Australia) have no single civil tz — but those are the same sub-national-jurisdiction cases already deferred; not a new problem.

## Change log
- 2026-07-06: **mechanism fix** — consumer-data codegen is **`c5n`** (Go binary; MSBuild target for C#, Vite plugin/CLI for TS), not a Roslyn source generator (rejected: C#-only → twin generator). Pattern unchanged (typed constants of `f8n` types, one-way dep); only the tool.
- 2026-07-02: split out of `DESIGN.md` — full data/lookups content moved here verbatim; `DESIGN.md` keeps a summary + link.
