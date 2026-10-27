# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN go build -trimpath -ldflags="-s -w" -o /out/classgame-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

ENV HOST=0.0.0.0 \
    PORT=8080

COPY --from=builder /out/classgame-api /app/classgame-api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/classgame-api"]
