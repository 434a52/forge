# c5n — codegen tooling

**`c5n`** — numeronym for *codegen* (c·odege·n), in the `f8n`/`l10n`/`a11y` family. The build tool that turns **source data** — CLDR / `iso-codes` (`f8n`'s own canonical types) and consumer-authored YAML (their tax/FX/etc. data) — into **typed, cross-language-conformant code** (C#, TS, … Swift likely) plus the `EffectiveDated` accessors. A **central, publicly-visible portfolio piece** — its own quality is part of the demonstration.

> **Standalone shared engine.** `c5n` is its **own project** — `f8n`, `l10n`, and `doppel` all feed it templates + schemas; it is *not* an `f8n`-internal component. First consumer is `f8n` (see `../f8n/DESIGN.md`). Repo layout (one repo vs per-project) is a roadmap-level concern — resolved: monorepo.

## Decision: written in Go
Portability was the priority (it runs in *every* consumer's build), and no single runtime is pre-present in both a pure-.NET CI and a pure-Node CI. Go gives a **single self-contained static binary** — zero runtime added to any ecosystem, trivial cross-compilation, minimal deps — the esbuild / hugo / terraform pattern. New-language cost is manageable (small language; C#-fluent → productive in days; Claude-assisted). A pure-.NET enterprise consumer is a *solved special case* the binary also covers, so no second path is needed for it.

**Rejected:**
- **Roslyn source generator** — best C# DX, but inherently C#-only → needs a *twin* TS generator = two conformance-critical codebases (the divergence risk designed out everywhere else).
- **TS/Node core** — zero new languages, but forces Node into a pure-.NET CI. Fine for OSS, friction for enterprise; loses the zero-runtime property.

## Architecture: portable core + thin wrappers
- **Core (the Go binary)** — the conformance-critical transform: read source → **normalised model** → emit per-target code. Written once.
- **Wrappers** — per-ecosystem invocation only, no logic: MSBuild target (.NET), Vite plugin / npm script (TS), optional SHA-pinned GitHub Action. **Same invocation model both sides (a prebuild step)** → unifies C# and TS instead of making C# special (source generators would have made C# the asymmetric case).

## Multi-target emitter (the seam)
"More outputs likely (Swift, …)" ⇒ the emitter is **target-agnostic from day one**: a shared normalised model + **per-target templates** (`text/template`), never C#-and-TS hardcoded. **Build C# + TS now; defer the Swift/Kotlin/… rooms.** Adding a target = a template + a vector-runner + a CI matrix line — bounded work, not a rewrite. (Seam/room maxim: design multi-target, build two.)

## Generation model
The concrete elaboration of the seam. **Three inputs, one binary:** *schema* files (the types + how to build them), *data* files (typed instances), *project config* (which targets). c5n reads all three → a normalised in-memory model → per-target emit. The dividing line that makes it reusable: **c5n generates the minimal typed *boundary* — the object model and any typed surfaces; the consumer hand-writes the *algorithm* behind it.** Nothing in the pipeline is domain-specific — c5n knows only what the schema declares. *(The "generate data / hand-write behaviour" framing below is the f8n-shaped shadow of this; the l10n stress-test generalised it to "generate the typed boundary / hand-write the algorithm" — see **Stress-tested against l10n**.)*

### Generated builds the object model; hand-written is behaviour only
The generated code constructs fully-typed values (`new Money(…)`, `new TaxRate(…)`) — not raw primitives the consumer has to wrap. The consumer's hand-written half adds *logic* (money math, parsing, interpolation, plural selection) on top of those values, and nothing else. c5n emits ctor calls only for **types it already knows** — where *knows* = **the schema declared the type, including how to construct it in each target language**. So c5n stays domain-blind (it hardcodes no `Money`), yet the output is ready-to-use. Dependency direction is **one-way: generated code → hand-written runtime types** (the generated file references the hand-written `Money`; never the reverse).

The general form (l10n stress-test): the generated artefact is sometimes a value (f8n) and sometimes a **thin typed façade over a hand-written runtime call** (l10n message accessors) — either way it's *wiring*, and the *algorithm* it wires into is hand-written. The irreducible generated minimum is **the typed signature** (types are compile-time — they can't be data); everything behind it can be data + runtime.

### Schema — a small typed IDL (not JSON Schema)
Types are declared **once**, in their own file(s), separate from data. Not JSON Schema — that's a *validation* language (pattern/required/anyOf), maps badly to clean nominal types, and drags in a real dependency (against the "≈ just `yaml.v3`" surface). What we want is a tiny **IDL**, proto/TypeSpec-shaped: records, enums, scalars — closed and nominal. Every type is one uniform declaration = **fields + a construction recipe**; the only axis is whether c5n **emits the type** or merely **constructs/references** it:

- **`external: true`** — a hand-written runtime type (`Money`, `Percentage`, `Country`). c5n never emits the type body; it only needs the recipe to build instances.
- **generated** — c5n emits the type too (an enum, or a **record / model type body**). Same declaration; c5n also writes the definition. *(Emitting record bodies — not just enums and external-type instances — is the one capability the model/`fromJson` consumers add; see **Further consumers**.)*

```yaml
# schema/f8n.types.yaml — declared once; language-blind except the construction recipes

Percentage:                 # external: hand-written runtime primitive
  external: true
  fields: { value: decimal }
  emit:                     # override — the default ctor won't do (factory / literal suffix)
    csharp: "new Percentage({value}m)"   # note the C# decimal 'm' suffix
    ts:     "new Percentage({value})"

Country:                    # external; no emit: → positional-ctor *convention* (ctor args = fields)
  external: true
  key: alpha3               # identity → names Country.GBR; also the reference target
  fields: { alpha2: string, alpha3: string, numeric: int, name: string, callingCode: int,
            defaultCurrency: Currency, capitalTz: string }

Currency:                   # external; a table<Currency> → one constant per key (Currency.GBP)
  external: true
  key: code                 # identity → names Currency.GBP; also the reference target
  fields: { code: string, numeric: int, name: string, symbol: string, minorUnits: int }

TaxRate:                    # external; convention: new TaxRate(jurisdiction, taxType, category, rate)
  external: true
  fields: { jurisdiction: Country, taxType: TaxType, category: TaxCategory, rate: Percentage }

TaxType:     { kind: enum } # generated — emitted as C# enum / TS union; members drawn from data
TaxCategory: { kind: enum }
```

**Construction = convention-by-default + per-type override.** Most types get a positional-ctor convention the writer knows (`new T(f1, f2, …)`) — zero per-type config. Only the awkward ones (factories like `Money.of`, literal suffixes like `m`, singleton refs) carry an explicit `emit:` block, and it lives **beside the type** (all of `Percentage` in one place). Recipes are per-target-language. Cost of "add Swift" = a new writer **+ a `swift:` line on the few special types**, not every type.

**Identity = the `key:` field (`table<T>`).** A `table<T>` emits one named constant per row; `key:` names which declared field is the identity — it becomes the **constant's name** (`Country.GBR`, from `alpha3`) and the **reference target** (a `defaultCurrency: GBP` elsewhere resolves to `Currency.GBP` by matching Currency's `code`). The id is therefore an ordinary field in the ctor like any other — **no key-prepend magic** — and data is authored as a plain **list** of rows (the id appears once). *(A map authoring form — key implicit, id not a stored field — stays available for identities that aren't domain fields, e.g. l10n's namespace names; f8n uses the explicit `key:` + list form.)* Validation: the `key:` field must be **unique** across the table (the uniqueness the map form gave structurally).

