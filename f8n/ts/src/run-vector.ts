/**
 * run-vector — executes a vector file and reports what this implementation produced.
 *
 * Deliberately thin: it makes no assertions and uses no test framework. Comparison belongs
 * to the driver, so the same driver can be pointed at somebody else's implementation and
 * audit it the same way. This one reads the cases and ignores any expected values in the
 * file, which is what keeps it from grading its own work.
 */

import { readFileSync } from "node:fs";
import { findCountry } from "./countries.js";
import type { Currency } from "./currency.js";
import { AllocationRule } from "./allocation.js";
import { EffectiveDated } from "./effectivedated.js";
import { BHD, EUR, GBP, JPY, USD } from "./generated/currency.data.js";
import { LocalDate } from "./localdate.js";
import { Money } from "./money.js";
import { TaxRate } from "./taxrate.js";
import { TaxCategory } from "./generated/taxcategory.data.js";
import { TaxType } from "./generated/taxtype.data.js";
import { divideWithRounding, RoundingMode } from "./rounding.js";
import { Percentage } from "./percentage.js";

interface VectorCase {
  id: string;
  op: string;
  in: string[];
}

interface VectorGroup {
  cases: VectorCase[];
}

interface VectorFile {
  subject: string;
  groups: VectorGroup[];
}

interface CaseResult {
  id: string;
  out?: string;
  error?: string;
}

// The op alone is not enough to dispatch: "parse" means one thing for a Percentage and
// another for a LocalDate. The subject names which type's vectors these are, so one runner
// per language covers every f8n subject rather than one binary per type.
const knownOperations = new Map<string, Set<string>>([
  ["f8n.Percentage", new Set([
    "fromPercent", "fromProportion", "parse",
    "property.percentMatchesProportion", "property.roundTrip",
  ])],
  ["f8n.LocalDate", new Set(["parse", "property.roundTrip", "property.orderIsTotal"])],
  ["f8n.EffectiveDated", new Set([
    "asOf", "property.asOfAtEachBoundary", "property.orderIndependence",
  ])],
  ["f8n.Country", new Set(["find", "property.formsAgree"])],
  ["f8n.Json", new Set([
    "moneyToJson", "moneyFromJson", "percentageToJson", "localDateToJson", "taxRateToJson",
    "property.valueRoundTrip", "property.wireRoundTrip",
  ])],
  ["f8n.Allocation", new Set([
    "allocate", "allocateByWeights", "property.conserves", "property.signSymmetric",
  ])],
  ["f8n.Rounding", new Set(["divide", "property.signSymmetric"])],
  ["f8n.Money", new Set([
    "fromMajor", "fromMinor", "multiplyByRate", "divideBy",
    "property.minorMatchesMajor", "property.addSubtractIsExact",
  ])],
]);

// The vectors name a currency by its code, and the currencies are generated — so this is
// also the first place a vector run reaches c5n's output rather than only hand-written code.
function lookupCurrency(code: string): Currency {
  switch (code) {
    case "GBP": return GBP;
    case "EUR": return EUR;
    case "USD": return USD;
    case "JPY": return JPY;
    case "BHD": return BHD;
    default: throw new Error(`unknown currency ${code}`);
  }
}

function parseMode(name: string): RoundingMode {
  switch (name) {
    case "HalfEven": return RoundingMode.HalfEven;
    case "HalfUp": return RoundingMode.HalfUp;
    default: throw new Error(`unknown rounding mode ${name}`);
  }
}

/**
 * divide(-n, d) == -divide(n, d). The property the HalfUp decision was made to preserve:
 * away-from-zero keeps it, toward-positive-infinity does not.
 */
function signSymmetric(numerator: string, denominator: string, mode: string): string {
  const n = BigInt(numerator);
  const d = BigInt(denominator);
  const m = parseMode(mode);

  const positive = divideWithRounding(n, d, m);
  const negated = divideWithRounding(-n, d, m);
  if (negated !== -positive) {
    return `${numerator}/${denominator} gave ${positive} but -${numerator}/${denominator} gave ${negated}`;
  }
  return "true";
}

