# c5n — codegen tooling

**`c5n`** — numeronym for *codegen* (c·odege·n), in the `f8n`/`l10n`/`a11y` family. The build tool that turns **source data** — CLDR / `iso-codes` (`f8n`'s own canonical types) and consumer-authored YAML (their tax/FX/etc. data) — into **typed, cross-language-conformant code** (C#, TS, … Swift likely) plus the `EffectiveDated` accessors. A **central, publicly-visible portfolio piece** — its own quality is part of the demonstration.

> **Standalone shared engine.** `c5n` is its **own project** — `f8n`, `l10n`, and `doppel` all feed it templates + schemas; it is *not* an `f8n`-internal component. First consumer is `f8n` (see `../f8n/DESIGN.md`). Repo layout (one repo vs per-project) is a roadmap-level concern — resolved: monorepo.

> **Sequence.** This doc is the *why*; `./PLAN.md` is the ordered implementation plan — phases, steps, and the checkpoint each one ends at.

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
Types are declared **once**, in their own file(s), separate from data. Not JSON Schema — that's a *validation* language (pattern/required/anyOf), maps badly to clean nominal types, and drags in a real dependency (against the "≈ just `yaml.v3`" surface). What we want is a tiny **IDL**, proto/TypeSpec-shaped: records, enums, scalars — closed and nominal.

**The deeper reason (and the standing answer to "why not OpenAPI / JSON Schema + an off-the-shelf generator?"):** those standards describe the **shape** of a payload; they have no vocabulary for **behaviour over** that shape — cross-field and cross-type validation, discriminator defaults and their runtime resolution, capability metadata, patch semantics. In a model where the value lives in those rules, a shape-only standard can describe the types and none of what matters, so the generated client gets types without semantics — precisely the part that must not be allowed to diverge. Reach for the standard first and adopt it if it fits; own a small IDL only where it demonstrably cannot express the semantics. *(Ordering matters for the argument as much as for the engineering: tried-then-built is judgment, built-then-justified is not-invented-here.)* Every type is one uniform declaration = **fields + a construction recipe**; the only axis is whether c5n **emits the type** or merely **constructs/references** it:

- **`external: true`** — a hand-written runtime type (`Money`, `Percentage`, `Country`). c5n never emits the type body; it only needs the recipe to build instances.
- **generated** — c5n emits the type too (an enum, or a **record / model type body**). Same declaration; c5n also writes the definition. *(Emitting record bodies — not just enums and external-type instances — is the one capability the model/`fromJson` consumers add; see **Further consumers**.)*

```yaml
# schema/f8n.types.yaml — declared once; language-blind except the construction recipes

Percentage:                 # external: hand-written runtime primitive
  external: true
  fields: { value: string }   # an exact Rational, canonical "num/den" — see ../f8n/DESIGN.md
  emit:                       # override — the ctor won't do, and the recipe names the unit
    csharp: "Percentage.FromPercent({value})"
    ts:     "Percentage.fromPercent({value})"

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

TaxType:     { kind: enum, members: [VAT] }   # generated — c5n emits the type body
TaxCategory: { kind: enum, members: [Standard, Reduced, ZeroRated] }
```

**A bare number that needs a unit or a scale must be constructed by name.** The positional-ctor convention is safe only where every field is self-describing. Where it is not, the same literal has two readings that differ by a factor — and no check anywhere in the pipeline can tell them apart, because both are valid values of the declared type:

| type | the ambiguity | factor |
|---|---|---|
| `Percentage` | proportion (`0.175`) vs percent (`17.5`) | 100 |
| `Money` | major units (`12.34`) vs minor units (`1234`) | 10^dp, and *currency-dependent* — JPY 1, BHD 1000 |
| `ExchangeRate` | which way round the rate runs | the inverse |

`f8n` already resolves the third by putting the `(from, to)` pair **in the type**, so the direction cannot be dropped. The rule generalises that: **name the unit at the construction site** — `FromPercent` / `FromProportion`, `FromMajor` / `FromMinor` — and never let a positional ctor take a number whose meaning lives in a comment. It is the same reasoning that leaves `AllocationRule` with no default: a silent interpretation of a caller's number is the failure mode, so the API makes the caller say which they mean. In c5n this lands in one reviewed place — the type's `emit:` recipe — rather than in every data author's head, on every row.

**Construction = convention-by-default + per-type override.** Most types get a positional-ctor convention the writer knows (`new T(f1, f2, …)`) — zero per-type config. Only the awkward ones (factories like `Money.of`, parse-from-string primitives, singleton refs) carry an explicit `emit:` block, and it lives **beside the type** (all of `Percentage` in one place). Recipes are per-target-language.

**A recipe chooses the shape of the call, not the spelling of the literals.** `{field}` substitutes the **fully emitted argument expression** — quoted strings, target-suffixed decimals, resolved references — exactly what the convention would have passed positionally. So a recipe supplies the factory name, the argument order, any wrapping; it never re-renders a value. (Literal decoration is the value-emitter's job and is already per-target: a `decimal` carries C#'s `m` suffix without any recipe asking for it.) Cost of "add Swift" = a new writer **+ a `swift:` line on the few special types**, not every type.

**Identity = the `key:` field (`table<T>`).** A `table<T>` emits one named constant per row; `key:` names which declared field is the identity — it becomes the **constant's name** (`Country.GBR`, from `alpha3`) and the **reference target** (a `defaultCurrency: GBP` elsewhere resolves to `Currency.GBP` by matching Currency's `code`). The id is therefore an ordinary field in the ctor like any other — **no key-prepend magic** — and data is authored as a plain **list** of rows (the id appears once). *(A map authoring form — key implicit, id not a stored field — stays available for identities that aren't domain fields, e.g. l10n's namespace names; f8n uses the explicit `key:` + list form.)* Validation: the `key:` field must be **unique** across the table (the uniqueness the map form gave structurally).

### Enums — members are declared, and a member name is a wire token
An enum's members are listed in the schema, so a data value **selects** a member and can
never create one; a value naming an undeclared member is a validation error, exactly the
guarantee a reference to a table row gets. Drawing members from the data instead — the
earlier sketch — was rejected on two counts. It makes a typo **mint a member** rather than
fail, and since an enum serialises as text (`../f8n/DESIGN.md` → *Enums travel as their
member name*), that typo mints a **wire token** — the published-contract failure the
reference check exists to prevent. And it makes the emitted API depend on data coverage: an
enum nothing referenced yet could not be emitted at all, so a type would appear and
disappear as data was added. Declared members make an enum the first unit c5n emits from the
**schema alone**.

