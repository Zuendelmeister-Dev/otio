FROM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/presense-opcua/go.mod ./
COPY apps/presense-opcua/*.go ./

RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/presense-opcua .;     else       CGO_ENABLED=0 go build -o /out/presense-opcua .;     fi

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/presense-opcua /app/presense-opcua

ENTRYPOINT ["/app/presense-opcua"]
