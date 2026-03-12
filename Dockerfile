FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o env-king ./main.go

FROM alpine:3.20

WORKDIR /app

RUN mkdir -p /app/workdir

COPY --from=builder /app/env-king /usr/local/bin/env-king
COPY config.dev.yaml /app/config.dev.yaml

EXPOSE 8080

ENV ENV_KING_WORKDIR=/app/workdir
ENV ENV_KING_CONFIG=/app/config.dev.yaml

CMD ["env-king", "server"]

