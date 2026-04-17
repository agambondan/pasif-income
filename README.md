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

## Requirement Umum
- Go 1.21+
- FFmpeg
- Python 3.x
- Docker & Docker Compose

## Lisensi
Private / Personal Use
