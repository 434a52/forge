# l10n — pressure-testing the codegen shape

`DESIGN.md` → *Codegen structure* is marked **candidate**, and `../c5n/DESIGN.md` repeats the
caveat: a codegen-native redesign, well-reasoned, **not yet validated against a real message
set**. This is the validation, and it is deliberately paper-only — the point is to find what
breaks before anything is built on it, and before `c5n` grows `tree<T>` and typed-shim emit to
serve a shape that might not hold.

**What is under test.** Three parts, from `DESIGN.md`:

1. Generated — a `tree<T>` of **typed shims**, signature derived from the `{arg:hint}` hints.
2. Generated — the message as **data** (params + recursive parts), the DSL parsed at build time.
3. Hand-written — one **`render` interpreter** per language, which is the entire conformance
   surface.

**How to use this doc.** Each case below is a message *shape*, the question it forces, and
three artefacts to write out by hand: the **typed signature** (both languages), the **message
data**, and the **runtime call**. A case passes when all three can be written without
inventing new machinery. Where a case needs something the candidate does not have, that is the
finding — record it rather than patching around it.

Each case carries a **stand-in** so the walk can start immediately, and a slot for the **real
example**. Stand-ins are derived from `DESIGN.md`'s own hard cases; they are not a substitute
for a real corpus, which will have edges nobody invents.

---

## Already found, before any case was walked

**The candidate describes one tree where there are two.** "A `tree<T>` of typed shims" and
"the message as data" are named as parts of one structure, and they come apart the moment a
second locale exists:

- **The typed-shim tree is locale-neutral.** One of it. Its signatures derive from the
  *canonical* (en-GB) message's hints.
- **The message-data trees are per locale.** Welsh carries six plural branches where English
  carries two, so the data differs per locale while the signature must not.

Different leaf types, different output layouts, different tree-shaking needs — and `c5n` would
be emitting both from related but distinct sources. **This implies a rule `DESIGN.md` leaves
unstated:** the signature comes from the canonical message *only*, and every locale's data must
conform to it. That is checkable at build time, and it is what makes the pipeline's "protect
structural tokens, translate prose only" load-bearing rather than merely tidy — a translator
who introduced an interpolation would otherwise produce data the shim cannot call.

Carry this through every case below: **write the artefacts for two locales, not one.**

---

## The lens the last seven cases were walked with

**The harness can only drive an input and compare an output.** Everything else is invisible to
it. So:

> **Anything not reachable from an input or visible in an output is untestable — and untestable
> code inside the conformance surface is where two languages drift apart in silence.**

That is not a testing preference; for this design it is a *design* constraint, because
conformance is the entire claim. Applied, it decides cases 2, 8 and 9 outright and sharpens 4
and 11: ambient locale is out (2), the escape bug lives above the boundary vectors can reach
(8), platform date formatting is out (9), an unreachable fallback branch becomes a throw (4),
and a runtime fallback chain loses to one resolved at lowering (11).

## Cases

Ordered roughly by how much they can break. `⚠` marks the ones I expect to be load-bearing.

### 1. No arguments — **RESOLVED**
**Stand-in:** `Save`
**Real:** *(to add)*

**A sole-literal message lowers to a bare string, not a one-element parts list.** One branch in
the interpreter — *if the data is a string, return it* — in exchange for most of a real corpus
skipping the loop entirely. It is a difference in data *shape*, so both languages must agree on
it, so a vector covers it.

**And the trap the case was hiding: arity is not the discriminator between a message and a
constant.** A zero-arg message and a constant are both "a string with no arguments", so arity is
the tempting answer and it is the wrong one:

> **A message is translated prose. A constant is an authored per-locale value.**

Different pipelines, different review, different failure modes — case 12's approval gate applies
to one and not the other. So the kind is **declared**, never inferred from shape.

Keeping zero-arg messages as **functions** while constants emit as **values** (case 14) then
keeps the two leaf kinds distinct at the call site: `Messages.Save()` against
`Constants.SupportPhone`. *Held rather than proved:* .NET's own resx emits string resources as
**properties**, so a C# developer might well have written a property here and the
emitter-takes-the-pain test is genuinely arguable. The tiebreaker is leaf-kind legibility, not
idiom — worth revisiting if the call sites read badly.

### 2. ⚠ One value hint — **RESOLVED: it locates the conformance boundary**
**Stand-in:** `{limit:currency} per institution`
**Real:** *(to add)*

The signature is uncontroversial — `PerInstitution(Money limit)` / `perInstitution(limit: Money)`.
The locale question is the case, and it turns out to place the conformance boundary for the whole
design.

**Ambient locale is disqualified, and not on taste.** If the interpreter reads
`CultureInfo.CurrentCulture` or JS's default `Intl` locale, a vector must set a hidden input out
of band — and the two mechanisms resolve differently, so the harness would be comparing two
functions that do not take the same arguments. **Already paid for once:** `LocalDate.ToString`
was `CurrentCulture`-sensitive where `Percentage` was invariant, and only a vector found it.

The tension that looks fatal — locale must be explicit, yet `Save(locale)` on every shim is grim
ergonomics — dissolves once the boundary is placed precisely:

> **The conformance boundary is `render(data, args, locale) → string`.** Below it, ambient
> locale is **forbidden**. Above it, how a shim obtains the locale is an ergonomics choice, free
> to differ per language.

Shims are generated, so they are verified by regenerate-and-diff, not by vectors. C# can
therefore hand back a locale-bound catalogue while TS hands back a bound `t` from a composable —
idiomatic divergence, no conformance hole. **The emitter-takes-the-pain rule applies above the
boundary and stops dead at it**, which is the first limit that rule has been given by something
other than the wire format.

