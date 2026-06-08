# CHANGES — Datastar LLM Guide reconciliation

Reconciliation of the community skeleton (njreid Gist, last updated 2025-09) against
the official reference (`/reference/attributes`, `/reference/actions`,
`/reference/sse_events`, `/guide/getting_started`) and the v1.0.0–v1.0.2 GitHub
release notes. Target: **v1.0.2**.

Status legend: **VERIFIED** (skeleton matched official) · **FIXED** (skeleton error
or old syntax corrected to official) · **ADDED** (in official, missing from skeleton)
· **REMOVED** (in skeleton, not in official) · **UNCERTAIN** (could not confirm
against official — needs human/runtime check).

---

## ✅ Resolved by direct inspection of the v1.0.2 bundle

These were confirmed against the shipped bundle (`web/static/js/datastar.js`,
v1.0.2) — implementation source, no summarization — so they are **no longer
uncertain**. Evidence: `grep` of `name:"..."` plugin names and of the `__self` string.

1. **`data-on` `__self`** — RESOLVED. The string `__self` occurs **exactly once** in
   the bundle, inside the `data-ignore` implementation
   (`hasAttribute(\`${Dt}__self\`)` with `Dt="ignore"`). **`data-on` has no `__self`.**
   (Task brief point C was wrong on this.) Guide now documents this with a core
   alternative (`evt.target === evt.currentTarget`).

2. **Plugin-event attribute naming** — RESOLVED. The registered plugin names are
   literally `on-intersect`, `on-interval`, `on-signal-patch`, `init` — **dash form**.
   No colon form exists; the guide's dash naming is correct. (Task brief example
   `data-on:intersect` was wrong.)

3. **`data-on-load` vs `data-init`** — RESOLVED. The bundle registers a plugin named
   `init` and has **no** `load`/`on-load`. `data-init` is correct; `data-on-load`
   does not exist in v1.0.2.

Full core plugin list found in the bundle: `attr, bind, class, computed, effect,
indicator, init, json-signals, on, on-intersect, on-interval, on-signal-patch, peek,
ref, setAll, show, signals, style, text, toggleAll` + SSE `datastar-patch-elements`,
`datastar-patch-signals`. (Pro plugins and `ignore`/`ignore-morph`/`preserve-attr`
are registered separately and not in this `name:` list.)

4. **`data-on-interval` `__duration`** — RESOLVED by reading the bundle: the default
   interval is **1000ms (1s)**; `__duration.<time>` sets the interval; appending
   `.leading` runs the expression once immediately on attach (plain `setInterval`
   otherwise).

## ✅ Verified on real runtime (Claude in Chrome, /datastar-test, APP_ENV=dev)

A reference page (`web/components/datastar_reference.templ`, served at `/datastar-test`
in dev only) exercises the core features. Observed against v1.0.2:

- §1 bind/text, §2 `data-on:click`, §8 `data-computed`/`data-show`/`data-class` — all work.
- §3 `data-on-interval`: the `.leading` counter stays exactly +1 ahead of the plain one
  → confirms **immediate first run**; default interval 1s.
- §4 clicking the inner button increments the `__self` parent (so `__self` is a **no-op**
  on `data-on`) while the `evt.target===evt.currentTarget` alternative does **not** →
  confirms `data-on` has no `__self`.
- §5 `data-init`="ran", `data-on-load`="not-run" → confirms `data-on-load` is gone.
- §6 `data-on-intersect`(dash)="fired", `data-on:intersect`(colon)="no" → confirms
  dash-only naming.
- §7 checkbox `data-bind` reflects immediately (v1.0.2 `input` default).

**New pitfall found during verification:** attribute keys are **lowercased** by the HTML
parser, so `data-signals:fooBar` creates `foobar`, not `fooBar`. Documented in guide §11.

## ⚠️ Still UNCERTAIN

None. All four earlier UNCERTAINs were resolved against the v1.0.2 bundle **and**
confirmed on real runtime. (Per the guide's header, the official reference and real
runtime remain the source of truth.)

---

## Mandatory corrections (task points A–D)

### A. Syntax modernization — **FIXED** (whole document)
All dash-form usages converted to colon form; both forms noted exactly once (guide §1).

| Skeleton (old) | Guide (current) | Status |
| --- | --- | --- |
| `data-on-click` | `data-on:click` | FIXED |
| `data-bind-foo` | `data-bind:foo` | FIXED |
| `data-class-active` | `data-class:active` | FIXED |
| `data-computed-isValid` | `data-computed:isValid` | FIXED |
| `data-attr-title` | `data-attr:title` | FIXED |
| `data-signals-foo` | `data-signals:foo` | FIXED |
| `data-indicator-fetching` | `data-indicator:loading` (example) | FIXED |

> Note: dash-named **plugin attributes** (`data-on-intersect`, `data-init`, …) are
> their own attribute names and stay dashed — see UNCERTAIN #2.

### B. `data-bind` modifiers — **ADDED** (absent from skeleton)
- `__event.<eventName>` — choose which event(s) write back to the signal. **ADDED**
- `__prop.<propertyName>` — bind through a specific DOM property; argument converted
  to camelCase (v1.0.1). **ADDED**
- Independent use of `__prop`/`__event` (v1.0.1) documented. **ADDED**
- `__case` on `data-bind`. **ADDED**

### C. `data-on` modifiers — **ADDED** (skeleton had only `__debounce`)
| Modifier | Status |
| --- | --- |
| `__debounce` | VERIFIED |
| `__prevent`, `__stop`, `__once`, `__passive`, `__capture` | ADDED |
| `__window`, `__document` (v1.0.0), `__outside` | ADDED |
| `__throttle`, `__delay`, `__case`, `__viewtransition` | ADDED |
| `__self` | RESOLVED — belongs to `data-ignore`, not `data-on` |
Argument syntax (`__debounce.300ms`) and chaining
(`__window__debounce.500ms.leading`) documented. **ADDED**

