# Datastar LLM Guide (v1.0.2)

> ⚠️ **Read this first.** This document is an **AI-assisted draft**. The Datastar
> author does **not** endorse AI-generated documentation. Always confirm against
> the official reference at <https://data-star.dev> and against real runtime
> behavior before relying on anything here.
>
> - **Target version:** v1.0.2
> - **CDN:** `https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.2/bundles/datastar.js`
> - Items that could **not** be verified against the official reference are marked
>   **`[UNCERTAIN]`** inline and collected at the top of `CHANGES.md`. Do not treat
>   them as fact.

This guide was built by taking a community LLM guide as a skeleton and reconciling
every entry, one at a time, against the official reference pages
(`/reference/attributes`, `/reference/actions`, `/reference/sse_events`,
`/guide/getting_started`). Where the skeleton and the official docs disagreed, the
official docs win.

---

## 0. Install

```html
<script type="module"
  src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.2/bundles/datastar.js"></script>
```

Self-hosting the bundle is recommended for production. With Go (or any backend),
serve the file as a static asset and add a cache-busting query (`?v=<build-hash>`).

---

## 1. Notation conventions (read once, applies everywhere)

**Colon (`:`) is the canonical key/modifier separator** used throughout this guide:
`data-on:click`, `data-bind:foo`, `data-class:active`, `data-computed:isValid`,
`data-attr:disabled`, `data-signals:count`.

> **Both forms exist.** The dash form (`data-on-click`, `data-bind-foo`, …) is also
> accepted and is what older guides use. This note is stated **once, here**; the rest
> of the document uses only the colon form. Dash-named *plugin attributes* such as
> `data-on-intersect` / `data-on-interval` / `data-init` are their **own attribute
> names** (not `data-on:` events) and are written with dashes — see §4.2.

**`$` usage — do not mix these up:**

- Inside **expressions** (the value of `data-text`, `data-show`, `data-class:*`,
  `data-on:*`, `data-computed:*`, `data-effect`, …) you reference a signal **with**
  `$`: `data-text="$message"`, `data-show="$count > 0"`.
- As the **target/name** of `data-bind`, `data-signals:`, `data-computed:`,
  `data-ref:`, `data-indicator:` you write the **bare signal name, without `$`**:
  `data-bind:foo`, `data-signals:count="0"`, `data-ref:el`.

**Event object in `data-on` expressions is `evt`** (not `$event`):
`data-on:input="$q = evt.target.value"`.

**Modifier ownership — never mix these:**

- `__event`, `__prop` belong to **`data-bind`** (§5).
- `__prevent`, `__stop`, `__once`, `__window`, `__document`, `__outside`,
  `__throttle`, `__delay`, `__debounce`, `__passive`, `__capture`, `__case`,
  `__viewtransition` belong to **`data-on`** (§6).
- Modifier arguments are appended with `.`: `__debounce.300ms`. Modifiers chain:
  `__window__debounce.500ms.leading`.

---

## 2. Core philosophy

Datastar is a **backend-driven, hypermedia** framework. The backend is the source
of truth: a frontend action issues an HTTP request, and the server replies with
**zero or more SSE events** that patch either DOM elements or signals. Frontend
reactivity is declared with `data-*` attributes over **signals** (reactive state,
referenced as `$name`). DOM updates use an ID-matching **morph** strategy.

Three response patterns (see §7 for wire format):

| Pattern | Backend Content-Type | Effect |
| --- | --- | --- |
| Patch elements | `text/html` (in an SSE stream) | morph/replace DOM via `datastar-patch-elements` |
| Patch signals | `application/json` (in an SSE stream) | update signals via `datastar-patch-signals` |
| Execute script | `text/javascript` | run script |

---

## 3. Signals (quick reference)

```html
<!-- define / initialize (bare names; objects may nest) -->
<div data-signals:count="0" data-signals:name="'John'"></div>
<div data-signals="{count: 0, form: {name: ''}}"></div>

<!-- read in expressions (with $) -->
<span data-text="$count"></span>
<div data-show="$name != ''"></div>
```

---

## 4. Attributes — Core `[core]`

