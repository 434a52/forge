/**
 * A calendar date with no time and no zone — the day a rate takes effect, not an instant.
 *
 * The wire and authoring form is the ISO 8601 calendar date, and only that: `YYYY-MM-DD`,
 * every field zero-padded to its full width. The parser is strict in the way the whole f8n
 * wire format is strict — anything else is an error, never a coercion — because a lenient
 * parser lets two producers emit different bytes for one value and the contract quietly
 * becomes a dialect. See f8n/DESIGN.md → Wire format.
 *
 * Not `Date`: the point of the type is that TypeScript and C# accept and reject exactly the
 * same strings, and `Date` accepts a great deal more than this grammar, silently carries a
 * time and a zone, and turns an invalid day into a different valid one. This is a
 * conformance surface, so the grammar is implemented here and pinned by vectors.
 */
export class LocalDate {
  /** The wire form: the canonical YYYY-MM-DD string. See Percentage.toJSON on the spelling. */
  toJSON(): string {
    return this.toString();
  }

  /** Reads the wire form. Strict — it shares `parse`, so the wire accepts one spelling. */
  static fromJson(value: unknown): LocalDate {
    if (typeof value !== "string") {
      throw new Error(`a LocalDate is the string "YYYY-MM-DD", got ${typeof value}`);
    }
    return LocalDate.parse(value);
  }

  /** Proleptic Gregorian year, 1 through 9999. */
  readonly year: number;

  /** Month of year, 1 through 12. */
  readonly month: number;

  /** Day of month, 1 through the length of that month in that year. */
  readonly day: number;

  private constructor(year: number, month: number, day: number) {
    this.year = year;
    this.month = month;
    this.day = day;
  }

  /** Builds a date from its parts, rejecting one that is not a real calendar day. */
  static of(year: number, month: number, day: number): LocalDate {
    if (year < 1 || year > 9999) {
      throw new Error(`year ${year} is outside 0001-9999`);
    }
    if (month < 1 || month > 12) {
      throw new Error(`month ${month} is outside 01-12`);
    }
    const length = daysInMonth(year, month);
    if (day < 1 || day > length) {
      throw new Error(
        `day ${day} is outside 01-${pad(length, 2)} for month ${pad(month, 2)} of ${pad(year, 4)}`,
      );
    }
    return new LocalDate(year, month, day);
  }

  /**
   * Reads the canonical form `YYYY-MM-DD`. Strict: exactly ten characters, ASCII digits
   * only, both separators present, and a day that exists in that month.
   *
   * A character walk rather than a regex, for the same reason `Percentage` parses by hand:
   * two regex engines can differ in dialect at the edges, and this grammar has to mean the
   * same thing in every target — which a loop is by construction.
   */
  static parse(text: string): LocalDate {
    if (typeof text !== "string") {
      throw new Error("a date is required");
    }
    if (text.length !== 10) {
      throw new Error(`a date is exactly ten characters, YYYY-MM-DD; got "${text}"`);
    }
    if (text[4] !== "-" || text[7] !== "-") {
      throw new Error(`a date separates its parts with '-': YYYY-MM-DD; got "${text}"`);
    }
    const year = readNumber(text, 0, 4);
    const month = readNumber(text, 5, 2);
    const day = readNumber(text, 8, 2);
    return LocalDate.of(year, month, day);
  }

  /** The canonical form — always ten characters, so it round-trips through parse. */
  toString(): string {
    return `${pad(this.year, 4)}-${pad(this.month, 2)}-${pad(this.day, 2)}`;
  }

  /** Chronological order: negative, zero or positive, as a comparator wants. */
  compareTo(other: LocalDate): number {
    if (this.year !== other.year) {
      return this.year - other.year;
    }
    if (this.month !== other.month) {
      return this.month - other.month;
    }
    return this.day - other.day;
  }

  equals(other: LocalDate): boolean {
    return this.compareTo(other) === 0;
  }
}

/**
 * Reads a fixed-width run of ASCII digits.
 *
 * Deliberately not a `\d` test with the unicode flag, and deliberately not `Number(...)`,
 * which accepts leading signs, whitespace and exponent forms. The range check is what keeps
 * the accepted set identical to C#'s.
 */
function readNumber(text: string, start: number, length: number): number {
  let value = 0;
  for (let i = start; i < start + length; i++) {
    const c = text[i]!;
    if (c < "0" || c > "9") {
      throw new Error(`a date is digits and '-' only; got "${text}"`);
    }
    value = value * 10 + (c.charCodeAt(0) - 48);
  }
  return value;
}

/** Length of a month, in the proleptic Gregorian calendar. */
function daysInMonth(year: number, month: number): number {
  switch (month) {
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
      return isLeapYear(year) ? 29 : 28;
  }
}

/** Gregorian leap rule: every fourth year, except centuries that are not multiples of 400. */
function isLeapYear(year: number): boolean {
  if (year % 400 === 0) {
    return true;
  }
  if (year % 100 === 0) {
    return false;
  }
  return year % 4 === 0;
}

/** Left-pads with zeros to the field's full width, so the output is canonical. */
function pad(value: number, width: number): string {
  return String(value).padStart(width, "0");
}
