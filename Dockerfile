# Build Stage
FROM golang:alpine AS build-stage

RUN apk update && apk upgrade && \
    apk add --no-cache bash

WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o /app/main .

# Release Stage
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

# Copy the compiled binary from build stage
COPY --from=build-stage /app/main /main

# Copy environment configuration
COPY .env /.env

# Copy GCS secret key file
COPY gcs-secret-key.json /gcs-secret-key.json

# Set proper permissions for the secret key file
# Note: distroless doesn't have chmod, so we need to handle this in the build stage
USER nonroot:nonroot

EXPOSE 8888

ENTRYPOINT [ "/main" ]