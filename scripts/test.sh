#!/usr/bin/env sh
set -eu

printf '%s\n' 'Running Go tests for OT.io monorepo...'

for module in apps/presense-modbus apps/presense-opcua apps/sense apps/lense apps/dispense shared/mqttx; do
  printf '\n==> %s\n' "$module"
  (cd "$module" && go test ./...)
done

printf '\n%s\n' 'Running shared JavaScript tests...'
node shared/web/iot-ui.test.js
node shared/web/standard-chart.test.js

printf '\n%s\n' 'All tests finished successfully.'
