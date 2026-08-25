using System.Globalization;
using System.Numerics;
using System.Text;

namespace F8n;

/// <summary>
/// A dimensionless proportion, held as an exact fraction over big integers.
/// </summary>
/// <remarks>
/// <para>
/// The stored value is the <em>proportion</em>, never the percent number: a rate printed as
/// "17.5%" is held as 7/40. "%" is a constructor and a formatter — presentation belongs to
/// l10n — so the two never meet in the value. See f8n/DESIGN.md.
/// </para>
/// <para>
/// Canonical form: gcd-reduced, <c>Den &gt; 0</c>, sign carried on <c>Num</c>, zero as 0/1,
/// integers keeping /1. Equal values therefore have identical components, so equality is a
/// component compare and the canonical string is unique per value.
/// </para>
/// <para>
/// Nothing here touches a binary float. A proportion is exact by construction and stays that
/// way; the whole reason for the Rational substrate is that a decimal materialises 1/3 and
/// loses it.
/// </para>
/// </remarks>
public readonly struct Percentage : IEquatable<Percentage>
{
    /// <summary>Numerator, carrying the sign.</summary>
    public BigInteger Num { get; }

    /// <summary>Denominator, always greater than zero.</summary>
    public BigInteger Den { get; }

    private Percentage(BigInteger num, BigInteger den)
    {
        Num = num;
        Den = den;
    }

    /// <summary>Reduces to canonical form: gcd-reduced, positive denominator, zero as 0/1.</summary>
    private static Percentage Canonical(BigInteger num, BigInteger den)
    {
        if (den.IsZero)
        {
            throw new FormatException("a denominator of zero is not a proportion");
        }
        if (den.Sign < 0)
        {
            num = -num;
            den = -den;
        }
        if (num.IsZero)
        {
            return new Percentage(BigInteger.Zero, BigInteger.One);
        }
        var divisor = BigInteger.GreatestCommonDivisor(BigInteger.Abs(num), den);
        return new Percentage(num / divisor, den / divisor);
    }

    /// <summary>
    /// The value as authored — 0.175 is the proportion 7/40, which presents as 17.5%.
    /// </summary>
    public static Percentage FromProportion(string value)
    {
        var (num, den) = ParseDecimal(value);
        return Canonical(num, den);
    }

    /// <summary>
    /// The percent number as a document states it — 17.5 is 7/40. Dividing by a hundred in
    /// rational space is exact, so nothing is lost by authoring data the readable way.
    /// </summary>
    public static Percentage FromPercent(string value)
    {
        var (num, den) = ParseDecimal(value);
        return Canonical(num, den * 100);
    }

    /// <summary>
    /// Convenience for hand-written C#, where a decimal literal is already exact.
    /// </summary>
    /// <remarks>
    /// Routed through the string form deliberately: one parsing implementation means the
    /// literal path and the generated path cannot drift apart, and a decimal's invariant
    /// text is its exact digits, so the trip costs nothing.
    /// </remarks>
    public static Percentage FromProportion(decimal value)
    {
        return FromProportion(value.ToString(CultureInfo.InvariantCulture));
    }

    /// <inheritdoc cref="FromProportion(decimal)"/>
    public static Percentage FromPercent(decimal value)
    {
        return FromPercent(value.ToString(CultureInfo.InvariantCulture));
    }

    /// <summary>
    /// Reads the canonical wire form "num/den". Strict: the input must already be canonical,
    /// so that a value's encoding is unique and wire equality stays string equality. A
    /// reducible or negatively-denominated fraction is an error, not something to tidy up.
    /// </summary>
    public static Percentage Parse(string canonical)
    {
        if (canonical is null)
        {
            throw new FormatException("expected a canonical proportion, got null");
        }
        var slash = canonical.IndexOf('/');
        if (slash < 0)
        {
            throw new FormatException($"expected \"num/den\", got \"{canonical}\"");
        }

        var num = ParseInteger(canonical.Substring(0, slash), allowSign: true);
        var den = ParseInteger(canonical.Substring(slash + 1), allowSign: false);
        if (den.IsZero)
        {
            throw new FormatException("a denominator of zero is not a proportion");
        }

        var reduced = Canonical(num, den);
        if (reduced.Num != num || reduced.Den != den)
        {
            throw new FormatException($"\"{canonical}\" is not canonical — expected \"{reduced}\"");
        }
        return reduced;
    }

    /// <summary>
    /// Parses a plain decimal into an exact fraction: the digits become the numerator and the
    /// fraction length chooses the power of ten beneath it.
    /// </summary>
    /// <remarks>
    /// Written as a character walk rather than a regex on purpose. Two regex engines can
    /// differ in dialect on the edges, and this grammar has to mean exactly the same thing in
    /// every target language — a loop over characters is identical by construction.
    /// Deliberately rejected: exponent form (targets spell it differently), a leading plus
    /// (one representation only), grouping separators (presentation, so l10n's), and
    /// whitespace.
    /// </remarks>
    private static (BigInteger Num, BigInteger Den) ParseDecimal(string value)
    {
        if (string.IsNullOrEmpty(value))
        {
            throw new FormatException("expected a decimal value, got an empty string");
        }

        var index = 0;
        var negative = false;
        if (value[0] == '-')
        {
            negative = true;
            index = 1;
        }

        // Integer part: at least one digit, and no leading zero unless it is exactly "0" —
        // so a value has one spelling rather than several that all parse.
        var integerStart = index;
        while (index < value.Length && value[index] >= '0' && value[index] <= '9')
        {
            index++;
        }
        var integerDigits = index - integerStart;
        if (integerDigits == 0)
        {
            throw new FormatException($"\"{value}\" has no digit before the decimal point");
        }
        if (integerDigits > 1 && value[integerStart] == '0')
        {
            throw new FormatException($"\"{value}\" has a leading zero");
        }

        var digits = new StringBuilder(value.Substring(integerStart, integerDigits));
        var fractionDigits = 0;

        if (index < value.Length && value[index] == '.')
        {
            index++;
            var fractionStart = index;
            while (index < value.Length && value[index] >= '0' && value[index] <= '9')
            {
                index++;
            }
            fractionDigits = index - fractionStart;
            if (fractionDigits == 0)
            {
                throw new FormatException($"\"{value}\" ends in a decimal point");
            }
            digits.Append(value, fractionStart, fractionDigits);
        }

        if (index != value.Length)
        {
            throw new FormatException($"\"{value}\" is not a plain decimal — unexpected '{value[index]}'");
        }

        var num = BigInteger.Parse(digits.ToString(), CultureInfo.InvariantCulture);
        if (negative)
        {
            num = -num;
        }
        return (num, BigInteger.Pow(10, fractionDigits));
    }

    /// <summary>Reads an integer with no leading zeros, so each value has one spelling.</summary>
    private static BigInteger ParseInteger(string text, bool allowSign)
    {
        if (text.Length == 0)
        {
            throw new FormatException("expected an integer, got an empty string");
        }

        var start = 0;
        var negative = false;
        if (allowSign && text[0] == '-')
        {
            negative = true;
            start = 1;
        }
        if (start >= text.Length)
        {
            throw new FormatException($"\"{text}\" has no digits");
        }
        for (int i = start; i < text.Length; i++)
        {
            if (text[i] < '0' || text[i] > '9')
            {
                throw new FormatException($"\"{text}\" is not an integer");
            }
        }
        if (text.Length - start > 1 && text[start] == '0')
        {
            throw new FormatException($"\"{text}\" has a leading zero");
        }

        var value = BigInteger.Parse(text.Substring(start), CultureInfo.InvariantCulture);
        return negative ? -value : value;
    }

    /// <summary>The canonical wire form, "num/den".</summary>
    public override string ToString()
    {
        return Num.ToString(CultureInfo.InvariantCulture) + "/" + Den.ToString(CultureInfo.InvariantCulture);
    }

    public bool Equals(Percentage other)
    {
        return Num == other.Num && Den == other.Den;
    }

    public override bool Equals(object? obj)
    {
        return obj is Percentage other && Equals(other);
    }

    public override int GetHashCode()
    {
        return HashCode.Combine(Num, Den);
    }

    public static bool operator ==(Percentage left, Percentage right) => left.Equals(right);

    public static bool operator !=(Percentage left, Percentage right) => !left.Equals(right);
}
