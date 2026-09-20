# Local validation — 2026-09-19

Validated on Windows with Docker Desktop, kind 0.31.0 / Kubernetes 1.35.0, one control-plane container and three worker containers. CloudNativePG 1.28.4, RabbitMQ Cluster Operator 2.23.0 and cert-manager 1.21.2 were installed in the isolated `otio-ha` test cluster.

- All repository Go tests and shared JavaScript checks passed via `scripts/test.ps1`.
- `go vet` passed for ha-agent, Sense, Lense and shared/mqttx.
- Both the existing Helm chart tests and `scripts/ha-chart.test.js` passed.
- Linux application images built successfully, including their Go tests.
- Rolling updates of Sense and Lense completed with both replicas Ready. Only the elected leader carried the active Service label.
- The final `test-failover.ps1` run passed for Sense, Lense, one MQTT broker and the PostgreSQL primary. For each application, a different pod UID took the Lease and became the sole ready Service endpoint. After every recovery, the script recorded a fresh database baseline and observed additional metric rows.
- PostgreSQL promoted a different primary. Deleting its pod gracefully took approximately three minutes because of the operator's default 180-second smart-shutdown window. This is an observed result, not a recovery-time SLA.

The tests revealed and fixed concurrent schema initialization (now transactionally serialized with a PostgreSQL advisory lock), PowerShell command argument forwarding, and standby readiness/Service selection. The separate GitHub Actions workflow reproduces the cluster exercise; it has been added but has not been executed on GitHub in this local session.

Not tested: physical-worker failure, control-plane loss, network partitions, long process pauses, disk loss, backup recovery, or zero-loss/duplicate-free telemetry. MQTT remains QoS 0. Dispense has no persistent forwarding queue. See README.md for the scope and operating limits.