**This sharpens, rather than contradicts, `DESIGN.md`'s *Locale scope — never `CurrentCulture`*.**
That item's "controlled ambient" scope — an `AsyncLocal`-flowed `IDisposable` with an explicit
override — is correct, and it lives **above** the boundary: it resolves a locale and hands it
down. What must not happen is a formatter *below* the boundary reading it.

**The invariant, stated:** *nothing below the boundary reads ambient culture.* Every `f8n` and
CLDR call inside `render` takes locale explicitly. That is the `ToString` incident generalised
into a rule — the same shape as f8n's *injected clock, never `DateTime.Now`*.

### 3. ⚠ Plural with coupled digits — **RESOLVED: the coupling is removed**
**Was:** `{count:plural|one={count:number-0} item|other={count:number-0} items}`
**Now:** `{count:plural|one={count} item|other={count} items}`

The `-0` was doing two jobs: printing the number (needed) and declaring its precision, which
CLDR uses as the `v` operand (the source of the rule). **Plural drives on an integer**, so
`v = 0` structurally and the second job disappears — no `-N` on a plural driver, a bare
`{count}` in the branch, and no rule to check because disagreement is inexpressible.

It also shrinks the runtime: with `v = w = f = t = 0` and `n = i`, the interpreter evaluates
**one operand rather than six**, and clauses like Latvian's `v = 2 and f % 100 = 11..19` are
structurally dead.

**Why this simplification is safe where the day's others were not:**

> A **restriction** makes some inputs inexpressible. An **approximation** makes some inputs
> wrong.

`zero`-means-none accepts 20 and renders a false sentence in Latvian — an approximation.
Integer-only plurals are exactly correct for every input accepted, because the CLDR rules
reduce for integers; a class of message simply cannot be written. **This is the test to apply
to every remaining simplification in this doc.**

**Seam, named rather than built:** fractional pluralisation returns with **long-form unit
names in prose** ("2.5 kilowatt-hours") — *not* with unit formatting generally, since unit
symbols do not pluralise ("12.5 kWh" is invariant). So ampersand's unit needs do not reopen
this. *Unverified:* that short/narrow unit widths are count-invariant in CLDR — `en.xml`
truncated before the units section and the chart URL 404'd.

### 4. ⚠ A target locale with more categories — **RESOLVED**
**Stand-in:** the case above, in `cy` — six categories where `en` has two.
**Real:** *(to add — and which locales are actually in scope, with their category sets)*

Category selection cannot happen at build time, because the count is not known then. So rule
evaluation is necessarily runtime, and the only real question is which half is data and which is
code. **The two halves go opposite ways:**

- **The rule table is generated per locale**, from one CLDR extraction. Hand-written per language
  it would be two copies of ~200 locales' rules, and they would drift. This is `c5n`'s own value
  proposition applied to l10n itself.
- **The evaluator is hand-written per language**, inside the conformance surface, driven by
  vectors that walk counts per locale — with **Latvian named explicitly**, per the fixture
  consequence recorded on 2026-08-27.

**Case 3 pays a second dividend here.** With integer-only drivers the rules can be **pre-reduced
at extraction time**: Latvian's `v = 2 and f % 100 = 11..19` clause is dropped by the extractor
rather than evaluated-and-failed at runtime. Smaller table, smaller evaluator, and the
restriction is doing work in a place it was not designed for.

**And a correction to `DESIGN.md`:** it says the runtime falls back to `other` where a category
is missing. Under the two-trees build check — a locale's data carries exactly its own category
set — that path is **unreachable**, and unreachable code inside the conformance surface is
untestable by construction, which is precisely where divergence hides. It should **throw**. The
vector format already expresses that: `"error": true` appears in six of the eight `f8n` vector
files.

### 5. ⚠ Nested selectors — **RESOLVED: nesting is removed, and the shape gets simpler**
**Stand-in:** `{who:select|self=You have {n:plural|one=an account|other={n} accounts}|other={name} has {n:plural|…}}`
**Real:** *(none — nesting has not been needed in accounting or banking work across several
builds of this stack. That absence is itself evidence, and it is what made the question worth
asking rather than assuming.)*

**This is the first case to test the candidate's actual claim** rather than its syntax, and the
claim holds — better than holds. Nesting comes out entirely, and what replaces it makes the
data model *smaller*.

#### The discriminator: select keys are locale-invariant, plural categories are not

`male`/`female`/`other`, `self`/`other` — author-chosen and identical in every language. Plural
categories are decided **per locale**: two in English, six in Welsh, and which apply is not the
author's choice.

That maps exactly onto the two trees found before any case was walked:

| | selection | lives in |
|---|---|---|
| **`select`** | locale-**invariant** keys | the **locale-neutral shim tree** |
| **`plural`** | locale-**varying** categories | the **per-locale data trees** |

#### So `select` hoists into the shim, and `plural` stays in the data

```yaml
invite:
  female: "{host} invites {guest} to her party"
  male:   "{host} invites {guest} to his party"
  other:  "{host} invites {guest} to their party"
```

`c5n` generates **one** shim — `invite(gender, host, guest)` — which switches on `gender` and
calls the right message. The authored messages are whole sentences, the generated code does the
outer selection, and **the message data never nests.**

#### What it buys

- **The parts model stays a flat list.** No tree, no recursion — the interpreter is a loop.
  That is the conformance surface *not* growing, which is what this case existed to test.
- **`select` becomes a generated enum.** Its keys are a closed set known at build time, so an
  invalid value is a **compile error** rather than a silent fall-through to `other`. The typed
  north star reaches somewhere it currently does not.
