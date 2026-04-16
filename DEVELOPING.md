# Running the demo

Three processes, three terminal tabs.

## 1. NATS server

```
nats-server -c demo/nats.conf
```

Starts NATS with JetStream on `localhost:4222` and WebSocket on `localhost:4223`.
WebSocket must be explicitly configured — `nats-server -js` alone does not enable it.

## 2. Cart domain service

```
cd demo/cart
go run .
```

Connects to NATS, creates the `PERHAP_STORE` stream if it doesn't exist,
and subscribes to `perhap.store.*.>` ready to process cart events.

## 3. Frontend

```
cd demo/web
npm install       # first time only
npm run build     # or: npm run watch (rebuilds on save)
npx serve .       # or any static file server
```

Open `http://localhost:3000` (or whatever port `serve` picks).

## How it works

- Each browser session gets its own cart entity ID (stored in `sessionStorage`)
- Clicking "Add to cart" publishes an `item_added` event to NATS
- The cart service reduces the event and publishes a model update
- The frontend receives the update via NATS.ws subscription and re-renders
- The `data-watch` element dims while the backend window is open (`data-pending="true"`)

## WebSocket port

NATS serves WebSocket connections on port `4223` by default. If yours is different,
update the `<meta name="perhap-server">` tag in `demo/web/index.html`.
