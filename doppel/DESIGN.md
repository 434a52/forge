# doppel — design

**doppel** — realistic, locale-aware **synthetic data** for development and testing. Built on `f8n` (real Country/Currency/Money/Locale) and `l10n` (localised formatting), generated codegen-native via `c5n`. The differentiator is **coherence** — whole personas whose parts *agree* — not random field values. Personal open-source (destined). Codegen-coupled; lives in this monorepo. See `../f8n/DESIGN.md`, `../l10n/DESIGN.md`, `../c5n/DESIGN.md`.

> **Status: design in progress.** Core settled; more sections to come.

> **Core architectural stance:** *locale is a runtime dimension, not a type dimension* — uniform entity types + locale-scoped data + locale-keyed strategies, **never locale subclasses or generics**. The design's biggest deliberate choice; see *Locale is a runtime dimension* below.

## North star
1. **Coherence over randomness** — a persona is a *consistent set of facts*: an address whose county/town/postcode agree, a name that fits the age, accounts/salary/employer that line up. The differentiator.
2. **Locale-correct by construction** — via `f8n` + `l10n`; a locale switch re-coheres the whole persona.
3. **Codegen-native** — designed *for* `c5n` (data generated, behaviour hand-written), like `f8n`/`l10n`.
4. **Unmistakeably synthetic** — realism of *facts*, never of *authenticity*; nothing output is passable as a real document or a real identifier.

## The coherence engine
**Coherence is dependency, not randomness.** Independent random fields give a postcode that doesn't match the town, a name that doesn't fit the age, a salary that doesn't match the transactions. Coherence comes from generating in **dependency order**, each value conditioned on what's already chosen. The mechanism is deliberately simple — **caller-composed graph + generator-enforced edges**, not a global constraint solver:

- The **caller composes** the entity graph by passing related entities into generators.
- Each **generator enforces its local edge rules** on what it's given.

```csharp
var doppel = Doppel.EnGB(42);          // locale at the root, optional seed
var adult  = doppel.Identity.Adult();
var child  = doppel.Identity.Child([adult]);   // child's address = pick from a parent's
```

`Child([adults])` takes its parents and inherits an address from one. The same shape generalises: `Adult(employer)`, `Account(bank)`, a marriage linking two adults — pass the related entity, the generator wires the coherence. **The generation order falls out of the conditioning** (e.g. gender + birth-year are chosen before the name, because the name is drawn from that cohort).

### Entity graph
Typed entities, each with its own generator:
- **Person** (domestic) — identity, address, accounts, salary, employer, marital status.
- **Business** — subtypes **generic**, **bank**, **not-for-profit**, each with distinct rules (a bank provides accounts; an NFP has charity-shaped fields; generic is the default employer).

Composed by reference: `person.employer` → a built business; `person.accounts[]` → held at a **bank**; `child` → its parents; marriage → a spouse Person. **Recursion is bounded**: a bank *is* a Business subtype and businesses hold accounts at a bank, so a bank is a **leaf provider** — it does not generate its own bank.

**Pooling (scale axis).** Single-persona demo: entities can be freshly built and attached. Coherent *populations* (test datasets): shared entities matter (many personas per bank, many employees per employer) → a **pool** sized to the run, sampled into; single-persona is the degenerate pool-of-one. **Banks** are a fixed **curated pool** (see Names).