Each entry: **name / one-line purpose / minimal example / tier**. Modifiers, where
relevant, are listed; `__case.camel|.kebab|.snake|.pascal` is available on most
key-based attributes and is not repeated every time.

### 4.1 State, display & binding

**`data-signals`** `[core]` — Define or patch signal values.
```html
<div data-signals:count="0"></div>
```
Modifiers: `__ifmissing` (only set if signal absent), `__case`.

**`data-computed`** `[core]` — Read-only signal derived from an expression.
```html
<div data-computed:sum="$x + $y"></div>
```

**`data-bind`** `[core]` — Two-way binding to a form element; preserves signal type.
The value is the **bare signal name** (no `$`).
```html
<input data-bind:foo />
```
Modifiers: `__event`, `__prop`, `__case` — see §5.

**`data-text`** `[core]` — Bind element text content to an expression.
```html
<div data-text="$message"></div>
```

**`data-show`** `[core]` — Show/hide the element based on an expression.
```html
<div data-show="$isVisible" style="display: none"></div>
```

**`data-class`** `[core]` — Conditionally toggle CSS classes.
```html
<div data-class:font-bold="$active"></div>
<div data-class="{active: $a, disabled: !$b}"></div>
```

**`data-attr`** `[core]` — Set HTML attribute values reactively.
```html
<button data-attr:disabled="$isBusy">Save</button>
```

**`data-style`** `[core]` — Set inline CSS properties reactively.
```html
<div data-style:display="$hidden ? 'none' : 'flex'"></div>
```

### 4.2 Events & lifecycle

**`data-on`** `[core]` — Attach an event listener. `evt` is the event object.
```html
<button data-on:click="$count++">+</button>
```
Modifiers: see §6.

**`data-on-intersect`** `[core]` — Run when the element enters/exits the viewport.
(Its own attribute name; dash form.)
```html
<div data-on-intersect__once="$loaded = true"></div>
```
Modifiers: `__once`, `__exit`, `__half`, `__full`, `__threshold.25|.75`, `__delay`,
`__debounce`, `__throttle`, `__viewtransition`.

**`data-on-interval`** `[core]` — Run an expression at a fixed interval.
```html
<div data-on-interval__duration.500ms="$count++"></div>
```
Modifier: `__duration.<time>` sets the interval (**default 1s** when omitted);
append `.leading` to also run once immediately on attach
(e.g. `__duration.500ms.leading`). Confirmed against the v1.0.2 bundle.

**`data-on-signal-patch`** `[core]` — Run when any signal changes.
```html
<div data-on-signal-patch="console.log('signals changed')"></div>
```
Modifiers: `__delay`, `__debounce`, `__throttle`.

**`data-on-signal-patch-filter`** `[core]` — Restrict which signals trigger
`data-on-signal-patch`.
```html
<div data-on-signal-patch-filter="{include: /^counter$/}"></div>
```

**`data-init`** `[core]` — Run an expression when the element is initialized.
(Replaces the old `data-on-load` from older guides. Confirmed against the v1.0.2
bundle: there is a plugin named `init` and no `load`/`on-load`.)
```html
<div data-init="$count = 1"></div>
<div data-init__delay.500ms="$ready = true"></div>
```
Modifiers: `__delay`, `__viewtransition`.

**`data-ref`** `[core]` — Create a signal that references the DOM element.
```html
<div data-ref:panel></div>
```

**`data-indicator`** `[core]` — Boolean signal that is true while a fetch is in
flight.
```html
<button data-on:click="@get('/api')" data-indicator:loading>Load</button>
<span data-show="$loading">…</span>
```

**`data-effect`** `[core]` — Run an expression on init and whenever its
dependencies change.
```html
<div data-effect="$total = $a + $b"></div>
```

### 4.3 Morphing & debugging

**`data-ignore`** `[core]` — Exclude the element (and, by default, descendants)
from Datastar processing.
```html
<div data-ignore><span>Not processed</span></div>
```
Modifier: `__self` (ignore this element only, still process descendants).

**`data-ignore-morph`** `[core]` — Skip this element during DOM morphing.
```html
<div data-ignore-morph>Preserved across patches</div>
```

