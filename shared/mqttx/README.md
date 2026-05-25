# Shared MQTT transport helpers

This package contains MQTT functionality that is shared by OT.io modules.

It intentionally keeps only transport-level behavior here:

- broker URL creation
- client option defaults
- bounded connect handling
- JSON publish helpers
- topic builders and parsers
- connection checks

Application-specific behavior stays inside the runtime modules:

- Sense decides what to read and when to publish.
- Lense decides what to store and how to render health.
- Dispense decides what to filter and forward.

This split keeps the monorepo useful: common transport contracts live once, while each agent keeps its own domain logic.