The same fact **dissolves the member-normalisation question** rather than answering it:
**no casing is applied anywhere.** The name in the schema is the name in C#, the name in TS,
and the token on the wire — one spelling in three places. Any automatic PascalCase would be
rewriting a published contract, and it would turn `VAT` into `Vat`. What c5n does check is
*shape*: a member must be a legal identifier in every target, which catches the mistake an
author actually makes (`zero-rated` where they meant `ZeroRated`). A member that happens to
be a **keyword** in one target is left to that target's compiler — an unlikely collision in
domain vocabulary, failing loudly in a gate that already runs.

**TS spelling: a const object plus a union of its values — not a TS `enum`.**

```ts
export const TaxCategory = { Standard: "Standard", Reduced: "Reduced" } as const;
export type TaxCategory = (typeof TaxCategory)[keyof typeof TaxCategory];
```

`TaxCategory.Standard` then reads **identically in both targets**, so the shared
value-emitter has one reference spelling rather than one per language — which matters
because consumers use these types in both. It also makes the two agree on the wire *by
construction*: the TS runtime value **is** the token C# serialises for the same member, with
no converter for anyone to keep in sync. A TS `enum` gives neither — it is number-backed, so
the runtime value stops being the token, and it is not erasable syntax, so type-stripping
runtimes reject it. A bare string union was rejected too: with no value to reference, the
emitted expression would have to differ per target, which is the divergence the shared
resolver exists to remove.

One asymmetry it leaves, deliberately: the TS union accepts a raw `"Standard"` off the wire
directly, where C# needs a parse. That is deserialisation being cheaper on one side, not the
contract differing.

### Lookups — `key:` identifies, `lookup:` also finds
A `table<T>` always emits an index on its `key:`, and one more per field listed in `lookup:`:

```yaml
Country:
  key: alpha3                  # identity: the constant's name, the reference target, the wire form
  lookup: [alpha2, numeric]    # additionally indexed
```

The two are **different declarations rather than a list of equals**, and that asymmetry is
the point. `key:` is the canonical form; the others are ways in. A value arrives from foreign
systems in forms that are not the identity — a country is identified by its alpha-3 code and
turns up as an alpha-2 constantly — and those need resolving without becoming second wire
forms. Make them interchangeable and nothing says which is canonical, which is precisely how
one value acquires three encodings.

**c5n emits the precise accessors and stops there.** One per index — `ByAlpha3`, `ByAlpha2`,
`ByNumeric` — each taking that field's declared type and returning the row or nothing. A
dispatcher that takes *"a code, in whatever form"* is **hand-written beside the type**,
because deciding which form a string is requires knowing that alpha-2 and alpha-3 have
disjoint widths and that numeric is three digits. That is domain knowledge about ISO 3166;
c5n knows only that the fields are `string, string, int`. The standing guardrail holds
without needing to be argued again: *c5n emits the typed boundary, the algorithm is
hand-written.*

Validation treats a lookup field like the key — **unique across every file that feeds the
type**, since a lookup is a promise that a value finds one row. Lookup fields must be scalars;
indexing a reference or a nested value has no obvious key and would be inventing semantics
nobody has asked for. A miss returns null/undefined rather than throwing: a code that names no
row is an ordinary outcome when the value came from somewhere else.

*(Casing an accessor name — `alpha2` → `ByAlpha2` — is not the casing this design refuses for
enum members. A member name is a published wire token; an accessor name is an identifier c5n
chose, and nothing outside the generated code depends on its spelling.)*

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
    - { from: 2011-01-04, rate: 20 }     # as the notice states it; the recipe says what 20 means
    - { from: 2010-01-01, rate: 17.5 }
```

**`common`-hoisting** is a general affordance (not tax-specific): any field constant across a collection lifts to `common:`; each row carries only what varies; c5n merges `common ⊕ row` when constructing the value. Emitted code is identical to writing every field out — that equivalence is the whole claim, so it is what the tests pin. It's where much of the "data stays clean" ergonomics lives across all three consumers.

**A row that also sets a hoisted field is an error, not an override — and the reasoning is post-agent.** A cascade (row wins) is the conventional choice, and it was the *right* choice when rejecting a file cost a person retyping every row by hand; leniency bought real ergonomics and the ambiguity was the price. That cost is now close to zero, so the trade has inverted: leniency keeps the ambiguity and buys nothing. `common:` reads as authoritative, which is precisely what makes a row quietly differing from it invisible in review — the silent-divergence failure this design keeps closing elsewhere. It is also the **reversible** direction: erroring now can be relaxed later and no existing data breaks, where starting lenient and tightening would. If a genuine "constant except here" case appears, that is a **defaults** feature, added deliberately — a different thing from *constant*, and one that should not arrive by accident as the side effect of a merge rule.

*The general form is worth stating once, because it will recur: an ergonomic leniency is a trade against **human keystrokes**. Where an agent does the expanding, the keystrokes are free and only the ambiguity is left, so the strict path becomes correct with no downside. Re-derive, don't inherit, the conventional answer.*

**Hoisting the identity is rejected.** A `table<T>`'s `key:` field is what differs per row by definition, so `common:` cannot carry it. Caught at validation rather than left to the merge, where it would surface as every row claiming the same name — several confusing errors standing in for one clear one.

**The merge runs after validation, never before.** A mistake in `common:` — a misspelled field, a reference that resolves to nothing — would otherwise be copied into every row and reported once per row, naming `common:` in none of them. That is the 1.2 failure one layer up: reporting the wreckage rather than the mistake.

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

### Series — the envelope is declared by the type, and the unit is the series
`EffectiveDated<T>` is the first collection kind that is not a plain list of values: each
entry carries an **envelope** — the date it takes effect from — alongside the fields that
construct the `T`. Four things follow, and none of them is temporal: c5n never learns what a
date is, or that a series keys on a field called `from`.

**The envelope is declared, in the schema, on the series type.** A series is `kind: series`
with an `envelope:`, and c5n reads the split from that declaration:

```yaml
EffectiveDated:
  external: true            # hand-written f8n runtime type; c5n constructs, never emits
  kind: series
  envelope: { from: LocalDate }         # what keys an entry; the rest construct the T
  emit:
    csharp: "EffectiveDated.Of({entries})"
    ts:     "EffectiveDated.of([{entries}])"