### Data — positionally typed, `common`-hoisted
Data files stay clean: a collection binds to a schema type **once** (a top-level `type:`), and every nested field's type is looked up from the schema recursively. **No per-value type tags** — position carries the type. (The sole exception: a genuinely polymorphic field, whose declared type is a union — only there does a value carry a discriminator. Rare; zero cost when absent.)

```yaml
# data/currencies.yaml — a list of rows; identity is the `code` field (the schema key)
type: table<Currency>
items:
  - { code: GBP, numeric: 826, name: "Pound Sterling", symbol: "£", minorUnits: 2 }
  - { code: EUR, numeric: 978, name: "Euro",           symbol: "€", minorUnits: 2 }
```

```yaml
# data/countries.yaml — a list of rows; identity is the `alpha3` field (the schema key)
type: table<Country>
items:
  - { alpha2: GB, alpha3: GBR, numeric: 826, name: "United Kingdom", callingCode: 44, defaultCurrency: GBP, capitalTz: Europe/London }
  - { alpha2: FR, alpha3: FRA, numeric: 250, name: "France",         callingCode: 33, defaultCurrency: EUR, capitalTz: Europe/Paris }
```

```yaml
# data/tax/gb-vat.yaml — a series; the invariant identity is hoisted to `common`, not per-row
VatStandard:
  type: EffectiveDated<TaxRate>
  common: { jurisdiction: GBR, taxType: VAT, category: standard }
  rows:
    - { from: 2011-01-04, rate: 0.20 }
    - { from: 2010-01-01, rate: 0.175 }
```

