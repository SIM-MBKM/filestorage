FROM golang:alpine AS build-stage

RUN apk update && apk upgrade && \
    apk add --no-cache bash

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/main .


# Release Stage
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/main /main
COPY .env /.env

EXPOSE 8888

USER nonroot:nonroot

ENTRYPOINT [ "/main" ]