# Quality Review

This document records current quality expectations for the OT.io beta.

## Design principles

- Shared functionality belongs in `shared`.
- Agents should stay thin and focused.
- UI components and charts should be reused across modules.
- Configuration workflows should be consistent across modules.
- Tests should cover exported functions and important behavior.

## Current refactoring direction

The MQTT implementation has been moved into shared functionality so Sense, Lense and Dispense do not duplicate transport behavior.

Further candidates for shared packages:

- configuration validation and diff handling
- log aggregation
- health snapshots
- connection graph data structures
- common UI pages and cards
- common route/filter logic

## SOLID-oriented review

| Principle | Current direction |
|---|---|
| Single Responsibility | Each module has a focused role: simulation, collection, observation or distribution. |
| Open/Closed | New protocols and targets should be added as modules without changing unrelated modules. |
| Liskov Substitution | Future protocol readers and target writers should follow common interfaces. |
| Interface Segregation | Modules should depend on small contracts, not broad application internals. |
| Dependency Inversion | App modules should depend on shared interfaces and packages, not duplicated infrastructure code. |