- **Translators get whole sentences.** Nesting produces *fragments* — three genders × six Welsh
  categories is eighteen fragments for one sentence — and fragment assembly is the classic
  route to poor translation, because nobody can see what they are agreeing with. Three complete
  sentences is better practice independently of what it does for the engine.
- **Nesting leaves the syntax**, and its interactions with escaping and with the multiplicative
  per-locale data go with it.

#### Limits, stated rather than discovered later

- **`plural` inside `plural` is not expressible.** No real message has been constructed that
  wants it.
- **A `select` whose branches differ only inside one plural case** duplicates the whole message
  per key. Correct, just more keys — and they are authored sentences rather than fragments.
- **Message count multiplies by select cardinality.** Fine at three; a `select` with many keys
  is almost certainly data-driven and should not be a message selector at all. **Worth a stated
  limit** rather than left to judgement.

### 6. Branches using different arguments — **RESOLVED, after the case changed shape**
**Stand-in:** `{plan:select|free=Upgrade for {price:currency}|paid=Renews {renews:date-medium}}`
**Real:** *(to add)*

**Case 5 dissolved the stand-in as written** — `select` hoisted out of the message and into the
shim — so this is now two sibling messages with different signatures, and the live question is
whether `c5n` emits a dispatcher over them at all.

**A union dispatcher is a trap.** `Plan(PlanKind plan, Money price, IsoDate renews)` either
forces the caller to invent a `renews` on the free branch, or makes it nullable — and a null on
the branch that needs it is a **runtime** failure, in the one case that exists to test the typed
claim.

The tempting rule is *emit dispatch when the branches are signature-compatible*. **It is the same
inference failure as case 14's resolve-by-lookup:** add an argument to one branch and the
dispatcher silently disappears from the API, breaking every call site with an error that names
the wrong thing.

**So dispatch is declared, not inferred.** The author marks a sibling group as dispatchable;
adding a divergent argument then fails the build with *branches are no longer
signature-compatible*, which names the actual problem. Second time this lesson has been paid for
in this doc, which is fair evidence it generalises:

> **Structural facts are declared, never recovered by lookup.** Inference makes an unrelated
> edit change an API silently; a declaration makes it fail by name.

### 7. ⚠ Exact-value match — **REAL, resolved**
**Real:** `{count:plural|none=no accounts|1=account|other=accounts}` — how it would be written
where zero is reachable.

**Resolved: bare numeric keys — `0=`, `1=`.** No `none`, no `=` sigil.

*This one reversed twice before settling; the reasoning is kept because each turn found
something, and the final position rests on the last of them.*

**Why not ICU's `=0=`.** The doubled `=` is an artifact of pairing ICU's `=N` with this
design's `key=value` pipes — ICU never makes you write it. It also forces `DESIGN.md`'s
second-`=` parse wrinkle. Dropping the sigil removes both, and loses nothing: **categories are
words and exact values are numbers**, so a bare number can never collide with a category name.
One rule — split on the first `=` — covers every key.

**Why not `none`.** It is the *divergence* from CLDR, not the alignment: ICU has `=0` and no
`none`. Under a stance of *stay closer to CLDR, simplify only where we should*, it loses. And
it cannot be extended — see the principle below — so choosing it would drop exact matches
beyond zero permanently.

**Why the word/number mix is not arbitrary.** ICU makes the same distinction, writing `=0` and
`one` in the same braces. Numeric-versus-word is CLDR's own line; this just spells it without
the redundant character.

#### The principle: borrow a category name and you inherit its grammar, not its number

Verified against `plurals.xml` and `ordinals.xml`. Three instances, each stronger than the last:

- **`zero` is not 0.** Latvian's cardinal `zero` is `n % 10 = 0 or n % 100 = 11..19 …` — it
  matches 0, 10, 11…19, 20, 30. `zero`-as-"none" renders *"no accounts"* for twenty accounts.
  (Eight locales do define it as `n = 0`; `lv` and `prg` do not, and one is enough.)
- **`two` is not 2.** Breton's `two` is `n % 10 = 2 and n % 100 != 12,72,92` → 2, 22, 32, 42 but
  **not** 12, 72, 92. Scottish Gaelic's is `n = 2,12`; Slovenian's 2, 102, 202; Manx's 2, 12,
  22. Six-plus languages, against Latvian's one.
- **The same word differs *within* one language.** Welsh's `zero` is `n = 0` for **cardinals**
  and `n = 0,7,8,9` for **ordinals**. Same name, same locale, two rule tables, different sets.

The numeric-looking category names are a convenience of the spec, not a definition. **`none`
is safe only because CLDR never used the word** — it sits outside the namespace and carries no
grammar to inherit, and there is no second free word.

#### Ordinals are missing entirely, and that is the larger CLDR gap
CLDR defines ordinal rules **separately** (`ordinals.xml`, `<plurals type="ordinal">`, roughly
120–130 locales). Neither `DESIGN.md` nor `inline-formatting.md` mentions them; there is no
`selectordinal`-equivalent hint.

English needs **four** ordinal categories where it needs two cardinal ones:

| count | rule | matches |
|---|---|---|
| `one` | `n % 10 = 1 and n % 100 != 11` | 1st, 21st, 31st |
| `two` | `n % 10 = 2 and n % 100 != 12` | 2nd, 22nd |
| `few` | `n % 10 = 3 and n % 100 != 13` | 3rd, 23rd |
| `other` | — | 4th, **11th, 12th, 13th** |

This is why ICU has `plural` and `selectordinal` as distinct keywords: **the hint selects the
rule table.** The hint-driven design here would handle it correctly if the hint existed — the
gap is the hint, not the architecture.

