import { Currency } from "./currency";

/**
 * ISO 3166 country identity: alpha-2, alpha-3 (identity), numeric code, official name,
 * E.164 calling code, default currency, and the capital city's civil timezone. Identity +
 * reference data; localised names / formatting live in l10n.
 *
 * Hand-written half. c5n emits the country table (Country.GBR, …) into a generated module
 * that constructs these instances, resolving defaultCurrency to a Currency reference
 * (Currency.GBP). See f8n/DESIGN.md.
 */
export class Country {
  constructor(
    /** ISO 3166 alpha-2 code (e.g. "GB"). */
    readonly alpha2: string,
    /** ISO 3166 alpha-3 code (e.g. "GBR") — the identity / table key. */
    readonly alpha3: string,
    /** ISO 3166 numeric code (e.g. 826). */
    readonly numeric: number,
    /** Official English name (invariant; localised names are l10n's). */
    readonly name: string,
    /** E.164 country calling code (e.g. 44 for GB). */
    readonly callingCode: number,
    /** Default currency (ISO 4217). */
    readonly defaultCurrency: Currency,
    /** IANA civil timezone id of the capital city (e.g. "Europe/London" for GB). Per-subdivision
     *  zones for multi-zone countries are deferred to subdivision data (see f8n/DESIGN.md). */
    readonly capitalTz: string,
  ) {}
}
