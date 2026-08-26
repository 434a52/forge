namespace F8n;

/// <summary>
/// One rate of one tax, in one jurisdiction, for one category of supply.
/// </summary>
/// <remarks>
/// <para>
/// Identity and value only. It carries no dates: a rate that changes is an
/// <see cref="EffectiveDated{T}"/> of these, so the thing that varies over time is the
/// series and not the rate — which is what lets the invariant half hoist to <c>common:</c>
/// in the data file and be written once.
/// </para>
/// <para>
/// The rate is a <see cref="Percentage"/>, an exact proportion. "20%" is one presentation of
/// 1/5 and belongs to l10n; nothing here ever holds the percent number.
/// </para>
/// </remarks>
public class TaxRate
{
    /// <summary>The country whose tax this is.</summary>
    public Country Jurisdiction { get; }

    /// <summary>Which tax — VAT, and others as jurisdictions are added.</summary>
    public TaxType TaxType { get; }

    /// <summary>Which band of supply the rate applies to.</summary>
    public TaxCategory Category { get; }

    /// <summary>The rate, as an exact proportion — 1/5, not "20%".</summary>
    public Percentage Rate { get; }

    public TaxRate(Country jurisdiction, TaxType taxType, TaxCategory category, Percentage rate)
    {
        Jurisdiction = jurisdiction;
        TaxType = taxType;
        Category = category;
        Rate = rate;
    }
}
