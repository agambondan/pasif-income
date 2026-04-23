# syntax=docker/dockerfile:1

FROM golang:1.26.2-alpine3.23 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api cmd/api/main.go

FROM python:3.14.4-slim

WORKDIR /app
COPY --from=builder /app/api .
COPY scripts/ ./scripts/

# Set environment for Playwright/Chromium
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

EXPOSE 8080

CMD ["./api"]
