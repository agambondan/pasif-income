# Platform Auth

Dokumen ini menjelaskan pendekatan auth untuk upload otomatis ke platform sosial.

## Prinsip

- Jangan minta user mengisi password platform di dashboard.
- Pakai OAuth / official account linking.
- Simpan token di backend, bukan di browser.
- Sediakan revoke flow dan status koneksi per akun.

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

Dan expose endpoint seperti:

- `POST /api/integrations/:platform/connect`
- `GET /api/integrations`
- `GET /api/integrations/:platform/accounts`
- `POST /api/integrations/:platform/:account_id/revoke`
