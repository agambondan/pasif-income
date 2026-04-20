# Audience Engagement & Community Agent

Fitur ini menutup loop engagement setelah video tayang.

## Apa yang dilakukan

- Mengambil comment thread YouTube untuk video yang sudah `completed`.
- Menghasilkan draft reply dengan Gemini sesuai niche, topic, dan persona brand.
- Menyimpan draft reply ke Postgres supaya tampil di dashboard.
- Jika `COMMUNITY_AUTO_REPLY_ENABLED=true`, sistem juga mencoba mengirim reply ke YouTube secara otomatis.

## Alur

1. Worker membaca akun YouTube API yang sudah terhubung.
2. Worker mengambil distribution job `completed` dengan `external_id` valid.
3. Worker memuat comment thread terbaru dari video tersebut.
4. Gemini menyusun balasan singkat dan relevan.
5. Draft disimpan ke tabel `community_reply_drafts`.
6. Dashboard menampilkan draft reply, status, dan waktu sync terakhir.

## Environment

- `COMMUNITY_SYNC_INTERVAL_SECONDS`
- `COMMUNITY_MAX_COMMENTS_PER_VIDEO`
- `COMMUNITY_AUTO_REPLY_ENABLED`

## Catatan

- Fitur ini diprioritaskan untuk akun YouTube API karena comment reply resmi tersedia lewat YouTube Data API.
- Reply otomatis tetap best-effort. Kalau API menolak komentar, draft tetap tersimpan untuk review manual.
