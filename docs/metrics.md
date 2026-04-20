# Metrics

Dokumen ini menjelaskan layer analytics untuk `pasif-income`.

## Tujuan

Analytics dipakai untuk menjawab pertanyaan operasional yang paling penting:

- video mana yang benar-benar dapat views
- akun atau niche mana yang paling efektif
- seberapa cepat pertumbuhan performa per video
- apakah distribution loop berjalan sehat setelah upload

## Current MVP

MVP analytics sekarang fokus ke YouTube API:

- backend menyimpan snapshot metrik video ke Postgres
- snapshot diambil dari YouTube Data API untuk video yang sudah ter-publish
- dashboard menampilkan ringkasan views, likes, comments, dan histori snapshot
- worker background melakukan sync periodik
- endpoint manual tersedia untuk memaksa refresh

## Data Model

Tabel utama:

- `video_metric_snapshots`

Field penting:

- `user_id`
- `generation_job_id`
- `distribution_job_id`
- `account_id`
- `platform`
- `external_id`
- `video_title`
- `view_count`
- `like_count`
- `comment_count`
- `collected_at`

Snapshot disimpan berulang kali, sehingga grafik growth bisa dibangun dari histori.

## Sync Flow

Alur sync metrik sekarang:

1. Ambil connected accounts milik user.
2. Ambil distribution job yang sudah `completed`.
3. Filter job YouTube yang punya `external_id`.
4. Panggil YouTube Data API `videos.list(part=snippet,statistics,id=...)`.
5. Simpan snapshot ke Postgres.

Endpoint yang tersedia:

- `GET /api/metrics`
- `POST /api/metrics/sync`

## Dashboard

Halaman `videos` sekarang menampilkan:

- total tracked videos
- total views
- total likes
- total comments
- latest metric snapshot per video
- recent snapshot history

## Next Step

Setelah ini, ekstensi paling bernilai adalah:

- chart growth per video atau per niche
- scheduled sync yang lebih cerdas per platform
- alert kalau performa drop tajam
- grouping metrics per niche, bukan hanya per video
