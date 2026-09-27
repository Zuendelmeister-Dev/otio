FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY apps/protocol-lab/go.mod apps/protocol-lab/go.sum ./
RUN go mod download
COPY apps/protocol-lab ./
RUN go test ./...
RUN CGO_ENABLED=0 go build -o /protocol-lab .
FROM alpine:3.24
RUN apk add --no-cache ca-certificates
COPY --from=build /protocol-lab /usr/local/bin/protocol-lab
EXPOSE 8500 1502 1503 4842 1102 1504
ENTRYPOINT ["protocol-lab"]