**`data-preserve-attr`** `[core]` — Keep specific attribute values during morphing.
```html
<details open data-preserve-attr="open">…</details>
```

**`data-json-signals`** `[core]` — Render current signals as JSON (debugging).
```html
<pre data-json-signals></pre>
<pre data-json-signals="{include: /user/}"></pre>
```
Modifier: `__terse`.

---

## 5. `data-bind` modifiers (often missing from older guides)

These are modifiers of **`data-bind`**, not `data-on`. Since v1.0.1 they can be
used **independently** of each other.

**`__event.<eventName>`** — Choose which DOM event(s) write the value back into the
signal. Chainable for multiple events.
```html
<input data-bind:title__event.change />
<input data-bind:title__event.input.change />
```

**`__prop.<propertyName>`** — Bind through a specific DOM **property** instead of the
inferred native binding. Since v1.0.1 the property name is **converted to camelCase**.
```html
<input type="checkbox" data-bind:state__prop.indeterminate />
```

**`__case.camel|.kebab|.snake|.pascal`** — Control case conversion of the signal name.

> Default bind events by element type are framework-managed; note v1.0.2 changed the
> default event for **checkbox/radio** to `input` (see §8).

---

## 6. `data-on` modifiers (full list)

Arguments append with `.` (`__debounce.300ms`); modifiers chain
(`__window__debounce.500ms.leading`).

| Modifier | Meaning | Minimal example |
| --- | --- | --- |
| `__prevent` | `evt.preventDefault()` | `<form data-on:submit__prevent="@post('/x')">` |
| `__stop` | `evt.stopPropagation()` | `<a data-on:click__stop="$open=true">` |
| `__once` | Fire at most once | `<div data-on:click__once="$seen=true">` |
| `__passive` | Passive listener | `<div data-on:scroll__passive="$y=evt.target.scrollTop">` |
| `__capture` | Capture phase | `<div data-on:click__capture="…">` |
| `__window` | Listen on `window` | `<div data-on:resize__window="$w=window.innerWidth">` |
| `__document` | Listen on `document` (added v1.0.0) | `<div data-on:keydown__document="…">` |
| `__outside` | Fire when event occurs outside the element | `<div data-on:click__outside="$open=false">` |
| `__debounce.<t>` | Debounce; `.leading`, `.notrailing` | `<input data-on:input__debounce.300ms="@get('/s')">` |
| `__throttle.<t>` | Throttle; `.noleading`, `.trailing` | `<div data-on:scroll__throttle.200ms="…">` |
| `__delay.<t>` | Delay execution | `<div data-on:click__delay.500ms="…">` |
| `__case.<style>` | Case-convert the event name | `<div data-on:my-event__case.camel="…">` |
| `__viewtransition` | Wrap handler in a View Transition | `<div data-on:click__viewtransition="…">` |

> **There is no `__self` on `data-on`.** Confirmed against the v1.0.2 bundle: the
> string `__self` appears exactly once, inside the `data-ignore` implementation
> (`data-ignore__self`). For "only when the event target is the element itself",
> compare in the expression: `data-on:click="evt.target === evt.currentTarget && (…)"`.

---

## 7. SSE wire format (server → client)

Each event ends with a blank line (two `\n`). Keys are written as
`data: <key> <value>` lines.

### `datastar-patch-elements`

Data-line keys:

- `selector` — CSS selector for the target (optional; not needed for `outer`/`replace`).
- `mode` — one of `outer` (default), `inner`, `replace`, `prepend`, `append`,
  `before`, `after`, `remove`.
- `namespace` — `svg` or `mathml`.
- `useViewTransition` — `true`/`false` (default `false`).
- `viewTransitionSelector` — CSS selector scoping the view transition **(added v1.0.2)**.
- `elements` — HTML payload (may span multiple `data: elements` lines).

```
event: datastar-patch-elements
data: elements <div id="foo">Hello world!</div>

```

```
event: datastar-patch-elements
data: selector #foo
data: mode inner
data: useViewTransition true
data: viewTransitionSelector #main
data: elements <div>
data: elements   Hello world!
data: elements </div>

```

