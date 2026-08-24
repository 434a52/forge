# ampersand — design

**ampersand** — a consumer **home-electrification planner**, in the `c5n`/`f8n`/`l10n` family. Answers the homeowner's question: *can I run solar + battery + heat pump + EV charger(s) on my existing supply — without blowing the fuse or the budget — and how?* Top-of-stack consumer product; dogfoods the whole stack. Skeleton — agenda open. See `../f8n/DESIGN.md`, `../l10n/DESIGN.md`, `../palette/DESIGN.md`, `../lattice/DESIGN.md`, `../etch/DESIGN.md`, `../a11y/DESIGN.md`.

## The name
**amp** (the supply constraint — a UK home is ~60–100A single-phase) **+ & / "and"** (running solar *and* battery *and* heat pump *and* chargers, all together). The product is a pile of "ands" that has to fit inside the amps. The `&` glyph is the mark.

## What it is
- **The product** — a **home energy economics sandbox**. Describe your home and kit, choose a tariff, adjust the controls, and watch what it costs you across any span from a day to a year. The verb is *change something and see the number move*; the flagship question is **"is another 10 kWh battery — or more panels on the garage roof — actually worth it?"**
- **Not a constraint checker.** Fuse headroom matters (see *The binding constraint*), but a tool whose headline is "will it trip" is a boring product nobody buys. The headline is **money**: when to charge, when to discharge, what to do with surplus, what happens when the export price collapses, whether to dump into hot water or run the heat pump — and what each choice is worth over a year.
- **Honest by design.** A tool that says *"don't buy the second battery — £5k for £300 a year is a 16-year payback"* is far more useful, and far more trustworthy, than one that always says yes. Every vendor calculator is a sales instrument; being the one that tells you when the answer is **no** is the position.
- **The engine (`ballast`)** — the dispatch simulator and optimiser: a hard power ceiling, competing loads bidding for a scarce cheap-rate window, and a multi-objective cost function (import cost vs export revenue vs battery degradation vs comfort). *Decorator* = eligibility/constraints, *strategy* = the optimisation algorithm. The name holds: an electrical ballast **limits current**, and current limiting turns out to be what decides whether the capital spend pays back.

## The centrepiece — one chart, any timescale
**One chart, one Y axis, continuous zoom.** kW on Y, time on X, from 24 hours out to a year. At long ranges the value is *average power per bucket* rather than instantaneous — which is the same quantity, because **the area under the curve is always energy**. Same axis, same mental model, learnable once; only the bucket size changes.

- **Stacked by source and sink** — demand by device; generation, battery and grid import as the supply side.
- **Totals beneath are money** — always the integral of what's on screen: import cost, export revenue, net, and delta against a baseline. Zoom the chart and the totals follow.
- **A min/max band** behind the mean covers peaks lost to aggregation, if and when peaks matter.

*(Visual system — colour, marks, legend, the range control — is a separate pass, to be specified against the dataviz guidance rather than improvised here.)*

## Rendering — SVG, and what animates
**SVG, not canvas.** Crisp at any zoom, styleable directly from `palette` tokens, carries ARIA, animates natively. Canvas only wins at far higher point densities than this will reach.

**The one performance caveat: DOM size.** A year at hourly resolution is 8,760 points across several series. Render each series as **a single `<path>` carrying many coordinates — never one element per point** — which keeps the DOM at a handful of nodes rather than tens of thousands. At day resolution (96 points at 15-minute buckets) none of this matters.

**Two animations, doing different jobs:**
- **State transitions** — move a slider and the curve morphs. This is the one that sells the product: it makes cause and effect visceral (drop the export price and *watch* the dispatch reorganise). Technically `d`-attribute interpolation, which needs matched point counts between states or a tweening approach — a real but tractable problem, and worth doing well, because it is the thing a design-minded reviewer will fixate on.
- **A playhead sweep** — run through a simulated day with loads switching on and off against the ceiling. Cheap to build, disproportionately effective in a demo, and it explains the mechanism better than any static frame.

**Accessibility here is a demonstration, not an obligation.** Honour `prefers-reduced-motion`; give the data a non-visual representation (a table behind a toggle, or an ARIA summary); announce meaningful state changes through `a11y`'s announcer. **An animated chart that remains fully usable without the animation is the "inclusive by construction" claim shown rather than asserted** — on the most visible surface in the stack.

*(Consumes `etch` for the SVG component layer. Visual system — colour, marks, legend, the range control — remains its own pass against the dataviz guidance.)*

## Winter is the discovery, not the framing
**Show the year whole and let the user find it.** Most people have never thought it through, and being lectured about winter is worse than seeing it: the year view makes the shape self-evident — a 8–12 week generation dead spot, heat demand peaking, and the cost sink sitting right on top of it.

What the tool then reveals, in order:
1. **Summer is boring** — everything is in surplus, no decision matters, every strategy works. (This is the only season vendor calculators show.)
2. **In the dead spot the homeowner is grid-dependent**, and the battery's *job changes*: no longer solar-shifting (there's nothing to shift) but **tariff arbitrage** — buy cheap overnight, avoid peak import. Same hardware, completely different mode, and almost nobody explains it.
3. **Therefore the answers are seasonal.** A second battery may earn nothing from May to September and pay for itself in January. That's a surprising, quantified, honest answer — and precisely the kind no vendor will give.