Whether ordinals are wanted is a corpus question ("1st January" is usually a `date` hint's job;
rankings are rarer). But under *stay closer to CLDR* it should be **named as deferred rather
than absent**, since nothing currently records that the capability exists and was passed over.

**And it reinforces two other cases.** The example never displays `count` at all, so case 13's
`v` operand is undefined for the whole message — the exact branches do not rescue that, since
`other` is still selected by a rule that needs `v`. And as first written it carried `many=`,
which is not an English category, so the branch can never fire: case 13a arriving by accident
for the second time in two real examples, which is reasonable evidence that an inapplicable
category key should fail the build.

### 8. ⚠ Escapes — **RESOLVED, and it needs a harness rung that does not exist**
**Stand-in:** `Use \{braces\} and a pipe \| here`
**Real:** *(to add — do real messages hit these in anger?)*

**The data holds the resolved literal.** `Use \{braces\}` lowers to the characters
`Use {braces}`. Anything else pushes DSL syntax into the interpreter, which is exactly what this
design removed. Consequence, stated rather than discovered later: **message data does not
round-trip to source** — nothing distinguishes `\{` from a brace that never needed escaping.
Fine; the source is the source of truth and the data is a build artefact.

**The finding is where the bug actually lives.** The data is emitted **as C# and TS source
literals**, so every literal is re-escaped *for the target language* — quotes, backslashes, `${`
inside a template string. That is precisely *a value quietly altered in transit*: invisible on
the page, obvious to a test.

**And a normal vector cannot reach it.** Vectors drive `data → output`; this bug lives **above**
the interpreter, in the emitter. Catching it needs the input to be **DSL source**, run through
generate → compile → execute in both languages.

So the interpreter gains **zero** escape logic — the conformance surface does not grow — but the
**harness** gains a rung. Under *we can only test input and output*, that is the honest reading:
the test has to start where the input really is. `DESIGN.md` calls this the archetypal
demo-passes-prod-breaks bug; the reason it earns that name is that the obvious place to test it
is one layer below where it happens.

### 9. ⚠ A temporal hint — **RESOLVED, and it fires a deferred trigger**
**Stand-in:** `Renews on {renews:date-medium}`
**Real:** *(to add)*

**The interpreter formats.** If the caller pre-formats, the parameter is a `string`, the typed
north star dies for every temporal message, and the two languages can format differently with
nothing to catch it.

**But it must not delegate to the platform.** `Intl.DateTimeFormat` and .NET's `CultureInfo` are
two different ICU copies at two different versions. They disagree, and the disagreements *move* —
ICU 72 changed the space before AM/PM to a narrow no-break space and broke test suites across the
JS ecosystem. Delegate, and the harness goes red on a **runtime upgrade** rather than on a code
change: the worst possible signal, because it teaches you to distrust the test. *(High confidence
on the general fact; the specific version behaviour is exactly the sort of thing to pin rather
than trust — see the deferred toolchain-pinning item.)*

So the interpreter applies **CLDR patterns from generated per-locale data** — which is what
`DESIGN.md`'s "pattern-applier" already said, now with its reason. The same reason `f8n`
hand-writes `Money` rather than leaning on `decimal.ToString("C")`.

**This is not a third tree.** The per-locale tree becomes *everything CLDR says about this
locale, plus this app's messages* — message data, the plural rule table (case 4), date and time
patterns, number symbols — dynamically imported as one unit. That is what makes registration and
dynamic import (cases 10 and 12) **one** mechanism rather than three.

**It fires the CLDR-extractor trigger**, deferred with its trigger named as *l10n locale
formatting*. This is that moment.

### 10. ⚠ Deep namespace and tree-shaking — **RESOLVED from production**
**Real:** a shared catalogue across several teams — a shared subtree for UI chrome and
constants (support numbers, emails, company registration number), plus a namespace per team.

**The two writers differ in output *layout*, not merely in syntax**, and that is correct rather
than a problem:

| target | shape | why |
|---|---|---|
| **TS** | per-locale exports, locales **dynamically imported**, tree-shakable throughout | bytes are the cost in a browser, and a bundler can only drop what it can prove unreferenced — which needs static imports, not one nested object |
| **C#** | **a single package, everything in it** | the consumer is a server: the assembly loads once and a few hundred KB of strings is nothing |

So c5n's writers own their **file partitioning**, not just their spelling — a heavier difference
in kind than `partial class` versus `*.data.ts`, and the last plausible way this design could
have turned out worse than it looks. It didn't.

**And the two-trees split decomposes the problem cleanly**, which is why it is not one question:

- **The shim tree is locale-neutral**, so **static imports and tree-shaking** select it. No
  declaration required; the bundler does it.
- **The data trees are per-locale**, so **dynamic import** loads them, selected by an explicit
  **registration** (case 12).

Different mechanisms for different trees. Arrived at here from plural categories, and in
production from bundle size — same structure either way.

### 11. A message a locale does not have — **RESOLVED**
**Stand-in:** a key present in `en-GB`, absent in `cy`.
**Real:** *(to add)*

**Build failure is wrong** — case 12 established that gating *generation* stalls development
behind translation. **A runtime fallback chain is wrong too:** the interpreter would need
multi-tree lookup, vectors would have to drive *sets* of trees rather than one, and a missing
message stays invisible until it renders.

**Fallback resolves at lowering.** Welsh's tree carries the en-GB message for that key; every
tree is complete; lookup is a single hit; the interpreter gains no fallback path at all. It is
the same resolver constants already use (case 14) — **one mechanism, not two** — and it settles a
structural question below: locale fallback is **generated, not runtime**.

