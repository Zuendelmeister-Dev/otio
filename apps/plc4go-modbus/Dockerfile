FROM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/plc4go-modbus/go.mod ./
COPY apps/plc4go-modbus/*.go ./

RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/plc4go-modbus .;     else       CGO_ENABLED=0 go build -o /out/plc4go-modbus .;     fi

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/plc4go-modbus /app/plc4go-modbus

ENTRYPOINT ["/app/plc4go-modbus"]
