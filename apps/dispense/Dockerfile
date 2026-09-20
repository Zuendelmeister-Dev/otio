FROM golang:1.26.8-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/dispense/go.mod ./
COPY shared/mqttx /shared/mqttx
COPY apps/dispense/*.go ./
COPY apps/dispense/static ./static
COPY shared/web/*.js ./static/

RUN go mod tidy
RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/dispense .;     else       CGO_ENABLED=0 go build -o /out/dispense .;     fi

FROM alpine:3.24

WORKDIR /app

COPY --from=build /out/dispense /app/dispense
COPY apps/dispense/static ./static
COPY shared/web/*.js ./static/

ENTRYPOINT ["/app/dispense"]
