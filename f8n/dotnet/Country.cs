namespace F8n;

/// <summary>
/// ISO 3166 country identity: alpha-2, alpha-3 (identity), numeric code, English display name,
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

    /// <summary>
    /// Finds a country from a code in any of its forms — alpha-2, alpha-3, or numeric.
    /// </summary>
    /// <remarks>
    /// <para>
    /// An <em>ingestion</em> helper, and deliberately not what the wire uses. A country
    /// travels as its alpha-3 identity and nothing else, because accepting three forms there
    /// would give one value three encodings and break the rule that wire equality is value
    /// equality (f8n/DESIGN.md → Wire format). This exists for the other direction: a code
    /// arriving from a payment processor, an address form or a browser, where the form is
    /// whatever that system happens to use.
    /// </para>
    /// <para>
    /// Leniency here is safe in a way it usually is not, because the three forms occupy
    /// disjoint shapes — two letters, three letters, three digits — so nothing is guessed.
    /// </para>
    /// <para>
    /// Case is normalised with <c>ToUpperInvariant</c>, never <c>ToUpper</c>. Under a Turkish
    /// locale <c>"ie".ToUpper()</c> is <c>"İE"</c> — a dotted capital I — and Ireland would
    /// stop resolving on machines in one country. That trap is the strongest argument for
    /// this living here once rather than at every call site that has a code to look up.
    /// </para>
    /// </remarks>
    public static Country? Find(string code)
    {
        if (string.IsNullOrEmpty(code))
        {
            return null;
        }
        var normalised = code.ToUpperInvariant();

        if (normalised.Length == 3 && IsAllAsciiDigits(normalised))
        {
            return ByNumeric(int.Parse(normalised, System.Globalization.CultureInfo.InvariantCulture));
        }
        return normalised.Length switch
        {
            2 => ByAlpha2(normalised),
            3 => ByAlpha3(normalised),
            _ => null,
        };
    }

    /// <summary>
    /// Whether every character is an ASCII digit.
    /// </summary>
    /// <remarks>
    /// Not <c>char.IsDigit</c>, which is true for the digits of many scripts — the same trap
    /// <see cref="LocalDate"/> avoids. A numeric country code is ASCII or it is not one.
    /// </remarks>
    private static bool IsAllAsciiDigits(string text)
    {
        foreach (var c in text)
        {
            if (c < '0' || c > '9')
            {
                return false;
            }
        }
        return true;
    }
}
