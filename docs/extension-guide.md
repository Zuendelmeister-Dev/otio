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

## Add a custom Dispense module

Use Dispense when you need to forward OT.io data to another system.

Steps:

1. define the target system
2. implement the target writer
3. define route and filter configuration
4. subscribe to the shared input transport
5. map messages to the target format
6. publish health, status, logs and errors
7. expose UI/API status
8. add tests for filtering, mapping and forwarding

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
- tests
