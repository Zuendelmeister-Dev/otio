FROM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/sense/go.mod ./
COPY shared/mqttx /shared/mqttx
COPY apps/sense ./
COPY shared/web/*.js ./static/

RUN go mod tidy
RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/sense .;     else       CGO_ENABLED=0 go build -o /out/sense .;     fi

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/sense /app/sense
COPY apps/sense/static ./static
COPY shared/web/*.js ./static/

ENTRYPOINT ["/app/sense"]
