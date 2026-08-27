using System.Numerics;

namespace F8n;

/// <summary>
/// How the units left over by an exact partition are handed out.
/// </summary>
/// <remarks>
/// <para>
/// Parameterised and with <b>no default</b>, unlike <see cref="RoundingMode"/>. A jurisdiction
/// or a contract can mandate an apportionment method, and a silently chosen one is
/// non-compliance that nothing later detects — so the caller must say which they mean.
/// </para>
/// <para>
/// A closed set of built-ins, deliberately. A caller-supplied distributor would move the
/// "both languages identical" guarantee into the caller's own two callbacks, which f8n cannot
/// test and therefore should not promise. A genuinely new mandated method arrives here as a
/// named built-in with its own vectors.
/// </para>
/// </remarks>
public readonly struct AllocationRule
{
    internal enum Method
    {
        LargestRemainder,
        Sequential,
        Designated,
    }

    internal Method Rule { get; }

    internal int Part { get; }

    private AllocationRule(Method rule, int part)
    {
        Rule = rule;
        Part = part;
    }

    /// <summary>
    /// Hamilton: leftover units go to the largest fractional remainders, ties by ascending
    /// index. The usual choice where no method is mandated.
    /// </summary>
    public static AllocationRule LargestRemainder => new(Method.LargestRemainder, 0);

    /// <summary>The first parts absorb the leftover, one unit each.</summary>
    public static AllocationRule Sequential => new(Method.Sequential, 0);

    /// <summary>One nominated part — a designated ledger line — absorbs all of the residual.</summary>
    public static AllocationRule Designated(int part)
    {
        if (part < 0)
        {
            throw new ArgumentOutOfRangeException(nameof(part), part, "a part index is not negative");
        }
        return new AllocationRule(Method.Designated, part);
    }
}

/// <summary>
/// A conserving partition of an integer quantity — the second conformance surface, after
/// <see cref="Rounding.DivideWithRounding"/>.
/// </summary>
/// <remarks>
/// <para>
/// <b>Not rounding.</b> Rounding each part independently does not conserve: £100 split three
/// ways at 2 dp gives 33.33 three times, and a penny disappears. This takes the exact share of
/// each part, keeps the whole units, and then hands out the units left over — so the parts sum
/// to the whole exactly, by construction rather than by checking.
/// </para>
/// <para>
/// <b>Sign is handled by working on the magnitude.</b> The design says the shares are
/// "floored", which is right for a non-negative total and wrong for a negative one: flooring
/// -1.5 gives -2 where negating the floor of 1.5 gives -1, and the two disagree. Since
/// <c>allocate(-m) == -allocate(m)</c> is a stated requirement, the sign is lifted out first
/// and reapplied at the end, which makes the symmetry true by construction instead of a
/// property that has to hold.
/// </para>
/// </remarks>
public static class Allocation
{
    /// <summary>
    /// Splits <paramref name="total"/> into parts proportional to <paramref name="weights"/>,
    /// conserving exactly.
    /// </summary>
    public static BigInteger[] Distribute(BigInteger total, IReadOnlyList<BigInteger> weights, AllocationRule rule)
    {
        if (weights is null || weights.Count == 0)
        {
            throw new ArgumentException("a partition needs at least one part", nameof(weights));
        }

        var weightTotal = BigInteger.Zero;
        foreach (var weight in weights)
        {
            if (weight.Sign < 0)
            {
                throw new ArgumentException("a weight is not negative", nameof(weights));
            }
            weightTotal += weight;
        }
        if (weightTotal.IsZero)
        {
            throw new ArgumentException("the weights are all zero, so there is nothing to be proportional to", nameof(weights));
        }
        if (rule.Rule == AllocationRule.Method.Designated && rule.Part >= weights.Count)
        {
            throw new ArgumentOutOfRangeException(nameof(rule), rule.Part, $"there is no part {rule.Part} in a partition of {weights.Count}");
        }

        var negative = total.Sign < 0;
        var magnitude = BigInteger.Abs(total);

        // Exact share of part i is magnitude * wᵢ / weightTotal. Both operands are
        // non-negative here, so integer division is a floor and the remainder is the
        // fractional part's numerator — which is what the largest-remainder rule ranks on.
        var parts = new BigInteger[weights.Count];
        var remainders = new BigInteger[weights.Count];
        var allocated = BigInteger.Zero;
        for (var i = 0; i < weights.Count; i++)
        {
            var scaled = magnitude * weights[i];
            parts[i] = scaled / weightTotal;
            remainders[i] = scaled - (parts[i] * weightTotal);
            allocated += parts[i];
        }

        var leftover = magnitude - allocated;
        foreach (var i in Recipients(leftover, remainders, rule))
        {
            parts[i] += BigInteger.One;
        }

        if (negative)
        {
            for (var i = 0; i < parts.Length; i++)
            {
                parts[i] = -parts[i];
            }
        }
        return parts;
    }

    /// <summary>Which parts receive one of the leftover units, in the rule's order.</summary>
    private static IEnumerable<int> Recipients(BigInteger leftover, BigInteger[] remainders, AllocationRule rule)
    {
        var count = (int)leftover;
        switch (rule.Rule)
        {
            case AllocationRule.Method.Designated:
                // One line absorbs all of it, so this yields the same index repeatedly.
                for (var n = 0; n < count; n++)
                {
                    yield return rule.Part;
                }
                break;

            case AllocationRule.Method.Sequential:
                for (var i = 0; i < count; i++)
                {
                    yield return i;
                }
                break;

            default:
                // Hamilton. Ordering by remainder descending with ties by ascending index is
                // what makes this deterministic — and the tie-break is not a detail: an equal
                // split gives every part the same remainder, so ties are the normal case
                // rather than the edge one, and without a stated order the two languages could
                // disagree on every three-way split of an odd amount.
                var order = new int[remainders.Length];
                for (var i = 0; i < order.Length; i++)
                {
                    order[i] = i;
                }
                Array.Sort(order, (left, right) =>
                {
                    var byRemainder = remainders[right].CompareTo(remainders[left]);
                    return byRemainder != 0 ? byRemainder : left.CompareTo(right);
                });
                for (var i = 0; i < count; i++)
                {
                    yield return order[i];
                }
                break;
        }
    }
}
