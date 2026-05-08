FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /bin/inventory-service ./cmd/consumer

FROM golang:1.24-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
COPY vendor/ vendor/
EXPOSE 3003
CMD ["air", "-c", "/air.toml"]

FROM alpine:3.20
RUN apk add --no-cache wget ca-certificates tzdata
COPY --from=builder /bin/inventory-service /inventory-service
EXPOSE 3003
ENTRYPOINT ["/inventory-service"]
