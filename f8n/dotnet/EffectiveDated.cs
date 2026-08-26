namespace F8n;

/// <summary>
/// Entry point for building an <see cref="EffectiveDated{T}"/> with the value type inferred.
/// </summary>
/// <remarks>
/// A non-generic class holding a generic method, so generated code writes
/// <c>EffectiveDated.Of((date, value), …)</c> and never names the value type twice.
/// </remarks>
public static class EffectiveDated
{
    /// <summary>Builds a series from its entries, in any order.</summary>
    public static EffectiveDated<T> Of<T>(params (LocalDate From, T Value)[] entries)
    {
        return new EffectiveDated<T>(entries);
    }
}

/// <summary>
/// A value that changes on known dates — a tax rate, a threshold, a band — and the lookup
/// that asks what it was on a given day.
/// </summary>
/// <remarks>
/// <para>
/// Holds only what it was told: no interpolation, no extrapolation, and no default. A date
/// before the earliest entry has no answer, and saying so is the point — the alternative is
/// returning the oldest known value for a day it did not apply to, which is a wrong number
/// that looks like a right one.
/// </para>
/// <para>
/// The boundary is inclusive: a rate effective <em>from</em> a date applies <em>on</em> that
/// date. Both languages implement it that way, and the same rule is what the vectors pin.
/// </para>
/// </remarks>
public sealed class EffectiveDated<T>
{
    // Newest first, so the as-of walk stops at the first entry that has come into effect.
    private readonly (LocalDate From, T Value)[] entries;

    internal EffectiveDated((LocalDate From, T Value)[] source)
    {
        var ordered = (source ?? Array.Empty<(LocalDate, T)>()).ToArray();
        // Sorted here rather than assumed: the order is whatever a data file happened to be
        // written in, and relying on an author to keep it right is a silent failure waiting
        // to happen. Sorting costs nothing at startup and cannot be got wrong.
        Array.Sort(ordered, (left, right) => right.From.CompareTo(left.From));

        for (var i = 1; i < ordered.Length; i++)
        {
            if (ordered[i].From == ordered[i - 1].From)
            {
                throw new ArgumentException(
                    $"two entries take effect on {ordered[i].From}, so which one applies is undefined");
            }
        }
        entries = ordered;
    }

    /// <summary>How many entries the series holds.</summary>
    public int Count => entries.Length;

    /// <summary>The date the earliest entry took effect; null when the series is empty.</summary>
    public LocalDate? EarliestFrom => entries.Length == 0 ? null : entries[^1].From;

    /// <summary>
    /// The value in effect on a given date, if the series covers it.
    /// </summary>
    /// <remarks>
    /// A linear walk from the newest entry. A series of rates holds a handful of them and
    /// the recent end is what is asked for, so the obvious loop is also the fast path; a
    /// binary search is a change to make when a series is long enough to want one.
    /// </remarks>
    public bool TryAsOf(LocalDate on, out T value)
    {
        foreach (var entry in entries)
        {
            if (entry.From <= on)
            {
                value = entry.Value;
                return true;
            }
        }
        value = default!;
        return false;
    }
}