```

This is what "**temporality is declared, never sniffed**" actually means in the
implementation: a row's `from:` is the envelope *because the type said so*, and an entry
that omits it is a validation error naming the declaration — never a value quietly absorbed
as one of `T`'s own fields. Building the field name into c5n would have been shorter and
would have made the engine know a thing about time.

**A data file holds several named collections.** A file of tax rates carries many
`EffectiveDated<TaxRate>` series, so "the TaxRate one" does not identify anything — the
series needs a name, and the name is what the output unit is called (`VatStandard.g.cs`, not
`TaxRate.g.cs`). A table needs no name, since one type is one unit however many files feed
it. `type:` at the top level is what distinguishes the two shapes:

```yaml
VatStandard:                              # named collections — the series form
  type: EffectiveDated<TaxRate>
  common: { jurisdiction: GBR, taxType: VAT, category: Standard }
  items:
    - { from: 2011-01-04, rate: 20 }      # as the notice states it; the recipe says what 20 means
    - { from: 2010-01-01, rate: 17.5 }
```

**One spelling for the entries: `items:`.** An earlier draft of this doc used `rows:` for a
series and `items:` for a table. Two words for the same idea is something a data author has
to remember for no gain, and c5n's own errors say "row" for an entry either way.

**A series recipe is required, and takes `{entries}`.** There is no positional-ctor
convention to fall back on — a collection is built by a factory taking a list, and what that
factory is called is per-language. `{entries}` is the one reserved placeholder that does not
name a declared field; the pair syntax inside it is the target's own (a C# tuple, a TS
array), the same class of per-target spelling as string quoting.

**The envelope cannot be hoisted to `common:`**, for the reason an identity cannot: it is
what differs per entry. Caught at validation rather than at the merge, where it would
surface as every entry claiming the same moment.

**No new scalar was needed.** `from: 2011-01-04` goes through an ordinary external type with
a parse recipe — `LocalDate.Parse("2011-01-04")` — which the one-field-scalar rule already
covers. So the temporal design stays entirely in `f8n`, where the ISO-8601 subset and its
strict grammar belong, and c5n gains no notion of a date. The same route is what any future
unit-bearing scalar should take.

Emitted C#, and the one place the two targets differ in *shape* rather than spelling:

```csharp
public static class VatStandard
{
    public static readonly EffectiveDated<TaxRate> Series = EffectiveDated.Of(
        (LocalDate.Parse("2011-01-04"), new TaxRate(TaxType.VAT, TaxCategory.Standard, Percentage.FromPercent("20"))),
        (LocalDate.Parse("2010-01-01"), new TaxRate(TaxType.VAT, TaxCategory.Standard, Percentage.FromPercent("17.5"))));
}
```

```ts
export const VatStandard = EffectiveDated.of([
  [LocalDate.parse("2011-01-04"), new TaxRate(TaxType.VAT, TaxCategory.Standard, Percentage.fromPercent("20"))],
]);
```

C# has no top-level value, so the series hangs off a static class named for it and reached as
`VatStandard.Series`; TypeScript exports it directly. Everything else — the envelope
position, the hoisted identity, the enum references, the nested constructors — is identical.

### The value-emitter (the conformance-critical heart)
For each field the emitter resolves, **driven purely by the declared field type**, to one of three shapes:

- **literal** — scalar → per-language literal formatting (`0.20m`, `826`, a `DateOnly(…)`);
- **reference** — the value matches an enum member or a table row's **key-field value** → emit the reference (`Currency.GBP`, `TaxType.VAT`), not a fresh ctor;
- **nested ctor** — the field's type is a constructible type → recurse (`Percentage.FromPercent("20")`).

**A one-field type may be authored as a plain scalar.** `rate: 17.5`, not `rate: { value: 17.5 }` — the mapping would carry nothing the schema does not already say. This is the wrapper case (`Percentage` over an exact `Rational`) and it is what keeps data files reading like the documents they were copied from. A type with more than one field needs an authored mapping, since which field a lone scalar belongs to would otherwise be a guess; that is an error naming the type and the value it was handed.

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
        (new DateOnly(2011, 1, 4), new TaxRate(Country.GBR, TaxType.VAT, TaxCategory.Standard, Percentage.FromPercent("20"))),
        (new DateOnly(2010, 1, 1), new TaxRate(Country.GBR, TaxType.VAT, TaxCategory.Standard, Percentage.FromPercent("17.5"))));
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
  csharp: { out: dotnet/Generated/, namespace: F8n }   # F8n, not a sub-namespace: the tables are
  ts:     { out: ts/src/generated/ }                    # `partial class` extensions of the hand-written types
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

Everything else those consumers need — HTTP, client state, patch/offline queue, UI composition — is **hand-maintained**, correctly. The engine stayed small because the complexity went to the hand-written bucket (the standing guardrail: *c5n emits wiring + data; the algorithm is hand-written*). Three structurally-different consumers (l10n, portfolio, a code-first attribute-annotated model layer) resolving to the same engine and the same composition pattern is the strongest evidence the shape is right. *(How a code-first source reaches c5n was revised — see **Code-first sources emit the spec** below.)*

### Code-first sources emit the spec — c5n reads YAML only (2026-08-22)
**Revises the earlier "language-native reader" idea (a Roslyn front-end inside c5n).** c5n does **not** parse foreign source languages. Its input is **YAML, and only YAML**. Where a model is authored code-first — attribute-annotated types in a host language — that ecosystem's **own build emits the spec**:

```
attributed C# model  →  (MSBuild task / source generator, in C#)  →  schema.yaml  →  c5n  →  C# + TS
```

Why this way round:

- **The Go surface stays tiny.** A Roslyn front-end means hosting a C# compiler API from Go (or shelling out to one anyway) — the single largest thing that could be added to a binary whose whole pitch is "≈ just `yaml.v3`". Emitting YAML from C# is a small, ordinary C# tool written in the language that already understands the model.
- **One reader, forever.** Every future source — another language, a design tool, a hand-written file — converges on the same YAML spec instead of adding a front-end to c5n. c5n's input surface never grows.
- **The spec becomes a reviewable artifact.** The emitted YAML is committed, so a contract change shows up **as a diff in a pull request** rather than as an invisible consequence of an attribute edit. That is worth a great deal on its own.
- **It preserves spec-as-oracle.** Code-first sources now *converge on* the spec rather than bypassing it, so the same prose-spec → golden-vector → code chain applies to them unchanged.

Constraints that come with it:

- **No build cycle.** The assembly that *declares* the model must not depend on generated output. Model-declaring project → spec → generated code → everything else.
- **Commit the spec, guard the drift.** The emitted YAML is checked in and CI re-runs the emitter and asserts no diff (the same drift-guard shape used for generated code, and a natural `c5n check` job). Hand-editing a generated spec is the one way this design fails; the guard is what makes that impossible to do quietly.

### Contract identity — generated artifacts carry their version (2026-08-22)
Every generated bundle carries a **contract identity** (a hash of the normalised model, or a declared version) so a consumer and a producer can *assert* compatibility rather than assume it. Where generated types cross a wire, the identity travels with the request and a mismatch **fails closed** — reject and tell the client to update, never accept a mutation from a client that may have understood the contract differently.

The motivating case is a growing discriminated-union set. Adding a concrete type to a union is a regeneration and a release for every consumer — which is correct and unavoidable, because a new concrete type arrives with behaviour, so something ships either way. The real risk isn't the regeneration; it's a **stale consumer still talking to a moved contract**. A contract identity turns that from a silent, data-corrupting failure into a loud, recoverable one. Cheap seam now; no retrofit available later, because by then consumers are deployed.

### Generated rules carry stable identity (2026-08-22)
A generated validator must emit a **stable rule identity** alongside its result — the same rule resolving to the same identity in every target language. It is what lets a consumer reconcile locally-evaluated results with authoritative ones from a server: the same rule reported from both sides is recognisably *one* rule, so it can be deduplicated, and a client-evaluated result can be superseded by the authority's verdict rather than displayed twice.

Without generated identity, reconciliation degrades to matching on rendered message strings — fragile, and broken outright by localisation. The identity is generated (never hand-assigned), travels with the problem rather than the message, and is paired with the property's own stable key. This is a small emitter requirement with an outsized payoff, and it only exists because both sides are generated from one declarative source.

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

### The vector dataset is the artifact (dataset → runners → parity)
One **language-neutral dataset** of input → expected, and a thin **runner per language** that executes it. Parity is transitive through the shared dataset, so adding a target is one more spoke rather than an O(N²) pairwise blow-up, and the same runner is what a third party audits with.

**Rationale travels with the numbers, not in a separate document.** An earlier version of this design made a prose *specification* the artifact, with vectors derived from it. That is cut. For the great majority of rules the expected value is self-evident — `add` adds — and a spec paragraph restating it is ceremony. For the few where it is not (rounding at ties, `allocate`'s remainder distribution, net↔gross ordering, the date seam) what is actually needed is a sentence of rationale and an **authority citation beside the vector**, where it cannot drift away from the number it explains. So a vector group carries its own *why*: which rule this is, where the expected value came from, and who says so. *(The former design's own line — "a fully-specified worked example **is** a golden vector" — argues for keeping the two together; splitting them into separate artifacts was the error.)*

**The caveat survives, because it is true of any dataset: conformance is not correctness.** A green run proves every language agrees with the dataset. It says nothing about whether the dataset is right — uniform correctness and uniform wrongness are indistinguishable from inside.

```
   authority (an HMRC notice, ISO tables, the maths)   ← the one human, independent pass
        │
   the vector + its stated rationale      green ⇒ every language ≡ the dataset,
        │                                         NOT the dataset ≡ truth
   runners (C#, TS, …)
```

**But for most of what is under test, there is no external authority — and that is by design.** `HalfEven(2.5) = 2` is not a policy question, it is the definition of the mode. Vectors target **component functions** (`divide-with-rounding` per mode, `allocate` per rule, `Rational` normalisation, minor↔major conversion at a scale) whose semantics the library *defines*, so there is nothing to consult. Deliberately **not** under test: compositions like "VAT on an invoice", where per-line-vs-per-invoice and the mandated rounding are the *caller's* choices — which is why `RoundingMode` is caller-supplied and `AllocationRule` has no default. A rule the library refuses to choose is not a rule its vectors can encode.

An authority binds in three places, none of them arithmetic, and each has its own pipeline:

- **Format grammars** — the ISO-8601 subset, E.164. The standard sets accept/reject, so the **negative** cases carry the weight and the citation is the standard itself.
- **Canonical data** — ISO 4217 minor units, ISO 3166 codes, IANA zones. Not vectors: the cron → diff → **PR** pipeline, which already has a human gate.
- **Locale formatting** — CLDR, and `l10n`'s, derived-and-baked against its own fixtures.

So the residual human cost is small and lands where it belongs: confirming a **grammar** matches its standard, and reviewing a **data** diff. Everything else is consistency, which is what machines are for.

**Properties are the independent derivation path.** A bulk vector captures a value from one
implementation; a second implementation written against that dataset would agree with it
whether or not either is right. A **property** carries no captured value — it asserts what
must hold for *every* input, so its expected result comes from the rule. `fromPercent(x)`
equals `fromProportion(x/100)`; `parse(canonical(x))` equals `x`; a comparison is a total
order; every entry of a series is in effect on its own start date. These are what a captured
dataset structurally cannot give, which is why they sit beside it rather than inside it. A
failing property reports *what* broke rather than "false", so the diagnosis is in the run.

**Pinning the toolchain is correctness only where the toolchain can move a result.** The
recurring instinct is to pin exact versions once behaviour is under test. But the thing a pin
guards is a *runtime-dependent result*, and f8n has deliberately arranged to have none:
parsers are character walks rather than regexes, formatting is invariant-culture, arithmetic
is on big integers. The dependency is engineered out rather than pinned around, which is the
stronger form of the same guarantee and does not rot. The pin becomes correctness at the
first behaviour that reads the runtime's locale data — `l10n`'s formatting, where ICU is the
subject rather than an accident — and belongs in the job that measures it. *(Verified by
re-running the datasets under hostile locales; kept as a technique rather than a CI stage,
since a check that cannot fail reads as coverage it does not provide.)*

**A clean session stays useful as a technique, not a stage.** Give a fresh session a rule's rationale and nothing else, and see whether it reproduces the expected value. Read a disagreement as *"the rationale is underspecified"* rather than *"the arithmetic is wrong"* — fresh context catches transcription slips but shares trained-in defaults, so it will bless a wrong rule that is clearly stated. Used that way it audits how precisely the rule is written, which is the part most worth auditing.

## GitHub automation (f8n's *own* repo only)
Consumers get **the tool + instructions** and run it in their own build/CI as they choose — *not* on our GitHub. Our automation is for f8n's own pipeline:
- **Canonical-data pipeline** — cron rebuilds the source from CLDR / `iso-codes` → diff → **PR** (machine-generates → human-gates), keeping f8n's data current.
- **Generated code** — **PR-on-change + drift-guard-on-PR**: a workflow regenerates and opens a PR when source moves; a separate check regenerates into a temp and `git diff --exit-code`s against the committed output, failing on drift. Result: generated types **live in the repo** (reviewable, diffable — you *see* "VAT 17.5% → 20%" as a code diff) **and** are guaranteed in sync. Same machine-generates → human-gates → git-reviews spine as the data pipeline.

## Quality bar
Central and publicly visible → the bar is **exemplary**: idiomatic Go, a legible/visible conformance harness, and reproducible + signed releases as evidence of trustworthy build-tooling craft.

## Build order & what's deferrable
Critical path is **spec + codegen**; conformance tooling is a room. Written to merge into the roadmap's dependency layers.

**Dependency edges (for the cross-project merge).** c5n is **upstream of** `f8n`, `l10n`, `doppel` — none can generate its types without it — so in the global order c5n sits *below* `f8n` as a base-layer tool, and **co-evolves with its first consumer `f8n`** (bootstrapping: c5n leads `f8n` slightly; built together). any cross-project ordering should place c5n **beneath `f8n`** in the dependency layers, rather than off to one side under meta-tooling.

**Build now (critical path):** the Go codegen; the **vector dataset and its runners**; generated data + hand-written behaviour. The runner used to sit in the room below — it is on the critical path because the alternative was a prose spec standing in for it, and the mechanism is cheaper than the document it was substituting for.

**Seam now (cheap, don't skip — retrofitting is expensive):**
- **deterministic / invariant serialization** on every runtime type (the future driver diffs on it; f8n already assigns `Money` invariant-serialize — hold that line everywhere);
- **rationale recorded beside a vector wherever its expected value is not self-evident**, while the reasoning is fresh (else the dataset is archaeology later, and nobody can tell a deliberate expected value from a typo);
- a **stable vector format** — treat it as a published contract from the first vector, since third-party runners read it.

**Room (backfill freely — additive, no rework):**
- **native per-language harnesses (A)** — running the same vectors from `dotnet test` / `vitest` for editor integration and familiar failure output. A convenience over the uniform driver, not a replacement: **nothing may assume a native-only harness**, or the neutral runner stops being the thing a third party can audit with.
- further targets — each is one more spoke.

**Debt retired.** The previous plan deferred the runner and accepted "no parity net while the conformance-critical money math is written". Building the runner alongside the first behaviour removes that, and the first vectors are deliberately something small (the rational parse) rather than money math, so the harness gets shaped on an easy case.

## Open questions
- ~~**Own repo/project?**~~ **Resolved 2026-07-06:** `c5n` is a **standalone project/package** (its own dir, own identity — not an f8n component) that **lives in the monorepo** alongside f8n/l10n/doppel + the consumer libs. Standalone ≠ separate repo: co-located so one PR proves the whole `spec→vectors→code→parity` chain, but published as its own artifact. (Repo layout resolved at the roadmap level: monorepo.)
- ~~**Vector oracle** — what produces the golden vectors, and how the edges are independently verified.~~ **Resolved 2026-07-03, revised 2026-08-25:** the **dataset itself is the artifact**, with each non-obvious vector carrying its rationale and authority citation beside the numbers; edges are hand-derived and authority-checked, the interior is generated from them. The original resolution routed this through a separate prose specification — cut, because it duplicated the vectors for the many rules whose expected value is self-evident, and separated the reasoning from the numbers for the few where it is not. See **The vector dataset is the artifact**.
- **Consumer generated-code policy** — checked-in vs generated-on-build is the consumer's call; we document the PR + drift-guard pattern as the recommended shape (it's what f8n's own repo uses).
- **Generation-model details to firm up** (all bounded, none structural): the exact collection-kind spelling (`list`/`table`/`EffectiveDated`); and the polymorphic-field discriminator syntax (bites `l10n`'s plain-vs-interpolated leaf, not `f8n`).
- ~~**Enum-member normalisation** — how data's `standard` maps to `TaxCategory.Standard` (casing/aliasing).~~ **Resolved 2026-08-26: there is no normalisation.** Members are declared in the schema, and the declared name is emitted verbatim in every target. The question only existed while members were to be drawn from data; once an enum serialises as **text**, the member name is a wire token, and a generator applying casing to it is a generator rewriting a published contract (it also turns `VAT` into `Vat`). c5n validates a member's *shape* — a legal identifier in every target — and nothing else. See **Enums**.
- ~~**Rate authoring form.**~~ **Resolved 2026-08-25:** **data authors the percent number the source document states — `rate: 17.5` — and the type's `emit:` recipe names the unit: `Percentage.FromPercent({value})`.** The stored value is still the dimensionless proportion `7/40`; `FromPercent` divides by 100 in rational space, which is exact. Nothing is lost and the data reads like the notice it was copied from.

  Rejected on the way: making **`Parse` itself** take a decimal proportion. `Parse` has to accept the canonical `"num/den"` so a value round-trips, which fixes its number-space to the proportion — and then `"0.175"` means 17.5% while `"17.5"` means 1750%, with a data author's most natural input silently wrong by a factor of 100. Also rejected, harder: reading rational form as a proportion and decimal form as a percent, which makes `"1/2"` mean 50% and `"0.5"` mean 0.5%. `Parse` therefore stays the **canonical wire form only**; human authoring goes through the named constructors. Two consequences, both spec rules (see the spec seed, `PLAN.md` step 3.1):
  - **`%` is a constructor, never the stored form** (`../f8n/DESIGN.md`). `Percentage` holds a dimensionless proportion — the operand `Money × p → Money` consumes — and "17.5%" is one *presentation* of it, which belongs to `l10n`. The named constructor is what keeps the two apart at every site where a number enters.
  - **The decimal parser must not route through a binary float.** The obvious TypeScript implementation is `Number(s)` — `float64`, which cannot hold what the data can, and exactly the defect c5n itself carried when data was decoded through `any`. Both targets must parse the digit string into an exact numerator and denominator directly. This is now a **conformance surface**: c5n emits the *authored* text (`Percentage.Parse("0.175")`), so C# and TS must agree precisely on what that string means, and that agreement is what the golden vectors pin.
- ~~**Output paths are derived from the type, not the source.**~~ **Resolved 2026-08-25: output is named for what it *declares* — the emitted unit — and tables are grouped accordingly.** A `table<T>` emits **one unit per type**, however many data files feed it: splitting reference data across files (per region, per source, per reviewer) is an authoring convenience, and the output does not inherit that shape. `EffectiveDated` will emit **one unit per named series**, since the series is what it declares. *Rejected: naming output after the source file* — it would put `partial class TaxRate` in `GbVat.g.cs` and three unrelated series in a file named after none of them, and it requires deriving a legal identifier from an arbitrary path (hyphens, digits, casing, non-ASCII) identically in every target. Naming by declaration keeps the file name matching the type it declares, gives TS the granularity tree-shaking wants, and turns a clash into a **symbol** collision — a real error with a real message — rather than a path clash nobody can act on. The former behaviour lost data silently: two files, one path, second write wins, and `c5n check` then failed straight after a clean build advising a rebuild that could not help.

## Change log
- 2026-08-26: **`attribution:` — a licence notice travels with the artefact.** A manifest can declare a notice and the source paths it applies to, and c5n reproduces it in the header of everything generated from a matching source. The alternative — a NOTICE file at the repo root — depends on a person remembering, and it does not travel: a generated file that leaves the repo leaves the obligation behind. Declared in `c5n.yaml` rather than per data file for two reasons: the named-collection form has no free top-level key, so a reserved one would be a wart; and an obligation belongs beside the targets and source globs, in the project's authoritative statement of its outputs. Matching uses the same two glob shapes the source patterns do, so a pattern that selects sources selects them here too. Notices are deduplicated per unit, so several files under one licensed source produce one notice. *(Driven by f8n resolving its data sourcing to CLDR, whose Unicode licence obliges exactly this.)*
- 2026-08-26: **`lookup:` — a table can declare secondary indexes.** Every `table<T>` now emits an accessor for its `key:`, and one more per field listed in `lookup:`. The two declarations stay deliberately different: `key:` is the canonical form — the constant's name, the reference target, and what a value takes on the wire — while `lookup:` fields are *ways in*, for values arriving from systems that use another form. Collapsing them into one list would leave nothing saying which is canonical, which is how one value acquires several encodings. c5n emits the **precise** accessors only; a dispatcher taking a code in whatever form is hand-written, because deciding which form a string is needs domain knowledge (alpha-2 and alpha-3 have disjoint widths) that the schema does not hold. Lookup fields are validated unique per type across every file, and restricted to scalars.
- 2026-08-26: **an int with a leading zero is rejected.** Found while adding a currency whose ISO numeric code is conventionally written `048`. Scalars are emitted as the authored text, and `048` is a decimal literal in C# but a **syntax error in a TypeScript module** — so one data file would have compiled in one target and not the other. It is also two spellings of one value, which this design refuses everywhere else. The error names both readings, since an author who wrote it most likely wanted the ISO *string* and should declare the field as one.
- 2026-08-26: **properties recorded as the independent derivation path, and toolchain pinning re-aimed.** A bulk vector captures a value from one implementation, and a second implementation written against that dataset agrees with it whether or not either is right; a **property** carries no captured value, so its expected result comes from the rule. That is the gap it closes, and it is why properties sit beside the dataset rather than inside it. Separately, the standing instruction to pin toolchains exactly once behaviour is under test was **narrowed**: a pin guards a *runtime-dependent result*, and f8n has arranged to have none — character-walk parsers, invariant formatting, big-integer arithmetic — so the guarantee is held by construction rather than by a version number, and the trigger moves to l10n's locale formatting, where ICU is the subject. Confirmed by re-running under hostile locales, and kept as a technique rather than a CI stage, since a check that cannot fail reads as coverage it does not provide.
- 2026-08-26: **series — `EffectiveDated<T>`, with the envelope declared by the type.** The second collection kind, and the first whose entries are not plain values: each carries an **envelope** alongside the fields that construct the `T`. The split comes from a schema declaration (`kind: series` + `envelope:`), not from c5n, which is what makes "temporality is declared, never sniffed" real in the implementation — an entry missing its `from:` is an error naming the declaration, and c5n never learns that a series keys on a field called `from`. Decisions recorded with it: a data file holds **several named collections** (distinguished by `type:` at the top level), since a file of tax rates holds many `EffectiveDated<TaxRate>` and the *name* is what the output unit is called; **one spelling, `items:`**, replacing this doc's earlier `rows:`/`items:` split, which was two words for one idea; a series **recipe is required** and takes the reserved `{entries}` placeholder, since a collection has no positional-ctor convention to fall back on; and the **envelope cannot be hoisted to `common:`**, for the reason an identity cannot. **No new scalar was needed** — the date goes through an ordinary external type with a parse recipe, so the temporal design stays in f8n and c5n gains no notion of a date, which is the route any future unit-bearing scalar should take.
- 2026-08-26: **`common:`-hoisting, with overlap an error rather than a cascade.** Any field constant across a collection lifts to `common:` and each row carries only what varies; the emitted code is identical to writing every field out, which is the whole claim and so is what the tests pin byte-for-byte. Three rules came with it. A row that also sets a hoisted field is an **error** — and the reasoning is explicitly post-agent: a cascade was right when rejecting a file cost a person retyping the rows, but with an agent doing the expanding the keystrokes are free, so leniency keeps only the ambiguity. It is also the reversible direction (relaxing later breaks no data; tightening later would), and a real "constant except here" case is a *defaults* feature to be added deliberately, not a merge rule arriving by accident. **Hoisting the identity is rejected**, since a key varies by definition. And the **merge runs after validation**, so a mistake in `common:` is reported once against `common:` instead of once per row it was copied into — the 1.2 failure one layer up. Generalised for reuse: *an ergonomic leniency is a trade against human keystrokes; where an agent does the typing, re-derive the answer rather than inheriting the conventional one.*
- 2026-08-26: **enums — members are declared, not drawn from data, and no casing is applied anywhere.** Revises the earlier sketch (`kind: enum`, "members drawn from data"), which Phase 1's own reasoning had already superseded: collected members make a typo **create** a member rather than fail, and since an enum serialises as text the typo mints a **wire token** — the failure the reference check exists to prevent. Declared members also make an enum the first unit emitted from the **schema alone**, where every prior unit derived from a data file; under the rejected design an enum nothing referenced could not have been emitted at all, so the public API would have depended on data coverage. That fact **dissolves the member-normalisation open question** instead of answering it — the schema name is the C# name, the TS name and the wire token, one spelling in three places, which is also the only rule under which `VAT` survives as `VAT`. c5n checks a member's *shape* (a legal identifier in every target, catching `zero-rated`) and leaves target keywords to the target compiler. **TS spelling: a const object plus a union of its values, not a TS `enum`** — so `TaxCategory.Standard` reads identically in both targets and the TS runtime value *is* the token C# serialises, with no converter to keep in sync; a TS `enum` is number-backed and not erasable syntax, and a bare string union would have forced a per-target reference spelling. Also pinned in `../f8n/DESIGN.md` → *Enums travel as their member name* (C# must be configured to write strings, not the serialiser's numeric default).
- 2026-08-25: **cut the separate specification; the vector dataset is the artifact, and the runner moves onto the critical path.** The previous design made a prose spec the oracle, with vectors derived from it — sound reasoning, but it was carrying weight only because the runner that should carry it had been deferred, and it duplicated the numbers for every rule whose expected value is self-evident. Now: one language-neutral dataset plus a thin runner per language, with **rationale and authority citation recorded beside each non-obvious vector**, where they cannot drift from the value they explain. What survives unchanged is the caveat, which was never about specs: **conformance is not correctness** — green proves every language agrees with the dataset, not that the dataset is right — so correctness still costs one human pass per non-obvious rule against the authority. The clean-session cross-check is kept as a technique for auditing how precisely a rule is stated, rather than as a pipeline stage. **Accepted debt retired:** the parity net now exists while the money math is written, and the first vectors are deliberately the rational parse rather than money, so the harness is shaped on an easy case.
- 2026-08-25: **output is named for what it declares.** Resolves the output-path question: a `table<T>` emits **one unit per type**, merging every data file that feeds it, and `EffectiveDated` will emit one unit per named series. Splitting data across files is an authoring convenience the output should not inherit. Naming output after the *source file* was rejected — it puts `partial class TaxRate` in `GbVat.g.cs`, puts several unrelated series in a file named after none of them, and requires deriving a legal identifier from an arbitrary path identically in every target. The prior behaviour lost data silently (two files, one path, last write wins) and left `c5n check` failing immediately after a clean build with advice that could not help; a duplicate output path is now an error.
- 2026-08-25: **resolved the rate authoring form — named constructors, not a wider `Parse`.** Data authors the percent number the source document states (`rate: 17.5`) and the `emit:` recipe names the unit (`Percentage.FromPercent`); the stored value is still the exact proportion `7/40`, since dividing by 100 in rational space loses nothing. Widening `Parse` to take a decimal proportion was rejected: `Parse` must accept the canonical `"num/den"` to round-trip, which fixes its number-space, so a data author's most natural input would have been silently wrong by 100×. Generalised into a standing rule — **a bare number that needs a unit or a scale must be constructed by name** — with the three instances tabulated (`Percentage` proportion-vs-percent, `Money` major-vs-minor and currency-dependent, `ExchangeRate` direction, which f8n already solves by typing the `(from, to)` pair). Also pinned: the decimal parser **must not route through a binary float**, since the obvious `Number(s)` in TypeScript is `float64` and reintroduces the defect c5n itself carried when data was decoded through `any`; because c5n emits authored text rather than a normalised form, that parser is a conformance surface the golden vectors have to pin.
- 2026-08-25: **split the sequence out into `PLAN.md`** — phases → steps with `✓` marked in place, so this doc stays the *why* and the plan carries the ordering and its checkpoints. **Added CI** (`.github/workflows/ci.yml`): the drift-guard and the per-target compile checks now run on push and PR rather than by hand — one `engine` job (do the sources still produce the committed output?) and one job per target (does the committed output still compile?), with actions pinned by commit SHA. The two target jobs are separate rather than a `strategy.matrix` only because their toolchains share no command yet; the vector runner is what merges them. Also recorded two open questions raised by the next slice — the **rate authoring form**, and **output paths derived from the type rather than the source**, where two data files feeding one type collide.
- 2026-08-25: **corrected the `Percentage` schema example and pinned recipe semantics.** The example still showed `value: decimal` with a C# `m` suffix written into the recipe — superseded by f8n's 2026-07-17 decision that `Percentage` is an exact **`Rational`**, serialised as the canonical string `"num/den"` precisely so a decimal is never materialised. Updated to `value: string` + a parse recipe. Also pinned the substitution rule that the old example left ambiguous: **`{field}` is the fully emitted argument expression**, so a recipe never re-spells a literal (which is what made `{value}m` look necessary). Implementation notes from the same pass: `emit:` is now wired up (it had been parsed and ignored), and data values are carried as **authored source text** rather than decoded through `any` — the latter routed every fractional value through float64, so a declared decimal reached the target with different digits and compiled clean while being wrong.
- 2026-08-22: **generated rules carry stable identity.** A generated validator emits a rule identity that resolves identically in every target language, so locally-evaluated and authoritative results can be **reconciled** — one rule recognised as one rule, deduplicated, with the authority's verdict superseding a local one rather than doubling it. Without it, reconciliation falls back to matching rendered message strings, which localisation breaks outright. Identity is generated (never hand-assigned), travels with the problem rather than the message, and pairs with the property's stable key. Small emitter requirement, outsized payoff — and only possible because both sides generate from one declarative source.
- 2026-08-22: **code-first sources emit YAML — c5n never parses foreign source languages**; plus contract identity and the OpenAPI positioning. **Revised** the earlier "language-native reader (Roslyn) lowers to the normalised model" idea: the direction inverts — an attributed C# model is turned into `schema.yaml` by a **C# build step**, and c5n's input stays **YAML only**. Keeps the Go surface at "≈ just `yaml.v3`", means every future source converges on one reader instead of adding a front-end, makes the **spec a committed, diffable artifact** (a contract change becomes a PR diff), and preserves spec-as-oracle. Constraints recorded: no build cycle (the model-declaring assembly must not depend on generated output), and commit-the-spec + CI drift-guard. **Added contract identity** — generated bundles carry a hash/version, asserted across a wire, **failing closed** on mismatch; motivated by growing discriminated-union sets, where regeneration is correct and unavoidable but a *stale consumer against a moved contract* is the silent, unrecoverable failure. **Added the standing "why not OpenAPI/JSON Schema" answer**: those standards describe payload *shape* but have no vocabulary for *behaviour over* the shape (cross-field/cross-type validation, discriminator defaults, capability metadata, patch semantics) — so a shape-only generator emits types without the semantics that must not diverge; try the standard first, own an IDL only where it demonstrably cannot express the rules.
- 2026-07-09: **identity via a declared `key:` field + list authoring form** (from building f8n's first real data). A `table<T>` names its constant from the field the type's `key:` points at (`Country.GBR` from `alpha3`, `Currency.GBP` from `code`), and that field is also the reference target; **ctor args are the explicit declared fields — no key-prepend magic** — and data is authored as a **list** of rows (the id appears once). The **map form** (key implicit, id not a stored field) stays available for identities that aren't domain fields (l10n namespaces). Added a **key-uniqueness** validation note. Updated the schema/data/emitted examples to match `../f8n/` (Currency `key: code` + a default `symbol`; Country `key: alpha3`; `tz` → `capitalTz`, the capital's civil zone). Also corrected the `c5n.yaml` example — generated tables are `partial class` extensions, so `namespace: F8n` (not the earlier `F8n.Data`, which wouldn't merge); example out-paths aligned to f8n's real layout (`dotnet/Generated/`, `ts/src/generated/`). Golden output for these types is hand-written under those dirs as the emitter's byte-for-byte target.
- 2026-07-04: **Further consumers (portfolio + a code-first model layer)** — thrown adversarially, both fit with **no new machinery beyond record-body emit**. Added: **record/model type-body emit** (generated types now include records, not just enums/external instances); **`fromJson`** (generated deserialiser, recurses through the schema — reusable); **validation** = generated fn composing hand-written validator fns (l10n composition pattern; *where cross-language conformance earns its keep* — "client matches server" is the golden-vector problem; declarative rules only, server-only checks excluded). Clarified **nested records ≠ `tree<T>`** (fixed-schema nesting is records-with-list-fields; `tree<T>` stays for recursive/self-similar cases). Everything else (HTTP, state, patch, UI) is hand-maintained — the guardrail held, engine stayed small. Retracted earlier over-specs (typed API-client generator; normalised-JSON-as-published-contract framing) — a code-first source is just another reader (Roslyn) lowering to the normalised model.
- 2026-07-04: **l10n stress-test** (pressure-testing the reusability thesis against the hardest consumer). Engine holds; bounded additions: **`tree<T>`** collection kind (recursive generalisation of `table<T>`; leaf type → nested scopes vs nested value), **typed-shim emit** (a message accessor = a typed façade over a hand-written runtime call; the irreducible generated part is just the typed signature), **fine-grained output layout** (per-message/locale files for tree-shaking), **consumers lower exotic DSLs upstream** (c5n never parses `{arg:hint}` — l10n owns the front-end + the runtime interpreter). Principle generalised: "generate data / hand-write behaviour" → **"generate the typed boundary / hand-write the algorithm."** Calibration: "one schema for all" → **one shared meta-model + engine, per-consumer schemas.** Perf/conformance rationale: build-time parse-to-data (not runtime re-parse) + one interpreter per language (not inlined bodies) = tiny conformance surface. **Concrete l10n code-shape held as candidate** (codegen-native redesign, diverges from the conventional translation-client/string-resolver structure; not yet validated against a real message set).
- 2026-07-03: added **Build order & what's deferrable** — critical path = spec + codegen; conformance is a room. Conformance runner direction chosen: **(B) a uniform Go driver + per-language `run-vector` CLIs** = one neutral runner that doubles as the third-party audit tool — **planned but deferred**, backfill is additive. Three seams to honour now so backfill stays cheap: deterministic/invariant serialization, worked-examples accreted into the spec as edges are derived, a stable vector wire-format. Accepted debt: no parity net while the money math is written. Includes dependency edges for the cross-project merge (c5n is upstream of f8n/l10n/doppel; co-evolves with f8n; belongs beneath f8n in the dependency layering).
- 2026-07-03: **Specification as the oracle** (spec → vectors → code) — resolves the "vector oracle" open question. A prose spec (LLM-context; `memeplan` instruction-executed thesis) is the source of truth: author generates vectors from it, a clean Claude session cross-checks, both generate the runtime code, third parties audit against it. Load-bearing caveat: the spec is the common ancestor of tests *and* code, so green proves **code≡spec, not spec≡truth** — correctness relocates to a single human pass of **spec⟵authority per rule** (HMRC/ISO/maths). Boundary vectors are hand-authored into the spec + authority-checked; the interior is generated. Clean-session disagreement = "spec underspecified," not "sum wrong." Residual risk is *rule selection* (jurisdiction VAT rounding, date-seam), checked against the published rule rather than the arithmetic.
- 2026-07-03: added **Generation model** (worked examples). **Generated code builds the typed object model; the consumer hand-writes *behaviour only*** — c5n emits ctor calls for "types we already know" = schema-declared, *including per-language construction*, so it stays domain-blind yet outputs ready-to-use values. **Schema** = a small typed IDL (records/enums/scalars; construction = convention-by-default + per-type `emit:` override; *not* JSON Schema), separate from **data** (positionally typed, `common`-hoisted, no per-value tags). **Collection kinds** `list`/`table`/`EffectiveDated` — **temporality is *declared* (the type), never sniffed from a data key**; the series envelope/value split is driven by the type. The **value-emitter** (literal / generated-symbol reference / nested ctor) is the shared, conformance-critical core. **Target selection is explicit at the project boundary** (`c5n.yaml` — the drift-guard manifest), never in data. **Emitters are template bundles** (data, not code — no Go `plugin`/dynamic-lib loading); trust-ranked **in-tree > third-party bundle > external-process** escape hatch. Rejected: primitives-only output (would push object-model wrapping onto the consumer) and JSON-Schema-as-schema.
- 2026-07-02: promoted to its **own project folder** `c5n/DESIGN.md` (was `f8n/codegen.md` → `f8n/c5n.md`); reframed from "split out of f8n" to standalone shared engine.
- 2026-07-02: named **`c5n`** (numeronym for *codegen*). (Vet bare package id on npm/NuGet/crates before publishing — `c5n` collides with an unrelated TV channel; namespace-scoped ids sidestep it.)
- 2026-07-02: created — **Go** decision + rationale (rejected Roslyn source-gen / Node core); portable-core + thin-wrappers architecture; multi-target emitter seam; esbuild-style distribution + supply-chain (zero consumer deps, reproducible/signed); golden-vector conformance CI (hub not O(N²), pinned toolchains, Swift-on-Linux); f8n's own GitHub automation (canonical-data cron→PR, generated-code PR + drift-guard).
