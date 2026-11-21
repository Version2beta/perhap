# Perhap

An event-sourced, reducer-based framework for full-stack reactive applications.

Events are the primary data. State is a derived view. The same event language spans browser to backend to any downstream service, with NATS at the center.

## Core idea

Every change to your system is an immutable event stored in NATS JetStream. Services reduce those events into models. The browser subscribes to model updates in real time. When a client sends an event, the backend reduces it and the updated model flows back automatically — no polling, no REST, no GraphQL.

```
Browser → publish event → NATS → Service reduces → publish model update → Browser
```

## NATS subject conventions

| Subject | Direction | Purpose |
|---|---|---|
| `perhap.{context}.{entityId}.{eventType}` | client → server | Publish an event |
| `perhap.{context}.{entityId}._model` | server → all | Model update after reduction |
| `perhap.{context}.{entityId}._exception` | server → all | Validation failure |
| `_perhap.{context}.{entityId}.model` | client → server | Hydration request (request-reply) |

The `_perhap.` prefix on hydration requests keeps them outside the JetStream stream filter, so NATS does not intercept the request-reply with a PubAck.

## Status and deployment limits

Implemented: Go reducers/validation, in-memory entity windows, JetStream replay,
browser watch/tell, DOM bindings and in-memory optimistic updates.
Planned: persistent snapshot cache, IndexedDB cache/outbox, offline replay,
summary subscriptions and code generation. See the specification as design intent,
not a list of shipped features.

Known limits: run **one service instance per context/domain**, with only one domain
model serving a context/entity. NATS queue groups distribute messages, not entity
ownership; multiple replicas can diverge and hydration can have multiple responders.
Live delivery uses core NATS (not a durable JetStream consumer); missed events are
recovered on activation/replay, not automatically while an entity remains active.
`LateEventPolicy` is currently reserved and ignored: events at or before the
in-memory snapshot boundary are skipped without an exception. Full replay sorts
all retained events, so late events can affect state after restart. Do not depend
on a late-event policy for correctness yet.

JetStream history must be retained for replay. Replay restores the last valid
canonical UUID-ordered prefix (and the invalid working history for later repair),
not necessarily the exact last-valid value observed under out-of-order live arrival.
Reducers/validators must be deterministic and models JSON-serializable.
The demo NATS configuration is unauthenticated development-only; production needs
TLS and application-scoped NATS permissions/credential issuance.

### Publication-readiness changes

The current local changes forward browser `.creds` authentication, preserve
optimistic state when idle-close clears pending (for both watch callbacks and
`perhap:model` listeners), validate replayed prefixes, and discard reducer
mutations and emitted events when the reducer returns an error. Replay retains
invalid working history so later events can repair it. These fixes do not yet
make incomplete or failed replay safe.

API/protocol examples, installation instructions, CI development commands and
portable documentation links have been corrected. The project uses the
[MIT License](LICENSE).

### Recovery direction: high availability deferred

For this publication scope, **high availability and automatic entity distribution
are deferred**. The intended recovery model is one isolated service process per
domain, with a separate durable consumer for each context/domain: stop on an
unrecoverable failure, restart, replay and resume. The previous process must be
terminated before its replacement starts; a queue group does not enforce this.
The existing restriction of one domain model per context/entity still applies.

This is a design direction, **not an implemented delivery guarantee**. Safe
crash-and-recover processing still requires:

- Durable JetStream delivery rather than only a Core NATS queue subscription.
- Acknowledgment only after a durable processing outcome.
- Stable effect identities and deduplication for retries after a crash between
  publishing an effect and acknowledging its input; no exactly-once promise.
- Complete replay before serving authoritative state, without a gap between
  replay and live consumption.
- Explicit blocked/error state for corrupt history or consistently failing
  events, rather than an endless restart loop or partial state reported as valid.

### Remaining reliability work

The next implementation pass must address replay transport/decode failures and
late-event processing. Proposed contracts, still to be finalized, are durable
late-event admission/application-order records, monotonic model revisions for
browser corrections, durable rejection with an exception on reducer errors,
and requiring complete retained history until durable checkpoints exist.
None of these proposals should be treated as shipped behavior.

A real authenticated browser integration test (including denied access) remains
a validation gap. Existing connection-option and mocked transport tests do not
establish authenticated end-to-end behavior.

Dependency remediation updates the Go dependencies and frontend lockfile, including
`esbuild` past the `0.28.1` patched minimum for `GHSA-g7r4-m6w7-qqqr`, and raises
the Go minimum to `1.26.11` for standard-library security fixes. GitHub alert closure
requires publishing these changes and a subsequent Dependabot rescan; local
remediation alone does not close remote alerts.

## Prerequisites and installation

