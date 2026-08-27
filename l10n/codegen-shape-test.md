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

## Cases

Ordered roughly by how much they can break. `⚠` marks the ones I expect to be load-bearing.

### 1. No arguments
**Stand-in:** `Save`
**Real:** *(to add)*
**Forces:** the baseline. What does a zero-arg shim cost, and is the data a bare literal or
still a parts list? If a zero-arg message pays for a parts list and an interpreter call, most
of a real corpus pays it.

### 2. One value hint
**Stand-in:** `{limit:currency} per institution`
**Real:** *(to add)*
**Forces:** the basic typed signature — `limit: Money`. Who applies the locale: does the shim
capture it, take it per call, or read ambient state?

### 3. ⚠ Plural with coupled digits
**Stand-in:** `{count:plural|one={count:number-0} item|other={count:number-0} items}`
**Real:** *(to add)*
**Forces:** one argument in two roles — selector operand *and* formatted value. Plus the rule
that "a single digit-count governs both". **Is that rule checked, or assumed?** A message
whose branches disagree (`number-0` in one, `number-2` in another) is authorable today; if
nothing rejects it, `1.00 item` ships.

### 4. ⚠ A target locale with more categories
**Stand-in:** the case above, in `cy` — six categories where `en` has two.
**Real:** *(to add — and which locales are actually in scope, with their category sets)*
**Forces:** the two-tree split above, concretely. Signature must be identical across locales;
data must not be. Also: the runtime falls back to `other` — where does the *category set* come
from, generated per locale or from CLDR plural rules at runtime?

### 5. ⚠ Nested selectors
**Stand-in:** `{gender:select|male={count:plural|one=…|other=…}|female={count:plural|one=…|other=…}}`
**Real:** *(to add — does the corpus actually nest, and how deep?)*
**Forces:** recursion in the data model, and whether the interpreter stays small. This is the
case that decides whether "message as data" is genuinely data or a small program. If the
interpreter needs its own scope handling, the conformance surface is bigger than claimed.

### 6. Branches using different arguments
**Stand-in:** `{plan:select|free=Upgrade for {price:currency}|paid=Renews {renews:date-medium}}`
**Real:** *(to add)*
**Forces:** the signature is the **union** of all branches' arguments, so some are unused on
any given call. Are they optional, or required-and-ignored? Nullable in C#?

### 7. ⚠ Exact-value match — **REAL**, and the syntax should change
**Real:** `{count:plural|none=no accounts|1=account|other=accounts}` — how it would be written
where zero is reachable.
**Forces:** the ordering rule (exact before category, a conformance rule either way), whether
an exact match is a distinct part kind or a category with a funny name — and two syntax
findings, one adopting and one rejecting.

> **Superseded below.** The bare-number recommendation was overturned once the translation
> pipeline was brought into the argument. Kept for the reasoning, which still holds on its own
> terms; the resolution is *Word-based keys throughout*, further down.

