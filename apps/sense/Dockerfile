FROM golang:1.26.8-alpine AS ha-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /ha
COPY apps/ha-agent/ ./
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /ha-agent .
FROM golang:1.26.8-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/sense/go.mod ./
COPY shared/mqttx /shared/mqttx
COPY apps/sense ./
COPY examples/03-protocol-lab /examples/03-protocol-lab
COPY shared/web/*.js ./static/

RUN go mod tidy
RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/sense .;     else       CGO_ENABLED=0 go build -o /out/sense .;     fi

FROM alpine:3.24

WORKDIR /app
COPY --from=ha-build /ha-agent /app/ha-agent

COPY --from=build /out/sense /app/sense
COPY apps/sense/static ./static
COPY shared/web/*.js ./static/

ENTRYPOINT ["/app/sense"]