Use Go 1.26.11 or newer (see `go.mod`), Node.js 25 with npm (matching CI),
and `nats-server` with JetStream and WebSocket support for the demo.
From a standalone Go application:

```sh
go mod init example.com/myapp # if not already a module
go get github.com/version2beta/perhap/sdk/go@77b51dd01ec61ea99fb923afd833242b4f9d0bde
```

The revision above is the reviewed baseline, not a release tag; use the desired
published revision when available. Import the SDK from the repository module:

```go
import (
    "context"
    "log"
    "time" // for optional builder settings below
    perhap "github.com/version2beta/perhap/sdk/go"
    "github.com/nats-io/nats.go"
)
```

The browser SDK is currently source in `demo/web`, not a published npm SDK.
Build it using the development instructions and serve the resulting `perhap.js`.

## Go SDK

### Define your model

```go
type Cart struct {
    Items    map[string]int `json:"items"`
    Subtotal float64        `json:"subtotal"`
}
```

### Define event payload types

```go
type ItemAdded struct {
    SKU      string  `json:"sku"`
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
}

type ItemRemoved struct {
    SKU string `json:"sku"`
}
```

### Write pure reducer functions

```go
func onItemAdded(cart Cart, ev ItemAdded) Cart {
    if cart.Items == nil {
        cart.Items = make(map[string]int)
    }
    cart.Items[ev.SKU] += ev.Quantity
    cart.Subtotal += float64(ev.Quantity) * ev.Price
    return cart
}

func onItemRemoved(cart Cart, ev ItemRemoved) Cart {
    delete(cart.Items, ev.SKU)
    return cart
}
```

### Wire up the domain and run

```go
func main() {
    nc, err := nats.Connect(nats.DefaultURL)
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    d := perhap.NewDomain[Cart]("store", "cart")
    perhap.On(d, "item_added",   onItemAdded)
    perhap.On(d, "item_removed", onItemRemoved)

    svc, err := perhap.NewService(nc, d.ServiceConfig())
    if err != nil {
        log.Fatal(err)
    }

    log.Fatal(svc.Run(context.Background()))
}
```

`NewService` creates the JetStream stream automatically if it does not exist. `Run` subscribes to all events in the context and blocks until the context is cancelled.

### Validation

```go
func validateCart(cart Cart) []perhap.ValidationError {
    if cart.Subtotal < 0 {
        return []perhap.ValidationError{{Field: "subtotal", Message: "cannot be negative"}}
    }
    return nil
}

d.Validate(validateCart)
```

When validation fails the model stays at the last valid state and an `_exception` event is published.

### Domain builder options

```go
d.WithWindowSize(50)                        // snapshot after 50 events (default 100)
d.WithWindowIdleClose(200 * time.Millisecond) // mark pending=false after 200ms idle (default 500ms)
d.WithPassivateTimeout(10 * time.Minute)    // evict idle supervisors after 10m (default 5m)
d.WithStreamReplicas(3)                     // JetStream replication factor (default 1)
```

### Event type

```go
type Event struct {
    ID            string          // UUIDv6 — timestamp-ordered
    EntityID      string
    Context       string
    Type          string
    CausationID   *string
    CorrelationID string
    Data          json.RawMessage // raw payload, already unmarshaled by Domain
}

func (e Event) Time() (time.Time, error) // extract timestamp from UUID
```

### Supervisor lifecycle

Each entity gets its own supervisor goroutine, created on demand and passivated after idle. On creation (or after passivation/restart) the supervisor replays retained JetStream history to restore state, subject to the deployment limits above.

The window holds recent events that haven't been folded into a snapshot yet. When the window exceeds `WindowSize`, the oldest half is folded into the snapshot. After `WindowIdleClose` with no new events, `pending` is set to false and a model update is published. After `PassivateTimeout` with no new events, the supervisor exits.

## TypeScript SDK

### Setup

```html
<meta name="perhap-server" content="ws://localhost:4223">
<script type="module" src="perhap.js"></script>
```

`perhap.js` auto-connects on load using the server from the meta tag (defaults to `ws://localhost:4223`). Export `ready` resolves once connected.

`configure({ credentials })` and `<meta name="perhap-credentials">` accept the
**contents of a NATS user .creds file**, not a path, URL or bearer token. Explicit
options override meta tags. Configure credentials before the first connection:
set meta tags before loading the auto-initializing bundle, or use the `core.ts`
entry point for programmatic configuration. Credentials delivered to a browser
are visible to that user; issue narrowly scoped credentials, never service secrets.

### Declarative bindings

The simplest path: put NATS subjects and data paths directly in HTML.