/** The two constructors name different units for the same value, so they must agree. */
function minorMatchesMajor(minor: string, major: string, code: string): string {
  const currency = lookupCurrency(code);
  const fromMinor = Money.fromMinor(BigInt(minor), currency);
  const fromMajor = Money.fromMajor(major, currency);
  if (!fromMinor.equals(fromMajor)) {
    return `${minor} minor units is ${fromMinor.amount}, but the major form ${major} is ${fromMajor.amount}`;
  }
  return "true";
}

/**
 * (a + b) - b == a. Addition is integer work on minor units, so it is exact by construction —
 * and this is what says so rather than assuming it.
 */
function addSubtractIsExact(first: string, second: string, code: string): string {
  const currency = lookupCurrency(code);
  const a = Money.fromMajor(first, currency);
  const b = Money.fromMajor(second, currency);
  const roundTripped = a.add(b).subtract(b);
  if (!roundTripped.equals(a)) {
    return `(${first} + ${second}) - ${second} gave ${roundTripped.amount}, not ${first}`;
  }
  return "true";
}

// A series the vectors own, so they pin the lookup's semantics rather than f8n's tax data —
// which will grow, and would take the expected values with it. Deliberately written oldest
// first, since the type sorts its own entries and must not depend on authoring order.
const fixtureEntries: [LocalDate, string][] = [
  [LocalDate.parse("2010-01-01"), "A"],
  [LocalDate.parse("2011-01-04"), "B"],
];

const fixture = EffectiveDated.of<string>(fixtureEntries);

/** Probe dates spanning the series: before it, on each boundary, between, and after. */
const probes = [
  "2009-12-31", "2010-01-01", "2010-06-15", "2011-01-03", "2011-01-04", "2011-01-05", "9999-12-31",
];

// Properties: invariants that hold for every input, so the expected value is "true" and
// comes from the rule rather than from any implementation. That is what makes them the
// independent derivation path — a value captured from one language and blessed by another
// written against the capture would still satisfy the dataset, and would not satisfy these.

/** 17.5% and the proportion 0.175 are the same value; the pairs are derived by dividing by a hundred. */
function percentMatchesProportion(percent: string, proportion: string): string {
  const fromPercent = Percentage.fromPercent(percent);
  const fromProportion = Percentage.fromProportion(proportion);
  if (fromPercent.toString() !== fromProportion.toString()) {
    return `${percent}% is ${fromPercent.toString()}, but the proportion ${proportion} is ${fromProportion.toString()}`;
  }
  return "true";
}

/**
 * parse(canonical(x)) == x. The canonical form is unique per value, so a value that survives
 * a round trip has exactly one encoding — which is what makes wire equality mean value
 * equality.
 */
function percentageRoundTrip(percent: string): string {
  const canonical = Percentage.fromPercent(percent).toString();
  const reparsed = Percentage.parse(canonical).toString();
  if (reparsed !== canonical) {
    return `${percent}% canonicalises to ${canonical} but reparses to ${reparsed}`;
  }
  return "true";
}

function dateRoundTrip(text: string): string {
  const canonical = LocalDate.parse(text).toString();
  const reparsed = LocalDate.parse(canonical).toString();
  if (reparsed !== canonical) {
    return `${text} canonicalises to ${canonical} but reparses to ${reparsed}`;
  }
  return "true";
}

/**
 * A comparison is a total order: equal to itself, antisymmetric, and transitive. The dates
 * arrive in ascending order and every pair is checked in both directions.
 */
