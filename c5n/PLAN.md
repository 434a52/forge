# c5n — implementation plan

The *what next*, ordered by dependency. `DESIGN.md` holds the *why*; this holds the
sequence and doubles as session handover — the next session starts at the first unchecked
step. Mark `✓` in place as steps land.

**Where it stands:** `build` and `check` work end to end for `table<T>`, and f8n's
Currency/Country slice is generated, committed, and green in all four gates. The engine
emits **two of the three value shapes** and **one of the four collection kinds**.

## Phase 0 — bootstrap ✓

- ✓ CLI shell — `version`, `build [dir]`, `check [dir]`
- ✓ schema reader — the typed IDL, parsed via `yaml.Node` so declaration order survives
  (field order is ctor-arg order)
- ✓ data reader — values carried as **authored source text**, never decoded through `any`
  (decoding routes every fractional value through `float64`, which silently alters a
  declared decimal before it is ever emitted)
- ✓ value-emitter — the **literal** and **reference** shapes
- ✓ `emit:` recipes — per-target construction override; `{field}` substitutes the *fully
  emitted* argument expression, so a recipe chooses the shape of the call, never the
  spelling of a literal
- ✓ C# and TS writers for `table<T>`, each with its co-existence convention
  (`partial class` / a `*.data.ts` module the hand-written index imports)
- ✓ `c5n build` reproduces f8n's hand-written golden output byte-for-byte
- ✓ `c5n check` drift-guard — out-of-date, missing, and orphaned output

**Checkpoint:** `go test`, `c5n check`, `dotnet build`, `tsc --noEmit` all green on f8n's
Currency/Country slice.

## Phase 1 — gates and validation ✓

Cheap, and it is what stops a silently-wrong emit reaching committed output. Steps 1.2–1.4
are each a Go test plus an error path; no emitter change.

- ✓ **1.1 — CI** (`.github/workflows/ci.yml`). Runs the four gates on push and PR: one
  `engine` job (`go test` + the drift-guard, asking *do the sources still produce the
  committed output?*) and one job per target (*does the committed output still compile?*).
  Actions pinned by commit SHA — build-time tooling is the highest-risk supply-chain
  surface — with `.github/dependabot.yml` supplying the update path that pinning otherwise
  removes. The two target jobs become a real `strategy.matrix` once the vector runner gives
  them a shared command.
- ✓ **1.2 — reject undeclared fields in a data row.** The writers walk only the *declared*
  fields, so a field the schema has no place for — added by an author expecting it to
  appear, or left behind by a rename — was dropped in complete silence, with the output
  regenerating and compiling without it. Same class as the `float64` bug: invisible on the
  page, wrong in the artefact. A *misspelled* key was already caught, but by the absence it
  left ("field capitalTz: missing from row"), which names the field spelled correctly and
  not the one that is wrong. Validation now runs once over schema + data before any writer
  sees them, reports every problem in one pass, and names the file, the row by its key, the
  offending field, and what is declared.
- ✓ **1.3 — resolve references inside c5n.** A `defaultCurrency: XXX` with no matching row
  was emitted verbatim as `Currency.XXX`, so the first thing to notice was the target's
  compiler — meaning it was caught only if someone compiled, reported against generated code
  rather than the data file that is wrong, and reported once per language. c5n holds every
  table in memory, so it answers this itself and names where the real identities live.
- ✓ **1.4 — key uniqueness.** Checked per *type* rather than per file: two files declaring
  one identity is the same collision as one file doing it twice, and the harder one to see,
  since neither file looks wrong alone. A row missing its key field is rejected too — it has
  no name to emit a constant under.

**Checkpoint reached.** Each failure produces a c5n error naming file, row and field, all
problems reported in one run, and the ordering that makes the diagnosis good is pinned by an
integration test through `generate` — remove the validation call and that test fails with
the degraded message, rather than passing quietly.

## Phase 2 — the tax-rate slice

f8n's next data file (`data/tax/gb-vat.yaml`, worked through in `DESIGN.md` →
*Generation model*) needs exactly four engine capabilities that do not exist — and nothing
else. One vertical slice, ordered by dependency; every step ends with both targets
compiling.

- ✓ **2.0a — rate authoring form.** Resolved: data authors the percent number the source
  document states (`rate: 17.5`) and the `emit:` recipe names the unit
  (`Percentage.FromPercent`); `Parse` stays the canonical wire form. See `DESIGN.md` →
  *Open questions* for what was rejected and why, and for the standing rule it generalises
  to — a bare number needing a unit or scale is constructed **by name**.
- ✓ **2.0b — output is named for what it declares.** A `table<T>` emits one unit per type,
  merging every data file that feeds it, with rows in source order and the header naming all
  of them; `EffectiveDated` will emit one unit per named series at 2.4. Replaces the previous
  path-per-type-name behaviour, under which two files declaring one type silently overwrote
  each other and the drift-guard then failed immediately after a clean build. A duplicate
  output path is now an error rather than a race between writes.