**`common`-hoisting** is a general affordance (not tax-specific): any field constant across a collection lifts to `common:`; each row carries only what varies; c5n merges `common ⊕ row` when constructing the value. Emitted code is identical to writing every field out. It's where much of the "data stays clean" ergonomics lives across all three consumers.

### Collection kinds — temporality is *declared*, never sniffed
A small fixed set of collection kinds, selected by the declared `type:`:

| kind | shape | envelope | emitted |
|---|---|---|---|
| `list<T>` | ordered values | none | array of `T` |
| `table<T>` | keyed by identity | none | one constant per key-field value (`Country.GBR`) |
| `tree<T>` | recursively nested, keyed | none | nested scopes *or* nested value (by leaf type) |
| `EffectiveDated<T>` | temporal series | key field **declared by the type** | as-of series |

**`tree<T>` is the recursive generalisation of `table<T>`** — a node is a keyed branch or a leaf `T`, arbitrary depth. The *leaf type* decides emission: a **generated symbol** → nested *scopes* (namespaces/classes/modules — l10n's namespace tree, `l10n.account.open.button`); a **value** → a nested *data literal* (doppel's `UK › county › town`, CLDR hierarchies). One recursive walk; the per-leaf branch (symbol-def vs value-expr) is the one c5n already makes. (Fine-grained *file* partitioning for tree-shaking is a separate output-layout concern, not the `tree<T>` kind itself.)

**"Is it a time series?" = "did you declare `EffectiveDated`?"** — nothing in the data file signals it. `EffectiveDated`'s *type declaration* names its key field (`from`), so a row's `from:` is the envelope **because the type said so**; a row that omits it or uses a wrong key is a **validation error**, not a guess. The envelope/value split (which fields key the series vs. construct the `T`) is entirely driven by the declared type. (`EffectiveDated<Key,Value>` is an external f8n runtime type — see `../f8n/data-lookups.md`.)

### The value-emitter (the conformance-critical heart)
For each field the emitter resolves, **driven purely by the declared field type**, to one of three shapes:

- **literal** — scalar → per-language literal formatting (`0.20m`, `826`, a `DateOnly(…)`);
- **reference** — the value matches an enum member or a table row's **key-field value** → emit the reference (`Currency.GBP`, `TaxType.VAT`), not a fresh ctor;
- **nested ctor** — the field's type is a constructible type → recurse (`new Percentage(0.20m)`).

