# Multi-stage Dockerfile untuk Golang service
# Optimized untuk container-to-container communication

# Build Stage
FROM golang:1.23-alpine AS build-stage

# Install dependencies untuk build
RUN apk update && apk upgrade && \
    apk add --no-cache \
    bash \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go module files untuk better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build static binary yang tidak depend pada CGO
# Flags explanation:
# -a: force rebuilding of packages
# -installsuffix cgo: untuk memastikan tidak ada CGO dependencies
# -ldflags: linker flags untuk static linking
# -trimpath: remove file system paths dari binary
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -a \
    -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -trimpath \
    -o main .

# Production Stage - Option 1: Distroless (Recommended untuk HTTP-only)
FROM gcr.io/distroless/static-debian12:nonroot AS production-distroless

# Set working directory
WORKDIR /app

# Copy timezone data untuk proper time handling
COPY --from=build-stage /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary dan configuration files
COPY --from=build-stage /app/main /app/main
COPY --from=build-stage /app/.env /app/.env
COPY --from=build-stage /app/gcs-secret-key.json /app/gcs-secret-key.json

# Set timezone
ENV TZ=Asia/Jakarta

# Expose port
EXPOSE 8002

# Run as non-root user (distroless default)
ENTRYPOINT ["/app/main"]

# Production Stage - Option 2: Alpine (Alternative untuk debugging)
FROM alpine:3.19 AS production-alpine

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary dan config
COPY --from=build-stage /app/main /app/main
COPY --from=build-stage /app/.env /app/.env
COPY --from=build-stage /app/gcs-secret-key.json /app/gcs-secret-key.json

# Set proper permissions
RUN chown -R appuser:appgroup /app && \
    chmod +x /app/main

# Set timezone
ENV TZ=Asia/Jakarta

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8002

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD /app/main health || exit 1

ENTRYPOINT ["/app/main"]

# Production Stage - Option 3: Scratch (Minimal untuk HTTP-only)
FROM scratch AS production-scratch

# Copy timezone data
COPY --from=build-stage /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd file untuk user info
COPY --from=build-stage /etc/passwd /etc/passwd

# Copy binary dan config ke /app directory
COPY --from=build-stage /app/main /app/main
COPY --from=build-stage /app/.env /app/.env
COPY --from=build-stage /app/gcs-secret-key.json /app/gcs-secret-key.json

# Set timezone
ENV TZ=Asia/Jakarta

# Expose port
EXPOSE 8005

# Run as non-root (user nobody = uid 65534)
USER 65534:65534

ENTRYPOINT ["/app/main"]