**The interaction this case predicted with case 4 is real, and costs one field.** A fallen-back
message carries the *source* locale's categories, so case 4's *data matches this locale's
category set* check would fire falsely — unless each entry is **tagged with its source locale**.

That tag is not overhead invented to satisfy a check. It is independently required for `lang=` on
the rendered fragment: without it a screen reader pronounces English prose with Welsh phonetics.
**WCAG 3.1.2 *Language of Parts***, landing on the l10n × a11y seam `DESIGN.md` already names —
and the same tag surfaces untranslated strings in dev mode.

**Cost:** duplication, bounded by *translation debt* rather than corpus size, and trending to zero
as translation completes. That is the right shape for a cost to have.

### 12. An unapproved translation — **RESOLVED from production**
**Real:** a client-side **registration step on init** naming the namespaces an app will use,
with runtime checks that only registered namespaces can be used.

**Approval gates the *deploy*, not the *generation*.** That is the right way round: gate
generation and an unapproved string breaks a local build, so development stalls behind
translation. Instead everything generates, dev falls back to the source locale, and the door
is where it stops.

**And registration is what makes the gate affordable.** Without it the gate must examine every
message in every shipped locale, so a half-translated namespace nobody uses blocks a release.
The declaration narrows it to the surface actually in play — and the *"only registered
namespaces may be used"* check is what makes the declaration trustworthy rather than a manifest
that drifts.

**One declaration, three jobs:** what to dynamically load, what the deploy gate must check, and
what the runtime will permit.

**It stays a runtime check because the catalogue is shared.** A build-time version — c5n emits
only the registered namespaces, so using an unregistered one is a *compile* error — is strictly
stronger and is available only where the generated package serves one app. A catalogue shared
across teams cannot depend on one consumer's registration without ceasing to be shareable.

*(Ownership hint, not yet a requirement: a shared subtree alongside per-team subtrees suggests
approval gating is naturally **per-subtree** rather than global.)*

### 13. ⚠ A value hint and a selector that is never displayed — **REAL**
**Real:** `You have {balance:currency} in {count:plural|one=account|other=accounts}`
**Forces:** three things, and the middle one is a gap in the design.

- Two arguments of different kinds in one message — one value hint, one selector — so the
  signature draws from two hint families. Expected to be fine; confirm.
- **`count` selects but is never rendered.** ~~There is no digit count, so `v` is
  unspecified.~~ **Resolved by case 3:** a plural argument is an integer, so `v = 0` whether or
  not the value is displayed. The gap closes without a default having to be chosen.
- **What type is a bare `plural` argument?** **Resolved: an integer.** The hint table's
  `number-N` → `FixedDecimal` mapping does not apply to a plural driver.

### 13a. A mistyped category name
**Real:** the case above, as first written: `{count:plural|ome=account|other=accounts}`
**Forces:** `ome` is not a category in any locale's set. **Is that a build error?** If not, the
branch is unreachable and every count silently renders `accounts` — including "1 accounts",
which reads as a bug in the prose rather than in the tooling. Categories are a closed set per
locale and the front-end knows it, so this should fail at build time and name the message.

*Worth contrasting with the sibling typo in the same string — `acount` for `account`. That one
is prose, and nothing can catch it; it is what translation review is for. A mistyped
**category** is structural and the build should refuse it. The two typos arriving together is
a clean illustration of where the machine boundary actually falls.*

### 14. ⚠ A link whose URL is a locale-varying constant — **REAL**
**Real:** `Do you agree to the [terms & condition]({terms-and-conditions-url})`, where
`terms-and-conditions-url` is a key in a **constants data file**, with per-locale overrides and
en-GB as the default.
**Forces:** `inline-formatting.md` covers `[text](url)` thoroughly — but its own example says
*"external link — URL is a param"*. A constant is a **different binding**, and it is not in
either doc.

- **It is an optional parameter with the constant as its default** *(clarified 2026-08-27)*.
  The caller may pass a URL; if they do not, the constant applies. So the name *is* in the
  signature — as optional — rather than absent from it.
- **Optionality should be declared in the message, not inferred from the constants file.**
  Resolve-by-lookup ("if the name matches a constants key it is optional") has one loud
  failure and one silent one, and only the silent one matters. *Loud:* remove or rename a
  constant and a previously-optional parameter becomes required, failing to compile at every
  call site — fine. *Silent:* **add** a constant whose name collides with an existing required
  parameter and that parameter quietly becomes optional, so call sites that forget it now
  compile and render the constant. Adding a constant would change the arity of unrelated
  messages. Marking it in the message — `{terms-and-conditions-url?}` or similar — means a
  message has to opt in, and nothing can change underneath it. *(Syntax undecided.)*
- **The signature gains an optional parameter**, which must be optional *identically* in both
  languages (`url?: string` / `string? url = null`) or the conformance claim has a hole. Open:
  can a constant default anything other than a string — a `Money`, a date? The answer bounds
  the feature.
- **Fallback.** Per-locale overrides with an en-GB default is the same fallback shape messages
  need (cases 4 and 11). One resolver, used twice — not two mechanisms.
- **It still resolves at lowering time — but into a *default*, not a literal.** Since the
  caller may override, the value cannot be inlined as a plain literal: at render time nobody
  knows yet whether an argument arrived. It does not need a runtime constants tree either. The
  **shim cannot carry the default**, because the shim tree is locale-neutral and the constant
  is locale-varying; so the **per-locale message data carries it**, and the parts model gains
  one node meaning *interpolate `name`, or this literal if absent*. The interpreter gains one
  branch.

  That is still the good outcome: constants are consumed **entirely at lowering**, there is no
  third tree at runtime, and `c5n` never learns they exist. One small part kind rather than
  none is a fair price for the caller being able to override. A constant change is a
  regeneration the drift-guard already catches.
