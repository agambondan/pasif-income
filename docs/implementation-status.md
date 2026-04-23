# Implementation Status

Dokumen ini menjadi ringkasan kerja aktual di repo `pasif-income`.

Catatan: `done` di sini berarti produk/portal operasional lengkap. Dengan definisi itu, faceless channel dan podcast clipper masih **partial** karena pipeline teknis sudah ada, tetapi portal/UX operasional belum matang.

## Partial

- Dashboard web sudah berjalan sebagai control panel.
- API backend sudah menyediakan endpoint untuk login, jobs, clips, accounts, platforms, dan health.
- Faceless content generator sudah punya pipeline teknis end-to-end:
  - script
  - voiceover dengan preset selectable dari dashboard
  - images
  - assembly
  - upload ke storage
- Podcast clipper sudah punya pipeline teknis end-to-end:
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
- Chromium profile provisioning sudah aktif per platform/email, dan connect browser profile akan membuka login session sekali saat setup awal.
- Chromium profile root sekarang shared via `browser_data` volume dan backend membaca `BROWSER_PROFILES_DIR` terlebih dulu, supaya launcher melihat profile path yang sama.
- Browser launcher sekarang jalan di bawah `xvfb-run` di container, jadi flow login tetap hidup walau display host tidak bisa dibind ke Docker.
- Startup lokal menyemai satu browser QA account `YouTube QA Browser` agar dashboard punya destination `ready` untuk smoke flow tanpa setup manual.
- Generator dan clipper sekarang memakai account picker yang dikelompokkan per platform, lengkap dengan Select All/Clear, badge auth method, dan gating browser profile yang belum `ready`.
- Voice preset dashboard sudah bisa dipilih di generator page, dengan fallback default dari `VOICE_TYPE` atau `en-US-Standard-A`.
- Portal creator, clipper, dan integrations sekarang punya shell copy yang lebih konsisten, job/distribution summary yang lebih jelas, serta empty state yang lebih operasional.
- Browser automation upload sekarang memakai wait signal yang lebih spesifik per platform, dan Instagram selector sudah lebih cocok untuk flow media upload/reels.
- Distribution worker sudah memproses pending `distribution_jobs`.
- Metrics worker sudah melakukan sync snapshot metrik YouTube ke Postgres.
- Dashboard analytics sudah menampilkan growth by niche, video, platform, dan akun.
- Dashboard analytics sekarang juga menampilkan alert kalau performa drop tajam.
- Quality control agent sekarang memblokir upload kalau render atau isi konten gagal review.
- Branding profile per niche sekarang menambahkan avatar cache, watermark, dan intro/outro.
- Niche research sekarang bisa menarik signal tren live dan menyarankan topic langsung ke dashboard.
- Dynamic affiliate insertion sekarang menempelkan disclosure, link, dan pin comment ke job production.
- Audience engagement agent sekarang menarik komentar YouTube dan menyimpan draft reply, dengan auto-reply opsional via env.
- Publisher adapter sudah punya:
  - YouTube API path
  - Chromium profile fallback

## Current Notes

- Portal creator dan clipper masih perlu hardening UX/ops sebelum dianggap produk final.
- OAuth/API connect dan browser profile connect sekarang dipisah di UI integrations.
- Chromium profile connect membuat profile path nyata, lalu membuka login browser sekali saat setup awal.
- Chromium profile connect sekarang memakai shared profile root yang sama antara app dan launcher, jadi path profile tidak lagi container-local.
- UI integrations sekarang menampilkan status browser profile berdasarkan isi folder profile, supaya operator tahu mana yang masih `needs_login` dan mana yang sudah `ready`.
- UI integrations juga punya `Refresh Status` untuk probe ulang browser profile setelah login manual.
- Chromium profile login sekarang di-queue ke host-side launcher, bukan dibuka dari container app.
- YouTube API connect sekarang memakai OAuth redirect + token exchange, dengan scope read untuk analytics.
- Backend publish path sekarang dipisah jelas:
  - YouTube API upload adapter
  - Chromium profile upload adapter
  - fallback hanya berlaku di jalur YouTube ketika diaktifkan via env
- Dashboard videos sekarang punya panel analytics metrik dasar.
- `distribution_jobs` dan metrics masih diproses dengan worker polling sederhana, belum queue/broker terpisah.
- QC reviewer masih heuristik-first dengan Gemini sebagai reviewer tambahan kalau kredensial ada.
- Avatar branding masih cache-first; belum ada dashboard editor untuk memilih persona secara manual.
- Trend research masih live-fetch first; belum ada penyimpanan histori research per niche.
- Affiliate catalog masih config-driven; belum ada marketplace sync atau tracking klik.
- Community reply drafts masih fokus ke YouTube API; belum ada panel moderation lengkap atau export ke platform lain.
- Distribution failover per destination sekarang akan enqueue akun cadangan satu platform saat publish gagal.

## Not Started

- platform-specific upload adapter yang benar-benar real
- Chromium browser automation yang benar-benar menekan UI platform
- Smart scheduling untuk drip feed upload (Done).
- UI checkbox platform dan account yang sudah connected (Done).
- Comparison view between accounts on same platform (Done).
- Batch processing untuk metrics sync (Done).

## Recommended Next Order

1. Phase 1 production usability hardening untuk creator, clipper, integrations, dan shell dashboard.
2. Real browser automation per platform.
3. UI checkbox platform dan account yang sudah connected.
4. Smart scheduling untuk drip feed upload.
5. Comparison view between accounts on same platform.
6. Scheduled sync yang lebih cerdas per platform.

## Roadmap Reference

- [Future Roadmap](./future-roadmap.md)
- Phase 1 di roadmap tersebut sekarang menjadi prioritas utama untuk hardening portal operasional.

## Related Docs

- [Workflow Design](./workflow.md)
- [Distribution Matrix](./distribution-matrix.md)
- [Platform Auth](./platform-auth.md)
- [Pipeline Produksi](./pipeline.md)
