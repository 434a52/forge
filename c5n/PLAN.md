# c5n — implementation plan

The *what next*, ordered by dependency. `DESIGN.md` holds the *why*; this holds the
sequence and doubles as session handover — the next session starts at the first unchecked
step. Mark `✓` in place as steps land.

**Where it stands:** `build` and `check` work end to end for `table<T>`, and f8n's
Currency/Country slice is generated, committed, and green in every gate. The engine
emits **all three value shapes** and **two of the four collection kinds**, and it emits a
type **body** (enums, from the schema alone) as well as instances of hand-written types.
**Phase 2 is complete** — the UK VAT series is generated, committed and compiling in both
targets. Three vector datasets (67 cases) run in CI across both languages, and the net has
been seen to fail: a deliberately naive leap-year rule turned `localdate` red in C# while TS
stayed green.

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
- ✓ **2.1a — nested ctor (the third value shape).** The field's declared type is
  constructible, so the emitter recurses. A **one-field type may be authored as a plain
  scalar** (`rate: 17.5`, not `rate: { value: 17.5 }`) — the wrapper case, and what keeps
  data reading like the document it came from; more than one field needs a mapping, since a
  lone scalar would otherwise be a guess. The three shapes are now one shared recursive
  resolver rather than a branch per writer, which is the point: a wrong expression here is
  wrong data in every target at once. TS additionally imports the hand-written types the
  values construct; C# needs nothing, sharing the namespace via `partial`.
- ✓ **2.1b — `Percentage`, hand-written in C# and TS.** An exact `Rational` over big
  integers: `FromPercent`, `FromProportion`, canonical `num/den`, equality, and a strict
  `Parse` that rejects a non-canonical encoding so a value has exactly one spelling. Both
  decimal parsers are **character walks rather than regexes** — two regex engines can differ
  in dialect at the edges, and this grammar has to mean the same thing in every target, which
  a loop is by construction. C# additionally takes `decimal`, routed through the string form
  so there is one parsing implementation rather than two that could drift; TS rejects a
  `number` in the type *and* at runtime, since JavaScript callers reach it too.
- ✓ **2.2 — enums.** The first type c5n emits a **body** for rather than instances, and the
  first unit emitted from the **schema alone** — every unit before this derived from a data
  file, so `generate` gained a second emission pass. **Members are declared, not drawn from
  data:** a value selects a member and can never create one, which matters because an enum
  serialises as text, so a minted member is a minted wire token. That also **dissolves** the
  member-normalisation question rather than answering it — the declared name is emitted
  verbatim in both targets, the only rule under which `VAT` survives as `VAT`; c5n checks a
  member's *shape* (a legal identifier everywhere, catching `zero-rated`) and leaves target
  keywords to the target compiler. **TS is a const object plus a union of its values**, not a
  TS `enum`: `TaxCategory.Standard` then reads identically in both targets, so the shared
  resolver keeps one reference spelling, and the TS runtime value *is* the token C#
  serialises. Landed in f8n as `TaxType` and `TaxCategory`, generated with no data file
  referencing them yet — which is the emission path being exercised, not a placeholder.
- ✓ **2.3 — `common:`-hoisting.** Any field constant across a collection lifts to `common:`;
  each row carries only what varies. **No emitter change** — the merge happens before any
  writer sees a row, and the output is identical to writing every field out, which is the
  whole claim and so is pinned byte-for-byte against a written-out twin. Three rules decided
  with it: a row that also sets a hoisted field is an **error, not a cascade** (the
  post-agent inversion — leniency was a trade against human keystrokes, and an agent expands
  the rows for free, so only the ambiguity is left; it is also the reversible direction);
  **hoisting the identity is rejected**, since a key varies by definition; and the **merge
  runs after validation**, so a mistake in `common:` is reported once against `common:`
  rather than once per row it was copied into. No f8n data uses it until 2.4 — a consequence
  of the ordering, so it is covered by engine tests until the series arrives.