- `terms & condition` carries an `&`, which `inline-formatting.md` already handles
  (auto-escaped by the sink).

**And constants are bigger than this case treats them.** In production they are **first-class
addressable values** — support telephone numbers, emails, a company registration number — used
*directly* by an app (`constants.supportPhone`) and not only as a message parameter's default.

They are **a separate emission path, and they do not emit as functions.** A constant takes no
arguments, so a function would be ceremony; it emits as a **value**. That is `tree<T>`'s
existing leaf rule doing its job — *"the leaf type decides emission: a generated symbol →
nested scopes; a value → a nested data literal"* — with messages taking the symbol branch and
constants the value branch. The design anticipated two leaf kinds; this is the second arriving.

Shape: a **locale-neutral type** naming the keys, and **per-locale data objects** with the
fallback already resolved at lowering (as this case established). The type then gives the
exhaustiveness check for free — a constant missing from *every* locale is a compile error,
while one missing from a single locale simply falls back.

---

## Structural questions, not per-message

- **Two trees, one or two `c5n` collection kinds?** Does the shim tree and the per-locale data
  tree both fall out of `tree<T>`, or does one need something else?
- ~~**Where does locale fallback live?**~~ **Answered (case 11): generated, at lowering.** Every
  locale's tree is complete, the interpreter has no fallback path, and each entry carries its
  source locale — a tag `lang=` needs independently. Output multiplies by the *translation debt*,
  not by the corpus, and shrinks as translation completes.
- ~~**Is a constants file a `c5n` concern at all?**~~ **Answered (case 14): no.** Constants
  resolve entirely at lowering; `c5n` sees per-locale data objects and a locale-neutral type, and
  never learns that constants exist. The preferred outcome, now confirmed rather than assumed.
- **What does `c5n` need that it does not have?** The deliverable of this exercise. Entries so
  far, from the cases actually walked:
  1. **Two trees** — a locale-neutral tree of typed shims, and per-locale trees of message
     data. Different leaf types, different output layouts. *(Found before any case was walked.)*
  2. **The parts model is a flat list, never a tree** — no recursion in the interpreter,
     because `select` hoists out of the data. *(Case 5.)*
  3. **Shim-level dispatch on a generated enum** — the shim switches on a `select` argument
     whose keys are a closed build-time set. *(Case 5.)*
  4. **A parameter node carrying a default** — for a constant supplying an optional argument's
     value, resolved at lowering. *(Case 14.)*
  5. **Writers own their output layout, not just their syntax** — TS emits per-locale,
     dynamically-importable, tree-shakable units; C# emits a single package. Confirmed against
     a production system. *(Case 10.)*
  6. **Constants are a separate emission path, emitting values rather than functions** — a
     locale-neutral type over per-locale data objects, fallback resolved at lowering. This is
     `tree<T>`'s *value* leaf branch, where messages are its *symbol* branch. *(Case 14.)*
  7. **Locale is an explicit parameter of the interpreter** — `render(data, args, locale)` is the
     conformance boundary. Nothing below it reads ambient culture; how a shim *above* it obtains
     the locale is free to differ per language. *(Case 2.)*
  8. **Per-locale data carries more than messages** — the pre-reduced plural rule table and the
     CLDR date/time patterns and number symbols are generated per locale alongside the message
     data, and dynamically imported as one unit. *(Cases 4 and 9.)*
  9. **Dispatch is declared, not inferred** — a sibling group is marked dispatchable, and
     signature-incompatible branches then fail the build by name rather than silently removing an
     API. *(Case 6.)*
  10. **A sole-literal message lowers to a bare string**, and a leaf's *kind* (message versus
     constant) is declared rather than inferred from arity. *(Case 1.)*
  11. **Every entry carries its source locale**, so a fallen-back message is distinguishable from
     a translated one — needed by case 4's build check, by `lang=`, and by dev-mode surfacing.
     *(Case 11.)*
  Still expected but unspecified: the `tree<T>` declaration itself for the namespace tree.

  **And one requirement that is not `c5n`'s.** The escape bug lives in the *emitter*, above the
  boundary vectors reach, so the harness needs a rung whose input is **DSL source** — generate,
  compile and execute in both languages — rather than message data. *(Case 8.)*

## Where the walk landed

**Every case is resolved, and no case forced new machinery mid-walk.** Three tightened the design
by *removing* something (nesting, the digit-coupling rule, the unreachable `other` fallback), and
the requirements list came out at eleven entries against a redesign of the whole codegen shape.

**But the banner stays on, because this doc's own pass criterion is not met.** It asks for every
case written out *in both languages and two locales*; that was done only where the artefacts were
doing work. Lifting the banner on a criterion recorded two paragraphs later as unmet would be
exactly the kind of thing this exercise exists to catch. What the walk earns is *"no case broke
it"*, which is weaker than *"validated"* and worth keeping distinct.

**Stated honestly, what the walk did not do:**

- **No real corpus.** Cases 7, 10, 12, 13 and 14 came from real messages or a production system;
  the rest ran on stand-ins. Stand-ins are invented, so they cannot contain the edge nobody
  thought of — which is the entire reason a corpus is worth more than a walk.
- **The artefacts were not written out longhand.** The method above asks for the typed signature,
  the message data and the runtime call, in both languages and two locales, for every case. In
  practice the cases were resolved by argument, with artefacts written only where they were doing
  work. That is a weaker discipline than the doc set for itself, and it is where a missed
  interaction would hide.
