# Distribution Matrix

Dokumen ini menjelaskan target distribusi konten untuk `pasif-income` dan bagaimana dashboard akan mengontrolnya.

## Status Saat Ini

- `creator` sudah bisa menjalankan pipeline teknis faceless per job, tetapi portal operasionalnya masih partial.
- `clipper` sudah bisa menjalankan pipeline teknis clip pendek, tetapi portal operasionalnya masih partial.
- Pipeline upload masih **single target** per job.
- Belum ada UI checkbox untuk memilih multiple platform atau multiple account.
- OAuth connect untuk akun platform sekarang diarahkan ke flow real: YouTube API OAuth atau Chromium profile linking.

## Three Draft Tracks

Dokumen ini sekarang menjadi pegangan untuk tiga area kerja berikut:

1. `DB schema draft`
2. `API payload draft`
3. `Dashboard UI draft`

Satu layer tambahan yang harus disiapkan bersamaan adalah:

- `platform account linking` via API atau Chromium profile

## Target Model

Satu job bisa punya banyak tujuan distribusi:

- banyak platform
- banyak akun dalam satu platform
- kombinasi keduanya

Contoh:

- 1 video -> YouTube Shorts utama
- 1 video -> TikTok akun A dan akun B
- 1 video -> Instagram Reels akun brand

## Data Model yang Diinginkan

Job perlu menyimpan daftar tujuan distribusi, misalnya:

```json
{
  "niche": "stoicism",
  "topic": "how to control your mind",
  "destinations": [
    {
      "platform": "youtube",
      "account_id": "yt-main",
      "publish": true
    },
    {
      "platform": "tiktok",
      "account_id": "tt-brand",
      "publish": true
    }
  ]
}
```

### Draft Schema

Tabel inti yang kemungkinan dibutuhkan:

- `generation_jobs`
- `job_destinations`
- `connected_accounts`
- `publish_attempts`

Relasi kasar:

- satu job punya banyak destination
- satu destination mengarah ke satu akun terhubung
- satu destination bisa punya beberapa attempt publish

Field penting yang perlu ada:

- `platform`
- `account_id`
- `status`
- `remote_post_id`
- `remote_url`
- `error`
- `created_at`
- `updated_at`

## Draft API Payload

Contoh payload untuk submit job:

```json
{
  "niche": "stoicism",
  "topic": "how to control your mind",
  "destinations": [
    {
      "platform": "youtube",
      "account_id": "yt-main",
      "publish": true
    },
    {
      "platform": "tiktok",
      "account_id": "tt-brand",
      "publish": true
    }
  ]
}
```

Contoh respons job:

```json
{
  "id": "job_001",
  "status": "queued",
  "destinations": [
    {
      "platform": "youtube",
      "account_id": "yt-main",
      "status": "pending"
    }
  ]
}
```

## UI Konsep

Dashboard idealnya punya:

- checkbox per platform
- list akun di bawah platform tersebut
- toggle `publish` per tujuan
- ringkasan sebelum submit job
- status koneksi per akun
- tombol `Connect account` untuk API linking atau Chromium profile linking

Contoh flow:

1. User isi niche dan topic.
2. User pilih platform tujuan.
3. User pilih satu atau banyak akun per platform.
4. User submit job.
5. Worker memproses satu video lalu fan-out upload ke semua tujuan.

## Platform Login / Account Linking

Dashboard sebaiknya **tidak** minta user memasukkan password platform.

Model yang disarankan:

1. User klik `Connect account`.
2. Jika platform punya API yang matang, dashboard pakai OAuth resmi.
3. Jika platform lebih cocok browser-based, backend buat Chromium profile per email.
4. User login sekali di profile itu.
5. Backend simpan profile path dan status koneksi.
6. Dashboard menampilkan akun yang sudah terhubung sebagai checkbox option.

Kenapa ini lebih baik:

- lebih aman daripada simpan password
- lebih sesuai flow resmi API platform
- satu user bisa punya banyak akun terhubung
- token bisa di-refresh atau dicabut tanpa ubah UI

### Status Per Platform

- YouTube: paling jelas untuk OAuth upload, jadi kandidat API-first terbaik.
- TikTok: API ada, tapi coverage dan approval bisa lebih ketat, jadi Chromium profile tetap berguna untuk MVP.
- Instagram: sering lebih sensitif terhadap approval dan jenis akun, jadi browser automation sering lebih praktis untuk tahap awal.

## Workflow

1. `web` kirim request ke `app`.
2. `app` simpan job dan destinations ke database.
3. Worker yang sesuai memproses video.
4. Upload dilakukan per destination.
5. Dashboard menampilkan status per destination dan status job utama.

## Catatan Implementasi

- Untuk tahap awal, bisa mulai dari single upload adapter yang menerima daftar destination.
- Setelah itu baru pecah menjadi adapter per platform kalau API tiap platform butuh credential atau metadata yang berbeda.
- Untuk traceability, setiap upload perlu simpan:
  - platform
  - account id
  - asset id / URL
  - status
  - error message jika gagal

## Implementation Order

Urutan yang paling aman:

1. Tambah `connected_accounts`.
2. Tambah `job_destinations`.
3. Tambah UI checkbox + tombol connect account.
4. Tambah upload adapter per platform.
5. Tambah publish attempt logging.
6. Baru fan-out upload dari satu job ke banyak destination.

## Status Cross-Reference

Kalau butuh ringkasan current vs planned, lihat [Implementation Status](./implementation-status.md).

## Related Docs

- [Workflow Design](./workflow.md)
- [Pipeline Produksi](./pipeline.md)
- [Distribusi Clip](./distribution.md)