## The binding constraint — absorption, not safety
The fuse is not a safety footnote; it is **an economic constraint on capital decisions**. In winter, savings are capped by **how much cheap-rate energy can physically be absorbed before the window closes** — and everything competes for the same window:

| Load | Draw |
|---|---|
| 2 × EV charger | ~14 kW |
| Heat pump at low ambient (COP down, draw up) | 4–5 kW |
| Battery inverter, each | 3.6–5 kW |

Against roughly **23 kW at 100 A**, **~18 kW at 80 A**, **~14 kW at 60 A** — where two cars alone have consumed everything. The binding number is:

```
absorbable cheap energy  =  window length × headroom under the supply ceiling
```

**Which makes supply size a primary input, not a detail** — 60/80/100 A produces genuinely different answers for the same house and the same kit. It also makes the **DNO upgrade an answerable capex question**: *"60 A → 100 A costs £X and unlocks £Y a year of absorbable cheap energy."*

And it is what makes the optimiser necessary rather than decorative: which car charges first, both at 3.5 kW or one at 7, heat pump before or after the cars, battery trickled across the whole window at reduced rate — real trade-offs with different costs and no obvious answer.

## EV charging — in v1, and it's what makes the system interesting
**One or two cars, toggleable.** The second car *is* the stress case, and "what if we get another EV" is one of the flagship questions.

**Presence model: away 08:00–18:00 on weekdays; home at weekends.** Sharper than it looks because of what the weekday pattern excludes — **the commuter car never sees the solar window**, so on weekdays it is a pure grid load bidding for the cheap-rate window. That yields a genuinely counter-intuitive output: **more panels do nothing for your weekday commute.** Most people assume the opposite.

**Weekends are a distinct regime, and they earn their place.** Home all day, generation available, no departure deadline — the one time the car *can* take solar. It sharpens the seasonal contrast: in July the weekend car soaks surplus for nothing; in January there is no surplus and it is back to bidding for the cheap window.

- **Simulate state of charge; charging *frequency* is an output, not an input.** Battery capacity, each departure consumes the commute's kWh, the car must hold enough at every departure, and charging may happen whenever it is home. *"You only need to charge Tuesday and Friday"* then falls out as a **result** — which is a better demo, gives the optimiser real slack (skip expensive nights, batch into cheap ones, exploit weekend sun), and means a low-mileage car with a big battery behaves entirely differently from a high-mileage one with a small battery without either being configured differently.
- **Demand configurable per car.** Input in **miles/day** with a mi/kWh default (~3.5) and convert — users think in miles, the model wants kWh.
- **Charge rate is a per-car *maximum* the optimiser may go below.** Essential: all-or-nothing 7 kW leaves `ballast` almost no options, whereas modulated charging (one car at 7, one at 3.5; both trickled across a window) is where the trade-offs live.
- **Constraint: enough charge at each departure**, not "full every night." That is what makes the scheduling real, and with two cars it is what makes the window genuinely scarce.

**Known simplifications, with the direction of error stated** (per the honesty positioning): a fixed weekday/weekend pattern with no holidays, working-from-home days, or irregular trips — all of which add charging opportunities, so the model **mildly overstates grid dependence**. Conservative, which is the right direction to be wrong in, but say so.

**Default scenario should be feasible.** Start with a configuration where the constraint doesn't bite, so the user *discovers the cliff* by adjusting — a second car, higher mileage, a shorter window, a 60 A supply. A default that's already infeasible reads as broken rather than as insight.

**Deferred, but worth naming as a seam:** bidirectional charging. A car battery is 50–80 kWh — it dwarfs any home battery — so if V2G becomes generally available **every answer in this tool changes**, and "should I buy a home battery at all?" becomes a live question. Not v1; don't design it in, don't design it out.

## Solar arrays — multiple planes, asked cheaply
**Multiple arrays, each with kWp, azimuth and tilt.** Single-figure kWp cannot answer the flagship question, because *"what if I add the garage roof?"* is a question about a **different roof plane** facing a different way.

**Ask orientation precisely; derive tilt.** At UK latitudes **orientation dominates and pitch barely matters**: south-facing output varies only ~5–8% across pitches from 20° to 50°, while east/west loses ~15–20% against south and north loses 40%+. So spend the user's attention on the compass and estimate the pitch from **house type** (Victorian/1930s ~40–45°, post-war/modern ~30–35°, bungalow ~30°, flat-roof installs framed at ~10–15°). Same trick as the heat pump nameplate: ask the question they can answer, derive the rest.

**The insight multi-array unlocks:** an **east/west split generates less total energy than south but more *usable* energy** — output spreads across morning and evening instead of spiking at noon when the house is empty. Large effect without a battery, mostly absorbed with one. Almost no homeowner knows this, and it changes what "more panels" is worth.

