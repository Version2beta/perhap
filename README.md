# Perhap

An event-sourced, reducer-based framework for full-stack reactive applications.

Events are the primary data. State is a derived view. The same event language spans browser to backend to any downstream service, with NATS at the center and generated frontend reducers derived from backend definitions.

## Specification

[spec/specification.md](spec/specification.md)

## Status

Pre-implementation. Specification in progress.

## Lineage

Perhap began as an Elixir framework for reactive domain-driven design. This is its second realization: language-agnostic, bus-centered, with Go as the reference backend SDK and TypeScript as the frontend SDK.

The original Elixir implementation is at [../perhap_elixir](../perhap_elixir).
