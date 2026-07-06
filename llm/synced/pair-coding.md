# Pair-coding

Conventions for writing code *with* an LLM. Left to its defaults the model reaches for clever, compressed, "impressive" code; these rules pull it back toward code a human maintains happily. **Readability and explicitness over terseness — always.** Boring, obvious code is the goal, not a compromise.

## Versioning

- **Start semver at `1.0.0`, never `0.x`.** The `^` caret range treats `0.x` specially: `^0.1.0` allows `0.1.x` but *not* `0.2.0`, so every minor bump reads as breaking and consumers get pinned. Starting at `1.0.0` restores normal caret semantics and sidesteps the whole class of dependency-range surprises. (Explore pre-1.0 locally if you must; the *first published* version is `1.0.0`.)

## Naming

- **Descriptive by default** — `customerBalance`, not `cb`; `retryCount`, not `n`. The name carries the meaning.
- **Abbreviate only the universally-known** — `sb` (StringBuilder), `ms` (MemoryStream), `i`/`j` (loop indices), `id`. Conventions the reader already holds; everything else, spell out.

## Explicitness over terseness

- **Expand the syntax; never take an optional shortcut for brevity.** Always brace `if`/`for`/`while` bodies — `{ }` even on a one-line body. It kills the "added a second statement, forgot the braces" bug and keeps diffs clean.
- **No clever one-liners.** If a line needs decoding — a dense ternary chain, a nested comprehension, a fluent pipeline doing five things — break it into named steps. The compiler doesn't care; the reader does. Write for whoever debugs this at 2am.

## Tools

- **Prefer plain code over regex.** Reach for regex only when it's genuinely the right tool (real pattern-matching); for "starts with / contains / split on", use string methods — readable and debuggable. When regex *is* right, comment what it matches so it doesn't rot into a write-only incantation.

## Why

LLMs are trained on a lot of code that optimises for looking clever, so left alone they write terse, dense, one-line-wonders that demo well and maintain badly. These rules bias the other way — toward the explicit, obvious, boring code a team actually wants to own.
