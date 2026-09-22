FROM golang:1.26.8-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY apps/presense-opcua/go.mod ./
COPY apps/presense-opcua/*.go ./

RUN go test ./...
RUN if [ -n "$TARGETOS" ] && [ -n "$TARGETARCH" ]; then       CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -o /out/presense-opcua .;     else       CGO_ENABLED=0 go build -o /out/presense-opcua .;     fi

FROM alpine:3.24

WORKDIR /app

COPY --from=build /out/presense-opcua /app/presense-opcua
COPY shared/web/*.js /app/static/

ENTRYPOINT ["/app/presense-opcua"]
