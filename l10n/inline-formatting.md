# l10n — inline formatting

Simple inline formatting — **bold, italic, code, links** — inside l10n message strings. Extends the interpolation model in `DESIGN.md` (→ *Interpolation & hints*, *Codegen structure*, *Translation pipeline*). Design-stage: the shape is settled; two edges (flanking, new-tab policy) are flagged under **Open decisions**.

**Thesis:** inline markup is **four more node types in the parts-as-data model + one method per render sink** — *not* a markdown library. The formatting rides the interpolation pipeline that already exists; nothing new is parsed at runtime, and the cross-language conformance surface does not grow.

## Why not a markdown library

Reaching for `markdown-it` (TS) / `Markdig` (C#) fights four of l10n's own north stars:

- **Conformance.** Two libraries = **twin parsers that diverge** on edge cases — the exact divergence l10n and `c5n` exist to design out.
- **Wrong layer.** The DSL is parsed to data *at build time*; the runtime never re-parses (see `DESIGN.md` → *Codegen structure*). A runtime markdown lib is the wrong layer *and* the wrong count (two of them).
- **Composition.** The markup must parse *together* with `{arg:hint}` and the `\{ \} \|` escaping as **one grammar**. A standalone lib knows nothing of the interpolation syntax — you'd be refereeing two parsers' overlap.
- **Target + safety.** Markdown libs emit an HTML *string* → `v-html` of (partly machine-translated, partly user-interpolated) content. We want **structured parts, rendered per target**.

The fix is to extend the existing message front-end with a tiny markup subset, lowered into the **same recursive parts model** the interpolation already uses. `c5n` stays domain-blind; l10n owns the front-end (the established pattern).

## Syntax — a markdown subset

Markdown because devs and translators already know it, it's terse, and `[text](url)` → `<a>` is also what the **email** target needs. Bold/italic/code are trivial; links carry the design weight (below).

```yaml
# en-GB · account namespace   (bare {name} = plain string; or {name:text} if hints stay mandatory)
account:
  welcome:  "Welcome back, **{name}**."                        # bold around an interpolation
  status:   "Your account is *read-only* until verified."
  apiHint:  "Send the `Authorization` header with each call."
  terms:    "Please read the [terms and conditions]({termsUrl})."   # external link — URL is a param
  invoice:  "Download [invoice {number}]({invoiceUrl})."       # link text contains an interpolation
  consent:  "{businessName} agrees to the [T&C]({termsUrl})."  # user input + link in one string; & auto-escapes
  basket:   "{count:plural|one=You have **1** item|other=You have **{count:number-0}** items}."
  literal:  "Prices marked \\* include VAT."                   # escaped * → literal asterisk
```

- **The link `ref` is a param, never a baked URL** — `[text]({ref})`. Keeps URLs out of the translation files (translators never touch them; refs may be dynamic). Typed like any `{arg:hint}`.
- **Nesting falls out of the recursive model** — `**{count} items**` just works; markers join the existing escape set.

## Rendering — one walk, many sinks

Inline formatting forces `render` to generalise from `parts → string` to **`parts → T` via a target sink**: the UI target needs vnodes (or safe HTML), not a string you'd `v-html` blindly. The **walk stays single** — that's what keeps the conformance surface singular; two `renderToX` functions would be two walks that drift.

```ts
// Generated: the parsed message as data (language-neutral).
type Part =
  | { kind: "text";   value: string }
  | { kind: "arg";    name: string; hint: Hint }            // {amount:currency}
  | { kind: "bold";   children: Part[] }
  | { kind: "italic"; children: Part[] }
  | { kind: "code";   children: Part[] }
  | { kind: "link";   ref: string; children: Part[] }       // [text]({ref})
  | { kind: "select"; arg: string; mode: "plural" | "select"; branches: Record<string, Part[]> };

// One per target. T = what the target builds (string, HTML string, VNode[], PDF runs…).
interface Renderer<T> {
  text(value: string): T;
  value(formatted: string): T;      // an interpolated value, already formatted
  bold(children: T[]): T;
  italic(children: T[]): T;
  code(children: T[]): T;
  link(ref: string, children: T[]): T;
  join(parts: T[]): T;
}

// Hand-written, one per language. THE conformance surface: walk order,
// formatter calls, and plural selection must be identical in C# and TS.
function render<T>(parts: Part[], args: Args, locale: Locale, out: Renderer<T>): T {
  return out.join(parts.map((p) => renderPart(p, args, locale, out)));
}

function renderPart<T>(part: Part, args: Args, locale: Locale, out: Renderer<T>): T {
  switch (part.kind) {
    case "text":   { return out.text(part.value); }
    case "arg":    { return out.value(formatValue(args[part.name], part.hint, locale)); }
    case "bold":   { return out.bold(part.children.map((c) => renderPart(c, args, locale, out))); }
    case "italic": { return out.italic(part.children.map((c) => renderPart(c, args, locale, out))); }
    case "code":   { return out.code(part.children.map((c) => renderPart(c, args, locale, out))); }
    case "link":   { return out.link(part.ref, part.children.map((c) => renderPart(c, args, locale, out))); }
    case "select": {
      const branch = part.branches[selectBranch(part, args, locale)] ?? part.branches["other"];
      return render(branch, args, locale, out);
    }
  }
}
```

`formatValue` / `selectBranch` are unchanged from the current design (the formatter machinery + `PluralCategory`).

## Sinks

```ts
// Plain string — the golden-vector target (and logs).
const plainText: Renderer<string> = {
  text: (v) => v, value: (v) => v,
  bold: (c) => c.join(""), italic: (c) => c.join(""), code: (c) => c.join(""),
  link: (ref, c) => c.join(""),           // or `${c.join("")} (${ref})`
  join: (parts) => parts.join(""),
};

// Safe HTML string — UI (v-html) AND email, conformance-vectorable.
const htmlSafe: Renderer<string> = {
  text: (v) => escapeHtml(v), value: (v) => escapeHtml(v),
  bold: (c) => `<strong>${c.join("")}</strong>`,
  italic: (c) => `<em>${c.join("")}</em>`,
  code: (c) => `<code>${c.join("")}</code>`,
  link: (ref, c) => {
    const external = /^https?:/i.test(ref);
    // In-string links are external by construction (internal routes = the dev's <router-link>).
    // Same-tab external nav unloads the document → the SPA's in-memory state is gone; so: new tab.
    const attrs = external ? ` target="_blank" rel="noopener noreferrer"` : "";
    return `<a href="${safeHref(ref)}"${attrs}>${c.join("")}</a>`;
  },
  join: (parts) => parts.join(""),
};

// Vue vnodes — only for messages needing live components (see Links).
import { h, type VNode } from "vue";
const vue: Renderer<VNode | string> = {
  text: (v) => v, value: (v) => v,
  bold: (c) => h("strong", c), italic: (c) => h("em", c), code: (c) => h("code", c),
  link: (ref, c) => h(resolveLink(ref), c),
  join: (parts) => parts,
};
```

## Safety — construct, don't sanitise

The reason this is *small* code, not scary code: the hard problem is **sanitising arbitrary untrusted HTML** (that's DOMPurify, thousands of lines). We never do that — we **construct** HTML from a closed, known set of parts, so arbitrary HTML never exists to clean. Construct-safe is a handful of lines; sanitise-untrusted is not.

- **`v-html` of a by-construction-safe string has the same injection surface as vnodes.** `v-html`'s danger is *untrusted* strings; a string built with escaped text, a closed tag set, and a validated href carries no vector. So the `htmlSafe` sink is as safe as the vnode sink — and nicer FE DX, plus it serves email (which cannot take vnodes) and is conformance-vectorable. Prefer it as the default; keep `vue` for the component case.
- **Own the escaper — do not borrow Vue's.** Vue client-side doesn't string-escape at all (it uses `textContent`); its SSR `escapeHtml` (`@vue/shared`) is TS-only and internal. Conformance needs the escaper **identical in C# and TS**, so it must be *ours*, spec'd and golden-vectored — the same reason we own ISO parsing rather than trust platform parsers to agree (don't trust `WebUtility.HtmlEncode` and Vue's `escapeHtml` to match).
- **One escaper covers everything** — the 5-char set (`& < > " '`) is safe for both text content *and* double-quoted attributes. This holds **only because the construction set is closed**: a fixed handful of tags, `href` the only attribute, always double-quoted — never an unquoted attribute, `<script>`, `<style>`, or event-handler context. That invariant is load-bearing; state it so nobody later adds a `style="{x}"` and silently breaks it.

### Interpolated values are leaves — never markup

The invariant that makes **user data and a link safe in the *same* string**: structure comes only from the template (parsed once, at build time); an interpolated value is always an escaped leaf, **never re-scanned for markup at runtime**. So the two untrusted things in

```yaml
consent: "{businessName} agrees to the [T&C]({termsUrl})."
```

are covered by one rule, not two special cases:

- **`{businessName}` (user input) cannot inject or reach the link.** It renders via `out.value → escapeHtml`; a value of `<script> [Evil](javascript:…) Ltd` is inert text — its `[` `<` `{` are never structure, because markup is recognised in the *template* only. User data physically can't become a link, break the link, or open a tag.
- **The link is author-controlled**, its `ref` scheme-allowlisted + attribute-escaped (`safeHref`).

`htmlSafe` → `{escaped businessName} agrees to the <a href="{safeHref(termsUrl)}" rel="…">T&amp;C</a>.`

**Highest-risk slot: user input in the `ref` position** (`[text]({userUrl})`) — defended by the scheme allowlist, but that check is then the *only* guard, and open-redirect / phishing validation is beyond `safeHref`. User input as link *text* or plain interpolation is a low-risk leaf; keep `ref`s dev-controlled where you can.

### The `<a>` concentrates the care

Bold/italic/code are trivial (fixed tag, escaped children). The anchor holds the whole security surface — and it's still ~12 lines:

```ts
const SAFE_SCHEMES = new Set(["http:", "https:", "mailto:", "tel:"]);

function safeHref(ref: string): string {
  const hasScheme = /^[a-z][a-z0-9+.-]*:/i.test(ref);
  if (hasScheme) {
    const scheme = ref.slice(0, ref.indexOf(":") + 1).toLowerCase();
    if (!SAFE_SCHEMES.has(scheme)) {
      throw new Error(`l10n: refusing unsafe link scheme in "${ref}"`);   // kills javascript:/data:
    }
  }
  return escapeHtml(ref);   // attribute-escape so the URL can't break out of the quotes
}
```

Three things to get right — **scheme allowlist, attribute-escape the href, `rel` on external** — none hard, but this is where the golden vectors point hardest. Note vnodes do **not** save you here: a `javascript:` href executes regardless of vnode-vs-`v-html`, because the danger is the URL scheme, not HTML parsing. The scheme check is yours in *every* sink.

## Links — in-string vs the dev's `<router-link>`

The fork resolves cleanly by who owns the link:

- **External / static links → in the string.** `[text]({url})` → safe `<a href>` via `htmlSafe`. Works in email, conformance-vectored. The common mid-sentence link case. **Opens in a new tab** (`_blank` + `rel`) — same-tab nav would unload the document and dump the SPA's in-memory state; a rare same-tab external link is a dev-owned `<a>`, like internal routes.
- **Internal / routed / live-component links → the FE dev's `<router-link>`, never a route in the string:**
  - **Whole-string link** ("Go to dashboard" is the entire label): the dev wraps it — `<router-link :to="…">{{ t('nav.dashboard') }}</router-link>`. **Zero l10n machinery.** Most internal links.
  - **Fragment mid-sentence** ("visit your [dashboard] to continue"): the string exposes a **named slot/placeholder**, and the dev fills it with `<router-link>` — component interpolation, à la vue-i18n's `<i18n-t>`. Needs the `vue` (vnode) sink. **The edge case.**

Routes and components never enter the translation files, either way.

## Conformance boundary

- **Golden vectors run a string sink** (`plainText` or `htmlSafe`) and assert **C# render ≡ TS render**. That pins the walk, the formatter calls, plural selection, *and the escaper* — the things that must not diverge.
- **The `vue` sink and the slot/component path are UI-only** (no C# / no email equivalent), so — like anything Vue-platform — they sit **outside** the cross-language vectors and are covered by component tests. Rich targets never enlarge the conformance surface; the walk does the load-bearing work once.

## Translation pipeline

A refinement of the existing extract-placeholders step (`DESIGN.md` → *Translation pipeline*), not a new mechanism:

- **Markers and link `ref`s are structural** — protect them like `{arg:hint}` placeholders; the LLM never rewrites `**`, `` ` ``, `[](…)`, or a URL.
- **The wrapped *text* is translatable** — unlike opaque `{arg}` placeholders, the prose inside `**bold**` / `[link text]` must be translated. So: protect the markers + refs, translate the text between them.

## Open decisions

- **Flanking vs always-escape for `*` / `` ` `` / `[`.** Treat them as markup only in a valid pair (`**x**`, `[x](y)`), lone ones literal, `\*` to force. Markdown-style *flanking rules* (whitespace-flanked `*` isn't emphasis — "2 * 3" just works) vs *always-escape* (simpler grammar, noisier strings). **Lean flanking.**
- **New-tab affordance (`opens in new tab` cue).** *Resolved:* in-string external links default to `target="_blank"` + `rel="noopener noreferrer"` — same-tab nav unloads the document and dumps the SPA's in-memory state (real loss mid-consent-flow; bfcache is routinely disabled by authed-SPA behaviours, so it can't be relied on). No override syntax — a rare same-tab external link is a dev-owned `<a>`, mirroring the internal-link fork. *Still open:* the a11y cue itself — sink-emitted (visually-hidden span / icon, static ⇒ vectorable) vs FE-owned via `a[target="_blank"]` CSS. **Lean: FE owns the icon, sink owns the attributes.**
- **Record the closed-set invariant** (one escaper is only safe while the tag/attribute set stays closed and href stays double-quoted) as a standing constraint on the sinks.

## Change log

- 2026-07-20: **resolved the new-tab open decision** — in-string external links now default to `target="_blank"` + `rel="noopener noreferrer"`. Rationale: same-tab nav unloads the document and dumps the SPA's in-memory state (real loss mid-flow; bfcache can't be relied on — authed SPAs routinely disable it). No override syntax — rare same-tab external links are dev-owned `<a>`, mirroring the internal-link fork. Updated the `htmlSafe` `link` sink. Remaining sub-decision: the "opens in new tab" a11y cue (**lean: FE owns the icon via `a[target=_blank]`, sink owns the attributes**).
- 2026-07-20: added the **user-input-beside-a-link** worked example (`{businessName} agrees to the [T&C]({termsUrl})`) + made the load-bearing invariant explicit — *structure comes from the template only; interpolated values are escaped leaves, never re-scanned for markup at runtime*, so user data can't inject or reach a link in the same string. Flagged the `ref` position as the highest-risk slot. Noted the **consent / T&C link** as the forcing case for the new-tab open decision (**lean: in-string external links → `_blank` + mandatory `rel`**).
- 2026-07-08: created. Inline formatting = markdown subset (`**bold**` / `*italic*` / `` `code` `` / `[text]({ref})`) lowered into the existing recursive parts model — **not** a markdown library (rejected: twin-parser divergence, wrong layer, must compose with `{arg:hint}`, unsafe HTML-string output). Render generalises `parts → string` to `parts → T` via a per-target **sink** (`Renderer<T>`), keeping the walk single = one conformance surface. Sinks: `plainText` (vectors), `htmlSafe` (UI via `v-html` + email, vectorable), `vue` (vnodes, component cases). **Safety = construct-not-sanitise**: `v-html` of a by-construction-safe string ≡ vnode safety; own a tiny spec'd/vectored escaper (5-char set + href scheme allowlist), don't borrow Vue's (TS-only, breaks C#≡TS); the `<a>` holds the care (scheme allowlist + attribute-escape + `rel`). **Links**: external → in-string `<a href>`; internal → the dev's `<router-link>` (whole-string wrap = common, fragment slot = edge). Conformance vectors run the string sink; vnode/slot paths are UI-only, outside the vectors. Translation pipeline protects markers/refs, translates wrapped text.
