#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."

printf '%s\n' 'Running Go tests for OT.io monorepo...'

for module in apps/presense-modbus apps/presense-opcua apps/sense apps/lense apps/dispense apps/plc4go-modbus apps/protocol-lab apps/ha-agent shared/mqttx; do
  printf '\n==> %s\n' "$module"
  (cd "$module" && go test ./...)
done

printf '\n%s\n' 'Running shared JavaScript tests...'
node shared/web/iot-ui.test.js
node shared/web/standard-chart.test.js
node shared/web/workspace-ui.test.js
node scripts/kubernetes-config.test.js
node scripts/protocol-deployment.test.js
node shared/web/live-messages.test.js
node scripts/repository-check.cjs

printf '\n%s\n' 'All tests finished successfully.'
