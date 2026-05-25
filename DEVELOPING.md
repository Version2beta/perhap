# Developing Perhap

## Running the demo

Three processes, three terminal tabs.

### 1. NATS server

```
nats-server -c demo/nats.conf
```

Starts NATS with JetStream on `localhost:4222` and WebSocket on `localhost:4223`.
WebSocket must be explicitly configured — `nats-server -js` alone does not enable it.

### 2. Cart domain service

```
cd demo/cart
go run .
```

Connects to NATS, creates the `PERHAP_STORE` stream if it doesn't exist,
and subscribes to `perhap.store.*.>` ready to process cart events.

### 3. Frontend

```
cd demo/web
npm install       # first time only
npm run build     # or: npm run watch (rebuilds on save)
npx serve .       # or any static file server
```

Open the URL printed by `serve` (usually `http://localhost:3000`).

### How it works

- Each browser session gets its own cart entity ID (stored in `sessionStorage`)
- Clicking "Add to cart" publishes an `item_added` event to NATS
- The cart service reduces the event and publishes a `_model` update
- The frontend receives the update via NATS WebSocket subscription and re-renders
- The `data-watch` element dims while the backend window is open (`data-pending="true"`)

## WebSocket port

NATS serves WebSocket connections on port `4223` by default (configured in `demo/nats.conf`).
If yours is different, update the `<meta name="perhap-server">` tag in `demo/web/index.html`.

## Go SDK development

The Go SDK lives in `sdk/go/`. It is a standard Go module (`package perhap`).

```
go test ./sdk/go/...          # run tests
go vet ./sdk/go/...           # lint
```

Key files:

| File | Purpose |
|---|---|
| `event.go` | Event, ModelUpdate, ValidationError types; NewEventID (UUIDv6) |
| `domain.go` | Domain builder: NewDomain, On, Validate, ServiceConfig |
| `service.go` | Service: creates stream, subscribes, dispatches to supervisors |
| `supervisor.go` | Per-entity goroutine: window, snapshot, reduction, passivation |
| `service_integration_test.go` | Embedded-NATS coverage for replay and `_exception` storage |

### Supervisor lifecycle (mental model)

1. Event arrives on `perhap.{context}.{entityId}.{eventType}`
2. `getOrCreateSupervisor` looks up or creates a supervisor for that entityId
3. On creation, state is replayed from JetStream (so restarts are transparent)
4. The supervisor inserts the event into its in-memory window, sorted by UUID
5. It re-reduces from the snapshot through the full window
6. If valid, `lastValid` is updated; if invalid, an `_exception` is emitted
7. A `_model` update is published with the new state
8. After `WindowIdleClose` with no new events, `pending=false` and a final `_model` is published
9. After `PassivateTimeout` with no new events, the supervisor exits and is removed from the registry
10. The next event for this entity triggers a fresh replay and a new supervisor

### JetStream stream

One stream per context, named `PERHAP_{CONTEXT}` (e.g. `PERHAP_STORE`).
Subject filter: `perhap.{context}.>`.
Internal subjects (`_model`, `_exception`) match this filter but are skipped by the queue subscriber.
Hydration requests use `_perhap.` prefix to stay outside the stream filter.
The default replication factor is `1`; production deployments should override it with `WithStreamReplicas(...)`.

## TypeScript SDK development

The TypeScript SDK is split into focused modules under `demo/web/src/`.

```
cd demo/web
npm test         # pure state + UUID tests under Node
npm run build    # one-shot
npm run watch    # watch mode
```

Outputs to `demo/web/perhap.js` (ES module, bundled).

Key files:

| File | Purpose |
|---|---|
| `perhap.ts` | public entry point and auto-init |
| `core.ts` | connection lifecycle, watch/tell API |
| `model-state.ts` | pure state transitions, pending tracking, buffer helpers |
| `dom.ts` | `data-watch` / `data-tell` binding layer |
| `uuid.ts` | monotonic UUIDv6 generator for the browser |

### Hydration sequence

1. Subscribe to `perhap.{context}.{entityId}._model` — buffer updates until hydrated
2. Send a request to `_perhap.{context}.{entityId}.model` with 5s timeout
3. On response: mark hydrated, apply hydration, drain buffer (apply anything newer than hydration)
4. On timeout: mark hydrated, apply any buffered updates in order (don't leave the UI blank)

### Stale update detection

Each `ModelUpdate` carries `reduced_through` — the UUID of the last event applied.
Because UUIDv6 is timestamp-ordered, string comparison (`a > b`) correctly orders updates.
Confirmed updates clear any optimistic event IDs that sort at or before `reduced_through`.
Stale updates are dropped if `reduced_through` is not newer than the current state.

### Pending semantics

`ModelState.pending` is the union of:

- backend pending from the wire-format `pending` flag
- in-memory optimistic events that have been published locally but not yet confirmed by a newer backend update

## Troubleshooting

### Frontend shows nothing / stays blank

- Check browser console for `[perhap]` log lines
- Confirm NATS is running with WebSocket: `nats-server -c demo/nats.conf` (not just `-js`)
- Confirm the `perhap-server` meta tag matches the WebSocket port
- Confirm the domain service is running and connected

### "NATS connection refused" in the Go service

- Check NATS is running: `nats-server -c demo/nats.conf`
- Default NATS TCP port is 4222. WebSocket (4223) is only for browsers.

### Events published but model never updates

- Check that the subject format matches: `perhap.{context}.{entityId}.{eventType}`
- Check that the service context/domain matches the published subject
- Use `nats sub 'perhap.>'` to watch all traffic

### Model loses state after service restart

State is replayed from JetStream on supervisor creation — restarts should be transparent.
If state is lost, check that the JetStream stream still exists: `nats stream info PERHAP_{CONTEXT}`.

### Hydration request times out

The hydration request goes to `_perhap.{context}.{entityId}.model`.
The service must be running and subscribed. Check for `perhap: service {context}/{domain} running` in the service log.
If the entity has no events yet, the hydration response will have an empty reduction — this is normal.
