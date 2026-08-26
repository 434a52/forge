import type { Country } from "./country.js";
import type { Percentage } from "./percentage.js";
import type { TaxCategory } from "./generated/taxcategory.data.js";
import type { TaxType } from "./generated/taxtype.data.js";

/**
 * One rate of one tax, in one jurisdiction, for one category of supply.
 *
 * Identity and value only. It carries no dates: a rate that changes is an `EffectiveDated`
 * of these, so the thing that varies over time is the series and not the rate — which is
 * what lets the invariant half hoist to `common:` in the data file and be written once.
 *
 * The rate is a `Percentage`, an exact proportion. "20%" is one presentation of 1/5 and
 * belongs to l10n; nothing here ever holds the percent number.
 */
export class TaxRate {
  constructor(
    /** The country whose tax this is. */
    readonly jurisdiction: Country,
    /** Which tax — VAT, and others as jurisdictions are added. */
    readonly taxType: TaxType,
    /** Which band of supply the rate applies to. */
    readonly category: TaxCategory,
    /** The rate, as an exact proportion — 1/5, not "20%". */
    readonly rate: Percentage,
  ) {}
}
