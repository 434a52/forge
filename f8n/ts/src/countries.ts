import type { Country } from "./country.js";
import { byAlpha2, byAlpha3, byNumeric } from "./generated/country.data.js";

/**
 * Finds a country from a code in any of its forms — alpha-2, alpha-3, or numeric.
 *
 * An *ingestion* helper, and deliberately not what the wire uses. A country travels as its
 * alpha-3 identity and nothing else, because accepting three forms there would give one value
 * three encodings and break the rule that wire equality is value equality (f8n/DESIGN.md →
 * Wire format). This exists for the other direction: a code arriving from a payment
 * processor, an address form or a browser, where the form is whatever that system uses.
 *
 * Leniency here is safe in a way it usually is not, because the three forms occupy disjoint
 * shapes — two letters, three letters, three digits — so nothing is guessed.
 *
 * A free function rather than a static on `Country`: the indexes live in the generated
 * module, which imports `Country`, so a static would close a cycle. It is also the house
 * pattern — TypeScript free functions, tree-shakable, where C# uses a member.
 *
 * `toUpperCase` is locale-independent in JavaScript (`toLocaleUpperCase` is the locale one),
 * so this side is safe by default where C# is not — which is exactly why the normalisation is
 * written deliberately in both rather than left to each call site.
 */
export function findCountry(code: string): Country | undefined {
  if (typeof code !== "string" || code.length === 0) {
    return undefined;
  }
  const normalised = code.toUpperCase();

  if (normalised.length === 3 && isAllAsciiDigits(normalised)) {
    return byNumeric(Number(normalised));
  }
  if (normalised.length === 2) {
    return byAlpha2(normalised);
  }
  if (normalised.length === 3) {
    return byAlpha3(normalised);
  }
  return undefined;
}

/**
 * Whether every character is an ASCII digit.
 *
 * An explicit range rather than a `\d` test, matching the C# side, which cannot use
 * `char.IsDigit` — it is true for the digits of many scripts. A numeric country code is ASCII
 * or it is not one.
 */
function isAllAsciiDigits(text: string): boolean {
  for (const c of text) {
    if (c < "0" || c > "9") {
      return false;
    }
  }
  return true;
}
