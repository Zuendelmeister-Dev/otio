GO_MODULES := apps/presense-modbus apps/presense-opcua apps/sense apps/lense apps/dispense apps/plc4go-modbus apps/protocol-lab apps/ha-agent shared/mqttx

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

test-workspace:
	go test ./apps/presense-modbus/... ./apps/presense-opcua/... ./apps/sense/... ./apps/lense/... ./apps/dispense/... ./apps/plc4go-modbus/... ./apps/protocol-lab/... ./apps/ha-agent/... ./shared/mqttx/...

build:
	docker compose build

compose-up:
	docker compose up --build

compose-down:
	docker compose down

coverage:
	./scripts/coverage.ps1
