# OT.io Extension Guide

OT.io is designed as a modular monorepo. Shared functionality belongs in shared packages. Application modules should stay thin and focused.

## Shared functionality

Use shared packages for cross-cutting functionality:

- MQTT and transport helpers
- topic conventions
- message envelopes
- configuration validation helpers
- UI chart and component helpers
- logging formats
- test helpers

## Add a custom Presense module

Use Presense when you need a simulator for demos or tests.

Steps:

1. create `apps/presense-<protocol>`
2. implement the protocol server
3. expose generated values
4. use environment variables for simulator configuration
5. add a Dockerfile
6. add a service to `docker-compose.yml`
7. add a Sense source configuration that reads the simulator
8. add tests for generator behavior and protocol output

## Add a custom Sense module

Use Sense when you need to read a source system. This is the primary extension point for real OT environments because new source protocols and vendor-specific data models will be common.

Steps:

1. create or extend a source reader
2. define JSON configuration for the protocol
3. read values from the endpoint
4. map values into OT.io metric messages
5. publish through the shared transport package
6. publish health, status, logs and errors
7. expose UI/API status
8. add tests for configuration parsing, polling and publish behavior

For a protocol extension, implement `Reader` and optionally `Subscriber` under `apps/sense/internal/protocols`, then register those implementations in `NewRegistry`. Add its capability, supported read modes and form presets to [`shared/protocolmeta/catalog.json`](../shared/protocolmeta/catalog.json). Run `node scripts/generate-protocol-capabilities.cjs` to update the checked-in browser bundle and Sense capability table. The validator checks the registered implementation; catalog metadata does not implement a driver. Use the [Sense implementation guide](sense.md#writing-a-custom-sense-extension) for the exact interfaces.

## Add a custom Dispense output

Use Dispense when you need to forward OT.io data to another system.

Steps:

1. define the target system and what its successful acceptance means
2. implement `outputWriter` in `apps/dispense`, honoring the context deadline
3. extend output configuration validation and writer construction
4. reuse input subscription, routing, filtering and the bounded delivery worker
5. classify failures with `deliveryError` as terminal, retryable or delivery-uncertain
6. expose accurate output readiness and status without credentials
7. add tests for acceptance, retry limits, cancellation and duplicate behavior

MQTT and HTTP/HTTPS POST writers are included. The queue is process-local and bounded; it is not a durable outbox. HTTP retries reuse one message ID, while MQTT QoS 0 cannot prove receipt by the target. New writers must document their own acceptance and duplicate semantics. See [adding another output](dispense.md#adding-another-output) and [delivery rules](dispense.md#delivery-retries-and-duplicates).

## Design rule

Do not duplicate shared behavior inside agents. If multiple modules need the same behavior, move it to `shared`.

## Module-specific extension guides

The module documents contain more concrete implementation guidance:

- [IoT Presense extension guide](presense.md#writing-a-custom-presense-extension)
- [IoT Sense extension guide](sense.md#writing-a-custom-sense-extension)
- [IoT Dispense extension guide](dispense.md#writing-a-custom-dispense-extension)


## Sense runtime shell

The Sense runtime is meant to be reused. For new source protocols, avoid copying the complete Sense app. Add only the protocol-specific code below `apps/sense/internal/protocols`.

The generic runtime already provides:

- HTTP API
- UI routing
- configuration workflow
- source status handling
- MQTT publishing through shared transport helpers
- logs
- quick metrics

A new protocol normally needs only:

- a reader implementation
- optionally a subscriber implementation
- configuration examples
- a capability entry and regenerated form/documentation assets
- tests
