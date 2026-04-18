# Workflow Design

The system follows an event-driven, staged workflow:

## Current vs Target

- Current code already runs the generation, clipping, persistence, and dashboard control paths.
- Current distribution is still single-target and storage-oriented.
- Target distribution is multi-platform and multi-account, driven from API-linked or Chromium-profile-linked destinations.
- The current implementation snapshot lives in [Implementation Status](./implementation-status.md).

## 1. Ingestion Stage
- System takes a video URL (YouTube, etc.).
- `yt-dlp` downloads the video.
- `ffmpeg` extracts the audio for transcription.

## 2. Intelligence Stage
- `Whisper` (or local STT service) generates a transcript with word-level timestamps.
- `Gemini 1.5 Pro` analyzes the transcript using a "Viral Strategist" prompt.
- Result: A JSON list of viral segments with start/end times and headlines.

## 3. Production Stage
- For each segment:
    - `face_tracker.py` (MediaPipe) scans the segment to find the subject's face center.
    - `ffmpeg` performs a vertical crop centered on the face.
    - `ffmpeg` renders the final clip with optional captions.

## 4. Approval Stage (WIP)
- Clips are stored in `MinIO`.
- A Next.js dashboard allows the user to approve, reject, or request a re-edit with feedback.

## 5. Distribution Stage
- Approved clips are queued for upload to TikTok, YouTube Shorts, and Instagram Reels.
- Current code still treats upload as a single target.
- Multi-platform and multi-account distribution is planned in [Distribution Matrix](./distribution-matrix.md).
