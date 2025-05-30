# Stage 1: Build Stage
FROM golang:1.24-alpine AS builder

# Create .aws folder for credentials (testing only)
RUN mkdir -p /home/appuser/.aws

# Create a non-root user and group in the build stage
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -a -o app ./cmd/server

# Stage 2: Production Stage (scratch)
FROM scratch

# Create .aws folder for credentials (testing only)
COPY --from=builder /home/appuser/.aws /home/appuser/.aws

# Copy the binary from the builder and set correct ownership
COPY --from=builder /app/app /app/app
COPY --from=builder /app/migrations/ /migrations/

# Copy non-root user and group details from the build stage
COPY --from=builder /etc/passwd /etc/group /etc/

# Copy certificates from build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set ownership and permissions for the binary
USER appuser

# Run the app as the non-root user
ENTRYPOINT ["/app/app"]