function orderIsTotal(ascending: string[]): string {
  const dates = ascending.map((text) => LocalDate.parse(text));
  for (let i = 0; i < dates.length; i++) {
    if (dates[i]!.compareTo(dates[i]!) !== 0) {
      return `${dates[i]!.toString()} does not compare equal to itself`;
    }
    for (let j = i + 1; j < dates.length; j++) {
      if (dates[i]!.compareTo(dates[j]!) >= 0) {
        return `${dates[i]!.toString()} does not sort before ${dates[j]!.toString()}`;
      }
      if (dates[j]!.compareTo(dates[i]!) <= 0) {
        return `${dates[j]!.toString()} does not sort after ${dates[i]!.toString()}`;
      }
    }
  }
  return "true";
}

/**
 * Every form of one country finds the same row. This is what makes accepting three forms a
 * convenience rather than three different answers.
 */
function formsAgree(alpha2: string, alpha3: string, numeric: string): string {
  const byTwo = findCountry(alpha2);
  const byThree = findCountry(alpha3);
  const byNumber = findCountry(numeric);
  if (byTwo === undefined || byThree === undefined || byNumber === undefined) {
    return `one of ${alpha2}/${alpha3}/${numeric} found nothing`;
  }
  if (byTwo.alpha3 !== byThree.alpha3 || byTwo.alpha3 !== byNumber.alpha3) {
    return `${alpha2} found ${byTwo.alpha3}, ${alpha3} found ${byThree.alpha3}, ${numeric} found ${byNumber.alpha3}`;
  }
  return "true";
}

function parseRule(spec: string): AllocationRule {
  if (spec.startsWith("Designated:")) {
    return AllocationRule.designated(Number(spec.slice("Designated:".length)));
  }
  switch (spec) {
    case "LargestRemainder": return AllocationRule.largestRemainder;
    case "Sequential": return AllocationRule.sequential;
    default: throw new Error(`unknown allocation rule ${spec}`);
  }
}

function parseWeights(csv: string): bigint[] {
  return csv.split(",").map((part) => BigInt(part));
}

function renderParts(parts: Money[]): string {
  return parts.map((part) => part.amount).join(" ");
}

/**
 * sum(allocate(m, rule)) == m, exactly. The invariant the whole operation exists for, and the
 * reason it is not a rounding op: rounding each part independently loses the odd unit.
 */
function conserves(amount: string, code: string, weights: string, rule: string): string {
  const currency = lookupCurrency(code);
  const whole = Money.fromMajor(amount, currency);
  const parts = whole.allocateByWeights(parseWeights(weights), parseRule(rule));

  let sum = Money.fromMinor(0n, currency);
  for (const part of parts) {
    sum = sum.add(part);
  }
  if (!sum.equals(whole)) {
    return `${whole.amount} split ${weights} by ${rule} summed to ${sum.amount}`;
  }
  return "true";
}

/**
 * allocate(-m) == -allocate(m), componentwise. True by construction because the sign is lifted
 * out before the partition and reapplied after — which is why the implementation works on the
 * magnitude rather than flooring signed shares.
 */
function allocationSignSymmetric(amount: string, code: string, weights: string, rule: string): string {
  const currency = lookupCurrency(code);
  const positive = Money.fromMajor(amount, currency).allocateByWeights(parseWeights(weights), parseRule(rule));
  const negated = Money.fromMajor(amount, currency).negate().allocateByWeights(parseWeights(weights), parseRule(rule));

  for (let i = 0; i < positive.length; i++) {
    if (!negated[i]!.equals(positive[i]!.negate())) {
      return `part ${i}: ${positive[i]!.amount} negated is ${positive[i]!.negate().amount}, but the negative split gave ${negated[i]!.amount}`;
    }
  }
  return "true";
}

/** fromJson(toJSON(x)) == x — nothing is lost on the way out. */
function jsonValueRoundTrip(amount: string, code: string): string {
  const original = Money.fromMajor(amount, lookupCurrency(code));
  const back = Money.fromJson(JSON.parse(JSON.stringify(original)));
  if (!back.equals(original)) {
    return `${original.toString()} serialised and read back as ${back.toString()}`;
  }
  return "true";
}

