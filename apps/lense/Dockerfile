FROM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/lense/go.mod ./
COPY shared/mqttx /shared/mqttx
COPY apps/lense/*.go ./
COPY apps/lense/static ./static
COPY apps/lense/config ./config
COPY shared/web/*.js ./static/

RUN go mod tidy
RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/lense .;     else       CGO_ENABLED=0 go build -o /out/lense .;     fi

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/lense /app/lense
COPY apps/lense/static ./static
COPY apps/lense/config ./config
COPY shared/web/*.js ./static/

ENTRYPOINT ["/app/lense"]