- ✓ **2.4a — `EffectiveDated<T>`, the engine half.** The second collection kind. The
  envelope/value split comes from a schema declaration (`kind: series` + `envelope:`), so a
  row's `from:` is the envelope *because the type said so* and an entry without it is an
  error naming the declaration — c5n never learns that a series keys on `from`. A data file
  now holds **several named collections** (`type:` at the top level distinguishes the two
  shapes), and the unit is the **series**, not the value type: `VatStandard.g.cs`, since one
  file holds many `EffectiveDated<TaxRate>`. Settled with it: **one spelling, `items:`**
  (this replaces the design's earlier `rows:`); a series **recipe is required** and takes
  the reserved `{entries}`; the **envelope cannot be hoisted** to `common:`. **No new scalar
  was needed** — the date is an external type with a parse recipe, so the temporal design
  stays in f8n.
- ✓ **2.4b — the f8n half.** `LocalDate`, `EffectiveDated<T>` and `TaxRate`, hand-written in
  both languages, and `data/tax/gb-vat.yaml` generating, compiling and typechecking. Two
  conformance surfaces came with them and both are pinned: **`LocalDate.parse`** — the ISO
  8601 calendar date, character-walked rather than regexed, 26 cases of which 19 are
  rejections, including Arabic-Indic digits, which `char.IsDigit` accepts and JavaScript's
  `\d` does not (the obvious implementation in each language disagrees there); and
  **`EffectiveDated.asOf`** — the inclusive boundary, where being off by one makes every rate
  wrong for exactly one day in one language. The runner now dispatches on the vector file's
  `subject`, so one runner per language covers every f8n type instead of one binary each.
  Deliberately partial: the VAT series carries the two entries the design worked through,
  not the full history, which is a data task with an authority attached.

**Checkpoint reached.** `gb-vat.yaml` generates, compiles and typechecks in both targets,
drift-guard green, and 67 vector cases across three datasets agree in both languages. Phase 2
is done: the tax slice is real, and every capability it needed — nested constructors, enums,
`common:`-hoisting, the series — is exercised by committed data rather than by tests alone.

## Phase 3 — the vector dataset and its runners

Runs **alongside Phase 2, from 2.1 onwards** — not after it. The dataset is the parity net,
and the whole point of building it now is to have it *while* the conformance-critical
behaviour is written rather than retrofitted around it. The first vectors are deliberately
the rational parse rather than money math, so the harness is shaped on an easy case.

- ✓ **3.1 — the vector format, and what it targets.** A language-neutral file of
  input → expected. A **published contract from the first vector**, since third-party runners
  read it; JSON rather than YAML, so a runner needs no parser dependency in either language.

  Vectors target **component functions**, never compositions: `divide-with-rounding` per
  mode, `allocate` per rule, `Rational` normalise/parse/canonical, minor↔major conversion at
  a scale, the wire grammar, the ISO subset. Not "VAT on an invoice" — per-line-vs-per-invoice
  and the mandated rounding are the *caller's* choices, which is exactly why `RoundingMode` is
  caller-supplied and `AllocationRule` has no default. A rule f8n refuses to choose is not one
  its vectors can encode.

  Four kinds of case, and the distinction matters when reading a failure:

  | kind | source | notes |
  |---|---|---|
  | bulk | generated from C#, frozen on capture | a later behaviour change goes red, and someone decides which value was right |
  | boundary | generated too — *chosen* for coverage, not derived differently (ties, zero, sign, scale limits, seams) | |
  | reject | first-class for the grammars, where the standard sets the boundary | cite the standard |
  | authority example | transcribed from a published worked example | marked as such: its value originates outside our code |

  Plus **properties**, which carry no expected value at all and so depend on no
  implementation: `sum(allocate(m, rule)) == m`, `allocate(−m) == −allocate(m)`,
  `FromPercent("17.5") == FromProportion("0.175")`, `parse(canonical(x)) == x`. Several are
  already stated as invariants in `../f8n/DESIGN.md` and are sitting there unused as checks.

  **Standing rule:** an implementation is written from the rule, **never from the dataset**.
  That is what keeps captured values audited — an independently-written TS turns a captured
  C# slip red. Generate the TS implementation from the vectors and that check is gone.

  *Landed as `f8n/vectors/percentage.json` — 35 cases, 15 of them rejections. Two of the four
  kinds are exercised so far (bulk and reject); no transcribed authority example exists yet,
  since `Percentage` has no external authority to transcribe from — the first will arrive with
  a rate or a format grammar.*
- ✓ **3.1b — properties.** Invariants asserted in both languages, carrying no expected value
  and so depending on no implementation: `sum(allocate(m, rule)) == m`,
  `allocate(−m) == −allocate(m)`, `FromPercent("17.5") == FromProportion("0.175")`,
  `parse(canonical(x)) == x`. Several are already stated as invariants in `../f8n/DESIGN.md`
  and sit there unused. They catch classes of error a captured vector cannot, because they do
  not encode a value — and they are the *independent derivation path*, which is the thing
  capturing from C# does not give us.

  *Landed for what exists: `fromPercent(x) == fromProportion(x/100)` over eight pairs
  including the `0.5%` → `0.005` trap, canonical round-trips for `Percentage` and
  `LocalDate`, `compareTo` as a total order, and — for `EffectiveDated` — every entry being
  in effect on its own start date, plus **order independence**, which pins the claim that the
  type sorts its own entries and was otherwise untested. A failing property reports what
  broke rather than "false". The `allocate` invariants listed above **landed 2026-08-27**
  with `Money`'s conserving partition — see `../f8n/vectors/allocation.json`. Seen to fail: comparing day before month turned `prop-date-order` red in C# alone,
  with the message `2010-01-02 does not sort before 2010-02-01`.*
- ✓ **3.2 — a `run-vector` CLI per language.** Thin by design: read the dataset, execute
  each case, report what it got. No assertions, no test framework, and it ignores any
  expected values in the file — which is what stops a runner grading its own work. An
  unknown op exits non-zero rather than reporting a case error, or a reject case would
  "pass" for entirely the wrong reason.
- ✓ **3.3 — the uniform Go driver** (`c5n/cmd/conform`). Runs each language's CLI, compares
  against the dataset, reports every divergence with wanted-versus-got. `-capture <runner>`
  writes a runner's results back as the expectations and **says what changed** — a capture
  over an existing value means behaviour moved, which is a decision, not a refresh. A
  rejection is recorded as a bare `true`, never the message: each language words its own,
  and pinning the text would make a reworded error a breaking change.
- ✓ **3.4 — wired into CI** as a `conform` job.
- ✓ **3.5 — resolved, and both halves of it changed shape.**

  **The matrix is dropped, not deferred.** Its premise — "they finally share a command" —
  turned out false. `csharp` runs `dotnet build` behind `setup-dotnet`; `ts` runs
  `npm ci && tsc` behind `setup-node`. The vector runner never merged them; it created a
  *third* job, `conform`, which is where the two languages actually meet and is one job
  already. A matrix over the compile jobs would need a condition on every step, deduplicate
  nothing, and blur which target failed.

  **Exact toolchain pinning is re-aimed at its real trigger.** "The conformance runner
  landed" was necessary but not sufficient: an exact pin guards a *runtime-dependent result*,
  and f8n has none. Every parser is a character walk, every format is invariant-culture, the
  arithmetic is big-integer — the dependency a pin would guard has been engineered out rather
  than pinned around, which is the stronger form of the same guarantee. The trigger is the
  first behaviour that reads the runtime's locale data, which is **l10n's formatting**, where
  ICU is the subject rather than an accident. Pin exactly then, in the job that measures it.

  *Verified rather than assumed: the full dataset was re-run under `LC_ALL=tr_TR.UTF-8` and
  `de_DE.UTF-8` with an offset `TZ`, and every case held in both languages. Recorded as a
  **technique**, not a stage — it cannot fail today, and a test that cannot fail reads as
  coverage it does not provide. One latent dependency did turn up while checking and is
  fixed: `LocalDate.ToString` interpolated, which uses `CurrentCulture`, where `Percentage`
  had been explicitly invariant throughout.*

**Phase 3 complete.** Conformance runs in CI across both languages over every dataset in
`f8n/vectors/` — found by glob, not listed, since a vector file nobody runs is worse than
none. And the net **has now been seen to fail**: replacing the Gregorian leap rule with a
naive `year % 4` turned `leap-century-not-400` red in C# while TS stayed green, naming the
case and reporting wanted-versus-got. That was the outstanding condition on this checkpoint.

## Rooms — deferred, additive, no rework

Listed so they read as *chosen*, not forgotten. Each backfills without touching what
Phases 1–3 build (`DESIGN.md` → *Build order & what's deferrable*).

- **Native per-language harnesses** — running the same vectors from `dotnet test` / `vitest`
  for editor integration and familiar failure output. A convenience on top of the uniform
  driver (Phase 3), never a replacement: nothing may assume a native-only harness, or the
  neutral runner stops being the thing a third party can audit with.
- **Template-bundle refactor of the emitters.** In-tree writers may carry Go logic by
  design; the pure-template bar exists for *third-party* bundles, and no third-party bundle
  consumer exists yet.
- **Distribution** — npm/NuGet wrappers, the MSBuild target, the Vite plugin, signing,
  reproducible-build attestation. Waits for a consumer outside this repo.
- **`tree<T>`, `fromJson`, validation emit, contract identity, rule identity** — designed
  and consumer-driven; l10n and portfolio pull these in, not f8n.
- **Swift, and any third target.**
- **Exact toolchain pinning in CI** — *deferred to l10n, with the trigger corrected*.
  `DESIGN.md` files pinning under **correctness, not hygiene**, and that is right: what makes
  it correctness is a result the runtime can move. f8n has no such result — character-walk
  parsers, invariant formatting, big-integer arithmetic — so the guarantee is held by
  construction rather than by a version number, and an exact pin would rot into manual bumps
  (Dependabot manages the action SHAs, not these inputs) while guarding nothing. The trigger
  is **l10n's locale formatting**, where ICU is the subject; pin exactly there, in the job
  that measures it. Go needs nothing either way — `go-version-file` already defers to
  `go.mod`, which cannot drift from the module it builds.

## Change log
- 2026-08-26: **Phase 3 closed — properties landed, and 3.5 resolved by being wrong.**
  Properties assert what holds for every input, so the expected value comes from the rule
  rather than from an implementation, which is the independent derivation path a captured
  vector cannot provide. Eight percent/proportion pairs, round-trips, `compareTo` as a total
  order, and `EffectiveDated`'s order-independence — the last pinning a claim nothing else
  tested. Seen to fail, like the leap rule before it. **3.5's matrix was dropped rather than
  deferred:** its premise was that the vector runner would give the two compile jobs a shared
  command, and it did not — it created a third job where the languages actually meet, while
  the compile jobs still run different commands behind different setup actions. **Exact
  toolchain pinning was re-aimed:** the trigger is not "vectors exist" but "a result the
  runtime can move", and f8n deliberately has none, so the guarantee is held by construction.
  Confirmed under hostile locales, and recorded as a technique rather than a CI stage because
  it cannot currently fail. One real fix fell out of checking: `LocalDate.ToString` was
  interpolated and so `CurrentCulture`-sensitive, where `Percentage` had been invariant
  throughout.
- 2026-08-26: **2.4b done, and Phase 2 with it.** `LocalDate`, `EffectiveDated<T>` and
  `TaxRate` hand-written in both languages; `data/tax/gb-vat.yaml` generates, compiles and
  typechecks. Two new conformance surfaces, both pinned by vectors: the ISO 8601 date
  grammar (26 cases, 19 rejections) and the as-of boundary (6). The runner now dispatches on
  the vector file's `subject`, so one runner per language serves every type rather than one
  binary each, and CI globs `f8n/vectors/*.json` rather than naming files. **Phase 3's
  checkpoint is also reached:** the parity net was deliberately broken — a naive `year % 4`
  leap rule — and turned red in C# alone, which is the condition that checkpoint had been
  carrying unmet.
- 2026-08-26: **2.4 split into engine and f8n halves; 2.4a done.** The engine now reads and
  emits `EffectiveDated<T>` exactly as the design specifies, verified against a scratch
  project in both targets. Splitting it was the consequence of one decision: the envelope's
  date goes through an ordinary external type with a parse recipe rather than a new `date`
  scalar, so c5n needs no notion of time and the whole temporal design — the ISO-8601
  subset, its strict grammar, the conformance vectors it will need — stays in f8n as 2.4b,
  alongside the hand-written `EffectiveDated<T>` and `TaxRate` the generated code constructs.
- 2026-08-26: **2.3 done — `common:`-hoisting, overlap rejected.** The equivalence is the
  feature, so the test generates a hoisted file and a written-out twin and compares the
  output byte-for-byte in both targets. The design call is recorded in `DESIGN.md`: a row
  overriding a hoisted field errors rather than cascading, on the post-agent reasoning that
  an ergonomic leniency trades against human keystrokes and an agent expands rows for free.
  Merging moved *after* validation so a mistake in `common:` names `common:` once, and
  reference resolution now runs through one shared helper for a row and for `common:` alike.
- 2026-08-26: **2.2 done — enums, with members declared rather than drawn from data.** The
  structural half is that an enum is the first unit generated from the **schema alone**, so
  `generate` now runs a schema pass before the data pass and the output-path rule moved to
  one home shared by both. The design half is recorded in `DESIGN.md`: collected members
  would let a typo mint a member — and, since an enum serialises as text, a wire token with
  it — and would make the emitted API depend on data coverage. Declaring them **dissolves**
  the member-normalisation open question, since the declared name is the name in C#, in TS
  and on the wire. TS emits a const object plus a union of its values rather than a TS
  `enum`, so a reference reads identically in both targets. f8n gained `TaxType` and
  `TaxCategory`; every gate green, conformance included.
- 2026-08-25: **Phase 3 is the vector dataset and its runners, not a prose spec** — and it
  runs alongside Phase 2 rather than after it. The separate specification is cut (see
  `DESIGN.md`): it was carrying weight only because the runner had been deferred, and it
  restated the obvious for every rule whose expected value is self-evident. Rationale and
  authority citation now live **beside the vector** they explain. The conformance runner
  moves out of *Rooms* onto the critical path; what stays a room is the native per-language
  harness, as a convenience over the neutral driver. Also resolved **2.0b** — output is named
  for what it declares, so a `table<T>` emits one unit per type across however many files
  feed it.
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
  the vector dataset and runners that must run alongside it (Phase 3). Two defects found while surveying the
  engine are captured as 1.2 and 1.3, and two design questions raised by the slice are
  routed to `DESIGN.md` → *Open questions* rather than decided here.