**Inverter capacity is a separate input from panel kWp, and clipping is modelled.** Oversized arrays clip, so added panels can produce *nothing* on exactly the sunny days they were bought for. This bites hardest in the UK because **G98 permits ~3.68 kW per phase without DNO approval and above that needs a G99 application**, so many installs are deliberately inverter-limited. It makes an honest answer to the garage-roof question sometimes *"your inverter can't take it, and going bigger means a DNO application"* — the DNO theme arriving from a second direction. *(Verify current G98/G99 thresholds — fast-moving.)*

**Shading is the model's largest single unknown and should be admitted as such.** A chimney or a tree costs 20–30% and cannot be derived from anything cheap to ask. MVP: a qualitative per-array input (none / light / moderate / heavy → a stated derate), plainly labelled as the biggest uncertainty in the solar figures. Terrain shading comes free from the irradiance source; local obstructions do not.

**Also:** system losses default to the standard ~14% (inverter, DC, temperature, soiling), exposed as advanced; and **panel degradation ~0.5%/yr** belongs in the payback arithmetic even if not in the simulation.

## Weather years — real and synthetic
**v1 is a year picker over real years.** No synthetic composition to begin with — real years mean nothing is spliced, nothing is discontinuous at month boundaries, nothing is colder than was ever observed, and nothing needs labelling as artificial. **Every year actually happened**, which suits the honesty position with no caveats attached. Synthetic composites (below) are a later refinement, not the starting point.

**Highlight years from the data, per location.** Since data is fetched per postcode, the annotations are location-specific — *"2010 was **your** coldest winter"* is both available and far more compelling than a national claim. Two tiers, in build order:
- **Weather-derived (v1)** — cheap and pre-computable: coldest winter (heating degree days Nov–Mar), dullest winter (winter irradiance), brightest year, warmest year.
- **Outcome-derived (later)** — once simulation runs: *"your worst year was 2010 — £340 more than typical."* A far stronger label than any weather statistic, and the honest-range argument made personal. Second-order work: it needs a full run per year.

**Two data resolutions.** **Daily** aggregates are sufficient for ranking years and generating labels; **hourly** is needed only for the simulation itself. Fetch daily across the whole record, hourly only for the year actually selected — a large saving on fetch, storage and cache-warm time.

**Historic extremes sit in their own group.** 1963 and 1981 are worth offering as explicit stress tests, but flagged as such rather than mixed silently into a recent-years list — the climate has shifted since, so presenting them as peers of 2020s years would quietly overstate their likelihood.

**Weather year is a first-class control, not a setting.** A strategy tuned to one year is overfitted, and the interesting output is not *the optimal schedule* but **how a strategy holds up across years**.

- **Real named years** — last year, a notably cold one, a notably hot one. Defensible and relatable: *this actually happened*. Historical reanalysis reaches back to 1940, so a genuine cold snap from the early eighties is available.
- **Synthetic representative years (deferred — a refinement, not v1)** — cold / average / warm, composed from real data. The established **typical-meteorological-year** method: for each calendar month, rank the historical months by a metric (heating degree days, mean irradiance) and splice the median — or a cold/warm percentile — into a composite of real months.
- *Two wrinkles to handle honestly:* splicing months from different years creates **discontinuities at month boundaries** (standard TMY methods smooth across a few days); and picking the coldest January and coldest February from different years yields a winter **colder than any observed**. Fine as a stress test, but it must be **labelled synthetic**, never presented as a year that happened. Given the honesty positioning, that labelling is not optional.
- **Report a range, not a number** — *"between £820 and £1,140 a year depending on the weather."* Every competitor quotes one confident figure; quoting the spread costs a loop and buys credibility.

**And running a counterfactual must say so:** 1981's weather against today's kit and today's tariffs is a legitimate question and *not* what happened in 1981.

## The heat model — on the critical path
**Winter economics are only credible with a heat demand model, and that requires the building.** Solar comes from an API; heat demand has to be modelled, and it dominates every winter figure.

**Ask for the heat pump's nameplate output — it is a proxy for the building.** Installers size to the calculated heat loss at design temperature, so the rating *is* an estimate of the fabric:

```
heat loss coefficient (W/K)  ≈  nameplate output (W) ÷ (setpoint − design outside temp)
```

An 8 kW unit at 21 °C indoor against a −3 °C design day is a ~24 K ΔT ⟹ **~333 W/K**. One question a homeowner can genuinely answer — *it's on the box, on the quote, on the paperwork* — yields the entire heat demand model, where *"what's your heat loss coefficient?"* yields nothing and *"what's your EPC?"* yields very little.

**Flow temperature comes from one more askable question:** *radiators or underfloor?* — underfloor ~35 °C, radiators ~45–50 °C, and the difference is worth roughly a full COP point.

**Three honesty caveats to carry:**
- **Installers oversize**, sometimes substantially ⟹ inflates derived heat loss ⟹ **overstates demand and cost**. Conservative.
- **Some units are sized to cover hot water too**, inflating it further.
- **Defrost cycles** in cold, humid conditions push real COP below any published curve — precisely the winter condition this tool exists to model. Apply a deliberate derate around 0–5 °C rather than pretending the curve holds.

