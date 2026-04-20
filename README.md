# Pasif Income Suite

Kumpulan tool otomatisasi konten untuk membangun aset digital (pasif income) menggunakan AI.

## Struktur Project

Project ini dikelola dalam format monorepo:

### [01-faceless-channel](./01-faceless-channel)
Generator konten pendek (TikTok/Shorts/Reels) otomatis.
- **AI Strategist**: Gemini 1.5 Pro.
- **Voiceover**: TTS (gTTS).
- **Visuals**: Automated Slideshow with dynamic captions.

### [02-podcast-clips-factory](./02-podcast-clips-factory)
Pabrik klip pendek dari video durasi panjang (podcast/seminar).
- **Face Tracking**: MediaPipe (Vision AI).
- **Orchestration**: Go Backend with FFmpeg.
- **Infrastructure**: Self-hosted with MinIO & PostgreSQL.

## Auth & Runtime

- Gemini adapter menerima `GEMINI_API_KEY`, `GEMINI_ACCESS_TOKEN`, atau auth file lokal di `~/.gemini/oauth_creds.json`.
- Codex adapter menerima `OPENAI_API_KEY`, `OPENAI_ACCESS_TOKEN`, atau auth file lokal di `~/.codex/auth.json`.
- `cmd/creator`, `cmd/api`, dan `cmd/clipper` sudah membaca credential lokal itu saat startup.

## Requirement Umum
- Go 1.21+
- FFmpeg
- Python 3.x
- Docker & Docker Compose

## Lisensi
Private / Personal Use
