FROM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod ./
COPY *.go ./

RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/presense-modbus .;     else       CGO_ENABLED=0 go build -o /out/presense-modbus .;     fi

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/presense-modbus /app/presense-modbus

ENTRYPOINT ["/app/presense-modbus"]
