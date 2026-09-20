#!/usr/bin/env bash
set -euo pipefail
# Requires kubectl, Node.js and a running Example 05 in the current context.
namespace=${OTIO_NAMESPACE:-otio}
kubectl -n "$namespace" wait --for=condition=Available deployment --all --timeout=300s
get() { kubectl -n "$namespace" exec deployment/lense -- wget -qO- -T 15 "$1"; }
read_value() {
  kubectl -n "$namespace" exec deployment/lense -- wget -qO- -T 15 \
    --header='Content-Type: application/json' --post-data="$1" http://protocol-lab:8500/api/read |
    node -e 'let s="";process.stdin.on("data",d=>s+=d);process.stdin.on("end",()=>{const r=JSON.parse(s);if(typeof r.value!=="number"||!Number.isFinite(r.value))process.exit(1);console.log(r.protocol,r.value)})'
}
get http://lense:8000/protocol-lab/api/catalog | node -e 'let s="";process.stdin.on("data",d=>s+=d);process.stdin.on("end",()=>{if(JSON.parse(s).items.length<10)process.exit(1)})'
read_value '{"protocol":"modbus-tcp","connection":"modbus-tcp://protocol-lab:1502?default-unit-identifier=1","address":"holding-register:1:UINT"}'
read_value '{"protocol":"modbus-rtu-tcp","connection":"modbus-rtu:tcp://protocol-lab:1503?default-unit-identifier=1","address":"holding-register:1:UINT"}'
read_value '{"protocol":"opcua-tcp","connection":"opc.tcp://protocol-lab:4842","address":"ns=1;s=Temperature"}'
read_value '{"protocol":"mqtt","connection":"tcp://mqtt:1883","address":"lab/temperature"}'
read_value '{"protocol":"amqp","connection":"amqp://lab:lab@rabbitmq:5672/","address":"amq.topic/lab.temperature"}'
healthy=false
for attempt in $(seq 1 30); do
  if get http://lense:8000/api/agents | node -e 'let s="";process.stdin.on("data",d=>s+=d);process.stdin.on("end",()=>{const a=JSON.parse(s).items;process.exit(a.length===5&&a.every(x=>x.healthy&&x.connected)?0:1)})'; then
    healthy=true
    break
  fi
  sleep 2
done
if [ "$healthy" != true ]; then echo 'Not all five sources became healthy' >&2; exit 1; fi
count=$(kubectl -n "$namespace" exec deployment/postgres -- sh -ec 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "SELECT count(DISTINCT agent_id) FROM metric_events WHERE metric_value IS NOT NULL AND ts > now() - interval '\''2 minutes'\''"')
if [ "${count//[[:space:]]/}" != 5 ]; then echo 'Missing recent persisted numeric telemetry for one or more sources' >&2; exit 1; fi
echo 'Kubernetes native reads, brokers, topology and persisted telemetry passed'