Not a full SAP assessment. The defensible minimum:

```
heat demand  =  heat loss coefficient (W/K) × degree-hours (from the same weather feed)
electrical draw  =  heat demand ÷ COP(outside temperature)
```

- **COP against outside temperature is essential, not a refinement.** A fixed COP makes winter look far too cheap and quietly breaks the headline number — COP is worst exactly when demand is highest.
- **Fabric input stays light** — derived from the heat pump nameplate as above, with an advanced override (direct W/K, direct COP curve) for people who know their numbers.
- **Thermal storage is the main non-battery flexibility in winter** — hot water tank and fabric pre-heating. The immersion-versus-export question only makes sense if they're modelled, so they are in scope.

## Data sources (verify terms before depending on any of them)
- **`postcodes.io`** — postcode → lat/lon. Free, open, UK, no key.
- **Open-Meteo historical archive** — hourly ERA5 reanalysis back to 1940: `shortwave/direct/diffuse_radiation` **and temperature in one call**. The primary source, and the correlation matters: a cold dull January day must be cold *and* dull in both series, because that's the case the economics turn on.
- **PVGIS** — PV-specific output for a typical year or for actual historical years, with tilt/azimuth/system-loss parameters.
- **NASA POWER** — free global hourly fallback, no key.
- **Caching is mandatory, and it is also the whole answer to rate limits.** Historical weather is **static** — a postcode-year never changes, so it is fetched once, ever.
  - **Cache by grid cell, not by postcode.** ERA5 is ~30 km resolution, so hundreds of postcodes resolve to the same underlying data. Rounding lat/lon to ~0.25° puts mainland UK in a few hundred cells; twenty years of *daily* data across all of them is trivial to store and collapses the fetch count to near nothing. Hourly, pulled only for years a user actually selects, stays modest.
  - **Pre-bake a handful of locations for the demo** so the dependency is off the critical path entirely.
  - **Server-side fetch on cache miss, with backoff**, covers arbitrary postcodes as the secondary path.
- **Rejected: client-side fetch with upload to the server.** It would distribute rate limits across user IPs, but it means **simulating on data the client can fabricate** — untrusted input treated as canonical. The standing line applies: *the client gets the vocabulary, the server owns the verdict*, and weather is an input to the verdict. It also solves a problem that caching has already solved.
- *Licensing:* free tiers are frequently non-commercial and a publicly-hosted demo site is a grey area. Resolve it deliberately before launch.

## Flagship outputs
The things the tool can say that a homeowner cannot work out and a vendor will not tell them:

- **"Don't buy the second battery."** *Window is 5 hours; after two cars and the heat pump there's 6 kW of headroom — 30 kWh. The existing battery takes 10. A second one can be filled, but only by dropping a car to 3.5 kW so it doesn't finish on cheap rate. Battery saves £180, car costs £240. No.*
- **"Your second battery earns nothing for five months and pays for itself in the dark ones."**
- **"Upgrading your supply costs £X and unlocks £Y a year."**
- **"This strategy is optimal in a typical winter and costs you £400 more in a cold one."**
- **The export-price pivot** — one slider, and the whole dispatch reorganises: when export pays, exporting surplus is fine and dull; when it collapses, every self-consumed kWh is worth the avoided import price, and the immersion element, pre-heated hot water and a bigger battery all become rational at once. Visually legible in five seconds, and the question the market is anxious about.

## Modelling essentials that decide credibility
Cheap to design in, expensive to retrofit:
- **Battery degradation must carry a cost per cycle.** Without it the optimiser cycles the battery to death chasing 2p arbitrage. This single term separates a credible model from a toy.
- **COP as a curve against outside temperature** (above).
- **Round-trip efficiency (~85–90%) and standing losses** — small terms that decide whether marginal strategies actually pay.
- **The baseline determines the headline.** "Saves £X" against *no battery* and against *dumb charging* are wildly different numbers. Choose deliberately and always show which.