/**
 * toJSON(fromJson(w)) == w, over a CANONICAL w — nothing else survives a round trip, which
 * holds only because the wire form has one encoding per value and unknown properties are
 * rejected rather than dropped.
 */
function jsonWireRoundTrip(wire: string): string {
  const back = JSON.stringify(Money.fromJson(JSON.parse(wire)));
  if (back !== wire) {
    return `${wire} read back and re-emitted as ${back}`;
  }
  return "true";
}

/** Every entry is in effect on the day it takes effect. */
function asOfAtEachBoundary(): string {
  for (const [from, expected] of fixtureEntries) {
    const actual = fixture.asOf(from);
    if (actual === undefined) {
      return `no value at ${from.toString()}, which is an entry's own effective date`;
    }
    if (actual !== expected) {
      return `at ${from.toString()} the series gave "${actual}", not the entry's own "${expected}"`;
    }
  }
  return "true";
}

/**
 * The type sorts its own entries, so the order a data file happened to be written in cannot
 * change an answer. Asserted rather than assumed: nothing else tests it, and the failure it
 * prevents is a data file rearranged for readability changing a rate.
 */
function orderIndependence(): string {
  const other = EffectiveDated.of<string>([...fixtureEntries].reverse());
  for (const probe of probes) {
    const date = LocalDate.parse(probe);
    const first = fixture.asOf(date) ?? "(none)";
    const second = other.asOf(date) ?? "(none)";
    if (first !== second) {
      return `at ${probe} the two orderings gave "${first}" and "${second}"`;
    }
  }
  return "true";
}