## Identity — names, gender, titles, family
The **Person** entity carries: prefix, first/middle/last name, gender, DOB, email, phone, country of birth/residence, place of birth, height, eye colour, address + address history, job title, tax reference. Detail:
- **Gender** — `Other (default) | Male | Female`. `Adult(gender? = null)` → specified, or random when null. Inclusive by construction (and synthetic, so no real-PII concern).
- **First names — conditioned on (birth-year, gender).** Sourced from **children's-names-by-year** open data (UK ONS = OGL, US SSA = public domain), so a name *fits the age* (fashion shifts by decade) — a real coherence dimension, not just realism. The **per-year frequencies are the weights**. Lists split by gender; **Other draws from the union**. Sparse locales fall back to the `*`/language list (unweighted-by-year).
- **Titles** — gender-skewed **weighted** list (`Mr` heavy for Male; `Mrs`/`Ms`/`Miss` for Female; `Mx` for Other; `Dr` neutral across all), locale-scoped (`Herr`/`Frau`, `M.`/`Mme`), and age-aware (`Master`/`Miss` for children). `title ← gender (+ age)`.
- **Surnames** — a *separate*, non-gendered, locale-scoped-by-frequency list.
- **Middle name** — from the same (gender, birth-year) first-name pools.
- **Phone** — `f8n.PhoneNumber` (country from *country of residence*); number generated in **reserved fictional ranges** (Ofcom `07700 900xxx` / `020 7946 0xxx`; NANP `555-01xx`) → provably not a real number.
- **Countries & place of birth** — `f8n.Country` for **country of birth** and **country of residence** (residence drives locale/address); **place of birth** a real town within the country of birth. Birth ≠ residence allowed (immigrant personas, via scoped `pick`).
- **Physical** — height (gender-conditioned range), eye colour (weighted list).
- **Job title** — *compositional from parts* (seniority + function), locale-scoped, coherent with the employer's industry.
- **Tax reference** — a synthetic ID via `template` (format-only where the real one has no checksum, e.g. NI number).
- **Family coherence** — marriage and parentage propagate the surname via a **locale-scoped policy** (the convention is *not* universal, so it must not be hardcoded):
  - **en-GB default:** male+female marriage → the male's surname; otherwise random partner's. Child follows the male in a male+female marriage, else a random parent's.
  - **Structurally different conventions** (Spanish double-surname, Icelandic patronymic, keep-maiden-name cultures) → **per-locale generators**, deferred as rooms.
  - Marital status is a real attribute; a married Person implies a **spouse entity** in the graph.

