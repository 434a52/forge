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
| `Country.defaultCurrency` | **CLDR** `currencyData/region` | Carries `from`/`to`, so "current" is a query rather than a lookup — and `tender="true\|false"`, which has to be filtered on or non-tender currencies come through |
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
4. **`territoryContainment`** — the UN M.49 tree (World → Continent → Subcontinent → Country),
   plus three non-geographic groupings (`EU`, `EZ`, `UN`). Useful for display grouping and
   filtering, and interesting for a second reason: it is a genuine **`tree<T>`** consumer
   inside `f8n`, which would exercise `c5n`'s one unbuilt collection kind without waiting for
   `l10n`. **Not useful for tax, for two independent reasons — see below.**
5. **`likelySubtags`** — arrives with `Locale`, not before.

**Deliberately not:** `territoryInfo` (population, GDP, literacy). Real data, wrong library —
that is world-facts rather than domain primitives, and belongs in `doppel` if anywhere.

### Containment is geographic, never political

Worth knowing before anyone reaches for it to answer a sovereignty question, because it looks
like it should and does not. The UK's overseas territories are placed by **where they are**:

| Territory | M.49 placement |
|---|---|
| Bermuda | Americas → Northern America |
| Cayman Islands, Turks & Caicos | Americas → Caribbean |
| Falkland Islands | Americas → South America |
| Gibraltar | Europe → Southern Europe |
| Pitcairn | Oceania → Polynesia |
| British Indian Ocean Territory | **Africa → Eastern Africa** |

GB itself is World [001] → Europe [150] → Northern Europe [154]. The Crown Dependencies —
Jersey, Guernsey, Isle of Man — sit **beside** GB in Northern Europe rather than inside it,
which is constitutionally correct and means "the British Isles" is not expressible either.

**There is no sovereignty edge in the data.** Containment answers *where is it*, never *who
governs it*, and assembling "the UK and its territories" from it means hand-curating a list —
the thing the pipeline exists to stop.

### `EU` and `EZ` are not tax data, for two independent reasons

An earlier reading of this shortlist promoted `territoryContainment` on the grounds that EU
membership is a VAT fact and therefore squarely `f8n`'s domain. **That was wrong twice over,
and both corrections are worth keeping.**

**1. The group carries no dates.** Verified from the DTD: `group` declares
`type`, `contains`, `grouping`, `status (deprecated | grouping)`, `draft` and `references` —
and **no `from`/`to`**. Compare the `currency` element under `region`, which declares
`from`, `to`, `tender`, `digits`, `rounding` and `cashRounding`. So currency-by-territory is a
dated relation and containment is a **current-state snapshot**. It cannot answer "was GB in the
EU in 2019", which is precisely the shape of question effective-dated tax data asks.

**2. The EU VAT territory is not EU membership.** Even a dated membership list would be the
wrong data. The *fiscal* territory differs from the political one by a list of exclusions —
Åland (FI), the Canary Islands, Ceuta and Melilla (ES), the French overseas departments,
Heligoland and Büsingen (DE) — and at least one inclusion, Monaco being treated as France for
fiscal purposes. A rule keyed on "is a member state" would be wrong for every one of them.

**And `f8n` does not need it.** `TaxRate` is keyed by jurisdiction and carried in an
`EffectiveDated` series, so the rate series *is* the answer to "what applied there, then".
Membership would only matter for cross-border machinery — reverse charge, OSS — which is far
outside the primitive layer. So `EU`/`EZ` stay what they honestly are: **display and grouping
metadata**, ranked accordingly.

### One bonus from reading the DTD

The `currency` element under `region` carries **`tender="true|false"`**, distinguishing legal
tender from a currency merely in circulation. That is the correct filter for
`Country.defaultCurrency`, and without it the extractor would silently pick up non-tender
entries. `cashRounding` appears **both** there and on `fractions/info`, so the extractor needs
to establish which one it should be reading rather than assuming.

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

The **licence terms** and the **deprecations** are verified from source. The **containment
placements** above are read from CLDR's own published containment chart. The **`group` and
`currency` attribute lists** are read from `ldmlSupplemental.dtd` and are verified.

Everything else — `weekData`'s and `measurementData`'s exact fields, the `fractions/info`
structure — is from the LDML spec's prose rather than the DTD or the data files, so treat it as
unconfirmed until an extractor reads it. That is cheap: a wrong path fails loudly on the first
run rather than quietly producing wrong data.

## Change log
- 2026-08-27: **containment researched properly, and an over-promotion corrected.** M.49 is
  **geographic, not political** — the UK's overseas territories scatter across four continents
  (BIOT is filed under *Eastern Africa*) and the Crown Dependencies sit beside GB rather than
  inside it, so there is no sovereignty edge to query. The `EU`/`EZ` groupings were briefly
  promoted here as tax-relevant and that was wrong twice: `group` carries **no date
  attributes** (verified against the DTD, where `currency` under `region` *does* carry
  `from`/`to`), so it cannot answer a historical membership question; and the **EU VAT
  territory is not EU membership** anyway, differing by Åland, the Canaries, Ceuta and
  Melilla, the French overseas departments, Heligoland, Büsingen, and Monaco-as-France. `f8n`
  needs none of it — `TaxRate` is jurisdiction-keyed and effective-dated, so the series is the
  answer. Bonus from the DTD: `currency` carries **`tender="true|false"`**, which is the
  correct filter for `defaultCurrency` and would otherwise have been missed.
- 2026-08-27: created. Two corrections to the CLDR-only sourcing decision — telephone codes
  were removed from CLDR in v34 (libphonenumber instead), and there is no capital's-timezone
  field anywhere, so `capitalTz` is renamed `primaryTz` and sourced from IANA's public-domain
  tz database, whose first-listed zone is the most populous rather than the capital's. Plus a
  shortlist of further CLDR data that fits, of which `weekData` and `measurementData` look
  strongest and `territoryContainment` would give `tree<T>` its first consumer.
