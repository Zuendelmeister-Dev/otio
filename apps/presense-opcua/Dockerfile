FROM golang:1.22-alpine AS build
WORKDIR /src
COPY apps/presense-opcua/go.mod ./
COPY apps/presense-opcua/*.go ./
RUN go test ./...
RUN CGO_ENABLED=0 go build -o /out/presense-opcua .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/presense-opcua /app/presense-opcua
ENTRYPOINT ["/app/presense-opcua"]