```
event: datastar-patch-elements
data: selector #foo
data: mode remove

```

### `datastar-patch-signals`

Data-line keys:

- `onlyIfMissing` — `true`/`false` (default `false`): only set signals that don't exist.
- `signals` — a valid `data-signals` value (object). Set a signal to `null` to remove it.

```
event: datastar-patch-signals
data: signals {foo: 1, bar: 2}

```

```
event: datastar-patch-signals
data: onlyIfMissing true
data: signals {foo: 1, bar: 2}

```

> Backend SDKs (e.g. `datastar-go`) generate these events for you; you rarely write
> the raw bytes by hand, but the keys above are what the SDK options map to.

---

## 8. Version differences (v1.0.0 → v1.0.2)

**v1.0.0**
- `data-bind` modifiers `__prop` and `__event` **added**.
- `data-on` modifier `__document` **added**.
- New client event `datastar-prop-change` emitted when properties change during morph.
- **Breaking:** backend option `retryMaxWaitMs` **renamed** to `retryMaxWait`.
- Fixes: `type="submit"` inputs include their value on submit; `__viewtransition`
  no longer interferes with other modifiers; radio `checked` respected on bind;
  morphing improved for input/select/textarea.

**v1.0.1**
- `__prop` / `__event` can now be used **independently** of each other.
- The `__prop` argument is now **converted to camelCase**.
- Fix: `data-bind` on a `select` with no initial value set the signal type to a
  number; it is now correctly a **string**.

**v1.0.2**
- SSE: `viewTransitionSelector` **added** to `datastar-patch-elements`.
- An in-flight fetch with the **same method + URL** is now auto-cancelled when a new
  one starts.
- **Behavior change:** the default `data-bind` event for **checkbox/radio** changed
  to `input` (faster reactivity; not a breaking change).
- Fixes: `data-bind` with modifiers misbehaved for checkboxes when the signal was an
  array; fetch retry now correctly retries after 5xx responses.

---

## 9. Actions — `@`

### Core `[core]`

**Backend (HTTP) actions** — `@get` / `@post` / `@put` / `@patch` / `@delete`.
Send a request; the response is an SSE stream of patch events (or is auto-handled by
Content-Type: `text/event-stream`→SSE, `text/html`→DOM patch,
`application/json`→signal patch, `text/javascript`→script).
```html
<button data-on:click="@get('/endpoint')">Load</button>
```
Signature: `@get(uri: string, options?)`. Options include: `contentType`,
`filterSignals`, `selector`, `headers`, `openWhenHidden`, `retry`, `retryInterval`,
`retryScaler`, `retryMaxWait`, `retryMaxCount`, `requestCancellation`.
(Note `retryMaxWait` — renamed from `retryMaxWaitMs` in v1.0.0.)

**`@setAll(value, filter?)`** `[core]` — Set all matching signals.
```html
<button data-on:click="@setAll(true, {include: /^todo\./})">Check all</button>
```

**`@toggleAll(filter?)`** `[core]` — Toggle all matching boolean signals.
```html
<button data-on:click="@toggleAll({include: /^todo\./})">Toggle all</button>
```

**`@peek(callable)`** `[core]` — Read a signal without creating a reactive
subscription.
```html
<div data-text="$foo + @peek(() => $bar)"></div>
```

### Pro `[Pro]` (paid plugins — code intentionally omitted)

Per Datastar's license, Pro plugin source is not reproduced here. Use the official
docs and consider the core alternative.

- **`@clipboard(text, isBase64?)`** `[Pro]` — Copy text to the clipboard.
  *Core alternative:* call `navigator.clipboard.writeText(...)` inside a `data-on:click`.
  Docs: <https://data-star.dev/reference/actions>
- **`@fit(v, oldMin, oldMax, newMin, newMax, clamp?, round?)`** `[Pro]` — Linear range
  remap. *Core alternative:* compute the same formula in a `data-computed`.
  Docs: <https://data-star.dev/reference/actions>
- **`@intl(type, value, options?, locale?)`** `[Pro]` — Locale-aware formatting
  (`datetime`, `number`, `pluralRules`, `relativeTime`, `list`, `displayNames`).
  *Core alternative:* compute via the native `Intl` API in a `data-computed`.
  Docs: <https://data-star.dev/reference/actions>

