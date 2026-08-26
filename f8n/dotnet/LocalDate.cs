namespace F8n;

/// <summary>
/// A calendar date with no time and no zone — the day a rate takes effect, not an instant.
/// </summary>
/// <remarks>
/// <para>
/// The wire and authoring form is the ISO 8601 calendar date, and only that:
/// <c>YYYY-MM-DD</c>, every field zero-padded to its full width. The parser is strict in the
/// way the whole f8n wire format is strict — anything else is an error, never a coercion —
/// because a lenient parser lets two producers emit different bytes for one value and the
/// contract quietly becomes a dialect. See f8n/DESIGN.md → Wire format.
/// </para>
/// <para>
/// Not <c>DateTime</c> and not <c>DateOnly</c>: the point of the type is that C# and
/// TypeScript accept and reject exactly the same strings, and a BCL parser brings its own
/// accepted forms and its own culture sensitivity, neither of which TypeScript can match.
/// This is a conformance surface, so the grammar is implemented here and pinned by vectors.
/// </para>
/// </remarks>
public readonly struct LocalDate : IEquatable<LocalDate>, IComparable<LocalDate>
{
    /// <summary>Proleptic Gregorian year, 1 through 9999.</summary>
    public int Year { get; }

    /// <summary>Month of year, 1 through 12.</summary>
    public int Month { get; }

    /// <summary>Day of month, 1 through the length of that month in that year.</summary>
    public int Day { get; }

    private LocalDate(int year, int month, int day)
    {
        Year = year;
        Month = month;
        Day = day;
    }

    /// <summary>Builds a date from its parts, rejecting one that is not a real calendar day.</summary>
    public static LocalDate Of(int year, int month, int day)
    {
        if (year < 1 || year > 9999)
        {
            throw new FormatException($"year {year} is outside 0001-9999");
        }
        if (month < 1 || month > 12)
        {
            throw new FormatException($"month {month} is outside 01-12");
        }
        var length = DaysInMonth(year, month);
        if (day < 1 || day > length)
        {
            throw new FormatException($"day {day} is outside 01-{length:D2} for month {month:D2} of {year:D4}");
        }
        return new LocalDate(year, month, day);
    }

    /// <summary>
    /// Reads the canonical form <c>YYYY-MM-DD</c>. Strict: exactly ten characters, ASCII
    /// digits only, both separators present, and a day that exists in that month.
    /// </summary>
    /// <remarks>
    /// A character walk rather than a regex or a BCL parser, for the same reason
    /// <see cref="Percentage"/> parses by hand: two regex engines can differ in dialect at
    /// the edges, and this grammar has to mean the same thing in every target — which a
    /// loop is by construction.
    /// </remarks>
    public static LocalDate Parse(string text)
    {
        if (text is null)
        {
            throw new FormatException("a date is required");
        }
        if (text.Length != 10)
        {
            throw new FormatException($"a date is exactly ten characters, YYYY-MM-DD; got \"{text}\"");
        }
        if (text[4] != '-' || text[7] != '-')
        {
            throw new FormatException($"a date separates its parts with '-': YYYY-MM-DD; got \"{text}\"");
        }
        var year = ReadNumber(text, 0, 4);
        var month = ReadNumber(text, 5, 2);
        var day = ReadNumber(text, 8, 2);
        return Of(year, month, day);
    }

    /// <summary>
    /// Reads a fixed-width run of ASCII digits.
    /// </summary>
    /// <remarks>
    /// Deliberately not <c>char.IsDigit</c>, which is true for the Unicode digits of many
    /// scripts — Arabic-Indic '٤' among them. Using it would make C# accept strings
    /// TypeScript rejects, which is precisely the divergence this type exists to prevent.
    /// </remarks>
    private static int ReadNumber(string text, int start, int length)
    {
        var value = 0;
        for (var i = start; i < start + length; i++)
        {
            var c = text[i];
            if (c < '0' || c > '9')
            {
                throw new FormatException($"a date is digits and '-' only; got \"{text}\"");
            }
            value = (value * 10) + (c - '0');
        }
        return value;
    }

    /// <summary>Length of a month, in the proleptic Gregorian calendar.</summary>
    private static int DaysInMonth(int year, int month)
    {
        switch (month)
        {
            case 1:
            case 3:
            case 5:
            case 7:
            case 8:
            case 10:
            case 12:
                return 31;
            case 4:
            case 6:
            case 9:
            case 11:
                return 30;
            default:
                return IsLeapYear(year) ? 29 : 28;
        }
    }

    /// <summary>Gregorian leap rule: every fourth year, except centuries that are not multiples of 400.</summary>
    private static bool IsLeapYear(int year)
    {
        if (year % 400 == 0)
        {
            return true;
        }
        if (year % 100 == 0)
        {
            return false;
        }
        return year % 4 == 0;
    }

    /// <summary>The canonical form — always ten characters, so it round-trips through Parse.</summary>
    public override string ToString()
    {
        return $"{Year:D4}-{Month:D2}-{Day:D2}";
    }

    /// <summary>Chronological order.</summary>
    public int CompareTo(LocalDate other)
    {
        if (Year != other.Year)
        {
            return Year.CompareTo(other.Year);
        }
        if (Month != other.Month)
        {
            return Month.CompareTo(other.Month);
        }
        return Day.CompareTo(other.Day);
    }

    public bool Equals(LocalDate other) => CompareTo(other) == 0;

    public override bool Equals(object? obj) => obj is LocalDate other && Equals(other);

    public override int GetHashCode() => HashCode.Combine(Year, Month, Day);

    public static bool operator ==(LocalDate left, LocalDate right) => left.Equals(right);

    public static bool operator !=(LocalDate left, LocalDate right) => !left.Equals(right);

    public static bool operator <(LocalDate left, LocalDate right) => left.CompareTo(right) < 0;

    public static bool operator >(LocalDate left, LocalDate right) => left.CompareTo(right) > 0;

    public static bool operator <=(LocalDate left, LocalDate right) => left.CompareTo(right) <= 0;

    public static bool operator >=(LocalDate left, LocalDate right) => left.CompareTo(right) >= 0;
}
