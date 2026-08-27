/**
 * A dimensionless proportion, held as an exact fraction over big integers.
 *
 * The stored value is the *proportion*, never the percent number: a rate printed as "17.5%"
 * is held as 7/40. "%" is a constructor and a formatter — presentation belongs to l10n — so
 * the two never meet in the value. See f8n/DESIGN.md.
 *
 * Canonical form: gcd-reduced, `den > 0`, sign carried on `num`, zero as 0/1, integers
 * keeping /1. Equal values therefore have identical components, so equality is a component
 * compare and the canonical string is unique per value.
 *
 * Nothing here touches a binary float. `Number` cannot hold what the data can, which is the
 * whole reason this sits on bigint — the same defect, one layer up, as decoding data through
 * `any`.
 */
export class Percentage {
  /**
   * The wire form: the canonical "num/den" string.
   *
   * Spelled toJSON, not toJson — JSON.stringify recognises only the spec name, and calls it
   * on nested values too, so a Percentage inside another object serialises correctly without
   * that object knowing anything about it. Returns a value, never a string of JSON: what is
   * returned gets spliced, so a JSON string would arrive double-encoded.
   */
  toJSON(): string {
    return this.toString();
  }

  /**
   * Reads the wire form. Strict — it shares `parse`, which accepts the canonical encoding and
   * nothing else, so a value has exactly one spelling on the wire.
   */
  static fromJson(value: unknown): Percentage {
    if (typeof value !== "string") {
      throw new Error(`a Percentage is the string "num/den", got ${typeof value}`);
    }
    return Percentage.parse(value);
  }

  /** Numerator, carrying the sign. */
  readonly num: bigint;

  /** Denominator, always greater than zero. */
  readonly den: bigint;

  private constructor(num: bigint, den: bigint) {
    this.num = num;
    this.den = den;
  }

  /** Reduces to canonical form: gcd-reduced, positive denominator, zero as 0/1. */
  private static canonical(num: bigint, den: bigint): Percentage {
    if (den === 0n) {
      throw new Error("a denominator of zero is not a proportion");
    }
    if (den < 0n) {
      num = -num;
      den = -den;
    }
    if (num === 0n) {
      return new Percentage(0n, 1n);
    }
    const divisor = greatestCommonDivisor(abs(num), den);
    return new Percentage(num / divisor, den / divisor);
  }

  /** The value as authored — 0.175 is the proportion 7/40, which presents as 17.5%. */
  static fromProportion(value: string): Percentage {
    const [num, den] = parseDecimal(value);
    return Percentage.canonical(num, den);
  }

  /**
   * The percent number as a document states it — 17.5 is 7/40. Dividing by a hundred in
   * rational space is exact, so nothing is lost by authoring data the readable way.
   */
  static fromPercent(value: string): Percentage {
    const [num, den] = parseDecimal(value);
    return Percentage.canonical(num, den * 100n);
  }

  /**
   * Reads the canonical wire form "num/den". Strict: the input must already be canonical, so
   * a value's encoding is unique and wire equality stays string equality. A reducible or
   * negatively-denominated fraction is an error, not something to tidy up.
   */
  static parse(canonical: string): Percentage {
    requireString(canonical, "a canonical proportion");
    const slash = canonical.indexOf("/");
    if (slash < 0) {
      throw new Error(`expected "num/den", got "${canonical}"`);
    }

    const num = parseInteger(canonical.slice(0, slash), true);
    const den = parseInteger(canonical.slice(slash + 1), false);
    if (den === 0n) {
      throw new Error("a denominator of zero is not a proportion");
    }

    const reduced = Percentage.canonical(num, den);
    if (reduced.num !== num || reduced.den !== den) {
      throw new Error(`"${canonical}" is not canonical — expected "${reduced.toString()}"`);
    }
    return reduced;
  }

  /** The canonical wire form, "num/den". */
  toString(): string {
    return `${this.num}/${this.den}`;
  }

  equals(other: Percentage): boolean {
    return this.num === other.num && this.den === other.den;
  }
}

/**
 * Guards the string contract at runtime as well as in the types.
 *
 * `fromPercent(12.34)` is a compile error in TypeScript, but plain JavaScript callers reach
 * these functions too — and a number arriving here is float64, which is the exact loss the
 * type exists to prevent. Better a thrown error than a quietly wrong proportion.
 */
function requireString(value: unknown, what: string): asserts value is string {
  if (typeof value !== "string") {
    throw new Error(`expected ${what} as a string, got ${typeof value} — a number cannot hold it exactly`);
  }
}

/**
 * Parses a plain decimal into an exact fraction: the digits become the numerator and the
 * fraction length chooses the power of ten beneath it.
 *
 * Written as a character walk rather than a regex on purpose. Two regex engines can differ in
 * dialect on the edges, and this grammar has to mean exactly the same thing in every target
 * language — a loop over characters is identical by construction. Deliberately rejected:
 * exponent form (targets spell it differently), a leading plus (one representation only),
 * grouping separators (presentation, so l10n's), and whitespace.
 */
function parseDecimal(value: string): [bigint, bigint] {
  requireString(value, "a decimal value");
  if (value.length === 0) {
    throw new Error("expected a decimal value, got an empty string");
  }

  let index = 0;
  let negative = false;
  if (value[0] === "-") {
    negative = true;
    index = 1;
  }

  // Integer part: at least one digit, and no leading zero unless it is exactly "0" — so a
  // value has one spelling rather than several that all parse.
  const integerStart = index;
  while (index < value.length && value[index] >= "0" && value[index] <= "9") {
    index++;
  }
  const integerDigits = index - integerStart;
  if (integerDigits === 0) {
    throw new Error(`"${value}" has no digit before the decimal point`);
  }
  if (integerDigits > 1 && value[integerStart] === "0") {
    throw new Error(`"${value}" has a leading zero`);
  }

  let digits = value.slice(integerStart, index);
  let fractionDigits = 0;

  if (index < value.length && value[index] === ".") {
    index++;
    const fractionStart = index;
    while (index < value.length && value[index] >= "0" && value[index] <= "9") {
      index++;
    }
    fractionDigits = index - fractionStart;
    if (fractionDigits === 0) {
      throw new Error(`"${value}" ends in a decimal point`);
    }
    digits += value.slice(fractionStart, index);
  }

  if (index !== value.length) {
    throw new Error(`"${value}" is not a plain decimal — unexpected '${value[index]}'`);
  }

  let num = BigInt(digits);
  if (negative) {
    num = -num;
  }
  return [num, 10n ** BigInt(fractionDigits)];
}

/** Reads an integer with no leading zeros, so each value has one spelling. */
function parseInteger(text: string, allowSign: boolean): bigint {
  if (text.length === 0) {
    throw new Error("expected an integer, got an empty string");
  }

  let start = 0;
  let negative = false;
  if (allowSign && text[0] === "-") {
    negative = true;
    start = 1;
  }
  if (start >= text.length) {
    throw new Error(`"${text}" has no digits`);
  }
  for (let i = start; i < text.length; i++) {
    if (text[i] < "0" || text[i] > "9") {
      throw new Error(`"${text}" is not an integer`);
    }
  }
  if (text.length - start > 1 && text[start] === "0") {
    throw new Error(`"${text}" has a leading zero`);
  }

  const value = BigInt(text.slice(start));
  return negative ? -value : value;
}

function abs(value: bigint): bigint {
  return value < 0n ? -value : value;
}

function greatestCommonDivisor(a: bigint, b: bigint): bigint {
  while (b !== 0n) {
    [a, b] = [b, a % b];
  }
  return a;
}
