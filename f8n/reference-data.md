# f8n — reference data: what comes from where

Which upstream supplies each field `f8n` publishes, under what licence, and what was ruled
out. Written 2026-08-27, after the CLDR-only sourcing decision (`DESIGN.md` → banner) turned
out to be *nearly* right — two of `Country`'s fields cannot come from CLDR at all, and finding
that out is most of what this document is for.

> **Status: research, not implementation.** No extractor exists yet. The committed tables are
> still the hand-authored skeleton. This is here so the next person to open the pipeline does
> not repeat the search — and so the two corrections below are not rediscovered the hard way.

## Source per field

| Field | Source | Notes |
|---|---|---|
| `Currency.code`, `.numeric`, `.name` | **CLDR** | `name` is the CLDR *English display* name, which Unicode notes may differ from the ISO 4217 name |
| `Currency.minorUnits` | **CLDR** `currencyData/fractions` → `digits` | The *accounting* scale |
| `Currency.symbol` | **CLDR** | Canonical symbol only; locale-specific presentation is `l10n`'s |
| `Country.alpha2`, `.alpha3`, `.numeric`, `.name` | **CLDR** | |
| `Country.defaultCurrency` | **CLDR** `currencyData/region` | Carries date ranges, so "legal tender *now*" is a query, not a lookup |
| `Country.callingCode` | **libphonenumber** | **Not CLDR** — see below |
| `Country.primaryTz` | **IANA tzdb** `zone1970.tab` | **Not CLDR**, and not the capital's — see below |

## Correction 1 — CLDR has no telephone codes

`telephoneCodeData` was **deprecated in CLDR v34 and the data removed**, explicitly because
numbering plans change faster than CLDR's release cadence. The spec points at
**libphonenumber** (Google, Apache-2.0).

So `Country.callingCode` needs a third upstream. It stays as a field — the values are correct
and the data is useful — but the pipeline must know it comes from somewhere else, and the
Apache-2.0 notice obligation arrives with it.

*Worth noting when `PhoneNumber` is built:* that primitive will need libphonenumber-derived
data anyway, so the dependency is one f8n is taking on regardless, not one `Country` drags in
alone. If it were being designed from scratch today the calling code would arguably sit with
`PhoneNumber` rather than with `Country`; it is not worth moving now.

## Correction 2 — `capitalTz` was the wrong name, not the wrong plan

CLDR has no "capital's timezone" field. `primaryZones` exists in the DTD but the LDML spec
does not document it, so its coverage is unknown and it is not something to build on.

**IANA's tz database is the right source** — `zone1970.tab`, keyed by ISO 3166 alpha-2, and
**public domain**, so no licence question at all. But its ordering rule is stated and it is not
the capital's:

> "makes some geographical sense, and (2) puts the most populous timezones first, where that
> does not contradict (1)"

The first-listed zone for a country is therefore the **most populous**, which is a different
definition from *the capital's* — and, for a civil-time default, a better one.

`DESIGN.md` already had the right words: *"a single well-defined representative zone"*. That
describes IANA's data exactly. "The capital's" was a gloss laid over it, and the gloss is what
was wrong, so the field is renamed **`primaryTz`** rather than the source being changed. The
committed values are unchanged — every country in the skeleton set has one zone, or (for the
US) a first-listed zone that is also the capital's.

The alternative was keeping "capital" and hand-curating the list, which is exactly what the
pipeline exists to stop.

## Further CLDR data worth taking

Ranked by fit. None of it is built; this is the shortlist a pipeline should be shaped to reach.

1. **`cashDigits` / `cashRounding`** (`currencyData`) — closes the cash-rounding Room. CHF is
   2 dp for accounting and rounds to 0.05 in cash, and `f8n` currently cannot say so.
2. **`weekData`** — `firstDay`, `weekendStart`, `weekendEnd`, `minDays` per territory.
   Invariant territory reference data, exactly `f8n`-shaped, and what any date or calendar
   surface needs. `l10n` will want it.
3. **`measurementData`** — default measurement system and paper size per territory. Same
   shape, and directly useful to **ampersand**, which renders kWh, m² and °C and otherwise
   ends up hardcoding a territory test somewhere it should not be.
4. **`territoryContainment`** — the UN M.49 tree (World → Continent → Subcontinent → Country).
   Useful for grouping, and interesting for a second reason: it is a genuine **`tree<T>`**
   consumer inside `f8n`, which would exercise `c5n`'s one unbuilt collection kind without
   waiting for `l10n`.
5. **`likelySubtags`** — arrives with `Locale`, not before.

**Deliberately not:** `territoryInfo` (population, GDP, literacy). Real data, wrong library —
that is world-facts rather than domain primitives, and belongs in `doppel` if anywhere.

## Licences

| Source | Licence | Obligation |
|---|---|---|
| CLDR | Unicode License V3 | Reproduce the copyright and permission notice |
| IANA tzdb | **Public domain** | None |
| libphonenumber | Apache-2.0 | NOTICE and attribution |

All three are compatible with an MIT `f8n`. The attribution mechanism already handles this
per-source: `c5n`'s `attribution:` matches a source path pattern and emits the notice into the
header of anything generated from it, so a file derived from CLDR carries the Unicode notice
and one derived from IANA carries nothing — which is correct, and is why the rule is
path-matched rather than project-wide.

## Confidence

The **licence terms** and the **deprecations** are verified from source. The **element and
attribute names** are from the LDML spec's prose rather than the DTD or the data files, so
treat them as unconfirmed until an extractor reads them — which is cheap, because a wrong path
fails loudly on the first run rather than quietly producing wrong data.

## Change log
- 2026-08-27: created. Two corrections to the CLDR-only sourcing decision — telephone codes
  were removed from CLDR in v34 (libphonenumber instead), and there is no capital's-timezone
  field anywhere, so `capitalTz` is renamed `primaryTz` and sourced from IANA's public-domain
  tz database, whose first-listed zone is the most populous rather than the capital's. Plus a
  shortlist of further CLDR data that fits, of which `weekData` and `measurementData` look
  strongest and `territoryContainment` would give `tree<T>` its first consumer.
