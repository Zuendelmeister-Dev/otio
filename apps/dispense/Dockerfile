FROM golang:1.22-alpine AS build
WORKDIR /src
COPY apps/dispense/go.mod ./
COPY shared/mqttx /shared/mqttx
COPY apps/dispense/*.go ./
COPY apps/dispense/static ./static
COPY shared/web/*.js ./static/
RUN go mod tidy
RUN go test ./...
RUN CGO_ENABLED=0 go build -o /out/dispense .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/dispense /app/dispense
COPY apps/dispense/static ./static
COPY shared/web/*.js ./static/
ENTRYPOINT ["/app/dispense"]
