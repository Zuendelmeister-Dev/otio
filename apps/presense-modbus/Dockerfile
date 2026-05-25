FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go test ./...
RUN CGO_ENABLED=0 go build -o /out/presense-modbus .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/presense-modbus /app/presense-modbus
ENTRYPOINT ["/app/presense-modbus"]
