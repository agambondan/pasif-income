# Implementation Status

Dokumen ini menjadi ringkasan kerja aktual di repo `pasif-income`.

## Done

- Dashboard web sudah berjalan sebagai control panel.
- API backend sudah menyediakan endpoint untuk login, jobs, clips, accounts, platforms, dan health.
- Faceless content generator sudah punya pipeline end-to-end:
  - script
  - voiceover
  - images
  - assembly
  - upload ke storage
- Podcast clipper sudah punya pipeline end-to-end:
  - download
  - transcribe
  - analyze
  - crop/render
  - upload ke storage
  - save metadata ke database
- Postgres sudah dipakai untuk persist:
  - users
  - sessions
  - connected accounts
  - generation jobs
  - distribution jobs
  - clips
- MinIO sudah dipakai sebagai object storage.
- Docker stack sudah dipisah ke service yang berbeda:
  - `app`
  - `web`
  - `clipper`
  - `creator`
- Session-based dashboard auth sudah aktif via cookie backend.
- Chromium profile provisioning sudah aktif per platform/email.
- Distribution worker sudah memproses pending `distribution_jobs`.
- Metrics worker sudah melakukan sync snapshot metrik YouTube ke Postgres.
- Publisher adapter sudah punya:
  - YouTube API path
  - Chromium profile fallback

## Current Notes

- OAuth connect untuk Chromium profile sudah membuat profile path nyata dan mengikat ke session user.
- YouTube API connect sekarang memakai OAuth redirect + token exchange, dengan scope read untuk analytics.
- Dashboard videos sekarang punya panel analytics metrik dasar.
- `distribution_jobs` dan metrics masih diproses dengan worker polling sederhana, belum queue/broker terpisah.

## Not Started

- retry/failover per destination
- platform-specific upload adapter yang benar-benar real
- Chromium browser automation yang benar-benar menekan UI platform
- chart growth analytics per niche atau per video

## Recommended Next Order

1. Retry/failover per destination.
2. Real browser automation per platform.
3. Chart growth analytics per niche atau per video.
4. UI checkbox platform dan account yang sudah connected.
5. Smart scheduling untuk drip feed upload.

## Related Docs

- [Workflow Design](./workflow.md)
- [Distribution Matrix](./distribution-matrix.md)
- [Platform Auth](./platform-auth.md)
- [Pipeline Produksi](./pipeline.md)