**Adopt the bare number: `1=`, not `=1=`.** `DESIGN.md` uses ICU's `=0` sigil and pays for it
with a documented parse wrinkle — *"a key starting with `=` splits on the second `="`*. But
category names are alphabetic (`zero one two few many other`) and exact values are numeric, so
**a bare number cannot collide with a category** and the sigil buys nothing. Drop it and the
uniform split-on-first-`=` rule covers every key. The whole thing reduces to one line a
translator can hold: **numbers are exact, words are categories.**

**Reject `none=`, and reject the tempting collapse behind it.** The instinct that a uniform
word list reads better than a mix is right, and it leads somewhere worse: if `zero` and `none`
are both words, why not drop `none` and let **`zero` mean "n = 0" everywhere** — a real category
where the locale has one, an exact match where it does not? It would remove exact matches from
the syntax altogether. It is also **wrong, and dangerously so.**

**Latvian's `zero` category is `n % 10 = 0 or n % 100 = 11..19 or v = 2 and f % 100 = 11..19`**
— it matches 0, 10, 11…19, 20, 30, 40. A message using `zero` to mean "none" renders **"no
accounts" for twenty accounts** in Latvian.

Verified against `common/supplemental/plurals.xml`. Ten locale codes carry a `zero` category,
and they split two ways:

| condition | locales |
|---|---|
| `n = 0` | `cy`, `ar`, `ars`, `kw`, `cv`, `ksh`, `blo`, `lag` |
| `n % 10 = 0 or n % 100 = 11..19 or v = 2 and f % 100 = 11..19` | **`lv`, `prg`** |

**And this is the part worth keeping.** Welsh's `zero` *is* `n = 0` — so the collapse would have
worked perfectly in Welsh, the language anyone would reach for as the six-category test case.
It fails only in Latvian (Prussian being a revival with negligible use). **Validated against the
obvious representative locale, the bug ships.**

That is the *uniform correctness or uniform wrongness* problem one layer up from where this
design usually worries about it, and it is directly actionable: **the fixture locale set has to
name Latvian specifically**, not merely "a language with many categories".

So the separation of exact matches from categories is load-bearing rather than inherited — and
this is *why ICU has `=0` at all*, which reads as ceremony until you meet a language where
`zero` does not mean zero. It also settles the aesthetics: **the visual mix is doing semantic
work.** A number can never be mistaken for a category, and in a uniform word list `none` and
`zero` would look like the same kind of key while denoting very different sets — with both
present in Latvian and nothing to tell a translator which is arithmetic and which grammatical.

**Pin the related subtlety while it is in view: `1=` and `one` are not the same test.** Exact
`1` matches n = 1 only; category `one` follows CLDR's rules, so `1.0` with a visible decimal is
`other` in English. Both are legitimately needed, and the numeric/word split is what makes that
legible rather than folkloric.

**Resolution — word-based keys throughout, and the pipeline is what makes it safe.**

The objection to `none` was never mechanical: it works exactly as `=0` does. It was that a
Latvian translator seeing `none` and `zero` together has no cue which is arithmetic and which
grammatical. **But translators do not invent keys — the pipeline generates them** (`DESIGN.md`:
the target's categories are generated, en writes `one`/`other`, cy expands to six). A Latvian
translator receives both as pre-created branches with whatever the translator UI labels them,
rather than as raw text to reason about. The ambiguity is a tooling problem, and the tooling is
already in the design.

It also resolves further than "both always appear", because the front-end can compute the
difference **from CLDR itself** — it knows each locale's `zero` condition:

| locale group | what the translator gets |
|---|---|
| `cy`, `ar`, `ars`, `kw`, `cv`, `ksh`, `blo`, `lag` — `zero` **is** `n = 0` | `none` folds into `zero`; one branch, nothing to disambiguate |
| `lv`, `prg` — `zero` is **not** `n = 0` | both, and both are genuinely needed |
| everything else — no `zero` category | `none` only |

Each locale's generated data carries the branches that locale actually needs, which is what
per-locale generation is for. The awkward case reduces to **one living language**, whose
translator is the person best placed to understand it.

So: **word-based keys, `none` reserved, no numeric keys and no `=` sigil.** The key space is
`{CLDR categories} ∪ {none}` — a closed set the front-end validates against, which also yields
the case 13a build errors (`ome=`, `many=`) for free.

**The cost, stated rather than allowed to happen quietly: this drops exact matches other than
zero.** ICU's `=1` and `=2` have no word available — `one` is taken by the category, and they
are not the same test (`1.0` is exact-`1` but category-`other`). Likely an acceptable loss:
"exactly two" special-casing is rare, and a caller can select a different message. But it is a
decision to take deliberately, not a side effect of preferring words.

**And it does not weaken the Latvian finding — it relocates it.** The collapse that fails is
still the one that makes `zero` *mean* "none". Keeping them as separate keys, with the
per-locale folding above, is precisely what the counter-example demands.

**And it reinforces two other cases.** It never displays `count` at all, so case 13's `v`
operand is undefined for the whole message — the exact branches do not rescue that, since
`other` is still selected by a plural rule that needs `v`. And as first written it carried
`many=`, which **is not an English category** (en has only `one` and `other`), so the branch can
never fire. That is case 13a again, arriving by accident for the second time in two real
examples — reasonable evidence that an inapplicable category key should fail the build rather
than sit unreachable.

### 8. Escapes
**Stand-in:** `Use \{braces\} and a pipe \| here`
**Real:** *(to add — do real messages hit these in anger?)*
**Forces:** whether escapes survive lowering to data, and whether the data holds the *resolved*
literal or the escaped source. `DESIGN.md` calls this "the archetypal demo-passes-prod-breaks
bug", so the answer belongs in fixtures, not prose.

### 9. A temporal hint
**Stand-in:** `Renews on {renews:date-medium}`
**Real:** *(to add)*
**Forces:** binding to `f8n`'s temporal type, and **who formats** — the interpreter calling a
formatter, or a pre-formatted string from the caller. If the caller formats, the typed
signature is weaker than claimed; if the interpreter does, it needs CLDR date patterns, which
is a second data tree.

### 10. ⚠ Deep namespace and tree-shaking
**Stand-in:** `account.open.button.label`, alongside a sibling subtree that is never used.
**Real:** *(to add — real namespace depth and breadth)*
**Forces:** `tree<T>` emitting nested scopes, and the open question `DESIGN.md` already flags:
C# nested static classes give typed access but **may not trim**, where TS modules plus
per-message files do. If the two languages need different output layouts, `c5n`'s per-target
writers have to differ structurally rather than in spelling.

### 11. A message a locale does not have
**Stand-in:** a key present in `en-GB`, absent in `cy`.
**Real:** *(to add)*
**Forces:** the shim exists (it comes from canonical), so what does the data tree hold? A
fallback entry, a hole the runtime resolves, or a build failure? Interacts with case 4's
fallback question.

### 12. An unapproved translation
**Stand-in:** a `cy` entry with `approved: false`.
**Real:** *(to add)*
**Forces:** `DESIGN.md`'s pipeline blocks unapproved entries at the CD gate. Does *generation*
read only approved entries — and if so, is an unapproved locale a hole (case 11) or the
previous approved value? This is where the pipeline and the codegen meet, and neither doc says.

### 13. ⚠ A value hint and a selector that is never displayed — **REAL**
**Real:** `You have {balance:currency} in {count:plural|one=account|other=accounts}`
**Forces:** three things, and the middle one is a gap in the design.

- Two arguments of different kinds in one message — one value hint, one selector — so the
  signature draws from two hint families. Expected to be fine; confirm.
- **`count` selects but is never rendered.** `DESIGN.md`'s rule is that *"the format hint's
  digit count **is** the plural's `v` operand"* — which assumes the pluralised value is also
  displayed. Here it is not, so **there is no digit count and `v` is unspecified.** CLDR plural
  selection needs it. `v = 0` is the sane default, but it is a *decision the design does not
  state*, and it is a conformance surface: both languages must choose identically or `1.0`
  categorises differently in each. Rarely visible in English; visible in Welsh and Arabic.
- **What type is a bare `plural` argument?** The hint table maps `number-N` to
  `FixedDecimal`/number, but `plural` alone maps to nothing. Undecided.

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

---

## Structural questions, not per-message

- **Two trees, one or two `c5n` collection kinds?** Does the shim tree and the per-locale data
  tree both fall out of `tree<T>`, or does one need something else?
- **Where does locale fallback live** — `en-GB` → `en` → root? Generated (output multiplies) or
  runtime (a lookup chain)?
- **Is a constants file a `c5n` concern at all?** If constants resolve at lowering time
  (case 14) then `c5n` never sees them and the answer is no — which is the preferred outcome,
  and worth confirming rather than assuming.
- **What does `c5n` need that it does not have?** The deliverable of this exercise. Expect at
  minimum `tree<T>`, typed-shim emit, and fine-grained output layout — but the *specification*
  of each is what is missing, and it should come from these cases rather than from the stress
  test that proposed them.

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
