import { Currency } from "./currency.js";

/**
 * ISO 3166 country identity: alpha-2, alpha-3 (identity), numeric code, English display name,
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
    readonly primaryTz: string,
  ) {}

  /**
   * The wire form: the identity alone. A reference type travels as its key and never as an
   * inlined record — one rule for every table row (f8n/DESIGN.md -> Reference types travel as
   * their key). Reading one back needs the generated index, which imports this module, so
   * `fromJson` is a free function in the lookup module rather than a static here.
   */
  toJSON(): string {
    return this.alpha3;
  }
}
