# Next beta — protocols, diagnostics and Kubernetes HA

Status: prepared for review; no release version or tag assigned. OT.io remains a private beta.

## Added

- Protocol Lab with native Modbus TCP, RTU-over-TCP and OPC UA Binary round trips, MQTT/AMQP sampling and simulator controls. Experimental readers cover S7, EtherNet/IP, BACnet/IP, KNXnet/IP and IEC 104. See the [protocol matrix](../../protocols.md) for exact operations and exclusions.
- Protocol-specific configuration forms alongside JSON, editable Presense generator settings, and copyable Sense source mappings.
- An integrated PostgreSQL viewer in Lense: table/column discovery, restricted read-only SELECT queries, bounded previews and CSV extracts. Readable scrolling tables and a full-value dialog handle long text and JSON. No separate database-client container is required.
- Kubernetes/Kustomize (Example 05), Helm (Example 06) and an active/passive HA example (Example 07).
- HA supervision for Sense/Lense with Kubernetes Leases, healthy standby pods and leader-only Service routing. The HA stack includes a three-node RabbitMQ MQTT cluster and three CloudNativePG instances with synchronous replication.
- Kubernetes Secrets for shared MQTT credentials, a complete PostgreSQL DSN override, read-only Helm-managed Sense configuration and transactional schema migration locking.
- CI workflows for unit/static checks, coverage, deployment smoke tests and the HA failover exercise.

## Security

- Update Protocol Lab's AMQP client from 1.10.0 to 1.13.0 to address [GO-2026-6372](https://pkg.go.dev/vuln/GO-2026-6372), an oversized broker-payload issue. Rebuild Protocol Lab images to include the fix.

## Improved and fixed

- Consistent graph navigation, horizontal scrolling/dragging, clearer component grouping, graph placement and Lense-to-PostgreSQL connectivity.
- Retained status history and explicit empty states when selecting non-metric namespace topics.
- Protocol Lab navigation and service-unavailable feedback.
- MQTT timeout reporting, Modbus response validation and IPv6 endpoint handling.
- Dispense forwarding of non-numeric payloads, invalid configuration handling and forwarding counters.
- Shared UI regression tests, native protocol tests, configuration/API tests, migration tests and failure-recovery checks.

## Upgrade and compatibility

Rebuild the selected Compose example (`docker compose -f examples/01-local-docker-compose/docker-compose.yml up --build -d`) and reload existing browser tabs for the updated shared assets. Keep existing data volumes; `down -v` is not an upgrade step. Lense applies its schema changes at startup in a transaction protected by an advisory lock. Back up persistent data before updating a deployment with valuable data.

Existing `modbus-tcp` and HTTP-demo `opcua` configurations remain supported. Native OPC UA uses `lab-opcua-tcp`; the old HTTP demo is not silently reinterpreted as native OPC UA. Existing JSON editing remains available.

Mosquitto remains the MQTT broker in the standard examples. RabbitMQ in Protocol Lab serves AMQP; the separate HA example uses its MQTT plugin for clustering. Example 07 uses Helm-managed configuration instead of per-pod UI writes. A rolling leader replacement still has a takeover gap.

Development requires Go 1.27.1 and Node.js 24. Protocol Lab builds with Go 1.27.1; the other app images use Go 1.26.8. The charts and HA operators have their own documented prerequisites.

## Known limits

- IP connectivity alone does not imply implementation: HART-IP, DNP3 and IEC 61850/MMS are not implemented. Non-IP fieldbuses require suitable gateways. Physical-device interoperability for experimental drivers is unverified.
- Native OPC UA simulator security is lab-only; production authentication/TLS, backup and operational hardening remain separate work.
- HA demonstrates automatic recovery after the exercised pod/application failures. It does not establish fencing against partitions, physical-host resilience, exactly-once processing or zero-loss delivery.
- Sense/Lense still use MQTT QoS 0. Dispense's buffer is chart history, not a persistent forwarding queue. Failed or disconnected forwarding can lose messages.
- The database viewer intentionally excludes joins, arbitrary functions, multi-statement SQL and writes. It shares Lense's access boundary and is not an administrative SQL client.

See [verification](VERIFICATION.md), [UI workflows](../../ui-workflows.md) and [HA validation](../../../examples/07-kubernetes-ha/VALIDATION.md).
