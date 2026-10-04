# ha-agent

A direct-process supervisor for the Sense and Lense active/passive Kubernetes example. Default application entrypoints are unchanged; HA containers override them with `/app/ha-agent -- /app/sense` or `/app/ha-agent -- /app/lense`.

Required environment: `HA_LEASE_NAME`, `POD_NAMESPACE`, `POD_NAME`, `POD_UID`, `HA_READY_URL` (loopback application health URL). The pod must have in-cluster Kubernetes credentials and permission to get/update its pre-created Lease and patch pod labels in its namespace. Missing configuration fails closed. Each collector group and Lense group needs a distinct Lease; pod UIDs distinguish process hosts. The Lease is not released early at shutdown.

Port 8099 exposes `/live` for both active and standby. `/ready` reports healthy standbys and, for the leader, checks the running child and its HTTP endpoint. Only a leader with a healthy child is labeled `otio.io/active=true`; Services select this label. Startup clears stale labels, and patches verify the pod UID. Standbys therefore never receive ordinary UI/API Service traffic. Killing the child or losing leadership terminates the supervisor; Kubernetes restarts it. Child termination on lost leadership is immediate to limit overlap. Use direct Go application binaries; shell wrappers spawning detached grandchildren are unsupported.

This is leader election without fencing. It does not promise exactly-once processing or immunity to process pauses, clock-rate skew or network partitions. See the complete operating and data-loss boundaries in `examples/07-kubernetes-ha/README.md`.

The supervisor continues checking after activation. A child that stops answering for `HA_FAILURE_TIMEOUT` (default `15s`) is terminated and the supervisor stops renewing its Lease. The startup deadline is separate: `HA_STARTUP_TIMEOUT` defaults to `2m` and also bounds failed endpoint activation. Both settings accept Go durations of at least `1s`. Recovery before the failure deadline clears the timer. These probes measure application responsiveness; broker outages are reported by the application and do not automatically kill it.

Tests exercise canceled leadership terminating a real subprocess, stopped-before-start behavior, sustained hangs, transient recovery, startup deadlines and failed Kubernetes endpoint activation. The example's failover script additionally checks application replacement, broker recovery and database primary change against a real Kubernetes cluster.
