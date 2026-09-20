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
| Shared MQTT transport | Unit coverage expanded | Timeout/error/success, JSON failures, hooks and IPv6 covered; real broker reconnect tests remain. |
| Shared UI charts | Covered by JavaScript tests | Continue adding regression tests for hover and legend behavior. |
| Configuration diff and apply | In progress | Add tests for validation, snapshots and rollback. |
| Log aggregation | In progress | Add tests for grouping and severity filters. |
| Connection graph rendering | In progress | Existing regression tests should be expanded. |
| Presense generators | In progress | Add tests for each generator mode. |
| Sense protocol readers | Unit coverage expanded | Local TCP tests cover Modbus success and malformed responses; cancellation remains a gap. |
| Dispense routing | Unit coverage expanded | Filtering, target topic mapping, non-numeric values, bounded history, invalid config and publish failures covered. |

See the [September 2026 review](review-2026-09.md) for measured coverage and integration-test limitations.
