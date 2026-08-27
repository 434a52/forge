/**
 * ISO 4217 currency identity: alpha code, numeric code, English display name, minor-unit
 * exponent. Identity + reference data only — symbol / localised name live in l10n,
 * money math lives on Money.
 *
 * Hand-written half of the type. c5n emits the currency table (Currency.GBP, …) into a
 * generated module that constructs these instances. See f8n/DESIGN.md.
 */
export class Currency {
  constructor(
    /** ISO 4217 alpha code (e.g. "GBP") — the identity / table key. */
    readonly code: string,
    /** ISO 4217 numeric code (e.g. 826 for GBP). */
    readonly numeric: number,
    /** CLDR English display name (invariant; localised names are l10n's, from the same source). Not the ISO 4217 name, which CLDR notes may differ. */
    readonly name: string,
    /** Default/canonical currency symbol (e.g. "£"). Locale-specific presentation is l10n's. */
    readonly symbol: string,
    /** Minor-unit exponent — decimal places in the smallest unit (GBP 2, JPY 0, BHD 3). */
    readonly minorUnits: number,
  ) {}

  /**
   * The wire form: the identity alone. A reference type travels as its key and never as an
   * inlined record — one rule for every table row (f8n/DESIGN.md -> Reference types travel as
   * their key). Reading one back needs the generated index, which imports this module, so
   * `fromJson` is a free function in the lookup module rather than a static here.
   */
  toJSON(): string {
    return this.code;
  }
}
