import { LocalDate } from "./localdate.js";

/**
 * A value that changes on known dates — a tax rate, a threshold, a band — and the lookup
 * that asks what it was on a given day.
 *
 * Holds only what it was told: no interpolation, no extrapolation, and no default. A date
 * before the earliest entry has no answer, and saying so is the point — the alternative is
 * returning the oldest known value for a day it did not apply to, which is a wrong number
 * that looks like a right one.
 *
 * The boundary is inclusive: a rate effective *from* a date applies *on* that date. Both
 * languages implement it that way, and the same rule is what the vectors pin.
 */
export class EffectiveDated<T> {
  // Newest first, so the as-of walk stops at the first entry that has come into effect.
  private readonly entries: readonly (readonly [LocalDate, T])[];

  private constructor(entries: (readonly [LocalDate, T])[]) {
    this.entries = entries;
  }

  /** Builds a series from its entries, in any order. */
  static of<V>(entries: (readonly [LocalDate, V])[]): EffectiveDated<V> {
    // Sorted here rather than assumed: the order is whatever a data file happened to be
    // written in, and relying on an author to keep it right is a silent failure waiting to
    // happen. Sorting costs nothing at startup and cannot be got wrong.
    const ordered = [...entries].sort((left, right) => right[0].compareTo(left[0]));

    for (let i = 1; i < ordered.length; i++) {
      if (ordered[i]![0].equals(ordered[i - 1]![0])) {
        throw new Error(
          `two entries take effect on ${ordered[i]![0].toString()}, so which one applies is undefined`,
        );
      }
    }
    return new EffectiveDated<V>(ordered);
  }

  /** How many entries the series holds. */
  get count(): number {
    return this.entries.length;
  }

  /** The date the earliest entry took effect; undefined when the series is empty. */
  get earliestFrom(): LocalDate | undefined {
    return this.entries.length === 0 ? undefined : this.entries[this.entries.length - 1]![0];
  }

  /**
   * The value in effect on a given date, or undefined if the series does not cover it.
   *
   * A linear walk from the newest entry. A series of rates holds a handful of them and the
   * recent end is what is asked for, so the obvious loop is also the fast path; a binary
   * search is a change to make when a series is long enough to want one.
   */
  asOf(on: LocalDate): T | undefined {
    for (const [from, value] of this.entries) {
      if (from.compareTo(on) <= 0) {
        return value;
      }
    }
    return undefined;
  }
}
