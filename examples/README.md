# OT.io examples

## 01 Local Docker Compose

Runs the normal local Docker Compose example on one machine.

## 02 Raspberry Pi ARM distributed

Deploys the original OT.io application across four Raspberry Pis using Ansible.

```text
examples/02-raspberry-pi-arm-distributed
```

## 03 Protocol Lab

[Run the native protocol demo](03-protocol-lab/README.md): Modbus TCP, an RTU TCP tunnel, OPC UA Binary, MQTT/AMQP publishing, Sense collection and Lense.

## 04 Device gateways

[Connect physical devices](04-device-gateways/README.md) with S7, EtherNet/IP, BACnet/IP, KNXnet/IP or IEC 104; see which fieldbuses need a hardware gateway.

## 05 Kubernetes

[Deploy with Kustomize](05-kubernetes/README.md): Protocol Lab, Sense, Lense and brokers on kind or an existing cluster, with persistent Postgres/Sense storage and an end-to-end smoke test.

## 06 Helm

[Deploy with Helm](06-helm/README.md): the Protocol Lab stack as a self-contained chart with values, release-based resource names, persistent storage and Helm tests.

## 07 Kubernetes HA

[Deploy the HA example](07-kubernetes-ha/README.md): active/passive Sense/Lense, clustered MQTT and replicated PostgreSQL; includes failover exercises and explicit delivery limitations.