## Histories, finance & amounts
- **Anchored sequences** — any *history* (address, employment, transactions) is generated by anchoring each element's date to the previous one: each address's `moveInDate` = `past`/`future` from the prior move, walking the chain, bounded by the persona's lifetime. The general sequence-coherence pattern — the temporal engine, iterated.
- **Amounts** — a `Money` generator with `min`/`max`; the **currency comes from the account/locale** (a GB account in GBP), never a free choice → coherence.
- **Bank account** — held at a **bank** (fictional, curated pool); type (business/individual/joint), currency (`f8n.Currency`), account name, number, sort code, **IBAN**, **SWIFT/BIC**; an active window (opened … now/closed) that **bounds its transactions**. **Identifiers are bank-derived — coherence and safety at once:** IBAN/SWIFT/sort-code *encode* bank + country, so their country portion comes from the account's country and their bank portion from the bank entity (a fictional bank carries a synthetic bank code that flows into *all* its accounts); the rest is synthesised with an **invalid check digit** (IBAN mod-97) → provably fake.
- **Payment card** — network (Visa/Mastercard/Amex — drives number prefix/length + CVV length), number, CVV, issue + expiry (= issue + validity), **holder name from the person** (coherent projection). Safe via **published test numbers** (valid-Luhn *and* reserved) or **invalid-Luhn**.
- **Transactions** — `minDate`/`maxDate` (**clamped to the account's window** — no transaction before the account existed), `minAmount`/`maxAmount`, and a **category** (a single specified one, or random). **Categories are locale-scoped data** (`* > en > GB`) — plausible-per-locale (Groceries, Dining, Travel, …).
- **Coherence depth (planned — endorsed):**
  - **Category-conditioned amounts** — a plausible range *per category* (Rent £800–2000, Coffee £2–6) rather than one global bound.
  - **Regular vs ad-hoc** — salary (monthly, *from the employer*, matching the salary figure), rent, subscriptions are *scheduled* and coherent with employment/address/salary; distinct from ad-hoc categorised spending — the money story that lands.

## Documents
A **Document module** — `Document.DrivingLicence(person)`, `Document.Passport(person)`, (`Statement`, …). **Each generator takes a Person** and is a **coherent formatted projection** of them: name, DOB, address, nationality, place of birth are pulled *from the person*; the document adds its own synthetic fields.
- **Driving licence** — number (spec-formatted via `template`), issue date (bounded `past`), **expiry = issue + validity period** (coherent, not independent), issuing authority, real vehicle **categories** (A/B/C… standard codes), place of birth.
- **Passport** — number, **MRZ line 1 / line 2** (ICAO 9303 format), issue + expiry dates, issuing authority, nationality, place of birth.
- **Country-specific → no `*`** — licences/passports are inherently national, so their data is region-scoped only (`en > GB`, no global level); see the loading refinement.
- **Rendering** — each document is an **SVG template with text placeholders** (filled from the person's data, localised via `l10n`); templates are locale-scoped/per-type (a GB licence ≠ a DE one). This is the **`svg` project's** job (SVG components → PNG, localised + accessible), so doppel's document rendering *consumes `svg`* (a later-layer dependency). **Photos are gendered grey silhouettes** — never a real or realistic face.

Documents are the **sharp end of the dual-use gate** — see below.

## Organisation module
Generates the **Business entities** persons reference (a person's employer *is* a company from here; their accounts *are* at a bank from here — the coherence loop closes). Types share a core and add type-specific fields:
- **company** — registered name, trading name, **company type** (locale types: Ltd/PLC, GmbH/AG, SA…), group name, registration number, tax reference, **VAT number**, registration date, address, web domain + URL, employee count, **industry code** (SIC group/code).
- **bank** — the core + a **Fitch credit rating**.
- **not-for-profit** — the core + a **charity number**.

- **Domain taxonomies are doppel's own reference data.** SIC/industry codes, Fitch scale, company types are **locale-scoped reference data doppel ships** (UK SIC vs NACE vs NAICS; Ltd vs GmbH) — exactly the "consuming layer brings its own taxonomies" that `f8n`'s scope boundary anticipates (`f8n` deliberately ships neither). c5n-generated like everything else.
- **Names are compositional, not list-picked.** Org names are *assembled from parts* — a name-template of slots filled from **locale-scoped, type-specific part pools** (a bank's parts ≠ a charity's ≠ a generic company's), with the **company-type suffix** matching the locale. Conditioned on type (+ optionally industry). Same pick-over-pools primitives, assembled.
- **registrationDate is a coherence anchor** — no accounts, employees, or employment-of-a-person *before* the org registered; downstream dates clamp to it (same edge as the account window).
- **Identifiers** — registration/tax/VAT/charity numbers via the `template` fn under the safe-default checksum policy (VAT has check digits → invalid-by-default unless a valid test value is explicitly wanted).

### Domains & email
- **Web domain / URL** — built from the org's name parts + a **TLD** (`* + region` locale data: global `.com`/`.org` + region `.co.uk`).
- **Email — free / paid / organisation.** Organisation email = `local@org-domain` — coherent with the employer *and* safe (the org domain is fictional). Free/paid = generic providers.
- **Safety (design-rigour) — domains must not collide with real ones.** A generated domain, or a `@gmail.com`-style address with a real-looking local part, can **coincide with a real registered domain or a real person's inbox** (collision/privacy risk). Use **RFC 2606/6761 reserved domains** (`example.com`, `.test`, `.invalid`) — provably non-deliverable, guaranteed non-colliding — for the generic free/paid case, and a **fictional org domain** for organisation email. **Never real provider domains with plausible local parts.** The email/domain analog of "provably-fake identifiers."

## Reference data — real structure, synthetic leaf
Coherence rests on **real structure** with a **synthetic leaf** — coherent yet provably-not-a-real-address:

- **Geo hierarchy** (region → county → town, with the town's real postcode **prefix/district**) from **open data**. *Licensing is a trust decision:* prefer **Wikidata (CC0)** (settlements, admin hierarchy, postcode districts, populations for weighting) over **Wikipedia (CC BY-SA)**, whose *share-alike* conflicts with an MIT/public tool. Also safe: **OGL** (gov/ONS geo), **GeoNames (CC BY)**. Pick per source-licence, MIT-compatible (data is embedded/redistributed). Confirm at ingest.
- **Address line 1** = a **generic road name** (curated per-locale list) + a **random house number** → synthetic, provably fake.
- **Postcode** = the town's **real prefix** (district/outcode — all open data gives; full unit → address is licensed PAF, *not* open) + a **synthetic tail** (template-filled, valid format). The prefix **scopes to the town** (coherence), the tail is fake. The open-data constraint hands you the safe design for free. *Prefix structure is per-locale* (UK outcode, German PLZ leading digits, …) — carried in the locale-scoped data.

**Honesty on the guarantee:** even a generic street can coincide with a real building ("1 High Street, [real town]"). The guarantee is an *unmistakeably-synthetic persona*, **not** "no string ever coincides with a real address."

## Locale is a runtime dimension, not a type dimension
The core architectural stance — and where the design most deliberately differs from a naive locale-aware model. **Locale variation never touches the type system.** Modelling it with **locale-specific subclasses** (`UkAddress`, `UkPerson`) or **generics** (`Address<GB>`) is the trap: locales are added constantly, so type-encoded variation makes every new locale a rippling code/type change, and the generics *leak through every consuming layer*.

Instead, locale variation splits into two runtime things:
- **Data** — value pools + formats (names, company types, TLDs, address field-sets/formats) → locale-scoped **data**.
- **Behaviour** — rules that genuinely differ (surname conventions, address composition) → locale-keyed **strategies/generators**, selected at runtime.

Neither needs a subclass or a type parameter:
- **Entities stay uniform types** — `Person`, `Address`, `Organisation`, never `UkPerson`/`Address<GB>` — so nothing generic threads through consumers.
- **The locale binds to the `Doppel` instance** — `Doppel.EnGB(seed)` resolves its pools + strategies once; `Adult()` returns a plain `Person`.
- This is also why the design is cleanly **cross-language + codegen-native** — flat uniform types are trivially `c5n`-generatable and C#/TS-identical, where generics/subclasses are the cross-language *and* leak headache.

**Address is the exemplar.** One uniform `Address` over CLDR's *bounded* field set (recipient, street-address-lines, dependent-locality, locality, admin-area, postal-code, …); the locale drives *which* fields are used + *how* they lay out (CLDR format data, applied by `l10n`). PT's differences are a different used-set + format over the same type — *not* a subclass. Field set + formats come from **CLDR** (already `f8n`'s source); `c5n` generates them; doppel populates the locale's fields at generation time. **Adding a locale is pure data** (+ a strategy only if its behaviour truly differs), never a type change.

**Principle:** *locale-varying structure → one uniform type + locale-scoped data + locale-keyed strategies; never generics or subtypes.*

## Locale-scoped data & loading
Every data file is keyed by locale component — neutral `*` → language → region — and for the current locale you **load `*` at every level plus the lists whose key matches, unioned**:

```yaml
# names.yaml  (same shape for banks, roads, surnames, id-templates — every data file)
firstNames:
  "*":  [ ... ]        # global / locale-neutral — always loaded
  en:
    "*": [ ... ]       # any-English — loaded when language = en
    GB:  [ ... ]       # British    — loaded when region = GB
  de:
    "*": [ ... ]
    DE:  [ ... ]
```

For `en-GB` → the pool is `"*"` ∪ `en."*"` ∪ `en.GB`. Uniform across every file — names, banks, roads, surnames, identifier templates, address formats.

- **Union, not override** — these are *sample pools* (more eligible = richer), so they accumulate — the opposite of `l10n`'s override-to-one-value. Don't "fix" one to the other.
- **Always-loaded `*` is a fail-safe** — a sparse locale still yields a non-empty pool; generation never dead-ends.
- **Keep the level structure queryable** — don't pre-flatten; `pick` must address the full accumulated pool *or* a specific level (`en.GB` only).
- **`*` is optional.** Some data is inherently country-specific (driving licences, passports) → region-scoped with *no* global level. Absence of `*` means **no global fallback**, so an unsupported locale yields an empty pool → **fail-closed** (error, don't fabricate). You can't fake a document format you haven't authored, and erroring beats emitting a nonsense one. (Contrast the name pools, where `*` deliberately guarantees a fallback.)
- **Fits the stack:** `c5n` generates the scoped data (its `tree<T>`); doppel hand-writes the accumulate-resolver; matching reuses **`f8n.Locale`** (don't reinvent it).

## Randomiser, `pick` & `template` — the primitives
Shared, public primitives; all seeded off the randomiser.

- **Randomiser** — the seeded RNG, exposing typed **`next{Type}(min?, max?)`** methods (`nextInteger`, `nextDouble`, `nextDate`, …) — the base under `pick`/`template`. **Own the PRNG in both languages, not the platform RNG:** TS has no seedable built-in; C#'s `System.Random(seed)` isn't reproducible across .NET versions (algorithm changed) → seeds would silently stop reproducing on a runtime upgrade. Own a small documented PRNG (same "own it, don't trust the platform" discipline as `f8n` ISO parsing). Seedable, or auto-random. **Per-language determinism; cross-language not a goal** (that would drag the whole generation path onto a golden-vector surface; nobody needs `persona-42` byte-identical across languages). *Cheap partial seam if ever wanted: same algorithm both languages.*
- **`pick`** — overloaded selection over a pool: uniform, **weighted**, **locale-scoped** (draw from a specific language/region, not just the current-locale union), and combinations. **Seeded.** Loading is *eligibility*; `pick` is *selection*. Scoped `pick` also buys **deliberate diversity** (a `fr` name in a GB population on purpose). Weights come from two places: *in the data* (per-entry frequency → realism) and *at the call site* (level-blend / generator policy).
- **`template`** — a pattern-based string generator: **char classes + literal characters** (e.g. `[0-9][0-9]-[0-9][0-9]-[0-9][0-9]`), one seeded `pick` per class position. The primitive for **synthetic identifiers**. **Templates are locale-scoped data** (`* > en > GB`), with **multiple weighted variants** per identifier (e.g. postcodes have several shapes) → `pick` a template by weight, then fill it. Variable-length formats are handled by variant-selection, not regex quantifiers.

### Temporal generation & the reference clock
Date/Time/DateTime get **domain-aware** generation — optional **component bounds** (year/month/hour) with **sensible defaults** (e.g. year 1900–2099), and **`future(from)` / `past(from)`** relative to an anchor. `future`/`past` are the **temporal arm of the coherence engine**: a persona's timeline is a chain of anchored dates — `reference-now → DOB (past, bounded to a plausible age) → birth-year (drives the name cohort) → employment start (past, after ~18) → account opened → transactions → now` — each generated *relative to a prior anchor*, so it stays ordered and consistent.

- **Returns `f8n` temporal types** (`IsoDate`/`IsoTime`/`IsoDateTime`), not platform date types — cross-language consistency + reuse.
- **The reference "now" is injected, never wall-clock.** Otherwise seed-reproducibility breaks — a DOB derived from wall-clock `now` would age every run, so a shared persona would drift. Reuses `f8n`'s **injected-clock** discipline (`TimeProvider`, never `DateTime.UtcNow`); the as-of reference is part of the run's reproducible state, so a seed reproduces the *same* persona whenever it runs.

## Synthetic identifiers
Bank/business entities imply **account numbers, sort codes, IBANs, BICs, company numbers, VAT numbers** — none from `f8n`; doppel-synthetic, built by the `template` fn from locale-scoped weighted templates.

**Validity is a deliberate safety fork (the identifier-level dual-use gate).** The template gives *format*-validity; checksum-validity is separate:
- **Safe by default** — a reserved/**test range** where the format has one (test cards, test IBANs — valid *and* known-fake), else **checksum-invalid** (fails any validator → provably fake, unusable).
- **Checksum-valid** — a **gated opt-in**, for test data that must pass validation; raises misusability, flagged.
- Per-identifier: IBAN/card/VAT have checksums; NI/company-number are format-only; sort codes have no checksum but *map to real banks*, so they want reserved/test values regardless.

## Provably-fake by construction
The "unmistakeably synthetic" north star made concrete — wherever a generated value could **collide with or impersonate a real one**, use the officially-reserved fiction/test range, or make it provably-invalid. One rule, many surfaces:
- **Identifiers** (IBAN, card, VAT…) — reserved/test ranges, else checksum-invalid.
- **Emails/domains** — RFC 2606/6761 reserved (`example.com`, `.test`, `.invalid`), or fictional org domains.
- **Phones** — reserved fictional number ranges (Ofcom `07700 900xxx`, NANP `555-01xx`).
- **Addresses** — generic street + random number; real postcode *prefix* + synthetic tail.
- **Documents** — checksum-invalid MRZ/numbers; images watermarked/omitted.
- **Document photos** — gendered grey silhouettes, never a real/realistic face (no likeness, no biometric artifact).
- **Bank/company names** — fictional, never real brands.

*Reserved-or-invalid, never coincidentally-real.*

## API shape
- **`Doppel.EnGB(42)`** — locale fixed at the root (accumulated pools resolve here), seed optional (`(42)` reproducible / `()` random). One typed factory per configured locale (`c5n`-generated).
- **Domain namespaces** — `doppel.Identity.Adult()`/`.Child([…])`, with `.Finance`/`.Business` alongside. Discoverable, grouped generators.
- **Generators take dependencies** — coherence via passed entities.
- **C# + TS** — cross-language via *shared data* (`c5n`-conformant) + *shared behaviour shape*, deliberately **not** seed-identical output.

## Showcase & demo
The **visual `f8n` + `l10n` demo** — coherent personas are money-heavy (accounts, transactions, statements, salaries), shown *visually*. **Persona explorer:** *"generate a person"* → a coherent persona; drill into each feature; a **locale switcher** (e.g. UK/DE/FR/CY) makes the combined coherence tangible in one click. Client-side (browser TS → static site); **seed-based**, any persona shareable by URL.

## Dual-use flag (design-rigour) — the pre-public gate
Fake **documents** (statements, IDs) as images are misusable (fraud), and a *public* generator raises exposure. Irreversible once public. Decide **before** any public surface:
- **Lead with the coherent-data explorer**, not photo-realistic documents.
- **No real branding/logos** on generated documents (trademark + fraud); fictional names already help.
- **Heavily watermark** document images, or **omit** IDs/passports from the public demo entirely.
- Realism at the level of *facts*, never *authenticity*; synthetic identifiers in reserved/test ranges.

**The Document module is the sharp end** — fake licences/passports are the highest-misuse output, so the controls concentrate there, harder than the general policy:
- **Numbers and MRZ: checksum-invalid by default** — an ICAO 9303 MRZ has check digits; a *valid*-MRZ passport could pass automated readers (a serious fraud artifact). Valid checksums here are the extreme case (never, or maximally gated).
- **The data is fine; the image is the risk.** The person's document *fields* are just coherent data (the safe explorer); misuse lives in the rendered **image** and valid identifiers — so watermark/"SPECIMEN", no real security features, or **omit document images from any public surface**. Real authorities are factual as data, but an image bearing them + a valid number is a fraud tool.

## c5n split
- **c5n generates:** the locale factories (`Doppel.EnGB` per configured locale) and the reference **data** — locale-scoped `tree<T>`, weighted lists (names-by-year, banks, surnames, roads), identifier **templates**, and address field-set + per-locale **format** data.
- **Hand-written:** the **randomiser**, **`pick`**, **`template`**, and the **generators** (the coherence algorithm). These are what would join the golden-vector harness *if* cross-language determinism were ever wanted.

## Deferred (rooms)
- **Exotic surname/name conventions** — Spanish double-surname, Icelandic patronymic, keep-maiden-name (per-locale generators).
- **Document-image rendering** — via the **`svg` project** (SVG templates → PNG; grey-silhouette photos); watermark/omit policy details. Depends on `svg` (a later-layer build).
- **Cross-language seed portability** (shared conformant PRNG + aligned generation path + vectors).
- **Population-scale pooling** beyond the curated bank pool (shared employers; referential business↔business customer/supplier graphs).

## Decisions log
- **Coherence = caller-composed graph + generator-enforced edges** (local rules, no global solver); generation order emerges from conditioning.
- **Entity graph:** Person + Business{generic, bank, not-for-profit}; composed by reference; bank recursion bounded; curated bank pool; pooling for populations.
- **Identity:** gender `Other|Male|Female` (`Adult(gender?)`); first names conditioned on (birth-year, gender) from open cohort data, per-year frequency = weights, Other = union; titles gender-skewed/weighted/locale-scoped/age-aware; surnames a separate locale-scoped list.
- **Family coherence via a locale-scoped surname policy** (en-GB default = male+female → male's, else random; child follows male in male+female marriage else random parent); exotic conventions deferred; marital status + spouse-as-entity.
- **Reference data: real structure + synthetic leaf**; open-data geo via **Wikidata (CC0)** not Wikipedia; generic road + random number; **postcode = town's real prefix + synthetic tail** (prefix scopes to town; per-locale prefix structure).
- **Locale is a runtime dimension, not a type dimension** (the biggest change) — uniform entity types + locale-scoped data + locale-keyed strategies; **never locale subclasses or generics**. Locale binds to the `Doppel` instance; entities stay plain (`Person`, not `UkPerson`). Address = CLDR field set + per-locale format via `l10n`.
- **Names fictional** for banks/companies (trademark + fraud-amplification avoided).
- **Locale-scoped data**, `* + matching` **union** (accumulate, not override); `*` always (fail-safe); levels queryable; matching via `f8n.Locale`.
- **Own PRNG per language** (not platform); **per-language determinism**, cross-language deferred.
- **`pick`** overloaded (weighted/scoped/seeded); **`template`** = classes + literals, locale-scoped weighted variants.
- **Typed randomiser API** (`next{Type}(min?, max?)`, component-bounded dates with sensible defaults, `future`/`past` from an anchor) returning **`f8n` temporal types**; **reference clock injected** (not wall-clock) so seeds stay reproducible (reuses f8n's injected-clock).
- **Synthetic identifiers** safe-by-default (reserved/test range or checksum-invalid); checksum-valid a gated opt-in.
- **API root `Doppel.EnGB(seed?)`**; domain namespaces; dependency-passing generators.
- **Histories via anchored sequences** (each element's date anchored to the previous — address/employment/transactions).
- **Financial instruments** — BankAccount (type/bank/currency/number/sort-code/IBAN/SWIFT) + PaymentCard (network/number/CVV/expiry/holder-from-person); **identifiers bank-derived** (IBAN/SWIFT/sort-code encode bank+country → coherence) with **invalid check digits / published test numbers** (safety).
- **Amounts/transactions coherence-bounded** — currency from account/locale; transaction dates within the account's window; category from locale-scoped data. Optional depth: category-conditioned amounts + regular (salary/rent) vs ad-hoc transactions.
- **Person field set** — prefix/first/middle/last name, gender, DOB, email, phone (`f8n.PhoneNumber`, reserved ranges), country of birth/residence + place of birth (`f8n.Country`), height (gender-conditioned), eye colour, address + history, job title (compositional, industry-coherent), tax reference.
- **Provably-fake by construction** — one rule across identifiers/emails/phones/addresses/documents/names: *reserved-or-invalid, never coincidentally-real.*
- **Organisation module** — company/bank/not-for-profit (shared core + type fields: company-type, reg/tax/VAT/charity numbers, SIC industry, Fitch, employee count, domain/URL/email); the business entities persons reference. Domain taxonomies (SIC/Fitch/company-type) are **doppel's own** locale-scoped reference data (per f8n's scope boundary). Org names **compositional** (locale-scoped, type-specific part pools); registrationDate a coherence anchor.
- **Domains/emails non-colliding** — reserved domains (RFC 2606/6761: `example.com`/`.test`/`.invalid`) or fictional org domains; never real providers with plausible local parts. Org email = `local@fictional-org-domain` (coherent + safe). TLDs `* + region`.
- **Document module** — `Document.X(person)` = coherent projection of a Person + synthetic doc fields (spec-formatted numbers, ICAO MRZ, issue/expiry, authority, real categories); expiry = issue + validity.
- **`*` optional in locale data** — country-specific data (documents) is region-only; absent `*` → fail-closed for unsupported locales.
- **Dual-use gate** before any public surface; **documents are the acute case** — numbers/MRZ checksum-invalid by default; safety on the *image* (watermark/specimen/omit), not the data.
- **c5n generates data + locale factories; behaviour hand-written.**

## Change log
- 2026-07-09: **elevated "locale is a runtime dimension, not a type dimension" to the core architectural stance** — the biggest deliberate change from a naive locale-aware model. Locale variation → locale-scoped **data** + locale-keyed **strategies**; entities stay **uniform types** (no `UkAddress`/`UkPerson` subclasses, no `Address<GB>` generics); locale binds to the `Doppel` instance. Generalised beyond address (address kept as the exemplar); added an early core-stance banner.
- 2026-07-09: added **document rendering** — SVG templates with text placeholders (localised via `l10n`, locale-scoped per document type) rendered via the **`svg` project** (→ PNG, a later-layer dependency); **gendered grey-silhouette photos** (never a realistic face) elevated to a first-class safety lever (added to provably-fake + dual-use).
- 2026-07-09: added **financial instruments** — BankAccount (type, bank, currency, number, sort code, IBAN, SWIFT/BIC) + PaymentCard (network, number, CVV, issue/expiry, holder-from-person); **bank-derived identifiers** (IBAN/SWIFT/sort-code encode bank+country = coherence) with invalid check digits or published test numbers (safety); `f8n.Currency`/`f8n.Money` reuse.
- 2026-07-09: enumerated the **Person field set** (adds middle name; phone via `f8n.PhoneNumber` in reserved fictional ranges; country-of-birth/residence + place-of-birth via `f8n.Country`; height/eye-colour; compositional industry-coherent job title; tax reference) and consolidated the **"provably-fake by construction"** principle (reserved/test ranges or invalid — identifiers, emails/domains, phones, addresses, documents, names).
- 2026-07-09: added the **Organisation module** — company/bank/not-for-profit (shared core + type-specific fields: SIC industry, Fitch, VAT/charity/registration numbers, locale company-types); domain taxonomies (SIC/Fitch/company-type) as doppel's own locale-scoped reference data (per f8n's scope boundary); **compositional type-specific org names**; registrationDate as a coherence anchor; **domains/email** (TLDs `*`+region; free/paid/organisation) with a **non-collision safety rule** (RFC 2606/6761 reserved domains / fictional org domains, never real providers).
- 2026-07-09: added the **Document module** (DrivingLicence/Passport take a Person → coherent projection + synthetic doc fields incl. ICAO MRZ; expiry = issue + validity; real licence categories); **`*` now optional** in locale data (country-specific docs region-only → fail-closed for unsupported locales); **sharpened the dual-use gate for documents** (numbers/MRZ checksum-invalid by default; safety on the image, not the data).
- 2026-07-09: added **Histories, finance & amounts** — anchored-sequence histories (address/employment/transactions), `Money` amounts (min/max, currency from account/locale), transactions (dates clamped to the account window, amount + category), locale-scoped categories; flagged optional depth (category-conditioned amounts, regular-vs-ad-hoc incl. salary-income coherence).
- 2026-07-09: added the **typed randomiser API + temporal generation** — `next{Type}(min?, max?)`, component bounds with sensible defaults (year 1900–2099), and `future`/`past`-from-anchor as the temporal-coherence arm (`reference-now → DOB → birth-year → name → employment/accounts/transactions`); returns `f8n` temporal types; **injected reference clock** (not wall-clock) for seed-reproducibility.
- 2026-07-09: **folded in the "more" pass** — Identity section (names-by-birth-year+gender, gender enum, gender-skewed titles, surnames + locale-scoped family surname policy); **locale-polymorphic structures** resolution (one uniform `Address` type + CLDR field-set/format data, never generics — with the generics-leak rationale); postcode **prefix-scoping** (town's real prefix + synthetic tail); the **`template`** primitive (classes + literals; locale-scoped weighted templates) + the identifier **checksum policy** (safe-by-default, valid opt-in). More sections still to come.
- 2026-07-09: core design settled (superseded the scaffold agenda) — coherence engine, entity graph, real-structure/synthetic-leaf reference data, fictional names, locale-scoped union-loading, own-PRNG + `pick`, `Doppel.EnGB(seed?)`, synthetic identifiers, demo, dual-use gate, c5n split.
- 2026-07-08: created — scaffolded as a forge project dir; seed frame + open agenda.
