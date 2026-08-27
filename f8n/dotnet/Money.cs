using System.Globalization;
using System.Numerics;
using System.Text;

namespace F8n;

/// <summary>
/// An amount in a currency, held as an exact integer count of that currency's minor units.
/// </summary>
/// <remarks>
/// <para>
/// GBP 2 dp, JPY 0, BHD 3 — the scale is the currency's, never the caller's, so a Money is
/// always at the precision its currency actually has. Add and subtract are exact. The
/// precision-losing operations take a <see cref="RoundingMode"/> and land back on the
/// currency's dp; there is no hidden working precision, and anything needing more asks for it
/// explicitly. See f8n/DESIGN.md.
/// </para>
/// <para>
/// A bare number cannot say whether it is major units or minor, and the factor between them
/// is currency-dependent — so there is no constructor that takes one. <see cref="FromMajor"/>
/// and <see cref="FromMinor"/> name the unit at the call site.
/// </para>
/// </remarks>
public readonly struct Money : IEquatable<Money>, IComparable<Money>
{
    /// <summary>The currency, which supplies the scale.</summary>
    public Currency Currency { get; }

    /// <summary>The exact count of minor units — 1234 for GBP 12.34.</summary>
    public BigInteger Minor { get; }

    private Money(Currency currency, BigInteger minor)
    {
        Currency = currency;
        Minor = minor;
    }

    /// <summary>An amount given as a count of minor units — pence, cents, sen.</summary>
    public static Money FromMinor(BigInteger minor, Currency currency)
    {
        if (currency is null)
        {
            throw new ArgumentNullException(nameof(currency));
        }
        return new Money(currency, minor);
    }

    /// <summary>
    /// An amount in major units, in the canonical wire form: exactly the currency's decimal
    /// places, zeros included.
    /// </summary>
    /// <remarks>
    /// Strict, and deliberately not lenient: the same grammar serves the wire, so accepting a
    /// second spelling of one value would break the property that wire equality is value
    /// equality. Human input — grouping, symbols, a comma decimal separator — is l10n's
    /// boundary and must not share this parser.
    /// </remarks>
    public static Money FromMajor(string amount, Currency currency)
    {
        if (currency is null)
        {
            throw new ArgumentNullException(nameof(currency));
        }
        if (string.IsNullOrEmpty(amount))
        {
            throw new FormatException("an amount is required");
        }

        var index = 0;
        var negative = amount[0] == '-';
        if (negative)
        {
            index = 1;
        }

        var units = ReadUnits(amount, ref index, currency.MinorUnits);
        var fraction = ReadFraction(amount, ref index, currency.MinorUnits);
        if (index != amount.Length)
        {
            throw new FormatException($"\"{amount}\" has trailing characters");
        }

        var minor = (units * Pow10(currency.MinorUnits)) + fraction;
        if (negative && minor.IsZero)
        {
            // Money is an integer count, so -0 is not a second zero. Allowing it would give
            // one value two encodings and break wire equality.
            throw new FormatException("zero is unsigned: \"-0\" is not an amount");
        }
        return new Money(currency, negative ? -minor : minor);
    }

    /// <summary>Reads the integer part: "0", or a non-zero digit followed by digits.</summary>
    private static BigInteger ReadUnits(string amount, ref int index, int minorUnits)
    {
        var start = index;
        while (index < amount.Length && amount[index] >= '0' && amount[index] <= '9')
        {
            index++;
        }
        if (index == start)
        {
            throw new FormatException($"\"{amount}\" has no integer part (write \"0\", not \"\")");
        }
        var digits = amount.Substring(start, index - start);
        if (digits.Length > 1 && digits[0] == '0')
        {
            throw new FormatException($"\"{amount}\" has a leading zero, so the value has two spellings");
        }
        return BigInteger.Parse(digits, CultureInfo.InvariantCulture);
    }

    /// <summary>Reads the fractional part: exactly the currency's decimal places, or none at 0 dp.</summary>
    private static BigInteger ReadFraction(string amount, ref int index, int minorUnits)
    {
        if (minorUnits == 0)
        {
            if (index < amount.Length && amount[index] == '.')
            {
                throw new FormatException($"\"{amount}\" has decimals, but this currency has none");
            }
            return BigInteger.Zero;
        }

        if (index >= amount.Length || amount[index] != '.')
        {
            throw new FormatException($"\"{amount}\" is short: this currency is written to {minorUnits} decimal places, zeros included");
        }
        index++;

        var start = index;
        while (index < amount.Length && amount[index] >= '0' && amount[index] <= '9')
        {
            index++;
        }
        var digits = index - start;
        if (digits != minorUnits)
        {
            throw new FormatException($"\"{amount}\" has {digits} decimal place(s); this currency is written to exactly {minorUnits}");
        }
        return BigInteger.Parse(amount.Substring(start, digits), CultureInfo.InvariantCulture);
    }

    private static BigInteger Pow10(int exponent) => BigInteger.Pow(10, exponent);

    /// <summary>
    /// The canonical major-unit form — always exactly the currency's decimal places, so it
    /// round-trips through <see cref="FromMajor"/> and equal values have identical strings.
    /// </summary>
    public string Amount
    {
        get
        {
            var negative = Minor.Sign < 0;
            var magnitude = BigInteger.Abs(Minor);
            var scale = Pow10(Currency.MinorUnits);
            var units = magnitude / scale;
            var fraction = magnitude - (units * scale);

            var text = new StringBuilder();
            if (negative)
            {
                text.Append('-');
            }
            text.Append(units.ToString(CultureInfo.InvariantCulture));
            if (Currency.MinorUnits > 0)
            {
                text.Append('.');
                text.Append(fraction.ToString(CultureInfo.InvariantCulture).PadLeft(Currency.MinorUnits, '0'));
            }
            return text.ToString();
        }
    }

    /// <summary>Exact. Adding amounts in different currencies is a bug, not a conversion.</summary>
    public Money Add(Money other)
    {
        RequireSameCurrency(other, "add");
        return new Money(Currency, Minor + other.Minor);
    }

    /// <summary>Exact.</summary>
    public Money Subtract(Money other)
    {
        RequireSameCurrency(other, "subtract");
        return new Money(Currency, Minor - other.Minor);
    }

    /// <summary>Exact.</summary>
    public Money Negate() => new Money(Currency, -Minor);

    /// <summary>Exact: a whole number of minor units times a whole number stays whole.</summary>
    public Money Multiply(BigInteger factor) => new Money(Currency, Minor * factor);

    /// <summary>
    /// Applies a proportion, landing back on the currency's dp.
    /// </summary>
    /// <remarks>
    /// Integer multiply by the rational's numerator, then the shared divide-with-rounding by
    /// its denominator — the identical function a power-of-ten scale reaches, which is why an
    /// exact <see cref="Percentage"/> added no new conformance surface.
    /// </remarks>
    public Money Multiply(Percentage rate, RoundingMode mode)
    {
        var scaled = Minor * rate.Num;
        return new Money(Currency, Rounding.DivideWithRounding(scaled, rate.Den, mode));
    }

    /// <summary>
    /// Splits into <paramref name="parts"/> equal shares that sum back to this amount exactly.
    /// </summary>
    /// <remarks>
    /// A different discipline from <see cref="Divide"/>, which is why it is a different
    /// operation and takes an <see cref="AllocationRule"/> rather than a
    /// <see cref="RoundingMode"/>. Dividing £100 three ways gives 33.33 three times and loses
    /// a penny; this gives 33.34, 33.33, 33.33 and loses nothing.
    /// </remarks>
    public Money[] Allocate(int parts, AllocationRule rule)
    {
        if (parts < 1)
        {
            throw new ArgumentOutOfRangeException(nameof(parts), parts, "a partition has at least one part");
        }
        var weights = new BigInteger[parts];
        Array.Fill(weights, BigInteger.One);
        return AllocateByWeights(weights, rule);
    }

    /// <summary>
    /// Splits proportionally to <paramref name="weights"/>, summing back to this amount exactly.
    /// </summary>
    /// <remarks>
    /// Weights are whole numbers rather than proportions, which loses nothing: 60/40 and 3/2
    /// describe the same split, and integers keep every share exact without a common
    /// denominator to agree on. The only rounding anywhere is the deterministic hand-out of
    /// the leftover units.
    /// </remarks>
    public Money[] AllocateByWeights(IReadOnlyList<BigInteger> weights, AllocationRule rule)
    {
        var shares = Allocation.Distribute(Minor, weights, rule);
        var parts = new Money[shares.Length];
        for (var i = 0; i < shares.Length; i++)
        {
            parts[i] = new Money(Currency, shares[i]);
        }
        return parts;
    }

    /// <summary>Divides, landing back on the currency's dp. Does not conserve — see allocate.</summary>
    public Money Divide(BigInteger divisor, RoundingMode mode)
    {
        return new Money(Currency, Rounding.DivideWithRounding(Minor, divisor, mode));
    }

    private void RequireSameCurrency(Money other, string operation)
    {
        if (!ReferenceEquals(Currency, other.Currency) && Currency.Code != other.Currency.Code)
        {
            throw new InvalidOperationException(
                $"cannot {operation} {other.Currency.Code} and {Currency.Code}: convert one first, with a rate and a date");
        }
    }

    /// <summary>Order within one currency. Comparing across currencies is meaningless, so it throws.</summary>
    public int CompareTo(Money other)
    {
        RequireSameCurrency(other, "compare");
        return Minor.CompareTo(other.Minor);
    }

    public bool Equals(Money other)
    {
        return Currency.Code == other.Currency.Code && Minor == other.Minor;
    }

    public override bool Equals(object? obj) => obj is Money other && Equals(other);

    public override int GetHashCode() => HashCode.Combine(Currency.Code, Minor);

    /// <summary>The amount and its currency — for logs and debugging, never for display.</summary>
    public override string ToString() => Amount + " " + Currency.Code;
}