This resolver is target-independent and **shared core** — it is the one logic-bearing part, and where correctness lives (a wrong ctor is wrong data *everywhere* it's emitted), so it is what the golden vectors must pin hardest.

Emitted C# from the data above (generated `partial`s; hand-written `Country.cs` / `Money.cs` hold behaviour):

```csharp
// Country.g.cs — constant named from the key field (alpha3); ctor args = declared fields, in order
public partial class Country {
    public static readonly Country GBR = new Country("GB", "GBR", 826, "United Kingdom", 44, Currency.GBP, "Europe/London");
    public static readonly Country FRA = new Country("FR", "FRA", 250, "France", 33, Currency.EUR, "Europe/Paris");
}
// TaxRates.g.cs — common merged into each row; reference + nested-ctor + literal all visible
public static class VatStandard {
    public static readonly EffectiveDated<TaxRate> Series = EffectiveDated.Of(
        (new DateOnly(2011, 1, 4), new TaxRate(Country.GBR, TaxType.VAT, TaxCategory.Standard, new Percentage(0.20m))),
        (new DateOnly(2010, 1, 1), new TaxRate(Country.GBR, TaxType.VAT, TaxCategory.Standard, new Percentage(0.175m))));
}
```

The TS writer emits the same data against the same hand-written types, differing only in idiom and co-existence convention:

```ts
// country.data.ts (generated) — imports the hand-written Country class, exports instances
export const GBR = new Country("GB", "GBR", 826, "United Kingdom", 44, Currency.GBP, "Europe/London");
```

### Target selection — explicit, at the *project* boundary
Targets are chosen **only** in a project-root `c5n.yaml`, never in the data (which stays language-blind). One `c5n build` emits **all configured targets together**; the per-ecosystem wrappers (MSBuild target, Vite plugin) just *trigger* the same build — they don't select. A pure-.NET consumer lists one target; f8n's own repo lists both.

```yaml
# c5n.yaml — the ONLY place emitters/targets are selected
targets:
  csharp: { out: src/Generated/, namespace: F8n.Data }
  ts:     { out: packages/f8n/src/generated/ }
sources:
  schema: schema/*.yaml
  data:   data/**/*.yaml
```

Why explicit *and* central: this config is the **manifest the drift-guard checks against.** "Regenerate into a temp and `git diff --exit-code`" only works if there's one authoritative statement of the complete expected output — all targets, all paths. Scatter target-selection (or infer it from which wrapper ran) and the drift-guard can't know the full set. (Per-file `targets: [csharp]` restriction stays available as an *override* for genuine exceptions — a C#-only helper — but the default is "all project targets.") Config stays deliberately minimal for now; more knobs get added when a real need appears, not speculatively.

### Emitters are template bundles
An emitter is **almost entirely declarative** — literal-formatting rules, the construction convention + `emit:` overrides, identifier casing/escaping, and the co-existence convention. The one logic-bearing piece (the value-emitter above) is shared core. So an emitter is a **template bundle** (`text/template` + a small funcmap + config) — *data, not code* — and c5n never dynamically loads a compiled library (Go's `plugin` package is ruled out: exact-version/flag-lock, no Windows, and it breaks the single-static-binary property the Go decision was made for). An emitter is *always* a bundle; only **provenance** differs, and the extension paths rank by trust:

1. **In-tree (blessed)** — C#/TS/Swift live in the c5n repo (embedded via `go:embed`), vetted, conformance-tested, compiled into the one signed binary. Highest trust.
2. **Third-party bundle** — ship a bundle, point c5n at it (`--emitter ./kotlin`). No fork, no c5n release, no Go — same format. Output is a reviewable diff the drift-guard surfaces. This is what makes c5n *generally useful* (the community adds Kotlin/Python/Rust without waiting on us) rather than three-projects-useful.
3. **External-process escape hatch** — for an emitter that needs real logic beyond templates: the protoc model (`c5n-emit-foo` on PATH, normalised model piped as JSON, any language). **Flagged trust boundary:** it runs unvetted third-party code at *build time* (highest-risk per `../llm/synced/design-rigour.md`) and spreads signing/reproducibility across artifacts. Documented fallback, not the default.

**In-tree emitters may carry logic — the pure-template bar is for *bundles*.** "Data, not code" applies to *third-party* bundles (which ship no Go). A **blessed in-tree** emitter, compiled into the binary, can add a **Go pre-render pass** that shapes the normalised model into a render-ready structure, then a simple `text/template` emits it — like the shared value-emitter, just per-target. **CSS (`palette`) is the exemplar:** media-query + cascade *precedence* is resolved in Go (lowering `tokens × themes × breakpoints` → an *ordered* flat rule-list), and the template is a mechanical walk. So complex targets don't need a more powerful template engine — the logic goes in the Go pre-render; **`text/template` (stdlib) stays the renderer**, preserving the minimal-dep property. *(A portable template language — Handlebars, not logic-less Mustache — is the deferred option only for lowering the third-party-bundle authoring barrier; a build-time dep, not needed for in-tree emitters.)*

### Co-existence with hand-written code (a per-writer concern)
Because generated code builds the object model and hand-written code adds behaviour, each writer also defines **how the two halves merge** in its language — itself part of "adding a writer":
- **C#** — a `partial class` (generated `Country.g.cs` + hand-written `Country.cs`).
- **TS** — a generated `*.data.ts` module the hand-written `index.ts` imports/re-exports (no `partial`; declaration-merging or a base class are alternatives).

### Stress-tested against l10n (candidate refinements)
l10n is the hardest consumer — a deep namespace *tree* of typed *functions* (not flat typed *data*), a `{arg:hint}` message DSL, plurals, nested selectors. Running its real shape through the model: **the engine holds; the additions are bounded seams, not a redesign.** What it forced:

- **`tree<T>`** (added above) — the namespace tree; also doppel/CLDR hierarchies.
- **typed-shim emit** — a message accessor is a trivial typed façade (`fn(args): string`) over a hand-written runtime call; ≈ emitting a typed constant. The only irreducibly-generated part is the **typed signature**.
- **fine-grained output layout** — per-message / per-locale files for tree-shaking; the writer partitions output, not "one file per type."
- **consumers lower exotic DSLs upstream** — c5n does *not* parse `{arg:hint}`. l10n owns a front-end that lowers its DSL → c5n's structured model (params + parts as data); c5n stays domain-blind. The **algorithm** (formatting, plural eval, the interpreter) is hand-written l10n runtime — its conformance surface.
- **calibration** — "one schema for all consumers" is really **one shared *meta-model* + engine; per-consumer schemas.** l10n's schema is structurally unlike f8n's; the *framework* is what's reused (always the actual thesis).

**Why this shape wins (perf + conformance):** parsing the DSL to *data at build time* (vs a runtime resolver re-parsing each call) removes per-call parse cost; a *single* hand-written interpreter per language (vs inlining each message body) keeps the **conformance surface** tiny — the dominant axis for l10n's verified-cross-language north star. That is codegen's *grain*: emit data + a thin typed surface, keep logic in a shared runtime. (Corroborated by the design-history observation: a prior l10n implementation retrofitted codegen onto a hand-first design; designing *for* codegen from day one selects this structure instead — the codegen-native-by-construction principle.)

> **Held as candidate.** The concrete l10n code-shape (typed shim + AST-as-data + a `render` interpreter) is a codegen-native **redesign**, deliberately diverging from the conventional translation-client / string-resolver structure. Well-reasoned but **not yet validated against a real message set** — pressure-test before settling. Full l10n structure → `../l10n/DESIGN.md`.

### Further consumers (portfolio · a code-first model consumer) — the model held
Two more consumers thrown at the design adversarially; both fit with **no new machinery beyond record-body emit**:

- **Flat & nested models of f8n primitives.** Portfolio's `Bank { products: list<Product> }`, `Product { rates: list<Rate>, deposit: Money }` are ordinary **nested records** (a field typed `list<Child>`), built by the recursive value-emitter — **not `tree<T>`**, which stays reserved for recursive, self-similar, arbitrary-depth structures (l10n namespaces, doppel geo). *Fixed-schema nesting ≠ a tree.* Mechanism: record fields can be collection-typed (`Bank.products: list<Product>`).
- **`fromJson`** — a generated deserialiser: wiring that maps JSON → the typed model, constructing f8n types; the primitive parsing is hand-written f8n runtime. Recurses *through the schema* (`Bank.fromJson` composes `Product.fromJson` composes `Rate.fromJson`). A reusable feature — anything crossing the wire wants it.
- **validation** — a generated fn **composing hand-written validator fns** per declared metadata (the l10n formatter-composition shape, in the validation domain). Cross-field is just another declared validator wired in. **This is where cross-language conformance earns its keep** — "client matches server" *is* the golden-vector problem (the same rule, C# ≡ TS). Covers *declarative* rules; server-only / async checks are hand-written and sit outside the conformance guarantee.

Everything else those consumers need — HTTP, client state, patch/offline queue, UI composition — is **hand-maintained**, correctly. The engine stayed small because the complexity went to the hand-written bucket (the standing guardrail: *c5n emits wiring + data; the algorithm is hand-written*). Three structurally-different consumers (l10n, portfolio, a code-first attribute-annotated model layer) resolving to the same engine and the same composition pattern is the strongest evidence the shape is right. *(A code-first source is read by a language-native front-end — e.g. Roslyn for C# — that lowers to c5n's normalised model, same as any other reader; c5n stays domain-blind.)*

## Distribution & supply chain
- **esbuild-style**: an npm wrapper carrying platform binaries (integrity-pinned in the lockfile) + a NuGet/MSBuild package (pinned via `packages.lock.json`).
- **Consumer dependency graph = zero.** `yaml.v3` and everything else are **statically linked inside the binary** at *our* build time. The consumer inherits nothing — not in their lockfile, `package.json`, `.csproj`, or transitive tree. Contrast a Node core, where the consumer would absorb our whole transitive tree.
- **Trust transforms, doesn't vanish:** "audit N packages" → "trust one binary." Back it with **reproducible builds** (Go is good at this — same compiler + flags → identical bytes), **signing + pinned checksums**, and **public-CI provenance** (SLSA / build attestation). Because this is public + central, that story is *demonstration*, not just hygiene.
- **`CGO_ENABLED=0`** → fully static, no libc → runs on Alpine/musl as well as glibc (kills the "works in Debian, breaks in Alpine" class of bugs).
- **Dependency surface ≈ just `yaml.v3`** (consumer YAML). Everything else is stdlib: `encoding/json` + `encoding/xml` (CLDR / `iso-codes`), `text/template` (emit). One well-vetted dep for a highest-risk (build-time) tool.

## Conformance & parity testing (f8n's own CI)
- **Golden-vector hub** — one **language-neutral** vector set (input → expected). Each target's generated code is tested **independently against the vectors**; parity between languages is **transitive through the shared oracle**. Adding a target is *one more spoke*, not an O(N²) pairwise blow-up.
- **Two checks per target**, both needing that toolchain: **compiles?** (emitted code is valid C#/TS/Swift) and **conforms?** (behaviour matches the vectors). A `strategy.matrix` over targets runs them in parallel.
- **Oracle caveat** (recurring *conformance ≠ correctness*): all N conforming to the *same* vectors = uniform correctness **or uniform wrongness**. Resolved by making a prose **specification** the oracle — see **Specification as the oracle** below.
- **Pin toolchain versions** (containerised / exact `setup-*`). This is *correctness*, not hygiene: a runner-image bump (e.g. a new **ICU version**) can shift locale/format output — the same ICU-drift risk flagged for `Locale`. Pinned = reproducible conformance.
- **Swift on Linux**, not macOS, where possible — macOS runners bill at **10× Linux** minutes and are slower; reserve them for genuinely Apple-platform behaviour.
- Tooling in Actions is trivial: `setup-dotnet` (`dotnet test`), `setup-node` (`tsc` + `vitest`), Swift Linux toolchain.

### Specification as the oracle (spec → vectors → code)
Resolves the "what independently verifies the vectors" problem by **being our own standards body** — the pattern W3C/crypto standards use where there's no external NIST: publish a spec, derive a conformance suite from it, make every implementation prove itself against the suite.

A prose **specification** states the *exact* process for each calculation (each `Money` op, the rounding mode, the BST date-seam, allocation-remainder policy). It's authored once, human-gated, versioned — an LLM-context in the corpus sense (cf. `../llm/` and the `memeplan` "instruction authored → refined → executed" thesis). Everything downstream derives from it: the author generates the golden vectors from it; a **clean** Claude session re-derives them from it as a cross-check; both sides generate the hand-written runtime code from it; a third party audits the software against it.

**The load-bearing caveat — the spec is the common ancestor of *both* tests and code.** So a green suite proves **code ≡ spec** and **all languages ≡ spec ≡ each other** — it does **not** prove the spec is *correct*. The oracle has *moved* from the implementation up to the spec (a real gain: one reviewable artifact, not thousands of opaque vectors), but correctness is not free — it is *relocated*. The one check that stays un-mechanizable is the top edge:

```
   authority (HMRC notice / ISO tables / the maths)   ← the ONE human, independent pass
        │
       SPEC ──┬── vectors (the oracle)
              ├── runtime code (under test)      green ⇒ code≡spec, NOT spec≡truth
              └── third-party audit
```

- **Spec ⟵ authority is the entire correctness guarantee** — a human confirming the spec says what the authority says. One pass *per rule*, one-time, not per vector. Everything below the spec is mere consistency.
- **The clean session is clean from the *derivation*, not from the *spec*.** It catches transcription/arithmetic slips, but will bless a *wrong rule* the spec contains, and may resolve a *spec ambiguity* the same wrong way (shared model priors — fresh context does not decorrelate trained-in defaults). So a clean-session **disagreement is high signal**: read it as *"the spec is underspecified,"* not *"a sum was wrong."* Used this way it audits the spec's **precision** — the thing that most needs auditing.
- **Register split.** Prose carries the *why* + rule-selection + the authority citation; **worked examples + explicit algorithm** (mode named, integer-minor-unit steps shown) carry the exact *what*. Precision test: if a clean session given only the spec can't reproduce a number exactly, the spec is underspecified.
- **A fully-specified worked example *is* a golden vector.** So the **spec carries the boundary vectors** (few, hand-derived, authority-checked — the independent, expensive bit) and **generation fills the interior** (many, cheap — parity + regression). *Edges verified once against authority in the spec; bulk generated.* That is what produces the vectors and how the edges are independently verified.

One-liner: **spec → vectors → code buys conformance and cross-language parity mechanically; correctness costs exactly one human pass of spec-against-authority per rule** — the only thing that was ever going to need a real oracle.

## GitHub automation (f8n's *own* repo only)
Consumers get **the tool + instructions** and run it in their own build/CI as they choose — *not* on our GitHub. Our automation is for f8n's own pipeline:
- **Canonical-data pipeline** — cron rebuilds the source from CLDR / `iso-codes` → diff → **PR** (machine-generates → human-gates), keeping f8n's data current.
- **Generated code** — **PR-on-change + drift-guard-on-PR**: a workflow regenerates and opens a PR when source moves; a separate check regenerates into a temp and `git diff --exit-code`s against the committed output, failing on drift. Result: generated types **live in the repo** (reviewable, diffable — you *see* "VAT 17.5% → 20%" as a code diff) **and** are guaranteed in sync. Same machine-generates → human-gates → git-reviews spine as the data pipeline.

## Quality bar
Central and publicly visible → the bar is **exemplary**: idiomatic Go, a legible/visible conformance harness, and reproducible + signed releases as evidence of trustworthy build-tooling craft.

## Build order & what's deferrable
Critical path is **spec + codegen**; conformance tooling is a room. Written to merge into the roadmap's dependency layers.

**Dependency edges (for the cross-project merge).** c5n is **upstream of** `f8n`, `l10n`, `doppel` — none can generate its types without it — so in the global order c5n sits *below* `f8n` as a base-layer tool, and **co-evolves with its first consumer `f8n`** (bootstrapping: c5n leads `f8n` slightly; built together). any cross-project ordering should place c5n **beneath `f8n`** in the dependency layers, rather than off to one side under meta-tooling.

**Build now (critical path):** the Go codegen; the spec (source of truth *and* seed vectors); generated data + hand-written behaviour.

**Seam now (cheap, don't skip — retrofitting is expensive):**
- **deterministic / invariant serialization** on every runtime type (the future driver diffs on it; f8n already assigns `Money` invariant-serialize — hold that line everywhere);
- **worked examples accreted into the spec** as edges are hand-derived, while the reasoning is fresh (else the vector set is archaeology later);
- a **stable vector wire-format** — emerges *with* the spec; treat as a published contract once vectors exist.

**Room (backfill freely — additive, no rework):**
- native conformance runners (A);
- **chosen future direction — the uniform Go driver + per-language `run-vector` CLIs (B):** one neutral runner that doubles as the third-party audit tool. Depends only on the three seams above; **nothing built now may assume a native-only harness.**

**Accepted debt:** deferring the runner ⇒ no parity/regression net while the conformance-critical money math is written; seam #2 (worked-examples-as-you-go) is what keeps that debt small.

## Open questions
- ~~**Own repo/project?**~~ **Resolved 2026-07-06:** `c5n` is a **standalone project/package** (its own dir, own identity — not an f8n component) that **lives in the monorepo** alongside f8n/l10n/doppel + the consumer libs. Standalone ≠ separate repo: co-located so one PR proves the whole `spec→vectors→code→parity` chain, but published as its own artifact. (Repo layout resolved at the roadmap level: monorepo.)
- ~~**Vector oracle** — what produces the golden vectors, and how the edges are independently verified.~~ **Resolved 2026-07-03:** a prose **specification** is the oracle (spec → vectors → code); boundary vectors are hand-authored into the spec and checked against the authority, the interior is generated. See **Specification as the oracle**.
- **Consumer generated-code policy** — checked-in vs generated-on-build is the consumer's call; we document the PR + drift-guard pattern as the recommended shape (it's what f8n's own repo uses).
- **Generation-model details to firm up** (all bounded, none structural): the exact collection-kind spelling (`list`/`table`/`EffectiveDated`); enum-member normalisation (how data's `standard` maps to `TaxCategory.Standard` — casing/aliasing); and the polymorphic-field discriminator syntax (bites `l10n`'s plain-vs-interpolated leaf, not `f8n`).

## Change log
- 2026-07-09: **identity via a declared `key:` field + list authoring form** (from building f8n's first real data). A `table<T>` names its constant from the field the type's `key:` points at (`Country.GBR` from `alpha3`, `Currency.GBP` from `code`), and that field is also the reference target; **ctor args are the explicit declared fields — no key-prepend magic** — and data is authored as a **list** of rows (the id appears once). The **map form** (key implicit, id not a stored field) stays available for identities that aren't domain fields (l10n namespaces). Added a **key-uniqueness** validation note. Updated the schema/data/emitted examples to match `../f8n/` (Currency `key: code` + a default `symbol`; Country `key: alpha3`; `tz` → `capitalTz`, the capital's civil zone).
- 2026-07-04: **Further consumers (portfolio + a code-first model layer)** — thrown adversarially, both fit with **no new machinery beyond record-body emit**. Added: **record/model type-body emit** (generated types now include records, not just enums/external instances); **`fromJson`** (generated deserialiser, recurses through the schema — reusable); **validation** = generated fn composing hand-written validator fns (l10n composition pattern; *where cross-language conformance earns its keep* — "client matches server" is the golden-vector problem; declarative rules only, server-only checks excluded). Clarified **nested records ≠ `tree<T>`** (fixed-schema nesting is records-with-list-fields; `tree<T>` stays for recursive/self-similar cases). Everything else (HTTP, state, patch, UI) is hand-maintained — the guardrail held, engine stayed small. Retracted earlier over-specs (typed API-client generator; normalised-JSON-as-published-contract framing) — a code-first source is just another reader (Roslyn) lowering to the normalised model.
- 2026-07-04: **l10n stress-test** (pressure-testing the reusability thesis against the hardest consumer). Engine holds; bounded additions: **`tree<T>`** collection kind (recursive generalisation of `table<T>`; leaf type → nested scopes vs nested value), **typed-shim emit** (a message accessor = a typed façade over a hand-written runtime call; the irreducible generated part is just the typed signature), **fine-grained output layout** (per-message/locale files for tree-shaking), **consumers lower exotic DSLs upstream** (c5n never parses `{arg:hint}` — l10n owns the front-end + the runtime interpreter). Principle generalised: "generate data / hand-write behaviour" → **"generate the typed boundary / hand-write the algorithm."** Calibration: "one schema for all" → **one shared meta-model + engine, per-consumer schemas.** Perf/conformance rationale: build-time parse-to-data (not runtime re-parse) + one interpreter per language (not inlined bodies) = tiny conformance surface. **Concrete l10n code-shape held as candidate** (codegen-native redesign, diverges from the conventional translation-client/string-resolver structure; not yet validated against a real message set).
- 2026-07-03: added **Build order & what's deferrable** — critical path = spec + codegen; conformance is a room. Conformance runner direction chosen: **(B) a uniform Go driver + per-language `run-vector` CLIs** = one neutral runner that doubles as the third-party audit tool — **planned but deferred**, backfill is additive. Three seams to honour now so backfill stays cheap: deterministic/invariant serialization, worked-examples accreted into the spec as edges are derived, a stable vector wire-format. Accepted debt: no parity net while the money math is written. Includes dependency edges for the cross-project merge (c5n is upstream of f8n/l10n/doppel; co-evolves with f8n; belongs beneath f8n in the dependency layering).
- 2026-07-03: **Specification as the oracle** (spec → vectors → code) — resolves the "vector oracle" open question. A prose spec (LLM-context; `memeplan` instruction-executed thesis) is the source of truth: author generates vectors from it, a clean Claude session cross-checks, both generate the runtime code, third parties audit against it. Load-bearing caveat: the spec is the common ancestor of tests *and* code, so green proves **code≡spec, not spec≡truth** — correctness relocates to a single human pass of **spec⟵authority per rule** (HMRC/ISO/maths). Boundary vectors are hand-authored into the spec + authority-checked; the interior is generated. Clean-session disagreement = "spec underspecified," not "sum wrong." Also: I over-projected naive-calculator use onto Chris (tax-software background — works in integer minor units by hand); residual risk is *rule selection* (jurisdiction VAT rounding, date-seam), checked against the published rule, not the arithmetic.
- 2026-07-03: added **Generation model** (worked examples). **Generated code builds the typed object model; the consumer hand-writes *behaviour only*** — c5n emits ctor calls for "types we already know" = schema-declared, *including per-language construction*, so it stays domain-blind yet outputs ready-to-use values. **Schema** = a small typed IDL (records/enums/scalars; construction = convention-by-default + per-type `emit:` override; *not* JSON Schema), separate from **data** (positionally typed, `common`-hoisted, no per-value tags). **Collection kinds** `list`/`table`/`EffectiveDated` — **temporality is *declared* (the type), never sniffed from a data key**; the series envelope/value split is driven by the type. The **value-emitter** (literal / generated-symbol reference / nested ctor) is the shared, conformance-critical core. **Target selection is explicit at the project boundary** (`c5n.yaml` — the drift-guard manifest), never in data. **Emitters are template bundles** (data, not code — no Go `plugin`/dynamic-lib loading); trust-ranked **in-tree > third-party bundle > external-process** escape hatch. Rejected: primitives-only output (would push object-model wrapping onto the consumer) and JSON-Schema-as-schema.
- 2026-07-02: promoted to its **own project folder** `c5n/DESIGN.md` (was `f8n/codegen.md` → `f8n/c5n.md`); reframed from "split out of f8n" to standalone shared engine.
- 2026-07-02: named **`c5n`** (numeronym for *codegen*). (Vet bare package id on npm/NuGet/crates before publishing — `c5n` collides with an unrelated TV channel; namespace-scoped ids sidestep it.)
- 2026-07-02: created — **Go** decision + rationale (rejected Roslyn source-gen / Node core); portable-core + thin-wrappers architecture; multi-target emitter seam; esbuild-style distribution + supply-chain (zero consumer deps, reproducible/signed); golden-vector conformance CI (hub not O(N²), pinned toolchains, Swift-on-Linux); f8n's own GitHub automation (canonical-data cron→PR, generated-code PR + drift-guard).