## Architecture — thin client, engine on the server (decided)
**`ballast` runs server-side in C#; the client is thin.** The client gets the *vocabulary* — `f8n` primitives for live formatting, validation and chart rendering — and nothing else. It never holds the engine.
- **Why:** business rules implemented in the front end are re-implemented by every subsequent client (a Swift app would start from zero), are harder to test properly, and — as client code — are untrusted by construction, so anything enforced only there isn't enforced at all. Written once, server-side, in the language the rest of the stack's business logic uses.
- **The line:** *the client gets the vocabulary, the server owns the verdict.* Codegen gives the client enough for good UX (instant formatting, responsive charts, optimistic display); anything that decides something defers to the HTTP round trip. This is also what makes an optimistic client *safe* — client and server can't disagree about the primitives, because both are generated from one source and proven identical by golden vectors.
- **Why C# server-side, when this is a *calculation* rather than a verdict (revisited 2026-08-23).** The strongest argument for server-side execution — *client code is untrusted* — **does not apply here**: nothing is enforced, no money moves, and the user is modelling their own house. `ballast` is advisory, so on principle it could live anywhere, and the placement is a **performance, cost and stack-coherence** decision rather than a doctrinal one. It stays in C# because: **`f8n`'s C# half would otherwise have no consumer in the stack** (badly weakening the one-source-two-languages claim, which needs a demo on *both* sides); a year of hourly simulation ×20 for a sweep is uncertain on mid-range mobile, which matters for a consumer product; and simulation results are **deterministic and therefore cacheable**, so genuinely novel computation is rare and the server cost is likely small. **Revisit only with a measured number** — build it, time a year-run, and decide against data rather than worry. *(A TS engine shared with the client is the alternative if that number turns out bad; the offline-first shape that motivates client-side computation is a different problem from this one.)*
- **Not in tension with `l10n`'s "no API round-trip".** That claim is about *presentation* (strings, formatting, on-device); this one is about *decision*. Different axes.
- **Consequence:** ampersand is a genuinely deployed full-stack app (C# API + thin Vue client), not a static toy — scenario solving is an API call. Hosting follows: containerised API on a scale-to-zero container PaaS, static client on free hosting.

## Interactivity without leaking the engine into the client
Sliders want instant feedback; the engine lives on the server. That tension is exactly where thin clients rot — *"just this one calculation, so the slider feels nice"* is how a dispatch model ends up in the browser one function at a time. **Classify the controls; don't compromise the line.**

**Structural changes** — add a battery, change kWp, change supply size, swap tariff. These alter the answer fundamentally, a moment's work is expected, and a **round trip with a proper loading state** is perfectly good UX.

**Hero sliders** — export price above all — need to feel instant, and they have the property that rescues them: **they are one-dimensional.** So the server **precomputes the sweep** — simulate at ~20 points across the range and ship the whole set — and the client **scrubs between server-computed answers**. Instant, smooth, gives the morph animation, and *the client computes nothing*: it selects and interpolates between results the server produced.

> **The line, stated so it's testable: the client may interpolate *between* server-supplied answers. It may never compute one.**

That is a rule that holds in code review, and it is the difference between a responsive thin client and a leaky one.

- **Sweep at low resolution, detail on demand.** Twenty full-year hourly datasets is a heavy payload; ship the sweep as **daily buckets plus totals** (all the year view and the money figures need while dragging) and fetch **hourly only for the selected point** when the user zooms into a day. Same two-resolution principle as the weather data.
- **Cost is manageable and cacheable.** A ~200 ms year-simulation makes a 20-point sweep a few seconds — fine as an initial load or a background job — and identical inputs give identical results, so it caches like anything else.
- **Forbidden outright: any local approximation of dispatch to make the UI feel quicker.** That is the leak, and it is worse than a slow slider, because the approximation will disagree with the server in precisely the marginal cases the tool exists to get right.

## Infrastructure (assumed Hetzner for now)
**One small VM running Docker Compose. Three containers: Caddy · API (.NET) · Postgres.**

**Why self-hosted rather than a container PaaS.** Compute at this traffic is nearly free anywhere; **the managed database is what costs money** (~£7–15/mo floor), and the data need has gone past "just a cache", so a database is required. A single VM carries all three services for roughly the price of the database alone, with a **fixed, predictable bill** and **no cold start** — which matters disproportionately here, because a scale-to-zero container's first request is exactly the one a reviewer makes.

**And the choice is reversible, so it isn't worth agonising over.** A clean Dockerfile plus a build-and-push pipeline makes the host a *deployment target*, not an architecture. Moving to a managed container platform later is a change of target and a connection string, not a rebuild.

### Sizing — the driver here is CPU, not RAM
Ordinary shape: OS ~200–300 MB · .NET API idle ~100–150 MB · Postgres ~250–500 MB tuned small · Caddy ~30 MB. **1 GB is tight, 2 GB comfortable, 4 GB generous.** Disk is a non-issue (daily weather across a few hundred grid cells is trivial; hourly for selected years is a few GB) — 40 GB is plenty.

**But this is a compute-bound app, not a CRUD one.** A 20-point sweep of year-long hourly simulations is real work, and the sizing question is *how long one year-simulation takes* — which is unknown until `ballast` exists. So: **start around 2 vCPU / 4 GB, measure early, resize.** A VM resize is a reboot, not a migration. Watch **shared-vCPU throttling** under sustained simulation load; the dedicated-vCPU tiers cost meaningfully more and may become necessary.

### TLS — Caddy, not nginx + certbot
Automatic Let's Encrypt issuance **and renewal**, on by default, from about three lines of config — plus HTTP→HTTPS redirect, HTTP/2 and HTTP/3, and modern TLS defaults. The nginx equivalent is ~30–40 lines across several files *plus* certbot *plus* a renewal timer *plus* a reload hook.

The point isn't the line count, it's the failure mode it removes: **certificate renewal is the classic thing that silently breaks a side project months later**, when nobody is watching — quite possibly in the week someone is looking at it.

### Deploy — pipeline, not SSH
**Actions builds the image → pushes to a registry (ghcr.io) → the box pulls it.** Never `ssh` + `docker run`. This is what preserves both the reversibility above and the CI/CD story; it is the same pipeline a managed platform would use, minus the final step.

### Data strategy — the database is disposable by construction
Split by **what is expensive to recreate**:

| Data | Value | Recovery |
|---|---|---|
| Weather cache (by grid cell) | none | re-fetch |
| Simulation results (by input hash) | none | re-run |
| **Demo scenario definitions** | **real — hand-authored** | **seeded from the repo** |
| Schema / migrations | none | in the repo |

**The only valuable data is the part that was authored, so it lives in the repo as version-controlled seed data applied on startup.** Total database loss then costs a redeploy, not a recovery — which is stronger than any backup schedule, because manually-triggered backups happen twice and then never again. It also makes **local dev and production identical from a data standpoint**, so cloning the repo yields a working system with one command.

- **Take the near-free safety net anyway:** provider-level automated VM backups (~20% of server cost — around €1/mo) cover the *"fat-fingered a migration at 11pm"* case with zero effort and nothing to remember.
- **The trigger that inverts this:** the moment users can save their own scenarios, that is real data and proper backups become necessary. Know the line rather than discovering it afterwards.

### Hardening — the short list that matters
- **Postgres is never published to the host.** Internal Compose network only; no `ports:` mapping. (The common and costly mistake.)
- **Firewall: 80/443 only**, plus SSH restricted.
- **`unattended-upgrades`** for OS patching, so the box doesn't rot.
- **Secrets out of the repo** — env file on the box, or injected at deploy from Actions secrets.
- **An uptime check** (a free tier is fine) — a demo that must stay presentable for a year should not be able to fall over silently.

## North star (to firm up)
1. **Consumer-legible** — a homeowner gets it in seconds; the depth is underneath (turtles-all-the-way-down when they dig).
2. **Real engine, simulated I/O** — the brain that *would* drive a real HEMS, proven by simulation on synthetic + real open data (tariff APIs, solar profiles). No live hardware.
3. **Dogfoods the stack** — `f8n` (units/cost/money), `l10n`, `palette`/`lattice`/`etch` (dashboards + charts), `a11y`.

## Design agenda (open)
- **User & job-to-be-done** — homeowner-first (decided — relatable to a non-specialist audience); the installer/advisor view is a *deferred* second surface over the same core (shape the seam, defer the room).
- **Optimiser or strategy knobs?** — *"here's the optimal schedule and here's why"* is a stronger product and a harder build than user-selected strategies. Likely middle: optimiser by default, knobs exposed for people who want to fight it.
- ~~**v1 kit list**~~ **Resolved 2026-08-23:** solar + battery + heat pump + hot water/immersion **+ EV (1–2 cars)**. The EV is what makes the system interesting — see *EV charging*.
- **Visual system** — colour, marks, legend, the range control; specified against the dataviz guidance as its own pass.
- **The `ballast` engine** — the constraint + objective model; the strategies; conserving/priority allocation for power (echoes `f8n`'s `allocate` discipline — a scarce resource split to sum exactly to a budget).
- **Data** — tariff (Octopus Agile/Go), solar irradiance/generation profiles, device load models; synthetic vs real open data. *(Verify current tariff / DNO / G98–G99 specifics before relying on them — fast-moving.)*
- **Solver latency / UX** — a server round-trip per scenario: what's debounced, what's precomputed, what a "live-style dashboard" means when the engine is remote. (Scope the API surface so a what-if is one call, not many.)
- **The simulation** — day/year modelling; what to visualise (amp-budget spend, schedule timeline, solar/tariff overlays, battery state, scenario what-ifs).
- **Scope guard** — build the brain + simulation + UI; **no live device integration** (that's the sinkhole).

## Change log
- 2026-08-23: **C#-server-side placement re-examined and confirmed, with the reasoning corrected.** The *untrusted client* argument **does not apply** to a simulator — nothing is enforced and it is the user's own house — so `ballast` is a **calculation, not a verdict**, and its placement is a performance/cost/stack-coherence decision rather than a doctrinal one. Kept in C# on three grounds: **`f8n`'s C# half would otherwise have no consumer**, weakening the one-source-two-languages claim (which needs a demo on both sides); client-side year-simulation ×20 is uncertain on mid-range mobile, which matters for a consumer product; and results are **deterministic and cacheable**, so novel computation is rare and server cost is probably small. **Revisit only with a measured year-run**, not on worry. A shared-TS engine is the named alternative if that number is bad; offline-first (which genuinely requires local computation) is a different problem shape.
- 2026-08-23: **infrastructure written up (assuming Hetzner).** One small VM, Docker Compose, **three containers — Caddy · .NET API · Postgres**. Rationale for self-hosting over a container PaaS: compute is nearly free anywhere but **the managed database carries the cost floor**, the data need has outgrown "just a cache", and a scale-to-zero cold start lands on exactly the first request a reviewer makes; a single VM covers all three for about the price of the database alone, at a fixed predictable bill. **Reversible by construction** — a clean Dockerfile plus build-and-push makes the host a deployment target, not an architecture. **Sizing is CPU-driven, not RAM-driven** (compute-bound: a 20-point sweep of year-long hourly sims is real work), and the per-simulation cost is unknown until `ballast` exists ⟹ **start ~2 vCPU / 4 GB, measure, resize** (a reboot, not a migration); watch shared-vCPU throttling. **Caddy replaces nginx + certbot** — automatic issuance *and renewal* removes the category of failure where a cert silently expires months later. **Deploy is Actions → ghcr.io → pull, never SSH + docker run.** **Data strategy: the database is disposable by construction** — hand-authored demo scenarios are version-controlled seed data applied at startup, everything else is recreatable cache, so total loss costs a redeploy (stronger than a backup schedule, since manual backups drift) and local dev matches prod; provider VM backups (~€1/mo) taken anyway as a zero-effort net; **trigger recorded — user-saved scenarios invert this and require real backups**. Hardening short list: **Postgres never published to the host**, 80/443 + restricted SSH, `unattended-upgrades`, secrets out of the repo, and an uptime check so a long-lived demo can't fail silently.
- 2026-08-23: **interactivity settled without leaking the engine client-side.** Sliders want instant feedback and the engine is server-side — the classic route by which a thin client rots one convenience function at a time. Resolved by **classifying controls**: *structural* changes (battery, kWp, supply size, tariff) round-trip with a loading state; *hero sliders* (export price above all) are **one-dimensional**, so the server **precomputes a ~20-point sweep** and the client **scrubs between server-computed answers** — instant, smooth, morph animation intact, and the client still computes nothing. **Line recorded as testable:** *the client may interpolate **between** server-supplied answers; it may never compute one.* Sweep ships at **daily resolution plus totals** with **hourly fetched only for the selected point** (same two-resolution principle as the weather data); cost is a few seconds of background work and caches like anything else. **Explicitly forbidden: local approximation of dispatch to make the UI feel quicker** — worse than a slow slider, because the approximation disagrees with the server in exactly the marginal cases the tool exists to get right.
- 2026-08-23: **rendering settled (SVG + animation) and the fetch architecture decided.** *Rendering* — **SVG over canvas** (token-styleable, ARIA-carrying, natively animatable; canvas only wins at far higher densities), with the one caveat that a year of hourly data must render as **one `<path>` per series, never an element per point**. Two animations with distinct jobs: **state transitions** (slider → curve morphs; the one that sells the product, and the thing a design-minded reviewer fixates on — `d`-interpolation with matched point counts) and a **playhead sweep** through a simulated day (cheap, demo-effective, explains the mechanism). Accessibility treated as demonstration rather than obligation: `prefers-reduced-motion`, a non-visual data representation, announced state changes — an animated chart fully usable without the animation *is* the inclusive-by-construction claim, shown. *Fetch* — **rejected client-side fetch with upload**: it would distribute rate limits but means simulating on data the client can fabricate (untrusted input as canonical), against the standing *client gets the vocabulary, server owns the verdict* line — and it solves a problem caching already solves. **Cache by grid cell (~0.25°), not by postcode** — ERA5's ~30 km resolution means hundreds of postcodes share data, putting mainland UK in a few hundred cells; daily across the whole record, hourly only for selected years; pre-baked demo locations; server-side fetch with backoff on miss.
- 2026-08-23: **v1 weather selection is a year picker over *real* years; synthetics deferred.** Real years avoid splicing, month-boundary discontinuities, colder-than-observed artefacts and the labelling burden entirely — everything happened, which fits the honesty position with no caveats. **Highlights computed from the data, per location** (*"2010 was **your** coldest winter"* beats a national claim), in two tiers: **weather-derived now** (heating degree days Nov–Mar, winter irradiance, brightest/warmest — cheap and pre-computable) and **outcome-derived later** (*"your worst year cost £340 more than typical"* — stronger, but needs a full simulation run per year). **Two data resolutions**: daily for ranking and labels across the whole record, hourly only for the selected year — large saving on fetch, storage and cache-warm. **Historic extremes (1963, 1981) grouped separately as stress tests** rather than mixed into recent years, since the climate has shifted and presenting them as peers would overstate their likelihood.
- 2026-08-23: **solar modelled as multiple arrays; battery capacity configurable.** Single-kWp cannot answer *"what if I add the garage roof?"* — that is a question about a **different roof plane**, so each array carries its own kWp, azimuth and tilt. **Ask orientation, derive tilt from house type**: at UK latitudes orientation dominates (E/W −15–20%, N −40%+ against south) while pitch barely matters (~5–8% across 20–50°), so the user's attention goes where it changes the answer. Records the **east/west insight** — less total energy, *more usable* energy, since output spreads either side of noon instead of spiking when the house is empty (large without a battery, mostly absorbed with one). **Inverter capacity separated from panel kWp with clipping modelled** — added panels can yield nothing on the sunny days they were bought for, and UK installs are frequently inverter-limited because **G98 allows ~3.68 kW/phase before a G99 application is needed**, making *"your inverter can't take it"* a legitimate and useful answer to the garage-roof question (DNO theme, second direction; thresholds to be verified). **Shading admitted as the largest single unknown** — qualitative per-array derate, plainly labelled; terrain shading is free from the irradiance source, local obstructions are not. Plus standard ~14% system losses (advanced-exposed) and ~0.5%/yr panel degradation in the payback arithmetic.
- 2026-08-23: **weekends modelled as a distinct regime; charging frequency becomes an output; heat pump nameplate becomes the fabric input.** *EV* — presence is now weekday-away / weekend-home, and weekends earn their place as **the one regime where the car can take solar** (sharpening the seasonal contrast). Replaced configured charging frequency with **state-of-charge simulation**: capacity, per-departure consumption, enough-at-every-departure constraint, charge whenever home — so *"you only need to charge Tuesday and Friday"* is a **result**, the optimiser gains real slack, and low-mileage/big-battery behaves differently from high-mileage/small-battery for free. *Heat model* — **ask for the heat pump's nameplate output as a proxy for the building** (installers size to calculated heat loss, so `HLC ≈ nameplate ÷ design ΔT`; an 8 kW unit ⟹ ~333 W/K): one answerable question yields the whole demand model where EPC or heat-loss questions yield nothing. Flow temp inferred from *radiators or underfloor* (~45–50 °C vs ~35 °C, worth ~a full COP point). Caveats carried: **installer oversizing** and **hot-water-inclusive sizing** both inflate derived heat loss (conservative), and **defrost cycles** push real COP below published curves in exactly the cold humid conditions that matter ⟹ deliberate derate around 0–5 °C. V2G note extended — it is a live purchase blocker today, and a scenario the tool should eventually answer.
- 2026-08-23: **EV charging confirmed for v1 (1–2 cars).** Presence model for MVP: away 08:00–18:00 daily — deliberately sharp, because it means **the commuter car never sees the solar window**, so on weekdays it is a pure grid load bidding for the cheap-rate window (and yields the counter-intuitive output that *more panels do nothing for your car*). Demand configurable per car in kWh/day, **input as miles/day** with a mi/kWh conversion. **Charge rate is a per-car maximum the optimiser may go below** — modulated charging is essential, since all-or-nothing 7 kW leaves `ballast` almost no options and removes the trade-offs that make the answers interesting. Deadline: enough charge by 08:00, which is what makes the window genuinely scarce with two cars. Simplifications recorded **with direction of error** (weekends treated as weekdays ⟹ overstates grid dependence by ~2/7 of car demand — conservative, but stated), and the hard deadline flagged as a shaped seam. **Default scenario must be feasible** so the user discovers the cliff by adjusting rather than meeting a broken-looking state. **V2G named as a seam and deferred** — a 50–80 kWh car battery dwarfs any home battery, so general V2G availability would change every answer.
- 2026-08-23: **product reframed — an energy *economics* sandbox, not a fuse calculator.** The headline is money (when to charge/discharge, what to do with surplus, immersion vs export vs heat pump, what happens when export price collapses), with the flagship question *"is another battery / more panels worth it?"* **Centrepiece settled: one chart, one Y axis (kW), continuous zoom from a day to a year** — long ranges are average power per bucket, area is always energy, totals beneath are money and follow the zoom. **Winter is the discovery, not the framing** — show the year whole and let the shape reveal itself; the battery's job *changes* in the dead spot from solar-shifting to tariff arbitrage, so the answers are seasonal. **The fuse is rehabilitated as an economic constraint**: savings are capped by absorbable cheap energy = window length × headroom, everything bids for the same short window (2 EVs ≈ 14 kW + heat pump 4–5 kW + inverters vs ~23 kW at 100 A), which makes supply size a primary input, the DNO upgrade an answerable capex question, and the optimiser genuinely necessary. `ballast` keeps its name — current limiting is what decides whether capex pays back. Added **weather years as a first-class control** (real named years back to 1940, plus TMY-style synthetic cold/average/warm composites — with the boundary-discontinuity and colder-than-observed wrinkles flagged, and synthetic labelled as such), **robustness across years rather than optimisation for one**, and **report a range not a number**. Added the **heat model as critical path** (heat-loss coefficient × degree-hours ÷ COP-versus-temperature; light fabric input; thermal storage in scope), **data sources** (postcodes.io · Open-Meteo ERA5 for radiation *and* temperature in one call · PVGIS · NASA POWER; caching mandatory; licensing to resolve), **flagship outputs**, and **modelling essentials** (degradation cost per cycle, COP curve, round-trip and standing losses, explicit baseline).
- 2026-08-22: **architecture decided — `ballast` is server-side C#, the client is thin.** The client carries `f8n` vocabulary only (formatting/validation/charts); every decision defers to the API. Rationale: written-once (the next client doesn't reinvent), properly testable, and client code is untrusted by construction. Recorded the line — *client gets the vocabulary, server owns the verdict* — and why it does not contradict `l10n`'s on-device "no API round-trip" claim (presentation vs decision). Added solver-latency/UX to the open agenda.
- 2026-08-22: created — scaffolded as a forge project dir (top-of-stack consumer product). **Energy** domain, **consumer-focused**, **visual** (dashboards and timelines). Constraint-solver engine named `ballast` (current-limiting — the mechanism is the name). Agenda open; scope held to brain + simulation + UI (no live hardware).
