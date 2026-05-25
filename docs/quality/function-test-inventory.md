# Function Test Inventory

This beta contains a growing set of unit and regression tests.

## Target

Every exported function should have either:

- a direct unit test
- an integration test reference
- a documented reason why direct testing is not useful

## Current focus areas

| Area | Status | Notes |
|---|---|---|
| Shared MQTT transport | In progress | Add tests for topics, envelope handling and reconnect behavior. |
| Shared UI charts | Covered by JavaScript tests | Continue adding regression tests for hover and legend behavior. |
| Configuration diff and apply | In progress | Add tests for validation, snapshots and rollback. |
| Log aggregation | In progress | Add tests for grouping and severity filters. |
| Connection graph rendering | In progress | Existing regression tests should be expanded. |
| Presense generators | In progress | Add tests for each generator mode. |
| Sense protocol readers | In progress | Add tests for Modbus and OPC UA parsing behavior. |
| Dispense routing | In progress | Add tests for filtering and target topic mapping. |
