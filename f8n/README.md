# f8n

Domain primitives — Money, Currency, Country, Locale, PhoneNumber, rate types, temporal types — generated cross-language (C# + TS) from canonical data, with **hand-written behaviour verified by golden vectors**.

Design: `./DESIGN.md`, `./data-lookups.md`.

**Status: early.** The Currency and Country tables are generated from `data/` by c5n and
compile in both targets. The money and rate types — the hand-written behaviour, and the
spec that verifies it — are still to come.
