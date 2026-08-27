import type { Currency } from "./currency.js";
import { byCode } from "./generated/currency.data.js";
import type { Percentage } from "./percentage.js";
import { divideWithRounding, type RoundingMode } from "./rounding.js";

/**
 * Money's wire shape. A named type so the round trip is checked at compile time as well as at
 * run time, and so the field names are stated in exactly one place.
 */
export interface MoneyJson {
  readonly amount: string;
  readonly currency: string;
}

/**
 * An amount in a currency, held as an exact integer count of that currency's minor units.
 *
 * GBP 2 dp, JPY 0, BHD 3 — the scale is the currency's, never the caller's, so a Money is
 * always at the precision its currency actually has. Add and subtract are exact. The
 * precision-losing operations take a `RoundingMode` and land back on the currency's dp; there
 * is no hidden working precision, and anything needing more asks for it explicitly. See
 * f8n/DESIGN.md.
 *
 * A bare number cannot say whether it is major units or minor, and the factor between them is
 * currency-dependent — so there is no constructor that takes one. `fromMajor` and `fromMinor`
 * name the unit at the call site. The count is a `bigint`, never a `number`: a `number` is a
 * float64 and cannot hold what a ledger can.
 */
export class Money {
  /** The currency, which supplies the scale. */
  readonly currency: Currency;

  /** The exact count of minor units — 1234n for GBP 12.34. */
  readonly minor: bigint;

  private constructor(currency: Currency, minor: bigint) {
    this.currency = currency;
    this.minor = minor;
  }

  /** An amount given as a count of minor units — pence, cents, sen. */
  static fromMinor(minor: bigint, currency: Currency): Money {
    if (typeof minor !== "bigint") {
      throw new Error("a minor-unit count must be a bigint, not a number");
    }
    return new Money(currency, minor);
  }

  /**
   * An amount in major units, in the canonical wire form: exactly the currency's decimal
   * places, zeros included.
   *
   * Strict, and deliberately not lenient: the same grammar serves the wire, so accepting a
   * second spelling of one value would break the property that wire equality is value
   * equality. Human input — grouping, symbols, a comma decimal separator — is l10n's boundary
   * and must not share this parser.
   */
  static fromMajor(amount: string, currency: Currency): Money {
    if (typeof amount !== "string" || amount.length === 0) {
      throw new Error("an amount is required");
    }

    let index = 0;
    const negative = amount[0] === "-";
    if (negative) {
      index = 1;
    }

    const units = readUnits(amount, index, currency.minorUnits);
    index = units.next;
    const fraction = readFraction(amount, index, currency.minorUnits);
    index = fraction.next;
    if (index !== amount.length) {
      throw new Error(`"${amount}" has trailing characters`);
    }

    const minor = units.value * pow10(currency.minorUnits) + fraction.value;
    if (negative && minor === 0n) {
      // Money is an integer count, so -0 is not a second zero. Allowing it would give one
      // value two encodings and break wire equality.
      throw new Error('zero is unsigned: "-0" is not an amount');
    }
    return new Money(currency, negative ? -minor : minor);
  }

  /**
   * The canonical major-unit form — always exactly the currency's decimal places, so it
   * round-trips through `fromMajor` and equal values have identical strings.
   */
  get amount(): string {
    const negative = this.minor < 0n;
    const magnitude = negative ? -this.minor : this.minor;
    const scale = pow10(this.currency.minorUnits);
    const units = magnitude / scale;
    const fraction = magnitude - units * scale;

    let text = negative ? "-" : "";
    text += units.toString();
    if (this.currency.minorUnits > 0) {
      text += "." + fraction.toString().padStart(this.currency.minorUnits, "0");
    }
    return text;
  }

  /**
   * The wire form: `{"amount":"12.34","currency":"GBP"}`.
   *
   * This is the one primitive whose wire form is not its field layout — it holds `minor` and a
   * `Currency`, and it sends a major-unit string and a code — which is exactly when a type
   * needs a `toJSON` at all. A type that merely *holds* a Money needs none: JSON.stringify
   * walks its properties and fires this hook itself.
   *
   * Spelled toJSON, not toJson. The wrong spelling is silently ignored and the raw fields are
   * emitted instead — which here throws rather than lying, because `minor` is a bigint and
   * JSON.stringify refuses to serialise one. That loudness is a property of the bigint, not of
   * care, and it is a standing reason not to narrow the field to a number.
   */
  toJSON(): MoneyJson {
    return { amount: this.amount, currency: this.currency.code };
  }

