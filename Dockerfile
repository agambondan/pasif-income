# 1. Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN go build -o api cmd/api/main.go

# 2. Final Stage
FROM python:3.11-alpine

# Install FFmpeg, curl, and build dependencies for python libs
RUN apk add --no-cache ffmpeg curl build-base libffi-dev

# Install yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp

# Install Python deps for Vision Agent
# Note: mediapipe and opencv might be heavy for alpine, but let's try.
# If this fails, we might need a different approach for vision.
RUN pip install opencv-python-headless numpy

WORKDIR /app
COPY --from=builder /app/api .
COPY scripts/ ./scripts/

EXPOSE 8080

CMD ["./api"]