### D. SSE wire format — **ADDED / VERIFIED**
- Event names `datastar-patch-elements` / `datastar-patch-signals` — VERIFIED
  (skeleton already used "patch", not "merge").
- `datastar-patch-elements` data keys `selector`, `mode`, `namespace`,
  `useViewTransition`, `viewTransitionSelector`, `elements` — **ADDED**
- `mode` values `outer`(default)/`inner`/`replace`/`prepend`/`append`/`before`/
  `after`/`remove` — **ADDED**
- `datastar-patch-signals` keys `onlyIfMissing`, `signals` (+ `null` to remove) — **ADDED**
- Raw on-the-wire examples — **ADDED**
- `viewTransitionSelector` — **ADDED** (v1.0.2)

---

## Attribute-by-attribute

### Core
| Attribute | Status | Note |
| --- | --- | --- |
| `data-signals` | FIXED | dash→colon; `__ifmissing`/`__case` documented |
| `data-computed` | FIXED | dash→colon |
| `data-bind` | FIXED | dash→colon; modifiers added (point B) |
| `data-text` | VERIFIED | |
| `data-show` | VERIFIED | `style="display:none"` pairing noted |
| `data-class` | FIXED | dash→colon |
| `data-attr` | FIXED | dash→colon |
| `data-style` | VERIFIED | |
| `data-on` | FIXED | dash→colon; modifiers added (point C) |
| `data-on-intersect` | VERIFIED | core; modifiers listed |
| `data-on-interval` | VERIFIED | core |
| `data-on-signal-patch` | ADDED | not in skeleton |
| `data-on-signal-patch-filter` | ADDED | not in skeleton |
| `data-init` | ADDED / FIXED | replaces skeleton's `data-on-load` (UNCERTAIN #3) |
| `data-ref` | FIXED | dash→colon |
| `data-indicator` | FIXED | dash→colon |
| `data-effect` | VERIFIED | |
| `data-ignore` | VERIFIED | `__self` modifier noted |
| `data-ignore-morph` | VERIFIED | clarified as its own attribute |
| `data-preserve-attr` | ADDED | not in skeleton |
| `data-json-signals` | ADDED | not in skeleton; `__terse` |
| `data-on-load` | REMOVED | not in current official reference → use `data-init` |

### Pro
| Attribute | Status | Note |
| --- | --- | --- |
| `data-persist` | VERIFIED | Pro classification correct |
| `data-query-string` | VERIFIED | |
| `data-replace-url` | VERIFIED | |
| `data-animate` | VERIFIED | |
| `data-view-transition` | VERIFIED | |
| `data-custom-validity` | VERIFIED | |
| `data-match-media` | ADDED | not in skeleton |
| `data-on-raf` | ADDED | not in skeleton |
| `data-on-resize` | ADDED | not in skeleton |
| `data-scroll-into-view` | VERIFIED | modifiers listed |

---

## Action-by-action

| Action | Tier | Status |
| --- | --- | --- |
| `@get` / `@post` / `@put` / `@patch` / `@delete` | core | VERIFIED (options listed; `retryMaxWait` rename noted) |
| `@setAll` | core | VERIFIED |
| `@toggleAll` | core | VERIFIED |
| `@peek` | core | VERIFIED |
| `@clipboard` | Pro | VERIFIED (code omitted per license) |
| `@fit` | Pro | VERIFIED (code omitted) |
| `@intl` | Pro | ADDED (code omitted) |

---

## Version notes added (verified against GitHub releases)

| Version | Item | Status |
| --- | --- | --- |
| v1.0.0 | `data-bind` `__prop`/`__event` added | VERIFIED |
| v1.0.0 | `data-on` `__document` added | VERIFIED |
| v1.0.0 | `datastar-prop-change` client event added | VERIFIED |
| v1.0.0 | `retryMaxWaitMs` → `retryMaxWait` (breaking) | VERIFIED |
| v1.0.1 | `__prop`/`__event` usable independently | VERIFIED |
| v1.0.1 | `__prop` argument → camelCase | VERIFIED |
| v1.0.1 | `select` bind signal type fixed (number → string) | VERIFIED |
| v1.0.2 | SSE `viewTransitionSelector` added | VERIFIED |
| v1.0.2 | same method+URL in-flight request auto-cancelled | VERIFIED |
| v1.0.2 | checkbox/radio default bind event → `input` | VERIFIED |
| v1.0.2 | checkbox-array bind fix; 5xx retry fix | VERIFIED |

---

## Coverage report

Counts are against the official reference pages for v1.0.2.

| Category | Covered / Total | UNCERTAIN |
| --- | --- | --- |
| Attributes (core 21 + Pro 10) | **31 / 31** | — |
| Actions (core 8 + Pro 3) | **11 / 11** | — |
| `data-on` modifiers | **13 / 13** documented | — (`__self` confirmed NOT a data-on modifier) |
| `data-bind` modifiers | **3 / 3** (`__event`, `__prop`, `__case`) | — |
| SSE events + data keys | **2 / 2** events, all keys | — |

**Totals:** 60 / 60 reference items covered. **UNCERTAIN: 0** — all four earlier
UNCERTAINs (`__self` ownership, plugin-event attribute naming, `data-on-load`→`data-init`,
`data-on-interval` `__duration`) were **resolved** by direct inspection of the v1.0.2
bundle. See top.

Pro plugin source is intentionally **not** included (license). `data-on-load` is the
only item **removed** (not present in the current official reference). Backend SDK
specifics (e.g. `datastar-go`) are out of scope for this front-end reference and were
not enumerated.
