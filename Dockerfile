# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/asnakech-api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
	&& adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/asnakech-api /app/asnakech-api

USER appuser
EXPOSE 8080

ENV PORT=8080 \
	ENV=production \
	LOG_LEVEL=info

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/asnakech-api"]