```html
<!-- Watch an entity: subscribes + hydrates automatically -->
<div data-watch="store/cart-abc123">

  <!-- Bind reduction fields to element text -->
  <span data-bind="subtotal"></span>
  <span data-bind="items.sku-001"></span>

  <!-- Bind meta-properties -->
  <span data-bind="@pending">saving…</span>
  <span data-bind="@valid"></span>
  <span data-bind="@entity-id"></span>
  <span data-bind="@context"></span>

  <!-- Bind to element attributes -->
  <div data-bind-attr="data-subtotal:subtotal"></div>

  <!-- Publish events on click -->
  <button data-tell="item_added" data-tell-data='{"sku":"sku-001","quantity":1,"price":9.99}'>
    Add to cart
  </button>

  <!-- Publish on form submit (all fields serialized) -->
  <form data-tell="checkout_started">
    <input name="email" type="email">
    <button type="submit">Checkout</button>
  </form>

  <!-- Publish on input change -->
  <input data-tell="quantity_changed" name="quantity" type="number">

</div>
```

Nested `data-watch` elements each manage their own entity independently.

Attribute bindings read reduction fields only; use the `data-pending` CSS hook
for pending metadata.

### JavaScript API

```typescript
import { configure, watch, tell, ready } from './perhap.js'

// Optional: override server (otherwise uses meta tag or default)
await configure({ server: 'ws://localhost:4223' })
await ready // configure logs failures; ready resolves only after successful connection

// Watch an entity
const unwatch = watch('store', 'cart-abc123', (state) => {
  console.log(state.reduction)   // current model
  console.log(state.pending)     // true while backend is processing
  console.log(state.valid)       // false if last reduction failed validation
  console.log(state.entityId)    // 'cart-abc123'
  console.log(state.context)     // 'store'
})

// Publish an event
tell('store', 'cart-abc123', 'item_added', { sku: 'sku-001', quantity: 1, price: 9.99 })

// Stop watching
unwatch()
```

### Optimistic updates

Provide a frontend reducer to `watch()` for instant UI feedback before the server responds (`renderCart` is your callback;
this TypeScript must be compiled before use in a browser):

```typescript
watch('store', 'cart-abc123', renderCart, (reduction, eventType, data) => {
  if (eventType === 'item_added') {
    const payload = data as { sku: string; quantity: number }
    const items = { ...(reduction.items as Record<string, number> ?? {}) }
    items[payload.sku] = (items[payload.sku] ?? 0) + payload.quantity
    return { ...reduction, items }
  }
  return reduction
})
```

When `tell()` is called, the frontend reducer applies immediately and `state.pending` is set to `true`. The SDK keeps that optimistic pending state in memory for active watches until a newer backend update confirms it.

### Global model event

Every update (including optimistic) dispatches a `perhap:model` event on `window`:

```javascript
window.addEventListener('perhap:model', (e) => {
  const { entity_id, context, reduction, pending, valid } = e.detail
})
```

### ModelState

```typescript
interface ModelState {
  entityId: string
  context: string
  reduction: Record<string, unknown> | null  // null until first update arrives
  backendPending: boolean                    // server window is open
  valid: boolean                             // last reduction passed validation
  reducedThrough: string | null              // UUIDv6 of most recent event applied
  pending: boolean                           // backendPending || in-memory optimistic events in flight
}
```

### Source layout

The browser SDK is split into a small set of focused modules:

| File | Purpose |
|---|---|
| `demo/web/src/perhap.ts` | public entry point and browser auto-init |
| `demo/web/src/core.ts` | NATS connection, watch/tell API, optimistic tracking |
| `demo/web/src/model-state.ts` | pure state transitions and buffer handling |
| `demo/web/src/dom.ts` | declarative DOM bindings and mutation wiring |
| `demo/web/src/uuid.ts` | monotonic UUIDv6 generation for the browser |

### CSS state hooks

`data-watch` elements receive these attributes on every update:

```css
[data-watch][data-pending="true"] { opacity: 0.6; }
[data-watch][data-valid="false"]  { border-color: red; }
```

`data-entity-id` and `data-context` are also set, enabling scoped selectors.

## Running the demo

Three terminal tabs:

```bash
# 1. NATS with JetStream + WebSocket
nats-server -c demo/nats.conf

# 2. Cart service
cd demo/cart && go run .

# 3. Frontend
cd demo/web && npm ci && npm test && npm run build && npx serve .
```

Open the URL printed by `serve`. Each browser tab gets its own cart (session-scoped entity ID).

See [DEVELOPING.md](DEVELOPING.md) for troubleshooting and development workflow.

## Specification

[spec/specification.md](spec/specification.md) — the full design intent, including late event policies, snapshot semantics, and planned tooling.

## License

[MIT License](LICENSE). Copyright (c) 2026 Rob Martin.

## Lineage

Perhap began as an Elixir framework for reactive domain-driven design. This is its second realization: language-agnostic, bus-centered, with Go as the reference backend SDK and TypeScript as the frontend SDK.
