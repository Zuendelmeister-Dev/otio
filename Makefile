GO_MODULES := apps/presense-modbus apps/presense-opcua apps/sense apps/lense apps/dispense apps/plc4go-modbus apps/protocol-lab apps/ha-agent shared/mqttx shared/protocolmeta

.PHONY: test test-go test-js test-workspace build compose-up compose-down coverage

test: test-go test-js

test-go:
	@set -e; for module in $(GO_MODULES); do \
		echo ""; \
		echo "==> $$module"; \
		(cd $$module && go test ./...); \
	done

test-js:
	node shared/web/iot-ui.test.js
	node shared/web/standard-chart.test.js
	node shared/web/workspace-ui.test.js
	node scripts/kubernetes-config.test.js
	node scripts/generate-protocol-capabilities.cjs --check
	node scripts/protocol-capabilities.test.js
	node scripts/lense-config-workflow.test.js
	node scripts/sense-config-workflow.test.js
	node scripts/arm-bundle.test.js
	node shared/web/operator-ui.test.js

test-workspace:
	go test ./apps/presense-modbus/... ./apps/presense-opcua/... ./apps/sense/... ./apps/lense/... ./apps/dispense/... ./apps/plc4go-modbus/... ./apps/protocol-lab/... ./apps/ha-agent/... ./shared/mqttx/... ./shared/protocolmeta/...

build:
	docker compose build

compose-up:
	docker compose up --build

compose-down:
	docker compose down

coverage:
	./scripts/coverage.ps1