  /**
   * Reads the wire form. Strict in every direction: the amount goes through `fromMajor`, so
   * the currency's scale is enforced; the currency must name a row that exists; and an
   * unexpected property is rejected rather than ignored.
   *
   * Rejecting unknown properties is what makes the wire round trip hold —
   * `toJSON(fromJson(w)) == w`. Ignore an extra field and it is dropped on the way back out,
   * so one value would have two encodings and wire equality would stop meaning value
   * equality. The strictness is not fastidiousness; it is what the property rests on.
   */
  static fromJson(value: unknown): Money {
    if (typeof value !== "object" || value === null || Array.isArray(value)) {
      throw new Error("a Money is an object with amount and currency");
    }
    const keys = Object.keys(value);
    const unexpected = keys.filter((key) => key !== "amount" && key !== "currency");
    if (unexpected.length > 0) {
      throw new Error(`a Money has amount and currency only; got ${unexpected.join(", ")}`);
    }

    const { amount, currency } = value as Partial<MoneyJson>;
    if (typeof amount !== "string") {
      throw new Error("a Money needs an amount, as a string");
    }
    if (typeof currency !== "string") {
      throw new Error("a Money needs a currency code, as a string");
    }
    const found = byCode(currency);
    if (found === undefined) {
      throw new Error(`unknown currency ${currency}`);
    }
    return Money.fromMajor(amount, found);
  }

  /** Exact. Adding amounts in different currencies is a bug, not a conversion. */
  add(other: Money): Money {
    this.requireSameCurrency(other, "add");
    return new Money(this.currency, this.minor + other.minor);
  }

  /** Exact. */
  subtract(other: Money): Money {
    this.requireSameCurrency(other, "subtract");
    return new Money(this.currency, this.minor - other.minor);
  }

  /** Exact. */
  negate(): Money {
    return new Money(this.currency, -this.minor);
  }

  /** Exact: a whole number of minor units times a whole number stays whole. */
  multiplyBy(factor: bigint): Money {
    return new Money(this.currency, this.minor * factor);
  }

  /**
   * Applies a proportion, landing back on the currency's dp.
   *
   * Integer multiply by the rational's numerator, then the shared divide-with-rounding by its
   * denominator — the identical function a power-of-ten scale reaches, which is why an exact
   * `Percentage` added no new conformance surface.
   */
  multiplyByRate(rate: Percentage, mode: RoundingMode): Money {
    const scaled = this.minor * rate.num;
    return new Money(this.currency, divideWithRounding(scaled, rate.den, mode));
  }

  /** Divides, landing back on the currency's dp. Does not conserve — see allocate. */
  divideBy(divisor: bigint, mode: RoundingMode): Money {
    return new Money(this.currency, divideWithRounding(this.minor, divisor, mode));
  }

  /** Order within one currency. Comparing across currencies is meaningless, so it throws. */
  compareTo(other: Money): number {
    this.requireSameCurrency(other, "compare");
    if (this.minor === other.minor) {
      return 0;
    }
    return this.minor < other.minor ? -1 : 1;
  }

  equals(other: Money): boolean {
    return this.currency.code === other.currency.code && this.minor === other.minor;
  }

  /** The amount and its currency — for logs and debugging, never for display. */
  toString(): string {
    return `${this.amount} ${this.currency.code}`;
  }

  private requireSameCurrency(other: Money, operation: string): void {
    if (this.currency.code !== other.currency.code) {
      throw new Error(
        `cannot ${operation} ${other.currency.code} and ${this.currency.code}: convert one first, with a rate and a date`,
      );
    }
  }
}

/** Reads the integer part: "0", or a non-zero digit followed by digits. */
function readUnits(amount: string, start: number, _minorUnits: number): { value: bigint; next: number } {
  let index = start;
  while (index < amount.length && amount[index]! >= "0" && amount[index]! <= "9") {
    index++;
  }
  if (index === start) {
    throw new Error(`"${amount}" has no integer part (write "0", not "")`);
  }
  const digits = amount.slice(start, index);
  if (digits.length > 1 && digits[0] === "0") {
    throw new Error(`"${amount}" has a leading zero, so the value has two spellings`);
  }
  return { value: BigInt(digits), next: index };
}

/** Reads the fractional part: exactly the currency's decimal places, or none at 0 dp. */
function readFraction(amount: string, start: number, minorUnits: number): { value: bigint; next: number } {
  if (minorUnits === 0) {
    if (start < amount.length && amount[start] === ".") {
      throw new Error(`"${amount}" has decimals, but this currency has none`);
    }
    return { value: 0n, next: start };
  }

  if (start >= amount.length || amount[start] !== ".") {
    throw new Error(
      `"${amount}" is short: this currency is written to ${minorUnits} decimal places, zeros included`,
    );
  }

  let index = start + 1;
  while (index < amount.length && amount[index]! >= "0" && amount[index]! <= "9") {
    index++;
  }
  const digits = index - (start + 1);
  if (digits !== minorUnits) {
    throw new Error(
      `"${amount}" has ${digits} decimal place(s); this currency is written to exactly ${minorUnits}`,
    );
  }
  return { value: BigInt(amount.slice(start + 1, index)), next: index };
}

function pow10(exponent: number): bigint {
  return 10n ** BigInt(exponent);
}
