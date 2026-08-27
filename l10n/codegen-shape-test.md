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

### 7. Exact-value match
**Stand-in:** `{count:plural|=0=No items|one=1 item|other={count:number-0} items}`
**Real:** *(to add)*
**Forces:** `=0` is checked *before* category rules, and the parse splits on the second `=`.
Is an exact match a distinct part kind in the data, or a category with a funny name? The
interpreter's ordering is a conformance rule either way.

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