---

## 10. Attributes — Pro `[Pro]` (code intentionally omitted)

Descriptions + core alternatives + official link only.

- **`data-persist`** `[Pro]` — Persist signals to local/session storage
  (modifier `__session`). *Core alt:* `data-on-signal-patch` writing to
  `localStorage`, restore in `data-init`.
- **`data-query-string`** `[Pro]` — Sync signals to URL query params.
  *Core alt:* read/write `location.search` in `data-init` / `data-on-signal-patch`.
- **`data-replace-url`** `[Pro]` — Replace the browser URL without reload.
  *Core alt:* `history.replaceState(...)` in an expression.
- **`data-animate`** `[Pro]` — Animate element attributes over time.
  *Core alt:* CSS transitions/animations driven by `data-class`/`data-style`.
- **`data-view-transition`** `[Pro]` — Set the View Transition API name.
  *Core alt:* the `__viewtransition` modifier on `data-on` and `useViewTransition`
  on SSE patches.
- **`data-custom-validity`** `[Pro]` — Reactive HTML5 validation message.
  *Core alt:* `el.setCustomValidity(...)` in a `data-on:input` handler.
- **`data-match-media`** `[Pro]` — Sync a CSS media query result to a signal.
  *Core alt:* `matchMedia(...)` listener wired in `data-init`.
- **`data-on-raf`** `[Pro]` — Run every `requestAnimationFrame` (modifier `__throttle`).
- **`data-on-resize`** `[Pro]` — Run when element dimensions change
  (modifiers `__debounce`, `__throttle`).
- **`data-scroll-into-view`** `[Pro]` — Scroll the element into view (modifiers
  `__smooth`, `__instant`, `__auto`, `__hstart|hcenter|hend|hnearest`,
  `__vstart|vcenter|vend|vnearest`, `__focus`).
  *Core alt:* `el.scrollIntoView({...})` in an expression.

Official reference: <https://data-star.dev/reference/attributes>

---

## 11. Common pitfalls

- **`$` placement:** `data-bind:foo` (bare) vs `data-text="$foo"` (with `$`).
- **Attribute keys are lowercased by the HTML parser.** `data-signals:fooBar` /
  `data-bind:fooBar` become `foobar`, which will **not** match `$fooBar` in
  expressions (expressions preserve case). Use lowercase signal names, or kebab-case
  with `__case.camel`. (Verified on real runtime: a camelCase key silently created a
  separate lowercased signal alongside the camelCase one used in the expression.)
- **Event object is `evt`**, not `$event`.
- **Modifier ownership:** `__event`/`__prop` → `data-bind`; `__prevent`/`__stop`/
  `__once`/`__window`/`__debounce`/… → `data-on`.
- **`data-on-intersect`/`-interval`/`-signal-patch`/`data-init`** are distinct
  attribute names (dash form), not `data-on:<event>`.
- **`__outside` placement:** put `data-on:click__outside` on the **outer container
  that includes the toggle button**, not on the inner panel. If it's on the inner
  panel, the toggle's own click counts as "outside" and closes the panel the instant
  it opens. (Found via real runtime in recipe §8 at `/datastar/recipes`.)
- **`data-show`**: pair with `style="display: none"` to avoid a flash before hydration.
- **Pro vs core:** `data-persist`, `data-query-string`, `data-replace-url`,
  `data-animate`, `data-view-transition`, `data-custom-validity`, `data-match-media`,
  `data-on-raf`, `data-on-resize`, `data-scroll-into-view`, `@clipboard`, `@fit`,
  `@intl` are **Pro**. Everything else above is **core**.
- **Attribute key case** (repeat, because it bites): `data-signals:fooBar` →
  `foobar`. Keep signal names lowercase.

---

## 12. Backend SSE implementation (datastar-go)

Datastar is backend-driven: the `@get`/`@post`/… actions hit your server and the
server replies with SSE events that patch elements or signals (the wire format in §7).
In Go, use `github.com/starfederation/datastar-go/datastar` (verified against
`datastar-go v1.2.1`):

