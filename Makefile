GO_MODULES := apps/presense-modbus apps/presense-opcua apps/sense apps/lense apps/dispense shared/mqttx

.PHONY: test test-go test-js test-workspace build compose-up compose-down coverage

test: test-go test-js

test-go:
	@for module in $(GO_MODULES); do \
		echo ""; \
		echo "==> $$module"; \
		(cd $$module && go test ./...); \
	done

test-js:
	node shared/web/iot-ui.test.js
	node shared/web/standard-chart.test.js

test-workspace:
	go test ./apps/presense-modbus/... ./apps/presense-opcua/... ./apps/sense/... ./apps/lense/... ./apps/dispense/... ./shared/mqttx/...

build:
	docker compose build

compose-up:
	docker compose up --build

compose-down:
	docker compose down

coverage:
	./scripts/coverage.ps1
