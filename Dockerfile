# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.1

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/gingonic-concurrency .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /bin/gingonic-concurrency /app/gingonic-concurrency
COPY config/config.yaml /app/config/config.yaml

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/gingonic-concurrency"]
