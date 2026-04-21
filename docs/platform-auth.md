# Platform Auth

Dokumen ini menjelaskan pendekatan auth untuk upload otomatis ke platform sosial.

## Current State

- Backend sekarang punya flow connect Chromium profile yang memakai session user aktif, lalu mengantre browser login untuk profile itu saat setup awal.
- `/api/auth/{platform}` mendukung `chromium_profile` untuk semua platform dan `api` untuk YouTube.
- Flow YouTube API melakukan OAuth redirect, exchange token, dan simpan access token/refresh token ke backend.
- OAuth scope YouTube sekarang mencakup upload dan readonly supaya metrics sync bisa jalan dari token yang sama.
- Backend publish path sekarang dipisah ke adapter YouTube API dan adapter Chromium profile, supaya boundary auth dan upload lebih jelas.

## Prinsip

- Jangan minta user mengisi password platform di dashboard.
- Pakai OAuth resmi kalau platform memang menyediakan API upload yang matang.
- Untuk MVP browser, pakai 1 email = 1 Chromium profile.
- Simpan token atau session state di backend, bukan di browser.
- Sediakan revoke flow dan status koneksi per akun.

## MVP Direction

MVP sekarang diarahkan ke:

- `API` untuk platform yang memang stabil dan resmi, dimulai dari YouTube OAuth
- `Chromium profile automation` untuk platform yang tidak enak di API

Aturan operasional yang dipakai:

- 1 email platform = 1 Chromium profile
- 1 Chromium profile hanya dipakai untuk 1 platform account
- setelah akun login sekali, upload berikutnya memakai profile yang sama
- kalau platform punya API yang layak, backend tetap boleh pakai jalur API

Untuk tahap awal, `browser` di UI berarti `chromium_profile`.

Alur browser profile yang dipakai sekarang:

1. user isi email profile
2. backend membuat profile path untuk email itu
3. backend mengantre browser login sekali untuk profile tersebut
4. publish berikutnya memakai profile path yang sama
5. UI membedakan action API dan Browser Profile supaya flow-nya tidak tercampur

Status profile yang ditampilkan UI dihitung dari folder profile:

- `missing`
- `provisioned`
- `needs_login`
- `ready`

UI juga menyediakan `Refresh Status` untuk memaksa backend membaca ulang state folder profile setelah operator login di browser.

Host-side launcher:

- request login Chromium profile ditulis ke `.runtime/browser-launch-requests`
- script host `scripts/browser_launcher.py watch` membuka Chromium di desktop host dari file request itu
- backend tidak lagi mencoba membuka browser dari container app

## YouTube

YouTube upload secara resmi menggunakan OAuth 2.0.

- request upload butuh token akses user
- operasi yang mengubah data memerlukan OAuth token
- untuk upload video, API resmi memakai `videos.insert`

## TikTok

TikTok Content Posting API juga memakai user authorization.

- user harus authorize app
- direct post/upload memakai token user
- API mendukung `FILE_UPLOAD` dan `PULL_FROM_URL`
- account info dipakai untuk render UI dan validasi before publish

## Instagram / Meta

Untuk Instagram, pendekatan yang paling aman adalah:

- connect akun via jalur official Meta
- simpan token hasil OAuth
- hanya tampilkan akun yang memang berhasil diverifikasi oleh backend

Catatan:

- implementasi detail Meta/Instagram lebih sensitif terhadap account type dan approval app
- sebaiknya desain backend dibuat abstrak, supaya UI tidak tergantung satu vendor saja

## Recommended Backend Shape

Backend simpan data ini:

- provider/platform
- account label
- external account id
- token access
- token refresh jika ada
- expiry
- scopes
- status koneksi
- chromium profile path untuk akun browser-based

Dan expose endpoint seperti:

- `POST /api/integrations/:platform/connect`
- `GET /api/integrations`
- `GET /api/integrations/:platform/accounts`
- `POST /api/integrations/:platform/:account_id/revoke`