- **Two cases resolved on an argument that could be wrong** rather than on evidence: case 1's
  function-versus-property choice, and case 6's declared dispatch, which trades ergonomics for a
  loud failure. Both are cheap to reverse.

## What a pass looks like

Every case above written out in both languages and two locales, with no new machinery invented
mid-walk, and a `c5n` requirements list that is shorter than the redesign it replaces. Anything
less and the candidate is not settled — it is just unfalsified.

**A fail is a good outcome and should not be resisted.** The conventional shape
(translation-client / string-resolver) exists and works; the candidate's claim is that
designing *for* codegen from day one beats retrofitting it. That claim is worth testing
honestly, and the cost of finding out here is a document, where the cost of finding out later
is `c5n` grown to serve a shape that does not hold.

## Change log
- 2026-08-29: **the remaining seven cases walked — 1, 2, 4, 6, 8, 9, 11 — and every case is now
  resolved.** Walked under one lens, stated at the top: *the harness can only drive an input and
  compare an output, so anything unreachable from an input or invisible in an output is
  untestable — and untestable code inside the conformance surface is where two languages drift
  apart in silence.* It decided three cases outright. **Case 2 placed the conformance boundary
  for the whole design** — `render(data, args, locale)`, ambient locale forbidden below it and an
  ergonomics choice above it — which is also the first limit the emitter-takes-the-pain rule has
  been given by something other than the wire format, and it generalises the `LocalDate.ToString`
  incident into an invariant. **Case 9 rejected platform date formatting**: `Intl` and
  `CultureInfo` are two ICU copies at two versions, so delegating would turn a *runtime upgrade*
  into a red harness — and it fires the deferred CLDR-extractor trigger. **Case 8 found the one
  bug the vectors cannot reach**, since literal re-escaping happens in the emitter, above the
  boundary they drive; the interpreter gains no escape logic but the harness needs a
  source-level rung. **Case 4** split the plural rules the useful way — generated table,
  hand-written evaluator, pre-reduced at extraction thanks to case 3 — and turned `DESIGN.md`'s
  unreachable `other` fallback into a throw. **Case 11** resolves fallback at lowering, so every
  tree is complete and the interpreter has no fallback path, with each entry tagged by source
  locale — a tag `lang=` needs anyway (WCAG 3.1.2). **Case 6** changed shape under case 5 and
  landed on *declared dispatch, never inferred*, which is case 14's lesson a second time and now
  stated as a general rule: **structural facts are declared, never recovered by lookup**.
  **Case 1** found that arity is not the discriminator between a message and a constant.
  Requirements list is now eleven entries plus one harness rung; only the `tree<T>` declaration
  remains unspecified. Verdict and its honest limits recorded under *Where the walk landed*.
- 2026-08-29: **cases 10 and 12 resolved from production, and constants turn out to be bigger
  than case 14 treated them.** The two writers legitimately differ in **output layout** — TS
  per-locale and tree-shakable because bytes are the cost in a browser, C# a single package
  because a server's assembly loads once — which was the last plausible way this design could
  have turned out worse than it looks, and it didn't. The two-trees split decomposes it: static
  imports select the locale-neutral shims, dynamic import selects the per-locale data.
  **Approval gates the deploy, not the generation** — otherwise an unapproved string breaks a
  local build — and a **registration declaration** does three jobs at once: what to load, what
  the gate must check, and what the runtime permits. It stays a *runtime* check because the
  catalogue is shared across teams; a per-app catalogue could make it a compile error instead.
  And **constants are first-class addressable values on a separate emission path**, emitting as
  values rather than functions — which is `tree<T>`'s existing *value* leaf branch, with
  messages as its *symbol* branch. Requirements list is now six entries; only the `tree<T>`
  declaration itself remains unspecified.
- 2026-08-27: **case 5 resolved — nesting is removed, and the codegen shape gets simpler.** The
  first case to test the candidate's actual claim rather than its syntax, and the claim holds.
  The discriminator is that **`select` keys are locale-invariant while plural categories are
  not**, which maps straight onto the two trees: select belongs in the locale-neutral shim,
  plural in the per-locale data. So `select` becomes **shim-level dispatch on a generated
  enum** — an invalid value is a compile error rather than a silent fall-through to `other` —
  and the **parts model stays a flat list**, so the interpreter loops rather than recurses and
  the conformance surface does not grow. Translators gain whole sentences instead of the
  eighteen fragments a gender × Welsh-plural nest would produce, which is better practice on
  its own terms. Limits recorded: no `plural` inside `plural`, and a wide `select` is data
  rather than a message selector. *No real example exists — nesting has not been needed in
  accounting or banking work across several builds, and that absence is what made the question
  worth asking.*
- 2026-08-27: **cases 3 and 13 resolved together — plural drives on an integer.** The `-N` on a
  plural driver was declaring precision for CLDR's `v` operand, which is what forced the
  digit-coupling rule; restricting drivers to integers makes `v = 0` structurally, so the rule,
  the hint and case 13's unspecified `v` all disappear at once, and the interpreter drops from
  six operands to one. Recorded with the general test this suggests for the rest of the doc:
  **a restriction makes some inputs inexpressible, an approximation makes some inputs wrong** —
  today's rejected simplifications were all the latter, this is the former. Seam named:
  fractional plurals return with long-form unit names in prose, not with unit formatting, since
  symbols do not pluralise.
