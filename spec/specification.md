# Perhap Specification

**Version**: 0.4.0-draft  
**Status**: Working Draft  
**Repository**: github.com/version2beta/perhap

**Implementation note**: this specification is broader than the current reference SDK. The shipped browser SDK currently implements in-memory optimistic pending for active watches; persistent outbox and offline sections remain design targets rather than completed runtime features.

---

## Table of Contents

1. [Vision](#1-vision)
2. [Reactive Manifesto Compliance](#2-reactive-manifesto-compliance)
3. [Core Concepts](#3-core-concepts)
4. [Architecture Overview](#4-architecture-overview)
5. [Message Bus](#5-message-bus)
6. [Protocol](#6-protocol)
7. [Event Ordering and Consistency](#7-event-ordering-and-consistency)
8. [Domain Service Specification](#8-domain-service-specification)
9. [Go SDK](#9-go-sdk)
10. [Frontend SDK](#10-frontend-sdk-typescript)
11. [Declarative Attribute Layer](#11-declarative-attribute-layer)
12. [Code Generation](#12-code-generation)
13. [Pending State and Validity](#13-pending-state-and-validity)
14. [Cross-Entity Events](#14-cross-entity-events)
15. [Offline Operation](#15-offline-operation)
16. [Development Process](#16-development-process)
17. [Example Domain](#17-example-domain)

---

## 1. Vision

### The Problem

Most software stores current state. A database row represents a thing as it is now. When something changes, the row is updated. History is discarded. This model is simple to understand but carries significant costs: you cannot reconstruct how the system arrived at its current state, you cannot build new projections of past data, and multiple services sharing the same mutable state create coupling that compounds over time.

### The Claim

Perhap is a framework for building applications where **events are the primary data and state is a derived view**. Rather than storing what things are, Perhap stores what happened. Current state is computed by reducing a sequence of immutable events — the same way a bank balance is computed from a ledger of transactions, not by overwriting a single number.

The event log is the only system of record. Everything else — every model, every projection, every snapshot — is a derived artifact. This is not a performance compromise. It is the correct epistemology for a system that must reason about time, history, and causality.

This is not a new idea. Event sourcing, Command Query Responsibility Segregation (CQRS), and Domain-Driven Design (DDD) have each explored parts of this space. Perhap synthesizes them into a cohesive framework with a specific additional claim: that the same event language can span the full stack, from browser to backend to any downstream service, with a shared message bus at the center and generated frontend reducers derived from backend definitions.

### Everything Is a Reaction

Perhap has no concept of commands. There are only events — immutable facts stated in past tense. A user clicking "request time off" produces a `time_off_requested` event. A payment processor completing produces a `payment_confirmed` event. A rules engine deciding produces an `approval_auto_approved` event. All participants in the system — human or automated, frontend or backend — express themselves through events. Everything is a reaction to something that happened.

This is a deliberate philosophical commitment, not a naming convention. Commands imply a request that might be rejected. Events are facts. They happened. The system's job is to react to them honestly, including when the reaction reveals that the resulting state is invalid.

### Lineage

Perhap began as an Elixir framework for reactive domain-driven design (rDDD), described in talks at Code Mesh London (2015) and elsewhere. This specification describes its second realization: a language-agnostic architecture in which the message bus is the framework, Go provides the reference backend SDK, and TypeScript provides the frontend SDK. The core ideas are unchanged. The implementation is more general.

The intellectual lineage includes:

- **Domain-Driven Design** (Eric Evans) — bounded contexts, ubiquitous language, aggregates, events
- **The Reactive Manifesto** — responsive, resilient, elastic, message-driven systems
- **Rich Hickey's work on identity** — identity as a series of causally related states over time, not a mutable current value
- **CQRS and Event Sourcing** (Greg Young) — separating the write model from the read model, storing facts not state
- **The Elm Architecture** — unidirectional data flow, pure reducer functions, explicit state management
- **CRDTs** — convergent data structures, commutativity as a design property

### What Perhap Is Not

Perhap is not an application. It is a framework — infrastructure and convention. It has no opinions about your domain. It provides the plumbing; you provide the meaning.

Perhap is not a database. It uses NATS JetStream as a durable event log and delegates persistence concerns to an infrastructure layer it does not own.

Perhap is not a replacement for careful domain modeling. The framework is only as good as the event model it runs. A poorly modeled domain produces a poorly behaved application regardless of the infrastructure beneath it.

---

## 2. Reactive Manifesto Compliance

The [Reactive Manifesto](https://www.reactivemanifesto.org) defines four properties of reactive systems: **Responsive**, **Resilient**, **Elastic**, and **Message-Driven**. Perhap is designed to satisfy all four. Where it deviates, it does so deliberately, and this section names those deviations explicitly.

### Responsive

Reactive systems respond in a timely manner. Perhap achieves responsiveness through:

- **Optimistic frontend updates**: user actions produce immediate UI state changes before backend confirmation
- **Push-based model delivery**: confirmed state arrives via NATS subscription, not polling
- **Per-entity concurrency**: each entity processes events independently; one slow entity does not block others

### Resilient

Reactive systems stay responsive in the face of failure. Perhap's resilience model:

- **Entity isolation**: the entity supervisor pattern ensures that a failure in one entity's processing does not propagate to others
- **Durable event log**: JetStream persists events before reducers run; a service crash does not lose events
- **Replay on recovery**: a recovering service replays its entity's event history from JetStream to reconstruct current state

**Deviation**: A single NATS node is a single point of failure. Single-node NATS is appropriate for development only. Production deployments require a NATS cluster of at least three nodes for JetStream quorum. This is a deployment requirement, not a framework limitation, but it must be satisfied for resilience claims to hold.

### Elastic

Reactive systems stay responsive under varying workload. Perhap's elasticity model:

- **Stateless domain services**: reducers are pure functions; multiple instances can run concurrently for different entities
- **NATS queue groups**: multiple instances of a domain service subscribe with the same queue group name; NATS distributes messages across instances. Each message for a given entity is delivered to exactly one instance, ensuring that entity supervisors are not duplicated across instances.
- **Per-entity processing**: the unit of work is an entity, not a service; load distributes at entity granularity. One domain service instance handles all events for a given entity at a time — this is a serialization guarantee, not a limitation.

**Deviation — location coupling**: Perhap subjects embed entity UUIDs (`perhap.{context}.{entity_id}.{type}`). Clients must know entity UUIDs to publish and subscribe. This is a deliberate deviation from strict location transparency — in Perhap, the entity UUID *is* the address, and routing by identity is the desired behavior. Clients that know an entity's identity should be able to reach it directly.

### Message-Driven

Reactive systems rely on asynchronous message passing. Perhap is fundamentally message-driven via NATS. All communication between domain services is through published events on named subjects.

**Deviation — initial model hydration**: When a frontend context loads, it opens a NATS subscription and then performs a request/reply to fetch the current model state, buffering any updates that arrive before the hydration reply. This is a request/reply pattern layered on top of message-driven communication. The justification is pragmatic: the current confirmed model must be established before buffered updates can be correctly ordered and applied. The hydration is a deliberate, bounded deviation isolated to context initialization and does not affect the ongoing event flow.

**Deviation — development topology**: Running a single NATS node without clustering compromises resilience as noted above. The framework cannot enforce deployment topology; applications are responsible for satisfying the resilience property in production.

---

## 3. Core Concepts

These definitions are precise. Where a term has common usage in DDD or event sourcing literature that conflicts with Perhap's usage, the conflict is noted.

### Event

An **event** is an immutable record of something that happened. Events are facts, stated in past tense. They are never updated or deleted.

Every event has:
- A unique identifier (UUIDv6, assigned at source — see [Event Ordering and Consistency](#7-event-ordering-and-consistency))
- An entity ID identifying what the event pertains to
- A context identifying the bounded context
- A type identifying what happened (e.g., `item_added`, `time_off_requested`)
- A timestamp derived from the UUID
- A causation ID (the event that caused this event, if any)
- A correlation ID (the originating session or request)
- A data payload (arbitrary, defined by the domain)

Events flow in one direction: they are appended to the log and never retracted. All events are stored. An event that produces an invalid model state is not rejected — it is stored, the resulting state is validated, and if invalid, an exception event is emitted. The event log is always complete.

### Reducer

A **reducer** is a pure function that takes a model and an event and returns a new model and zero or more new events.

```
reduce(event_type, model, event) → (new_model, []event)
```

Reducers have no side effects. They do not write to databases, call external services, or depend on global state. The same reducer called with the same inputs always produces the same outputs. This property makes reducers trivially testable, safely distributable across nodes, and suitable for execution on both the backend and the frontend.

Reducers must be:
- **Pure**: no side effects, no I/O, no global state access
- **Total**: every event type the service handles must have a defined case; unhandled event types return the model unchanged
- **Deterministic**: same inputs always produce same outputs

For events that can arrive concurrently from genuinely independent sources, reducers should be designed to be **commutative** where possible — producing the same result regardless of the order they are applied. See [Event Ordering and Consistency](#7-event-ordering-and-consistency) for the full treatment.

### Entity

An **entity** is a thing with a stable identity and an event history. Its current state is always computed — never stored as a final value.

Entities are identified by UUIDv6. The identity is stable; the model changes over time as events accumulate. This is the functional programming view of identity: not "what a thing is now" but "the series of states it has passed through."

### Model

The **canonical model** of an entity is its complete, ordered event history in JetStream. This is the only representation that is always correct.

For practical use, a model is carried as a **snapshot** plus an ordered set of **window events**: `model = (snapshot, [e₁, e₂, ..., eₙ])`. The current state is computed by `reduce(snapshot, window)` where window events are sorted by UUID. The snapshot is a performance artifact — a cached reduction to a known point in time — not a semantic truth. No snapshot is ever definitively correct; it reflects the events that were known when it was taken.

What clients receive as a "model update" is the current best reduction — a convenience projection, explicitly labeled as such with metadata indicating what events produced it. Clients who need to verify or re-reduce can fetch the source events from JetStream.

Multiple domain services can maintain different models of the same entity, each projecting the event history differently for different purposes.

### Reduction Window

The **reduction window** is the ordered set of events held by the entity supervisor since the last snapshot. Events in the window are sorted by UUID (source timestamp). When a new event arrives, it is inserted into the window at its correct UUID position and the window is re-reduced from the snapshot. This means arrival order is irrelevant — events are always reduced in source-timestamp order.

The window defines the out-of-order tolerance of the system. An event that arrives within the window period is correctly incorporated. An event that arrives after its window has passed is a late event, handled according to domain policy.

### Valid State

A model is **valid** when its current reduction satisfies all domain invariants declared by the reducer. Validity is assessed after every re-reduction. If the reduction is valid, the model advances. If the reduction is invalid, the model holds at the last valid reduction, the `valid` flag is set false, and an exception event is emitted.

This gives the model four states, defined by two orthogonal flags:

| `pending` | `valid` | Meaning |
|---|---|---|
| `true` | `true` | Window open, reduction good so far |
| `true` | `false` | Window open, current reduction is invalid |
| `false` | `true` | Window closed, confirmed correct — the normal state |
| `false` | `false` | Window closed, confirmed invalid, exception event emitted |

A model with `valid: false` holds at the last known valid reduction. It is stale but never wrong. This is true eventual consistency: the value is always a valid state that actually existed; the `valid` flag indicates whether it is current.

### Exception Event

An **exception event** is emitted when an entity's reduction produces an invalid state. It is not an error in the conventional sense — it is an event like any other, stored in JetStream, routed through the bus, and handled by reducers and policies. There are no errors in Perhap; there are only events.

Exception events flow to subscribers including the frontend, can trigger automated remediation policies, and can route to human review queues. When corrective events arrive and produce a valid reduction, the model advances and `valid` returns to `true`.

### Context

A **context** is a bounded domain — a namespace that groups related entities, events, and domain services. Contexts correspond to bounded contexts in DDD. They define the scope of a ubiquitous language: a term means what it means within a context, and may mean something different in another.

Contexts are the unit of subscription for the frontend. A frontend client subscribes to events within a context for a specific entity. When the user's context changes, subscriptions change.

### Domain Service

A **domain service** is a process that subscribes to events for one or more entity types within a context, runs reducers, and publishes model updates and new events. Domain services are the compute layer of Perhap.

Domain services are stateless with respect to each other. Their state lives in the event log. Multiple instances of the same domain service can run concurrently without coordination, using NATS queue groups for load distribution.

### Entity Supervisor

An **entity supervisor** is a short-lived process, managed by a domain service, that maintains the reduction window for a single entity and serializes event processing. The supervisor holds the snapshot and window; when an event arrives, it inserts the event into the window, re-reduces from the snapshot, validates, and publishes the result.

Supervisors are ephemeral. Their state — the snapshot and window — is reconstructed from JetStream on activation. A supervisor crash loses nothing.

### Domain

A **domain** is the named sub-projection of an entity maintained by a specific domain service. Multiple domain services within the same context can project the same entity's event history differently. The model wire format includes the `domain` field to identify which domain service produced the reduction. Cache keys include domain to allow multiple projections of the same entity to coexist. Domain names are lowercase identifiers matching the domain service's declared name (e.g., `cart`, `inventory`, `analytics`).

### Backend-Only and Frontend-Visible Aggregates

An aggregate is **frontend-visible** if its reduction is directly rendered in a frontend context. It carries the `// +perhap:frontend` annotation and has a generated TypeScript model type.

An aggregate is **backend-only** if its model is never directly rendered in a frontend context. The frontend needs only to know the *outcomes* of its processing (expressed as events it emits downstream), not its internal state. Backend-only aggregates typically represent infrastructure, automation pipelines, or approval workflows where the intermediate state is opaque to end users.

The test: "Does any frontend component need to read a field from this entity's reduction directly?" If yes, frontend-visible. If no, backend-only.

### Pending State

**Pending** is a first-class state of the frontend model. The `pending` flag has two sources:

- **Backend window pending**: the backend supervisor's reduction window is still open — the `pending` field in the model wire format is `true`. More events may arrive that change the reduction.
- **Frontend optimistic pending**: the frontend has published events to the outbox that have not yet been confirmed by the backend.

The frontend `ModelState.pending` is `true` when either condition holds. The wire format `pending` field represents the backend window state only. These are distinct — an entity can have a closed backend window (`pending: false` on the wire) while still having unconfirmed frontend events in the outbox.

For entities downstream of an async boundary — a human approval step, an external payment processor, a scheduled job — pending state may persist for minutes, hours, or indefinitely. This is correct behavior. Falsely optimistic UI is a data integrity error.

For entities downstream of an async boundary — a human approval step, an external payment processor, a scheduled job — pending state may persist for minutes, hours, or indefinitely. This is correct behavior. Falsely optimistic UI is a data integrity error.

### Outbox

The **outbox** is a persistent local queue of events that the frontend has generated but not yet confirmed by the backend. The outbox lives in IndexedDB. It is the mechanism for offline operation and causal ordering of cross-entity events.

### Ubiquitous Language

**Ubiquitous language** is the shared vocabulary of a bounded context — terms that mean the same thing to domain experts, developers, and the system itself. Event names, entity names, and field names in Perhap are part of the ubiquitous language. They are chosen in collaboration with domain experts and used consistently across all artifacts: code, documentation, and conversation.

The ubiquitous language is enforced in code. Event type strings, model field names, and subject names must match the agreed vocabulary exactly. Divergence between code and language is a defect.

### Data Sanity

A system is **sane** when its internal model of the world is consistent with the user's experience of the world. Sanity is maintained by storing all events — including those that produce invalid states — and by emitting exception events rather than silently accepting invalid state or silently rejecting events.

Rejecting an event that represents something a user actually did creates an inconsistency between reality and the model. Perhap stores the event, validates the resulting state, and if invalid, emits an exception event and holds the model at the last valid state. The invalidity is surfaced for resolution, not hidden or discarded.

---

## 4. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Frontend Applications                                       │
│  (TypeScript, any framework)                                 │
│                                                              │
│  Context Manager → Outbox → Reducer Runner → Model Cache    │
└──────────────────────────┬──────────────────────────────────┘
                           │ NATS.ws (WebSocket)
┌──────────────────────────▼──────────────────────────────────┐
│  NATS + JetStream                                            │
│                                                              │
│  Subject namespace: perhap.{context}.{entity_id}.{type}     │
│  JetStream streams: durable event log per context           │
└──────┬───────────────────┬───────────────────┬──────────────┘
       │                   │                   │
┌──────▼──────┐   ┌────────▼──────┐   ┌───────▼──────┐
│  Domain     │   │  Domain       │   │  Domain      │
│  Service    │   │  Service      │   │  Service     │
│  (Go)       │   │  (TypeScript) │   │  (any lang)  │
│             │   │               │   │              │
│  Reducers   │   │  Reducers     │   │  Reducers    │
│  Supervisors│   │  Supervisors  │   │  Supervisors │
└─────────────┘   └───────────────┘   └──────────────┘
```

The **NATS bus** is the center of the architecture. Everything communicates through it: frontend applications, backend domain services, and any downstream consumers. No component calls another component directly.

**Frontend applications** connect to NATS via NATS.ws. They publish events when users act, subscribe to entity subjects to receive model updates, and maintain a local model cache and outbox for optimistic updates and offline operation.

**Domain services** subscribe to event subjects for their context, run reducers against incoming events, and publish new events and model updates. A domain service is a consumer of the bus, not a server.

**JetStream streams** provide the durable event log. Every event published on a Perhap subject is automatically persisted. The bus and the event store are the same infrastructure. JetStream is the system of record.

---

## 5. Message Bus

### NATS and JetStream

Perhap uses [NATS](https://nats.io) as its message bus and [JetStream](https://docs.nats.io/nats-concepts/jetstream) as its durable event store. NATS provides subject-based pub/sub, request/reply, and queue groups. JetStream adds persistence, replay, and consumer tracking.

NATS is a single binary with no external dependencies, originally written in Ruby (2010) and rewritten in Go (2012). JetStream was added in 2021 and is embedded in the same binary. A development instance starts with:

```
nats-server -js
```

Production deployments require a cluster of at least three nodes for JetStream quorum and resilience.

### Clustering and Redundancy

NATS clustering is built into the `nats-server` binary. Nodes discover each other via configured routes and form a cluster that routes messages between themselves. From a client's perspective, the cluster is a single logical broker.

JetStream uses the **Raft consensus algorithm** for stream replication. Each stream is configured with a replication factor:

```
Replication: 3  (production)
Replication: 1  (development)
```

A message is acknowledged only after it has been written to the configured number of nodes. The cluster survives the loss of `floor(n/2)` nodes: a three-node cluster survives one failure; a five-node cluster survives two.

**The minimum production configuration is three NATS nodes with JetStream replication factor 3.** This is a deployment requirement. The framework cannot enforce it; applications are responsible for satisfying it.

### Subject Naming Convention

All Perhap subjects follow this convention:

```
perhap.{context}.{entity_id}.{event_type}
```

Examples:
```
perhap.store.018e7a3b-2d9b-6f00-b1a2-000000000001.item_added
perhap.hr.018f2c1a-4e8c-6f00-a3b4-000000000002.time_off_requested
perhap.hr.018f2c1a-4e8c-6f00-a3b4-000000000002.time_off_approved
```

Model update subjects use a reserved prefix to distinguish them from domain events:

```
perhap.{context}.{entity_id}._model
```

Wildcard subscriptions:
```
perhap.store.018e7a3b-2d9b-6f00-b1a2-000000000001.>   ← all events for one entity
perhap.hr.*.time_off_requested                         ← all time-off requests in hr
perhap.hr.>                                            ← all events in hr context
```

### JetStream Streams

Each context corresponds to a JetStream stream:

```
Stream name:  PERHAP_{CONTEXT}   (e.g., PERHAP_HR)
Subjects:     perhap.{context}.>
Retention:    limits-based, MaxAge set to domain/compliance requirement
Replication:  3 (production), 1 (development)
Storage:      file
```

Model update and internal control subjects are excluded from the event stream — they are ephemeral push notifications and request/reply endpoints, not domain events:

```
Excluded subjects: perhap.{context}.*._model
                   perhap.{context}.*._model.get
```

All Perhap-internal subjects use the `_` prefix on the terminal token. Any future internal subjects follow this convention and must be added to the stream exclusion list. Domain event types must not start with `_`.

### Event Log Retention

JetStream retains events according to configured limits. For Perhap, the appropriate policy is **limits-based retention** with `MaxAge` set to the domain's data retention requirement. Since the event log is the system of record, it must be retained at least as long as the oldest snapshot that references events within it. Beyond that minimum, retention is a business and compliance decision.

For long-lived streams, **tiered storage** is recommended: recent events on fast NVMe storage for active window and snapshot operations, older events on lower-cost storage for audit, compliance, and rare full replays.

The event log must not be truncated below the retention horizon of the oldest snapshot cache entry. Doing so makes snapshot rebuilding impossible for affected entities.

### Idempotency and Deduplication

NATS JetStream provides at-least-once delivery. Under network failure, consumer restart, or redelivery, the same event may arrive more than once. Idempotency is a system-level requirement.

Idempotency is enforced at the infrastructure level using JetStream's built-in deduplication: when publishing an event, set the `Nats-Msg-Id` header to the event's UUID. JetStream deduplicates within a configurable window (default 2 minutes, extendable). The same event UUID is never stored twice.

```go
msg := nats.NewMsg(subject)
msg.Header.Set("Nats-Msg-Id", event.ID)
msg.Data = eventJSON
js.PublishMsg(msg)
```

The entity supervisor additionally tracks the UUID of the last processed event as a safety net against duplicates that arrive outside the deduplication window.

### Client Authentication and Permissions

Frontend clients connect with scoped credentials issued at session establishment. Credentials restrict publish and subscribe permissions to subjects the client is authorized to access:

```
Publish:   perhap.store.{user_owned_entities}.>
Subscribe: perhap.store.{user_owned_entities}.>
           perhap.store.{user_owned_entities}._model
Deny:      perhap.hr.>
```

Domain services connect with service credentials scoped to their context.

Credential issuance and the authentication flow are outside Perhap's scope. Perhap requires that credentials be scoped appropriately; the mechanism is the application's responsibility.

### NATS.ws

Frontend clients connect to NATS via WebSocket using the [nats.ws](https://github.com/nats-io/nats.ws) library. The connection endpoint must be configured at application startup. TLS is required in production.

```typescript
import { connect, StringCodec } from 'nats.ws'

const nc = await connect({
  servers: 'wss://nats.example.com:443',
  authenticator: credsAuthenticator(userCredentials)
})
```

---

## 6. Protocol

### Event Wire Format

All events, whether published by frontend clients or backend services, use this JSON structure:

```json
{
  "id": "018e7a3b-2d9b-6f00-b1a2-000000000001",
  "entity_id": "018f2c1a-4e8c-6f00-a3b4-000000000002",
  "context": "store",
  "type": "item_added",
  "causation_id": "018e7a3b-2d9b-6f00-b1a2-000000000000",
  "correlation_id": "018e7a00-0000-6f00-0000-000000000000",
  "data": {}
}
```

**id**: UUIDv6, assigned at the source of the event (the client or service that observed it happening). UUIDv6 provides 100-nanosecond timestamp resolution with lexicographic sort order. The `id` is the canonical ordering key for events on the same entity. See [Event Ordering and Consistency](#7-event-ordering-and-consistency).

**causation_id**: The ID of the event that caused this event. Omit this field (or set to `null`) for originating events with no prior cause. This field enables tracing event chains. Consumers walking causal chains should treat a missing or null `causation_id` as a chain terminus.

**correlation_id**: The ID of the originating user session or external request. All events in a chain share the same `correlation_id`.

**data**: Arbitrary JSON. Schema is defined by the domain service for each event type.

Note: the event wire format has no `timestamp` field. The timestamp is encoded in the UUIDv6 `id` and can be extracted from it. This prevents clock skew from producing inconsistent timestamps between the `id` and an independent timestamp field.

### Model Wire Format

Model updates published to `perhap.{context}.{entity_id}._model`:

```json
{
  "entity_id": "018f2c1a-4e8c-6f00-a3b4-000000000002",
  "context": "store",
  "domain": "cart",
  "snapshot_uuid": "018e7a00-0000-6f00-0000-000000000000",
  "window": [
    "018e7a3b-2d9b-6f00-b1a2-000000000001",
    "018e7a3c-1a2b-6f00-c3d4-000000000002"
  ],
  "reduced_through": "018e7a3c-1a2b-6f00-c3d4-000000000002",
  "pending": false,
  "valid": true,
  "reduction": {}
}
```

**snapshot_uuid**: The UUID of the last event folded into the snapshot. Events with UUIDs before this are represented by the snapshot.

**window**: The ordered list of event UUIDs in the current reduction window, sorted by UUID. This is metadata — it identifies which events contributed to the reduction. The frontend uses the `reduction` field directly as its model starting point; it does not re-reduce the window events. The window UUIDs are used for staleness detection (via `reduced_through` comparison) and for causal bookkeeping.

**reduced_through**: The UUID of the most recent event in the window. The frontend uses this to determine whether a received model is newer than its current state by comparing UUIDs lexicographically.

**pending**: True when the backend's reduction window is still open — more events may arrive that change the reduction. The window closes after `IdleTimeout` passes with no new events. See [Pending State and Validity](#13-pending-state-and-validity) for the distinction between backend pending and frontend optimistic pending.

**valid**: True when the current reduction passed all domain validation. False when an invalid state was detected and the model is held at the last valid reduction. See [Pending State and Validity](#13-pending-state-and-validity).

**reduction**: The current best reduction of the entity — a convenience projection for clients. This is a derived view, not the canonical model. It reflects `reduce(snapshot, window)` at the time of publication.

### Event Acknowledgment

When a frontend client publishes an event, it uses NATS request/reply to receive a durability acknowledgment:

```typescript
const sc = StringCodec()
const reply = await nc.request(
  `perhap.store.${entityId}.item_added`,
  sc.encode(JSON.stringify(event)),
  { timeout: 5000 }
)
const ack = JSON.parse(sc.decode(reply.data))
// ack: { stored: true } or { stored: false, reason: "..." }
```

The acknowledgment confirms that the event has been persisted to JetStream. It does not indicate that the reducer has run or that the model is valid — those arrive separately on the model subject. The acknowledgment is a durability guarantee: the event is in the log.

If no acknowledgment arrives within the timeout, the event enters the outbox for retry.

### Initial Model Hydration

When a frontend context loads, the client must avoid a race between the hydration reply and subscription updates. The correct sequence:

1. **Open subscription first**, buffering any incoming model updates without applying them
2. **Request the current model** via `_model.get`
3. **Apply the hydration reply** as the starting model state, recording its `reduced_through` UUID
4. **Drain the buffer**: apply any buffered model updates whose `reduced_through` UUID is lexicographically greater than the hydration UUID
5. Begin processing new subscription updates normally

```typescript
const buffer: ModelUpdate[] = []
const sub = nc.subscribe(`perhap.store.${entityId}.>`, {
  callback: (msg) => {
    if (isModelUpdate(msg)) buffer.push(parse(msg))
  }
})

const reply = await nc.request(
  `perhap.store.${entityId}._model.get`,
  Empty,
  { timeout: 5000 }
)
const hydration = JSON.parse(sc.decode(reply.data))
// hydration: full model wire format including reduced_through UUID

// apply hydration, then drain buffer
applyModel(hydration)
for (const update of buffer) {
  if (update.reducedThrough > hydration.reducedThrough) applyModel(update)
}
buffer.length = 0
// hand off subscription to normal update handler
```

A domain service listens on `perhap.{context}.{entity_id}._model.get` and replies with the current model. Model updates with a `reduced_through` UUID lexicographically less than or equal to the hydration UUID are discarded as already-applied.

---

## 7. Event Ordering and Consistency

This section covers the distributed systems properties that Perhap is designed to satisfy and the mechanisms that achieve them.

### UUIDv6: Ordering at Source

Event IDs are **UUIDv6**, assigned at the source of the event — the client or service that observed or initiated it. This is deliberate.

UUIDv6 encodes a 60-bit timestamp at 100-nanosecond resolution since October 15, 1582 (the Gregorian calendar epoch), rearranged from UUIDv1's layout for correct lexicographic sort order. Two events with the same source assigned UUIDs in sequence will always sort in emission order.

**Why at source, not at the broker**: most events for a given entity originate from a single source — one user's browser session, one backend service reacting to a prior event. When the source assigns the UUID, the 100-nanosecond resolution provides a coherent local clock for that entity's event chain. Clock skew between sources is real but the effect on ordering is bounded by the skew magnitude. JetStream arrival order, by contrast, is determined by network latency, which is unbounded and has no relationship to the order events actually occurred.

**Node ID**: UUIDv6 (like UUIDv1) includes a 48-bit node ID. Use a random node ID rather than a MAC address. The MAC address is a privacy and security exposure; a random node ID is explicitly permitted by RFC 4122 and provides equivalent uniqueness guarantees.

### Canonical Ordering

The UUID of an event is its **canonical position** in the entity's history. JetStream sequence numbers reflect arrival order at the broker and are used as monotonic references for consumer tracking, deduplication, and snapshot bookkeeping — not for ordering events relative to each other.

When two events arrive at the supervisor out of order (C before B), the UUID sort produces the correct reduction order regardless of arrival sequence.

### The Reduction Window

The entity supervisor implements ordering through the **reduction window**:

1. An incoming event is inserted into the window at its correct UUID position
2. The window is re-sorted by UUID
3. The supervisor re-reduces from the snapshot through all window events in order
4. The result is validated

This makes arrival order irrelevant within the window period. The window is the infrastructure-level mechanism that achieves the benefit of commutative reducers without requiring all reducers to be written as commutative operations.

### Idempotency

At-least-once delivery means the same event may be delivered more than once. The system handles this at two layers:

**JetStream deduplication** (primary): the `Nats-Msg-Id` header carries the event UUID. JetStream discards duplicate publishes within the configured deduplication window. The same UUID is never stored twice.

**Supervisor deduplication** (safety net): the supervisor maintains a set of all event UUIDs currently in the window (`window_uuids`), plus the `snapshot_uuid` boundary. An event is a duplicate if its UUID is `≤ snapshot_uuid` (already folded) or is present in `window_uuids`. This set is persisted alongside the snapshot at passivation, ensuring deduplication survives supervisor restarts even after the JetStream deduplication window expires.

### Commutativity

The window mechanism handles out-of-order arrival for events that arrive within the window period. Commutativity — the property that `reduce(reduce(model, A), B) = reduce(reduce(model, B), A)` — is an additional design property that provides robustness beyond what the infrastructure guarantees.

Commutativity is **not a universal requirement** for all reducers. It is specifically relevant in three cases:

**Genuinely simultaneous events**: two independent sources emit events for the same entity with UUIDs that fall within 100 nanoseconds of each other. Because UUIDv6 includes a 48-bit node ID, the sort order for same-timestamp events from different nodes is deterministic — but determined by node ID, which has no causal meaning. Reducers for event pairs that can plausibly be simultaneous from independent sources should be designed to be commutative, since the sort order, while deterministic, is semantically arbitrary.

**Clock skew beyond the window**: if two causally related events (B caused by A) have their UUID timestamps inverted due to clock skew, the UUID sort will reduce them in the wrong order. Validation catches this and emits an exception event. A commutative reducer would handle the inversion gracefully without producing an invalid state. Commutativity is the difference between a hard exception and a silent correct result.

**Snapshot merging during late event rebuild**: when a very late event requires rebuilding from an earlier snapshot, partial computations from different starting points may need to be reconciled. Model structures designed as join-semilattices (CRDT state-based approach) make this merging well-defined and correct.

**The recommended approach for commutative reducer design**: accumulate facts rather than computing running values. Instead of a counter that decrements with a floor, maintain separate monotonic accumulators for additions and removals, combining them at read time. A set of facts is inherently unordered; a derived value is not.

### Late Events

An event arrives **late** when its UUID timestamp falls before the current snapshot boundary — the window has already advanced past it.

Three policies are supported, declared per domain:

**Accept and rebuild**: the late event is inserted into the event history at its correct UUID position. The supervisor rehydrates from the last snapshot that precedes the late event's UUID, re-reduces all subsequent events including the late event, and validates. Expensive but correct. Appropriate for high-integrity domains.

**Reject with exception**: the late event is stored in JetStream but not applied. An exception event is emitted documenting the late arrival. Appropriate when late events indicate a system problem to be investigated rather than a routine occurrence.

**Accept with flag**: the late event is applied at the current head with a metadata flag indicating it arrived out of its window. The model is marked as potentially affected by out-of-window ordering. Appropriate for low-stakes domains where approximate ordering is acceptable.

The late event policy is a domain declaration, not a framework default. Domains must declare their policy explicitly.

---

## 8. Domain Service Specification

A **domain service** is any process that participates in Perhap event processing. It connects to NATS, subscribes to event subjects, manages entity supervisors, runs reducers, validates reductions, and publishes results.

Any language with a NATS client library can implement a domain service. The Go SDK is the reference implementation. The specification below is language-agnostic.

### Joining the Bus

A domain service joins Perhap by:

1. Connecting to NATS with service credentials
2. Subscribing to `perhap.{context}.*.>` with a queue group named `perhap.{context}.{domain}`
3. Listening on `perhap.{context}.{entity_id}._model.get` for hydration requests
4. Publishing events to `perhap.{context}.{entity_id}.{event_type}` with `Nats-Msg-Id` set to the event UUID
5. Publishing model updates to `perhap.{context}.{entity_id}._model`

No other integration is required. The bus is the API.

### The Reducer Interface

```
reduce(event_type: String, model: M, event: Event) → (M, []Event)
```

Returns the new model and any new events to publish. The reducer does not validate; validation is a separate concern called by the supervisor after reduction.

### The Validator Interface

```
validate(model: M) → []ValidationError
```

Returns an empty list if the model is valid, or a list of validation errors describing what invariants were violated. Called by the supervisor after every reduction. The validator is declared alongside the reducer as part of the domain service definition.

### Exception Events

When `validate` returns errors, the supervisor emits an exception event:

```json
{
  "id": "...",
  "entity_id": "...",
  "context": "hr",
  "type": "_exception",
  "causation_id": "...",
  "correlation_id": "...",
  "data": {
    "window_uuids": ["uuid-A", "uuid-C"],
    "validation_errors": ["approved requires prior submitted state"],
    "last_valid_uuid": "uuid-A",
    "policy": "reject"
  }
}
```

Exception events flow through the same bus as all other events. They are stored in JetStream. They can trigger automated remediation policies. They route to the frontend via subscription. There are no errors in Perhap — only events.

### Entity Supervisor Lifecycle

The supervisor maintains:

```
snapshot:           M                // cached reduction to snapshot_uuid
snapshot_uuid:      UUID             // UUID of last event folded into snapshot
window:             []Event          // events since snapshot, sorted by UUID
window_uuids:       Set[UUID]        // UUIDs of all events currently in window
last_valid_model:   M                // last reduction that passed validation
valid:              bool             // current reduction valid?
pending:            bool             // window still open? (cleared by idle timeout)
```

**Activation**: triggered by an incoming event for an entity with no active supervisor. The supervisor loads the snapshot and window from the snapshot cache and JetStream. The `window_uuids` set is reconstructed from the loaded window.

**Event processing**:
1. **Deduplication**: if `event.UUID <= snapshot_uuid` or `event.UUID ∈ window_uuids`, discard as duplicate
2. Insert event into window at correct UUID position; add UUID to `window_uuids`
3. Re-reduce: `reduce(snapshot, window[0], window[1], ...)`
4. Validate the result
5. If valid: update `last_valid_model`, set `valid: true`
6. If invalid: set `valid: false`, emit `_exception` event, hold `last_valid_model`
7. Publish model update with full wire format including window UUIDs and valid flag
8. Set `pending: true`; reset idle timer

**Exception event dispatch**: `_exception` events are published to NATS and stored in JetStream but are **not** dispatched to the user-defined reducer. The supervisor handles them at the framework layer. Reducers never receive `_exception` as an event type.

**Window advancement**: `WindowSize` is the maximum number of events to hold in the window. When `window.length > WindowSize`, fold the oldest `ceil(WindowSize / 2)` events into the snapshot:
1. Take `n = ceil(WindowSize / 2)` oldest events from the window
2. `new_snapshot = reduce(snapshot, window[0..n-1])`
3. Update `snapshot_uuid` to `window[n-1].UUID`
4. Remove those events from `window`; remove their UUIDs from `window_uuids`
5. Persist the new snapshot to the snapshot cache

Example: `WindowSize = 100`. When 101 events are in the window, fold the oldest 50, leaving 51. This hysteresis prevents continuous advancement.

**Window closing (pending → false)**: after `IdleTimeout` with no new events, the supervisor sets `pending: false` and publishes a final model update. This signals that the reduction window has stabilized and no further reordering is expected.

**Passivation**: after a second configurable idle period beyond the window close, the supervisor persists its snapshot (including `window_uuids`) and terminates. Re-activated on the next event. Persisting `window_uuids` alongside the snapshot ensures the deduplication set survives passivation.

### Snapshot Cache

The snapshot cache persists the supervisor's snapshot between activations. The interface:

```
save(entity_id, context, domain, snapshot_uuid, snapshot) → Result
load(entity_id, context, domain) → Result(snapshot_uuid, snapshot)
```

The snapshot cache is a performance layer, not the system of record. Error semantics:

- **`Load` returns not-found**: supervisor rebuilds from JetStream full replay. This is expected for new entities and after cache eviction.
- **`Load` returns error**: treat as not-found; log the error; rebuild from JetStream.
- **`Save` returns error**: log the error; continue processing. The snapshot will be rebuilt from JetStream on next activation. Do not halt event processing for a cache write failure.

If the cache is lost entirely, all supervisors rebuild from JetStream on next activation. This is expensive but correct. The reference implementation uses PostgreSQL with a schema of:

```sql
CREATE TABLE perhap_snapshots (
  entity_id      TEXT NOT NULL,
  context        TEXT NOT NULL,
  domain         TEXT NOT NULL,
  snapshot_uuid  TEXT NOT NULL,
  window_uuids   TEXT[] NOT NULL,  -- for dedup set reconstruction
  snapshot_data  JSONB NOT NULL,
  saved_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (entity_id, context, domain)
);
```

### Polyglot Participation

A domain service in any language joins Perhap by implementing the reducer and validator interfaces and following the bus protocol. NATS clients exist for Go, TypeScript, Python, Rust, Java, Elixir, C, C#, Ruby, and others.

---

## 9. Go SDK

The Go SDK is the reference implementation of the domain service specification.

### Core Types

```go
// Event as received from NATS
type Event struct {
    ID            string          `json:"id"`            // UUIDv6, assigned at source
    EntityID      string          `json:"entity_id"`
    Context       string          `json:"context"`
    Type          string          `json:"type"`
    CausationID   string          `json:"causation_id"`
    CorrelationID string          `json:"correlation_id"`
    Data          json.RawMessage `json:"data"`
}

// Reducer function type
type Reducer[M any] func(eventType string, model M, event Event) (M, []Event)

// Validator function type
type Validator[M any] func(model M) []ValidationError

type ValidationError struct {
    Field   string
    Message string
}

// Snapshot cache interface
type SnapshotCache[M any] interface {
    Save(ctx context.Context, entityID, context, domain, snapshotUUID string, model M) error
    Load(ctx context.Context, entityID, context, domain string) (snapshotUUID string, model M, err error)
}

// Domain service configuration
type ServiceConfig[M any] struct {
    Context          string
    Domain           string
    Reducer          Reducer[M]
    Validator        Validator[M]
    SnapshotCache    SnapshotCache[M]
    WindowSize       int           // max events in window before snapshot advancement
                                  // when window.len > WindowSize, oldest ceil(WindowSize/2)
                                  // events are folded into the snapshot. Default: 100.
    WindowIdleClose  time.Duration // idle before pending→false published. Default: 500ms.
    PassivateTimeout time.Duration // idle after window close before supervisor terminates
    LateEventPolicy  LateEventPolicy
}

type LateEventPolicy int
const (
    LateEventReject  LateEventPolicy = iota
    LateEventRebuild
    LateEventAcceptWithFlag
)
```

### Service Lifecycle

```go
svc := perhap.NewService(nc, config)
if err := svc.Run(ctx); err != nil {
    log.Fatal(err)
}
```

`Run` subscribes to `perhap.{context}.*.>` with a NATS queue group, activates entity supervisors as events arrive, and handles graceful shutdown on context cancellation.

### Entity Supervisor (Go)

The Go SDK implements entity supervisors as goroutines with a channel inbox. The service maintains a registry of active supervisors keyed by entity ID. Supervisors passivate after the configured idle timeout.

---

## 10. Frontend SDK (TypeScript)

The frontend SDK is a TypeScript library providing context management, model caching, outbox management, and reducer execution. It is framework-agnostic.

### Jamstack Integration

Perhap is well suited to Jamstack architectures, where content-driven pages are rendered at build time or via CDN edge, and interactivity is added as isolated islands. The recommended stack:

**[Astro](https://astro.build)** — static site generator with first-class markdown support and a component island model. Pages are HTML by default; interactive regions are explicit islands using any supported component framework. Astro's island architecture maps naturally to Perhap's context subscription model: each island corresponds to one or more entity subscriptions.

**[Svelte](https://svelte.dev)** — component framework for interactive islands. Svelte compiles to vanilla JavaScript with no runtime overhead. Its reactive store model is compatible with Perhap's push-based model updates.

**[Lit](https://lit.dev)** — a thin (~5 KB) library for authoring native Web Components with reactive properties and declarative templates. Lit's `connectedCallback` / `disconnectedCallback` lifecycle maps directly to Perhap subscribe / unsubscribe: the component subscribes when it enters the DOM and unsubscribes when it leaves.

The declarative attribute layer (Section 11) works with any of these or with plain HTML. The choice of island framework is independent of Perhap's event and subscription model.

### Web Components

Web Components are a browser-native standard for encapsulated custom elements. They have no framework dependency and interoperate with any surrounding framework or plain HTML.

A Perhap Web Component follows this pattern:

```typescript
import { LitElement, html } from 'lit'
import { customElement, property } from 'lit/decorators.js'
import { watch, tell } from '@perhap/sdk'

@customElement('cart-summary')
class CartSummary extends LitElement {
  @property() context = ''
  @property() entityId = ''

  private model: CartModel | null = null
  private unsubscribe?: () => void

  connectedCallback() {
    super.connectedCallback()
    this.unsubscribe = watch(this.context, this.entityId, (state) => {
      this.model = state.reduction
      this.requestUpdate()
    })
  }

  disconnectedCallback() {
    super.disconnectedCallback()
    this.unsubscribe?.()
  }

  render() {
    if (!this.model) return html`<slot name="loading"></slot>`
    return html`
      <span class="item-count">${this.model.itemCount}</span>
      <button @click=${this.checkout}>Checkout</button>
    `
  }

  private checkout() {
    tell(this.context, this.entityId, 'checkout_started', {})
  }
}
```

```html
<cart-summary context="store" entity-id="018e7a3b-2d9b-6f00-b1a2-000000000001">
  <span slot="loading">Loading cart...</span>
</cart-summary>
```

`connectedCallback` maps to subscribe. `disconnectedCallback` maps to unsubscribe. The component is a self-contained subscription unit.

### SDK Surface

The SDK exposes two primary functions and a configuration mechanism:

```typescript
// Configure once, at application startup
// Reads from <meta name="perhap-server" content="wss://..."> by default
perhap.configure({ server: 'wss://nats.example.com:443', credentials })

// Subscribe to an entity — returns an unsubscribe function
function watch<M>(
  context: string,
  entityId: string,
  onUpdate: (state: ModelState<M>) => void
): () => void

// Publish an event
function tell(
  context: string,
  entityId: string,
  eventType: string,
  data: unknown,
  options?: { causationId?: string }
): void
```

These two functions are sufficient for all component authoring. The full context manager, outbox, and cache machinery runs beneath them.

### Context Manager

The context manager handles the subscription lifecycle for a single entity:

```typescript
interface ModelState<M> {
  reduction: M | null
  pending: boolean    // true when backend window open OR frontend has unconfirmed outbox events
  backendPending: boolean  // the wire-format pending flag from the backend only
  valid: boolean
  reducedThrough: string | null  // UUID of most recent event in reduction
}

async function subscribe<M>(
  nc: NatsConnection,
  context: string,
  entityId: string,
  reducer: FrontendReducer<M>,
  onUpdate: (state: ModelState<M>) => void
): Promise<() => void>
```

`ModelState.pending` is `backendPending || outboxHasPendingEventsFor(entityId)`. The two sources are tracked independently; `pending` is the derived union.

On subscribe:
1. Load reduction from IndexedDB cache (if available), render immediately
2. **Open NATS subscription** on `perhap.{context}.{entityId}.>`, buffering model updates
3. Request current model via `_model.get`; apply as starting state
4. Drain buffered updates: apply any with `reduced_through` UUID newer than hydration
5. Apply any pending outbox events for this entity using the frontend reducer (model-transformation only — cross-entity event emission is suppressed during this replay to avoid duplicating events already in the outbox)

On unsubscribe (context change):
1. Close NATS subscription
2. Leave outbox and cache intact — context may return

### Frontend Reducer

The frontend reducer produces a new model and optionally emits events for other entities. It does not return backend events:

```typescript
type FrontendReducer<M> = (
  model: M,
  event: Event
) => {
  model: M
  events?: OutboxEvent[]
}
```

Frontend reducers are generated from annotated backend reducer definitions. See [Code Generation](#12-code-generation).

### Model Cache

The model cache persists the last confirmed reduction for each entity in IndexedDB:

```typescript
interface CachedModel<M> {
  entityId: string
  context: string
  domain: string
  reducedThrough: string    // UUID
  valid: boolean
  reduction: M
  cachedAt: number
}
```

Cache entries are keyed by `{context}/{entityId}/{domain}`. Written on every confirmed model update. Read on context load for immediate rendering.

### Outbox

The outbox persists unconfirmed events in IndexedDB:

```typescript
interface OutboxEntry {
  id: string               // UUIDv6, assigned at publish time — canonical ordering key
  entityId: string
  context: string
  eventType: string
  data: unknown
  causationId: string
  correlationId: string
  attempts: number
  status: 'pending' | 'inflight' | 'confirmed' | 'failed'
}
```

On reconnection, pending outbox entries are flushed in UUID order. Confirmed entries are retained briefly for causal reference, then purged. The outbox flushes **sequentially** — each event is acknowledged before the next is published. This ensures arrival order at NATS matches the UUID order for same-source events, preventing avoidable reordering.

### Pending State Lifecycle (Frontend)

1. User action → frontend reducer runs → optimistic model update
2. Event written to outbox with status `pending`
3. `ModelState.pending = true` (frontend has unconfirmed outbox event)
4. Event published to NATS (or queued if offline)
5. On NATS ack: outbox entry status → `inflight`
6. On model update received with newer `reduced_through`: update `reduction` and `backendPending` from wire format; update `valid` flag; if all outbox events for this entity are confirmed, `pending` becomes `backendPending` only

### Summary Models

Summary models are lightweight projections of entities that must remain current regardless of the active context — notification counts, cart totals, unread messages:

```typescript
interface SummarySubscription<S> {
  entityId: string
  reducer: FrontendReducer<S>
  onUpdate: (summary: S) => void
}

function registerSummary<S>(config: SummarySubscription<S>): () => void
```

Summary models are subscribed at application startup and remain active for the application lifetime.

**Offline behavior**: when the NATS connection drops, summary subscriptions freeze at their last received value. Their IndexedDB cache entries are read on reconnect to display the last-known value during the reconnection window. On reconnect, summary subscriptions re-request via `_model.get` to catch up before resuming subscription updates.

---

## 11. Declarative Attribute Layer

The declarative attribute layer is a lightweight (~150 lines of JavaScript) enhancement that brings Perhap subscriptions and event publishing to any HTML element via data attributes. No custom component library is required. Standard HTML elements are enhanced in place.

### Configuration

```html
<meta name="perhap-server" content="wss://nats.example.com:443">
<meta name="perhap-credentials" content="...">
<script type="module" src="/perhap.js"></script>
```

The SDK reads server and credential configuration from `<meta>` tags at load time. No JavaScript configuration required for typical deployments.

**Security note**: embedding credentials in a `<meta>` tag is convenient for development but is a plaintext exposure risk in production — credentials appear in page source, browser history, and any analytics or error-tracking tools on the page. The recommended production pattern: server-render the `<meta name="perhap-credentials">` tag dynamically with a short-lived, session-scoped token rather than embedding long-lived credentials statically. The credential issuance mechanism is outside Perhap's scope; the application is responsible for scoping credentials appropriately before delivery to the browser.

### data-watch

`data-watch` subscribes to an entity. It can be placed on any container element.

```html
<div data-watch="store/018e7a3b-2d9b-6f00-b1a2-000000000001">
  <!-- children may use data-bind and data-tell -->
</div>
```

The value is `{context}/{entityId}`. On insertion into the DOM, the SDK calls `watch()` for the specified entity. On removal from the DOM, the subscription is automatically cancelled. The developer never calls unsubscribe explicitly.

### data-bind

`data-bind` sets the `textContent` of an element to a named property of the watched entity's current reduction.

```html
<div data-watch="store/018e7a3b-2d9b-6f00-b1a2-000000000001">
  <span data-bind="itemCount"></span>
  <span data-bind="totalPrice"></span>
</div>
```

For nested objects, use dot notation:

```html
<span data-bind="shipping.address.city"></span>
```

**Attribute binding**: to bind a reduction property to an HTML attribute rather than `textContent`, use `data-bind-attr`:

```html
<!-- bind avatarUrl to the src attribute -->
<img data-bind-attr="src:avatarUrl" alt="User avatar">

<!-- bind to multiple attributes -->
<a data-bind-attr="href:profileUrl" data-bind="displayName"></a>

<!-- disable button when reduction says unavailable -->
<button data-bind-attr="disabled:outOfStock" data-tell="added_to_cart">Add</button>
```

The value is `attributeName:propertyPath`. Boolean attributes (`disabled`, `hidden`, `checked`) are set/removed based on the truthiness of the reduction property.

**Meta-properties** (prefixed with `@`) bind to model metadata rather than reduction fields:

| Meta-property | Value |
|---|---|
| `@pending` | `"true"` or `"false"` |
| `@valid` | `"true"` or `"false"` |
| `@entity-id` | the entity UUID |
| `@context` | the context name |
| `@reduced-through` | the UUID of the most recent event |

```html
<span data-bind="@pending" hidden></span>
```

### data-tell

`data-tell` publishes a Perhap event when the element is activated. Context and entity ID are inherited from the nearest `data-watch` ancestor.

**On a button** — fires on click:

```html
<button data-tell="checkout_started">Checkout</button>
```

**On a form** — fires on submit; calls `event.preventDefault()` (the form does not navigate or POST); serializes `FormData` as the event's data payload:

```html
<form data-tell="item_added">
  <input name="productId" value="prod-123">
  <input name="quantity" type="number" value="1">
  <button type="submit">Add to Cart</button>
</form>
```

**On an input** — fires on change:

```html
<input data-tell="quantity_changed" name="quantity" type="number">
```

The event's `data` field is populated:
- **Form**: `FormData` serialized as an object. Multi-value fields (e.g., `<select multiple>`, repeated field names) serialize as arrays. File inputs are excluded from serialization — use Web Components for file upload. Field names are used as-is (no structure reconstruction from `address[city]`-style names).
- **Input**: `{ name: fieldName, value: currentValue }`
- **Button**: `{}` (no data; use `data-tell-data` for static payloads)

```html
<!-- static payload on a button -->
<button data-tell="item_removed" data-tell-data='{"productId":"prod-123"}'>
  Remove
</button>
```

### Scope Resolution

`data-tell` resolves its target entity by walking up the DOM tree to find the nearest `data-watch` ancestor:

```html
<div data-watch="store/cart-123">         <!-- scope A -->
  <span data-bind="itemCount"></span>

  <div data-watch="store/product-456">    <!-- scope B -->
    <span data-bind="name"></span>
    <span data-bind="price"></span>
    <!-- resolves to scope B: store/product-456 -->
    <button data-tell="added_to_wishlist">Save</button>
  </div>

  <!-- resolves to scope A: store/cart-123 -->
  <button data-tell="checkout_started">Checkout</button>
</div>
```

There is no `data-bind-source` equivalent — `data-bind` always reads from the nearest `data-watch` ancestor. If you need to display data from one entity while sending events to another, use sibling `data-watch` scopes or a Web Component. The deliberate constraint is that a single `data-watch` element is always both the display source and the tell target; mixing them within one element is a sign the markup should be restructured.

For the unusual case where a control must tell a different entity than its nearest watch ancestor, use `data-tell-target` to specify the target explicitly:

```html
<div data-watch="store/cart-123">
  <button
    data-tell="support_requested"
    data-tell-target="support/session-789"
  >Help</button>
</div>
```

### Multiple Entities

**Nested scopes** — the natural model for related entities where one is contextually inside another:

```html
<div data-watch="store/cart-123">
  <span data-bind="itemCount"></span>

  <div data-watch="store/product-456">
    <span data-bind="name"></span>
    <span data-bind="price"></span>
    <button data-tell="added_to_wishlist">Save</button>
  </div>

  <button data-tell="checkout_started">Checkout</button>
</div>
```

**Sibling scopes** — the natural model for parallel independent entities, such as a product comparison view:

```html
<div class="comparison">
  <div data-watch="store/product-111">
    <span data-bind="name"></span>
    <span data-bind="price"></span>
    <button data-tell="added_to_cart">Add</button>
  </div>

  <div data-watch="store/product-222">
    <span data-bind="name"></span>
    <span data-bind="price"></span>
    <button data-tell="added_to_cart">Add</button>  <!-- different entity -->
  </div>
</div>
```

Each `data-watch` is its own subscription with its own lifecycle. The same event type name on sibling `data-tell` elements correctly targets different entity IDs.

### CSS State Rendering

The SDK writes the current model state as data attributes on the `data-watch` element:

```html
<!-- while pending -->
<div data-watch="store/cart-123" data-pending="true" data-valid="true">
  ...
</div>

<!-- confirmed, valid -->
<div data-watch="store/cart-123" data-pending="false" data-valid="true">
  ...
</div>

<!-- confirmed, invalid -->
<div data-watch="store/cart-123" data-pending="false" data-valid="false">
  ...
</div>
```

CSS selectors handle all state-based presentation:

```css
[data-watch][data-pending="true"] {
  opacity: 0.7;
}

[data-watch][data-valid="false"] {
  outline: 2px solid var(--color-error);
}

[data-watch][data-pending="true"] [data-tell] {
  pointer-events: none;
}
```

No JavaScript state checks needed in CSS or templates. Attributes are the contract between the SDK and the stylesheet.

Note: `data-pending` and `data-valid` hold the string values `"true"` or `"false"` — this is standard HTML attribute behavior. CSS attribute selectors (`[data-pending="true"]`) work correctly with string values. In JavaScript, check `element.dataset.pending === "true"` rather than treating the attribute as boolean.

### Subscription Lifecycle

The SDK uses a `MutationObserver` on the document body to watch for elements entering and leaving the DOM:

1. **Element with `data-watch` added to DOM** → `watch()` called → subscription active
2. **Element with `data-watch` removed from DOM** → `unsubscribe()` called → subscription cancelled; outbox and cache for the entity are left intact (context may return)
3. **`data-watch` attribute value changed on an existing element** → treated as a context change: old subscription cancelled (outbox and cache for the old entity left intact), new subscription opened for the new entity
4. **`data-watch` element moved within the DOM** → no change (connected the whole time)

The developer never manages subscription lifecycle explicitly. DOM presence equals active subscription.

An optional `IntersectionObserver` mode is available for elements that may be off-screen for extended periods (e.g., items in a long list):

```html
<div data-watch="store/product-456" data-watch-visible-only>
  ...
</div>
```

When `data-watch-visible-only` is set: the NATS subscription remains open at all times (preserving connection and ensuring no updates are missed), but DOM updates (`data-bind` writes, `data-pending`/`data-valid` attribute updates) are deferred while the element is outside the viewport. When the element becomes visible, the latest received model state is applied immediately. This is a rendering optimization only — it does not reduce NATS connection count or message delivery.

### Initialization Sequence

On page load, the SDK:
1. Reads `<meta name="perhap-server">` for the NATS WebSocket endpoint
2. Reads `<meta name="perhap-credentials">` for connection credentials
3. Connects to NATS via NATS.ws
4. Scans the initial DOM for all `data-watch` elements and opens subscriptions
5. Starts `MutationObserver` for ongoing lifecycle management

Elements already in the DOM at load time are treated as if they were just added.

---

## 12. Code Generation

Perhap provides a code generation tool (`perhap-gen`) that derives TypeScript types and frontend reducers from annotated Go reducer definitions. Generation uses the Claude API.

### Annotation

Backend reducers intended for frontend participation are annotated:

```go
// +perhap:frontend
type CartReducer struct{}

func (r CartReducer) Reduce(eventType string, model Cart, event Event) (Cart, []Event) {
    // ...
}
```

Backend-only reducers are not annotated and produce no frontend artifacts.

### What Gets Generated

For each annotated reducer, `perhap-gen` produces:

1. **TypeScript model type** — derived from the Go struct
2. **TypeScript event types** — a discriminated union of all event types the reducer handles
3. **TypeScript frontend reducer** — the model transformation cases, stripped of event emission logic
4. **TypeScript tests** — derived from backend test fixtures, covering each event type

### Generation Process

`perhap-gen` reads annotated Go files, extracts type definitions and reducer logic, and submits them to the Claude API with a structured prompt:

```
Given this Go reducer and model definition:
[source]

Generate:
1. TypeScript interfaces for the model and all event data types
2. A frontend reducer function (model, event) => { model, events? }
   implementing only the model transformation, not event emission
3. Type mappings per the table below
4. Tests for each event type using these fixtures: [fixtures]

The output must compile without errors under strict TypeScript.
```

Generated files include a hash of the Go source in a header comment. CI fails if the hash is stale.

### Type Mapping

| Go Type | TypeScript Type | Notes |
|---|---|---|
| `string` | `string` | |
| `int`, `int32` | `number` | |
| `int64` | `bigint` | JavaScript `number` cannot represent all int64 values above 2^53. `bigint` is required for correctness. Generated code uses `BigInt()` for parsing. |
| `float32`, `float64` | `number` | |
| `bool` | `boolean` | |
| `time.Time` | `string` | RFC 3339 / ISO 8601 with timezone offset. Not parsed to `Date` — treated as an opaque string. Use a date library in the UI layer if formatting is needed. |
| `*T` | `T \| null` | |
| `[]T` | `T[]` | |
| `map[K]V` | `Record<K, V>` | |
| `json.RawMessage` | `unknown` | |

### Human Review

Generated code is committed to the repository and reviewed like any other code. It is not applied automatically. The generator is a drafting tool, not a deployment pipeline.

---

## 13. Pending State and Validity

Pending state and validity are distinct concepts that together define the trustworthiness of a model.

### Two Flags, Four States

**`pending`** has two layers:
- In the **model wire format**: `pending: true` means the backend supervisor's reduction window is still open. The window closes after `WindowIdleClose` passes with no new events, at which point the supervisor publishes a final model update with `pending: false`.
- In **`ModelState` on the frontend**: `pending: true` means the wire-format `pending` is true OR the frontend has unconfirmed outbox events for this entity. These are tracked separately; the UI sees their union.

**`valid`** indicates that the current reduction satisfies all domain invariants. An invalid model holds at the last valid reduction.

| Wire `pending` | `valid` | Meaning |
|---|---|---|
| `true` | `true` | Window open, reduction good so far — normal in-progress state |
| `true` | `false` | Window open, current reduction is invalid, exception emitted |
| `false` | `true` | Window closed, confirmed correct — the definitive state |
| `false` | `false` | Window closed, confirmed invalid, awaiting remediation |

This is **true eventual consistency**: the served value is always a valid state that actually existed. It may be stale when `valid: false`, but it is never a state that never existed.

### Async Boundaries

The most important use of pending state is at **async boundaries** — where processing passes to a human or external system:

- A manager approval step
- A payment processor response
- An inventory reservation
- A compliance review

At these boundaries, pending state may persist for extended periods. This is correct. A pending time-off request is not an approved time-off request. Showing a confirmed outcome before it is confirmed is a data integrity error.

### Backend-Only Aggregates as Async Boundary Signals

A backend-only aggregate is one whose internal model is never directly rendered in a frontend context. The frontend needs only to know the *outcomes* of its processing (expressed as events it emits downstream), not its internal state. The test: "Does any frontend component need to read a field from this entity's reduction directly?" If yes, frontend-visible. If no, backend-only.

When an event chain passes through a backend-only aggregate before reaching a frontend-visible entity, the backend-only aggregate is often an async boundary. The frontend cannot optimistically update the downstream entity because it cannot replicate the opaque transformation. The correct frontend behavior is to show the downstream entity as pending until the confirmed model arrives.

### Validity and Data Sanity

All events are stored. An event that produces an invalid state is not rejected — it is stored, validation runs on the resulting reduction, and if invalid, an exception event is emitted and the model holds at the last valid state. The event log is complete. The model is valid. These properties are maintained simultaneously.

---

## 14. Cross-Entity Events

A frontend reducer may emit events for entities other than the one it is reducing. These are optimistic updates to related entities.

### Emission

```typescript
case 'item_added':
  return {
    model: addItemToCart(model, event.data),
    events: [{
      entityId: event.data.productId,
      context: 'store',
      eventType: 'inventory_reserved',
      data: { qty: event.data.qty },
      causationId: event.id
    }]
  }
```

Emitted events go directly to the outbox, keyed to their target entity.

### Lazy Application

Emitted events for entities not in the current context sit in the outbox until that entity comes into context. At context load time, the context manager applies all pending outbox events for that entity on top of the confirmed model. The pending optimistic state is applied immediately, before the subscription opens.

### Causal Ordering

Outbox entries include a `causationId`. When flushing the outbox, caused events are not published before their causing event is confirmed. This preserves causal ordering across entity boundaries.

### Depth Limit

Frontend reducers emit events at most one level deep. Recursive frontend event chains are not supported. Deeper cascades are handled by backend services and delivered via subscription.

---

## 15. Offline Operation

### Local Stores

The frontend maintains two IndexedDB stores:

**Model cache**: the last confirmed reduction for each entity. Keyed by `{context}/{entityId}/{domain}`.

**Outbox**: pending and inflight events. Keyed by event UUID.

### Offline Event Handling

When the NATS connection is unavailable:

1. User actions produce optimistic model updates via frontend reducers
2. Events are written to the outbox with status `pending`
3. NATS publish is deferred
4. The model the user sees is: confirmed model from cache + reduce(pending outbox events)

### Reconnection

On NATS reconnect:

1. Flush pending outbox entries in UUID order, sequentially
2. Re-establish model subscriptions for active context
3. Request current model via `_model.get`
4. If `reduced_through` UUID of received model is newer than the earliest pending outbox event, optimistic state may have been superseded — surface this if the entity is visible

### Conflict Handling

Offline operation assumes sequential ownership. If another client modified the same entity while this client was offline, the backend's confirmed model is authoritative. The framework does not merge conflicting state. The confirmed model is presented; visible discrepancies are left to the application to handle.

---

## 16. Development Process

Perhap is not only a runtime framework. It is a development methodology rooted in Domain-Driven Design. The quality of a Perhap application is determined primarily by the quality of its event model.

### Phase 1: Domain Discovery

Domain discovery produces the ubiquitous language and the event model. It precedes any code.

#### Event Storming

[Event storming](https://www.eventstorming.com) (Alberto Brandolini) is the primary discovery tool. A facilitated session with domain experts, developers, and stakeholders produces a shared understanding of the domain.

The outputs that matter to Perhap:

- **Domain events** (orange): things that happened, past tense — these become Perhap events directly
- **Triggers** (blue): what initiates each event — a user action, an automated rule, an external signal. In event storming these are often called "commands"; in Perhap they are themselves events. A user's intention is expressed as a past-tense event: `time_off_requested`, not "request time off."
- **Aggregates** (yellow): entities that own events and maintain models
- **Policies** (purple): automated reactions — "when X, then Y" — these become backend domain service reducers
- **Read models** (green): projections needed by the UI — these become frontend models

#### Ubiquitous Language

Every term produced in event storming becomes part of the ubiquitous language. Terms must be agreed upon by domain experts and used consistently in:

- Event type names (`time_off_requested`, not `leaveRequest`)
- Entity names (`Employee`, not `User` or `StaffMember`)
- Field names (`requestedDays`, not `days` or `numberOfDays`)
- Code, documentation, and conversation

The ubiquitous language is enforced in code. Event type strings, model field names, and subject names must match the agreed vocabulary exactly. Divergence between code and language is a defect.

### Phase 2: Gap Analysis

After the initial event storm, domain experts invariably miss events. The gaps are predictable: compensating events, lifecycle transitions, audit events, boundary events, and policy triggers. Identifying these before coding prevents structural problems that are expensive to fix later.

#### GenAI-Assisted Gap Analysis

Perhap provides a gap analysis tool that submits the event storm output to an LLM and receives:

1. **Missing compensating events**: for every positive outcome, what are the negative and reversal paths?
2. **Missing lifecycle transitions**: states with no entry or exit event
3. **Orphaned events**: events with no reducer, or reducers referencing undefined events
4. **Missing policies**: events that should trigger automated responses but don't
5. **Cross-context leakage**: events carrying data belonging to a different bounded context
6. **Commutativity concerns**: event pairs that may arrive concurrently and whose combined reducer may not be commutative
7. **Proposed event definitions**: draft definitions for each identified gap
8. **Questions requiring domain expertise**: gaps the analysis cannot resolve without domain knowledge

The analysis produces proposals, not decisions. Every proposed addition must be reviewed and validated by domain experts before it enters the event model. The gap analysis surfaces what might be missing; the domain experts determine what is actually missing.

#### The Review Cycle

```
Event storm → Gap analysis → Domain expert review → Updated event model
                                     ↑                        ↓
                               Still gaps?  ←─────────────────┘
```

Iterate until the event model captures the domain completely.

### Phase 3: Domain Modeling

With a complete event model, define:

- **Bounded contexts**: which entities and events belong together
- **Aggregate boundaries**: which entities are reduced independently
- **Backend-only aggregates**: infrastructure, no frontend representation
- **Frontend-visible aggregates**: which aggregates the UI renders
- **Summary models**: lightweight projections always visible regardless of context
- **Async boundaries**: where human or external processing creates genuine uncertainty
- **Late event policies**: per-aggregate declaration of how late events are handled
- **Commutativity requirements**: which event pairs require commutative reducers

### Phase 4: Implementation

1. Implement backend reducers and validators (Go, or target language)
2. Write reducer tests against event fixtures, including out-of-order delivery scenarios
3. Annotate frontend-visible reducers for code generation
4. Run `perhap-gen` to generate TypeScript types and frontend reducers
5. Review generated code
6. Implement frontend UI using the declarative attribute layer, Web Components, or both

For content-driven pages, the declarative attribute layer (Section 11) is often sufficient — add `data-watch`, `data-bind`, and `data-tell` attributes to existing markup. For complex interactive regions, author Web Components using `watch()` and `tell()` directly, and mount them as islands in Astro or the host framework of choice.

Code generation is a translation step, not a design step.

### Phase 5: Iteration

New requirements produce new events. New events go through the same process. The event log is additive — new event types are added, existing ones are never modified.

---

## 17. Example Domain

### Time-Off Management

This domain illustrates the key Perhap patterns: user-initiated events, a backend-only approval aggregate, two separate frontend applications, automated and manual approval paths, async boundaries, pending state, and validity.

### Entities

**Employee** — a person who can request time off  
**TimeOffRequest** — a request, submitted by an employee  
**ApprovalDecision** — manager's decision (backend-only)  
**Schedule** — confirmed schedule including approved time off  
**ApprovalQueue** — manager's view of pending requests (read model)

### Events

```
time_off_requested        Employee submits a request
time_off_cancelled        Employee cancels before decision

approval_initiated        System begins approval process
accrual_balance_checked   Balance verified
balance_insufficient      Request invalid due to insufficient balance
approval_auto_approved    Rules engine approved automatically
approval_queued           Routed to manager for review

time_off_approved         Manager approves
time_off_denied           Manager denies

schedule_updated          Approved time off added to schedule
schedule_conflict_found   Conflict detected; approval cannot complete

_exception                Reduction produced invalid state (framework-generated)
```

### Bounded Contexts

**employee-app**: TimeOffRequest (FE+BE), Schedule (FE+BE)  
**manager-app**: ApprovalQueue (FE+BE), ApprovalDecision (BE-only)

`ApprovalQueue` is a Perhap entity, not a passive projection. In Perhap, "read model" means an entity whose primary purpose is to aggregate information from other entities for display — it has its own event history, its own reduction, and its own entity UUID. Each manager has their own `ApprovalQueue` entity, keyed by the manager's user ID. The `approval_queued` event updates the manager's queue; `time_off_approved` and `time_off_denied` remove the entry. The manager-app subscribes to this entity exactly as it would any other. There is no special projection infrastructure.

### Event Flows

**Manual approval:**
```
Employee publishes time_off_requested
  → TimeOffRequest: pending: true, valid: true

Backend:
  time_off_requested
    → approval_initiated
    → accrual_balance_checked
    → approval_queued           (balance sufficient)
      OR balance_insufficient   (balance insufficient)

Manager app receives approval_queued → renders pending request
Manager publishes time_off_approved

Backend:
  time_off_approved
    → schedule_updated          (no conflict)
      OR schedule_conflict_found (conflict detected → _exception)

Employee app receives schedule_updated
  → Schedule: pending: false, valid: true
```

**Automated approval:**
```
Employee publishes time_off_requested
  → TimeOffRequest: pending: true, valid: true

Backend:
  time_off_requested
    → approval_initiated
    → accrual_balance_checked
    → approval_auto_approved
    → schedule_updated

Employee app receives schedule_updated (~200ms)
  → Schedule: pending: false, valid: true
```

### Pending and Valid States in This Domain

| Situation | Entity | pending | valid |
|---|---|---|---|
| Request submitted, awaiting decision | TimeOffRequest | true | true |
| Auto-approval in progress | Schedule | true | true |
| Awaiting manager (may take days) | TimeOffRequest | true | true |
| Schedule conflict on approval | Schedule | false | false |
| Approved and scheduled | Schedule | false | true |
| Denied | TimeOffRequest | false | true |

### The Async Boundary

The human manager is the async boundary in the manual path. `ApprovalDecision` is backend-only precisely because it represents a process opaque to both frontend applications. Neither app knows how approval works — only what the outcome was. The `Schedule` entity remains pending, correctly, until the manager acts.

### Commutativity in This Domain

`time_off_requested` and `time_off_cancelled` are potentially concurrent: an employee might cancel very shortly after submitting, with both events in flight simultaneously. The `TimeOffRequest` reducer should be designed so that applying these two events in either order produces the same result: a cancelled request. This is achieved by reducing to state facts (`submitted_at`, `cancelled_at`) and deriving the status from them at read time, rather than implementing a strict state machine transition.

---

*End of Specification v0.4.0-draft*
