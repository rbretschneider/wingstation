# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /wingstation \
    ./cmd/wingstation

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="WingStation"
LABEL org.opencontainers.image.description="Read-only Docker container dashboard for homelabbers"
LABEL org.opencontainers.image.source="https://github.com/rbretschneider/wingstation"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=builder /wingstation /wingstation

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/wingstation"]
