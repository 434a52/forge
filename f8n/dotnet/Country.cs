namespace F8n;

/// <summary>
/// ISO 3166 country identity: alpha-2, alpha-3 (identity), numeric code, official name,
/// E.164 calling code, default currency, and the capital city's civil timezone. Identity +
/// reference data; localised names / formatting live in l10n.
/// </summary>
/// <remarks>
/// Hand-written half. c5n emits the country table (Country.GBR, …) into a generated partial
/// that instantiates this ctor, resolving defaultCurrency to a Currency table reference
/// (Currency.GBP). See f8n/DESIGN.md.
/// </remarks>
public partial class Country
{
    /// <summary>ISO 3166 alpha-2 code (e.g. "GB").</summary>
    public string Alpha2 { get; }

    /// <summary>ISO 3166 alpha-3 code (e.g. "GBR") — the identity / table key.</summary>
    public string Alpha3 { get; }

    /// <summary>ISO 3166 numeric code (e.g. 826).</summary>
    public int Numeric { get; }

    /// <summary>Official English name (invariant; localised names are l10n's).</summary>
    public string Name { get; }

    /// <summary>E.164 country calling code (e.g. 44 for GB).</summary>
    public int CallingCode { get; }

    /// <summary>Default currency (ISO 4217) for the country.</summary>
    public Currency DefaultCurrency { get; }

    /// <summary>IANA civil timezone id of the capital city (e.g. "Europe/London" for GB).
    /// A single well-defined representative zone — per-subdivision zones for multi-zone
    /// countries (US, …) are deferred to subdivision data (see f8n/DESIGN.md).</summary>
    public string CapitalTz { get; }

    public Country(
        string alpha2, string alpha3, int numeric, string name,
        int callingCode, Currency defaultCurrency, string capitalTz)
    {
        Alpha2 = alpha2;
        Alpha3 = alpha3;
        Numeric = numeric;
        Name = name;
        CallingCode = callingCode;
        DefaultCurrency = defaultCurrency;
        CapitalTz = capitalTz;
    }
}
