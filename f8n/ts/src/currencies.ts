import type { Currency } from "./currency.js";
import { byCode } from "./generated/currency.data.js";

/**
 * Reads a currency from the wire, where it travels as its ISO 4217 alpha code.
 *
 * A free function rather than a static on `Currency` because the index lives in the generated
 * module, which imports `Currency`; a static would close a cycle. Same shape as
 * `countryFromJson`, and the same reason.
 */
export function currencyFromJson(value: unknown): Currency {
  if (typeof value !== "string") {
    throw new Error(`a Currency is its alpha code, as a string; got ${typeof value}`);
  }
  const found = byCode(value);
  if (found === undefined) {
    throw new Error(`unknown currency ${value}`);
  }
  return found;
}
