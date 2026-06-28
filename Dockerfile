FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/main ./cmd/cmd.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main ./

ARG APP_PORT=8080
ENV APP_PORT=${APP_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:${APP_PORT:-8080}/health || exit 1

CMD ["/app/main"]