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
    nc, _ := nats.Connect(nats.DefaultURL)
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

Each entity gets its own supervisor goroutine, created on demand and passivated after idle. On creation (or after passivation/restart) the supervisor replays from JetStream to restore prior state — no history is lost across restarts.

The window holds recent events that haven't been folded into a snapshot yet. When the window exceeds `WindowSize`, the oldest half is folded into the snapshot. After `WindowIdleClose` with no new events, `pending` is set to false and a model update is published. After `PassivateTimeout` with no new events, the supervisor exits.

## TypeScript SDK

### Setup

```html
<meta name="perhap-server" content="ws://localhost:4223">
<script type="module" src="perhap.js"></script>
```

`perhap.js` auto-connects on load using the server from the meta tag (defaults to `ws://localhost:4223`). Export `ready` resolves once connected.

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
  <div data-bind-attr="aria-busy:pending"></div>

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

### JavaScript API

```typescript
import { configure, watch, tell, ready } from './perhap.js'

// Optional: override server (otherwise uses meta tag or default)
await configure({ server: 'ws://localhost:4223' })

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

Provide a frontend reducer to `watch()` for instant UI feedback before the server responds:

```typescript
watch('store', 'cart-abc123', renderCart, (reduction, eventType, data) => {
  if (eventType === 'item_added') {
    const items = { ...reduction.items as Record<string, number> }
    items[data.sku] = (items[data.sku] ?? 0) + data.quantity
    return { ...reduction, items }
  }
  return reduction
})
```

When `tell()` is called, the frontend reducer applies immediately and `state.pending` is set to `true`. Once the server's authoritative update arrives it replaces the optimistic state.

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
  pending: boolean                           // backendPending || optimistic update in flight
}
```

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
cd demo/web && npm install && npm run build && npx serve .
```

Open the URL printed by `serve`. Each browser tab gets its own cart (session-scoped entity ID).

See [DEVELOPING.md](DEVELOPING.md) for troubleshooting and development workflow.

## Specification

[spec/specification.md](spec/specification.md) — the full design intent, including late event policies, snapshot semantics, and planned tooling.

## Lineage

Perhap began as an Elixir framework for reactive domain-driven design. This is its second realization: language-agnostic, bus-centered, with Go as the reference backend SDK and TypeScript as the frontend SDK.
