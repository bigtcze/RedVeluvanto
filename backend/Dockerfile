# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
RUN CGO_ENABLED=0 go build -o redveluvanto .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/redveluvanto .
COPY --from=builder /build/migrations ./migrations

EXPOSE 8090

CMD ["./redveluvanto", "serve", "--http=0.0.0.0:8090"]
