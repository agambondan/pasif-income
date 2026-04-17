# Podcast Clips Factory

This project automates the conversion of long-form videos into viral short-form clips.

## Features

- **Automated Selection**: AI (Gemini) analyzes transcripts to find the most engaging segments.
- **Auto-Framing**: Vision AI (MediaPipe) tracks faces to ensure subjects are centered in vertical (9:16) crops.
- **Dynamic Captions**: Ready-to-use pipeline for burning in captions (WIP).
- **Self-Hosted**: Docker-based architecture with local transcription and storage.
- **Distribution Matrix**: Traceable plan for multi-platform and multi-account upload.
- **Platform Auth**: OAuth-first account linking for upload destinations.

## Tech Stack

- **Backend**: Go (Golang)
- **AI**: Gemini 1.5 Pro (Strategy), Whisper (STT), MediaPipe (Vision)
- **Tools**: FFmpeg, yt-dlp, Python
- **Infrastructure**: PostgreSQL, MinIO, Docker

## Getting Started

1.  **Dependencies**:
    - Ensure `yt-dlp`, `ffmpeg`, and `python3` are installed on your host.
    - Install Python dependencies: `pip install mediapipe opencv-python numpy`.
2.  **Environment Variables**:
    - `GEMINI_API_KEY`: Your Google Gemini API Key.
    - `VIDEO_URL`: (Optional) The URL of the video to process.
3.  **Run Services**:
    ```bash
    docker-compose up -d
    ```
4.  **Run the Factory**:
    ```bash
    go mod tidy
    make run
    ```

## Project Structure

- `cmd/api`: Entry point.
- `internal/core`: Domain models and port interfaces.
- `internal/services`: Core workflow orchestrator.
- `internal/adapters`: Implementations for third-party tools and APIs.
- `scripts`: Helper scripts (e.g., face tracking).
- `docs/distribution-matrix.md`: Current target model for publish destinations and dashboard traceability.
- `docs/platform-auth.md`: OAuth-first approach for linking upload accounts.