function execute(subject: string, op: string, inputs: string[]): string {
  switch (`${subject}.${op}`) {
    case "f8n.Percentage.fromPercent":
      return Percentage.fromPercent(inputs[0]).toString();
    case "f8n.Percentage.fromProportion":
      return Percentage.fromProportion(inputs[0]).toString();
    case "f8n.Percentage.parse":
      return Percentage.parse(inputs[0]).toString();
    case "f8n.LocalDate.parse":
      return LocalDate.parse(inputs[0]).toString();
    // A day the series does not cover is a defined outcome, not a rejected input, so it
    // reports a value of its own rather than throwing — an error would put it in the same
    // bucket as a malformed date, which is a different thing entirely.
    case "f8n.EffectiveDated.asOf":
      return fixture.asOf(LocalDate.parse(inputs[0])) ?? "(none)";
    case "f8n.Percentage.property.percentMatchesProportion":
      return percentMatchesProportion(inputs[0], inputs[1]);
    case "f8n.Percentage.property.roundTrip":
      return percentageRoundTrip(inputs[0]);
    case "f8n.LocalDate.property.roundTrip":
      return dateRoundTrip(inputs[0]);
    case "f8n.LocalDate.property.orderIsTotal":
      return orderIsTotal(inputs);
    case "f8n.EffectiveDated.property.asOfAtEachBoundary":
      return asOfAtEachBoundary();
    case "f8n.EffectiveDated.property.orderIndependence":
      return orderIndependence();
    case "f8n.Rounding.divide":
      return divideWithRounding(BigInt(inputs[0]), BigInt(inputs[1]), parseMode(inputs[2])).toString();
    case "f8n.Rounding.property.signSymmetric":
      return signSymmetric(inputs[0], inputs[1], inputs[2]);
    case "f8n.Money.fromMajor":
      return Money.fromMajor(inputs[0], lookupCurrency(inputs[1])).amount;
    case "f8n.Money.fromMinor":
      return Money.fromMinor(BigInt(inputs[0]), lookupCurrency(inputs[1])).amount;
    case "f8n.Money.multiplyByRate":
      return Money.fromMajor(inputs[0], lookupCurrency(inputs[1]))
        .multiplyByRate(Percentage.fromPercent(inputs[2]), parseMode(inputs[3])).amount;
    case "f8n.Money.divideBy":
      return Money.fromMajor(inputs[0], lookupCurrency(inputs[1]))
        .divideBy(BigInt(inputs[2]), parseMode(inputs[3])).amount;
    case "f8n.Money.property.minorMatchesMajor":
      return minorMatchesMajor(inputs[0], inputs[1], inputs[2]);
    case "f8n.Money.property.addSubtractIsExact":
      return addSubtractIsExact(inputs[0], inputs[1], inputs[2]);
    case "f8n.Country.find":
      return findCountry(inputs[0])?.alpha3 ?? "(none)";
    case "f8n.Country.property.formsAgree":
      return formsAgree(inputs[0], inputs[1], inputs[2]);
    case "f8n.Json.moneyToJson":
      return JSON.stringify(Money.fromMajor(inputs[0], lookupCurrency(inputs[1])));
    case "f8n.Json.moneyFromJson":
      return Money.fromJson(JSON.parse(inputs[0])).toString();
    case "f8n.Json.percentageToJson":
      return JSON.stringify(Percentage.fromPercent(inputs[0]));
    case "f8n.Json.localDateToJson":
      return JSON.stringify(LocalDate.parse(inputs[0]));
    // The nested case: TaxRate has no toJSON of its own, so this is JSON.stringify walking a
    // container and firing each value's own hook.
    case "f8n.Json.taxRateToJson":
      return JSON.stringify(new TaxRate(
        findCountry(inputs[0])!, TaxType.VAT, TaxCategory.Standard,
        Percentage.fromPercent(inputs[1])));
    case "f8n.Json.property.valueRoundTrip":
      return jsonValueRoundTrip(inputs[0], inputs[1]);
    case "f8n.Json.property.wireRoundTrip":
      return jsonWireRoundTrip(inputs[0]);
    case "f8n.Allocation.allocate":
      return renderParts(Money.fromMajor(inputs[0], lookupCurrency(inputs[1]))
        .allocate(Number(inputs[2]), parseRule(inputs[3])));
    case "f8n.Allocation.allocateByWeights":
      return renderParts(Money.fromMajor(inputs[0], lookupCurrency(inputs[1]))
        .allocateByWeights(parseWeights(inputs[2]), parseRule(inputs[3])));
    case "f8n.Allocation.property.conserves":
      return conserves(inputs[0], inputs[1], inputs[2], inputs[3]);
    case "f8n.Allocation.property.signSymmetric":
      return allocationSignSymmetric(inputs[0], inputs[1], inputs[2], inputs[3]);
    default:
      throw new Error(`unknown op ${op} for ${subject}`);
  }
}

function main(): number {
  const path = process.argv[2];
  if (path === undefined) {
    process.stderr.write("usage: run-vector <vectors.json>\n");
    return 1;
  }

  let document: VectorFile;
  try {
    document = JSON.parse(readFileSync(path, "utf8")) as VectorFile;
  } catch (cause) {
    process.stderr.write(`run-vector: cannot read ${path}: ${String(cause)}\n`);
    return 1;
  }

  const operations = knownOperations.get(document.subject);
  if (operations === undefined) {
    process.stderr.write(`run-vector: unknown subject "${document.subject}"\n`);
    return 2;
  }

  const results: CaseResult[] = [];
  for (const group of document.groups) {
    for (const testCase of group.cases) {
      // An unknown op is a fault in the harness, not a failing case — reporting it as a
      // case error would let a reject case "pass" for entirely the wrong reason.
      if (!operations.has(testCase.op)) {
        process.stderr.write(`run-vector: ${testCase.id}: unknown op "${testCase.op}"\n`);
        return 2;
      }
      try {
        results.push({ id: testCase.id, out: execute(document.subject, testCase.op, testCase.in) });
      } catch (cause) {
        results.push({ id: testCase.id, error: cause instanceof Error ? cause.message : String(cause) });
      }
    }
  }

  process.stdout.write(JSON.stringify(results) + "\n");
  return 0;
}

process.exitCode = main();
