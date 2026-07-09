namespace F8n;

/// <summary>
/// ISO 4217 currency identity: alpha code, numeric code, official name, and minor-unit
/// exponent. Identity + reference data only — locale-specific presentation (symbol,
/// localised name) lives in l10n; money math lives on Money.
/// </summary>
/// <remarks>
/// Hand-written half of the type. c5n emits the currency table (Currency.GBP, …) into a
/// generated partial that instantiates this ctor. See f8n/DESIGN.md.
/// </remarks>
public partial class Currency
{
    /// <summary>ISO 4217 alpha code (e.g. "GBP") — the identity / table key.</summary>
    public string Code { get; }

    /// <summary>ISO 4217 numeric code (e.g. 826 for GBP).</summary>
    public int Numeric { get; }

    /// <summary>Official English name (invariant; localised display names are l10n's).</summary>
    public string Name { get; }

    /// <summary>Default/canonical currency symbol (e.g. "£"). Locale-specific presentation
    /// (US$ vs $, symbol placement, narrow vs full form) is l10n's.</summary>
    public string Symbol { get; }

    /// <summary>Minor-unit exponent — decimal places in the smallest unit (GBP 2, JPY 0, BHD 3).</summary>
    public int MinorUnits { get; }

    public Currency(string code, int numeric, string name, string symbol, int minorUnits)
    {
        Code = code;
        Numeric = numeric;
        Name = name;
        Symbol = symbol;
        MinorUnits = minorUnits;
    }
}