```go
import "github.com/starfederation/datastar-go/datastar"

func handler(w http.ResponseWriter, r *http.Request) {
    // 1. read signals the client sent (from data-bind / data-signals)
    var sig struct {
        Query string `json:"query"`
    }
    datastar.ReadSignals(r, &sig)

    // 2. open the SSE stream
    sse := datastar.NewSSE(w, r)

    // 3a. patch DOM — by raw HTML string, by templ component, or by fmt
    sse.PatchElements(`<div id="x">hi</div>`, datastar.WithSelectorID("x"), datastar.WithModeInner())
    sse.PatchElementTempl(MyList(items), datastar.WithSelectorID("list"), datastar.WithModeInner())

    // 3b. patch signals — marshal a Go value into client signals
    sse.MarshalAndPatchSignals(map[string]any{"count": 1})

    // 3c. misc
    sse.RemoveElementByID("toast")
    sse.ExecuteScript("document.getElementById('d').showModal()")
    sse.Redirect("/done")
}
```

**Key API**
- `datastar.NewSSE(w, r) *ServerSentEventGenerator`
- `datastar.ReadSignals(r, &v)` — unmarshal client signals into a struct pointer
- `sse.PatchElements(html, opts…)` / `PatchElementTempl(c, opts…)` / `PatchElementf(fmt, …)`
- `sse.MarshalAndPatchSignals(v)` / `PatchSignals([]byte)` / `…IfMissing` variants
- `sse.RemoveElementByID(id)` / `RemoveElement(selector)`
- `sse.ExecuteScript(js)` / `sse.Redirect(url)` / `sse.ConsoleLog(msg)`
- **Patch options:** `WithSelector(sel)` / `WithSelectorID(id)` / `WithSelectorf(...)` /
  `WithMode{Outer,Inner,Replace,Append,Prepend,Before,After,Remove}()` /
  `WithUseViewTransitions(true)` / `WithViewTransitions()`

These options map 1:1 to the SSE `selector` / `mode` / `useViewTransition` data keys in §7.

### Live recipes
A runnable, source-annotated recipe set is served at **`/datastar/recipes`** (public,
DB-free in-memory demos): two-way bind, server round-trip counter, in-memory TODO CRUD,
live search, indicator, polling, dialog, dropdown, and a **JS-free virtual scroll**
(server round-trip windowing). Source to copy from:
`web/components/datastar_recipes.templ` + `internal/handlers/datastar_recipes.go`.

> **Virtual scroll — core vs Pro.** The core recipe keeps the DOM to a fixed number
> of rows by re-fetching the visible window on scroll (`data-on:scroll__throttle`
> sets a start-index signal → `@get` → server patches only that window with a
> `translateY` offset; a spacer holds the full height). It avoids infinite-scroll's
> DOM bloat. It re-fetches the window on scroll, so smoothness depends on round-trip
> latency — but in practice it holds up well: smooth on localhost, and **verified
> comfortably usable on a typical VPS** (`__throttle` caps round-trips and overscan
> hides the gaps). Noticeable stutter only shows on high-latency links (distant
> servers, flaky mobile). Pro's *Rocket Virtual Scroll* keeps windowing client-side
> (rAF / overscan / DOM recycling, no per-scroll round-trip), so it stays smooth even
> under high latency. Use core for most cases; reach for Pro for very large lists that
> must stay buttery-smooth on slow links.
>
> **Fixed row height is required.** The core recipe assumes every row is the same
> height (`scrollTop / rowH` → index, spacer = `rows * rowH`, `translateY(start * rowH)`).
> Variable heights break all of this: you'd need cumulative offsets (prefix sums) +
> binary search to map scrollTop→index, and measure-then-correct after render — not
> feasible with `data-*` alone (needs real JS, and gets complex fast). Use this for
> uniform rows (tables, fixed-size cards); for variable-height content (chat, wrapping
> text, mixed media) use pagination/infinite-scroll or Pro.
> (Note: *infinite scroll* is core via `data-on-intersect`; *virtual scroll* and
> Rocket Virtual Scroll are different things — see the discussion that produced this.)
