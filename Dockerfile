FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/main ./cmd/cmd.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main ./
CMD ["/app/main"]