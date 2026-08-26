using System.Numerics;

namespace F8n;

/// <summary>How a precision-losing operation resolves a value that falls between two results.</summary>
/// <remarks>
/// Caller-supplied at every precision-losing operation rather than defaulted per call site:
/// a jurisdiction can mandate a method, and a silently chosen one is the kind of
/// non-compliance nothing later detects. <c>allocate</c> takes no mode at all — it conserves
/// via an <c>AllocationRule</c> instead. See f8n/DESIGN.md.
/// </remarks>
public enum RoundingMode
{
    /// <summary>Banker's rounding: an exact half goes to the even result. The default elsewhere in f8n.</summary>
    HalfEven,

    /// <summary>An exact half goes away from zero — so -2.5 rounds to -3, not -2.</summary>
    HalfUp,
}

/// <summary>
/// The one function the cross-language conformance claim rests on.
/// </summary>
/// <remarks>
/// <para>
/// Everything else in f8n's arithmetic is exact integer work on big integers, which cannot
/// diverge between languages. Division is the single place a result must be *chosen*, so it
/// is the surface the golden vectors target — and it is written once, here, rather than
/// inlined at each call site.
/// </para>
/// <para>
/// <c>Money × Percentage</c> reaches it with the rational's denominator in place of a power
/// of ten, which is why making <c>Percentage</c> exact added no new conformance surface.
/// </para>
/// </remarks>
public static class Rounding
{
    /// <summary>
    /// Divides and resolves the remainder by the given mode. Exact when the division is exact.
    /// </summary>
    public static BigInteger DivideWithRounding(BigInteger numerator, BigInteger denominator, RoundingMode mode)
    {
        if (denominator.IsZero)
        {
            throw new DivideByZeroException("a denominator of zero has no quotient");
        }
        // Normalise the sign onto the numerator, so the true quotient's sign is the
        // numerator's and the comparisons below need only one case.
        if (denominator.Sign < 0)
        {
            numerator = -numerator;
            denominator = -denominator;
        }

        // Both languages truncate toward zero on integer division, so the truncated quotient
        // and the remainder's sign are the same in each without any adjustment.
        var truncated = numerator / denominator;
        var remainder = numerator - (truncated * denominator);
        if (remainder.IsZero)
        {
            return truncated;
        }

        // Compare twice the remainder against the divisor rather than halving anything:
        // halving would be the one place a division could reintroduce the problem.
        var twiceRemainder = BigInteger.Abs(remainder) * 2;
        var awayFromZero = numerator.Sign < 0 ? truncated - 1 : truncated + 1;

        if (twiceRemainder > denominator)
        {
            return awayFromZero;
        }
        if (twiceRemainder < denominator)
        {
            return truncated;
        }

        // Exactly half.
        return mode switch
        {
            RoundingMode.HalfUp => awayFromZero,
            // To even: keep the truncated quotient when it is already even, else step away
            // from zero. Sign-symmetric, like HalfUp — one answer for negatives library-wide.
            RoundingMode.HalfEven => truncated.IsEven ? truncated : awayFromZero,
            _ => throw new ArgumentOutOfRangeException(nameof(mode), mode, "unknown rounding mode"),
        };
    }
}