- [ ] **2.1 — nested ctor + `Percentage`.** The third value shape: the field's declared type
  is constructible, so the emitter recurses. This is the conformance-critical heart — a
  wrong expression here is wrong data in every target at once, and it is what the golden
  vectors must pin hardest. Needs f8n's hand-written `Percentage` (an exact `Rational`) in
  C# and TS: parse, canonical form, equality — not yet the arithmetic.
- [ ] **2.2 — enums.** The first type c5n emits a **body** for rather than instances. Forces
  the member-normalisation question `DESIGN.md` lists: how data's `standard` becomes
  `TaxCategory.Standard` in C#, and what the TS spelling is (`enum` vs a string-literal
  union).
- [ ] **2.3 — `common:`-hoisting.** Merge `common ⊕ row` at the data layer; emitted output is
  identical to writing every field out. A reader change with no emitter change — third
  because 2.1 and 2.2 settle what a row is.
- [ ] **2.4 — `EffectiveDated<T>`.** The second collection kind. The envelope/value split is
  driven by the declared type — a row's `from:` is the envelope *because the type said so*,
  and a missing or wrong key is a validation error, never a guess. Needs a hand-written
  `EffectiveDated<T>` on both sides (minimal as-of lookup only). Note the data shape departs
  from `table<T>`: several **named series** per file, not one `type:` + `items:`.

**Checkpoint:** `gb-vat.yaml` generates, compiles and typechecks in both targets, drift-guard
green.

## Phase 3 — seed the spec

The prose spec is the oracle (`DESIGN.md` → *Specification as the oracle*), and the seam it
rests on is *worked examples accreted as the edges are derived, while the reasoning is
fresh*. That seam needs a file to accrete **into** before Phase 2 derives its first rules;
writing it afterwards is archaeology, which is the accepted debt `DESIGN.md` already names.
Runs alongside Phase 2, not after it.

- [ ] **3.1 — create the spec**, first section: `Rational` canonical form, decimal→rational
  exactness, and exactly what `Percentage.Parse` accepts. Written with 2.1.
- [ ] **3.2 — hand-derive the boundary vectors for those rules** into the spec as worked
  examples, each checked against the authority (here, the maths). A fully-specified worked
  example *is* a golden vector.

**Checkpoint:** a clean session, given only the spec, reproduces every worked example
exactly. If it cannot, the spec is underspecified — that is the test, and a disagreement is
signal about precision rather than arithmetic.

## Rooms — deferred, additive, no rework

Listed so they read as *chosen*, not forgotten. Each backfills without touching what
Phases 1–3 build (`DESIGN.md` → *Build order & what's deferrable*).

- **Conformance runner** — the uniform Go driver plus per-language `run-vector` CLIs. The
  direction is already chosen; it is what turns the target CI jobs into a real matrix, and
  it doubles as the third-party audit tool.
- **Template-bundle refactor of the emitters.** In-tree writers may carry Go logic by
  design; the pure-template bar exists for *third-party* bundles, and no third-party bundle
  consumer exists yet.
- **Distribution** — npm/NuGet wrappers, the MSBuild target, the Vite plugin, signing,
  reproducible-build attestation. Waits for a consumer outside this repo.
- **`tree<T>`, `fromJson`, validation emit, contract identity, rule identity** — designed
  and consumer-driven; l10n and portfolio pull these in, not f8n.
- **Swift, and any third target.**
- [ ] **Exact toolchain pinning in CI** — *outstanding*. The target jobs currently ask only
  *does it compile*, and a feature band (`10.0.x`, `22.x`) answers that. Once vectors run,
  the toolchain version becomes part of the result — a runtime-library bump can move
  formatted output, which is why `DESIGN.md` files pinning under **correctness, not
  hygiene**. **Do it in the same pass as the conformance runner**, not before: pinning
  exactly while nothing tests behaviour buys nothing and rots into manual bumps. Go needs
  nothing — `go-version-file` already defers to `go.mod`, which cannot drift from the module
  it builds.

## Change log
- 2026-08-25: **Phase 1 complete.** Validation is one pass over schema + data ahead of every
  writer, holding three checks that share a key index: undeclared fields, identities declared
  twice, and references naming a row nothing declares. Reference and uniqueness checking were
  the ones the target compiler used to do — late, against the wrong artefact, once per
  language. Added an integration test through `generate` pinning that validation runs *before*
  the writers, since that ordering is what makes a misspelled key report the misspelling
  rather than the absence it leaves behind.
- 2026-08-25: **Dependabot for the SHA-pinned actions**, and the Node pin loosened to a
  feature band to match .NET — the *compile-only doesn't need exactness* argument applies
  equally to both, and the lockfile plus `npm ci` is what actually fixes the TypeScript
  version. **Exact toolchain pinning is now tracked as an outstanding task** under Rooms,
  with its trigger named: the conformance runner, same pass.
- 2026-08-25: created. Records Phase 0 as landed, and sequences the next work: validation
  gates (Phase 1), the tax-rate slice that the next f8n data file requires (Phase 2), and
  the spec seed that must run alongside it (Phase 3). Two defects found while surveying the
  engine are captured as 1.2 and 1.3, and two design questions raised by the slice are
  routed to `DESIGN.md` → *Open questions* rather than decided here.