- 2026-08-27: **case 7 settled on bare numeric keys, and ordinals surfaced as the larger gap.**
  The deciding argument was a change of stance — *stay closer to CLDR, simplify only where we
  should* — under which `none` is the divergence and `=N` the alignment; bare numbers keep
  ICU's capability while dropping the `=1=` artifact and the second-`=` parse wrinkle. Case 7
  rewritten rather than accreting supersede-notes, since it had reversed twice. Added the
  principle in full with its three verified instances, the third being the sharpest: **Welsh's
  `zero` is `n = 0` for cardinals and `n = 0,7,8,9` for ordinals** — one word, one language,
  two tables. And recorded that **ordinal rules are absent from the design entirely**, where
  CLDR covers ~120–130 locales and English alone needs four ordinal categories.
- 2026-08-27: **the Latvian finding generalises — *borrow a category name and you inherit its
  grammar, not its number*.** Checked whether word-based exact matches extend past `none`
  (could there be a `two=`?). They cannot: `two` is a category in a dozen locales and is
  frequently not exactly 2 — **Breton** matches 22, 32, 42 but not 12, 72, 92; Scottish Gaelic
  matches 2 and 12; Slovenian 2, 102, 202. Six-plus languages against Latvian's one, so the
  failure is the same shape with a larger blast radius. `none` is safe only because CLDR never
  used the word; there is no second free one. Leaves an explicit either/or — words-only with
  zero as the sole exact match, or a visually distinct form for exact matches — turning on
  whether any real message has needed "exactly 2".
- 2026-08-27: **case 7 resolved the other way — word-based keys throughout, and my objection
  was answered by the pipeline.** `none` reserved alongside the CLDR categories; no numeric
  keys, no `=` sigil. The concern was translator legibility, not mechanism, and translators do
  not invent keys — the pipeline generates the target's set, so `none` and `zero` arrive as
  labelled branches. Better still, the front-end can fold them **per locale** from CLDR's own
  rules: where a locale's `zero` *is* `n = 0` (cy, ar, kw, …) `none` folds into it, and only
  `lv`/`prg` need both. The awkward case reduces to one living language. Cost recorded
  explicitly: exact matches other than zero are dropped, since `one` is taken by the category
  and no word remains for `=1`.
- 2026-08-27: **verified the plural facts from `plurals.xml`, and the counter-example got
  sharper.** Welsh does have six categories, so the claim used throughout stands. Ten locale
  codes carry a `zero` category and they split cleanly: eight define it as `n = 0`, while **`lv`
  and `prg` alone** define it as `n % 10 = 0 or n % 100 = 11..19 …`. Since Prussian is a
  revival, **Latvian is effectively the only living language where `zero` does not mean zero** —
  and **Welsh's `zero` is `n = 0`**, so the collapse this doc rejects would have passed a test
  against the obvious six-category locale and failed only in Latvian. Consequence for the
  fixtures: the locale set must name Latvian specifically, not just "a language with many
  categories".
- 2026-08-27: **Latvian settles the exact-match question, against the tidier answer.** The
  instinct that a uniform word list beats a mix leads to collapsing `none` into `zero` and
  dropping exact matches entirely — which fails, because **Latvian's `zero` category matches
  0, 10, 11…19, 20, 30**, so "no accounts" would render for twenty accounts. The separation is
  load-bearing, and it is why ICU has `=0` at all. What survives from the earlier finding:
  bare numbers rather than the `=` sigil, since a number cannot collide with a category name.
  Also noted: the same example carried `many=`, which is not an English category, making it the
  second real example in two to contain an unfireable branch.
  **Welsh's six categories confirmed** from `plurals.xml` — `zero` (n = 0), `one`, `two`, `few`
  (n = 3), `many` (n = 6), `other` — so the claim used throughout this doc stands; an earlier
  chart fetch had simply missed the row.
- 2026-08-27: **case 7 has a real example, and it argues the syntax should change.** Writing
  exact matches as bare numbers (`1=`) rather than ICU's `=1=` is unambiguous — categories are
  words, exact values are numbers — and it removes the second-`=` parse wrinkle `DESIGN.md`
  documents. `none=` should not be added: it duplicates `0`, and it crowds the one genuinely
  subtle corner of the syntax, where `zero` is a grammatical category rather than a test for
  zero. Also pinned: `1=` and `one` are different tests, which is not obvious and matters at
  `1.0`.
- 2026-08-27: **case 14 revised — the constant is a *default* for an optional parameter, not a
  binding of its own.** That corrects an overstatement: resolve-by-lookup fails loudly when a
  constant is removed (every call site stops compiling) and silently only when one is *added*
  that collides with a required parameter's name, changing an unrelated message's arity.
  Optionality should therefore be marked in the message rather than inferred. It also revises
  where the constant lands: not an inlined literal, since the caller may override, but a
  default carried in the per-locale message data — one new part kind, no runtime constants
  tree, and still nothing for `c5n` to know about.
- 2026-08-27: **first two real examples, and both found something.** A plural driver that is
  never displayed leaves the `v` operand unspecified, since the coupling rule assumes the
  pluralised value is also rendered — and a bare `plural` argument has no declared type. A link
  whose URL is a **locale-varying constant** rather than a parameter is in neither design doc:
  it must stay out of the typed signature, it needs explicit syntax rather than
  resolve-by-lookup, and it probably resolves at lowering time, which would keep it out of
  `c5n` and out of the interpreter entirely. Added a case for a mistyped plural category, which
  arrived by accident and is exactly the class of structural error the build should refuse.
- 2026-08-27: created, as the validation the `candidate` banner has been asking for since
  2026-07-04. Twelve cases derived from `DESIGN.md`'s own hard cases, with stand-ins so the
  walk can start before the real corpus is available. One finding recorded before any case was
  walked: **the candidate describes one tree where there are two** — a locale-neutral shim tree
  and per-locale data trees — which implies an unstated rule that signatures derive from the
  canonical message only.
